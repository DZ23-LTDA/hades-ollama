package agent

import (
	"errors"
	"strings"
)

var ErrPluginOrganizationScope = errors.New("plugin is not owned by the requested organization")

func pluginOwnedByOrganization(owner, organizationID string) bool {
	owner = strings.TrimSpace(owner)
	organizationID = strings.TrimSpace(organizationID)
	return owner != "" && organizationID != "" && owner == organizationID
}

// organizationOwnsRecord accepts ownerless legacy records only in the local
// single-user scope. Tenant scopes always require an explicit matching owner.
func organizationOwnsRecord(owner, organizationID string) bool {
	owner = strings.TrimSpace(owner)
	organizationID = strings.TrimSpace(organizationID)
	if organizationID == LocalOrganizationID {
		return owner == "" || owner == LocalOrganizationID
	}
	return pluginOwnedByOrganization(owner, organizationID)
}

// pluginAccessibleByOrganization is intentionally tenant-only. Ownerless
// plugins are reserved for explicitly trusted local-mode APIs and must never
// be treated as belonging to every organization.
func pluginAccessibleByOrganization(owner, organizationID string) bool {
	if strings.TrimSpace(organizationID) == LocalOrganizationID {
		return pluginGlobal(owner) || strings.TrimSpace(owner) == LocalOrganizationID
	}
	return pluginOwnedByOrganization(owner, organizationID)
}

func pluginGlobal(owner string) bool {
	return strings.TrimSpace(owner) == ""
}
