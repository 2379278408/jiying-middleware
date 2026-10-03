package gin

import (
	"github.com/gin-gonic/gin"
	"jiying/middleware/core"
)

func init() {
	initLogin()
	initUsers()
	initPublic()
	initWebMenu()
	initSystem()
	initMisc()
	initMisc2()
	initCloudJS()
	initPluginsAPI()
	initScriptsAPI()
}

func initLogin() {
	core.Server.POST("/login", PostLogin)
	core.Server.GET("/login", GetLogin)
	core.Server.GET("/icp", GetICPP)
	core.Server.POST("/sendCode2", SendCode2)
	core.Server.POST("/forgetPassword", ForgetPassword)
	core.Server.POST("/logout", PostLogout)
}

func initUsers() {
	au := core.Server.Group("/users")
	au.Use(AuthRequired(), requireConsoleMaster())
	au.GET("", ListUsers)
	au.POST("/save", SaveUser)
	au.POST("/delete", DeleteUser)
	au.POST("/toggle", ToggleUser)
}

func initPublic() {
	core.Server.GET("/register", func(c *gin.Context) {
		jsonOK(c, gin.H{"open": registerOpen()})
	})
}
