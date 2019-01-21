package svc

import (
	"net/http"
	"reflect"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/endpoints"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/mxk/awsscan/scan"
	"github.com/mxk/go-terraform/tfx"
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

func (s s3Svc) ListBucketInventoryConfigurations(gbl *s3.GetBucketLocationOutput) (q []s3.ListBucketInventoryConfigurationsInput) {
	// Pagination missing from SDK: https://github.com/aws/aws-sdk-go-v2/issues/256
	s.setBucketName(&q, gbl)
	return
}

func (s s3Svc) ListBucketMetricsConfigurations(gbl *s3.GetBucketLocationOutput) (q []s3.ListBucketMetricsConfigurationsInput) {
	s.setBucketName(&q, gbl)
	return
}

func (s s3Svc) setBucketName(q interface{}, out *s3.GetBucketLocationOutput) {
	if s.inRegion(out) {
		name := s.Input(out).(*s3.GetBucketLocationInput).Bucket
		v := reflect.MakeSlice(reflect.TypeOf(q).Elem(), 1, 1)
		v.Index(0).FieldByName("Bucket").Set(reflect.ValueOf(name))
		reflect.ValueOf(q).Elem().Set(v)
	}
}

//
// Post-processing
//

func (s s3Svc) Bucket(out *s3.GetBucketLocationOutput) error {
	if !s.inRegion(out) {
		return nil
	}
	return s.MakeResources("aws_s3_bucket", tfx.AttrGen{
		"id": *s.Input(out).(*s3.GetBucketLocationInput).Bucket,
	})
}

func (s s3Svc) BucketInventoryConfigurations(out *s3.ListBucketInventoryConfigurationsOutput) error {
	bucket := *s.Input(out).(*s3.ListBucketInventoryConfigurationsInput).Bucket
	return s.ImportResources("aws_s3_bucket_inventory", tfx.AttrGen{
		"#":  len(out.InventoryConfigurationList),
		"id": func(i int) string { return bucket + ":" + *out.InventoryConfigurationList[i].Id },
	})
}

func (s s3Svc) BucketMetricsConfigurations(out *s3.ListBucketMetricsConfigurationsOutput) error {
	bucket := *s.Input(out).(*s3.ListBucketMetricsConfigurationsInput).Bucket
	return s.ImportResources("aws_s3_bucket_metric", tfx.AttrGen{
		"#":  len(out.MetricsConfigurationList),
		"id": func(i int) string { return bucket + ":" + *out.MetricsConfigurationList[i].Id },
	})
}

func (s s3Svc) BucketNotificationConfiguration(out *s3.GetBucketNotificationConfigurationOutput) error {
	return s.ImportResources("aws_s3_bucket_notification", tfx.AttrGen{
		"id": *s.Input(out).(*s3.GetBucketNotificationConfigurationInput).Bucket,
	})
}

func (s s3Svc) BucketPolicy(out *s3.GetBucketPolicyOutput) error {
	return s.ImportResources("aws_s3_bucket_policy", tfx.AttrGen{
		"id": *s.Input(out).(*s3.GetBucketPolicyInput).Bucket,
	})
}

func (s s3Svc) inRegion(out *s3.GetBucketLocationOutput) bool {
	loc := string(out.LocationConstraint)
	switch loc {
	case "":
		loc = endpoints.UsEast1RegionID
	case "EU":
		loc = endpoints.EuWest1RegionID
	}
	return loc == s.Region
}
