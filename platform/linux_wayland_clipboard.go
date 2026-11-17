//go:build linux

package platform

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"io"
	"os"
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

const (
	// wl_data_device_manager requests
	wlDataDevMgrCreateDataSource uint16 = 0
	wlDataDevMgrGetDataDevice    uint16 = 1

	// wl_data_device requests & events
	wlDataDevSetSelection   uint16 = 1
	wlDataDevEventDataOffer uint16 = 0
	wlDataDevEventEnter     uint16 = 1
	wlDataDevEventLeave     uint16 = 2
	wlDataDevEventMotion    uint16 = 3
	wlDataDevEventDrop      uint16 = 4
	wlDataDevEventSelection uint16 = 5

	// wl_data_offer requests & events
	wlDataOfferAccept     uint16 = 0
	wlDataOfferReceive    uint16 = 1
	wlDataOfferDestroy    uint16 = 2
	wlDataOfferFinish     uint16 = 4
	wlDataOfferSetActions uint16 = 5
	wlDataOfferEventOffer uint16 = 0

	// wl_data_source requests & events
	wlDataSourceOffer          uint16 = 0
	wlDataSourceDestroy        uint16 = 1
	wlDataSourceEventSend      uint16 = 1
	wlDataSourceEventCancelled uint16 = 2

	wlDataDevMgrDndActionCopy uint32 = 1
)

type wlDataOfferState struct {
	id    uint32
	mimes []string
}

type wlDataDeviceState struct {
	disp        *wlDisplay
	devID       uint32
	devVer      uint32
	mu          sync.Mutex
	offers      map[uint32]*wlDataOfferState
	selectionID uint32

	dndOfferID   uint32
	dndSurfaceID uint32
	dndSerial    uint32

	activeSourceID uint32
	sourceText     string
	sourcePNG      []byte
	sourceHasText  bool
	sourceHasPNG   bool
}

func newWLDataDeviceState(d *wlDisplay) *wlDataDeviceState {
	if d.dataDevMgrID == 0 || d.seatID == 0 {
		return nil
	}
	devID := d.conn.allocID(nil)
	s := &wlDataDeviceState{
		disp:   d,
		devID:  devID,
		devVer: 3,
		offers: make(map[uint32]*wlDataOfferState),
	}
	d.conn.setHandler(devID, s.handleDataDeviceEvent)

	var w wlWriter
	w.uint32(devID)
	w.uint32(d.seatID)
	_ = d.conn.send(d.dataDevMgrID, wlDataDevMgrGetDataDevice, w.bytes())
	return s
}

func (s *wlDataDeviceState) handleDataDeviceEvent(ev wlEvent) {
	r := wlReader{data: ev.Payload}

	switch ev.Opcode {
	case wlDataDevEventDataOffer:
		offerID, ok := r.uint32()
		if !ok || offerID == 0 {
			return
		}
		offer := &wlDataOfferState{id: offerID}
		s.mu.Lock()
		s.offers[offerID] = offer
		s.mu.Unlock()

		s.disp.conn.setHandler(offerID, func(oev wlEvent) {
			if oev.Opcode == wlDataOfferEventOffer {
				or := wlReader{data: oev.Payload}
				if mime, ok := or.string(); ok {
					s.mu.Lock()
					offer.mimes = append(offer.mimes, mime)
					s.mu.Unlock()
				}
			}
		})

	case wlDataDevEventSelection:
		offerID, ok := r.uint32()
		if !ok {
			return
		}
		s.mu.Lock()
		old := s.selectionID
		s.selectionID = offerID
		s.mu.Unlock()
		if old != 0 && old != offerID {
			s.destroyOffer(old)
		}

	case wlDataDevEventEnter:
		serial, ok1 := r.uint32()
		surfID, ok2 := r.uint32()
		_, _ = r.fixed() // x
		_, _ = r.fixed() // y
		offerID, ok3 := r.uint32()
		if !ok1 || !ok2 || !ok3 {
			return
		}
		s.mu.Lock()
		s.dndSerial = serial
		s.dndSurfaceID = surfID
		s.dndOfferID = offerID
		var hasURI bool
		if offer := s.offers[offerID]; offer != nil {
			for _, m := range offer.mimes {
				if m == "text/uri-list" {
					hasURI = true
					break
				}
			}
		}
		s.mu.Unlock()

		if offerID != 0 {
			var aw wlWriter
			aw.uint32(serial)
			if hasURI {
				aw.string("text/uri-list")
			} else {
				aw.uint32(0) // null mime -> отклоняем
			}
			_ = s.disp.conn.send(offerID, wlDataOfferAccept, aw.bytes())

			if hasURI {
				var sw wlWriter
				sw.uint32(wlDataDevMgrDndActionCopy)
				sw.uint32(wlDataDevMgrDndActionCopy)
				_ = s.disp.conn.send(offerID, wlDataOfferSetActions, sw.bytes())
			}
		}

	case wlDataDevEventLeave:
		s.mu.Lock()
		old := s.dndOfferID
		s.dndOfferID = 0
		s.dndSurfaceID = 0
		s.mu.Unlock()
		if old != 0 {
			s.destroyOffer(old)
		}

	case wlDataDevEventDrop:
		s.mu.Lock()
		offerID := s.dndOfferID
		surfID := s.dndSurfaceID
		s.dndOfferID = 0
		s.dndSurfaceID = 0
		s.mu.Unlock()

		if offerID == 0 || surfID == 0 {
			return
		}
		win := s.disp.windowBySurface(surfID)
		if win == nil || !win.cfg.AcceptDropFiles {
			s.destroyOffer(offerID)
			return
		}
		data, err := s.receiveOffer(offerID, "text/uri-list")
		_ = s.disp.conn.send(offerID, wlDataOfferFinish, nil)
		s.destroyOffer(offerID)
		if err == nil && len(data) > 0 {
			files := parseXDNDURIList(string(data))
			if len(files) > 0 {
				win.emit(Event{
					Type:  EventDropFiles,
					Files: files,
				})
			}
		}
	}
}

func (s *wlDataDeviceState) destroyOffer(offerID uint32) {
	if offerID == 0 {
		return
	}
	s.mu.Lock()
	delete(s.offers, offerID)
	s.mu.Unlock()
	_ = s.disp.conn.send(offerID, wlDataOfferDestroy, nil)
	s.disp.conn.removeHandler(offerID)
}

func (s *wlDataDeviceState) receiveOffer(offerID uint32, mime string) ([]byte, error) {
	var fds [2]int
	if err := unix.Pipe2(fds[:], unix.O_CLOEXEC); err != nil {
		return nil, err
	}
	readFD, writeFD := fds[0], fds[1]

	var w wlWriter
	w.string(mime)
	err := s.disp.conn.sendWithFDs(offerID, wlDataOfferReceive, w.bytes(), []int{writeFD})
	_ = unix.Close(writeFD)
	if err != nil {
		_ = unix.Close(readFD)
		return nil, err
	}

	// Сбрасываем исходящий запрос на сервер, чтобы источник начал писать в pipe
	_ = s.disp.conn.roundtrip()

	f := os.NewFile(uintptr(readFD), "wl-offer-pipe")
	defer f.Close()
	_ = f.SetReadDeadline(time.Now().Add(2 * time.Second))
	return io.ReadAll(io.LimitReader(f, 32*1024*1024))
}

func (s *wlDataDeviceState) readText() (string, error) {
	s.mu.Lock()
	selID := s.selectionID
	var offer *wlDataOfferState
	if selID != 0 {
		offer = s.offers[selID]
	}
	if s.activeSourceID != 0 && s.sourceHasText {
		txt := s.sourceText
		s.mu.Unlock()
		return txt, nil
	}
	var mimes []string
	if offer != nil {
		mimes = append([]string(nil), offer.mimes...)
	}
	s.mu.Unlock()

	if selID == 0 {
		return "", nil
	}
	target := pickTextMIME(mimes)
	if target == "" {
		return "", nil
	}
	data, err := s.receiveOffer(selID, target)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (s *wlDataDeviceState) writeText(text string) error {
	return s.publishSource(text, true, nil, false, []string{
		"text/plain;charset=utf-8",
		"text/plain",
		"UTF8_STRING",
	})
}

func (s *wlDataDeviceState) readImage() (image.Image, error) {
	s.mu.Lock()
	selID := s.selectionID
	if s.activeSourceID != 0 && s.sourceHasPNG {
		raw := append([]byte(nil), s.sourcePNG...)
		s.mu.Unlock()
		return png.Decode(bytes.NewReader(raw))
	}
	var mimes []string
	if offer := s.offers[selID]; offer != nil {
		mimes = append([]string(nil), offer.mimes...)
	}
	s.mu.Unlock()

	if selID == 0 {
		return nil, errors.New("буфер обмена пуст")
	}
	hasPNG := false
	for _, m := range mimes {
		if m == "image/png" {
			hasPNG = true
			break
		}
	}
	if !hasPNG {
		return nil, errors.New("в буфере обмена нет изображения image/png")
	}
	data, err := s.receiveOffer(selID, "image/png")
	if err != nil {
		return nil, err
	}
	return png.Decode(bytes.NewReader(data))
}

func (s *wlDataDeviceState) writeImage(img image.Image) error {
	if img == nil {
		return errors.New("nil image")
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return err
	}
	return s.publishSource("", false, buf.Bytes(), true, []string{"image/png"})
}

func (s *wlDataDeviceState) publishSource(text string, hasText bool, pngBytes []byte, hasPNG bool, mimes []string) error {
	serial := s.latestInputSerial()
	if serial == 0 {
		return errors.New("нет активного serial ввода Wayland для записи в буфер обмена")
	}

	s.mu.Lock()
	if s.activeSourceID != 0 {
		oldID := s.activeSourceID
		s.activeSourceID = 0
		_ = s.disp.conn.send(oldID, wlDataSourceDestroy, nil)
		s.disp.conn.unregisterFDEvents(oldID)
		s.disp.conn.removeHandler(oldID)
	}
	s.sourceText = text
	s.sourceHasText = hasText
	s.sourcePNG = pngBytes
	s.sourceHasPNG = hasPNG
	s.mu.Unlock()

	var srcID uint32
	srcID = s.disp.conn.allocID(func(ev wlEvent) {
		switch ev.Opcode {
		case wlDataSourceEventSend:
			r := wlReader{data: ev.Payload}
			mime, ok := r.string()
			if !ok || len(ev.FDs) == 0 {
				return
			}
			fd := ev.FDs[0]
			ev.FDs[0] = -1

			s.mu.Lock()
			txt, okTxt := s.sourceText, s.sourceHasText
			pngData, okPNG := s.sourcePNG, s.sourceHasPNG
			s.mu.Unlock()

			go func() {
				defer func() { _ = unix.Close(fd) }()
				if mime == "image/png" && okPNG {
					_, _ = unix.Write(fd, pngData)
				} else if okTxt {
					_, _ = unix.Write(fd, []byte(txt))
				}
			}()

		case wlDataSourceEventCancelled:
			s.mu.Lock()
			if s.activeSourceID == srcID {
				s.activeSourceID = 0
			}
			s.mu.Unlock()
			_ = s.disp.conn.send(srcID, wlDataSourceDestroy, nil)
			s.disp.conn.unregisterFDEvents(srcID)
			s.disp.conn.removeHandler(srcID)
		}
	})
	s.disp.conn.registerFDEvent(srcID, wlDataSourceEventSend)

	var cw wlWriter
	cw.uint32(srcID)
	if err := s.disp.conn.send(s.disp.dataDevMgrID, wlDataDevMgrCreateDataSource, cw.bytes()); err != nil {
		s.disp.conn.unregisterFDEvents(srcID)
		s.disp.conn.removeHandler(srcID)
		return err
	}

	for _, m := range mimes {
		var mw wlWriter
		mw.string(m)
		_ = s.disp.conn.send(srcID, wlDataSourceOffer, mw.bytes())
	}

	s.mu.Lock()
	s.activeSourceID = srcID
	s.mu.Unlock()

	var sw wlWriter
	sw.uint32(srcID)
	sw.uint32(serial)
	return s.disp.conn.send(s.devID, wlDataDevSetSelection, sw.bytes())
}

func (s *wlDataDeviceState) latestInputSerial() uint32 {
	if s.disp.seat == nil {
		return 0
	}
	s.disp.seat.mu.Lock()
	defer s.disp.seat.mu.Unlock()
	if s.disp.seat.lastInputSerial != 0 {
		return s.disp.seat.lastInputSerial
	}
	return s.disp.seat.pointerEnterSerial
}

func pickTextMIME(mimes []string) string {
	preferred := []string{"text/plain;charset=utf-8", "UTF8_STRING", "text/plain"}
	for _, want := range preferred {
		for _, m := range mimes {
			if m == want {
				return m
			}
		}
	}
	return ""
}

