//go:build linux

package platform

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"
)

const (
	sniObjectPath  dbus.ObjectPath = "/StatusNotifierItem"
	sniMenuPath    dbus.ObjectPath = "/MenuBar"
	sniIface                       = "org.kde.StatusNotifierItem"
	sniWatcherName                 = "org.kde.StatusNotifierWatcher"
	sniWatcherPath dbus.ObjectPath = "/StatusNotifierWatcher"
	dbusMenuIface                  = "com.canonical.dbusmenu"
)

var sniInstanceCounter atomic.Uint32

type sniPixmap struct {
	Width  int32
	Height int32
	Data   []byte
}

type sniToolTip struct {
	IconName    string
	IconData    []sniPixmap
	Title       string
	Description string
}

type dbusMenuLayout struct {
	ID       int32
	Props    map[string]dbus.Variant
	Children []dbus.Variant
}

type dbusMenuItemProps struct {
	ID    int32
	Props map[string]dbus.Variant
}

type dbusMenuEvent struct {
	ID        int32
	EventID   string
	Data      dbus.Variant
	Timestamp uint32
}

type sniTray struct {
	conn        *dbus.Conn
	serviceName string
	sniProps    *prop.Properties
	menuProps   *prop.Properties

	mu          sync.Mutex
	cfg         TrayConfig
	menuItems   []TrayMenuItem
	menuRev     uint32
	pixmap      []sniPixmap
	tooltipText string
	closed      atomic.Bool
}

func newSNITray(cfg TrayConfig) (Tray, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, err
	}

	// Проверяем наличие org.kde.StatusNotifierWatcher на шине D-Bus
	var hasWatcher bool
	if err := conn.BusObject().Call("org.freedesktop.DBus.NameHasOwner", 0, sniWatcherName).Store(&hasWatcher); err != nil || !hasWatcher {
		_ = conn.Close()
		return nil, errors.New("StatusNotifierWatcher недоступен на шине D-Bus")
	}

	idx := sniInstanceCounter.Add(1)
	svcName := fmt.Sprintf("org.kde.StatusNotifierItem-%d-%d", os.Getpid(), idx)
	reply, err := conn.RequestName(svcName, dbus.NameFlagDoNotQueue)
	if err != nil || reply != dbus.RequestNameReplyPrimaryOwner {
		_ = conn.Close()
		return nil, errors.New("не удалось занять имя D-Bus для StatusNotifierItem")
	}

	rgba, w, h := cfg.IconRGBA, cfg.IconWidth, cfg.IconHeight
	if len(rgba) < w*h*4 || w <= 0 || h <= 0 {
		rgba, w, h = defaultLinuxTrayIconRGBA()
	}

	t := &sniTray{
		conn:        conn,
		serviceName: svcName,
		cfg:         cfg,
		menuItems:   append([]TrayMenuItem(nil), cfg.Menu...),
		menuRev:     1,
		pixmap:      []sniPixmap{rgbaToSNIPixmap(rgba, w, h)},
		tooltipText: cfg.Tooltip,
	}

	if err := conn.Export(t, sniObjectPath, sniIface); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if err := conn.Export(&sniMenuHandler{tray: t}, sniMenuPath, dbusMenuIface); err != nil {
		_ = conn.Close()
		return nil, err
	}

	t.sniProps, err = prop.Export(conn, sniObjectPath, t.buildSNIPropSpec())
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	t.menuProps, err = prop.Export(conn, sniMenuPath, t.buildMenuPropSpec())
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	_ = conn.Export(introspect.NewIntrospectable(&introspect.Node{
		Name: string(sniObjectPath),
		Interfaces: []introspect.Interface{
			introspect.IntrospectData,
			prop.IntrospectData,
			{Name: sniIface, Methods: introspect.Methods(t)},
		},
	}), sniObjectPath, "org.freedesktop.DBus.Introspectable")

	menuHandler := &sniMenuHandler{tray: t}
	_ = conn.Export(introspect.NewIntrospectable(&introspect.Node{
		Name: string(sniMenuPath),
		Interfaces: []introspect.Interface{
			introspect.IntrospectData,
			prop.IntrospectData,
			{Name: dbusMenuIface, Methods: introspect.Methods(menuHandler)},
		},
	}), sniMenuPath, "org.freedesktop.DBus.Introspectable")

	watcher := conn.Object(sniWatcherName, sniWatcherPath)
	if err := watcher.Call(sniWatcherName+".RegisterStatusNotifierItem", 0, svcName).Err; err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("RegisterStatusNotifierItem: %w", err)
	}

	return t, nil
}

func (t *sniTray) buildSNIPropSpec() prop.Map {
	t.mu.Lock()
	pix := t.pixmap
	tip := t.tooltipText
	t.mu.Unlock()

	return prop.Map{
		sniIface: {
			"Category":   {Value: "ApplicationStatus", Writable: false, Emit: prop.EmitTrue},
			"Id":         {Value: "anyui", Writable: false, Emit: prop.EmitTrue},
			"Title":      {Value: tip, Writable: false, Emit: prop.EmitTrue},
			"Status":     {Value: "Active", Writable: false, Emit: prop.EmitTrue},
			"IconName":   {Value: "", Writable: false, Emit: prop.EmitTrue},
			"IconPixmap": {Value: pix, Writable: false, Emit: prop.EmitTrue},
			"ToolTip":    {Value: sniToolTip{Title: tip}, Writable: false, Emit: prop.EmitTrue},
			"Menu":       {Value: sniMenuPath, Writable: false, Emit: prop.EmitTrue},
			"ItemIsMenu": {Value: false, Writable: false, Emit: prop.EmitTrue},
		},
	}
}

func (t *sniTray) buildMenuPropSpec() prop.Map {
	return prop.Map{
		dbusMenuIface: {
			"Version":       {Value: uint32(3), Writable: false, Emit: prop.EmitTrue},
			"TextDirection": {Value: "ltr", Writable: false, Emit: prop.EmitTrue},
			"Status":        {Value: "normal", Writable: false, Emit: prop.EmitTrue},
			"IconThemePath": {Value: []string{}, Writable: false, Emit: prop.EmitTrue},
		},
	}
}

// Методы org.kde.StatusNotifierItem
func (t *sniTray) Activate(x, y int32) *dbus.Error {
	_, _ = x, y
	t.mu.Lock()
	fn := t.cfg.OnClick
	t.mu.Unlock()
	if fn != nil {
		go fn()
	}
	return nil
}

func (t *sniTray) SecondaryActivate(x, y int32) *dbus.Error {
	_, _ = x, y
	t.mu.Lock()
	fn := t.cfg.OnRightClick
	t.mu.Unlock()
	if fn != nil {
		go fn()
	}
	return nil
}

func (t *sniTray) ContextMenu(x, y int32) *dbus.Error {
	return t.SecondaryActivate(x, y)
}

func (t *sniTray) Scroll(delta int32, orientation string) *dbus.Error {
	_, _ = delta, orientation
	return nil
}

// Методы интерфейса platform.Tray
func (t *sniTray) SetTooltip(tooltip string) error {
	if t.closed.Load() {
		return nil
	}
	t.mu.Lock()
	t.tooltipText = tooltip
	t.mu.Unlock()

	if t.sniProps != nil {
		t.sniProps.SetMust(sniIface, "Title", tooltip)
		t.sniProps.SetMust(sniIface, "ToolTip", sniToolTip{Title: tooltip})
	}
	_ = t.conn.Emit(sniObjectPath, sniIface+".NewTitle")
	_ = t.conn.Emit(sniObjectPath, sniIface+".NewToolTip")
	return nil
}

func (t *sniTray) SetIconRGBA(rgba []byte, width, height int) error {
	if t.closed.Load() {
		return nil
	}
	if len(rgba) < width*height*4 || width <= 0 || height <= 0 {
		return errors.New("некорректная иконка трея")
	}
	pix := []sniPixmap{rgbaToSNIPixmap(rgba, width, height)}
	t.mu.Lock()
	t.pixmap = pix
	t.mu.Unlock()

	if t.sniProps != nil {
		t.sniProps.SetMust(sniIface, "IconPixmap", pix)
	}
	_ = t.conn.Emit(sniObjectPath, sniIface+".NewIcon")
	return nil
}

func (t *sniTray) SetMenu(items []TrayMenuItem) {
	if t.closed.Load() {
		return
	}
	t.mu.Lock()
	t.menuItems = append([]TrayMenuItem(nil), items...)
	t.menuRev++
	rev := t.menuRev
	t.mu.Unlock()

	_ = t.conn.Emit(sniMenuPath, dbusMenuIface+".LayoutUpdated", rev, int32(0))
}

func (t *sniTray) ShowNotification(n TrayNotification) error {
	return sendLinuxNotification(n)
}

func (t *sniTray) Close() error {
	if t.closed.Swap(true) {
		return nil
	}
	if t.conn != nil {
		_, _ = t.conn.ReleaseName(t.serviceName)
		return t.conn.Close()
	}
	return nil
}

// rgbaToSNIPixmap конвертирует RGBA в ARGB32 (network byte order) для StatusNotifierItem.
func rgbaToSNIPixmap(rgba []byte, width, height int) sniPixmap {
	argb := make([]byte, width*height*4)
	for i := 0; i+3 < len(argb) && i+3 < len(rgba); i += 4 {
		argb[i] = rgba[i+3]   // A
		argb[i+1] = rgba[i]   // R
		argb[i+2] = rgba[i+1] // G
		argb[i+3] = rgba[i+2] // B
	}
	return sniPixmap{
		Width:  int32(width),
		Height: int32(height),
		Data:   argb,
	}
}

type sniMenuHandler struct {
	tray *sniTray
}

func (m *sniMenuHandler) GetLayout(parentID int32, recursionDepth int32, propertyNames []string) (uint32, dbusMenuLayout, *dbus.Error) {
	_ = propertyNames
	m.tray.mu.Lock()
	rev := m.tray.menuRev
	items := append([]TrayMenuItem(nil), m.tray.menuItems...)
	m.tray.mu.Unlock()

	if parentID != 0 {
		return rev, dbusMenuLayout{ID: parentID, Props: map[string]dbus.Variant{}}, nil
	}

	var children []dbus.Variant
	if recursionDepth != 0 {
		children = make([]dbus.Variant, 0, len(items))
		for i, item := range items {
			children = append(children, dbus.MakeVariant(dbusMenuLayout{
				ID:    int32(i + 1),
				Props: buildDBusMenuProps(item),
			}))
		}
	}

	root := dbusMenuLayout{
		ID: 0,
		Props: map[string]dbus.Variant{
			"children-display": dbus.MakeVariant("submenu"),
		},
		Children: children,
	}
	return rev, root, nil
}

func (m *sniMenuHandler) GetGroupProperties(ids []int32, propertyNames []string) ([]dbusMenuItemProps, *dbus.Error) {
	_ = propertyNames
	m.tray.mu.Lock()
	items := append([]TrayMenuItem(nil), m.tray.menuItems...)
	m.tray.mu.Unlock()

	res := make([]dbusMenuItemProps, 0, len(ids))
	for _, id := range ids {
		idx := int(id) - 1
		if idx >= 0 && idx < len(items) {
			res = append(res, dbusMenuItemProps{
				ID:    id,
				Props: buildDBusMenuProps(items[idx]),
			})
		}
	}
	return res, nil
}

func (m *sniMenuHandler) GetProperty(id int32, name string) (dbus.Variant, *dbus.Error) {
	m.tray.mu.Lock()
	items := append([]TrayMenuItem(nil), m.tray.menuItems...)
	m.tray.mu.Unlock()

	idx := int(id) - 1
	if idx >= 0 && idx < len(items) {
		props := buildDBusMenuProps(items[idx])
		if v, ok := props[name]; ok {
			return v, nil
		}
	}
	return dbus.MakeVariant(""), nil
}

func (m *sniMenuHandler) Event(id int32, eventID string, data dbus.Variant, timestamp uint32) *dbus.Error {
	_, _ = data, timestamp
	if eventID != "clicked" {
		return nil
	}
	m.tray.mu.Lock()
	var fn func()
	idx := int(id) - 1
	if idx >= 0 && idx < len(m.tray.menuItems) && !m.tray.menuItems[idx].Disabled && !m.tray.menuItems[idx].Separator {
		fn = m.tray.menuItems[idx].OnClick
	}
	m.tray.mu.Unlock()
	if fn != nil {
		go fn()
	}
	return nil
}

func (m *sniMenuHandler) EventGroup(events []dbusMenuEvent) ([]int32, *dbus.Error) {
	var failed []int32
	for _, ev := range events {
		_ = m.Event(ev.ID, ev.EventID, ev.Data, ev.Timestamp)
	}
	return failed, nil
}

func (m *sniMenuHandler) AboutToShow(id int32) (bool, *dbus.Error) {
	_ = id
	return false, nil
}

func (m *sniMenuHandler) AboutToShowGroup(ids []int32) ([]int32, []int32, *dbus.Error) {
	_ = ids
	return nil, nil, nil
}

func buildDBusMenuProps(item TrayMenuItem) map[string]dbus.Variant {
	if item.Separator {
		return map[string]dbus.Variant{
			"type":    dbus.MakeVariant("separator"),
			"visible": dbus.MakeVariant(true),
		}
	}
	props := map[string]dbus.Variant{
		"label":   dbus.MakeVariant(item.Label),
		"enabled": dbus.MakeVariant(!item.Disabled),
		"visible": dbus.MakeVariant(true),
	}
	if item.Checked {
		props["toggle-type"] = dbus.MakeVariant("checkmark")
		props["toggle-state"] = dbus.MakeVariant(int32(1))
	}
	return props
}

