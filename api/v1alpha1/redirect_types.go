/*
Copyright 2022.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RedirectSpec defines the desired state of Redirect
type RedirectSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Source is the source URL from which the redirection happens
	Source string `json:"source,omitempty"`

	// Target is the destination URL to which the redirection happen
	Target string `json:"target,omitempty"`

	// Code is the URL Code used for the redirection. Default 308
	Code int `json:"code,omitempty"`
}

// RedirectStatus defines the observed state of Redirect
type RedirectStatus struct {
	Target      string   `json:"target,omitempty"`
	IngressName []string `json:"ingressNames,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Redirect is the Schema for the redirects API
type Redirect struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RedirectSpec   `json:"spec,omitempty"`
	Status RedirectStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RedirectList contains a list of Redirect
type RedirectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Redirect `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Redirect{}, &RedirectList{})
}
