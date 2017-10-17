# kubernetes-namespace-reservation
An admission webhook that prevents the creation of specified namespaces

## Installation
1. with latest `oc`: ` oc cluster up --version=latest --loglevel=1`, then pay attention, you have a small window.  You will see a message scroll across `I1016 11:32:41.564270    4498 helper.go:585] Copying OpenShift config to local directory /tmp/openshift-config506467289`.  You need to save that temp dir!  `cp -r /tmp/openshift-config506467289 /tmp/foo`
2. clone this repo
3. `make build-image`
4. `oc create namespace openshift-namespace-reservation`
5. `oc process -f artifacts/install/rbac-template.yaml | oc auth reconcile -f -`
6. ```oc process -f artifacts/install/apiserver-template.yaml -p "SERVICE_SERVING_CERT_CA=`cat '/tmp/foo/master/service-signer.crt' | base64``" | oc apply -f -```
7. `oc create -f  artifacts/example/reserve-deads.yaml` will reserve the `deads` namespace.
8. `oc new-project deads` should produce "Error from server (Forbidden): "deads" is reserved"
