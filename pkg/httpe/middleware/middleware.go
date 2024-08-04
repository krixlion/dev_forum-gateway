package middleware

import "github.com/krixlion/dev_forum-gateway/pkg/httpe"

// Applies middleware to the supplied handler and returns it.
// Middleware is applied in order it is provided.
func Apply(h httpe.HandlerEFunc, ms ...func(httpe.HandlerEFunc) httpe.HandlerEFunc) httpe.HandlerEFunc {
	for _, m := range ms {
		h = m(h)
	}
	return h
}
