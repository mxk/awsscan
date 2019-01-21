package svc

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	"github.com/mxk/awsscan/scan"
)

type elasticacheSvc struct{ *scan.Ctx }

var _ = scan.Register(elasticache.EndpointsID, elasticache.New, elasticacheSvc{},
	[]elasticache.DescribeCacheClustersInput{},
	[]elasticache.DescribeCacheSubnetGroupsInput{},
	[]elasticache.DescribeReplicationGroupsInput{},
)

func (s elasticacheSvc) ListTagsForResource(dcc *elasticache.DescribeCacheClustersOutput) (q []elasticache.ListTagsForResourceInput) {
	if n := len(dcc.CacheClusters); n > 0 {
		q = make([]elasticache.ListTagsForResourceInput, n)
		for i := range q {
			q[i].ResourceName = s.ARN("cluster:", aws.StringValue(dcc.CacheClusters[i].CacheClusterId))
		}
	}
	return
}
