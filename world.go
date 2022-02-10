package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"

	"github.com/jackc/pgx/v4"
	"github.com/klauspost/compress/zstd"
)

const MapBlockSize = 16
const MapBlockVolume = MapBlockSize * MapBlockSize * MapBlockSize
const NodeSizeInBytes = 4

type Node struct {
	ID     uint16
	Param1 uint8
	Param2 uint8
}

func readU8(r io.Reader) (uint8, error) {
	var value uint8
	err := binary.Read(r, binary.BigEndian, &value)
	return value, err
}

func readU16(r io.Reader) (uint16, error) {
	var value uint16
	err := binary.Read(r, binary.BigEndian, &value)
	return value, err
}

func readString(r io.Reader) (string, error) {
	length, err := readU16(r)
	if err != nil {
		return "", err
	}

	buf := make([]byte, length)
	_, err = io.ReadFull(r, buf)
	if err != nil {
		return "", err
	}

	return string(buf), nil
}

type MapBlock struct {
	mappings map[uint16]string
	nodeData []byte
}

func DecodeMapBlock(data []byte) (*MapBlock, error) {
	reader := bytes.NewReader(data)

	version, err := readU8(reader)
	if err != nil {
		return nil, err
	}

	if version != 29 {
		return nil, fmt.Errorf("unsupported block version: %v", version)
	}

	z, err := zstd.NewReader(reader)
	if err != nil {
		return nil, err
	}

	data, err = io.ReadAll(z)
	if err != nil {
		return nil, err
	}

	reader = bytes.NewReader(data)

	// Skip:
	// - uint8 flags
	// - uint16 lighting_complete
	// - uint32 timestamp
	// - uint8 mapping version
	_, err = reader.Seek(1+2+4+1, io.SeekCurrent)
	if err != nil {
		return nil, err
	}

	mappingCount, err := readU16(reader)
	if err != nil {
		return nil, err
	}

	mappings := make(map[uint16]string)
	for i := 0; i < int(mappingCount); i++ {
		id, err := readU16(reader)
		if err != nil {
			return nil, err
		}
		name, err := readString(reader)
		if err != nil {
			return nil, err
		}

		mappings[id] = name
	}

	// Skip uint8 contentWidth, uint8 paramsWidth
	_, err = reader.Seek(1+1, io.SeekCurrent)
	if err != nil {
		return nil, err
	}

	nodeData := make([]byte, MapBlockVolume*NodeSizeInBytes)
	_, err = io.ReadFull(reader, nodeData)
	if err != nil {
		return nil, err
	}

	return &MapBlock{
		mappings: mappings,
		nodeData: nodeData,
	}, nil
}

func (b *MapBlock) ResolveName(id uint16) string {
	return b.mappings[id]
}

func (b *MapBlock) GetNode(x, y, z int) Node {
	index := z*MapBlockSize*MapBlockSize + y*MapBlockSize + x
	idHi := uint16(b.nodeData[2*index])
	idLo := uint16(b.nodeData[2*index+1])
	param1 := b.nodeData[2*MapBlockVolume+index]
	param2 := b.nodeData[3*MapBlockVolume+index]
	return Node{
		ID:     (idHi << 8) | idLo,
		Param1: param1,
		Param2: param2,
	}
}

type Backend interface {
	GetBlockData(x, y, z int) ([]byte, error)
	Close()
}

type PgBackend struct {
	conn *pgx.Conn
}

func NewPgBackend(dsn string) (*PgBackend, error) {
	conn, err := pgx.Connect(context.Background(), dsn)
	if err != nil {
		log.Panic(err)
	}
	return &PgBackend{
		conn: conn,
	}, nil
}

func (p *PgBackend) Close() {
	p.conn.Close(context.Background())
}

func (p *PgBackend) GetBlockData(x, y, z int) ([]byte, error) {
	var data []byte
	err := p.conn.QueryRow(context.Background(), "SELECT data FROM blocks WHERE posx=$1 and posy=$2 and posz=$3", x, y, z).Scan(&data)
	if err != nil {
		return []byte{}, err
	}
	return data, nil
}

type World struct {
	backend Backend
}

func NewWorldWithBackend(backend Backend) World {
	return World{
		backend: backend,
	}
}

func (w *World) GetBlock(x, y, z int) (*MapBlock, error) {
	data, err := w.backend.GetBlockData(x, y, z)
	if err != nil {
		return nil, err
	}

	block, err := DecodeMapBlock(data)
	if err != nil {
		return nil, err
	}

	return block, nil
}
