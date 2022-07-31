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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ShortLinkSpec defines the desired state of ShortLink
type ShortLinkSpec struct {
	// Alias is the short name (vanity name) of the shortening. If omitted, a random alias will be choosen
	Alias string `json:"alias,omitempty"`

	// Target specifies the target to which we will redirect
	Target string `json:"target,omitempty"`
}

// ShortLinkStatus defines the observed state of ShortLink
type ShortLinkStatus struct {
	// Count represents the amount of time, this ShortLink has been called
	Count int `json:"count,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ShortLink is the Schema for the shortlinks API
type ShortLink struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ShortLinkSpec   `json:"spec,omitempty"`
	Status ShortLinkStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ShortLinkList contains a list of ShortLink
type ShortLinkList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ShortLink `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ShortLink{}, &ShortLinkList{})
}
