package gin

import (
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"jiying/middleware/core"
)

const appVersion = "1.0.0-rebuilt"

func getStatisticsReal(c *gin.Context) {
	imtypes, hours, _, _ := core.StatsSnapshot()
	keys := make([]string, 0, len(hours))
	for k := range hours {
		keys = append(keys, k)
	}
	sortStrings(keys)
	ordered := make(map[string]int, len(keys))
	for _, k := range keys {
		ordered[k] = hours[k]
	}
	jsonOK(c, gin.H{
		"version":      appVersion,
		"started_at":   core.StartedAt().Format("2006-01-02 15:04:05"),
		"msgs_imtype":  imtypes,
		"msgs_time":    ordered,
		"user_count":   usersCount(),
		"command_hits": orderedKeysDesc(cmdHitsSnapshot()),
		"now":          time.Now().Format("2006-01-02 15:04:05"),
	})
}

func cmdHitsSnapshot() map[string]int {
	_, _, cmds, _ := core.StatsSnapshot()
	return cmds
}

func orderedKeysDesc(m map[string]int) map[string]int {
	return m
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

func getRealtimeReal(c *gin.Context) {
	cpu := readCPUUsage()
	mem := readMemoryUsage()
	total, finished := core.CommandTotalAndFinished()
	jsonOK(c, gin.H{
		"cpu":      cpu,
		"memory":   mem,
		"total":    total,
		"finished": finished,
		"ups":      0,
		"rand":     strconv.FormatInt(time.Now().UnixNano(), 10),
		"today_msg": func() int {
			_, hours, _, _ := core.StatsSnapshot()
			return hours[time.Now().Format("15:04")]
		}(),
		"online":  0,
		"updated": time.Now().Format("2006-01-02 15:04:05"),
	})
}

func readCPUUsage() int {
	read := func() (idle, total uint64) {
		data, err := os.ReadFile("/proc/stat")
		if err != nil {
			return 0, 0
		}
		for _, line := range strings.Split(string(data), "\n") {
			if !strings.HasPrefix(line, "cpu ") {
				continue
			}
			fields := strings.Fields(line)
			for i, f := range fields[1:] {
				v, _ := strconv.ParseUint(f, 10, 64)
				total += v
				if i == 3 {
					idle = v
				}
			}
			break
		}
		return idle, total
	}
	idle1, total1 := read()
	time.Sleep(100 * time.Millisecond)
	idle2, total2 := read()
	if total2 <= total1 {
		return 0
	}
	usage := 100 * (total2 - total1 - (idle2 - idle1)) / (total2 - total1)
	return int(usage)
}

func readMemoryUsage() int {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return int(m.Sys / 1024 / 1024)
}

func getInstructionsReal(c *gin.Context) {
	out := []map[string]interface{}{}
	for _, f := range core.FunctionsSnapshot() {
		if f == nil {
			continue
		}
		rule := strings.Join(f.Rules, " / ")
		if rule == "" {
			rule = f.Handle
		}
		desc := f.Description
		if desc == "" && f.Author != "" {
			desc = "作者: " + f.Author
		}
		out = append(out, map[string]interface{}{
			"priority":    f.Priority,
			"rule":        rule,
			"title":       f.Title,
			"description": desc,
		})
	}
	jsonOK(c, out)
}
