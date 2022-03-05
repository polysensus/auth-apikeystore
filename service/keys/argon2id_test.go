package keys

import (
	"reflect"
	"testing"
)

func TestNewAPIKey(t *testing.T) {
	type args struct {
		alg  string
		opts []APIKeyOption
	}
	tests := []struct {
		name               string
		args               args
		want               APIKey
		decodedDisplayName string
		wantClientID       string
		wantErr            bool
		wantGenErr         bool
	}{
		// TODO: Add test cases.
		{
			"minimal good", args{alg: "argon2id:3 64MB 32"},
			APIKey{
				alg: Alg{
					String:         "argon2id:3 64MB 32",
					ParamsArgon2ID: ParamsArgon2ID{Time: 3, Memory: 64, KeyLen: 32}},
			}, "", "", false, false,
		},
		{
			"long display name", args{
				alg:  "argon2id:3 64MB 32",
				opts: []APIKeyOption{WithDisplayName("0123456789123456longer than the allowed 16 chars")}},
			APIKey{
				alg: Alg{
					String:         "argon2id:3 64MB 32",
					ParamsArgon2ID: ParamsArgon2ID{Time: 3, Memory: 64, KeyLen: 32}},
				DisplayName: "0123456789123456longer than the allowed 16 chars",
			}, "0123456789123456", "", false, false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewAPIKey(tt.args.alg, tt.args.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAPIKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil && tt.wantErr {
				t.Fatalf("expted an erro from NewAPIKey but didn't get one")
			}

			apikey, err := got.Generate()
			key := got.Key

			if err == nil && tt.wantGenErr {
				t.Fatalf("expted an error from Generate but didn't get one")
			}

			if !(tt.wantErr && tt.wantGenErr) {
				// if we didn't expect any errors we should be able to recover the password
				ak, password, err := Decode(apikey)
				if err != nil {
					t.Fatalf("unexpted error decoding apikey %s: %v", apikey, err)
				}

				if tt.decodedDisplayName != "" {
					if tt.decodedDisplayName != ak.DisplayName {
						t.Fatalf("decoded display name incorrect got `%s', want `%s'", ak.DisplayName, tt.decodedDisplayName)
					}
				}
				if !ak.MatchPassword(password, key) {
					t.Errorf("failed to recover password")
				}
			}

			// if we don't want an err, check that the key is not empty. But as
			// it is random zero it out before the DeepEqual check. If there is
			// an err the key should always be empty
			if !tt.wantErr {
				// Dito clientID, if its not provided its random
				if tt.wantClientID == "" {
					if len(got.ClientID) == 0 {
						t.Errorf("NewAPIKey() = %v and has empty clientID, wanted %s", got, tt.wantClientID)
					}
					got.ClientID = ""
				}

			}

			if !tt.wantGenErr {
				if len(got.Salt) == 0 {
					t.Errorf("NewAPIKey() = %v and has empty salt", got)
				}
				got.Salt = nil
				if len(got.Key) == 0 {
					t.Errorf("NewAPIKey() = %v and has empty key", got)
				}
				got.Key = nil
			}

			// we always expect an empty (zero valued) APIKey on error from
			// NewAPIKey and generate does not mutate the state of the APIKey
			// object
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewAPIKey() = %v, want %v", got, tt.want)
			}
		})
	}
}
