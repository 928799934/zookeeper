package leader

import "github.com/samuel/go-zookeeper/zk"

const (
	StateConnected    zk.State = 100
	StateDisconnected zk.State = 0
	StateLeader       zk.State = 1000
	StateFollower     zk.State = 1001
)
