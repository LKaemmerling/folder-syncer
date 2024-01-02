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

func main() {
	start := os.Getenv("SYNC_FROM")
	end := os.Getenv("SYNC_TO")

	if start == "" || end == "" {
		PrintlnAndExit("ENV SYNC_FROM and SYNC_TO needs to be specified", 1)
	}
	var fileSizes = map[string]int64{}
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	ticker := time.NewTicker(2 * time.Second)

	go func() {
		for {
			select {
			case t := <-ticker.C:
				fmt.Printf("Sycing folders at %s\n", t.String())
				err := syncFiles(start, end, fileSizes)
				fmt.Printf("File Sizes map:%v", fileSizes)
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

func syncFiles(fromPath, toPath string, fileSizes map[string]int64) error {
	files, err := os.ReadDir(fromPath)
	if err != nil {
		return err
	}

	for _, file := range files {
		fromFilePath := filepath.Join(fromPath, file.Name())
		toFilePath := filepath.Join(toPath, file.Name())
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
				if f, ok := fileSizes[fromFilePath]; ok {
					if f == fSize {
						fmt.Printf("%s is a file, copy, because last modified is %s\n", file.Name(), fInfo.ModTime())
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
						delete(fileSizes, fromFilePath)
					} else {
						fmt.Printf("%s size does not equal: stored %d; from run %d\n", file.Name(), f, fSize)
					}
				} else {
					fmt.Printf("%s is not known in file size metric, wait for next run\n", file.Name())
				}
			} else {
				fmt.Printf("%s is a file, ignore, because last modified is to recent %s\n", file.Name(), fInfo.ModTime())
				fileSizes[fromFilePath] = fSize

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
