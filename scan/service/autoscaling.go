package service

import (
	"github.com/LuminalHQ/cloudcover/awsscan/scan"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
)

type autoscalingSvc struct{}

func init() { scan.Register(autoscalingSvc{}) }

func (autoscalingSvc) ID() string           { return autoscaling.EndpointsID }
func (autoscalingSvc) NewFunc() interface{} { return autoscaling.New }
func (autoscalingSvc) Roots() []interface{} {
	return []interface{}{
		[]*autoscaling.DescribeAutoScalingGroupsInput{nil},
		[]*autoscaling.DescribeAutoScalingInstancesInput{nil},
		[]*autoscaling.DescribeLaunchConfigurationsInput{nil},
	}
}

func (autoscalingSvc) DescribePolicies(dasg *autoscaling.DescribeAutoScalingGroupsOutput) (q []*autoscaling.DescribePoliciesInput) {
	setForEach(&q, "AutoScalingGroupName", dasg.AutoScalingGroups, "AutoScalingGroupName")
	return
}
