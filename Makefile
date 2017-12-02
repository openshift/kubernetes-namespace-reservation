all: build
.PHONY: all

REPO ?=

build:
	go build -o _output/bin/namespace-reservation-server github.com/openshift/kubernetes-namespace-reservation/cmd/namespacereservationserver
.PHONY: build

build-image:
	GOOS=linux go build -o _output/bin/namespace-reservation-server github.com/openshift/kubernetes-namespace-reservation/cmd/namespacereservationserver
	REPO=$(REPO) hack/build-image.sh
.PHONY: build-image

push-image:
	docker push $(REPO):latest

clean:
	rm -rf _output
.PHONY: clean

update-deps:
	hack/update-deps.sh
.PHONY: generate
