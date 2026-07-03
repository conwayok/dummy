package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"
)

var AppName = "default-name"
var AppPort = 9999

type Code string

const (
	CodeOk            Code = "ok"
	CodeHeaderInvalid      = "header_invalid"
)

const (
	DummyResponseCodeHeaderKey string = "X-Dummy-Response-Code"
	DummySleepHeaderKey        string = "X-Dummy-Sleep"
)

type DummyResponse struct {
	Code              Code                   `json:"code"`
	Message           string                 `json:"message"`
	HostName          string                 `json:"host_name"`
	AppName           string                 `json:"app_name"`
	UnixTsMs          int64                  `json:"unix_ts_ms"`
	SourceIp          string                 `json:"source_ip"`
	RequestMethod     string                 `json:"request_method"`
	RequestUrl        string                 `json:"request_url"`
	RequestHeaders    http.Header            `json:"request_headers"`
	RequestBody       map[string]interface{} `json:"request_body"`
	ServerNetworkInfo []NetworkInterfaceInfo `json:"server_network_info"`
}

type NetworkInterfaceInfo struct {
	Name      string     `json:"name"`
	Addresses []net.Addr `json:"addresses"`
}

func init() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)
}

func handler(w http.ResponseWriter, r *http.Request) {
	hostname, _ := os.Hostname()
	hostNetworkInterfaces, _ := net.Interfaces()
	interfaceInfos := make([]NetworkInterfaceInfo, 0)
	for _, i := range hostNetworkInterfaces {
		addresses, _ := i.Addrs()
		interfaceInfos = append(interfaceInfos, NetworkInterfaceInfo{
			Name:      i.Name,
			Addresses: addresses,
		})
	}

	var reqBody map[string]interface{}
	body, _ := io.ReadAll(r.Body)
	if len(body) > 0 {
		reqBody = make(map[string]interface{})
		_ = json.Unmarshal(body, &reqBody)
	}

	res := DummyResponse{
		Code:              CodeOk,
		Message:           "success",
		UnixTsMs:          time.Now().UnixMilli(),
		HostName:          hostname,
		SourceIp:          r.RemoteAddr,
		AppName:           AppName,
		RequestMethod:     r.Method,
		RequestUrl:        r.RequestURI,
		RequestHeaders:    r.Header,
		RequestBody:       reqBody,
		ServerNetworkInfo: interfaceInfos,
	}

	responseCode := 200
	if responseCodeOption := r.Header.Get(DummyResponseCodeHeaderKey); responseCodeOption != "" {
		if parsed, err := strconv.Atoi(responseCodeOption); err == nil && parsed >= 100 && parsed <= 599 {
			responseCode = parsed
		} else {
			res.Code = CodeHeaderInvalid
			res.Message = fmt.Sprintf("%s header invalid", DummyResponseCodeHeaderKey)
		}
	}

	if sleepOption := r.Header.Get(DummySleepHeaderKey); sleepOption != "" {
		if parsed, err := strconv.Atoi(sleepOption); err == nil && parsed > 0 {
			time.Sleep(time.Duration(parsed) * time.Millisecond)
		} else {
			res.Code = CodeHeaderInvalid
			res.Message = fmt.Sprintf("%s header invalid", DummySleepHeaderKey)
		}
	}

	slog.Info("received request", "data", res)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(responseCode)
	_ = json.NewEncoder(w).Encode(res)
}

func main() {
	if configuredAppName := os.Getenv("DUMMY_APP_NAME"); configuredAppName != "" {
		AppName = configuredAppName
	}

	if configuredPort := os.Getenv("DUMMY_HTTP_PORT"); configuredPort != "" {
		if port, err := strconv.Atoi(configuredPort); err == nil && port > 0 && port <= 65535 {
			AppPort = port
		} else {
			slog.Error("configured port out of range")
			os.Exit(1)
		}
	}

	http.HandleFunc("/", handler)

	slog.Info("server started", "port", AppPort)

	if err := http.ListenAndServe(fmt.Sprintf(":%d", AppPort), nil); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
