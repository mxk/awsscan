package service

import (
	"github.com/LuminalHQ/cloudcover/awsscan/scan"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
)

type lambdaSvc struct{}

func init() { scan.Register(lambdaSvc{}) }

func (lambdaSvc) Name() string         { return lambda.ServiceName }
func (lambdaSvc) NewFunc() interface{} { return lambda.New }
func (lambdaSvc) Roots() []interface{} {
	return []interface{}{
		[]*lambda.ListFunctionsInput{nil},
	}
}

func (lambdaSvc) ListAliases(lf *lambda.ListFunctionsOutput) (q []*lambda.ListAliasesInput) {
	setForEach(&q, "FunctionName", lf.Functions, "FunctionName")
	return
}

// TODO: GetPolicy for aliases

func (lambdaSvc) GetPolicy(lf *lambda.ListFunctionsOutput) (q []*lambda.GetPolicyInput) {
	setForEach(&q, "FunctionName", lf.Functions, "FunctionName")
	return
}
