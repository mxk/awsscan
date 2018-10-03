package svc

import (
	"strings"

	"github.com/LuminalHQ/cloudcover/fdb/aws/scan"
	"github.com/aws/aws-sdk-go-v2/service/route53"
)

type route53Svc struct{ *scan.Ctx }

var _ = scan.Register(route53.EndpointsID, route53.New, route53Svc{},
	[]route53.ListHostedZonesInput{},
)

func (s route53Svc) ListResourceRecordSets(lhz *route53.ListHostedZonesOutput) (q []route53.ListResourceRecordSetsInput) {
	s.Split(&q, "HostedZoneId", lhz.HostedZones, "Id")
	if s.Mode(scan.CloudAssert) {
		for i := range q {
			j := strings.LastIndexByte(*q[i].HostedZoneId, '/')
			*q[i].HostedZoneId = (*q[i].HostedZoneId)[j+1:]
		}
	}
	return
}
