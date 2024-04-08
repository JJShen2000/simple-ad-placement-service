package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"

	// "strconv"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/gin-gonic/gin"
	"github.com/jjshen2000/simple-ads/models"
	"github.com/stretchr/testify/assert"
)

var schema = `
DROP TABLE IF EXISTS condition_country;

DROP TABLE IF EXISTS advertisement_condition;

DROP TABLE IF EXISTS advertisement;

CREATE TABLE advertisement (
    id INT AUTO_INCREMENT PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    start_at DATETIME NOT NULL,
    end_at DATETIME NOT NULL
)

CREATE TABLE advertisement_condition (
    id INT AUTO_INCREMENT PRIMARY KEY,
    advertisement_id INT NOT NULL,
    age_start TINYINT UNSIGNED, -- 0-100
    age_end TINYINT UNSIGNED,   -- 0-100
    gender CHAR(2),             -- M, F, MF
	unlimited_country BOOL,
    platform TINYINT UNSIGNED,  -- bit-wise 'android', 'ios', 'web'
    FOREIGN KEY (advertisement_id) REFERENCES advertisement(id)
);

CREATE TABLE condition_country (
    condition_id INT,
    country_code CHAR(2), -- ISO-3166 alpha 2 code
    KEY (condition_id, country_code),
    FOREIGN KEY (condition_id) REFERENCES advertisement_condition(id)
);

CREATE INDEX idx_start_at ON advertisement (start_at);

CREATE INDEX idx_end_at ON advertisement (end_at);

CREATE INDEX idx_advertisement_id ON advertisement_condition (advertisement_id);
`

func testDbInit() *sqlx.DB {
	testDB, err := sqlx.Connect("mysql", "root:testpassword@tcp(localhost:3309)/ads")
	if err != nil {
		log.Fatalln("Failed to connect to test MySQL:", err)
	}
	queries := strings.Split(schema, "\n\n")

	for _, query := range queries {
		testDB.Exec(query)
	}
	return testDB
}

func TestCreateAdvertisement(t *testing.T) {
	// testDB := testDbInit()

	// defer testDB.Close()

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

	router.POST("/api/v1/ad", CreateAdvertisementHandler(nil))

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
			time.Sleep(100 * time.Millisecond)
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
	// type expected struct {
	// 	offset   int
	// 	limit    int
	// 	age      int
	// 	gender   string
	// 	country  string
	// 	platform string
	// }

	testCases := []struct {
		name         string
		queryParams  map[string]string
		expectedErr  string
		expectedData listParams
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
			expectedData: listParams{
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
			expectedData: listParams{
				offset:   0,
				limit:    5,
				age:      0,
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
			expectedData: listParams{},
		},
		{
			name: "Invalid limit",
			queryParams: map[string]string{
				"offset": "1",
				"limit":  "0",
			},
			expectedErr:  "invalid limit",
			expectedData: listParams{},
		},
		{
			name: "Invalid age(-1)",
			queryParams: map[string]string{
				"age": "-1",
			},
			expectedErr:  "invalid age",
			expectedData: listParams{},
		},
		{
			name: "Invalid age(0)",
			queryParams: map[string]string{
				"age": "0",
			},
			expectedErr:  "invalid age",
			expectedData: listParams{},
		},
		{
			name: "Invalid gender",
			queryParams: map[string]string{
				"gender": "G",
			},
			expectedErr:  "invalid gender",
			expectedData: listParams{},
		},
		{
			name: "Invalid country",
			queryParams: map[string]string{
				"country": "UU",
			},
			expectedErr:  "invalid country",
			expectedData: listParams{},
		},
		{
			name: "Invalid platform",
			queryParams: map[string]string{
				"platform": "aandroid",
			},
			expectedErr:  "invalid platform",
			expectedData: listParams{},
		},
	}

	// Iterate over test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/path", http.NoBody)
			q := req.URL.Query()
			for key, value := range tc.queryParams {
				q.Add(key, value)
			}
			req.URL.RawQuery = q.Encode()
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			params, err := parseListParams(c)

			// Assertions
			if tc.expectedErr == "" {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedData, params)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErr)
			}
		})
	}
}

func TestBuildQuery(t *testing.T) {
	testCases := []struct {
		name         string
		params       listParams
		expectedSQL  string
		expectedArgs []interface{}
	}{
		{
			name: "NoFilters",
			params: listParams{
				offset:   0,
				limit:    10,
				age:      0,
				gender:   "",
				country:  "",
				platform: "",
			},
			expectedSQL: `SELECT a.title, a.end_at FROM advertisement AS a
 WHERE NOW() < a.end_at AND NOW() > a.start_at ORDER BY end_at ASC LIMIT ? OFFSET ?`,
			expectedArgs: []interface{}{10, 0},
		},
		{
			name: "Age 20",
			params: listParams{
				offset:   0,
				limit:    10,
				age:      20,
				gender:   "",
				country:  "",
				platform: "",
			},
			expectedSQL: `SELECT a.title, a.end_at FROM advertisement AS a
 INNER JOIN advertisement_condition AS ac ON a.id = ac.advertisement_id
 WHERE NOW() < a.end_at AND NOW() > a.start_at AND ? BETWEEN ac.age_start AND ac.age_end ORDER BY end_at ASC LIMIT ? OFFSET ?`,
			expectedArgs: []interface{}{20, 10, 0},
		},
		{
			name: "gender F",
			params: listParams{
				offset:   0,
				limit:    10,
				age:      0,
				gender:   "F",
				country:  "",
				platform: "",
			},
			expectedSQL: `SELECT a.title, a.end_at FROM advertisement AS a
 INNER JOIN advertisement_condition AS ac ON a.id = ac.advertisement_id
 WHERE NOW() < a.end_at AND NOW() > a.start_at AND ac.gender != ? ORDER BY end_at ASC LIMIT ? OFFSET ?`,
			expectedArgs: []interface{}{"M", 10, 0},
		},
		{
			name: "gender M",
			params: listParams{
				offset:   0,
				limit:    10,
				age:      0,
				gender:   "M",
				country:  "",
				platform: "",
			},
			expectedSQL: `SELECT a.title, a.end_at FROM advertisement AS a
 INNER JOIN advertisement_condition AS ac ON a.id = ac.advertisement_id
 WHERE NOW() < a.end_at AND NOW() > a.start_at AND ac.gender != ? ORDER BY end_at ASC LIMIT ? OFFSET ?`,
			expectedArgs: []interface{}{"F", 10, 0},
		},
		{
			name: "country TW",
			params: listParams{
				offset:   0,
				limit:    10,
				age:      0,
				gender:   "",
				country:  "TW",
				platform: "",
			},
			expectedSQL: `SELECT a.title, a.end_at FROM advertisement AS a
 INNER JOIN advertisement_condition AS ac ON a.id = ac.advertisement_id
 INNER JOIN condition_country AS cc ON ac.id = cc.condition_id
 WHERE NOW() < a.end_at AND NOW() > a.start_at AND cc.country_code = ? ORDER BY end_at ASC LIMIT ? OFFSET ?`,
			expectedArgs: []interface{}{"TW", 10, 0},
		},
		{
			name: "platform ios",
			params: listParams{
				offset:   0,
				limit:    10,
				age:      0,
				gender:   "",
				country:  "",
				platform: "ios",
			},
			expectedSQL: `SELECT a.title, a.end_at FROM advertisement AS a
 INNER JOIN advertisement_condition AS ac ON a.id = ac.advertisement_id
 WHERE NOW() < a.end_at AND NOW() > a.start_at AND (platform & ?) = ? ORDER BY end_at ASC LIMIT ? OFFSET ?`,
			expectedArgs: []interface{}{uint8(2), uint8(2), 10, 0},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			query, args := buildQuery(tc.params)

			assert.Equal(t, tc.expectedSQL, query)

			if !reflect.DeepEqual(args, tc.expectedArgs) {
				t.Errorf("Unexpected arguments. Got: %v, Expected: %v", args, tc.expectedArgs)
			}
		})
	}
}

func TestListActiveAdvertisements(t *testing.T) {
	testCases := []struct {
		name       string
		request    string
		statusCode int
		response   string
	}{
		{
			name:       "Success",
			request:    "",
			statusCode: http.StatusOK,
			response:   `{"items":null}`,
		},
		{
			name:       "Invalid offset",
			request:    "?offset=0",
			statusCode: http.StatusBadRequest,
			response:   "",
		},
		{
			name:       "Invalid age",
			request:    "?age=0",
			statusCode: http.StatusBadRequest,
			response:   "",
		},
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()

	// testDB := testDbInit()

	// defer testDB.Close()

	router.GET("/api/v1/ad", ListActiveAdvertisementsHandler(nil))

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/api/v1/ad"+tc.request, http.NoBody)
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

func TestIntegration(t *testing.T) {
	// testDB := testDbInit()

	// defer testDB.Close()

	var adList []models.Advertisement
	numAdActive := 30
	numAdBefore := 20
	numAdFuture := 10
	adCnt := 1

	for i := 1; i <= numAdActive; i++ {
		ad := models.Advertisement{
			ID:      adCnt,
			Title:   fmt.Sprintf("AD %d", adCnt),
			StartAt: time.Now(),
			EndAt:   time.Now().Add(time.Hour * 24 * 7),
			Conditions: []models.Conditions{
				{
					AgeStart: 18,
					AgeEnd:   65,
					Gender:   []string{"M", "F"},
					Country:  []string{"TW", "US"},
					Platform: []string{"android", "ios", "web"},
				},
			},
		}

		if adCnt%4 >= 1 {
			ad.Conditions = append(ad.Conditions, models.Conditions{
				AgeStart: 18,
				AgeEnd:   65,
				Gender:   []string{"M"},
			})
		}

		if adCnt%4 >= 2 {
			ad.Conditions = append(ad.Conditions, models.Conditions{
				AgeStart: 66,
				AgeEnd:   100,
				Country:  []string{"TW", "US"},
				Platform: []string{"web"},
			})
		}

		if adCnt%4 >= 3 {
			ad.Conditions = append(ad.Conditions, models.Conditions{
				AgeStart: 1,
				AgeEnd:   17,
				Country:  []string{"JP"},
				Platform: []string{"ios", "web"},
			})
		}
		adCnt += 1
		adList = append(adList, ad)
	}

	for i := 1; i <= numAdBefore; i++ {
		ad := models.Advertisement{
			ID:      adCnt,
			Title:   fmt.Sprintf("AD %d", adCnt),
			StartAt: time.Now().Add(-time.Hour * 24 * 14),
			EndAt:   time.Now().Add(-time.Hour * 24 * 7),
			Conditions: []models.Conditions{
				{
					AgeStart: 18,
					AgeEnd:   65,
					Gender:   []string{"M", "F"},
					Country:  []string{"TW", "US"},
					Platform: []string{"android", "ios", "web"},
				},
			},
		}
		adCnt += 1
		adList = append(adList, ad)
	}

	for i := 1; i <= numAdFuture; i++ {
		ad := models.Advertisement{
			ID:      adCnt,
			Title:   fmt.Sprintf("AD %d", adCnt),
			StartAt: time.Now().Add(time.Hour * 24 * 7),
			EndAt:   time.Now().Add(time.Hour * 24 * 14),
			Conditions: []models.Conditions{
				{
					AgeStart: 18,
					AgeEnd:   65,
					Gender:   []string{"M", "F"},
					Country:  []string{"TW", "US"},
					Platform: []string{"android", "ios", "web"},
				},
			},
		}
		adCnt += 1
		adList = append(adList, ad)
	}

	// mock
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.POST("/api/v1/ad", CreateAdvertisementHandler(nil))
	router.GET("/api/v1/ad", ListActiveAdvertisementsHandler(nil))

	// add these to mock database via api
	for _, ad := range adList {
		jsonData, err := json.Marshal(ad)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		// t.Run("create"+strconv.Itoa(i), func(t *testing.T) {
		req, err := http.NewRequest("POST", "/api/v1/ad", bytes.NewReader([]byte(string(jsonData))))
		assert.NoError(t, err)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
		// })
	}

	testCases := []struct {
		name     string
		request  string
		response string
	}{
		{
			name:     "all",
			request:  "",
			response: "",
		},
		{
			name:     "offset",
			request:  "?offest=10",
			response: "",
		},
	}

	// construct list requests
	for _, tc := range testCases {
		// t.Run(tc.name, func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/v1/ad"+tc.request, http.NoBody)
		assert.NoError(t, err)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		log.Println(w.Body.String())
		// Check the HTTP response
		assert.Equal(t, tc.response, w.Body.String())
		// })
	}
}
