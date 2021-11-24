package config

type ServiceEndpoints struct {
	UserGraphQL                 string
	PlayStoreUpsertSubscription string
	AppStoreUpsertSubscription  string
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

type NewebPayStore struct {
	CallbackHost        string
	CallbackProtocol    string
	ClientBackPath      string
	ID                  string
	IsAbleToModifyEmail int8 // Use 1
	LoginType           int8 // Use 0
	NotifyProtocol      string
	NotifyHost          string
	NotifyPath          string
	Is3DSecure          int8   // Use 1
	RespondType         string // Use JSON
	ReturnPath          string
	Version             string // Use 1.6
}

type FeatureToggles struct {
	Bucket string
	Object string
	Type   string
}

type Conf struct {
	Address                     string
	FirebaseCredentialFilePath  string
	FirebaseRealtimeDatabaseURL string
	Port                        int
	RedisService                RedisService
	ServiceEndpoints            ServiceEndpoints
	NewebPayStore               NewebPayStore
	TokenSecretName             string
	V0RESTfulSvrTargetURL       string
	FeatureToggles              FeatureToggles
	PrivilegedEmailDomains      map[string]bool
}

func (c *Conf) Valid() bool {
	return true
}
