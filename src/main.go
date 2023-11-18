package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type targetResource struct {
	kind string
	name string
}

func main() {
	if strings.HasPrefix(filepath.Base(os.Args[0]), "kubectl-") && (len(os.Args) == 1 || os.Args[1] == "--help" || os.Args[1] == "-h") {
		fmt.Println("use with kubectl autons <command> [<resource-type> <resource>|<resource-type>/<resource>]")
		os.Exit(0)
	}

	args := os.Args[1:]

	if len(args) < 2 {
		log.Fatalf("insufficient command, use with kubectl autons <command> [<resource-type> <resource>|<resource-type>/<resource>]")
	}

	runIfNsExists(args)

	targetResource := targetResourceOrDie(args)
	namespaces := namespacesOrDie(targetResource)

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

func targetResourceOrDie(args []string) targetResource {
	cmd := args[0]
	cmdResourceName := args[1]

	if cmd == "logs" {
		return targetResource{name: strings.TrimPrefix(strings.TrimPrefix(cmdResourceName, "/pod"), "pods/"), kind: "pods"}
	}
	if cmd == "port-forward" {
		if strings.Contains(cmdResourceName, "/") {
			splitRes := strings.Split(cmdResourceName, "/")
			if len(splitRes) == 2 {
				return targetResource{name: splitRes[1], kind: splitRes[0]}
			} else {
				log.Fatalf("Couldn't parse resource name with <resource-type>/<resource> definition make sure its provided in the format <resource-type>/<resource>, e.g: kubectl autons port-forward pods/<pod-name>")
			}
		} else {
			return targetResource{name: cmdResourceName, kind: "pods"}
		}
	} else {
		if strings.Contains(cmdResourceName, "/") {
			splitRes := strings.Split(cmdResourceName, "/")
			if len(splitRes) == 2 {
				return targetResource{name: splitRes[1], kind: splitRes[0]}
			} else {
				log.Fatalf("Couldn't parse resource name with <resource-type>/<resource> definition make sure its provided in the format <resource-type>/<resource>, e.g: kubectl autons port-forward pods/<pod-name>")
			}
		} else if len(args) > 2 {
			return targetResource{name: args[2], kind: cmdResourceName}
		} else {
			log.Fatalf("Couldn't parse resource name, make sure its provided in the format <resource-type>/<resource> or <resource-type> <resource>, e.g: kubectl autons port-forward pods/<pod-name>")
		}
	}

	log.Fatal("Couldn't parse resource/resource-name/resource-version")

	return targetResource{}
}

func namespacesOrDie(target targetResource) []string {
	cmd := exec.Command("kubectl", "get", target.kind, "--all-namespaces")

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		log.Fatalf("Error finding namespace for resource %s of kind %s: %s", target.name, target.kind, err.Error())

	}

	var matches []string

	output := out.String()
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, target.name) {
			matches = append(matches, strings.Split(line, " ")[0])
		}
	}

	if len(matches) == 0 {
		log.Fatalf("Couldn't find namespace for resource %s of kind %s", target.name, target.kind)
	}

	return unique(matches)

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
