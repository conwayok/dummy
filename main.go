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
var SafeMode = true

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
	HostName          string                 `json:"host_name,omitempty"`
	AppName           string                 `json:"app_name"`
	UnixTsMs          int64                  `json:"unix_ts_ms"`
	SourceIp          string                 `json:"source_ip,omitempty"`
	RequestMethod     string                 `json:"request_method,omitempty"`
	RequestUrl        string                 `json:"request_url,omitempty"`
	RequestHeaders    http.Header            `json:"request_headers,omitempty"`
	RequestBody       map[string]interface{} `json:"request_body,omitempty"`
	ServerNetworkInfo []NetworkInterfaceInfo `json:"server_network_info,omitempty"`
}

type NetworkInterfaceInfo struct {
	Name      string     `json:"name"`
	Addresses []net.Addr `json:"addresses"`
}

func handler(w http.ResponseWriter, r *http.Request) {
	var res DummyResponse

	if SafeMode {
		res = DummyResponse{
			Code:     CodeOk,
			Message:  "success",
			UnixTsMs: time.Now().UnixMilli(),
			AppName:  AppName,
		}
	} else {
		hostname, _ := os.Hostname()
		hostNetworkInterfaces, _ := net.Interfaces()
		var interfaceInfos []NetworkInterfaceInfo

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

		res = DummyResponse{
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

	slog.Info("handle request", "data", res)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(responseCode)
	_ = json.NewEncoder(w).Encode(res)
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

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

	if safeModeEnv := os.Getenv("DUMMY_SAFE_MODE"); safeModeEnv != "" {
		if safeModeEnv == "false" {
			SafeMode = false
		} else {
			slog.Warn("invalid value for DUMMY_SAFE_MODE, defaulting to true")
		}
	}

	http.HandleFunc("/", handler)

	slog.Info("server started", "port", AppPort, "app_name", AppName, "safe_mode", SafeMode)

	if err := http.ListenAndServe(fmt.Sprintf(":%d", AppPort), nil); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
