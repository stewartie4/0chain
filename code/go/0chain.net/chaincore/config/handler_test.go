package config

import (
	"fmt"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func init() {
	SetupHandlers()
}

func TestGetConfigHandler(t *testing.T) {
	w := httptest.NewRecorder()
	w.Header().Set("Content-Type", "text/plain;charset=UTF-8")
	c := viper.AllSettings()
	bs, err := yaml.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fmt.Fprintf(w, "%v", string(bs)); err != nil {
		t.Fatal(err)
	}

	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name string
		args args
		want http.ResponseWriter
	}{
		{
			name: "OK",
			args: args{w: httptest.NewRecorder()},
			want: w,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			GetConfigHandler(tt.args.w, tt.args.r)
			if !reflect.DeepEqual(tt.args.w, tt.want) {
				t.Errorf("GetConfigHandler() got = %v, want = %v", tt.args.w, tt.want)
			}
		})
	}
}
