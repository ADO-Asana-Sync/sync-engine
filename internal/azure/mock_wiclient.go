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

func (m *MockWIClient) GetWorkItem(
	ctx context.Context,
	args workitemtracking.GetWorkItemArgs,
) (*workitemtracking.WorkItem, error) {
	ret := m.Called(ctx, args)
	var wi *workitemtracking.WorkItem
	if ret.Get(0) != nil {
		wi = ret.Get(0).(*workitemtracking.WorkItem)
	}
	return wi, ret.Error(1)
}
