package interceptor

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"github.com/devlibx/gox-base/errors"
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

func (h *hmacSha256Interceptor) Intercept(ctx context.Context, body any) (bodyModified bool, modifiedBody any, err error) {
	mac := hmac.New(sha256.New, []byte(h.config.Key))
	var buf bytes.Buffer
	switch b := body.(type) {
	case []byte:
		buf.Write(b)
		break
	case string:
		buf.WriteString(b)
		break
	default:
		return false, nil, errors.New("invalid body type for interceptor hmac-sha256")
	}

	// Calculate HMAC and keep it to enrich headers
	mac.Write(buf.Bytes())
	h.hash = hex.EncodeToString(mac.Sum(nil))
	return false, body, nil
}

func (h *hmacSha256Interceptor) EnrichHeaders(ctx context.Context) (headers map[string]string, err error) {
	return map[string]string{
		h.config.HashHeaderKey: h.hash,
	}, nil
}
