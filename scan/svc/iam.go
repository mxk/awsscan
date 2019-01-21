package svc

import (
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/mxk/awsscan/scan"
	"github.com/mxk/go-terraform/tfx"
)

type iamSvc struct{ *scan.Ctx }

var _ = scan.Register(iam.EndpointsID, iam.New, iamSvc{},
	[]iam.GetAccountPasswordPolicyInput{},
	[]iam.ListGroupsInput{},
	[]iam.ListInstanceProfilesInput{},
	[]iam.ListOpenIDConnectProvidersInput{},
	[]iam.ListPoliciesInput{{Scope: iam.PolicyScopeTypeLocal}},
	[]iam.ListRolesInput{},
	[]iam.ListSAMLProvidersInput{},
	[]iam.ListUsersInput{},
)

func (iamSvc) HandleError(req *aws.Request, err *scan.Err) {
	if err.Status == http.StatusNotFound {
		switch req.Operation.Name {
		case "GetAccountPasswordPolicy", "GetLoginProfile":
			err.Ignore = true
		}
	}
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

func (s iamSvc) ListAccessKeys(lu *iam.ListUsersOutput) (q []iam.ListAccessKeysInput) {
	s.Split(&q, "UserName", lu.Users, "UserName")
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

//
// Post-processing
//

func (s iamSvc) AccessKeys(out *iam.ListAccessKeysOutput) error {
	return s.MakeResources("aws_iam_access_key", tfx.AttrGen{
		"id":   s.Strings(out.AccessKeyMetadata, "AccessKeyId"),
		"user": func(i int) string { return *out.AccessKeyMetadata[i].UserName },
	})
}

func (s iamSvc) AccountPasswordPolicy(*iam.GetAccountPasswordPolicyOutput) error {
	return s.ImportResources("aws_iam_account_password_policy", tfx.AttrGen{
		"id": "PasswordPolicy",
	})
}

func (s iamSvc) GroupPolicies(out *iam.ListGroupPoliciesOutput) error {
	group := *s.Input(out).(*iam.ListGroupPoliciesInput).GroupName
	return s.MakeResources("aws_iam_group_policy", tfx.AttrGen{
		"#":  len(out.PolicyNames),
		"id": func(i int) string { return group + ":" + out.PolicyNames[i] },
	})
}

func (s iamSvc) GroupPolicyAttachments(out *iam.ListAttachedGroupPoliciesOutput) error {
	group := *s.Input(out).(*iam.ListAttachedGroupPoliciesInput).GroupName
	return s.ImportResources("aws_iam_group_policy_attachment", tfx.AttrGen{
		"#":  len(out.AttachedPolicies),
		"id": func(i int) string { return group + "/" + *out.AttachedPolicies[i].PolicyArn },
	})
}

func (s iamSvc) Groups(out *iam.ListGroupsOutput) error {
	name := s.Strings(out.Groups, "GroupName")
	return firstError(
		s.ImportResources("aws_iam_group", tfx.AttrGen{
			"id": name,
		}),
		s.MakeResources("aws_iam_group_membership", tfx.AttrGen{
			"id":    name,
			"group": name,
			"name":  name,
		}),
	)
}

func (s iamSvc) InstanceProfiles(out *iam.ListInstanceProfilesOutput) error {
	return s.ImportResources("aws_iam_instance_profile", tfx.AttrGen{
		"id": s.Strings(out.InstanceProfiles, "InstanceProfileName"),
	})
}

func (s iamSvc) OpenIDConnectProviders(out *iam.ListOpenIDConnectProvidersOutput) error {
	return s.ImportResources("aws_iam_openid_connect_provider", tfx.AttrGen{
		"id": s.Strings(out.OpenIDConnectProviderList, "Arn"),
	})
}

func (s iamSvc) Policies(out *iam.ListPoliciesOutput) error {
	return s.ImportResources("aws_iam_policy", tfx.AttrGen{
		"id": s.Strings(out.Policies, "Arn"),
	})
}

func (s iamSvc) Roles(out *iam.ListRolesOutput) error {
	return s.ImportResources("aws_iam_role", tfx.AttrGen{
		"id": s.Strings(out.Roles, "RoleName"),
	})
}

func (s iamSvc) RolePolicies(out *iam.ListRolePoliciesOutput) error {
	role := *s.Input(out).(*iam.ListRolePoliciesInput).RoleName
	return s.ImportResources("aws_iam_role_policy", tfx.AttrGen{
		"#":  len(out.PolicyNames),
		"id": func(i int) string { return role + ":" + out.PolicyNames[i] },
	})
}

func (s iamSvc) RolePolicyAttachments(out *iam.ListAttachedRolePoliciesOutput) error {
	role := *s.Input(out).(*iam.ListAttachedRolePoliciesInput).RoleName
	return s.ImportResources("aws_iam_role_policy_attachment", tfx.AttrGen{
		"#":  len(out.AttachedPolicies),
		"id": func(i int) string { return role + "/" + *out.AttachedPolicies[i].PolicyArn },
	})
}

func (s iamSvc) SAMLProviders(out *iam.ListSAMLProvidersOutput) error {
	return s.ImportResources("aws_iam_saml_provider", tfx.AttrGen{
		"id": s.Strings(out.SAMLProviderList, "Arn"),
	})
}

func (s iamSvc) User(out *iam.ListUsersOutput) error {
	return s.ImportResources("aws_iam_user", tfx.AttrGen{
		"id": s.Strings(out.Users, "UserName"),
	})
}

func (s iamSvc) UserPolicies(out *iam.ListUserPoliciesOutput) error {
	user := *s.Input(out).(*iam.ListUserPoliciesInput).UserName
	return s.ImportResources("aws_iam_user_policy", tfx.AttrGen{
		"#":  len(out.PolicyNames),
		"id": func(i int) string { return user + ":" + out.PolicyNames[i] },
	})
}

func (s iamSvc) UserPolicyAttachments(out *iam.ListAttachedUserPoliciesOutput) error {
	user := *s.Input(out).(*iam.ListAttachedUserPoliciesInput).UserName
	return s.ImportResources("aws_iam_user_policy_attachment", tfx.AttrGen{
		"#":  len(out.AttachedPolicies),
		"id": func(i int) string { return user + "/" + *out.AttachedPolicies[i].PolicyArn },
	})
}

func firstError(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}
