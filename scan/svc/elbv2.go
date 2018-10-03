package svc

import (
	"github.com/LuminalHQ/cloudcover/fdb/aws/scan"
	"github.com/aws/aws-sdk-go-v2/service/elbv2"
)

type elbv2Svc struct{ *scan.Ctx }

var _ = scan.Register(elbv2.EndpointsID, elbv2.New, elbv2Svc{},
	[]elbv2.DescribeLoadBalancersInput{},
	[]elbv2.DescribeTargetGroupsInput{},
)

func (s elbv2Svc) DescribeListeners(dlb *elbv2.DescribeLoadBalancersOutput) (q []elbv2.DescribeListenersInput) {
	s.Split(&q, "LoadBalancerArn", dlb.LoadBalancers, "LoadBalancerArn")
	return
}

func (s elbv2Svc) DescribeLoadBalancerAttributes(dlb *elbv2.DescribeLoadBalancersOutput) (q []elbv2.DescribeLoadBalancerAttributesInput) {
	s.Split(&q, "LoadBalancerArn", dlb.LoadBalancers, "LoadBalancerArn")
	return
}

func (s elbv2Svc) DescribeRules(dl *elbv2.DescribeListenersOutput) (q []elbv2.DescribeRulesInput) {
	s.Split(&q, "ListenerArn", dl.Listeners, "ListenerArn")
	return
}

func (s elbv2Svc) DescribeTagsLoadBalancers(dlb *elbv2.DescribeLoadBalancersOutput) (q []elbv2.DescribeTagsInput) {
	s.Group(&q, "ResourceArns", dlb.LoadBalancers, "LoadBalancerArn", -1)
	return
}

func (s elbv2Svc) DescribeTagsTargetGroups(dtg *elbv2.DescribeTargetGroupsOutput) (q []elbv2.DescribeTagsInput) {
	s.Group(&q, "ResourceArns", dtg.TargetGroups, "TargetGroupArn", -1)
	return
}

func (s elbv2Svc) DescribeTargetGroupAttributes(dtg *elbv2.DescribeTargetGroupsOutput) (q []elbv2.DescribeTargetGroupAttributesInput) {
	s.Split(&q, "TargetGroupArn", dtg.TargetGroups, "TargetGroupArn")
	return
}

func (s elbv2Svc) DescribeTargetHealth(dtg *elbv2.DescribeTargetGroupsOutput) (q []elbv2.DescribeTargetHealthInput) {
	s.Split(&q, "TargetGroupArn", dtg.TargetGroups, "TargetGroupArn")
	return
}
