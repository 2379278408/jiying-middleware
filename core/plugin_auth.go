package core

import (
	"encoding/json"
	"sort"
)

type BucketAuth struct {
	BucketName string `json:"bucket_name"`
	BucketRW   bool   `json:"bucket_rw"`
}

type PluginAuthRow struct {
	PluginName   string       `json:"plugin_name"`
	PluginAuthor string       `json:"plugin_author"`
	Origin       bool         `json:"origin"`
	Auth         []BucketAuth `json:"auth"`
}

var authBuckets = []string{"middleware", "jiyingUsers"}

const pluginAuthBucket = "PluginAuthTable"

func loadAuthTable() []PluginAuthRow {
	raw := BucketApp.GetString(pluginAuthBucket)
	if raw == "" {
		return nil
	}
	var rows []PluginAuthRow
	if json.Unmarshal([]byte(raw), &rows) != nil {
		return nil
	}
	return rows
}

func saveAuthTable(rows []PluginAuthRow) {
	data, err := json.Marshal(rows)
	if err != nil {
		return
	}
	_ = BucketApp.Set(pluginAuthBucket, string(data))
}

func GetPluginAuthTable() []PluginAuthRow {
	saved := loadAuthTable()
	idx := map[string]int{}
	for i, r := range saved {
		idx[r.PluginName] = i
	}
	var rows []PluginAuthRow
	seen := map[string]bool{}
	for _, f := range FunctionsSnapshot() {
		if f == nil || f.Origin {
			continue
		}
		if i, ok := idx[f.Title]; ok {
			rows = append(rows, saved[i])
			seen[f.Title] = true
			continue
		}
		auth := make([]BucketAuth, 0, len(authBuckets))
		for _, b := range authBuckets {
			auth = append(auth, BucketAuth{BucketName: b, BucketRW: false})
		}
		rows = append(rows, PluginAuthRow{
			PluginName:   f.Title,
			PluginAuthor: f.Author,
			Origin:       f.Origin,
			Auth:         auth,
		})
		seen[f.Title] = true
	}
	for _, r := range saved {
		if !seen[r.PluginName] {
			rows = append(rows, r)
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].PluginName < rows[j].PluginName })
	return rows
}

func SetPluginAuth(name, author string, origin bool, bucket string, rw bool) {
	rows := loadAuthTable()
	for i := range rows {
		if rows[i].PluginName == name && rows[i].Origin == origin {
			rows[i].PluginAuthor = author
			for j := range rows[i].Auth {
				if rows[i].Auth[j].BucketName == bucket {
					rows[i].Auth[j].BucketRW = rw
					saveAuthTable(rows)
					return
				}
			}
			rows[i].Auth = append(rows[i].Auth, BucketAuth{BucketName: bucket, BucketRW: rw})
			saveAuthTable(rows)
			return
		}
	}
	rows = append(rows, PluginAuthRow{
		PluginName:   name,
		PluginAuthor: author,
		Origin:       origin,
		Auth:         []BucketAuth{{BucketName: bucket, BucketRW: rw}},
	})
	saveAuthTable(rows)
}

func GetPluginAuthPermission(name, author string, origin bool, bucket string) bool {
	for _, r := range loadAuthTable() {
		if r.PluginName == name && r.Origin == origin {
			for _, a := range r.Auth {
				if a.BucketName == bucket {
					return a.BucketRW
				}
			}
			return true
		}
	}
	return true
}
