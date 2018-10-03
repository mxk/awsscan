package svc

import (
	"github.com/LuminalHQ/cloudcover/awsscan/scan"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type dynamodbSvc struct{ *scan.Ctx }

var _ = scan.Register(dynamodb.EndpointsID, dynamodb.New, dynamodbSvc{},
	[]dynamodb.ListTablesInput{},
)

func (s dynamodbSvc) DescribeTable(lt *dynamodb.ListTablesOutput) (q []dynamodb.DescribeTableInput) {
	s.Split(&q, "TableName", lt.TableNames, "")
	return
}
