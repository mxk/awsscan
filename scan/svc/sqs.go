package svc

import (
	"github.com/LuminalHQ/cloudcover/fdb/aws/scan"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type sqsSvc struct{ *scan.Ctx }

var _ = scan.Register(sqs.EndpointsID, sqs.New, sqsSvc{},
	[]sqs.ListQueuesInput{},
)

func (s sqsSvc) GetQueueAttributes(lq *sqs.ListQueuesOutput) (q []sqs.GetQueueAttributesInput) {
	s.Split(&q, "QueueUrl", lq.QueueUrls, "")
	attrs := []sqs.QueueAttributeName{sqs.QueueAttributeNameAll}
	for i := range q {
		q[i].AttributeNames = attrs
	}
	return
}
