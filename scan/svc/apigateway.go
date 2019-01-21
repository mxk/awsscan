package svc

import (
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/mxk/cloudcover/awsscan/scan"
)

type apigatewaySvc struct{ *scan.Ctx }

var _ = scan.Register(apigateway.EndpointsID, apigateway.New, apigatewaySvc{},
	[]apigateway.GetAccountInput{},
	[]apigateway.GetApiKeysInput{},
	[]apigateway.GetClientCertificatesInput{},
	[]apigateway.GetDomainNamesInput{},
	[]apigateway.GetRestApisInput{},
	[]apigateway.GetUsagePlansInput{},
	[]apigateway.GetVpcLinksInput{},
)

func (apigatewaySvc) HandleError(req *aws.Request, err *scan.Err) {
	err.Ignore = (err.Status == http.StatusNotFound &&
		req.Operation.Name == "GetAccount") ||
		(err.Status == http.StatusBadRequest &&
			req.Operation.Name == "GetVpcLinks")
}

func (s apigatewaySvc) GetAuthorizers(gra *apigateway.GetRestApisOutput) (q []apigateway.GetAuthorizersInput) {
	s.Split(&q, "RestApiId", gra.Items, "Id")
	return
}

func (s apigatewaySvc) GetBasePathMappings(gdn *apigateway.GetDomainNamesOutput) (q []apigateway.GetBasePathMappingsInput) {
	s.Split(&q, "DomainName", gdn.Items, "DomainName")
	return
}

func (s apigatewaySvc) GetDeployments(gra *apigateway.GetRestApisOutput) (q []apigateway.GetDeploymentsInput) {
	s.Split(&q, "RestApiId", gra.Items, "Id")
	return
}
