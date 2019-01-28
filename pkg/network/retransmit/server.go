package retransmit

import (
	"encoding/json"
	"net/http"
)

type HttpServer struct {
	retransmitter *Retransmitter
}

func NewHttpServer(r *Retransmitter) *HttpServer {
	return &HttpServer{
		retransmitter: r,
	}
}

type ActiveConnection struct {
	Addr      string `json:"addr"`
	Direction string `json:"direction"`
}

func (a *HttpServer) ActiveConnections(rw http.ResponseWriter, r *http.Request) {
	var out []ActiveConnection
	addr2peer := a.retransmitter.ActiveConnections()
	addr2peer.Each(func(id string, p *PeerInfo) {
		out = append(out, ActiveConnection{
			Addr:      id,
			Direction: p.Peer.Direction().String(),
		})
	})

	bts, err := json.Marshal(out)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	rw.WriteHeader(http.StatusOK)
	rw.Write(bts)
}

func (a *HttpServer) KnownPeers(rw http.ResponseWriter, r *http.Request) {
	out := a.retransmitter.KnownPeers()
	bts, err := json.Marshal(out)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	rw.WriteHeader(http.StatusOK)
	rw.Write(bts)
}
