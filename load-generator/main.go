package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	yaml "gopkg.in/yaml.v2"
)

const MaxBatchSize = 500

type Config struct {
	BaseURL  string        `yaml:"BaseURL"`
	Duration time.Duration `yaml:"Duration"`
	Ads      *AdsConfig    `yaml:"Ads"`
	Users    *UsersConfig  `yaml:"Users"`
}

func LoadConfig(path string) (*Config, error) {
	bs, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, err
	}
	bs = []byte(os.ExpandEnv(string(bs)))

	config := new(Config)
	return config, yaml.Unmarshal(bs, config)
}

type AdsConfig struct {
	Count int `yaml:"Count"`
}

type UsersConfig struct {
	Count                   int           `yaml:"Count"`
	ClickthroughProbability float64       `yaml:"ClickthroughProbability"`
	ViewFrequency           time.Duration `yaml:"ViewFrequency"`
	ViewVariance            float64       `yaml:"ViewVariance"`
	ConsiderDuration        time.Duration `yaml:"ConsiderDuration"`
	ConsiderVariance        float64       `yaml:"ConsiderVariance"`
}

type Ads struct {
	Ads []uuid.UUID
}

func NewAds(n int) *Ads {
	ads := &Ads{Ads: make([]uuid.UUID, n)}
	for i := 0; i < n; i++ {
		ads.Ads[i] = uuid.New()
	}
	return ads
}

func (ads *Ads) Random() uuid.UUID {
	return ads.Ads[rand.Intn(len(ads.Ads))]
}

type Batch map[string]Records

type Records []Record

type Record map[string]interface{}

type Publisher struct {
	URL      string
	BatchKey string
	wg       *sync.WaitGroup
	records  chan Record
	errors   chan error
	done     chan struct{}
}

func NewPublisher(url, batchKey string) *Publisher {
	return &Publisher{
		URL:      url,
		BatchKey: batchKey,
		wg:       new(sync.WaitGroup),
		records:  make(chan Record),
		errors:   make(chan error),
	}
}

func (publisher *Publisher) Records() chan<- Record {
	return publisher.records
}

func (publisher *Publisher) Go(ctx context.Context) {
	defer close(publisher.errors)

	batches := make(chan Batch)
	bodies := make(chan io.Reader)

	publisher.wg.Add(3)
	go publisher.batch(publisher.records, batches)
	go publisher.marshal(batches, bodies, publisher.errors)
	go publisher.post(bodies, publisher.errors)

	publisher.wg.Wait()
}

func (publisher *Publisher) Errors() <-chan error {
	return publisher.errors
}

func (publisher *Publisher) batch(records <-chan Record, batches chan<- Batch) {
	defer close(batches)

	batch := make(Records, 0, 500)
	defer func() {
		if len(batch) > 0 {
			batches <- Batch{
				publisher.BatchKey: batch,
			}
		}
	}()

	for record := range records {
		batch = append(batch, record)
		if len(batch) == 500 {
			batches <- Batch{
				publisher.BatchKey: batch,
			}
			batch = make(Records, 0, 500)
		}
	}
}

func (publisher *Publisher) marshal(batches <-chan Batch, bodies chan<- io.Reader, errors chan<- error) {
	defer close(bodies)

	buf := new(bytes.Buffer)
	encoder := json.NewEncoder(buf)

	for batch := range batches {
		if err := encoder.Encode(batch); err != nil {
			errors <- fmt.Errorf("could not encode events batch: %s", err)
		} else {
			bodies <- bytes.NewReader(buf.Bytes())
			buf.Reset()
		}
	}
}

func (publisher *Publisher) post(bodies <-chan io.Reader, errors chan<- error) {
	for body := range bodies {
		resp, err := http.Post(publisher.URL, "application/json", body)
		if err != nil {
			fmt.Print("E")
			errors <- err
			continue
		}

		if resp.StatusCode != http.StatusOK {
			fmt.Print("F")
			errors <- fmt.Errorf("non-%d status code publishing events batch: %d", http.StatusOK, resp.StatusCode)
			continue
		}

		defer resp.Body.Close()
		response := new(Response)
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			fmt.Print("F")
			errors <- fmt.Errorf("error decoding batch write response: %s", err)
			continue
		}

		for _, record := range response.Records {
			if record.Failed {
				errors <- fmt.Errorf("%s: %s", record.ErrorCode, record.ErrorMessage)
			}
		}

		fmt.Print(".")
	}
}

type Response struct {
	FailedCount int               `json:"failed_count"`
	Records     []*ResponseRecord `json:"records"`
}

type ResponseRecord struct {
	Failed       bool   `json:"failed"`
	ErrorCode    string `json:"error_code"`
	ErrorMessage string `json:"error_message"`
}

type Users struct {
	Config      *Config
	wg          *sync.WaitGroup
	ads         chan uuid.UUID
	impressions chan<- Record
	clicks      chan<- Record
}

func NewUsers(config *Config, impressions, clicks chan<- Record) *Users {
	return &Users{
		Config:      config,
		wg:          new(sync.WaitGroup),
		ads:         make(chan uuid.UUID),
		impressions: impressions,
		clicks:      clicks,
	}
}

func (users *Users) Ads() chan<- uuid.UUID {
	return users.ads
}

func (users *Users) Go(ctx context.Context) {
	defer close(users.impressions)
	defer close(users.clicks)

	users.wg.Add(users.Config.Users.Count)
	for i := 0; i < users.Config.Users.Count; i++ {
		go users.simulate(ctx)
	}

	users.wg.Wait()
}

func (users *Users) simulate(ctx context.Context) {
	defer users.wg.Done()

	for ad := range users.ads {
		impressionID := uuid.New()
		users.impressions <- Record{
			"impression_id": impressionID.String(),
			"ad_id":         ad.String(),
			"occurred_at":   time.Now().UTC().Format("2006-01-02 15:04:05"),
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(users.Config.Users.ConsiderDuration):
		}

		if rand.Float64() < users.Config.Users.ClickthroughProbability {
			users.clicks <- Record{
				"impression_id": impressionID.String(),
				"click_id":      uuid.New().String(),
				"ad_id":         ad.String(),
				"occurred_at":   time.Now().UTC().Format("2006-01-02 15:04:05"),
			}
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(users.Config.Users.ViewFrequency):
		}
	}
}

var (
	configFile string
)

func main() {
	flag.StringVar(&configFile, "c", configFile, "The config file for the load test")
	flag.Parse()

	config, err := LoadConfig(configFile)
	if err != nil {
		log.Panic(err)
	}

	baseURL, err := url.Parse(config.BaseURL)
	if err != nil {
		log.Panicf("could not parse base URL: %s", err)
	}

	ads := NewAds(config.Ads.Count)

	impressions := NewPublisher(baseURL.ResolveReference(&url.URL{
		Path: filepath.Join(baseURL.Path, "/impressions"),
	}).String(), "impressions")
	clicks := NewPublisher(baseURL.ResolveReference(&url.URL{
		Path: filepath.Join(baseURL.Path, "/clicks"),
	}).String(), "clicks")
	users := NewUsers(config, impressions.Records(), clicks.Records())

	ctx, cancel := context.WithTimeout(context.Background(), config.Duration)

	go impressions.Go(ctx)
	go clicks.Go(ctx)
	go users.Go(ctx)

	wg := new(sync.WaitGroup)
	wg.Add(2)
	go func() {
		defer wg.Done()
		for err := range impressions.Errors() {
			fmt.Println()
			log.Printf("%s", err)
		}
	}()
	go func() {
		defer wg.Done()
		for err := range clicks.Errors() {
			fmt.Println()
			log.Printf("%s", err)
		}
	}()

	defer wg.Wait()

	signals := make(chan os.Signal)
	go func() {
		<-signals
		cancel()
	}()
	signal.Notify(signals, os.Interrupt)

	for {
		select {
		case <-ctx.Done():
			close(users.Ads())
			return
		case users.Ads() <- ads.Random():
		}
	}
}
