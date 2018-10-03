package svc

import (
	"net/http"
	"reflect"

	"github.com/LuminalHQ/cloudcover/fdb/aws/scan"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/endpoints"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type s3Svc struct{ *scan.Ctx }

var _ = scan.Register(s3.EndpointsID, s3.New, s3Svc{},
	[]s3.ListBucketsInput{},
)

func (s3Svc) HandleError(req *aws.Request, err *scan.Err) {
	if err.Status == http.StatusNotFound {
		switch req.Operation.Name {
		case "GetBucketCors",
			"GetBucketPolicy",
			"GetBucketTagging",
			"GetBucketWebsite":
			err.Ignore = true
		}
	}
}

func (s s3Svc) GetBucketAcl(gbl *s3.GetBucketLocationOutput) (q []s3.GetBucketAclInput) {
	s.setBucketName(&q, gbl)
	return
}

func (s s3Svc) GetBucketCors(gbl *s3.GetBucketLocationOutput) (q []s3.GetBucketCorsInput) {
	s.setBucketName(&q, gbl)
	return
}

func (s s3Svc) GetBucketLocation(lb *s3.ListBucketsOutput) (q []s3.GetBucketLocationInput) {
	s.Split(&q, "Bucket", lb.Buckets, "Name")
	return
}

func (s s3Svc) GetBucketLogging(gbl *s3.GetBucketLocationOutput) (q []s3.GetBucketLoggingInput) {
	s.setBucketName(&q, gbl)
	return
}

func (s s3Svc) GetBucketNotificationConfiguration(gbl *s3.GetBucketLocationOutput) (q []s3.GetBucketNotificationConfigurationInput) {
	s.setBucketName(&q, gbl)
	return
}

func (s s3Svc) GetBucketPolicy(gbl *s3.GetBucketLocationOutput) (q []s3.GetBucketPolicyInput) {
	s.setBucketName(&q, gbl)
	return
}

func (s s3Svc) GetBucketTagging(gbl *s3.GetBucketLocationOutput) (q []s3.GetBucketTaggingInput) {
	s.setBucketName(&q, gbl)
	return
}

func (s s3Svc) GetBucketWebsite(gbl *s3.GetBucketLocationOutput) (q []s3.GetBucketWebsiteInput) {
	s.setBucketName(&q, gbl)
	return
}

func (s s3Svc) setBucketName(q interface{}, gbl *s3.GetBucketLocationOutput) {
	// TODO: Is "EU" ever used?
	loc := string(gbl.LocationConstraint)
	if loc == s.Region || (loc == "" && s.Region == endpoints.UsEast1RegionID) {
		name := s.Input(gbl).(*s3.GetBucketLocationInput).Bucket
		v := reflect.MakeSlice(reflect.TypeOf(q).Elem(), 1, 1)
		v.Index(0).FieldByName("Bucket").Set(reflect.ValueOf(name))
		reflect.ValueOf(q).Elem().Set(v)
	}
}
