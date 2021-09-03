package payment

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-querystring/query"
	"github.com/mirror-media/apigateway/graph/member/model"
)

type Store struct {
	CallbackDomain      string
	CallbackProtocol    string
	ClientBackPath      string // FIXME Unknown
	ID                  string // FIXME Unknown
	IsAbleToModifyEmail int8   // Use 1
	LoginType           int8   // Use 0
	NotifyProtocol      string
	NotifyDomain        string // FIXME Unknown
	NotifyPath          string // FIXME Unknown
	P3D                 int8   // Use 1
	RespondType         string // Use JSON
	ReturnPath          string // FIXME Unknown
	Version             string // Use 1.6
}

type Merchandise struct {
	Code      string  `url:"code"`
	PostID    string  `url:"postId,omitempty"`
	PostSlug  string  `url:"postSlug,omitempty"`
	PostTitle string  `url:"postTitle,omitempty"`
	Amount    float64 `url:"amount"`
}

type PurchaseInfo struct {
	Merchandise
	PurchasedAtUnixTime int64  `url:"purchasedAtUnixTime"`
	OrderNumber         string `url:"orderNumber"`
	MemberFirebaseID    string `url:"memberFirebaseId,omitempty"`
	ReturnPath          string `url:"returnPath"`
}

func (s Store) getNotifyURL(purchaseInfo PurchaseInfo) (string, error) {
	protocol := s.NotifyProtocol
	domain := s.NotifyDomain
	path := s.NotifyPath
	return getCallbackUrl(protocol, domain, path, nil)
}

func (s Store) getReturnURL(purchaseInfo PurchaseInfo) (string, error) {
	protocol := s.CallbackProtocol
	domain := s.CallbackDomain
	path := s.ReturnPath
	return getCallbackUrl(protocol, domain, path, &purchaseInfo)
}

func (s Store) getClientBackPath(purchaseInfo PurchaseInfo) (string, error) {
	protocol := s.CallbackProtocol
	domain := s.CallbackDomain
	path := s.ClientBackPath
	return getCallbackUrl(protocol, domain, path, &purchaseInfo)
}

func getCallbackUrl(protocol, domain, path string, purchaseInfo *PurchaseInfo) (string, error) {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if purchaseInfo == nil {
		return fmt.Sprintf("%s://%s%s", protocol, domain, path), nil
	}
	v, err := query.Values(purchaseInfo)
	return fmt.Sprintf("%s://%s%s?%s", protocol, domain, path, v.Encode()), err
}

