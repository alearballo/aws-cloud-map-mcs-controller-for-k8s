package cloudmap

import (
	"errors"
	"fmt"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/common"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/model"
	"k8s.io/apimachinery/pkg/util/cache"
	"time"
)

const (
	nsKeyPrefix    = "ns"
	svcKeyPrefix   = "svc"
	endptKeyPrefix = "endpt"

	defaultCacheSize = 1024
	defaultNsTTL     = 2 * time.Minute
	defaultSvcTTL    = 2 * time.Minute
	defaultEndptTTL  = 5 * time.Second
)

type ServiceDiscoveryClientCache interface {
	GetNamespace(namespaceName string) (namespace *model.Namespace, found bool)
	CacheNamespace(namespace *model.Namespace)
	CacheNilNamespace(namespaceName string)
	GetServiceId(namespaceName string, serviceName string) (serviceId string, found bool)
	CacheServiceId(namespaceName string, serviceName string, serviceId string)
	GetEndpoints(namespaceName string, serviceName string) (endpoints []*model.Endpoint, found bool)
	CacheEndpoints(namespaceName string, serviceName string, endpoints []*model.Endpoint)
	EvictEndpoints(namespaceName string, serviceName string)
}

type sdCache struct {
	log    common.Logger
	cache  *cache.LRUExpireCache
	config *SdCacheConfig
}

type SdCacheConfig struct {
	NsTTL    time.Duration
	SvcTTL   time.Duration
	EndptTTL time.Duration
}

func NewServiceDiscoveryClientCache(cacheConfig *SdCacheConfig) ServiceDiscoveryClientCache {
	return &sdCache{
		log:    common.NewLogger("cloudmap"),
		cache:  cache.NewLRUExpireCache(defaultCacheSize),
		config: cacheConfig,
	}
}

func NewDefaultServiceDiscoveryClientCache() ServiceDiscoveryClientCache {
	return NewServiceDiscoveryClientCache(
		&SdCacheConfig{
			NsTTL:    defaultNsTTL,
			SvcTTL:   defaultSvcTTL,
			EndptTTL: defaultEndptTTL,
		})
}

func (sdCache *sdCache) GetNamespace(nsName string) (ns *model.Namespace, found bool) {
	key := sdCache.buildNsKey(nsName)
	entry, exists := sdCache.cache.Get(key)
	if !exists {
		return nil, false
	}

	if entry == nil {
		return nil, true
	}

	nsEntry, ok := entry.(model.Namespace)
	if !ok {
		sdCache.log.Error(errors.New("failed to retrieve namespace from cache"), "", "nsName", nsName)
		sdCache.cache.Remove(key)
		return nil, false
	}

	return &nsEntry, true
}

func (sdCache *sdCache) CacheNamespace(namespace *model.Namespace) {
	key := sdCache.buildNsKey(namespace.Name)
	sdCache.cache.Add(key, *namespace, sdCache.config.NsTTL)
}

func (sdCache *sdCache) CacheNilNamespace(nsName string) {
	key := sdCache.buildNsKey(nsName)
	sdCache.cache.Add(key, nil, sdCache.config.NsTTL)
}

func (sdCache *sdCache) GetServiceId(nsName string, svcName string) (svcId string, found bool) {
	key := sdCache.buildSvcKey(nsName, svcName)
	entry, exists := sdCache.cache.Get(key)
	if !exists {
		return "", false
	}

	svcId, ok := entry.(string)
	if !ok {
		sdCache.log.Error(errors.New("failed to retrieve service ID from cache"), "",
			"nsName", nsName, "svcName", svcName)
		sdCache.cache.Remove(key)
		return "", false
	}

	return svcId, true
}

func (sdCache *sdCache) CacheServiceId(nsName string, svcName string, svcId string) {
	key := sdCache.buildSvcKey(nsName, svcName)
	sdCache.cache.Add(key, svcId, sdCache.config.SvcTTL)
}

func (sdCache *sdCache) GetEndpoints(nsName string, svcName string) (endpts []*model.Endpoint, found bool) {
	key := sdCache.buildEndptsKey(nsName, svcName)
	entry, exists := sdCache.cache.Get(key)
	if !exists {
		return nil, false
	}

	endpts, ok := entry.([]*model.Endpoint)
	if !ok {
		sdCache.log.Error(errors.New("failed to retrieve endpoints from cache"), "",
			"ns", "nsName", "svc", svcName)
		sdCache.cache.Remove(key)
		return nil, false
	}

	return endpts, true
}

func (sdCache *sdCache) CacheEndpoints(nsName string, svcName string, endpts []*model.Endpoint) {
	key := sdCache.buildEndptsKey(nsName, svcName)
	sdCache.cache.Add(key, endpts, sdCache.config.EndptTTL)
}

func (sdCache *sdCache) EvictEndpoints(nsName string, svcName string) {
	key := sdCache.buildEndptsKey(nsName, svcName)
	sdCache.cache.Remove(key)
}

func (sdCache *sdCache) buildNsKey(nsName string) (cacheKey string) {
	return fmt.Sprintf("%s:%s", nsKeyPrefix, nsName)
}

func (sdCache *sdCache) buildSvcKey(nsName string, svcName string) (cacheKey string) {
	return fmt.Sprintf("%s:%s:%s", svcKeyPrefix, nsName, svcName)
}

func (sdCache *sdCache) buildEndptsKey(nsName string, svcName string) string {
	return fmt.Sprintf("%s:%s:%s", endptKeyPrefix, nsName, svcName)
}
