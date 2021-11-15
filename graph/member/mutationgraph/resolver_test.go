package mutationgraph

import (
	"testing"
	"time"
)

func Test_createOrderNumberByTaipeiTZ(t *testing.T) {
	utcPM10, _ := time.Parse(time.RFC3339, "2021-11-07T22:00:00+00:00")
	utcAM10, _ := time.Parse(time.RFC3339, "2021-11-07T10:00:00+00:00")
	astAM10, _ := time.Parse(time.RFC3339, "2021-11-07T10:00:00-09:00")
	type args struct {
		t  time.Time
		id uint64
	}
	tests := []struct {
		name            string
		args            args
		wantOrderNumber string
	}{
		{
			name: "22:00 at utc",
			args: args{
				t:  utcPM10,
				id: 1,
			},
			wantOrderNumber: "M21110800001",
		},
		{
			name: "10:00 at utc",
			args: args{
				t:  utcAM10,
				id: 1,
			},
			wantOrderNumber: "M21110700001",
		},
		{
			name: "10:00 at ast",
			args: args{
				t:  astAM10,
				id: 1,
			},
			wantOrderNumber: "M21110800001",
		},
		{
			name: "10:00 at ast",
			args: args{
				t:  astAM10,
				id: 100001,
			},
			wantOrderNumber: "M21110800001",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOrderNumber := createOrderNumberByTaipeiTZ(tt.args.t, tt.args.id)
			if gotOrderNumber != tt.wantOrderNumber {
				t.Errorf("createOrderNumberByTaipeiTZ() = %v, want %v", gotOrderNumber, tt.wantOrderNumber)
			}
		})
	}
}
