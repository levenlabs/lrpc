// Package lrpchttp implements a layer for translating http requests into
// lrpc.Calls, and translating the Call's response back into http.
//
// This is done through the use of codecs, which are essentially interfaces
// implementing the protocol over which data is transferred over http.
package lrpchttp

import (
	"net/http"

	"github.com/levenlabs/lrpc"

	"context"
)

// Codec describes a type which can translate an incoming http request into an
// rpc request, and send back the response for the request
type Codec interface {
	// Called when the request first comes in, translate the http information
	// into an lrpc.Call which can be then used in an lrpc.Handler. The returned
	// Call should use the given Context as its underlying Context, although it
	// may add more context layers on top of it.
	NewCall(context.Context, http.ResponseWriter, *http.Request) (lrpc.Call, error)

	// Used to marshal and send back a response for the lrpc.Call. If an error
	// is returned it will be sent back as a 500 response.
	//
	// Note for implementors: ContextResponseWriter can be used to retrieve the
	// underlying http.ResponseWriter for the Call
	Respond(lrpc.Call, interface{}) error
}

// HTTPHandler takes a Codec which can translate http requests to rpc calls, and
// a handler for those calls, and returns an http.Handler which puts it all
// together.
//
// If there is an error calling NewCall on the Codec that error will be returned
// as a 400
func HTTPHandler(codec Codec, h lrpc.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ctx = context.WithValue(ctx, contextKeyRequest, r)
		ctx = context.WithValue(ctx, contextKeyResponseWriter, w)

		c, err := codec.NewCall(ctx, w, r)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		res := h.ServeRPC(c)

		if err := codec.Respond(c, res); err != nil {
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

// ContextRequest takes in a Context from a Call generated from HTTPHandler and
// returns the original *http.Request object for the Call
func ContextRequest(ctx context.Context) *http.Request {
	r, _ := ctx.Value(contextKeyRequest).(*http.Request)
	return r
}

// ContextResponseWriter teks in a Context from a Call generated from
// HTTPHandler and returns the original http.ResponseWriter for the Call
func ContextResponseWriter(ctx context.Context) http.ResponseWriter {
	rw, _ := ctx.Value(contextKeyResponseWriter).(http.ResponseWriter)
	return rw
}
