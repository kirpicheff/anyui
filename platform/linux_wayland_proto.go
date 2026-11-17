//go:build linux

package platform

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"golang.org/x/sys/unix"
)

const (
	wlDisplayID uint32 = 1

	// Опкоды wl_display
	wlDisplaySync        uint16 = 0
	wlDisplayGetRegistry uint16 = 1

	// События wl_display
	wlDisplayEventError    uint16 = 0
	wlDisplayEventDeleteID uint16 = 1

	// Опкоды wl_registry
	wlRegistryBind uint16 = 0

	// События wl_registry
	wlRegistryEventGlobal       uint16 = 0
	wlRegistryEventGlobalRemove uint16 = 1

	// События wl_callback
	wlCallbackEventDone uint16 = 0
)

// wlGlobal описывает глобальный объект, объявленный сервером Wayland.
type wlGlobal struct {
	Name      uint32
	Interface string
	Version   uint32
}

// wlEvent представляет входящее сообщение от Wayland-сервера.
type wlEvent struct {
	ObjectID uint32
	Opcode   uint16
	Payload  []byte
	FDs      []int
}

// wlEventHandler обрабатывает событие конкретного объекта Wayland.
type wlEventHandler func(ev wlEvent)

// wlConn инкапсулирует соединение по wire-протоколу Wayland без CGO.
type wlConn struct {
	fd int

	sendMu sync.Mutex

	objMu    sync.RWMutex
	nextID   uint32
	handlers map[uint32]wlEventHandler

	globalsMu sync.RWMutex
	globals   map[string]wlGlobal

	readBuf []byte
	fdQueue []int

	wakeR int
	wakeW int

	closed atomic.Bool
}

// connectWayland открывает Unix-сокет Wayland и инициализирует базовый реестр.
func connectWayland() (*wlConn, error) {
	sockPath, err := resolveWaylandSocketPath()
	if err != nil {
		return nil, err
	}

	fd, err := unix.Socket(unix.AF_UNIX, unix.SOCK_STREAM|unix.SOCK_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("wayland socket: %w", err)
	}

	if err := unix.Connect(fd, &unix.SockaddrUnix{Name: sockPath}); err != nil {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("wayland connect %s: %w", sockPath, err)
	}

	var pipeFDs [2]int
	if err := unix.Pipe2(pipeFDs[:], unix.O_NONBLOCK|unix.O_CLOEXEC); err != nil {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("wayland wake pipe: %w", err)
	}

	c := &wlConn{
		fd:       fd,
		nextID:   2, // 1 зарезервирован за wl_display
		handlers: make(map[uint32]wlEventHandler),
		globals:  make(map[string]wlGlobal),
		readBuf:  make([]byte, 0, 16384),
		wakeR:    pipeFDs[0],
		wakeW:    pipeFDs[1],
	}

	c.setHandler(wlDisplayID, func(ev wlEvent) {
		switch ev.Opcode {
		case wlDisplayEventError:
			r := wlReader{data: ev.Payload}
			objID, _ := r.uint32()
			code, _ := r.uint32()
			msg, _ := r.string()
			_ = objID
			_ = code
			_ = msg
			c.Close()
		case wlDisplayEventDeleteID:
			r := wlReader{data: ev.Payload}
			if id, ok := r.uint32(); ok {
				c.removeHandler(id)
			}
		}
	})

	return c, nil
}

func resolveWaylandSocketPath() (string, error) {
	disp := strings.TrimSpace(os.Getenv("WAYLAND_DISPLAY"))
	if disp == "" {
		disp = "wayland-0"
	}
	if filepath.IsAbs(disp) {
		return disp, nil
	}
	runtimeDir := strings.TrimSpace(os.Getenv("XDG_RUNTIME_DIR"))
	if runtimeDir == "" {
		return "", errors.New("XDG_RUNTIME_DIR не задан")
	}
	return filepath.Join(runtimeDir, disp), nil
}

func (c *wlConn) Close() {
	if c.closed.Swap(true) {
		return
	}
	_ = unix.Close(c.fd)
	_ = unix.Close(c.wakeR)
	_ = unix.Close(c.wakeW)
	for _, fd := range c.fdQueue {
		_ = unix.Close(fd)
	}
	c.fdQueue = nil
}

func (c *wlConn) wakeUp() {
	if c.closed.Load() {
		return
	}
	var b [1]byte
	b[0] = 1
	_, _ = unix.Write(c.wakeW, b[:])
}

func (c *wlConn) drainWakePipe() {
	var buf [64]byte
	for {
		_, err := unix.Read(c.wakeR, buf[:])
		if err != nil {
			break
		}
	}
}

func (c *wlConn) allocID(handler wlEventHandler) uint32 {
	c.objMu.Lock()
	id := c.nextID
	c.nextID++
	if handler != nil {
		c.handlers[id] = handler
	}
	c.objMu.Unlock()
	return id
}

func (c *wlConn) setHandler(id uint32, handler wlEventHandler) {
	c.objMu.Lock()
	if handler == nil {
		delete(c.handlers, id)
	} else {
		c.handlers[id] = handler
	}
	c.objMu.Unlock()
}

func (c *wlConn) removeHandler(id uint32) {
	c.objMu.Lock()
	delete(c.handlers, id)
	c.objMu.Unlock()
}

func (c *wlConn) dispatch(ev wlEvent) {
	c.objMu.RLock()
	h := c.handlers[ev.ObjectID]
	c.objMu.RUnlock()
	if h != nil {
		h(ev)
	}
	// Закрываем неиспользованные fd, если обработчик их не забрал
	for _, fd := range ev.FDs {
		if fd >= 0 {
			_ = unix.Close(fd)
		}
	}
}

// send отправляет сообщение Wayland без файловых дескрипторов.
func (c *wlConn) send(objectID uint32, opcode uint16, payload []byte) error {
	return c.sendWithFDs(objectID, opcode, payload, nil)
}

// sendWithFDs отправляет сообщение Wayland с опциональными файловыми дескрипторами (SCM_RIGHTS).
func (c *wlConn) sendWithFDs(objectID uint32, opcode uint16, payload []byte, fds []int) error {
	if c.closed.Load() {
		return errors.New("соединение Wayland закрыто")
	}
	totalLen := 8 + len(payload)
	if totalLen > 0xFFFF {
		return fmt.Errorf("слишком большое сообщение Wayland: %d", totalLen)
	}

	pkt := make([]byte, totalLen)
	binary.LittleEndian.PutUint32(pkt[0:4], objectID)
	binary.LittleEndian.PutUint32(pkt[4:8], (uint32(totalLen)<<16)|uint32(opcode))
	copy(pkt[8:], payload)

	var oob []byte
	if len(fds) > 0 {
		oob = unix.UnixRights(fds...)
	}

	c.sendMu.Lock()
	defer c.sendMu.Unlock()

	offset := 0
	for offset < len(pkt) {
		n, err := unix.SendmsgN(c.fd, pkt[offset:], oob, nil, unix.MSG_NOSIGNAL)
		if err != nil {
			if errors.Is(err, unix.EINTR) {
				continue
			}
			return err
		}
		if n == 0 {
			return io.ErrUnexpectedEOF
		}
		offset += n
		oob = nil // ancillary data передаётся только с первым фрагментом
	}
	return nil
}

// roundtrip выполняет синхронный wl_display.sync и ждёт ответа сервера.
func (c *wlConn) roundtrip() error {
	done := false
	cbID := c.allocID(func(ev wlEvent) {
		if ev.Opcode == wlCallbackEventDone {
			done = true
		}
	})
	var w wlWriter
	w.uint32(cbID)
	if err := c.send(wlDisplayID, wlDisplaySync, w.bytes()); err != nil {
		c.removeHandler(cbID)
		return err
	}
	for !done && !c.closed.Load() {
		if err := c.readAndDispatch(true); err != nil {
			return err
		}
	}
	c.removeHandler(cbID)
	return nil
}

// bindGlobal привязывает глобальный объект по имени интерфейса.
func (c *wlConn) bindGlobal(registryID uint32, iface string, maxVersion uint32, handler wlEventHandler) (uint32, uint32, error) {
	c.globalsMu.RLock()
	g, ok := c.globals[iface]
	c.globalsMu.RUnlock()
	if !ok {
		return 0, 0, fmt.Errorf("глобальный интерфейс Wayland %q не найден", iface)
	}
	ver := g.Version
	if maxVersion > 0 && ver > maxVersion {
		ver = maxVersion
	}
	id := c.allocID(handler)
	var w wlWriter
	w.uint32(g.Name)
	w.string(iface)
	w.uint32(ver)
	w.uint32(id)
	if err := c.send(registryID, wlRegistryBind, w.bytes()); err != nil {
		c.removeHandler(id)
		return 0, 0, err
	}
	return id, ver, nil
}

func (c *wlConn) hasGlobal(iface string) bool {
	c.globalsMu.RLock()
	_, ok := c.globals[iface]
	c.globalsMu.RUnlock()
	return ok
}

// waitEvents ждёт событий на сокете Wayland или пробуждения через wakePipe.
func (c *wlConn) waitEvents(timeoutMs int) (waylandReady bool, wakeReady bool, err error) {
	fds := []unix.PollFd{
		{Fd: int32(c.fd), Events: unix.POLLIN},
		{Fd: int32(c.wakeR), Events: unix.POLLIN},
	}
	for {
		n, pollErr := unix.Poll(fds, timeoutMs)
		if pollErr != nil {
			if errors.Is(pollErr, unix.EINTR) {
				continue
			}
			return false, false, pollErr
		}
		if n == 0 {
			return false, false, nil
		}
		if fds[0].Revents&(unix.POLLERR|unix.POLLHUP|unix.POLLNVAL) != 0 {
			return true, false, io.EOF
		}
		wReady := fds[0].Revents&unix.POLLIN != 0
		kReady := fds[1].Revents&unix.POLLIN != 0
		return wReady, kReady, nil
	}
}

// readAndDispatch читает доступные данные из сокета и диспетчеризует все полные сообщения.
func (c *wlConn) readAndDispatch(blocking bool) error {
	if c.closed.Load() {
		return io.EOF
	}
	if blocking && len(c.readBuf) < 8 {
		if err := c.readOnce(); err != nil {
			return err
		}
	} else if !blocking {
		// Неблокирующее чтение всего, что есть в буфере сокета
		for {
			ready, _, err := c.waitEvents(0)
			if err != nil {
				return err
			}
			if !ready {
				break
			}
			if err := c.readOnce(); err != nil {
				return err
			}
		}
	}

	for len(c.readBuf) >= 8 {
		objID := binary.LittleEndian.Uint32(c.readBuf[0:4])
		sizeOp := binary.LittleEndian.Uint32(c.readBuf[4:8])
		msgSize := int(sizeOp >> 16)
		opcode := uint16(sizeOp & 0xFFFF)
		if msgSize < 8 {
			return fmt.Errorf("некорректный размер сообщения Wayland: %d", msgSize)
		}
		if len(c.readBuf) < msgSize {
			if !blocking {
				break
			}
			if err := c.readOnce(); err != nil {
				return err
			}
			continue
		}

		payload := make([]byte, msgSize-8)
		copy(payload, c.readBuf[8:msgSize])

		// Копируем оставшийся хвост в начало буфера
		rem := len(c.readBuf) - msgSize
		copy(c.readBuf[:rem], c.readBuf[msgSize:])
		c.readBuf = c.readBuf[:rem]

		var evFDs []int
		if len(c.fdQueue) > 0 && needsFDForEvent(objID, opcode, c) {
			evFDs = []int{c.fdQueue[0]}
			c.fdQueue = c.fdQueue[1:]
		}

		c.dispatch(wlEvent{
			ObjectID: objID,
			Opcode:   opcode,
			Payload:  payload,
			FDs:      evFDs,
		})
	}

	return nil
}

func (c *wlConn) readOnce() error {
	var buf [8192]byte
	var oob [1024]byte
	for {
		n, oobn, _, _, err := unix.Recvmsg(c.fd, buf[:], oob[:], unix.MSG_CMSG_CLOEXEC)
		if err != nil {
			if errors.Is(err, unix.EINTR) {
				continue
			}
			return err
		}
		if n == 0 {
			return io.EOF
		}
		c.readBuf = append(c.readBuf, buf[:n]...)
		if oobn > 0 {
			scms, scmErr := unix.ParseSocketControlMessage(oob[:oobn])
			if scmErr == nil {
				for _, scm := range scms {
					fds, fdErr := unix.ParseUnixRights(&scm)
					if fdErr == nil {
						c.fdQueue = append(c.fdQueue, fds...)
					}
				}
			}
		}
		return nil
	}
}

// needsFDForEvent проверяет, ожидает ли данное событие файловый дескриптор.
// Это гарантирует, что переданные через SCM_RIGHTS fd попадают строго в нужное событие.
func needsFDForEvent(objID uint32, opcode uint16, c *wlConn) bool {
	c.objMu.RLock()
	_, ok := c.handlers[objID]
	c.objMu.RUnlock()
	if !ok {
		return false
	}
	// Помечаем известные типы объектов в wlConn не требуется — достаточно проверять fd-несущие события:
	// wl_keyboard.keymap (opcode 0) и wl_data_device/offer не приходят одновременно с другими fd-событиями,
	// а wl_keyboard.keymap имеет payload >= 8 байт (format uint32 + size uint32).
	// Точную привязку делаем через таблицу fdEventObjects.
	return c.isFDExpectingEvent(objID, opcode)
}

var (
	wlFDEventsMu sync.RWMutex
	wlFDEvents   = make(map[uint32]uint16) // objectID -> opcode, который ожидает fd
)

func (c *wlConn) registerFDEvent(objID uint32, opcode uint16) {
	wlFDEventsMu.Lock()
	wlFDEvents[objID] = opcode
	wlFDEventsMu.Unlock()
}

func (c *wlConn) unregisterFDEvents(objID uint32) {
	wlFDEventsMu.Lock()
	delete(wlFDEvents, objID)
	wlFDEventsMu.Unlock()
}

func (c *wlConn) isFDExpectingEvent(objID uint32, opcode uint16) bool {
	wlFDEventsMu.RLock()
	wantOp, ok := wlFDEvents[objID]
	wlFDEventsMu.RUnlock()
	return ok && wantOp == opcode
}

// wlWriter собирает payload сообщения Wayland с выравниванием до 4 байт.
type wlWriter struct {
	buf []byte
}

func (w *wlWriter) uint32(v uint32) {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], v)
	w.buf = append(w.buf, b[:]...)
}

func (w *wlWriter) int32(v int32) {
	w.uint32(uint32(v))
}

func (w *wlWriter) fixed(v float64) {
	w.int32(int32(math.Round(v * 256.0)))
}

func (w *wlWriter) string(s string) {
	if s == "" {
		// Пустая строка в Wayland: длина 1 (только '\0') или 0 для null; для не-null передаём '\0'.
		w.uint32(1)
		w.buf = append(w.buf, 0, 0, 0, 0)
		return
	}
	rawLen := len(s) + 1
	w.uint32(uint32(rawLen))
	w.buf = append(w.buf, s...)
	w.buf = append(w.buf, 0)
	for len(w.buf)%4 != 0 {
		w.buf = append(w.buf, 0)
	}
}

func (w *wlWriter) array(data []byte) {
	w.uint32(uint32(len(data)))
	w.buf = append(w.buf, data...)
	for len(w.buf)%4 != 0 {
		w.buf = append(w.buf, 0)
	}
}

func (w *wlWriter) bytes() []byte {
	return w.buf
}

// wlReader читает аргументы входящего сообщения Wayland.
type wlReader struct {
	data []byte
	pos  int
}

func (r *wlReader) uint32() (uint32, bool) {
	if r.pos+4 > len(r.data) {
		return 0, false
	}
	v := binary.LittleEndian.Uint32(r.data[r.pos : r.pos+4])
	r.pos += 4
	return v, true
}

func (r *wlReader) int32() (int32, bool) {
	v, ok := r.uint32()
	return int32(v), ok
}

func (r *wlReader) fixed() (float64, bool) {
	v, ok := r.int32()
	return float64(v) / 256.0, ok
}

func (r *wlReader) string() (string, bool) {
	n, ok := r.uint32()
	if !ok {
		return "", false
	}
	if n == 0 {
		return "", true
	}
	padded := (int(n) + 3) &^ 3
	if r.pos+padded > len(r.data) {
		return "", false
	}
	s := string(r.data[r.pos : r.pos+int(n)-1])
	r.pos += padded
	return s, true
}

func (r *wlReader) array() ([]byte, bool) {
	n, ok := r.uint32()
	if !ok {
		return nil, false
	}
	padded := (int(n) + 3) &^ 3
	if r.pos+padded > len(r.data) {
		return nil, false
	}
	out := make([]byte, int(n))
	copy(out, r.data[r.pos:r.pos+int(n)])
	r.pos += padded
	return out, true
}

