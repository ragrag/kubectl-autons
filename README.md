# kubectl-autons

Automatic namespace detection for kubectl commands

## Introduction

Are you frustrated by having to specify the namespace for every kubectl command?

Do you wish you could just run commands like `kubectl get pod my-pod` and not worry about specifying the namespace?

Now you can 📬

## Getting started
### Installation with [Krew](https://krew.sigs.k8s.io/)
1. Make sure you have [Krew Installed](https://krew.sigs.k8s.io/docs/user-guide/setup/install/)
1. Run ```kubectl krew install autons```
2. You're ready to go! Run `kubectl autons <command>` and never worry about namespaces again!

### Manual Installation
1. Download the latest release for your platfrom from github releases or build the binary from source with go build
2. Make sure to rename the binary name to `kubectl-autons` and that it is executable
3. Move the binary to a directory in your PATH
4. You're ready to go! Run `kubectl autons <command>` and never worry about namespaces again!

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
