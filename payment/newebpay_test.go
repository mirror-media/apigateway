package payment

import (
	"testing"
	"time"

	"github.com/mirror-media/apigateway/graph/member/model"
)

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
				CallbackInfo{
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
				CallbackInfo{
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
				CallbackInfo{
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
				CallbackInfo{
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
				CallbackInfo{
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
				CallbackInfo{
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
				CallbackInfo{
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
				CallbackInfo{
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
				CallbackInfo{
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
