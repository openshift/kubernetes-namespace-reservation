# kubernetes-namespace-reservation

An admission webhook that prevents the creation of specified namespaces

## Installation on Kubernetes 1.9+

0. make sure to have at least Kubernetes 1.9, kubectl is working and that jq is installed
1. clone this repo
2. `make build-image push-image REPO=<your-docker-username>/namespace-reservation-server`
3. adapt the namespace-reservation-server image in artifacts/kube-install/apiserver-list.yaml.template to
   your chosen Docker REPO.
4. `hack/install-kube.sh`

Then test the setup:

5. `kubectl create -f artifacts/example/reserve-deads.yaml` will reserve the `deads` namespace.
6. `kubectl create namespace deads` should produce "Error from server (Forbidden): "deads" is reserved"
