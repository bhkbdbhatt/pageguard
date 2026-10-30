// Package scope carries authenticated identity through a request so the
// service layer can resolve projects without a second parameter thread.
package scope

import (
	"github.com/pageguard/pageguard/internal/domain"
)

// Scope is the authorization identity attached to a request by the API auth
// middleware. Admin keys without a project pin are unrestricted; a scoped key
// pins ProjectID, which is also the default project for queries that omit it.
type Scope struct {
	KeyID     domain.ID
	ProjectID domain.ID // pinned project; empty means unrestricted
	Roles     []string  // domain.Role* values
}
