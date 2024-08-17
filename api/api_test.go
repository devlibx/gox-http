package goxHttpApi

import (
	"context"
	"errors"
	"github.com/devlibx/gox-base/v2/test"
	"github.com/devlibx/gox-http/v3/command"
	"github.com/devlibx/gox-http/v3/testhelper"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGoxHttpContext_WithNonExistingApiName(t *testing.T) {
	cf, _ := test.MockCf(t)

	config := command.Config{}
	err := testhelper.GetTestConfig(&config)
	assert.NoError(t, err)

	goxHttpCtx, err := NewGoxHttpContext(cf, &config)
	assert.NoError(t, err)

	_, err = goxHttpCtx.Execute(context.TODO(), command.NewGoxRequestBuilder("t").Build())
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrCommandNotRegisteredForApi))
}
