package dev

import (
	"fmt"
	//"io"
	"math/rand"
	//"os"
	//"path/filepath"
	//"regexp"
	"time"

	"github.com/udhos/jazigo/conf"
	//"github.com/udhos/jazigo/store"
)

func Spawner(tab DeviceUpdater, logger hasPrintf, reqChan chan FetchRequest, repository string, options *conf.Options) {

	logger.Printf("Spawner: starting")

	for {
		req, ok := <-reqChan
		if !ok {
			logger.Printf("Spawner: request channel closed")
			break
		}

		replyChan := req.ReplyChan // alias

		devId := req.Id
		d, getErr := tab.GetDevice(devId)
		if getErr != nil {
			if replyChan != nil {
				replyChan <- FetchResult{DevId: devId, Msg: fmt.Sprintf("Spawner: could not find device: %v", getErr), Code: FETCH_ERR_GETDEV, Begin: time.Now()}
			}
			continue
		}

		opt := options.Get()                                             // get current global data
		go d.Fetch(logger, replyChan, 0, repository, opt.MaxConfigFiles) // spawn per-request goroutine
	}

	logger.Printf("Spawner: exiting")
}

func Scan(tab DeviceUpdater, devices []*Device, logger hasPrintf, opt *conf.AppConfig, reqChan chan FetchRequest) (int, int, int) {

	begin := time.Now()
	deviceCount := len(devices)
	wait := 0       // requests pending
	nextDevice := 0 // device iterator
	req := FetchRequest{ReplyChan: make(chan FetchResult)}
	maxConcurrency := opt.MaxConcurrency // alias
	holdtime := opt.Holdtime             // alias
	elapMax := 0 * time.Second
	elapMin := 24 * time.Hour
	success := 0
	skipped := 0
	deleted := 0

	for nextDevice < deviceCount || wait > 0 {
		// launch requests
		for ; nextDevice < deviceCount; nextDevice++ {
			if maxConcurrency > 0 && wait >= maxConcurrency {
				break // max concurrent limit reached
			}

			d := devices[nextDevice]

			if d.Deleted {
				deleted++
				continue
			}

			if h := d.Holdtime(time.Now(), holdtime); h > 0 {
				// do not handle device yet (holdtime not expired)
				logger.Printf("Scan: %s skipping due to holdtime=%s", d.Id, h)
				skipped++
				continue
			}

			req.Id = d.Id
			reqChan <- req

			wait++ // launched
			logger.Printf("Scan: launched: %s count=%d/%d wait=%d max=%d", req.Id, nextDevice, deviceCount, wait, maxConcurrency)
		}

		if wait < 1 {
			continue
		}

		// wait one response
		r := <-req.ReplyChan
		wait-- // received

		end := time.Now()
		elap := end.Sub(r.Begin)
		logger.Printf("Scan: recv %s %s %s %s msg=[%s] code=%d wait=%d remain=%d skipped=%d elap=%s", r.Model, r.DevId, r.DevHostPort, r.Transport, r.Msg, r.Code, wait, deviceCount-nextDevice, skipped, elap)

		good := r.Code == FETCH_ERR_NONE
		updateDeviceStatus(tab, r.DevId, good, end, logger, holdtime)

		if good {
			success++
		}
		if elap < elapMin {
			elapMin = elap
		}
		if elap > elapMax {
			elapMax = elap
		}
	}

	elapsed := time.Since(begin)
	average := elapsed / time.Duration(deviceCount)

	logger.Printf("Scan: finished elapsed=%s devices=%d success=%d skipped=%d deleted=%d average=%s min=%s max=%s", elapsed, deviceCount, success, skipped, deleted, average, elapMin, elapMax)

	return success, deviceCount - success, skipped + deleted
}

func ScanDevices(tab DeviceUpdater, devices []*Device, logger hasPrintf, maxConcurrency int, delayMin, delayMax time.Duration, repository string, maxFiles int, holdtime time.Duration) (int, int, int) {

	deviceCount := len(devices)

	logger.Printf("ScanDevices: starting devices=%d maxConcurrency=%d", deviceCount, maxConcurrency)
	if deviceCount < 1 {
		logger.Printf("ScanDevices: aborting")
		return 0, 0, 0
	}

	begin := time.Now()
	random := rand.New(rand.NewSource(begin.UnixNano()))

	resultCh := make(chan FetchResult)

	logger.Printf("ScanDevices: per-device delay before starting: %d-%d ms", delayMin/time.Millisecond, delayMax/time.Millisecond)

	wait := 0
	nextDevice := 0
	elapMax := 0 * time.Second
	elapMin := 24 * time.Hour
	success := 0
	skipped := 0
	deleted := 0

	for nextDevice < deviceCount || wait > 0 {

		// launch additional devices
		for ; nextDevice < deviceCount; nextDevice++ {
			// there are devices to process

			if maxConcurrency > 0 && wait >= maxConcurrency {
				break // max concurrent limit reached
			}

			d := devices[nextDevice]

			if d.Deleted {
				deleted++
				continue
			}

			if h := d.Holdtime(time.Now(), holdtime); h > 0 {
				// do not handle device yet (holdtime not expired)
				logger.Printf("device: %s skipping due to holdtime=%s", d.Id, h)
				skipped++
				continue
			}

			// launch one additional per-device goroutine

			r := random.Float64()
			var delay time.Duration
			if delayMax > 0 {
				delay = time.Duration(round(r*float64(delayMax-delayMin))) + delayMin
			}
			go d.Fetch(logger, resultCh, delay, repository, maxFiles) // per-device goroutine
			wait++
		}

		if wait < 1 {
			continue
		}

		// wait for one device to finish
		//logger.Printf("device wait: devices=%d wait=%d remain=%d skipped=%d", deviceCount, wait, deviceCount-nextDevice, skipped)
		r := <-resultCh
		wait--
		end := time.Now()
		elap := end.Sub(r.Begin)
		logger.Printf("device result: %s %s %s %s msg=[%s] code=%d wait=%d remain=%d skipped=%d elap=%s", r.Model, r.DevId, r.DevHostPort, r.Transport, r.Msg, r.Code, wait, deviceCount-nextDevice, skipped, elap)

		good := r.Code == FETCH_ERR_NONE
		updateDeviceStatus(tab, r.DevId, good, end, logger, holdtime)

		if good {
			success++
		}
		if elap < elapMin {
			elapMin = elap
		}
		if elap > elapMax {
			elapMax = elap
		}
	}

	elapsed := time.Since(begin)
	average := elapsed / time.Duration(deviceCount)

	logger.Printf("ScanDevices: finished elapsed=%s devices=%d success=%d skipped=%d deleted=%d average=%s min=%s max=%s", elapsed, deviceCount, success, skipped, deleted, average, elapMin, elapMax)

	return success, deviceCount - success, skipped + deleted
}

func updateDeviceStatus(tab DeviceUpdater, devId string, good bool, last time.Time, logger hasPrintf, holdtime time.Duration) {
	d, getErr := tab.GetDevice(devId)
	if getErr != nil {
		logger.Printf("updateDeviceStatus: '%s' not found: %v", getErr)
		return
	}

	now := time.Now()
	h1 := d.Holdtime(now, holdtime)

	d.lastTry = last
	d.lastStatus = good
	if d.lastStatus {
		d.lastSuccess = d.lastTry
	}

	tab.UpdateDevice(d)

	h2 := d.Holdtime(now, holdtime)
	logger.Printf("updateDeviceStatus: device %s holdtime: old=%v new=%v", devId, h1, h2)
}
