package auth_interceptor

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/NBISweden/bp-dod-sda-gateway/pkg/observability"
	"github.com/jwx-go/jwkfetch/v4"
	"github.com/lestrrat-go/jwx/v4/jwk"
	"github.com/lestrrat-go/jwx/v4/jws"
	"github.com/lestrrat-go/jwx/v4/jwt"
)

type Authenticator struct {
	jwkKeySet     jwk.Set
	ketSetFetched time.Time
}
type AuthCtxInfo struct {
	Subject  string
	RawToken string
}

func NewAuthenticator() (*Authenticator, error) {
	a := &Authenticator{}

	jwkURL, err := url.ParseRequestURI(jwtPubKeyUrl)
	if err != nil || jwkURL.Scheme == "" || jwkURL.Host == "" {
		if err != nil {
			return nil, err
		}

		return nil, fmt.Errorf("jwtPubKeyUrl is not a proper URL (%s)", jwkURL)
	}

	// validate that we can fetch key set from url on start up
	if _, err := a.getJwkKeySet(context.Background()); err != nil {
		return nil, err
	}

	return a, nil
}

func (a *Authenticator) getJwkKeySet(ctx context.Context) (jwk.Set, error) {
	if a.ketSetFetched.Add(jwtKeySetCacheTTL).After(time.Now()) {
		return a.jwkKeySet, nil
	}

	client := jwkfetch.NewClient()
	set, err := client.Fetch(ctx, jwtPubKeyUrl)
	if err != nil {
		if a.jwkKeySet != nil {
			slog.Warn("failed to fetch jwt key set, keeping old key", slog.Any("error", err))

			return a.jwkKeySet, nil
		}

		return nil, fmt.Errorf("failed to fetch jwt key set: %w", err)
	}
	it := set.All()
	for _, v := range it {
		keyID, ok := v.KeyID()
		if !ok || keyID == "" {
			if err := jwk.AssignKeyID(v); err != nil {
				if a.jwkKeySet != nil {
					slog.Warn("failed to assign key id, keeping old key", slog.Any("error", err))

					return a.jwkKeySet, nil
				}

				return nil, fmt.Errorf("AssignKeyID failed: %v", err)
			}
		}
	}

	a.ketSetFetched = time.Now()
	a.jwkKeySet = set

	return a.jwkKeySet, nil
}

func (a *Authenticator) Authenticate(ctx context.Context, req *http.Request) (any, error) {
	ctx, span := observability.StartSpan(ctx, "Authenticate")
	defer span.End()

	if a == nil {
		span.Error("error validating token keyset", errors.New("authenticator not initialized"))

		return nil, connect.NewError(connect.CodeInternal, nil)
	}

	authStr := req.Header.Get("Authorization")
	if authStr == "" {
		span.Info("no Authorization header present")

		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}

	headerParts := strings.Split(authStr, " ")
	if headerParts[0] != "Bearer" {
		span.Info("unsupported authorization scheme")

		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}
	if len(headerParts) != 2 || headerParts[1] == "" {
		span.Info("token string is missing from authorization header")

		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}

	tokenStr := headerParts[1]

	keySet, err := a.getJwkKeySet(ctx)
	if err != nil {
		span.Error("failed to get jwk ket set", err)

		return nil, connect.NewError(connect.CodeInternal, nil)
	}

	token, err := jwt.Parse([]byte(tokenStr), jwt.WithKeySet(keySet, jws.WithInferAlgorithmFromKey(true)), jwt.WithValidate(true))
	if err != nil {
		span.Info("failed to parse token", slog.Any("error", err))

		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}

	tokenAudiences, ok := token.Audience()
	if !ok {
		span.Info("token has no audience")

		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}

	validAud := false
	for _, tokenAud := range tokenAudiences {
		if tokenAud == allowedAudience {
			validAud = true

			break
		}
	}

	if !validAud {
		span.Info("token audience not valid",
			slog.String("allowed-audience", allowedAudience),
			slog.String("token-audience", strings.Join(tokenAudiences, ",")),
		)

		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}

	subject, ok := token.Subject()
	if !ok {
		span.Info("token has no subject")

		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}

	return AuthCtxInfo{
		Subject:  subject,
		RawToken: tokenStr,
	}, nil
}
