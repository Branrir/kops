/*
Copyright 2017 The Kubernetes Authors.

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

package validation

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/kops/cloudmock/aws/mockec2"

	"k8s.io/kops/upup/pkg/fi"
	"k8s.io/kops/upup/pkg/fi/cloudup/awsup"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kops/pkg/apis/kops"
)

func TestAWSValidateEBSCSIDriver(t *testing.T) {
	grid := []struct {
		Input          kops.ClusterSpec
		ExpectedErrors []string
	}{
		{
			Input: kops.ClusterSpec{
				ExternalCloudControllerManager: &kops.CloudControllerManagerConfig{},
				CloudProvider: kops.CloudProviderSpec{
					AWS: &kops.AWSSpec{
						EBSCSIDriver: &kops.EBSCSIDriverSpec{
							Enabled: fi.PtrTo(false),
						},
					},
				},
			},
			ExpectedErrors: []string{"Forbidden::spec.cloudProvider.aws.ebsCSIDriver.enabled"},
		},
		{
			Input: kops.ClusterSpec{
				ExternalCloudControllerManager: &kops.CloudControllerManagerConfig{},
				CloudProvider: kops.CloudProviderSpec{
					AWS: &kops.AWSSpec{
						EBSCSIDriver: &kops.EBSCSIDriverSpec{
							Enabled: fi.PtrTo(true),
						},
					},
				},
			},
		},
		{
			Input: kops.ClusterSpec{
				ExternalCloudControllerManager: &kops.CloudControllerManagerConfig{},
				CloudProvider: kops.CloudProviderSpec{
					AWS: &kops.AWSSpec{},
				},
				KubeControllerManager: &kops.KubeControllerManagerConfig{
					ExternalCloudVolumePlugin: "aws",
				},
			},
		},
	}
	for _, g := range grid {
		g.Input.KubernetesVersion = "1.21.0"
		cluster := &kops.Cluster{
			Spec: g.Input,
		}
		errs := awsValidateEBSCSIDriver(cluster)

		testErrors(t, g.Input, errs, g.ExpectedErrors)
	}
}

func TestValidateInstanceGroupSpec(t *testing.T) {
	grid := []struct {
		Input          kops.InstanceGroupSpec
		ExpectedErrors []string
	}{
		{
			Input: kops.InstanceGroupSpec{
				AdditionalSecurityGroups: []string{},
			},
		},
		{
			Input: kops.InstanceGroupSpec{
				AdditionalSecurityGroups: []string{"sg-1234abcd"},
			},
		},
		{
			Input: kops.InstanceGroupSpec{
				AdditionalSecurityGroups: []string{"sg-1234abcd", ""},
			},
			ExpectedErrors: []string{"Invalid value::spec.additionalSecurityGroups[1]"},
		},
		{
			Input: kops.InstanceGroupSpec{
				AdditionalSecurityGroups: []string{" ", ""},
			},
			ExpectedErrors: []string{
				"Invalid value::spec.additionalSecurityGroups[0]",
				"Invalid value::spec.additionalSecurityGroups[1]",
			},
		},
		{
			Input: kops.InstanceGroupSpec{
				AdditionalSecurityGroups: []string{"--invalid"},
			},
			ExpectedErrors: []string{"Invalid value::spec.additionalSecurityGroups[0]"},
		},
		{
			Input: kops.InstanceGroupSpec{
				MachineType: "t2.micro",
				Image:       "ami-073c8c0760395aab8",
			},
		},
		{
			Input: kops.InstanceGroupSpec{
				MachineType: "t2.invalidType",
				Image:       "ami-073c8c0760395aab8",
			},
			ExpectedErrors: []string{"Invalid value::test-nodes.spec.machineType"},
		},
		{
			Input: kops.InstanceGroupSpec{
				MachineType: "m4.large",
				Image:       "ami-073c8c0760395aab8",
			},
			ExpectedErrors: []string{},
		},
		{
			Input: kops.InstanceGroupSpec{
				MachineType: "c5.large",
				Image:       "ami-073c8c0760395aab8",
			},
			ExpectedErrors: []string{},
		},
		{
			Input: kops.InstanceGroupSpec{
				MachineType: "a1.large",
				Image:       "ami-073c8c0760395aab8",
			},
			ExpectedErrors: []string{
				"Invalid value::test-nodes.spec.machineType",
			},
		},
		{
			Input: kops.InstanceGroupSpec{
				SpotDurationInMinutes: fi.PtrTo(int64(55)),
			},
			ExpectedErrors: []string{
				"Unsupported value::test-nodes.spec.spotDurationInMinutes",
			},
		},
		{
			Input: kops.InstanceGroupSpec{
				SpotDurationInMinutes: fi.PtrTo(int64(380)),
			},
			ExpectedErrors: []string{
				"Unsupported value::test-nodes.spec.spotDurationInMinutes",
			},
		},
		{
			Input: kops.InstanceGroupSpec{
				SpotDurationInMinutes: fi.PtrTo(int64(125)),
			},
			ExpectedErrors: []string{
				"Unsupported value::test-nodes.spec.spotDurationInMinutes",
			},
		},
		{
			Input: kops.InstanceGroupSpec{
				SpotDurationInMinutes: fi.PtrTo(int64(120)),
			},
			ExpectedErrors: []string{},
		},
		{
			Input: kops.InstanceGroupSpec{
				InstanceInterruptionBehavior: fi.PtrTo("invalidValue"),
			},
			ExpectedErrors: []string{
				"Unsupported value::test-nodes.spec.instanceInterruptionBehavior",
			},
		},
		{
			Input: kops.InstanceGroupSpec{
				InstanceInterruptionBehavior: fi.PtrTo("terminate"),
			},
			ExpectedErrors: []string{},
		},
		{
			Input: kops.InstanceGroupSpec{
				InstanceInterruptionBehavior: fi.PtrTo("hibernate"),
			},
			ExpectedErrors: []string{},
		},
		{
			Input: kops.InstanceGroupSpec{
				InstanceInterruptionBehavior: fi.PtrTo("stop"),
			},
			ExpectedErrors: []string{},
		},
		{
			Input: kops.InstanceGroupSpec{
				MaxInstanceLifetime: &metav1.Duration{Duration: time.Duration(100 * float64(time.Second))},
			},
			ExpectedErrors: []string{
				"Invalid value::test-nodes.spec.maxInstanceLifetime",
			},
		},
		{
			Input: kops.InstanceGroupSpec{
				MaxInstanceLifetime: &metav1.Duration{Duration: time.Duration(86400 * float64(time.Second))},
			},
			ExpectedErrors: []string{},
		},
		{
			Input: kops.InstanceGroupSpec{
				MaxInstanceLifetime: &metav1.Duration{Duration: time.Duration(0 * float64(time.Second))},
			},
			ExpectedErrors: []string{},
		},
	}
	cloud := awsup.BuildMockAWSCloud("us-east-1", "abc")
	mockEC2 := &mockec2.MockEC2{}
	cloud.MockEC2 = mockEC2

	mockEC2.Images = append(mockEC2.Images, &ec2.Image{
		CreationDate:   aws.String("2016-10-21T20:07:19.000Z"),
		ImageId:        aws.String("ami-073c8c0760395aab8"),
		Name:           aws.String("focal"),
		OwnerId:        aws.String(awsup.WellKnownAccountUbuntu),
		RootDeviceName: aws.String("/dev/xvda"),
		Architecture:   aws.String("x86_64"),
	})

	for _, g := range grid {
		ig := &kops.InstanceGroup{
			ObjectMeta: v1.ObjectMeta{
				Name: "test-nodes",
			},
			Spec: g.Input,
		}
		errs := awsValidateInstanceGroup(ig, cloud)

		testErrors(t, g.Input, errs, g.ExpectedErrors)
	}
}

func TestMixedInstancePolicies(t *testing.T) {
	grid := []struct {
		Input          kops.InstanceGroupSpec
		ExpectedErrors []string
	}{
		{
			Input: kops.InstanceGroupSpec{
				MachineType: "m4.large",
				Image:       "ami-073c8c0760395aab8",
				MixedInstancesPolicy: &kops.MixedInstancesPolicySpec{
					Instances: []string{
						"m4.large",
						"t3.medium",
						"c5.large",
					},
				},
			},
			ExpectedErrors: nil,
		},
		{
			Input: kops.InstanceGroupSpec{
				MachineType: "m4.large",
				Image:       "ami-073c8c0760395aab8",
				MixedInstancesPolicy: &kops.MixedInstancesPolicySpec{
					Instances: []string{
						"a1.large",
						"c4.large",
						"c5.large",
					},
				},
			},
			ExpectedErrors: []string{"Invalid value::spec.mixedInstancesPolicy.instances[0]"},
		},
		{
			Input: kops.InstanceGroupSpec{
				MachineType: "g4dn.xlarge",
				Image:       "ami-073c8c0760395aab8",
				MixedInstancesPolicy: &kops.MixedInstancesPolicySpec{
					Instances: []string{
						"g4dn.xlarge",
						"g4ad.16xlarge",
					},
				},
			},
		},
		{
			Input: kops.InstanceGroupSpec{
				MachineType: "g4dn.xlarge",
				Image:       "ami-073c8c0760395aab8",
				MixedInstancesPolicy: &kops.MixedInstancesPolicySpec{
					Instances: []string{
						"g4dn.xlarge",
						"g4ad.16xlarge",
						"c4.xlarge",
					},
				},
			},
			ExpectedErrors: []string{"Forbidden::spec.mixedInstancesPolicy.instances[2]"},
		},
		{
			Input: kops.InstanceGroupSpec{
				MachineType: "m4.large",
				Image:       "ami-073c8c0760395aab8",
				MixedInstancesPolicy: &kops.MixedInstancesPolicySpec{
					Instances: []string{
						"t3.medium",
						"c4.large",
						"c5.large",
					},
					OnDemandAboveBase: fi.PtrTo(int64(231)),
				},
			},
			ExpectedErrors: []string{"Invalid value::spec.mixedInstancesPolicy.onDemandAboveBase"},
		},
	}
	cloud := awsup.BuildMockAWSCloud("us-east-1", "abc")
	mockEC2 := &mockec2.MockEC2{}
	cloud.MockEC2 = mockEC2

	mockEC2.Images = append(mockEC2.Images, &ec2.Image{
		CreationDate:   aws.String("2016-10-21T20:07:19.000Z"),
		ImageId:        aws.String("ami-073c8c0760395aab8"),
		Name:           aws.String("focal"),
		OwnerId:        aws.String(awsup.WellKnownAccountUbuntu),
		RootDeviceName: aws.String("/dev/xvda"),
		Architecture:   aws.String("x86_64"),
	})

	for _, g := range grid {
		ig := &kops.InstanceGroup{
			ObjectMeta: v1.ObjectMeta{
				Name: "test-nodes",
			},
			Spec: g.Input,
		}
		errs := awsValidateInstanceGroup(ig, cloud)

		testErrors(t, g.Input, errs, g.ExpectedErrors)
	}
}

func TestInstanceMetadataOptions(t *testing.T) {
	cloud := awsup.BuildMockAWSCloud("us-east-1", "abc")

	mockEC2 := &mockec2.MockEC2{}
	cloud.MockEC2 = mockEC2

	mockEC2.Images = append(mockEC2.Images, &ec2.Image{
		CreationDate:   aws.String("2016-10-21T20:07:19.000Z"),
		ImageId:        aws.String("ami-073c8c0760395aab8"),
		Name:           aws.String("focal"),
		OwnerId:        aws.String(awsup.WellKnownAccountUbuntu),
		RootDeviceName: aws.String("/dev/xvda"),
		Architecture:   aws.String("x86_64"),
	})

	tests := []struct {
		ig       *kops.InstanceGroup
		expected []string
	}{
		{
			ig: &kops.InstanceGroup{
				ObjectMeta: v1.ObjectMeta{
					Name: "some-ig",
				},
				Spec: kops.InstanceGroupSpec{
					Role: "Node",
					InstanceMetadata: &kops.InstanceMetadataOptions{
						HTTPPutResponseHopLimit: fi.PtrTo(int64(1)),
						HTTPTokens:              fi.PtrTo("abc"),
					},
					MachineType: "t3.medium",
				},
			},
			expected: []string{"Unsupported value::spec.instanceMetadata.httpTokens"},
		},
		{
			ig: &kops.InstanceGroup{
				ObjectMeta: v1.ObjectMeta{
					Name: "some-ig",
				},
				Spec: kops.InstanceGroupSpec{
					Role: "Node",
					InstanceMetadata: &kops.InstanceMetadataOptions{
						HTTPPutResponseHopLimit: fi.PtrTo(int64(-1)),
						HTTPTokens:              fi.PtrTo("required"),
					},
					MachineType: "t3.medium",
				},
			},
			expected: []string{"Invalid value::spec.instanceMetadata.httpPutResponseHopLimit"},
		},
	}

	for _, test := range tests {
		errs := ValidateInstanceGroup(test.ig, cloud, true)
		testErrors(t, test.ig.ObjectMeta.Name, errs, test.expected)
	}
}

func TestLoadBalancerSubnets(t *testing.T) {
	cidr := "10.0.0.0/24"
	tests := []struct {
		lbType         *string
		class          *string
		clusterSubnets []string
		lbSubnets      []kops.LoadBalancerSubnetSpec
		expected       []string
	}{
		{ // valid (no privateIPv4Address, no allocationID)
			clusterSubnets: []string{"a", "b", "c"},
			lbSubnets: []kops.LoadBalancerSubnetSpec{
				{
					Name:               "a",
					PrivateIPv4Address: nil,
					AllocationID:       nil,
				},
				{
					Name:               "b",
					PrivateIPv4Address: nil,
					AllocationID:       nil,
				},
			},
		},
		{ // valid (with privateIPv4Address)
			clusterSubnets: []string{"a", "b", "c"},
			lbSubnets: []kops.LoadBalancerSubnetSpec{
				{
					Name:               "a",
					PrivateIPv4Address: fi.PtrTo("10.0.0.10"),
					AllocationID:       nil,
				},
				{
					Name:               "b",
					PrivateIPv4Address: nil,
					AllocationID:       nil,
				},
			},
		},
		{ // empty subnet name
			clusterSubnets: []string{"a", "b", "c"},
			lbSubnets: []kops.LoadBalancerSubnetSpec{
				{
					Name:               "",
					PrivateIPv4Address: nil,
					AllocationID:       nil,
				},
			},
			expected: []string{"Required value::spec.api.loadBalancer.subnets[0].name"},
		},
		{ // subnet not found
			clusterSubnets: []string{"a", "b", "c"},
			lbSubnets: []kops.LoadBalancerSubnetSpec{
				{
					Name:               "d",
					PrivateIPv4Address: nil,
					AllocationID:       nil,
				},
			},
			expected: []string{"Not found::spec.api.loadBalancer.subnets[0].name"},
		},
		{ // empty privateIPv4Address, no allocationID
			clusterSubnets: []string{"a", "b", "c"},
			lbSubnets: []kops.LoadBalancerSubnetSpec{
				{
					Name:               "a",
					PrivateIPv4Address: fi.PtrTo(""),
					AllocationID:       nil,
				},
			},
			expected: []string{"Required value::spec.api.loadBalancer.subnets[0].privateIPv4Address"},
		},
		{ // empty no privateIPv4Address, with allocationID
			clusterSubnets: []string{"a", "b", "c"},
			lbSubnets: []kops.LoadBalancerSubnetSpec{
				{
					Name:               "a",
					PrivateIPv4Address: nil,
					AllocationID:       fi.PtrTo(""),
				},
			},
			expected: []string{"Required value::spec.api.loadBalancer.subnets[0].allocationID"},
		},
		{ // invalid privateIPv4Address, no allocationID
			clusterSubnets: []string{"a", "b", "c"},
			lbSubnets: []kops.LoadBalancerSubnetSpec{
				{
					Name:               "a",
					PrivateIPv4Address: fi.PtrTo("invalidip"),
					AllocationID:       nil,
				},
			},
			expected: []string{"Invalid value::spec.api.loadBalancer.subnets[0].privateIPv4Address"},
		},
		{ // privateIPv4Address not matching subnet cidr, no allocationID
			clusterSubnets: []string{"a", "b", "c"},
			lbSubnets: []kops.LoadBalancerSubnetSpec{
				{
					Name:               "a",
					PrivateIPv4Address: fi.PtrTo("11.0.0.10"),
					AllocationID:       nil,
				},
			},
			expected: []string{"Invalid value::spec.api.loadBalancer.subnets[0].privateIPv4Address"},
		},
		{ // invalid class - with privateIPv4Address, no allocationID
			class:          fi.PtrTo(string(kops.LoadBalancerClassClassic)),
			clusterSubnets: []string{"a", "b", "c"},
			lbSubnets: []kops.LoadBalancerSubnetSpec{
				{
					Name:               "a",
					PrivateIPv4Address: fi.PtrTo("10.0.0.10"),
					AllocationID:       nil,
				},
			},
			expected: []string{"Forbidden::spec.api.loadBalancer.subnets[0].privateIPv4Address"},
		},
		{ // invalid class - no privateIPv4Address, with allocationID
			class:          fi.PtrTo(string(kops.LoadBalancerClassClassic)),
			clusterSubnets: []string{"a", "b", "c"},
			lbSubnets: []kops.LoadBalancerSubnetSpec{
				{
					Name:               "a",
					PrivateIPv4Address: nil,
					AllocationID:       fi.PtrTo("eipalloc-222ghi789"),
				},
			},
			expected: []string{"Forbidden::spec.api.loadBalancer.subnets[0].allocationID"},
		},
		{ // invalid type external for private IP
			lbType:         fi.PtrTo(string(kops.LoadBalancerTypePublic)),
			clusterSubnets: []string{"a", "b", "c"},
			lbSubnets: []kops.LoadBalancerSubnetSpec{
				{
					Name:               "a",
					PrivateIPv4Address: fi.PtrTo("10.0.0.10"),
					AllocationID:       nil,
				},
			},
			expected: []string{"Forbidden::spec.api.loadBalancer.subnets[0].privateIPv4Address"},
		},
		{ // invalid type Internal for public IP
			lbType:         fi.PtrTo(string(kops.LoadBalancerTypeInternal)),
			clusterSubnets: []string{"a", "b", "c"},
			lbSubnets: []kops.LoadBalancerSubnetSpec{
				{
					Name:               "a",
					PrivateIPv4Address: nil,
					AllocationID:       fi.PtrTo("eipalloc-222ghi789"),
				},
			},
			expected: []string{"Forbidden::spec.api.loadBalancer.subnets[0].allocationID"},
		},
	}

	for _, test := range tests {
		cluster := kops.Cluster{
			Spec: kops.ClusterSpec{
				API: kops.APISpec{
					LoadBalancer: &kops.LoadBalancerAccessSpec{
						Class: kops.LoadBalancerClassNetwork,
						Type:  kops.LoadBalancerTypeInternal,
					},
				},
				CloudProvider: kops.CloudProviderSpec{
					AWS: &kops.AWSSpec{},
				},
			},
		}
		if test.class != nil {
			cluster.Spec.API.LoadBalancer.Class = kops.LoadBalancerClass(*test.class)
		}
		if test.lbType != nil {
			cluster.Spec.API.LoadBalancer.Type = kops.LoadBalancerType(*test.lbType)
		}
		for _, s := range test.clusterSubnets {
			cluster.Spec.Networking.Subnets = append(cluster.Spec.Networking.Subnets, kops.ClusterSubnetSpec{
				Name: s,
				CIDR: cidr,
			})
		}
		cluster.Spec.API.LoadBalancer.Subnets = test.lbSubnets
		errs := awsValidateCluster(&cluster, true)
		testErrors(t, test, errs, test.expected)
	}
}

func TestAWSAuthentication(t *testing.T) {
	tests := []struct {
		backendMode      string
		identityMappings []kops.AWSAuthenticationIdentityMappingSpec
		expected         []string
	}{
		{ // valid
			backendMode: "CRD",
			identityMappings: []kops.AWSAuthenticationIdentityMappingSpec{
				{
					ARN:      "arn:aws:iam::123456789012:role/KopsExampleRole",
					Username: "foo",
				},
				{
					ARN:      "arn:aws:iam::123456789012:user/KopsExampleUser",
					Username: "foo",
				},
			},
		},
		{ // valid, multiple backendModes
			backendMode: "CRD,MountedFile",
			identityMappings: []kops.AWSAuthenticationIdentityMappingSpec{
				{
					ARN:      "arn:aws:iam::123456789012:role/KopsExampleRole",
					Username: "foo",
				},
				{
					ARN:      "arn:aws:iam::123456789012:user/KopsExampleUser",
					Username: "foo",
				},
			},
		},
		{ // forbidden backendMode
			backendMode: "MountedFile",
			identityMappings: []kops.AWSAuthenticationIdentityMappingSpec{
				{
					ARN:      "arn:aws:iam::123456789012:role/KopsExampleRole",
					Username: "foo",
				},
			},
			expected: []string{"Forbidden::spec.authentication.aws.backendMode"},
		},
		{ // invalid identity ARN
			backendMode: "CRD",
			identityMappings: []kops.AWSAuthenticationIdentityMappingSpec{
				{
					ARN:      "arn:aws:iam::123456789012:policy/KopsExampleRole",
					Username: "foo",
				},
			},
			expected: []string{"Invalid value::spec.authentication.aws.identityMappings[0].arn"},
		},
	}

	for _, test := range tests {
		cluster := kops.Cluster{
			Spec: kops.ClusterSpec{
				Authentication: &kops.AuthenticationSpec{
					AWS: &kops.AWSAuthenticationSpec{
						BackendMode:      test.backendMode,
						IdentityMappings: test.identityMappings,
					},
				},
				CloudProvider: kops.CloudProviderSpec{
					AWS: &kops.AWSSpec{},
				},
			},
		}
		errs := awsValidateCluster(&cluster, true)
		testErrors(t, test, errs, test.expected)
	}
}

func TestAWSAdditionalRoutes(t *testing.T) {
	tests := []struct {
		name                   string
		clusterCIDR            string
		additionalClusterCIDRs []string
		providerId             string
		subnetType             kops.SubnetType
		route                  []kops.RouteSpec
		expected               []string
	}{
		{
			name:        "valid pcx",
			clusterCIDR: "100.64.0.0/10",
			subnetType:  kops.SubnetTypePrivate,
			route: []kops.RouteSpec{
				{
					CIDR:   "10.0.0.0/8",
					Target: "pcx-abcdef",
				},
			},
		},
		{
			name:        "valid instance",
			clusterCIDR: "100.64.0.0/10",
			subnetType:  kops.SubnetTypePrivate,
			route: []kops.RouteSpec{
				{
					CIDR:   "10.0.0.0/8",
					Target: "i-abcdef",
				},
			},
		},
		{
			name:        "valid nat",
			clusterCIDR: "100.64.0.0/10",
			subnetType:  kops.SubnetTypePrivate,
			route: []kops.RouteSpec{
				{
					CIDR:   "10.0.0.0/8",
					Target: "nat-abcdef",
				},
			},
		},
		{
			name:        "valid transit gateway",
			clusterCIDR: "100.64.0.0/10",
			subnetType:  kops.SubnetTypePrivate,
			route: []kops.RouteSpec{
				{
					CIDR:   "10.0.0.0/8",
					Target: "tgw-abcdef",
				},
			},
		},
		{
			name:        "valid internet gateway",
			clusterCIDR: "100.64.0.0/10",
			subnetType:  kops.SubnetTypePrivate,
			route: []kops.RouteSpec{
				{
					CIDR:   "10.0.0.0/8",
					Target: "igw-abcdef",
				},
			},
		},
		{
			name:        "valid egress only internet gateway",
			clusterCIDR: "100.64.0.0/10",
			subnetType:  kops.SubnetTypePrivate,
			route: []kops.RouteSpec{
				{
					CIDR:   "10.0.0.0/8",
					Target: "eigw-abcdef",
				},
			},
		},
		{
			name:        "bad cluster cidr",
			clusterCIDR: "not cidr",
			subnetType:  kops.SubnetTypePrivate,
			route: []kops.RouteSpec{
				{
					CIDR:   "10.0.0.0/8",
					Target: "pcx-abcdef",
				},
			},
			expected: []string{"Invalid value::spec.networking.networkCIDR"},
		},
		{
			name:        "bad cidr",
			clusterCIDR: "100.64.0.0/10",
			subnetType:  kops.SubnetTypePrivate,
			route: []kops.RouteSpec{
				{
					CIDR:   "bad cidr",
					Target: "pcx-abcdef",
				},
			},
			expected: []string{"Invalid value::spec.networking.subnets[0].additionalRoutes[0].cidr"},
		},
		{
			name:        "bad target",
			clusterCIDR: "100.64.0.0/10",
			subnetType:  kops.SubnetTypePrivate,
			route: []kops.RouteSpec{
				{
					CIDR:   "10.0.0.0/8",
					Target: "unknown-abcdef",
				},
			},
			expected: []string{"Invalid value::spec.networking.subnets[0].additionalRoutes[0].target"},
		},
		{
			name:        "target more specific",
			clusterCIDR: "100.64.0.0/10",
			subnetType:  kops.SubnetTypePrivate,
			route: []kops.RouteSpec{
				{
					CIDR:   "100.64.1.0/24",
					Target: "pcx-abcdef",
				},
			},
			expected: []string{"Forbidden::spec.networking.subnets[0].additionalRoutes[0].target"},
		},
		{
			name:                   "target more specific additionalCIDR",
			clusterCIDR:            "100.64.0.0/16",
			additionalClusterCIDRs: []string{"100.66.0.0/16", "100.67.0.0/16"},
			subnetType:             kops.SubnetTypePrivate,
			route: []kops.RouteSpec{
				{
					CIDR:   "100.66.2.0/24",
					Target: "pcx-abcdef",
				},
			},
			expected: []string{"Forbidden::spec.networking.subnets[0].additionalRoutes[0].target"},
		},
		{
			name:        "duplicates cidr",
			clusterCIDR: "100.64.0.0/10",
			subnetType:  kops.SubnetTypePrivate,
			route: []kops.RouteSpec{
				{
					CIDR:   "10.0.0.0/8",
					Target: "pcx-abcdef",
				},
				{
					CIDR:   "10.0.0.0/8",
					Target: "tgw-abcdef",
				},
			},
			expected: []string{"Duplicate value::spec.networking.subnets[0].additionalRoutes[1].cidr"},
		},
		{
			name:        "shared subnet",
			clusterCIDR: "100.64.0.0/10",
			subnetType:  kops.SubnetTypePrivate,
			providerId:  "123456",
			route: []kops.RouteSpec{
				{
					CIDR:   "10.0.0.0/8",
					Target: "pcx-abcdef",
				},
			},
			expected: []string{"Forbidden::spec.networking.subnets[0].additionalRoutes"},
		},
		{
			name:        "not a private subnet",
			clusterCIDR: "100.64.0.0/10",
			subnetType:  kops.SubnetTypePublic,
			route: []kops.RouteSpec{
				{
					CIDR:   "10.0.0.0/8",
					Target: "pcx-abcdef",
				},
			},
			expected: []string{"Forbidden::spec.networking.subnets[0].additionalRoutes"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cluster := kops.Cluster{
				Spec: kops.ClusterSpec{
					CloudProvider: kops.CloudProviderSpec{
						AWS: &kops.AWSSpec{},
					},
					Networking: kops.NetworkingSpec{
						NetworkCIDR:            test.clusterCIDR,
						AdditionalNetworkCIDRs: test.additionalClusterCIDRs,
						Subnets: []kops.ClusterSubnetSpec{
							{
								Name:             "us-east-1a",
								ID:               test.providerId,
								Type:             test.subnetType,
								AdditionalRoutes: test.route,
							},
						},
					},
				},
			}
			errs := validateNetworking(&cluster, &cluster.Spec.Networking, field.NewPath("spec", "networking"), false, &cloudProviderConstraints{})
			testErrors(t, test, errs, test.expected)
		})
	}
}
