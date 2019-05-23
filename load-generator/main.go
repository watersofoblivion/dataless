package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
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

	"github.com/cenkalti/backoff"
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

type Users struct {
	Users []uuid.UUID
}

func NewUsers(n int) *Users {
	users := &Users{Users: make([]uuid.UUID, n)}
	for i := 0; i < n; i++ {
		users.Users[i] = uuid.New()
	}
	return users
}

type Batch map[string]Records

type Records []Record

type Record map[string]interface{}

type Response struct {
	FailedCount int               `json:"failed_count"`
	Records     []*ResponseRecord `json:"records"`
}

type ResponseRecord struct {
	Failed       bool   `json:"failed"`
	ErrorCode    string `json:"error_code"`
	ErrorMessage string `json:"error_message"`
}

func jitter(d time.Duration, variability float64) time.Duration {
	dInt64 := int64(d)
	fD := float64(dInt64)
	r := rand.Float64() * fD * variability
	slid := r - (r / 2.0)
	return time.Duration(int64(fD + slid))
}

func batcher(wg *sync.WaitGroup, batchKey string, records <-chan Record, batches chan<- Batch) {
	defer wg.Done()
	defer close(batches)

	rs := make(Records, 0, MaxBatchSize)
	defer func() {
		if len(records) > 0 {
			batches <- Batch{batchKey: rs}
		}
	}()

	for record := range records {
		rs = append(rs, record)
		if len(rs) == MaxBatchSize {
			batches <- Batch{batchKey: rs}
			rs = make(Records, 0, MaxBatchSize)
		}
	}
}

func publisher(ctx context.Context, wg *sync.WaitGroup, batches <-chan Batch, endpoint string) {
	defer wg.Done()

	for batch := range batches {
		publish := publishBatch(endpoint, batch)
		eb := backoff.WithContext(backoff.NewExponentialBackOff(), ctx)
		if err := backoff.RetryNotify(publish, eb, notifyBackoff); err != nil {
			log.Printf("events dropped: %s", err)
		}
	}
}

func publishBatch(endpoint string, batch Batch) func() error {
	buf := new(bytes.Buffer)
	encoder := json.NewEncoder(buf)

	return func() error {
		buf.Reset()
		if err := encoder.Encode(batch); err != nil {
			return fmt.Errorf("could not encode events batch: %s", err)
		}

		resp, err := http.Post(endpoint, "application/json", buf)
		if err != nil {
			return fmt.Errorf("error publishing events batch: %s", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("non-%d status code publishing events batch: %d", http.StatusOK, resp.StatusCode)
		}

		response := new(Response)
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return fmt.Errorf("error decoding batch write response: %s", err)
		}

		newRecords := make(Records, 0, len(batch["records"]))
		for i, record := range response.Records {
			if record.Failed {
				newRecords = append(newRecords, batch["records"][i])
			}
		}

		if numRetry := len(newRecords); numRetry > 0 {
			batch = Batch{"records": newRecords}
			return fmt.Errorf("Retrying %d events", numRetry)
		}

		fmt.Print(".")
		return nil
	}
}

func notifyBackoff(err error, bo time.Duration) {
	fmt.Println()
	log.Printf("%s.  Backing off %s", err, bo)
}

func simulateUser(wg *sync.WaitGroup, usersWg *sync.WaitGroup, ctx context.Context, config *Config, userID uuid.UUID, ads <-chan uuid.UUID, records chan<- Record) {
	defer wg.Done()
	defer usersWg.Done()

	sessionID := uuid.New()
	for ad := range ads {
		contextID := uuid.New()
		impressionID := uuid.New()
		records <- Record{
			"session_id":  sessionID,
			"context_id":  contextID,
			"actor_type":  "customer",
			"actor_id":    userID,
			"event_type":  "impression",
			"event_id":    impressionID,
			"object_type": "ad",
			"object_id":   ad,
			"occurred_at": time.Now(),
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(config.Users.ConsiderDuration):
		}

		if rand.Float64() < config.Users.ClickthroughProbability {
			records <- Record{
				"session_id":  sessionID,
				"context_id":  contextID,
				"parent_id":   impressionID,
				"actor_type":  "customer",
				"actor_id":    userID,
				"event_type":  "click",
				"event_id":    uuid.New(),
				"object_type": "ad",
				"object_id":   ad,
				"occurred_at": time.Now(),
			}
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(config.Users.ViewFrequency):
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
	earl := baseURL.ResolveReference(&url.URL{
		Path: filepath.Join(baseURL.Path, "/data/events"),
	}).String()

	ads := NewAds(config.Ads.Count)
	users := NewUsers(config.Users.Count)

	wg := new(sync.WaitGroup)
	ctx, cancel := context.WithTimeout(context.Background(), config.Duration)

	batches := make(chan Batch)
	wg.Add(1)
	go publisher(ctx, wg, batches, earl)

	records := make(chan Record)
	wg.Add(1)
	go batcher(wg, "events", records, batches)

	usersWg := new(sync.WaitGroup)
	go func() {
		usersWg.Wait()
		close(records)
	}()

	adStream := make(chan uuid.UUID)
	wg.Add(config.Users.Count)
	usersWg.Add(config.Users.Count)

	for i := 0; i < config.Users.Count; i++ {
		go simulateUser(wg, usersWg, ctx, config, users.Users[i], adStream, records)
	}

	signals := make(chan os.Signal)
	go func() {
		<-signals
		cancel()
	}()
	signal.Notify(signals, os.Interrupt)

	defer wg.Wait()
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			close(adStream)
			return
		case adStream <- ads.Random():
		}
	}
}
