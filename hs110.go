package hs1xxplug

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

type Hs1xxPlug struct {
	IPAddress string
}

func (p *Hs1xxPlug) TurnOn() (err error) {
	json := `{"system":{"set_relay_state":{"state":1}}}`
	data := encrypt(json)
	_, err = send(p.IPAddress, data)
	return
}

func (p *Hs1xxPlug) TurnOff() (err error) {
	json := `{"system":{"set_relay_state":{"state":0}}}`
	data := encrypt(json)
	_, err = send(p.IPAddress, data)
	return
}

func (p *Hs1xxPlug) SystemInfo() (results string, err error) {
	json := `{"system":{"get_sysinfo":{}}}`
	data := encrypt(json)
	reading, err := send(p.IPAddress, data)
	if err == nil {
		results = decrypt(reading[4:])
	}
	return
}

func (p *Hs1xxPlug) MeterInfo() (results string, err error) {
	json := `{"system":{"get_sysinfo":{}}, "emeter":{"get_realtime":{},"get_vgain_igain":{}}}`
	data := encrypt(json)
	reading, err := send(p.IPAddress, data)
	if err == nil {
		results = decrypt(reading[4:])
	}
	return
}

func (p *Hs1xxPlug) DailyStats(month int, year int) (results string, err error) {
	json := fmt.Sprintf(`{"emeter":{"get_daystat":{"month":%d,"year":%d}}}`, month, year)
	data := encrypt(json)
	reading, err := send(p.IPAddress, data)
	if err == nil {
		results = decrypt(reading[4:])
	}
	return
}

func encrypt(plaintext string) []byte {
	n := len(plaintext)
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, uint32(n))
	ciphertext := []byte(buf.Bytes())

	key := byte(0xAB)
	payload := make([]byte, n)
	for i := 0; i < n; i++ {
		payload[i] = plaintext[i] ^ key
		key = payload[i]
	}

	for i := 0; i < len(payload); i++ {
		ciphertext = append(ciphertext, payload[i])
	}

	return ciphertext
}

func decrypt(ciphertext []byte) string {
	n := len(ciphertext)
	key := byte(0xAB)
	var nextKey byte
	for i := 0; i < n; i++ {
		nextKey = ciphertext[i]
		ciphertext[i] = ciphertext[i] ^ key
		key = nextKey
	}
	return string(ciphertext)
}


func readExactly(conn net.Conn, data []byte) (err error) {
	for rd:=0; rd<len(data); {
		sz,err := conn.Read(data[rd:])
		if err != nil {
			 return err
		}
		rd += sz
		if sz == 0 {
			err = fmt.Errorf("read 0 bytes after %d/%d", rd, len(data))
			return err
		}
	}
	return
}

func send(ip string, payload []byte) (data []byte, err error) {
	// 10 second timeout
	conn, err := net.DialTimeout("tcp", ip+":9999", time.Duration(10)*time.Second)
	if err != nil {
		fmt.Println("Cannot connnect to plug:", err)
		data = nil
		return
	}
	_, err = conn.Write(payload)

	/*
	Changed by comzine
	Thanks to mikemrm (https://github.com/mikemrm/Go-TPLink-SmartPlug)
	 */
	buff := make([]byte, 2048)
	n, err := conn.Read(buff)
	data = buff[:n]

	if err != nil {
		fmt.Println("Cannot read data (size) from plug:", err)
		return
	}
	size := binary.BigEndian.Uint32(data[0:4])
	data = data[0:(4+size)]
	err = readExactly(conn, data[4:])
        if err != nil {
                fmt.Println("Cannot read data (",size," bytes) from plug:", err)
        }
	// and don't leave the connection open!
	_ = conn.Close()

	return
}

/**
Returns the current power consumption of the device
Created by comzine
 */
func (p *Hs1xxPlug) GetPowerConsumption() (float64, error) {
	m := make(map[string]interface{})
	s, err := p.MeterInfo()
	if err != nil {
		return 0, err
	}
	err = json.Unmarshal([]byte(s), &m)
	if err != nil {
		return 0, err
	}
	//fmt.Println(m)
	if power, ok :=m["emeter"].(map[string]interface {})["get_realtime"].(map[string]interface {})["power_mw"].(float64); ok {
		return power, err
	} else {
		return 0, err
	}
}

/**
Returns the alias of the device
Created by comzine
*/
func (p *Hs1xxPlug) GetAliasName() (string, error) {
	m := make(map[string]interface{})
	s, err := p.SystemInfo()
	if err != nil {
		return "", err
	}
	err = json.Unmarshal([]byte(s), &m)
	if err != nil {
		return "", err
	}
	//fmt.Println(m)
	if alias, ok :=m["system"].(map[string]interface {})["get_sysinfo"].(map[string]interface {})["alias"].(string); ok {
		return alias, err
	} else {
		return "", err
	}
}
