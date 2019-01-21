package svc

import (
	"strconv"

	"github.com/aws/aws-sdk-go-v2/service/elb"
	"github.com/mxk/cloudcover/awsscan/scan"
	"github.com/mxk/cloudcover/x/tfx"
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

func (s elbSvc) DescribeLoadBalancerPolicies(dlb *elb.DescribeLoadBalancersOutput) (q []elb.DescribeLoadBalancerPoliciesInput) {
	if !s.Mode(scan.CloudAssert) {
		s.Split(&q, "LoadBalancerName", dlb.LoadBalancerDescriptions, "LoadBalancerName")
		return
	}
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

//
// Post-processing
//

func (s elbSvc) LoadBalancerPolicies(out *elb.DescribeLoadBalancerPoliciesOutput) error {
	name := *s.Input(out).(*elb.DescribeLoadBalancerPoliciesInput).LoadBalancerName
	return s.MakeResources("aws_load_balancer_policy", tfx.AttrGen{
		"#":  len(out.PolicyDescriptions),
		"id": func(i int) string { return name + ":" + *out.PolicyDescriptions[i].PolicyName },
	})
}

func (s elbSvc) LoadBalancers(out *elb.DescribeLoadBalancersOutput) error {
	for i := range out.LoadBalancerDescriptions {
		lb := &out.LoadBalancerDescriptions[i]
		name := *lb.LoadBalancerName
		err := firstError(
			s.ImportResources("aws_elb", tfx.AttrGen{"id": name}),
			s.backendServerPolicies(name, lb.BackendServerDescriptions),
			s.listenerPolicies(name, lb.ListenerDescriptions),
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s elbSvc) backendServerPolicies(name string, out []elb.BackendServerDescription) error {
	return s.MakeResources("aws_load_balancer_backend_server_policy", tfx.AttrGen{
		"#":  len(out),
		"id": func(i int) string { return elbPort(name, out[i].InstancePort) },
	})
}

func (s elbSvc) listenerPolicies(name string, out []elb.ListenerDescription) error {
	return s.MakeResources("aws_load_balancer_listener_policy", tfx.AttrGen{
		"#":  len(out),
		"id": func(i int) string { return elbPort(name, out[i].Listener.LoadBalancerPort) },
	})
}

func elbPort(name string, port *int64) string {
	return name + ":" + strconv.FormatInt(*port, 10)
}
