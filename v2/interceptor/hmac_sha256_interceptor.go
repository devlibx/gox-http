package interceptor

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"github.com/devlibx/gox-base/errors"
	"github.com/go-resty/resty/v2"
	"log/slog"
	"sort"
	"strings"
	"time"
)

type hmacSha256Interceptor struct {
	config    *HmacConfig
	hash      string
	timestamp time.Time
}

func (h *hmacSha256Interceptor) Info() (name string, enabled bool) {
	return "hmac-sha256", true
}

func (h *hmacSha256Interceptor) interceptRestyRequest(ctx context.Context, request *resty.Request) (requestModified bool, modifiedRequest *resty.Request, err error) {
	if h.config.Key == "" {
		return false, request, errors.New("key is missing in hmac config - we must have set secret key for hash generation")
	}

	mac := hmac.New(sha256.New, []byte(h.config.Key))

	// Build payload to calculate HMAC
	// Step 1 - add raw body to payload
	var buf bytes.Buffer
	switch b := request.Body.(type) {
	case []byte:
		buf.Write(b)
		break
	case string:
		buf.WriteString(b)
		break
	case nil:
		break
	default:
		return false, nil, errors.New("invalid body type for interceptor hmac-sha256")
	}

	// Step 2 - If we also want to append timestamp then add it
	ts := fmt.Sprintf("%d", time.Now().UnixMilli())
	// Hijack timestamp if it is set in context (used in testing)
	if t, ok := ctx.Value("__testing_ts__").(string); ok && t != "" {
		ts = t
	}
	if h.config.TimestampHeaderKey != "" {
		request.SetHeader(h.config.TimestampHeaderKey, ts)
		if buf.Len() > 0 {
			buf.WriteString("#" + ts)
		} else {
			buf.WriteString(ts)
		}
	}

	// Step 3 - add all required headers also as a part of hash payload
	headersToAppend := make([]string, 0)
	if h.config.HeadersToIncludeInSignature != nil && len(h.config.HeadersToIncludeInSignature) > 0 {

		// Make a list of <header, value> pair
		for _, header := range h.config.HeadersToIncludeInSignature {
			if h.config.ConvertHeaderKeysToLowerCase == true {
				header = strings.ToLower(header)
			}
			value := request.Header.Get(header)
			if value != "" {
				headersToAppend = append(headersToAppend, fmt.Sprintf("%s=%s", header, value))
			}
		}

		// If we have headers then sort them lexically and append them with "#"
		if len(headersToAppend) > 0 {
			sort.Strings(headersToAppend)
			buf.WriteString("#" + strings.Join(headersToAppend, "#"))
		}
	}

	// Step 4 - calculate hash
	mac.Write(buf.Bytes())
	h.hash = base64.StdEncoding.EncodeToString(mac.Sum(nil))

	// Update headers with hash
	request.SetHeader(h.config.HashHeaderKey, h.hash)
	if h.config.TimestampHeaderKey != "" {
		request.SetHeader(h.config.TimestampHeaderKey, ts)
	}

	if h.config.DumpDebug {
		slog.Debug("updated request with HMAC Sha256 hash and timestamp", "hash", h.hash, "timestamp", ts, "body", buf.String())
	}

	return true, request, nil
}

func (h *hmacSha256Interceptor) Intercept(ctx context.Context, input any) (inputModified bool, modifiedInput any, err error) {
	if r, ok := input.(*resty.Request); ok {
		return h.interceptRestyRequest(ctx, r)
	} else {
		return false, input, errors.New("input must be resty request - current implementation only supports resty request")
	}
}
