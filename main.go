package main

import (
    "log"
    "net/http"
    "os"
    "path/filepath"
    "strings"
    "sync"
    "time"
    "os/exec"
)

var (
    lastModTime = time.Time{}
    mu sync.Mutex
    root = "../"
    extensions = []string{".html", ".css", ".js", ".ts"}
)

func findFiles(root string, exts []string) ([]string, error) {
    var files []string
    err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
        if err != nil || info.IsDir() || strings.Contains(path, "live") {
            return nil
        }
        for _, ext := range exts {
            if filepath.Ext(path) == ext {
                files = append(files, path)
            }
        }
        return nil
    })
    return files, err
}

func watchFiles() {
    files, _ := findFiles(root, extensions)
    modTimes := make(map[string]time.Time)
    
    // initialize modTimes with current values
    for _, file := range files {
        if info, err := os.Stat(file); err == nil {
            modTimes[file] = info.ModTime()
        }
    }
    
    for {
		var isChanged bool
        for _, file := range files {
            info, _ := os.Stat(file)
            if info.ModTime() != modTimes[file] {
				isChanged = true
                log.Println("File changed:", file)
                modTimes[file] = info.ModTime()
                
                if filepath.Ext(file) == ".ts" {
                    log.Println("Compiling TypeScript project")
                    cmd := exec.Command("tsc", "--project", "../tsconfig.json")
                    if err := cmd.Run(); err != nil {
                        log.Println("TypeScript compilation error:", err)
                    }
                }
                
                mu.Lock()
                lastModTime = info.ModTime()
                mu.Unlock()
            }
        }
		sleepTime := 1 * time.Second
		if isChanged {
			sleepTime = 4 * time.Second
		} 
		time.Sleep(sleepTime)
		
    }
}

func updatesHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")

    lastEvent := ""
    for {
        mu.Lock() 
        current := lastModTime.String()
        mu.Unlock()

		if strings.Contains(current, ".ts") {
			log.Println("TypeScript file changed, sending reload event to client")
			continue
		}

        if current != lastEvent {
			log.Println("Sending update event to client")
            w.Write([]byte("data: " + current + "\n\n"))
            if f, ok := w.(http.Flusher); ok {
                f.Flush()
            }
            lastEvent = current
        }
        time.Sleep(3 * time.Second)
    }
}

func main() {
    var rootDir string
    filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
        if !info.IsDir() && filepath.Base(path) == "index.html" {
            rootDir = filepath.Dir(path)
            return filepath.SkipAll
        }
        return nil
    })
    if rootDir == "" {
        rootDir = root
    }
    
    http.Handle("/", http.FileServer(http.Dir(rootDir)))
    http.HandleFunc("/live-reload.js", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/javascript")
        http.ServeFile(w, r, "live-reload.js")
    })
    http.HandleFunc("/updates", updatesHandler)

    go watchFiles()
    
	log.Println("Server started at http://localhost:8080")
    http.ListenAndServe(":8080", nil)
}