package core

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

// 控制台角色，权限由高到低：主控 > 运维 > 只读。
const (
	RoleMaster   = "master"
	RoleOps      = "ops"
	RoleReadonly = "readonly"
)

const rolesKey = "jiying_roles"

// consoleAdminID 是控制台引导管理员账号 ID（由 develop/users.ensureAdmin 创建）。
const consoleAdminID = "admin"

var roleRank = map[string]int{RoleMaster: 3, RoleOps: 2, RoleReadonly: 1}

var rolesMu sync.Mutex

// loadRoles 读取显式角色表；不在表中的用户默认只读。
func loadRoles() map[string]string {
	out := map[string]string{}
	if BucketApp == nil {
		return out
	}
	raw := BucketApp.GetString(rolesKey)
	if raw == "" {
		return out
	}
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}

func saveRoles(roles map[string]string) {
	if BucketApp == nil {
		return
	}
	raw, _ := json.Marshal(roles)
	BucketApp.Set(rolesKey, string(raw))
}

// IsConsoleAdmin 判断是否控制台引导管理员；该账号始终是主控。
func IsConsoleAdmin(userID string) bool {
	return strings.TrimSpace(userID) == consoleAdminID
}

// privilegedUser 报告用户是否具备主控级别权限：系统内调用、主控列表成员或控制台引导管理员。
// 命令分发与功能授权共用这一判定，避免两条路径出现权限口径差异。
func privilegedUser(userID string) bool {
	return userID == "system" || IsMaster(userID) || IsConsoleAdmin(userID)
}

// UserRole 返回用户角色：主控列表中的用户与控制台引导管理员始终是主控。
func UserRole(userID string) string {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return ""
	}
	if IsMaster(userID) || IsConsoleAdmin(userID) {
		return RoleMaster
	}
	rolesMu.Lock()
	defer rolesMu.Unlock()
	role := loadRoles()[userID]
	if _, ok := roleRank[role]; !ok {
		return RoleReadonly
	}
	return role
}

// HasRole 判断用户是否达到指定角色等级。
func HasRole(userID, min string) bool {
	need, ok := roleRank[min]
	if !ok {
		return false
	}
	got, ok := roleRank[UserRole(userID)]
	if !ok {
		return false
	}
	return got >= need
}

// SetUserRole 设置用户角色；主控角色由 masters 列表控制，不在此维护。
func SetUserRole(userID, role string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return fmt.Errorf("缺少用户")
	}
	if role != RoleOps && role != RoleReadonly {
		return fmt.Errorf("角色只能是 ops 或 readonly")
	}
	rolesMu.Lock()
	defer rolesMu.Unlock()
	roles := loadRoles()
	roles[userID] = role
	saveRoles(roles)
	return nil
}

// DeleteUserRole 移除显式角色，用户回落到默认只读。
func DeleteUserRole(userID string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return fmt.Errorf("缺少用户")
	}
	rolesMu.Lock()
	defer rolesMu.Unlock()
	roles := loadRoles()
	if _, ok := roles[userID]; !ok {
		return fmt.Errorf("角色不存在")
	}
	delete(roles, userID)
	saveRoles(roles)
	return nil
}

// ListUserRoles 以 userID->role 返回显式角色表。
func ListUserRoles() map[string]string {
	rolesMu.Lock()
	defer rolesMu.Unlock()
	return loadRoles()
}

// requireRole 返回一个校验控制台会话或 API Key 权限等级的中间件。
func requireRole(min string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if v, ok := c.Get(ctxAPIKeyScope); ok {
			scope, _ := v.(string)
			need, needOK := roleRank[min]
			got, gotOK := roleRank[scope]
			if !needOK || !gotOK || got < need {
				c.AbortWithStatusJSON(403, gin.H{"code": 403, "message": "权限不足"})
				return
			}
			c.Next()
			return
		}
		user := consoleUser(c)
		if !HasRole(user, min) {
			c.AbortWithStatusJSON(403, gin.H{"code": 403, "message": "权限不足"})
			return
		}
		c.Next()
	}
}

func registerRolesAPI() {
	g := Server.Group("/jiying/api", requireLogin)
	g.GET("/roles", func(c *gin.Context) {
		jiyingJSON(c, gin.H{
			"me":      UserRole(consoleUser(c)),
			"me_id":   consoleUser(c),
			"roles":   ListUserRoles(),
			"masters": ListMasters(),
		})
	})
	g.POST("/masters", requireRole(RoleMaster), func(c *gin.Context) {
		user := strings.TrimSpace(c.PostForm("user"))
		if err := AddMaster(user); err != nil {
			jiyingErr(c, err.Error())
			return
		}
		RecordAudit(consoleUser(c), "master.add", user, "", "成功")
		jiyingJSON(c, "ok")
	})
	g.POST("/masters/delete", requireRole(RoleMaster), func(c *gin.Context) {
		user := strings.TrimSpace(c.PostForm("user"))
		if err := RemoveMaster(user); err != nil {
			jiyingErr(c, err.Error())
			return
		}
		RecordAudit(consoleUser(c), "master.delete", user, "", "成功")
		jiyingJSON(c, "ok")
	})
	g.POST("/roles", requireRole(RoleMaster), func(c *gin.Context) {
		user := strings.TrimSpace(c.PostForm("user"))
		role := strings.TrimSpace(c.PostForm("role"))
		if err := SetUserRole(user, role); err != nil {
			RecordAudit(consoleUser(c), "role.set", user, role, "失败: "+err.Error())
			jiyingErr(c, err.Error())
			return
		}
		RecordAudit(consoleUser(c), "role.set", user, role, "成功")
		jiyingJSON(c, "ok")
	})
	g.POST("/roles/delete", requireRole(RoleMaster), func(c *gin.Context) {
		user := strings.TrimSpace(c.PostForm("user"))
		if err := DeleteUserRole(user); err != nil {
			RecordAudit(consoleUser(c), "role.delete", user, "", "失败: "+err.Error())
			jiyingErr(c, err.Error())
			return
		}
		RecordAudit(consoleUser(c), "role.delete", user, "", "成功")
		jiyingJSON(c, "ok")
	})
}

// requestRole 返回当前请求的有效角色：API Key 请求以 Key 作用域为准，会话请求按用户角色。
func requestRole(c *gin.Context) string {
	if v, ok := c.Get(ctxAPIKeyScope); ok {
		if scope, _ := v.(string); scope != "" {
			return scope
		}
	}
	return UserRole(consoleUser(c))
}

// requestHasRole 判断当前请求是否达到指定角色；用于同一接口内的可选高风险分支。
func requestHasRole(c *gin.Context, min string) bool {
	need, ok := roleRank[min]
	if !ok {
		return false
	}
	got, ok := roleRank[requestRole(c)]
	return ok && got >= need
}
