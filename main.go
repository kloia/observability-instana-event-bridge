package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

type Config struct {
	Server struct {
		Port string `json:"port"`
		Hostname string `json:"hostname"`
	} `json:"server"`

	Limits struct {
		MaxFieldLength int `json:"maxFieldLength"`
	} `json:"limits"`

	Outputs struct {
		Syslog struct {
			Enabled  bool   `json:"enabled"`
			Protocol string `json:"protocol"`
			Host     string `json:"host"`
			Port     int    `json:"port"`
		} `json:"syslog"`

		File struct {
			Enabled bool   `json:"enabled"`
			Path    string `json:"path"`
		} `json:"file"`
	} `json:"outputs"`
}

type WebhookPayload struct {
	Issue Issue `json:"issue"`
}

type Issue struct {
	ID               string              `json:"id"`
	Type             string              `json:"type"`
	State            string              `json:"state"`
	Start            int64               `json:"start"`
	End              int64               `json:"end"`
	Severity         int                 `json:"severity"`
	Text             string              `json:"text"`
	Description      string              `json:"description"`
	Suggestion       string              `json:"suggestion"`
	Link             string              `json:"link"`
	EntityType       string              `json:"entityType"`
	CustomZone       string              `json:"customZone"`
	AvailabilityZone string              `json:"availabilityZone"`
	Zone             string              `json:"zone"`
	FQDN             string              `json:"fqdn"`
	Entity           string              `json:"entity"`
	EntityLabel      string              `json:"entityLabel"`
	Tags             string              `json:"tags"`
	Container        string              `json:"container"`
	Service          string              `json:"service"`
	ContainerNames   []string            `json:"containerNames"`
	MetricNames      []string            `json:"metricNames"`
	CustomPayloads   map[string][]string `json:"customPayloads"`
}

var config Config

func main() {

	loadConfig()

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/webhook/instana", webhookHandler)

	addr := ":" + config.Server.Port

	log.Printf(
		"instana-event-bridge started on %s",
		addr,
	)

	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func healthHandler(
	w http.ResponseWriter,
	r *http.Request,
) {

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func webhookHandler(
	w http.ResponseWriter,
	r *http.Request,
) {

	if r.Method != http.MethodPost {

		http.Error(
			w,
			"method not allowed",
			http.StatusMethodNotAllowed,
		)

		return
	}

	var payload WebhookPayload

	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {

		log.Printf(
			"invalid payload: %v",
			err,
		)

		http.Error(
			w,
			"invalid payload",
			http.StatusBadRequest,
		)

		return
	}

	syslogMessage := buildSyslogMessage(
		payload.Issue,
	)

	log.Printf(
		"event processed type=%s id=%s",
		payload.Issue.Type,
		payload.Issue.ID,
	)

	if config.Outputs.Syslog.Enabled {

		err := sendSyslog(syslogMessage)
		if err != nil {

			log.Printf(
				"syslog send failed: %v",
				err,
			)
		}
	}

	if config.Outputs.File.Enabled {

		err := writeToFile(syslogMessage)
		if err != nil {

			log.Printf(
				"file write failed: %v",
				err,
			)
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("received"))
}

func buildSyslogMessage(issue Issue) string {

	timestamp := time.Now().Format(
		"Jan 02 15:04:05",
	)

	var b strings.Builder

	b.WriteString(fmt.Sprintf(
		"<134>%s %s ",
		timestamp,
		getHostname(),
	))

	addField(&b, "source", "instana")
	addField(&b, "event_type", issue.Type)
	addField(&b, "state", issue.State)

	if issue.Severity != 0 {

		addField(
			&b,
			"severity",
			mapSeverity(issue.Severity),
		)

	} else {

		addField(
			&b,
			"severity",
			"not available",
		)
	}

	addField(
		&b,
		"raw_severity",
		fmt.Sprintf("%d", issue.Severity),
	)

	addField(
		&b,
		"event_id",
		issue.ID,
	)

	addField(
		&b,
		"start_time",
		epochMillisToTime(issue.Start),
	)

	addField(
		&b,
		"end_time",
		epochMillisToTime(issue.End),
	)

	addField(
		&b,
		"message",
		issue.Text,
	)

	addField(
		&b,
		"description",
		issue.Description,
	)

	addField(
		&b,
		"suggestion",
		issue.Suggestion,
	)

	addField(
		&b,
		"entity_type",
		issue.EntityType,
	)

	addField(
		&b,
		"custom_zone",
		issue.CustomZone,
	)

	addField(
		&b,
		"availability_zone",
		issue.AvailabilityZone,
	)

	addField(
		&b,
		"zone",
		issue.Zone,
	)

	addField(
		&b,
		"host",
		issue.FQDN,
	)

	addField(
		&b,
		"entity",
		issue.Entity,
	)

	addField(
		&b,
		"entity_label",
		issue.EntityLabel,
	)

	addField(
		&b,
		"container",
		issue.Container,
	)

	addField(
		&b,
		"service",
		issue.Service,
	)

	addField(
		&b,
		"container_names",
		joinOrDefault(issue.ContainerNames),
	)

	addField(
		&b,
		"metric_names",
		joinOrDefault(issue.MetricNames),
	)

	addField(
		&b,
		"tags",
		issue.Tags,
	)

	addField(
		&b,
		"custom_payloads",
		mapToJSONString(issue.CustomPayloads),
	)

	addField(
		&b,
		"link",
		issue.Link,
	)

	return strings.TrimSpace(
		b.String(),
	)
}

func addField(
	b *strings.Builder,
	key string,
	value string,
) {

	value = sanitize(value)

	value = escapeQuotes(value)

	value = truncateField(value)

	if value == "" {
		value = "not available"
	}

	b.WriteString(fmt.Sprintf(
		`%s="%s" `,
		key,
		value,
	))
}

func sendSyslog(message string) error {

	address := fmt.Sprintf(
		"%s:%d",
		config.Outputs.Syslog.Host,
		config.Outputs.Syslog.Port,
	)

	conn, err := net.Dial(
		config.Outputs.Syslog.Protocol,
		address,
	)
	if err != nil {
		return err
	}

	defer conn.Close()

	_, err = conn.Write([]byte(message))

	return err
}

func writeToFile(message string) error {

	path := config.Outputs.File.Path

	file, err := os.OpenFile(
		path,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0644,
	)
	if err != nil {
		return err
	}

	defer file.Close()

	_, err = file.WriteString(
		message + "\n",
	)

	return err
}

func sanitize(value string) string {

	value = strings.ReplaceAll(
		value,
		"\n",
		" ",
	)

	value = strings.ReplaceAll(
		value,
		"\r",
		" ",
	)

	value = strings.TrimSpace(value)

	var b bytes.Buffer

	for _, r := range value {

		if r >= 32 && r <= 126 {
			b.WriteRune(r)
		}
	}

	return b.String()
}

func escapeQuotes(value string) string {

	value = strings.ReplaceAll(
		value,
		`"`,
		`\\"`,
	)

	return value
}

func truncateField(value string) string {

	maxLength := config.Limits.MaxFieldLength

	if maxLength <= 0 {
		return value
	}

	if len(value) <= maxLength {
		return value
	}

	return value[:maxLength] + "...TRUNCATED"
}

func mapSeverity(severity int) string {

	switch severity {

	case 5:
		return "CRITICAL"

	case 10:
		return "WARNING"

	case 20:
		return "INFO"

	default:
		return "UNKNOWN"
	}
}

func epochMillisToTime(ms int64) string {

	if ms == 0 {
		return "not available"
	}

	t := time.UnixMilli(ms)

	return t.Format(time.RFC3339)
}

func joinOrDefault(values []string) string {

	if len(values) == 0 {
		return "not available"
	}

	return strings.Join(
		values,
		",",
	)
}

func mapToJSONString(v interface{}) string {

	if v == nil {
		return "not available"
	}

	data, err := json.Marshal(v)
	if err != nil {
		return "not available"
	}

	result := string(data)

	if result == "null" || result == "" {
		return "not available"
	}

	return result
}

func getHostname() string {

	if config.Server.Hostname != "" {
		return config.Server.Hostname
	}

	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		return "localhost"
	}

	return hostname
}

func loadConfig() {

	file, err := os.Open("config.json")
	if err != nil {

		log.Fatalf(
			"config open failed: %v",
			err,
		)
	}

	defer file.Close()

	err = json.NewDecoder(file).Decode(&config)
	if err != nil {

		log.Fatalf(
			"config parse failed: %v",
			err,
		)
	}
}