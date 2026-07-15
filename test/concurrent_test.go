// Package main 并发预订测试 — 验证行锁并发控制
//
// 使用方法:
//   go run test/concurrent_test.go
//
// 预期结果: 10 个并发请求同时预订同一间房，只有一个成功，其余返回"已被预订"错误
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
)

const baseURL = "http://localhost:8080/api"

type loginResp struct {
	Data struct {
		AccessToken string `json:"access_token"`
	} `json:"data"`
}

type orderResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		OrderID    int64   `json:"order_id"`
		OrderNo    string  `json:"order_no"`
		TotalPrice float64 `json:"total_price"`
	} `json:"data"`
}

func main() {
	fmt.Println("==================== 并发预订测试 ====================")
	fmt.Println("目标: 10个请求同时抢同一间房，验证行锁防护")
	fmt.Println("预期: 只有1个成功，其余9个返回冲突错误")
	fmt.Println()

	// 1. 登录获取 token
	token := login()
	fmt.Printf("[1] 登录成功, token=%s...\n\n", token[:30])

	// 2. 10 并发预订
	successCount := 0
	failCount := 0
	var mu sync.Mutex
	var wg sync.WaitGroup

	fmt.Println("[2] 开始 10 路并发预订...")
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ok := bookRoom(token, idx)
			mu.Lock()
			if ok {
				successCount++
			} else {
				failCount++
			}
			mu.Unlock()
		}(i)
	}
	wg.Wait()

	fmt.Println()
	fmt.Println("==================== 测试结果 ====================")
	fmt.Printf("✅ 成功: %d\n", successCount)
	fmt.Printf("❌ 失败: %d\n", failCount)
	if successCount == 1 && failCount == 9 {
		fmt.Println("🎉 行锁并发控制工作正常！只有1个请求成功预订。")
	} else if successCount > 1 {
		fmt.Println("⚠️  警告: 多个请求都成功了，行锁可能未生效！")
	} else {
		fmt.Println("⚠️  警告: 没有请求成功，请检查房间是否可用。")
	}
}

func login() string {
	body, _ := json.Marshal(map[string]string{
		"username": "13800008888",
		"password": "123456",
	})
	resp, err := http.Post(baseURL+"/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		// Fallback: use admin
		body, _ = json.Marshal(map[string]string{
			"username": "admin",
			"password": "123456",
		})
		resp, _ = http.Post(baseURL+"/auth/login", "application/json", bytes.NewReader(body))
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(data, &result)
	dataMap, _ := result["data"].(map[string]interface{})
	return dataMap["access_token"].(string)
}

func bookRoom(token string, idx int) bool {
	body, _ := json.Marshal(map[string]interface{}{
		"room_id":    1, // 假设房间ID=1存在
		"guest_name": fmt.Sprintf("测试用户%d", idx),
		"id_card":    "330106199001011234",
		"check_in":   "2026-07-20",
		"check_out":  "2026-07-23",
	})
	req, _ := http.NewRequest("POST", baseURL+"/orders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("  [goroutine %d] 请求失败: %v\n", idx, err)
		return false
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)

	var orderR orderResp
	json.Unmarshal(data, &orderR)

	if orderR.Code == 0 {
		fmt.Printf("  [goroutine %d] ✅ 预订成功! order_id=%d\n", idx, orderR.Data.OrderID)
		return true
	}
	fmt.Printf("  [goroutine %d] ❌ 失败: %s\n", idx, orderR.Message)
	return false
}
