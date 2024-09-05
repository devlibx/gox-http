package httpCommand

import (
	"crypto/tls"
	"fmt"
	"github.com/devlibx/gox-http/v4/command"
	"github.com/go-resty/resty/v2"
	"net/http/httptrace"
	"sort"
	"strings"
	"time"
)

type HttpCallTracking struct {
	StartTimeOfHttpCall time.Time                `json:"start_time_of_http_call"`
	Events              []HttpCallTrackingEvents `json:"events"`
}

type HttpCallTrackingEvents struct {
	Name                                  string        `json:"name,omitempty"`
	Time                                  time.Time     `json:"time,omitempty"`
	DurationFromStartOfHttpCallToThisStep time.Duration `json:"duration_from_start_of_http_call_to_this_step,omitempty"`
}

func (h *HttpCallTracking) String() string {
	sort.Slice(h.Events, func(i, j int) bool {
		return h.Events[i].Time.Before(h.Events[j].Time)
	})

	parts := make([]string, 0)
	for _, e := range h.Events {
		parts = append(
			parts,
			fmt.Sprintf("%s: %v (at %v)", e.Name, e.DurationFromStartOfHttpCallToThisStep, e.Time.Format("15:04:05.000")),
		)
	}
	if len(parts) > 0 {
		return strings.Join(parts, " => ")
	}
	return ""
}

// HttpTrackingFunc is a callback to get the tracing info to log
type HttpTrackingFunc func(request *command.GoxRequest, r *resty.Request, fullPath string, tracingEvent HttpCallTracking)

// HttpTrackingFuncSingleton is a default implementation of HttpTrackingFunc
var HttpTrackingFuncSingleton HttpTrackingFunc = func(request *command.GoxRequest, r *resty.Request, fullPath string, tracingEvent HttpCallTracking) {
	// No Op
}

func (h *HttpCallTracking) trackHttp(request *command.GoxRequest, r *resty.Request, api *command.Api, server *command.Server) {

	// If it is disabled in server and api level then do not track it
	if api.EnableHttpConnectionTracing == false && server.EnableHttpConnectionTracing == false {
		return
	}

	// Setup tracking object
	h.StartTimeOfHttpCall = time.Now()
	h.Events = make([]HttpCallTrackingEvents, 0)

	// Function to log event
	logHttpTrackingEventFunction := func(name string) {
		now := time.Now()
		h.Events = append(h.Events, HttpCallTrackingEvents{
			Name:                                  name,
			Time:                                  now,
			DurationFromStartOfHttpCallToThisStep: now.Sub(h.StartTimeOfHttpCall),
		})
	}

	// Tracker to capture different events at each stage
	trace := &httptrace.ClientTrace{
		DNSStart: func(info httptrace.DNSStartInfo) {
			logHttpTrackingEventFunction("DnsStart")
		},
		DNSDone: func(info httptrace.DNSDoneInfo) {
			logHttpTrackingEventFunction("DnsDone")
		},
		ConnectStart: func(network, addr string) {
			logHttpTrackingEventFunction("ConnectStart")
		},
		ConnectDone: func(network, addr string, err error) {
			logHttpTrackingEventFunction("ConnectDone")
		},
		GotConn: func(info httptrace.GotConnInfo) {
			logHttpTrackingEventFunction("GotConn")
		},
		GetConn: func(hostPort string) {
			logHttpTrackingEventFunction("GetConn")
		},
		WroteHeaders: func() {
			logHttpTrackingEventFunction("WroteHeaders")
		},
		WroteRequest: func(info httptrace.WroteRequestInfo) {
			logHttpTrackingEventFunction("WroteRequest")
		},
		TLSHandshakeStart: func() {
			logHttpTrackingEventFunction("TLSHandshakeStart")
		},
		TLSHandshakeDone: func(state tls.ConnectionState, err error) {
			logHttpTrackingEventFunction("TLSHandshakeStart")
		},
		PutIdleConn: func(err error) {
			logHttpTrackingEventFunction("PutIdleConn")
		},
		GotFirstResponseByte: func() {
			logHttpTrackingEventFunction("GotFirstResponseByte")
		},
	}
	r.SetContext(httptrace.WithClientTrace(r.Context(), trace))
}
