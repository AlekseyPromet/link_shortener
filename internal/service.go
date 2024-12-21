package internal

import (
	"context"
	"time"

	h2 "github.com/speps/go-hashids/v2"

	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"regexp"

	"github.com/redis/go-redis/v9"
)

var allRegexps = make(map[string]*regexp.Regexp)
var hashId *h2.HashID

func init() {
	allRegexps["email"] = regexp.MustCompile(`(?P<name>[-\w\d\.]+?)(?:\s+at\s+|\s*@\s*|\s*(?:[\[\]@]){3}\s*)(?P<host>[-\w\d\.]*?)\s*(?:dot|\.|(?:[\[\]dot\.]){3,5})\s*(?P<domain>\w+)`)
	allRegexps["bitcoin"] = regexp.MustCompile(`\b([13][a-km-zA-HJ-NP-Z1-9]{25,34}|bc1[ac-hj-np-zAC-HJ-NP-Z02-9]{11,71})`)
	allRegexps["ssn"] = regexp.MustCompile(`\d{3}-\d{2}-\d{4}`)
	allRegexps["uri"] = regexp.MustCompile(`[\w]+://[^/\s?#]+[^\s?#]+(?:\?[^\s#]*)?(?:#[^\s]*)?`)
	allRegexps["tel"] = regexp.MustCompile(`\+\d{1,4}?[-.\s]?\(?\d{1,3}?\)?[-.\s]?\d{1,4}[-.\s]?\d{1,4}[-.\s]?\d{1,9}`)

	data := h2.NewData()
	data.Alphabet = "QWERTYUIOPASDFGHJKLZXCVBNMqwertyuiopasdfghjklzxcvbnm1234567890"
	data.MinLength = 12

	var err error
	hashId, err = h2.NewWithData(data)
	if err != nil {
		panic(err)
	}
}

type ServiceConfig struct {
	Host    string `json:"host"`
	Port    int    `json:"port"`
	Count   int    `json:"count"`
	Version string `json:"version"`
}

func LoadServiceConfig(filePath string) (*ServiceConfig, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file %w", err)
	}
	defer file.Close()

	var config ServiceConfig
	err = json.NewDecoder(file).Decode(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to decode config file: %w", err)
	}

	slog.Info("Loaded config from file")

	return &config, nil
}

type ServiceShortnessLink struct {
	redisClient *redis.Client
}

func NewServiceShortnessLink(ctx context.Context, cfg *ServiceConfig, redisClient *redis.Client) error {

	service := &ServiceShortnessLink{
		redisClient: redisClient,
	}

	handler := http.NewServeMux()
	handler.HandleFunc("GET /short", service.GetFullURLbyShortLink)
	handler.HandleFunc("POST /short", service.CreateShortLink)

	return http.ListenAndServe(fmt.Sprintf("%v:%v", cfg.Host, cfg.Port), handler)
}

func (s *ServiceShortnessLink) CreateShortLink(w http.ResponseWriter, r *http.Request) {

	err := r.ParseForm()
	if err != nil {
		slog.Debug("parse form error")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Failed to parse form: %v", err)
		return
	}

	fullURL := r.PostForm.Get("fullUrl")

	if len(fullURL) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Missing source URL parameter")
		return
	}

	if len(fullURL) > 512 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Source URL exceeds maximum length of 512 characters")
		return
	}

	ttl := r.PostForm.Get("ttl")
	if len(ttl) == 0 {
		ttl = "3600s"
	} else if len(ttl) < 100 {
		ttl += "s"
	}

	ttlDuration, err := time.ParseDuration(ttl)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Invalid TTL format: %v", err)
		return
	}

	isValid := false
	for _, regexpCompile := range allRegexps {
		if regexpCompile.MatchString(fullURL) {
			isValid = true
			break
		}
	}

	if !isValid {
		slog.Debug("uncnown format source ", slog.String("input", fullURL))
	}

	shortLink, err := hashId.EncodeHex(fullURL)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Failed to generate short link: %v", err)
		return
	}

	strCmd := s.redisClient.Get(context.Background(), shortLink)
	if strCmd.Err() != redis.Nil {
		w.WriteHeader(http.StatusConflict)
		fmt.Fprintf(w, "Short link is exist")
		return
	}

	if err := s.redisClient.Set(context.Background(), shortLink, fullURL, time.Duration(ttlDuration)*time.Second).Err(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Failed to store short link in Redis: %v", err)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "%s", shortLink)
	w.WriteHeader(http.StatusOK)
}

func (s *ServiceShortnessLink) GetFullURLbyShortLink(w http.ResponseWriter, r *http.Request) {
	// Retrieve short link from request parameters
	shortLink := r.URL.Query().Get("short")

	// Retrieve full URL from Redis
	fullURL, err := s.redisClient.Get(context.Background(), shortLink).Result()
	if err == redis.Nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "Short link not found")
		return

	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Failed to retrieve full URL from Redis: %v", err)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "%s", fullURL)
}
