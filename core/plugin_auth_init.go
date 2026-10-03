package core

func initPluginAuth() {
	if rows := loadAuthTable(); len(rows) > 0 {
		GetMyLog().Info("插件授权表已加载 %d 条", len(rows))
	}
}
