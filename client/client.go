package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"time"

	"moul.io/http2curl"

	"github.com/afex/hystrix-go/hystrix"
	"github.com/go-redis/redis"
	"github.com/medicplus-inc/medicplus-kit/types"
)

// Method represents the enum for http call method
type Method string

// Enum value for http call method
const (
	POST   Method = "POST"
	PUT    Method = "PUT"
	DELETE Method = "DELETE"
	GET    Method = "GET"
	PATCH  Method = "PATCH"
)

// ResponseError represents struct of Authorization Type
type ResponseError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	StatusCode int    `json:"statusCode"`
	Error      error  `json:"error"`
	Info       string `json:"info"`
}

// AuthorizationTypeStruct represents struct of Authorization Type
type AuthorizationTypeStruct struct {
	HeaderName      string
	HeaderType      string
	HeaderTypeValue string
	Token           string
}

// AuthorizationType represents the enum for http authorization type
type AuthorizationType AuthorizationTypeStruct

// Enum value for http authorization type
var (
	Basic       = AuthorizationType(AuthorizationTypeStruct{HeaderName: "Authorization", HeaderType: "Basic", HeaderTypeValue: "Basic "})
	Bearer      = AuthorizationType(AuthorizationTypeStruct{HeaderName: "Authorization", HeaderType: "Bearer", HeaderTypeValue: "Bearer "})
	AccessToken = AuthorizationType(AuthorizationTypeStruct{HeaderName: "X-Access-Token", HeaderType: "Auth0", HeaderTypeValue: ""})
	Secret      = AuthorizationType(AuthorizationTypeStruct{HeaderName: "Secret", HeaderType: "Secret", HeaderTypeValue: ""})
	APIKey      = AuthorizationType(AuthorizationTypeStruct{HeaderName: "APIKey", HeaderType: "APIKey", HeaderTypeValue: ""})
)

//
// Private constants
//

const apiURL = "https://127.0.0.1:8080"
const defaultHTTPTimeout = 80 * time.Second
const maxNetworkRetriesDelay = 5000 * time.Millisecond
const minNetworkRetriesDelay = 500 * time.Millisecond

//
// Private variables
//

var httpClient = &http.Client{Timeout: defaultHTTPTimeout}

// GenericHTTPClient represents an interface to generalize an object to implement HTTPClient
type GenericHTTPClient interface {
	Do(req *http.Request) (string, *ResponseError)
	CallClient(ctx context.Context, path string, method Method, request interface{}, result interface{}, isAcknowledgeNeeded bool) *ResponseError
	CallClientWithCachingInRedis(ctx context.Context, durationInSecond int, path string, method Method, request interface{}, result interface{}, isAcknowledgeNeeded bool) *ResponseError
	CallClientWithCachingInRedisWithDifferentKey(ctx context.Context, durationInSecond int, path string, pathToBeStoredAsKey string, method Method, request interface{}, result interface{}, isAcknowledgeNeeded bool) *ResponseError
	CallClientWithCircuitBreaker(ctx context.Context, path string, method Method, request interface{}, result interface{}, isAcknowledgeNeeded bool) *ResponseError
	CallClientWithBaseURLGiven(ctx context.Context, url string, method Method, request interface{}, result interface{}, isAcknowledgeNeeded bool) *ResponseError
	CallClientWithRequestInBytes(ctx context.Context, path string, method Method, request []byte, result interface{}) *ResponseError
	AddAuthentication(ctx context.Context, authorizationType AuthorizationType)
}

// HTTPClient represents the service http client
type HTTPClient struct {
	redisClient        *redis.Client
	APIURL             string
	HTTPClient         *http.Client
	MaxNetworkRetries  int
	UseNormalSleep     bool
	AuthorizationTypes []AuthorizationType
	ClientName         string
}

func (c *HTTPClient) shouldRetry(err error, res *http.Response, retry int) bool {
	if retry >= c.MaxNetworkRetries {
		return false
	}

	if err != nil {
		return true
	}

	return false
}

func (c *HTTPClient) sleepTime(numRetries int) time.Duration {
	if c.UseNormalSleep {
		return 0
	}

	// exponentially backoff by 2^numOfRetries
	delay := minNetworkRetriesDelay + minNetworkRetriesDelay*time.Duration(1<<uint(numRetries))
	if delay > maxNetworkRetriesDelay {
		delay = maxNetworkRetriesDelay
	}

	// generate random jitter to prevent thundering herd problem
	jitter := rand.Int63n(int64(delay / 4))
	delay -= time.Duration(jitter)

	if delay < minNetworkRetriesDelay {
		delay = minNetworkRetriesDelay
	}

	return delay
}

// Do calls the api http request and parse the response into v
func (c *HTTPClient) Do(req *http.Request) (string, *ResponseError) {
	var res *http.Response
	var err error

	for retry := 0; ; {
		res, err = c.HTTPClient.Do(req)

		if !c.shouldRetry(err, res, retry) {
			break
		}

		sleepDuration := c.sleepTime(retry)
		retry++

		time.Sleep(sleepDuration)
	}
	if err != nil {
		return "", &ResponseError{
			Code:       "Internal Server Error",
			Message:    "Error while retry",
			Error:      err,
			StatusCode: 500,
			Info:       fmt.Sprintf("Error when retrying to call [%s]", c.APIURL),
		}
	}
	defer res.Body.Close()

	resBody, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return "", &ResponseError{
			Code:       string(res.StatusCode),
			Message:    "",
			StatusCode: res.StatusCode,
			Error:      err,
			Info:       fmt.Sprintf("Error when retrying to call [%s]", c.APIURL),
		}
	}

	errResponse := &ResponseError{
		Code:       string(res.StatusCode),
		Message:    "",
		StatusCode: res.StatusCode,
		Error:      nil,
		Info:       "",
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		err = json.Unmarshal([]byte(string(resBody)), errResponse)
		if err != nil {
			errResponse.Error = err
		}
		if errResponse.Info != "" {
			errResponse.Message = errResponse.Info
		}
		errResponse.Error = fmt.Errorf("Error while calling %s: %v", req.URL.String(), errResponse.Message)

		return "", errResponse
	}

	return string(resBody), errResponse
}

// CallClient do call client
func (c *HTTPClient) CallClient(ctx context.Context, path string, method Method, request interface{}, result interface{}, isAcknowledgeNeeded bool) *ResponseError {
	var jsonData []byte
	var err error
	var response string
	var errDo *ResponseError

	if request != nil && request != "" {
		jsonData, err = json.Marshal(request)
		if err != nil {
			errDo = &ResponseError{
				Error: err,
			}
			return errDo
		}
	}

	urlPath, err := url.Parse(fmt.Sprintf("%s/%s", c.APIURL, path))
	if err != nil {
		errDo = &ResponseError{
			Error: err,
		}
		return errDo
	}

	req, err := http.NewRequest(string(method), urlPath.String(), bytes.NewBuffer(jsonData))
	if err != nil {
		errDo = &ResponseError{
			Error: err,
		}
		return errDo
	}

	for _, authorizationType := range c.AuthorizationTypes {
		if authorizationType.HeaderType != "APIKey" {
			req.Header.Add(authorizationType.HeaderName, fmt.Sprintf("%s%s", authorizationType.HeaderTypeValue, authorizationType.Token))
		}
	}
	req.Header.Add("Content-Type", "application/json")

	requestRaw := types.Metadata{}
	if request != nil && request != "" {
		err = json.Unmarshal(jsonData, &requestRaw)
		if err != nil {
			errDo = &ResponseError{
				Error: err,
			}
			return errDo
		}
	}

	response, errDo = c.Do(req)

	//TODO change logging to store in elastic search / hadoop
	command, _ := http2curl.GetCurlCommand(req)
	log.Printf(`
	[Calling %s] 
	curl: %v
	response: %v 
	`, c.ClientName, command.String(), response)

	if errDo != nil && (errDo.Error != nil || errDo.Message != "") {
		return errDo
	}

	if response != "" && result != nil {
		err = json.Unmarshal([]byte(response), result)
		if err != nil {
			errDo = &ResponseError{
				Error: err,
			}
			return errDo
		}
	}

	return errDo
}

// CallClientWithCachingInRedis call client with caching in redis
func (c *HTTPClient) CallClientWithCachingInRedis(ctx context.Context, durationInSecond int, path string, method Method, request interface{}, result interface{}, isAcknowledgeNeeded bool) *ResponseError {
	var jsonData []byte
	var err error
	var response string
	var errDo *ResponseError

	if request != nil && request != "" {
		jsonData, err = json.Marshal(request)
		if err != nil {
			errDo = &ResponseError{
				Error: err,
			}
			return errDo
		}
	}

	urlPath, err := url.Parse(fmt.Sprintf("%s/%s", c.APIURL, path))
	if err != nil {
		errDo = &ResponseError{
			Error: err,
		}
		return errDo
	}

	//collect from redis if already exist
	val, errRedis := c.redisClient.Get("apicaching:" + urlPath.String()).Result()
	if errRedis != nil {
		log.Printf(`
		======================================================================
		Error Collecting Caching in "CallClientWithCachingInRedis":
		"key": %s
		Error: %v
		======================================================================
		`, "apicaching:"+urlPath.String(), errRedis)
	}

	if val != "" {
		isSuccess := true
		if errJSON := json.Unmarshal([]byte(val), &result); errJSON != nil {
			log.Printf(`
			======================================================================
			Error Collecting Caching in "CallClientWithCachingInRedis":
			"key": %s,
			Error: %v,
			======================================================================
			`, "apicaching:"+urlPath.String(), errJSON)
			isSuccess = false
		}
		if isSuccess {
			return nil
		}
	}

	req, err := http.NewRequest(string(method), urlPath.String(), bytes.NewBuffer(jsonData))
	if err != nil {
		errDo = &ResponseError{
			Error: err,
		}
		return errDo
	}

	for _, authorizationType := range c.AuthorizationTypes {
		if authorizationType.HeaderType != "APIKey" {
			req.Header.Add(authorizationType.HeaderName, fmt.Sprintf("%s%s", authorizationType.HeaderTypeValue, authorizationType.Token))
		}
	}
	req.Header.Add("Content-Type", "application/json")

	response, errDo = c.Do(req)

	//TODO change logging to store in elastic search / hadoop
	command, _ := http2curl.GetCurlCommand(req)
	log.Printf(`
	[Calling %s] 
	curl: %v
	response: %v 
	`, c.ClientName, command.String(), response)

	if errDo != nil && (errDo.Error != nil || errDo.Message != "") {
		return errDo
	}

	if response != "" && result != nil {
		err = json.Unmarshal([]byte(response), result)
		if err != nil {
			errDo = &ResponseError{
				Error: err,
			}
			return errDo
		}

		if errRedis = c.redisClient.Set(
			fmt.Sprintf("%s:%s", "apicaching", urlPath.String()),
			response,
			time.Second*time.Duration(durationInSecond),
		).Err(); err != nil {
			log.Printf(`
			======================================================================
			Error Storing Caching in "CallClientWithCachingInRedis":
			"key": %s,
			Error: %v,
			======================================================================
			`, "apicaching:"+urlPath.String(), err)
		}
	}

	return errDo
}

// CallClientWithCachingInRedisWithDifferentKey call client with caching in redis with different key
func (c *HTTPClient) CallClientWithCachingInRedisWithDifferentKey(ctx context.Context, durationInSecond int, path string, pathToBeStoredAsKey string, method Method, request interface{}, result interface{}, isAcknowledgeNeeded bool) *ResponseError {
	var jsonData []byte
	var err error
	var response string
	var errDo *ResponseError

	if request != nil && request != "" {
		jsonData, err = json.Marshal(request)
		if err != nil {
			errDo = &ResponseError{
				Error: err,
			}
			return errDo
		}
	}

	urlPath, err := url.Parse(fmt.Sprintf("%s/%s", c.APIURL, path))
	if err != nil {
		errDo = &ResponseError{
			Error: err,
		}
		return errDo
	}

	urlPathToBeStored, err := url.Parse(fmt.Sprintf("%s/%s", c.APIURL, pathToBeStoredAsKey))
	if err != nil {
		errDo = &ResponseError{
			Error: err,
		}
		return errDo
	}

	//collect from redis if already exist
	val, errRedis := c.redisClient.Get("apicaching:" + urlPathToBeStored.String()).Result()
	if errRedis != nil {
		log.Printf(`
		======================================================================
		Error Collecting Caching in "CallClientWithCachingInRedisWithDifferentKey":
		"key": %s
		Error: %v
		======================================================================
		`, "apicaching:"+urlPathToBeStored.String(), errRedis)
	}

	if val != "" {
		isSuccess := true
		if errJSON := json.Unmarshal([]byte(val), &result); errJSON != nil {
			log.Printf(`
			======================================================================
			Error Collecting Caching in "CallClientWithCachingInRedisWithDifferentKey":
			"key": %s,
			Error: %v,
			======================================================================
			`, "apicaching:"+urlPathToBeStored.String(), errJSON)
			isSuccess = false
		}
		if isSuccess {
			return nil
		}
	}

	req, err := http.NewRequest(string(method), urlPath.String(), bytes.NewBuffer(jsonData))
	if err != nil {
		errDo = &ResponseError{
			Error: err,
		}
		return errDo
	}

	for _, authorizationType := range c.AuthorizationTypes {
		if authorizationType.HeaderType != "APIKey" {
			req.Header.Add(authorizationType.HeaderName, fmt.Sprintf("%s%s", authorizationType.HeaderTypeValue, authorizationType.Token))
		}
	}
	req.Header.Add("Content-Type", "application/json")

	response, errDo = c.Do(req)

	//TODO change logging to store in elastic search / hadoop
	command, _ := http2curl.GetCurlCommand(req)
	log.Printf(`
	[Calling %s] 
	curl: %v
	response: %v 
	`, c.ClientName, command.String(), response)

	if errDo != nil && (errDo.Error != nil || errDo.Message != "") {
		return errDo
	}

	if response != "" && result != nil {
		err = json.Unmarshal([]byte(response), result)
		if err != nil {
			errDo = &ResponseError{
				Error: err,
			}
			return errDo
		}

		if errRedis = c.redisClient.Set(
			fmt.Sprintf("%s:%s", "apicaching", urlPathToBeStored.String()),
			response,
			time.Second*time.Duration(durationInSecond),
		).Err(); err != nil {
			log.Printf(`
			======================================================================
			Error Storing Caching in "CallClientWithCachingInRedisWithDifferentKey":
			"key": %s,
			Error: %v,
			======================================================================
			`, "apicaching:"+urlPathToBeStored.String(), err)
		}
	}

	return errDo
}

// CallClientWithCircuitBreaker do call client with circuit breaker (async)
func (c *HTTPClient) CallClientWithCircuitBreaker(ctx context.Context, path string, method Method, request interface{}, result interface{}, isAcknowledgeNeeded bool) *ResponseError {
	var jsonData []byte
	var err error
	var response string
	var errDo *ResponseError

	Sethystrix(c.ClientName)
	err = hystrix.Do(c.ClientName, func() error {
		if request != nil {
			jsonData, err = json.Marshal(request)
			if err != nil {
				errDo = &ResponseError{
					Error: err,
				}
				return errDo.Error
			}
		}

		urlPath, err := url.Parse(fmt.Sprintf("%s/%s", c.APIURL, path))
		if err != nil {
			errDo = &ResponseError{
				Error: err,
			}
			return errDo.Error
		}

		req, err := http.NewRequest(string(method), urlPath.String(), bytes.NewBuffer(jsonData))
		if err != nil {
			errDo = &ResponseError{
				Error: err,
			}
			return errDo.Error
		}

		for _, authorizationType := range c.AuthorizationTypes {
			if authorizationType.HeaderType != "APIKey" {
				req.Header.Add(authorizationType.HeaderName, fmt.Sprintf("%s%s", authorizationType.HeaderTypeValue, authorizationType.Token))
			}
		}
		req.Header.Add("Content-Type", "application/json")

		response, errDo = c.Do(req)

		//TODO change logging to store in elastic search / hadoop
		command, _ := http2curl.GetCurlCommand(req)
		log.Printf(`
	[Calling %s] 
	curl: %v
	response: %v 
	`, c.ClientName, command.String(), response)

		if errDo != nil && (errDo.Error != nil || errDo.Message != "") {
			return errDo.Error
		}

		if response != "" && result != nil {
			err = json.Unmarshal([]byte(response), result)
			if err != nil {
				errDo = &ResponseError{
					Error: err,
				}
				return errDo.Error
			}
		}
		return nil
	}, nil)

	return errDo
}

// CallClientWithBaseURLGiven do call client with base url given
func (c *HTTPClient) CallClientWithBaseURLGiven(ctx context.Context, url string, method Method, request interface{}, result interface{}, isAcknowledgeNeeded bool) *ResponseError {
	var jsonData []byte
	var err error
	var response string
	var errDo *ResponseError

	if request != nil && request != "" {
		jsonData, err = json.Marshal(request)
		if err != nil {
			errDo = &ResponseError{
				Error: err,
			}
			return errDo
		}
	}

	req, err := http.NewRequest(string(method), url, bytes.NewBuffer(jsonData))
	if err != nil {
		errDo = &ResponseError{
			Error: err,
		}
		return errDo
	}

	for _, authorizationType := range c.AuthorizationTypes {
		if authorizationType.HeaderType != "APIKey" {
			req.Header.Add(authorizationType.HeaderName, fmt.Sprintf("%s%s", authorizationType.HeaderTypeValue, authorizationType.Token))
		}
	}
	req.Header.Add("Content-Type", "application/json")

	response, errDo = c.Do(req)

	//TODO change logging to store in elastic search / hadoop
	command, _ := http2curl.GetCurlCommand(req)
	log.Printf(`
	[Calling %s] 
	curl: %v
	response: %v 
	`, c.ClientName, command.String(), response)

	if errDo != nil && (errDo.Error != nil || errDo.Message != "") {
		return errDo
	}

	if response != "" && result != nil {
		err = json.Unmarshal([]byte(response), result)
		if err != nil {
			errDo = &ResponseError{
				Error: err,
			}
			return errDo
		}
	}

	return errDo
}

// CallClientWithRequestInBytes do call client with request in bytes and omit acknowledge process - specific case for consumer
func (c *HTTPClient) CallClientWithRequestInBytes(ctx context.Context, path string, method Method, request []byte, result interface{}) *ResponseError {
	var jsonData []byte = request
	var err error
	var response string
	var errDo *ResponseError

	urlPath, err := url.Parse(fmt.Sprintf("%s/%s", c.APIURL, path))
	if err != nil {
		errDo = &ResponseError{
			Error: err,
		}
		return errDo
	}

	req, err := http.NewRequest(string(method), urlPath.String(), bytes.NewBuffer(jsonData))
	if err != nil {
		errDo = &ResponseError{
			Error: err,
		}
		return errDo
	}

	for _, authorizationType := range c.AuthorizationTypes {
		if authorizationType.HeaderType != "APIKey" {
			req.Header.Add(authorizationType.HeaderName, fmt.Sprintf("%s%s", authorizationType.HeaderTypeValue, authorizationType.Token))
		}
	}
	req.Header.Add("Content-Type", "application/json")

	response, errDo = c.Do(req)

	//TODO change logging to store in elastic search / hadoop
	command, _ := http2curl.GetCurlCommand(req)
	log.Printf(`
	[Calling %s] 
	curl: %v
	response: %v 
	`, c.ClientName, command.String(), response)

	if errDo != nil && (errDo.Error != nil || errDo.Message != "") {
		return errDo
	}

	if response != "" && result != nil {
		err = json.Unmarshal([]byte(response), result)
		if err != nil {
			errDo = &ResponseError{
				Error: err,
			}
			return errDo
		}
	}

	return errDo
}

// AddAuthentication do add authentication
func (c *HTTPClient) AddAuthentication(ctx context.Context, authorizationType AuthorizationType) {
	isExist := false
	for key, singleAuthorizationType := range c.AuthorizationTypes {
		if singleAuthorizationType.HeaderType == authorizationType.HeaderType {
			c.AuthorizationTypes[key].Token = authorizationType.Token
			isExist = true
			break
		}
	}

	if isExist == false {
		c.AuthorizationTypes = append(c.AuthorizationTypes, authorizationType)
	}
}

// NewHTTPClient creates the new http client
func NewHTTPClient(
	config HTTPClient,
	redisClient *redis.Client,
) *HTTPClient {
	if config.HTTPClient == nil {
		config.HTTPClient = httpClient
	}

	if config.APIURL == "" {
		config.APIURL = apiURL
	}

	return &HTTPClient{
		APIURL:             config.APIURL,
		HTTPClient:         config.HTTPClient,
		MaxNetworkRetries:  config.MaxNetworkRetries,
		UseNormalSleep:     config.UseNormalSleep,
		AuthorizationTypes: config.AuthorizationTypes,
		ClientName:         config.ClientName,
		redisClient:        redisClient,
	}
}

// Sethystrix setting for client
func Sethystrix(nameClient string) {
	hystrix.ConfigureCommand(nameClient, hystrix.CommandConfig{
		Timeout:               5000,
		MaxConcurrentRequests: 100,
		ErrorPercentThreshold: 20,
	})
}
