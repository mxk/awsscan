package svc

import (
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/mxk/awsscan/scan"
)

type snsSvc struct{ *scan.Ctx }

var _ = scan.Register(sns.EndpointsID, sns.New, snsSvc{},
	[]sns.ListSubscriptionsInput{},
	[]sns.ListTopicsInput{},
)

func (s snsSvc) GetSubscriptionAttributes(ls *sns.ListSubscriptionsOutput) (q []sns.GetSubscriptionAttributesInput) {
	s.Split(&q, "SubscriptionArn", ls.Subscriptions, "SubscriptionArn")
	return
}

func (s snsSvc) GetTopicAttributes(lt *sns.ListTopicsOutput) (q []sns.GetTopicAttributesInput) {
	s.Split(&q, "TopicArn", lt.Topics, "TopicArn")
	return
}
