package fipcontroller

import (
	"context"
	"github.com/cbeneke/hcloud-fip-controller/internal/pkg/configuration"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"net"
	"testing"
)

func TestNodeAddress(t *testing.T) {
	nodeName := "node-1"
	tests := []struct {
		name    string
		nodeName string
		addressType configuration.NodeAddressType
		objects []runtime.Object
		err     error
		result  net.IP
	}{
		{
			name: "successful get external ip",
			nodeName: nodeName,
			addressType: configuration.NodeAddressTypeExternal,
			objects: []runtime.Object{
				&v1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: nodeName,
					},
					Spec: v1.NodeSpec{},
					Status: v1.NodeStatus{
						Addresses: []v1.NodeAddress{
							{
								Type:    v1.NodeExternalIP,
								Address: "1.2.3.4",
							},
						},
					},
				},
			},
			result: net.ParseIP("1.2.3.4"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			kubermaticFakeClient := fake.NewSimpleClientset(test.objects...)
			controller := Controller{
				HetznerClient:    nil,
				KubernetesClient: kubermaticFakeClient,
				Configuration:    nil,
				Logger:           logrus.New(),
			}

			address, err := controller.nodeAddress(context.Background(), test.nodeName, test.addressType)

			if err == nil {
				if test.err != nil {
					t.Fatalf("error should be [%v] but was [nil]", test.err)
				}
			} else {
				if test.err == nil {
					t.Fatalf("error should be [nil] but was [%v]", err)
				}
				if err.Error() != test.err.Error() {
					t.Fatalf("error should be [%v] but was [%v]", test.err, err)
				}
			}

			if address == nil {
				if test.result != nil {
					t.Fatalf("result should be [%s] but was [nil]", test.result.String())
				}
			} else {
				if test.result == nil {
					t.Fatalf("result should be [nil] but was [%s]", address.String())
				}
				if !address.Equal(test.result) {
					t.Fatalf("result should be [%s] but was [%s]", test.result.String(), address.String())
				}
			}
		})
	}
}
