// Package lrpchttp implements a layer for translating http requests into
// lrpc.Calls, and translating the Call's response back into http.
//
// This is done through the use of codecs, which are essentially interfaces
// implementing the protocol over which data is transferred over http.
package lrpchttp

import (
	"net/http"

	"github.com/levenlabs/lrpc"

	"golang.org/x/net/context"
)

// Codec describes a type which can translate an incoming http request into an
// rpc request
type Codec interface {
	NewCall(http.ResponseWriter, *http.Request) (lrpc.Call, error)
}

// HTTPHandler takes a Codec which can translate http requests to rpc calls, and
// a handler for those calls, and returns an http.Handler which puts it all
// together.
//
// If there is an error calling NewCall on the Codec that error will be returned
// as a 400
func HTTPHandler(c Codec, h lrpc.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := c.NewCall(w, r)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		c = wrapCall(w, r, c)

		res := h.ServeRPC(c)
		if err := c.MarshalResponse(w, res); err != nil {
			// this probably won't ever go through, but might as well try
			http.Error(w, err.Error(), 500)
			return
		}
	})
}

type contextKey int

const (
	contextKeyRequest contextKey = iota
	contextKeyResponseWriter
)

type wrapper struct {
	lrpc.Call
	ctx context.Context
}

func (w wrapper) GetContext() context.Context {
	return w.ctx
}

func wrapCall(w http.ResponseWriter, r *http.Request, c lrpc.Call) lrpc.Call {
	ctx := c.GetContext()
	ctx = context.WithValue(ctx, contextKeyRequest, r)
	ctx = context.WithValue(ctx, contextKeyResponseWriter, w)
	return wrapper{c, ctx}
}

// ContextRequest takes in a Context from a Call generated from HTTPHandler and
// returns the original *http.Request object for the Call
func ContextRequest(ctx context.Context) *http.Request {
	return ctx.Value(contextKeyRequest).(*http.Request)
}

// ContextResponseWriter teks in a Context from a Call generated from
// HTTPHandler and returns the original http.ResponseWriter for the Call
func ContextResponseWriter(ctx context.Context) http.ResponseWriter {
	return ctx.Value(contextKeyResponseWriter).(http.ResponseWriter)
}
