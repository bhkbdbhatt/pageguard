package domain

import (
	"crypto/rand"
	"encoding/hex"
)

// ID is a 128-bit opaque identifier rendered as a lowercase hex string.
// Using a hex string (rather than a UUID library) keeps the foundation
// dependency-free while remaining globally unique enough for scan data.
type ID string

// NewID returns a fresh, random 128-bit identifier.
func NewID() ID {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand failing is unrecoverable; the process cannot generate IDs.
		panic("pageguard: crypto/rand unavailable: " + err.Error())
	}
	return ID(hex.EncodeToString(b[:]))
}

// String implements fmt.Stringer.
func (id ID) String() string { return string(id) }

// IsEmpty reports whether the ID is the zero value.
func (id ID) IsEmpty() bool { return id == "" }
