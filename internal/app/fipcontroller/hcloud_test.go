package fipcontroller

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cbeneke/hcloud-fip-controller/internal/pkg/configuration"
	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/hetznercloud/hcloud-go/hcloud/schema"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type testEnv struct {
	Server *httptest.Server
	Mux    *http.ServeMux
	Client *hcloud.Client
}

func newTestEnv() testEnv {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	client := hcloud.NewClient(
		hcloud.WithEndpoint(server.URL),
		hcloud.WithToken("token"),
		hcloud.WithBackoffFunc(func(retries int) time.Duration {
			return 0
		}))
	return testEnv{
		Server: server,
		Mux:    mux,
		Client: client,
	}
}

func (t testEnv) Teardown() {
}

func TestFloatingIPs(t *testing.T) {
	tests := []struct{
		name string
		confFloatingIPs []string
		confFloatingIPsLabelSelector string
		serverIPs []schema.FloatingIP
		resultIPs []*hcloud.FloatingIP
	} {
		{
			name: "successful simple case",
			serverIPs: []schema.FloatingIP{
				{
					ID: 1,
					Type: "ipv4",
					IP: "1.2.3.4",
				},
			},
			resultIPs: []*hcloud.FloatingIP{
				{
					ID: 1,
				},
			},
		},
		{
			name: "successful old config",
			confFloatingIPs: []string{
				"1.2.3.4",
			},
			serverIPs: []schema.FloatingIP{
				{
					ID: 1,
					Type: "ipv4",
					IP: "1.2.3.4",
				},
			},
			resultIPs: []*hcloud.FloatingIP{
				{
					ID: 1,
				},
			},
		},
		{
			name: "successful label selector",
			confFloatingIPsLabelSelector: "foob",
			serverIPs: []schema.FloatingIP{
				{
					ID: 1,
					Type: "ipv4",
					IP: "1.2.3.4",
					Labels: map[string]string{
						"foo": "bar",
					},
				},
			},
			resultIPs: []*hcloud.FloatingIP{
				{
					ID: 1,
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

			controller := Controller{
				HetznerClient:    testEnv.Client,
				Backoff: wait.Backoff{
					Steps: 1,
				},
				Configuration:    &configuration.Configuration{
					FloatingIPLabelSelector: test.confFloatingIPsLabelSelector,
					HcloudFloatingIPs: test.confFloatingIPs,
				},
				Logger:           logrus.New(),
			}

			ps, err := controller.getFloatingIPs(context.Background())
			fmt.Println(len(ps))

			if err != nil {
				t.Fatalf("Error should be [nil] but was %v", err)
			}

			if ps != nil {
				if test.resultIPs == nil {
					t.Fatalf("floatingIps should be [nil] but were length %d", len(ps))
				}
			} else {
				if test.resultIPs != nil {
					t.Fatalf("floatingIps should be length %d but was [nil]", len(test.resultIPs))
				}
			}

			for _, ip := range ps {
				hasIp := false
				for _, resIP := range test.resultIPs {
					if resIP.ID == ip.ID {
						hasIp = true
					}
				}
				if !hasIp {
					t.Fatalf("FloatingIPs should be length %d but was length %d", len(test.resultIPs), len(ps))
				}
			}
		})
	}
}

func TestFloatingIp(t *testing.T) {
	tests := []struct {
		name string
		inputIP string
		serverIPs []schema.FloatingIP
		err error
		resultIP *hcloud.FloatingIP
	}{
		{
			name: "test ipv4 successful",
			inputIP: "1.2.3.4",
			serverIPs: []schema.FloatingIP{
				{
					ID:   1,
					Type: "ipv4",
					IP:   "1.2.3.4",
				},
			},
			resultIP: hcloud.FloatingIPFromSchema(schema.FloatingIP{
				ID: 1,
				Type: "ipv4",
				IP: "1.2.3.4",
			}),
		},
		{
			name: "test ipv6 successful",
			inputIP: "2600::",
			serverIPs: []schema.FloatingIP{
				{
					ID:   1,
					Type: "ipv6",
					IP:   "2600::/64",
				},
			},
			resultIP: hcloud.FloatingIPFromSchema(schema.FloatingIP{
				ID: 1,
				Type: "ipv6",
				IP: "2600::",
			}),
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

			controller := Controller{
				HetznerClient:    testEnv.Client,
				Backoff: wait.Backoff{
					Steps: 1,
				},
				Logger:           logrus.New(),
			}

			ip, err := controller.floatingIP(context.Background(), test.inputIP)

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

			if ip == nil {
				if test.resultIP != nil {
					t.Fatalf("result should be [%d] but was [nil]", test.resultIP.ID)
				}
			} else {
				if test.resultIP == nil {
					t.Fatalf("result should be [nil] but was [%d]", ip.ID)
				}
				if ip.ID != test.resultIP.ID  {
					t.Fatalf("result should be [%d] but was [%d]", test.resultIP.ID, ip.ID)
				}
			}
		})
	}
}

func TestServer(t *testing.T) {
	tests := []struct{
		name string
		inputIPS []net.IP
		servers []schema.Server
		resultServers []*hcloud.Server
		err error
	}{
		{
			name: "test public ipv4 success",
			inputIPS: []net.IP{
				net.ParseIP("1.2.3.4"),
			},
			servers: []schema.Server{
				{
					ID: 1,
					PublicNet: schema.ServerPublicNet{
						IPv4: schema.ServerPublicNetIPv4{
							IP: "1.2.3.4",
						},
					},
				},
			},
			resultServers: []*hcloud.Server{
				{
					ID: 1,
				},
			},
		},
		{
			name: "test private ipv4 success",
			inputIPS: []net.IP{
				net.ParseIP("1.2.3.4"),
			},
			servers: []schema.Server{
				{
					ID: 1,
					PrivateNet: []schema.ServerPrivateNet{
						{
							IP: "1.2.3.4",
						},
					},
				},
			},
			resultServers: []*hcloud.Server{
				{
					ID: 1,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testEnv := newTestEnv()
			defer testEnv.Teardown()

			testEnv.Mux.HandleFunc("/servers", func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode(schema.ServerListResponse{
					Servers: test.servers,
				})
			})

			controller := Controller{
				HetznerClient:    testEnv.Client,
				Backoff: wait.Backoff{
					Steps: 1,
				},
				KubernetesClient: nil,
				Configuration:    nil,
				Logger:           logrus.New(),
			}

			servers, err := controller.servers(context.Background(), test.inputIPS)

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

			if servers == nil {
				if test.resultServers != nil {
					t.Fatalf("result should be serverArray with length %d but was [nil]", len(test.resultServers))
				}
			} else {
				if test.resultServers == nil {
					t.Fatalf("result should be [nil] but was serverArray with length %d", len(servers))
				}
			}

			for i, server := range servers {
				if server == nil {
					t.Fatal("[nil] in serverlist not allowed")
				} else {
					if server.ID != test.resultServers[i].ID {
						t.Fatalf("result should be [%d] but was [%d]", test.resultServers[i].ID, server.ID)
					}
				}
			}
		})
	}
}

