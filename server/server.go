// Package server define the necessary component of a server
package server

import (
	"context"
	"fmt"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"firebase.google.com/go/v4/db"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/mirror-media/apigateway/cache"
	"github.com/mirror-media/apigateway/config"
	"github.com/mirror-media/apigateway/token"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"
)

type ServiceEndpoints struct {
	UserGraphQL string
}

type Server struct {
	Conf                   *config.Conf
	Engine                 *gin.Engine
	FirebaseApp            *firebase.App
	firebaseClient         *auth.Client
	firebaseDatabaseClient *db.Client
	Services               *ServiceEndpoints
	UserSvrToken           token.Token
	Rdb                    cache.Rediser
}

func init() {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetReportCaller(true)
}

func (s *Server) Run() error {
	return s.Engine.Run(fmt.Sprintf("%s:%d", s.Conf.Address, s.Conf.Port))
}

func NewServer(c config.Conf) (*Server, error) {

	engine := gin.Default()

	opt := option.WithCredentialsFile(c.FirebaseCredentialFilePath)

	config := &firebase.Config{
		DatabaseURL: c.FirebaseRealtimeDatabaseURL,
	}
	app, err := firebase.NewApp(context.Background(), config, opt)
	if err != nil {
		return nil, errors.Wrap(err, "error initializing app")
	}

	firebaseClient, err := app.Auth(context.Background())
	if err != nil {
		return nil, errors.Wrap(err, "fail to initialize thr Firebase Auth Client")
	}

	dbClient, err := app.Database(context.Background())
	if err != nil {
		return nil, errors.Wrap(err, "fail to initialize the Firebase Database Client")
	}

	var rdb cache.Rediser

	switch c.RedisService.Type {
	case "cluster":
		if len(c.RedisService.Addresses) == 0 {
			return nil, errors.New("there's no redis address provided")
		}
		// TODO refactor
		addrs := make([]string, 0, len(c.RedisService.Addresses))
		for _, a := range c.RedisService.Addresses {
			addrs = append(addrs, fmt.Sprintf("%s:%d", a.Addr, a.Port))
		}
		rdb = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    addrs,
			Password: c.RedisService.Password,
		})
	case "single":
		if len(c.RedisService.Addresses) == 0 {
			return nil, errors.New("there's no redis address provided")
		} else if len(c.RedisService.Addresses) > 1 {
			logrus.Warnf("single type Redis accepts only the first address, but %d addresses are provided", len(c.RedisService.Addresses))
		}

		// TODO refactor
		// Only the first address is used because it's a single instance
		addrs := make([]string, 0, len(c.RedisService.Addresses))
		for _, a := range c.RedisService.Addresses {
			addrs = append(addrs, fmt.Sprintf("%s:%d", a.Addr, a.Port))
		}
		rdb = redis.NewClient(&redis.Options{
			Addr:     addrs[0],
			Password: c.RedisService.Password,
		})
	case "sentinel":
		if len(c.RedisService.Addresses) == 0 {
			return nil, errors.New("there's no redis address provided")
		}
		// TODO refactor
		addrs := make([]string, 0, len(c.RedisService.Addresses))
		for _, a := range c.RedisService.Addresses {
			addrs = append(addrs, fmt.Sprintf("%s:%d", a.Addr, a.Port))
		}
		rdb = redis.NewFailoverClient(&redis.FailoverOptions{
			SentinelAddrs: addrs,
			Password:      c.RedisService.Password,
		})
	default:
		return nil, errors.New(fmt.Sprintf("unsupported redis type(%s)", c.RedisService.Type))
	}

	gatewayToken, err := token.NewGatewayToken(c.TokenSecretName, c.ProjectID)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to retrieve the latest token(%s)", c.TokenSecretName)
	}

	s := &Server{
		Conf:                   &c,
		Engine:                 engine,
		FirebaseApp:            app,
		firebaseClient:         firebaseClient,
		firebaseDatabaseClient: dbClient,
		Rdb:                    rdb,
		Services: &ServiceEndpoints{
			UserGraphQL: c.ServiceEndpoints.UserGraphQL,
		},
		UserSvrToken: gatewayToken,
	}
	return s, nil
}
