package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jensneuse/graphql-go-tools/pkg/graphql"
	"github.com/mirror-media/apigateway/graph"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/tidwall/sjson"
)

type CtxKey string

const (
	//CtxGinContexKey is the key of a *gin.Context
	CtxGinContexKey CtxKey = "CtxGinContext"
	//CtxFirebaseClientKey is the key of a *auth.Client
	CtxFirebaseClientKey CtxKey = "CtxFirebaseClient"
	//CtxFirebaseDatabaseClientKey is the key of a *db.Client
	CtxFirebaseDatabaseClientKey CtxKey = "CtxFirebaseDBClient"
)
const (
	// GCtxTokenKey is the key of a token.Token in *gin.Context
	GCtxTokenKey string = "GCtxToken"
	// GCtxUserIDKey is the key of a string of a User ID in *gin.Context
	GCtxUserIDKey string = "GCtxUserID"
)

func PatchNullVariablesInGraphqlVariables(c *gin.Context) {
	logger := logrus.WithFields(logrus.Fields{
		"path": c.FullPath(),
	})
	bodyReader := c.Request.Body
	defer bodyReader.Close()

	body, err := io.ReadAll(bodyReader)
	if err != nil {
		logger.Info(err)
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("cannot read http request body"))
		return
	}

	var j json.RawMessage
	err = json.Unmarshal(body, &j)
	if err != nil {
		logger.Info(err)
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("cannot marshal http request body as json"))
		return
	}

	if j != nil {
		j, err = patchNullVariablesInGraphql(j)
		if err != nil {
			logger.Info(err)
			c.AbortWithError(http.StatusBadRequest, fmt.Errorf("cannot patch \"null\" in graphql variables"))
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(j))
	} else {
		c.Request.Body = io.NopCloser(bytes.NewReader(body))
	}

	ginContextToContextMiddleware(c)
	c.Next()
}

func patchNullVariablesInGraphql(input []byte) ([]byte, error) {
	reader := bytes.NewReader(input)
	graphqlRequest := graphql.Request{}
	if err := graphql.UnmarshalRequest(reader, &graphqlRequest); err != nil {
		err := errors.Wrap(err, "cannot unmarshal graphql request")
		return nil, err
	}

	variables, err := graphqlRequest.Variables.MarshalJSON()
	if err != nil {
		err = errors.Wrap(err, "cannot get variables")
		return nil, err
	}

	if !bytes.Equal(variables, []byte("null")) {
		variables, err = graph.ReplaceNullString(variables)
		if err != nil {
			err = errors.Wrap(err, "cannot replace null string in variables")
			return nil, err
		}
		input, err = sjson.SetRawBytes(input, "variables", variables)
		if err != nil {
			err = errors.Wrap(err, "cannot set variables to input")
			return nil, err
		}
	}

	return input, nil
}

// PrintPayloadDebug prints the request body to stdout. Do not use it in production
func PrintPayloadDebug(c *gin.Context) {
	req := c.Request

	body, _ := io.ReadAll(req.Body)

	fmt.Println(string(body))

	req.Body = io.NopCloser(bytes.NewReader(body))

	c.Next()
}
