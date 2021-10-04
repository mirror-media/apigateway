package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cloud.google.com/go/storage"
	formatter "github.com/bcgodev/logrus-formatter-gke"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	ffclient "github.com/thomaspoignant/go-feature-flag"

	"github.com/mirror-media/apigateway/config"
	"github.com/mirror-media/apigateway/featureflag"
	"github.com/mirror-media/apigateway/server"
	"github.com/spf13/viper"
)

func init() {
	logrus.SetFormatter(&formatter.GKELogFormatter{})
	logrus.SetReportCaller(true)
}

func main() {
	v := viper.NewWithOptions(viper.KeyDelimiter("::"))
	// name of config file (without extension)
	v.SetConfigName("config")
	// optionally look for config in the working directory
	v.AddConfigPath("./configs")
	// Find and read the config file
	err := v.ReadInConfig()
	// Handle errors reading the config file
	if err != nil {
		logrus.Fatalf("fatal error config file: %s", err)
	}

	var cfg config.Conf
	err = v.Unmarshal(&cfg)
	if err != nil {
		logrus.Fatalf("unable to decode into struct, %v", err)
	}

	client, err := storage.NewClient(context.Background())
	if err != nil {
		logrus.Fatal(err)
	}
	object := client.Bucket(cfg.FeatureToggles.Bucket).Object(cfg.FeatureToggles.Object)

	ffclient.Init(ffclient.Config{
		PollingInterval: 60 * time.Second,
		Logger:          log.New(os.Stdout, "", 0),
		Context:         context.Background(),
		Retriever: &featureflag.Bucket{
			Object: object,
		},
		FileFormat:              cfg.FeatureToggles.Type,
		StartWithRetrieverError: false,
	})
	// Check init errors.
	if err != nil {
		logrus.Fatal(err)
	}
	// defer closing ffclient
	defer ffclient.Close()

	svr, err := server.NewServer(cfg)
	if err != nil {
		err = errors.Wrap(err, "failed to create new server")
		logrus.Fatal(err)
	}
	err = server.SetHealthRoute(svr)
	if err != nil {
		logrus.Fatalf("error setting up health route: %v", err)
	}

	err = server.SetRoute(svr)
	if err != nil {
		logrus.Fatalf("error setting up route: %v", err)
	}

	httpSVR := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", svr.Conf.Address, svr.Conf.Port),
		Handler: svr.Engine,
	}

	// Initializing the server in a goroutine so that
	// it won't block the graceful shutdown handling below

	go func() {
		logrus.Infof("server listening to %s", httpSVR.Addr)
		if err = httpSVR.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			err = errors.Wrap(shutdown(httpSVR, nil), err.Error())
			logrus.Fatalf("listen: %s\n", err)
		} else if err != nil {
			err = errors.Wrap(shutdown(nil, nil), err.Error())
			logrus.Fatalf("error server closed: %s\n", err)
		}
	}()
	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal, 1)
	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be catch, so don't need add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logrus.Println("Shutting down server...")

	if err := shutdown(httpSVR, nil); err != nil {
		logrus.Fatalf("Server forced to shutdown:", err)
	}
	os.Exit(0)
}

func shutdown(server *http.Server, cancelMemberSubscription context.CancelFunc) error {
	if server != nil {
		// The context is used to inform the server it has 5 seconds to finish
		// the request it is currently handling
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			return err
		}
	}
	if cancelMemberSubscription != nil {
		cancelMemberSubscription()
	}
	return nil
}
