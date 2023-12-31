package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type cfg struct {
	fileSizes map[string]int64

	fileNamePrefix string
}

func main() {
	start := os.Getenv("SYNC_FROM")
	end := os.Getenv("SYNC_TO")
	prefixNewFileNameWith := os.Getenv("PREFIX_WITH")

	if start == "" || end == "" {
		PrintlnAndExit("ENV SYNC_FROM and SYNC_TO needs to be specified", 1)
	}
	cfg := &cfg{fileSizes: map[string]int64{}, fileNamePrefix: prefixNewFileNameWith}
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	ticker := time.NewTicker(2 * time.Second)

	go func() {
		for {
			select {
			case t := <-ticker.C:
				fmt.Printf("Sycing folders at %s\n", t.String())
				err := syncFiles(start, end, cfg)
				fmt.Printf("File Sizes map:%v", cfg.fileSizes)
				if err != nil {
					PrintlnAndExit(err.Error(), 1)
				}
			}
		}
	}()
	<-c
	ticker.Stop()
	PrintlnAndExit("over and out", 0)
}

func PrintlnAndExit(msg string, exitCode int) {
	fmt.Println(msg)
	os.Exit(exitCode)
}

func syncFiles(fromPath, toPath string, cfg *cfg) error {
	files, err := os.ReadDir(fromPath)
	if err != nil {
		return err
	}

	for _, file := range files {
		fromFilePath := filepath.Join(fromPath, file.Name())

		if file.IsDir() {
			fmt.Printf("%s is a dir, ignore\n", file.Name())
			continue
		} else {
			if strings.HasPrefix(file.Name(), ".") {
				continue
			}
			fInfo, err := os.Stat(fromFilePath)
			fSize := fInfo.Size()
			if err != nil {
				return err
			}
			if fInfo.ModTime().Before(time.Now().Add(-5 * time.Second)) {
				if f, ok := cfg.fileSizes[fromFilePath]; ok {
					if f == fSize {
						fmt.Printf("%s is a file, copy, because last modified is %s\n", file.Name(), fInfo.ModTime())
						nF := fmt.Sprintf("%s_%s", cfg.fileNamePrefix, strings.ToLower(file.Name()))
						toFilePath := filepath.Join(toPath, nF)

						err = copyFile(fromFilePath, toFilePath)
						if err != nil {
							return err
						}
						fmt.Printf("%s is a file, copy successfull, delete old\n", file.Name())
						err = os.Remove(fromFilePath)
						if err != nil {
							return err
						}
						fmt.Printf("%s is a file, delete old sucessfull\n", file.Name())
						delete(cfg.fileSizes, fromFilePath)
					} else {
						fmt.Printf("%s size does not equal: stored %d; from run %d\n", file.Name(), f, fSize)
					}
				} else {
					cfg.fileSizes[fromFilePath] = fSize
					fmt.Printf("%s is not known in file size metric, wait for next run\n", file.Name())
				}
			} else {
				fmt.Printf("%s is a file, ignore, because last modified is to recent %s\n", file.Name(), fInfo.ModTime())
			}
		}
	}

	return nil
}

func copyFile(src, dest string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	err = os.WriteFile(dest, input, 0644)
	if err != nil {
		return err
	}

	return nil
}
