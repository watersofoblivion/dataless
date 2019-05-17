package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"io"
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

type AdImpressions struct {
	Impressions []*AdImpression `json:"impressions"`
}

type AdImpression struct {
	Ad   uuid.UUID `json:"ad"`
	User uuid.UUID `json:"user"`
	At   time.Time `json:"at"`
}

// func (datum *AdImpression) MarshalJSON() ([]byte, error) {
// 	return json.Marshal(map[string]interface{}{
// 		"Ad":   datum.Ad.String(),
// 		"User": datum.User.String(),
// 		"At":   datum.At.Format("2006-01-02 15:04:05.000"),
// 	})
// }

type AdClicks struct {
	Clicks []*AdClick `json:"clicks"`
}

type AdClick struct {
	Ad   uuid.UUID `json:"ad"`
	User uuid.UUID `json:"user"`
	At   time.Time `json:"at"`
}

// func (datum *AdClick) MarshalJSON() ([]byte, error) {
// 	return json.Marshal(map[string]interface{}{
// 		"Ad":   datum.Ad.String(),
// 		"User": datum.User.String(),
// 		"At":   datum.At.Format("2006-01-02 15:04:05.000"),
// 	})
// }

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

	impressionBatches := make(chan *AdImpressions)
	wg.Add(1)
	go func() {
		defer wg.Done()

		buf := new(bytes.Buffer)
		encoder := json.NewEncoder(buf)
		earl := baseURL.ResolveReference(&url.URL{
			Path: filepath.Join(baseURL.Path, "/data/ad/impressions"),
		}).String()

		for batch := range impressionBatches {
			buf.Reset()
			if err := encoder.Encode(batch); err != nil {
				log.Printf("could not encode impressions batch: %s", err)
			}

			resp, err := http.Post(earl, "application/json", buf)
			if err != nil {
				log.Printf("error publishing impressions batch: %s", err)
			}

			if resp.StatusCode != http.StatusOK {
				log.Printf("non-%d status code publishing impressions batch: %d", http.StatusOK, resp.StatusCode)
				defer resp.Body.Close()
				io.Copy(os.Stderr, resp.Body)
			}
		}
	}()

	clickBatches := make(chan *AdClicks)
	wg.Add(1)
	go func() {
		defer wg.Done()

		buf := new(bytes.Buffer)
		encoder := json.NewEncoder(buf)
		earl := baseURL.ResolveReference(&url.URL{
			Path: filepath.Join(baseURL.Path, "/data/ad/clicks"),
		}).String()

		for batch := range clickBatches {
			buf.Reset()
			if err := encoder.Encode(batch); err != nil {
				log.Printf("could not encode impressions batch: %s", err)
			}

			resp, err := http.Post(earl, "application/json", buf)
			if err != nil {
				log.Printf("error publishing impressions batch: %s", err)
			}

			if resp.StatusCode != http.StatusOK {
				log.Printf("non-%d status code publishing impressions batch: %d", http.StatusOK, resp.StatusCode)
				defer resp.Body.Close()
				io.Copy(os.Stderr, resp.Body)
			}
		}
	}()

	impressions := make(chan *AdImpression)
	wg.Add(1)
	go func() {
		defer wg.Done()
		batch := new(AdImpressions)

		defer func() {
			if len(batch.Impressions) > 0 {
				impressionBatches <- batch
			}

			close(impressionBatches)
		}()

		for impression := range impressions {
			batch.Impressions = append(batch.Impressions, impression)
			if len(batch.Impressions) == 500 {
				impressionBatches <- batch
				batch = new(AdImpressions)
			}
		}
	}()

	clicks := make(chan *AdClick)
	wg.Add(1)
	go func() {
		defer wg.Done()
		batch := new(AdClicks)

		defer func() {
			if len(batch.Clicks) > 0 {
				clickBatches <- batch
			}

			close(clickBatches)
		}()

		for click := range clicks {
			batch.Clicks = append(batch.Clicks, click)
			if len(batch.Clicks) == 500 {
				clickBatches <- batch
				batch = new(AdClicks)
			}
		}
	}()

	usersWg := new(sync.WaitGroup)
	go func() {
		usersWg.Wait()
		close(impressions)
		close(clicks)
	}()

	events := make(chan *Ad)
	wg.Add(config.Users.Number)
	usersWg.Add(config.Users.Number)
	for i := 0; i < config.Users.Number; i++ {
		i := i
		go func() {
			defer wg.Done()
			defer usersWg.Done()

			for ad := range events {
				impressions <- &AdImpression{Ad: ad.ID, User: users.Users[i].ID, At: time.Now()}

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
					}
					m[tag] = struct{}{}
				}
				clickthroughProbability := float64(common) / float64(len(m)) * config.Users.MaximumClickThroughRate
				if rand.Float64() < clickthroughProbability {
					clicks <- &AdClick{Ad: ad.ID, User: users.Users[i].ID, At: time.Now()}
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
			close(events)
			break outer
		case events <- ads.Random():
		}
	}

	cancel()
	wg.Wait()
}
