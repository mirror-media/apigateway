package config

type ServiceEndpoints struct {
	UserGraphQL string
}

type RedisAddress struct {
	Addr string
	Port int
}

type RedisCache struct {
	TTL int
}

// RedisService represents a object of a redis service. If the type is sentinel, the first address is always treated as the master.
type RedisService struct {
	Addresses []RedisAddress // 1. ip:port, 2. dns:port
	Cache     RedisCache
	Password  string
	Type      string // 1. single, 2. sentinel, 3. cluster
}

type Conf struct {
	Address                     string
	FirebaseCredentialFilePath  string
	FirebaseRealtimeDatabaseURL string
	Port                        int
	ProjectID                   string
	PubSubSubscribeMember       string
	PubSubTopicMember           string
	RedisService                RedisService
	ServiceEndpoints            ServiceEndpoints
	TokenSecretName             string
	V0RESTfulSvrTargetURL       string
}

func (c *Conf) Valid() bool {
	return true
}
