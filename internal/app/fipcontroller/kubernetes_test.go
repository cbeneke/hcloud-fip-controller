package fipcontroller

import (
	"context"
	"fmt"
	"github.com/cbeneke/hcloud-fip-controller/internal/pkg/configuration"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"net"
	"reflect"
	"strings"
	"testing"
)

func createTestNode(nodeName string, addressList []v1.NodeAddress, nodeReady v1.ConditionStatus) *v1.Node {
	return &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: nodeName,
		},
		Spec: v1.NodeSpec{},
		Status: v1.NodeStatus{
			Addresses: addressList,
			Conditions: []v1.NodeCondition{
				{
					Type:   v1.NodeReady,
					Status: nodeReady,
				},
			},
		},
	}
}

func TestNodeAddressList(t *testing.T) {
	nodeName := "node-1"
	tests := []struct {
		name        string
		podName     string
		addressType configuration.NodeAddressType
		objects     []runtime.Object
		err         error
		resultList  [][]net.IP
	}{
		{
			name:        "successful get external ip",
			podName:     "fip",
			addressType: configuration.NodeAddressTypeExternal,
			objects: []runtime.Object{
				createTestNode(nodeName, []v1.NodeAddress{
					{
						Type:    v1.NodeExternalIP,
						Address: "1.2.3.4",
					},
				}, v1.ConditionTrue),
				&v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fip",
						Namespace: "fip",
						Labels: map[string]string{
							"foo": "bar",
						},
					},
					Spec: v1.PodSpec{
						NodeName: nodeName,
					},
					Status: v1.PodStatus{
						HostIP: "1.2.3.4",
					},
				},
			},
			resultList: [][]net.IP{
				{
					net.ParseIP("1.2.3.4"),
				},
			},
		},
		{
			name:        "successful get external ip from node",
			addressType: configuration.NodeAddressTypeExternal,
			objects: []runtime.Object{
				createTestNode(nodeName, []v1.NodeAddress{
					{
						Type:    v1.NodeExternalIP,
						Address: "1.2.3.4",
					},
				}, v1.ConditionTrue),
			},
			resultList: [][]net.IP{
				{
					net.ParseIP("1.2.3.4"),
				},
			},
		},
		{
			name:        "fail wrong pod name",
			addressType: configuration.NodeAddressTypeExternal,
			podName:     "fop",
			objects: []runtime.Object{
				&v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fip",
						Namespace: "fip",
					},
				},
			},
			err: fmt.Errorf("could not get information about pod: Could not get pod information: pods \"fop\" not found"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			kubernetesFakeClient := fake.NewSimpleClientset(test.objects...)
			controller := Controller{
				HetznerClient:    nil,
				KubernetesClient: kubernetesFakeClient,
				Configuration: &configuration.Configuration{
					PodName:   test.podName,
					Namespace: "fip",
				},
				Logger: logrus.New(),
			}

			addressList, err := controller.nodeAddressList(context.Background(), test.addressType)

			if !reflect.DeepEqual(test.err, err) {
				t.Fatalf("error should be [%v] but was [%v]", test.err, err)
			}

			if !reflect.DeepEqual(test.resultList, addressList) {
				t.Fatalf("result should be %v but was %v", test.resultList, addressList)
			}
		})
	}
}

func TestIsNodeHealthy(t *testing.T) {
	tests := []struct {
		name      string
		node      v1.Node
		isHealthy bool
	}{
		{
			name:      "test successful",
			node:      *createTestNode("node-1", []v1.NodeAddress{}, v1.ConditionTrue),
			isHealthy: true,
		},
		{
			name:      "test not healthy",
			node:      *createTestNode("node-1", []v1.NodeAddress{}, v1.ConditionFalse),
			isHealthy: false,
		},
		{
			name:      "test unknown",
			node:      *createTestNode("node-1", []v1.NodeAddress{}, v1.ConditionUnknown),
			isHealthy: false,
		},
		{
			name: "test missing condition status",
			node: v1.Node{
				Status: v1.NodeStatus{
					Conditions: []v1.NodeCondition{
						{
							Type: "Foo",
						},
					},
				},
			},
			isHealthy: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			isHealthy := isNodeHealthy(test.node)

			if isHealthy != test.isHealthy {
				t.Fatalf("should be %t but was %t", isHealthy, test.isHealthy)
			}
		})
	}
}

func TestPodLabelSelector(t *testing.T) {
	tests := []struct {
		name             string
		podLabelSelector string
		podName          string
		objects          []runtime.Object
		resultString     []string
		err              bool
	}{
		{
			name:             "test selector already existing",
			podLabelSelector: "foo=bar",
			objects:          []runtime.Object{},
			resultString:     []string{"foo=bar"},
		},
		{
			name:    "test successful find pod",
			podName: "fip",
			objects: []runtime.Object{
				&v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fip",
						Namespace: "fip",
						Labels: map[string]string{
							"foo": "bar",
							"bar": "foo",
						},
					},
				},
			},
			resultString: []string{"foo=bar","bar=foo"},
		},
		{
			name:    "test successful find pod no labels",
			podName: "fip",
			objects: []runtime.Object{
				&v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fip",
						Namespace: "fip",
					},
				},
			},
			resultString: []string{""},
		},
		{
			name:    "test error wrong pod name",
			podName: "fop",
			objects: []runtime.Object{
				&v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fip",
						Namespace: "fip",
					},
				},
			},
			resultString: []string{""},
			err:          true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			controller := Controller{
				KubernetesClient: fake.NewSimpleClientset(test.objects...),
				Configuration: &configuration.Configuration{
					PodLabelSelector: test.podLabelSelector,
					PodName:          test.podName,
					Namespace:        "fip",
				},
				Logger: logrus.New(),
			}

			selector, err := controller.createPodLabelSelector(context.Background())

			if (err != nil) != test.err {
				t.Fatalf("Err should exist? (%t) but was [%v]", test.err, err)
			}

			s1 := strings.Split(selector, ",")

			// result is in random order -> deep reflect not possible
			for _, s := range s1 {
				hasString := false
				for _, split := range test.resultString {
					if s == split {
						hasString = true
					}
				}
				if !hasString {
					t.Fatalf("Selector should be [%s] but was [%s]", test.resultString, s1)
				}
			}
		})
	}
}
