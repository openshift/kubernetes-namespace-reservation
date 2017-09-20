all: build
.PHONY: all

build:
	go build -o _output/bin/namespace-reservation-server github.com/openshift/kubernetes-namespace-reservation/cmd/namespacereservationserver
.PHONY: build

clean:
	rm -rf _output
.PHONY: clean

update-deps:
	hack/update-deps.sh
.PHONY: generate
