package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
)

var AppName = "default-name"
var AppPort = 9999

const (
	Ok                         string = "ok"
	HeaderInvalid              string = "header_invalid"
	DummyResponseCodeHeaderKey string = "X-Dummy-Response-Code"
	DummySleepHeaderKey        string = "X-Dummy-Sleep"
)

type DummyResponse struct {
	Code              string                 `json:"code"`
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
	_ = os.MkdirAll("logs", 0755)
	logFileHandler, err := os.OpenFile("logs/dummy.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0755)
	if err != nil {
		fmt.Printf("error %s", err)
	}
	log.SetFormatter(&log.JSONFormatter{})
	mw := io.MultiWriter(os.Stdout, logFileHandler)
	log.SetOutput(mw)
	log.SetLevel(log.InfoLevel)
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
		Code:              Ok,
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
			res.Code = HeaderInvalid
			res.Message = fmt.Sprintf("%s header invalid", DummyResponseCodeHeaderKey)
		}
	}

	if sleepOption := r.Header.Get(DummySleepHeaderKey); sleepOption != "" {
		if parsed, err := strconv.Atoi(sleepOption); err == nil && parsed > 0 {
			time.Sleep(time.Duration(parsed) * time.Millisecond)
		} else {
			res.Code = HeaderInvalid
			res.Message = fmt.Sprintf("%s header invalid", DummySleepHeaderKey)
		}
	}

	log.WithFields(log.Fields{"json": &res}).Info("received request")

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
			log.Fatal("port out of range")
		}
	}

	http.HandleFunc("/", handler)

	log.Infof("Starting server on :%d", AppPort)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", AppPort), nil); err != nil {
		log.Fatal(err)
	}
}
