package progress

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"sync"
	"time"
)

// defaultLockTimeout dipakai bila pemanggil tidak menetapkan deadline.
const defaultLockTimeout = 2 * time.Second

// Retry lock: spin singkat untuk kontensi biasa, lalu backoff agar tidak
// membuang CPU bila lock dipegang lama.
const (
	spinAttempts = 25
	spinDelay    = 2 * time.Millisecond
	backoffDelay = 20 * time.Millisecond
)

// mutexes menjaga lock intra-proses. Tanpa ini, goroutine yang baru selesai
// memegang lock langsung mencobanya lagi dan memenangkan race, sehingga
// penunggu lain bisa kelaparan (terukur: 8 goroutine, lock dipegang 10 ms,
// tunggu maksimum 900 ms). Dengan mutex per path, penyerah lock harus
// mengantre di mutex dulu, jadi handoff menjadi adil.
var mutexes sync.Map // path -> *sync.Mutex

func lockMutex(path string) *sync.Mutex {
	m, _ := mutexes.LoadOrStore(path, &sync.Mutex{})
	return m.(*sync.Mutex)
}

// lockHandle memegang satu berkas lock.
type lockHandle struct {
	f *os.File
}

// acquire mengambil lock eksklusif pada path, dengan batas context.
//
// Lock file terpisah dari file data: atomic replace mengganti inode file data,
// sehingga lock pada file data tidak melindungi berkas baru.
//
// Blocking LockFileEx dengan overlapped nil di Windows menyebabkan access
// violation (terukur), sehingga lock diambil dengan retry berbatas context.
func acquire(ctx context.Context, path string) (func(), error) {
	m := lockMutex(path)
	m.Lock()
	return acquireHeld(ctx, path, m)
}

// acquireHeld mengambil lock dengan mutex intra-proses sudah dipegang.
func acquireHeld(ctx context.Context, path string, m *sync.Mutex) (func(), error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, defaultLockTimeout)
		defer cancel()
	}
	var owned bool
	defer func() {
		if !owned {
			m.Unlock()
		}
	}()

	f, err := openLockFile(path)
	if err != nil {
		return nil, fmt.Errorf("buka lock %s: %w", path, err)
	}
	h := &lockHandle{f: f}

	for attempt := 0; ; attempt++ {
		if err := h.tryLock(); err == nil {
			owned = true
			return func() {
				_ = h.unlock()
				_ = f.Close()
				m.Unlock()
			}, nil
		}
		select {
		case <-ctx.Done():
			_ = f.Close()
			return nil, fmt.Errorf("data sedang dipakai aplikasi lain (%s): %w; coba lagi", path, ctx.Err())
		case <-time.After(retryDelay(attempt)):
		}
	}
}

// retryDelay memberi jeda acak untuk percobaan ke-n.
func retryDelay(attempt int) time.Duration {
	if attempt < spinAttempts {
		return spinDelay + time.Duration(rand.Int64N(int64(spinDelay)))
	}
	return backoffDelay + time.Duration(rand.Int64N(int64(backoffDelay)))
}

// errLockHeld membedakan lock yang sedang dipegang proses lain.
var errLockHeld = errors.New("lock sedang dipegang")

func (h *lockHandle) tryLock() error { return flockLock(h.f) }
func (h *lockHandle) unlock() error  { return flockUnlock(h.f) }
