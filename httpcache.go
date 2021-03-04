package httpcache

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/golevi/cache-handler/config"
	"github.com/golevi/cache-handler/stores"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// Cache stuff
type Cache struct {
	Config   config.Config
	Store    stores.CacheStore
	Deciders []func(c config.Config, w http.ResponseWriter, r *http.Request) bool

	logger *zap.Logger
}

type cacheResponse struct {
	Status        string      `json:"status"`
	StatusCode    int         `json:"status_code"`
	Headers       http.Header `json:"headers"`
	Body          []byte      `json:"body"`
	ContentLength int64       `json:"content_length"`
}

func (c *Cache) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	labels := prometheus.Labels{"handler": "cache"}

	// Loop through deciders to see whether or not this request should be cached
	// or if we should bypass it and send it to the origin.
	for _, decider := range c.Deciders {
		if decider(c.Config, w, r) {
			w.Header().Add("Cache-Status", "bypass")
			ch := httpMetrics.cacheBypass.With(labels)
			ch.Inc()

			return next.ServeHTTP(w, r)
		}
	}

	// We aren't going to bypass this request, so we are going to need to create
	// a key based upon the details of the request.
	key := key(r)

	// Check to see if we have this key in our cache-store. If so, then we can
	// retrieve it and return it to the client without hitting the origin.
	if c.Store.Has(key) {
		// Since we have the key, add the header.
		w.Header().Add("Cache-Status", "hit")

		// Increment our cache hit metric.
		ch := httpMetrics.cacheHit.With(labels)
		ch.Inc()

		// Get the response from our cache-store.
		response, err := c.Store.Get(key)
		if err != nil {
			return err
		}

		// Create a cacheResponse struct so we can reconstruct the response.
		var cr = cacheResponse{}
		err = json.Unmarshal((response).([]byte), &cr)
		if err != nil {
			return err
		}

		// Loop through all the headers from the response and set them.
		for name, values := range cr.Headers {
			for _, value := range values {
				w.Header().Add(name, value)
			}
		}

		// Write the HTTP status header.
		w.WriteHeader(cr.StatusCode)
		// Write the body of the response.
		w.Write(cr.Body)

		// Return nil since we don't need to do anything else.
		return nil
	}

	// Wasn't cached :(
	w.Header().Add("Cache-Status", "miss")

	ch := httpMetrics.cacheMiss.With(labels)
	ch.Inc()

	// Save to cache
	recorder := httptest.NewRecorder()

	// Next, please.
	next.ServeHTTP(recorder, r)

	body, err := ioutil.ReadAll(recorder.Result().Body)
	if err != nil {
		c.logger.Error(err.Error())
	}

	cr := &cacheResponse{
		Status:        recorder.Result().Status,
		StatusCode:    recorder.Result().StatusCode,
		Headers:       recorder.Result().Header,
		Body:          body,
		ContentLength: recorder.Result().ContentLength,
	}

	response, err := json.Marshal(cr)
	if err != nil {
		c.logger.Error(err.Error())
	}

	c.Store.Put(key, response, time.Second*time.Duration(c.Config.Expire))

	for name, values := range cr.Headers {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}
	w.WriteHeader(recorder.Code)
	w.Write(body)

	return nil
}

func key(r *http.Request) string {
	return "request:" + r.Method + ":" + r.Host + ":" + r.URL.Path
}
