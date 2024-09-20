package basic

import (
	"context"
	"strings"

	v1 "k8s.io/api/rbac/v1"
)

const (
	kubectlImage = "docker.io/bitnami/kubectl:latest"
)

func (s *Suite) TestRBAC() {
	const namePrefix = "rbac"
	ctx := context.Background()

	target, err := s.Knuu.NewInstance(namePrefix + "-target")
	s.Require().NoError(err)
	s.Require().NoError(target.Build().SetImage(ctx, kubectlImage))
	s.Require().NoError(target.Build().SetStartCommand("sleep", "infinity"))
	s.Require().NoError(target.Build().Commit(ctx))

	policyRule := v1.PolicyRule{
		Verbs:     []string{"get", "list", "watch"},
		APIGroups: []string{""},
		Resources: []string{"pods"},
	}
	s.Require().NoError(target.Security().AddPolicyRule(policyRule))

	// Test logic

	s.Require().NoError(target.Execution().Start(ctx))

	_, err = target.Execution().ExecuteCommand(ctx, "kubectl", "get", "pods")
	s.Require().NoError(err)

	exitCode, err := target.Execution().ExecuteCommand(ctx, "echo", "$?")
	s.Require().NoError(err)

	exitCode = strings.TrimSpace(exitCode)
	s.Assert().Equal("0", exitCode)
}
