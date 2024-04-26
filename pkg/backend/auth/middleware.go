package backend

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"go.savla.dev/littr/models"

	"github.com/golang-jwt/jwt"
)

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//ctx := r.Context()

		/*perm, ok := ctx.Value("acl.permission").(YourPermissionType)
		  		if !ok || !perm.IsAdmin() {
		    			http.Error(w, http.StatusText(403), 403)
		    			return
		  		}*/

		// skip those routes
		if r.URL.Path == "/api" ||
			r.URL.Path == "/api/auth" ||
			r.URL.Path == "/api/dump" ||
			r.URL.Path == "/api/flow/live" ||
			r.URL.Path == "/api/auth/password" ||
			(r.URL.Path == "/api/users" && r.Method == "POST") {
			next.ServeHTTP(w, r)
			return
		}

		resp := response{}
		l := NewLogger(r, "auth")
		resp.AuthGranted = false

		secret := os.Getenv("APP_PEPPER")

		var accessCookie *http.Cookie
		var refreshCookie *http.Cookie
		var user models.User
		var err error

		if refreshCookie, err = r.Cookie("refresh-token"); err != nil {
			// logout --- missing refresh token
			resp.Message = "client unauthorized"
			resp.Code = http.StatusUnauthorized

			l.Println(resp.Message, resp.Code)
			resp.Write(w)
			return
		}

		// decode the contents of refreshCookie
		refreshClaims := ParseRefreshToken(refreshCookie.Value, secret)

		// refresh token is expired => user should relogin
		if refreshClaims.Valid() != nil {
			resp.Message = "refresh token expired"
			resp.Code = http.StatusUnauthorized

			l.Println(resp.Message, resp.Code)
			resp.Write(w)
			return
		}

		var userClaims *UserClaims
		accessCookie, err = r.Cookie("access-token")
		if err == nil {
			userClaims = ParseAccessToken(accessCookie.Value, secret)
		}

		// access cookie not present or access token is expired
		if err != nil || (userClaims != nil && userClaims.StandardClaims.Valid() != nil) {
			refreshSum := sha256.New()
			refreshSum.Write([]byte(refreshCookie.Value))
			sum := fmt.Sprintf("%x", refreshSum.Sum(nil))

			rawNick, found := TokenCache.Get(sum)
			if !found {
				voidCookie := &http.Cookie{
					Name:     "refresh-token",
					Value:    "",
					Expires:  time.Now().Add(time.Second * 1),
					Path:     "/",
					HttpOnly: true,
				}

				http.SetCookie(w, voidCookie)

				resp.Message = "the refresh token has been invalidated"
				resp.Code = http.StatusUnauthorized

				l.Println(resp.Message, resp.Code)
				resp.Write(w)
				return
			}

			nickname, ok := rawNick.(string)
			if !ok {
				resp.Message = "cannot assert data type for nickname"
				resp.Code = http.StatusInternalServerError

				l.Println(resp.Message, resp.Code)
				resp.Write(w)
				return
			}

			// invalidate refresh token on non-existing user referenced
			user, ok = getOne(UserCache, nickname, models.User{})
			if !ok {
				deleteOne(TokenCache, sum)

				voidCookie := &http.Cookie{
					Name:     "refresh-token",
					Value:    "",
					Expires:  time.Now().Add(time.Second * 1),
					Path:     "/",
					HttpOnly: true,
				}

				http.SetCookie(w, voidCookie)

				resp.Message = "referenced user not found"
				resp.Code = http.StatusUnauthorized

				l.Println(resp.Message, resp.Code)
				resp.Write(w)
				return
			}

			userClaims := UserClaims{
				Nickname: nickname,
				User:     user,
				StandardClaims: jwt.StandardClaims{
					IssuedAt:  time.Now().Unix(),
					ExpiresAt: time.Now().Add(time.Minute * 15).Unix(),
				},
			}

			// issue a new access token using refresh token's validity
			accessToken, err := NewAccessToken(userClaims, secret)
			if err != nil {
				resp.Message = "error creating the access token: " + err.Error()
				resp.Code = http.StatusInternalServerError

				l.Println(resp.Message, resp.Code)
				resp.Write(w)
				return
			}

			accessCookie := &http.Cookie{
				Name:     "access-token",
				Value:    accessToken,
				Expires:  time.Now().Add(time.Minute * 15),
				Path:     "/",
				HttpOnly: true,
			}

			http.SetCookie(w, accessCookie)

			resp.Users = make(map[string]models.User)
			resp.Users[nickname] = user

			/*resp.Message = "ok, new access token issued"
			resp.Code = http.StatusOK

			l.Println(resp.Message, resp.Code)
			resp.Write(w)
			return*/
		}

		/*resp.Users = make(map[string]models.User)
		resp.Users[user.Nickname] = user

		resp.Message = "auth granted"
		resp.Code = http.StatusOK

		l.Println(resp.Message, resp.Code)
		resp.Write(w)
		return*/

		refreshSum := sha256.New()
		refreshSum.Write([]byte(refreshCookie.Value))
		sum := fmt.Sprintf("%x", refreshSum.Sum(nil))

		rawNick, found := TokenCache.Get(sum)
		if !found {
			voidCookie := &http.Cookie{
				Name:     "refresh-token",
				Value:    "",
				Expires:  time.Now().Add(time.Second * 1),
				Path:     "/",
				HttpOnly: true,
			}

			http.SetCookie(w, voidCookie)

			resp.Message = "the refresh token has been invalidated"
			resp.Code = http.StatusUnauthorized

			l.Println(resp.Message, resp.Code)
			resp.Write(w)
			return
		}

		nickname, ok := rawNick.(string)
		if !ok {
			resp.Message = "cannot assert data type for nickname"
			resp.Code = http.StatusInternalServerError

			l.Println(resp.Message, resp.Code)
			resp.Write(w)
			return
		}

		ctx := context.WithValue(r.Context(), "nickname", nickname)
		noteUsersActivity(nickname)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}