package payment

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/google/go-querystring/query"
	"github.com/mirror-media/apigateway/graph/member/model"
)

type NewebpayLoginType int8

const LoginNotAvailable NewebpayLoginType = 0
const LoginAvailable NewebpayLoginType = 1

type Boolean int8

const TRUE Boolean = 1
const False Boolean = 0

type NewebpayRespondType string

const RespondWithJSON NewebpayRespondType = "JSON"

type NewebPayStore struct {
	CallbackHost        string
	CallbackProtocol    string
	ClientBackPath      string            // ? Unknown
	ID                  string            // ? Unknown
	IsAbleToModifyEmail Boolean           // Use 1
	LoginType           NewebpayLoginType // Use 0
	NotifyProtocol      string
	NotifyHost          string              // ? Unknown
	NotifyPath          string              // ? Unknown
	Is3DSecure          Boolean             // Use 1
	RespondType         NewebpayRespondType // Use JSON
	ReturnPath          string              // ? Unknown
	Version             string              // Use 1.6
}

type Merchandise struct {
	Code      string  `url:"code"`
	PostID    string  `url:"postId,omitempty"`
	PostSlug  string  `url:"postSlug,omitempty"`
	PostTitle string  `url:"postTitle,omitempty"`
	Amount    float64 `url:"amount,omitempty"`
}

type PurchaseInfo struct {
	Merchandise
	PurchasedAtUnixTime int64  `url:"purchasedAtUnixTime,omitempty"`
	OrderNumber         string `url:"orderNumber,omitempty"`
	MemberFirebaseID    string `url:"memberFirebaseId,omitempty"`
	ReturnPath          string `url:"returnPath,omitempty"`
}

func (s NewebPayStore) getNotifyURL(purchaseInfo PurchaseInfo) (string, error) {
	protocol := s.NotifyProtocol
	domain := s.NotifyHost
	path := s.NotifyPath
	return getCallbackUrl(protocol, domain, path, &PurchaseInfo{
		Merchandise: Merchandise{
			Code: purchaseInfo.Code,
		},
	})
}

func (s NewebPayStore) getReturnURL(purchaseInfo PurchaseInfo) (string, error) {
	protocol := s.CallbackProtocol
	domain := s.CallbackHost
	path := s.ReturnPath
	return getCallbackUrl(protocol, domain, path, &purchaseInfo)
}

func (s NewebPayStore) getClientBackPath(purchaseInfo PurchaseInfo) (string, error) {
	protocol := s.CallbackProtocol
	domain := s.CallbackHost
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

type NewebpayTradeInfo struct {
	Amt                 int                 `url:"Amt"`
	ClientBackURL       string              `url:"ClientBackURL,omitempty"`
	Email               string              `url:"Email"`
	IsAbleToModifyEmail Boolean             `url:"EmailModify"`
	LoginType           NewebpayLoginType   `url:"LoginType"`
	MerchantOrderNo     string              `url:"MerchantOrderNo"`
	NotifyURL           string              `url:"NotifyURL,omitempty"`
	RespondType         NewebpayRespondType `url:"RespondType,omitempty"`
	ReturnURL           string              `url:"ReturnURL,omitempty"`
	StoreID             string              `url:"MerchantID"`
	TimeStamp           string              `url:"TimeStamp"`
	TokenTerm           string              `url:"TokenTerm,omitempty"`
	Version             string              `url:"Version"`
}

type NewebpayTradeInfoAgreement struct {
	NewebpayTradeInfo
	CreditAgreement int8    `url:"CREDITAGREEMENT"` // Use 1
	ItemDesc        string  `url:"ItemDesc"`
	OrderComment    string  `url:"OrderComment"`
	P3D             Boolean `url:"P3D"`
}

type NewebpayTradeInfoMGP struct {
	NewebpayTradeInfo
	ItemDesc        string `url:"ItemDesc"`
	OrderComment    string `url:"OrderComment,omitempty"`
	ItemDescription string `url:"ItemDesc"`
	TradeLimit      int    `url:"TradeLimit"`
}

type NewebpayAgreementInfo struct {
	Amount              int
	Email               string
	IsAbleToModifyEmail Boolean
	LoginType           NewebpayLoginType
	RespondType         NewebpayRespondType
	ItemDesc            string
	OrderComment        string
	TokenTerm           string
}

// Ref: https://github.com/mirror-media/apigateway/files/6866871/NewebPay_._._AGREEMENT_.1.0.6.pdf
func (s NewebPayStore) CreateNewebpayAgreementPayload(agreementInfo NewebpayAgreementInfo, purchaseInfo PurchaseInfo) (payload string, err error) {
	// Validate the data at the beginning for short circuit
	if purchaseInfo.PurchasedAtUnixTime <= 0 {
		return "", fmt.Errorf("purchaseInfo has invalid PurchasedAtUnixTime(%d)", purchaseInfo.PurchasedAtUnixTime)
	} else if agreementInfo.Amount <= 0 {
		return "", fmt.Errorf("agreementInfo has invalid amount(%d)", agreementInfo.Amount)
	} else if agreementInfo.Email == "" {
		return "", fmt.Errorf("agreementInfo has no email")
	} else if agreementInfo.OrderComment == "" {
		return "", fmt.Errorf("agreementInfo has no OrderComment")
	} else if i := strings.IndexAny(agreementInfo.OrderComment, unsafeCharacters); i != -1 {
		return "", fmt.Errorf("agreementInfo.OrderComment contains unsafe a character(%s)", (agreementInfo.OrderComment)[i:i+1])
	} else if err := validatePurchaseCode(purchaseInfo); err != nil {
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

	tradeInfo := NewebpayTradeInfoAgreement{
		NewebpayTradeInfo: NewebpayTradeInfo{
			Amt:                 agreementInfo.Amount,
			ClientBackURL:       clientBackURL,
			Email:               agreementInfo.Email,
			IsAbleToModifyEmail: Boolean(s.IsAbleToModifyEmail),
			LoginType:           s.LoginType,
			MerchantOrderNo:     purchaseInfo.OrderNumber,
			NotifyURL:           notifyURL,
			RespondType:         RespondWithJSON,
			ReturnURL:           returnURL,
			StoreID:             s.ID,
			TimeStamp:           strconv.FormatInt(purchaseInfo.PurchasedAtUnixTime, 10),
			TokenTerm:           agreementInfo.TokenTerm,
			Version:             s.Version,
		},
		CreditAgreement: 1,
		OrderComment:    agreementInfo.OrderComment,
		ItemDesc:        agreementInfo.ItemDesc,
		P3D:             s.Is3DSecure,
	}
	v, err := query.Values(tradeInfo)
	payload = v.Encode()
	return payload, err
}

type NewebpayMGPInfo struct {
	Amount              int
	Email               string
	IsAbleToModifyEmail Boolean
	LoginType           NewebpayLoginType
	RespondType         NewebpayRespondType
	ItemDescription     string // ? What should it be?
	OrderComment        string
	TokenTerm           string
}

// Ref: https://www.newebpay.com/website/Page/download_file?name=%E8%97%8D%E6%96%B0%E9%87%91%E6%B5%81Newebpay_MPG%E4%B8%B2%E6%8E%A5%E6%89%8B%E5%86%8A_MPG_1.1.0.pdf
func (s NewebPayStore) CreateNewebpayMPGPayload(newebpayMGPInfo NewebpayMGPInfo, purchaseInfo PurchaseInfo) (payload string, err error) {
	// Validate the data at the beginning for short circuit

	// Validate the data at the beginning for short circuit
	if purchaseInfo.PurchasedAtUnixTime <= 0 {
		return "", fmt.Errorf("purchaseInfo has invalid PurchasedAtUnixTime(%d)", purchaseInfo.PurchasedAtUnixTime)
	} else if newebpayMGPInfo.Amount <= 0 {
		return "", fmt.Errorf("newebpayMGPInfo has invalid amount(%d)", newebpayMGPInfo.Amount)
	} else if newebpayMGPInfo.Email == "" {
		return "", fmt.Errorf("newebpayMGPInfo has no email")
	} else if newebpayMGPInfo.ItemDescription == "" {
		return "", fmt.Errorf("newebpayMGPInfo has no ItemDescription")
	} else if i := strings.IndexAny(newebpayMGPInfo.ItemDescription, unsafeCharacters); i != -1 {
		return "", fmt.Errorf("newebpayMGPInfo.ItemDescription contains unsafe a character(%s)", (newebpayMGPInfo.ItemDescription)[i:i+1])
	} else if purchaseInfo.Code == "" {
		return "", fmt.Errorf("purchaseInfo has no code")
	} else if err := validatePurchaseCode(purchaseInfo); err != nil {
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

	tradeInfo := NewebpayTradeInfoMGP{
		NewebpayTradeInfo: NewebpayTradeInfo{
			Amt:                 newebpayMGPInfo.Amount,
			ClientBackURL:       clientBackURL,
			Email:               newebpayMGPInfo.Email,
			IsAbleToModifyEmail: s.IsAbleToModifyEmail,
			LoginType:           s.LoginType,
			MerchantOrderNo:     purchaseInfo.OrderNumber,
			NotifyURL:           notifyURL,
			RespondType:         "JSON",
			ReturnURL:           returnURL,
			StoreID:             s.ID,
			TimeStamp:           strconv.FormatInt(purchaseInfo.PurchasedAtUnixTime, 10),
			TokenTerm:           newebpayMGPInfo.TokenTerm,
			Version:             s.Version,
		},
		ItemDescription: newebpayMGPInfo.ItemDescription,
		OrderComment:    newebpayMGPInfo.OrderComment,
		TradeLimit:      900,
	}
	v, err := query.Values(tradeInfo)
	payload = v.Encode()
	return payload, err
}

func validatePurchaseCode(purchaseInfo PurchaseInfo) error {
	codes := map[string]interface{}{
		model.SubscriptionFrequencyTypeMonthly.String(): nil,
		model.SubscriptionFrequencyTypeYearly.String():  nil,
		model.SubscriptionFrequencyTypeOneTime.String(): nil,
	}

	if _, ok := codes[purchaseInfo.Code]; !ok {
		return fmt.Errorf("purchaseInfo has invalid code(%s)", purchaseInfo.Code)
	} else if purchaseInfo.Code != model.SubscriptionFrequencyTypeOneTime.String() && (purchaseInfo.PostID != "" || purchaseInfo.PostSlug != "" || purchaseInfo.PostTitle != "") {
		return fmt.Errorf("purchaseInfo code is not %s, but postID is provided", model.SubscriptionFrequencyTypeOneTime)
	} else if purchaseInfo.Code == model.SubscriptionFrequencyTypeOneTime.String() && purchaseInfo.PostID == "" {
		return fmt.Errorf("purchaseInfo code is %s, but postID is not provided", model.SubscriptionFrequencyTypeOneTime)
	}
	return nil
}
