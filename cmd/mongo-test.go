package main

import (
	"context"
	"github.com/scbunn/mongo-test-harness/pkg/templates"
	"os"
	"strconv"
	"sync"
	"text/template"
	"time"

	// 3rd Party
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	mgoBSON "gopkg.in/mgo.v2/bson"
)

var (
	renderLatency = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "mongo_test_harness",
			Name:      "render_latency",
			Help:      "There is no helping you",
			Buckets:   []float64{0.01, 0.02, 0.03, 0.04, 0.05, 0.10, 0.20, 0.30, 0.40, 1},
		},
	)

	requestCount = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "mongo_test_harness",
		Name:      "request_count_total",
	})
)

func render(name string, t *template.Template) string {
	start := time.Now()
	tpl, err := templates.RenderTemplate(name, t)
	if err != nil {
		log.Fatal(err)
	}
	duration := time.Since(start)
	renderLatency.Observe(duration.Seconds())
	return tpl
}

func pushMetrics(pusher *push.Pusher) error {
	err := pusher.Add()
	if err != nil {
		return err
	}
	return nil
}

func main() {
	log.SetFormatter(&log.TextFormatter{
		DisableColors: false,
		FullTimestamp: true,
	})
	log.Info("Starting")

	// MongoDB Setup
	timeout := 1 * time.Second
	mongoOptions := options.Client().ApplyURI("mongodb://localhost:27017")
	mongoOptions.ConnectTimeout = &timeout
	mongoOptions.SocketTimeout = &timeout
	mongoOptions.ServerSelectionTimeout = &timeout

	mongoClient, err := mongo.Connect(context.TODO(), mongoOptions)
	if err != nil {
		log.Fatal(err)
	}

	var wg sync.WaitGroup
	registry := prometheus.NewRegistry()
	registry.MustRegister(renderLatency)
	registry.MustRegister(requestCount)
	hostname, _ := os.Hostname()
	pusher := push.New("http://localhost:9091", "foo_job").Gatherer(registry).
		Grouping("instance", hostname)

	t, err := templates.ParseTemplates("templates")
	if err != nil {
		log.Fatal(err)
	}

	templateChan := make(chan string, 1) // keep 20 rendered templates around
	exitChan := make(chan bool)
	go generateDocuments("file1.template", t, exitChan, templateChan)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go makeRequest(templateChan, &wg, i, mongoClient)
	}
	log.Info("Making requests until timeout...")
	wg.Wait()

	// flush all metrics to the push gateway
	if err = pushMetrics(pusher); err != nil {
		log.Error(err)
	}
	log.Info("Done")
}

func generateDocuments(name string, t *template.Template, exitChan chan bool,
	tplChan chan string) {
	var count int
	for {
		select {
		case <-exitChan:
			log.WithFields(log.Fields{
				"templates generated": count,
			}).Info("gernatedDocuments asked to quit")
			return
		default:
		}

		// if the channel buffer isn't full render another one and push it
		// otherwise block
		tplChan <- render(name, t)
		count++
	}
}

func makeRequest(t chan string, waitGroup *sync.WaitGroup, id int, client *mongo.Client) {
	// make requests for 30 seconds and then exit
	log.Info("Staring request " + strconv.Itoa(id))
	defer waitGroup.Done()
	timeout := time.After(30 * time.Second)
	var newCount int
	var oldCount int
	tpl := <-t // block until we get our first template
	var document interface{}
	err := mgoBSON.UnmarshalJSON([]byte(tpl), &document)
	if err != nil {
		log.WithFields(log.Fields{
			"error":    err,
			"template": tpl,
		}).Error("Error converting template to BSON")
		return
	}

	log.WithFields(log.Fields{
		"go routine id": id,
	}).Info("Got first template")
	collection := client.Database("testing").Collection("one")
	for {
		select {
		case <-timeout:
			log.WithFields(log.Fields{
				"go routine id":          id,
				"new templates received": newCount,
				"old templates used":     oldCount,
			}).Info("request go routine timed out")
			return
			//		case tpl = <-t:
			//			newCount++ // a new template is on the channel, use it
		default:
			oldCount++ // keep using the template that you have
		}
		// make a mongo request
		requestCount.Inc()
		collection.InsertOne(context.TODO(), document)
	}
}
