package turn_server

import (
	"github.com/pion/turn/v2"
	"net"
	"strconv"
)

type Server struct {
	udpListener net.PacketConn
	turnServer  *turn.Server
	AuthHandler func(userName, realm string, srcAddr net.Addr) (string, bool)
	PublicIP    string
	Port        int
}

func NewTurnServer(publicIp, realm string, port int) (*Server, error) {
	server := &Server{PublicIP: publicIp, Port: port}

	udpListener, err := net.ListenPacket("udp4", "0.0.0.0:"+strconv.Itoa(port))
	if err != nil {
		return nil, err
	}
	server.udpListener = udpListener

	turnServer, err := turn.NewServer(turn.ServerConfig{
		PacketConnConfigs: []turn.PacketConnConfig{
			{
				PacketConn: udpListener,
				RelayAddressGenerator: &turn.RelayAddressGeneratorStatic{
					RelayAddress: net.ParseIP(publicIp),
					Address:      "0.0.0.0",
				},
			},
		},
		Realm:       realm,
		AuthHandler: server.HandlerAuthenticate,
	})
	if err != nil {
		return nil, err
	}
	server.turnServer = turnServer
	return server, nil
}

func (s *Server) HandlerAuthenticate(userName, realm string, srcAddr net.Addr) ([]byte, bool) {
	if s.AuthHandler != nil {
		if password, ok := s.AuthHandler(userName, realm, srcAddr); ok {
			return turn.GenerateAuthKey(userName, realm, password), true
		}
	}
	return nil, false
}

func (s *Server) Close() error {
	return s.turnServer.Close()
}
