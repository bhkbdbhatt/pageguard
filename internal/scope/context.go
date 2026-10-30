package scope

import (
	"context"

	"github.com/pageguard/pageguard/internal/domain"
)

type scopeKey struct{}

// With returns a context carrying s. Passing nil clears the scope.
func With(ctx context.Context, s *Scope) context.Context {
	return context.WithValue(ctx, scopeKey{}, s)
}

// From context reports the attached scope, if any.
func From(ctx context.Context) (*Scope, bool) {
	s, ok := ctx.Value(scopeKey{}).(*Scope)
	return s, ok
}

// Admin reports whether the request carries an admin-role key.
func Admin(ctx context.Context) bool {
	s, ok := From(ctx)
	return ok && s != nil && s.HasRole(domain.RoleAdmin)
}

// Writer reports whether the request may mutate (writer or admin).
func Writer(ctx context.Context) bool {
	s, ok := From(ctx)
	return ok && s != nil && s.HasRole(domain.RoleWriter)
}

// HasRole reports whether the scope carries role (admins carry all).
func (s *Scope) HasRole(role string) bool {
	if s == nil {
		return false
	}
	for _, r := range s.Roles {
		if r == domain.RoleAdmin || r == role {
			return true
		}
	}
	return false
}
