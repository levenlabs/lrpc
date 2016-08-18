package lrpchttp

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	. "testing"

	"context"

	"github.com/levenlabs/lrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testCodec is a codec used for testing. The request path (sans prefixed slash)
// is used as the method name, and the body as a string is the argument. It only
// supports responses which are strings.
type testCodec struct{}

func (testCodec) NewCall(ctx context.Context, w http.ResponseWriter, r *http.Request) (lrpc.Call, error) {
	return testCodecCall{ctx, r}, nil
}

func (testCodec) Respond(c lrpc.Call, res interface{}) error {
	w := ContextResponseWriter(c.Context())
	_, err := fmt.Fprint(w, res.(string))
	return err
}

type testCodecCall struct {
	ctx context.Context
	r   *http.Request
}

func (tcc testCodecCall) Context() context.Context {
	return tcc.ctx
}

func (tcc testCodecCall) Method() string {
	return tcc.r.URL.Path[1:]
}

func (tcc testCodecCall) UnmarshalArgs(i interface{}) error {
	bodyB, _ := ioutil.ReadAll(tcc.r.Body)
	body := string(bodyB)
	reflect.Indirect(reflect.ValueOf(i)).Set(reflect.ValueOf(body))
	return nil
}

func TestHTTPHandler(t *T) {
	wCh := make(chan http.ResponseWriter, 1)
	rCh := make(chan *http.Request, 1)
	h := HTTPHandler(testCodec{}, lrpc.HandlerFunc(func(c lrpc.Call) interface{} {
		wCh <- ContextResponseWriter(c.Context())
		rCh <- ContextRequest(c.Context())
		var s string
		if err := c.UnmarshalArgs(&s); err != nil {
			return err
		}
		return c.Method() + ":" + s
	}))

	r, err := http.NewRequest("GET", "/foo", bytes.NewBufferString("bar"))
	require.Nil(t, err)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	assert.Equal(t, w, <-wCh)
	assert.Equal(t, r, <-rCh)
	assert.Equal(t, "foo:bar", w.Body.String())
}
