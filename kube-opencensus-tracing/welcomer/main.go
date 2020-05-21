package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptrace"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/opentracing-contrib/go-gin/ginhttp"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"

	"go.opencensus.io/trace"

	"go.opencensus.io/plugin/ochttp"
)

var (
	guestrackerhost = os.Getenv("GUEST_TRACKER_HOST")
)

func main() {
	if guestrackerhost == "" {
		guestrackerhost = "localhost:8081"
	}
	fmt.Println("GUEST_TRACKER_HOST =", guestrackerhost)

	collectorEndpointURI := "http://localhost:14268/api/traces"

	cfg := &config.Configuration{
		Sampler: &config.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &config.ReporterConfig{
			LogSpans:          true,
			CollectorEndpoint: collectorEndpointURI,
		},
	}
	tracer, closer, err := cfg.New(
		"welcomer",
		config.Logger(jaeger.StdLogger),
	)

	opentracing.SetGlobalTracer(tracer)
	if err != nil {
		log.Fatal("error in jaeger init", err)
	}

	defer closer.Close()

	r := gin.Default()
	r.Use(ginhttp.Middleware(opentracing.GlobalTracer()))
	r.GET("/welcome", func(c *gin.Context) {
		//_, span := trace.StartSpan(c, "/welcome")
		// http_server_route=/welcome tag is set
		// ochttp.SetRoute(c.Request.Context(), "/welcome")

		log.Println("Logss.... please come in j")
		//defer span.End()
		fmt.Println(c.Request.Header)
		fmt.Println(c.Request.Host)

		welcomeHandler(c)
	})
	//r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
	http.ListenAndServe( // nolint: errcheck
		"0.0.0.0:8080", r)
}

func welcomeHandler(c *gin.Context) {
	//Send post request to another service
	guesttracker(c)
	c.JSON(200, gin.H{
		"message": "Hello Folks .. You are welcome(Shhh... and also tracked by guesttracker)!!",
	})
}

func guesttracker(c *gin.Context) {
	reqBody, err := json.Marshal(map[string]string{
		"username": "Bruce Wayne",
		"email":    "batman@loreans.com",
	})
	if err != nil {
		print(err)
	}

	client := &http.Client{Transport: &ochttp.Transport{}}
	fmt.Println(c)
	fmt.Println(c.Request.Context())
	context := c.Request.Context()
	span := trace.FromContext(c)
	defer span.End()
	span.Annotate([]trace.Attribute{trace.StringAttribute("annotated", "welcomervalue")}, "welcomervalue-->guesttracker annotation check")
	span.AddAttributes(trace.StringAttribute("span-add-attribute", "welcomervalue"))
	time.Sleep(time.Millisecond * 125)

	r, _ := http.NewRequest("POST", "http://"+guestrackerhost+"/track-guest", bytes.NewBuffer(reqBody))
	clientTrace := ochttp.NewSpanAnnotatingClientTrace(r, span)
	context = httptrace.WithClientTrace(context, clientTrace)
	r = r.WithContext(context)

	resp, err := client.Do(r)

	if err != nil {
		print(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		print(err)
	}
	fmt.Println(string(body))

}
