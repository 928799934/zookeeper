package conf

import "time"

// Conf 配置
type Conf struct {
	Addrs   []string      // zk 地址(x.x.x.x:xxxx)
	Timeout time.Duration // session 会话超时时间
	Path    string        // zk 路径
}
