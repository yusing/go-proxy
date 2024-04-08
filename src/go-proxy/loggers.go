package main

import "github.com/sirupsen/logrus"

var palog = logrus.WithField("?", "panel")
var cfgl = logrus.WithField("?", "config")
var hrlog = logrus.WithField("?", "http")
var srlog = logrus.WithField("?", "stream")
var wlog = logrus.WithField("?", "watcher")
var aclog = logrus.WithField("?", "autocert")