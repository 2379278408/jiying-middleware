package core

import (
	"io"
	"os"
	"strings"
)

// maxLogTailBytes 限制读取日志文件尾部时最多读取的字节数，
// 避免为取最后几百行而把整天的日志文件读进内存。
const maxLogTailBytes = 2 << 20 // 2 MiB

// readTailLines 从文件尾部读取最多 tail 行；只扫描末尾 maxBytes 字节，
// 若从中间截断则丢弃首行残段，保证返回的每一行都完整。
func readTailLines(path string, tail, maxBytes int) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	size := info.Size()
	start := int64(0)
	if maxBytes > 0 && size > int64(maxBytes) {
		start = size - int64(maxBytes)
	}
	if _, err := f.Seek(start, io.SeekStart); err != nil {
		return nil, err
	}
	buf := make([]byte, size-start)
	if _, err := io.ReadFull(f, buf); err != nil && err != io.ErrUnexpectedEOF {
		return nil, err
	}
	text := string(buf)
	// 从文件中间开始时，首行是被截断的残段，直接丢弃。
	if start > 0 {
		if idx := strings.IndexByte(text, '\n'); idx >= 0 {
			text = text[idx+1:]
		} else {
			text = ""
		}
	}
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return []string{}, nil
	}
	if tail > 0 && len(lines) > tail {
		lines = lines[len(lines)-tail:]
	}
	return lines, nil
}
