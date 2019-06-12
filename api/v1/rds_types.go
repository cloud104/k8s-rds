/*

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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RdsFinalizer ...
const RdsFinalizer = "rds.k8s.io"

// RdsSpec defines the desired state of Rds
type RdsSpec struct {
	AvailabilityZone      string               `json:"availabilityZone"`
	BackupRetentionPeriod int64                `json:"backupRetentionPeriod,omitempty"`
	Class                 string               `json:"class"`
	CopyTagsToSnapshot    bool                 `json:"copyTagsToSnapshot,omitempty"`
	DBName                string               `json:"dbname"`
	DBParameterGroupName  string               `json:"parameterGroup,omitempty"`
	DBSnapshotIdentifier  string               `json:"snapshotIdentifier"`
	DBSubnetGroupName     string               `json:"subnetGroupName"`
	Engine                string               `json:"engine"`
	EngineVersion         string               `json:"engineVersion"`
	Iops                  int64                `json:"iops,omitempty"`
	MultiAZ               bool                 `json:"multiaz,omitempty"`
	Password              v1.SecretKeySelector `json:"password"`
	PubliclyAccessible    bool                 `json:"publicAccess,omitempty"`
	Size                  int64                `json:"size"`
	StorageEncrypted      bool                 `json:"encrypted,omitempty"`
	StorageType           string               `json:"storageType,omitempty"`
	Tags                  map[string]string    `json:"tags"`
	Username              string               `json:"username"`
	VpcSecurityGroupIds   string               `json:"vpcSecurityGroupIds,omitempty"`
}

// RdsStatus defines the observed state of Rds
type RdsStatus struct {
	State   string `json:"state,omitempty" description:"State of the deploy"`
	Message string `json:"message,omitempty" description:"Detailed message around the state"`
}

// +kubebuilder:object:root=true

// Rds is the Schema for the rds API
type Rds struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RdsSpec   `json:"spec,omitempty"`
	Status RdsStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RdsList contains a list of Rds
type RdsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Rds `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Rds{}, &RdsList{})
}

func (r *Rds) Is(state string) bool {
	return r.Status.State == state
}

func NewStatus(message string, state string) RdsStatus {
	return RdsStatus{
		Message: message,
		State:   state,
	}
}
