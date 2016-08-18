// Package json2 implements an lrpchttp.Codec interface for using the JSON RPC2
// protocol
package json2

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"

	"github.com/levenlabs/lrpc"
	"github.com/levenlabs/lrpc/lrpchttp"

	"context"
)

// ErrCode is an integer used to identify errors over JSON RPC2
type ErrCode int

const (
	// ErrParse is used when invalid JSON was received by the server
	ErrParse ErrCode = -32700
	// ErrInvalidRequest is used when the JSON sent is not a valid Request
	// object
	ErrInvalidRequest ErrCode = -32600
	// ErrNoMethod is used when the method does not exist / is not available
	ErrNoMethod ErrCode = -32601
	// ErrInvalidParams is used when invalid method parameters were sent
	ErrInvalidParams ErrCode = -32602
	// ErrInternal is used when  an internal JSON-rpc error is encountered
	ErrInternal ErrCode = -32603
	// ErrServer is reserved for implementation-defined server-errors
	ErrServer ErrCode = -32000
)

// Error is an error implementation which contains additional information used
// by JSON RPC2
type Error struct {
	// Required. A Number that indicates the error type that occurred.
	Code ErrCode `json:"code"`

	// Required. A string providing a short description of the error. The
	// message SHOULD be limited to a concise single sentence.
	Message string `json:"message"`

	// Optional. A primitive or structured value that contains additional
	// information about the error.
	Data interface{} `json:"data"`
}

func (e *Error) Error() string {
	return e.Message
}

// Response implements a response object for the JSON RPC2 protocol
//
// If being used as part of a client request/response, before unmarshalling the
// response into this struct, be sure to fill Result with a pointer type of the
// expected return.
type Response struct {
	Version string `json:"jsonrpc"`

	// The object that was returned by the invoked method
	Result interface{} `json:"result,omitempty"`

	// An Error object if there was an error invoking the method
	Error *Error `json:"error,omitempty"`

	// This must be the same id as the request it is responding to.
	ID *json.RawMessage `json:"id"`
}

// Request implements a request object for the JSON RPC2 protocol
type Request struct {
	Version string `json:"jsonrpc"`

	// A string containing the name of the method to be invoked.
	Method string `json:"method"`

	// A structured value to pass as arguments to the method.
	Params *json.RawMessage `json:"params"`

	// The request id. MUST be a string, number or null.
	// Our implementation will not do type checking for id.
	// It will be copied as it is.
	ID *json.RawMessage `json:"id"`
}

// NewRequest encodes the method and its parameters into a new Request object,
// which can be marshalled and sent to a JSONRPC2 endpoint.
func NewRequest(method string, params interface{}) (Request, error) {
	idr := make([]byte, 16)
	if _, err := rand.Read(idr); err != nil {
		return Request{}, err
	}
	id := hex.EncodeToString(idr)
	r := Request{
		Version: "2.0",
		Method:  method,
	}

	{
		b, err := json.Marshal(id)
		if err != nil {
			return Request{}, err
		}
		jr := json.RawMessage(b)
		r.ID = &jr
	}
	{
		b, err := json.Marshal(params)
		if err != nil {
			return Request{}, err
		}
		jr := json.RawMessage(b)
		r.Params = &jr
	}

	return r, nil
}

// call is an implementation of the lrpc.Call interface for the JSON RPC2
// protocol
type call struct {
	ctx context.Context
	req Request
}

func (c call) Context() context.Context {
	return c.ctx
}

func (c call) Method() string {
	return c.req.Method
}

func (c call) UnmarshalArgs(i interface{}) error {
	return json.Unmarshal(*c.req.Params, i)
}

type ctxKey int

const ctxRequest ctxKey = 0

// ContextRequest can be called on a Context returned from an lrcp.Call sourced
// from json2.Codec. It returns the Request object that the request was
// unmarshalled into. Returns nil if the context doesn't have the Request object
// in it.
func ContextRequest(ctx context.Context) *Request {
	r, _ := ctx.Value(ctxRequest).(*Request)
	return r
}

// Codec implements the lrpchttp.Codec interface
//
//	httpHandler := lrpchttp.HTTPHandler(json2.Codec{}, h)
//
type Codec struct{}

// NewCall implements the lrpchttp.Codec interface
func (Codec) NewCall(ctx context.Context, w http.ResponseWriter, r *http.Request) (lrpc.Call, error) {
	c := call{}
	if err := json.NewDecoder(r.Body).Decode(&c.req); err != nil {
		return nil, err
	}
	c.ctx = context.WithValue(ctx, ctxRequest, &c.req)
	return c, nil
}

// Respond implements the lrpchttp.Codec interface
func (Codec) Respond(cc lrpc.Call, i interface{}) error {
	w := lrpchttp.ContextResponseWriter(cc.Context())
	c := cc.(call)

	var res Response
	if err, ok := i.(error); ok {
		jerr, ok := i.(*Error)
		if !ok {
			jerr = &Error{Code: ErrServer, Message: err.Error()}
		}
		res.Error = jerr
	} else {
		res.Result = i
	}
	res.Version = "2.0"
	res.ID = c.req.ID

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(w).Encode(&res)
}
