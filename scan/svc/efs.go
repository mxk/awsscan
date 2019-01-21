package svc

import (
	"github.com/aws/aws-sdk-go-v2/service/efs"
	"github.com/mxk/cloudcover/awsscan/scan"
)

type efsSvc struct{ *scan.Ctx }

var _ = scan.Register(efs.EndpointsID, efs.New, efsSvc{},
	[]efs.DescribeFileSystemsInput{},
)

func (s efsSvc) DescribeMountTargetSecurityGroups(dmt *efs.DescribeMountTargetsOutput) (q []efs.DescribeMountTargetSecurityGroupsInput) {
	s.Split(&q, "MountTargetId", dmt.MountTargets, "MountTargetId")
	return
}

func (s efsSvc) DescribeMountTargets(dfs *efs.DescribeFileSystemsOutput) (q []efs.DescribeMountTargetsInput) {
	s.Split(&q, "FileSystemId", dfs.FileSystems, "FileSystemId")
	return
}

func (s efsSvc) DescribeTags(dfs *efs.DescribeFileSystemsOutput) (q []efs.DescribeTagsInput) {
	s.Split(&q, "FileSystemId", dfs.FileSystems, "FileSystemId")
	return
}
