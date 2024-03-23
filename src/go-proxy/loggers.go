package main

import "github.com/sirupsen/logrus"

var palog = logrus.WithField("component", "panel")
var prlog = logrus.WithField("component", "provider")
var cfgl = logrus.WithField("component", "config")
var hrlog = logrus.WithField("component", "http_proxy")
var srlog = logrus.WithField("component", "stream")
var wlog = logrus.WithField("component", "watcher")
var aclog = logrus.WithField("component", "autocert")