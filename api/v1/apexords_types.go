/*
Copyright 2021 Henry Xie.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ApexOrdsSpec defines the desired state of ApexOrds
type ApexOrdsSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// sys apex ords schema passwords will be genrated randomly and shown in operator logs
	// The CDB name for oracle 19c database
	Dbname string `json:"dbname"`

	// The PDB name as well as the service name
	Dbservice string `json:"dbservice"`

	// The Database listening port,default is 1521
	// +optional
	Dbport string `json:"dbport,omitempty"`

	// Specify the Ords(Oracle Rest Data Service) name
	Ordsname string `json:"ordsname"`

	//Specify to install Apex runtime only,default is false
	// +optional
	Apexruntimeonly bool `json:"apexruntimeonly,omitempty"`
}

// ApexOrdsStatus defines the observed state of ApexOrds
type ApexOrdsStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ApexOrds is the Schema for the apexords API
type ApexOrds struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApexOrdsSpec   `json:"spec,omitempty"`
	Status ApexOrdsStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ApexOrdsList contains a list of ApexOrds
type ApexOrdsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ApexOrds `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ApexOrds{}, &ApexOrdsList{})
}
