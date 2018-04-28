package service

import (
	"github.com/LuminalHQ/cloudcover/awsscan/scan"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
)

type ecsSvc struct{}

func init() { scan.Register(ecsSvc{}) }

func (ecsSvc) ID() string           { return ecs.EndpointsID }
func (ecsSvc) NewFunc() interface{} { return ecs.New }
func (ecsSvc) Roots() []interface{} {
	return []interface{}{
		[]*ecs.ListClustersInput{nil},
		[]*ecs.ListTaskDefinitionsInput{nil},
	}
}

func (ecsSvc) DescribeClusters(lc *ecs.ListClustersOutput) (q []*ecs.DescribeClustersInput) {
	// TODO: Batch requests?
	setForEach(&q, "Clusters", lc.ClusterArns, "")
	return
}

func (ecsSvc) DescribeServices(ls *ecs.ListServicesOutput) (q []*ecs.DescribeServicesInput) {
	// TODO: Batch requests?
	lsq := ls.SDKResponseMetadata().Request.Params.(*ecs.ListServicesInput)
	for _, v := range ls.ServiceArns {
		q = append(q, &ecs.DescribeServicesInput{
			Cluster:  lsq.Cluster,
			Services: []string{v},
		})
	}
	return
}

func (ecsSvc) DescribeTaskDefinition(ltd *ecs.ListTaskDefinitionsOutput) (q []*ecs.DescribeTaskDefinitionInput) {
	setForEach(&q, "TaskDefinition", ltd.TaskDefinitionArns, "")
	return
}

func (ecsSvc) ListServices(lc *ecs.ListClustersOutput) (q []*ecs.ListServicesInput) {
	setForEach(&q, "Cluster", lc.ClusterArns, "")
	return
}
