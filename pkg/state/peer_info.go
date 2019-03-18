package state

//import (
//	"encoding/binary"
//	"github.com/wavesplatform/gowaves/pkg/proto"
//	. "github.com/wavesplatform/gowaves/pkg/util/collect_writes"
//	"io"
//	"net"
//	"time"
//)
//
//type peerInfoKey [13]byte
//
//type PeerInfo struct {
//	IP            net.IP
//	Nonce         uint64
//	Port          uint16
//	Name          string
//	LastConnected time.Time
//}
//
//func (a *PeerInfo) key() peerInfoKey {
//	k := peerInfoKey{}
//	k[0] = knownPeersPrefix
//	copy(k[1:5], a.IP.To4())
//	binary.BigEndian.PutUint64(k[5:], a.Nonce)
//	return k
//}
//
//func (a *PeerInfo) WriteTo(w io.Writer) (int, error) {
//	k := a.key()
//	c := new(CollectInt)
//	c.W(w.Write(k[1:]))
//
//	b16 := make([]byte, 2)
//	binary.BigEndian.PutUint16(b16, a.Port)
//	c.W(w.Write(b16))
//
//	c.W(proto.U8String{S: a.Name}.WriteTo(w))
//
//	b64 := make([]byte, 8)
//	v := uint64(a.LastConnected.Unix())
//	binary.BigEndian.PutUint64(b64, v)
//	c.W(w.Write(b64))
//
//	return c.Ret()
//}
//
//func (a *PeerInfo) ReadFrom(r io.Reader) error {
//	ip := make([]byte, 4)
//	_, err := r.Read(ip)
//	if err != nil {
//		return err
//	}
//	a.IP = net.IP(ip)
//
//	nonce := make([]byte, 8)
//	_, err = r.Read(nonce)
//	if err != nil {
//		return err
//	}
//	a.Nonce = binary.BigEndian.Uint64(nonce)
//
//	port := make([]byte, 2)
//	_, err = r.Read(port)
//	if err != nil {
//		return err
//	}
//	a.Port = binary.BigEndian.Uint16(port)
//
//	name := proto.U8String{}
//	err = name.ReadFrom(r)
//	if err != nil {
//		return err
//	}
//	a.Name = name.S
//
//	unix := make([]byte, 8)
//	_, err = r.Read(unix)
//	v := binary.BigEndian.Uint64(unix)
//	a.LastConnected = time.Unix(int64(v), 0)
//
//	return nil
//}
