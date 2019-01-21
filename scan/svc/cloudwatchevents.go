package svc

import (
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchevents"
	"github.com/mxk/cloudcover/awsscan/scan"
	"github.com/mxk/cloudcover/x/tfx"
)

type cloudwatcheventsSvc struct{ *scan.Ctx }

var _ = scan.Register(cloudwatchevents.EndpointsID, cloudwatchevents.New, cloudwatcheventsSvc{},
	[]cloudwatchevents.ListRulesInput{},
)

func (s cloudwatcheventsSvc) DescribeRule(lr *cloudwatchevents.ListRulesOutput) (q []cloudwatchevents.DescribeRuleInput) {
	s.Split(&q, "Name", lr.Rules, "Name")
	return
}

func (s cloudwatcheventsSvc) ListTargetsByRule(lr *cloudwatchevents.ListRulesOutput) (q []cloudwatchevents.ListTargetsByRuleInput) {
	s.Split(&q, "Rule", lr.Rules, "Name")
	return
}

//
// Post-processing
//

func (s cloudwatcheventsSvc) Rules(out *cloudwatchevents.ListRulesOutput) error {
	return s.ImportResources("aws_cloudwatch_event_rule", tfx.AttrGen{
		"id": s.Strings(out.Rules, "Name"),
	})
}

func (s cloudwatcheventsSvc) Targets(out *cloudwatchevents.ListTargetsByRuleOutput) error {
	rule := *s.Input(out).(*cloudwatchevents.ListTargetsByRuleInput).Rule
	target := s.Strings(out.Targets, "Id")
	return s.MakeResources("aws_cloudwatch_event_target", tfx.AttrGen{
		"id":        func(i int) string { return rule + "-" + target[i] },
		"rule":      rule,
		"target_id": target,
	})
}
