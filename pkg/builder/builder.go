package builder

import (
	"os"

	"smscp.xyz/internal/api"
	"smscp.xyz/internal/db"
	"smscp.xyz/internal/security"
	"smscp.xyz/internal/sms/twilio"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

var databaseConn, databaseErr = db.DBConnDefault()

func Build() (*gin.Engine, error) {
	if databaseErr != nil {
		return nil, databaseErr
	}

	router := gin.Default()
	router.LoadHTMLGlob("web/html/*")
	store := cookie.NewStore([]byte(os.Getenv("SESSION_SECRET")))
	router.Use(sessions.Sessions("lasso_sessions", store))

	sms := twilio.SMSDefault(os.Getenv("TWILIO_ID"), os.Getenv("TWILIO_SECRET"), os.Getenv("TWILIO_FROM"))
	security := security.SecurityDefault(os.Getenv("JWT_SECRET"))
	data := db.DBDefault(databaseConn, security)
	app := api.AppDefault(data, sms)

	router.GET("/", app.Page)
	router.POST("/", app.Page)

	router.POST("/user/login", app.UserLogin)
	router.POST("/user/create", app.UserCreate)
	router.POST("/user/update", app.UserUpdate)
	router.POST("/user/logout", app.UserLogout)

	router.POST("/note/create", app.NoteCreate)
	router.GET("/note/list/:page", app.NoteListJSON)

	router.POST("/cli/user/login", app.UserLoginCLI)
	router.POST("/cli/user/create", app.UserCreateCLI)
	router.POST("/cli/note/create", app.NoteCreateCLI)
	router.POST("/cli/note/latest", app.NoteLatestCLI)

	router.POST("/hook/sms/receive", app.HookSMS)

	return router, nil
}
