package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"

	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/resolution/common"
	"github.com/tektoncd/pipeline/pkg/resolution/resolver/framework"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"knative.dev/pkg/injection/sharedmain"
)

type HTTPResolver struct{}

func main() {
	log.Println("Starting Tekton Resolver...")
	sharedmain.Main("controller",
		framework.NewController(context.Background(), &HTTPResolver{}),
	)
	log.Println("Tekton Resolver started.")

}

// Initialize initializes the resolver.
func (r *HTTPResolver) Initialize(ctx context.Context) error {
	// Add initialization logic here if needed
	return nil
}

// GetName returns the name of the resolver.
func (r *HTTPResolver) GetName(context.Context) string {
	return "HTTPResolver"
}

// GetSelector returns a label selector for the resolver.
func (r *HTTPResolver) GetSelector(context.Context) map[string]string {
	return map[string]string{
		common.LabelKeyResolverType: "http",
	}
}

// ValidateParams validates the parameters passed to the resolver.
func (r *HTTPResolver) ValidateParams(ctx context.Context, params []pipelinev1.Param) error {
	// Add validation logic here if needed
	if len(params) > 0 {
		return errors.New("no params allowed")
	}
	return nil
}

// Resolve fetches CRDs from HTTP URLs.
func (r *HTTPResolver) Resolve(ctx context.Context, params []pipelinev1.Param) (framework.ResolvedResource, error) {
	urls := []string{
		// "https://a364-197-15-20-39.ngrok-free.app/pipelines/tasks",
		// "https://a364-197-15-20-39.ngrok-free.app/pipelines/pipeline",
		// "https://f3f8-197-15-81-207.ngrok-free.app/pipelines/pipeline",
		"https://895d-2a02-8071-184-9a60-1c2-31ca-f8c5-9a0f.ngrok-free.app/pipelines/tasks",
	}

	var crds []byte
	for _, url := range urls {
		err := applyYAMLFiles(url)
		if err != nil {
			log.Fatalf("Error applying YAML files: %v", err)
		}
		crd, err := fetchCRD(url)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch CRD from %s: %v", url, err)
		}
		crds = append(crds, crd...)
	}

	// Return the resolved resource containing the CRDs data
	return &CRDResource{data: crds}, nil
}

// CRDResource represents a resolved CRD fetched from an HTTP URL.
type CRDResource struct {
	data []byte
}

// Data returns the CRD data.
func (c *CRDResource) Data() []byte {
	return c.data
}

// Annotations returns any metadata needed alongside the CRD data.
func (c *CRDResource) Annotations() map[string]string {
	return nil
}

// RefSource returns the source reference of the remote data.
func (c *CRDResource) RefSource() *pipelinev1.RefSource {
	return nil
}

// // fetchCRD fetches the CRD from the given URL.
// func fetchCRD(url string) ([]byte, error) {
// 	resp, err := http.Get(url)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer resp.Body.Close()

// 	body, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		return nil, err
// 	}

//		return body, nil
//	}
func fetchCRD(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Error fetching CRD from %s: %v", url, err)
		return nil, err
	}
	defer resp.Body.Close()

	// Check for expected content type
	if contentType := resp.Header.Get("Content-Type"); contentType != "application/json;charset=UTF-8" {
		log.Printf("Unexpected content type received from %s: %s", url, contentType)
		return nil, fmt.Errorf("unexpected content type: %s", contentType)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body from %s: %v", url, err)
		return nil, err
	}

	return body, nil
}

// func applyYAMLFiles(url string) error {
// 	cmd := exec.Command("kubectl", "apply", "-f", url)
// 	output, err := cmd.CombinedOutput()
// 	if err != nil {
// 		return err
// 	}

// 	log.Printf("YAML files applied successfully from %s:\n%s", url, output)
// 	return nil
// }

func applyYAMLFiles(clientset *kubernetes.Clientset, url string) error {
	// 1. Download the YAML data
	data, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download YAML from %s: %v", url, err)
	}
	defer data.Body.Close()

	// 2. Read the YAML data into a byte array
	yamlData, err := io.ReadAll(data.Body)
	if err != nil {
		return fmt.Errorf("failed to read YAML data: %v", err)
	}

	// 3. Decode the YAML data into Kubernetes runtime objects
	decoder := yaml.NewDecoder(bytes.NewReader(yamlData))
	var obj runtime.Object
	for {
		// Decode the next document in the YAML stream
		err := decoder.Decode(&obj)
		if err == io.EOF {
			// Reached the end of the stream
			break
		}
		if err != nil {
			return fmt.Errorf("failed to decode YAML: %v", err)
		}

		// 4. Apply the decoded object to the Kubernetes cluster
		_, err = clientset.RESTClient().Post().
			Namespace("").
			Resource(obj.GetObjectKind().GroupVersionResource().Resource).
			Body(obj).
			Do(context.TODO())
		if err != nil {
			return fmt.Errorf("failed to apply object: %v", err)
		}
	}

	return nil
}
