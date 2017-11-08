package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"git.yale.edu/spinup/reaper/common"
	"git.yale.edu/spinup/reaper/reaper"
	"git.yale.edu/spinup/reaper/search"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

// Version is the main version number
const Version = reaper.Version

// VersionPrerelease is a prerelease marker
const VersionPrerelease = reaper.VersionPrerelease

var (
	configFileName = flag.String("config", "config/config.json", "Configuration file.")
	version        = flag.Bool("version", false, "Display version information and exit.")
	globalWg       sync.WaitGroup
)

type BySchedule []string

type RenewalSecret struct {
	RenewedAt string `json:"renewed_at"`
	Secret    string `json:"secret"`
}

func main() {
	flag.Parse()
	if *version {
		vers()
	}

	log.Infof("Reaper version %s%s", Version, VersionPrerelease)

	configFile, err := os.Open(*configFileName)
	if err != nil {
		log.Fatalln("Unable to open config file", err)
	}

	r := bufio.NewReader(configFile)
	config, err := common.ReadConfig(r)
	if err != nil {
		log.Fatalf("Unable to read configuration from %s.  %+v", *configFileName, err)
	}

	// Set the loglevel, info if it's unset
	switch config.LogLevel {
	case "error":
		log.SetLevel(log.ErrorLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "debug":
		log.SetLevel(log.DebugLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}

	log.Debugf("Loaded Config: %+v", config)

	// Setup context to allow goroutines to be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = Start(ctx, config)
	if err != nil {
		cancel()
		log.Fatalln("Couldn't initialize schedule routines", err)
	}

	srv := startHTTPServer(cancel, config)

	// Waitgroup waits for all goroutines to exit
	globalWg.Wait()

	log.Info("Stopping HTTP server")
	if err := srv.Shutdown(context.TODO()); err != nil {
		log.Fatalln("Failed to shutdown HTTP server cleanly", err)
	}
	log.Info("Done. exiting")
}

func vers() {
	fmt.Printf("Reaper Version: %s%s\n", Version, VersionPrerelease)
	os.Exit(0)
}

// Start fires up the batching routine loop which will do a search for each step;
// Notify, Decommission, Destroy, and then execute those steps.
func Start(ctx context.Context, config common.Config) error {
	interval, err := time.ParseDuration(config.Interval)
	if err != nil {
		log.Errorf("Couldn't parse interval duration %s. %+v", interval, err)
		return err
	}
	ticker := time.NewTicker(interval)

	// sort notifier schedule
	sort.Sort(BySchedule(config.Notify.Age))

	globalWg.Add(1)
	// launch a goroutine to run our batch on a schedule
	go func() {
		defer globalWg.Done()
		log.Infof("Initializing the batching routine loop.")
		for {
			select {
			case <-ticker.C:
				var wg sync.WaitGroup
				log.Infoln("Batch routine running...")

				wg.Add(1)
				go runNotifier(config, &wg)

				wg.Add(1)
				go runDecommissioner(config, &wg)

				wg.Add(1)
				go runDestroyer(config, &wg)

				log.Infoln("Batch routine sleeping...")
				wg.Wait()
			case <-ctx.Done():
				log.Infoln("Shutdown the batch routine")
				return
			}
		}
	}()

	return nil
}

// startHTTPServer registers the api endpoints and starts the webserver listening
func startHTTPServer(cancel func(), config common.Config) *http.Server {
	router := mux.NewRouter()
	api := router.PathPrefix("/v1").Subrouter()
	api.HandleFunc("/reaper/ping", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte{})
			return
		}
		log.Debug("Got ping request, responding pong.")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("pong"))
	})

	api.HandleFunc("/reaper/shutdown", func(w http.ResponseWriter, r *http.Request) {
		log.Infoln("Received shutdown request, cancelling goroutines.")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
		cancel()
		return
	})

	srv := &http.Server{
		Handler:      handlers.LoggingHandler(os.Stdout, router),
		Addr:         config.Listen,
		WriteTimeout: 30 * time.Second,
		ReadTimeout:  30 * time.Second,
	}

	go func() {
		log.Infof("Starting listener on :8080")
		if err := srv.ListenAndServe(); err != nil {
			log.Infof("Httpserver: ListenAndServe() error: %s", err)
			log.Fatal(err)
		}
	}()

	return srv
}

// runNotifier runs the routine to search for resources with renewed_at dates within a given range configured for notification.
// If the age-based notification threshold is crossed and a notification hasn't been sent:
// - Update the notified_at tag on the instance
// - Bail on this resource and continue to the next if tagging fails
// - Notify with a Notifier
// - Rollback tag if the notification fails
func runNotifier(config common.Config, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Infoln("Launching Notifier...")

	finder, err := search.NewFinder(&config)
	if err != nil {
		log.Errorln("Couldn't configure a new finder", err)
		return
	}

	ages := config.Notify.Age
	lte := fmt.Sprintf("now-%s", ages[0])
	log.Debugf("%s >= renewed_at", lte)

	// Query for anything older than the oldest age with the configured filters and status created
	termfilter := append(termFilters(config.Filter), search.TermQuery{Term: "status", Value: "created"})
	resources, err := finder.DoDateRangeQuery("resources", &search.DateRangeQuery{
		Field:      "yale:renewed_at",
		Format:     "YYYY/MM/dd HH:mm:ss",
		Lte:        lte,
		TermFilter: termfilter,
	})

	if err != nil {
		log.Errorln("Failed to execute date range query", err)
		return
	}

	// loop over the returned resources
	for _, r := range resources {
		log.Debugf("Checking returned resource: %+v", r)

		if r.Org == "" {
			log.Errorf("Cannot operate on a resource without an org.  ID: %s", r.ID)
			continue
		}

		// time of the last renewal
		renewedAt, err := time.Parse("2006/01/02 15:04:05", r.RenewedAt)
		if err != nil {
			log.Errorf("%s Couldn't parse renewed_at (%s) as a time value. %s", r.ID, r.RenewedAt, err.Error())
			continue
		}
		log.Infof("%s last renewed at %s", r.ID, renewedAt.String())

		decomAge, err := parseDuration(config.Decommission.Age)
		if err != nil {
			log.Errorf("%s Couldn't parse %s as a duration. %s", r.ID, config.Decommission.Age, err.Error())
			return
		}
		decomAt := renewedAt.Add(decomAge)

		renewalLink, err := generateRenewalToken(r.RenewedAt, config.Token)
		if err != nil {
			log.Errorf("Failed to generate renewal token, %s", err.Error())
			continue
		}

		if r.NotifiedAt == "" {
			log.Infof("%s Notified At is not set, Notifying on age threshold %s", r.ID, ages[0])

			err = NewTagger(config.Tagging.Endpoint, config.Tagging.Token, r.ID, r.Org, map[string]string{
				"yale:notified_at": time.Now().Format("2006/01/02 15:04:05"),
			}).Tag()

			if err != nil {
				log.Errorf("Failed to update tag for %s, not notifying, %s", r.ID, err.Error())
				continue
			}

			err := NewNotifier(config.Notify.Endpoint, config.Notify.Token, map[string]string{
				"netid":      r.CreatedBy,
				"link":       renewalLink,
				"expire_on":  decomAt.Format("2006/01/02 15:04:05"),
				"renewed_at": renewedAt.Format("2006/01/02 15:04:05"),
				"fqdn":       r.FQDN,
			}).Notify()

			if err != nil {
				log.Errorf("Failed to notify.  Rolling back notified_at tag to ''. %s", err.Error())

				err = NewTagger(config.Tagging.Endpoint, config.Tagging.Token, r.ID, r.Org, map[string]string{
					"yale:notified_at": "",
				}).Tag()

				if err != nil {
					log.Errorf("Failed to roll back notified_at tag for %s, %s", r.ID, err.Error())
				}

				continue
			}
		} else {
			// time of the last notification
			notifiedAt, err := time.Parse("2006/01/02 15:04:05", r.NotifiedAt)
			if err != nil {
				log.Errorf("%s Couldn't parse notified_at (%s) as a time value. %s", r.ID, r.NotifiedAt, err.Error())
				continue
			}
			log.Infof("%s last notified at %s", r.ID, notifiedAt.String())

			// range over the ages and check if we've notified since the age threshold was crossed
			for _, age := range ages {
				log.Debugf("Checking for notification on age %s", age)

				ageDuration, err := parseDuration(age)
				if err != nil {
					log.Errorf("%s Couldn't parse %s as a duration. %s", r.ID, age, err.Error())
					continue
				}

				// time the age threshold was crossed
				ageThresholdAt := renewedAt.Add(ageDuration)
				log.Infof("%s crossed the %s age threshold at %s", r.ID, age, ageThresholdAt.String())

				if ageThresholdAt.Before(time.Now()) && notifiedAt.Before(ageThresholdAt) {
					log.Infof("%s notified (%s) before age threshold (%s) was crossed (%s). Notifying", r.ID, notifiedAt.String(), age, ageThresholdAt.String())

					err = NewTagger(config.Tagging.Endpoint, config.Tagging.Token, r.ID, r.Org, map[string]string{
						"yale:notified_at": time.Now().Format("2006/01/02 15:04:05"),
					}).Tag()

					if err != nil {
						log.Errorf("Failed to update tag for %s, not notifying, %s", r.ID, err.Error())
						continue
					}

					err := NewNotifier(config.Notify.Endpoint, config.Notify.Token, map[string]string{
						"netid":      r.CreatedBy,
						"link":       renewalLink,
						"expire_on":  decomAt.Format("2006/01/02 15:04:05"),
						"renewed_at": renewedAt.Format("2006/01/02 15:04:05"),
						"fqdn":       r.FQDN,
					}).Notify()

					if err != nil {
						log.Errorf("Failed to notify.  Rolling back notified_at tag to %s. %s", r.NotifiedAt, err.Error())

						err = NewTagger(config.Tagging.Endpoint, config.Tagging.Token, r.ID, r.Org, map[string]string{
							"yale:notified_at": r.NotifiedAt,
						}).Tag()

						if err != nil {
							log.Errorf("Failed to roll back notified_at tag for %s, %s", r.ID, err.Error())
						}
					}

					// stop notifying if we matched
					break
				}

				log.Debugf("%s has been notified (%s) since crossing the %s age threshold (%s)", r.ID, notifiedAt.String(), age, ageThresholdAt.String())
			}
		}
	}

	return
}

// runDecommissioner runs the routine to search for resources with renewed_at dates
// within the decommission age and the destroy age
func runDecommissioner(config common.Config, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Infoln("Launching Decommissioner...")

	finder, err := search.NewFinder(&config)
	if err != nil {
		log.Errorln("Couldn't configure a new finder", err)
		return
	}

	newer := config.Decommission.Age
	older := config.Destroy.Age

	lte := fmt.Sprintf("now-%s", newer)
	gt := fmt.Sprintf("now-%s", older)
	log.Debugf("%s >= renewed_at > %s", lte, gt)

	resources, err := finder.DoDateRangeQuery("resources", &search.DateRangeQuery{
		Field:      "yale:renewed_at",
		Format:     "YYYY/MM/dd HH:mm:ss",
		Lte:        lte,
		Gt:         gt,
		TermFilter: termFilters(config.Filter),
	})

	if err != nil {
		log.Errorln("Failed to execute date range query", err)
		return
	}

	log.Debugf("Got %d resource(s) from decommission query for age %s", len(resources), newer)

	for _, r := range resources {
		if r.Status == "created" {
			log.Infof("Decommissioning resource ID %s on %s age", r.ID, newer)
		} else {
			log.Infof("%s is not in created state (%s). Not proceeding with decom", r.ID, r.Status)
		}
	}
}

// runDestroyer runs the routine to search for resources with renewed_at dates
// beyond the destroy age
func runDestroyer(config common.Config, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Infoln("Launching Destroyer...")
	finder, err := search.NewFinder(&config)
	if err != nil {
		log.Errorln("Couldn't configure a new finder", err)
		return
	}

	newer := config.Destroy.Age

	lte := fmt.Sprintf("now-%s", newer)
	log.Debugf("%s >= renewed_at", lte)

	resources, err := finder.DoDateRangeQuery("resources", &search.DateRangeQuery{
		Field:      "yale:renewed_at",
		Format:     "YYYY/MM/dd HH:mm:ss",
		Lte:        lte,
		TermFilter: termFilters(config.Filter),
	})

	if err != nil {
		log.Errorln("Failed to execute date range query", err)
		return
	}

	log.Debugf("Got %d resource(s) from destruction query for age %s", len(resources), newer)

	for _, r := range resources {
		if r.Status == "decom" {
			log.Infof("Destroying resource ID %s on %s age", r.ID, newer)
		} else {
			log.Infof("%s is not in decom state (%s). Not proceeding with destruction", r.ID, r.Status)
		}
	}
}

func generateRenewalToken(renewedAt, secret string) (string, error) {
	str, err := json.Marshal(RenewalSecret{
		RenewedAt: renewedAt,
		Secret:    secret,
	})
	if err != nil {
		return "", err
	}

	log.Debugf("Marshalled secret JSON string %s", str)

	token, err := bcrypt.GenerateFromPassword([]byte(str), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	log.Debugln("Secret hash:", string(token))

	return string(token), nil
}

// parseDuration parses durations of days, weeks and months (in the most simplistic way)
// since time.ParseDuration only supports up to hours https://github.com/golang/go/issues/11473
// If there's a parsing error, return 0 and the error.  Originally, this returned MAXINT64 and the
// error, but time.ParseDuration(foo) returns 0 on error and I wanted to stay consistent.
func parseDuration(d string) (time.Duration, error) {
	switch {
	case strings.HasSuffix(d, "d"):
		t := strings.TrimSuffix(d, "d")
		num, err := strconv.ParseInt(t, 10, 64)
		if err != nil {
			return time.Duration(0), err
		}
		return time.Duration(num*24) * time.Hour, nil
	case strings.HasSuffix(d, "w"):
		t := strings.TrimSuffix(d, "w")
		num, err := strconv.ParseInt(t, 10, 64)
		if err != nil {
			return time.Duration(0), err
		}
		return time.Duration(num*7*24) * time.Hour, nil
	case strings.HasSuffix(d, "mo"):
		t := strings.TrimSuffix(d, "mo")
		num, err := strconv.ParseInt(t, 10, 64)
		if err != nil {
			return time.Duration(0), err
		}
		return time.Duration(num*30*24) * time.Hour, nil
	default:
		return time.ParseDuration(d)
	}
}

// Len is required to satisfy sort.Interface
func (s BySchedule) Len() int {
	return len(s)
}

// Swap is required to satisfy sort.Interface
func (s BySchedule) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Less is required to satisfy sort.Interface
func (s BySchedule) Less(i, j int) bool {
	di, _ := parseDuration(s[i])
	dj, _ := parseDuration(s[j])
	return di < dj
}

func termFilters(filters map[string]string) []search.TermQuery {
	var tqs []search.TermQuery
	for key, value := range filters {
		tqs = append(tqs, search.TermQuery{Term: key, Value: value})
	}
	return tqs
}
