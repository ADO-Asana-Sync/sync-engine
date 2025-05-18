package azure

import (
	"context"

	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/core"
	"github.com/stretchr/testify/mock"
)

type MockCoreClient struct{ mock.Mock }

func (m *MockCoreClient) GetProjects(
	ctx context.Context,
	args core.GetProjectsArgs,
) (*core.GetProjectsResponseValue, error) {
	ret := m.Called(ctx, args)
	if ret.Get(0) == nil {
		return nil, ret.Error(1)
	}
	return ret.Get(0).(*core.GetProjectsResponseValue), ret.Error(1)
}
