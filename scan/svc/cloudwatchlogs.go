package svc

import (
	"github.com/LuminalHQ/cloudcover/awsscan/scan"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
)

type cloudwatchlogsSvc struct{ *scan.Ctx }

var _ = scan.Register(cloudwatchlogs.EndpointsID, cloudwatchlogs.New, cloudwatchlogsSvc{},
	[]cloudwatchlogs.DescribeLogGroupsInput{},
	[]cloudwatchlogs.DescribeMetricFiltersInput{},
)

func (s cloudwatchlogsSvc) ListTagsLogGroup(dlg *cloudwatchlogs.DescribeLogGroupsOutput) (q []cloudwatchlogs.ListTagsLogGroupInput) {
	s.Split(&q, "LogGroupName", dlg.LogGroups, "LogGroupName")
	return
}
