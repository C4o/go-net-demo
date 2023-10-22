package logger

import (
	"log"
	"os"
	"sync/atomic"

	"github.com/fatih/color"
)

const (
	INFO = iota
	DEBUG
	ERROR
	FATAL
	VERBOSE
)

var (
	file   *os.File
	red    = color.New(color.FgRed).SprintFunc()
	yellow = color.New(color.FgYellow).SprintFunc()
	green  = color.New(color.FgGreen).SprintFunc()
	blue   = color.New(color.BgBlue).SprintFunc()

	Debug   atomic.Bool
	Verbose bool
)

func Init(save bool) (*os.File, error) {
	Debug.Store(false)

	// help to print code location
	//log.SetFlags(log.LstdFlags | log.Lshortfile)

	if !save {
		return nil, nil
	}

	var err error
	file, err = os.OpenFile("a.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Printf("failed to open file: %v", err)
	} else {
		log.SetOutput(file)
	}

	return file, err
}

func Logf(level int, format string, args ...interface{}) {
	switch level {
	case INFO:
		log.Printf(green("[INFO] "+format), args...)
	case DEBUG:
		if !Debug.Load() {
			return
		}
		log.Printf(yellow("[DEBUG] "+format), args...)
	case ERROR:
		log.Printf(red("[ERROR] "+format), args...)
	case VERBOSE:
		if !Verbose {
			return
		}
		log.Printf(blue("[DEEP-DEBUG] "+format), args...)
	case FATAL:
		log.Panicf(format, args...)
	}
}

func SwitchDebug(debug bool) {
	Debug.Swap(debug)
}
