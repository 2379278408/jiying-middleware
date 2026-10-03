package gin

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"jiying/middleware/core"
)

type Rec struct {
	ID       int64                  `json:"id"`
	ON       bool                   `json:"on,omitempty"`
	Position int64                  `json:"position,omitempty"`
	Fields   map[string]interface{} `json:"fields,omitempty"`
}

type BucketStore struct {
	bucketName string
	key        string
	once       bool
	list       []*Rec
	seq        int64
}

func NewBucketStore(bucketName, key string) *BucketStore {
	b := &BucketStore{bucketName: bucketName, key: key}
	b.load()
	return b
}

func (b *BucketStore) bucket() core.Bucket {
	bk, err := core.MakeBucket(b.bucketName, 0)
	if err != nil {
		return nil
	}
	return bk
}

func (b *BucketStore) load() {
	if b.once {
		return
	}
	b.once = true
	bk := b.bucket()
	if bk == nil {
		return
	}
	data := bk.GetBytes(b.key)
	if len(data) == 0 {
		return
	}
	var arr []*Rec
	if json.Unmarshal(data, &arr) == nil {
		b.list = arr
	}
	for _, r := range arr {
		if r.ID > b.seq {
			b.seq = r.ID
		}
	}
}

func (b *BucketStore) save() error {
	bk := b.bucket()
	if bk == nil {
		return fmt.Errorf("bucket 不可用")
	}
	data, err := json.Marshal(b.list)
	if err != nil {
		return err
	}
	if !bk.Set(b.key, string(data)) {
		return fmt.Errorf("保存失败")
	}
	return nil
}

func (b *BucketStore) List() []*Rec { return b.list }

func (b *BucketStore) Get(id int64) *Rec {
	for _, r := range b.list {
		if r.ID == id {
			return r
		}
	}
	return nil
}

func (b *BucketStore) Add(r *Rec) error {
	b.seq++
	r.ID = b.seq
	for _, old := range b.list {
		if recEqual(old, r) {
			return nil
		}
	}
	b.list = append(b.list, r)
	return b.save()
}

func (b *BucketStore) Upd(r *Rec) error {
	for i, old := range b.list {
		if old.ID == r.ID {
			b.list[i] = r
			return b.save()
		}
	}
	return fmt.Errorf("未找到")
}

func (b *BucketStore) Del(id int64) error {
	out := b.list[:0]
	for _, r := range b.list {
		if r.ID != id {
			out = append(out, r)
		}
	}
	b.list = out
	return b.save()
}

func (b *BucketStore) SetField(ids []int64, set func(r *Rec)) error {
	for _, r := range b.list {
		for _, id := range ids {
			if r.ID == id {
				set(r)
				break
			}
		}
	}
	return b.save()
}

func recEqual(a, b *Rec) bool {
	if a.Fields == nil || b.Fields == nil {
		return false
	}
	na, _ := a.Fields["name"].(string)
	nb, _ := b.Fields["name"].(string)
	return na != "" && na == nb
}

func parseIDs(c *gin.Context) []int64 {
	if id := c.Param("id"); id != "" {
		if n, err := strconv.ParseInt(id, 10, 64); err == nil {
			return []int64{n}
		}
	}
	raw, _ := c.GetRawData()
	if len(raw) == 0 {
		raw = []byte(c.PostForm("ids"))
	}
	var ids []int64
	json.Unmarshal(raw, &ids)
	return ids
}

func regCRUD(group *gin.RouterGroup, path string, store *BucketStore) {
	group.GET(path, func(c *gin.Context) {
		jsonOK(c, store.List())
	})
	group.GET(path+"/:id", func(c *gin.Context) {
		ids := parseIDs(c)
		if len(ids) == 1 {
			if r := store.Get(ids[0]); r != nil {
				jsonOK(c, r)
				return
			}
		}
		jsonFailMsg(c, "未找到")
	})
	group.POST(path, func(c *gin.Context) {
		var r Rec
		if err := c.ShouldBindJSON(&r); err != nil {
			jsonFailMsg(c, "参数错误")
			return
		}
		if err := store.Add(&r); err != nil {
			jsonFailMsg(c, "新增失败")
			return
		}
		jsonOK(c, store.List())
	})
	group.PUT(path, func(c *gin.Context) {
		var r Rec
		if err := c.ShouldBindJSON(&r); err != nil {
			jsonFailMsg(c, "参数错误")
			return
		}
		if err := store.Upd(&r); err != nil {
			jsonFailMsg(c, "修改失败")
			return
		}
		jsonOK(c, store.List())
	})
	group.DELETE(path+"/:id", func(c *gin.Context) {
		ids := parseIDs(c)
		for _, id := range ids {
			_ = store.Del(id)
		}
		jsonOK(c, store.List())
	})
}

func regOperSwitch(group *gin.RouterGroup, path string, store *BucketStore) {
	setOn := func(on bool) gin.HandlerFunc {
		return func(c *gin.Context) {
			raw, _ := c.GetRawData()
			var ids []int64
			if json.Unmarshal(raw, &ids) != nil || len(ids) == 0 {
				jsonFailMsg(c, "参数错误")
				return
			}
			_ = store.SetField(ids, func(r *Rec) { r.ON = on })
			jsonOK(c, store.List())
		}
	}
	g := group.Group(path)
	g.POST("/enable", setOn(true))
	g.POST("/disable", setOn(false))
}

func splitInt64s(s string) []int64 {
	var out []int64
	for _, p := range strings.Split(s, ",") {
		if n, err := strconv.ParseInt(strings.TrimSpace(p), 10, 64); err == nil {
			out = append(out, n)
		}
	}
	return out
}
