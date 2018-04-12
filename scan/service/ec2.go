package service

import (
	"github.com/LuminalHQ/cloudcover/awsscan/scan"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

type ec2Svc struct{}

func init() { scan.Register(ec2Svc{}) }

func (ec2Svc) Name() string         { return ec2.ServiceName }
func (ec2Svc) NewFunc() interface{} { return ec2.New }
func (ec2Svc) Roots() []interface{} {
	return []interface{}{
		[]*ec2.DescribeAddressesInput{nil},
		[]*ec2.DescribeAvailabilityZonesInput{nil},
		[]*ec2.DescribeCustomerGatewaysInput{nil},
		[]*ec2.DescribeDhcpOptionsInput{nil},
		[]*ec2.DescribeInstancesInput{nil},
		[]*ec2.DescribeInternetGatewaysInput{nil},
		[]*ec2.DescribeNatGatewaysInput{nil},
		[]*ec2.DescribeNetworkAclsInput{nil},
		[]*ec2.DescribeNetworkInterfacesInput{nil},
		[]*ec2.DescribeRouteTablesInput{nil},
		[]*ec2.DescribeSecurityGroupsInput{nil},
		[]*ec2.DescribeSubnetsInput{nil},
		[]*ec2.DescribeVolumesInput{nil},
		[]*ec2.DescribeVpcEndpointsInput{nil},
		[]*ec2.DescribeVpcPeeringConnectionsInput{nil},
		[]*ec2.DescribeVpcsInput{nil},
		[]*ec2.DescribeVpnConnectionsInput{nil},
		[]*ec2.DescribeVpnGatewaysInput{nil},
	}
}

func (ec2Svc) DescribeInstanceAttribute(di *ec2.DescribeInstancesOutput) (q []*ec2.DescribeInstanceAttributeInput) {
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
				q = append(q, &ec2.DescribeInstanceAttributeInput{
					Attribute:  a,
					InstanceId: id,
				})
			}
		}
	}
	return
}

func (ec2Svc) DescribeVpcAttribute(dv *ec2.DescribeVpcsOutput) (q []*ec2.DescribeVpcAttributeInput) {
	attrs := [...]ec2.VpcAttributeName{
		ec2.VpcAttributeNameEnableDnsSupport,
		ec2.VpcAttributeNameEnableDnsHostnames,
	}
	for i := range dv.Vpcs {
		id := dv.Vpcs[i].VpcId
		for _, a := range attrs {
			q = append(q, &ec2.DescribeVpcAttributeInput{
				Attribute: a,
				VpcId:     id,
			})
		}
	}
	return
}
