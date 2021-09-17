package graph

import (
	"reflect"
	"testing"
)

func TestReplaceNullString(t *testing.T) {
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
			name: "one level",
			args: args{
				input: []byte(`{"key": "null"}`),
			},
			want: []byte(`{"key":null}`),
		},
		{
			name: "multi levels",
			args: args{
				input: []byte("{\n\n\"key1\": \"null\", \"key2\": {\n\t\"key1\": \"null\", \n\t\t\"key2\": [\"null\",\"null\",\"null\",\"null\", {\n\t\t\"key1\": \"null\", \"key2\": \"MF\"}]}}}"),
			},
			want: []byte(`{"key1":null,"key2":{"key1":null,"key2":[null,null,null,null,{"key1":null,"key2":"MF"}]}}`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReplaceNullString(tt.args.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReplaceNullString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReplaceNullString() = %v, want %v", got, tt.want)
			}
		})
	}
}
