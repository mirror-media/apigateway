package main

import (
	"fmt"

	apigateway "github.com/mirror-media/mm-apigateway"
	"github.com/mirror-media/mm-apigateway/config"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func main() {

	// name of config file (without extension)
	viper.SetConfigName("config")
	// optionally look for config in the working directory
	viper.AddConfigPath(".")
	// Find and read the config file
	err := viper.ReadInConfig()
	// Handle errors reading the config file
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %s", err))
	}

	var cfg config.Conf
	err = viper.Unmarshal(&cfg)
	if err != nil {
		panic(fmt.Errorf("unable to decode into struct, %v", err))
	}

	server, err := apigateway.NewServer(cfg)
	if err != nil {
		panic(err)
	}
	err = apigateway.SetHealthRoute(server)
	if err != nil {
		log.Fatalf("error setting up health route: %v", err)
		panic(err)
	}

	err = apigateway.SetRoute(server)
	if err != nil {
		log.Fatalf("error setting up route: %v", err)
		panic(err)
	}

	err = server.Run()
	if err != nil {
		log.Fatalf("error runing server: %v", err)
		panic(err)
	}
}
