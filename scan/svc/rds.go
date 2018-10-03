package svc

import (
	"github.com/LuminalHQ/cloudcover/awsscan/scan"
	"github.com/aws/aws-sdk-go-v2/service/rds"
)

type rdsSvc struct{ *scan.Ctx }

var _ = scan.Register(rds.EndpointsID, rds.New, rdsSvc{},
	[]rds.DescribeDBClustersInput{},
	[]rds.DescribeDBInstancesInput{},
	[]rds.DescribeDBSubnetGroupsInput{},
)

func (s rdsSvc) DescribeDBClusterSnapshotAttributes(ddcs *rds.DescribeDBClusterSnapshotsOutput) (q []rds.DescribeDBClusterSnapshotAttributesInput) {
	s.Split(&q, "DBClusterSnapshotIdentifier", ddcs.DBClusterSnapshots, "DBClusterSnapshotIdentifier")
	return
}

func (s rdsSvc) DescribeDBClusterSnapshots(ddc *rds.DescribeDBClustersOutput) (q []rds.DescribeDBClusterSnapshotsInput) {
	s.Split(&q, "DBClusterIdentifier", ddc.DBClusters, "DBClusterIdentifier")
	return
}

func (s rdsSvc) ListTagsForResourceDBClusters(ddc *rds.DescribeDBClustersOutput) (q []rds.ListTagsForResourceInput) {
	s.Split(&q, "ResourceName", ddc.DBClusters, "DBClusterArn")
	return
}

func (s rdsSvc) ListTagsForResourceDBInstances(ddi *rds.DescribeDBInstancesOutput) (q []rds.ListTagsForResourceInput) {
	s.Split(&q, "ResourceName", ddi.DBInstances, "DBInstanceArn")
	return
}

func (s rdsSvc) ListTagsForResourceDBSubnetGroups(ddsg *rds.DescribeDBSubnetGroupsOutput) (q []rds.ListTagsForResourceInput) {
	s.Split(&q, "ResourceName", ddsg.DBSubnetGroups, "DBSubnetGroupArn")
	return
}
