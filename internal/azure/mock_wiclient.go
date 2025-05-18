package azure

import (
	"context"

	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/workitemtracking"
	"github.com/stretchr/testify/mock"
)

type MockWIClient struct{ mock.Mock }

func (m *MockWIClient) QueryByWiql(
	ctx context.Context,
	args workitemtracking.QueryByWiqlArgs,
) (*workitemtracking.WorkItemQueryResult, error) {
	ret := m.Called(ctx, args)
	return ret.Get(0).(*workitemtracking.WorkItemQueryResult), ret.Error(1)
}
