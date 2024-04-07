package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jjshen2000/simple-ads/models"
	"github.com/stretchr/testify/assert"
)

func TestCreateAdvertisement(t *testing.T) {
	tests := []struct {
		name       string
		payload    []byte
		statusCode int
		response   string
	}{
		{
			name: "Success",
			payload: []byte(`{
				"title": "AD 56",
				"startAt": "2023-12-10T03:00:00.000Z",
				"endAt": "2024-12-31T16:00:00.000Z",
				"conditions": [
					{
						"ageStart": 20,
						"ageEnd": 30,
						"country": ["TW", "JP"],
						"platform": ["android", "ios"]
					}
				]
			}`),
			statusCode: http.StatusCreated,
			response:   `{"message":"Advertisement created successfully"}`,
		},
		{
			name: "Invalid json",
			payload: []byte(`{
				"titletitle": "AD 56"
			}`),
			statusCode: http.StatusBadRequest,
			response:   "",
		},
		{
			name: "Invalid Advertisement (Missing Title)",
			payload: func() []byte {
				invalidAd := models.Advertisement{
					StartAt: time.Now(),
					EndAt:   time.Now().Add(24 * time.Hour),
				}
				invalidAdJSON, _ := json.Marshal(invalidAd)
				return invalidAdJSON
			}(),
			statusCode: http.StatusBadRequest,
			response:   `{"error":"Key: 'Advertisement.Title' Error:Field validation for 'Title' failed on the 'required' tag"}`,
		},
		{
			name: "Advertisement without constraint country",
			payload: []byte(`{
				"title": "AD 56",
				"startAt": "2023-12-10T03:00:00.000Z",
				"endAt": "2024-12-31T16:00:00.000Z",
				"conditions": [
					{
						"ageStart": 20,
						"ageEnd": 30,
						"country": [],
						"platform": ["android", "ios"]
					}
				]
			}`),
			statusCode: http.StatusCreated,
			response:   `{"message":"Advertisement created successfully"}`,
		},
		{
			name: "Advertisement with empty country",
			payload: []byte(`{
				"title": "AD 56",
				"startAt": "2023-12-10T03:00:00.000Z",
				"endAt": "2024-12-31T16:00:00.000Z",
				"conditions": [
					{
						"ageStart": 20,
						"ageEnd": 30,
						"platform": ["android", "ios"]
					}
				]
			}`),
			statusCode: http.StatusCreated,
			response:   `{"message":"Advertisement created successfully"}`,
		},
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.POST("/api/v1/ad", CreateAdvertisement)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", "/api/v1/ad", bytes.NewReader(tt.payload))
			assert.NoError(t, err)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Check the HTTP status code
			assert.Equal(t, tt.statusCode, w.Code)

			if tt.response != "" {
				// Check the HTTP response
				assert.Equal(t, tt.response, w.Body.String())
			}
		})
	}
}

func TestGetPlatformBits(t *testing.T) {
	tests := []struct {
		name          string
		platforms     []string
		expectedBits  uint8
		expectedError bool
	}{
		{
			name:         "No Platforms",
			platforms:    []string{},
			expectedBits: 7,
		},
		{
			name:         "Valid Platforms",
			platforms:    []string{"android", "ios"},
			expectedBits: 3,
		},
		{
			name:         "Valid Platforms",
			platforms:    []string{"android", "web"},
			expectedBits: 5,
		},
		{
			name:          "Invalid Platform",
			platforms:     []string{"android", "invalid", "web"},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bits := getPlatformBits(tt.platforms)
			if tt.expectedError {
				if bits != 0 {
					t.Errorf("Expected error, got bits: %d", bits)
				}
			} else {
				if bits != tt.expectedBits {
					t.Errorf("Expected bits: %d, got bits: %d", tt.expectedBits, bits)
				}
			}
		})
	}
}

func TestIsValidPlatform(t *testing.T) {
	tests := []struct {
		name           string
		platform       string
		expectedResult bool
	}{
		{
			name:           "Empty Platform",
			platform:       "",
			expectedResult: true,
		},
		{
			name:           "Valid Platform",
			platform:       "android",
			expectedResult: true,
		},
		{
			name:           "Invalid Platform",
			platform:       "invalid",
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidPlatform(tt.platform)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}
