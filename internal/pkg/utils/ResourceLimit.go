package utils

import "sync"

/*
 * ResourceLimit handles resource limits for leaderelection
 *
 */
type ResourceLimit struct {
	mux sync.Mutex
	// lock is used for wait function in case limit is reached
	lock *sync.Cond
	// tracks the currently available resources
	currentLimit int
	// Needed for updating the current limit
	totalLimit int
}

/*
 * Returns a new ResourceLimit with its values already filled
 */
func NewResourceLimit(limit int) ResourceLimit {
	r := ResourceLimit{currentLimit: limit, totalLimit: limit}
	r.lock = sync.NewCond(&r)
	return r
}

/*
 * Wait until resources become available again
 */
func (r ResourceLimit) WaitUntilResource() {
	// TODO: research cleaner code
	r.cond.Wait()
}

/*
 * Try to acquire a resource, return true if a resource was acquired successfully
 */
func (r ResourceLimit) Acquire() bool {
	r.mux.Lock()
	if r.currentLimit < 1 {
		r.mux.Unlock()
		return false
	}
	r.currentLimit--
	r.mux.Unlock()
	return true
}

/*
 * Releases a resource
 */
func (r ResourceLimit) Release() {
	r.mux.Lock()
	r.currentLimit++
	r.mux.Unlock()
}

/*
 * Updates the total available resources
 * This could change the limit to a negative value in which case resources will not be
 * released, but no resources can be aquired until the limit is at least 1 again
 */
func (r ResourceLimit) UpdateLimit(limit int) {
	r.mux.Lock()
	diff := limit - r.totalLimit
	r.currentLimit += diff
	r.totalLimit = limit
	r.mux.Unlock()
}
