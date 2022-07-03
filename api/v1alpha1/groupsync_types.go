/*
Copyright 2022 Lars Kellogg-Stedman.

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

// GroupSyncSpec defines the desired state of GroupSync
type (
	GithubToken struct {
		Namespace string `json:"namespace"`
		Secret    string `json:"secret"`
	}

	GroupSyncSpec struct {
		Organization string            `json:"organization"`
		GithubToken  GithubToken       `json:"githubToken"`
		SyncAllTeams bool              `json:"syncAllTeams"`
		Teams        map[string]string `json:"teams"`
	}
)

// GroupSyncStatus defines the observed state of GroupSync
type GroupSyncStatus struct {
	LastSyncTime   string `json:"lastSyncTime"`
	LastSyncStatus string `json:"lastSyncStatus"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// GroupSync is the Schema for the groupsyncs API
type GroupSync struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GroupSyncSpec   `json:"spec,omitempty"`
	Status GroupSyncStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// GroupSyncList contains a list of GroupSync
type GroupSyncList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GroupSync `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GroupSync{}, &GroupSyncList{})
}