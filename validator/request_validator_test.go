package validator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/devlibx/gox-base/v2"
	httpHelper "github.com/devlibx/gox-base/v2/http_helper"
	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"log"
	"net/http"
	"testing"
	"time"
)

const port = "18123"

// User represents the structure for user input
type User struct {
	Name       string   `json:"name" binding:"required,max=10"`
	Email      string   `json:"email" binding:"required,email"`
	Address    Address  `json:"address" binding:"required"`
	AddressPtr *Address `json:"address_ptr" binding:"required"`
}

type Address struct {
	FlatNo int `json:"flat_no" binding:"required"`
}

type RequestValidatorTestSuite struct {
	suite.Suite
	server *http.Server

	portHelper    *httpHelper.PortHelper
	closeFunction func()
}

// SetupSuite runs once before the suite starts
func (s *RequestValidatorTestSuite) SetupSuite() {
	t := s.T()
	var err error

	s.portHelper, s.closeFunction, err = httpHelper.NewPortHelper()
	assert.NoError(t, err)

	// Initialize Gin router
	r := gin.Default()
	r.Use(RequestValidatorErrorHandlingMiddleware(DoNotSkipRequestValidationFunc, nil))

	// Endpoint to handle user input
	r.POST("/user", func(c *gin.Context) {
		var user User
		if err := c.ShouldBindJSON(&user); err != nil {
			_ = c.AbortWithError(http.StatusBadRequest, err)
		} else {
			c.JSON(http.StatusOK, gin.H{"message": "User input is valid", "user": user})
		}
	})
	// Endpoint to handle user input
	r.POST("/user_no_bad_request", func(c *gin.Context) {
		var user User
		if err := c.ShouldBindJSON(&user); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, map[string]string{"status": "not_ok"})
		} else {
			c.JSON(http.StatusOK, gin.H{"message": "User input is valid", "user": user})
		}
	})

	// Create a new server
	s.closeFunction()
	s.server = &http.Server{
		Addr:    fmt.Sprintf(": %d", s.portHelper.Port),
		Handler: r,
	}
	// Start the server in a goroutine
	go func() {
		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Could not listen on :%d: %v\n", s.portHelper.Port, err)
		}
	}()
}

// TearDownSuite runs once after the suite finishes
func (s *RequestValidatorTestSuite) TearDownSuite() {
	s.closeFunction()

	if err := s.server.Shutdown(context.Background()); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
}

func (s *RequestValidatorTestSuite) TestValidation_NoErrorCase() {
	// Create a Resty client
	client := resty.New()
	user := User{
		Name:       "John Doe",
		Email:      "a@b.com",
		Address:    Address{FlatNo: 1},
		AddressPtr: &Address{FlatNo: 1},
	}

	// Make a POST request to the server
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(user).
		Post(fmt.Sprintf("http://localhost:%d/user", s.portHelper.Port))
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusOK, resp.StatusCode())

}

func (s *RequestValidatorTestSuite) TestValidation_Case_1() {
	// Create a Resty client
	client := resty.New()
	user := User{
		Name: "John Doe",
	}

	// Make a POST request to the server
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(user).
		Post(fmt.Sprintf("http://localhost:%d/user", s.portHelper.Port))
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusBadRequest, resp.StatusCode())

	info := RequestValidationErrorInfo{}
	err = json.Unmarshal(resp.Body(), &info)
	assert.NoError(s.T(), err)
	assert.True(s.T(), len(info.ErrorMappingInfo) > 0)
	assert.Equal(s.T(), "Error:Field validation for 'Email' failed on the 'required' tag", info.ErrorMappingInfo["User.Email"])

	fmt.Println(resp)

	time.Sleep(100 * time.Millisecond)
}

func (s *RequestValidatorTestSuite) TestValidation_Case_NoPassingErrorInController() {
	// Create a Resty client
	client := resty.New()
	user := User{
		Name: "John Doe",
	}

	// Make a POST request to the server
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(user).
		Post(fmt.Sprintf("http://localhost:%d/user_no_bad_request", s.portHelper.Port))
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusBadRequest, resp.StatusCode())

	info := gox.StringObjectMap{}
	err = json.Unmarshal(resp.Body(), &info)
	assert.NoError(s.T(), err)
	assert.True(s.T(), len(info) > 0)
	assert.Equal(s.T(), "not_ok", info.StringOrEmpty("status"))

	time.Sleep(100 * time.Millisecond)
}

func TestRequestValidatorTestSuite(t *testing.T) {
	suite.Run(t, new(RequestValidatorTestSuite))
}
