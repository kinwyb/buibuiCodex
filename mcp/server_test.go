package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// API 响应的数据结构（以 Open-Meteo 等公开 API 为例，此处定义通用格式）
type WeatherResponse struct {
	City string `json:"city"`

	Temperature float64 `json:"temperature_celsius"`
	Condition   string  `json:"condition"`
	Humidity    int     `json:"humidity_percent"`
	WindSpeed   float64 `json:"wind_speed_kmh"`
}

// 模拟或调用外部 API 获取天气数据
func fetchWeatherData(city string) (*WeatherResponse, error) {
	//// 示例使用 wttr.in 的 JSON 格式接口
	//apiURL := fmt.Sprintf("https://wttr.in/%s?format=j1", url.QueryEscape(city))
	//
	//client := &http.Client{Timeout: 5 * time.Second}
	//req, err := http.NewRequest("GET", apiURL, nil)
	//if err != nil {
	//	return nil, fmt.Errorf("创建请求失败: %w", err)
	//}
	//// 设置 User-Agent 防止被部分开放接口拦截
	//req.Header.Set("User-Agent", "Go-MCP-Weather-Client/1.0")
	//
	//resp, err := client.Do(req)
	//if err != nil {
	//	return nil, fmt.Errorf("请求外部 API 失败: %w", err)
	//}
	//defer resp.Body.Close()
	//
	//if resp.StatusCode != http.StatusOK {
	//	return nil, fmt.Errorf("外部 API 返回错误代码: %d", resp.StatusCode)
	//}
	//
	//body, err := io.ReadAll(resp.Body)
	//if err != nil {
	//	return nil, fmt.Errorf("读取响应数据失败: %w", err)
	//}
	//
	//// 解析 wttr.in 返回的 JSON 结构
	//var rawData struct {
	//	CurrentCondition []struct {
	//		TempC       string `json:"temp_C"`
	//		WeatherDesc []struct {
	//			Value string `json:"value"`
	//		} `json:"weatherDesc"`
	//		Humidity string `json:"humidity"`
	//		WindKmh  string `json:"windspeedKmph"`
	//	} `json:"current_condition"`
	//}
	//
	//if err := json.Unmarshal(body, &rawData); err != nil || len(rawData.CurrentCondition) == 0 {
	//	return nil, fmt.Errorf("解析天气 JSON 数据失败")
	//}
	//
	//curr := rawData.CurrentCondition[0]
	//var temp float64
	//var humidity int
	//var wind float64
	//
	//fmt.Sscanf(curr.TempC, "%f", &temp)
	//fmt.Sscanf(curr.Humidity, "%d", &humidity)
	//fmt.Sscanf(curr.WindKmh, "%f", &wind)
	//
	//desc := "未知"
	//if len(curr.WeatherDesc) > 0 {
	//	desc = curr.WeatherDesc[0].Value
	//}

	return &WeatherResponse{
		City:        city,
		Temperature: 24.5,
		Condition:   "Partly Cloudy",
		Humidity:    60,
		WindSpeed:   12.8,
	}, nil
}

func Start() {

	// 1. 设置全局 slog 为 Debug 级别并打印到 stdout/stderr
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug, // 开启 Debug 日志
	}))
	slog.SetDefault(logger)

	// 1. 初始化 MCP 服务
	s := server.NewMCPServer("weather-service", "1.0.0", server.WithToolCapabilities(true),
		server.WithLogging())

	// 2. 定义 Tool 工具描述与参数约束
	weatherTool := mcp.NewTool("get_weather",
		mcp.WithDescription("Get weather forecast and meteorological data for a given city / 获取指定城市的实时天气预报和气象数据。包含天气、温度、湿度等。"),
		mcp.WithString("city",
			mcp.Required(),
			mcp.Description("要查询的城市名称（例如: Beijing, Tokyo, London）"),
		),
	)

	// 3. 绑定处理 Handler
	s.AddTool(weatherTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// 从请求中解析入参
		city, ok := req.Params.Arguments.(map[string]any)["city"].(string)
		if !ok || city == "" {
			return mcp.NewToolResultError("缺少必填参数: city"), nil
		}

		// 调用外部 API 获取数据
		weatherData, err := fetchWeatherData(city)
		if err != nil {
			// 将外部错误友好地包装为 MCP Tool 错误返回给模型
			return mcp.NewToolResultError(fmt.Sprintf("查询天气失败: %v", err)), nil
		}

		// 序列化 JSON 结构，方便模型读取标准结构
		jsonBytes, err := json.MarshalIndent(weatherData, "", "  ")
		if err != nil {
			return mcp.NewToolResultError("JSON 序列化失败"), nil
		}

		// 返回文本/JSON 格式结果
		return mcp.NewToolResultText(string(jsonBytes)), nil
	})

	// 4. 启动 http 服务
	httpServer := server.NewStreamableHTTPServer(s, server.WithDisableLocalhostProtection(true), server.WithHTTPContextFunc(httpContext))
	if err := httpServer.Start(":9090"); err != nil {
		log.Fatalf("启动失败: %v", err)
	}
}

func httpContext(ctx context.Context, r *http.Request) context.Context {
	for k, v := range r.Header {
		fmt.Println(k, v)
	}
	return ctx
}
