package azure

import (
	"fmt"
	"time"
)

// WorkItem represents the fields we care about on an Azure DevOps work item.
type WorkItem struct {
	ID           int
	AssignedTo   string
	ChangedDate  time.Time
	CreatedDate  time.Time
	State        string
	Title        string
	URL          string
	WorkItemType string
}

// checkRequiredProperties checks if the given property names are present (non-zero/non-empty) on the WorkItem.
// Returns an error with the message "missing property <property name>" for the first missing property found.
func (wi WorkItem) checkRequiredProperties(properties ...string) error {
	checkers := map[string]func() bool{
		"ID":           func() bool { return wi.ID != 0 },
		"Title":        func() bool { return wi.Title != "" },
		"WorkItemType": func() bool { return wi.WorkItemType != "" },
		"URL":          func() bool { return wi.URL != "" },
		"AssignedTo":   func() bool { return wi.AssignedTo != "" },
		"State":        func() bool { return wi.State != "" },
		"ChangedDate":  func() bool { return !wi.ChangedDate.IsZero() },
		"CreatedDate":  func() bool { return !wi.CreatedDate.IsZero() },
	}
	for _, prop := range properties {
		checker, ok := checkers[prop]
		if !ok {
			return fmt.Errorf("missing property %s", prop)
		}
		if !checker() {
			return fmt.Errorf("missing property %s", prop)
		}
	}
	return nil
}

// FormatTitle returns a formatted work item title in the format:
// <Work Item Type> <Work Item Number>: <Work Item Title>
// Returns an error if any required field is missing.
func (wi WorkItem) FormatTitle() (string, error) {
	if err := wi.checkRequiredProperties("Title", "WorkItemType", "ID"); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s %d: %s", wi.WorkItemType, wi.ID, wi.Title), nil
}

// FormatTitleWithLink returns a formatted work item title in the format:
// <a href="URL"><Work Item Type> <Work Item Number>:</a> <Work Item Title>
// Returns an error if any required field is missing, including URL.
func (wi WorkItem) FormatTitleWithLink() (string, error) {
	if err := wi.checkRequiredProperties("Title", "WorkItemType", "ID", "URL"); err != nil {
		return "", err
	}
	prefix := fmt.Sprintf("%s %d:", wi.WorkItemType, wi.ID)
	return fmt.Sprintf(`<a href="%s">%s</a> %s`, wi.URL, prefix, wi.Title), nil
}
