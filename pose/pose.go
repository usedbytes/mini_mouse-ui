package pose

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"time"
	"github.com/usedbytes/bot_matrix/datalink"
)

const Endpoint = 0x1
type PoseReport struct {
	Timestamp time.Time
	Heading float64
}

type Pose struct {}

func (p *Pose) Receive(pkt *datalink.Packet) interface{} {
	if pkt.Endpoint != Endpoint {
		return fmt.Errorf("Invalid endpoint for pose report %d", pkt.Endpoint)
	}

	report := new(PoseReport)
	buf := bytes.NewBuffer(pkt.Data)

	var nsec int64
	binary.Read(buf, binary.LittleEndian, &nsec)
	report.Timestamp = time.Unix(0, nsec)

	binary.Read(buf, binary.LittleEndian, &report.Heading)

	return report
}

func (r *PoseReport) Packet() *datalink.Packet {
	pkt := datalink.Packet{ Endpoint: Endpoint }

	buf := &bytes.Buffer{}
	binary.Write(buf, binary.LittleEndian, r.Timestamp.UnixNano())
	binary.Write(buf, binary.LittleEndian, r.Heading)

	pkt.Data = buf.Bytes()

	return &pkt
}
