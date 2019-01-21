package svc

import (
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/mxk/awsscan/scan"
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
