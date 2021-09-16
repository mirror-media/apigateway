module github.com/mirror-media/apigateway

go 1.16

replace github.com/jensneuse/graphql-go-tools => github.com/mirror-media/graphql-go-tools v1.25.1

require (
	firebase.google.com/go/v4 v4.6.0
	github.com/99designs/gqlgen v0.14.0
	github.com/bcgodev/logrus-formatter-gke v1.0.0
	github.com/gin-gonic/gin v1.7.4
	github.com/go-redis/redis/v8 v8.11.3
	github.com/golang-jwt/jwt/v4 v4.0.0
	github.com/google/go-querystring v1.1.0
	github.com/jensneuse/abstractlogger v0.0.4
	github.com/jensneuse/graphql-go-tools v1.20.2
	github.com/jensneuse/graphql-go-tools/examples/federation v0.0.0-20210910154601-7707a291adb6
	github.com/machinebox/graphql v0.2.2
	github.com/matryer/is v1.4.0 // indirect
	github.com/pkg/errors v0.9.1
	github.com/rs/xid v1.3.0
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/viper v1.8.1
	github.com/tidwall/sjson v1.2.2
	github.com/vektah/gqlparser/v2 v2.2.0
	google.golang.org/api v0.56.0
)
