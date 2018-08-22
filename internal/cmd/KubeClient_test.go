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
	"context"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"testing"
	"time"
)

func mockClientWithNode(name string, unschedulable, haveNode bool) *fake.Clientset {
	var mockObjs []runtime.Object
	if haveNode {
		node := v1.Node{
			Spec: v1.NodeSpec{},
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
			Status: v1.NodeStatus{
				Conditions: []v1.NodeCondition{
					{
						Type:   v1.NodeReady,
						Status: v1.ConditionTrue,
					},
				},
			},
		}
		if unschedulable {
			node.Spec.Unschedulable = true
		}
		mockObjs = append(mockObjs, &node)
	}
	namespace := v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "default",
		},
	}
	mockObjs = append(mockObjs, &namespace)

	if !unschedulable {
		dummyPod := v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dummyPod",
				Namespace: "default",
			},
			Spec: v1.PodSpec{
				NodeName: name,
			},
		}
		mockObjs = append(mockObjs, &dummyPod)
	}
	return fake.NewSimpleClientset(mockObjs...)
}

// TestKubeClientWait tests whether KubeClient waits for a node to appear
func TestKubeClientWait(t *testing.T) {
	logrus.SetLevel(logrus.FatalLevel)

	// Check timeout
	fakeKube := mockClientWithNode("test", false, false)

	uut := KubeClient{
		client: fakeKube,
	}
	ctx, _ := context.WithTimeout(context.Background(), 1*time.Second)
	err := uut.WaitForNode(ctx)
	if err == nil {
		t.Fatal("Expected error missing")
	}
	if err.Error() != "context deadline exceeded" {
		t.Fatalf("Unexpected error: '%s'", err)
	}

	// Check normal node
	fakeKube = mockClientWithNode("test", false, true)
	uut = KubeClient{
		client: fakeKube,
	}
	ctx, _ = context.WithTimeout(context.Background(), 1*time.Second)
	err = uut.WaitForNode(ctx)
	if err != nil {
		t.Fatalf("Unexpected error: '%s'", err)
	}

	// Check unschedulable node
	fakeKube = mockClientWithNode("test", true, true)
	uut = KubeClient{
		client: fakeKube,
	}
	ctx, _ = context.WithTimeout(context.Background(), 1*time.Second)
	err = uut.WaitForNode(ctx)
	if err != nil {
		t.Fatalf("Unexpected error: '%s'", err)
	}

	uut.node = ""
	uut.nodeRef = nil
	uut.findNode()
	if uut.nodeRef == nil || uut.nodeRef.Spec.Unschedulable {
		t.Fatal("Node in unexpected state")
	}
}

// TestKubeClientDrain tests whether KubeClient correctly drains a node on shutdown
// Since the mock of individual evictions is incorrect at this point, we only check error codes
func TestKubeClientDrain(t *testing.T) {
	logrus.SetLevel(logrus.FatalLevel)

	fakeKube := mockClientWithNode("test", false, true)
	uut := KubeClient{
		client: fakeKube,
	}
	ctx, _ := context.WithTimeout(context.Background(), 1*time.Second)
	err := uut.WaitForNode(ctx)
	if err != nil {
		t.Fatalf("Unexpected error: '%s'", err)
	}
	err = uut.DrainNode(ctx)
	if err != nil {
		t.Fatalf("Unexpected error: '%s'", err)
	}
}
