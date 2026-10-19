package core

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"jiying/middleware/develop/a_boltdb"
	"jiying/middleware/utils"
)

type CronCmd struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Command      string   `json:"command"`
	Cron         string   `json:"cron"`
	Owner        string   `json:"owner"`
	Type         []string `json:"type"`
	Log          string   `json:"log"`
	Status       bool     `json:"status"`
	UseVersion   string   `json:"use_version"`
	CreateOn     string   `json:"create_on"`
	UpdateOn     string   `json:"update_on"`
	LastRun      string   `json:"last_run"`
	LastSuccess  string   `json:"last_success"`
	Start        string   `json:"start"`
	End          string   `json:"end"`
	Operator     string   `json:"operator"`
	Report       bool     `json:"report"`
	GroupManager []string `json:"group_manager"`
	UserManager  []string `json:"user_manager"`
	GroupIdWhite []string `json:"group_id_white"`
	GroupIdBlack []string `json:"group_id_black"`
	UserIdWhite  []string `json:"user_id_white"`
	UserIdBlack  []string `json:"user_id_black"`

	IsOnce          bool   `json:"is_once,omitempty"`
	LastRunningTime string `json:"last_running_time,omitempty"`
	LastStatus      string `json:"last_status,omitempty"`
	LastError       string `json:"last_error,omitempty"`
	NextRunAt       string `json:"next_run_at,omitempty"`
	OwnerRole       string `json:"owner_role,omitempty"`
	RunCount        int    `json:"run_count"`
	FailCount       int    `json:"fail_count"`
	CronEntityId    cron.EntryID
}

type cronCmdContainer struct {
	sync.RWMutex
	Bucket    *a_boltdb.Store
	MaxId     *utils.IDGenerator
	Position  *utils.PositionGenerator
	Pts       []*CronCmd
	cronSched *cron.Cron
}

var ccs = &cronCmdContainer{}

// cronSpecParser 同时接受 5 段（标准）与 6 段（含秒）表达式，
// 校验与挂载共用同一解析器，避免“校验通过但挂载失败”。
var cronSpecParser = cron.NewParser(
	cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
)

func initCronCmds() {
	bkt, err := a_boltdb.NewBucket("cronCmds", 8)
	if err != nil {
		GetMyLog().Warn("定时任务桶创建失败: %v", err)
		bkt = nil
	}
	ccs.Bucket = bkt
	ccs.MaxId = utils.NewIDGenerator(0)
	ccs.Position = utils.NewPositionGenerator()
	ccs.cronSched = cron.New(cron.WithParser(cronSpecParser))
	ccs.cronSched.Start()
	ccs.afterLoad()
	ccs.each(func(c *CronCmd) {
		ccs.schedule(c)
	})
	registerSysCommand("定时任务管理", "cron_manage", 5, func() {
		ccs.reportAll()
	})
	if ccs.cronSched != nil {
		_, _ = ccs.cronSched.AddFunc("0 */10 * * * *", ccs.cleanOnceTasks)
	}
}

func (c *cronCmdContainer) each(fn func(*CronCmd)) {
	c.RLock()
	defer c.RUnlock()
	for _, p := range c.Pts {
		if p != nil {
			fn(p)
		}
	}
}

func (c *cronCmdContainer) add(cmd *CronCmd) string {
	c.Lock()
	defer c.Unlock()
	if cmd.ID == "" {
		if c.MaxId == nil {
			c.MaxId = utils.NewIDGenerator(0)
		}
		cmd.ID = fmt.Sprintf("%d", c.MaxId.NewID())
	}
	if cmd.CreateOn == "" {
		cmd.CreateOn = time.Now().Format("2006-01-02 15:04:05")
	}
	c.Pts = append(c.Pts, cmd)
	c.save(cmd)
	if c.cronSched != nil {
		c.schedule(cmd)
	}
	return cmd.ID
}

func (c *cronCmdContainer) save(cmd *CronCmd) {
	if c.Bucket == nil || cmd == nil {
		return
	}
	c.Bucket.Set(cmd.ID, cmd)
}

func (c *cronCmdContainer) schedule(cmd *CronCmd) {
	if cmd == nil || !cmd.Status || cmd.Cron == "" {
		return
	}
	if c.cronSched == nil {
		return
	}
	spec := cmd.Cron
	if !isValidCronSpec(spec) {
		GetMyLog().Warn("定时任务 %s 表达式无效: %s", cmd.Name, spec)
		return
	}
	eid, err := c.cronSched.AddFunc(spec, func() {
		enqueueCron(cmd)
	})
	if err != nil {
		GetMyLog().Warn("定时任务 %s 挂载失败: %v", cmd.Name, err)
		return
	}
	cmd.CronEntityId = eid
}

// rescheduleLocked 移除旧调度并按当前状态重新挂载。调用方必须持有 ccs 写锁。
func (c *cronCmdContainer) rescheduleLocked(cmd *CronCmd) {
	if cmd == nil {
		return
	}
	if c.cronSched != nil && cmd.CronEntityId > 0 {
		c.cronSched.Remove(cmd.CronEntityId)
		cmd.CronEntityId = 0
	}
	c.schedule(cmd)
}

func (c *cronCmdContainer) cleanOnceTasks() {
	c.Lock()
	defer c.Unlock()
	out := c.Pts[:0]
	for _, p := range c.Pts {
		if p != nil && p.IsOnce && p.LastRunningTime != "" {
			if c.cronSched != nil && p.CronEntityId > 0 {
				c.cronSched.Remove(p.CronEntityId)
			}
			if c.Bucket != nil {
				c.Bucket.Delete(p.ID)
			}
			GetMyLog().Info("一次性定时任务已完成清理 %s", p.ID)
			continue
		}
		out = append(out, p)
	}
	c.Pts = out
}

func (c *cronCmdContainer) delete(id string) {
	c.Lock()
	defer c.Unlock()
	out := c.Pts[:0]
	for _, p := range c.Pts {
		if p != nil && p.ID == id {
			if c.Bucket != nil {
				c.Bucket.Delete(id)
			}
			continue
		}
		out = append(out, p)
	}
	c.Pts = out
}

func (c *cronCmdContainer) toggle(id string) {
	c.Lock()
	defer c.Unlock()
	for _, p := range c.Pts {
		if p != nil && p.ID == id {
			p.Status = !p.Status
			c.save(p)
			return
		}
	}
}

func (c *cronCmdContainer) afterLoad() {
	if c.Bucket == nil {
		return
	}
	for _, k := range c.Bucket.Keys() {
		data := c.Bucket.GetBytes(k)
		if len(data) == 0 {
			continue
		}
		var cmd CronCmd
		if err := json.Unmarshal(data, &cmd); err != nil {
			continue
		}
		cmd.ID = k
		c.Pts = append(c.Pts, &cmd)
		if c.MaxId != nil {
			var n int64
			if _, err := fmt.Sscanf(k, "%d", &n); err == nil {
				c.MaxId.SetMaxId(n)
			}
		}
	}
}

func (c *cronCmdContainer) reportAll() {
	c.RLock()
	defer c.RUnlock()
	for _, p := range c.Pts {
		if p == nil {
			continue
		}
		st := "停用"
		if p.Status {
			st = "启用"
		}
		GetMyLog().Info("定时任务 [%s] %s %s → %s", p.ID, p.Name, st, p.Cron)
	}
}

func runCronCmd(cmd *CronCmd) {
	if cmd == nil {
		return
	}
	// 读取快照后立即释放锁，执行期间不阻塞定时任务管理操作。
	ccs.RLock()
	owner := strings.TrimSpace(cmd.Owner)
	execUser := cronIdentity(cmd)
	name, command := cmd.Name, cmd.Command
	isOnce := cmd.IsOnce
	execID := beginExecution(executionKindCron, name, command, owner)
	ccs.RUnlock()
	status, detail, errText := executionSuccess, "", ""
	defer func() {
		if r := recover(); r != nil {
			status = executionFailed
			errText = fmt.Sprintf("执行异常: %v", r)
			GetMyLog().Warn("定时任务 %s 执行异常: %v", cmd.ID, r)
		}
		ccs.Lock()
		cmd.LastRun = time.Now().Format("2006-01-02 15:04:05")
		cmd.RunCount++
		if status == executionSuccess {
			cmd.LastSuccess = cmd.LastRun
			cmd.LastStatus = executionSuccess
			cmd.LastError = ""
			cmd.FailCount = 0
		} else {
			cmd.FailCount++
			cmd.LastStatus = executionFailed
			cmd.LastError = truncateText(errText, 300)
			if cmd.FailCount >= 3 {
				cmd.Status = false
				cmd.LastError = "连续失败 3 次，已暂停: " + cmd.LastError
			}
		}
		ccs.save(cmd)
		ccs.Unlock()
		finishExecution(execID, status, detail, errText)
		if status != executionSuccess {
			GetMyLog().Warn("定时任务 %s 执行失败: %s", cmd.Name, errText)
		}
	}()
	ccs.Lock()
	cmd.LastRunningTime = time.Now().Format("2006-01-02 15:04:05")
	ccs.Unlock()
	if isOnce {
		defer ccs.cleanOnceTasks()
	}
	if strings.TrimSpace(command) == "" {
		status = executionSkipped
		return
	}
	GetMyLog().Info("定时任务 %s 执行: %s", name, command)
	out, err := dispatchLocalCommandFor(command, execUser)
	detail = out
	if err != nil {
		status = executionFailed
		errText = err.Error()
		return
	}
}

func addCronCmd(label, cronSpec, command string) {
	ccs.add(&CronCmd{
		Name:      label,
		Cron:      cronSpec,
		Command:   command,
		Status:    true,
		Type:      []string{"system"},
		OwnerRole: RoleMaster,
	})
}

func isValidCronSpec(spec string) bool {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return false
	}
	_, err := cronSpecParser.Parse(spec)
	return err == nil
}

// CronCmdsSnapshot 返回全部定时任务快照，供内部与主控视图使用。
func CronCmdsSnapshot() []*CronCmd {
	return CronCmdsSnapshotFor("", RoleMaster)
}

// CronCmdsSnapshotFor 按操作者权限返回定时任务快照：
// 主控可见全部，其他角色只能看到自己创建的任务，避免泄露命令与会话参数。
func CronCmdsSnapshotFor(actor, role string) []*CronCmd {
	ccs.RLock()
	defer ccs.RUnlock()
	now := time.Now()
	out := make([]*CronCmd, 0, len(ccs.Pts))
	for _, p := range ccs.Pts {
		if p == nil || !cronManageable(p, actor, role) {
			continue
		}
		cp := *p
		if cp.Status {
			cp.NextRunAt = nextCronRun(cp.Cron, now)
		} else {
			cp.NextRunAt = ""
		}
		out = append(out, &cp)
	}
	return out
}

// nextCronRun 计算下一次触发时间；表达式无效或已无后续时返回空串。
func nextCronRun(spec string, now time.Time) string {
	sched, err := cronSpecParser.Parse(strings.TrimSpace(spec))
	if err != nil {
		return ""
	}
	next := sched.Next(now)
	if next.IsZero() {
		return ""
	}
	return next.Format("2006-01-02 15:04:05")
}

// cronIdentity 返回定时任务执行时采用的身份。
// 只有主控任务以 system 运行，其余任务必须绑定创建者本人，避免普通任务获得系统权限。
func cronIdentity(cmd *CronCmd) string {
	if cmd == nil {
		return "system"
	}
	owner := strings.TrimSpace(cmd.Owner)
	switch cmd.OwnerRole {
	case RoleMaster:
		return "system"
	case RoleOps, RoleReadonly:
		if owner != "" && owner != "system" {
			return owner
		}
		return "system"
	}
	if owner == "" || owner == "system" {
		return "system"
	}
	return owner
}

// cronManageable 判断操作者能否管理某条定时任务：主控可管理全部，其他人仅限自己创建的。
func cronManageable(cmd *CronCmd, actor, role string) bool {
	if cmd == nil {
		return false
	}
	actor = strings.TrimSpace(actor)
	if role == RoleMaster {
		return true
	}
	return actor != "" && cmd.Owner == actor
}

// cronTaskAllows 判断用户是否有权触发名为 label 的定时任务。
func cronTaskAllows(label, userID string) bool {
	label = strings.TrimSpace(label)
	if label == "" {
		return false
	}
	if privilegedUser(userID) {
		return true
	}
	ccs.RLock()
	defer ccs.RUnlock()
	for _, p := range ccs.Pts {
		if p != nil && p.Name == label && cronIdentity(p) == userID {
			return true
		}
	}
	return false
}

func RunCronCmdsByIds(ids []string, actor, role string) int {
	ccs.RLock()
	defer ccs.RUnlock()
	ran := 0
	for _, id := range ids {
		for _, p := range ccs.Pts {
			if p == nil || p.ID != id {
				continue
			}
			if !cronManageable(p, actor, role) {
				break
			}
			go runCronCmd(p)
			ran++
			break
		}
	}
	return ran
}

// AddCronCmdFrom 创建定时任务，归属与角色由服务端会话决定，客户端传入的 owner 一律忽略。
func AddCronCmdFrom(fields map[string]interface{}, actor, role string) *CronCmd {
	actor = strings.TrimSpace(actor)
	role = strings.TrimSpace(role)
	if actor == "" || actor == "unknown" || roleRank[role] == 0 {
		return nil
	}
	cmd := &CronCmd{
		Name:      strField(fields, "name"),
		Command:   strField(fields, "command"),
		Cron:      strField(fields, "cron"),
		Status:    true,
		Type:      []string{"web"},
		Owner:     actor,
		OwnerRole: role,
	}
	if v, ok := fields["status"].(bool); ok {
		cmd.Status = v
	}
	if v, ok := fields["type"].([]interface{}); ok {
		cmd.Type = strSlice(v)
	}
	id := ccs.add(cmd)
	cmd.ID = id
	GetMyLog().Info("新增定时任务 %s %s", id, cmd.Name)
	return cmd
}

// UpdateCronCmdFrom 兼容旧控制台调用：忽略错误并返回更新后的任务。
func UpdateCronCmdFrom(id string, fields map[string]interface{}, actor, role string) *CronCmd {
	cmd, _ := UpdateCronCmd(id, fields, actor, role)
	return cmd
}

// UpdateCronCmd 更新定时任务并立即重排调度；仅创建者或主控可改，归属与角色保持原值。
// 表达式或命令非法时返回错误且不修改原任务，避免出现“保存成功但不再触发”。
func UpdateCronCmd(id string, fields map[string]interface{}, actor, role string) (*CronCmd, error) {
	ccs.Lock()
	defer ccs.Unlock()
	for _, p := range ccs.Pts {
		if p == nil || p.ID != id {
			continue
		}
		if !cronManageable(p, actor, role) {
			return nil, fmt.Errorf("定时任务不存在或无权操作")
		}
		name := p.Name
		if v, ok := fields["name"].(string); ok && strings.TrimSpace(v) != "" {
			name = strings.TrimSpace(v)
		}
		command := p.Command
		if v, ok := fields["command"].(string); ok {
			command = strings.TrimSpace(v)
		}
		spec := p.Cron
		if v, ok := fields["cron"].(string); ok {
			spec = strings.TrimSpace(v)
		}
		if name == "" || command == "" {
			return nil, fmt.Errorf("缺少名称/命令")
		}
		if !isValidCronSpec(spec) {
			return nil, fmt.Errorf("Cron 表达式无效，支持 5 段或 6 段（含秒）写法")
		}
		if err := validateCronFeatureCommand(command); err != nil {
			return nil, err
		}
		p.Name = name
		p.Command = command
		p.Cron = spec
		if v, ok := fields["status"].(bool); ok {
			p.Status = v
		}
		if v, ok := fields["type"].([]interface{}); ok {
			p.Type = strSlice(v)
		}
		p.UpdateOn = time.Now().Format("2006-01-02 15:04:05")
		ccs.rescheduleLocked(p)
		ccs.save(p)
		cp := *p
		return &cp, nil
	}
	return nil, fmt.Errorf("定时任务不存在")
}

func DeleteCronCmdsByIds(ids []string, actor, role string) int {
	ccs.Lock()
	defer ccs.Unlock()
	deleted := 0
	for _, id := range ids {
		out := ccs.Pts[:0]
		for _, p := range ccs.Pts {
			if p != nil && p.ID == id && cronManageable(p, actor, role) {
				if ccs.cronSched != nil && p.CronEntityId > 0 {
					ccs.cronSched.Remove(p.CronEntityId)
				}
				if ccs.Bucket != nil {
					ccs.Bucket.Delete(id)
				}
				deleted++
				continue
			}
			out = append(out, p)
		}
		ccs.Pts = out
	}
	if deleted > 0 {
		GetMyLog().Info("删除 %d 个定时任务", deleted)
	}
	return deleted
}

func SetCronCmdStatus(id string, on bool, actor, role string) bool {
	ccs.Lock()
	defer ccs.Unlock()
	for _, p := range ccs.Pts {
		if p == nil || p.ID != id {
			continue
		}
		if !cronManageable(p, actor, role) {
			return false
		}
		p.Status = on
		p.UpdateOn = time.Now().Format("2006-01-02 15:04:05")
		if ccs.cronSched != nil {
			if p.CronEntityId > 0 {
				ccs.cronSched.Remove(p.CronEntityId)
				p.CronEntityId = 0
			}
			if on && p.Cron != "" {
				cmd := p
				eid, err := ccs.cronSched.AddFunc(p.Cron, func() {
					enqueueCron(cmd)
				})
				if err == nil {
					p.CronEntityId = eid
				}
			}
		}
		ccs.save(p)
		return true
	}
	return false
}

func strField(fields map[string]interface{}, key string) string {
	if v, ok := fields[key].(string); ok {
		return v
	}
	return ""
}

func strSlice(v []interface{}) []string {
	out := make([]string, 0, len(v))
	for _, item := range v {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}
