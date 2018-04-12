package service

import (
	"github.com/LuminalHQ/cloudcover/awsscan/scan"
	"github.com/aws/aws-sdk-go-v2/service/cloudtrail"
)

type cloudtrailSvc struct{}

func init() { scan.Register(cloudtrailSvc{}) }

func (cloudtrailSvc) Name() string         { return cloudtrail.ServiceName }
func (cloudtrailSvc) NewFunc() interface{} { return cloudtrail.New }
func (cloudtrailSvc) Roots() []interface{} {
	return []interface{}{
		[]*cloudtrail.DescribeTrailsInput{nil},
	}
}

func (cloudtrailSvc) GetEventSelectors(dt *cloudtrail.DescribeTrailsOutput) (q []*cloudtrail.GetEventSelectorsInput) {
	setForEach(&q, "TrailName", dt.TrailList, "TrailARN")
	return
}

func (cloudtrailSvc) GetTrailStatus(dt *cloudtrail.DescribeTrailsOutput) (q []*cloudtrail.GetTrailStatusInput) {
	setForEach(&q, "Name", dt.TrailList, "TrailARN")
	return
}

func (cloudtrailSvc) ListTags(dt *cloudtrail.DescribeTrailsOutput) (q []*cloudtrail.ListTagsInput) {
	// Name does not work, must be ARN
	setForEach(&q, "ResourceIdList", dt.TrailList, "TrailARN")
	return
}
