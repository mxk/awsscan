package svc

import (
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/mxk/awsscan/scan"
	"github.com/mxk/go-terraform/tfx"
)

type autoscalingSvc struct{ *scan.Ctx }

var _ = scan.Register(autoscaling.EndpointsID, autoscaling.New, autoscalingSvc{},
	[]autoscaling.DescribeAutoScalingGroupsInput{},
	[]autoscaling.DescribeAutoScalingInstancesInput{},
	[]autoscaling.DescribeLaunchConfigurationsInput{},
	[]autoscaling.DescribePoliciesInput{},
	[]autoscaling.DescribeScheduledActionsInput{},
)

func (s autoscalingSvc) DescribeLifecycleHooks(dasg *autoscaling.DescribeAutoScalingGroupsOutput) (q []autoscaling.DescribeLifecycleHooksInput) {
	s.Split(&q, "AutoScalingGroupName", dasg.AutoScalingGroups, "AutoScalingGroupName")
	return
}

func (s autoscalingSvc) DescribeNotificationConfigurations(dasg *autoscaling.DescribeAutoScalingGroupsOutput) (q []autoscaling.DescribeNotificationConfigurationsInput) {
	s.Group(&q, "AutoScalingGroupNames", dasg.AutoScalingGroups, "AutoScalingGroupName", 1600)
	if s.Mode(scan.CloudAssert) && len(q) == 0 {
		q = make([]autoscaling.DescribeNotificationConfigurationsInput, 1)
	}
	return
}

//
// Post-processing
//

func (s autoscalingSvc) AutoScalingGroups(out *autoscaling.DescribeAutoScalingGroupsOutput) error {
	return s.ImportResources("aws_autoscaling_group", tfx.AttrGen{
		"id": s.Strings(out.AutoScalingGroups, "AutoScalingGroupName"),
	})
}

func (s autoscalingSvc) LaunchConfigurations(out *autoscaling.DescribeLaunchConfigurationsOutput) error {
	return s.ImportResources("aws_launch_configuration", tfx.AttrGen{
		"id": s.Strings(out.LaunchConfigurations, "LaunchConfigurationName"),
	})
}

func (s autoscalingSvc) LifecycleHooks(out *autoscaling.DescribeLifecycleHooksOutput) error {
	name := s.Strings(out.LifecycleHooks, "LifecycleHookName")
	return s.MakeResources("aws_autoscaling_lifecycle_hook", tfx.AttrGen{
		"id":                     name,
		"autoscaling_group_name": s.Strings(out.LifecycleHooks, "AutoScalingGroupName"),
		"name":                   name,
	})
}

func (s autoscalingSvc) Policies(out *autoscaling.DescribePoliciesOutput) error {
	name := s.Strings(out.ScalingPolicies, "PolicyName")
	return s.MakeResources("aws_autoscaling_policy", tfx.AttrGen{
		"id":                     name,
		"autoscaling_group_name": s.Strings(out.ScalingPolicies, "AutoScalingGroupName"),
		"name":                   name,
	})
}

func (s autoscalingSvc) ScheduledActions(out *autoscaling.DescribeScheduledActionsOutput) error {
	name := s.Strings(out.ScheduledUpdateGroupActions, "ScheduledActionName")
	return s.MakeResources("aws_autoscaling_schedule", tfx.AttrGen{
		"id":                     name,
		"autoscaling_group_name": s.Strings(out.ScheduledUpdateGroupActions, "AutoScalingGroupName"),
		"scheduled_action_name":  name,
	})
}

func (s autoscalingSvc) mockOutput(out interface{}) {
	dasg, ok := out.(*autoscaling.DescribeAutoScalingGroupsOutput)
	if ok && s.Mode(scan.CloudAssert) {
		dasg.AutoScalingGroups = nil
	}
}
