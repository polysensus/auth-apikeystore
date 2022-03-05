package keys

import (
	"reflect"
	"testing"
)

func TestParseAlg(t *testing.T) {
	type args struct {
		alg string
	}
	tests := []struct {
		name    string
		args    args
		want    Alg
		wantErr bool
	}{
		// TODO: Add test cases.
		{"happy standard", args{alg: "argon2id:3 64MB 32"}, Alg{ParamsArgon2ID: ParamsArgon2ID{3, 64, 32}}, false},
		{"happy small and fast", args{alg: "argon2id:1 16MB 16"}, Alg{ParamsArgon2ID: ParamsArgon2ID{1, 16, 16}}, false},
		{"missing alg", args{alg: "3 64MB 32"}, Alg{}, true},
		{"bad alg", args{alg: "argon2id3 64MB 32"}, Alg{}, true},
		{"missing part", args{alg: "argon2id:64M 32"}, Alg{}, true},
		{"time to large", args{alg: "argon2id:6 64M 32"}, Alg{}, true},
		{"time to small", args{alg: "argon2id:0 64M 32"}, Alg{}, true},
		{"memory to large", args{alg: "argon2id:6 65M 32"}, Alg{}, true},
		{"memory to small", args{alg: "argon2id:3 15M 32"}, Alg{}, true},
		{"keylen to large", args{alg: "argon2id:3 64M 65"}, Alg{}, true},
		{"keylen to small", args{alg: "argon2id:3 64M 15"}, Alg{}, true},
		{"bad memory suffix", args{alg: "argon2id:3 64M 32"}, Alg{}, true},
		{"bad memory suffix", args{alg: "argon2id:3 64MX 32"}, Alg{}, true},
		{"bad memory suffix", args{alg: "argon2id:3 64 32"}, Alg{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseAlg(tt.args.alg)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseAlg() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseAlg() = %v, want %v", got, tt.want)
			}
		})
	}
}
