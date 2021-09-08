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

		logger := logrus.WithFields(logrus.Fields{
			"path": c.FullPath(),
		})

		if strings.HasSuffix(pathBaseToStrip, "/") {
			pathBaseToStrip = pathBaseToStrip + "/"
		}
		trimmedPath := strings.TrimPrefix(c.Request.URL.Path, pathBaseToStrip)

		var err error
		var tokenState string

		tokenSaved, exist := c.Get(middleware.GCtxTokenKey)
		if !exist {
			tokenState = "No Bearer token available"
		} else {
			tokenState = tokenSaved.(token.Token).GetTokenState()
		}

		var subscribedPostIDs = make(map[string]interface{})
		var hasPremiumPrivilege bool
		// Workaround without refactoring
		// "/story" path will not check member and subscription state
		isOriginalPathStory := (trimmedPath == "/story")

		if tokenState == token.OK && !isOriginalPathStory {
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
			// data := model.Member{}
			member := struct {
				Member model.Member `json:"member"`
			}{}

			client := graphql.NewClient(memberGraphqlEndpoint, graphql.WithHTTPClient(httpclient.DefaultNetHttpClient))
			err = client.Run(context.TODO(), req, &member)
			if err != nil {
				logger.Errorf("cannot fetch member and subscription state from member server")
				c.AbortWithStatusJSON(http.StatusInternalServerError, Reply{
					TokenState: tokenState,
				})
				return
			}

			data := member.Member
			if data.Subscription != nil {
				for _, s := range data.Subscription {
					if *s.Frequency == model.SubscriptionFrequencyTypeOneTime {
						subscribedPostIDs[*s.PostID] = nil
					} else {
						hasPremiumPrivilege = true
						break
					}
				}
			}
		}

		var body []byte
		redisKey := fmt.Sprintf("%s.%s.%s.%s", "apigateway", "proxy", "uri", c.Request.RequestURI)
		if cmd := rdb.Get(context.TODO(), redisKey); cmd == nil {
			// cache doesn't exist, do fetch reverse proxy
			logger.Infof("cache for uri(%s) cannot be fetched", c.Request.RequestURI)
		} else if body, err = cmd.Bytes(); err != nil {
			// cache can't be understood, do fetch reverse proxy
			logger.Warnf("cache for uri(%s) cannot be converted to bytes", c.Request.RequestURI)
		} else {
			switch path := c.Request.URL.Path; {
			case strings.HasSuffix(path, "/getposts") || strings.HasSuffix(path, "/posts") || strings.HasSuffix(path, "/post"):
				// break the switch to continue with response from proxied request
				var itemsLength int
				if itemsLength, body, err = modifyPostItems(logger, body, subscribedPostIDs, hasPremiumPrivilege); err != nil {
					logger.Warnf("modifyPostItems in cache encounter error: %s", err)
					break
				}

				if body, err = removePodtItemsHtml(body, itemsLength); err != nil {
					logger.Warnf("encounter error when deleting html in cache:", err)
					break
				}
				c.Header("GW-Cache", time.Now().Format(time.RFC3339))
				c.AbortWithStatusJSON(http.StatusOK, Reply{
					TokenState: tokenState,
					Data:       json.RawMessage(body),
				})
				return
			}
		}

		reverseProxy := httputil.ReverseProxy{
			Director:       director,
			ModifyResponse: ModifyReverseProxyResponse(c, rdb, cacheTTL, tokenState, subscribedPostIDs, hasPremiumPrivilege),
		}
		reverseProxy.ServeHTTP(c.Writer, c.Request)
	}
}

func ModifyReverseProxyResponse(c *gin.Context, rdb cache.Rediser, cacheTTL int, tokenState string, subscribedPostIDs map[string]interface{}, hasPremiumPrivilege bool) func(*http.Response) error {
	logger := logrus.WithFields(logrus.Fields{
		"path": c.FullPath(),
	})
	return func(r *http.Response) error {
		body, err := io.ReadAll(r.Body)
		_ = r.Body.Close()
		if err != nil {
			logger.Errorf("encounter error when reading proxy response:", err)
			return err
		}

		// Save the complete post as early as we can and run in in a goroutine
		redisKey := fmt.Sprintf("%s.%s.%s.%s", "apigateway", "proxy", "uri", c.Request.RequestURI)
		go func(rdb cache.Rediser, body []byte, redisKey string) {
			if err = rdb.Set(context.TODO(), redisKey, body, time.Duration(cacheTTL)*time.Second).Err(); err != nil {
				logger.Warnf("setting redis cache(%s) encountered error: %v", redisKey, err)
			}
		}(rdb, body, redisKey)

		switch path := r.Request.URL.Path; {
		// TODO refactor condition
		case strings.HasSuffix(path, "/getposts") || strings.HasSuffix(path, "/posts") || strings.HasSuffix(path, "/post"):

			// modify body if the item falls into a "member only" category

			var itemsLength int
			if itemsLength, body, err = modifyPostItems(logger, body, subscribedPostIDs, hasPremiumPrivilege); err != nil {
				logger.Errorf("modifyPostItems encounter error: %s", err)
				return err
			}

			if body, err = removePodtItemsHtml(body, itemsLength); err != nil {
				logger.Errorf("encounter error when deleting html:", err)
				return err
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

func modifyPostItems(logger *logrus.Entry, body []byte, subscribedPostIDs map[string]interface{}, hasPremiumPrivilege bool) (postItemsLength int, modifiedBody []byte, err error) {
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
		err = fmt.Errorf("unmarshal post body encountered error: %v", err)
		return 0, nil, err
	}

	// modify body at the end and truncate the post depending on the post and member state
	for i, item := range items.Items {
		for _, category := range item.Categories {
			isPostPremium := category.IsMemberOnly != nil && *category.IsMemberOnly
			if isPostPremium && isPostToBeTruncate(isPostPremium, item.ID, hasPremiumPrivilege, subscribedPostIDs) {
				truncatedAPIData := item.Content.APIData[0:3]
				body, err = sjson.SetBytes(body, fmt.Sprintf("_items.%d.content.apiData", i), truncatedAPIData)
				if err != nil {
					err = fmt.Errorf("encounter error when truncating apiData: %v", err)
					return 0, nil, err
				}
				body, err = sjson.SetBytes(body, fmt.Sprintf("_items.%d.isTruncated", i), true)
				if err != nil {
					err = fmt.Errorf("encounter error setting isTruncated to true for _items.%d: %v", i, err)
					return 0, nil, err
				}
				break
			} else {
				body, err = sjson.SetBytes(body, fmt.Sprintf("_items.%d.isTruncated", i), false)
				if err != nil {
					err = fmt.Errorf("encounter error setting isTruncated to false for _items.%d: %v", i, err)
					return 0, nil, err
				}
			}
		}
	}
	return len(items.Items), body, err
}

func removePodtItemsHtml(body []byte, itemsLength int) (modifiedBody []byte, err error) {
	for i := 0; i < itemsLength-1; i++ {
		body, err = sjson.DeleteBytes(body, fmt.Sprintf("_items.%d.content.html", i))
		if err != nil {
			return nil, err
		}
	}
	return body, err
}

func isPostToBeTruncate(isPostPremium bool, postID string, hasPremiumPrivilege bool, subscribedIds map[string]interface{}) bool {
	_, postSubscribed := subscribedIds[postID]
	return isPostPremium && !hasPremiumPrivilege && !postSubscribed
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
