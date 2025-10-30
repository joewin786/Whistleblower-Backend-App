package notifications

import "github.com/pusher/pusher-http-go/v5"

var Client = pusher.Client{
	AppID:   "2070740",
	Key:     "239d12869d6500d25b16",
	Secret:  "36b74312e5c93fac7a09",
	Cluster: "ap1", // misalnya "ap1"
	Secure:  true,
}
