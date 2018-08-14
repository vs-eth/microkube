package cmd

import (
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"k8s.io/client-go/kubernetes"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"time"
	log "github.com/sirupsen/logrus"
	"github.com/pkg/errors"
	av1 "k8s.io/api/core/v1"
	"encoding/json"
	"k8s.io/api/policy/v1beta1"
)

type kubeBoolPatch struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value bool `json:"value"`
}

type KubeClient struct {
	client *kubernetes.Clientset
	node string
	nodeRef *av1.Node
}

func NewKubeClient (kubeconfig string) (*KubeClient, error) {
	obj := KubeClient{
		node: "",
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, errors.Wrap(err, "Couldn't read kubeconfig")
	}
	obj.client, err = kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "Couldn't init kube client")
	}
	return &obj, nil
}

func (k *KubeClient) findNode() {
	if k.node != "" {
		return
	}
	nodeList, err := k.client.CoreV1().Nodes().List(v1.ListOptions{})
	if err != nil {
		log.WithFields(log.Fields{
			"app":       "microkube",
			"component": "kube-interface",
		}).WithError(err).Fatalf("Couldn't list nodes!")
		os.Exit(-1)
	}
	if len(nodeList.Items) < 1 {
		log.WithFields(log.Fields{
			"app":       "microkube",
			"component": "kube-interface",
		}).Info("No node registered yet")
	}
	if len(nodeList.Items) > 1 {
		log.WithFields(log.Fields{
			"app":       "microkube",
			"component": "kube-interface",
			"nodeList":  nodeList,
		}).Fatalf("Too many nodes registered")
		os.Exit(-1)
	}
	k.nodeRef = &nodeList.Items[0]
	k.node = k.nodeRef.Name
}

func (k *KubeClient) setNodeUnschedulable(unschedulable, firstPass bool) {
	// If we add the taint, try adding the attribute first ;)
	payload := []kubeBoolPatch{{
		Op: "replace",
		Path: "/spec/unschedulable",
		Value: unschedulable,
	}}
	if firstPass && unschedulable {
		payload[0].Op = "add"
	}
	payloadBin, _ := json.Marshal(payload)
	_, err := k.client.CoreV1().Nodes().Patch(k.nodeRef.ObjectMeta.Name, types.JSONPatchType, payloadBin)
	if err != nil {
		if firstPass && unschedulable {
			k.setNodeUnschedulable(unschedulable, false)
		} else {
			log.WithFields(log.Fields{
				"app":       "microkube",
				"component": "kube-interface",
				"node":      k.nodeRef.ObjectMeta.Name,
			}).WithError(err).Warn("Couldn't (un)cordon node!")
		}
	}
}

func (k *KubeClient) DrainNode() {
	// Force client to refresh node
	k.node = ""
	k.findNode()
	if k.nodeRef == nil {
		log.WithFields(log.Fields{
			"app":       "microkube",
			"component": "kube-interface",
		}).Fatalf("No node found while draining node?")
		os.Exit(-1)
	}
	// Step 1: Disable scheduling on the node
	k.setNodeUnschedulable(true, true)
	// Step 2: Try to remove all pods. This needs to be done pod-by-pod
	pods, err := k.client.CoreV1().Pods(av1.NamespaceAll).List(v1.ListOptions{})
	if err != nil {
		log.WithFields(log.Fields{
			"app":       "microkube",
			"component": "kube-interface",
		}).WithError(err).Fatalf("Couldn't list pods")
		os.Exit(-1)
	}
	var pendingPods []av1.Pod
	for _,pod := range pods.Items {
		// Create eviction for this pod
		TEN := int64(10)
		eviction := v1beta1.Eviction{
			TypeMeta: v1.TypeMeta{
				APIVersion: "v1beta1",
				Kind: "Eviction",
			},
			ObjectMeta: v1.ObjectMeta{
				Name: pod.Name,
				Namespace: pod.Namespace,
			},
			DeleteOptions: &v1.DeleteOptions{
				GracePeriodSeconds: &TEN,
			},
		}
		log.WithFields(log.Fields{
			"app":       "microkube",
			"component": "kube-interface",
			"namespace": pod.Namespace,
			"pod": pod.Name,
		}).Info("Evicting pod...")
		err := k.client.PolicyV1beta1().Evictions(eviction.Namespace).Evict(&eviction)
		if err != nil {
			log.WithFields(log.Fields{
				"app":       "microkube",
				"component": "kube-interface",
				"namespace": pod.Namespace,
				"pod": pod.Name,
			}).WithError(err).Warn("Couldn't evict pod!")
		} else {
			pendingPods = append(pendingPods, pod)
		}
	}
	log.WithFields(log.Fields{
		"app":       "microkube",
		"component": "kube-interface",
	}).Info("Waiting for evicted pods to stop...")
	for {
		runningPods := 0
		for _, pod := range pendingPods {
			_, err := k.client.CoreV1().Pods(pod.Namespace).Get(pod.Name, v1.GetOptions{})
			logCtx := log.WithFields(log.Fields{
				"app":       "microkube",
				"component": "kube-interface",
				"namespace": pod.Namespace,
				"pod":       pod.Name,
			})
			if err != nil {
				if apierrors.IsNotFound(err) {
					logCtx.Debug("Pod is gone")
				} else {
					logCtx.Warn("Couldn't check pod state, assuming it's dead")
				}
			} else {
				runningPods++
				logCtx.Info("Pod is still running")
			}
		}
		if runningPods == 0 {
			log.WithFields(log.Fields{
				"app":       "microkube",
				"component": "kube-interface",
			}).Info("All pods gone!")
			return
		}
		time.Sleep(2*time.Second)
	}
}

func (k *KubeClient) WaitForNode() {
	for {
		// Always refresh
		k.node = ""
		k.findNode()
		if k.nodeRef == nil {
			time.Sleep(1*time.Second)
			continue
		}
		nodeReady := false
		statusChecked := false
		for _, condition := range k.nodeRef.Status.Conditions {
			if condition.Type == av1.NodeReady {
				statusChecked = true
				nodeReady = condition.Status == av1.ConditionTrue
			}
		}
		if !statusChecked {
			log.WithFields(log.Fields{
				"app": "microkube",
				"component": "kube-interface",
			}).Warn("Node status is unavailable")
		}
		if nodeReady {
			log.WithFields(log.Fields{
				"app": "microkube",
				"component": "kube-interface",
				"canSchedule": !k.nodeRef.Spec.Unschedulable,
			}).Info("Node now ready!")

			if k.nodeRef.Spec.Unschedulable {
				k.setNodeUnschedulable(false, true)
			}
			return
		}
		time.Sleep(1*time.Second)
	}
}