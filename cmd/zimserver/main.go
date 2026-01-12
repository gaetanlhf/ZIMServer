package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gaetanlhf/ZIMServer/internal/web"
)

var version = "dev"

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
)

func main() {
	tempSet := flag.NewFlagSet("", flag.ContinueOnError)
	tempSet.Usage = func() {} // Disable default usage
	showHelp := tempSet.Bool("h", false, "Show this help message")
	tempSet.Bool("help", false, "Show this help message")
	showVersion := tempSet.Bool("v", false, "Show version")
	tempSet.Bool("version", false, "Show version")

	if err := tempSet.Parse(os.Args[1:]); err != nil {
		printUsage()
		os.Exit(1)
	}

	if *showHelp || (tempSet.Lookup("help") != nil && tempSet.Lookup("help").Value.String() == "true") {
		printUsage()
		os.Exit(0)
	}
	if *showVersion || (tempSet.Lookup("version") != nil && tempSet.Lookup("version").Value.String() == "true") {
		fmt.Printf("ZIMServer %s\n", version)
		os.Exit(0)
	}

	if len(os.Args) < 2 {
		logError("No ZIM files or directories specified. Use -h for help.")
		os.Exit(1)
	}
	runServeCommand(os.Args[1:])
}

func runServeCommand(args []string) {
	serveCmd := flag.NewFlagSet("serve", flag.ContinueOnError)
	serveCmd.Usage = func() {} // Disable default usage

	host := serveCmd.String("host", "localhost", "HTTP server host")
	serveCmd.String("H", "localhost", "HTTP server host (short)")

	port := serveCmd.String("port", "8080", "HTTP server port")
	serveCmd.String("p", "8080", "HTTP server port (short)")

	serveCmd.Bool("h", false, "Show this help message")
	serveCmd.Bool("help", false, "Show this help message")
	serveCmd.Bool("v", false, "Show version")
	serveCmd.Bool("version", false, "Show version")

	if err := serveCmd.Parse(args); err != nil {
		printUsage()
		os.Exit(1)
	}

	if serveCmd.Lookup("H").Value.String() != "localhost" {
		*host = serveCmd.Lookup("H").Value.String()
	}
	if serveCmd.Lookup("p").Value.String() != "8080" {
		*port = serveCmd.Lookup("p").Value.String()
	}

	paths := serveCmd.Args()
	allPaths := make([]string, 0)
	allPaths = append(allPaths, paths...)

	if len(allPaths) == 0 {
		logError("No ZIM files or directories specified")
		os.Exit(1)
	}

	runServer(*host, *port, allPaths)
}

func printUsage() {
	fmt.Printf("%sZIMServer - A modern and lightweight alternative to kiwix-serve for your zim files %s\n\n", colorCyan, colorReset)
	fmt.Printf("%sUsage:%s\n", colorYellow, colorReset)
	fmt.Println("  zimserver [options] [files/directories...]")
	fmt.Println()
	fmt.Printf("%sOptions:%s\n", colorYellow, colorReset)
	fmt.Println("  -H, --host <host>        HTTP server host (default: localhost)")
	fmt.Println("  -p, --port <port>        HTTP server port (default: 8080)")
	fmt.Println("  -h, --help               Show this help message")
	fmt.Println("  -v, --version            Show version")
	fmt.Println()
	fmt.Printf("%sExamples:%s\n", colorYellow, colorReset)
	fmt.Println("  zimserver file1.zim file2.zim")
	fmt.Println("  zimserver ./zim-files")
	fmt.Println("  zimserver --host 0.0.0.0 --port 3000 ./zim-files")
	fmt.Println("  zimserver file1.zim ./zim-dir")
}

func runServer(host, port string, paths []string) {
	logSuccess("ZIMServer starting (version: %s)", version)

	server, err := web.NewServer(version)
	if err != nil {
		logError("Failed to create server: %v", err)
		os.Exit(1)
	}

	addr := host + ":" + port
	logInfo("Listen on %shttp://%s:%s%s", colorCyan, host, port, colorReset)

	httpServer := &http.Server{
		Addr:     addr,
		Handler:  server,
		ErrorLog: log.New(os.Stderr, "", log.LstdFlags),
	}

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logError("Server error: %v", err)
			os.Exit(1)
		}
	}()

	go func() {
		loadZimFiles(server, paths)
		printLoadedArchives(server, host, port)
		go watchFiles(server, paths)
	}()

	select {}
}

func waitForStableFiles(zimFiles []string) []string {
	var wg sync.WaitGroup
	stableChan := make(chan string, len(zimFiles))

	for _, file := range zimFiles {
		wg.Add(1)

		go func(file string) {
			defer wg.Done()

			baseName := filepath.Base(file)

			initialSize := int64(-1)
			initialModTime := time.Time{}
			attempts := 0

			for attempts < 5 {
				info, err := os.Stat(file)
				if err != nil {
					logWarning("Cannot stat %s%s%s: %v", colorCyan, baseName, colorReset, err)
					return
				}

				currentSize := info.Size()
				currentModTime := info.ModTime()

				if initialSize == -1 {
					initialSize = currentSize
					initialModTime = currentModTime
					attempts++
					time.Sleep(500 * time.Millisecond)
					continue
				}

				if currentSize == initialSize && currentModTime.Equal(initialModTime) {
					stableChan <- file
					return
				}

				attempts++
				time.Sleep(500 * time.Millisecond)
			}

			if attempts >= 5 {
				logWarning("%s%s%s failed to stabilize after %d checks. Skipping load.", colorCyan, baseName, colorReset, attempts)
			}
		}(file)
	}

	wg.Wait()
	close(stableChan)

	stableFiles := make([]string, 0, len(zimFiles))
	for file := range stableChan {
		stableFiles = append(stableFiles, file)
	}

	return stableFiles
}

func loadZimFiles(server *web.Server, paths []string) {
	rawFiles := collectZimFiles(paths)

	if len(rawFiles) == 0 {
		logError("No ZIM files found")
		return
	}

	zimFiles := waitForStableFiles(rawFiles)

	if len(zimFiles) == 0 {
		logError("No stable ZIM files found to load")
		return
	}

	var wg sync.WaitGroup

	for _, zimFile := range zimFiles {
		wg.Add(1)

		go func(file string) {
			defer wg.Done()

			baseName := filepath.Base(file)

			if err := server.LoadZIM(file); err != nil {
				log.Printf("%s✗%s %sFailed to load %s%s%s: %s%v%s",
					colorRed, colorReset,
					colorRed,
					colorCyan, baseName, colorReset,
					colorRed, err, colorReset,
				)
			} else {
				logSuccess("Loaded ZIM: %s%s%s", colorCyan, baseName, colorReset)
			}
		}(zimFile)
	}

	wg.Wait()
}

func collectZimFiles(paths []string) []string {
	zimFiles := make([]string, 0)
	seen := make(map[string]bool)

	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			logWarning("Cannot access %s%s%s: %v", colorCyan, path, colorReset, err)
			continue
		}

		if info.IsDir() {
			entries, err := os.ReadDir(path)
			if err != nil {
				logWarning("Cannot read directory %s%s%s: %v", colorCyan, path, colorReset, err)
				continue
			}

			for _, entry := range entries {
				if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".zim") {
					fullPath := filepath.Join(path, entry.Name())
					if !seen[fullPath] {
						zimFiles = append(zimFiles, fullPath)
						seen[fullPath] = true
					}
				}
			}
		} else if strings.HasSuffix(path, ".zim") {
			if !seen[path] {
				zimFiles = append(zimFiles, path)
				seen[path] = true
			}
		}
	}

	return zimFiles
}

type fileState struct {
	size    int64
	modTime time.Time
	stable  bool
}

func watchFiles(server *web.Server, paths []string) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	fileStates := make(map[string]*fileState)
	loadedFiles := make(map[string]bool)

	initialFiles := collectZimFiles(paths)
	for _, f := range initialFiles {
		if info, err := os.Stat(f); err == nil {
			fileStates[f] = &fileState{
				size:    info.Size(),
				modTime: info.ModTime(),
				stable:  true,
			}
		}
	}

	for range ticker.C {
		currentFiles := collectZimFiles(paths)
		currentMap := make(map[string]bool)

		for _, f := range currentFiles {
			currentMap[f] = true
			info, err := os.Stat(f)
			if err != nil {
				continue
			}

			state, exists := fileStates[f]
			if !exists {
				fileStates[f] = &fileState{
					size:    info.Size(),
					modTime: info.ModTime(),
					stable:  false,
				}
				logInfo("New file detected: %s%s%s", colorCyan, filepath.Base(f), colorReset)
				continue
			}

			if !state.stable {
				if state.size == info.Size() && state.modTime.Equal(info.ModTime()) {
					state.stable = true
					if err := server.LoadZIM(f); err != nil {
						logWarning("Failed to load %s%s%s: %v", colorCyan, filepath.Base(f), colorReset, err)
					} else {
						loadedFiles[f] = true
						logSuccess("Loaded ZIM: %s%s%s", colorCyan, filepath.Base(f), colorReset)
					}
				} else {
					state.size = info.Size()
					state.modTime = info.ModTime()
				}
			}
		}

		for f := range loadedFiles {
			if !currentMap[f] {
				logWarning("File removed: %s%s%s", colorCyan, filepath.Base(f), colorReset)

				if loadedFiles[f] {
					baseName := filepath.Base(f)
					name := strings.TrimSuffix(baseName, filepath.Ext(baseName))

					if err := server.UnloadZIM(name); err != nil {
						logWarning("Failed to unload %s%s%s: %v", colorCyan, name, colorReset, err)
					}
					delete(loadedFiles, f)
				}
				delete(fileStates, f)
			}
		}

		for f := range fileStates {
			if !currentMap[f] {
				delete(fileStates, f)
			}
		}
	}
}

func printLoadedArchives(server *web.Server, host, port string) {
	archives := server.ListArchives()

	count := len(archives)

	archiveWord := "archive"
	if count > 1 {
		archiveWord = "archives"
	}

	if count > 0 {
		logInfo("Loaded %d %s:", count, archiveWord)
		for _, archive := range archives {
			log.Printf("  %s-%s %s: %shttp://%s:%s/viewer/%s/%s", colorGreen, colorReset, archive.Metadata.Title, colorCyan, host, port, archive.Name, colorReset)
		}
	} else if count == 0 {
		logInfo("No archives loaded.")
	}
}

func logSuccess(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	log.Printf("%s✓%s %s", colorGreen, colorReset, msg)
}

func logInfo(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	log.Printf("%sℹ%s %s", colorBlue, colorReset, msg)
}

func logWarning(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	log.Printf("%s⚠%s %s", colorYellow, colorReset, msg)
}

func logError(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	log.Printf("%s✗%s %s", colorRed, colorReset, msg)
}
