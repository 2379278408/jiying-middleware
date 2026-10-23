package core

import "testing"

func setupRolesBucket(t *testing.T, name string) {
	t.Helper()
	t.Setenv("JIYING_DATA", t.TempDir())
	b, err := MakeBucket(name, 6)
	if err != nil {
		t.Fatal(err)
	}
	BucketApp = b
	BucketApp.Delete(rolesKey)
	t.Cleanup(func() { BucketApp.Delete(rolesKey) })
}

func TestUserRoleDefaultsToReadonly(t *testing.T) {
	setupRolesBucket(t, "roles-default")
	if got := UserRole("nobody"); got != RoleReadonly {
		t.Fatalf("unknown user should be readonly, got %q", got)
	}
	if HasRole("nobody", RoleOps) {
		t.Fatal("readonly must not pass ops")
	}
	if HasRole("nobody", RoleMaster) {
		t.Fatal("readonly must not pass master")
	}
}

func TestMasterListWins(t *testing.T) {
	setupRolesBucket(t, "roles-master")
	BucketApp.Set("masters", "boss")
	if got := UserRole("boss"); got != RoleMaster {
		t.Fatalf("masters-listed user should be master, got %q", got)
	}
	if !HasRole("boss", RoleOps) || !HasRole("boss", RoleMaster) {
		t.Fatal("master must satisfy every level")
	}
}

func TestConsoleAdminIsMaster(t *testing.T) {
	setupRolesBucket(t, "roles-console-admin")
	if got := UserRole(consoleAdminID); got != RoleMaster {
		t.Fatalf("console admin should be master, got %q", got)
	}
	if !IsConsoleAdmin(" admin ") || IsConsoleAdmin("someone") {
		t.Fatal("IsConsoleAdmin should match only the trimmed admin ID")
	}
}

func TestSetAndDeleteUserRole(t *testing.T) {
	setupRolesBucket(t, "roles-set")
	if err := SetUserRole("ops1", RoleOps); err != nil {
		t.Fatal(err)
	}
	if !HasRole("ops1", RoleOps) {
		t.Fatal("ops user must pass ops")
	}
	if HasRole("ops1", RoleMaster) {
		t.Fatal("ops user must not pass master")
	}
	if err := SetUserRole("bad", "root"); err == nil {
		t.Fatal("invalid role should be rejected")
	}
	if err := DeleteUserRole("ops1"); err != nil {
		t.Fatal(err)
	}
	if got := UserRole("ops1"); got != RoleReadonly {
		t.Fatalf("deleted role should fall back to readonly, got %q", got)
	}
	if err := DeleteUserRole("ops1"); err == nil {
		t.Fatal("deleting a missing role should error")
	}
}

func TestListUserRolesExcludesImplicit(t *testing.T) {
	setupRolesBucket(t, "roles-list")
	if err := SetUserRole("ops1", RoleOps); err != nil {
		t.Fatal(err)
	}
	roles := ListUserRoles()
	if roles["ops1"] != RoleOps {
		t.Fatalf("explicit role missing: %+v", roles)
	}
	if _, ok := roles["ghost"]; ok {
		t.Fatal("implicit users must not appear in the role table")
	}
}

func TestScriptConsoleRoleIsOps(t *testing.T) {
	setupRolesBucket(t, "roles-script")
	if got := UserRole(ScriptConsoleUser); got != RoleOps {
		t.Fatalf("script console role = %q, want %q", got, RoleOps)
	}
	if HasRole(ScriptConsoleUser, RoleMaster) {
		t.Fatal("script console must not inherit master")
	}
	if !HasRole(ScriptConsoleUser, RoleOps) {
		t.Fatal("script console should keep ops for plugin management")
	}
}

func TestAppRoleOverride(t *testing.T) {
	setupRolesBucket(t, "roles-app")
	rolesMu.Lock()
	all := loadAppRoles()
	all["app-a"] = map[string]string{"alice": RoleOps, "bob": RoleReadonly}
	saveAppRoles(all)
	rolesMu.Unlock()
	if got := UserRoleForApp("alice", "app-a"); got != RoleOps {
		t.Fatalf("app role = %q, want %q", got, RoleOps)
	}
	if UserRoleForApp("alice", "app-b") != RoleReadonly {
		t.Fatal("app role must not leak to other apps")
	}
	if got := UserRoleForApp("admin", "app-a"); got != RoleMaster {
		t.Fatalf("console admin should stay master, got %q", got)
	}
}
