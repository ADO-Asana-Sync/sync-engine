package azure

import (
	"strings"
	"testing"
)

func TestWorkItemFormatTitle(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		wi      WorkItem
		want    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "basic formatting",
			wi:      WorkItem{WorkItemType: "Bug", ID: 123, Title: "Crash on launch"},
			want:    "Bug 123: Crash on launch",
			wantErr: false,
		},
		{
			name:    "feature work item",
			wi:      WorkItem{WorkItemType: "Feature", ID: 456, Title: "Add login support"},
			want:    "Feature 456: Add login support",
			wantErr: false,
		},
		{
			name:    "empty title",
			wi:      WorkItem{WorkItemType: "Task", ID: 789, Title: ""},
			wantErr: true,
			errMsg:  "missing property Title",
		},
		{
			name:    "empty type",
			wi:      WorkItem{WorkItemType: "", ID: 101, Title: "Untyped work item"},
			wantErr: true,
			errMsg:  "missing property WorkItemType",
		},
		{
			name:    "zero number",
			wi:      WorkItem{WorkItemType: "Bug", ID: 0, Title: "No number"},
			wantErr: true,
			errMsg:  "missing property ID",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := tt.wi.FormatTitle()
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("WorkItem.FormatTitle() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWorkItemFormatTitleWithLink(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		wi      WorkItem
		want    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "basic formatting with link",
			wi:      WorkItem{WorkItemType: "Bug", ID: 123, Title: "Crash on launch", URL: "https://dev.azure.com/org/project/_workitems/edit/123"},
			want:    `<a href="https://dev.azure.com/org/project/_workitems/edit/123">Bug 123:</a> Crash on launch`,
			wantErr: false,
		},
		{
			name:    "feature work item with link",
			wi:      WorkItem{WorkItemType: "Feature", ID: 456, Title: "Add login support", URL: "https://dev.azure.com/org/project/_workitems/edit/456"},
			want:    `<a href="https://dev.azure.com/org/project/_workitems/edit/456">Feature 456:</a> Add login support`,
			wantErr: false,
		},
		{
			name:    "empty title",
			wi:      WorkItem{WorkItemType: "Task", ID: 789, Title: "", URL: "https://dev.azure.com/org/project/_workitems/edit/789"},
			wantErr: true,
			errMsg:  "missing property Title",
		},
		{
			name:    "empty type",
			wi:      WorkItem{WorkItemType: "", ID: 101, Title: "Untyped work item", URL: "https://dev.azure.com/org/project/_workitems/edit/101"},
			wantErr: true,
			errMsg:  "missing property WorkItemType",
		},
		{
			name:    "zero number",
			wi:      WorkItem{WorkItemType: "Bug", ID: 0, Title: "No number", URL: "https://dev.azure.com/org/project/_workitems/edit/0"},
			wantErr: true,
			errMsg:  "missing property ID",
		},
		{
			name:    "missing URL",
			wi:      WorkItem{WorkItemType: "Bug", ID: 123, Title: "Crash on launch", URL: ""},
			wantErr: true,
			errMsg:  "missing property URL",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := tt.wi.FormatTitleWithLink()
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("WorkItem.FormatTitleWithLink() = %q, want %q", got, tt.want)
			}
		})
	}
}
