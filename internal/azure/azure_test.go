package azure

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/workitemtracking"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ADO-Asana-Sync/sync-engine/internal/testutil"
)

func TestGetChangedWorkItems(t *testing.T) {
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
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockWI := new(MockWIClient)
			if tt.name == "returns error when newWorkItemClient fails" {
				tt.a.newWorkItemClient = func(ctx context.Context, conn *azuredevops.Connection) (WIClient, error) {
					return nil, tt.mockErr
				}
			} else {
				mockWI.
					On("QueryByWiql", mock.Anything, mock.MatchedBy(func(args workitemtracking.QueryByWiqlArgs) bool {
						return args.Wiql != nil && args.Wiql.Query != nil && *args.Wiql.Query == tt.query
					})).
					Return(tt.result, tt.mockErr)
				tt.a.newWorkItemClient = func(ctx context.Context, conn *azuredevops.Connection) (WIClient, error) {
					return mockWI, nil
				}
			}

			got, err := tt.a.GetChangedWorkItems(tt.args.ctx, tt.args.lastSync)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)

			mockWI.AssertExpectations(t)
		})
	}
}
