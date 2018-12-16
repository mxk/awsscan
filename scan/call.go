package scan

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"hash"
	"math"
	"reflect"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/awserr"
)

// Call is one specific instance of an API call.
type Call struct {
	ID    string         `json:"-"`                // Unique call ID
	Stats *Stats         `json:"#stats,omitempty"` // Call statistics
	Src   map[string]int `json:"src,omitempty"`    // Source call IDs
	In    interface{}    `json:"in,omitempty"`     // API *Input struct
	Out   []interface{}  `json:"out,omitempty"`    // API *Output struct
	Err   *Err           `json:"err,omitempty"`    // Decoded error

	bat     *batch
	req     *aws.Request
	skipOut bitSet
}

// id generates a base64-encoded SHA-512/256 call ID. The hashed string is:
//
//	"<account>/<region>/<service>.<api>?<json-encoded-input>"
//
// JSON encoding of input is not compacted (every field is present). This format
// ensures that IDs are unique within the document (there should not be multiple
// API calls with identical parameters), yet stable across multiple scans.
func (c *Call) id(b *bytes.Buffer, j *json.Encoder, h hash.Hash) string {
	b.WriteString(c.bat.ctx.ars)
	b.WriteByte('.')
	b.WriteString(c.bat.lnk.api)
	b.WriteByte('?')
	j.Encode(c.In)
	h.Write(b.Bytes()[:b.Len()-1]) // Hash without the trailing '\n'
	b.Reset()
	enc := base64.StdEncoding
	buf := b.Bytes()[:enc.EncodedLen(h.Size())]
	enc.Encode(buf, h.Sum(buf[len(buf):]))
	return string(buf)
}

// exec makes the API call, appends output to c.Out, and sets c.err on error.
func (c *Call) exec() {
	// Copy input struct to avoid modifying the original during pagination
	in := []reflect.Value{c.bat.ctx.client, reflect.ValueOf(c.In)}
	cpy := reflect.New(in[1].Type().Elem())
	if !in[1].IsNil() {
		cpy.Elem().Set(in[1].Elem())
	}
	in[1] = cpy

	// TODO: Let scanner deal with throttling, don't block workers

	// Pager also works for non-paginated APIs
	p := aws.Pager{NewRequest: func() (*aws.Request, error) {
		c.req = c.bat.lnk.req.Call(in)[0].Field(0).Interface().(*aws.Request)
		c.bat.ctx.iface.UpdateRequest(c.req)
		c.Stats.request()
		return c.req, nil
	}}
	for p.Next() {
		c.Stats.response(c.req)
		c.Out = append(c.Out, p.CurrentPage())
	}
	if c.Err = decodeErr(p.Err()); c.Err != nil {
		c.Stats.response(c.req)
	}
}

// Err contains information about an API call error.
type Err struct {
	Status    int    // HTTP status code
	Code      string // AWS error code
	Message   string // Error message
	RequestID string // AWS request ID
	Ignore    bool   // Error was expected, safe to ignore

	Cause *Err `json:",omitempty"` // Original cause

	err error // Original error
}

// decodeErr converts a non-nil err into a new Err instance.
func decodeErr(err error) *Err {
	if err == nil {
		return nil
	}
	if err, ok := err.(awserr.Error); ok {
		e := &Err{
			Code:    err.Code(),
			Message: err.Message(),
			Cause:   decodeErr(err.OrigErr()),
			err:     err,
		}
		if err, ok := err.(awserr.RequestFailure); ok {
			e.Status = err.StatusCode()
			e.RequestID = err.RequestID()
		}
		return e
	}
	return &Err{Message: err.Error(), err: err}
}

// String implements fmt.Stringer interface.
func (e *Err) String() string {
	return e.err.Error()
}

// Stats contains performance information for one or more calls. All times are
// in seconds.
type Stats struct {
	Order    int // Call issue order (-1 for combined stats)
	Requests int // Total number of requests
	Retries  int // Number of retried requests
	Errors   int // Number of terminal, non-ignored errors

	QueueTime    float64 // Time spent waiting for a worker
	ExecTime     float64 // Execution time (to worker and back)
	MinRoundTrip float64 // Fastest response time
	MaxRoundTrip float64 // Slowest response time

	start time.Time // Current state start time (queue/exec)
	req   time.Time // Current request start time
}

// Combine updates s with stats from t.
func (s *Stats) Combine(t *Stats) {
	if s == nil || t == nil {
		return
	}
	s.Order = -1
	s.Requests += t.Requests
	s.Retries += t.Retries
	s.Errors += t.Errors
	s.QueueTime += t.QueueTime
	s.ExecTime += t.ExecTime
	if t.MaxRoundTrip > s.MaxRoundTrip {
		if s.MaxRoundTrip == 0 {
			s.MinRoundTrip = t.MinRoundTrip
		}
		s.MaxRoundTrip = t.MaxRoundTrip
	}
	if t.MinRoundTrip < s.MinRoundTrip {
		s.MinRoundTrip = t.MinRoundTrip
	}
}

// RoundTimes rounds all times to the nearest millisecond.
func (s *Stats) RoundTimes() {
	if s != nil {
		round := func(t float64) float64 { return math.Round(t*1e3) / 1e3 }
		s.QueueTime = round(s.QueueTime)
		s.ExecTime = round(s.ExecTime)
		s.MinRoundTrip = round(s.MinRoundTrip)
		s.MaxRoundTrip = round(s.MaxRoundTrip)
	}
}

// ready starts the queue timer.
func (s *Stats) ready(order int) {
	if s != nil {
		s.start = time.Now()
		s.Order = order
	}
}

// exec starts the execution timer.
func (s *Stats) exec() {
	if s != nil {
		now := time.Now()
		s.QueueTime = now.Sub(s.start).Seconds()
		s.start = now
	}
}

// request starts the round trip timer.
func (s *Stats) request() {
	if s != nil {
		s.req = time.Now()
		s.Requests++
	}
}

// response calculates round trip time.
func (s *Stats) response(req *aws.Request) {
	if s != nil {
		if d := time.Since(s.req).Seconds(); d > s.MaxRoundTrip {
			if s.MaxRoundTrip == 0 {
				s.MinRoundTrip = d
			}
			s.MaxRoundTrip = d
		} else if d < s.MinRoundTrip {
			s.MinRoundTrip = d
		}
		// TODO: Remove when retries are handled by the scheduler
		s.Requests += req.RetryCount
		s.Retries += req.RetryCount
	}
}

// done marks the call as finished.
func (s *Stats) done(err *Err) {
	if s != nil {
		s.ExecTime = time.Since(s.start).Seconds()
		if err != nil && !err.Ignore {
			s.Errors++
		}
	}
}
