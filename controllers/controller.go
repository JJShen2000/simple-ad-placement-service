package controller

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/biter777/countries"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"

	dbpkg "github.com/jjshen2000/simple-ads/db"
	"github.com/jjshen2000/simple-ads/models"
	
)

var platformMap = map[string]uint8{
	"android": 1,
	"ios":     2,
	"web":     4,
}

func CreateAdvertisementHandler(testDB *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		CreateAdvertisement(c, testDB)
	}
}

// Handler for creating advertisement
func CreateAdvertisement(c *gin.Context, testDB *sqlx.DB) {
	db := testDB
	if testDB == nil {
		db = dbpkg.GetDB()
	}
	
	validate := models.GetValidate()
	var ad models.Advertisement

	if err := c.BindJSON(&ad); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate the advertisement data
	if err := validate.Struct(ad); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Start a transaction
	tx := db.MustBegin()

	// Insert advertisement
	insertAd := `INSERT INTO advertisement (title, start_at, end_at) VALUES (?, ?, ?)`
	result, err := tx.Exec(insertAd, ad.Title, ad.StartAt, ad.EndAt)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error(insert advertisement)": err.Error()})
		return
	}

	// Get ID of the inserted advertisement
	adID, _ := result.LastInsertId()

	// Insert advertisement conditions
	for _, condition := range ad.Conditions {
		genderVal := strings.Join(condition.Gender, "")
		if genderVal == "FM" || len(condition.Gender) == 0 {
			genderVal = "MF"
		}

		unlimited_country := false
		if len(condition.Country) == 0 {
			unlimited_country = true
		}

		platformBits := getPlatformBits(condition.Platform)

		insertCondition := `
		INSERT INTO advertisement_condition 
			(advertisement_id, age_start, age_end, gender, unlimited_country, platform) 
		VALUES 
			(?, ?, ?, ?, ?, ?)
		`

		_, err := tx.Exec(insertCondition, adID, condition.AgeStart, condition.AgeEnd, genderVal, unlimited_country, platformBits)
		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error(insert condition)": err.Error()})
			return
		}

		// Get ID of the inserted condition
		conditionID, _ := result.LastInsertId()

		// Insert condition countries
		for _, country := range condition.Country {
			insertCountry := `
				INSERT INTO condition_country (condition_id, country_code) VALUES (?, ?)
			`
			_, err := tx.Exec(insertCountry, conditionID, country)
			if err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error(insert country)": err.Error()})
				return
			}
		}
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error(commit)": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Advertisement created successfully"})
}

// getPlatformBits returns bits value mapping from slice of platforms.
//
// If no platform is indicated in the slice, return 7 (111 in binary).
// If the slice contains an invalid platform, return 0.
func getPlatformBits(platforms []string) uint8 {
	var platformBits uint8
	for _, p := range platforms {
		bit, found := platformMap[p]
		if !found { // invalid
			return 0
		}
		platformBits |= bit
	}
	if platformBits == 0 { // no specific platform is indicated
		platformBits = 7
	}
	return platformBits
}

// isValidPlatform checks if the given platform is valid.
// It returns true if the platform is empty (indicating no platform specified),
// or if the platform exists in the platformMap; otherwise, it returns false.
func isValidPlatform(platform string) bool {
	return platform == "" || platformMap[platform] != 0
}

type listParams struct {
	offset   int
	limit    int
	age      int
	gender   string
	country  string
	platform string
}

// Parse request parameters for listing active advertisements
func parseListParams(c *gin.Context) (params listParams, err error) {
	offsetStr := c.DefaultQuery("offset", "1")
	params.offset, err = strconv.Atoi(offsetStr)
	if err != nil || params.offset < 1 {
		err = errors.New("invalid offset")
		return
	}
	params.offset -= 1

	limitStr := c.DefaultQuery("limit", "5")
	params.limit, err = strconv.Atoi(limitStr)
	if err != nil || params.limit < 1 || params.limit > 100 {
		err = errors.New("invalid limit")
		return
	}

	ageStr := c.DefaultQuery("age", "0")
	params.age, err = strconv.Atoi(ageStr)
	if err != nil || (c.Query("age") != "" && (params.age < 1 || params.age > 100)) {
		err = errors.New("invalid age")
		return
	}

	params.gender = c.Query("gender")
	if params.gender != "" && params.gender != "M" && params.gender != "F" {
		err = errors.New("invalid gender")
		return
	}

	params.country = c.Query("country")
	if params.country != "" && countries.ByName(params.country) == countries.Unknown {
		err = errors.New("invalid country")
		return
	}

	params.platform = c.Query("platform")
	if params.platform != "" && !isValidPlatform(params.platform) {
		err = errors.New("invalid platform")
		return
	}
	return
}

// buildQuery constructs a SQL query string and its corresponding arguments based on provided parameters.
func buildQuery(params listParams) (query string, args []interface{}) {
	query = "SELECT a.title, a.end_at FROM advertisement AS a\n"

	if params.age != 0 || params.gender != "" || params.country != "" || params.platform != "" {
		query += " INNER JOIN advertisement_condition AS ac ON a.id = ac.advertisement_id\n"
	}

	if params.country != "" {
		query += " INNER JOIN condition_country AS cc ON ac.id = cc.condition_id\n"
	}

	query += " WHERE NOW() < a.end_at AND NOW() > a.start_at"
	if params.age != 0 {
		query += " AND ? BETWEEN ac.age_start AND ac.age_end"
		args = append(args, params.age)
	}

	if params.gender != "" {
		if params.gender == "M" {
			query += " AND ac.gender != ?"
			args = append(args, "F")
		} else if params.gender == "F" {
			query += " AND ac.gender != ?"
			args = append(args, "M")
		}
	}

	if params.country != "" {
		query += " AND cc.country_code = ?"
		args = append(args, params.country)
	}

	if params.platform != "" {
		query += " AND (platform & ?) = ?"
		platformMask := platformMap[params.platform]
		args = append(args, platformMask, platformMask)
	}

	query += " ORDER BY end_at ASC LIMIT ? OFFSET ?"
	args = append(args, params.limit, params.offset)

	return query, args
}

func ListActiveAdvertisementsHandler(testDB *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ListActiveAdvertisements(c, testDB)
	}
}

// Handler for listing active advertisements
func ListActiveAdvertisements(c *gin.Context, testDB *sqlx.DB) {
	// Parse parameters *******************************************************************
	params, err := parseListParams(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build query *******************************************************************
	query, args := buildQuery(params)

	// Execute query
	db := testDB
	if testDB == nil {
		db = dbpkg.GetDB()
	}
	rows, err := db.Queryx(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch advertisements"})
		return
	}
	defer rows.Close()

	// results
	type retAd struct {
		Title string    `json:"title"`
		EndAt time.Time `json:"endAt"`
	}
	var ads []retAd

	for rows.Next() {
		var ad retAd
		var endAtStr string
		err := rows.Scan(&ad.Title, &endAtStr)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse advertisement"})
			return
		}

		ad.EndAt, err = time.Parse("2006-01-02 15:04:05", endAtStr)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse endAt timestamp"})
			return
		}
		ads = append(ads, ad)
	}

	c.JSON(http.StatusOK, gin.H{"items": ads})
}
