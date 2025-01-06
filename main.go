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

func init() {
    rand.Seed(time.Now().UnixNano())
}

func generateRandomPath(length int) string {
    const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
    sb := strings.Builder{}
    sb.Grow(length)
    for i := 0; i < length; i++ {
        sb.WriteByte(charset[rand.Intn(len(charset))])
    }
    return sb.String()
}

func loadPaths() {
    if _, err := os.Stat(pathsFile); os.IsNotExist(err) {
        generateAndSavePaths()
        return
    }
    data, err := os.ReadFile(pathsFile)
    if err != nil {
        log.Printf("Error reading paths file (%s): %v. Generating new paths.\n", pathsFile, err)
        generateAndSavePaths()
        return
    }
    if err := json.Unmarshal(data, &paths); err != nil {
        log.Printf("Error unmarshalling paths file (%s): %v. Generating new paths.\n", pathsFile, err)
        generateAndSavePaths()
        return
    }
}

func generateAndSavePaths() {
    paths.StatsPath = generateRandomPath(16)
    paths.StatsJSONPath = generateRandomPath(16)
    paths.ResetPath = generateRandomPath(16)
    data, err := json.Marshal(paths)
    if err != nil {
        log.Printf("Error marshalling newly generated paths: %v\n", err)
        return
    }
    if err := os.WriteFile(pathsFile, data, 0644); err != nil {
        log.Printf("Error writing newly generated paths to file: %v\n", err)
    }
}

func loadStats() {
    if _, err := os.Stat(statsFile); os.IsNotExist(err) {
        return
    }
    data, err := os.ReadFile(statsFile)
    if err != nil {
        log.Printf("Error reading stats file (%s): %v. Continuing with default stats.\n", statsFile, err)
        return
    }
    if err := json.Unmarshal(data, &stats); err != nil {
        log.Printf("Error unmarshalling stats file (%s): %v. Continuing with default stats.\n", statsFile, err)
        return
    }
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
        log.Printf("Error saving statistics: %v\n", err)
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
        log.Printf("Error encoding JSON stats: %v\n", err)
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
    <meta charset="UTF-8" />
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
        <button type="submit">CLEAR ALL STATISTICS</button>
    </form>
</body>
</html>
`
    t, err := template.New("stats").Parse(tmpl)
    if err != nil {
        log.Printf("Error parsing HTML template: %v\n", err)
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
        log.Printf("Error executing HTML template: %v\n", err)
        http.Error(w, "Error rendering statistics", http.StatusInternalServerError)
    }
}

func resetHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
        return
    }

    stats.Mutex.Lock()
    stats.TotalRedirects = 0
    stats.Paths = make(map[string]int)
    stats.StartTime = time.Now()
    stats.Mutex.Unlock()

    if err := saveStats(); err != nil {
        log.Printf("Error saving reset stats: %v\n", err)
        http.Error(w, "Error saving statistics", http.StatusInternalServerError)
        return
    }

    http.Redirect(w, r, "/"+paths.StatsPath, http.StatusSeeOther)
}

func main() {
    baseURL := flag.String("base", "https://example.com", "Base URL to redirect to")
    addr := flag.String("addr", ":8080", "Address and port to listen on")
    flag.Parse()
    loadPaths()
    loadStats()
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        redirectHandler(w, r, *baseURL)
    })
    http.HandleFunc("/"+paths.StatsJSONPath, statsHandlerJSON)
    http.HandleFunc("/"+paths.StatsPath, statsHandlerHTML)
    http.HandleFunc("/"+paths.ResetPath, resetHandler)
    log.Printf("Server is running on %s and redirecting requests to %s\n", *addr, *baseURL)
    log.Printf("Statistics (HTML) at http://localhost%s/%s\n", *addr, paths.StatsPath)
    log.Printf("Statistics (JSON) at http://localhost%s/%s\n", *addr, paths.StatsJSONPath)
    log.Printf("Reset stats at http://localhost%s/%s (POST)\n", *addr, paths.ResetPath)
    if err := http.ListenAndServe(*addr, nil); err != nil {
        log.Fatalf("Server failed to start: %v\n", err)
    }
}
