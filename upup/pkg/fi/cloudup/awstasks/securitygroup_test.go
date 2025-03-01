/*
Copyright 2019 The Kubernetes Authors.

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

package awstasks

import (
	"context"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"k8s.io/kops/cloudmock/aws/mockec2"
	"k8s.io/kops/upup/pkg/fi"
	"k8s.io/kops/upup/pkg/fi/cloudup/awsup"
)

func TestParseRemovalRule(t *testing.T) {
	testNotParse(t, "port 22")
	testNotParse(t, "port22")
	testNotParse(t, "port=a")
	testNotParse(t, "port=22-23")

	testParsesAsPort(t, "port=22", 22, 22)
	testParsesAsPort(t, "port=443", 443, 443)
	testParsesAsPort(t, "port=22:23", 22, 23)
	testParsesAsPort(t, "port=-1", -1, -1)
}

func testNotParse(t *testing.T, rule string) {
	r, err := ParseRemovalRule(rule)
	if err == nil {
		t.Fatalf("expected failure to parse removal rule %q, got %v", rule, r)
	}
}

func testParsesAsPort(t *testing.T, rule string, fromPort int, toPort int) {
	r, err := ParseRemovalRule(rule)
	if err != nil {
		t.Fatalf("unexpected failure to parse rule %q: %v", rule, err)
	}
	portRemovalRule, ok := r.(*PortRemovalRule)
	if !ok {
		t.Fatalf("unexpected rule type for rule %q: %T", r, err)
	}
	if portRemovalRule.FromPort != fromPort {
		t.Fatalf("unexpected fromPort for %q, expecting %d, got %q", rule, fromPort, r)
	}
	if portRemovalRule.ToPort != toPort {
		t.Fatalf("unexpected toPort for %q, expecting %d, got %q", rule, toPort, r)
	}
}

func TestPortRemovalRule(t *testing.T) {
	r := &PortRemovalRule{FromPort: 22, ToPort: 23}
	testMatches(t, r, &ec2.SecurityGroupRule{FromPort: aws.Int64(22), ToPort: aws.Int64(23)})

	testNotMatches(t, r, &ec2.SecurityGroupRule{FromPort: aws.Int64(0), ToPort: aws.Int64(0)})
	testNotMatches(t, r, &ec2.SecurityGroupRule{FromPort: aws.Int64(22), ToPort: aws.Int64(22)})
	testNotMatches(t, r, &ec2.SecurityGroupRule{FromPort: aws.Int64(23), ToPort: aws.Int64(23)})
	testNotMatches(t, r, &ec2.SecurityGroupRule{FromPort: aws.Int64(20), ToPort: aws.Int64(23)})
	testNotMatches(t, r, &ec2.SecurityGroupRule{FromPort: aws.Int64(22), ToPort: aws.Int64(24)})
	testNotMatches(t, r, &ec2.SecurityGroupRule{ToPort: aws.Int64(23)})
	testNotMatches(t, r, &ec2.SecurityGroupRule{FromPort: aws.Int64(22)})
	testNotMatches(t, r, &ec2.SecurityGroupRule{})

	r = &PortRemovalRule{FromPort: -1, ToPort: -1}
	testMatches(t, r, &ec2.SecurityGroupRule{FromPort: aws.Int64(-1), ToPort: aws.Int64(-1)})
}

func TestPortRemovalRule_Zero(t *testing.T) {
	r := &PortRemovalRule{FromPort: 0, ToPort: 0}
	testMatches(t, r, &ec2.SecurityGroupRule{FromPort: aws.Int64(0), ToPort: aws.Int64(0)})

	testNotMatches(t, r, &ec2.SecurityGroupRule{FromPort: aws.Int64(0), ToPort: aws.Int64(20)})
	testNotMatches(t, r, &ec2.SecurityGroupRule{ToPort: aws.Int64(0)})
	testNotMatches(t, r, &ec2.SecurityGroupRule{FromPort: aws.Int64(0)})
	testNotMatches(t, r, &ec2.SecurityGroupRule{})
}

func testMatches(t *testing.T, rule *PortRemovalRule, permission *ec2.SecurityGroupRule) {
	if !rule.Matches(permission) {
		t.Fatalf("rule %q failed to match permission %q", rule, permission)
	}
}

func testNotMatches(t *testing.T, rule *PortRemovalRule, permission *ec2.SecurityGroupRule) {
	if rule.Matches(permission) {
		t.Fatalf("rule %q unexpectedly matched permission %q", rule, permission)
	}
}

func TestSecurityGroupCreate(t *testing.T) {
	ctx := context.TODO()

	cloud := awsup.BuildMockAWSCloud("us-east-1", "abc")
	c := &mockec2.MockEC2{}
	cloud.MockEC2 = c

	// We define a function so we can rebuild the tasks, because we modify in-place when running
	buildTasks := func() map[string]fi.CloudupTask {
		vpc1 := &VPC{
			Name:      s("vpc1"),
			Lifecycle: fi.LifecycleSync,
			CIDR:      s("172.20.0.0/16"),
			Tags:      map[string]string{"Name": "vpc1"},
		}
		sg1 := &SecurityGroup{
			Name:        s("sg1"),
			Lifecycle:   fi.LifecycleSync,
			Description: s("Description"),
			VPC:         vpc1,
			Tags:        map[string]string{"Name": "sg1"},
		}

		return map[string]fi.CloudupTask{
			"sg1":  sg1,
			"vpc1": vpc1,
		}
	}

	{
		allTasks := buildTasks()
		sg1 := allTasks["sg1"].(*SecurityGroup)
		vpc1 := allTasks["vpc1"].(*VPC)

		target := &awsup.AWSAPITarget{
			Cloud: cloud,
		}

		context, err := fi.NewCloudupContext(ctx, target, nil, cloud, nil, nil, nil, allTasks)
		if err != nil {
			t.Fatalf("error building context: %v", err)
		}

		if err := context.RunTasks(testRunTasksOptions); err != nil {
			t.Fatalf("unexpected error during Run: %v", err)
		}

		if fi.ValueOf(sg1.ID) == "" {
			t.Fatalf("ID not set after create")
		}

		if len(c.SecurityGroups) != 1 {
			t.Fatalf("Expected exactly one SecurityGroup; found %v", c.SecurityGroups)
		}

		expected := &ec2.SecurityGroup{
			Description: s("Description"),
			GroupId:     sg1.ID,
			VpcId:       vpc1.ID,
			Tags: []*ec2.Tag{
				{
					Key:   aws.String("Name"),
					Value: aws.String("sg1"),
				},
			},
			GroupName: s("sg1"),
		}
		actual := c.SecurityGroups[*sg1.ID]
		if !reflect.DeepEqual(actual, expected) {
			t.Fatalf("Unexpected SecurityGroup: expected=%v actual=%v", expected, actual)
		}
	}

	{
		allTasks := buildTasks()
		checkNoChanges(t, ctx, cloud, allTasks)
	}
}
