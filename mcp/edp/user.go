package edp

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/kinwyb/buibuiCodex/mcp/models"
)

// GetUserInfo 使用 token 查询当前登录用户信息。
func GetUserInfo(token string) (*models.UserInfo, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, errors.New("token 为空")
	}

	result, err := DoClient[models.UserInfo](
		NewRequest(http.MethodGet, "/login/userInfo", WithToken(token)),
	)
	if err != nil {
		return nil, fmt.Errorf("获取用户信息失败: %w", err)
	}
	if !result.Data.RequestSucc {
		return nil, errors.New("获取用户信息失败")
	}
	return &result.Data, nil
}
