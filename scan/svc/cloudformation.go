package svc

import (
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/mxk/cloudcover/awsscan/scan"
)

type cloudformationSvc struct{ *scan.Ctx }

var _ = scan.Register(cloudformation.EndpointsID, cloudformation.New, cloudformationSvc{},
	[]cloudformation.DescribeStacksInput{},
	[]cloudformation.ListExportsInput{},
	[]cloudformation.ListStackSetsInput{},
)

func (s cloudformationSvc) DescribeStackSet(lss *cloudformation.ListStackSetsOutput) (q []cloudformation.DescribeStackSetInput) {
	s.Split(&q, "StackSetName", lss.Summaries, "StackSetId")
	return
}

func (s cloudformationSvc) ListImports(le *cloudformation.ListExportsOutput) (q []cloudformation.ListImportsInput) {
	s.Split(&q, "ExportName", le.Exports, "Name")
	return
}

func (s cloudformationSvc) ListStackInstances(lss *cloudformation.ListStackSetsOutput) (q []cloudformation.ListStackInstancesInput) {
	s.Split(&q, "StackSetName", lss.Summaries, "StackSetId")
	return
}

func (s cloudformationSvc) ListStackResources(ls *cloudformation.DescribeStacksOutput) (q []cloudformation.ListStackResourcesInput) {
	s.Split(&q, "StackName", ls.Stacks, "StackId")
	return
}
