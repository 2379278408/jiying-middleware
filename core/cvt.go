package core

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"jiying/middleware/develop/a_boltdb"
	"jiying/middleware/utils"
)

type Convert struct {
	ID            int64    `json:"id"`
	IsCron        bool     `json:"is_cron"`
	Start         string   `json:"start"`
	End           string   `json:"end"`
	Title         string   `json:"title"`
	Type          []string `json:"type"`
	GroupId       string   `json:"group_id"`
	GroupManager  []string `json:"group_manager"`
	UserManager   []string `json:"user_manager"`
	GroupIdWhite  []string `json:"group_id_white"`
	GroupIdBlack  []string `json:"group_id_black"`
	UserIdWhite   []string `json:"user_id_white"`
	UserIdBlack   []string `json:"user_id_black"`
	ConvertFrom   string   `json:"convert_from"`
	ConvertTo     string   `json:"convert_to"`
	ConvertResult string   `json:"convert_result"`
	On            bool     `json:"on"`
	Forword       bool     `json:"forword"`
	Normal        bool     `json:"normal"`
	Before        bool     `json:"before"`
}

type cvtContainer struct {
	sync.RWMutex
	Bucket       *a_boltdb.Store
	MaxId        *utils.IDGenerator
	Position     *utils.PositionGenerator
	Pts          []*Convert
	compileCache sync.Map
}

var cvtC = &cvtContainer{}

func initCvt() {
	bkt, err := a_boltdb.NewBucket("cvt", 3)
	if err != nil {
		GetMyLog().Warn("转换桶创建失败: %v", err)
		bkt = nil
	}
	cvtC.Bucket = bkt
	cvtC.MaxId = utils.NewIDGenerator(0)
	cvtC.Position = utils.NewPositionGenerator()
	cvtC.afterLoad()
}

func (c *cvtContainer) afterLoad() {
	if c.Bucket == nil {
		return
	}
	for _, k := range c.Bucket.Keys() {
		data := c.Bucket.GetBytes(k)
		if len(data) == 0 {
			continue
		}
		var cv Convert
		if err := json.Unmarshal(data, &cv); err != nil {
			continue
		}
		c.Pts = append(c.Pts, &cv)
		if c.MaxId != nil && cv.ID > c.MaxId.GetCurrentID() {
			c.MaxId.SetMaxId(cv.ID)
		}
	}
}

func (c *cvtContainer) add(cv *Convert) int64 {
	c.Lock()
	defer c.Unlock()
	if cv.ID == 0 {
		if c.MaxId == nil {
			c.MaxId = utils.NewIDGenerator(0)
		}
		cv.ID = c.MaxId.NewID()
	}
	c.Pts = append(c.Pts, cv)
	c.save(cv)
	return cv.ID
}

func (c *cvtContainer) save(cv *Convert) {
	if c.Bucket == nil || cv == nil {
		return
	}
	c.Bucket.Set(fmt.Sprintf("%d", cv.ID), cv)
}

func commandConvert(msg string) string {
	cvtC.RLock()
	defer cvtC.RUnlock()
	for _, cv := range cvtC.Pts {
		if cv == nil || !cv.On || cv.ConvertFrom == "" {
			continue
		}
		re, ok := cvtC.compileCache.Load(cv.ConvertFrom)
		if !ok {
			r, err := regexp.Compile(cv.ConvertFrom)
			if err != nil {
				continue
			}
			cvtC.compileCache.Store(cv.ConvertFrom, r)
			re = r
		}
		if replaced := (re.(*regexp.Regexp)).ReplaceAllString(msg, cv.ConvertTo); replaced != msg {
			return replaced
		}
	}
	return msg
}

func (c *cvtContainer) reportAll() {
	c.RLock()
	defer c.RUnlock()
	for _, cv := range c.Pts {
		if cv == nil {
			continue
		}
		st := "停用"
		if cv.On {
			st = "启用"
		}
		GetMyLog().Info("转换 [%d] %s(%s) %s → %s", cv.ID, cv.Title, st,
			shortText(cv.ConvertFrom), shortText(cv.ConvertTo))
	}
}

func shortText(s string) string {
	if len(s) <= 12 {
		return s
	}
	return s[:12] + "..."
}

var _ = strings.Contains
