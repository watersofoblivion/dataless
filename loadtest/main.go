package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
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

type Config struct {
	BaseURL  string        `yaml:"BaseURL"`
	Duration time.Duration `yaml:"Duration"`
	Ads      *AdsConfig    `yaml:"Ads"`
	Users    *UsersConfig  `yaml:"Users"`
}

type AdsConfig struct {
	Number int `yaml:"Number"`
}

type UsersConfig struct {
	Number                  int           `yaml:"Number"`
	ClickthroughProbability float64       `yaml:"ClickthroughProbability"`
	ViewFrequency           time.Duration `yaml:"ViewFrequency"`
	ViewVariance            float64       `yaml:"ViewVariance"`
	ConsiderDuration        time.Duration `yaml:"ConsiderDuration"`
	ConsiderVariance        float64       `yaml:"ConsiderVariance"`
}

type Events struct {
	Events []*Event `json:"events"`
}

type Event struct {
	Session    uuid.UUID
	Context    uuid.UUID
	Parent     uuid.UUID
	ActorType  string
	Actor      uuid.UUID
	EventType  string
	Event      uuid.UUID
	ObjectType string
	Object     uuid.UUID
	OccurredAt time.Time
}

func (event *Event) MarshalJSON() ([]byte, error) {
	evt := map[string]interface{}{
		"session_id":  event.Session.String(),
		"context_id":  event.Context.String(),
		"actor_type":  event.ActorType,
		"actor_id":    event.Actor.String(),
		"event_type":  event.EventType,
		"event_id":    event.Event.String(),
		"object_type": event.ObjectType,
		"object_id":   event.Object.String(),
		"occurred_at": event.OccurredAt.Format("2006-01-02 15:04:05.000"),
	}

	if event.Parent != uuid.Nil {
		evt["parent_id"] = event.Parent.String()
	}

	return json.Marshal(evt)
}

type BatchWriteResponse struct {
	Records []*BatchWriteRecord `json:"records"`
}

type BatchWriteRecord struct {
	Failed       bool   `json:"failed"`
	ErrorCode    string `json:"error_code"`
	ErrorMessage string `json:"error_message"`
}

func (record *BatchWriteRecord) Error() string {
	if record.Failed {
		return fmt.Sprintf("%s: %s", record.ErrorCode, record.ErrorMessage)
	}

	return ""
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

func jitter(d time.Duration, variability float64) time.Duration {
	dInt64 := int64(d)
	fD := float64(dInt64)
	r := rand.Float64() * fD * variability
	slid := r - (r / 2.0)
	return time.Duration(int64(fD + slid))
}

var (
	configFile string
)

func main() {
	flag.StringVar(&configFile, "c", configFile, "The config file for the load test")
	flag.Parse()

	f, err := os.OpenFile(configFile, os.O_RDONLY, 0)
	if err != nil {
		log.Panic(err)
	}
	defer f.Close()

	config := new(Config)
	if err := yaml.NewDecoder(f).Decode(config); err != nil {
		log.Panic(err)
	}

	baseURL, err := url.Parse(os.ExpandEnv(config.BaseURL))
	if err != nil {
		log.Panicf("could not parse base URL: %s", err)
	}

	ads := NewAds(config.Ads.Number)
	users := NewUsers(config.Users.Number)

	wg := new(sync.WaitGroup)
	ctx, cancel := context.WithTimeout(context.Background(), config.Duration)

	eventBatches := make(chan *Events)
	wg.Add(1)
	go func() {
		defer wg.Done()

		buf := new(bytes.Buffer)
		encoder := json.NewEncoder(buf)
		earl := baseURL.ResolveReference(&url.URL{
			Path: filepath.Join(baseURL.Path, "/data/events"),
		}).String()

		for batch := range eventBatches {
			eb := backoff.WithContext(backoff.NewExponentialBackOff(), ctx)
			err := backoff.RetryNotify(func() error {
				buf.Reset()
				if err := encoder.Encode(batch); err != nil {
					return fmt.Errorf("could not encode events batch: %s", err)
				}

				resp, err := http.Post(earl, "application/json", buf)
				if err != nil {
					return fmt.Errorf("error publishing events batch: %s", err)
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					return fmt.Errorf("non-%d status code publishing events batch: %d", http.StatusOK, resp.StatusCode)
				}

				batchWriteResponse := new(BatchWriteResponse)
				if err := json.NewDecoder(resp.Body).Decode(batchWriteResponse); err != nil {
					return fmt.Errorf("error decoding batch write response: %s", err)
				}

				newBatch := new(Events)
				for i, record := range batchWriteResponse.Records {
					if record.Failed {
						newBatch.Events = append(newBatch.Events, batch.Events[i])
					}
				}

				if numRetry := len(newBatch.Events); numRetry > 0 {
					batch = newBatch
					return fmt.Errorf("Retrying %d events", numRetry)
				}

				fmt.Print(".")
				return nil
			}, eb, func(err error, bo time.Duration) {
				fmt.Println()
				log.Printf("%s.  Backing off %s", err, bo)
			})
			if err != nil {
				log.Printf("events dropped: %s", err)
			}
		}
	}()

	events := make(chan *Event)
	wg.Add(1)
	go func() {
		defer wg.Done()
		batch := new(Events)

		defer func() {
			if len(batch.Events) > 0 {
				eventBatches <- batch
			}

			close(eventBatches)
		}()

		for impression := range events {
			batch.Events = append(batch.Events, impression)
			if len(batch.Events) == 500 {
				eventBatches <- batch
				batch = new(Events)
			}
		}
	}()

	usersWg := new(sync.WaitGroup)
	go func() {
		usersWg.Wait()
		close(events)
	}()

	adStream := make(chan uuid.UUID)
	wg.Add(config.Users.Number)
	usersWg.Add(config.Users.Number)
	for i := 0; i < config.Users.Number; i++ {
		i := i
		go func() {
			defer wg.Done()
			defer usersWg.Done()

			sessionID := uuid.New()

			for ad := range adStream {
				contextID := uuid.New()
				impressionID := uuid.New()

				events <- &Event{
					Session:    sessionID,
					Context:    contextID,
					ActorType:  "customer",
					Actor:      users.Users[i],
					EventType:  "impression",
					Event:      impressionID,
					ObjectType: "ad",
					Object:     ad,
					OccurredAt: time.Now(),
				}

				select {
				case <-ctx.Done():
					return
				case <-time.After(config.Users.ConsiderDuration):
				}

				if rand.Float64() < config.Users.ClickthroughProbability {
					events <- &Event{
						Session:    sessionID,
						Context:    contextID,
						Parent:     impressionID,
						ActorType:  "customer",
						Actor:      users.Users[i],
						EventType:  "click",
						Event:      uuid.New(),
						ObjectType: "ad",
						Object:     ad,
						OccurredAt: time.Now(),
					}
				}

				select {
				case <-ctx.Done():
					return
				case <-time.After(config.Users.ViewFrequency):
				}
			}
		}()
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
