package service

import (
	"github.com/LuminalHQ/cloudcover/awsscan/scan"
	"github.com/aws/aws-sdk-go-v2/service/rds"
)

type rdsSvc struct{}

func init() { scan.Register(rdsSvc{}) }

func (rdsSvc) ID() string           { return rds.EndpointsID }
func (rdsSvc) NewFunc() interface{} { return rds.New }
func (rdsSvc) Roots() []interface{} {
	return []interface{}{
		[]*rds.DescribeDBClustersInput{nil},
		[]*rds.DescribeDBInstancesInput{nil},
		[]*rds.DescribeDBSubnetGroupsInput{nil},
	}
}

func (rdsSvc) DescribeDBClusterSnapshotAttributes(ddcs *rds.DescribeDBClusterSnapshotsOutput) (q []*rds.DescribeDBClusterSnapshotAttributesInput) {
	setForEach(&q, "DBClusterSnapshotIdentifier", ddcs.DBClusterSnapshots, "DBClusterSnapshotIdentifier")
	return
}

func (rdsSvc) DescribeDBClusterSnapshots(ddc *rds.DescribeDBClustersOutput) (q []*rds.DescribeDBClusterSnapshotsInput) {
	setForEach(&q, "DBClusterIdentifier", ddc.DBClusters, "DBClusterIdentifier")
	return
}

// TODO: ListTagsForResource for DB instances and subnet groups

func (rdsSvc) ListTagsForResource(ddc *rds.DescribeDBClustersOutput) (q []*rds.ListTagsForResourceInput) {
	setForEach(&q, "ResourceName", ddc.DBClusters, "DBClusterArn")
	return
}
