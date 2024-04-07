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
	testCases := []struct {
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

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", "/api/v1/ad", bytes.NewReader(tc.payload))
			assert.NoError(t, err)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Check the HTTP status code
			assert.Equal(t, tc.statusCode, w.Code)

			if tc.response != "" {
				// Check the HTTP response
				assert.Equal(t, tc.response, w.Body.String())
			}
		})
	}
}

func TestGetPlatformBits(t *testing.T) {
	testCases := []struct {
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

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bits := getPlatformBits(tc.platforms)
			if tc.expectedError {
				if bits != 0 {
					t.Errorf("Expected error, got bits: %d", bits)
				}
			} else {
				if bits != tc.expectedBits {
					t.Errorf("Expected bits: %d, got bits: %d", tc.expectedBits, bits)
				}
			}
		})
	}
}

func TestIsValidPlatform(t *testing.T) {
	testCases := []struct {
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

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isValidPlatform(tc.platform)
			assert.Equal(t, tc.expectedResult, result)
		})
	}
}

func TestParseListParams(t *testing.T) {
	type expected struct {
		offset   int
		limit    int
		age      int
		gender   string
		country  string
		platform string
	}

	testCases := []struct {
		name         string
		queryParams  map[string]string
		expectedErr  string
		expectedData expected
	}{
		{
			name: "ValidParams",
			queryParams: map[string]string{
				"offset":   "1",
				"limit":    "5",
				"age":      "20",
				"gender":   "M",
				"country":  "US",
				"platform": "android",
			},
			expectedErr: "",
			expectedData: expected{
				offset:   0,
				limit:    5,
				age:      20,
				gender:   "M",
				country:  "US",
				platform: "android",
			},
		},
		{
			name:        "Empty Params",
			queryParams: map[string]string{},
			expectedErr: "",
			expectedData: expected{
				offset:   0,
				limit:    5,
				age:      -1,
				gender:   "",
				country:  "",
				platform: "",
			},
		},
		{
			name: "Invalid Offset",
			queryParams: map[string]string{
				"offset":   "0",
				"limit":    "5",
				"age":      "20",
				"gender":   "M",
				"country":  "US",
				"platform": "android",
			},
			expectedErr:  "invalid offset",
			expectedData: expected{},
		},
		{
			name: "Invalid limit",
			queryParams: map[string]string{
				"offset": "1",
				"limit":  "0",
			},
			expectedErr:  "invalid limit",
			expectedData: expected{},
		},
		{
			name: "Invalid age(-1)",
			queryParams: map[string]string{
				"age": "-1",
			},
			expectedErr:  "invalid age",
			expectedData: expected{},
		},
		{
			name: "Invalid age(0)",
			queryParams: map[string]string{
				"age": "0",
			},
			expectedErr:  "invalid age",
			expectedData: expected{},
		},
		{
			name: "Invalid gender",
			queryParams: map[string]string{
				"gender": "G",
			},
			expectedErr:  "invalid gender",
			expectedData: expected{},
		},
		{
			name: "Invalid country",
			queryParams: map[string]string{
				"country": "UU",
			},
			expectedErr:  "invalid country",
			expectedData: expected{},
		},
		{
			name: "Invalid platform",
			queryParams: map[string]string{
				"platform": "aandroid",
			},
			expectedErr:  "invalid platform",
			expectedData: expected{},
		},
	}

	// Iterate over test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/path", nil)
			q := req.URL.Query()
			for key, value := range tc.queryParams {
				q.Add(key, value)
			}
			req.URL.RawQuery = q.Encode()
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			offset, limit, age, gender, country, platform, err := parseListParams(c)

			// Assertions
			if tc.expectedErr == "" {
				assert.NoError(t, err)

				assert.Equal(t, tc.expectedData.offset, offset)
				assert.Equal(t, tc.expectedData.limit, limit)
				assert.Equal(t, tc.expectedData.age, age)
				assert.Equal(t, tc.expectedData.gender, gender)
				assert.Equal(t, tc.expectedData.country, country)
				assert.Equal(t, tc.expectedData.platform, platform)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErr)
			}
		})
	}
}
