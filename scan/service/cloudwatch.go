package service

import (
	"github.com/LuminalHQ/cloudcover/awsscan/scan"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
)

type cloudwatchSvc struct{}

func init() { scan.Register(cloudwatchSvc{}) }

func (cloudwatchSvc) ID() string           { return cloudwatch.EndpointsID }
func (cloudwatchSvc) NewFunc() interface{} { return cloudwatch.New }
func (cloudwatchSvc) Roots() []interface{} {
	return []interface{}{
		[]*cloudwatch.DescribeAlarmsInput{nil},
	}
}
