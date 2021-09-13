package payment

import (
	"testing"
)

func TestStore_GetNotifyURL(t *testing.T) {
	type fields struct {
		CallbackDomain      string
		CallbackProtocol    string
		ClientBackPath      string
		ID                  string
		IsAbleToModifyEmail Boolean
		LoginType           NewebpayLoginType
		NotifyDomain        string
		NotifyPath          string
		NotifyProtocol      string
		Is3DSecure          Boolean
		RespondType         NewebpayRespondType
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
			s := NewebPayStore{
				CallbackHost:        tt.fields.CallbackDomain,
				CallbackProtocol:    tt.fields.CallbackProtocol,
				ClientBackPath:      tt.fields.ClientBackPath,
				ID:                  tt.fields.ID,
				IsAbleToModifyEmail: tt.fields.IsAbleToModifyEmail,
				LoginType:           tt.fields.LoginType,
				NotifyHost:          tt.fields.NotifyDomain,
				NotifyPath:          tt.fields.NotifyPath,
				NotifyProtocol:      tt.fields.NotifyProtocol,
				Is3DSecure:          tt.fields.Is3DSecure,
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
		IsAbleToModifyEmail Boolean
		LoginType           NewebpayLoginType
		NotifyDomain        string
		NotifyPath          string
		NotifyProtocol      string
		Is3DSecure          Boolean
		RespondType         NewebpayRespondType
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
			s := NewebPayStore{
				CallbackHost:        tt.fields.CallbackDomain,
				CallbackProtocol:    tt.fields.CallbackProtocol,
				ClientBackPath:      tt.fields.ClientBackPath,
				ID:                  tt.fields.ID,
				IsAbleToModifyEmail: tt.fields.IsAbleToModifyEmail,
				LoginType:           tt.fields.LoginType,
				NotifyHost:          tt.fields.NotifyDomain,
				NotifyPath:          tt.fields.NotifyPath,
				NotifyProtocol:      tt.fields.NotifyProtocol,
				Is3DSecure:          tt.fields.Is3DSecure,
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
		IsAbleToModifyEmail Boolean
		LoginType           NewebpayLoginType
		NotifyDomain        string
		NotifyPath          string
		NotifyProtocol      string
		Is3DSecure          Boolean
		RespondType         NewebpayRespondType
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
			s := NewebPayStore{
				CallbackHost:        tt.fields.CallbackDomain,
				CallbackProtocol:    tt.fields.CallbackProtocol,
				ClientBackPath:      tt.fields.ClientBackPath,
				ID:                  tt.fields.ID,
				IsAbleToModifyEmail: tt.fields.IsAbleToModifyEmail,
				LoginType:           tt.fields.LoginType,
				NotifyHost:          tt.fields.NotifyDomain,
				NotifyPath:          tt.fields.NotifyPath,
				NotifyProtocol:      tt.fields.NotifyProtocol,
				Is3DSecure:          tt.fields.Is3DSecure,
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

func TestStore_CreateNewebpayAgreementPayload(t *testing.T) {
	type fields struct {
		CallbackDomain      string
		CallbackProtocol    string
		ClientBackPath      string
		ID                  string
		IsAbleToModifyEmail Boolean
		LoginType           NewebpayLoginType
		NotifyProtocol      string
		NotifyDomain        string
		NotifyPath          string
		Is3DSecure          Boolean
		RespondType         NewebpayRespondType
		ReturnPath          string
		Version             string
	}

	store := fields{
		ClientBackPath:      "clientback",
		CallbackDomain:      "clientbackdomain",
		CallbackProtocol:    "https",
		ID:                  "store id",
		IsAbleToModifyEmail: TRUE,
		LoginType:           LoginNotAvailable,
		NotifyPath:          "notify",
		NotifyProtocol:      "https",
		NotifyDomain:        "notifydomain",
		Is3DSecure:          TRUE,
		RespondType:         "JSON",
		ReturnPath:          "http://returnpath",
		Version:             "1.6",
	}
	type args struct {
		agreementInfo NewebpayAgreementInfo
		purchaseInfo  PurchaseInfo
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
				agreementInfo: NewebpayAgreementInfo{
					Amount:              8888,
					Email:               "email@mail.com",
					IsAbleToModifyEmail: TRUE,
					LoginType:           LoginAvailable,
					RespondType:         RespondWithJSON,
					OrderComment:        "comment",
					TokenTerm:           "firebaseID",
					CreationTimeUnix:    123,
				},
				purchaseInfo: PurchaseInfo{
					Merchandise: Merchandise{
						Code:   "yearly",
						Amount: 8888,
					},
					PurchasedAtUnixTime: 111,
					OrderNumber:         "ordernumber",
					MemberFirebaseID:    "memberid",
					ReturnPath:          "/story/abc", // ? vunlunrable?
				},
			},
			wantPayload: "Amt=8888&CREDITAGREEMENT=1&ClientBackURL=https%3A%2F%2Fclientbackdomain%2Fclientback%3Famount%3D8888%26code%3Dyearly%26memberFirebaseId%3Dmemberid%26orderNumber%3Dordernumber%26purchasedAtUnixTime%3D111%26returnPath%3D%252Fstory%252Fabc&Email=email%40mail.com&EmailModify=1&LoginType=0&MerchantID=store+id&MerchantOrderNo=ordernumber&NotifyURL=https%3A%2F%2Fnotifydomain%2Fnotify&OrderComment=comment&P3D=1&RespondType=JSON&ReturnURL=https%3A%2F%2Fclientbackdomain%2Fhttp%3A%2F%2Freturnpath%3Famount%3D8888%26code%3Dyearly%26memberFirebaseId%3Dmemberid%26orderNumber%3Dordernumber%26purchasedAtUnixTime%3D111%26returnPath%3D%252Fstory%252Fabc&TimeStamp=123&TokenTerm=firebaseID&Version=1.6",
			wantErr:     false,
		},
		{
			name:   "unsafe char",
			fields: store,
			args: args{
				agreementInfo: NewebpayAgreementInfo{
					OrderComment: "\n!",
				},
			},
			wantPayload: "",
			wantErr:     true,
		},
		{
			name:   "invalid empty agreement info",
			fields: store,
			args: args{
				agreementInfo: NewebpayAgreementInfo{},
			},
			wantPayload: "",
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewebPayStore{
				CallbackHost:        tt.fields.CallbackDomain,
				CallbackProtocol:    tt.fields.CallbackProtocol,
				ClientBackPath:      tt.fields.ClientBackPath,
				ID:                  tt.fields.ID,
				IsAbleToModifyEmail: tt.fields.IsAbleToModifyEmail,
				LoginType:           tt.fields.LoginType,
				NotifyProtocol:      tt.fields.NotifyProtocol,
				NotifyHost:          tt.fields.NotifyDomain,
				NotifyPath:          tt.fields.NotifyPath,
				Is3DSecure:          tt.fields.Is3DSecure,
				RespondType:         tt.fields.RespondType,
				ReturnPath:          tt.fields.ReturnPath,
				Version:             tt.fields.Version,
			}
			gotPayload, err := s.CreateNewebpayAgreementPayload(tt.args.agreementInfo, tt.args.purchaseInfo)
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
