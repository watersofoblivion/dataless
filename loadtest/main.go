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
	Events   int64         `yaml:"Events"`
	Tags     int           `yaml:"Tags"`
	Ads      *AdsConfig    `yaml:"Ads"`
	Users    *UsersConfig  `yaml:"Users"`
}

type AdsConfig struct {
	Number int `yaml:"Number"`
	Tags   int `yaml:"Tags"`
}

type UsersConfig struct {
	Number                  int               `yaml:"Number"`
	Tags                    int               `yaml:"Tags"`
	MaximumClickThroughRate float64           `yaml:"MaximumClickThroughRate"`
	ViewFrequency           *VariableDuration `yaml:"ViewFrequency"`
	ViewVariance            *VariableDuration `yaml:"ViewVariance"`
	ConsiderDuration        *VariableDuration `yaml:"ConsiderDuration"`
	ConsiderVariance        *VariableDuration `yaml:"ConsiderVariance"`
}

type VariableDuration struct {
	Interval time.Duration `yaml:"Interval"`
	Variance time.Duration `yaml:"Variance"`
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
	Ads []*Ad
}

func NewAds(nAds, adTags, nTags int) *Ads {
	ads := &Ads{Ads: make([]*Ad, nAds)}

	for i := 0; i < nAds; i++ {
		ad := &Ad{
			ID:   uuid.New(),
			Tags: make([]int, adTags),
		}
		ads.Ads[i] = ad

		for j := 0; j < adTags; j++ {
			ad.Tags[j] = rand.Intn(nTags)
		}
	}

	return ads
}

func (ads *Ads) Random() *Ad {
	return ads.Ads[rand.Intn(len(ads.Ads))]
}

type Ad struct {
	ID   uuid.UUID
	Tags []int
}

type Users struct {
	Users []*User
}

func NewUsers(nUsers, userTags, nTags int, viewFrequency, viewVariance, considerDuration, considerVariance, viewFrequencyVariance, viewVarianceVariance, considerDurationVariance, considerVarianceVariance int64) *Users {
	users := &Users{Users: make([]*User, nUsers)}

	for i := 0; i < nUsers; i++ {
		users.Users[i] = &User{
			ID:               uuid.New(),
			Tags:             make([]int, userTags),
			ViewFrequency:    viewFrequency + (rand.Int63n(viewFrequencyVariance) - (viewFrequencyVariance / 2)),
			ViewVariance:     viewVariance + (rand.Int63n(viewVarianceVariance) - (viewVarianceVariance / 2)),
			ConsiderDuration: considerDuration + (rand.Int63n(considerDurationVariance) - (considerDurationVariance / 2)),
			ConsiderVariance: considerVariance + (rand.Int63n(considerVarianceVariance) - (considerVarianceVariance / 2)),
		}

		for j := 0; j < userTags; j++ {
			users.Users[i].Tags[j] = rand.Intn(nTags)
		}
	}

	return users
}

type User struct {
	ID               uuid.UUID
	ViewFrequency    int64
	ViewVariance     int64
	ConsiderDuration int64
	ConsiderVariance int64
	Tags             []int
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

	ads := NewAds(config.Ads.Number, config.Ads.Tags, config.Tags)
	users := NewUsers(
		config.Users.Number, config.Users.Tags, config.Tags,
		int64(config.Users.ViewFrequency.Interval), int64(config.Users.ViewVariance.Interval), int64(config.Users.ConsiderDuration.Interval), int64(config.Users.ConsiderVariance.Interval),
		int64(config.Users.ViewFrequency.Variance), int64(config.Users.ViewVariance.Variance), int64(config.Users.ConsiderDuration.Variance), int64(config.Users.ConsiderVariance.Variance),
	)

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

				return nil
			}, eb, func(err error, bo time.Duration) {
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

	adStream := make(chan *Ad)
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
					Actor:      users.Users[i].ID,
					EventType:  "impression",
					Event:      impressionID,
					ObjectType: "ad",
					Object:     ad.ID,
					OccurredAt: time.Now(),
				}

				considerTime := users.Users[i].ConsiderDuration + (rand.Int63n(users.Users[i].ConsiderVariance) - (users.Users[i].ConsiderVariance / 2))
				select {
				case <-ctx.Done():
					return
				case <-time.After(time.Duration(considerTime)):
				}

				m := make(map[int]struct{})
				common := 0
				for tag := range users.Users[i].Tags {
					m[tag] = struct{}{}
				}
				for tag := range ad.Tags {
					if _, found := m[tag]; found {
						common++
					} else {
						m[tag] = struct{}{}
					}
				}
				clickthroughProbability := float64(common) / float64(len(m)) * config.Users.MaximumClickThroughRate
				if rand.Float64() < clickthroughProbability {
					events <- &Event{
						Session:    sessionID,
						Context:    contextID,
						Parent:     impressionID,
						ActorType:  "customer",
						Actor:      users.Users[i].ID,
						EventType:  "click",
						Event:      uuid.New(),
						ObjectType: "ad",
						Object:     ad.ID,
						OccurredAt: time.Now(),
					}
				}

				sleepTime := users.Users[i].ViewFrequency + (rand.Int63n(users.Users[i].ViewVariance) - (users.Users[i].ViewVariance / 2))
				select {
				case <-ctx.Done():
					return
				case <-time.After(time.Duration(sleepTime)):
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

outer:
	for i := int64(0); i < config.Events; i++ {
		select {
		case <-ctx.Done():
			close(adStream)
			break outer
		case adStream <- ads.Random():
		}
	}

	cancel()
	wg.Wait()
}
