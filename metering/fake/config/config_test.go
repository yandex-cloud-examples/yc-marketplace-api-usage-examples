package config

import (
	"reflect"
	"testing"
)

func TestNewConfig(t *testing.T) {
	type args struct {
		bindings string
		mode     string
	}
	tests := []struct {
		name    string
		args    args
		want    *Config
		wantErr bool
	}{
		{
			name: "empty",
			args: args{bindings: `[]`},
			want: &Config{
				SkuBindings: []SkuBinding{},
				Mode:        ALWAYS,
				endpoint:    "api.yc.local",
				Port:        8080,
			},
		},
		{
			name: "empty string",
			args: args{bindings: ""},
			want: &Config{
				SkuBindings: []SkuBinding{},
				Mode:        ALWAYS,
				endpoint:    "api.yc.local",
				Port:        8080,
			},
		},
		{
			name: "full",
			args: args{
				bindings: `[{"sku_id":"sku1","product_id":"product1"}]`,
				mode:     "always",
			},

			want: &Config{
				SkuBindings: []SkuBinding{
					{
						SkuID:     "sku1",
						ProductID: "product1",
					},
				},
				Mode:     ALWAYS,
				endpoint: "api.yc.local",
				Port:     8080,
			},
		},
		{
			name: "default mode",
			args: args{bindings: `[{"sku_id":"sku1","product_id":"product1"}]`},
			want: &Config{
				SkuBindings: []SkuBinding{{SkuID: "sku1", ProductID: "product1"}},
				Mode:        ALWAYS,
				endpoint:    "api.yc.local",
				Port:        8080,
			},
		},
		{
			name: "invalid mode",
			args: args{
				bindings: `[{"sku_id":"sku1","product_id":"product1"}]`,
				mode:     "invalid",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewConfig(tt.args.bindings, "8080", "", WorkMode(tt.args.mode))
			if (err != nil) != tt.wantErr {
				t.Errorf("NewConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewConfig() got = %v, want %v", got, tt.want)
			}
		})
	}
}
