package service

import (
	"github.com/LuminalHQ/cloudcover/awsscan/scan"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type dynamodbSvc struct{}

func init() { scan.Register(dynamodbSvc{}) }

func (dynamodbSvc) Name() string         { return dynamodb.ServiceName }
func (dynamodbSvc) NewFunc() interface{} { return dynamodb.New }
func (dynamodbSvc) Roots() []interface{} {
	return []interface{}{
		[]*dynamodb.ListTablesInput{nil},
	}
}

func (dynamodbSvc) DescribeTable(lt *dynamodb.ListTablesOutput) (q []*dynamodb.DescribeTableInput) {
	setForEach(&q, "TableName", lt.TableNames, "")
	return
}
