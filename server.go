package apigateway

import (
	"context"
	"fmt"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
	"github.com/mirror-media/apigateway/config"
	log "github.com/sirupsen/logrus"
	"google.golang.org/api/option"
)

type ServiceEndpoints struct {
	UserGraphQL string
}

type Server struct {
	conf           *config.Conf
	Engine         *gin.Engine
	FirebaseApp    *firebase.App
	FirebaseClient *auth.Client
	Services       *ServiceEndpoints
}

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetReportCaller(true)
}

func (s *Server) Run() error {
	return s.Engine.Run(fmt.Sprintf("%s:%d", s.conf.Address, s.conf.Port))
}

func NewServer(c config.Conf) (*Server, error) {

	engine := gin.Default()

	opt := option.WithCredentialsFile(c.FirebaseCredentialFilePath)
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		return nil, fmt.Errorf("error initializing app: %v", err)
	}

	firebaseClient, err := app.Auth(context.Background())
	if err != nil {
		return nil, fmt.Errorf("initialization of Firebase Auth Client encountered error: %s", err.Error())
	}

	s := &Server{
		conf:           &c,
		Engine:         engine,
		FirebaseApp:    app,
		FirebaseClient: firebaseClient,
		Services: &ServiceEndpoints{
			UserGraphQL: c.ServiceEndpoints.UserGraphQL,
		},
	}
	return s, nil
}
