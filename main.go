package main

import (
	"context"
	"fmt"
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

type resourceParseResult struct {
	resource string
	name     string
	versions []string
}

type apiResourceMeta struct {
	resources []string
	versions  map[string][]string
	pluralMap map[string]string
}

func main() {
	args := os.Args
	if len(args) < 3 {
		log.Fatalf("insufficient command, use with kubectl autons <command> [<resource-type> <resource>|<resource-type>/<resource>]")
	}

	discoveryClient, dynClient := k8sClient()

	clusterResources := resourcesOrDie(discoveryClient)
	parsedResource := parseResourceOrDie(discoveryClient, args, clusterResources)
	namespaces := namespacesOrDie(dynClient, clusterResources, parsedResource)

	if len(namespaces) > 1 {
		log.Fatalf("Found multiple namespaces for resource, please specify a namespace manually %s", namespaces)
	}

	argsWithNamespace := append(os.Args[1:], "--namespace", namespaces[0])

	cmd := exec.Command("kubectl", argsWithNamespace...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		os.Exit(1)
	}
}

func parseResourceOrDie(discoveryClient *discovery.DiscoveryClient, args []string, clusterResources *apiResourceMeta) *resourceParseResult {
	command := args[1]
	resourceOrPrefixed := args[2]
	fmt.Println(command, resourceOrPrefixed)

	var resource string
	var name string
	var versions []string

	if command == "logs" {
		resource = "pods"
		name = strings.TrimPrefix(strings.TrimPrefix(resourceOrPrefixed, "pod/"), "pods/")
		versions = clusterResources.versions["pods"]
	}
	if command == "port-forward" {
		resource = "pods"
		name = strings.TrimPrefix(strings.TrimPrefix(resourceOrPrefixed, "pod/"), "pods/")
		versions = clusterResources.versions["pods"]
		for _, r := range clusterResources.resources {
			if strings.HasPrefix(resourceOrPrefixed, r+"/") {
				resource = r
				name = strings.TrimPrefix(resourceOrPrefixed, r+"/")
				versions = clusterResources.versions[r]
				break
			}
		}
	} else {
		for _, r := range clusterResources.resources {
			if resourceOrPrefixed == r {
				resource = r
				if len(args) < 4 {
					log.Fatalf("Couldn't parse resource name, make sure its provided in the format <resource-type>/<resource> or <resource-type> <resource>, e.g: kubectl autons get pods <pod-name>")
				}
				name = args[3]
				versions = clusterResources.versions[r]
				break
			}

			if strings.HasPrefix(resourceOrPrefixed, r+"/") {
				resource = r
				name = strings.TrimPrefix(resourceOrPrefixed, r+"/")
				versions = clusterResources.versions[r]
				break
			}
		}
	}

	if resource == "" || name == "" || len(versions) == 0 {
		log.Fatal("Couldn't parse resource/resource-name/resource-version")
	}

	return &resourceParseResult{resource, name, versions}
}

func namespacesOrDie(dynClient *dynamic.DynamicClient, clusterResources *apiResourceMeta, parsedResource *resourceParseResult) []string {
	var namespaces []string

	for _, v := range unique(parsedResource.versions) {
		resourceSchema := schema.GroupVersionResource{Group: "", Version: v, Resource: clusterResources.pluralMap[parsedResource.resource]}
		list, err := dynClient.Resource(resourceSchema).Namespace("").List(context.TODO(), v1.ListOptions{})
		if err != nil {
			log.Fatalf("Error finding resources: %s", err.Error())
		}

		for _, d := range list.Items {
			if d.GetName() == parsedResource.name {
				namespaces = append(namespaces, d.GetNamespace())
			}
		}
	}

	if len(namespaces) == 0 {
		log.Fatalf("Couldn't find any namespaces for resource")
	}

	return unique(namespaces)
}

func resourcesOrDie(discoveryClient *discovery.DiscoveryClient) *apiResourceMeta {
	apiGroupList, err := discoveryClient.ServerGroups()
	if err != nil {
		log.Fatalf("error while parsing resources %v", err)
	}

	var resources []string
	var versions = make(map[string][]string)
	var pluralMap = make(map[string]string)

	for _, group := range apiGroupList.Groups {
		for _, version := range group.Versions {
			resourceList, err := discoveryClient.ServerResourcesForGroupVersion(version.GroupVersion)
			if err != nil {
				log.Fatalf("error while parsing resources %v", err)
			}
			for _, resource := range resourceList.APIResources {

				versions[resource.Name] = append(versions[resource.Name], version.GroupVersion)
				versions[resource.SingularName] = append(versions[resource.Name], version.GroupVersion)

				pluralMap[resource.Name] = resource.Name
				pluralMap[resource.SingularName] = resource.Name

				resources = append(resources, resource.Name, resource.SingularName)

				for _, shortName := range resource.ShortNames {
					versions[shortName] = append(versions[resource.Name], version.GroupVersion)
					pluralMap[shortName] = resource.Name
					resources = append(resources, shortName)
				}
			}
		}
	}

	return &apiResourceMeta{resources, versions, pluralMap}
}

func k8sClient() (*discovery.DiscoveryClient, *dynamic.DynamicClient) {
	var kubeconfig string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatalf("couldn't initialize client %v", err)
		panic(err.Error())
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		log.Fatalf("couldn't initialize client %v", err)
	}

	dynClient, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Fatalf("couldn't initialize client %v", err)
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
