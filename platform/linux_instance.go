//go:build linux

package platform

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

// SingleInstanceGuard удерживает блокировку первого экземпляра приложения на Linux.
type SingleInstanceGuard struct {
	mu         sync.Mutex
	lockFile   *os.File
	lockPath   string
	sockPath   string
	listener   net.Listener
	onSecond   func(args []string)
	ipcStarted bool
	closed     bool
}

// SetCallback задаёт обработчик повторного запуска приложения.
func (g *SingleInstanceGuard) SetCallback(fn func(args []string)) {
	if g == nil {
		return
	}
	g.mu.Lock()
	g.onSecond = fn
	g.mu.Unlock()
}

// StartIPC запускает приём сообщений от повторных запусков приложения по UNIX-сокету.
func (g *SingleInstanceGuard) StartIPC() error {
	if g == nil {
		return nil
	}
	g.mu.Lock()
	if g.ipcStarted || g.closed || g.listener == nil {
		g.mu.Unlock()
		return nil
	}
	g.ipcStarted = true
	g.mu.Unlock()
	go g.serve()
	return nil
}

// Close освобождает блокировку и закрывает IPC-сокет.
func (g *SingleInstanceGuard) Close() error {
	if g == nil {
		return nil
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.closed {
		return nil
	}
	g.closed = true

	if g.listener != nil {
		_ = g.listener.Close()
		g.listener = nil
	}
	if g.sockPath != "" {
		_ = os.Remove(g.sockPath)
	}
	if g.lockFile != nil {
		_ = unix.Flock(int(g.lockFile.Fd()), unix.LOCK_UN)
		_ = g.lockFile.Close()
		g.lockFile = nil
	}
	if g.lockPath != "" {
		_ = os.Remove(g.lockPath)
	}
	return nil
}

// AcquireSingleInstance гарантирует запуск только одной копии приложения с заданным appID.
// Если первый экземпляр уже запущен, передаёт ему args по UNIX-сокету и возвращает alreadyRunning = true.
func AcquireSingleInstance(appID string, args []string, onSecond func(args []string)) (*SingleInstanceGuard, bool, error) {
	if appID == "" {
		return nil, false, errors.New("appID не может быть пустым")
	}

	sum := sha256.Sum256([]byte(appID))
	hash := hex.EncodeToString(sum[:12])

	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		runtimeDir = os.TempDir()
	}
	lockPath := filepath.Join(runtimeDir, "anyui-"+hash+".lock")
	sockPath := filepath.Join(runtimeDir, "anyui-"+hash+".sock")

	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, false, err
	}

	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		_ = f.Close()
		if errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN) {
			_ = notifyFirstLinuxInstance(sockPath, args)
			return nil, true, nil
		}
		return nil, false, err
	}

	// Удаляем устаревший сокет, если предыдущий процесс завершился аварийно
	_ = os.Remove(sockPath)
	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		_ = unix.Flock(int(f.Fd()), unix.LOCK_UN)
		_ = f.Close()
		return nil, false, err
	}

	guard := &SingleInstanceGuard{
		lockFile: f,
		lockPath: lockPath,
		sockPath: sockPath,
		listener: ln,
		onSecond: onSecond,
	}

	if onSecond != nil {
		_ = guard.StartIPC()
	}
	return guard, false, nil
}

func (g *SingleInstanceGuard) serve() {
	for {
		conn, err := g.listener.Accept()
		if err != nil {
			g.mu.Lock()
			closed := g.closed
			g.mu.Unlock()
			if closed {
				return
			}
			continue
		}

		go func(c net.Conn) {
			defer c.Close()
			_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
			data, err := io.ReadAll(io.LimitReader(c, 1<<20))
			if err != nil {
				return
			}
			var args []string
			if len(data) > 0 {
				_ = json.Unmarshal(data, &args)
			}
			g.mu.Lock()
			cb := g.onSecond
			g.mu.Unlock()
			if cb != nil {
				cb(args)
			}
		}(conn)
	}
}

func notifyFirstLinuxInstance(sockPath string, args []string) error {
	conn, err := net.DialTimeout("unix", sockPath, 2*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()
	_ = conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	payload, err := json.Marshal(args)
	if err != nil {
		return err
	}
	_, err = conn.Write(payload)
	return err
}
