package svc

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudtrail"
	"github.com/mxk/cloudcover/awsscan/scan"
)

type cloudtrailSvc struct{ *scan.Ctx }

var _ = scan.Register(cloudtrail.EndpointsID, cloudtrail.New, cloudtrailSvc{},
	[]cloudtrail.DescribeTrailsInput{},
)

func (s cloudtrailSvc) GetEventSelectors(dt *cloudtrail.DescribeTrailsOutput) (q []cloudtrail.GetEventSelectorsInput) {
	s.Split(&q, "TrailName", s.local(dt.TrailList), "TrailARN")
	return
}

func (s cloudtrailSvc) GetTrailStatus(dt *cloudtrail.DescribeTrailsOutput) (q []cloudtrail.GetTrailStatusInput) {
	s.Split(&q, "Name", s.local(dt.TrailList), "TrailARN")
	return
}

func (s cloudtrailSvc) ListTags(dt *cloudtrail.DescribeTrailsOutput) (q []cloudtrail.ListTagsInput) {
	group := 20
	if s.Mode(scan.CloudAssert) {
		group = 1
	}
	s.Group(&q, "ResourceIdList", s.local(dt.TrailList), "TrailARN", group)
	return
}

func (s cloudtrailSvc) local(trails []cloudtrail.Trail) (local []*cloudtrail.Trail) {
	for i := range trails {
		if aws.StringValue(trails[i].HomeRegion) == s.Region {
			if local == nil {
				local = make([]*cloudtrail.Trail, 0, len(trails)-i)
			}
			local = append(local, &trails[i])
		}
	}
	return
}

func (s cloudtrailSvc) mockOutput(out interface{}) {
	if dt, ok := out.(*cloudtrail.DescribeTrailsOutput); ok {
		region := aws.String(s.Region)
		for i := range dt.TrailList {
			dt.TrailList[i].HomeRegion = region
		}
	}
}
