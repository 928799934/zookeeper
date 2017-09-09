package leader

import (
	"fmt"
	"github.com/samuel/go-zookeeper/zk"
)

// watchLeader 监听leader
func watchLeader(node *node) error {
	for {
		children, _, err := node.c.Children(node.path)
		if err != nil {
			DefaultLogger.Printf("e.c.Children(%v) error(%v)", node.path, err)
			return err
		}
		// children = [node-0000000152 node-0000000151 node-0000000150]
		// 倒叙检查, 监听比自己小的最大的node
		seq := node.seq
		for i := len(children) - 1; i > 0; i-- {
			p := children[i]
			s, err := parseSeq(p)
			if err != nil {
				DefaultLogger.Printf("parseSeq(%v) error(%v)", p, err)
				return err
			}
			if s < node.seq {
				seq = s
				continue
			}
			break
		}

		// leader ?
		if seq == node.seq {
			break
		}

		DefaultLogger.Printf("\nleader:%v \t my:%v \n", seq, node.seq)

		// watch leader
		path := fmt.Sprintf("%s/node-%.10d", node.path, seq)
		_, _, ch, err := node.c.GetW(path)
		if err != nil {
			DefaultLogger.Printf("this.c.GetW(%v) error(%v)", path, err)
			return err
		}

		for {
			event := <-ch
			// Disconnected Server
			if event.State == zk.StateDisconnected {
				return event.Err
			}

			// leader dead
			if event.Type == zk.EventNodeDeleted {
				break
			}
		}
	}
	return nil
}
