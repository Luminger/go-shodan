package shodan

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"time"
)

const (
	testClientToken = "TEST_TOKEN"
	stubsDir        = "stubs"
)

var (
	mux    *http.ServeMux
	server *httptest.Server
	client *Client
)

func setUpTestServe() {
	mux = http.NewServeMux()
	server = httptest.NewServer(mux)
	client = NewClient(nil, testClientToken)
	client.BaseURL = server.URL
	client.ExploitBaseURL = server.URL
	client.StreamBaseURL = server.URL
}

func getStub(t *testing.T, stubName string) []byte {
	stubPath := fmt.Sprintf("%s/%s.json", stubsDir, stubName)
	content, err := ioutil.ReadFile(stubPath)
	if err != nil {
		t.Errorf("getStub error %v", err)
	}

	return content
}

func tearDownTestServe() {
	server.Close()
}

func TestNewClient(t *testing.T) {
	client := NewClient(nil, testClientToken)
	assert.Equal(t, testClientToken, client.Token)
}

func TestNewClient_httpClient(t *testing.T) {
	transport := &http.Transport{Proxy: http.ProxyFromEnvironment}
	httpClient := &http.Client{Transport: transport}
	client := NewClient(httpClient, testClientToken)
	assert.ObjectsAreEqual(httpClient, client.Client)
}

func TestClient_buildURL_success(t *testing.T) {
	client := NewClient(nil, testClientToken)
	testOptions := struct {
		Page    int  `url:"page"`
		ShowAll bool `url:"show_all"`
	}{
		100,
		true,
	}
	testCases := []struct {
		path     string
		params   interface{}
		expected string
	}{
		{
			"/testing/test/1",
			nil,
			baseURL + "/testing/test/1?key=" + testClientToken,
		},
		{
			"/testing/test/2",
			testOptions,
			baseURL + "/testing/test/2?key=" + testClientToken + "&page=100&show_all=true",
		},
	}

	for _, caseParams := range testCases {
		url := client.buildURL(baseURL, caseParams.path, caseParams.params)

		assert.Equal(t, caseParams.expected, url)
	}
}

func TestClient_buildBaseURL(t *testing.T) {
	client := NewClient(nil, testClientToken)
	expected := client.BaseURL + "/test-base-url-building/?key=" + testClientToken
	actual := client.buildBaseURL("/test-base-url-building/", nil)

	assert.Equal(t, expected, actual)
}

func TestClient_buildExploitBaseURL(t *testing.T) {
	client := NewClient(nil, testClientToken)
	expected := client.ExploitBaseURL + "/test-exploit-url-building/?key=" + testClientToken
	actual := client.buildExploitBaseURL("/test-exploit-url-building/", nil)

	assert.Equal(t, expected, actual)
}

func TestClient_buildStreamBaseURL(t *testing.T) {
	client := NewClient(nil, testClientToken)
	expected := client.StreamBaseURL + "/test-stream-url-building/?key=" + testClientToken
	actual := client.buildStreamBaseURL("/test-stream-url-building/", nil)

	assert.Equal(t, expected, actual)
}

func TestClient_sendRequest_invalidURL(t *testing.T) {
	client := NewClient(nil, testClientToken)
	_, err := client.sendRequest("GET", ":/1232.22", nil)
	assert.NotNil(t, err)
}

func TestClient_executeRequest_textUnauthorized(t *testing.T) {
	setUpTestServe()
	defer tearDownTestServe()

	unauthorizedPath := "/http-error/401"

	errorText := "401 Unauthorized\n\n"
	errorText += "This server could not verify that you are authorized to access the document you requested.  " +
		"Either you supplied the wrong credentials (e.g., bad password), or your browser does not understand how to " +
		"supply the credentials required."

	mux.HandleFunc(unauthorizedPath, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, errorText, http.StatusUnauthorized)
	})

	url := client.buildBaseURL(unauthorizedPath, nil)
	err := client.executeRequest("GET", url, nil, nil)

	assert.NotNil(t, err)
}

func TestClient_executeRequest_jsonNotFound(t *testing.T) {
	setUpTestServe()
	defer tearDownTestServe()

	notFoundPath := "/http-error/404"

	mux.HandleFunc(notFoundPath, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error": "No information available for that IP."}`, http.StatusNotFound)
	})

	url := client.buildBaseURL(notFoundPath, nil)
	err := client.executeRequest("GET", url, nil, nil)

	assert.NotNil(t, err)
}

func TestClient_executeStreamRequest_success(t *testing.T) {
	setUpTestServe()
	defer tearDownTestServe()

	streamPath := "/stream/success"
	chunkLimit := 3

	mux.HandleFunc(streamPath, func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Errorf("Cannot use Flush")
		}

		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		for i := 0; i < chunkLimit; i++ {
			fmt.Fprintln(w, "chunk")
			flusher.Flush()
			time.Sleep(time.Millisecond * 500)
		}
	})

	url := client.buildStreamBaseURL(streamPath, nil)

	bytesChan := make(chan []byte)
	err := client.executeStreamRequest("GET", url, bytesChan)
	assert.Nil(t, err)

	receivedChunks := 0

	for {
		msg, open := <-bytesChan
		if !open {
			break
		}
		assert.NotEmpty(t, msg)
		receivedChunks++
	}

	assert.Equal(t, chunkLimit, receivedChunks)
}

func TestClient_executeStreamRequest_errorRequest(t *testing.T) {
	client := NewClient(nil, testClientToken)
	url := client.buildStreamBaseURL("/stream/error", nil)

	bytesChan := make(chan []byte)
	err := client.executeStreamRequest("GET", url, bytesChan)

	assert.NotNil(t, err)
}
