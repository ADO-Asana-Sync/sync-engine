package azure

import (
	"context"

	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/workitemtracking"
	"github.com/stretchr/testify/mock"
)

type MockWIClient struct{ mock.Mock }

// Ensure the mock still satisfies the interface at compile-time.
var _ WIClient = (*MockWIClient)(nil)

func (m *MockWIClient) QueryByWiql(
	ctx context.Context,
	args workitemtracking.QueryByWiqlArgs,
) (*workitemtracking.WorkItemQueryResult, error) {
	ret := m.Called(ctx, args)
	var result *workitemtracking.WorkItemQueryResult
	if ret.Get(0) != nil {
		result = ret.Get(0).(*workitemtracking.WorkItemQueryResult)
	}
	return result, ret.Error(1)
}
