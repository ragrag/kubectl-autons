package main

import (
	"context"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type targetResource struct {
	name        string
	apiResource *apiResource
}

type apiResource struct {
	apiName  string
	names    []string
	versions []apiResourceVersion
}

type apiResourceVersion struct {
	groupName string
	version   string
}

func main() {
	args := os.Args[1:]

	if len(args) < 2 {
		log.Fatalf("insufficient command, use with kubectl autons <command> [<resource-type> <resource>|<resource-type>/<resource>]")
	}

	runIfNsExists(args)

	discoveryClient, dynClient := k8sClient()

	apiResources := apiResourcesOrDie(discoveryClient)
	targetResource := targetResourceOrDie(discoveryClient, args, apiResources)
	namespaces := namespacesOrDie(dynClient, targetResource)

	if len(namespaces) > 1 {
		log.Fatalf("Found multiple namespaces for resource, please specify a namespace manually %s", namespaces)
	}

	argsWithNamespace := append(args, "--namespace", namespaces[0])

	cmd := exec.Command("kubectl", argsWithNamespace...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		os.Exit(1)
	}
}

func runIfNsExists(args []string) {
	for _, arg := range args {
		if arg == "--namespace" || arg == "-n" {
			cmd := exec.Command("kubectl", args...)

			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Stdin = os.Stdin

			if err := cmd.Run(); err != nil {
				os.Exit(1)
			} else {
				os.Exit(0)
			}
		}
	}
}

func apiResourcesOrDie(discoveryClient *discovery.DiscoveryClient) []apiResource {
	apiGroupList, err := discoveryClient.ServerGroups()
	if err != nil {
		log.Fatalf("error while parsing resources %v", err)
	}

	var apiResources []apiResource
	var apiResourceMp = make(map[string]apiResource)

	for _, group := range apiGroupList.Groups {
		for _, version := range group.Versions {
			resourceList, err := discoveryClient.ServerResourcesForGroupVersion(version.GroupVersion)
			if err != nil {
				log.Fatalf("error while parsing resources %v", err)
			}

			for _, resource := range resourceList.APIResources {
				apiName := resource.Name
				version := apiResourceVersion{groupName: group.Name, version: version.Version}

				var names []string

				names = append(names, resource.Name, resource.SingularName)
				names = append(names, resource.ShortNames...)

				// merge with cluster resources with same api-name
				names = append(apiResourceMp[resource.Name].names, names...)
				versions := append(apiResourceMp[resource.Name].versions, version)

				apiResourceMp[resource.Name] = apiResource{names: unique(names), versions: uniqueVersions(versions), apiName: apiName}
			}
		}
	}

	for _, v := range apiResourceMp {
		apiResources = append(apiResources, v)
	}

	return apiResources
}

func targetResourceOrDie(discoveryClient *discovery.DiscoveryClient, args []string, apiResources []apiResource) *targetResource {
	cmd := args[0]
	cmdResourceName := args[1]

	podName := strings.TrimPrefix(strings.TrimPrefix(cmdResourceName, "pod/"), "pods/")
	podResource := &apiResource{names: []string{"pods"}, apiName: "pods", versions: []apiResourceVersion{{groupName: "", version: "v1"}}}

	if cmd == "logs" {
		return &targetResource{name: podName, apiResource: podResource}
	}
	if cmd == "port-forward" {
		for _, cr := range apiResources {
			for _, name := range cr.names {
				if strings.HasPrefix(cmdResourceName, name+"/") {
					return &targetResource{name: strings.TrimPrefix(cmdResourceName, name+"/"), apiResource: &cr}
				}
			}
		}
		return &targetResource{name: podName, apiResource: podResource}
	} else {
		for _, cr := range apiResources {
			for _, name := range cr.names {
				if cmdResourceName == name {
					if len(args) < 3 {
						log.Fatalf("Couldn't parse resource name, make sure its provided in the format <resource-type>/<resource> or <resource-type> <resource>, e.g: kubectl autons get pods <pod-name>")
					}
					return &targetResource{name: args[2], apiResource: &cr}
				}

				if strings.HasPrefix(cmdResourceName, name+"/") {
					return &targetResource{name: strings.TrimPrefix(cmdResourceName, name+"/"), apiResource: &cr}
				}
			}
		}
	}

	log.Fatal("Couldn't parse resource/resource-name/resource-version")

	return nil
}

func namespacesOrDie(dynClient *dynamic.DynamicClient, resource *targetResource) []string {
	var namespaces []string

	for _, v := range resource.apiResource.versions {
		resourceSchema := schema.GroupVersionResource{Group: v.groupName, Version: v.version, Resource: resource.apiResource.apiName}
		resourceList, err := dynClient.Resource(resourceSchema).Namespace(v1.NamespaceAll).List(context.TODO(), v1.ListOptions{})

		if err != nil {
			log.Fatalf("Error finding resources: %s", err.Error())
		}

		for _, res := range resourceList.Items {
			if res.GetName() == resource.name {
				namespaces = append(namespaces, res.GetNamespace())
			}
		}
	}

	if len(namespaces) == 0 {
		log.Fatalf("Couldn't find any namespaces for resource")
	}

	return unique(namespaces)
}

func k8sClient() (*discovery.DiscoveryClient, *dynamic.DynamicClient) {
	var kubeconfig string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatalf("couldn't initialize client from kubeconfig %v", err)
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		log.Fatalf("couldn't initialize discovery client %v", err)
	}

	dynClient, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Fatalf("couldn't initialize dynamic client %v", err)
	}

	return discoveryClient, dynClient
}

func unique(s []string) []string {
	mp := make(map[string]bool)
	var list []string
	for _, entry := range s {
		v := mp[entry]
		if !v {
			mp[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func uniqueVersions(resourceVersions []apiResourceVersion) []apiResourceVersion {
	mp := make(map[string]bool)
	var list []apiResourceVersion
	for _, entry := range resourceVersions {
		v := mp[entry.groupName+"/"+entry.version]
		if !v {
			mp[entry.groupName+"/"+entry.version] = true
			list = append(list, entry)
		}
	}
	return list
}
