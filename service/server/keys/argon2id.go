package server

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	nanoid "github.com/matoous/go-nanoid"
	"golang.org/x/crypto/argon2"
)

const (
	saltLen       = 32
	passwordLen   = 32
	apiKeyNameLen = 16
	argon2Threads = 1

	// A note on why we go with a short clientid: We use the derived key as the
	// database primary key *not* the client id. We get the password handed to
	// us from the api key owner and we recover the derived key.  So its ok to
	// cut some corners with the id size, it only exists to satisfy the client
	// credentials requirements. Even if there is an id collision, the correct
	// entry will always be found (an uniquely so) using the key recovery.
	// PROVIDED that we always generate good SALTED random passwords.
	clientNanoIDLen = 16 // 1% in 5 million years

	apiKeyNumParts              = 5
	apiKeyDisplayNamePrefixPart = 0
	apiKeyIDPart                = 1
	apiKeyAlgPart               = 2
	apiKeySaltPart              = 3
	apiKeyPasswordPart          = 4
)

type APIKey struct {
	alg Alg
	// Salt is randomly generated when the password is generated. It is safe to (and must be) return to the api key holder
	Salt []byte
	// Key is derived from a randomly generated password. The key is
	// persistently stored. In the api key usage model this key is NOT
	// sensitive. But also is NOT returned to the user - instead, we return the
	// password and salt to the user. The password is NOT stored in this type
	// ever.
	Key []byte

	ClientID    string
	UserID      string
	DisplayName string
	Audience    string
	Scopes      string
}

type APIKeyOption func(*APIKey)

func WithClientID(clientID string) APIKeyOption {
	return func(ak *APIKey) {
		ak.ClientID = clientID
	}
}

func WithDisplayName(name string) APIKeyOption {
	return func(ak *APIKey) {
		ak.DisplayName = name
	}
}

func WithUserID(userID string) APIKeyOption {
	return func(ak *APIKey) {
		ak.UserID = userID
	}
}
func WithAudience(aud string) APIKeyOption {
	return func(ak *APIKey) {
		ak.Audience = aud
	}
}
func WithScopes(scopes string) APIKeyOption {
	return func(ak *APIKey) {
		ak.Scopes = scopes
	}
}

func NewAPIKey(alg string, opts ...APIKeyOption) (APIKey, error) {

	var err error

	ak := APIKey{}
	ak.alg, err = ParseAlg(alg)
	if err != nil {
		return APIKey{}, err
	}

	for _, o := range opts {
		o(&ak)
	}

	// If we didn't get an explicit client id, make one up
	if len(ak.ClientID) == 0 {
		ak.ClientID, err = nanoid.ID(clientNanoIDLen)
		if err != nil {
			return APIKey{}, nil
		}
	}
	return ak, nil
}

func Decode(apikey string) (APIKey, []byte, error) {

	b, err := base64.URLEncoding.DecodeString(apikey)
	if err != nil {
		return APIKey{}, nil, err
	}

	parts := strings.SplitN(string(b), ".", apiKeyNumParts+1)

	if len(parts) != apiKeyNumParts {

		return APIKey{}, nil, fmt.Errorf(
			"invalid number of '.' seperated parts api key. got %d, wanted %d", len(parts), apiKeyNumParts)
	}

	ak := APIKey{}

	ak.alg, err = ParseAlg(parts[apiKeyAlgPart])
	if err != nil {
		return APIKey{}, nil, err
	}

	ak.DisplayName = parts[apiKeyDisplayNamePrefixPart]
	ak.ClientID = parts[apiKeyIDPart]
	ak.Salt, err = base64.URLEncoding.DecodeString(parts[apiKeySaltPart])
	if err != nil {
		return APIKey{}, nil, err
	}
	password, err := base64.URLEncoding.DecodeString(parts[apiKeyPasswordPart])
	if err != nil {
		return APIKey{}, nil, err
	}

	return ak, password, nil
}

func (ak *APIKey) MatchPassword(password, key []byte) bool {

	ak.Key = argon2.IDKey(password, ak.Salt, ak.alg.Time, ak.alg.Memory, argon2Threads, ak.alg.KeyLen)

	return bytes.Equal(ak.Key, key)
}

func (ak *APIKey) generatePasword() ([]byte, error) {

	ak.Salt = make([]byte, saltLen)
	n, err := rand.Read(ak.Salt)
	if err != nil {
		return nil, err
	}
	if n != saltLen {
		return nil, fmt.Errorf("insufficient rand bytes generating salt")
	}

	password := make([]byte, passwordLen)
	n, err = rand.Read(password)
	if err != nil {
		return nil, err
	}
	if n != passwordLen {
		return nil, fmt.Errorf("insufficient rand bytes generating password")
	}

	ak.Key = argon2.IDKey(password, ak.Salt, ak.alg.Time, ak.alg.Memory, argon2Threads, ak.alg.KeyLen)

	return password, nil
}

func (ak *APIKey) Generate() (string, error) {
	password, err := ak.generatePasword()
	if err != nil {
		return "", err
	}
	salt := base64.URLEncoding.EncodeToString(ak.Salt)
	secret := base64.URLEncoding.EncodeToString(password)

	name := ak.DisplayName
	if len(name) > apiKeyNameLen {
		name = name[:apiKeyNameLen]
	}

	s := strings.Join([]string{name, ak.ClientID, ak.alg.String, salt, secret}, ".")
	return base64.URLEncoding.EncodeToString([]byte(s)), nil
}
