package importer

import "time"

type speedRegulator struct {
	prevSpeed  float64
	speed      float64
	maxSize    int
	increasing bool
	totalSize  int
}

func newSpeedRegulator() *speedRegulator {
	return &speedRegulator{maxSize: initTotalBatchSize, increasing: true}
}

func (r *speedRegulator) updateTotalSize(size uint32) {
	r.totalSize += int(size)
}

func (r *speedRegulator) incomplete() bool {
	return r.totalSize < r.maxSize
}

func (r *speedRegulator) calculateSpeed(start time.Time) {
	elapsed := time.Since(start)
	r.speed = float64(r.totalSize) / float64(elapsed)
	r.maxSize, r.increasing = r.calculateNextMaxSizeAndDirection()
	r.prevSpeed = r.speed
	r.totalSize = 0
}

func (r *speedRegulator) calculateNextMaxSizeAndDirection() (int, bool) {
	maxSize := r.maxSize
	increasing := r.increasing
	switch {
	case r.speed > r.prevSpeed && r.increasing:
		maxSize += sizeAdjustment
		maxSize = min(maxSize, MaxTotalBatchSize)
	case r.speed > r.prevSpeed && !r.increasing:
		maxSize -= sizeAdjustment
		maxSize = max(maxSize, initTotalBatchSize)
	case r.speed < r.prevSpeed && r.increasing:
		increasing = false
		maxSize -= sizeAdjustment
		maxSize = max(maxSize, initTotalBatchSize)
	case r.speed < r.prevSpeed && !r.increasing:
		increasing = true
		maxSize += sizeAdjustment
		maxSize = min(maxSize, MaxTotalBatchSize)
	}
	return maxSize, increasing
}
