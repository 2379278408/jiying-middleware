package core

import "testing"

func setupMarketBucket(t *testing.T, name string) {
	t.Helper()
	t.Setenv("JIYING_RES", t.TempDir())
	t.Setenv("JIYING_DATA", t.TempDir())
	b, err := MakeBucket(name, 6)
	if err != nil {
		t.Fatal(err)
	}
	BucketApp = b
	BucketApp.Delete(marketKey)
	BucketApp.Delete("jiying_features")
	t.Cleanup(func() {
		BucketApp.Delete(marketKey)
		BucketApp.Delete("jiying_features")
	})
	features.items = map[string]*Feature{}
}

func TestPublishPluginValidatesMeta(t *testing.T) {
	setupMarketBucket(t, "market-validate")
	bad := []PluginMeta{
		{Name: "../evil"},
		{Name: "ok", Identity: "root"},
		{Name: "ok", Scope: "global"},
		{Name: "ok", Kind: "binary"},
		{Name: "ok", Kind: "http", Target: "ftp://x"},
	}
	for i, m := range bad {
		if err := PublishPlugin(m, "x", false); err == nil {
			t.Fatalf("case %d should be rejected: %+v", i, m)
		}
	}
}

func TestMarketPublishInstallDelete(t *testing.T) {
	setupMarketBucket(t, "market-flow")
	meta := PluginMeta{
		Name: "greeter", Version: "1.0.0", Kind: "script", Target: "greeter.js",
		Identity: "user", Scope: "master",
	}
	if err := PublishPlugin(meta, "result = 'hi ' + args;", false); err != nil {
		t.Fatal(err)
	}
	if e, ok := ExportPlugin("greeter"); !ok || e.Content == "" || e.Identity != "user" {
		t.Fatalf("export mismatch: %+v ok=%v", e, ok)
	} else if e.Digest == "" || e.Digest != sha256Hex(e.Content) {
		t.Fatalf("digest missing or wrong: %+v", e)
	}
	items := ListMarket()
	if len(items) != 1 || items[0].Installed {
		t.Fatalf("fresh entry should be uninstalled: %+v", items)
	}
	if err := InstallMarketPlugin("greeter", false); err != nil {
		t.Fatal(err)
	}
	f, ok := GetFeature("greeter")
	if !ok || f.Identity != "user" || f.Scope != "master" {
		t.Fatalf("installed feature mismatch: %+v", f)
	}
	items = ListMarket()
	if !items[0].Installed {
		t.Fatalf("entry should report installed: %+v", items)
	}
	if err := DeleteMarketPlugin("greeter"); err != nil {
		t.Fatal(err)
	}
	if err := DeleteMarketPlugin("greeter"); err == nil {
		t.Fatal("deleting a missing plugin should error")
	}
}

func TestInstallMissingPlugin(t *testing.T) {
	setupMarketBucket(t, "market-missing")
	if err := InstallMarketPlugin("nope", false); err == nil {
		t.Fatal("installing a missing plugin should error")
	}
}

func TestCompareVersion(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.1", "1.0.0", 1},
		{"1.2.0", "1.10.0", -1},
		{"2", "1.9.9", 1},
		{"1.0.0", "1.0", 0},
		{"0.1.0", "", 1},
		{"", "", 0},
		{"1.0.0-alpha", "1.0.0", -1},
	}
	for _, c := range cases {
		if got := CompareVersion(c.a, c.b); got != c.want {
			t.Errorf("CompareVersion(%q,%q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestPublishRejectsDowngradeAllowsForceRollback(t *testing.T) {
	setupMarketBucket(t, "market-rollback")
	base := PluginMeta{Name: "mvp", Version: "1.0.0", Kind: "script", Target: "mvp.js"}
	if err := PublishPlugin(base, "result = 1;", false); err != nil {
		t.Fatal(err)
	}
	downgrade := base
	downgrade.Version = "0.9.0"
	if err := PublishPlugin(downgrade, "result = 2;", false); err == nil {
		t.Fatal("downgrade publish should be rejected without force")
	}
	if err := PublishPlugin(downgrade, "result = 2;", true); err != nil {
		t.Fatalf("force rollback publish should succeed: %v", err)
	}
	if e, _ := ExportPlugin("mvp"); e.Version != "0.9.0" || e.Content != "result = 2;" {
		t.Fatalf("rollback content mismatch: %+v", e)
	}
}

func TestInstallVerifiesDigestAndRejectsDowngrade(t *testing.T) {
	setupMarketBucket(t, "market-digest")
	meta := PluginMeta{Name: "dig", Version: "2.0.0", Kind: "script", Target: "dig.js"}
	if err := PublishPlugin(meta, "// [version: 2.0.0]\nresult = 2;", false); err != nil {
		t.Fatal(err)
	}
	tampered := ExportPluginMust(t, "dig")
	tampered.Digest = sha256Hex("other content")
	saveMarket(map[string]MarketEntry{"dig": tampered})
	if err := InstallMarketPlugin("dig", false); err == nil {
		t.Fatal("tampered digest should be rejected")
	}
	if err := InstallMarketPlugin("dig", true); err == nil {
		t.Fatal("force must not bypass digest verification")
	}
	restored := ExportPluginMust(t, "dig")
	restored.Content = "// [version: 2.0.0]\nresult = 2;"
	restored.Digest = sha256Hex(restored.Content)
	saveMarket(map[string]MarketEntry{"dig": restored})
	if err := InstallMarketPlugin("dig", false); err != nil {
		t.Fatalf("clean install should succeed: %v", err)
	}
	lower := restored
	lower.Version = "1.0.0"
	lower.Content = "// [version: 1.0.0]\nresult = 1;"
	lower.Digest = sha256Hex(lower.Content)
	saveMarket(map[string]MarketEntry{"dig": lower})
	if err := InstallMarketPlugin("dig", false); err == nil {
		t.Fatal("downgrade install over newer installed version should be rejected")
	}
	if err := InstallMarketPlugin("dig", true); err != nil {
		t.Fatalf("force downgrade install should succeed: %v", err)
	}
}

func ExportPluginMust(t *testing.T, name string) MarketEntry {
	t.Helper()
	e, ok := ExportPlugin(name)
	if !ok {
		t.Fatalf("plugin %q missing", name)
	}
	return e
}
