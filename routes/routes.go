package routes

import (
	"github.com/gin-gonic/gin"

	controller "github.com/jjshen2000/simple-ads/controllers"
)

func SetupRoutes() *gin.Engine {
	router := gin.Default()

	v1 := router.Group("/api/v1")
	{
		ad := v1.Group("ad")
		// Admin API: Create Advertisement
		ad.POST("", controller.CreateAdvertisement)

		// Public API: List Active Advertisements
		ad.GET("", controller.ListActiveAdvertisements)
	}

	return router
}
