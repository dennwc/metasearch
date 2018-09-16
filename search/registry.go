package search

import "context"

type ServiceFunc func(ctx context.Context) (Service, error)

var registry = make(map[string]ServiceFunc)

func RegisterService(name string, fnc ServiceFunc) {
	registry[name] = fnc
}

func ListServices() []ServiceFunc {
	var arr []ServiceFunc
	for _, fnc := range registry {
		arr = append(arr, fnc)
	}
	return arr
}
