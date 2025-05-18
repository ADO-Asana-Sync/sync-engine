package azure

import "fmt"

// WorkItem represents the fields we care about on an Azure DevOps work item.
type WorkItem struct {
	ID           int
	Title        string
	WorkItemType string
	State        string
	AssignedTo   string
	CreatedDate  string
	ChangedDate  string
	URL          string
}

// FormatTitle returns a formatted work item title in the format:
// <Work Item Type> <Work Item Number>: <Work Item Title>
// Returns an error if any required field is missing.
func (wi WorkItem) FormatTitle() (string, error) {
	if wi.Title == "" {
		return "", fmt.Errorf("missing work item title")
	}
	if wi.WorkItemType == "" {
		return "", fmt.Errorf("missing work item type")
	}
	if wi.ID == 0 {
		return "", fmt.Errorf("missing work item number")
	}
	return fmt.Sprintf("%s %d: %s", wi.WorkItemType, wi.ID, wi.Title), nil
}

// FormatTitleWithLink returns a formatted work item title in the format:
// <a href="URL"><Work Item Type> <Work Item Number>:</a> <Work Item Title>
// Returns an error if any required field is missing, including URL.
func (wi WorkItem) FormatTitleWithLink() (string, error) {
	if wi.Title == "" {
		return "", fmt.Errorf("missing work item title")
	}
	if wi.WorkItemType == "" {
		return "", fmt.Errorf("missing work item type")
	}
	if wi.ID == 0 {
		return "", fmt.Errorf("missing work item number")
	}
	if wi.URL == "" {
		return "", fmt.Errorf("missing work item URL")
	}
	prefix := fmt.Sprintf("%s %d:", wi.WorkItemType, wi.ID)
	return fmt.Sprintf(`<a href="%s">%s</a> %s`, wi.URL, prefix, wi.Title), nil
}
