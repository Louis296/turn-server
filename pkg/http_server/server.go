package http_server

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/louis296/turn-server/pkg/turn_server"
	"github.com/louis296/turn-server/pkg/util"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const sharedKey = "webrtc-turn-shared-key"

type TurnCredential struct {
	UserName string
	Password string
	TTL      int
	Uris     []string
}

type HttpServer struct {
	turn       *turn_server.Server
	expiredMap *util.ExpiredMap
}

func NewHttpServer(turn *turn_server.Server) *HttpServer {
	server := &HttpServer{
		turn:       turn,
		expiredMap: util.NewExpiredMap(),
	}
	server.turn.AuthHandler = server.authHandler
	return server
}

func (s *HttpServer) Bind(turnServerPath, host string, port int) {
	http.HandleFunc(turnServerPath, s.HandleTurnServerCredentials)
	panic(http.ListenAndServe(host+":"+strconv.Itoa(port), nil))
}

func (s *HttpServer) authHandler(userName, realm string, srcAddr net.Addr) (string, bool) {
	if info, ok := s.expiredMap.Get(userName); ok {
		crendendtial := info.(TurnCredential)
		return crendendtial.Password, true
	}
	return "", false
}

func (s *HttpServer) HandleTurnServerCredentials(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	writer.Header().Set("Access-Control-Allow-Origin", "*")

	params, err := url.ParseQuery(request.URL.RawQuery)
	if err != nil {
		return
	}
	service := params["service"][0]
	if service != "turn" {
		return
	}
	userName := params["username"][0]

	timeStamp := time.Now().Unix()
	turnUserName := fmt.Sprintf("%d:%s", timeStamp, userName)

	h := hmac.New(sha1.New, []byte(sharedKey))
	h.Write([]byte(turnUserName))

	turnPassword := base64.RawStdEncoding.EncodeToString(h.Sum(nil))

	ttl := 86400
	host := fmt.Sprintf("%s:%d", s.turn.PublicIP, s.turn.Port)
	credential := TurnCredential{
		UserName: turnUserName,
		Password: turnPassword,
		TTL:      ttl,
		Uris: []string{
			"turn:" + host + "?transport=udp",
		},
	}
	s.expiredMap.Set(turnUserName, credential, int64(ttl))
	json.NewEncoder(writer).Encode(credential)
}
