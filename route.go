package apigateway

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/gin-gonic/gin"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/mirror-media/apigateway/graph"
	"github.com/mirror-media/apigateway/graph/generated"
)

const localfile = "/Users/chiu/dev/mm/usersrv/static"

const (
	UserStateActivated              = 200
	UserStateDisabled               = 100
	UserStateMissingInProvider      = 300
	UserStateMissingInMirrorMedia   = 401
	UserStateRegistrationIncomplete = 402 // Not used currently
	UserStateRefreshTokenRevoked    = 501
)

// SetIDTokenStateOnly is a middleware to verify to idToken and save the result to the context
func SetIDTokenStateOnly(server *Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := log.WithFields(log.Fields{
			"path": c.FullPath(),
		})
		const BearerSchema = "Bearer "
		authHeader := c.GetHeader("Authorization")

		if !strings.HasPrefix(authHeader, BearerSchema) {
			c.Set("IDTokenState", "Not a Bearer token")
			c.Next()
			return
		}
		idToken := authHeader[len(BearerSchema):]
		// verify IfToken
		firebaseClient := server.FirebaseClient

		// Verify IDToken is valid
		cCtx := c.Copy()
		_, err := firebaseClient.VerifyIDTokenAndCheckRevoked(cCtx, idToken)
		if err != nil {
			logger.Printf("error verifying ID token: %v\n", err)
			c.Set("IDTokenState", err.Error())
			c.Next()
			return
		}
		c.Set("IDTokenState", "OK")
	}
}

// VerifyIDToken is a middleware to authenticate the request and save the result to the context
func VerifyIDToken(server *Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := log.WithFields(log.Fields{
			"path": c.FullPath(),
		})
		const BearerSchema = "Bearer "
		authHeader := c.GetHeader("Authorization")

		// exit if token is not valid
		if !strings.HasPrefix(authHeader, BearerSchema) {
			c.Set("IDTokenState", "Not a Bearer token")
			c.AbortWithStatusJSON(http.StatusForbidden, ErrorReply{
				Errors: []Error{{Message: "Not a Bearer token"}},
			})
			return
		}
		idToken := authHeader[len(BearerSchema):]
		// Token Verifycation
		firebaseClient := server.FirebaseClient
		// Verify IDToken
		// exit if it's not valid
		cCtx := c.Copy()
		token, err := firebaseClient.VerifyIDTokenAndCheckRevoked(cCtx, idToken)
		if err != nil {
			logger.Printf("error verifying ID token: %v\n", err)
			c.Set("IDTokenState", err.Error())
			c.AbortWithStatusJSON(http.StatusForbidden, ErrorReply{
				Errors: []Error{{Message: err.Error()}},
			})
			return
		}

		c.Set("IDTokenState", "OK")
		c.Set("UserID", token.Subject)
		c.Next()
	}
}

func GinContextToContextMiddleware(server *Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), "GinContextKey", c)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func FirebaseClientToContextMiddleware(server *Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), "FirebaseClient", server.FirebaseClient)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func singleJoiningSlash(a, b string) string {

	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")

	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}
func joinURLPath(a, b *url.URL) (path, rawpath string) {
	if a.RawPath == "" && b.RawPath == "" {
		return singleJoiningSlash(a.Path, b.Path), ""
	}

	// Same as singleJoiningSlash, but uses EscapedPath to determine
	// whether a slash should be added
	apath := a.EscapedPath()
	bpath := b.EscapedPath()
	aslash := strings.HasSuffix(apath, "/")
	bslash := strings.HasPrefix(bpath, "/")
	switch {
	case aslash && bslash:
		return a.Path + b.Path[1:], apath + bpath[1:]
	case !aslash && !bslash:
		return a.Path + "/" + b.Path, apath + "/" + bpath
	}

	return a.Path + b.Path, apath + bpath

}

func ModifyReverseProxyResponse(c *gin.Context) func(*http.Response) error {
	return func(r *http.Response) error {
		body, err := ioutil.ReadAll(r.Body)
		_ = r.Body.Close()
		if err != nil {
			return err
		}

		tokenState, _ := c.Get("IDTokenState")

		b, err := json.Marshal(Reply{
			TokenState: tokenState,
			Data:       json.RawMessage(body),
		})

		if err != nil {
			return err
		}

		r.Body = ioutil.NopCloser(bytes.NewReader(b))
		r.ContentLength = int64(len(b))
		r.Header.Set("Content-Length", strconv.Itoa(len(b)))
		return nil
	}
}

func NewSingleHostReverseProxy(target *url.URL, pathBaseToStrip string) func(c *gin.Context) {
	targetQuery := target.RawQuery
	director := func(req *http.Request) {
		if strings.HasSuffix(pathBaseToStrip, "/") {
			pathBaseToStrip = pathBaseToStrip + "/"
		}
		req.URL.Path = strings.TrimPrefix(req.URL.Path, pathBaseToStrip)
		req.URL.RawPath = strings.TrimPrefix(req.URL.RawPath, pathBaseToStrip)

		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path, req.URL.RawPath = joinURLPath(target, req.URL)

		if targetQuery == "" || req.URL.RawQuery == "" {
			req.URL.RawQuery = targetQuery + req.URL.RawQuery
		} else {
			req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
		}

		if _, ok := req.Header["User-Agent"]; !ok {
			// explicitly disable User-Agent so it's not set to default value
			req.Header.Set("User-Agent", "")
		}
	}

	reverseProxy := &httputil.ReverseProxy{Director: director}

	return func(c *gin.Context) {
		reverseProxy.ModifyResponse = ModifyReverseProxyResponse(c)
		reverseProxy.ServeHTTP(c.Writer, c.Request)
	}
}

type Reply struct {
	TokenState interface{} `json:"tokenState"`
	Data       interface{} `json:"data,omitempty"`
}

type Error struct {
	Message string `json:"message,omitempty"`
}
type ErrorReply struct {
	Errors []Error     `json:"errors,omitempty"`
	Data   interface{} `json:"data,omitempty"`
}

func SetHealthRoute(server *Server) error {

	if server.conf == nil || server.FirebaseApp == nil {
		return errors.New("config or firebase app is nil")
	}

	router := server.Engine
	router.GET("/health", func(c *gin.Context) {
		c.AbortWithStatus(http.StatusOK)
	})

	return nil
}

// SetRoute sets the routing for the gin engine
func SetRoute(server *Server) error {
	apiRouter := server.Engine.Group("/api")

	// Public API
	// v1 api
	v1Router := apiRouter.Group("/v1")
	v1tokenStateRouter := v1Router.Use(SetIDTokenStateOnly(server))
	v1tokenStateRouter.GET("/tokenState", func(c *gin.Context) {
		state, _ := c.Get("IDTokenState")
		c.JSON(http.StatusOK, Reply{
			TokenState: state,
		})
	})

	// Private API
	// v1 User
	v1TokenAuthenticatedWithFirebaseRouter := v1Router.Use(VerifyIDToken(server), GinContextToContextMiddleware(server), FirebaseClientToContextMiddleware(server))
	srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: &graph.Resolver{
		UserSrvURL: server.conf.ServiceEndpoints.UserGraphQL,
	}}))
	v1TokenAuthenticatedWithFirebaseRouter.POST("/graphql/user", gin.WrapH(srv))

	// v0 api proxy every request to the restful serverce
	v0Router := apiRouter.Group("/v0")
	v0tokenStateRouter := v0Router.Use(SetIDTokenStateOnly(server))
	proxyURL, err := url.Parse(server.conf.V0RESTfulSrvTargetURL)
	if err != nil {
		return err
	}

	v0tokenStateRouter.Any("/*wildcard", NewSingleHostReverseProxy(proxyURL, v0Router.BasePath()))

	// ===== Legacy DEMO code =====

	// r := server.Engine
	// r.Use(static.Serve("/", static.LocalFile(localfile, false)))

	// apiRouter.GET("/verifyToken", func(c *gin.Context) {
	// 	apiLogger := log.WithFields(log.Fields{
	// 		"path": c.FullPath(),
	// 	})
	// })

	// legacy user server below

	// apiRouter.GET("/verifyToken", func(c *gin.Context) {
	// 	apiLogger := log.WithFields(log.Fields{
	// 		"path": c.FullPath(),
	// 	})

	// 	const BearerSchema = "Bearer "
	// 	authHeader := c.GetHeader("Authorization")
	// 	idToken := authHeader[len(BearerSchema):]

	// 	token, err := firebaseClient.VerifyIDToken(c, idToken)
	// 	if err != nil {
	// 		apiLogger.Infof("error verifying ID token: %v", err)
	// 		// apiLogger.Infof("token: %v", idToken)
	// 		c.AbortWithStatus(http.StatusForbidden)
	// 		return
	// 	}

	// 	apiLogger.Infof("Verified ID token: %v", token)
	// 	c.Status(http.StatusOK)
	// })

	// apiRouter.GET("/users/:userID/attributes/state", func(c *gin.Context) {
	// 	apiLogger := log.WithFields(log.Fields{
	// 		"path": c.FullPath(),
	// 	})

	// 	firebaseID := c.Param("userID")

	// 	// Get user info from firebase
	// 	firebaseUser, err := firebaseClient.GetUser(c, firebaseID)
	// 	if err != nil {
	// 		apiLogger.Infof("firebase get user(%s) error: %v", firebaseID, err)
	// 		c.AbortWithStatusJSON(http.StatusBadRequest, map[string]int{"state": UserStateMissingInProvider})
	// 		return
	// 	}

	// 	// Get user info from db

	// 	type User struct {
	// 		ID    *int
	// 		State *int
	// 		Email *string
	// 	}
	// 	// TODO move db to server
	// 	db, err := NewDB()
	// 	if err != nil {
	// 		apiLogger.Infof("db open error: %v", err)
	// 		c.AbortWithStatus(http.StatusInternalServerError)
	// 		return
	// 	}
	// 	var user User
	// 	db.Where("firebase_id = ?", firebaseID).First(&user)

	// 	var stateToReturn int
	// 	if firebaseUser.Disabled {
	// 		stateToReturn = UserStateDisabled
	// 	} else if user.ID == nil {
	// 		stateToReturn = UserStateMissingInMirrorMedia
	// 	} else if user.Email == nil || *user.Email == "" {
	// 		stateToReturn = UserStateRegistrationIncomplete
	// 	} else {
	// 		stateToReturn = *user.State
	// 	}
	// 	c.JSON(http.StatusOK, map[string]int{"state": stateToReturn})
	// })

	// apiRouter.GET("/users/:userID", func(c *gin.Context) {
	// 	apiLogger := log.WithFields(log.Fields{
	// 		"path": c.FullPath(),
	// 	})

	// 	firebaseID := c.Param("userID")

	// 	type Address struct {
	// 		ID            *int64     `json:"-"`
	// 		Nationality   *string    `json:",omitempty"`
	// 		State         *string    `json:",omitempty"`
	// 		City          *string    `json:",omitempty"`
	// 		ZipCode       *string    `json:",omitempty"`
	// 		District      *string    `json:",omitempty"`
	// 		StreetAddress *string    `json:",omitempty"`
	// 		CreatedAt     *time.Time `json:",omitempty"`
	// 		UpdatedAt     *time.Time `json:",omitempty"`
	// 	}

	// 	type Image struct {
	// 		ID        *int64     `json:"-"`
	// 		URL       *string    `json:",omitempty"`
	// 		CreatedAt *time.Time `json:",omitempty"`
	// 		UpdatedAt *time.Time `json:",omitempty"`
	// 	}
	// 	type User struct {
	// 		FirebaseID            *string
	// 		Email                 *string
	// 		Name                  *string    `json:",omitempty"`
	// 		Nickname              *string    `json:",omitempty"`
	// 		Bio                   *string    `json:",omitempty"`
	// 		State                 *int       `json:",omitempty"`
	// 		Birthday              *time.Time `json:",omitempty"`
	// 		ImageID               *int64     `json:"-"`
	// 		Image                 *Image     `json:",omitempty"`
	// 		Gender                *int       `json:",omitempty"`
	// 		Phone                 *string    `json:",omitempty"`
	// 		AddressID             *int64     `json:"-"`
	// 		Address               *Address   `json:",omitempty"`
	// 		Point                 *int       `json:",omitempty"`
	// 		CreatedAt             *time.Time `json:",omitempty"`
	// 		UpdatedAt             *time.Time `json:",omitempty"`
	// 		MembershipValidBefore *time.Time `json:",omitempty"`
	// 		MembershipType        *int       `json:",omitempty"`
	// 		MembershipValidAfter  *time.Time `json:",omitempty"`
	// 		CreatedByOperator     *int64     `json:",omitempty"`
	// 	}

	// 	db, err := NewDB()
	// 	if err != nil {
	// 		apiLogger.Infof("db open error: %v", err)
	// 		c.AbortWithStatus(http.StatusInternalServerError)
	// 		return
	// 	}
	// 	var user User
	// 	db.Joins("Image").Joins("Address").Where("firebase_id = ?", firebaseID).First(&user)
	// 	apiLogger.Infof("firebase_id(%s):%+v", firebaseID, user)
	// 	apiLogger.Infof("firebase_id(%s)Image:%+v", firebaseID, user.Image)
	// 	if user.FirebaseID == nil {
	// 		c.AbortWithStatus(http.StatusBadRequest)
	// 		return
	// 	}
	// 	c.JSON(http.StatusOK, user)
	// })

	// apiRouter.POST("/users", func(c *gin.Context) {
	// 	apiLogger := log.WithFields(log.Fields{
	// 		"path": c.FullPath(),
	// 	})

	// 	type Address struct {
	// 		ID            *int64 `json:"-"`
	// 		Nationality   *string
	// 		State         *string
	// 		City          *string
	// 		ZipCode       *string
	// 		District      *string
	// 		StreetAddress *string
	// 		CreatedAt     *time.Time `json:"-"`
	// 		UpdatedAt     *time.Time `json:"-"`
	// 	}

	// 	type Image struct {
	// 		ID        *int64 `json:"-"`
	// 		URL       *string
	// 		CreatedAt *time.Time `json:"-"`
	// 		UpdatedAt *time.Time `json:"-"`
	// 	}
	// 	type User struct {
	// 		FirebaseID            *string
	// 		Email                 *string
	// 		Name                  *string
	// 		Nickname              *string
	// 		Bio                   *string
	// 		State                 *int
	// 		Birthday              *time.Time
	// 		ImageID               *int64 `json:"-"`
	// 		Image                 *Image
	// 		Gender                *int
	// 		Phone                 *string
	// 		AddressID             *int64 `json:"-"`
	// 		Address               *Address
	// 		Point                 *int
	// 		CreatedAt             *time.Time `json:"-"`
	// 		UpdatedAt             *time.Time `json:"-"`
	// 		MembershipValidBefore *time.Time
	// 		MembershipType        *int
	// 		MembershipValidAfter  *time.Time
	// 		CreatedByOperator     *int64
	// 	}

	// 	var user User

	// 	err = c.BindJSON(&user)
	// 	if err != nil {
	// 		apiLogger.Infof("parsing error: %v", err)
	// 		c.AbortWithStatus(http.StatusBadRequest)
	// 		return
	// 	}

	// 	// validate user
	// 	if user.FirebaseID == nil {
	// 		apiLogger.Info("firebase_id isn't provided")
	// 		c.AbortWithStatus(http.StatusBadRequest)
	// 		return
	// 	} else if user.Email == nil {
	// 		apiLogger.Infof("email is not provided for firebase_id(%s)", *user.FirebaseID)
	// 		c.AbortWithStatus(http.StatusBadRequest)
	// 		return
	// 	}

	// 	firebaseUserToUpdate := &auth.UserToUpdate{}
	// 	if user.Email != nil {
	// 		firebaseUserToUpdate.Email(*user.Email)
	// 	}

	// 	if user.Name != nil {
	// 		firebaseUserToUpdate.DisplayName(*user.Name)
	// 	}

	// 	if user.Phone != nil {
	// 		firebaseUserToUpdate.PhoneNumber(*user.Phone)
	// 	}

	// 	// TODO Update image

	// 	_, err = firebaseClient.UpdateUser(
	// 		context.Background(),
	// 		*user.FirebaseID,
	// 		firebaseUserToUpdate,
	// 	)
	// 	if err != nil {
	// 		apiLogger.Errorf("error updating firebase user: %v\n", err)
	// 		c.AbortWithStatusJSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("error updating firebase user: %v\n", err)})
	// 		return
	// 	}

	// 	// TODO move to server
	// 	db, err := NewDB()
	// 	if err != nil {
	// 		apiLogger.Infof("db open error: %v", err)
	// 		c.AbortWithStatus(http.StatusInternalServerError)
	// 		return
	// 	}

	// 	result := db.Create(&user)

	// 	if result.RowsAffected != 1 {
	// 		apiLogger.Infof("user with firebase_id(%s) is not created: %v", *user.FirebaseID, result.Error)
	// 		c.AbortWithStatus(http.StatusBadRequest)
	// 		return
	// 	}

	// 	c.Status(http.StatusOK)
	// })

	// apiRouter.DELETE("/users/:userID", func(c *gin.Context) {
	// 	apiLogger := log.WithFields(log.Fields{
	// 		"path": c.FullPath(),
	// 	})

	// 	firebaseID := c.Param("userID")
	// 	_, err := firebaseClient.UpdateUser(context.Background(), firebaseID, (&auth.UserToUpdate{}).Disabled(true))
	// 	if err != nil {
	// 		apiLogger.Infof("Disabling firebase_id(%s) failed: %v", firebaseID, err)
	// 		c.AbortWithStatusJSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("Disabling firebase_id(%s) failed: %v", firebaseID, err)})
	// 		return
	// 	}

	// 	// TODO move to server
	// 	db, err := NewDB()
	// 	if err != nil {
	// 		apiLogger.Infof("db open error: %v", err)
	// 		c.AbortWithStatus(http.StatusInternalServerError)
	// 		return
	// 	}

	// 	type User struct {
	// 		State int
	// 	}

	// 	db.Model(&User{}).Where("firebase_id = ?", firebaseID).Update("state", UserStateDisabled)

	// })

	// apiRouter.PATCH("/users/:userID", func(c *gin.Context) {
	// 	apiLogger := log.WithFields(log.Fields{
	// 		"path": c.FullPath(),
	// 	})

	// 	firebaseID := c.Param("userID")

	// 	type Address struct {
	// 		ID            *int64 `json:"-"`
	// 		Nationality   *string
	// 		State         *string
	// 		City          *string
	// 		ZipCode       *string
	// 		District      *string
	// 		StreetAddress *string
	// 		CreatedAt     *time.Time `json:"-"`
	// 		UpdatedAt     *time.Time `json:"-"`
	// 	}

	// 	type Image struct {
	// 		ID        *int64 `json:"-"`
	// 		URL       *string
	// 		CreatedAt *time.Time `json:"-"`
	// 		UpdatedAt *time.Time `json:"-"`
	// 	}
	// 	type User struct {
	// 		Email                 *string
	// 		Name                  *string
	// 		Nickname              *string
	// 		Bio                   *string
	// 		State                 *int
	// 		Birthday              *time.Time
	// 		ImageID               *int64 `json:"-"`
	// 		Image                 *Image
	// 		Gender                *int
	// 		Phone                 *string
	// 		AddressID             *int64 `json:"-"`
	// 		Address               *Address
	// 		Point                 *int
	// 		CreatedAt             *time.Time `json:"-"`
	// 		UpdatedAt             *time.Time `json:"-"`
	// 		MembershipValidBefore *time.Time
	// 		MembershipType        *int
	// 		MembershipValidAfter  *time.Time
	// 		CreatedByOperator     *int64
	// 	}

	// 	var user User

	// 	err = c.BindJSON(&user)
	// 	if err != nil {
	// 		apiLogger.Infof("user with firebase_id(%s) is not updated: %v", firebaseID, err)
	// 		c.AbortWithStatus(http.StatusBadRequest)
	// 		return
	// 	}

	// 	// Get user info from firebase
	// 	firebaseUser, err := firebaseClient.GetUser(c, firebaseID)
	// 	if err != nil {
	// 		c.AbortWithStatus(http.StatusInternalServerError)
	// 		return
	// 	} else if firebaseUser == nil {
	// 		c.AbortWithStatusJSON(http.StatusBadRequest, map[string]int{"state": UserStateMissingInProvider})
	// 		return
	// 	}

	// 	firebaseUserToUpdate := &auth.UserToUpdate{}
	// 	if user.Email != nil {
	// 		*user.State = UserStateActivated

	// 		firebaseUserToUpdate.Email(*user.Email)
	// 	} else {
	// 		user.State = nil
	// 	}

	// 	if user.Name != nil {
	// 		firebaseUserToUpdate.DisplayName(*user.Name)
	// 	}

	// 	if user.Phone != nil {
	// 		firebaseUserToUpdate.PhoneNumber(*user.Phone)
	// 	}

	// 	// TODO Update image

	// 	_, err = firebaseClient.UpdateUser(
	// 		context.Background(),
	// 		firebaseID,
	// 		firebaseUserToUpdate,
	// 	)
	// 	if err != nil {
	// 		apiLogger.Errorf("error updating user: %v\n", err)
	// 		c.AbortWithStatusJSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("error updating firebase user: %v\n", err)})
	// 		return
	// 	}

	// 	// TODO move to server
	// 	db, err := NewDB()
	// 	if err != nil {
	// 		apiLogger.Infof("db open error: %v", err)
	// 		c.AbortWithStatus(http.StatusInternalServerError)
	// 		return
	// 	}

	// 	result := db.Model(&user).Where("firebase_id = ?", firebaseID).Updates(user)

	// 	if result.RowsAffected != 1 {
	// 		apiLogger.Infof("user with firebase_id(%s) is not updated: %v", firebaseID, result.Error)
	// 		c.AbortWithStatus(http.StatusBadRequest)
	// 		return
	// 	}

	// 	c.Status(http.StatusOK)
	// })

	// apiRouter.DELETE("/users/:userID", func(c *gin.Context) {
	// 	apiLogger := log.WithFields(log.Fields{
	// 		"path": c.FullPath(),
	// 	})
	// }

	return nil
}
