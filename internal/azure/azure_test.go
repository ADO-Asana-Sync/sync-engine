package azure

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/core"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/workitemtracking"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ADO-Asana-Sync/sync-engine/internal/testutil"
)

func TestGetChangedWorkItems(t *testing.T) {
	t.Parallel() // Enable parallel execution of sub-tests
	const queryFmt = "SELECT [System.Id], [System.Title], [System.State] FROM workitems WHERE [System.ChangedDate] > '%s' ORDER BY [System.ChangedDate] DESC"
	type args struct {
		ctx      context.Context
		lastSync time.Time
	}

	testTime := time.Now()

	tests := []struct {
		name    string
		a       *Azure
		args    args
		query   string
		result  *workitemtracking.WorkItemQueryResult
		mockErr error
		want    []workitemtracking.WorkItemReference
		wantErr bool
		errMsg  string // Add error message for error cases
	}{
		{
			name:  "returns changed work items",
			a:     &Azure{},
			args:  args{ctx: context.Background(), lastSync: testTime},
			query: fmt.Sprintf(queryFmt, testTime.Format(time.RFC3339)),
			result: &workitemtracking.WorkItemQueryResult{
				WorkItems: &[]workitemtracking.WorkItemReference{{Id: testutil.Ptr(123)}},
			},
			mockErr: nil,
			want:    []workitemtracking.WorkItemReference{{Id: testutil.Ptr(123)}},
			wantErr: false,
		},
		{
			name:  "returns empty when no work items changed",
			a:     &Azure{},
			args:  args{ctx: context.Background(), lastSync: testTime},
			query: fmt.Sprintf(queryFmt, testTime.Format(time.RFC3339)),
			result: &workitemtracking.WorkItemQueryResult{
				WorkItems: &[]workitemtracking.WorkItemReference{},
			},
			mockErr: nil,
			want:    []workitemtracking.WorkItemReference(nil),
			wantErr: false,
		},
		{
			name:  "returns multiple changed work items",
			a:     &Azure{},
			args:  args{ctx: context.Background(), lastSync: testTime},
			query: fmt.Sprintf(queryFmt, testTime.Format(time.RFC3339)),
			result: &workitemtracking.WorkItemQueryResult{
				WorkItems: &[]workitemtracking.WorkItemReference{
					{Id: testutil.Ptr(123)},
					{Id: testutil.Ptr(456)},
				},
			},
			mockErr: nil,
			want: []workitemtracking.WorkItemReference{
				{Id: testutil.Ptr(123)},
				{Id: testutil.Ptr(456)},
			},
			wantErr: false,
		},
		{
			name:    "returns error when newWorkItemClient fails",
			a:       &Azure{},
			args:    args{ctx: context.Background(), lastSync: testTime},
			query:   "",
			result:  nil,
			mockErr: fmt.Errorf("failed to create work item client"),
			want:    nil,
			wantErr: true,
			errMsg:  "failed to create work item client",
		},
		{
			name:    "returns error when QueryByWiql fails",
			a:       &Azure{},
			args:    args{ctx: context.Background(), lastSync: testTime},
			query:   fmt.Sprintf(queryFmt, testTime.Format(time.RFC3339)),
			result:  nil,
			mockErr: fmt.Errorf("query failed"),
			want:    nil,
			wantErr: true,
			errMsg:  "query failed",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mockWI := new(MockWIClient)
			if tt.name == "returns error when newWorkItemClient fails" {
				tt.a.newWorkItemClient = func(ctx context.Context, conn *azuredevops.Connection) (WIClient, error) {
					return nil, tt.mockErr
				}
			} else {
				mockWI.
					On("QueryByWiql", mock.Anything, mock.MatchedBy(func(args workitemtracking.QueryByWiqlArgs) bool {
						return args.Wiql != nil && args.Wiql.Query != nil &&
							*args.Wiql.Query == tt.query &&
							args.TimePrecision != nil && *args.TimePrecision
					})).
					Return(tt.result, tt.mockErr)
				tt.a.newWorkItemClient = func(ctx context.Context, conn *azuredevops.Connection) (WIClient, error) {
					return mockWI, nil
				}
			}

			got, err := tt.a.GetChangedWorkItems(tt.args.ctx, tt.args.lastSync)
			if tt.wantErr {
				require.ErrorContains(t, err, tt.errMsg)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)

			mockWI.AssertExpectations(t)
		})
	}
}

func TestGetProjects(t *testing.T) {
	t.Parallel() // Enable parallel execution of sub-tests
	type args struct {
		ctx context.Context
	}
	// Use a slice of static UUIDs for deterministic tests
	uuids := []uuid.UUID{
		uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		uuid.MustParse("33333333-3333-3333-3333-333333333333"),
	}
	testProjects := []core.TeamProjectReference{
		{Id: testutil.Ptr(uuids[0]), Name: testutil.Ptr("Project1")},
		{Id: testutil.Ptr(uuids[1]), Name: testutil.Ptr("Project2")},
	}
	tests := []struct {
		name    string
		a       *Azure
		args    args
		result  *core.GetProjectsResponseValue
		mockErr error
		want    []core.TeamProjectReference
		wantErr bool
		pages   []*core.GetProjectsResponseValue // for continuation tests
	}{
		{
			name:    "returns a single project",
			a:       &Azure{},
			args:    args{ctx: context.Background()},
			result:  &core.GetProjectsResponseValue{Value: testProjects[:1]},
			mockErr: nil,
			want:    testProjects[:1],
			wantErr: false,
		},
		{
			name:    "returns multiple projects",
			a:       &Azure{},
			args:    args{ctx: context.Background()},
			result:  &core.GetProjectsResponseValue{Value: testProjects},
			mockErr: nil,
			want:    testProjects,
			wantErr: false,
		},
		{
			name:    "returns empty project list",
			a:       &Azure{},
			args:    args{ctx: context.Background()},
			result:  &core.GetProjectsResponseValue{Value: []core.TeamProjectReference{}},
			mockErr: nil,
			want:    nil,
			wantErr: false,
		},
		{
			name:    "returns error when newCoreClient fails",
			a:       &Azure{},
			args:    args{ctx: context.Background()},
			result:  nil,
			mockErr: fmt.Errorf("failed to create core client"),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "returns error when GetProjects fails",
			a:       &Azure{},
			args:    args{ctx: context.Background()},
			result:  nil,
			mockErr: fmt.Errorf("get projects failed"),
			want:    nil,
			wantErr: true,
		},
		{
			name: "returns projects from multiple pages",
			a:    &Azure{},
			args: args{ctx: context.Background()},
			pages: []*core.GetProjectsResponseValue{
				{
					Value: []core.TeamProjectReference{
						{Id: testutil.Ptr(uuids[0]), Name: testutil.Ptr("Project1")},
						{Id: testutil.Ptr(uuids[1]), Name: testutil.Ptr("Project2")},
					},
					ContinuationToken: "1",
				},
				{
					Value: []core.TeamProjectReference{
						{Id: testutil.Ptr(uuids[2]), Name: testutil.Ptr("Project3")},
					},
					ContinuationToken: "",
				},
			},
			mockErr: nil,
			want: []core.TeamProjectReference{
				{Id: testutil.Ptr(uuids[0]), Name: testutil.Ptr("Project1")},
				{Id: testutil.Ptr(uuids[1]), Name: testutil.Ptr("Project2")},
				{Id: testutil.Ptr(uuids[2]), Name: testutil.Ptr("Project3")},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mockCore := new(MockCoreClient)
			if tt.name == "returns error when newCoreClient fails" {
				tt.a.newCoreClient = func(ctx context.Context, conn *azuredevops.Connection) (CoreClient, error) {
					return nil, tt.mockErr
				}
			} else if tt.name == "returns error when GetProjects fails" {
				mockCore.On("GetProjects", mock.Anything, mock.Anything).Return(nil, tt.mockErr)
				tt.a.newCoreClient = func(ctx context.Context, conn *azuredevops.Connection) (CoreClient, error) {
					return mockCore, nil
				}
			} else if tt.pages != nil {
				// Generalized continuation token paging
				for i, page := range tt.pages {
					if i == 0 {
						mockCore.On("GetProjects", mock.Anything, mock.Anything).Return(page, nil).Once()
					} else {
						idx := i // capture loop var
						mockCore.On("GetProjects", mock.Anything, mock.MatchedBy(func(args core.GetProjectsArgs) bool {
							return args.ContinuationToken != nil && *args.ContinuationToken == idx
						})).Return(page, nil).Once()
					}
				}
				tt.a.newCoreClient = func(ctx context.Context, conn *azuredevops.Connection) (CoreClient, error) {
					return mockCore, nil
				}
			} else {
				mockCore.On("GetProjects", mock.Anything, mock.Anything).Return(tt.result, nil)
				tt.a.newCoreClient = func(ctx context.Context, conn *azuredevops.Connection) (CoreClient, error) {
					return mockCore, nil
				}
			}

			got, err := tt.a.GetProjects(tt.args.ctx)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			if tt.pages != nil {
				require.Len(t, got, len(tt.want))
				for i, proj := range tt.want {
					require.Equal(t, *proj.Name, *got[i].Name)
				}
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
			mockCore.AssertExpectations(t)
		})
	}
}
