package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"./redis"
)

var (
	_host          string
	_port          int
	_password      string
	_isRewriteAOF  bool
	_isMaster      bool
	_clusterClient *redis.ClusterClient
)

func main() {
	flag.StringVar(&_host, "h", "", "Host")
	flag.IntVar(&_port, "p", 0, "Port")
	flag.StringVar(&_password, "pwd", "", "Password(default: empty)")
	flag.BoolVar(&_isRewriteAOF, "aof", false, "Background rewrite AOF(default: false)")
	flag.BoolVar(&_isMaster, "master", false, "Execute on all master nodes(default: false)")
	flag.Parse()

	if len(_host) == 0 || _port == 0 {
		flag.Usage()
		return
	}

	initClusterClient([]string{getAddr(_host, _port)}, _password)
	defer _clusterClient.Close()

	allNodes := getAllNodes(_isMaster)
	completeCount := 0
	for _, node := range allNodes {
		fmt.Println("------------------" + node.GetAddr() + "(slave)------------------")
		waitBgsaveFinish(node)
		if bgsaveSuccess, err := startJob(node); err != nil || !bgsaveSuccess {
			fmt.Println("Background saving failed.", err)
			return
		}

		fmt.Println("Background saving started.")
		waitBgsaveFinish(node)
		completeCount++
		time.Sleep(time.Second * 5)
	}

	fmt.Println("------------------------------------")
	fmt.Println(strconv.Itoa(len(allNodes)) + " slave nodes has been completed!")
}

func getAllNodes(isMaster bool) []*redis.Client {
	clients := []*redis.Client{}

	if !isMaster {
		_clusterClient.ForEachSlaveSync(func(client *redis.Client) error {
			clients = append(clients, client)
			return nil
		})
	} else {
		_clusterClient.ForEachMasterSync(func(client *redis.Client) error {
			clients = append(clients, client)
			return nil
		})
	}

	if len(clients) == 0 {
		fmt.Println("No slave nodes!")
		os.Exit(-1)
	}

	return clients
}

func waitBgsaveFinish(client *redis.Client) {
	inProgress, err := getBgsaveInProgress(client)
	if err != nil {
		panic(err)
	}

	if !inProgress {
		return
	}

	for {
		time.Sleep(time.Second * 5)
		fmt.Println("check status...")
		inProgress, err = getBgsaveInProgress(client)
		if !inProgress {
			fmt.Println("finished!")
			break
		}
	}
}

func getBgsaveInProgress(client *redis.Client) (bool, error) {
	info, err := client.Info("Persistence").Result()
	if err != nil {
		return false, err
	}

	tag := "rdb_bgsave_in_progress:"
	startIndex := strings.Index(info, tag) + len(tag)
	return info[startIndex:startIndex+1] == "1", nil
}

func startJob(client *redis.Client) (bool, error) {
	if !_isRewriteAOF {
		result, err := client.BgSave().Result()
		if err != nil {
			return false, err
		}

		return result == "Background saving started", nil
	}

	result, err := client.BgRewriteAOF().Result()
	if err != nil {
		return false, err
	}

	return result == "Background append only file rewriting started", nil
}

func initClusterClient(addrs []string, password string) {
	fmt.Println("Cluster client initing..")

	_clusterClient = redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:        addrs,
		Password:     password,
		PoolSize:     32,
		MinIdleConns: 6,
	})

	pong, err := _clusterClient.Ping().Result()
	if err != nil {
		panic(err)
	}

	if pong != "PONG" {
		fmt.Println(pong)
		os.Exit(-1)
	}
}

func getAddr(host string, port int) string {
	return host + ":" + strconv.Itoa(port)
}
