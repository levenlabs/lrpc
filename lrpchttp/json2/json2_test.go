package json2

import (
	"bytes"
	"encoding/json"
	"errors"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strconv"
	. "testing"
	"time"

	"github.com/levenlabs/lrpc"
	"github.com/levenlabs/lrpc/lrpchttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var err2 = &Error{
	Code:    1,
	Message: "another error",
	Data:    map[string]interface{}{"foo": "bar"},
}

var h = lrpchttp.HTTPHandler(Codec{}, lrpc.ServeMux{}.
	HandleFunc("Echo", func(c lrpc.Call) interface{} {
		var i interface{}
		c.UnmarshalArgs(&i)
		return i
	}).
	HandleFunc("Error1", func(lrpc.Call) interface{} {
		return errors.New("some error")
	}).
	HandleFunc("Error2", func(lrpc.Call) interface{} {
		return err2
	}).
	HandleFunc("ContextRequest", func(c lrpc.Call) interface{} {
		r := ContextRequest(c.GetContext())
		return string(*r.Params)
	}),
)

var mux = lrpc.ServeMux{}.
	HandleFunc("foo", func(c lrpc.Call) interface{} { return nil })

func TestJSON2Codec(t *T) {
	rand := rand.New(rand.NewSource(time.Now().UnixNano()))

	requireJSON2Req := func(method string, args interface{}) interface{} {
		p, err := json.Marshal(args)
		require.Nil(t, err)
		pp := json.RawMessage(p)
		id := json.RawMessage(strconv.Itoa(rand.Int()))
		req := Request{
			Method: method,
			Params: &pp,
			ID:     &id,
		}

		body := new(bytes.Buffer)
		err = json.NewEncoder(body).Encode(&req)
		require.Nil(t, err, "%s", err)
		r, err := http.NewRequest("POST", "/", body)
		require.Nil(t, err)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)

		var res Response
		require.Nil(t, json.NewDecoder(w.Body).Decode(&res))
		assert.Equal(t, req.ID, res.ID)
		if res.Error != nil {
			return res.Error
		}
		return res.Result
	}
	args := map[string]string{"foo": "bar"}

	res := requireJSON2Req("Echo", args)
	assert.Equal(t, map[string]interface{}{"foo": "bar"}, res)

	res = requireJSON2Req("Error1", args)
	assert.Equal(t, &Error{Code: ErrServer, Message: "some error"}, res)

	res = requireJSON2Req("Error2", args)
	assert.Equal(t, err2, res)

	res = requireJSON2Req("ContextRequest", args)
	assert.Equal(t, `{"foo":"bar"}`, res)
}
