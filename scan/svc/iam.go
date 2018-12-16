package svc

import (
	"net/http"

	"github.com/LuminalHQ/cloudcover/awsscan/scan"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
)

type iamSvc struct{ *scan.Ctx }

var _ = scan.Register(iam.EndpointsID, iam.New, iamSvc{},
	[]iam.ListGroupsInput{},
	[]iam.ListInstanceProfilesInput{},
	[]iam.ListPoliciesInput{{Scope: iam.PolicyScopeTypeLocal}},
	[]iam.ListRolesInput{},
	[]iam.ListUsersInput{},
)

func (iamSvc) HandleError(req *aws.Request, err *scan.Err) {
	err.Ignore = err.Status == http.StatusNotFound &&
		req.Operation.Name == "GetLoginProfile"
}

func (s iamSvc) GetLoginProfile(lu *iam.ListUsersOutput) (q []iam.GetLoginProfileInput) {
	s.Split(&q, "UserName", lu.Users, "UserName")
	return
}

func (s iamSvc) GetGroupPolicy(lgp *iam.ListGroupPoliciesOutput) (q []iam.GetGroupPolicyInput) {
	s.Split(&q, "PolicyName", lgp.PolicyNames, "")
	s.CopyInput(q, "GroupName", lgp)
	return
}

func (s iamSvc) GetPolicy(lp *iam.ListPoliciesOutput) (q []iam.GetPolicyInput) {
	s.Split(&q, "PolicyArn", lp.Policies, "Arn")
	return
}

func (s iamSvc) GetPolicyVersion(lp *iam.ListPoliciesOutput) (q []iam.GetPolicyVersionInput) {
	s.Split(&q, "PolicyArn", lp.Policies, "Arn")
	s.Split(&q, "VersionId", lp.Policies, "DefaultVersionId")
	return
}

func (s iamSvc) GetRolePolicy(lrp *iam.ListRolePoliciesOutput) (q []iam.GetRolePolicyInput) {
	s.Split(&q, "PolicyName", lrp.PolicyNames, "")
	s.CopyInput(q, "RoleName", lrp)
	return
}

func (s iamSvc) GetUserPolicy(lup *iam.ListUserPoliciesOutput) (q []iam.GetUserPolicyInput) {
	s.Split(&q, "PolicyName", lup.PolicyNames, "")
	s.CopyInput(q, "UserName", lup)
	return
}

func (s iamSvc) ListAttachedGroupPolicies(lg *iam.ListGroupsOutput) (q []iam.ListAttachedGroupPoliciesInput) {
	s.Split(&q, "GroupName", lg.Groups, "GroupName")
	return
}

func (s iamSvc) ListAttachedRolePolicies(lr *iam.ListRolesOutput) (q []iam.ListAttachedRolePoliciesInput) {
	s.Split(&q, "RoleName", lr.Roles, "RoleName")
	return
}

func (s iamSvc) ListAttachedUserPolicies(lu *iam.ListUsersOutput) (q []iam.ListAttachedUserPoliciesInput) {
	s.Split(&q, "UserName", lu.Users, "UserName")
	return
}

func (s iamSvc) ListEntitiesForPolicy(lp *iam.ListPoliciesOutput) (q []iam.ListEntitiesForPolicyInput) {
	s.Split(&q, "PolicyArn", lp.Policies, "Arn")
	return
}

func (s iamSvc) ListGroupPolicies(lg *iam.ListGroupsOutput) (q []iam.ListGroupPoliciesInput) {
	s.Split(&q, "GroupName", lg.Groups, "GroupName")
	return
}

func (s iamSvc) ListGroupsForUser(lu *iam.ListUsersOutput) (q []iam.ListGroupsForUserInput) {
	s.Split(&q, "UserName", lu.Users, "UserName")
	return
}

func (s iamSvc) ListPolicyVersions(lp *iam.ListPoliciesOutput) (q []iam.ListPolicyVersionsInput) {
	s.Split(&q, "PolicyArn", lp.Policies, "Arn")
	return
}

func (s iamSvc) ListRolePolicies(lr *iam.ListRolesOutput) (q []iam.ListRolePoliciesInput) {
	s.Split(&q, "RoleName", lr.Roles, "RoleName")
	return
}

func (s iamSvc) ListUserPolicies(lu *iam.ListUsersOutput) (q []iam.ListUserPoliciesInput) {
	s.Split(&q, "UserName", lu.Users, "UserName")
	return
}

func (s iamSvc) UserResource(out *iam.ListUsersOutput) (bool, error) {
	return false, s.MakeResources("aws_iam_user", s.Strings(out.Users, "UserName"), nil)
}
