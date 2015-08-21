package main

import (
	rawlog "log"
)

// @author Robin Verlangen

type Log struct {

}

func (l *Log) Printf(format string, v ...interface{}) {
	rawlog.Printf(format, v...)
}

func (l *Log) Println(x string) {
	rawlog.Println(x)
}

func (l *Log) Fatal(v ...interface{}) {
	rawlog.Fatal(v...)
}

func newLog() *Log {
	return &Log{}
}