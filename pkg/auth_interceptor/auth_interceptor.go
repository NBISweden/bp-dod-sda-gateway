package auth_interceptor

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/NBISweden/bp-dod-sda-gateway/pkg/observability"
	"github.com/jwx-go/jwkfetch/v4"
	"github.com/lestrrat-go/jwx/v4/jwk"
	"github.com/lestrrat-go/jwx/v4/jws"
	"github.com/lestrrat-go/jwx/v4/jwt"
)

type Authenticator struct {
	jwtPubKeyURL string
}
type AuthCtxInfo struct {
	Subject  string
	RawToken string
}

func NewAuthenticator(jwtPubKeyURL string) (*Authenticator, error) {
	a := &Authenticator{
		jwtPubKeyURL: jwtPubKeyURL,
	}

	if jwtPubKeyURL == "" {
		return nil, errors.New("jwtPubKeyURL is empty")
	}

	jwkURL, err := url.ParseRequestURI(a.jwtPubKeyURL)
	if err != nil || jwkURL.Scheme == "" || jwkURL.Host == "" {
		if err != nil {
			return nil, err
		}

		return nil, fmt.Errorf("jwtPubKeyURL is not a proper URL (%s)", jwkURL)
	}

	// validate that we can fetch key set from url on start up
	if _, err := a.fetchJwtPubKeys(context.Background()); err != nil {
		return nil, err
	}

	return a, nil
}

func (a *Authenticator) fetchJwtPubKeys(ctx context.Context) (jwk.Set, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	client := jwkfetch.NewClient()
	set, err := client.Fetch(ctx, a.jwtPubKeyURL)
	if err != nil {
		return nil, fmt.Errorf("%s", err.Error())
	}
	it := set.All()
	for _, v := range it {
		keyID, ok := v.KeyID()
		if !ok || keyID == "" {
			if err := jwk.AssignKeyID(v); err != nil {
				return nil, fmt.Errorf("AssignKeyID failed: %v", err)
			}
		}
	}

	return set, nil
}

func (a *Authenticator) Authenticate(ctx context.Context, req *http.Request) (any, error) {
	ctx, span := observability.StartSpan(ctx, "Authenticate")
	defer span.End()

	if a == nil {
		return nil, errors.New("error validating token keyset")
	}

	authStr := req.Header.Get("Authorization")
	if authStr == "" {
		return nil, errors.New("no Authorization header present")
	}

	headerParts := strings.Split(authStr, " ")
	if headerParts[0] != "Bearer" {
		return "", errors.New("authorization scheme must be bearer")
	}
	if len(headerParts) != 2 || headerParts[1] == "" {
		return "", errors.New("token string is missing from authorization header")
	}

	tokenStr := headerParts[1]

	keySet, err := a.fetchJwtPubKeys(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch key set: %w", err)
	}

	token, err := jwt.Parse([]byte(tokenStr), jwt.WithKeySet(keySet, jws.WithInferAlgorithmFromKey(true)), jwt.WithValidate(true))
	if err != nil {
		return nil, err
	}

	tokenIssuer, ok := token.Issuer()
	if !ok {
		return nil, errors.New("token issuer not valid")
	}
	iss, err := url.ParseRequestURI(tokenIssuer)
	if err != nil || iss.Hostname() == "" {
		return nil, fmt.Errorf("failed to get issuer from token (%v)", iss)
	}
	// TODO validate aud, iss, etc

	subject, ok := token.Subject()
	if !ok {
		return nil, errors.New("token subject not valid")
	}

	return AuthCtxInfo{
		Subject:  subject,
		RawToken: tokenStr,
	}, nil
}
