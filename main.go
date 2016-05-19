package main

import (
	"bytes"
	"flag"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"text/template"
	"time"

	"github.com/odwrtw/transmission"
	"github.com/rs/xlog"
)

var (
	transmissionURL = flag.String("transmission-url", "http://localhost:9091/transmission/rpc", "The URL of the transmission RPC client")
	removeStalled   = flag.Bool("remove-stalled", false, "Remove stalled torrents")
	removeFinished  = flag.Bool("remove-finished", false, "Remove finished torrents")
	removeTemplate  = flag.String("remove-template", "", "A text/template file that is passed a torrent `t` and if evaluates to `true` will remove the torrent")
	cycles          = flag.Int("cycles", 5, "How many cycles a torrent should maintain stalled or finished cycle before being removed?")
	timeout         = flag.Duration("timeout", 30*time.Second, "How long to wait between cycles")

	tconn    *transmission.Client
	tstrikes = make(map[int]map[string]int)
	rt       *template.Template
	l        xlog.Logger
)

func main() {
	var err error

	flag.Parse()
	l = xlog.New(xlog.Config{
		Output: xlog.NewConsoleOutput(),
	})
	if *removeTemplate != "" {
		rtc, err := ioutil.ReadFile(*removeTemplate)
		if err != nil {
			l.Fatalf("error reading the filea %q: %s", *removeTemplate, err)
		}
		rt = template.Must(template.New("remove-template").Parse(string(rtc)))
	}
	tconn, err = transmission.New(transmission.Config{
		Address:    *transmissionURL,
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
	})
	if err != nil {
		l.Fatalf("error connection to transmission: %s", err)
	}
	signalC := make(chan os.Signal, 1)
	signal.Notify(signalC, os.Interrupt)
	timeoutTicker := time.NewTicker(*timeout)
	for {
		select {
		case <-signalC:
			timeoutTicker.Stop()
			return
		case <-timeoutTicker.C:
			cycle()
		}
	}
}

func cycle() {
	tseen := make(map[int]bool)
	ts, err := tconn.GetTorrents()
	l.SetField("tstrikes-map-length", len(tstrikes))
	l.SetField("transmission-num-torrents", len(ts))
	l.Info("running a cycle")
	if err != nil {
		l.Errorf("error getting the torrents: %s", err)
		return
	}

	// search and remove finished and stalled torrents
	for _, t := range ts {
		// mark the torrent as seen
		tseen[t.ID] = true
		// make sure we have it in the strikes map
		if tstrikes[t.ID] == nil {
			tstrikes[t.ID] = make(map[string]int)
		}
		// is it finished?
		if t.IsFinished {
			tstrikes[t.ID]["finished"]++
		}
		// is it stalled?
		if t.IsStalled {
			tstrikes[t.ID]["stalled"]++
		}
		// has this torrent been marked as finished for *cycles?
		if tstrikes[t.ID]["finished"] >= *cycles && (*removeFinished || removeTemplateTrue(t)) {
			l.Infof("The torrent %s has been finished for %d cycles and will be removed", t.Name, tstrikes[t.ID]["finished"])
			tconn.RemoveTorrents([]*transmission.Torrent{t}, false)
		}
		// has this torrent been marked as stalled for *cycles?
		if tstrikes[t.ID]["stalled"] >= *cycles && (*removeStalled || removeTemplateTrue(t)) {
			l.Infof("The torrent %s has been stalled for %d cycles and will be removed", t.Name, tstrikes[t.ID]["stalled"])
			tconn.RemoveTorrents([]*transmission.Torrent{t}, true)
		}
	}

	// Garbage-Collect the tstrikes map
	for id := range tstrikes {
		if !tseen[id] {
			delete(tstrikes, id)
		}
	}
}

func removeTemplateTrue(t *transmission.Torrent) bool {
	if rt == nil {
		return false
	}
	var buf bytes.Buffer
	if err := rt.Execute(&buf, t); err != nil {
		l.Errorf("error executing the remove template: %s", err)
		return false
	}
	if buf.String() == "true" {
		l.Infof("The template has evaluated to true for the torrent %q", t.Name)
		return true
	}
	return false
}
