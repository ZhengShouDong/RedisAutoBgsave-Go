# RedisAutoBgsave-Go
Auto background save(generate RDB file) for redis cluster(3.0+) all slave nodes.

## Dependent:
"github.com/go-redis/redis"

You should on your terminal execute `go get github.com/go-redis/redis`
and copy this code
```
func (c *ClusterClient) ForEachSlaveSync(fn func(client *Client) error) error {
	state, err := c.state.ReloadOrGet()
	if err != nil {
		return err
	}

	for _, slave := range state.Slaves {
		err := fn(slave.Client)
		if err != nil {
			return err
		}
	}

	return nil
}
```
to the `go-redis/redis/cluster.go`

## Finally
`cd` to the project dir, on your terminal execute `go build && ./RedisAutoBgsave-Go -host 127.0.0.1 -port 6379`
According to your actual situation, write the password options

Enjoy!
