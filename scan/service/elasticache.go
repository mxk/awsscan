package service

import (
	"github.com/LuminalHQ/cloudcover/awsscan/scan"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
)

type elasticacheSvc struct{}

func init() { scan.Register(elasticacheSvc{}) }

func (elasticacheSvc) ID() string           { return elasticache.EndpointsID }
func (elasticacheSvc) NewFunc() interface{} { return elasticache.New }
func (elasticacheSvc) Roots() []interface{} {
	return []interface{}{
		[]*elasticache.DescribeCacheClustersInput{nil},
		[]*elasticache.DescribeCacheSubnetGroupsInput{nil},
		[]*elasticache.DescribeReplicationGroupsInput{nil},
	}
}

// TODO: Need account and region for ARN
/*func (elasticacheSvc) DescribeTable(dcc *elasticache.DescribeCacheClustersOutput) (q []*elasticache.ListTagsForResourceInput) {
	return setForEach(q, "ResourceName", dcc.CacheClusters, "CacheClusterId").([]*elasticache.ListTagsForResourceInput)
}*/
