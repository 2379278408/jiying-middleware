package core

import "testing"

func TestSetAndGetPluginAuth(t *testing.T) {
	initBuckets()
	_ = BucketApp.Set(pluginAuthBucket, "")
	Functions = []*Function{{Title: "demo", Author: "tester"}}

	if !GetPluginAuthPermission("demo", "tester", false, "qls") {
		t.Fatal("无记录应默认允许")
	}

	SetPluginAuth("demo", "tester", false, "qls", true)
	if !GetPluginAuthPermission("demo", "tester", false, "qls") {
		t.Fatal("开启 qls 后应允许")
	}

	if !GetPluginAuthPermission("demo", "任意作者", false, "qls") {
		t.Fatal("author 不应参与匹配")
	}

	SetPluginAuth("demo", "tester", false, "qls", false)
	if GetPluginAuthPermission("demo", "tester", false, "qls") {
		t.Fatal("关闭 qls 后应拒绝")
	}

	if !GetPluginAuthPermission("demo", "tester", false, "jdNotify") {
		t.Fatal("未设置桶应默认允许")
	}

	rows := GetPluginAuthTable()
	var row *PluginAuthRow
	for i := range rows {
		if rows[i].PluginName == "demo" {
			row = &rows[i]
			break
		}
	}
	if row == nil {
		t.Fatal("授权表应包含 demo 行")
	}
	found := false
	for _, a := range row.Auth {
		if a.BucketName == "qls" {
			found = true
			if a.BucketRW {
				t.Fatal("qls 应已关闭")
			}
		}
	}
	if !found {
		t.Fatal("demo 行应含 qls 桶")
	}

	SetPluginAuth("ghost", "x", false, "qls", true)
	rows = GetPluginAuthTable()
	ghostSeen := false
	for _, r := range rows {
		if r.PluginName == "ghost" {
			ghostSeen = true
		}
	}
	if !ghostSeen {
		t.Fatal("已卸载插件的授权配置应保留回显")
	}
	_ = BucketApp.Set(pluginAuthBucket, "")
}
