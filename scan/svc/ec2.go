package svc

import (
	"github.com/LuminalHQ/cloudcover/awsscan/scan"
	"github.com/LuminalHQ/cloudcover/x/tfx"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

type ec2Svc struct{ *scan.Ctx }

var _ = scan.Register(ec2.EndpointsID, ec2.New, ec2Svc{},
	[]ec2.DescribeAddressesInput{},
	[]ec2.DescribeAvailabilityZonesInput{},
	[]ec2.DescribeCustomerGatewaysInput{},
	[]ec2.DescribeDhcpOptionsInput{},
	[]ec2.DescribeEgressOnlyInternetGatewaysInput{},
	[]ec2.DescribeFlowLogsInput{},
	[]ec2.DescribeInternetGatewaysInput{},
	[]ec2.DescribeKeyPairsInput{},
	[]ec2.DescribeLaunchTemplatesInput{},
	[]ec2.DescribeNatGatewaysInput{},
	[]ec2.DescribeNetworkAclsInput{},
	[]ec2.DescribePlacementGroupsInput{},
	[]ec2.DescribeRouteTablesInput{},
	[]ec2.DescribeSecurityGroupsInput{},
	[]ec2.DescribeSpotFleetRequestsInput{},
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

//
// Post-processing
//

func (s ec2Svc) Addresses(out *ec2.DescribeAddressesOutput) error {
	return s.ImportResources("aws_eip", tfx.AttrGen{
		"id": s.Strings(out.Addresses, "AllocationId"),
	})
}

func (s ec2Svc) CustomerGateways(out *ec2.DescribeCustomerGatewaysOutput) error {
	return s.ImportResources("aws_customer_gateway", tfx.AttrGen{
		"id": s.Strings(out.CustomerGateways, "CustomerGatewayId"),
	})
}

func (s ec2Svc) DhcpOptions(out *ec2.DescribeDhcpOptionsOutput) error {
	return s.ImportResources("aws_vpc_dhcp_options", tfx.AttrGen{
		"id": s.Strings(out.DhcpOptions, "DhcpOptionsId"),
	})
}

func (s ec2Svc) EgressOnlyInternetGateways(out *ec2.DescribeEgressOnlyInternetGatewaysOutput) error {
	for i := range out.EgressOnlyInternetGateways {
		id := *out.EgressOnlyInternetGateways[i].EgressOnlyInternetGatewayId
		err := s.MakeResources("aws_egress_only_internet_gateway", tfx.AttrGen{
			"id":     id,
			"vpc_id": s.Strings(out.EgressOnlyInternetGateways[i].Attachments, "VpcId"),
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (s ec2Svc) FlowLogs(out *ec2.DescribeFlowLogsOutput) error {
	return s.ImportResources("aws_flow_log", tfx.AttrGen{
		"id": s.Strings(out.FlowLogs, "FlowLogId"),
	})
}

func (s ec2Svc) Instances(out *ec2.DescribeInstancesOutput) error {
	for i := range out.Reservations {
		err := s.ImportResources("aws_instance", tfx.AttrGen{
			"id": s.Strings(out.Reservations[i].Instances, "InstanceId"),
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (s ec2Svc) InternetGateways(out *ec2.DescribeInternetGatewaysOutput) error {
	return s.ImportResources("aws_internet_gateway", tfx.AttrGen{
		"id": s.Strings(out.InternetGateways, "InternetGatewayId"),
	})
}

func (s ec2Svc) KeyPairs(out *ec2.DescribeKeyPairsOutput) error {
	return s.ImportResources("aws_key_pair", tfx.AttrGen{
		"id": s.Strings(out.KeyPairs, "KeyName"),
	})
}

func (s ec2Svc) LaunchTemplates(out *ec2.DescribeLaunchTemplatesOutput) error {
	return s.ImportResources("aws_launch_template", tfx.AttrGen{
		"id": s.Strings(out.LaunchTemplates, "LaunchTemplateId"),
	})
}

func (s ec2Svc) NatGateways(out *ec2.DescribeNatGatewaysOutput) error {
	return s.ImportResources("aws_nat_gateway", tfx.AttrGen{
		"id": s.Strings(out.NatGateways, "NatGatewayId"),
	})
}

func (s ec2Svc) NetworkAcls(out *ec2.DescribeNetworkAclsOutput) error {
	// Importer makes API calls
	return s.MakeResources("aws_network_acl", tfx.AttrGen{
		"id": s.Strings(out.NetworkAcls, "NetworkAclId"),
	})
}

func (s ec2Svc) NetworkInterfaces(out *ec2.DescribeNetworkInterfacesOutput) error {
	return s.ImportResources("aws_network_interface", tfx.AttrGen{
		"id": s.Strings(out.NetworkInterfaces, "NetworkInterfaceId"),
	})
}

func (s ec2Svc) PlacementGroups(out *ec2.DescribePlacementGroupsOutput) error {
	return s.ImportResources("aws_placement_group", tfx.AttrGen{
		"id": s.Strings(out.PlacementGroups, "GroupName"),
	})
}

func (s ec2Svc) RouteTables(out *ec2.DescribeRouteTablesOutput) error {
	// Importer makes API calls
	err := s.MakeResources("aws_route_table", tfx.AttrGen{
		"id": s.Strings(out.RouteTables, "RouteTableId"),
	})
	if err != nil {
		return err
	}
	for i := range out.RouteTables {
		for j := range out.RouteTables[i].Associations {
			assoc := &out.RouteTables[i].Associations[j]
			if *assoc.Main {
				continue // Ignore aws_main_route_table_association resources
			}
			err := s.MakeResources("aws_route_table_association", tfx.AttrGen{
				"id":             *assoc.RouteTableAssociationId,
				"route_table_id": *assoc.RouteTableId,
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s ec2Svc) SecurityGroups(out *ec2.DescribeSecurityGroupsOutput) error {
	// Importer makes API calls
	return s.MakeResources("aws_security_group", tfx.AttrGen{
		"id": s.Strings(out.SecurityGroups, "GroupId"),
	})
}

func (s ec2Svc) SpotFleetRequests(out *ec2.DescribeSpotFleetRequestsOutput) error {
	return s.MakeResources("aws_spot_fleet_request", tfx.AttrGen{
		"id": s.Strings(out.SpotFleetRequestConfigs, "SpotFleetRequestId"),
	})
}

func (s ec2Svc) Subnets(out *ec2.DescribeSubnetsOutput) error {
	return s.ImportResources("aws_subnet", tfx.AttrGen{
		"id": s.Strings(out.Subnets, "SubnetId"),
	})
}

func (s ec2Svc) Volumes(out *ec2.DescribeVolumesOutput) error {
	return s.ImportResources("aws_ebs_volume", tfx.AttrGen{
		"id": s.Strings(out.Volumes, "VolumeId"),
	})
}

func (s ec2Svc) Vpcs(out *ec2.DescribeVpcsOutput) error {
	// TODO: Skip default options?
	vpc := s.Strings(out.Vpcs, "VpcId")
	opts := s.Strings(out.Vpcs, "DhcpOptionsId")
	return firstError(
		s.ImportResources("aws_vpc", tfx.AttrGen{
			"id": vpc,
		}),
		s.MakeResources("aws_vpc_dhcp_options_association", tfx.AttrGen{
			"id":              func(i int) string { return opts[i] + "-" + vpc[i] },
			"dhcp_options_id": opts,
			"vpc_id":          vpc,
		}),
	)
}

func (s ec2Svc) VpnGateways(out *ec2.DescribeVpnGatewaysOutput) error {
	return s.ImportResources("aws_vpn_gateway", tfx.AttrGen{
		"id": s.Strings(out.VpnGateways, "VpnGatewayId"),
	})
}
