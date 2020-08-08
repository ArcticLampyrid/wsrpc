package wsrpc_test

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/1354092549/wsrpc"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var rpcServer *wsrpc.WebsocketRPC

type addArgs struct {
	A int `json:"a"`
	B int `json:"b"`
}
type addReply struct {
	Result int `json:"result"`
}

func rpcMethodAdd(a int, b int) int {
	return a + b
}

func rpcMethodHello(rpcConn *wsrpc.WebsocketRPCConn, name string) (string, string, error) {
	v, ok := rpcConn.Session["foo"].(string)
	if !ok {
		return "", "", errors.New("internal error")
	}
	return v, name, nil
}

type welcomeArgs struct {
	Name string `json:"name"`
}

type welcomeReply struct {
	Message string `json:"message"`
}

func rpcMethodWelcome(rpcConn *wsrpc.WebsocketRPCConn, args welcomeArgs, reply *welcomeReply) error {
	*reply = welcomeReply{Message: "Welcome, " + args.Name}
	return nil
}

func newRPCServer() *wsrpc.WebsocketRPC {
	rpcServer := wsrpc.NewWebsocketRPC()
	rpcServer.Register("add", rpcMethodAdd, []string{"a", "b"}, []string{"result"})
	rpcServer.Register("hello", rpcMethodHello, []string{"name"}, nil)
	rpcServer.RegisterExplicitly("welcome", rpcMethodWelcome)
	return rpcServer
}

func TestWebsocketRPC(t *testing.T) {
	go func() {
		rpcServer = newRPCServer()
		http.HandleFunc("/", wsHandler)
		err := http.ListenAndServe("localhost:7575", nil)
		if err != nil {
			t.Error(err)
		}
	}()
	//Wait for the server starting
	time.Sleep(time.Duration(2) * time.Second)
	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:7575/", nil)
	if err != nil {
		t.Error(err)
	}
	rpcClient := wsrpc.NewWebsocketRPC()
	rpcConn := rpcClient.Connect(conn)
	go func() {
		rpcConn.ServeConn()
	}()
	var addResult addReply
	err = rpcConn.CallExplicitly("add", addArgs{A: 1, B: 2}, &addResult)
	if err != nil {
		t.Error(err)
	}
	if addResult.Result != 3 {
		t.Error("expected 3 but got " + string(addResult.Result))
	}
	var hello func(name string) (string, string, error)
	rpcConn.MakeCall("hello", &hello, nil, nil)
	helloA, helloB, err := hello("wsrpc")
	if err != nil {
		t.Error(err)
	}
	if helloA != "Hello" {
		t.Error("expected \"Hello\" but got \"" + helloA + "\"")
	}
	if helloB != "wsrpc" {
		t.Error("expected \"wsrpc\" but got \"" + helloB + "\"")
	}

	var welcome func(name string) string
	rpcConn.MakeCall("welcome", &welcome, []string{"name"}, []string{"message"})
	welcomeResult := welcome("wsrpc")
	if welcomeResult != "Welcome, wsrpc" {
		t.Error("expected \"Welcome, wsrpc\" but got \"" + welcomeResult + "\"")
	}
}

func wsHandler(writer http.ResponseWriter, request *http.Request) {
	c, err := upgrader.Upgrade(writer, request, nil)
	if err != nil {
		return
	}
	defer c.Close()
	rpcConn := rpcServer.Connect(c)
	rpcConn.Session["foo"] = "Hello"
	rpcConn.ServeConn()
}
