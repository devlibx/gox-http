package validator

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"net/http"
)

type RequestValidationErrorInfo struct {
	Status           int               `json:"status,omitempty"`
	ErrorMappingInfo map[string]string `json:"error_mapping_info,omitempty"`
	GinCtx           *gin.Context      `json:"-"`
}

// SkipRequestValidationFunc is a function which can be used to skip request validation
type SkipRequestValidationFunc func(c *gin.Context) bool

// RequestValidationErrorConverter is a function which can be used to convert request validation error to any type
type RequestValidationErrorConverter func(requestValidationErrorInfo *RequestValidationErrorInfo) any

// RequestValidatorErrorHandlingMiddleware is a middleware which can be used to handle request validation error
func RequestValidatorErrorHandlingMiddleware(
	skipValidationFunction SkipRequestValidationFunc,
	requestValidationErrorConverter RequestValidationErrorConverter,
) gin.HandlerFunc {

	// If the request validation error converter is not provided then we will use the default one
	if requestValidationErrorConverter == nil {
		requestValidationErrorConverter = func(requestValidationErrorInfo *RequestValidationErrorInfo) any {
			return requestValidationErrorInfo
		}
	}

	return func(c *gin.Context) {

		// This provides the caller to skip the validation if they want to
		if skipValidationFunction == nil || skipValidationFunction(c) {
			return
		}

		// Process request - this will call the next middleware and finally get the result
		c.Next()

		// If the handler has set error during the request processing then only we want to continue
		// Otherwise we will return
		if len(c.Errors) == 0 {
			return
		}

		// We will store all the errors in this map
		errorMappingForResponse := map[string]string{}

		for _, err := range c.Errors {
			var validationErrs validator.ValidationErrors
			if errors.As(err.Err, &validationErrs) {
				for _, validationErr := range validationErrs {
					errorMappingForResponse[validationErr.Namespace()] = fmt.Sprintf("Error:Field validation for '%s' failed on the '%s' tag", validationErr.Field(), validationErr.Tag()) // validationErr.Error()
				}
			}
		}

		// If we did not find any error then we do not want to abort - if we do abort then
		// the error will be absorbed and we will miss it
		if len(errorMappingForResponse) == 0 {
			return
		}

		// Convert the error to the desired format
		toSend := requestValidationErrorConverter(&RequestValidationErrorInfo{
			Status:           http.StatusBadRequest,
			ErrorMappingInfo: errorMappingForResponse,
			GinCtx:           c,
		})
		if toSend == nil {
			return
		}

		// Return the error response in the specified format
		c.JSON(http.StatusBadRequest, toSend)
		c.Abort()
	}
}

// DoNotSkipRequestValidationFunc is a function which can be used to not skip request validation
func DoNotSkipRequestValidationFunc(c *gin.Context) bool {
	return false
}
