package main

import (
	"net/http"
	"time"
	"log"
	"encoding/json"
	"errors"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"bytes"
	"strconv"
)

func main() {
	router := mux.NewRouter()

	router.HandleFunc("/users", users).Methods("GET")
	router.HandleFunc("/users/{id}", user).Methods("GET")
	router.HandleFunc("/error", err).Methods("GET")
	router.Handle("/metrics", promhttp.HandlerFor(
		prometheus.DefaultGatherer,
		promhttp.HandlerOpts{},
	))

	log.Fatal(http.ListenAndServe(":8080", NewLogginHandler(router)))
}

func users(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, allUsers)
}

func err(w http.ResponseWriter, r *http.Request) {
	WriteErr(w, errors.New("some error"), http.StatusInternalServerError)
}

func user(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if id == "" {
		WriteErr(w, errors.New("please provide User id"), http.StatusBadRequest)
	}

	user := allUsers[id]
	if user == nil {
		WriteErr(w, errors.New("can't find user with ID: "+id), http.StatusNotFound)
	}

	WriteJSON(w, user)
}

type User struct {
	ID         string    `json:"id"`
	FirstName  string    `json:"firstName"`
	SecondName string    `json:"secondName"`
	BirthDate  time.Time `json:"birthDate"`
}

var allUsers = map[string]*User{
	"1": {
		ID:         "1",
		FirstName:  "Tom",
		SecondName: "Tailor",
		BirthDate:  time.Date(1988, 6, 1, 0, 0, 0, 0, time.UTC),
	},
	"2": {
		ID:         "2",
		FirstName:  "Tommy",
		SecondName: "Hilfiger",
		BirthDate:  time.Date(1956, 1, 14, 0, 0, 0, 0, time.UTC),
	},
	"3": {
		ID:         "3",
		FirstName:  "Coco",
		SecondName: "Chanel",
		BirthDate:  time.Date(1921, 4, 23, 0, 0, 0, 0, time.UTC),
	},
}

func WriteJSON(w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json, err := json.Marshal(response)
	if err != nil {
		return err
	}

	if _, err := w.Write(json); err != nil {
		return err
	}

	return nil
}

func WriteErr(w http.ResponseWriter, err error, httpCode int) {
	logrus.Error(err.Error())

	// write error to response
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	var errMap = map[string]interface{}{
		"httpStatus": httpCode,
		"error":      err.Error(),
	}

	errJson, _ := json.Marshal(errMap)
	http.Error(w, string(errJson), httpCode)
}

// callStats holds various stats associated with HTTP request-response pair.
type callStats struct {
	w       http.ResponseWriter
	code    int // response status code
	resSize int64
}

func newCallStats(w http.ResponseWriter) *callStats {
	return &callStats{w: w}
}

func (r *callStats) Header() http.Header {
	return r.w.Header()
}

func (r *callStats) WriteHeader(code int) {
	r.w.WriteHeader(code)
	r.code = code
}

func (r *callStats) Write(p []byte) (n int, err error) {
	if r.code == 0 {
		r.code = http.StatusOK
	}
	n, err = r.w.Write(p)
	r.resSize += int64(n)
	return
}

func (r *callStats) StatusCode() int {
	return r.code
}

func (r *callStats) ResponseSize() int64 {
	return r.resSize
}

// NewLogginHandler creates and returns LoggingHandler with custom metrics.
func NewLogginHandler(root http.Handler) http.Handler {
	duration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "monitoring",
			Subsystem: "rest",
			Name:      "http_durations_histogram_seconds",
			Help:      "Request time duration.",
		},
		[]string{"code", "method", "endpoint"},
	)

	requests := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "monitoring",
			Subsystem: "rest",
			Name:      "http_requests_total",
			Help:      "Total number of requests received.",
		},
		[]string{"code", "method", "endpoint"},
	)


	lh := LoggingHandler{root: root, duration: duration, requests: requests}

	prometheus.DefaultRegisterer.Register(lh)

	return lh
}

// LoggingHandler writes basic information about each request and response to
// the log.
type LoggingHandler struct {
	root http.Handler
	duration *prometheus.HistogramVec
	requests *prometheus.CounterVec
}

func (h LoggingHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	t := time.Now()
	stats := newCallStats(w)
	h.root.ServeHTTP(stats, req)

	elapsed := time.Since(t)

	buf := new(bytes.Buffer)
	buf.WriteString(getRemoteAddr(req))
	buf.WriteString(" - \"")
	buf.WriteString(req.Method)
	buf.WriteByte(' ')
	buf.WriteString(req.RequestURI)
	buf.WriteByte(' ')
	buf.WriteString(req.Proto)
	buf.WriteString("\" ")
	buf.WriteString(strconv.Itoa(stats.StatusCode()))
	buf.WriteByte(' ')
	buf.WriteString(strconv.FormatInt(stats.ResponseSize(), 10))
	buf.WriteString(" \"")
	buf.WriteString(req.UserAgent())
	buf.WriteString("\" Took: ")
	buf.WriteString(elapsed.String())

	h.requests.WithLabelValues(strconv.Itoa(stats.StatusCode()), strconv.Itoa(stats.StatusCode()), req.RequestURI).Inc()
	h.duration.WithLabelValues(strconv.Itoa(stats.StatusCode()), strconv.Itoa(stats.StatusCode()), req.RequestURI).Observe(elapsed.Seconds())

	logrus.Infof(buf.String())
}

func getRemoteAddr(r *http.Request) string {
	forwaredFor := r.Header.Get("X-Forwarded-For")
	if forwaredFor == "" {
		return r.RemoteAddr
	}

	return forwaredFor
}

// Describe implements prometheus.Collector interface.
func (d LoggingHandler) Describe(in chan<- *prometheus.Desc) {
	d.duration.Describe(in)
	d.requests.Describe(in)
}

// Collect implements prometheus.Collector interface.
func (d LoggingHandler) Collect(in chan<- prometheus.Metric) {
	d.duration.Collect(in)
	d.requests.Collect(in)
}
