package leader

import (
	"fmt"
	zkConf "github.com/928799934/zookeeper/conf"
	"github.com/samuel/go-zookeeper/zk"
	"strings"
	"sync"
	"time"
)

var (
	// 循环检测 leader身份
	waitTime time.Duration = time.Millisecond * 333
)

// Node 外部对象
type Node interface {
	Close()
	State() <-chan zk.State
}

// node 节点
type node struct {
	c      *zk.Conn        // zk链接
	handle <-chan zk.Event // zk事件
	path   string          // 路径
	acl    []zk.ACL        // 权限
	wg     sync.WaitGroup  // 协程锁
	seq    int             // 当前索引id
	notify chan zk.State   // 外部通知
	leader bool            // leader 标记
}

// NewNode 创建节点
func NewNode(conf *zkConf.Conf) (Node, error) {
	conn, handle, err := zk.Connect(conf.Addrs, conf.Timeout)
	if err != nil {
		DefaultLogger.Printf("zk.Connect(%v,%v) error(%v)", conf.Addrs, conf.Timeout, err)
		return nil, err
	}
	conn.SetLogger(DefaultLogger)

	node := &node{
		c:      conn,
		handle: handle,
		path:   conf.Path,
		acl:    zk.WorldACL(zk.PermAll),
		notify: make(chan zk.State, 1),
	}
	node.wg.Add(1)
	go node.handleStateChange()
	return node, nil
}

// Close 关闭节点
func (node *node) Close() {
	node.c.Close()
	node.wg.Wait()
	close(node.notify)
}

// State 返回节点事件
func (node *node) State() <-chan zk.State {
	return node.notify
}

// handleStateChange 链路事件变化
//  当zk 连接成功后 会触发 StateConnected
//  当zk 连接断开后 会触发 StateDisconnected
//  当 网络突然出现异常断开 又回复后 会触发 StateExpired 会话过期
//  当zk包 内部实现 会话机制 此处 会话过期 为会话机制的 验证处理
//  触发会话过期后 会再触发StateDisconnected 再从新触发 StateConnected
func (node *node) handleStateChange() {
	defer node.wg.Done()
	for event := range node.handle {
		switch event.State {
		case zk.StateConnected: // 链接建立
		case zk.StateExpired: // 会话过期
		case zk.StateHasSession: // 会话建立 单位时间内 会话只会成功建立一次
			// 初始化
			if err := node.ready(); err != nil {
				DefaultLogger.Printf("this.ready() error(%v)", err)
				return
			}
			// 启动工作协成
			node.wg.Add(1)
			go node.run()
		case zk.StateDisconnected: // 链接断开
			// 失去leader身份
			node.leader = false
		}
	}
}

// ready 初始化准备
func (node *node) ready() error {
	// create zk root path
	root := ""
	for _, str := range strings.Split(node.path, "/")[1:] {
		root += "/" + str
		if _, err := node.c.Create(root, []byte{}, 0, node.acl); err != nil {
			if err == zk.ErrNodeExists {
				continue
			}
			DefaultLogger.Printf("this.c.Create(%v, []byte{}, 0, %v) error(%v)", root, node.acl, err)
			return err
		}
	}

	// create zk node path
	prefix := fmt.Sprintf("%s/node-", node.path)
	path, err := node.c.Create(prefix, []byte{}, zk.FlagEphemeral|zk.FlagSequence, node.acl)
	if err != nil {
		DefaultLogger.Printf("this.c.Create(%v, []byte{}, zk.FlagEphemeral|zk.FlagSequence, %v) error(%v)", prefix, node.acl, err)
		return err
	}

	seq, err := parseSeq(path)
	if err != nil {
		DefaultLogger.Printf("parseSeq(%v) error(%v)", path, err)
		return err
	}
	node.seq = seq
	return nil
}

// run 工作协程 协成推出认为是 连接断开
func (node *node) run() {
	defer node.wg.Done()

	// 连接建立成功
	node.notify <- StateConnected
	defer func() {
		// 连接已断开
		node.notify <- StateDisconnected
	}()

	t := time.NewTicker(waitTime)
	defer t.Stop()

	for {
		if err := watchLeader(node); err != nil {
			// 链路终止
			if err == zk.ErrClosing {
				break
			}
			DefaultLogger.Printf("watch(this) error(%v)", err)
			break
		}

		// 拥有leader身份
		node.leader = true

		node.notify <- StateLeader

		for {
			<-t.C
			if !node.leader {
				break
			}
		}

		// 失去leader身份
		node.notify <- StateFollower

		// disconnected ?
		if node.c.State() != zk.StateHasSession {
			break
		}
	}
}
