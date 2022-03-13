package main

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"
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
	UnixTimestamp     int64                  `json:"unix_timestamp"`
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
	_ = os.MkdirAll("logs", os.ModeDir)
	logFileHandler, err := os.OpenFile("logs/dummy.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0755)
	if err != nil {
		fmt.Printf("error %s", err)
	}
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(logFileHandler)
	log.SetLevel(log.InfoLevel)
}

func main() {

	configuredAppName := os.Getenv("DUMMY_APP_NAME")
	if configuredAppName != "" {
		AppName = configuredAppName
	}

	configuredPort := os.Getenv("DUMMY_HTTP_PORT")

	if configuredPort != "" {
		port, err := strconv.Atoi(configuredPort)

		if err != nil {
			log.Fatal(err)
		}
		if port < 1 || port > 65535 {
			log.Fatal("port out of range")
		}

		AppPort = port
	}

	log.Info("start")

	hostname, _ := os.Hostname()
	hostNetworkInterfaces, _ := net.Interfaces()
	interfaceInfos := make([]NetworkInterfaceInfo, 0)
	for _, i := range hostNetworkInterfaces {

		addresses, _ := i.Addrs()

		n := NetworkInterfaceInfo{
			Name:      i.Name,
			Addresses: addresses,
		}

		interfaceInfos = append(interfaceInfos, n)
	}

	engine := gin.Default()

	engine.Any("/*all", func(context *gin.Context) {

		var reqBody map[string]interface{} = nil
		all, _ := ioutil.ReadAll(context.Request.Body)
		if len(all) > 0 {
			reqBody = make(map[string]interface{})
			_ = json.Unmarshal(all, &reqBody)
		}

		res := DummyResponse{
			Code:              Ok,
			Message:           "success",
			UnixTimestamp:     time.Now().UnixMilli(),
			HostName:          hostname,
			SourceIp:          context.ClientIP(),
			AppName:           AppName,
			RequestMethod:     context.Request.Method,
			RequestUrl:        context.Request.RequestURI,
			RequestHeaders:    context.Request.Header,
			RequestBody:       reqBody,
			ServerNetworkInfo: interfaceInfos,
		}

		responseCode := 200
		responseCodeOption := context.Request.Header.Get(DummyResponseCodeHeaderKey)
		if responseCodeOption != "" {
			parsed, err := strconv.Atoi(responseCodeOption)
			if err != nil || parsed < 100 || parsed > 599 {
				res.Code = HeaderInvalid
				res.Message = fmt.Sprintf("%s header invalid", DummyResponseCodeHeaderKey)
			} else {
				responseCode = parsed
			}
		}

		sleepOption := context.Request.Header.Get(DummySleepHeaderKey)
		if sleepOption != "" {
			parsed, err := strconv.Atoi(sleepOption)
			if err != nil || parsed < 1 {
				res.Code = HeaderInvalid
				res.Message = fmt.Sprintf("%s header invalid", DummyResponseCodeHeaderKey)
			} else {
				time.Sleep(time.Duration(parsed) * time.Millisecond)
			}
		}

		log.WithFields(log.Fields{
			"json": &res,
		}).Info("received request")

		context.JSON(responseCode, res)
	})

	_ = engine.Run(fmt.Sprintf(":%d", AppPort))
}
