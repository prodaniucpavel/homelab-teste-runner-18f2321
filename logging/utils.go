package logging

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/fatih/color"
	"log"
	"neptune_runner/config"
)

func IsDebugEnabled() bool {
	return config.LOG_ENABLED
}

func IsLogToFileEnabled() bool {
	return config.LOG_TO_FILE
}

func Success(format string, args ...interface{}) {
	color.Green("[+] "+format+"\n", args...)
}

func Info(format string, args ...interface{}) {
	color.Cyan("[*] "+format+"\n", args...)
}

func Err(format string, args ...interface{}) {
	color.Red("[-] "+format+"\n", args...)
}

func DebugInfo(format string, args ...interface{}) {
	if !IsDebugEnabled() {
		return
	}

	debug := color.YellowString("[DEBUG]")
	content := color.CyanString("[*] "+format, args...)

	fmt.Println(debug + content)

	if IsLogToFileEnabled() {
		log.Printf(format+"\n", args...)
	}
}

func DebugErr(format string, args ...interface{}) {
	if !IsDebugEnabled() {
		return
	}

	debug := color.YellowString("[DEBUG]")
	content := color.RedString("[-] "+format, args...)

	fmt.Println(debug + content)

	if IsLogToFileEnabled() {
		log.Printf(format+"\n", args...)
	}
}

func DebugStruct(complexStruct interface{}) {
	if !IsDebugEnabled() {
		return
	}

	color.Yellow("[DEBUG][STRUCT START]")
	spew.Dump(complexStruct)
	color.Yellow("[DEBUG][STRUCT END]")
}
