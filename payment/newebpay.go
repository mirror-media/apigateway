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

type TradeInfo struct {
	Amt                 int    `url:"Amt"`
	ClientBackURL       string `url:"ClientBackURL,omitempty"`
	Email               string `url:"Email"`
	IsAbleToModifyEmail int8   `url:"EmailModify"`
	LoginType           int8   `url:"LoginType"`
	MerchantOrderNo     string `url:"MerchantOrderNo"`
	NotifyURL           string `url:"NotifyURL,omitempty"`
	RespondType         string `url:"RespondType,omitempty"`
	ReturnURL           string `url:"ReturnURL,omitempty"`
	StoreID             string `url:"MerchantID"`
	TimeStamp           string `url:"TimeStamp"`
	Version             string `url:"Version"`
}

type TradeInfoAgreement struct {
	TradeInfo
	CreditAgreement int8   `url:"CREDITAGREEMENT"` // Use 1
	OrderComment    string `url:"OrderComment"`
	P3D             int8   `url:"P3D"`
	TokenTerm       string `url:"TokenTerm"`
}

type TradeInfoMGP struct {
	TradeInfo
	OrderComment    string `url:"OrderComment,omitempty"`
	ItemDescription string `url:"ItemDesc"`
}

const unsafeCharacters = ":/?#[]@!$&'()*+,;=<>%{}|\\^\"`\n"

// Ref: https://github.com/mirror-media/apigateway/files/6866871/NewebPay_._._AGREEMENT_.1.0.6.pdf
func (s Store) CreateNewebpayAgreementPayload(firebaseID, tokenTerm string, subscription model.Subscription, purchaseInfo PurchaseInfo) (payload string, err error) {
	// Validate the data at the beginning for short circuit
	if subscription.CreatedAt == nil {
		return "", fmt.Errorf("subscription(%s) has not creation time", subscription.ID)
	} else if subscription.Amount == nil {
		return "", fmt.Errorf("subscription(%s) has no amount", subscription.ID)
	} else if subscription.OrderNumber == nil {
		return "", fmt.Errorf("subscription(%s) has no OrderNumber", subscription.ID)
	} else if subscription.Email == nil {
		return "", fmt.Errorf("subscription(%s) has no email", subscription.ID)
	} else if subscription.Desc == nil {
		return "", fmt.Errorf("subscription(%s) has no descrption", subscription.ID)
	} else if i := strings.IndexAny(*subscription.Desc, unsafeCharacters); i != -1 {
		return "", fmt.Errorf("subscription(%s) description contains unsafe a character(%s)", subscription.ID, (*subscription.Desc)[i:i+1])
	}

	timestamp, err := time.Parse(time.RFC3339, *subscription.CreatedAt)
	if err != nil {
		return "", err
	}

	returnURL, err := s.getReturnURL(purchaseInfo)
	if err != nil {
		return "", nil
	}

	clientBackURL, err := s.getClientBackPath(purchaseInfo)
	if err != nil {
		return "", nil
	}

	notifyURL, err := s.getNotifyURL(purchaseInfo)
	if err != nil {
		return "", nil
	}

	tradeInfo := TradeInfoAgreement{
		TradeInfo: TradeInfo{
			Amt:                 *subscription.Amount,
			ClientBackURL:       clientBackURL,
			Email:               *subscription.Email,
			IsAbleToModifyEmail: s.IsAbleToModifyEmail,
			LoginType:           s.LoginType,
			MerchantOrderNo:     *subscription.OrderNumber,
			NotifyURL:           notifyURL,
			RespondType:         "JSON",
			ReturnURL:           returnURL,
			StoreID:             s.ID,
			TimeStamp:           strconv.FormatInt(timestamp.Unix(), 10),
			Version:             s.Version,
		},
		CreditAgreement: 1,
		// FIXME Are you sure?
		OrderComment: *subscription.Desc,
		P3D:          s.P3D,
		TokenTerm:    tokenTerm,
	}
	v, err := query.Values(tradeInfo)
	payload = v.Encode()
	return payload, err
}

// Ref: https://www.newebpay.com/website/Page/download_file?name=%E8%97%8D%E6%96%B0%E9%87%91%E6%B5%81Newebpay_MPG%E4%B8%B2%E6%8E%A5%E6%89%8B%E5%86%8A_MPG_1.1.0.pdf
func (s Store) CreateNewebpayMPGPayload(firebaseID, tokenTerm string, subscription model.Subscription, purchaseInfo PurchaseInfo) (payload string, err error) {
	// Validate the data at the beginning for short circuit
	if subscription.CreatedAt == nil {
		return "", fmt.Errorf("subscription(%s) has not creation time", subscription.ID)
	} else if subscription.Amount == nil {
		return "", fmt.Errorf("subscription(%s) has no amount", subscription.ID)
	} else if subscription.OrderNumber == nil {
		return "", fmt.Errorf("subscription(%s) has no OrderNumber", subscription.ID)
	} else if subscription.Email == nil {
		return "", fmt.Errorf("subscription(%s) has no email", subscription.ID)
	} else if subscription.Desc == nil {
		return "", fmt.Errorf("subscription(%s) has no descrption", subscription.ID)
	} else if subscription.Frequency == nil {
		return "", fmt.Errorf("subscription(%s) has no frequency", subscription.ID)
	} else if model.SubscriptionFrequencyTypeOneTime != *subscription.Frequency && purchaseInfo.Merchandise.PostID != "" {
		return "", fmt.Errorf("merchandise is not %s, but postID is provided", model.SubscriptionFrequencyTypeOneTime)
	} else if model.SubscriptionFrequencyTypeOneTime == *subscription.Frequency && purchaseInfo.Merchandise.PostID == "" {
		return "", fmt.Errorf("merchandise is %s, but postID is not provided", model.SubscriptionFrequencyTypeOneTime)
	} else if subscription.Desc == nil {
		return "", fmt.Errorf("subscription(%s) has no descrption", subscription.ID)
	} else if i := strings.IndexAny(*subscription.Desc, unsafeCharacters); i != -1 {
		return "", fmt.Errorf("subscription(%s) description contains unsafe a character(%s)", subscription.ID, (*subscription.Desc)[i:i+1])
	}

	timestamp, err := time.Parse(time.RFC3339, *subscription.CreatedAt)
	if err != nil {
		return "", err
	}

	returnURL, err := s.getReturnURL(purchaseInfo)
	if err != nil {
		return "", nil
	}

	clientBackURL, err := s.getClientBackPath(purchaseInfo)
	if err != nil {
		return "", nil
	}

	notifyURL, err := s.getNotifyURL(purchaseInfo)
	if err != nil {
		return "", nil
	}

	tradeInfo := TradeInfoMGP{
		TradeInfo: TradeInfo{
			Amt:                 *subscription.Amount,
			ClientBackURL:       clientBackURL,
			Email:               *subscription.Email,
			IsAbleToModifyEmail: s.IsAbleToModifyEmail,
			LoginType:           s.LoginType,
			MerchantOrderNo:     *subscription.OrderNumber,
			NotifyURL:           notifyURL,
			RespondType:         "JSON",
			ReturnURL:           returnURL,
			StoreID:             s.ID,
			TimeStamp:           strconv.FormatInt(timestamp.Unix(), 10),
			Version:             s.Version,
		},
		// FIXME Are you sure?
		ItemDescription: *subscription.Desc,
		// FIXME Are you sure?
		OrderComment: "",
	}
	v, err := query.Values(tradeInfo)
	payload = v.Encode()
	return payload, err
}
