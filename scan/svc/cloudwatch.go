package svc

import (
	"github.com/LuminalHQ/cloudcover/fdb/aws/scan"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
)

type cloudwatchSvc struct{ *scan.Ctx }

var _ = scan.Register(cloudwatch.EndpointsID, cloudwatch.New, cloudwatchSvc{},
	[]cloudwatch.DescribeAlarmsInput{},
)
