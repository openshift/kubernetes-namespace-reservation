/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package apiserver

import (
	"fmt"

	"github.com/openshift/kubernetes-namespace-reservation/pkg/registry/admissionreview"
	admissionv1alpha1 "k8s.io/api/admission/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apimachinery"
	"k8s.io/apimachinery/pkg/apimachinery/announced"
	"k8s.io/apimachinery/pkg/apimachinery/registered"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
)

var (
	groupFactoryRegistry = make(announced.APIGroupFactoryRegistry)
	registry             = registered.NewOrDie("")
	Scheme               = runtime.NewScheme()
	Codecs               = serializer.NewCodecFactory(Scheme)

	AdmissionGroupName           = "admission.online.openshift.io"
	AdmissionsSchemeGroupVersion = schema.GroupVersion{Group: AdmissionGroupName, Version: "v1alpha1"}
)

func init() {
	admissionv1alpha1.AddToScheme(Scheme)

	// we need to add the options to empty v1
	// TODO fix the server code to avoid this
	metav1.AddToGroupVersion(Scheme, schema.GroupVersion{Version: "v1"})

	// TODO: keep the generic API server from wanting this
	unversioned := schema.GroupVersion{Group: "", Version: "v1"}
	Scheme.AddUnversionedTypes(unversioned,
		&metav1.Status{},
		&metav1.APIVersions{},
		&metav1.APIGroupList{},
		&metav1.APIGroup{},
		&metav1.APIResourceList{},
	)
}

type Config struct {
	GenericConfig *genericapiserver.RecommendedConfig
	ExtraConfig   ExtraConfig
}

type ExtraConfig struct {
}

// NamespaceReservationServer contains state for a Kubernetes cluster master/api server.
type NamespaceReservationServer struct {
	GenericAPIServer *genericapiserver.GenericAPIServer
}

type completedConfig struct {
	GenericConfig genericapiserver.CompletedConfig
	ExtraConfig   *ExtraConfig
}

type CompletedConfig struct {
	// Embed a private pointer that cannot be instantiated outside of this package.
	*completedConfig
}

// Complete fills in any fields not set that are required to have valid data. It's mutating the receiver.
func (c *Config) Complete() CompletedConfig {
	completedCfg := completedConfig{
		c.GenericConfig.Complete(),
		&c.ExtraConfig,
	}

	completedCfg.GenericConfig.Version = &version.Info{
		Major: "1",
		Minor: "0",
	}

	return CompletedConfig{&completedCfg}
}

// New returns a new instance of NamespaceReservationServer from the given config.
func (c completedConfig) New() (*NamespaceReservationServer, error) {
	genericServer, err := c.GenericConfig.New("kubernetes-namespace-reservation", genericapiserver.EmptyDelegate) // completion is done in Complete, no need for a second time
	if err != nil {
		return nil, err
	}

	s := &NamespaceReservationServer{
		GenericAPIServer: genericServer,
	}

	accessor := meta.NewAccessor()
	versionInterfaces := &meta.VersionInterfaces{
		ObjectConvertor:  Scheme,
		MetadataAccessor: accessor,
	}
	interfacesFor := func(version schema.GroupVersion) (*meta.VersionInterfaces, error) {
		if version != admissionv1alpha1.SchemeGroupVersion {
			return nil, fmt.Errorf("unexpected version %v", version)
		}
		return versionInterfaces, nil
	}
	restMapper := meta.NewDefaultRESTMapper([]schema.GroupVersion{admissionv1alpha1.SchemeGroupVersion}, interfacesFor)
	restMapper.AddSpecific(
		admissionv1alpha1.SchemeGroupVersion.WithKind("AdmissionReview"),
		AdmissionsSchemeGroupVersion.WithResource("namespacereservations"),
		AdmissionsSchemeGroupVersion.WithResource("namespacereservation"),
		meta.RESTScopeRoot)

	admissionReview := admissionreview.NewREST()
	// TODO we're going to need a later k8s.io/apiserver so that we can get discovery to list a different group version for
	// our endpoint which we'll use to back some custom storage which will consume the AdmissionReview type and give back the correct response
	apiGroupInfo := genericapiserver.APIGroupInfo{
		GroupMeta: apimachinery.GroupMeta{
			GroupVersion:  AdmissionsSchemeGroupVersion,
			GroupVersions: []schema.GroupVersion{AdmissionsSchemeGroupVersion},
			SelfLinker:    runtime.SelfLinker(accessor),
			RESTMapper:    restMapper,
			InterfacesFor: interfacesFor,
			InterfacesByVersion: map[schema.GroupVersion]*meta.VersionInterfaces{
				admissionv1alpha1.SchemeGroupVersion: versionInterfaces,
			},
		},
		VersionedResourcesStorageMap: map[string]map[string]rest.Storage{},
		// TODO unhardcode this.  It was hardcoded before, but we need to re-evaluate
		OptionsExternalVersion: &schema.GroupVersion{Version: "v1"},
		Scheme:                 Scheme,
		ParameterCodec:         metav1.ParameterCodec,
		NegotiatedSerializer:   Codecs,
		SubresourceGroupVersionKind: map[string]schema.GroupVersionKind{
			"namespacereservations": admissionv1alpha1.SchemeGroupVersion.WithKind("AdmissionReview"),
		},
	}
	apiGroupInfo.GroupMeta.GroupVersion = AdmissionsSchemeGroupVersion
	v1alpha1storage := map[string]rest.Storage{
		"namespacereservations": admissionReview,
	}
	apiGroupInfo.VersionedResourcesStorageMap[AdmissionsSchemeGroupVersion.Version] = v1alpha1storage

	if err := s.GenericAPIServer.InstallAPIGroup(&apiGroupInfo); err != nil {
		return nil, err
	}

	return s, nil
}
