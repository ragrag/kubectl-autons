# kubectl-autons

Automatic namespace detection for k8s resources

## Introduction

Are you frustrated by having to specify the namespace for every kubectl command?

Do you wish you could just run `kubectl get pod my-pod` and not worry about specifying the namespace?

Now you can :name_badge:

## Getting started

1. Download the latest release for your platfrom from github releases or build the binary from source with go build
2. Make sure to rename the binary name to `kubectl-autons` and that it is executable
2. Move the binary to a directory in your PATH
3. Run `kubectl autons <command>` and never worry about namespaces again!

## Some Examples

```base
kubectl autons describe pod my-pod
```

```base
kubectl autons describe pod/my-pod
```

```base
kubectl autons logs my-pod
```

```base
kubectl autons port-forward my-pod 8080:8080
```

```base
kubectl autons port-forward svc/my-svc 8080:8080
```

## Caveats

in cases where a resource is found in multiple namespaces, an error is expected and you will have to specify the desired namespace manually
