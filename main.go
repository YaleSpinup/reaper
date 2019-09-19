package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"net/http"
	_ "net/http/pprof"
	"os"
	"sort"
	"sync"
	"time"

	report "github.com/YaleSpinup/eventreporter"

	"github.com/YaleSpinup/reaper/common"
	"github.com/YaleSpinup/reaper/reaper"
	"github.com/YaleSpinup/reaper/search"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	log "github.com/sirupsen/logrus"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// Version is the application version, it can be overriden at buildtime with ldflags
	Version = reaper.Version

	// VersionPrerelease is the prerelease marker, it can be overriden at buildtime with ldflags
	VersionPrerelease = reaper.VersionPrerelease

	// buildstamp is the timestamp the binary was built, it should be set at buildtime with ldflags
	buildstamp = "No BuildStamp Provided"

	// githash is the git sha of the built binary, it should be set at buildtime with ldflags
	githash = "No Git Commit Provided"

	// AppConfig is the global applicatoin configuration
	AppConfig common.Config

	// EventReporters is a slice of reporting endpoints
	EventReporters []report.Reporter

	// Webhooks is a slice of webhook providers
	Webhooks []Webhook

	globalWg sync.WaitGroup

	configFileName = flag.String("config", "config/config.json", "Configuration file.")
	version        = flag.Bool("version", false, "Display version information and exit.")
)

// RenewalTemplate is the html template for the renewal endpoint
const RenewalTemplate = `
<html>
<head>
<meta http-equiv="refresh" content="2;url={{.RedirectURL}}" />
<title>Success</title>
</head>
<body>
Success! Redirecting to the <a href="{{.RedirectURL}}">spinup portal</a>.
</body>
</html>`

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
	AppConfig = config

	// Set the loglevel, info if it's unset
	switch AppConfig.LogLevel {
	case "error":
		log.SetLevel(log.ErrorLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "debug":
		log.SetLevel(log.DebugLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}

	if AppConfig.LogLevel == "debug" {
		log.Debug("Starting profiler on 127.0.0.1:6080")
		go http.ListenAndServe("127.0.0.1:6080", nil)
	}

	log.Debugf("Loaded Config: %+v", config)

	err = configureEventReporters()
	if err != nil {
		log.Fatalln("Couldn't initialize event reporters", err)
	}

	reportEvent(fmt.Sprintf("Starting reaper %s%s (%s)", Version, VersionPrerelease, AppConfig.BaseURL), report.INFO)

	err = configureWebhooks()
	if err != nil {
		log.Fatalln("Couldn't initialize web hooks", err)
	}

	// Setup context to allow goroutines to be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = Start(ctx)
	if err != nil {
		cancel()
		log.Fatalln("Couldn't initialize schedule routines", err)
	}

	srv := startHTTPServer(cancel)

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
	fmt.Println("Git Commit Hash:", githash)
	fmt.Println("UTC Build Time:", buildstamp)
	os.Exit(0)
}

func configureEventReporters() error {
	for name, config := range AppConfig.EventReporters {
		switch name {
		case "slack":
			reporter, err := report.NewSlackReporter(config)
			if err != nil {
				return err
			}

			log.Debugf("Configured %s event reporter", name)
			EventReporters = append(EventReporters, reporter)
		default:
			msg := fmt.Sprintf("Unknown event reporter name, %s", name)
			return errors.New(msg)
		}
	}

	return nil
}

func configureWebhooks() error {
	for _, webhook := range AppConfig.Webhooks {
		wh, err := NewWebhook(webhook)
		if err != nil {
			return err
		}

		Webhooks = append(Webhooks, wh)
	}

	return nil
}

// startHTTPServer registers the api endpoints and starts the webserver listening
func startHTTPServer(cancel func()) *http.Server {
	router := mux.NewRouter()

	api := router.PathPrefix("/v1").Subrouter()
	api.HandleFunc("/reaper/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")

		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte{})
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("pong"))
	})

	api.Handle("/reaper/metrics", promhttp.Handler())

	api.HandleFunc("/reaper/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")

		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte{})
			return
		}
		w.WriteHeader(http.StatusOK)

		data, err := json.Marshal(struct {
			Version    string `json:"version"`
			GitHash    string `json:"githash"`
			BuildStamp string `json:"buildstamp"`
		}{
			Version:    fmt.Sprintf("%s%s", Version, VersionPrerelease),
			GitHash:    githash,
			BuildStamp: buildstamp,
		})

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte{})
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(data)
	})

	api.HandleFunc("/reaper/shutdown", func(w http.ResponseWriter, r *http.Request) {
		log.Infoln("Received shutdown request, cancelling goroutines.")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
		cancel()
		return
	})

	api.HandleFunc("/reaper/renew/{id:[A-Za-z0-9-]+}", RenewalHander)

	srv := &http.Server{
		Handler:      handlers.LoggingHandler(os.Stdout, router),
		Addr:         AppConfig.Listen,
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

// RenewalHander handles resource renewal
// - The request method is checked, it should be GET
// - Query parameter 'token' is retrieved from the request
// - The subject resource id is retrieved from the URL variable
// - Resource with the id 'id' is fetched from elasticsearch
// - Token is validated against the information pulled from the resource
// - If everything is good, the renewed_at tag is updated
func RenewalHander(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte{})
		return
	}

	// Look for the token in the query
	tokens, ok := r.URL.Query()["token"]
	if !ok || len(tokens) != 1 {
		log.Warnf("Token parameter is missing or of bad format for request: %s", r.URL)
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte{})
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte{})
		return
	}

	finder, err := search.NewFinder(&AppConfig)
	if err != nil {
		log.Errorln("Couldn't configure a new finder", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed connecting to elasticsearch"))
		return
	}

	resource, err := finder.DoGet("resources", "server", id)
	if err != nil {
		log.Errorf("Couldn't get the %s resource from elasticsearch, %s", id, err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed getting details about resource"))
		return
	}

	log.Debugf("Got resource %+v back from elasticsearch", resource)

	renewalSecret := &RenewalSecret{RenewedAt: resource.RenewedAt, Secret: AppConfig.EncryptionSecret}
	if err = renewalSecret.ValidateRenewalToken(tokens[0]); err != nil {
		log.Warnf("Failed to validate token string %s, %s", tokens[0], err.Error())
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte{})
		return
	}

	tagger := NewTagger(AppConfig.Tagging.Endpoint, AppConfig.Tagging.Token, id, resource.Org)
	newRenewedAt := time.Now().Format("2006/01/02 15:04:05")
	if err = tagger.Tag(map[string]string{"yale:renewed_at": newRenewedAt}); err != nil {
		log.Errorf("Failed to renew resource %s, %s", id, err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Unable to process renewal, please try again later."))
		return
	}

	msg := fmt.Sprintf("Renewed %s (%s) created by %s", resource.FQDN, id, resource.SupportDepartmentContact)
	log.Info(msg)
	reportEvent(msg, report.INFO)

	buffer := new(bytes.Buffer)
	tmpl, err := template.New("renewalTemplate").Parse(RenewalTemplate)
	if err != nil {
		log.Errorf("Failed to parse the renewal template: %s", err)
		http.Redirect(w, r, AppConfig.RedirectURL, 302)
	} else {
		err = tmpl.Execute(buffer, struct{ RedirectURL string }{RedirectURL: AppConfig.RedirectURL})
		if err != nil {
			log.Errorf("Failed to execute the renewal template: %s", err)
			http.Redirect(w, r, AppConfig.RedirectURL, 302)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write(buffer.Bytes())
		}
	}

	f, err := NewUserFetcher(AppConfig.UserDatasource)
	if err != nil {
		log.Errorf("Unable to configure user datasource for %s: %s", resource.SupportDepartmentContact, err)
		return
	}
	user, err := GetUserByID(f, resource.SupportDepartmentContact)
	if err != nil {
		log.Errorf("Unable to get details about user %s: %s", resource.SupportDepartmentContact, err)
		return
	}

	expireOn, err := GetDecomAt(newRenewedAt, AppConfig.Decommission.Age)
	if err != nil {
		log.Errorf("Unable to get the decomAt date for %s: %s", resource.ID, err)
		return
	}

	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		loc = time.FixedZone("UTC", 0)
	}

	// generate the renewal confirmation email
	body, err := ParseRenewalTemplate(map[string]string{
		"first":         user.First,
		"email":         user.Email,
		"netid":         resource.SupportDepartmentContact,
		"fqdn":          resource.FQDN,
		"expire_on":     expireOn.In(loc).Format("2006/01/02 15:04:05 MST"),
		"spinupURL":     AppConfig.SpinupURL,
		"spinupSiteURL": AppConfig.SpinupSiteURL,
	})

	if err != nil {
		log.Errorf("Unable to get the parse the renewal template for %s: %s", resource.ID, err)
		return
	}

	err = SendMail(AppConfig.Email.Mailserver, body, AppConfig.Email.From, AppConfig.Email.Password,
		"Your Spinup TryIT server renewal", user.Email, AppConfig.Email.Username)

	if err != nil {
		log.Errorf("Failed sending the renewal confirmation email: %s", err)
	}
}

// Start fires up the batching routine loop which will do a search for each step;
// Notify, Decommission, Destroy, and then execute those steps.
func Start(ctx context.Context) error {
	interval, err := time.ParseDuration(AppConfig.Interval)
	if err != nil {
		log.Errorf("Couldn't parse interval duration %s. %+v", interval, err)
		return err
	}
	ticker := time.NewTicker(interval)

	// sort notifier schedule
	sort.Sort(BySchedule(AppConfig.Notify.Age))

	globalWg.Add(1)
	// launch a goroutine to run our batch on a schedule
	go func() {
		defer globalWg.Done()

		finder, err := search.NewFinder(&AppConfig)
		if err != nil {
			log.Errorln("Couldn't configure a new finder", err)
			return
		}

		log.Infof("Initializing the batching routine loop.")
		for {
			select {
			case <-ticker.C:
				log.Infoln("Batch routine running...")
				destroy(*finder)
				decommission(*finder)
				notify(*finder)
				log.Infoln("Batch routine sleeping...")
			case <-ctx.Done():
				log.Infoln("Shutdown the batch routine")
				return
			}
		}
	}()

	return nil
}

// notify runs the routine to search for resources with renewed_at dates within a given range configured for notification.
// If the age-based notification threshold is crossed and a notification hasn't been sent:
// - Update the notified_at tag on the instance
// - Bail on this resource and continue to the next if tagging fails
// - Notify with a Notifier
// - Rollback tag if the notification fails
func notify(finder search.Finder) {
	log.Infoln("Launching Notifier...")

	ages := AppConfig.Notify.Age
	lte := fmt.Sprintf("now-%s", ages[0])
	log.Debugf("%s >= renewed_at", lte)

	// Query for anything older than the oldest age with the configured filters and status created
	termfilter := append(search.NewTermQueryList(AppConfig.Filter), search.TermQuery{Term: "status", Value: "created"})
	resources, err := finder.DoDateRangeQuery("resources", "server", &search.DateRangeQuery{
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
	for _, resource := range resources {
		log.Debugf("Checking returned resource: %+v", resource)

		if resource.Org == "" {
			log.Errorf("Cannot operate on a resource without an org.  ID: %s", resource.ID)
			continue
		}

		// time of the last renewal
		renewedAt, err := time.Parse("2006/01/02 15:04:05", resource.RenewedAt)
		if err != nil {
			log.Errorf("%s Couldn't parse renewed_at (%s) as a time value. %s", resource.ID, resource.RenewedAt, err.Error())
			continue
		}
		log.Infof("%s last renewed at %s", resource.ID, renewedAt.String())

		renewalSecret := &RenewalSecret{
			RenewedAt: resource.RenewedAt,
			Secret:    AppConfig.EncryptionSecret,
		}
		token, err := renewalSecret.GenerateRenewalToken()
		if err != nil {
			log.Errorf("Failed to generate renewal token, %s", err.Error())
			continue
		}
		renewalLink := fmt.Sprintf("%s/renew/%s?token=%s", AppConfig.BaseURL, resource.ID, token)
		log.Debugf("Generated renewal link: %s", renewalLink)

		if resource.NotifiedAt == "" {
			log.Infof("%s Notified At is not set, Notifying on age threshold %s", resource.ID, ages[0])
			err := sendNotification(resource, renewalLink, renewedAt)
			if err != nil {
				log.Errorf("Failed to notify. %s", err.Error())
				continue
			}

			sendWebhooks(&Event{
				ID:     resource.ID,
				Action: "notify",
			})
		} else {
			// time of the last notification
			notifiedAt, err := time.Parse("2006/01/02 15:04:05", resource.NotifiedAt)
			if err != nil {
				log.Errorf("%s Couldn't parse notified_at (%s) as a time value. %s", resource.ID, resource.NotifiedAt, err.Error())
				continue
			}
			log.Infof("%s last notified at %s", resource.ID, notifiedAt.String())

			// range over the ages and check if we've notified since the age threshold was crossed
			for _, age := range ages {
				log.Debugf("Checking for notification on age %s", age)

				ageDuration, err := parseDuration(age)
				if err != nil {
					log.Errorf("%s Couldn't parse %s as a duration. %s", resource.ID, age, err.Error())
					// If we can't parse the age, stop trying and move on
					break
				}

				// time the age threshold was crossed
				ageThresholdAt := renewedAt.Add(ageDuration)
				log.Debugf("%s %s age threshold: %s", resource.ID, age, ageThresholdAt.String())

				if ageThresholdAt.Before(time.Now()) && notifiedAt.Before(ageThresholdAt) {
					log.Infof("%s notified (%s) before age threshold (%s) was crossed (%s). Notifying", resource.ID, notifiedAt.String(), age, ageThresholdAt.String())

					err := sendNotification(resource, renewalLink, renewedAt)
					if err != nil {
						log.Errorf("Failed to notify. %s", err.Error())
					}

					sendWebhooks(&Event{
						ID:     resource.ID,
						Action: "notify",
					})

					// stop notifying if we matched
					break
				}

				log.Debugf("%s has been notified (%s) since crossing the %s age threshold (%s)", resource.ID, notifiedAt.String(), age, ageThresholdAt.String())
			}
		}
	}
}

func sendNotification(resource *search.Resource, renewalLink string, renewedAt time.Time) error {
	// try to get details about the user before we do _anything_ since it's the lightest touch
	f, err := NewUserFetcher(AppConfig.UserDatasource)
	if err != nil {
		msg := fmt.Sprintf("Unable to configure user datasource for %s", resource.SupportDepartmentContact)
		log.Error(msg+": "+err.Error(), resource.SupportDepartmentContact, err)
		reportEvent("FAILED "+msg, report.ERROR)
		return err
	}
	user, err := GetUserByID(f, resource.SupportDepartmentContact)
	if err != nil {
		msg := fmt.Sprintf("Unable to get details about user %s", resource.SupportDepartmentContact)
		log.Error(msg + ": " + err.Error())
		reportEvent("FAILED "+msg, report.ERROR)
		return err
	}

	// tag the instance with the new notification date
	tagger := NewTagger(AppConfig.Tagging.Endpoint, AppConfig.Tagging.Token, resource.ID, resource.Org)
	err = tagger.Tag(map[string]string{
		"yale:notified_at": time.Now().Format("2006/01/02 15:04:05"),
	})

	// if we can't tag, then bail all together, I just can't go on....
	if err != nil {
		msg := fmt.Sprintf("Unable to update tag for %s (%s)", resource.FQDN, resource.ID)
		log.Error(msg + ": " + err.Error())
		reportEvent("FAILED"+msg, report.ERROR)
		return err
	}

	// create a function for rolling back the tag if something fails
	rollBackTag := func() {
		err = tagger.Tag(map[string]string{
			"yale:notified_at": resource.NotifiedAt,
		})

		if err != nil {
			msg := fmt.Sprintf("Unable to roll back tag for %s (%s)", resource.FQDN, resource.ID)
			reportEvent("FAILED "+msg, report.ERROR)
			log.Errorf(msg + ": " + err.Error())
		}
	}

	reportEvent(fmt.Sprintf("Notifying %s for %s (%s)", resource.SupportDepartmentContact, resource.FQDN, resource.ID), report.INFO)

	// get the date that the instance will expire
	expireOn, err := GetDecomAt(resource.RenewedAt, AppConfig.Decommission.Age)
	if err != nil {
		log.Errorf("Unable to get the decomAt date for %s: %s", resource.ID, err)
		return err
	}

	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		loc = time.FixedZone("UTC", 0)
	}

	// generate the warning email from the warning template
	body, err := ParseWarningTemplate(map[string]string{
		"first":      user.First,
		"email":      user.Email,
		"netid":      resource.SupportDepartmentContact,
		"link":       renewalLink,
		"expire_on":  expireOn.In(loc).Format("2006/01/02 15:04:05 MST"),
		"renewed_at": renewedAt.In(loc).Format("2006/01/02 15:04:05 MST"),
		"fqdn":       resource.FQDN,
		"spinupURL":  AppConfig.RedirectURL,
	})

	// rollback the tag and bail if we're unable to parse the template with the given data
	if err != nil {
		reportEvent(fmt.Sprintf("FAILED to parse template for %s (%s), not sending email", resource.FQDN, resource.ID), report.ERROR)
		log.Errorf("FAILED to parse template.  Rolling back notified_at tag to ''. %s", err.Error())
		rollBackTag()
		return err
	}

	// send the mail to the user notifying them that their instance will expire
	err = SendMail(AppConfig.Email.Mailserver, body, AppConfig.Email.From, AppConfig.Email.Password,
		"Please renew your Spinup TryIT server", user.Email, AppConfig.Email.Username)

	// rollback the tag if we fail to send the email
	if err != nil {
		reportEvent(fmt.Sprintf("FAILED to send notification for %s (%s)", resource.FQDN, resource.ID), report.ERROR)
		log.Errorf("Failed to notify.  Rolling back notified_at tag to ''. %s", err.Error())
		rollBackTag()
	}

	return err
}

// decommission runs the routine to search for resources with renewed_at dates within the decommission age and the destroy age
func decommission(finder search.Finder) {
	log.Infoln("Launching Decommissioner...")

	// Query for anything older than the decommission age with the configured filters and status created
	termfilter := append(search.NewTermQueryList(AppConfig.Filter), search.TermQuery{Term: "status", Value: "created"})
	resources, err := finder.DoDateRangeQuery("resources", "server", &search.DateRangeQuery{
		Field:      "yale:renewed_at",
		Format:     "YYYY/MM/dd HH:mm:ss",
		Lte:        fmt.Sprintf("now-%s", AppConfig.Decommission.Age),
		TermFilter: termfilter,
	})

	if err != nil {
		log.Errorln("Failed to execute date range query", err)
		return
	}

	// loop over the returned resources
	for _, resource := range resources {
		log.Debugf("Checking returned resource: %+v", resource)

		if resource.Org == "" {
			log.Errorf("Cannot operate on a resource without an org.  ID: %s", resource.ID)
			continue
		}

		// time of the last renewal
		renewedAt, err := time.Parse("2006/01/02 15:04:05", resource.RenewedAt)
		if err != nil {
			log.Errorf("%s Couldn't parse renewed_at (%s) as a time value. %s", resource.ID, resource.RenewedAt, err.Error())
			continue
		}
		log.Infof("%s last renewed at %s", resource.ID, renewedAt.String())

		destroyAge, err := parseDuration(AppConfig.Destroy.Age)
		if err != nil {
			log.Errorf("%s Couldn't parse %s as a duration. %s", resource.ID, AppConfig.Destroy.Age, err.Error())
			return
		}
		// Add the destroy age to the renewed_at date to get the destroy_at date
		destroyAt := renewedAt.Add(destroyAge)

		if destroyAt.Before(time.Now()) {
			log.Warnf("%s has crossed the destroy threshold but hasn't been decommissioned (Destruction scheduled: %s)", resource.ID, destroyAt.String())
		}

		log.Infof("%s has crossed the decommision threshold. (Destruction scheduled: %s)", resource.ID, destroyAt.String())

		reportEvent(fmt.Sprintf("Decommissioning for %s (%s)", resource.FQDN, resource.ID), report.INFO)
		err = NewDecommissioner(AppConfig.Decommission.Endpoint, AppConfig.Decommission.Token, resource.ID, resource.Org).SetStatus()
		if err != nil {
			reportEvent(fmt.Sprintf("FAILED to decommission for %s (%s)", resource.FQDN, resource.ID), report.ERROR)
			log.Errorf("Unable to decommission %s, %s", resource.ID, err.Error())
			continue
		}

		sendWebhooks(&Event{
			ID:     resource.ID,
			Action: "decommission",
		})

		// try to get details about the user so we can notify them that their instance has been decommissioned, note that we
		// do this _after_ we decommission since we don't really care if we notified them and we want the decom to succeed even
		// if we can't send the email.
		f, err := NewUserFetcher(AppConfig.UserDatasource)
		if err != nil {
			log.Errorf("Unable to configure user datasource for %s: %s", resource.SupportDepartmentContact, err)
			continue
		}
		user, err := GetUserByID(f, resource.SupportDepartmentContact)
		if err != nil {
			log.Errorf("Unable to get details about user %s: %s", resource.SupportDepartmentContact, err)
			continue
		}

		// get the date that the instance will expire
		expireOn, err := GetDecomAt(resource.RenewedAt, AppConfig.Decommission.Age)
		if err != nil {
			log.Errorf("Unable to get the decomAt date for %s: %s", resource.ID, err)
			continue
		}

		loc, err := time.LoadLocation("America/New_York")
		if err != nil {
			loc = time.FixedZone("UTC", 0)
		}

		// generate the decom email
		body, err := ParseDecomTemplate(map[string]string{
			"first":     user.First,
			"email":     user.Email,
			"netid":     resource.SupportDepartmentContact,
			"fqdn":      resource.FQDN,
			"expire_on": expireOn.In(loc).Format("2006/01/02 15:04:05 MST"),
			"spinupURL": AppConfig.RedirectURL,
		})

		if err != nil {
			log.Errorf("Unable to get the parse the decom template for %s: %s", resource.ID, err)
			continue
		}

		err = SendMail(AppConfig.Email.Mailserver, body, AppConfig.Email.From, AppConfig.Email.Password,
			"Your Spinup TryIT server has been deleted", user.Email, AppConfig.Email.Username)
		if err != nil {
			log.Errorf("Failed sending the decom email: %s", err)
		}
	}
}

// destroy runs the routine to search for resources with renewed_at dates beyond the destroy age
func destroy(finder search.Finder) {
	log.Infoln("Launching Destroyer...")

	// Query for anything older than the destroy age with the configured filters and status decom
	termfilter := append(search.NewTermQueryList(AppConfig.Filter), search.TermQuery{Term: "status", Value: "decom"})
	resources, err := finder.DoDateRangeQuery("resources", "server", &search.DateRangeQuery{
		Field:      "yale:renewed_at",
		Format:     "YYYY/MM/dd HH:mm:ss",
		Lte:        fmt.Sprintf("now-%s", AppConfig.Destroy.Age),
		TermFilter: termfilter,
	})

	if err != nil {
		log.Errorln("Failed to execute date range query", err)
		return
	}

	// loop over the returned resources
	for _, resource := range resources {
		log.Debugf("Checking returned resource: %+v", resource)

		if resource.Org == "" {
			log.Errorf("Cannot operate on a resource without an org.  ID: %s", resource.ID)
			continue
		}

		// time of the last renewal
		renewedAt, err := time.Parse("2006/01/02 15:04:05", resource.RenewedAt)
		if err != nil {
			log.Errorf("%s Couldn't parse renewed_at (%s) as a time value. %s", resource.ID, resource.RenewedAt, err.Error())
			continue
		}

		log.Infof("%s last renewed at %s", resource.ID, renewedAt.String())
		log.Infof("%s has crossed the destruction threshold.", resource.ID)

		reportEvent(fmt.Sprintf("Destroying for %s (%s)", resource.FQDN, resource.ID), report.INFO)
		err = NewDestroyer(AppConfig.Decommission.Endpoint, AppConfig.Decommission.Token, resource.ID, resource.Org).Destroy()
		if err != nil {
			reportEvent(fmt.Sprintf("FAILED to destroy for %s (%s)", resource.FQDN, resource.ID), report.ERROR)
			log.Errorf("Unable to destroy %s, %s", resource.ID, err.Error())
		}

		sendWebhooks(&Event{
			ID:     resource.ID,
			Action: "destroy",
		})
	}

}

// reportEvent loops over all of the configured event reporters and sends the event to those reporters
func reportEvent(msg string, level report.Level) {
	e := report.Event{
		Message: msg,
		Level:   level,
	}

	for _, r := range EventReporters {
		err := r.Report(e)
		if err != nil {
			log.Errorf("Failed to report event (%s) %s", msg, err.Error())
		}
	}
}

// sendWebhooks loops over all of the configured webhooks and sends them
func sendWebhooks(e *Event) {
	for _, wh := range Webhooks {
		err := wh.Send(context.TODO(), e)
		if err != nil {
			log.Errorf("Failed to send webhook (%s) %s", wh.Endpoint, err.Error())
		}
	}
}

// GetDecomAt centralizes the calculation of a decommission date
func GetDecomAt(renewedAtString, decomAgeString string) (time.Time, error) {
	var decomAt time.Time

	// time of the last renewal
	renewedAt, err := time.Parse("2006/01/02 15:04:05", renewedAtString)
	if err != nil {
		return decomAt, err
	}

	decomAge, err := parseDuration(decomAgeString)
	if err != nil {
		return decomAt, err
	}

	decomAt = renewedAt.Add(decomAge)
	return decomAt, nil
}
