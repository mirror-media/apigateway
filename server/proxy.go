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
	"github.com/mirror-media/mm-apigateway/cache"
	"github.com/mirror-media/mm-apigateway/middleware"
	"github.com/mirror-media/mm-apigateway/token"
	"github.com/sirupsen/logrus"
	"github.com/tidwall/sjson"
)

func NewSingleHostReverseProxy(target *url.URL, pathBaseToStrip string, rdb cache.Rediser, cacheTTL int) func(c *gin.Context) {
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
				key = fmt.Sprintf("%s.%s.%s.%s", "apigateway", "post", "notmember", c.Request.RequestURI)
			} else {
				key = fmt.Sprintf("%s.%s.%s.%s", "apigateway", "post", "member", c.Request.RequestURI)
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

		reverseProxy := httputil.ReverseProxy{Director: director}
		reverseProxy.ModifyResponse = ModifyReverseProxyResponse(c, rdb, cacheTTL)
		reverseProxy.ServeHTTP(c.Writer, c.Request)
	}
}

func ModifyReverseProxyResponse(c *gin.Context, rdb cache.Rediser, cacheTTL int) func(*http.Response) error {
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
			if tokenState == token.OK {
				// TODO refactor redis cache code
				redisKey = fmt.Sprintf("%s.%s.%s.%s", "mm-apigateway", "post", "member", c.Request.RequestURI)
			} else {
				// TODO refactor redis cache code
				redisKey = fmt.Sprintf("%s.%s.%s.%s", "mm-apigateway", "post", "notmember", c.Request.RequestURI)

				// modify body if the item falls into a "member only" category
				for i, item := range items.Items {
					for _, category := range item.Categories {
						if category.IsMemberOnly != nil && *category.IsMemberOnly {
							truncatedAPIData := item.Content.APIData[0:3]
							body, err = sjson.SetBytes(body, fmt.Sprintf("_items.%d.content.apiData", i), truncatedAPIData)
							if err != nil {
								logger.Errorf("encounter error when truncating apiData:", err)
								return err
							}
							break
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
