package service

import (
	"reflect"

	"github.com/LuminalHQ/cloudcover/awsscan/scan"
	"github.com/aws/aws-sdk-go-v2/aws/endpoints"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type s3Svc struct{ *scan.State }

func init() { scan.Register(s3Svc{}) }

func (s3Svc) ID() string           { return s3.EndpointsID }
func (s3Svc) NewFunc() interface{} { return s3.New }
func (s3Svc) Roots() []interface{} {
	return []interface{}{
		[]*s3.ListBucketsInput{nil},
	}
}

func (s s3Svc) GetBucketAcl(gbl *s3.GetBucketLocationOutput) (q []*s3.GetBucketAclInput) {
	s.setBucketName(&q, gbl)
	return
}

func (s s3Svc) GetBucketCors(gbl *s3.GetBucketLocationOutput) (q []*s3.GetBucketCorsInput) {
	s.setBucketName(&q, gbl)
	return
}

func (s3Svc) GetBucketLocation(lb *s3.ListBucketsOutput) (q []*s3.GetBucketLocationInput) {
	setForEach(&q, "Bucket", lb.Buckets, "Name")
	return
}

func (s s3Svc) GetBucketLogging(gbl *s3.GetBucketLocationOutput) (q []*s3.GetBucketLoggingInput) {
	s.setBucketName(&q, gbl)
	return
}

func (s s3Svc) GetBucketNotificationConfiguration(gbl *s3.GetBucketLocationOutput) (q []*s3.GetBucketNotificationConfigurationInput) {
	s.setBucketName(&q, gbl)
	return
}

func (s s3Svc) GetBucketPolicy(gbl *s3.GetBucketLocationOutput) (q []*s3.GetBucketPolicyInput) {
	s.setBucketName(&q, gbl)
	return
}

func (s s3Svc) GetBucketTagging(gbl *s3.GetBucketLocationOutput) (q []*s3.GetBucketTaggingInput) {
	s.setBucketName(&q, gbl)
	return
}

func (s s3Svc) GetBucketWebsite(gbl *s3.GetBucketLocationOutput) (q []*s3.GetBucketWebsiteInput) {
	s.setBucketName(&q, gbl)
	return
}

func (s s3Svc) setBucketName(q interface{}, gbl *s3.GetBucketLocationOutput) {
	// TODO: Is "EU" ever used?
	loc := string(gbl.LocationConstraint)
	if loc != s.Region && !(loc == "" && s.Region == endpoints.UsEast1RegionID) {
		return
	}
	name := gbl.SDKResponseMetadata().Request.Params.(*s3.GetBucketLocationInput).Bucket
	v := reflect.ValueOf(q).Elem()
	e := reflect.New(v.Type().Elem().Elem())
	e.Elem().FieldByName("Bucket").Set(reflect.ValueOf(name))
	v.Set(reflect.Append(v, e))
}
