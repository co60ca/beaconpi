// Beacon Pi, a edge node system for iBeacons and Edge nodes made of Pi
// Copyright (C) 2017  Maeve Kennedy
// 
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
// 
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
// 
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

//
package beaconpi

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"time"
	"strconv"
	"encoding/hex"
)

const (
	DEFAULT_PORT = "32969"
	MAX_BEACONS  = 256
	MAX_LOGS     = 256
	MAX_CTRL		 = 65535
	CURRENT_VERSION = 0
)

type Uuid [16]byte

func (u Uuid) String() string {
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex.EncodeToString(u[0:4]), hex.EncodeToString(u[4:6]),
		hex.EncodeToString(u[6:8]), hex.EncodeToString(u[8:10]),
		hex.EncodeToString(u[10:16]))
}

type BeaconLog struct {
	Datetime    time.Time
	Rssi        int16
	BeaconIndex uint16
}

type BeaconData struct {
	Uuid  Uuid
	Major uint16
	Minor uint16
}

func (b *BeaconData) String() string {
	return fmt.Sprintf("%s,%d,%d", b.Uuid, b.Major, b.Minor)
}

const (
  VERSION_MASK = 0x0F
  RESPONSE_INVALID = 0x10
	RESPONSE_OK = 0x20
  RESPONSE_TOOMANY = 0x40
  RESPONSE_RESTART = 0x80
  RESPONSE_SHUTDOWN = 0x100
  RESPONSE_UPDATE = 0x200
	RESPONSE_BEACON_UPDATES = 0x400
	RESPONSE_INTERNAL_FAILURE = 0x800
  RESPONSE_SYSTEM = 0x8000
	// Requests have only 0xF0 to work with for flags
	REQUEST_BEACON_UPDATES = 0x10
	REQUEST_CONTROL_LOG = 0x20
	REQUEST_CONTROL_COMPLETE = 0x40
)


type BeaconLogPacket struct {
	Flags uint8
	// Sender uuid
	Uuid    Uuid
	Logs    []BeaconLog
	Beacons []BeaconData
	ControlData		string
}

type BeaconResponsePacket struct {
  Flags uint16
  //LengthData uint32
  Data string
}

func (b *BeaconLog) MarshalBinary() ([]byte, error) {
	outbuff := make([]byte, 12)
	buff := new(bytes.Buffer)
	littleEndianEncode(buff, b.Datetime.UnixNano()/1000)
	copy(outbuff[:8], buff.Bytes()[:8])
	littleEndianEncode(buff, b.Rssi)
	copy(outbuff[8:10], buff.Bytes()[:2])
	littleEndianEncode(buff, b.BeaconIndex)
	copy(outbuff[10:12], buff.Bytes()[:2])
	return outbuff, nil
}

func (b *BeaconData) MarshalBinary() ([]byte, error) {
	outbuff := make([]byte, 20)
	copy(outbuff[0:16], b.Uuid[:])
	buff := new(bytes.Buffer)
	littleEndianEncode(buff, b.Major)
	copy(outbuff[16:18], buff.Bytes()[:2])
	littleEndianEncode(buff, b.Minor)
	copy(outbuff[18:20], buff.Bytes()[:2])
	return outbuff, nil
}

func (b *BeaconLogPacket) MarshalBinary() ([]byte, error) {
	buff := new(bytes.Buffer)
	if len(b.Logs) > MAX_LOGS {
		return nil, errors.New("Protocol limits logs to 256")
	}
	if len(b.Beacons) > MAX_BEACONS {
		return nil, errors.New("Protocol limits beacons to 256")
	}
	if len(b.ControlData) > MAX_CTRL {
		return nil, errors.New("Protocol limits control data to 65535")
	}
	logsb := 12 * len(b.Logs)
	beacb := 20 * len(b.Beacons)
	controldata := len(b.ControlData)

	outbuff := make([]byte, 23+logsb+beacb+controldata)
	pointer := 0

	// 1 byte
	littleEndianEncode(buff, b.Flags)
	copy(outbuff, buff.Bytes()[:1])
	pointer += 1

	// 16 byte
	uuidbuf := b.Uuid[:]
	copy(outbuff[pointer:pointer+16], uuidbuf)
	pointer += 16

	// 2 byte
	littleEndianEncode(buff, uint16(len(b.Beacons)))
	copy(outbuff[pointer:pointer+2], buff.Bytes()[:2])
	pointer += 2
	littleEndianEncode(buff, uint16(len(b.Logs)))
	copy(outbuff[pointer:pointer+2], buff.Bytes()[:2])
	pointer += 2
	littleEndianEncode(buff, uint16(len(b.ControlData)))
	copy(outbuff[pointer:pointer+2], buff.Bytes()[:2])
	pointer += 2

	// Beacons
	for i := range b.Beacons {
		// 20 bytes each
		bdata, _ := b.Beacons[i].MarshalBinary()
		copy(outbuff[pointer:pointer+20], bdata)
		pointer += 20
	}
	// Logs
	for i := range b.Logs {
		// 12 bytes each
		ldata, _ := b.Logs[i].MarshalBinary()
		copy(outbuff[pointer:pointer+12], ldata)
		pointer += 12
	}
	// Control data
	copy(outbuff[pointer:pointer+len(b.ControlData)], []byte(b.ControlData))
	return outbuff, nil
}

func (b *BeaconLog) UnmarshalBinary(data []byte) error {
	if len(data) != 12 {
		return errors.New("Input data buffer not 12 bytes")
	}
	var temptime int64
	if err := littleEndianDecode(data[0:8], &temptime); err != nil {
		return err
	}
	sec := temptime / 1000000
	nsec := (temptime % 1000000) * 1000
	b.Datetime = time.Unix(sec, nsec)
	if err := littleEndianDecode(data[8:10], &b.Rssi); err != nil {
		return err
	}
	if err := littleEndianDecode(data[10:12], &b.BeaconIndex); err != nil {
		return err
	}
	return nil
}

func (b *BeaconData) UnmarshalBinary(data []byte) error {
	if len(data) != 20 {
		return errors.New("Input data buffer not 20 bytes")
	}
	copy(b.Uuid[:], data[0:16])
	if err := littleEndianDecode(data[16:18], &b.Major); err != nil {
		return err
	}
	if err := littleEndianDecode(data[18:20], &b.Minor); err != nil {
		return err
	}
	return nil
}

func (b *BeaconLogPacket) UnmarshalBinary(data []byte) error {
	if len(data) < 23 {
		return errors.New("Packet header too small")
	}
	pointer := 0
	if err := littleEndianDecode(data[pointer:pointer+1], &b.Flags); err != nil {
		return err
	}
	pointer += 1
	// Check for version 1
	if b.Flags & VERSION_MASK != CURRENT_VERSION {
		return errors.New("This version of the library only supports version 0" +
			" of the protocol, a higher version was presented")
	}
	copy(b.Uuid[:], data[pointer:pointer+16])
	pointer += 16
	var nbeacons uint16
	var nlogs uint16
	var ncontrol uint16
	if err := littleEndianDecode(data[pointer:pointer+2], &nbeacons); err != nil {
		return err
	}
	pointer += 2
	if err := littleEndianDecode(data[pointer:pointer+2], &nlogs); err != nil {
		return err
	}
	pointer += 2
	if err := littleEndianDecode(data[pointer:pointer+2], &ncontrol); err != nil {
		return err
	}
	pointer += 2
	// Check if the data remaining will still fit in the slice
	if nlogs > MAX_LOGS {
		return errors.New("Protocol limits logs to 256, sender sent invalid packet")
	}
	if nbeacons > MAX_BEACONS {
		return errors.New("Protocol limits beacons to 256, sender sent invalid packet")
	}
	if ncontrol > MAX_CTRL {
		return errors.New("Protocol limits control messages to 65535, sender sent invalid packet")
	}
	requiredlen := 20*int(nbeacons) + 12*int(nlogs) + int(ncontrol) + 23
	if len(data) < requiredlen {
		// Data is too small
		return errors.New("Input data buffer is too small to support number of beacons and logs")
	} else if len(data) > int(requiredlen) {
		return errors.New("Input data buffer is too long to support number of beacons and logs")
	}
	b.Beacons = make([]BeaconData, nbeacons)
	b.Logs = make([]BeaconLog, nlogs)
	for i := 0; i < int(nbeacons); i++ {
		err := b.Beacons[i].UnmarshalBinary(data[pointer : pointer+20])
		pointer += 20
		if err != nil {
			return fmt.Errorf("Error occured while parsing beacon data: %s", err)
		}
	}

	for i := 0; i < int(nlogs); i++ {
		err := b.Logs[i].UnmarshalBinary(data[pointer : pointer+12])
		pointer += 12
		if err != nil {
			return fmt.Errorf("Error occured while parsing log data: %s", err)
		}
	}
	b.ControlData = string(data[pointer:pointer+int(ncontrol)])
	return nil
}

func (b *BeaconResponsePacket) MarshalBinary() ([]byte, error) {
  bb := new(bytes.Buffer)
  if len(b.Data) > (1 << 30) {
    return []byte{}, errors.New("Data field is too long " + strconv.Itoa(len(b.Data)))
  }
  reqlen := len(b.Data) + 2 + 4
  resp := make([]byte, reqlen)
  littleEndianEncode(bb, b.Flags)
  copy(resp[0:2], bb.Bytes()[0:2])
  littleEndianEncode(bb, uint32(len(b.Data)))
  copy(resp[2:6], bb.Bytes()[0:4])
  copy(resp[6:], []byte(b.Data))

  return resp, nil
}

func (b *BeaconResponsePacket) UnmarshalBinary(d []byte) error {
  if len(d) < 6 {
    return errors.New("Response packet is minimum 6 bytes")
  }
  if err := littleEndianDecode(d[0:2], &b.Flags); err != nil {
    return err
  }
  if b.Flags & VERSION_MASK != CURRENT_VERSION {
    return errors.New("Version of packet is too new, we only support version <= 0")
  }

  var dl uint32
  if err := littleEndianDecode(d[2:6], &dl); err != nil {
    return err
  }
  if len(d) < int(dl) + 6 {
    return errors.New("Response packet is too short given data")
  }
	b.Data = string(d[6:])
	return nil
}

func littleEndianEncode(b *bytes.Buffer, i interface{}) {
	b.Reset()
	err := binary.Write(b, binary.LittleEndian, i)
	if err != nil {
		panic("Failed while encoding to little endian" + err.Error())
	}
}

func littleEndianDecode(b []byte, i interface{}) error {
	bb := bytes.NewBuffer(b)
	return binary.Read(bb, binary.LittleEndian, i)
}
