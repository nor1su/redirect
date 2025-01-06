package main

import (
	"encoding/json"
	"flag"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type Stats struct {
	TotalRedirects int            `json:"total_redirects"`
	Paths          map[string]int `json:"paths"`
	StartTime      time.Time      `json:"start_time"`
	Mutex          sync.Mutex     `json:"-"`
}

type Paths struct {
	StatsPath     string `json:"stats_path"`
	StatsJSONPath string `json:"stats_json_path"`
	ResetPath     string `json:"reset_path"`
}

type TemplateData struct {
	TotalRedirects int
	StartTime      time.Time
	Paths          map[string]int
	ResetPath      string
}

var (
	stats     = Stats{Paths: make(map[string]int), StartTime: time.Now()}
	paths     = Paths{}
	statsFile = "stats.json"
	pathsFile = "paths.json"
)

func generateRandomPath(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	sb := strings.Builder{}
	for i := 0; i < length; i++ {
		sb.WriteByte(charset[seededRand.Intn(len(charset))])
	}
	return sb.String()
}

func loadPaths() error {
	if _, err := os.Stat(pathsFile); os.IsNotExist(err) {
		paths.StatsPath = generateRandomPath(16)
		paths.StatsJSONPath = generateRandomPath(16)
		paths.ResetPath = generateRandomPath(16)
		data, err := json.Marshal(paths)
		if err != nil {
			return err
		}
		return os.WriteFile(pathsFile, data, 0644)
	}
	data, err := os.ReadFile(pathsFile)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &paths)
}

func savePaths() error {
	data, err := json.Marshal(paths)
	if err != nil {
		return err
	}
	return os.WriteFile(pathsFile, data, 0644)
}

func loadStats() error {
	if _, err := os.Stat(statsFile); os.IsNotExist(err) {
		return nil
	}
	data, err := os.ReadFile(statsFile)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &stats)
}

func saveStats() error {
	stats.Mutex.Lock()
	defer stats.Mutex.Unlock()

	data, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(statsFile, data, 0644)
}

func redirectHandler(w http.ResponseWriter, r *http.Request, baseURL string) {
	path := r.URL.Path
	query := r.URL.RawQuery
	targetURL := baseURL + path
	if query != "" {
		targetURL += "?" + query
	}
	stats.Mutex.Lock()
	stats.TotalRedirects++
	stats.Paths[path]++
	stats.Mutex.Unlock()
	if err := saveStats(); err != nil {
		http.Error(w, "Error saving statistics", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, targetURL, http.StatusMovedPermanently)
}

func statsHandlerJSON(w http.ResponseWriter, r *http.Request) {
	stats.Mutex.Lock()
	defer stats.Mutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(stats); err != nil {
		http.Error(w, "Error encoding statistics", http.StatusInternalServerError)
	}
}

func statsHandlerHTML(w http.ResponseWriter, r *http.Request) {
	stats.Mutex.Lock()
	defer stats.Mutex.Unlock()

	tmpl := `
	<!DOCTYPE html>
	<html>
	<head>
		<title>Redirect Statistics</title>
		<style>
			table {
				width: 50%;
				border-collapse: collapse;
				margin: 25px 0;
				font-size: 18px;
				text-align: left;
			}
			th, td {
				padding: 12px;
				border-bottom: 1px solid #ddd;
			}
			th {
				background-color: #f2f2f2;
			}
			button {
				padding: 10px 20px;
				font-size: 16px;
				margin-top: 20px;
			}
		</style>
	</head>
	<body>
		<h1>Redirect Statistics</h1>
		<p><strong>Total Redirects:</strong> {{.TotalRedirects}}</p>
		<p><strong>Server Start Time:</strong> {{.StartTime}}</p>
		<h2>Redirects by Path:</h2>
		<table>
			<tr>
				<th>Path</th>
				<th>Count</th>
			</tr>
			{{range $path, $count := .Paths}}
			<tr>
				<td>{{$path}}</td>
				<td>{{$count}}</td>
			</tr>
			{{end}}
		</table>
		<form action="/{{.ResetPath}}" method="POST">
			<button type="submit">Reset Statistics</button>
		</form>
	</body>
	</html>
	`
	t, err := template.New("stats").Parse(tmpl)
	if err != nil {
		http.Error(w, "Error processing template", http.StatusInternalServerError)
		return
	}

	data := TemplateData{
		TotalRedirects: stats.TotalRedirects,
		StartTime:      stats.StartTime,
		Paths:          stats.Paths,
		ResetPath:      paths.ResetPath,
	}

	if err := t.Execute(w, data); err != nil {
		http.Error(w, "Error rendering statistics", http.StatusInternalServerError)
	}
}

func resetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	stats.Mutex.Lock()
	stats = Stats{Paths: make(map[string]int), StartTime: time.Now()}
	stats.Mutex.Unlock()
	if err := saveStats(); err != nil {
		http.Error(w, "Error saving statistics", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/"+paths.StatsPath, http.StatusSeeOther)
}

func main() {
	baseURL := flag.String("base", "https://example.com", "Base URL to redirect to")
	addr := flag.String("addr", ":8080", "Address and port to listen on")
	flag.Parse()

	if err := loadPaths(); err != nil {
		log.Fatalf("Error loading paths: %v", err)
	}

	if err := loadStats(); err != nil {
		log.Fatalf("Error loading stats: %v", err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		redirectHandler(w, r, *baseURL)
	})
	http.HandleFunc("/"+paths.StatsJSONPath, statsHandlerJSON)
	http.HandleFunc("/"+paths.StatsPath, statsHandlerHTML)
	http.HandleFunc("/"+paths.ResetPath, resetHandler)

	log.Printf("Server is running on %s and redirecting requests to %s", *addr, *baseURL)
	log.Printf("Statistics available at http://localhost%s/%s and http://localhost%s/%s/json", *addr, paths.StatsPath, *addr, paths.StatsJSONPath)
	log.Printf("Reset statistics via http://localhost%s/%s", *addr, paths.ResetPath)

	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
