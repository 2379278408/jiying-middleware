package core

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

type ReplyInfo struct {
	Name    string   `json:"name"`
	Keyword []string `json:"keyword"`
	Reply   string   `json:"reply"`
	Enable  bool     `json:"enable"`
}

type replyTable struct {
	sync.RWMutex
	Items []ReplyInfo
}

var replies = &replyTable{}

func InitReplies() {
	replies.afterLoad()
	registerSysCommand("回复管理", "replies_manage", 5, func() {
		replies.reportAll()
	})
}

func (r *replyTable) afterLoad() {
	raw := BucketApp.GetString("Replies")
	if raw == "" {
		return
	}
	var items []ReplyInfo
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		GetMyLog().Warn("回复规则解析失败: %v", err)
		return
	}
	r.Lock()
	r.Items = items
	r.Unlock()
	GetMyLog().Info("回复规则已加载 %d 条", len(items))
}

func (r *replyTable) save() {
	r.RLock()
	data, _ := json.Marshal(r.Items)
	r.RUnlock()
	if BucketApp == nil {
		return
	}
	BucketApp.Set("Replies", string(data))
}

func (r *replyTable) add(rep ReplyInfo) {
	r.Lock()
	r.Items = append(r.Items, rep)
	r.Unlock()
	r.save()
}

func (r *replyTable) getReply(msg string) string {
	r.RLock()
	defer r.RUnlock()
	for _, it := range r.Items {
		if !it.Enable {
			continue
		}
		for _, kw := range it.Keyword {
			if kw != "" && strings.Contains(msg, kw) {
				return it.Reply
			}
		}
	}
	return ""
}

func (r *replyTable) reportAll() {
	r.RLock()
	n := len(r.Items)
	r.RUnlock()
	ReplyCurrent(fmt.Sprintf("回复规则 %d 条", n))
}
