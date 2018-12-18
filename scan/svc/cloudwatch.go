package svc

import (
	"github.com/LuminalHQ/cloudcover/awsscan/scan"
	"github.com/LuminalHQ/cloudcover/x/tfx"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
)

type cloudwatchSvc struct{ *scan.Ctx }

var _ = scan.Register(cloudwatch.EndpointsID, cloudwatch.New, cloudwatchSvc{},
	[]cloudwatch.DescribeAlarmsInput{},
	[]cloudwatch.ListDashboardsInput{},
)

//
// Post-processing
//

func (s cloudwatchSvc) Dashboards(out *cloudwatch.ListDashboardsOutput) error {
	return s.ImportResources("aws_cloudwatch_dashboard", tfx.AttrGen{
		"id": s.Strings(out.DashboardEntries, "DashboardName"),
	})
}
