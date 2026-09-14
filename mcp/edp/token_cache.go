package edp

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kinwyb/buibuiCodex/mcp/models"
)

var tokenCache *TokenCache

// TokenCacheEntry 缓存中的单个 token 条目。
type TokenCacheEntry struct {
	Token     string    `json:"token"`
	CachedAt  time.Time `json:"cached_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// TokenFileStore 文件缓存的 JSON 结构，key 为 userID。
type TokenFileStore map[string]TokenCacheEntry

// TokenCache 提供双层 token 缓存（内存 + 文件）。
type TokenCache struct {
	mu       sync.RWMutex
	cache    map[string]*TokenCacheEntry
	filePath string
	userMap  map[string]string
}

// NewTokenCache 创建 TokenCache 实例。
// cachePath 为空时使用默认路径 ./token-cache.json。
func NewTokenCache(cachePath string, userMap map[string]string) (*TokenCache, error) {
	if cachePath == "" {
		execDir, err := filepath.Abs(".")
		if err != nil {
			return nil, fmt.Errorf("获取当前执行目录失败: %w", err)
		}
		cachePath = filepath.Join(execDir, "token-cache.json")
	}

	tc := &TokenCache{
		cache:    make(map[string]*TokenCacheEntry),
		filePath: cachePath,
		userMap:  userMap,
	}

	if err := tc.loadFromFile(); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			// 文件损坏视为空缓存，不影响后续登录
		}
	}
	tokenCache = tc
	return tc, nil
}

// GetToken 获取有效 token。优先使用缓存，缓存无效时调用登录接口。
func (tc *TokenCache) GetToken(userID string) (string, error) {
	// 1. 检查内存缓存
	tc.mu.RLock()
	entry, ok := tc.cache[userID]
	tc.mu.RUnlock()
	if ok {
		if tc.validateToken(entry.Token) {
			return entry.Token, nil
		}
	}

	// 2. 从文件加载缓存
	if err := tc.loadFromFile(); err == nil {
		tc.mu.RLock()
		entry, ok = tc.cache[userID]
		tc.mu.RUnlock()
		if ok {
			if tc.validateToken(entry.Token) {
				return entry.Token, nil
			}
		}
	}

	// 3. 调用登录接口获取新 token
	token, err := tc.login(userID)
	if err != nil {
		return "", err
	}

	// 4. 写入缓存并持久化
	tc.mu.Lock()
	entry = &TokenCacheEntry{
		Token:     token,
		CachedAt:  time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	tc.cache[userID] = entry
	tc.mu.Unlock()

	_ = tc.saveToFile()

	return token, nil
}

// validateToken 调用 /login/userInfo 验证 token 是否有效。
func (tc *TokenCache) validateToken(token string) bool {

	req := NewRequest(http.MethodGet, "/login/userInfo", WithToken(token))
	result, err := DoClient[models.UserInfo](req)
	if err != nil {
		return false
	}

	return result.Data.Token != "" && result.Data.UserID != ""
}

// login 调用 /login 接口获取新 token。
func (tc *TokenCache) login(userID string) (string, error) {
	sec, err := secret(userID)
	if err != nil {
		return "", fmt.Errorf("登录失败: %w", err)
	}
	req := NewRequest(http.MethodPost, "/login/secretLogin", WithQueryParams(map[string]string{
		"username": userID,
		"secret":   sec,
	}))
	result, err := DoClient[string](req)
	if err != nil {
		return "", fmt.Errorf("登录失败: %w", err)
	}

	token := strings.TrimSpace(result.Data)
	// 去除 JSON 字符串的引号
	if len(token) >= 2 && token[0] == '"' && token[len(token)-1] == '"' {
		token = token[1 : len(token)-1]
	}
	if token == "" {
		return "", errors.New("登录返回空 token")
	}
	return token, nil
}

// Invalidate 从内存和文件缓存中移除指定用户的 token。
func (tc *TokenCache) Invalidate(userID string) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	delete(tc.cache, userID)
	_ = tc.saveToFile()
}

// InvalidateAll 清空所有缓存。
func (tc *TokenCache) InvalidateAll() {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	tc.cache = make(map[string]*TokenCacheEntry)
	_ = tc.saveToFile()
}

// loadFromFile 从 JSON 文件加载缓存到内存。
func (tc *TokenCache) loadFromFile() error {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	data, err := os.ReadFile(tc.filePath)
	if err != nil {
		return err
	}

	var store TokenFileStore
	if err := json.Unmarshal(data, &store); err != nil {
		return fmt.Errorf("解析缓存文件失败: %w", err)
	}

	tc.cache = make(map[string]*TokenCacheEntry)
	for userID, entry := range store {
		e := entry
		tc.cache[userID] = &e
	}

	return nil
}

// saveToFile 将内存缓存保存到 JSON 文件。
func (tc *TokenCache) saveToFile() error {
	dir := filepath.Dir(tc.filePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("创建缓存目录失败: %w", err)
	}

	store := make(TokenFileStore)
	for userID, entry := range tc.cache {
		store[userID] = *entry
	}

	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化缓存数据失败: %w", err)
	}

	if err := os.WriteFile(tc.filePath, data, 0600); err != nil {
		return fmt.Errorf("写入缓存文件失败: %w", err)
	}

	return nil
}

// 用户转换
func (tc *TokenCache) userConvert(userID string) string {
	if tc.userMap == nil {
		return userID
	}
	if newid, ok := tc.userMap[userID]; ok {
		return newid
	}
	return userID
}

// GetToken 获取token
func GetToken(userID string) (string, error) {
	if tokenCache == nil {
		cache, err := NewTokenCache("", nil)
		if err != nil {
			return "", err
		}
		tokenCache = cache
	}
	userID = tokenCache.userConvert(userID)
	return tokenCache.GetToken(userID)
}

// secret 登陆加密
func secret(userID string) (string, error) {
	tnow := time.Now()
	data := fmt.Sprintf("%d_%s", time.Now().Unix(), userID)
	ps, _ := strconv.Atoi(tnow.UTC().Format("200601"))
	vi := fmt.Sprintf("7F3FCDC334%d", ps+797000)
	encryptoData, e := AESCBCEncrypt([]byte(data), []byte(vi[0:16]))
	if e != nil {
		return "", fmt.Errorf("加密失败 %w", e)
	}
	return hex.EncodeToString(encryptoData), nil
}
