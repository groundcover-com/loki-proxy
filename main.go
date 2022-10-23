package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httputil"

	"github.com/gin-gonic/gin"
	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/grafana/loki/pkg/loghttp/push"
	"github.com/grafana/loki/pkg/logproto"
	"github.com/grafana/loki/pkg/util/log"
	_config "github.com/groundcover-com/loki-proxy/config"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/promql/parser"
)

const (
	HEALTHCHECK_ENDPOINT  = "/health"
	PUSH_ENDPOINT         = "/loki/api/v1/push"
	TENANT_ID_HEADER_NAME = "X-Scope-OrgID"
)

var (
	config       *_config.Config
	reverseProxy *httputil.ReverseProxy
)

func main() {
	config = _config.NewConfig()
	config.Print()

	targetUrl := config.TargetUrl()

	reverseProxy = &httputil.ReverseProxy{
		Director: func(request *http.Request) {
			request.URL = targetUrl
			request.RequestURI = ""
			request.Host = targetUrl.Host
			request.Header.Set(TENANT_ID_HEADER_NAME, config.Target.TenantId)
		},
	}

	router := gin.Default()
	router.POST(PUSH_ENDPOINT, handlePushRequest)
	router.GET(HEALTHCHECK_ENDPOINT, handleHealthCheck)
	router.Run(config.BindAddr())
}

func handleHealthCheck(c *gin.Context) {
	c.String(http.StatusOK, "ok")
}

func handlePushRequest(c *gin.Context) {
	var err error

	tenantId := c.Request.Header.Get(TENANT_ID_HEADER_NAME)

	if tenantId != "" {
		var pushRequest *logproto.PushRequest
		if pushRequest, err = push.ParseRequest(log.Logger, tenantId, c.Request, nil); err != nil {
			return
		}

		if err = appendTenantLabelToPushRequest(tenantId, pushRequest); err != nil {
			c.Writer.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err = rewriteRequestBody(c.Request, pushRequest); err != nil {
			c.Writer.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	reverseProxy.ServeHTTP(c.Writer, c.Request)
}

func appendTenantLabelToPushRequest(tenantId string, pushRequest *logproto.PushRequest) error {
	var err error

	tenantLabel := labels.Label{
		Name:  config.Target.LabelName,
		Value: tenantId,
	}

	for index, stream := range pushRequest.Streams {
		var streamLabels labels.Labels
		if streamLabels, err = parser.ParseMetric(stream.Labels); err != nil {
			return err
		}

		streamLabels = append(streamLabels, tenantLabel)
		pushRequest.Streams[index].Labels = streamLabels.String()
	}

	return nil
}

func rewriteRequestBody(request *http.Request, pushRequest *logproto.PushRequest) error {
	var err error

	var buffer []byte
	if buffer, err = proto.Marshal(pushRequest); err != nil {
		return err
	}
	buffer = snappy.Encode(nil, buffer)

	request.ContentLength = int64(len(buffer))
	request.Body = io.NopCloser(bytes.NewReader(buffer))

	return nil
}
