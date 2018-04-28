package service

import (
	"github.com/LuminalHQ/cloudcover/awsscan/scan"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

type snsSvc struct{}

func init() { scan.Register(snsSvc{}) }

func (snsSvc) ID() string           { return sns.EndpointsID }
func (snsSvc) NewFunc() interface{} { return sns.New }
func (snsSvc) Roots() []interface{} {
	return []interface{}{
		[]*sns.ListSubscriptionsInput{nil},
	}
}

func (snsSvc) GetSubscriptionAttributes(ls *sns.ListSubscriptionsOutput) (q []*sns.GetSubscriptionAttributesInput) {
	setForEach(&q, "SubscriptionArn", ls.Subscriptions, "SubscriptionArn")
	return
}
