package service

import (
	"github.com/LuminalHQ/cloudcover/awsscan/scan"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
)

type iamSvc struct{}

func init() { scan.Register(iamSvc{}) }

func (iamSvc) ID() string           { return iam.EndpointsID }
func (iamSvc) NewFunc() interface{} { return iam.New }
func (iamSvc) Roots() []interface{} {
	return []interface{}{
		[]*iam.ListGroupsInput{nil},
		[]*iam.ListInstanceProfilesInput{nil},
		[]*iam.ListPoliciesInput{{Scope: iam.PolicyScopeTypeLocal}},
		[]*iam.ListRolesInput{nil},
		[]*iam.ListUsersInput{nil},
	}
}

func (iamSvc) GetLoginProfile(lu *iam.ListUsersOutput) (q []*iam.GetLoginProfileInput) {
	setForEach(&q, "UserName", lu.Users, "UserName")
	return
}

func (iamSvc) GetGroupPolicy(lgp *iam.ListGroupPoliciesOutput) (q []*iam.GetGroupPolicyInput) {
	lgpq := lgp.SDKResponseMetadata().Request.Params.(*iam.ListGroupPoliciesInput)
	for _, v := range lgp.PolicyNames {
		q = append(q, &iam.GetGroupPolicyInput{
			PolicyName: aws.String(v),
			GroupName:  lgpq.GroupName,
		})
	}
	return
}

func (iamSvc) GetPolicy(lp *iam.ListPoliciesOutput) (q []*iam.GetPolicyInput) {
	setForEach(&q, "PolicyArn", lp.Policies, "Arn")
	return
}

func (iamSvc) GetPolicyVersion(lp *iam.ListPoliciesOutput) (q []*iam.GetPolicyVersionInput) {
	for i := range lp.Policies {
		q = append(q, &iam.GetPolicyVersionInput{
			PolicyArn: lp.Policies[i].Arn,
			VersionId: lp.Policies[i].DefaultVersionId,
		})
	}
	return
}

func (iamSvc) GetRolePolicy(lrp *iam.ListRolePoliciesOutput) (q []*iam.GetRolePolicyInput) {
	lrpq := lrp.SDKResponseMetadata().Request.Params.(*iam.ListRolePoliciesInput)
	for _, v := range lrp.PolicyNames {
		q = append(q, &iam.GetRolePolicyInput{
			PolicyName: aws.String(v),
			RoleName:   lrpq.RoleName,
		})
	}
	return
}

func (iamSvc) GetUserPolicy(lup *iam.ListUserPoliciesOutput) (q []*iam.GetUserPolicyInput) {
	lupq := lup.SDKResponseMetadata().Request.Params.(*iam.ListUserPoliciesInput)
	for _, v := range lup.PolicyNames {
		q = append(q, &iam.GetUserPolicyInput{
			PolicyName: aws.String(v),
			UserName:   lupq.UserName,
		})
	}
	return
}

func (iamSvc) ListAttachedGroupPolicies(lg *iam.ListGroupsOutput) (q []*iam.ListAttachedGroupPoliciesInput) {
	setForEach(&q, "GroupName", lg.Groups, "GroupName")
	return
}

func (iamSvc) ListAttachedRolePolicies(lr *iam.ListRolesOutput) (q []*iam.ListAttachedRolePoliciesInput) {
	setForEach(&q, "RoleName", lr.Roles, "RoleName")
	return
}

func (iamSvc) ListAttachedUserPolicies(lu *iam.ListUsersOutput) (q []*iam.ListAttachedUserPoliciesInput) {
	setForEach(&q, "UserName", lu.Users, "UserName")
	return
}

func (iamSvc) ListEntitiesForPolicy(lp *iam.ListPoliciesOutput) (q []*iam.ListEntitiesForPolicyInput) {
	setForEach(&q, "PolicyArn", lp.Policies, "Arn")
	return
}

func (iamSvc) ListGroupPolicies(lg *iam.ListGroupsOutput) (q []*iam.ListGroupPoliciesInput) {
	setForEach(&q, "GroupName", lg.Groups, "GroupName")
	return
}

func (iamSvc) ListGroupsForUser(lu *iam.ListUsersOutput) (q []*iam.ListGroupsForUserInput) {
	setForEach(&q, "UserName", lu.Users, "UserName")
	return
}

func (iamSvc) ListRolePolicies(lr *iam.ListRolesOutput) (q []*iam.ListRolePoliciesInput) {
	setForEach(&q, "RoleName", lr.Roles, "RoleName")
	return
}

func (iamSvc) ListUserPolicies(lu *iam.ListUsersOutput) (q []*iam.ListUserPoliciesInput) {
	setForEach(&q, "UserName", lu.Users, "UserName")
	return
}
