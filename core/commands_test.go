package core

import "testing"

func TestAddCommandDedup(t *testing.T) {
	backup := Functions
	Functions = nil
	t.Cleanup(func() { Functions = backup })
	_ = AddCommand(5, &Function{Title: "命令A", Handle: "icmd_a", Priority: 9})
	_ = AddCommand(5, &Function{Title: "命令B", Handle: "icmd_b", Priority: 3})
	_ = AddCommand(5, &Function{Title: "命令A", Handle: "icmd_a2", Priority: 9})
	if len(Functions) != 2 {
		t.Fatalf("重复标题应被跳过，实际 len=%d 期望 2", len(Functions))
	}
}

func TestAddCommandPriorityOrder(t *testing.T) {
	backup := Functions
	Functions = nil
	t.Cleanup(func() { Functions = backup })
	_ = AddCommand(0, &Function{Title: "低", Priority: 1})
	_ = AddCommand(0, &Function{Title: "高", Priority: 99})
	_ = AddCommand(0, &Function{Title: "中", Priority: 50})
	if Functions[0].Title != "高" || Functions[2].Title != "低" {
		t.Fatalf("优先级排序错误: %v,%v,%v", Functions[0].Title, Functions[1].Title, Functions[2].Title)
	}
}

func TestCronCmdContainer(t *testing.T) {
	ccs.Lock()
	ccs.Bucket = nil
	ccs.Unlock()

	id := ccs.add(&CronCmd{Name: "测试任务", Cron: "* * * * *", Command: "echo hi", Status: true})
	if id == "" {
		t.Fatalf("add 应返回 ID")
	}
	ccs.toggle(id)
	var got *CronCmd
	ccs.each(func(c *CronCmd) {
		if c.ID == id {
			got = c
		}
	})
	if got == nil || got.Status {
		t.Fatalf("toggle 后应变为停用")
	}
	ccs.delete(id)
	got = nil
	ccs.each(func(c *CronCmd) {
		if c.ID == id {
			got = c
		}
	})
	if got != nil {
		t.Fatalf("delete 后不应存在")
	}
}

func TestCommandConvertApply(t *testing.T) {
	cvtC.Lock()
	cvtC.Bucket = nil
	cvtC.MaxId = nil
	cvtC.Pts = nil
	cvtC.Unlock()
	cvtC.add(&Convert{Title: "缩写", ConvertFrom: `^\.jd$`, ConvertTo: "bing", On: true})
	out := commandConvert(".jd")
	if out != "bing" {
		t.Fatalf("转换未生效: %q", out)
	}
	if literal := commandConvert("hello"); literal != "hello" {
		t.Fatalf("不匹配时应原样返回: %q", literal)
	}
}

func TestRepliesMatch(t *testing.T) {
	replies.Lock()
	replies.Items = nil
	replies.Unlock()
	replies.add(ReplyInfo{Name: "欢迎", Keyword: []string{"你好"}, Reply: "你好呀", Enable: true})
	if got := replies.getReply("你好，世界"); got != "你好呀" {
		t.Fatalf("关键词应命中: %q", got)
	}
	if got := replies.getReply("再见"); got != "" {
		t.Fatalf("无关消息不应命中: %q", got)
	}
}
