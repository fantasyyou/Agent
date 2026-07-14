package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

const (
	apiURL = "https://api.deepseek.com/v1/chat/completions"
	model  = "deepseek-chat"
)

type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	Name       string     `json:"name,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function ToolCallFunction `json:"function"`
}

type ToolCallFunction struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type Request struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Tools    []Tool    `json:"tools"`
}

type Response struct {
	Choices []Choice `json:"choices"`
}

type Choice struct {
	Message Message `json:"message"`
}

func getAPIKey() string {
	return "sk-64a13c674b8141f3a3777e5bc7614bbe"
}

func getTools() []Tool {
	return []Tool{
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "add",
				Description: "加法运算，计算两个数的和",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"a": map[string]interface{}{"type": "number", "description": "第一个加数"},
						"b": map[string]interface{}{"type": "number", "description": "第二个加数"},
					},
					"required": []string{"a", "b"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "subtract",
				Description: "减法运算，计算两个数的差",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"a": map[string]interface{}{"type": "number", "description": "被减数"},
						"b": map[string]interface{}{"type": "number", "description": "减数"},
					},
					"required": []string{"a", "b"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "multiply",
				Description: "乘法运算，计算两个数的积",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"a": map[string]interface{}{"type": "number", "description": "第一个因数"},
						"b": map[string]interface{}{"type": "number", "description": "第二个因数"},
					},
					"required": []string{"a", "b"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "divide",
				Description: "除法运算，计算两个数的商",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"a": map[string]interface{}{"type": "number", "description": "被除数"},
						"b": map[string]interface{}{"type": "number", "description": "除数（不能为0）"},
					},
					"required": []string{"a", "b"},
				},
			},
		},
	}
}

func executeTool(name string, args json.RawMessage) string {
	var argsStr string
	if err := json.Unmarshal(args, &argsStr); err != nil {
		return fmt.Sprintf("解析参数字符串失败: %v", err)
	}
	var argsMap map[string]interface{}
	if err := json.Unmarshal([]byte(argsStr), &argsMap); err != nil {
		return fmt.Sprintf("解析参数JSON失败: %v", err)
	}
	a := argsMap["a"].(float64)
	b := argsMap["b"].(float64)

	switch name {
	case "add":
		return strconv.FormatFloat(a+b, 'f', -1, 64)
	case "subtract":
		return strconv.FormatFloat(a-b, 'f', -1, 64)
	case "multiply":
		return strconv.FormatFloat(a*b, 'f', -1, 64)
	case "divide":
		if b == 0 {
			return "错误：除数不能为0"
		}
		return strconv.FormatFloat(a/b, 'f', -1, 64)
	default:
		return fmt.Sprintf("未知工具：%s", name)
	}
}

func callAPI(messages []Message, tools []Tool) (*Response, error) {
	reqBody := Request{Model: model, Messages: messages, Tools: tools}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化失败: %v", err)
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+getAPIKey())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API错误 %d: %s", resp.StatusCode, string(body))
	}

	var response Response
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	return &response, nil
}

func main() {
	//if len(os.Args) < 2 {
	//	fmt.Println("DeepSeek Tool Demo - 四则运算计算器")
	//	fmt.Println("==================================")
	//	fmt.Println("用法: go run deepseek_tool_demo.go \"你的问题\"")
	//	fmt.Println()
	//	fmt.Println("示例:")
	//	fmt.Println("  go run deepseek_tool_demo.go \"123 + 456 等于多少？\"")
	//	fmt.Println("  go run deepseek_tool_demo.go \"100 - 25 等于多少？\"")
	//	fmt.Println("  go run deepseek_tool_demo.go \"计算 50 * 20\"")
	//	fmt.Println("  go run deepseek_tool_demo.go \"1000 / 40 是多少？\"")
	//	os.Exit(1)
	//}

	userQuestion := "123 + 456 等于多少？"
	fmt.Printf("用户问题: %s\n\n", userQuestion)

	messages := []Message{
		{Role: "system", Content: "你是一个计算器助手，当遇到数学问题时，请调用提供的工具进行计算。"},
		{Role: "user", Content: userQuestion},
	}

	tools := getTools()

	for {
		resp, err := callAPI(messages, tools)
		if err != nil {
			fmt.Printf("API调用失败: %v\n", err)
			return
		}

		if len(resp.Choices) == 0 {
			fmt.Println("空响应")
			return
		}

		msg := resp.Choices[0].Message

		if len(msg.ToolCalls) > 0 {
			fmt.Println("助手调用工具:")
			for _, tc := range msg.ToolCalls {
				name := tc.Function.Name
				args := tc.Function.Arguments
				toolCallID := tc.ID

				var argsStr string
				json.Unmarshal(args, &argsStr)
				var argsMap map[string]interface{}
				json.Unmarshal([]byte(argsStr), &argsMap)
				fmt.Printf("  - 工具: %s\n", name)
				fmt.Printf("    参数: a=%v, b=%v\n", argsMap["a"], argsMap["b"])

				result := executeTool(name, args)
				fmt.Printf("    结果: %s\n", result)

				messages = append(messages, msg)
				messages = append(messages, Message{
					Role:       "tool",
					Content:    result,
					Name:       name,
					ToolCallID: toolCallID,
				})
			}
			fmt.Println()
			continue
		}

		if msg.Content != "" {
			fmt.Printf("助手回答: %s\n", msg.Content)
			break
		}

		fmt.Println("未检测到有效响应")
		break
	}
}
