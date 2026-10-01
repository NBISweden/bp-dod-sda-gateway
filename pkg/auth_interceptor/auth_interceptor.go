package auth_interceptor

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

type Authenticator struct {
	Keyset jwk.Set
}
type Identity struct {
	Subject string
}

func NewAuthenticator(jwtPubKeyURL string) (*Authenticator, error) {
	a := new(Authenticator)

	if jwtPubKeyURL == "" {
		return nil, errors.New("jwtPubKeyURL is empty")
	}

	if err := a.fetchJwtPubKeyURL(jwtPubKeyURL); err != nil {
		return nil, err
	}

	return a, nil
}

func (a *Authenticator) fetchJwtPubKeyURL(jwtPubKeyURL string) error {
	jwkURL, err := url.ParseRequestURI(jwtPubKeyURL)
	if err != nil || jwkURL.Scheme == "" || jwkURL.Host == "" {
		if err != nil {
			return err
		}

		return fmt.Errorf("jwtPubKeyURL is not a proper URL (%s)", jwkURL)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	a.Keyset, err = jwk.Fetch(ctx, jwtPubKeyURL)
	if err != nil {
		return fmt.Errorf("jwk.Fetch failed (%v) for %s", err, jwtPubKeyURL)
	}

	for it := a.Keyset.Keys(ctx); it.Next(ctx); {
		pair := it.Pair()
		key := pair.Value.(jwk.Key)
		if err := jwk.AssignKeyID(key); err != nil {
			return fmt.Errorf("AssignKeyID failed: %v", err)
		}
	}

	return nil
}

func readTokenFromHeader(authStr string) (string, error) {
	headerParts := strings.Split(authStr, " ")
	if headerParts[0] != "Bearer" {
		return "", errors.New("authorization scheme must be bearer")
	}
	if len(headerParts) != 2 || headerParts[1] == "" {
		return "", errors.New("token string is missing from authorization header")
	}

	return headerParts[1], nil
}

func (a *Authenticator) Authenticate(_ context.Context, req *http.Request) (any, error) {
	// Verify signature by parsing the token with the given key
	if a == nil {
		return nil, errors.New("error validating token keyset")
	}
	switch {
	case req.Header.Get("X-Amz-Security-Token") != "":
		tokenStr := req.Header.Get("X-Amz-Security-Token")
		if tokenStr == "" {
			return nil, errors.New("no access token supplied")
		}
		token, err := jwt.Parse([]byte(tokenStr), jwt.WithKeySet(a.Keyset, jws.WithInferAlgorithmFromKey(true)), jwt.WithValidate(true))
		if err != nil {
			return nil, err
		}

		iss, err := url.ParseRequestURI(token.Issuer())
		if err != nil || iss.Hostname() == "" {
			return nil, fmt.Errorf("failed to get issuer from token (%v)", iss)
		}

		return token, nil

	case req.Header.Get("Authorization") != "":
		authStr := req.Header.Get("Authorization")
		tokenStr, err := readTokenFromHeader(authStr)
		if err != nil {
			return nil, fmt.Errorf("authorization header not valid: %w", err)
		}
		token, err := jwt.Parse([]byte(tokenStr), jwt.WithKeySet(a.Keyset, jws.WithInferAlgorithmFromKey(true)), jwt.WithValidate(true))
		if err != nil {
			return nil, fmt.Errorf("signed token not valid: %w", err)
		}

		iss, err := url.ParseRequestURI(token.Issuer())
		if err != nil || iss.Hostname() == "" {
			return nil, fmt.Errorf("failed to get issuer from token (%v)", iss)
		}

		return token, nil

	default:
		return nil, errors.New("no access token supplied")
	}
}
