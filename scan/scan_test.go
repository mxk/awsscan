package scan

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/LuminalHQ/cloudcover/x/arn"
	"github.com/LuminalHQ/cloudcover/x/awsmock"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/awserr"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type iamSvc struct{ *Ctx }

func (s iamSvc) ListUserPolicies(lu *iam.ListUsersOutput) (q []iam.ListUserPoliciesInput) {
	s.Split(&q, "UserName", lu.Users, "UserName")
	return
}

func (iamSvc) GetUserPolicy(lup *iam.ListUserPoliciesOutput) (q []iam.GetUserPolicyInput) {
	for _, v := range lup.PolicyNames {
		in := iam.GetUserPolicyInput{PolicyName: aws.String(v)}
		switch v {
		case "policy0", "policy1":
			in.UserName = aws.String("alice")
		default:
			in.UserName = aws.String("bob")
		}
		q = append(q, in)
	}
	return
}

func TestScan(t *testing.T) {
	want := Map{Calls: map[string][]*Call{
		"ListUsers": {
			{
				ID: "9eaSgLBAOYRgu43u2qkVW7+Hbybwy1jmQIoSAkHY6KU=",
				In: &iam.ListUsersInput{},
				Out: []interface{}{
					&iam.ListUsersOutput{
						Users: []iam.User{
							{UserName: aws.String("alice")},
							{UserName: aws.String("bob")},
						},
						IsTruncated: aws.Bool(true),
						Marker:      aws.String("1"),
					},
					&iam.ListUsersOutput{
						Users: []iam.User{
							{UserName: aws.String("carol")},
						},
						IsTruncated: aws.Bool(true),
						Marker:      aws.String("2"),
					},
					&iam.ListUsersOutput{
						IsTruncated: aws.Bool(false),
					},
				},
			},
		},
		"ListUserPolicies": {
			{
				ID:  "5hz3ht+VdCDkQXeSmKI86ST4laqZtU2rD3f6TjDu8+w=",
				Src: map[string]int{"9eaSgLBAOYRgu43u2qkVW7+Hbybwy1jmQIoSAkHY6KU=": 0},
				In:  &iam.ListUserPoliciesInput{UserName: aws.String("alice")},
				Out: []interface{}{&iam.ListUserPoliciesOutput{
					PolicyNames: []string{"policy0", "policy1"},
				}},
			},
			{
				ID:  "uX9M3gRodDoYZprIiuDNdE3quEZASNupokX/htGDOxk=",
				Src: map[string]int{"9eaSgLBAOYRgu43u2qkVW7+Hbybwy1jmQIoSAkHY6KU=": 0},
				In:  &iam.ListUserPoliciesInput{UserName: aws.String("bob")},
				Out: []interface{}{&iam.ListUserPoliciesOutput{
					PolicyNames: []string{"policy2"},
				}},
			},
			{
				ID:  "R5BT+RwLB8UeHTOa14+0yTq788BP8WP7gN3vsuaphxs=",
				Src: map[string]int{"9eaSgLBAOYRgu43u2qkVW7+Hbybwy1jmQIoSAkHY6KU=": 1},
				In:  &iam.ListUserPoliciesInput{UserName: aws.String("carol")},
				Out: []interface{}{&iam.ListUserPoliciesOutput{}},
			},
		},
		"GetUserPolicy": {
			{
				ID:  "f7+5QHlzagScTIGNehtKjnKWdNLxgnIKEwbWXLKX9X4=",
				Src: map[string]int{"5hz3ht+VdCDkQXeSmKI86ST4laqZtU2rD3f6TjDu8+w=": 0},
				In: &iam.GetUserPolicyInput{
					PolicyName: aws.String("policy0"),
					UserName:   aws.String("alice"),
				},
				Out: []interface{}{&iam.GetUserPolicyOutput{
					PolicyDocument: aws.String(`{"Version": "0"}`),
					PolicyName:     aws.String("policy0"),
					UserName:       aws.String("alice"),
				}},
			},
			{
				ID:  "bpmJP0pFWY+wFlBRBl2LGU7tV+spQjmJsCno8j3eAUI=",
				Src: map[string]int{"5hz3ht+VdCDkQXeSmKI86ST4laqZtU2rD3f6TjDu8+w=": 0},
				In: &iam.GetUserPolicyInput{
					PolicyName: aws.String("policy1"),
					UserName:   aws.String("alice"),
				},
				Out: []interface{}{&iam.GetUserPolicyOutput{
					PolicyDocument: aws.String(`{"Version": "1"}`),
					PolicyName:     aws.String("policy1"),
					UserName:       aws.String("alice"),
				}},
			},
			{
				ID:  "bJuO8LFOVNi6Rlz2ZLUKohykCYuaoRjSnTUYoqQb5Ck=",
				Src: map[string]int{"uX9M3gRodDoYZprIiuDNdE3quEZASNupokX/htGDOxk=": 0},
				In: &iam.GetUserPolicyInput{
					PolicyName: aws.String("policy2"),
					UserName:   aws.String("bob"),
				},
				Out: []interface{}{&iam.GetUserPolicyOutput{
					PolicyDocument: aws.String(`{"Version": "2"}`),
					PolicyName:     aws.String("policy2"),
					UserName:       aws.String("bob"),
				}},
			},
		},
	}}
	cfg := awsmock.Config(func(r *aws.Request) {
		switch in := r.Params.(type) {
		case *sts.GetCallerIdentityInput:
			if r.Config.Region == "" || r.Config.Region == "us-east-1" {
				out := r.Data.(*sts.GetCallerIdentityOutput)
				out.Account = aws.String("000000000000")
				out.Arn = aws.String("arn:aws:iam::000000000000:user/alice")
			} else {
				e := awserr.New("InvalidClientTokenId", "The security token included in the request is invalid.", nil)
				r.Error = awserr.NewRequestFailure(e, 403, "00000000-0000-0000-0000-000000000000")
			}
		case *iam.ListUsersInput:
			i, _ := strconv.Atoi(aws.StringValue(in.Marker))
			for _, call := range want.Calls["ListUsers"] {
				set(r.Data, call.Out[i])
				break
			}
		case *iam.ListUserPoliciesInput:
			user := aws.StringValue(in.UserName)
			for _, call := range want.Calls["ListUserPolicies"] {
				if *call.In.(*iam.ListUserPoliciesInput).UserName == user {
					set(r.Data, call.Out[0])
					break
				}
			}
		case *iam.GetUserPolicyInput:
			pol := aws.StringValue(in.PolicyName)
			for _, call := range want.Calls["GetUserPolicy"] {
				if *call.In.(*iam.GetUserPolicyInput).PolicyName == pol {
					set(r.Data, call.Out[0])
					break
				}
			}
		}
	})

	orig := svcRegistry
	defer func() { svcRegistry = orig }()
	svcRegistry = registry{}
	svcRegistry.register("iam", iam.EndpointsID, iam.New, iamSvc{}, []interface{}{
		[]iam.ListUsersInput{},
	})
	m, err := Account(&cfg, Opts{
		Regions:  []string{"aws-global"},
		Services: []string{"iam"},
		Workers:  1,
	})
	require.NoError(t, err)
	require.Len(t, m, 1)
	want.Ctx = arn.Ctx{"aws", "aws-global", "000000000000"}
	want.Service = "iam"
	require.Equal(t, &want, m[0])

	want.Calls = map[string][]*Call{
		"ListUsers": {
			{
				ID: "9eaSgLBAOYRgu43u2qkVW7+Hbybwy1jmQIoSAkHY6KU=",
				Out: []interface{}{
					IO{"Users": []iam.User{
						{UserName: aws.String("alice")},
						{UserName: aws.String("bob")},
					}},
					IO{"Users": []iam.User{
						{UserName: aws.String("carol")},
					}},
				},
			},
		},
		"ListUserPolicies": {
			{
				ID:  "5hz3ht+VdCDkQXeSmKI86ST4laqZtU2rD3f6TjDu8+w=",
				Src: map[string]int{"9eaSgLBAOYRgu43u2qkVW7+Hbybwy1jmQIoSAkHY6KU=": 0},
				In:  IO{"UserName": aws.String("alice")},
				Out: []interface{}{IO{
					"PolicyNames": []string{"policy0", "policy1"}},
				},
			},
			{
				ID:  "uX9M3gRodDoYZprIiuDNdE3quEZASNupokX/htGDOxk=",
				Src: map[string]int{"9eaSgLBAOYRgu43u2qkVW7+Hbybwy1jmQIoSAkHY6KU=": 0},
				In:  IO{"UserName": aws.String("bob")},
				Out: []interface{}{IO{
					"PolicyNames": []string{"policy2"},
				}},
			},
		},
		"GetUserPolicy": {
			{
				ID:  "f7+5QHlzagScTIGNehtKjnKWdNLxgnIKEwbWXLKX9X4=",
				Src: map[string]int{"5hz3ht+VdCDkQXeSmKI86ST4laqZtU2rD3f6TjDu8+w=": 0},
				In: IO{
					"PolicyName": aws.String("policy0"),
					"UserName":   aws.String("alice"),
				},
				Out: []interface{}{IO{
					"PolicyDocument": aws.String(`{"Version": "0"}`),
					"PolicyName":     aws.String("policy0"),
					"UserName":       aws.String("alice"),
				}},
			},
			{
				ID:  "bpmJP0pFWY+wFlBRBl2LGU7tV+spQjmJsCno8j3eAUI=",
				Src: map[string]int{"5hz3ht+VdCDkQXeSmKI86ST4laqZtU2rD3f6TjDu8+w=": 0},
				In: IO{
					"PolicyName": aws.String("policy1"),
					"UserName":   aws.String("alice"),
				},
				Out: []interface{}{IO{
					"PolicyDocument": aws.String(`{"Version": "1"}`),
					"PolicyName":     aws.String("policy1"),
					"UserName":       aws.String("alice"),
				}},
			},
			{
				ID:  "bJuO8LFOVNi6Rlz2ZLUKohykCYuaoRjSnTUYoqQb5Ck=",
				Src: map[string]int{"uX9M3gRodDoYZprIiuDNdE3quEZASNupokX/htGDOxk=": 0},
				In: IO{
					"PolicyName": aws.String("policy2"),
					"UserName":   aws.String("bob"),
				},
				Out: []interface{}{IO{
					"PolicyDocument": aws.String(`{"Version": "2"}`),
					"PolicyName":     aws.String("policy2"),
					"UserName":       aws.String("bob"),
				}},
			},
		},
	}
	m = Compact(m)
	require.Len(t, m, 1)
	assert.Equal(t, &want, m[0])
}

func set(dst, src interface{}) {
	// This wipes responseMetadata
	reflect.ValueOf(dst).Elem().Set(reflect.ValueOf(src).Elem())
}
