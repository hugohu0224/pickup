package api

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"pickup/internal/auth"
	"pickup/internal/global"
)

var googleOauthConfig *oauth2.Config

type GoogleUserInfo struct {
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
}

func init() {
	endpoint := global.Dv.GetString("ENDPOINT")
	httpType := global.Dv.GetString("HTTP_TYPE")

	googleOauthConfig = &oauth2.Config{
		RedirectURL:  fmt.Sprintf("%s://%s/v1/auth/google/callback", httpType, endpoint),
		ClientID:     global.Gv.GetString("web.client_id"),
		ClientSecret: global.Gv.GetString("web.client_secret"),
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email"},
		Endpoint:     google.Endpoint,
	}
}

func GetLoginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", gin.H{
		"error": c.Query("error"),
	})
}

func getUserInfo(ctx context.Context, token *oauth2.Token) (*GoogleUserInfo, error) {
	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))
	response, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, fmt.Errorf("failed getting user info: %w", err)
	}
	defer response.Body.Close()

	var userInfo GoogleUserInfo
	if err := json.NewDecoder(response.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed parsing user info: %w", err)
	}

	return &userInfo, nil
}

func RedirectToGoogleAuth(c *gin.Context) {
	url := googleOauthConfig.AuthCodeURL("state")
	c.Redirect(http.StatusTemporaryRedirect, url)
}

func GoogleAuthCallback(c *gin.Context) {
	token, err := googleOauthConfig.Exchange(c, c.Query("code"))
	if err != nil {
		redirectWithError(c, "Failed to exchange google access-token")
		return
	}

	userinfo, err := getUserInfo(c, token)
	if err != nil {
		redirectWithError(c, "Failed to get user info")
		return
	}

	hashedEmail := hashEmailTo8Chars(userinfo.Email)
	jwt, err := auth.GenerateJWT(hashedEmail, global.Dv.GetInt("JWT_EXPIRES_MIN"))
	if err != nil {
		redirectWithError(c, "Failed to generate token")
		return
	}

	global.UserJWTMap.Store(hashedEmail, jwt)

	c.SetCookie("jwt", jwt, 3600, "/", global.Dv.GetString("DOMAIN"), global.Dv.GetBool("COOKIE_SECURE"), true)
	c.Redirect(http.StatusFound, "/v1/game/room")
}

func redirectWithError(c *gin.Context, message string) {
	c.Redirect(http.StatusFound, fmt.Sprintf("/v1/auth/login?error=%s", message))
}

func hashEmailTo8Chars(email string) string {
	hasher := md5.New()
	hasher.Write([]byte(email))
	return hex.EncodeToString(hasher.Sum(nil))[:8]
}
