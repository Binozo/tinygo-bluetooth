package main

// This example implements a NUS (Nordic UART Service) peripheral.
// I can't find much official documentation on the protocol, but this can be
// helpful:
// https://learn.adafruit.com/introducing-adafruit-ble-bluetooth-low-energy-friend/uart-service
//
// Code to interact with a raw terminal is in separate files with build tags.

import (
	"time"

	"github.com/tinygo-org/bluetooth"
)

var (
	serviceUUID = bluetooth.NewUUID([16]byte{0x6E, 0x40, 0x00, 0x01, 0xB5, 0xA3, 0xF3, 0x93, 0xE0, 0xA9, 0xE5, 0x0E, 0x24, 0xDC, 0xCA, 0x9E})
	rxUUID      = bluetooth.NewUUID([16]byte{0x6E, 0x40, 0x00, 0x02, 0xB5, 0xA3, 0xF3, 0x93, 0xE0, 0xA9, 0xE5, 0x0E, 0x24, 0xDC, 0xCA, 0x9E})
	txUUID      = bluetooth.NewUUID([16]byte{0x6E, 0x40, 0x00, 0x03, 0xB5, 0xA3, 0xF3, 0x93, 0xE0, 0xA9, 0xE5, 0x0E, 0x24, 0xDC, 0xCA, 0x9E})
)

func main() {
	println("starting")
	adapter := bluetooth.DefaultAdapter
	must("enable BLE stack", adapter.Enable())
	adv := adapter.DefaultAdvertisement()
	must("config adv", adv.Configure(bluetooth.AdvertisementOptions{
		LocalName:    "NUS", // Nordic UART Service
		ServiceUUIDs: []bluetooth.UUID{serviceUUID},
		Interval:     bluetooth.NewAdvertisementInterval(100 * time.Millisecond),
	}))
	must("start adv", adv.Start())

	var rxChar bluetooth.Characteristic
	var txChar bluetooth.Characteristic
	must("add service", adapter.AddService(&bluetooth.Service{
		UUID: serviceUUID,
		Characteristics: []bluetooth.CharacteristicConfig{
			{
				Handle: &rxChar,
				UUID:   rxUUID,
				Flags:  bluetooth.CharacteristicWritePermission | bluetooth.CharacteristicWriteWithoutResponsePermission,
				WriteEvent: func(client bluetooth.Connection, offset int, value []byte) {
					for _, c := range value {
						// TODO: echo these characters back.
						putchar(c)
						if c == '\r' {
							putchar('\n')
						}
					}
				},
			},
			{
				Handle: &txChar,
				UUID:   txUUID,
				Flags:  bluetooth.CharacteristicNotifyPermission | bluetooth.CharacteristicReadPermission,
			},
		},
	}))

	initTerminal()
	defer restoreTerminal()
	print("NUS console enabled, use Ctrl-X to exit\r\n")
	var line []byte
	for {
		ch := getchar()
		putchar(ch)
		line = append(line, ch)

		// Send the current line to the central.
		if ch == '\x18' {
			// The user pressed Ctrl-X, exit the terminal.
			break
		} else if ch == '\r' {
			// Send another newline (consoles only seem to receive CR chars).
			putchar('\n')
			line = append(line, '\n')

			sendbuf := line // copy buffer
			// Reset the slice while keeping the buffer in place.
			line = line[:0]

			// Send the sendbuf after breaking it up in pieces.
			for len(sendbuf) != 0 {
				// Chop off up to 20 bytes from the sendbuf.
				partlen := 20
				if len(sendbuf) < 20 {
					partlen = len(sendbuf)
				}
				part := sendbuf[:partlen]
				sendbuf = sendbuf[partlen:]
				// This also sends a notification.
				_, err := txChar.Write(part)
				must("send notification", err)
			}
		}
	}
}

func must(action string, err error) {
	if err != nil {
		panic("failed to " + action + ": " + err.Error())
	}
}