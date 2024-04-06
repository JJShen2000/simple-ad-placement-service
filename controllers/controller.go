package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/biter777/countries"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"

	dbpkg "github.com/jjshen2000/simple-ads/db"
	"github.com/jjshen2000/simple-ads/models"
)

// Handler for creating advertisement
func CreateAdvertisement(c *gin.Context) {
	db := dbpkg.GetDB()
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

		var platformBits uint8
		if len(condition.Platform) == 0 {
			platformBits = 7
		} else {
			for _, p := range condition.Platform {
				switch p {
				case "android":
					platformBits |= 1 << 0
				case "ios":
					platformBits |= 1 << 1
				case "web":
					platformBits |= 1 << 2
				}
			}
		}

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

func isValidPlatform(platform string) bool {
	if platform == "" {
		return true
	}

	switch platform {
	case "android", "ios", "web":
		return true
	default:
		return false
	}
}

// Handler for listing active advertisements
func ListActiveAdvertisements(c *gin.Context) {
	// Parse parameters
	offsetStr := c.DefaultQuery("offset", "1")
	limitStr := c.DefaultQuery("limit", "5")
	ageStr := c.DefaultQuery("age", "-1")
	gender := c.Query("gender")
	country := c.Query("country")
	platform := c.Query("platform")

	// Convert them to corresponding types and validate them.
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid offset"})
		return
	}
	offset -= 1
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit"})
		return
	}

	age, err := strconv.Atoi(ageStr)
	if err != nil || (ageStr != "-1" && (age < 1 || age > 100)) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid age"})
		return
	}

	if gender != "" && gender != "M" && gender != "F" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid gender"})
		return
	}

	if country != "" && countries.ByName(country) == countries.Unknown {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid country"})
		return
	}

	if !isValidPlatform(platform) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid platform"})
		return
	}

	// Build query *******************************************************************
	db := dbpkg.GetDB()
	query := "SELECT title, end_at FROM advertisement WHERE NOW() > start_at AND NOW() < end_at"
	var args []interface{}

	if ageStr != "-1" {
		query += " AND ? BETWEEN age_start AND age_end"
		args = append(args, age)
	}

	if gender != "" {
		query += " AND gender = ?"
		args = append(args, gender)
	}

	if country != "" {
		query += " AND EXISTS (SELECT 1 FROM condition_country WHERE advertisement_condition.id = condition_country.condition_id AND country_code = ?)"
		args = append(args, country)
	}

	if platform != "" {
		query += " AND (platform & ?) = ?"
		var platformMask int
		switch platform {
		case "android":
			platformMask = 1
		case "ios":
			platformMask = 2
		case "web":
			platformMask = 4
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid platform"})
			return
		}
		args = append(args, platformMask, platformMask)
	}

	query += " ORDER BY end_at ASC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	fmt.Println(limit, " ", offset)
	// Execute query
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
