/*
 * Copyright 2018 The microkube authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmd

import (
	"encoding/json"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	av1 "k8s.io/api/core/v1"
	"k8s.io/api/policy/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"time"
)

// kubeBoolPatch is used to serialize a boolean change to JSON
type kubeBoolPatch struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value bool   `json:"value"`
}

// KubeClient abstracts operations on a running kubernetes cluster
type KubeClient struct {
	// Kubernetes client set for interacting with the real API
	client *kubernetes.Clientset
	// Name of the single node
	node string
	// Object reference to the single node
	nodeRef *av1.Node
}

// NewKubeClient creates a KubeClient object, configuring it from the provided kubeconfig. The connection will be
// established in this function
func NewKubeClient(kubeconfig string) (*KubeClient, error) {
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

// findNode ensures that there is only one node and updates the internal fields 'node' and 'nodeRef' to reference it
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
	}
	k.nodeRef = &nodeList.Items[0]
	k.node = k.nodeRef.Name
}

// setNodeUnschedulable sets a node (un)schedulable. The 'firstPass' parameter is required to be set to true by users
// as it is used internally for recursion
func (k *KubeClient) setNodeUnschedulable(unschedulable, firstPass bool) {
	// If we add the taint, try adding the attribute first ;)
	payload := []kubeBoolPatch{{
		Op:    "replace",
		Path:  "/spec/unschedulable",
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

// DrainNode drains a node, that is stopping all pods on it
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
	for _, pod := range pods.Items {
		// Create eviction for this pod
		TEN := int64(10) // We require a pointer to this!
		eviction := v1beta1.Eviction{
			TypeMeta: v1.TypeMeta{
				APIVersion: "v1beta1",
				Kind:       "Eviction",
			},
			ObjectMeta: v1.ObjectMeta{
				Name:      pod.Name,
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
			"pod":       pod.Name,
		}).Info("Evicting pod...")
		err := k.client.PolicyV1beta1().Evictions(eviction.Namespace).Evict(&eviction)
		if err != nil {
			log.WithFields(log.Fields{
				"app":       "microkube",
				"component": "kube-interface",
				"namespace": pod.Namespace,
				"pod":       pod.Name,
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
		time.Sleep(2 * time.Second)
	}
}

// WaitForNode delays execution until a single node exists and is in state 'Ready', removing the unschedulable taint
// if possible
func (k *KubeClient) WaitForNode() {
	for {
		// Always refresh
		k.node = ""
		k.findNode()
		if k.nodeRef == nil {
			time.Sleep(1 * time.Second)
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
				"app":       "microkube",
				"component": "kube-interface",
			}).Warn("Node status is unavailable")
		}
		if nodeReady {
			log.WithFields(log.Fields{
				"app":         "microkube",
				"component":   "kube-interface",
				"canSchedule": !k.nodeRef.Spec.Unschedulable,
			}).Info("Node now ready!")

			if k.nodeRef.Spec.Unschedulable {
				k.setNodeUnschedulable(false, true)
			}
			return
		}
		time.Sleep(1 * time.Second)
	}
}
