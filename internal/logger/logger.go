package logger

import (
	"fmt"
	"os"
	"time"
)

type Logger struct {
	channel chan string
	done    chan bool
}

func NewLogger() *Logger {
	return &Logger{
		channel: make(chan string, 100), // Буф. канал
		done:    make(chan bool),
	}
}

func (l *Logger) Start() {
	go func() {
		for {
			select {
			case message := <-l.channel:
				delay := time.Duration(1 + int(time.Now().UnixNano()%2)) * time.Second
				time.Sleep(delay)
				
				if err := l.writeToFile(message); err != nil {
					fmt.Printf("Error writing to log file: %v\n", err)
				}
				
				fmt.Printf("[LOG] %s\n", message)
				
			case <-l.done:
				fmt.Println("Logger stopped gracefully")
				return
			}
		}
	}()
}

func (l *Logger) writeToFile(message string) error {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("[%s] %s\n", timestamp, message)
	
	// Открываем\создаем файл для дозаписи
	f, err := os.OpenFile("logs/log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	
	_, err = f.WriteString(logEntry)
	return err
}

// сообщение в канал
func (l *Logger) Log(message string) {
	l.channel <- message
}

func (l *Logger) Stop() {
	l.done <- true
}