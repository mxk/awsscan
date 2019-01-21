package svc

import (
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/mxk/awsscan/scan"
	"github.com/mxk/go-terraform/tfx"
)

type dynamodbSvc struct{ *scan.Ctx }

var _ = scan.Register(dynamodb.EndpointsID, dynamodb.New, dynamodbSvc{},
	[]dynamodb.ListTablesInput{},
)

func (s dynamodbSvc) DescribeTable(lt *dynamodb.ListTablesOutput) (q []dynamodb.DescribeTableInput) {
	s.Split(&q, "TableName", lt.TableNames, "")
	return
}

//
// Post-processing
//

func (s dynamodbSvc) Tables(out *dynamodb.ListTablesOutput) error {
	return s.ImportResources("aws_dynamodb_table", tfx.AttrGen{
		"id": out.TableNames,
	})
}
