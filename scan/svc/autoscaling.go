package svc

import (
	"github.com/LuminalHQ/cloudcover/awsscan/scan"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
)

type autoscalingSvc struct{ *scan.Ctx }

var _ = scan.Register(autoscaling.EndpointsID, autoscaling.New, autoscalingSvc{},
	[]autoscaling.DescribeAutoScalingGroupsInput{},
	[]autoscaling.DescribeAutoScalingInstancesInput{},
	[]autoscaling.DescribeLaunchConfigurationsInput{},
)

func (s autoscalingSvc) DescribeNotificationConfigurations(dasg *autoscaling.DescribeAutoScalingGroupsOutput) (q []autoscaling.DescribeNotificationConfigurationsInput) {
	s.Group(&q, "AutoScalingGroupNames", dasg.AutoScalingGroups, "AutoScalingGroupName", 1600)
	if s.Mode(scan.CloudAssert) && len(q) == 0 {
		q = make([]autoscaling.DescribeNotificationConfigurationsInput, 1)
	}
	return
}

func (s autoscalingSvc) DescribePolicies(dasg *autoscaling.DescribeAutoScalingGroupsOutput) (q []autoscaling.DescribePoliciesInput) {
	s.Split(&q, "AutoScalingGroupName", dasg.AutoScalingGroups, "AutoScalingGroupName")
	return
}

func (s autoscalingSvc) mockOutput(out interface{}) {
	dasg, ok := out.(*autoscaling.DescribeAutoScalingGroupsOutput)
	if ok && s.Mode(scan.CloudAssert) {
		dasg.AutoScalingGroups = nil
	}
}
