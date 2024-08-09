package api

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"io"
	"net/http"
	"pickup/internal/auth"
	"pickup/internal/global"
)

var (
	googleOauthConfig *oauth2.Config
)

type GoogleUserInfo struct {
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
}

func GetLoginPage(c *gin.Context) {
	errorMsg := c.Query("error")
	c.HTML(http.StatusOK, "login.html", gin.H{
		"error": errorMsg,
	})
}

func getUserInfo(ctx context.Context, token *oauth2.Token) (*GoogleUserInfo, error) {
	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))
	response, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, fmt.Errorf("failed getting user info: %s", err.Error())
	}
	defer response.Body.Close()

	contents, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed reading response body: %s", err.Error())
	}

	var userInfo GoogleUserInfo
	err = json.Unmarshal(contents, &userInfo)
	if err != nil {
		return nil, fmt.Errorf("failed parsing user info: %s", err.Error())
	}

	return &userInfo, nil
}

func RedirectToGoogleAuth(c *gin.Context) {
	endpoint := global.Dv.GetString("ENDPOINT")
	httpType := global.Dv.GetString("HTTP_TYPE")

	googleOauthConfig = &oauth2.Config{
		RedirectURL:  fmt.Sprintf("%s://%s/v1/auth/google/callback", httpType, endpoint),
		ClientID:     global.Gv.GetString("web.client_id"),
		ClientSecret: global.Gv.GetString("web.client_secret"),
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email"},
		Endpoint:     google.Endpoint,
	}

	url := googleOauthConfig.AuthCodeURL("state")

	c.Redirect(http.StatusTemporaryRedirect, url)
}

func GoogleAuthCallback(c *gin.Context) {
	token, err := googleOauthConfig.Exchange(c, c.Query("code"))
	if err != nil {
		c.Redirect(http.StatusFound, "/v1/auth/login?error=Failed to exchange google access-token")
		return
	}

	userinfo, err := getUserInfo(c, token)
	if err != nil {
		c.Redirect(http.StatusFound, "/v1/auth/login?error=Failed to get user info")
		return
	}

	// for response security
	hashedEmail := hashEmailTo8Chars(userinfo.Email)
	jwt, err := auth.GenerateJWT(hashedEmail, global.Dv.GetInt("JWT_EXPIRES_MIN"))
	if err != nil {
		c.Redirect(http.StatusFound, "/v1/auth/login?error=Failed to generate token")
	}

	// register token to avoid using websocket without Google authentication.
	global.UserJWTMap.Store(hashedEmail, jwt)

	// set jwt
	c.SetCookie("jwt", jwt, 3600, "/", global.Dv.GetString("DOMAIN"), global.Dv.GetBool("COOKIE_SECURE"), true)

	// redirect
	c.Redirect(http.StatusFound, fmt.Sprintf("/v1/game/room"))
}

func hashEmailTo8Chars(email string) string {
	hasher := md5.New()
	hasher.Write([]byte(email))
	fullHash := hex.EncodeToString(hasher.Sum(nil))
	return fullHash[:8]
}
