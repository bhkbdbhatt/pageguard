package events

import "context"

// ctxBackground returns a background context. Store calls made from the
// emitter are fire-and-forget telemetry; a dedicated helper keeps the intent
// explicit and avoids importing context in every call site.
func ctxBackground() context.Context { return context.Background() }
