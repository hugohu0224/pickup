package api_test

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"pickup/internal/initial"
	"testing"
)

func TestGetWebSocketURLAndConnect(t *testing.T) {
	router := initial.InitRouters()
	w := httptest.NewRecorder()

	// get websocket url
	req, _ := http.NewRequest("GET", "/v1/game/ws-url", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// response
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "url")
	assert.NotEmpty(t, response["url"])

	// connect to websocket
	wsurl := response["url"]
	req, _ = http.NewRequest("GET", wsurl, nil)
	assert.Equal(t, http.StatusOK, w.Code)

}
