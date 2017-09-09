## Zookeeper Leader
This package implements zookeeper leader mode by Go.
## Usage:
---------
```go
package main

import (
	"fmt"
	zkConf "github.com/928799934/zookeeper/conf"
	zkLeader "github.com/928799934/zookeeper/leader"
	"sync"
	"testing"
	"time"
)

var (
	leader bool
	wg     sync.WaitGroup
)

func view() {
	defer wg.Done()
	for leader {
		fmt.Println("i'm leader")
		time.Sleep(time.Second / 2)
	}
	fmt.Println("i'm not leader")
}

func main() {
	conf := &zkConf.Conf{
		[]string{"10.60.82.109:2181", "10.60.82.109:2182", "10.60.82.109:2183"},
		time.Second,
		"/ElectMaster2",
	}
	ele, err := zkLeader.NewNode(conf)
	if err != nil {
		fmt.Printf("connect error(%v)", err)
		return
	}

	go func() {
		time.Sleep(time.Second * 30)
		ele.Close()
	}()

	for {
		e, ok := <-ele.State()
		if !ok {
			fmt.Println("break loop")
			break
		}
		switch e {
		case zkLeader.StateConnected:
		case zkLeader.StateDisconnected:
		case zkLeader.StateLeader:
			leader = true
			wg.Add(1)
			go view()
		case zkLeader.StateFollower:
			leader = false
		}
	}
	wg.Wait()
}
```
