# kubernetes-namespace-reservation

An admission webhook that prevents the creation of specified namespaces

## Installation on Kubernetes 1.9+

0. make sure to have at least Kubernetes 1.9, kubectl is working and that jq is installed
1. clone this repo
2. `make build-image push-image REPO=<your-docker-username>/namespace-reservation-server`
3. adapt the namespace-reservation-server image in [artifacts/kube-install/apiserver-list.yaml.template](artifacts/kube-install/apiserver-list.yaml.template)
   to your chosen Docker REPO.
4. `hack/install-kube.sh`, compare [install-kube.sh](hack/install-kube.sh)

Then test the setup:

5. `kubectl create -f artifacts/example/reserve-deads.yaml` will reserve the `deads` namespace, compare [reserve-deads.yaml](artifacts/example/reserve-deads.yaml).
6. `kubectl create namespace deads` should produce "Error from server (Forbidden): "deads" is reserved"

## Topology

The webhook is deployed as DaemonSet `server` in the namespace `openshift-namespace-reservation`. In
a real cluster this is to be restricted to the master nodes. The server pods get a TLS key and cert
injected by the secret `server-serving-cert`, self-signed by a local CA.

In front of the DaemonSet pods is a service named `server` in the same namespace.

The webhook is an API server itself. An APIService object named `v1beta1.admission.online.openshift.io` makes
the API group `v1beta1.admission.online.openshift.io/v1beta1` available within and outside of the cluster via
API aggregation of kube-apiserver. The group can be reached at `/apis/admission.online.openshift.io/v1beta1/namespacereservations`
of the kube-apiserver, i.e. via the `kubernetes.default.svc` service hostname inside the
cluster.

There are numerous advantages to registering the webhook server as an aggregated API:

- allows other kubernetes components to talk to the the admission webhook using the `kubernetes.default.svc` service
- allows other kubernetes components to use their in-cluster credentials to communicate with the webhook
- allows you to test the webhook using kubectl
- allows you to govern access to the webhook using RBAC
- prevents other extension API servers from leaking their service account tokens to the webhook

For more information, see: https://kubernetes.io/blog/2018/01/extensible-admission-is-beta

The admission webhook is registered via a `ValidatingWebhookConfiguration` object. The webhook URL used
for admission requests is https://kubernetes.default.svc/apis/admission.online.openshift.io/v1beta1/namespacereservations,
i.e. the kube-apiserver sends admission requests to itself. They are forwarded by the aggregator proxy code
to the actual webhook service and finally reach the webhook server.

## Trust

- kube-apiserver trusts itself under https://kubernetes.default.svc because the kube-apiserver CA cert
  is part of the `ValidatingWebhookConfiguration` object in the `caBundle` field (`KUBE_CA` in
  [artifacts/kube-install/apiserver-list.yaml.template](artifacts/kube-install/apiserver-list.yaml.template))
- kube-apiserver trusts https://server.openshift-namespace-reservation.svc because the local CA cert
  is part of the APIService object as field `caBundle` (`SERVICE_SERVING_CERT_CA` in
  [artifacts/kube-install/apiserver-list.yaml.template](artifacts/kube-install/apiserver-list.yaml.template)),
  therefore trusted by kube-apiserver and the webhook server answers with its server cert which is signed by
  that local CA.
- the webhook server trusts the admission requests from the kube-apiserver because
  - it finds the `extension-apiserver-authenticationa` ConfigMap in the `kube-system` namespace
  - which includes a CA cert of the kube-apiserver that the request's client cert is signed with.
