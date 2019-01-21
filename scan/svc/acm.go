package svc

import (
	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/mxk/cloudcover/awsscan/scan"
)

type acmSvc struct{ *scan.Ctx }

var _ = scan.Register(acm.EndpointsID, acm.New, acmSvc{},
	[]acm.ListCertificatesInput{},
)

func (s acmSvc) DescribeCertificate(lc *acm.ListCertificatesOutput) (q []acm.DescribeCertificateInput) {
	s.Split(&q, "CertificateArn", lc.CertificateSummaryList, "CertificateArn")
	return
}

func (s acmSvc) ListTagsForCertificate(lc *acm.ListCertificatesOutput) (q []acm.ListTagsForCertificateInput) {
	s.Split(&q, "CertificateArn", lc.CertificateSummaryList, "CertificateArn")
	return
}
