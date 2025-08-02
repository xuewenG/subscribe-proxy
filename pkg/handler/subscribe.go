package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xuewenG/subscribe-proxy/pkg/config"
	"github.com/xuewenG/subscribe-proxy/pkg/metrics"
)

type FileLock struct {
	mu    sync.Mutex
	group sync.WaitGroup
}

var fileLocks sync.Map

func getFileLock(path string) *FileLock {
	lock, _ := fileLocks.LoadOrStore(path, &FileLock{})
	return lock.(*FileLock)
}

func readCachedResponse(cacheFilePath string) (*config.CachedResponse, error) {
	if _, err := os.Stat(cacheFilePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("cache file %s does not exist", cacheFilePath)
	}

	cacheData, err := os.ReadFile(cacheFilePath)
	if err != nil {
		return nil, fmt.Errorf("error reading cache file %s: %v", cacheFilePath, err)
	}

	var cachedResponse config.CachedResponse
	if err := json.Unmarshal(cacheData, &cachedResponse); err != nil {
		return nil, fmt.Errorf("error unmarshaling cache data: %v", err)
	}

	if time.Now().After(cachedResponse.ExpireAt) {
		return nil, fmt.Errorf("cache file %s has expired", cacheFilePath)
	}

	return &cachedResponse, nil
}

func writeCachedResponse(cacheFilePath string, resp *http.Response) error {
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %v", err)
	}

	cachedResponse := config.CachedResponse{
		ExpireAt: time.Now().Add(time.Hour * 2),
		Headers:  make(map[string][]string),
		Body:     bodyBytes,
	}

	for key, values := range resp.Header {
		cachedResponse.Headers[strings.ToLower(key)] = values
	}

	cacheData, err := json.Marshal(cachedResponse)
	if err != nil {
		return fmt.Errorf("error marshaling cache data: %v", err)
	}

	os.MkdirAll(filepath.Dir(cacheFilePath), 0755)
	if err := os.WriteFile(cacheFilePath, cacheData, 0644); err != nil {
		return fmt.Errorf("error writing cache file %s: %v", cacheFilePath, err)
	}

	return nil
}

func sendOriginalRequest(targetUrl string, requestHeaders []config.RequestHeader) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, targetUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	for _, header := range requestHeaders {
		req.Header.Set(header.Name, header.Value)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error requesting Url %s: %v", targetUrl, err)
	}

	return resp, nil
}

func fetchAndCacheResponse(cacheFilePath string, targetUrl string, requestHeaders []config.RequestHeader) error {
	log.Printf("Requesting Original Subscribe: %s", targetUrl)

	resp, err := sendOriginalRequest(targetUrl, requestHeaders)
	if err != nil {
		return fmt.Errorf("error sending original request: %v", err)
	}
	defer resp.Body.Close()

	log.Printf("Response status: %d for Url: %s", resp.StatusCode, targetUrl)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error requesting Url %s: %d", targetUrl, resp.StatusCode)
	}

	if err := writeCachedResponse(cacheFilePath, resp); err != nil {
		return fmt.Errorf("error writing cache file %s: %v", cacheFilePath, err)
	}

	return nil
}

func serveCachedResponse(c *gin.Context, cachedResponse *config.CachedResponse, passResponseHeaders []string) error {
	for _, headerName := range passResponseHeaders {
		if values, exists := cachedResponse.Headers[strings.ToLower(headerName)]; exists {
			for _, value := range values {
				c.Header(headerName, value)
			}
		}
	}

	contentType := "text/plain"
	if values, exists := cachedResponse.Headers["content-type"]; exists && len(values) > 0 {
		contentType = strings.Join(values, ", ")
	}

	c.Data(http.StatusOK, contentType, cachedResponse.Body)
	return nil
}

func readAndServeCachedResponse(c *gin.Context, cacheFilePath string, passResponseHeaders []string) error {
	cachedResponse, err := readCachedResponse(cacheFilePath)
	if err != nil {
		return fmt.Errorf("error reading cached response: %v", err)
	}

	if err := serveCachedResponse(c, cachedResponse, passResponseHeaders); err != nil {
		return fmt.Errorf("error serving cached response: %v", err)
	}

	return nil
}

func selectSubscribe(user *config.User, subscribeGroup string) (*config.SubscribeGroup, *config.Subscribe, error) {
	var group *config.SubscribeGroup
	for _, g := range config.Config.SubscribeGroups {
		if g.Name == subscribeGroup {
			group = &g
			break
		}
	}
	if group == nil {
		return nil, nil, fmt.Errorf("subscribeGroup not found: %s", subscribeGroup)
	}

	var userSubscribeGroup *config.UserSubscribeGroup
	for _, usg := range user.SubscribeGroups {
		if usg.Name == subscribeGroup {
			userSubscribeGroup = &usg
			break
		}
	}
	if userSubscribeGroup == nil {
		return nil, nil, fmt.Errorf("user %s has no permission for subscribeGroup: %s", user.Name, subscribeGroup)
	}

	filteredSubscribes := make([]config.Subscribe, 0)
	for _, subscribe := range group.Subscribes {
		if slices.Contains(userSubscribeGroup.Subscribes, subscribe.Name) {
			filteredSubscribes = append(filteredSubscribes, subscribe)
		}
	}

	if len(filteredSubscribes) == 0 {
		return nil, nil, fmt.Errorf("no allowed subscribes found for user %s in group: %s", user.Name, subscribeGroup)
	}

	selectedSubscribe := filteredSubscribes[0]
	log.Printf("Selected subscribe: %s, URL: %s for user: %s", selectedSubscribe.Name, selectedSubscribe.Url, user.Name)

	return group, &selectedSubscribe, nil
}

func getUser(token string) (*config.User, error) {
	if token == "" {
		return nil, fmt.Errorf("no token provided")
	}

	var user *config.User
	for _, u := range config.Config.Users {
		if u.Token == token {
			user = &u
			break
		}
	}
	if user == nil {
		return nil, fmt.Errorf("invalid token")
	}

	return user, nil
}

func SubscribeProxy(c *gin.Context) {
	token := c.Query("token")
	user, err := getUser(token)
	if err != nil {
		log.Printf("Error getting user: %v", err)
		metrics.RecordInvalidToken(token)
		c.Status(http.StatusForbidden)
		return
	}

	subscribeGroup := c.Query("group")
	if subscribeGroup == "" && user.DefaultSubscribeGroup != "" {
		log.Printf("Using default subscribe group: %s for user: %s", user.DefaultSubscribeGroup, user.Name)
		subscribeGroup = user.DefaultSubscribeGroup
	}

	group, selectedSubscribe, err := selectSubscribe(user, subscribeGroup)
	if err != nil {
		log.Printf("Error selecting subscribe: %v", err)
		metrics.RecordUserSubscribe(user.Name, subscribeGroup, "")
		c.Status(http.StatusForbidden)
		return
	}

	metrics.RecordUserSubscribe(user.Name, subscribeGroup, selectedSubscribe.Name)

	cacheFilePath := filepath.Join(
		config.Config.CacheDir,
		fmt.Sprintf("%s-%s.json", subscribeGroup, selectedSubscribe.Name),
	)

	cachedResponse, err := readCachedResponse(cacheFilePath)
	if err == nil {
		if err := serveCachedResponse(c, cachedResponse, group.PassResponseHeaders); err == nil {
			log.Printf("Served from cache: %s", cacheFilePath)
		} else {
			log.Printf("Error serving cached response from %s: %v", cacheFilePath, err)
		}

		return
	}

	cacheFileLock := getFileLock(cacheFilePath)
	if !cacheFileLock.mu.TryLock() {
		log.Printf("Waiting for cache update: %s", cacheFilePath)
		cacheFileLock.group.Wait()

		if err := readAndServeCachedResponse(c, cacheFilePath, group.PassResponseHeaders); err != nil {
			log.Printf("Error reading and serving cached response from %s after lock: %v", cacheFilePath, err)
			c.Status(http.StatusInternalServerError)
		}

		return
	}

	defer cacheFileLock.mu.Unlock()

	cacheFileLock.group.Add(1)
	defer cacheFileLock.group.Done()

	metrics.RecordCacheUpdate(subscribeGroup, selectedSubscribe.Name)
	if err := fetchAndCacheResponse(cacheFilePath, selectedSubscribe.Url, group.RequestHeaders); err != nil {
		log.Printf("Error fetching and caching response from %s: %v", selectedSubscribe.Url, err)
		c.Status(http.StatusInternalServerError)
		return
	}

	metrics.RecordCacheUpdateSuccess(subscribeGroup, selectedSubscribe.Name)

	if err := readAndServeCachedResponse(c, cacheFilePath, group.PassResponseHeaders); err != nil {
		log.Printf("Error reading and serving cached response from %s after lock: %v", cacheFilePath, err)
		c.Status(http.StatusInternalServerError)
	}
}
