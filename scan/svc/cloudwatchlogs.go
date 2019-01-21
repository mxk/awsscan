package svc

import (
	_ "unsafe"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/mxk/awsscan/scan"
	"github.com/mxk/go-terraform/tfx"
	_ "github.com/terraform-providers/terraform-provider-aws/aws"
)

type cloudwatchlogsSvc struct{ *scan.Ctx }

var _ = scan.Register(cloudwatchlogs.EndpointsID, cloudwatchlogs.New, cloudwatchlogsSvc{},
	[]cloudwatchlogs.DescribeDestinationsInput{},
	[]cloudwatchlogs.DescribeLogGroupsInput{},
	[]cloudwatchlogs.DescribeMetricFiltersInput{},
	[]cloudwatchlogs.DescribeResourcePoliciesInput{},
)

func (s cloudwatchlogsSvc) DescribeSubscriptionFilters(dlg *cloudwatchlogs.DescribeLogGroupsOutput) (q []cloudwatchlogs.DescribeSubscriptionFiltersInput) {
	s.Split(&q, "LogGroupName", dlg.LogGroups, "LogGroupName")
	return
}

func (s cloudwatchlogsSvc) ListTagsLogGroup(dlg *cloudwatchlogs.DescribeLogGroupsOutput) (q []cloudwatchlogs.ListTagsLogGroupInput) {
	s.Split(&q, "LogGroupName", dlg.LogGroups, "LogGroupName")
	return
}

//
// Post-processing
//

func (s cloudwatchlogsSvc) Destinations(out *cloudwatchlogs.DescribeDestinationsOutput) error {
	dest := s.Strings(out.Destinations, "DestinationName")
	return firstError(
		s.ImportResources("aws_cloudwatch_log_destination", tfx.AttrGen{
			"id": dest,
		}),
		s.ImportResources("aws_cloudwatch_log_destination_policy", tfx.AttrGen{
			"id": dest,
		}),
	)
}

func (s cloudwatchlogsSvc) LogGroups(out *cloudwatchlogs.DescribeLogGroupsOutput) error {
	return s.ImportResources("aws_cloudwatch_log_group", tfx.AttrGen{
		"id": s.Strings(out.LogGroups, "LogGroupName"),
	})
}

func (s cloudwatchlogsSvc) MetricFilters(out *cloudwatchlogs.DescribeMetricFiltersOutput) error {
	name := s.Strings(out.MetricFilters, "FilterName")
	return s.MakeResources("aws_cloudwatch_log_metric_filter", tfx.AttrGen{
		"id":             name,
		"log_group_name": s.Strings(out.MetricFilters, "LogGroupName"),
		"name":           name,
	})
}

func (s cloudwatchlogsSvc) ResourcePolicies(out *cloudwatchlogs.DescribeResourcePoliciesOutput) error {
	return s.ImportResources("aws_cloudwatch_log_resource_policy", tfx.AttrGen{
		"id": s.Strings(out.ResourcePolicies, "PolicyName"),
	})
}

func (s cloudwatchlogsSvc) SubscriptionFilters(out *cloudwatchlogs.DescribeSubscriptionFiltersOutput) error {
	group := *s.Input(out).(*cloudwatchlogs.DescribeSubscriptionFiltersInput).LogGroupName
	dist := func(i int) *string {
		if d := string(out.SubscriptionFilters[i].Distribution); d != "" {
			return &d
		}
		return nil
	}
	return s.MakeResources("aws_cloudwatch_log_subscription_filter", tfx.AttrGen{
		"id":              func(i int) string { return cloudwatchLogsSubscriptionFilterId(group) },
		"destination_arn": s.Strings(out.SubscriptionFilters, "DestinationArn"),
		"distribution":    dist,
		"filter_pattern":  s.Strings(out.SubscriptionFilters, "FilterPattern"),
		"log_group_name":  group,
		"name":            s.Strings(out.SubscriptionFilters, "FilterName"),
		"role_arn":        func(i int) *string { return out.SubscriptionFilters[i].RoleArn },
	})
}

//go:linkname cloudwatchLogsSubscriptionFilterId github.com/terraform-providers/terraform-provider-aws/aws.cloudwatchLogsSubscriptionFilterId
func cloudwatchLogsSubscriptionFilterId(log_group_name string) string
