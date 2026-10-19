package core

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"testing"
)

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
	BucketApp.Delete(marketPublishersKey)
	BucketApp.Delete("jiying_features")
	t.Cleanup(func() {
		BucketApp.Delete(marketKey)
		BucketApp.Delete(marketPublishersKey)
		BucketApp.Delete("jiying_features")
	})
	features.items = map[string]*Feature{}
	marketLocalKeyCache.Lock()
	marketLocalKeyCache.key = nil
	marketLocalKeyCache.Unlock()
}

func resignMarketEntry(t *testing.T, e MarketEntry) MarketEntry {
	t.Helper()
	e.Digest = sha256Hex(pluginDigestSource(e.PluginMeta, e.Content))
	signed, err := signMarketMeta(e.PluginMeta)
	if err != nil {
		t.Fatal(err)
	}
	e.PluginMeta = signed
	return e
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
	} else if !e.Signed || e.Publisher != marketLocalPublisher || e.Signature == "" {
		t.Fatalf("new market entry should be signed: %+v", e)
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
	restored = resignMarketEntry(t, restored)
	saveMarket(map[string]MarketEntry{"dig": restored})
	if err := InstallMarketPlugin("dig", false); err != nil {
		t.Fatalf("clean install should succeed: %v", err)
	}
	lower := restored
	lower.Version = "1.0.0"
	lower.Content = "// [version: 1.0.0]\nresult = 1;"
	lower = resignMarketEntry(t, lower)
	saveMarket(map[string]MarketEntry{"dig": lower})
	if err := InstallMarketPlugin("dig", false); err == nil {
		t.Fatal("downgrade install over newer installed version should be rejected")
	}
	if err := InstallMarketPlugin("dig", true); err != nil {
		t.Fatalf("force downgrade install should succeed: %v", err)
	}
}

func TestMarketSignatureRejectsTamperedMetadata(t *testing.T) {
	setupMarketBucket(t, "market-signature")
	meta := PluginMeta{Name: "signed", Version: "1.0.0", Kind: "script", Target: "signed.js"}
	if err := PublishPlugin(meta, "result = 1;", false); err != nil {
		t.Fatal(err)
	}
	entry := ExportPluginMust(t, "signed")
	entry.Version = "9.9.9"
	saveMarket(map[string]MarketEntry{"signed": entry})
	if err := InstallMarketPlugin("signed", true); err == nil {
		t.Fatal("metadata tampering must fail signature verification even with force")
	}
	entry = ExportPluginMust(t, "signed")
	entry.Content = "result = 2;"
	entry.Digest = sha256Hex(entry.Content)
	saveMarket(map[string]MarketEntry{"signed": entry})
	if err := InstallMarketPlugin("signed", true); err == nil {
		t.Fatal("content tampering must fail signature verification even with force")
	}
}

func TestExternalPublisherSignature(t *testing.T) {
	setupMarketBucket(t, "market-external")
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	pubText := base64.StdEncoding.EncodeToString(pub)
	if _, err := RegisterMarketPublisher("vendor", pubText); err != nil {
		t.Fatal(err)
	}
	meta := PluginMeta{
		Name: "external", Version: "1.0.0", Kind: "script", Target: "external.js",
		Identity: "app", Scope: "public",
		Publisher: "vendor", Digest: sha256Hex("result = 3;"),
	}
	meta.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(priv, pluginSigningPayload(meta)))
	if err := PublishPlugin(meta, "result = 3;", false); err != nil {
		t.Fatal(err)
	}
	if err := InstallMarketPlugin("external", false); err != nil {
		t.Fatalf("signed external plugin should install: %v", err)
	}
	if _, ok := marketPublisherKey("vendor"); !ok {
		t.Fatal("publisher key should be registered")
	}
}

func TestPublishRejectsUnknownOrUnsignedExternalPublisher(t *testing.T) {
	setupMarketBucket(t, "market-external-bad")
	meta := PluginMeta{Name: "bad", Version: "1.0.0", Kind: "script", Target: "bad.js", Publisher: "missing"}
	if err := PublishPlugin(meta, "result = 1;", false); err == nil {
		t.Fatal("unknown external publisher should be rejected")
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := RegisterMarketPublisher("vendor2", base64.StdEncoding.EncodeToString(pub)); err != nil {
		t.Fatal(err)
	}
	meta.Publisher = "vendor2"
	meta.Digest = sha256Hex("result = 1;")
	meta.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(priv, []byte("wrong payload")))
	if err := PublishPlugin(meta, "result = 1;", false); err == nil {
		t.Fatal("invalid external signature should be rejected")
	}
}

func TestRegisterMarketPublisherRejectsKeyReuse(t *testing.T) {
	setupMarketBucket(t, "market-publisher-key")
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := RegisterMarketPublisher("vendor", base64.StdEncoding.EncodeToString(pub)); err != nil {
		t.Fatal(err)
	}
	other, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := RegisterMarketPublisher("vendor", base64.StdEncoding.EncodeToString(other)); err == nil {
		t.Fatal("replacing a publisher public key must be rejected")
	}
	if err := DeleteMarketPublisher("vendor"); err != nil {
		t.Fatal(err)
	}
	if _, err := RegisterMarketPublisher("vendor", base64.StdEncoding.EncodeToString(other)); err != nil {
		t.Fatal(err)
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

func TestRegisterMarketPublisherProtectsLocalName(t *testing.T) {
	setupMarketBucket(t, "market-local-name")
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := RegisterMarketPublisher(marketLocalPublisher, base64.StdEncoding.EncodeToString(pub)); err == nil {
		t.Fatal("local 名称不能被外部公钥占用")
	}
	local, err := localMarketPublicKey()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := RegisterMarketPublisher(marketLocalPublisher, local); err != nil {
		t.Fatalf("本机公钥应可幂等登记: %v", err)
	}
}
