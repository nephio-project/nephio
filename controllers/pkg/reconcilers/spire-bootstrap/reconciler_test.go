package spirebootstrap

import (
	"testing"

	v1 "k8s.io/api/core/v1"
)

// Tests
func TestCreateSpireAgentConfigMap(t *testing.T) {
	testCases := []struct {
		name          string
		namespace     string
		cluster       string
		serverAddress string
		serverPort    string
		expectError   bool
	}{
		{
			name:          "basic-config",
			namespace:     "spire",
			cluster:       "test-cluster",
			serverAddress: "spire-server",
			serverPort:    "8081",
			expectError:   false,
		},
		{
			name:          "empty-server",
			namespace:     "spire",
			cluster:       "test-cluster",
			serverAddress: "",
			serverPort:    "8081",
			expectError:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			configMap, err := createSpireAgentConfigMap(tc.name, tc.namespace, tc.cluster, tc.serverAddress, tc.serverPort)

			if tc.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if configMap == nil {
				t.Fatal("configMap is nil")
			}
			if configMap.Name != tc.name {
				t.Errorf("expected name %s, got %s", tc.name, configMap.Name)
			}
			if configMap.Namespace != tc.namespace {
				t.Errorf("expected namespace %s, got %s", tc.namespace, configMap.Namespace)
			}
			if _, exists := configMap.Data["agent.conf"]; !exists {
				t.Error("agent.conf not found in configMap data")
			}
		})
	}
}

func TestGetServiceExternalIP(t *testing.T) {
	testCases := []struct {
		name        string
		service     *v1.Service
		expectedIP  string
		expectError bool
	}{
		{
			name: "loadbalancer-with-ip",
			service: &v1.Service{
				Status: v1.ServiceStatus{
					LoadBalancer: v1.LoadBalancerStatus{
						Ingress: []v1.LoadBalancerIngress{
							{IP: "192.168.1.1"},
						},
					},
				},
			},
			expectedIP:  "192.168.1.1",
			expectError: false,
		},
		{
			name: "loadbalancer-with-hostname",
			service: &v1.Service{
				Status: v1.ServiceStatus{
					LoadBalancer: v1.LoadBalancerStatus{
						Ingress: []v1.LoadBalancerIngress{
							{Hostname: "example.com"},
						},
					},
				},
			},
			expectedIP:  "example.com",
			expectError: false,
		},
		{
			name: "external-ip",
			service: &v1.Service{
				Spec: v1.ServiceSpec{
					ExternalIPs: []string{"10.0.0.1"},
				},
			},
			expectedIP:  "10.0.0.1",
			expectError: false,
		},
		{
			name: "no-external-ip",
			service: &v1.Service{
				Spec: v1.ServiceSpec{},
			},
			expectedIP:  "",
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ip, err := getServiceExternalIP(tc.service)

			if tc.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if ip != tc.expectedIP {
				t.Errorf("expected IP %s, got %s", tc.expectedIP, ip)
			}
		})
	}
}
