package service

import (
	"github.com/LuminalHQ/cloudcover/awsscan/scan"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type sqsSvc struct{}

func init() { scan.Register(sqsSvc{}) }

func (sqsSvc) Name() string         { return sqs.ServiceName }
func (sqsSvc) NewFunc() interface{} { return sqs.New }
func (sqsSvc) Roots() []interface{} {
	return []interface{}{
		[]*sqs.ListQueuesInput{nil},
	}
}

func (sqsSvc) GetQueueAttributes(lq *sqs.ListQueuesOutput) (q []*sqs.GetQueueAttributesInput) {
	for _, v := range lq.QueueUrls {
		q = append(q, &sqs.GetQueueAttributesInput{
			AttributeNames: []sqs.QueueAttributeName{sqs.QueueAttributeNameAll},
			QueueUrl:       aws.String(v),
		})
	}
	return
}
