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
	defer func() {
		if r := recover(); r != nil {
			GetMyLog().Warn("定时任务 %s 执行异常: %v", cmd.ID, r)
		}
	}()
	cmd.LastRunningTime = time.Now().Format("2006-01-02 15:04:05")
	if cmd.IsOnce {
		defer ccs.cleanOnceTasks()
	}
	if cmd.Command != "" {
		GetMyLog().Info("定时任务 %s 执行: %s", cmd.Name, cmd.Command)
		dispatchLocalCommand(cmd.Command)
	}
	cmd.LastRun = time.Now().Format("2006-01-02 15:04:05")
	cmd.LastSuccess = time.Now().Format("2006-01-02 15:04:05")
	ccs.save(cmd)
}

func addCronCmd(label, cronSpec, command string) {
	ccs.add(&CronCmd{
		Name:    label,
		Cron:    cronSpec,
		Command: command,
		Status:  true,
		Type:    []string{"system"},
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

func CronCmdsSnapshot() []*CronCmd {
	ccs.RLock()
	defer ccs.RUnlock()
	out := make([]*CronCmd, 0, len(ccs.Pts))
	for _, p := range ccs.Pts {
		if p == nil {
			continue
		}
		out = append(out, p)
	}
	return out
}

func RunCronCmdsByIds(ids []string) int {
	ccs.RLock()
	defer ccs.RUnlock()
	ran := 0
	for _, id := range ids {
		for _, p := range ccs.Pts {
			if p == nil || p.ID != id {
				continue
			}
			go runCronCmd(p)
			ran++
			break
		}
	}
	return ran
}

func AddCronCmdFrom(fields map[string]interface{}) *CronCmd {
	cmd := &CronCmd{
		Name:    strField(fields, "name"),
		Command: strField(fields, "command"),
		Cron:    strField(fields, "cron"),
		Status:  true,
		Type:    []string{"web"},
		Owner:   strField(fields, "owner"),
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

func UpdateCronCmdFrom(id string, fields map[string]interface{}) *CronCmd {
	ccs.Lock()
	defer ccs.Unlock()
	for _, p := range ccs.Pts {
		if p == nil || p.ID != id {
			continue
		}
		if v, ok := fields["name"].(string); ok && v != "" {
			p.Name = v
		}
		if v, ok := fields["command"].(string); ok {
			p.Command = v
		}
		if v, ok := fields["cron"].(string); ok {
			p.Cron = v
		}
		if v, ok := fields["status"].(bool); ok {
			p.Status = v
		}
		if v, ok := fields["type"].([]interface{}); ok {
			p.Type = strSlice(v)
		}
		p.UpdateOn = time.Now().Format("2006-01-02 15:04:05")
		ccs.save(p)
		return p
	}
	return nil
}

func DeleteCronCmdsByIds(ids []string) int {
	ccs.Lock()
	defer ccs.Unlock()
	deleted := 0
	for _, id := range ids {
		out := ccs.Pts[:0]
		for _, p := range ccs.Pts {
			if p != nil && p.ID == id {
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

func SetCronCmdStatus(id string, on bool) bool {
	ccs.Lock()
	defer ccs.Unlock()
	for _, p := range ccs.Pts {
		if p == nil || p.ID != id {
			continue
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
