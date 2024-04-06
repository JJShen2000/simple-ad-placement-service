package main

import (
	"fmt"

	"github.com/jjshen2000/simple-ads/config"
	"github.com/jjshen2000/simple-ads/routes"
)

func main() {
	router := routes.SetupRoutes()
	config := config.GetConfig()
	router.Run(fmt.Sprintf("%s:%d", config.Server.IP, config.Server.Port))
}


