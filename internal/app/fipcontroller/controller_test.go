package fipcontroller

import (
	"context"
	"encoding/json"
	"github.com/cbeneke/hcloud-fip-controller/internal/pkg/configuration"
	"github.com/hetznercloud/hcloud-go/hcloud/schema"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/fake"
	"net/http"
	"testing"
)

func TestUpdateFloatingIPs(t *testing.T) {
	faultyServer := 2
	tests := []struct {
		name string
		serverIPs []schema.FloatingIP
		servers []schema.Server
		objects []runtime.Object
	}{
		{
			name: "successful assign",
			serverIPs: []schema.FloatingIP{
				{
					ID: 1,
					Type: "ipv4",
					IP: "1.2.3.4",
				},
			},
			servers: []schema.Server{
				{
					ID: 1,
					Name: "server-1",
					PublicNet: schema.ServerPublicNet{
						IPv4: schema.ServerPublicNetIPv4{
							IP: "1.2.3.4",
						},
					},
				},
			},
			objects: []runtime.Object{
				createTestNode("node-1", []v1.NodeAddress{
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
					Status: v1.PodStatus{
						HostIP: "1.2.3.4",
					},
				},
			},
		},
		{
			name: "successful re-assign from faulty node",
			serverIPs: []schema.FloatingIP{
				{
					ID:     1,
					IP:     "1.2.3.4",
					Type:   "ipv4",
					Server: &faultyServer,
				},
			},
			servers: []schema.Server{
				{
					ID: 1,
					Name: "server-1",
					PublicNet: schema.ServerPublicNet{
						IPv4: schema.ServerPublicNetIPv4{
							IP: "1.1.1.1",
						},
					},
				},
				{
					ID: 2,
					Name: "server-2",
					PublicNet: schema.ServerPublicNet{
						IPv4: schema.ServerPublicNetIPv4{
							IP: "1.2.3.4",
						},
					},
				},
			},
			objects: []runtime.Object{
				createTestNode("node-1", []v1.NodeAddress{
					{
						Type:    v1.NodeExternalIP,
						Address: "1.1.1.1",
					},
				}, v1.ConditionTrue),
				createTestNode("node-2", []v1.NodeAddress{
					{
						Type:    v1.NodeExternalIP,
						Address: "1.2.3.4",
					},
				}, v1.ConditionFalse),
				&v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fip-abcde",
						Namespace: "fip",
						Labels: map[string]string{
							"foo": "bar",
						},
					},
					Status: v1.PodStatus{
						HostIP: "1.1.1.1",
					},
				},
				&v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fip-bcdef",
						Namespace: "fip",
						Labels: map[string]string{
							"foo": "bar",
						},
					},
					Status: v1.PodStatus{
						HostIP: "1.2.3.4",
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testEnv := newTestEnv()
			defer testEnv.Teardown()

			testEnv.Mux.HandleFunc("/floating_ips", func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode(schema.FloatingIPListResponse{
					FloatingIPs: test.serverIPs,
				})
			})

			testEnv.Mux.HandleFunc("/servers", func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode(schema.ServerListResponse{
					Servers: test.servers,
				})
			})

			testEnv.Mux.HandleFunc("/floating_ips/1/actions/assign", func(w http.ResponseWriter, r *http.Request) {
				var reqBody schema.FloatingIPActionAssignRequest
				if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
					t.Fatal(err)
				}
				if reqBody.Server != 1 {
					t.Errorf("unexpected server ID: %d", reqBody.Server)
				}
				w.WriteHeader(201)
				json.NewEncoder(w).Encode(schema.FloatingIPActionAssignResponse{
					Action: schema.Action{
						ID: 1,
						Status: r.URL.Query().Get(":id"),
					},
				})
			})

			kubernetesFakeClient := fake.NewSimpleClientset(test.objects...)

			controller := Controller{
				HetznerClient:    testEnv.Client,
				KubernetesClient: kubernetesFakeClient,
				Backoff: wait.Backoff{
					Steps: 1,
				},
				Configuration:    &configuration.Configuration{},
				Logger:           logrus.New(),
			}

			err := controller.UpdateFloatingIPs(context.Background())

			if err != nil {
				t.Fatalf("Err should be [nil] but was %v", err)
			}
		})
	}
}
