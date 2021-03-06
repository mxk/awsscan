package svc

import (
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/mxk/awsscan/scan"
)

type kinesisSvc struct{ *scan.Ctx }

var _ = scan.Register(kinesis.EndpointsID, kinesis.New, kinesisSvc{},
	[]kinesis.ListStreamsInput{},
)

func (s kinesisSvc) DescribeStreamSummary(ls *kinesis.ListStreamsOutput) (q []kinesis.DescribeStreamSummaryInput) {
	s.Split(&q, "StreamName", ls.StreamNames, "")
	return
}

func (s kinesisSvc) ListTagsForStream(ls *kinesis.ListStreamsOutput) (q []kinesis.ListTagsForStreamInput) {
	s.Split(&q, "StreamName", ls.StreamNames, "")
	return
}
