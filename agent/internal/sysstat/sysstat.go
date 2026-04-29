// Package sysstat — реальные системные метрики агента.
//
// Phase 8: агент в каждом heartbeat отправляет CPU%/RAM%/load_avg + хеш
// текущего config.json для drift-detection.
//
// Linux-only (читает /proc). На non-linux хосте все Read* возвращают нули
// — это OK для production (агент крутится в docker:linux).
package sysstat

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Snapshot — текущие метрики агента.
type Snapshot struct {
	CPUPct     float64 `json:"cpu_pct"`
	RAMPct     float64 `json:"ram_pct"`
	LoadAvg1   float64 `json:"load_avg_1"`
	ConfigHash string  `json:"config_hash"`
	Timestamp  int64   `json:"timestamp"`
}

var (
	mu        sync.Mutex
	prevTotal uint64
	prevIdle  uint64
)

// Read — сбор метрик. Безопасно вызывать из разных горутин.
func Read(configPath string) Snapshot {
	cpu, _ := readCPU()
	ram, _ := readRAM()
	la, _ := readLoadAvg()
	hash, _ := hashFile(configPath)
	return Snapshot{
		CPUPct:     cpu,
		RAMPct:     ram,
		LoadAvg1:   la,
		ConfigHash: hash,
		Timestamp:  time.Now().Unix(),
	}
}

// readCPU — % CPU на основе разницы между двумя замерами /proc/stat.
// Первый вызов вернёт 0 (нет baseline'а).
func readCPU() (float64, error) {
	b, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0, err
	}
	line := strings.SplitN(string(b), "\n", 2)[0]
	if !strings.HasPrefix(line, "cpu ") {
		return 0, errors.New("unexpected /proc/stat")
	}
	fields := strings.Fields(line)[1:]
	if len(fields) < 5 {
		return 0, errors.New("short /proc/stat")
	}
	var total uint64
	for _, f := range fields {
		v, _ := strconv.ParseUint(f, 10, 64)
		total += v
	}
	idle, _ := strconv.ParseUint(fields[3], 10, 64)

	mu.Lock()
	defer mu.Unlock()
	dt, di := total-prevTotal, idle-prevIdle
	prevTotal, prevIdle = total, idle
	if dt == 0 {
		return 0, nil
	}
	return float64(dt-di) / float64(dt) * 100.0, nil
}

func readRAM() (float64, error) {
	b, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, err
	}
	var total, available uint64
	for _, line := range strings.Split(string(b), "\n") {
		switch {
		case strings.HasPrefix(line, "MemTotal:"):
			total = parseKB(line)
		case strings.HasPrefix(line, "MemAvailable:"):
			available = parseKB(line)
		}
	}
	if total == 0 {
		return 0, errors.New("no MemTotal")
	}
	used := total - available
	return float64(used) / float64(total) * 100.0, nil
}

func parseKB(line string) uint64 {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return 0
	}
	v, _ := strconv.ParseUint(fields[1], 10, 64)
	return v
}

func readLoadAvg() (float64, error) {
	b, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return 0, err
	}
	fields := strings.Fields(string(b))
	if len(fields) < 1 {
		return 0, errors.New("empty /proc/loadavg")
	}
	v, err := strconv.ParseFloat(fields[0], 64)
	return v, err
}

func hashFile(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}
