package gin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"jiying/middleware/core"
	"jiying/middleware/utils"
)

func jsonUnmarshal(b []byte, v interface{}) error { return json.Unmarshal(b, v) }

func initMisc2() {
	g := core.Server.Group("/")
	g.Use(AuthRequired())

	g.GET("/scripts/files", scriptFiles)
	g.GET("/skills/files", skillFiles)

	g.GET("/pluginAuth", func(c *gin.Context) { jsonOK(c, core.GetPluginAuthTable()) })
	g.POST("/pluginAuth", requireConsoleRole(core.RoleOps), postPluginAuth)
}

func jsonOKNil(c *gin.Context) { jsonOK(c, nil) }

func jsonFailMsg(c *gin.Context, msg string) {
	jsonErr(c, msg)
}

func logResp(c *gin.Context, kind string) {
	jsonOK(c, kind)
}

func initCloudJS() {
	g := core.Server.Group("/js")
	g.Use(AuthRequired())

	g.GET("/encrypt", func(c *gin.Context) { jsonOK(c, nil) })
	g.GET("/offline", func(c *gin.Context) { jsonOK(c, nil) })
	g.GET("/upload", func(c *gin.Context) { jsonOK(c, nil) })
}

type fileTreeNode struct {
	Name        string          `json:"name"`
	IsDirectory bool            `json:"is_directory"`
	Path        string          `json:"path,omitempty"`
	Children    []*fileTreeNode `json:"children,omitempty"`
}

func scriptFiles(c *gin.Context) {
	jsonOK(c, buildFileTree(scriptsRoot()))
}

func skillFiles(c *gin.Context) {
	jsonOK(c, buildFileTree(filepath.Join(utils.GetResHome(), "skills")))
}

func scriptsRoot() string {
	return filepath.Join(utils.GetResHome(), "scripts")
}

func buildFileTree(root string) []*fileTreeNode {
	rootChildren := []*fileTreeNode{}
	dirs := map[string]*fileTreeNode{}

	dirChildren := func(dirPath string) *[]*fileTreeNode {
		if dirPath == "" {
			return &rootChildren
		}
		parent := &rootChildren
		acc := ""
		for _, d := range strings.Split(dirPath, "/") {
			if acc == "" {
				acc = d
			} else {
				acc = acc + "/" + d
			}
			if n, ok := dirs[acc]; ok {
				parent = &n.Children
				continue
			}
			nd := &fileTreeNode{Name: d, IsDirectory: true}
			*parent = append(*parent, nd)
			dirs[acc] = nd
			parent = &nd.Children
		}
		return parent
	}

	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || path == root || info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(info.Name()))
		if ext != ".js" && ext != ".py" && ext != ".ts" && ext != ".mjs" && ext != ".json" && ext != ".md" {
			return nil
		}
		rel := strings.TrimPrefix(path, root+"/")
		dir := strings.Split(rel, "/")
		mount := dirChildren(strings.Join(dir[:len(dir)-1], "/"))
		*mount = append(*mount, &fileTreeNode{
			Name: info.Name(),
			Path: strings.Join(dir[:len(dir)-1], "/"),
		})
		return nil
	})
	sortNodes(rootChildren)
	return rootChildren
}

func sortNodes(nodes []*fileTreeNode) {
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].IsDirectory != nodes[j].IsDirectory {
			return nodes[i].IsDirectory
		}
		return nodes[i].Name < nodes[j].Name
	})
	for _, n := range nodes {
		if n.Children != nil {
			sortNodes(n.Children)
		}
	}
}

func postPluginAuth(c *gin.Context) {
	name := strings.TrimSpace(c.PostForm("plugin_name"))
	origin := c.PostForm("origin") == "true"
	bucket := strings.TrimSpace(c.PostForm("bucket_name"))
	action := strings.TrimSpace(c.PostForm("action"))
	if name == "" || bucket == "" {
		jsonFailMsg(c, "参数错误")
		return
	}
	core.SetPluginAuth(name, strings.TrimSpace(c.PostForm("plugin_author")), origin, bucket, action == "true")
	jsonOK(c, nil)
}
