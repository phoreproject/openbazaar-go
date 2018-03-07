package core

import (
	"fmt"
	"time"

	"github.com/OpenBazaar/openbazaar-go/repo"
	"github.com/op/go-logging"
)

type disputeNotifier struct {
	disputeCasesDB  repo.CaseStore
	notificationsDB repo.NotificationStore

	intervalDelay time.Duration
	logger        *logging.Logger
	runCount      int
	watchdogTimer *time.Ticker
	stopWorker    chan bool
}

func (n *OpenBazaarNode) StartDisputeNotifier() {
	n.DisputeNotifier = &disputeNotifier{
		disputeCasesDB:  n.Datastore.Cases(),
		notificationsDB: n.Datastore.Notifications(),
		intervalDelay:   time.Duration(10) * time.Minute,
		logger:          logging.MustGetLogger("disputeNotifier"),
	}
	go n.DisputeNotifier.Run()
}

func (d *disputeNotifier) RunCount() int { return d.runCount }

func (d *disputeNotifier) Run() {
	d.watchdogTimer = time.NewTicker(d.intervalDelay)
	d.stopWorker = make(chan bool)

	// Run once on start, then wait for watchdog
	if err := d.PerformTask(); err != nil {
		d.logger.Error("performTask failure:", err.Error())
	}
	for {
		select {
		case <-d.watchdogTimer.C:
			if err := d.PerformTask(); err != nil {
				d.logger.Error("performTask failure:", err.Error())
			}
		case <-d.stopWorker:
			d.watchdogTimer.Stop()
			return
		}
	}
}

func (d *disputeNotifier) Stop() {
	d.stopWorker <- true
	close(d.stopWorker)
}

func (d *disputeNotifier) PerformTask() error {
	d.runCount += 1
	d.logger.Infof("performTask started (count %d)", d.runCount)
	disputes, err := d.disputeCasesDB.GetDisputesForNotification()
	if err != nil {
		return err
	}

	var (
		executedAt         = time.Now()
		notificationsToAdd = make([]*repo.NotificationRecord, 0)

		fifteenDays    = time.Duration(15*24) * time.Hour
		thirtyDays     = time.Duration(30*24) * time.Hour
		fourtyFourDays = time.Duration(44*24) * time.Hour
		fourtyFiveDays = time.Duration(45*24) * time.Hour
	)
	for _, d := range disputes {
		var timeSinceCreation = executedAt.Sub(d.Timestamp)
		if d.LastNotifiedAt.Before(d.Timestamp) || d.LastNotifiedAt.Equal(d.Timestamp) {
			notificationsToAdd = append(notificationsToAdd, d.BuildZeroDayNotification(executedAt))
		}
		if d.LastNotifiedAt.Before(d.Timestamp.Add(fifteenDays)) && timeSinceCreation > fifteenDays {
			notificationsToAdd = append(notificationsToAdd, d.BuildFifteenDayNotification(executedAt))
		}
		if d.LastNotifiedAt.Before(d.Timestamp.Add(thirtyDays)) && timeSinceCreation > thirtyDays {
			notificationsToAdd = append(notificationsToAdd, d.BuildThirtyDayNotification(executedAt))
		}
		if d.LastNotifiedAt.Before(d.Timestamp.Add(fourtyFourDays)) && timeSinceCreation > fourtyFourDays {
			notificationsToAdd = append(notificationsToAdd, d.BuildFourtyFourDayNotification(executedAt))
		}
		if d.LastNotifiedAt.Before(d.Timestamp.Add(fourtyFiveDays)) && timeSinceCreation > fourtyFiveDays {
			notificationsToAdd = append(notificationsToAdd, d.BuildFourtyFiveDayNotification(executedAt))
		}
		if len(notificationsToAdd) > 0 {
			d.LastNotifiedAt = executedAt
		}
	}

	notificationTx, err := d.notificationsDB.BeginTransaction()
	if err != nil {
		return err
	}

	for _, n := range notificationsToAdd {
		var serializedNotification, err = n.MarshalNotificationToJSON()
		if err != nil {
			d.logger.Warning("marshaling notification:", err.Error())
			d.logger.Infof("failed marshal: %+v", n)
			continue
		}
		var template = "insert into notifications(notifID, serializedNotification, type, timestamp, read) values(?,?,?,?,?)"
		_, err = notificationTx.Exec(template, n.GetID(), serializedNotification, n.GetDowncaseType(), n.GetSQLTimestamp(), 0)
		if err != nil {
			d.logger.Warning("inserting notification:", err.Error())
			d.logger.Infof("failed insert: %+v", n)
			continue
		}
	}

	if err = notificationTx.Commit(); err != nil {
		if rollbackErr := notificationTx.Rollback(); rollbackErr != nil {
			err = fmt.Errorf(err.Error(), "\nand also failed during rollback:", rollbackErr.Error())
		}
		return fmt.Errorf("commiting notifications:", err.Error())
	}
	d.logger.Infof("created %d dispute notifications", len(notificationsToAdd))

	err = d.disputeCasesDB.UpdateDisputesLastNotifiedAt(disputes)
	d.logger.Infof("updated lastNotifiedAt on %d disputes", len(disputes))
	return nil
}
