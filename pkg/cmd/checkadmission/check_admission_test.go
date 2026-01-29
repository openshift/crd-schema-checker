package checkadmission

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path"
	"testing"
	"time"

	"github.com/openshift/crd-schema-checker/pkg/manifestcomparators"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	clientcmdapi "k8s.io/client-go/tools/clientcmd/api/v1"
	"k8s.io/klog/v2"
)

func TestAdmissionServer(t *testing.T) {
	testIOStreams, _, out, errOut := genericclioptions.NewTestIOStreams()

	tmpDir, err := os.MkdirTemp("", "kubernetes-kube-apiserver")
	if err != nil {
		t.Fatal(fmt.Errorf("failed to create temp dir: %v", err))
	}

	kubeconfig := clientcmdapi.Config{
		Kind:       "Config",
		APIVersion: "v1",
		Clusters: []clientcmdapi.NamedCluster{
			{
				Name: "dead",
				Cluster: clientcmdapi.Cluster{
					Server:                "localhost",
					InsecureSkipTLSVerify: true,
				},
			},
		},
		AuthInfos: []clientcmdapi.NamedAuthInfo{
			{
				Name: "dead",
				AuthInfo: clientcmdapi.AuthInfo{
					Token: "dead",
				},
			},
		},
		Contexts: []clientcmdapi.NamedContext{
			{
				Name: "dead",
				Context: clientcmdapi.Context{
					Cluster:   "dead",
					AuthInfo:  "dead",
					Namespace: "dead",
				},
			},
		},
		CurrentContext: "dead",
		Extensions:     nil,
	}
	yamlBytes, err := json.Marshal(kubeconfig)
	if err != nil {
		t.Fatal(fmt.Errorf("failed to create kubeconfig: %v", err))
	}
	fakeCoreKubeconfig := path.Join(tmpDir, "kubeconfig")
	os.WriteFile(fakeCoreKubeconfig, yamlBytes, 0644)
	if err != nil {
		t.Fatal(fmt.Errorf("failed to write kubeconfig: %v", err))
	}

	stopCh := make(chan struct{})
	errCh := make(chan error, 1)
	tearDown := func() {
		// Closing stopCh is stopping apiserver and cleaning up
		// after itself, including shutting down its storage layer.
		close(stopCh)

		// If the apiserver was started, let's wait for it to  shutdown clearly.
		if errCh != nil {
			err, ok := <-errCh
			if ok && err != nil {
				klog.Errorf("Failed to shutdown test server clearly: %v", err)
			}
		}
		t.Log(out.String())
		t.Log(errOut.String())

		os.RemoveAll(tmpDir)
	}
	defer func() {
		tearDown()
	}()

	admissionServerOptions := NewAdmissionCheckOptions(testIOStreams)
	admissionServerOptions.AdmissionServerOptions.RecommendedOptions.SecureServing.Listener,
		admissionServerOptions.AdmissionServerOptions.RecommendedOptions.SecureServing.BindPort,
		err = createLocalhostListenerOnFreePort()
	if err != nil {
		t.Fatal(fmt.Errorf("failed to create listener: %v", err))
	}
	admissionServerOptions.AdmissionServerOptions.RecommendedOptions.SecureServing.ServerCert.CertDirectory = tmpDir
	admissionServerOptions.AdmissionServerOptions.RecommendedOptions.Authentication = nil
	admissionServerOptions.AdmissionServerOptions.RecommendedOptions.Authorization = nil
	admissionServerOptions.AdmissionServerOptions.RecommendedOptions.Admission = nil
	admissionServerOptions.AdmissionServerOptions.RecommendedOptions.CoreAPI.CoreAPIKubeconfigPath = fakeCoreKubeconfig
	admissionServerOptions.AdmissionServerOptions.RecommendedOptions.Features.EnablePriorityAndFairness = false

	if err := admissionServerOptions.Complete(); err != nil {
		t.Fatal(err)
	}
	if err := admissionServerOptions.Validate([]string{}); err != nil {
		t.Fatal(err)
	}

	go func() {
		errCh <- admissionServerOptions.RunAdmissionServer(stopCh)
	}()

	serverURL := fmt.Sprintf("https://localhost:%d", admissionServerOptions.AdmissionServerOptions.RecommendedOptions.SecureServing.BindPort)
	if err := waitForServerReady(serverURL, 30*time.Second, errCh); err != nil {
		t.Fatal(fmt.Errorf("server failed to become ready: %v", err))
	}

	restConfig := &rest.Config{
		Host:          net.JoinHostPort("localhost", fmt.Sprintf("%d", admissionServerOptions.AdmissionServerOptions.RecommendedOptions.SecureServing.BindPort)),
		ContentConfig: rest.ContentConfig{},
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: true,
		},
		QPS:     1000,
		Burst:   10000,
		Timeout: 10 * time.Second,
		Dial:    nil,
		Proxy:   nil,
	}
	kubeClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		t.Fatal(err)
	}

	tests, err := manifestcomparators.AllTestsInDir("../../manifestcomparators/testdata")
	if err != nil {
		t.Fatal(err)
	}

	admissionTests := []*admissionComparatorTest{}
	for i := range tests {
		admissionTests = append(admissionTests, &admissionComparatorTest{
			restClient:     kubeClient.RESTClient(),
			ComparatorTest: tests[i],
		})
	}

	for _, test := range admissionTests {
		t.Run(test.ComparatorTest.Name, test.Test)
	}
}

func createLocalhostListenerOnFreePort() (net.Listener, int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, 0, err
	}

	// get port
	tcpAddr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		ln.Close()
		return nil, 0, fmt.Errorf("invalid listen address: %q", ln.Addr().String())
	}

	return ln, tcpAddr.Port, nil
}

func waitForServerReady(serverURL string, timeout time.Duration, errCh <-chan error) error {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 1 * time.Second,
	}

	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case err := <-errCh:
			return fmt.Errorf("server failed to start: %w", err)
		case <-ticker.C:
			if time.Now().After(deadline) {
				return fmt.Errorf("server did not become ready within %v", timeout)
			}
			resp, err := client.Get(serverURL + "/readyz")
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					return nil
				}
			}
		}
	}
}
