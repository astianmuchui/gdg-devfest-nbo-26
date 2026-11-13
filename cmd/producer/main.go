package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"
)

type environment struct {
	TempC         float64 `json:"temperature_c"`
	HumidityPct   float64 `json:"humidity_pct"`
	AirQualityPPM float64 `json:"air_quality_ppm"`
	AirQualityV   float64 `json:"air_quality_v"`
}

type reading struct {
	DeviceID    string      `json:"device_id"`
	Timestamp   int64       `json:"timestamp"`
	UptimeS     int64       `json:"uptime_s"`
	Environment environment `json:"environment"`
}

type device struct {
	id     string
	phase  float64
	booted time.Time
}

func (d *device) read(now time.Time) reading {
	t := float64(now.UnixMilli())/60000.0 + d.phase
	ppm := 600 + 250*math.Sin(t/3) + rand.Float64()*40
	return reading{
		DeviceID:  d.id,
		Timestamp: now.UnixMilli(),
		UptimeS:   int64(now.Sub(d.booted).Seconds()),
		Environment: environment{
			TempC:         round(24+4*math.Sin(t)+rand.Float64()*0.5, 2),
			HumidityPct:   round(55+12*math.Cos(t/2)+rand.Float64(), 2),
			AirQualityPPM: round(ppm, 1),
			AirQualityV:   round(ppm/1000*1.65, 3),
		},
	}
}

func round(v float64, places int) float64 {
	p := math.Pow10(places)
	return math.Round(v*p) / p
}

func main() {
	broker := flag.String("broker", "mqtt://localhost:1883", "MQTT broker URL")
	devices := flag.Int("devices", 1000, "number of simulated devices")
	interval := flag.Duration("interval", 2*time.Second, "time between readings per device")
	conns := flag.Int("conns", 10, "MQTT connections to spread devices across")
	qos := flag.Int("qos", 1, "MQTT QoS (0 or 1)")
	prefix := flag.String("prefix", "devices", "topic prefix, publishes to <prefix>/<device_id>/readings")
	duration := flag.Duration("duration", 0, "stop after this long (0 runs until Ctrl+C)")
	flag.Parse()

	if *conns < 1 {
		*conns = 1
	}
	if *conns > *devices {
		*conns = *devices
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if *duration > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *duration)
		defer cancel()
	}

	groups := make([][]*device, *conns)
	now := time.Now()
	for i := 0; i < *devices; i++ {
		d := &device{
			id:     fmt.Sprintf("DEV%05d", i),
			phase:  rand.Float64() * 2 * math.Pi,
			booted: now.Add(-time.Duration(rand.IntN(86400)) * time.Second),
		}
		groups[i%*conns] = append(groups[i%*conns], d)
	}

	var published, failed atomic.Uint64
	var wg sync.WaitGroup

	for c, group := range groups {
		client := connect(*broker, fmt.Sprintf("producer-%d-%d", os.Getpid(), c))
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer client.Disconnect(250)
			run(ctx, client, group, *interval, byte(*qos), *prefix, &published, &failed)
		}()
	}

	rate := float64(*devices) / interval.Seconds()
	log.Printf("producing: %d devices, every %s, %d conns, QoS %d, target %.0f msg/s", *devices, *interval, *conns, *qos, rate)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	var last uint64
	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			log.Printf("done: published=%d failed=%d", published.Load(), failed.Load())
			return
		case <-ticker.C:
			cur := published.Load()
			log.Printf("published %6d msg/s  total=%d failed=%d", cur-last, cur, failed.Load())
			last = cur
		}
	}
}

func connect(broker, clientID string) pahomqtt.Client {
	opts := pahomqtt.NewClientOptions().
		AddBroker(broker).
		SetClientID(clientID).
		SetCleanSession(true).
		SetAutoReconnect(true).
		SetConnectRetry(true).
		SetConnectTimeout(5 * time.Second)

	client := pahomqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("connect %s: %v", clientID, token.Error())
	}
	return client
}

func run(ctx context.Context, client pahomqtt.Client, group []*device, interval time.Duration, qos byte, prefix string, published, failed *atomic.Uint64) {
	step := interval / time.Duration(len(group))
	start := time.Now().Add(time.Duration(rand.Int64N(int64(step) + 1)))

	for n := 0; ; n++ {
		next := start.Add(time.Duration(n) * step)
		if wait := time.Until(next); wait > 0 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(wait):
			}
		} else if ctx.Err() != nil {
			return
		}

		d := group[n%len(group)]
		payload, err := json.Marshal(d.read(time.Now()))
		if err != nil {
			failed.Add(1)
			continue
		}

		token := client.Publish(prefix+"/"+d.id+"/readings", qos, false, payload)
		if qos > 0 {
			token.Wait()
			if token.Error() != nil {
				failed.Add(1)
				continue
			}
		}
		published.Add(1)
	}
}
