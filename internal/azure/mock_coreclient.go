package azure

import (
	"context"
	"fmt"

	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/core"
	"github.com/stretchr/testify/mock"
)

type MockCoreClient struct{ mock.Mock }

// Ensure the mock still satisfies the interface at compile-time.
var _ CoreClient = (*MockCoreClient)(nil)

func (m *MockCoreClient) GetProjects(
	ctx context.Context,
	args core.GetProjectsArgs,
) (*core.GetProjectsResponseValue, error) {
	ret := m.Called(ctx, args)
	if ret.Get(0) == nil {
		return nil, ret.Error(1)
	}
	if resp, ok := ret.Get(0).(*core.GetProjectsResponseValue); ok {
		return resp, ret.Error(1)
	}
	return nil, fmt.Errorf("MockCoreClient: expected *core.GetProjectsResponseValue, got %T", ret.Get(0))
}
