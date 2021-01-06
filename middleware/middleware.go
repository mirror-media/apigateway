package middleware

type CtxKey string

const (
	//CtxGinContexKey is the key of a *gin.Context
	CtxGinContexKey CtxKey = "CtxGinContext"
	//CtxFirebaseClient is the key of a *auth.Client
	CtxFirebaseClient CtxKey = "CtxFirebaseClient"
)
const (
	// GCtxTokenKey is the key of a token.Token
	GCtxTokenKey string = "GCtxToken"
	// GCtxUserIDKey is the key of a string of a User ID
	GCtxUserIDKey string = "GCtxUserID"
)
