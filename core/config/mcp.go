package config

import "strings"

type MCPServerConfig struct {
	Name      string            `description:"mcp名称" json:"name"`
	WorkSpace string            `description:"工作区" json:"-"`
	Port      int               `description:"监听地址" json:"port"`
	Version   string            `description:"mcp服务版本" json:"version"`
	BaseURL   string            `description:"基础信息" json:"baseURL"`
	UserMap   map[string]string `description:"用户映射" json:"userMap"`
	Host      string            `description:"公共路径" json:"host"`
	Issuer    string            `description:"授权服务器issuer" json:"issuer"`
	Audience  string            `description:"资源标识(令牌audience)" json:"audience"`
	JWKSURL   string            `description:"授权服务器JWKS地址" json:"jwksURL"`
}

func (m *MCPServerConfig) SetDefaults() {
	if m.Name == "" {
		m.Name = "buibui"
	}
	if m.Port <= 0 {
		m.Port = 9090
	}
	if m.Version == "" {
		m.Version = "1.0.0"
	}
	if m.BaseURL == "" {
		m.BaseURL = "http://localhost"
	}
	if m.Host == "" {
		m.Host = "http://127.0.0.1:9090"
	}
	if m.Issuer == "" {
		m.Issuer = "https://localhost"
	}
	if m.Audience == "" {
		m.Audience = m.Host
	}
	if m.JWKSURL == "" {
		m.JWKSURL = strings.TrimSuffix(m.Issuer, "/") + "/.well-known/jwks.json"
	}
}
