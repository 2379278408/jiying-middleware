package core

import "testing"

func TestFeatureAllowedForApp(t *testing.T) {
	all := &Feature{Name: "all"}
	if !featureAllowedForApp(all, "") || !featureAllowedForApp(all, "app-a") {
		t.Fatal("未声明应用白名单时应适用于所有应用")
	}
	one := &Feature{Name: "one", Apps: []string{"app-a"}}
	if !featureAllowedForApp(one, "app-a") || !featureAllowedForApp(one, "APP-A") {
		t.Fatal("白名单应大小写不敏感匹配")
	}
	if featureAllowedForApp(one, "app-b") || featureAllowedForApp(one, "") {
		t.Fatal("不在白名单内的应用应被拒绝")
	}
	star := &Feature{Name: "star", Apps: []string{"*"}}
	if !featureAllowedForApp(star, "anything") {
		t.Fatal("* 应允许任意应用")
	}
}

func TestSameStringSet(t *testing.T) {
	if !sameStringSet([]string{"a", "b"}, []string{"b", "a"}) {
		t.Fatal("忽略顺序")
	}
	if sameStringSet([]string{"a"}, []string{"a", "b"}) || sameStringSet([]string{"a", "a"}, []string{"a", "b"}) {
		t.Fatal("长度或重复项不同应视为不同集合")
	}
}
