package azure

import (
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
)

type Azure struct {
	Client *azuredevops.Connection
}

func (a *Azure) Connect(orgUrl, pat string) {
	clt := azuredevops.NewPatConnection(orgUrl, pat)
	a.Client = clt
}

func (a *Azure) GetChangedWorkItems() {
	// Get the changed work items.
}
