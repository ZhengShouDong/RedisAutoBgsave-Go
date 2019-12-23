package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis"
)

var (
	_host          string
	_port          int
	_password      string
	_clusterClient *redis.ClusterClient
)

func main() {
	flag.StringVar(&_host, "h", "", "Host")
	flag.IntVar(&_port, "p", 0, "Port")
	flag.StringVar(&_password, "pwd", "", "Password")
	flag.Parse()

	if len(_host) == 0 || _port == 0 {
		flag.Usage()
		return
	}

	initClusterClient([]string{getAddr(_host, _port)}, _password)
	defer _clusterClient.Close()

	slaveClients := getAllSlaveNodes()

	for _, client := range slaveClients {
		fmt.Println("------------------" + client.GetAddr() + "(slave)------------------")
		waitBgsaveFinish(client)
		if bgsaveSuccess, err := bgsave(client); err != nil || !bgsaveSuccess {
			fmt.Println("Background saving failed.", err)
			os.Exit(-1)
		}

		fmt.Println("Background saving started.")
		waitBgsaveFinish(client)
		time.Sleep(time.Second * 5)
	}

	fmt.Println("------------------------------------")
	fmt.Println(strconv.Itoa(len(slaveClients)) + " slave nodes was completed!")
}

func getAllSlaveNodes() []*redis.Client {
	clients := []*redis.Client{}

	_clusterClient.ForEachSlaveSync(func(client *redis.Client) error {
		clients = append(clients, client)
		return nil
	})

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

func bgsave(client *redis.Client) (bool, error) {
	result, err := client.BgSave().Result()
	if err != nil {
		return false, err
	}

	return result == "Background saving started", nil
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
