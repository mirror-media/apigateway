package middleware

import (
	"reflect"
	"testing"
)

func Test_patchVariablesInGraphql(t *testing.T) {
	type args struct {
		input []byte
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "with variables",
			args: args{
				input: []byte(`{"query":"mutation ($id: ID!, $orderNumber: String!) {\n  updatesubscription(id: $id, data: {orderNumber: $orderNumber}) {\n    orderNumber\n  }\n}\n","variables":{"id":"2","orderNumber":"null"}}`),
			},
			want: []byte(`{"query":"mutation ($id: ID!, $orderNumber: String!) {\n  updatesubscription(id: $id, data: {orderNumber: $orderNumber}) {\n    orderNumber\n  }\n}\n","variables":"{\"id\":\"2\",\"orderNumber\":null}"}`),
		},
		{
			name: "without variables",
			args: args{
				input: []byte(`{"query":"mutation { createsubscription(data:{\n    paymentMethod: newebpay,\n    status: paying,\n    email:\"iam@email\",\n    note:\"this is a note\",\n    frequency: monthly,\n    amount: 323,\n    orderNumber: \"fsdfsd\",\n    member:{connect:{\n      id: 1\n    }}\n  }){\n      id\n      isActive\n      frequency\n      nextFrequency\n  }\n}"}`),
			},
			want: []byte(`{"query":"mutation { createsubscription(data:{\n    paymentMethod: newebpay,\n    status: paying,\n    email:\"iam@email\",\n    note:\"this is a note\",\n    frequency: monthly,\n    amount: 323,\n    orderNumber: \"fsdfsd\",\n    member:{connect:{\n      id: 1\n    }}\n  }){\n      id\n      isActive\n      frequency\n      nextFrequency\n  }\n}"}`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := patchNullVariablesInGraphql(tt.args.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("patchVariablesInGraphqlVariables() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("patchVariablesInGraphqlVariables() = %v, want %v", got, tt.want)
			}
		})
	}
}
