package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jensneuse/graphql-go-tools/pkg/engine/datasource/httpclient"
	"github.com/machinebox/graphql"
	graphqlclient "github.com/machinebox/graphql"
	"github.com/mirror-media/apigateway/cache"
	"github.com/mirror-media/apigateway/graph/member/model"
	"github.com/mirror-media/apigateway/middleware"
	"github.com/mirror-media/apigateway/token"
	"github.com/sirupsen/logrus"
	"github.com/tidwall/sjson"
)

func NewSingleHostReverseProxy(target *url.URL, pathBaseToStrip string, rdb cache.Rediser, cacheTTL int, memberGraphqlEndpoint string) func(c *gin.Context) {
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
	return func(c *gin.Context) {
		// TODO refactor modification and cache code
		var tokenState string

		tokenSaved, exist := c.Get(middleware.GCtxTokenKey)
		if !exist {
			tokenState = "No Bearer token available"
		} else {
			tokenState = tokenSaved.(token.Token).GetTokenState()
		}

		switch path := c.Request.URL.Path; {
		case strings.HasSuffix(path, "/getposts") || strings.HasSuffix(path, "/posts") || strings.HasSuffix(path, "/post"):
			// Try to read cache first
			var key string
			if tokenState != token.OK {
				key = fmt.Sprintf("%s.%s.%s.%s", "apigateway", "post", "truncated", c.Request.RequestURI)
			} else {
				key = fmt.Sprintf("%s.%s.%s.%s", "apigateway", "post", "clean", c.Request.RequestURI)
			}

			cmd := rdb.Get(context.TODO(), key)
			// cache doesn't exist, do fetch reverse proxy
			if cmd == nil {
				break
			}
			body, err := cmd.Bytes()
			// cache can't be understood, do fetch reverse proxy
			if err != nil {
				break
			}

			c.AbortWithStatusJSON(http.StatusOK, Reply{
				TokenState: tokenState,
				Data:       json.RawMessage(body),
			})
			return
		}

		var subscribedPostIDs = make(chan map[string]interface{}, 1)
		var premiumPrivilege = make(chan bool, 1)
		var errChan = make(chan error, 1)
		if tokenState != token.OK {
			subscribedPostIDs <- nil
			premiumPrivilege <- false
			errChan <- nil
		} else {
			tokenState = tokenSaved.(token.Token).GetTokenState()
			go func() {
				if tokenState == token.OK {
					firebaseID := c.GetString(middleware.GCtxUserIDKey)
					gql := `
query ($firebaseId: String!) {
  member(where: {firebaseId: $firebaseId}) {
    subscription(where: {isActive: true}) {
      frequency
      postId
    }
  }
}
`
					req := graphqlclient.NewRequest(gql)
					req.Var("firebaseId", firebaseID)
					data := model.Member{}

					var ids = make(map[string]interface{})
					var privilege bool
					var err error

					client := graphql.NewClient(memberGraphqlEndpoint, graphql.WithHTTPClient(httpclient.DefaultNetHttpClient))
					err = client.Run(context.TODO(), req, &data)
					// defer sending values to chanels to make sure that every value is properly assigned in the workflow and the channel won't be block receiving
					defer func() {
						errChan <- err
						subscribedPostIDs <- ids
						premiumPrivilege <- privilege
					}()
					if err != nil {
						return
					}

					if data.Subscription != nil {
						for _, s := range data.Subscription {
							if *s.Frequency == model.SubscriptionFrequencyTypeOneTime {
								ids[*s.PostID] = nil
							} else {
								privilege = true
								break
							}
						}
					}
				}
			}()
		}

		reverseProxy := httputil.ReverseProxy{Director: director}
		reverseProxy.ModifyResponse = ModifyReverseProxyResponse(c, rdb, cacheTTL, subscribedPostIDs, premiumPrivilege, errChan)
		reverseProxy.ServeHTTP(c.Writer, c.Request)
	}
}

func ModifyReverseProxyResponse(c *gin.Context, rdb cache.Rediser, cacheTTL int, subscribedPostIDs chan map[string]interface{}, premiumPrivilege chan bool, errChan chan error) func(*http.Response) error {
	logger := logrus.WithFields(logrus.Fields{
		"path": c.FullPath(),
	})
	return func(r *http.Response) error {
		// check error first for short circuit
		select {
		case err := <-errChan:
			if err != nil {
				logger.Error(err)
				return err
			}
		case <-time.After(1 * time.Second):
			err := fmt.Errorf("timeout for one seconds for getting %s", "error from member subscription fetching")
			logger.Error(err)
			return err
		}

		body, err := io.ReadAll(r.Body)
		_ = r.Body.Close()
		if err != nil {
			logger.Errorf("encounter error when reading proxy response:", err)
			return err
		}

		var tokenState string

		tokenSaved, exist := c.Get(middleware.GCtxTokenKey)
		if !exist {
			tokenState = "No Bearer token available"
		} else {
			tokenState = tokenSaved.(token.Token).GetTokenState()
		}

		var redisKey string
		switch path := r.Request.URL.Path; {
		// TODO refactor condition
		case strings.HasSuffix(path, "/getposts") || strings.HasSuffix(path, "/posts") || strings.HasSuffix(path, "/post"):

			type Category struct {
				IsMemberOnly *bool `json:"isMemberOnly,omitempty"`
			}

			type ItemContent struct {
				APIData []interface{} `json:"apiData"`
			}
			type Item struct {
				ID         string      `json:"_id"`
				Content    ItemContent `json:"content"`
				Categories []Category  `json:"categories"`
			}
			type Resp struct {
				Items []Item `json:"_items"`
			}

			var items Resp
			err = json.Unmarshal(body, &items)
			if err != nil {
				logger.Errorf("Unmarshal post encountered error: %v", err)
				return err
			}

			// truncate the content if the user is not a member and the post falls into a member only category

			// modify body if the item falls into a "member only" category
			var subscribedIds map[string]interface{}
			var hasPremiumPrivilege bool

			select {
			case subscribedIds = <-subscribedPostIDs:
			case <-time.After(1 * time.Second):
				err = fmt.Errorf("timeout for one seconds for getting %s", "subscribedPostIDs")
				logger.Error(err)
				return err
			}

			select {
			case hasPremiumPrivilege = <-premiumPrivilege:
			case <-time.After(1 * time.Second):
				err = fmt.Errorf("timeout for one seconds for getting %s", "premiumPrivilege")
				logger.Error(err)
				return err
			}

			toTruncateIt := func(category Category, postID string) bool {
				if category.IsMemberOnly == nil || !*category.IsMemberOnly {
					return true
				}
				_, postSubscribed := subscribedIds[postID]
				return !hasPremiumPrivilege && !postSubscribed
			}

			var truncated bool

			for i, item := range items.Items {
				for _, category := range item.Categories {
					if toTruncateIt(category, item.ID) {
						truncatedAPIData := item.Content.APIData[0:3]
						body, err = sjson.SetBytes(body, fmt.Sprintf("_items.%d.content.apiData", i), truncatedAPIData)
						if err != nil {
							logger.Errorf("encounter error when truncating apiData:", err)
							return err
						}
						truncated = true
						body, err = sjson.SetBytes(body, fmt.Sprintf("_items.%d.isTruncated", i), true)
						if err != nil {
							logger.Errorf("encounter error setting isTruncated to true for _items.%d", i, err)
							return err
						}
						break
					} else {
						body, err = sjson.SetBytes(body, fmt.Sprintf("_items.%d.isTruncated", i), false)
						if err != nil {
							logger.Errorf("encounter error setting isTruncated to false for _items.%d", i, err)
							return err
						}
					}
				}
			}

			// remove html because only apidata is useful and html contains full content
			for i, _ := range items.Items {
				body, err = sjson.DeleteBytes(body, fmt.Sprintf("_items.%d.content.html", i))
				if err != nil {
					logger.Errorf("encounter error when deleting html:", err)
					return err
				}
			}

			// FIXME it's not a single condition anymore!
			if truncated {
				// TODO refactor redis cache code
				redisKey = fmt.Sprintf("%s.%s.%s.%s", "apigateway", "post", "truncated", c.Request.RequestURI)
			} else {
				// TODO refactor redis cache code
				redisKey = fmt.Sprintf("%s.%s.%s.%s", "apigateway", "post", "clean", c.Request.RequestURI)
			}
			// TODO refactor redis cache code
			err = rdb.Set(context.TODO(), redisKey, body, time.Duration(cacheTTL)*time.Second).Err()
			if err != nil {
				logger.Warnf("setting redis cache(%s) encountered error: %v", redisKey, err)
			}
		default:
		}

		b, err := json.Marshal(Reply{
			TokenState: tokenState,
			Data:       json.RawMessage(body),
		})

		if err != nil {
			logger.Errorf("Marshalling reply encountered error: %v", err)
			return err
		}

		r.Body = io.NopCloser(bytes.NewReader(b))
		r.ContentLength = int64(len(b))
		r.Header.Set("Content-Length", strconv.Itoa(len(b)))
		return nil
	}
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
