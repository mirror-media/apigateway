package middleware

import (
	"bytes"
	"fmt"
	"io"

	"github.com/gin-gonic/gin"
	"github.com/jensneuse/graphql-go-tools/pkg/graphql"
	"github.com/mirror-media/apigateway/graph"
	"github.com/pkg/errors"
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
		input, err = sjson.SetBytes(input, "variables", variables)
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
