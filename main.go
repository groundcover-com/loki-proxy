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
	"github.com/groundcover-com/loki-proxy/middlewares"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/promql/parser"
	"go.uber.org/zap"
)

const (
	HEALTHCHECK_ENDPOINT  = "/health"
	CONFIG_ENDPOINT       = "/config"
	PUSH_ENDPOINT         = "/loki/api/v1/push"
	TENANT_ID_HEADER_NAME = "X-Scope-OrgID"
)

var (
	config       *_config.Config
	reverseProxy *httputil.ReverseProxy
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	config, err := _config.NewConfig()
	if err != nil {
		logger.Fatal("failed to load config", zap.String("error", err.Error()))
	}
	config.Print()

	targetUrl, err := config.TargetUrl()
	if err != nil {
		logger.Fatal("failed to parse target URL", zap.String("error", err.Error()))
	}

	reverseProxy = &httputil.ReverseProxy{
		Director: func(request *http.Request) {
			request.URL = targetUrl
			request.RequestURI = ""
			request.Host = targetUrl.Host
			request.Header.Set(TENANT_ID_HEADER_NAME, config.Target.TenantId)
		},
	}

	router := gin.Default()
	router.Use(middlewares.MetricsMiddleware)
	router.POST(PUSH_ENDPOINT, handlePushRequest)
	router.GET(HEALTHCHECK_ENDPOINT, handleHealthCheck)
	router.GET(CONFIG_ENDPOINT, createHandleConfigEndpoint(config))
	router.GET(middlewares.METRICS_ENDPOINT, gin.WrapH(promhttp.Handler()))
	router.Run(config.BindAddr())
}

func createHandleConfigEndpoint(config *_config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.YAML(http.StatusOK, config)
	}
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
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		if err = rewriteRequestBody(c.Request, pushRequest); err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
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
