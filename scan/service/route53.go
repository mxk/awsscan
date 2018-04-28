package service

import (
	"github.com/LuminalHQ/cloudcover/awsscan/scan"
	"github.com/aws/aws-sdk-go-v2/service/route53"
)

type route53Svc struct{}

func init() { scan.Register(route53Svc{}) }

func (route53Svc) ID() string           { return route53.EndpointsID }
func (route53Svc) NewFunc() interface{} { return route53.New }
func (route53Svc) Roots() []interface{} {
	return []interface{}{
		[]*route53.ListHostedZonesInput{nil},
	}
}

func (route53Svc) ListResourceRecordSets(lhz *route53.ListHostedZonesOutput) (q []*route53.ListResourceRecordSetsInput) {
	setForEach(&q, "HostedZoneId", lhz.HostedZones, "Id")
	return
}
