package server

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
)

// MarshalBinary serializes Message to binary format
func (m Message) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Write ID length + ID
	idBytes := []byte(m.ID)
	if err := binary.Write(buf, binary.LittleEndian, uint16(len(idBytes))); err != nil {
		return nil, err
	}

	buf.Write(idBytes)

	// Write NextID length + NextID
	nextIDBytes := []byte(m.NextID)
	if err := binary.Write(buf, binary.LittleEndian, uint16(len(nextIDBytes))); err != nil {
		return nil, err
	}

	buf.Write(nextIDBytes)

	// Write Type length + Type
	typeBytes := []byte(m.MType)
	if err := binary.Write(buf, binary.LittleEndian, uint16(len(typeBytes))); err != nil {
		return nil, err
	}

	buf.Write(typeBytes)

	// Write User length + User
	userBytes := []byte(m.User)
	if err := binary.Write(buf, binary.LittleEndian, uint16(len(userBytes))); err != nil {
		return nil, err
	}

	buf.Write(userBytes)

	// Write Password length + Password
	passwordBytes := []byte(m.Password)
	if err := binary.Write(buf, binary.LittleEndian, uint16(len(passwordBytes))); err != nil {
		return nil, err
	}

	buf.Write(passwordBytes)

	// Write Topic name length + Topic name
	topicBytes := []byte(m.Topic.Name)
	if err := binary.Write(buf, binary.LittleEndian, uint16(len(topicBytes))); err != nil {
		return nil, err
	}

	buf.Write(topicBytes)

	// Write Body length + Body
	bodyBytes := []byte(m.Body)
	if err := binary.Write(buf, binary.LittleEndian, uint32(len(bodyBytes))); err != nil {
		return nil, err
	}

	buf.Write(bodyBytes)

	// Write BodyString length + BodyString
	bodyStringBytes := []byte(m.BodyString)
	if err := binary.Write(buf, binary.LittleEndian, uint32(len(bodyStringBytes))); err != nil {
		return nil, err
	}

	buf.Write(bodyStringBytes)

	// Write Timestamp (8 bytes)
	if err := binary.Write(buf, binary.LittleEndian, m.Timestamp); err != nil {
		return nil, err
	}

	// Write ACK (1 byte)
	ackByte := byte(0)
	if m.ACK {
		ackByte = 1
	}

	if err := binary.Write(buf, binary.LittleEndian, ackByte); err != nil {
		return nil, err
	}

	// Write Attempts (4 bytes)
	if err := binary.Write(buf, binary.LittleEndian, int32(m.Attempts)); err != nil {
		return nil, err
	}

	if err := binary.Write(buf, binary.LittleEndian, m.TTL); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// UnmarshalBinary deserializes binary data into Message
func UnmarshalBinary(data []byte, m *Message) error {
	buf := bytes.NewReader(data)

	// Read ID
	var idLen uint16
	if err := binary.Read(buf, binary.LittleEndian, &idLen); err != nil {
		return err
	}
	idBytes := make([]byte, idLen)
	if _, err := io.ReadFull(buf, idBytes); err != nil {
		return err
	}
	m.ID = string(idBytes)

	// Read NextID
	var nextIDLen uint16
	if err := binary.Read(buf, binary.LittleEndian, &nextIDLen); err != nil {
		return err
	}
	nextIDBytes := make([]byte, nextIDLen)
	if _, err := io.ReadFull(buf, nextIDBytes); err != nil {
		return err
	}
	m.NextID = string(nextIDBytes)

	// Read Format
	var typeLen uint16
	if err := binary.Read(buf, binary.LittleEndian, &typeLen); err != nil {
		return err
	}
	typeBytes := make([]byte, typeLen)
	if _, err := io.ReadFull(buf, typeBytes); err != nil {
		return err
	}
	m.MType = MType(typeBytes)

	// Read User
	var userLen uint16
	if err := binary.Read(buf, binary.LittleEndian, &userLen); err != nil {
		return err
	}
	userBytes := make([]byte, userLen)
	if _, err := io.ReadFull(buf, userBytes); err != nil {
		return err
	}
	m.User = string(userBytes)

	// Read Password
	var passwordLen uint16
	if err := binary.Read(buf, binary.LittleEndian, &passwordLen); err != nil {
		return err
	}
	passwordBytes := make([]byte, passwordLen)
	if _, err := io.ReadFull(buf, passwordBytes); err != nil {
		return err
	}
	m.Password = string(passwordBytes)

	// Read Topic
	var topicLen uint16
	if err := binary.Read(buf, binary.LittleEndian, &topicLen); err != nil {
		return err
	}
	topicBytes := make([]byte, topicLen)
	if _, err := io.ReadFull(buf, topicBytes); err != nil {
		return err
	}
	m.Topic = Topic{Name: string(topicBytes)}

	// Read Body
	var bodyLen uint32
	if err := binary.Read(buf, binary.LittleEndian, &bodyLen); err != nil {
		return err
	}
	bodyBytes := make([]byte, bodyLen)
	if _, err := io.ReadFull(buf, bodyBytes); err != nil {
		return err
	}
	m.Body = json.RawMessage(bodyBytes)

	// Read BodyString
	var bodyStringLen uint32
	if err := binary.Read(buf, binary.LittleEndian, &bodyStringLen); err != nil {
		return err
	}
	bodyStringBytes := make([]byte, bodyStringLen)
	if _, err := io.ReadFull(buf, bodyStringBytes); err != nil {
		return err
	}
	m.BodyString = string(bodyStringBytes)

	// Read Timestamp
	if err := binary.Read(buf, binary.LittleEndian, &m.Timestamp); err != nil {
		return err
	}

	// Read ACK
	var ackByte byte
	if err := binary.Read(buf, binary.LittleEndian, &ackByte); err != nil {
		return err
	}
	m.ACK = ackByte == 1

	// Read Attempts
	var attempts int32
	if err := binary.Read(buf, binary.LittleEndian, &attempts); err != nil {
		return err
	}
	m.Attempts = int(attempts)

	return nil
}
