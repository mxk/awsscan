package service

import (
	"github.com/LuminalHQ/cloudcover/awsscan/scan"
	"github.com/aws/aws-sdk-go-v2/service/elbv2"
)

type elbv2Svc struct{}

func init() { scan.Register(elbv2Svc{}) }

func (elbv2Svc) ID() string           { return elbv2.EndpointsID }
func (elbv2Svc) NewFunc() interface{} { return elbv2.New }
func (elbv2Svc) Roots() []interface{} {
	return []interface{}{
		[]*elbv2.DescribeLoadBalancersInput{nil},
		[]*elbv2.DescribeTargetGroupsInput{nil},
	}
}

func (elbv2Svc) DescribeListeners(dlb *elbv2.DescribeLoadBalancersOutput) (q []*elbv2.DescribeListenersInput) {
	setForEach(&q, "LoadBalancerArn", dlb.LoadBalancers, "LoadBalancerArn")
	return
}

func (elbv2Svc) DescribeLoadBalancerAttributes(dlb *elbv2.DescribeLoadBalancersOutput) (q []*elbv2.DescribeLoadBalancerAttributesInput) {
	setForEach(&q, "LoadBalancerArn", dlb.LoadBalancers, "LoadBalancerArn")
	return
}

func (elbv2Svc) DescribeRules(dl *elbv2.DescribeListenersOutput) (q []*elbv2.DescribeRulesInput) {
	setForEach(&q, "ListenerArn", dl.Listeners, "ListenerArn")
	return
}

// TODO: Need DescribeTags for TargetGroups

func (elbv2Svc) DescribeTags(dlb *elbv2.DescribeLoadBalancersOutput) (q []*elbv2.DescribeTagsInput) {
	arns := make([]string, len(dlb.LoadBalancers))
	for i := range dlb.LoadBalancers {
		arns[i] = *dlb.LoadBalancers[i].LoadBalancerArn
	}
	return append(q, &elbv2.DescribeTagsInput{ResourceArns: arns})
}

func (elbv2Svc) DescribeTargetGroupAttributes(dtg *elbv2.DescribeTargetGroupsOutput) (q []*elbv2.DescribeTargetGroupAttributesInput) {
	setForEach(&q, "TargetGroupArn", dtg.TargetGroups, "TargetGroupArn")
	return
}

func (elbv2Svc) DescribeTargetHealth(dtg *elbv2.DescribeTargetGroupsOutput) (q []*elbv2.DescribeTargetHealthInput) {
	setForEach(&q, "TargetGroupArn", dtg.TargetGroups, "TargetGroupArn")
	return
}
