package node

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestPushToPullHandler(t *testing.T) {
	pcde := "pcde"
	puri := "puri"
	id := "id"

	if err := pushDataCache.Add(puri+":"+id, pcde); err != nil {
		t.Fatal(err)
	}

	type args struct {
		ctx context.Context
		r   *http.Request
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		{
			name: "ERR",
			args: args{
				r: httptest.NewRequest(http.MethodGet, "/", nil),
			},
			wantErr: true,
		},
		{
			name: "OK",
			args: args{
				r: func() *http.Request {
					r := httptest.NewRequest(http.MethodGet, "/", nil)
					r.Form = map[string][]string{
						"_puri": {
							puri,
						},
						"id": {
							id,
						},
					}

					return r
				}(),
			},
			want: pcde,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PushToPullHandler(tt.args.ctx, tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("PushToPullHandler() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PushToPullHandler() got = %v, want %v", got, tt.want)
			}
		})
	}
}
