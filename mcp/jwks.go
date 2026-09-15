package mcp

import (
	"crypto/rsa"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"
)

// JWKS 中的单个密钥，仅解析本项目用到的 RSA 字段。
type jwk struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type jwkSet struct {
	Keys []jwk `json:"keys"`
}

// keyStore 从授权服务器的 JWKS 端点拉取并缓存 RSA 公钥。
// 缓存是为了避免每个请求都去请求 JWKS；kid 未命中时会强制刷新一次，
// 以便授权服务器轮换密钥后能自动跟上。
type keyStore struct {
	jwksURL string
	client  *http.Client
	ttl     time.Duration

	mu      sync.RWMutex
	keys    map[string]*rsa.PublicKey
	fetched time.Time
}

func newKeyStore(jwksURL string) *keyStore {
	return &keyStore{
		jwksURL: jwksURL,
		client: &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				// 内网授权服务器使用自签证书，这里跳过校验。
				// 生产环境请改为校验受信任的 CA（tls.Config{RootCAs: pool}）。
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
		ttl:  15 * time.Minute,
		keys: make(map[string]*rsa.PublicKey),
	}
}

// Key 按 kid 返回对应的 RSA 公钥。
func (s *keyStore) Key(kid string) (*rsa.PublicKey, error) {
	s.mu.RLock()
	cached, hit := s.keys[kid]
	fresh := time.Since(s.fetched) < s.ttl
	s.mu.RUnlock()

	if hit && fresh {
		return cached, nil
	}

	if err := s.refresh(); err != nil {
		// 刷新失败时，若手上有旧值则降级使用，避免授权服务器短暂抖动导致全部请求失败。
		if hit {
			return cached, nil
		}
		return nil, err
	}

	s.mu.RLock()
	pub, ok := s.keys[kid]
	s.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("jwks: 未找到 kid=%q 对应的公钥", kid)
	}
	return pub, nil
}

func (s *keyStore) refresh() error {
	resp, err := s.client.Get(s.jwksURL)
	if err != nil {
		return fmt.Errorf("jwks: 请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("jwks: 响应状态异常 %d", resp.StatusCode)
	}

	var set jwkSet
	if err := json.NewDecoder(resp.Body).Decode(&set); err != nil {
		return fmt.Errorf("jwks: 解析失败: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey, len(set.Keys))
	for _, k := range set.Keys {
		if k.Kty != "RSA" {
			continue
		}
		pub, err := k.rsaPublicKey()
		if err != nil {
			continue
		}
		keys[k.Kid] = pub
	}
	if len(keys) == 0 {
		return errors.New("jwks: 没有可用的 RSA 公钥")
	}

	s.mu.Lock()
	s.keys = keys
	s.fetched = time.Now()
	s.mu.Unlock()
	return nil
}

// rsaPublicKey 把 JWK 的 n / e 还原成 rsa.PublicKey（RFC 7517 附录 A）。
func (k jwk) rsaPublicKey() (*rsa.PublicKey, error) {
	nb, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		return nil, fmt.Errorf("jwks: modulus 解码失败: %w", err)
	}
	eb, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		return nil, fmt.Errorf("jwks: exponent 解码失败: %w", err)
	}
	e := 0
	for _, b := range eb {
		e = e<<8 | int(b)
	}
	if e == 0 {
		return nil, errors.New("jwks: exponent 为空")
	}
	return &rsa.PublicKey{N: new(big.Int).SetBytes(nb), E: e}, nil
}
