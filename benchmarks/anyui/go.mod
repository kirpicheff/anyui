module bench_anyui

go 1.26.2

require git.gtw.su/kirpich/anyui v0.0.0

require (
	github.com/godbus/dbus/v5 v5.2.4 // indirect
	github.com/jezek/xgb v1.3.1 // indirect
	golang.org/x/image v0.48.0 // indirect
	golang.org/x/sys v0.50.0 // indirect
	golang.org/x/text v0.44.0 // indirect
)

replace git.gtw.su/kirpich/anyui => ../..
