// +build metrics

package beaconpi

import (
	"bytes"
	"fmt"
	log "github.com/sirupsen/logrus"
	"gopkg.in/gomail.v2"
	"time"
)

const (
	MAX_MONITOR_MSG   = 4096
	TIMEOUT_EDGE_SYNC = time.Second * 10
	TIMEOUT_SEND      = time.Second * 600
	EDGE_SYNC_MAX     = 10.0
)

type monitor struct {
	msgqueue chan string
}

var m monitor

func startMonitor() {
	m.msgqueue = make(chan string, 1024)
}

func sendInfo(msg string) {
	if len(msg) > MAX_MONITOR_MSG {
		log.Infof("Length of msg was %d which was too large %d is the max msg: %s", len(msg), MAX_MONITOR_MSG, msg)
		return
	}
	m.msgqueue <- msg
}

func sendWarning(msg string) {
	if len(msg) > MAX_MONITOR_MSG {
		log.Infof("Length of msg was %d which was too large %d is the max msg: %s", len(msg), MAX_MONITOR_MSG, msg)
		return
	}
	m.msgqueue <- msg
}

func sendQueue() {
	log.Info("Sending Message Queue")
	//log.Debugf("mp: %v", mp)
	d := gomail.NewDialer(mp.SMTPHost, mp.SMTPPort, mp.SMTPUser, mp.SMTPPassphrase)
	msg := gomail.NewMessage()
	msg.SetHeader("From", mp.SMTPUser)
	msg.SetHeader("To", mp.MonitorEmail)
	msg.SetHeader("Subject", "Beaconpi Monitoring Service "+time.Now().Format(time.RFC3339))
	var buff bytes.Buffer
	// Drain until empty
	buff.WriteString("Messages: <br><ol>")
	count := 0

drain:
	for {
		select {
		case t := <-m.msgqueue:
			count += 1
			buff.WriteString(fmt.Sprintf("<li>%s</li>\n", t))
		default:
			break drain
		}
	}
	buff.WriteString("</ol>")

	msg.SetBody("text/html", buff.String())
	if count > 0 {
		log.Info("Sending message")
		err := d.DialAndSend(msg)
		log.Info("Sent message")
		if err != nil {
			log.Error("Failed to send message with", err)
		}
	}
}

func intSliceEqual(l []int, r []int) bool {
	if len(l) != len(r) {
		return false
	}
	for i, _ := range l {
		if l[i] != r[i] {
			return false
		}
	}
	return true
}

func metricsBackgroundTasks() {
	startMonitor()
	tickES := time.Tick(TIMEOUT_EDGE_SYNC)
	tickSend := time.Tick(TIMEOUT_SEND)
	var lastid int
	var msg []string

	sendInfo("Server Started")
	sendQueue()

	inactEdge, err := changedActiveEdges()
	if err != nil {
		log.Warnf("getInactiveEdges failed to fetch %s", err)
	}

	for {
		select {
		case _ = <-tickES:

			dbconfig := dbHandler{mp.DriverName, mp.DataSourceName}
			db, err := dbconfig.openDB()
			if err == nil {
				msg, lastid, err = dbGetErrorsSince(lastid, db)
				if err != nil {
					log.Warnf("Error getting db errors %s", err)
				}
				for _, v := range msg {
					sendWarning(v)
				}
			} else {
				log.Warnf("Failed to open DB %s", err)
			}

			// check inactive edges
			newedges, err := changedActiveEdges()
			if err != nil {
				log.Warnf("getInactiveEdges failed to fetch %s", err)
			} else {
				if !intSliceEqual(newedges, inactEdge) {
					sendWarning(fmt.Sprintf("Inactive edges changed from %#v to %#v at %s", inactEdge, newedges, time.Now().Format(time.RFC3339)))
					inactEdge = newedges
				}
			}

		case _ = <-tickSend:
			sendQueue()
		}
	}
}
