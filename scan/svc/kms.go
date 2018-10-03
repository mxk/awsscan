package svc

import (
	"github.com/LuminalHQ/cloudcover/awsscan/scan"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms"
)

type kmsSvc struct{ *scan.Ctx }

var _ = scan.Register(kms.EndpointsID, kms.New, kmsSvc{},
	[]kms.ListAliasesInput{},
	[]kms.ListKeysInput{},
)

var lrtPaginator = aws.Paginator{
	InputTokens:     []string{"Marker"},
	OutputTokens:    []string{"NextMarker"},
	LimitToken:      "Limit",
	TruncationToken: "Truncated",
}

func (kmsSvc) UpdateRequest(req *aws.Request) {
	if op := req.Operation; op.Name == "ListResourceTags" {
		op.Paginator = &lrtPaginator
	}
}

func (kmsSvc) HandleError(req *aws.Request, err *scan.Err) {
	err.Ignore = err.Code == "AccessDeniedException"
}

func (s kmsSvc) DescribeKey(lk *kms.ListKeysOutput) (q []kms.DescribeKeyInput) {
	s.Split(&q, "KeyId", lk.Keys, "KeyId")
	return
}

func (s kmsSvc) GetKeyPolicy(lkp *kms.ListKeyPoliciesOutput) (q []kms.GetKeyPolicyInput) {
	s.Split(&q, "PolicyName", lkp.PolicyNames, "")
	s.CopyInput(q, "KeyId", lkp)
	return
}

func (s kmsSvc) GetKeyRotationStatus(lk *kms.ListKeysOutput) (q []kms.GetKeyRotationStatusInput) {
	s.Split(&q, "KeyId", lk.Keys, "KeyId")
	return
}

func (s kmsSvc) ListGrants(lk *kms.ListKeysOutput) (q []kms.ListGrantsInput) {
	s.Split(&q, "KeyId", lk.Keys, "KeyId")
	return
}

func (s kmsSvc) ListKeyPolicies(lk *kms.ListKeysOutput) (q []kms.ListKeyPoliciesInput) {
	s.Split(&q, "KeyId", lk.Keys, "KeyId")
	return
}

func (s kmsSvc) ListResourceTags(lk *kms.ListKeysOutput) (q []kms.ListResourceTagsInput) {
	s.Split(&q, "KeyId", lk.Keys, "KeyId")
	return
}
