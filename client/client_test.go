package client_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/jsdelivr/globalping-cli/client"
	"github.com/jsdelivr/globalping-cli/model"

	"github.com/stretchr/testify/assert"
)

// Generate server for testing
func generateServer(json string) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(json))
	}))
	return server
}

func generateServerError(json string, statusCode int) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		w.Write([]byte(json))
	}))
	return server
}

// Dummy interface since we have mock responses
var opts = model.PostMeasurement{}

// PostAPI tests
func TestPostAPI(t *testing.T) {
	// Suppress error outputs
	os.Stdout, _ = os.Open(os.DevNull)
	for scenario, fn := range map[string]func(t *testing.T){
		"valid":      testPostValid,
		"no_probes":  testPostNoProbes,
		"validation": testPostValidation,
		"api_error":  testPostInternalError,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

// Test a valid call of PostAPI
func testPostValid(t *testing.T) {
	server := generateServer(`{"id":"abcd","probesCount":1}`)
	defer server.Close()
	client.ApiUrl = server.URL

	res, showHelp, err := client.PostAPI(opts)

	assert.Equal(t, "abcd", res.ID)
	assert.Equal(t, 1, res.ProbesCount)
	assert.False(t, showHelp)
	assert.NoError(t, err)
}

func testPostNoProbes(t *testing.T) {
	server := generateServerError(`{
    "error": {
      "message": "No suitable probes found",
      "type": "no_probes_found"
    }}`, 422)
	defer server.Close()
	client.ApiUrl = server.URL

	_, showHelp, err := client.PostAPI(opts)
	assert.EqualError(t, err, "no suitable probes found - please choose a different location")
	assert.True(t, showHelp)
}

func testPostValidation(t *testing.T) {
	server := generateServerError(`{
    "error": {
        "message": "Validation Failed",
        "type": "validation_error",
        "params": {
            "measurement": "\"measurement\" does not match any of the allowed types",
			"target": "\"target\" does not match any of the allowed types"
        }
    }}`, 400)
	defer server.Close()
	client.ApiUrl = server.URL

	_, showHelp, err := client.PostAPI(opts)
	assert.EqualError(t, err, "invalid parameters - please check the help for more information")
	assert.True(t, showHelp)
}

func testPostInternalError(t *testing.T) {
	server := generateServerError(`{
    "error": {
      "message": "Internal Server Error",
      "type": "api_error"
    }}`, 500)
	defer server.Close()
	client.ApiUrl = server.URL

	_, showHelp, err := client.PostAPI(opts)
	assert.EqualError(t, err, "err: internal server error - please try again later")
	assert.False(t, showHelp)
}

// GetAPI tests
func TestGetAPI(t *testing.T) {
	for scenario, fn := range map[string]func(t *testing.T){
		"valid":      testGetValid,
		"json":       testGetJson,
		"ping":       testGetPing,
		"traceroute": testGetTraceroute,
		"dns":        testGetDns,
		"mtr":        testGetMtr,
		"http":       testGetHttp,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func testGetValid(t *testing.T) {
	server := generateServer(`{"id":"abcd"}`)
	defer server.Close()
	client.ApiUrl = server.URL

	res, err := client.GetAPI("abcd")
	if err != nil {
		t.Error(err)
	}

	t.Logf("%+v", res)

	assert.Equal(t, "abcd", res.ID)
}

func testGetJson(t *testing.T) {
	server := generateServer(`{"id":"abcd"}`)
	defer server.Close()
	client.ApiUrl = server.URL

	res, err := client.GetApiJson("abcd")
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, `{"id":"abcd"}`, res)
}

func testGetPing(t *testing.T) {
	server := generateServer(`{
	"id": "abcd",
	"type": "ping",
	"status": "finished",
	"createdAt": "2023-02-17T18:11:52.825Z",
	"updatedAt": "2023-02-17T18:11:53.969Z",
	"probesCount": 1,
	"results": [
		{
		"probe": {
			"continent": "NA",
			"region": "Northern America",
			"country": "CA",
			"state": null,
			"city": "City",
			"asn": 7794,
			"longitude": -80.2222,
			"latitude": 43.3662,
			"network": "Network",
			"tags": [],
			"resolvers": [
			"1.1.1.1",
			"8.8.4.4"
			]
		},
		"result": {
			"status": "finished",
			"rawOutput": "PING",
			"resolvedAddress": "1.1.1.1",
			"resolvedHostname": "1.1.1.1:",
			"timings": [],
			"stats": {
				"min": 24.891,
				"max": 28.193,
				"avg": 27.088,
				"total": 3,
				"loss": 0,
				"rcv": 3,
				"drop": 0
			}
		}
	}]}`)
	defer server.Close()
	client.ApiUrl = server.URL

	res, err := client.GetAPI("abcd")
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, "abcd", res.ID)
	assert.Equal(t, "ping", res.Type)
	assert.Equal(t, "finished", res.Status)
	assert.Equal(t, "2023-02-17T18:11:52.825Z", res.CreatedAt)
	assert.Equal(t, "2023-02-17T18:11:53.969Z", res.UpdatedAt)
	assert.Equal(t, 1, res.ProbesCount)
	assert.Equal(t, 1, len(res.Results))

	assert.Equal(t, "NA", res.Results[0].Probe.Continent)
	assert.Equal(t, "Northern America", res.Results[0].Probe.Region)
	assert.Equal(t, "CA", res.Results[0].Probe.Country)
	assert.Equal(t, "", res.Results[0].Probe.State)
	assert.Equal(t, "City", res.Results[0].Probe.City)
	assert.Equal(t, 7794, res.Results[0].Probe.ASN)
	assert.Equal(t, "Network", res.Results[0].Probe.Network)
	assert.Equal(t, 0, len(res.Results[0].Probe.Tags))

	assert.Equal(t, "PING", res.Results[0].Result.RawOutput)
	assert.Equal(t, "1.1.1.1", res.Results[0].Result.ResolvedAddress)
	assert.Equal(t, 27.088, res.Results[0].Result.Stats["avg"])
	assert.Equal(t, 28.193, res.Results[0].Result.Stats["max"])
	assert.Equal(t, 24.891, res.Results[0].Result.Stats["min"])
	assert.Equal(t, float64(3), res.Results[0].Result.Stats["total"])
	assert.Equal(t, float64(3), res.Results[0].Result.Stats["rcv"])
	assert.Equal(t, float64(0), res.Results[0].Result.Stats["loss"])
	assert.Equal(t, float64(0), res.Results[0].Result.Stats["drop"])
}

func testGetTraceroute(t *testing.T) {
	server := generateServer(`{
	"id": "abcd",
	"type": "traceroute",
	"status": "finished",
	"createdAt": "2023-02-23T07:55:23.414Z",
	"updatedAt": "2023-02-23T07:55:25.496Z",
	"probesCount": 1,
	"results": [
		{
		"probe": {
			"continent": "EU",
			"region": "Northern Europe",
			"country": "GB",
			"state": null,
			"city": "London",
			"asn": 16276,
			"longitude": -0.1257,
			"latitude": 51.5085,
			"network": "OVH SAS",
			"tags": [],
			"resolvers": [
			"private"
			]
		},
		"result": {
			"rawOutput": "TRACEROUTE",
			"status": "finished",
			"resolvedAddress": "1.1.1.1",
			"resolvedHostname": "1.1.1.1",
			"hops": [
			{
				"resolvedHostname": "54.37.244.252",
				"resolvedAddress": "54.37.244.252",
				"timings": [
				{
					"rtt": 0.408
				},
				{
					"rtt": 0.502
				}
				]
			},
			{
				"resolvedHostname": "93.123.11.62",
				"resolvedAddress": "93.123.11.62",
				"timings": [
				{
					"rtt": 0.507
				},
				{
					"rtt": 0.524
				}
				]
			}
			]
	}}]}`)
	defer server.Close()
	client.ApiUrl = server.URL

	res, err := client.GetAPI("abcd")
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, "abcd", res.ID)
	assert.Equal(t, "traceroute", res.Type)
	assert.Equal(t, "finished", res.Status)
	assert.Equal(t, "2023-02-23T07:55:23.414Z", res.CreatedAt)
	assert.Equal(t, "2023-02-23T07:55:25.496Z", res.UpdatedAt)
	assert.Equal(t, 1, res.ProbesCount)
	assert.Equal(t, 1, len(res.Results))

	assert.Equal(t, "EU", res.Results[0].Probe.Continent)
	assert.Equal(t, "Northern Europe", res.Results[0].Probe.Region)
	assert.Equal(t, "GB", res.Results[0].Probe.Country)
	assert.Equal(t, "", res.Results[0].Probe.State)
	assert.Equal(t, "London", res.Results[0].Probe.City)
	assert.Equal(t, 16276, res.Results[0].Probe.ASN)
	assert.Equal(t, "OVH SAS", res.Results[0].Probe.Network)
	assert.Equal(t, 0, len(res.Results[0].Probe.Tags))

	assert.Equal(t, "TRACEROUTE", res.Results[0].Result.RawOutput)
	assert.Equal(t, "1.1.1.1", res.Results[0].Result.ResolvedAddress)
	assert.Equal(t, "1.1.1.1", res.Results[0].Result.ResolvedHostname)
}

func testGetDns(t *testing.T) {
	server := generateServer(`{
	"id": "abcd",
	"type": "dns",
	"status": "finished",
	"createdAt": "2023-02-23T08:00:37.431Z",
	"updatedAt": "2023-02-23T08:00:37.640Z",
	"probesCount": 1,
	"results": [
		{
		"probe": {
			"continent": "EU",
			"region": "Western Europe",
			"country": "NL",
			"state": null,
			"city": "Amsterdam",
			"asn": 60404,
			"longitude": 4.8897,
			"latitude": 52.374,
			"network": "Liteserver",
			"tags": [],
			"resolvers": [
			"185.31.172.240",
			"89.188.29.4"
			]
		},
		"result": {
			"status": "finished",
			"statusCodeName": "NOERROR",
			"statusCode": 0,
			"rawOutput": "DNS",
			"answers": [
			{
				"name": "jsdelivr.com.",
				"type": "A",
				"ttl": 30,
				"class": "IN",
				"value": "92.223.84.84"
			}
			],
			"timings": {
			"total": 15
			},
			"resolver": "185.31.172.240"
		}
	}]}`)
	defer server.Close()
	client.ApiUrl = server.URL

	res, err := client.GetAPI("abcd")
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, "abcd", res.ID)
	assert.Equal(t, "dns", res.Type)
	assert.Equal(t, "finished", res.Status)
	assert.Equal(t, "2023-02-23T08:00:37.431Z", res.CreatedAt)
	assert.Equal(t, "2023-02-23T08:00:37.640Z", res.UpdatedAt)
	assert.Equal(t, 1, res.ProbesCount)
	assert.Equal(t, 1, len(res.Results))

	assert.Equal(t, "EU", res.Results[0].Probe.Continent)
	assert.Equal(t, "Western Europe", res.Results[0].Probe.Region)
	assert.Equal(t, "NL", res.Results[0].Probe.Country)
	assert.Equal(t, "", res.Results[0].Probe.State)
	assert.Equal(t, "Amsterdam", res.Results[0].Probe.City)
	assert.Equal(t, 60404, res.Results[0].Probe.ASN)
	assert.Equal(t, "Liteserver", res.Results[0].Probe.Network)
	assert.Equal(t, 0, len(res.Results[0].Probe.Tags))

	assert.Equal(t, "DNS", res.Results[0].Result.RawOutput)
	assert.Equal(t, "finished", res.Results[0].Result.Status)
	assert.IsType(t, json.RawMessage{}, res.Results[0].Result.TimingsRaw)

	// Test timings
	timings, _ := client.DecodeTimings("dns", res.Results[0].Result.TimingsRaw)
	assert.Equal(t, float64(15), timings.Interface["total"])
	assert.Nil(t, timings.Arr)
}

func testGetMtr(t *testing.T) {
	server := generateServer(`{
	"id": "abcd",
	"type": "mtr",
	"status": "finished",
	"createdAt": "2023-02-23T08:08:25.187Z",
	"updatedAt": "2023-02-23T08:08:29.829Z",
	"probesCount": 1,
	"results": [
		{
		"probe": {
			"continent": "EU",
			"region": "Western Europe",
			"country": "NL",
			"state": null,
			"city": "Amsterdam",
			"asn": 54825,
			"longitude": 4.8897,
			"latitude": 52.374,
			"network": "Packet Host, Inc.",
			"tags": [],
			"resolvers": []
		},
		"result": {
			"status": "finished",
			"rawOutput": "MTR",
			"resolvedAddress": "92.223.84.84",
			"resolvedHostname": "92.223.84.84",
			"hops": [
			{
				"stats": {
				"min": 0.176,
				"max": 0.226,
				"avg": 0.2,
				"total": 3,
				"loss": 0,
				"rcv": 3,
				"drop": 0,
				"stDev": 0,
				"jMin": 0,
				"jMax": 0.2,
				"jAvg": 0.1
				},
				"asn": [],
				"timings": [
				{
					"rtt": 0.176
				},
				{
					"rtt": 0.216
				},
				{
					"rtt": 0.226
				}
				],
				"resolvedAddress": "172.19.66.225",
				"duplicate": false,
				"resolvedHostname": "172.19.66.225"
			},
			{
				"stats": {
				"min": 0.894,
				"max": 0.894,
				"avg": 0.9,
				"total": 1,
				"loss": 0,
				"rcv": 1,
				"drop": 0,
				"stDev": 0,
				"jMin": 0.9,
				"jMax": 0.9,
				"jAvg": 0.9
				},
				"asn": [
				199524
				],
				"timings": [
				{
					"rtt": 0.894
				}
				],
				"resolvedAddress": "92.223.84.84",
				"duplicate": true,
				"resolvedHostname": "92.223.84.84"
			}
			]
		}
	}]}`)
	defer server.Close()
	client.ApiUrl = server.URL

	res, err := client.GetAPI("abcd")
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, "abcd", res.ID)
	assert.Equal(t, "mtr", res.Type)
	assert.Equal(t, "finished", res.Status)
	assert.Equal(t, "2023-02-23T08:08:25.187Z", res.CreatedAt)
	assert.Equal(t, "2023-02-23T08:08:29.829Z", res.UpdatedAt)
	assert.Equal(t, 1, res.ProbesCount)
	assert.Equal(t, 1, len(res.Results))

	assert.Equal(t, "EU", res.Results[0].Probe.Continent)
	assert.Equal(t, "Western Europe", res.Results[0].Probe.Region)
	assert.Equal(t, "NL", res.Results[0].Probe.Country)
	assert.Equal(t, "", res.Results[0].Probe.State)
	assert.Equal(t, "Amsterdam", res.Results[0].Probe.City)
	assert.Equal(t, 54825, res.Results[0].Probe.ASN)
	assert.Equal(t, "Packet Host, Inc.", res.Results[0].Probe.Network)
	assert.Equal(t, 0, len(res.Results[0].Probe.Tags))

	assert.Equal(t, "MTR", res.Results[0].Result.RawOutput)
	assert.Equal(t, "finished", res.Results[0].Result.Status)
	assert.IsType(t, json.RawMessage{}, res.Results[0].Result.TimingsRaw)
}

func testGetHttp(t *testing.T) {
	server := generateServer(`{
	"id": "abcd",
	"type": "http",
	"status": "finished",
	"createdAt": "2023-02-23T08:16:11.335Z",
	"updatedAt": "2023-02-23T08:16:12.548Z",
	"probesCount": 1,
	"results": [
		{
		"probe": {
			"continent": "NA",
			"region": "Northern America",
			"country": "CA",
			"state": null,
			"city": "Pembroke",
			"asn": 577,
			"longitude": -77.1162,
			"latitude": 45.8168,
			"network": "Bell Canada",
			"tags": [],
			"resolvers": [
			"private",
			"private"
			]
		},
		"result": {
			"status": "finished",
			"resolvedAddress": "5.101.222.14",
			"headers": {
			"server": "nginx",
			"date": "Thu, 23 Feb 2023 08:16:12 GMT",
			"content-type": "text/html; charset=utf-8",
			"connection": "close",
			"location": "/",
			"cf-ray": "79de849d3fa30c33-AMS",
			"vary": "Accept-Encoding",
			"cf-cache-status": "DYNAMIC",
			"x-render-origin-server": "Render",
			"x-response-time": "1ms",
			"cache": "MISS, MISS",
			"x-id": "am3-up-gc88, td2-up-gc10",
			"x-nginx": "nginx-be, nginx-be"
			},
			"rawHeaders": "Server: nginx\nDate: Thu, 23 Feb 2023 08:16:12 GMT\nContent-Type: text/html; charset=utf-8\nConnection: close\nLocation: /\nCF-Ray: 79de849d3fa30c33-AMS\nVary: Accept-Encoding\nCF-Cache-Status: DYNAMIC\nx-render-origin-server: Render\nx-response-time: 1ms\nCache: MISS\nX-ID: am3-up-gc88\nX-NGINX: nginx-be\nCache: MISS\nX-ID: td2-up-gc10\nX-NGINX: nginx-be",
			"rawBody": null,
			"statusCode": 301,
			"statusCodeName": "Moved Permanently",
			"timings": {
			"total": 583,
			"download": 18,
			"firstByte": 450,
			"dns": 24,
			"tls": 70,
			"tcp": 19
			},
			"tls": {
			"authorized": true,
			"createdAt": "2023-02-18T00:00:00.000Z",
			"expiresAt": "2024-02-18T23:59:59.000Z",
			"issuer": {
				"C": "GB",
				"ST": "Greater Manchester",
				"L": "Salford",
				"O": "Sectigo Limited",
				"CN": "Sectigo RSA Domain Validation Secure Server CA"
			},
			"subject": {
				"CN": "jsdelivr.com",
				"alt": "DNS:jsdelivr.com, DNS:data.jsdelivr.com, DNS:www.jsdelivr.com"
			}
			},
			"rawOutput": "HTTP"
		}
	}]}`)
	defer server.Close()
	client.ApiUrl = server.URL

	res, err := client.GetAPI("abcd")
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, "abcd", res.ID)
	assert.Equal(t, "http", res.Type)
	assert.Equal(t, "finished", res.Status)
	assert.Equal(t, "2023-02-23T08:16:11.335Z", res.CreatedAt)
	assert.Equal(t, "2023-02-23T08:16:12.548Z", res.UpdatedAt)
	assert.Equal(t, 1, res.ProbesCount)
	assert.Equal(t, 1, len(res.Results))

	assert.Equal(t, "NA", res.Results[0].Probe.Continent)
	assert.Equal(t, "Northern America", res.Results[0].Probe.Region)
	assert.Equal(t, "CA", res.Results[0].Probe.Country)
	assert.Equal(t, "", res.Results[0].Probe.State)
	assert.Equal(t, "Pembroke", res.Results[0].Probe.City)
	assert.Equal(t, 577, res.Results[0].Probe.ASN)
	assert.Equal(t, "Bell Canada", res.Results[0].Probe.Network)
	assert.Equal(t, 0, len(res.Results[0].Probe.Tags))

	assert.Equal(t, "HTTP", res.Results[0].Result.RawOutput)
	assert.Equal(t, "finished", res.Results[0].Result.Status)
	assert.IsType(t, json.RawMessage{}, res.Results[0].Result.TimingsRaw)

	// Test timings
	timings, _ := client.DecodeTimings("dns", res.Results[0].Result.TimingsRaw)
	assert.Nil(t, timings.Arr)
	assert.Equal(t, float64(583), timings.Interface["total"])
	assert.Equal(t, float64(18), timings.Interface["download"])
	assert.Equal(t, float64(450), timings.Interface["firstByte"])
	assert.Equal(t, float64(24), timings.Interface["dns"])
	assert.Equal(t, float64(70), timings.Interface["tls"])
	assert.Equal(t, float64(19), timings.Interface["tcp"])
}
