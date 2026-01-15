package auth

import (
	"errors"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"net/http"

	"github.com/go-chi/render"
	"github.com/golang-jwt/jwt/v5"
	"github.com/golang-jwt/jwt/v5/request"
)

type Middleware struct {
	secret []byte
}

func NewMiddleware(secret string) *Middleware {
	return &Middleware{
		secret: []byte(secret),
	}
}

func (m *Middleware) Handler(next runtime.HandlerFunc) runtime.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		var claims accessTokenClaims
		token, err := request.ParseFromRequest(
			r,
			request.MultiExtractor{
				request.AuthorizationHeaderExtractor,
				request.ArgumentExtractor{"jwtToken"},
			},
			func(_ *jwt.Token) (_ interface{}, _ error) {
				return m.secret, nil
			},
			request.WithClaims(&claims),
		)
		if errors.Is(err, request.ErrNoTokenInRequest) {
			next(w, r, pathParams)
			return
		}
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			render.JSON(w, r, map[string]interface{}{"message": err.Error()})
			return
		}

		if !token.Valid {
			w.WriteHeader(http.StatusUnauthorized)
			render.JSON(w, r, map[string]interface{}{"message": "token is invalid"})
		}

		r = r.WithContext(contextWithUID(r.Context(), claims.UID))

		next(w, r, pathParams)
	}
}
