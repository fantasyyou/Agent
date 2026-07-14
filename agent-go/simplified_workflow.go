package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

const agentServiceURL = "http://127.0.0.1:8765"

type AgentResult struct {
	Role   string                 `json:"role"`
	Symbol string                 `json:"symbol"`
	Date   string                 `json:"date"`
	Report string                 `json:"report"`
	Data   map[string]interface{} `json:"data"`
}

func callAgentService(role, symbol, date string, deps map[string]interface{}) (AgentResult, error) {
	reqBody := map[string]interface{}{
		"role":   role,
		"symbol": symbol,
		"date":   date,
	}
	if deps != nil {
		reqBody["deps"] = deps
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return AgentResult{}, fmt.Errorf("marshal %s request: %w", role, err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(agentServiceURL+"/agent", "application/json", bytes.NewReader(payload))
	if err != nil {
		return AgentResult{}, fmt.Errorf("call agent service %s: %w", role, err)
	}
	defer resp.Body.Close()

	var output bytes.Buffer
	if _, err := output.ReadFrom(resp.Body); err != nil {
		return AgentResult{}, fmt.Errorf("read agent service %s response: %w", role, err)
	}
	if resp.StatusCode != http.StatusOK {
		return AgentResult{}, fmt.Errorf("agent service %s returned %s: %s", role, resp.Status, output.String())
	}

	var result AgentResult
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		return AgentResult{}, fmt.Errorf("decode agent service %s output: %w, output: %s", role, err, output.String())
	}

	return result, nil
}

func callAgentAsync(role, symbol, date string, wg *sync.WaitGroup, ch chan<- AgentResult, errCh chan<- error) {
	defer wg.Done()

	result, err := callAgentService(role, symbol, date, nil)
	if err != nil {
		errCh <- err
		return
	}
	ch <- result
}

func renderReport(marketR, fundR, finalR AgentResult) {
	fmt.Println("====== Market Analyst ======")
	fmt.Println(marketR.Report)
	fmt.Println()

	fmt.Println("====== Fundamental Analyst ======")
	fmt.Println(fundR.Report)
	fmt.Println()

	fmt.Println("====== Final Report ======")
	fmt.Println(finalR.Report)
}

func RunSimplifiedWorkflow(symbol, date string) error {
	// 1. 准备数据由 Python 侧模拟采集。

	// 2. 并行调用两个分析师（Goroutine）。
	var wg sync.WaitGroup
	resultCh := make(chan AgentResult, 2)
	errCh := make(chan error, 2)

	wg.Add(2)
	go callAgentAsync("market", symbol, date, &wg, resultCh, errCh)
	go callAgentAsync("fundamental", symbol, date, &wg, resultCh, errCh)
	wg.Wait()
	close(resultCh)
	close(errCh)

	if len(errCh) > 0 {
		return <-errCh
	}

	var marketR AgentResult
	var fundR AgentResult
	for result := range resultCh {
		switch result.Role {
		case "market":
			marketR = result
		case "fundamental":
			fundR = result
		}
	}

	// 3. 串行调用投资经理（依赖上面两份报告）。
	deps := map[string]interface{}{
		"market_report":      marketR.Report,
		"fundamental_report": fundR.Report,
	}
	finalR, err := callAgentService("final", symbol, date, deps)
	if err != nil {
		return err
	}

	// 4. 输出报告。
	renderReport(marketR, fundR, finalR)
	return nil
}

func main() {
	symbol := "002607.SZ"
	tradeDate := "2025-12-31"
	if len(os.Args) > 1 {
		symbol = os.Args[1]
	}
	if len(os.Args) > 2 {
		tradeDate = os.Args[2]
	}

	if err := RunSimplifiedWorkflow(symbol, tradeDate); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
