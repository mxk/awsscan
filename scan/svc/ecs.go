package svc

import (
	"github.com/LuminalHQ/cloudcover/fdb/aws/scan"
	"github.com/LuminalHQ/cloudcover/x/arn"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
)

type ecsSvc struct{ *scan.Ctx }

var _ = scan.Register(ecs.EndpointsID, ecs.New, ecsSvc{},
	[]ecs.ListTaskDefinitionsInput{},
)

func (s ecsSvc) DescribeClusters(lc *ecs.ListClustersOutput) (q []ecs.DescribeClustersInput) {
	s.Group(&q, "Clusters", lc.ClusterArns, "", 100)
	if s.Mode(scan.CloudAssert) {
		if len(q) == 0 {
			q = make([]ecs.DescribeClustersInput, 1)
		} else {
			for i := range q {
				c := q[i].Clusters
				for j := range c {
					c[j] = arn.ARN(c[j]).Name()
				}
			}
		}
	}
	return
}

func (s ecsSvc) DescribeServices(ls *ecs.ListServicesOutput) (q []ecs.DescribeServicesInput) {
	s.Group(&q, "Services", ls.ServiceArns, "", 10)
	s.CopyInput(q, "Cluster", ls)
	return
}

func (s ecsSvc) DescribeTaskDefinition(ltd *ecs.ListTaskDefinitionsOutput) (q []ecs.DescribeTaskDefinitionInput) {
	s.Split(&q, "TaskDefinition", ltd.TaskDefinitionArns, "")
	return
}

func (s ecsSvc) ListClusters() (q []ecs.ListClustersInput) {
	q = make([]ecs.ListClustersInput, 1)
	if s.Mode(scan.CloudAssert) {
		q[0].MaxResults = aws.Int64(100)
	}
	return
}

func (s ecsSvc) ListServices(lc *ecs.ListClustersOutput) (q []ecs.ListServicesInput) {
	s.Split(&q, "Cluster", lc.ClusterArns, "")
	if s.Mode(scan.CloudAssert) {
		for i := range q {
			q[i].Cluster = aws.String(arn.Value(q[i].Cluster).Name())
		}
	}
	return
}

func (s ecsSvc) mockOutput(out interface{}) {
	if s.Mode(scan.CloudAssert) {
		if lc, ok := out.(*ecs.ListClustersOutput); ok {
			lc.ClusterArns = []string{*s.ARN("cluster/test-cluster")}
		}
	}
}
