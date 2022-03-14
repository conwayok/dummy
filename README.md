# dummy

This is a web app that can be used as a "dummy" service, which is useful for testing scenarios.

### Run with Docker:

```console
docker run --rm -d -p 9999:9999 conwayok/dummy:latest
```

---

### HTTP Request:

- any HTTP request method is accepted, e.g., GET, POST, PUT, PATCH, DELETE...
- any /request/path/or?query=parameters&are=accepted
- any JSON request body
- request header ```X-Dummy-Response-Code``` can be used to set the http response code
- request header ```X-Dummy-Sleep``` can make the server wait for specified amount of milliseconds before sending response

For example:

```console
curl -X POST -H 'Content-Type: application/json' -d '{"hello":"world"}' http://localhost:9999/hello/world
```

Will get a response like:

```JSON
{
  "code": "ok",
  "message": "success",
  "host_name": "f8dc0448a6a9",
  "app_name": "default-name",
  "unix_timestamp": 1641404337328,
  "source_ip": "172.17.0.1",
  "request_method": "POST",
  "request_url": "/hello/world",
  "request_headers": {
    "Accept": [
      "*/*"
    ],
    "Content-Length": [
      "17"
    ],
    "Content-Type": [
      "application/json"
    ],
    "User-Agent": [
      "curl/7.68.0"
    ]
  },
  "request_body": {
    "hello": "world"
  },
  "server_network_info": [
    {
      "name": "lo",
      "addresses": [
        {
          "IP": "127.0.0.1",
          "Mask": "/wAAAA=="
        }
      ]
    },
    {
      "name": "sit0",
      "addresses": null
    },
    {
      "name": "eth0",
      "addresses": [
        {
          "IP": "172.17.0.2",
          "Mask": "//8AAA=="
        }
      ]
    }
  ]
}
```

For any http request, the app will also write the responses to the file ```/logs/dummy.log```

---

## Error codes
- `ok`: the request was successful
- `header_invalid`: the X-Dummy-* prefixed header was not in correct format

---

## Configuration

This app uses the following environment variables for configuration:

- ```DUMMY_APP_NAME``` Sets the app_name as seen in the above examples. Defaults to "default-name".
- ```DUMMY_HTTP_PORT``` Sets the port for HTTP server. Defaults to 9999. 




