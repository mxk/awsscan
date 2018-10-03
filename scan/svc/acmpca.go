package svc

import (
	"github.com/LuminalHQ/cloudcover/fdb/aws/scan"
	"github.com/aws/aws-sdk-go-v2/service/acmpca"
)

type acmpcaSvc struct{ *scan.Ctx }

var _ = scan.Register(acmpca.EndpointsID, acmpca.New, acmpcaSvc{},
	[]acmpca.ListCertificateAuthoritiesInput{},
)

func (s acmpcaSvc) ListTags(lca *acmpca.ListCertificateAuthoritiesOutput) (q []acmpca.ListTagsInput) {
	s.Split(&q, "CertificateAuthorityArn", lca.CertificateAuthorities, "Arn")
	return
}
