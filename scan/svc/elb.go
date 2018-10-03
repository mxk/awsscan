package svc

import (
	"github.com/LuminalHQ/cloudcover/awsscan/scan"
	"github.com/aws/aws-sdk-go-v2/service/elb"
)

type elbSvc struct{ *scan.Ctx }

var _ = scan.Register(elb.EndpointsID, elb.New, elbSvc{},
	[]elb.DescribeLoadBalancersInput{},
)

func (s elbSvc) DescribeTags(dlb *elb.DescribeLoadBalancersOutput) (q []elb.DescribeTagsInput) {
	s.Group(&q, "LoadBalancerNames", dlb.LoadBalancerDescriptions, "LoadBalancerName", 20)
	return
}

func (s elbSvc) DescribeLoadBalancerAttributes(dlb *elb.DescribeLoadBalancersOutput) (q []elb.DescribeLoadBalancerAttributesInput) {
	s.Split(&q, "LoadBalancerName", dlb.LoadBalancerDescriptions, "LoadBalancerName")
	return
}

func (elbSvc) DescribeLoadBalancerPolicies(dlb *elb.DescribeLoadBalancersOutput) (q []elb.DescribeLoadBalancerPoliciesInput) {
	for i := range dlb.LoadBalancerDescriptions {
		lb := &dlb.LoadBalancerDescriptions[i]
		for j := range lb.ListenerDescriptions {
			pols := lb.ListenerDescriptions[j].PolicyNames
			if len(pols) > 0 {
				q = append(q, elb.DescribeLoadBalancerPoliciesInput{
					LoadBalancerName: lb.LoadBalancerName,
					PolicyNames:      pols,
				})
			}
		}
	}
	return
}
