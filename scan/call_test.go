package scan

import (
	"bytes"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/awserr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCallID(t *testing.T) {
	enc := base64.StdEncoding
	h := sha512.New512_256()
	h.Write([]byte(`123456789012/us-east-1/test.API?{"A":"x","B":42,"C":true}`))
	want := enc.EncodeToString(h.Sum(nil))
	c := Call{
		In: struct {
			A string
			B int
			C bool
		}{"x", 42, true},
		bat: &batch{
			ctx: &Ctx{ars: "123456789012/us-east-1/test"},
			lnk: &link{api: "API"},
		},
	}
	var b bytes.Buffer
	b.Grow(128)
	j := json.NewEncoder(&b)
	j.SetEscapeHTML(false)
	h.Reset()
	assert.Equal(t, want, c.id(&b, j, h))
	b.Reset()
	h.Reset()
	assert.Equal(t, want, c.id(&b, j, h))
	b.Reset()
	buf := b.Bytes()[:len(want)]
	require.Equal(t, []byte(want), buf)
	sum, err := enc.DecodeString(want)
	require.NoError(t, err)
	assert.Equal(t, sum, buf[len(buf):len(buf)+h.Size()])
}

func TestErr(t *testing.T) {
	orig := errors.New("short and stout")
	err := awserr.New("ErrTeapot", "I'm a teapot", orig)
	req := awserr.NewRequestFailure(err, 418, "42")
	want := Err{
		Status:    418,
		Code:      "ErrTeapot",
		Message:   "I'm a teapot",
		RequestID: "42",
		Cause:     &Err{Message: "short and stout", err: orig},
		err:       req,
	}
	assert.Equal(t, &want, decodeErr(req))
	assert.Equal(t, "short and stout", want.Cause.String())
}

func TestStats(t *testing.T) {
	var s Stats
	var req aws.Request
	var err Err

	sleep := func(secs float64) {
		d := time.Duration(secs*1e3) * time.Millisecond
		s.start = s.start.Add(-d)
		s.req = s.req.Add(-d)
	}
	equal := func(expected, actual float64) float64 {
		if assert.InDelta(t, float64(expected), float64(actual), 0.005) {
			return expected
		}
		return actual
	}

	s.ready(1)
	sleep(0.050)
	s.exec()

	s.request()
	sleep(0.100)
	s.response(&req)
	s.MinRoundTrip = equal(0.100, s.MinRoundTrip)
	s.MaxRoundTrip = equal(0.100, s.MaxRoundTrip)

	s.request()
	sleep(0.200)
	req.RetryCount = 1
	s.response(&req)
	s.MinRoundTrip = equal(0.100, s.MinRoundTrip)
	s.MaxRoundTrip = equal(0.200, s.MaxRoundTrip)

	s.request()
	sleep(0.080)
	s.response(&req)
	s.MinRoundTrip = equal(0.080, s.MinRoundTrip)
	s.MaxRoundTrip = equal(0.200, s.MaxRoundTrip)

	sleep(0.020)
	s.done(&err)

	s.QueueTime = equal(0.050, s.QueueTime)
	s.ExecTime = equal(0.400, s.ExecTime)

	want := Stats{
		Order:    1,
		Requests: 5,
		Retries:  2,
		Errors:   1,

		QueueTime:    0.050,
		ExecTime:     0.400,
		MinRoundTrip: 0.080,
		MaxRoundTrip: 0.200,

		start: s.start,
		req:   s.req,
	}
	require.Equal(t, &want, &s)

	var c Stats
	c.Combine(&s)
	s.MinRoundTrip = 0.070
	c.Combine(&s)
	want = Stats{
		Order:    -1,
		Requests: 10,
		Retries:  4,
		Errors:   2,

		QueueTime:    0.100,
		ExecTime:     0.800,
		MinRoundTrip: 0.070,
		MaxRoundTrip: 0.200,
	}
	assert.Equal(t, &want, &c)
}
