package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

const defaultAddress = "127.0.0.1:19081"

type config struct {
	Address      string
	DatabasePath string
	SelfCheck    bool
}

func parseConfig(args []string) (config, error) {
	address := defaultAddress
	if portText := strings.TrimSpace(os.Getenv("PORT")); portText != "" {
		port, err := strconv.Atoi(portText)
		if err != nil || port < 1024 || port > 65535 {
			return config{}, fmt.Errorf("PORT 必须是 1024 到 65535 的端口号")
		}
		address = net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	}
	set := flag.NewFlagSet("timber-stage-qualifier", flag.ContinueOnError)
	set.StringVar(&address, "addr", address, "HTTP 监听地址")
	var databasePath string
	var selfCheck bool
	set.StringVar(&databasePath, "db", "timber-stage.db", "SQLite 数据库路径")
	set.BoolVar(&selfCheck, "self-check", false, "执行真实 HTTP 自检后退出")
	if err := set.Parse(args); err != nil {
		return config{}, err
	}
	if set.NArg() != 0 {
		return config{}, fmt.Errorf("存在未识别的位置参数")
	}
	if err := validateAddress(address); err != nil {
		return config{}, err
	}
	if strings.TrimSpace(databasePath) == "" {
		return config{}, fmt.Errorf("数据库路径不能为空")
	}
	return config{Address: address, DatabasePath: databasePath, SelfCheck: selfCheck}, nil
}

func validateAddress(address string) error {
	host, portText, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("监听地址必须采用 host:port 格式: %w", err)
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("监听地址必须使用明确的回环 IP")
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1024 || port > 65535 {
		return fmt.Errorf("监听端口必须在 1024 到 65535 之间")
	}
	return nil
}
