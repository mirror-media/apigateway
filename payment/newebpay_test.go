package payment

import (
	"testing"
	"time"

	"github.com/mirror-media/apigateway/graph/member/model"
)

func TestStore_CreateNewebpayAgreementPayload(t *testing.T) {
	amount := 999
	timeUnix := int64(1630665558)
	createAt := time.Unix(timeUnix, 0).Format(time.RFC3339)
	email := "aaa@bbb.cc"
	orderNumber := "c4ovmad948155cflse1g"
	description := "subscription description"
	subscription := model.Subscription{
		Amount:      &amount,
		CreatedAt:   &createAt,
		OrderNumber: &orderNumber,
		Email:       &email,
		Desc:        &description,
	}
	unsafeChar := "\n"
	unsafeCharSubscription := subscription
	unsafeCharSubscription.Desc = &unsafeChar
	type fields struct {
		ClientBackURL       string
		CreditAgreement     int8
		ID                  string
		IsAbleToModifyEmail int8
		LoginType           int8
		NotifyURL           string
		P3D                 int8
		RespondType         string
		ReturnURL           string
		Version             string
	}
	store := fields{
		ClientBackURL:       "http://ClientBackURL/1?a=b&c=d",
		CreditAgreement:     1,
		ID:                  "store id",
		IsAbleToModifyEmail: 1,
		LoginType:           0,
		NotifyURL:           "http://NotifyURL/1?a=b&c=d",
		P3D:                 1,
		RespondType:         "JSON",
		ReturnURL:           "http://ReturnURL",
		Version:             "1.6",
	}
	type args struct {
		tokenTerm    string
		subscription model.Subscription
		purchaseInfo PurchaseInfo
	}
	tests := []struct {
		name        string
		fields      fields
		args        args
		wantPayload string
		wantErr     bool
	}{
		{
			name:   "Successfull Recurring Case",
			fields: store,
			args: args{
				tokenTerm:    "token_term",
				subscription: subscription,
				purchaseInfo: PurchaseInfo{
					Merchandise: Merchandise{
						Code:      "yearly",
						PostID:    "postid",
						PostSlug:  "postslug",
						PostTitle: "posttitle",
						Amount:    8888,
					},
					PurchasedAtUnixTime: 111,
					OrderNumber:         "ordernumber",
					MemberFirebaseID:    "memberid",
					ReturnPath:          "/story/abc",
				},
			},
			wantPayload: "Amt=999&CREDITAGREEMENT=1&ClientBackURL=%3A%2F%2F%2Fhttp%3A%2F%2FClientBackURL%2F1%3Fa%3Db%26c%3Dd%3Famount%3D8888%26code%3Dyearly%26memberFirebaseId%3Dmemberid%26orderNumber%3Dordernumber%26postId%3Dpostid%26postSlug%3Dpostslug%26postTitle%3Dposttitle%26purchasedAtUnixTime%3D111%26returnPath%3D%252Fstory%252Fabc&Email=aaa%40bbb.cc&EmailModify=1&LoginType=0&MerchantID=store+id&MerchantOrderNo=c4ovmad948155cflse1g&NotifyURL=%3A%2F%2F%2Fhttp%3A%2F%2FNotifyURL%2F1%3Fa%3Db%26c%3Dd&OrderComment=subscription+description&P3D=1&RespondType=JSON&ReturnURL=%3A%2F%2F%2Fhttp%3A%2F%2FReturnURL%3Famount%3D8888%26code%3Dyearly%26memberFirebaseId%3Dmemberid%26orderNumber%3Dordernumber%26postId%3Dpostid%26postSlug%3Dpostslug%26postTitle%3Dposttitle%26purchasedAtUnixTime%3D111%26returnPath%3D%252Fstory%252Fabc&TimeStamp=1630665558&TokenTerm=token_term&Version=1.6",
			wantErr:     false,
		},
		{
			name:   "unsafe char",
			fields: store,
			args: args{
				tokenTerm:    "token_term",
				subscription: unsafeCharSubscription,
			},
			wantPayload: "",
			wantErr:     true,
		},
		{
			name:   "invalid empty subscription",
			fields: store,
			args: args{
				tokenTerm:    "token_term",
				subscription: model.Subscription{},
			},
			wantPayload: "",
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Store{
				ClientBackPath:      tt.fields.ClientBackURL,
				ID:                  tt.fields.ID,
				IsAbleToModifyEmail: tt.fields.IsAbleToModifyEmail,
				LoginType:           tt.fields.LoginType,
				NotifyPath:          tt.fields.NotifyURL,
				P3D:                 tt.fields.P3D,
				RespondType:         tt.fields.RespondType,
				ReturnPath:          tt.fields.ReturnURL,
				Version:             tt.fields.Version,
			}
			gotPayload, err := s.CreateNewebpayAgreementPayload(tt.args.tokenTerm, tt.args.subscription, tt.args.purchaseInfo)
			if (err != nil) != tt.wantErr {
				t.Errorf("Store.CreateNewebpayAgreementPayload() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotPayload != tt.wantPayload {
				t.Errorf("Store.CreateNewebpayAgreementPayload() = %v, want %v", gotPayload, tt.wantPayload)
			}
		})
	}
}

func TestStore_GetNotifyURL(t *testing.T) {
	type fields struct {
		CallbackDomain      string
		CallbackProtocol    string
		ClientBackPath      string
		ID                  string
		IsAbleToModifyEmail int8
		LoginType           int8
		NotifyDomain        string
		NotifyPath          string
		NotifyProtocol      string
		P3D                 int8
		RespondType         string
		ReturnPath          string
		Version             string
	}
	f := fields{
		NotifyPath:     "/notify-payment",
		NotifyProtocol: "https",
		NotifyDomain:   "domain",
	}
	type args struct {
		purchaseInfo PurchaseInfo
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "one-time",
			args: args{
				PurchaseInfo{
					Merchandise: Merchandise{
						Code:      "one_time",
						PostID:    "postid",
						PostSlug:  "postslug",
						PostTitle: "posttitle",
						Amount:    8888,
					},
					PurchasedAtUnixTime: 111,
					OrderNumber:         "ordernumber",
					MemberFirebaseID:    "memberid",
					ReturnPath:          "/story/abc",
				},
			},
			fields: f,
			want:   "https://domain/notify-payment",
		},
		{
			name: "monthly",
			args: args{
				PurchaseInfo{
					Merchandise: Merchandise{
						Code:   "monthly",
						Amount: 8888,
					},
					PurchasedAtUnixTime: 111,
					OrderNumber:         "ordernumber",
					MemberFirebaseID:    "memberid",
					ReturnPath:          "/story/abc",
				},
			},
			fields: f,
			want:   "https://domain/notify-payment",
		},
		{
			name: "yearly",
			args: args{
				PurchaseInfo{
					Merchandise: Merchandise{
						Code:   "yearly",
						Amount: 8888,
					},
					PurchasedAtUnixTime: 111,
					OrderNumber:         "ordernumber",
					MemberFirebaseID:    "memberid",
					ReturnPath:          "/story/abc",
				},
			},
			fields: f,
			want:   "https://domain/notify-payment",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Store{
				CallbackDomain:      tt.fields.CallbackDomain,
				CallbackProtocol:    tt.fields.CallbackProtocol,
				ClientBackPath:      tt.fields.ClientBackPath,
				ID:                  tt.fields.ID,
				IsAbleToModifyEmail: tt.fields.IsAbleToModifyEmail,
				LoginType:           tt.fields.LoginType,
				NotifyDomain:        tt.fields.NotifyDomain,
				NotifyPath:          tt.fields.NotifyPath,
				NotifyProtocol:      tt.fields.NotifyProtocol,
				P3D:                 tt.fields.P3D,
				RespondType:         tt.fields.RespondType,
				ReturnPath:          tt.fields.ReturnPath,
				Version:             tt.fields.Version,
			}
			got, err := s.getNotifyURL(tt.args.purchaseInfo)
			if (err != nil) != tt.wantErr {
				t.Errorf("Store.GetNotifyURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Store.GetNotifyURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStore_GetReturnURL(t *testing.T) {
	type fields struct {
		CallbackDomain      string
		CallbackProtocol    string
		ClientBackPath      string
		ID                  string
		IsAbleToModifyEmail int8
		LoginType           int8
		NotifyDomain        string
		NotifyPath          string
		NotifyProtocol      string
		P3D                 int8
		RespondType         string
		ReturnPath          string
		Version             string
	}
	f := fields{
		ReturnPath:       "/complete-purchase",
		CallbackDomain:   "domain",
		CallbackProtocol: "https",
	}
	type args struct {
		purchaseInfo PurchaseInfo
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "one-time",
			args: args{
				PurchaseInfo{
					Merchandise: Merchandise{
						Code:      "one_time",
						PostID:    "postid",
						PostSlug:  "postslug",
						PostTitle: "posttitle",
						Amount:    8888,
					},
					PurchasedAtUnixTime: 111,
					OrderNumber:         "ordernumber",
					MemberFirebaseID:    "memberid",
					ReturnPath:          "/story/abc",
				},
			},
			fields: f,
			want:   "https://domain/complete-purchase?amount=8888&code=one_time&memberFirebaseId=memberid&orderNumber=ordernumber&postId=postid&postSlug=postslug&postTitle=posttitle&purchasedAtUnixTime=111&returnPath=%2Fstory%2Fabc",
		},
		{
			name: "monthly",
			args: args{
				PurchaseInfo{
					Merchandise: Merchandise{
						Code:   "monthly",
						Amount: 8888,
					},
					PurchasedAtUnixTime: 111,
					OrderNumber:         "ordernumber",
					MemberFirebaseID:    "memberid",
					ReturnPath:          "/story/abc",
				},
			},
			fields: f,
			want:   "https://domain/complete-purchase?amount=8888&code=monthly&memberFirebaseId=memberid&orderNumber=ordernumber&purchasedAtUnixTime=111&returnPath=%2Fstory%2Fabc",
		},
		{
			name: "yearly",
			args: args{
				PurchaseInfo{
					Merchandise: Merchandise{
						Code:   "yearly",
						Amount: 8888,
					},
					PurchasedAtUnixTime: 111,
					OrderNumber:         "ordernumber",
					MemberFirebaseID:    "memberid",
					ReturnPath:          "/story/abc",
				},
			},
			fields: f,
			want:   "https://domain/complete-purchase?amount=8888&code=yearly&memberFirebaseId=memberid&orderNumber=ordernumber&purchasedAtUnixTime=111&returnPath=%2Fstory%2Fabc",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Store{
				CallbackDomain:      tt.fields.CallbackDomain,
				CallbackProtocol:    tt.fields.CallbackProtocol,
				ClientBackPath:      tt.fields.ClientBackPath,
				ID:                  tt.fields.ID,
				IsAbleToModifyEmail: tt.fields.IsAbleToModifyEmail,
				LoginType:           tt.fields.LoginType,
				NotifyDomain:        tt.fields.NotifyDomain,
				NotifyPath:          tt.fields.NotifyPath,
				NotifyProtocol:      tt.fields.NotifyProtocol,
				P3D:                 tt.fields.P3D,
				RespondType:         tt.fields.RespondType,
				ReturnPath:          tt.fields.ReturnPath,
				Version:             tt.fields.Version,
			}
			got, err := s.getReturnURL(tt.args.purchaseInfo)
			if (err != nil) != tt.wantErr {
				t.Errorf("Store.GetReturnURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Store.GetReturnURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStore_GetClientBackPath(t *testing.T) {
	type fields struct {
		CallbackDomain      string
		CallbackProtocol    string
		ClientBackPath      string
		ID                  string
		IsAbleToModifyEmail int8
		LoginType           int8
		NotifyDomain        string
		NotifyPath          string
		NotifyProtocol      string
		P3D                 int8
		RespondType         string
		ReturnPath          string
		Version             string
	}
	f := fields{
		ClientBackPath:   "/cancel-purchase",
		CallbackDomain:   "domain",
		CallbackProtocol: "https",
	}
	type args struct {
		protocol     string
		domain       string
		purchaseInfo PurchaseInfo
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "one-time",
			args: args{
				"https",
				"domain",
				PurchaseInfo{
					Merchandise: Merchandise{
						Code:      "one_time",
						PostID:    "postid",
						PostSlug:  "postslug",
						PostTitle: "posttitle",
						Amount:    8888,
					},
					PurchasedAtUnixTime: 111,
					OrderNumber:         "ordernumber",
					MemberFirebaseID:    "memberid",
					ReturnPath:          "/story/abc",
				},
			},
			fields: f,
			want:   "https://domain/cancel-purchase?amount=8888&code=one_time&memberFirebaseId=memberid&orderNumber=ordernumber&postId=postid&postSlug=postslug&postTitle=posttitle&purchasedAtUnixTime=111&returnPath=%2Fstory%2Fabc",
		},
		{
			name: "monthly",
			args: args{
				"https",
				"domain",
				PurchaseInfo{
					Merchandise: Merchandise{
						Code:   "monthly",
						Amount: 8888,
					},
					PurchasedAtUnixTime: 111,
					OrderNumber:         "ordernumber",
					MemberFirebaseID:    "memberid",
					ReturnPath:          "/story/abc",
				},
			},
			fields: f,
			want:   "https://domain/cancel-purchase?amount=8888&code=monthly&memberFirebaseId=memberid&orderNumber=ordernumber&purchasedAtUnixTime=111&returnPath=%2Fstory%2Fabc",
		},
		{
			name: "yearly",
			args: args{
				"https",
				"domain",
				PurchaseInfo{
					Merchandise: Merchandise{
						Code:   "yearly",
						Amount: 8888,
					},
					PurchasedAtUnixTime: 111,
					OrderNumber:         "ordernumber",
					MemberFirebaseID:    "memberid",
					ReturnPath:          "/story/abc",
				},
			},
			fields: f,
			want:   "https://domain/cancel-purchase?amount=8888&code=yearly&memberFirebaseId=memberid&orderNumber=ordernumber&purchasedAtUnixTime=111&returnPath=%2Fstory%2Fabc",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Store{
				CallbackDomain:      tt.fields.CallbackDomain,
				CallbackProtocol:    tt.fields.CallbackProtocol,
				ClientBackPath:      tt.fields.ClientBackPath,
				ID:                  tt.fields.ID,
				IsAbleToModifyEmail: tt.fields.IsAbleToModifyEmail,
				LoginType:           tt.fields.LoginType,
				NotifyDomain:        tt.fields.NotifyDomain,
				NotifyPath:          tt.fields.NotifyPath,
				NotifyProtocol:      tt.fields.NotifyProtocol,
				P3D:                 tt.fields.P3D,
				RespondType:         tt.fields.RespondType,
				ReturnPath:          tt.fields.ReturnPath,
				Version:             tt.fields.Version,
			}
			got, err := s.getClientBackPath(tt.args.purchaseInfo)
			if (err != nil) != tt.wantErr {
				t.Errorf("Store.GetClientBackPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Store.GetClientBackPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
