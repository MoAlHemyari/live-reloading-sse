package main

import (
	"net/http"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
	"sync"
	"strings"
)

var (
	lastModTime time.Time 
	mu	    sync.Mutex
)

func findFiles(root string, extensions []string, excludeDir string) ([]string, error) {
    var files []string

    err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }

        if strings.HasPrefix(path, excludeDir) && filepath.Base(path) != "live-reload.js" {
            if info.IsDir() {
                return filepath.SkipDir
            }
            return nil
        }

        if !info.IsDir() {
            for _, ext := range extensions {
                if filepath.Ext(path) == ext {
                    files = append(files, path)
                    break
                }
            }
        }

        return nil
    })

    if err != nil {
        return nil, err
    }
    return files, nil
}

func watchFiles() {
	extensions := []string{".html", ".css", ".js", ".ts"}

	excludeDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}

	files, err := findFiles("../", extensions, excludeDir)
	if err != nil {
		log.Fatal(err)
	}

	modTimes := make(map[string]time.Time)
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			log.Fatal(err)
		}
		modTimes[file] = info.ModTime()
	}

	for {
		time.Sleep(1 * time.Second)
		for _, file := range files {
			info, err := os.Stat(file)
			if err != nil {
				log.Fatal(err)
			}
			if info.ModTime() != modTimes[file] {
				log.Println("File Changed: ", file)
				modTimes[file] = info.ModTime()
			
				if filepath.Ext(file) == ".ts" {
					log.Println("Running tsc for: ", file)
					runTSC(file)
				} else {
					mu.Lock()
					lastModTime = info.ModTime()
					mu.Unlock()
				}
			}
		}
	}
}

func runTSC(file string) {
	cmd := exec.Command("tsc", file)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		log.Println("Error running tsc: ", err)
	}
}

func updatesHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")

    var lastEvent string
    for {
        mu.Lock()
        currentUpdate := lastModTime.String()
        mu.Unlock()

        if currentUpdate != lastEvent {
            _, err := w.Write([]byte("data: " + currentUpdate + "\n\n"))
            if err != nil {
                if isBrokenPipe(err) {
                    log.Println("Page has reloaded")
                } else {
                    log.Println("SSE error:", err)
                }
                return
            }
            if flusher, ok := w.(http.Flusher); ok {
                flusher.Flush()
            } else {
                log.Println("Streaming unsupported")
                return
            }

            lastEvent = currentUpdate
        }

        time.Sleep(1 * time.Second)
    }
}

func isBrokenPipe(err error) bool {
    return err != nil && (strings.Contains(err.Error(), "broken pipe") || strings.Contains(err.Error(), "reset by peer"))
}

func main() {
    http.Handle("/", http.FileServer(http.Dir("../")))
    http.Handle("/live-reload.js", http.FileServer(http.Dir("./")))
    http.HandleFunc("/updates", updatesHandler)
    go watchFiles()
    log.Println("Server started at http://localhost:8080")
    err := http.ListenAndServe(":8080", nil)
    if err != nil {
        log.Fatal(err)
    }
    select {}
}