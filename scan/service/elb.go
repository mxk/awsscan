package service

import (
	"github.com/LuminalHQ/cloudcover/awsscan/scan"
	"github.com/aws/aws-sdk-go-v2/service/elb"
)

type elbSvc struct{}

func init() { scan.Register(elbSvc{}) }

func (elbSvc) Name() string         { return elb.ServiceName }
func (elbSvc) NewFunc() interface{} { return elb.New }
func (elbSvc) Roots() []interface{} {
	return []interface{}{
		[]*elb.DescribeLoadBalancersInput{nil},
	}
}

func (elbSvc) DescribeTags(dlb *elb.DescribeLoadBalancersOutput) (q []*elb.DescribeTagsInput) {
	// TODO: Add util function for this and elbv2
	names := make([]string, len(dlb.LoadBalancerDescriptions))
	for i := range dlb.LoadBalancerDescriptions {
		names[i] = *dlb.LoadBalancerDescriptions[i].LoadBalancerName
	}
	return append(q, &elb.DescribeTagsInput{LoadBalancerNames: names})
}

func (elbSvc) DescribeLoadBalancerAttributes(dlb *elb.DescribeLoadBalancersOutput) (q []*elb.DescribeLoadBalancerAttributesInput) {
	setForEach(&q, "LoadBalancerName", dlb.LoadBalancerDescriptions, "LoadBalancerName")
	return
}

func (elbSvc) DescribeLoadBalancerPolicies(dlb *elb.DescribeLoadBalancersOutput) (q []*elb.DescribeLoadBalancerPoliciesInput) {
	for i := range dlb.LoadBalancerDescriptions {
		lb := &dlb.LoadBalancerDescriptions[i]
		for j := range lb.ListenerDescriptions {
			pols := lb.ListenerDescriptions[j].PolicyNames
			if len(pols) > 0 {
				q = append(q, &elb.DescribeLoadBalancerPoliciesInput{
					LoadBalancerName: lb.LoadBalancerName,
					PolicyNames:      pols,
				})
			}
		}
	}
	return
}
