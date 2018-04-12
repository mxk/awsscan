package service

import (
	"github.com/LuminalHQ/cloudcover/awsscan/scan"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
)

type cloudwatchlogsSvc struct{}

func init() { scan.Register(cloudwatchlogsSvc{}) }

func (cloudwatchlogsSvc) Name() string         { return cloudwatchlogs.ServiceName }
func (cloudwatchlogsSvc) NewFunc() interface{} { return cloudwatchlogs.New }
func (cloudwatchlogsSvc) Roots() []interface{} {
	return []interface{}{
		[]*cloudwatchlogs.DescribeLogGroupsInput{nil},
	}
}
