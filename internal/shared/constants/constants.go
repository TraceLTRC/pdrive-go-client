package constants

type ConfigKey string

const (
	TOKEN               ConfigKey = "token"
	API_URL             ConfigKey = "api_url"
	CONCURRENT_REQUESTS ConfigKey = "concurrent_requests"
)

const SPLIT_SIZE = 50 * 1024 * 1024 //50 MB
