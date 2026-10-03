package gin

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"jiying/middleware/core"
	"jiying/middleware/develop/users"
)

func ListUsers(c *gin.Context) {
	search := strings.TrimSpace(c.Query("searchValue"))
	if search == "" {
		search = strings.TrimSpace(c.Query("search"))
	}
	all := users.GetAll()
	data := make([]gin.H, 0, len(all))
	for _, u := range all {
		if search != "" && !strings.Contains(u.ID, search) && !strings.Contains(u.NickName, search) {
			continue
		}
		data = append(data, gin.H{
			"name":     u.NickName,
			"id":       u.ID,
			"username": u.ID,
			"nickname": u.NickName,
			"email":    u.Email,
			"disable":  u.Disable,
			"point":    u.Point,
			"charm":    u.Charm,
			"code":     u.Code,
			"reg_date": TimeFmt(u.RegDate),
		})
	}
	jsonOK(c, data)
}

func SaveUser(c *gin.Context) {
	username := strings.TrimSpace(c.PostForm("username"))
	if username == "" {
		jsonErr(c, "用户名不能为空")
		return
	}
	password := c.PostForm("password")
	u := users.GetUser(username)
	created := false
	if u == nil {
		u = users.NewUser()
		created = true
	}
	if !created && password != "" {
		if err := u.SetPassword(password); err != nil {
			jsonErr(c, err.Error())
			return
		}
	}
	if u.NickName == "" || c.PostForm("nickname") != "" {
		u.NickName = c.PostForm("nickname")
	}
	if u.RegDate.IsZero() {
		u.RegDate = time.Now()
	}
	if c.PostForm("email") != "" {
		u.Email = c.PostForm("email")
	}
	if v := c.PostForm("point"); v != "" {
		var p int64
		if _, err := fmtSscan(v, &p); err == nil {
			u.Point = p
		}
	}
	if c.PostForm("disable") == "true" || c.PostForm("disable") == "1" {
		u.Disable = true
	}
	if created {
		if err := u.Register(username, password, u.Email); err != nil {
			jsonErr(c, err.Error())
			return
		}
	} else if err := u.Save(); err != nil {
		jsonErr(c, err.Error())
		return
	}
	core.GetMyLog().Info("用户 %s 已保存", username)
	jsonOK(c, nil)
}

func DeleteUser(c *gin.Context) {
	username := strings.TrimSpace(c.PostForm("username"))
	if username == "" {
		jsonErr(c, "缺少用户名")
		return
	}
	u := users.GetUser(username)
	if u == nil {
		jsonOK(c, nil)
		return
	}
	if err := users.DeleteUser(username); err != nil {
		jsonErr(c, err.Error())
		return
	}
	dropUserFromMemory(username)
	core.GetMyLog().Info("用户 %s 已删除", username)
	jsonOK(c, nil)
}

func dropUserFromMemory(username string) {
	for i, u := range users.GetAll() {
		if u.ID == username {
			users.SetUsers(append(users.GetAll()[:i:i], users.GetAll()[i+1:]...))
			return
		}
	}
}

func ToggleUser(c *gin.Context) {
	username := strings.TrimSpace(c.PostForm("username"))
	u := users.GetUser(username)
	if u == nil {
		jsonErr(c, "用户不存在")
		return
	}
	u.Disable = c.PostForm("disable") == "true" || c.PostForm("disable") == "1"
	_ = u.Save()
	jsonOK(c, gin.H{"disable": u.Disable})
}

func TimeFmt(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}

func fmtSscan(v string, p *int64) (int, error) {
	n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
	if err != nil {
		return 0, err
	}
	*p = n
	return 1, nil
}

var _ = http.StatusOK
