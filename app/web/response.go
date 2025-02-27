package web

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
	"github.com/spf13/viper"

	"github.com/1024casts/snake/pkg/errno"
	"github.com/1024casts/snake/pkg/flash"
	"github.com/1024casts/snake/pkg/log"
)

type Resp struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func Response(c *gin.Context, err error, data interface{}) {
	code, message := errno.DecodeErr(err)

	// always return http.StatusOK
	c.JSON(http.StatusOK, Resp{
		Code:    code,
		Message: message,
		Data:    data,
	})

	return
}

func Redirect(c *gin.Context, redirectPath, errMsg string) {
	flash.SetMessage(c.Writer, errMsg)
	c.Redirect(http.StatusMovedPermanently, redirectPath)
	c.Abort()
}

// It is recommended to use an authentication key with 32 or 64 bytes.
// The encryption key, if set, must be either 16, 24, or 32 bytes to select
// AES-128, AES-192, or AES-256 modes.
var Store = sessions.NewCookieStore([]byte(viper.GetString("cookie.secret")))

func SetLoginCookie(ctx *gin.Context, userId uint64) {
	Store.Options.HttpOnly = true

	session := GetCookieSession(ctx)
	session.Options = &sessions.Options{
		Domain:   viper.GetString("cookie.domain"),
		MaxAge:   86400,
		Path:     "/",
		HttpOnly: true,
	}
	// 浏览器关闭，cookie删除，否则保存30天(github.com/gorilla/sessions 包的默认值)
	val := ctx.DefaultPostForm("remember_me", "0")
	log.Infof("[handler] remember_me val: %s", val)
	if val == "1" {
		session.Options.MaxAge = 86400 * 30
	}

	session.Values["user_id"] = userId

	req := Request(ctx)
	resp := ResponseWriter(ctx)
	err := session.Save(req, resp)
	if err != nil {
		log.Warnf("[handler] set login cookie, %v", err)
	}
}

func GetCookieSession(ctx *gin.Context) *sessions.Session {
	session, err := Store.Get(ctx.Request, viper.GetString("cookie.name"))
	if err != nil {
		log.Warnf("[handler] store get err, %v", err)
	}
	return session
}

func Request(ctx *gin.Context) *http.Request {
	return ctx.Request
}

func ResponseWriter(ctx *gin.Context) http.ResponseWriter {
	return ctx.Writer
}
