package httpCommand

import (
	"fmt"
	"github.com/devlibx/gox-http/v4/command"
	"github.com/go-resty/resty/v2"
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
