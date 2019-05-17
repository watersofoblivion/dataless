package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
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

type AdClicks struct {
	Clicks []*AdClick `json:"clicks"`
}

type AdClick struct {
	Ad   uuid.UUID `json:"ad"`
	User uuid.UUID `json:"user"`
	At   time.Time `json:"at"`
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

	impressionBatches := make(chan *AdImpressions)
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := new(bytes.Buffer)
		encoder := json.NewEncoder(buf)
		earl := baseURL.ResolveReference(&url.URL{
			Path: filepath.Join(baseURL.Path, "/data/ad/impressions"),
		}).String()

		for {
			select {
			case <-ctx.Done():
				return

			case batch, more := <-impressionBatches:
				if !more {
					return
				}

				buf.Reset()
				if err := encoder.Encode(batch); err != nil {
					log.Printf("could not encode impressions batch: %s", err)
				}

				log.Printf("Publishing impressions batch")
				resp, err := http.Post(earl, "application/json", buf)
				if err != nil {
					log.Printf("error publishing impressions batch: %s", err)
				}

				if resp.StatusCode != http.StatusOK {
					log.Printf("non-%d status code publishing impressions batch", resp.StatusCode)
				}
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

		for {
			select {
			case <-ctx.Done():
				return

			case batch, more := <-clickBatches:
				if !more {
					return
				}

				buf.Reset()
				if err := encoder.Encode(batch); err != nil {
					log.Printf("could not encode impressions batch: %s", err)
				}

				log.Printf("Publishing clicks batch")
				resp, err := http.Post(earl, "application/json", buf)
				if err != nil {
					log.Printf("error publishing impressions batch: %s", err)
				}

				if resp.StatusCode != http.StatusOK {
					log.Printf("non-%d status code publishing impressions batch", resp.StatusCode)
				}
			}
		}
	}()

	impressions := make(chan *AdImpression)
	wg.Add(1)
	go func() {
		defer wg.Done()
		batch := new(AdImpressions)

		defer func() {
			impressionBatches <- batch
			close(impressionBatches)
		}()

		for {
			select {
			case <-ctx.Done():
				return

			case impression, more := <-impressions:
				if !more {
					return
				}

				batch.Impressions = append(batch.Impressions, impression)
				if len(batch.Impressions) == 500 {
					impressionBatches <- batch
					batch = new(AdImpressions)
				}
			}
		}
	}()

	clicks := make(chan *AdClick)
	wg.Add(1)
	go func() {
		defer wg.Done()
		batch := new(AdClicks)

		defer func() {
			clickBatches <- batch
			close(clickBatches)
		}()

		for {
			select {
			case <-ctx.Done():
				return

			case click, more := <-clicks:
				if !more {
					return
				}

				batch.Clicks = append(batch.Clicks, click)
				if len(batch.Clicks) == 500 {
					clickBatches <- batch
					batch = new(AdClicks)
				}
			}
		}
	}()

	usersWg := new(sync.WaitGroup)
	go func() {
		log.Printf("Waiting on users")
		usersWg.Wait()
		log.Printf("Users complete")
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

			for {
				select {
				case <-ctx.Done():
					return

				case ad, more := <-events:
					if !more {
						return
					}

					impressions <- &AdImpression{Ad: ad.ID, User: users.Users[i].ID, At: time.Now()}

					considerTime := users.Users[i].ConsiderDuration + (rand.Int63n(users.Users[i].ConsiderVariance) - (users.Users[i].ConsiderVariance / 2))
					time.Sleep(time.Duration(considerTime))

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
					time.Sleep(time.Duration(sleepTime))
				}
			}
		}()
	}

outer:
	for i := int64(0); i < config.Events; i++ {
		select {
		case <-ctx.Done():
			break outer
		case events <- ads.Random():
		}

	}
	close(events)

	signals := make(chan os.Signal)
	go func() {
		<-signals
		cancel()
	}()
	signal.Notify(signals, os.Interrupt)

	wg.Wait()
}
