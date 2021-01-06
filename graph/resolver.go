package graph

import (
	"context"
	"fmt"

	"github.com/99designs/gqlgen/graphql"
	log "github.com/sirupsen/logrus"

	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	UserSrvURL string
}

func (r Resolver) IsRequestMatchingRequesterFirebaseID(ctx context.Context, firebaseID string) (bool, error) {

	gCTX, err := GinContextFromContext(ctx)
	if err != nil {
		return false, err
	}

	if firebaseID != gCTX.Value("UserID").(string) {
		return false, fmt.Errorf("member id(%s) is not allowed to perfrom action against member id(%s)", gCTX.Value("UserID"), firebaseID)
	}
	return true, nil
}

func GinContextFromContext(ctx context.Context) (*gin.Context, error) {
	ginContext := ctx.Value("GinContextKey")
	if ginContext == nil {
		err := fmt.Errorf("could not retrieve gin.Context")
		log.Error(err)
		return nil, err
	}

	gc, ok := ginContext.(*gin.Context)
	if !ok {
		err := fmt.Errorf("gin.Context has wrong type")
		log.Error(err)
		return nil, err
	}
	return gc, nil
}

func FirebaseClientFromContext(ctx context.Context) (*auth.Client, error) {
	gCTX, err := GinContextFromContext(ctx)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	logger := log.WithFields(log.Fields{
		"path": gCTX.FullPath(),
	})
	firebaseClientCtx := ctx.Value("FirebaseClient")

	client, ok := firebaseClientCtx.(*auth.Client)
	if !ok {
		err := fmt.Errorf("auth.Client has wrong type")
		logger.Error(err)
		return nil, err
	}
	return client, nil
}

func GetPreloads(ctx context.Context) []string {
	return GetNestedPreloads(
		graphql.GetOperationContext(ctx),
		graphql.CollectFieldsCtx(ctx, nil),
		"",
	)
}

func GetNestedPreloads(ctx *graphql.OperationContext, fields []graphql.CollectedField, prefix string) (preloads []string) {
	for _, column := range fields {
		prefixColumn := GetPreloadString(prefix, column.Name)
		preloads = append(preloads, prefixColumn)
		preloads = append(preloads, GetNestedPreloads(ctx, graphql.CollectFields(ctx, column.Selections, nil), prefixColumn)...)
	}
	return
}

func GetPreloadString(prefix, name string) string {
	if len(prefix) > 0 {
		return prefix + "." + name
	}
	return name
}

func Map(vs []string, f func(string) string) []string {
	vsm := make([]string, len(vs))
	for i, v := range vs {
		vsm[i] = f(v)
	}
	return vsm
}
