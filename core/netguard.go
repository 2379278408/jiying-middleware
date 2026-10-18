package core

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// allowPrivateOutbound 允许内网自建部署访问私网地址；云元数据地址始终拒绝。
func allowPrivateOutbound() bool {
	return strings.TrimSpace(os.Getenv("JIYING_ALLOW_PRIVATE_HOSTS")) == "1"
}

// isMetadataIP 判断云厂商元数据地址，这类地址在任何配置下都不可访问。
func isMetadataIP(ip net.IP) bool {
	if ip4 := ip.To4(); ip4 != nil {
		return ip4.Equal(net.IPv4(169, 254, 169, 254)) || ip4.Equal(net.IPv4(100, 100, 100, 200))
	}
	return ip.Equal(net.ParseIP("fd00:ec2::254"))
}

// isPrivateIP 判断回环、私网、链路本地、CGNAT 等不应由服务端主动访问的地址。
func isPrivateIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil {
		if ip4[0] == 0 || ip4[0] == 127 {
			return true
		}
		if ip4[0] == 169 && ip4[1] == 254 {
			return true
		}
		if ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127 {
			return true
		}
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() || ip.IsMulticast() {
		return true
	}
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	return false
}

// validateOutboundURL 校验服务端主动访问的地址：限制协议、禁止 userinfo，
// 并拒绝解析到回环/私网/链路本地/云元数据的地址。allowWS 为真时接受 ws/wss。
func validateOutboundURL(raw string, allowWS bool) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("地址为空")
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return fmt.Errorf("地址格式无效")
	}
	if u.User != nil {
		return fmt.Errorf("地址不能包含账号信息")
	}
	scheme := strings.ToLower(u.Scheme)
	switch scheme {
	case "http", "https":
	case "ws", "wss":
		if !allowWS {
			return fmt.Errorf("地址必须是 http 或 https")
		}
	default:
		return fmt.Errorf("不支持的协议: %s", scheme)
	}
	host := strings.TrimSuffix(strings.ToLower(u.Hostname()), ".")
	if host == "" {
		return fmt.Errorf("地址缺少主机")
	}
	if host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") {
		return fmt.Errorf("不允许访问本地地址")
	}
	ips, err := resolveHostIPs(host)
	if err != nil {
		return err
	}
	for _, ip := range ips {
		if isMetadataIP(ip) {
			return fmt.Errorf("不允许访问元数据地址")
		}
		if isPrivateIP(ip) && !allowPrivateOutbound() {
			return fmt.Errorf("不允许访问内网地址")
		}
	}
	return nil
}

func resolveHostIPs(host string) ([]net.IP, error) {
	if ip := net.ParseIP(host); ip != nil {
		return []net.IP{ip}, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
	if err != nil || len(ips) == 0 {
		return nil, fmt.Errorf("无法解析主机地址")
	}
	return ips, nil
}

// endpointsEqual 判断两个上游地址是否指向同一端点，用于掩码密钥场景绑定已存地址。
func endpointsEqual(a, b string) bool {
	ua, errA := url.Parse(strings.TrimSpace(a))
	ub, errB := url.Parse(strings.TrimSpace(b))
	if errA != nil || errB != nil || ua.Host == "" || ub.Host == "" {
		return strings.TrimRight(strings.TrimSpace(a), "/") == strings.TrimRight(strings.TrimSpace(b), "/")
	}
	if !strings.EqualFold(ua.Scheme, ub.Scheme) {
		return false
	}
	if !strings.EqualFold(ua.Host, ub.Host) {
		return false
	}
	return strings.TrimRight(ua.Path, "/") == strings.TrimRight(ub.Path, "/") && ua.RawQuery == ub.RawQuery
}

// safeOutboundHTTPClient 构造用于下载外部内容的客户端：连接前重新校验解析结果，
// 防止 DNS 重绑定绕过一次性 URL 校验；每次重定向也重新校验目标地址。
func safeOutboundHTTPClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, fmt.Errorf("地址格式无效")
			}
			ips, err := resolveHostIPs(host)
			if err != nil {
				return nil, err
			}
			var lastErr error
			for _, ip := range ips {
				if isMetadataIP(ip) {
					lastErr = fmt.Errorf("不允许访问元数据地址")
					continue
				}
				if isPrivateIP(ip) && !allowPrivateOutbound() {
					lastErr = fmt.Errorf("不允许访问内网地址")
					continue
				}
				conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
				if err == nil {
					return conn, nil
				}
				lastErr = err
			}
			if lastErr == nil {
				lastErr = fmt.Errorf("目标地址不可用")
			}
			return nil, lastErr
		},
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
		MaxIdleConns:          16,
		IdleConnTimeout:       30 * time.Second,
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("跳转次数过多")
			}
			return validateOutboundURL(req.URL.String(), false)
		},
	}
}
