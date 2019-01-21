package svc

import (
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/mxk/awsscan/scan"
)

type lambdaSvc struct{ *scan.Ctx }

var _ = scan.Register(lambda.EndpointsID, lambda.New, lambdaSvc{},
	[]lambda.ListFunctionsInput{},
)

func (lambdaSvc) HandleError(req *aws.Request, err *scan.Err) {
	err.Ignore = err.Status == http.StatusNotFound &&
		req.Operation.Name == "GetPolicy"
}

func (s lambdaSvc) ListAliases(lf *lambda.ListFunctionsOutput) (q []lambda.ListAliasesInput) {
	s.Split(&q, "FunctionName", lf.Functions, "FunctionName")
	return
}

func (s lambdaSvc) GetPolicyForAliases(la *lambda.ListAliasesOutput) (q []lambda.GetPolicyInput) {
	s.Split(&q, "Qualifier", la.Aliases, "FunctionVersion")
	s.CopyInput(q, "FunctionName", la)
	return
}

func (s lambdaSvc) GetPolicyForFunction(lf *lambda.ListFunctionsOutput) (q []lambda.GetPolicyInput) {
	s.Split(&q, "FunctionName", lf.Functions, "FunctionName")
	return
}
