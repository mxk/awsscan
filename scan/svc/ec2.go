package svc

import (
	"github.com/LuminalHQ/cloudcover/awsscan/scan"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

type ec2Svc struct{ *scan.Ctx }

var _ = scan.Register(ec2.EndpointsID, ec2.New, ec2Svc{},
	[]ec2.DescribeAddressesInput{},
	[]ec2.DescribeAvailabilityZonesInput{},
	[]ec2.DescribeCustomerGatewaysInput{},
	[]ec2.DescribeDhcpOptionsInput{},
	[]ec2.DescribeFlowLogsInput{},
	[]ec2.DescribeInternetGatewaysInput{},
	[]ec2.DescribeNatGatewaysInput{},
	[]ec2.DescribeNetworkAclsInput{},
	[]ec2.DescribeRouteTablesInput{},
	[]ec2.DescribeSecurityGroupsInput{},
	[]ec2.DescribeSubnetsInput{},
	[]ec2.DescribeVolumesInput{},
	[]ec2.DescribeVpcEndpointsInput{},
	[]ec2.DescribeVpcPeeringConnectionsInput{},
	[]ec2.DescribeVpcsInput{},
	[]ec2.DescribeVpnConnectionsInput{},
	[]ec2.DescribeVpnGatewaysInput{},
)

func (ec2Svc) DescribeInstanceAttribute(di *ec2.DescribeInstancesOutput) (q []ec2.DescribeInstanceAttributeInput) {
	// TODO: Copied from cloudassert, but should probably include others
	attrs := [...]ec2.InstanceAttributeName{
		ec2.InstanceAttributeNameUserData,
		ec2.InstanceAttributeNameDisableApiTermination,
		ec2.InstanceAttributeNameInstanceInitiatedShutdownBehavior,
	}
	for i := range di.Reservations {
		r := &di.Reservations[i]
		for j := range r.Instances {
			id := r.Instances[j].InstanceId
			for _, a := range attrs {
				q = append(q, ec2.DescribeInstanceAttributeInput{
					Attribute:  a,
					InstanceId: id,
				})
			}
		}
	}
	return
}

func (s ec2Svc) DescribeInstances() (q []ec2.DescribeInstancesInput) {
	if s.Mode(scan.CloudAssert) {
		// First call without parameters is used for govcloud detection
		q = make([]ec2.DescribeInstancesInput, 2)
		q[1].Filters = []ec2.Filter{{
			Name: aws.String("instance-state-name"),
			Values: []string{
				string(ec2.InstanceStateNamePending),
				string(ec2.InstanceStateNameRunning),
				string(ec2.InstanceStateNameShuttingDown),
				string(ec2.InstanceStateNameStopping),
				string(ec2.InstanceStateNameStopped),
			},
		}}
	} else {
		q = make([]ec2.DescribeInstancesInput, 1)
	}
	return
}

func (s ec2Svc) DescribeNetworkInterfaces() (q []ec2.DescribeNetworkInterfacesInput) {
	if s.Mode(scan.CloudAssert) {
		q = make([]ec2.DescribeNetworkInterfacesInput, 2)
		q[1].Filters = []ec2.Filter{{
			Name:   aws.String("attachment.device-index"),
			Values: []string{"0"},
		}}
	} else {
		q = make([]ec2.DescribeNetworkInterfacesInput, 1)
	}
	return
}

func (ec2Svc) DescribeVpcAttribute(dv *ec2.DescribeVpcsOutput) (q []ec2.DescribeVpcAttributeInput) {
	attrs := [...]ec2.VpcAttributeName{
		ec2.VpcAttributeNameEnableDnsSupport,
		ec2.VpcAttributeNameEnableDnsHostnames,
	}
	for i := range dv.Vpcs {
		id := dv.Vpcs[i].VpcId
		for _, a := range attrs {
			q = append(q, ec2.DescribeVpcAttributeInput{
				Attribute: a,
				VpcId:     id,
			})
		}
	}
	return
}
