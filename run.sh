#!/usr/bin/env bash
set -euo pipefail

LABEL=${1:?usage: ./run.sh <label> <seconds> [extra env, e.g. "GOGC=400 GOMEMLIMIT=1GiB"]}
SECS=${2:-60}
EXTRA=${3:-}

DEVICES=${DEVICES:-50000}
INTERVAL=${INTERVAL:-5s}
CONNS=${CONNS:-20}

cd "$(dirname "$0")"
mkdir -p results bin

go build -o bin/ingester ./cmd
go build -o bin/producer ./cmd/producer

LOG=results/$LABEL.log

env IOT_PORT=8090 \
	IOT_MQTT_CLIENT_ID=bench \
	IOT_SHARE_GROUP=bench \
	IOT_MQTT_TOPIC='bench/+/readings' \
	IOT_STALL_FOR=0s \
	$EXTRA \
	./bin/ingester >"$LOG" 2>&1 &
ING=$!
trap 'kill $ING 2>/dev/null || true' EXIT

sleep 3
echo "run=$LABEL secs=$SECS devices=$DEVICES interval=$INTERVAL extra=[$EXTRA]"
./bin/producer -prefix bench -devices "$DEVICES" -interval "$INTERVAL" -conns "$CONNS" \
	-duration "${SECS}s" >"results/$LABEL.producer.log" 2>&1
sleep 3

kill $ING 2>/dev/null || true
wait $ING 2>/dev/null || true
trap - EXIT

awk '
/^t=/ {
	line = $0
	gsub(/= +/, "=", line)
	n = split(line, f, " +")
	delete kv
	for (i = 1; i <= n; i++) {
		split(f[i], p, "=")
		v = p[2]
		gsub(/[^0-9.]/, "", v)
		kv[p[1]] = v + 0
	}
	if (kv["stored/s"] == 0) next
	rows++
	sr[rows] = kv["stored/s"]; cpu[rows] = kv["cpu_ms/1k"]; al[rows] = kv["allocs/msg"]
	mb[rows] = kv["alloc_MB/s"]; gc[rows] = kv["gc/s"]; gcc[rows] = kv["gc_cpu"]
	if (kv["sys"] > sys) sys = kv["sys"]
	dropped = kv["dropped"]; failed = kv["failed"]
}
END {
	lo = 6; hi = rows - 3
	if (hi < lo) { lo = 1; hi = rows }
	for (i = lo; i <= hi; i++) {
		c++; s1 += sr[i]; s2 += cpu[i]; s3 += al[i]; s4 += mb[i]; s5 += gc[i]; s6 += gcc[i]
	}
	if (c == 0) { print "no steady-state rows"; exit 1 }
	printf "steady rows=%d  stored/s=%.0f  cpu_ms/1k=%.1f  allocs/msg=%.1f  alloc_MB/s=%.1f  gc/s=%.1f  gc_cpu=%.1f%%  max_sys=%.1fMB  dropped=%d  failed=%d\n",
		c, s1/c, s2/c, s3/c, s4/c, s5/c, s6/c, sys, dropped, failed
}' "$LOG" | tee "results/$LABEL.summary"
