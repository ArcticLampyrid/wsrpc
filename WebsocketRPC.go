package wsrpc

import (
	"encoding/json"
	"errors"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

//LowLevelRPCMethod is an RPC method that send and receive raw messages
type LowLevelRPCMethod func(rpcConn *WebsocketRPCConn, arg json.RawMessage, reply *json.RawMessage) error

type rpcMessage struct {
	ID json.RawMessage `json:"id,omitempty"`

	JSONRPC string          `json:"jsonrpc"`
	Method  *string         `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`

	Result json.RawMessage `json:"result,omitempty"`
	Error  *RPCErrorInfo   `json:"error,omitempty"`
}

// WebsocketRPC represents an RPC service that run over websocket
type WebsocketRPC struct {
	method map[string]LowLevelRPCMethod
}

// WebsocketRPCConn represents an RPC connection to WebsocketRPC
type WebsocketRPCConn struct {
	//RPC is a pointer to the RPC service
	RPC *WebsocketRPC
	//Session saves the user defined session data
	Session map[string]interface{}
	//Timeout sets the time to wait for a response, default is 10 seconds
	Timeout time.Duration
	adapter MessageAdapter
	seq     uint64
	pending sync.Map
}

var typeOfPointToRPCConn = reflect.TypeOf((*WebsocketRPCConn)(nil))

// NewWebsocketRPC will create a websocket rpc object.
func NewWebsocketRPC() *WebsocketRPC {
	r := new(WebsocketRPC)
	r.method = make(map[string]LowLevelRPCMethod)
	return r
}

func (rpcConn *WebsocketRPCConn) allocRequestSeq(done chan *rpcMessage) json.RawMessage {
	seq := atomic.AddUint64(&rpcConn.seq, 1)
	rpcConn.pending.Store(seq, done)
	seqBytes, _ := json.Marshal(seq)
	seqRaw := json.RawMessage(seqBytes)
	return seqRaw
}

// ToRPCError is a helper function to convert error to RPCErrorInfo.
// If err is a RPCErrorInfo, then return it.
// If not, then create a RPCErrorInfo with Message = err.Error()
func ToRPCError(err error) RPCErrorInfo {
	var rpcErr RPCErrorInfo
	rpcErr, isRPCErr := err.(RPCErrorInfo)
	if !isRPCErr {
		rpcErr = RPCErrorInfo{
			Code:    -32000,
			Message: err.Error()}
	}
	return rpcErr
}

func (rpcConn *WebsocketRPCConn) processRequest(msg rpcMessage) *rpcMessage {
	if msg.Method == nil {
		return &rpcMessage{
			JSONRPC: "2.0",
			ID:      jsonNullValue,
			Error:   &RPCInvalidRequestError}
	}
	method, methodExists := rpcConn.RPC.method[*msg.Method]
	if !methodExists {
		if msg.ID == nil {
			return nil
		}
		return &rpcMessage{
			JSONRPC: "2.0",
			ID:      msg.ID,
			Error:   &RPCMothedNotFoundError}
	}
	result := jsonNullValue
	err := method(rpcConn, msg.Params, &result)
	if msg.ID == nil {
		return nil
	}
	if err != nil {
		rpcError := ToRPCError(err)
		return &rpcMessage{
			JSONRPC: "2.0",
			ID:      msg.ID,
			Error:   &rpcError}
	}
	return &rpcMessage{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Result:  result}
}

func (rpcConn *WebsocketRPCConn) processResponse(msg rpcMessage) {
	if msg.ID != nil {
		var seq uint64
		err := json.Unmarshal(msg.ID, &seq)
		if err == nil {
			if done, ok := rpcConn.pending.Load(seq); ok {
				rpcConn.pending.Delete(seq)
				doneChan, _ := done.(chan *rpcMessage)
				doneChan <- &msg
			}
		}
	}
}

func (rpcConn *WebsocketRPCConn) processMessage(rawMsg []byte) {
	var msgs []rpcMessage
	var err error
	responseInArray := IsJSONArray(rawMsg)
	if responseInArray {
		err = json.Unmarshal(rawMsg, &msgs)
	} else {
		msgs = make([]rpcMessage, 1)
		err = json.Unmarshal(rawMsg, &msgs[0])
	}
	var responses []*rpcMessage
	nResponse := 0
	if err != nil {
		responses = make([]*rpcMessage, 1)
		nResponse = 1
		responseInArray = false
		responses[0] = &rpcMessage{
			JSONRPC: "2.0",
			ID:      jsonNullValue,
			Error:   &RPCParseError}
	} else if len(msgs) == 0 {
		responses = make([]*rpcMessage, 1)
		nResponse = 1
		responseInArray = false
		responses[0] = &rpcMessage{
			JSONRPC: "2.0",
			ID:      jsonNullValue,
			Error:   &RPCInvalidRequestError}
	} else {
		responses = make([]*rpcMessage, len(msgs))
		for _, msg := range msgs {
			switch {
			case msg.Result != nil || msg.Error != nil:
				rpcConn.processResponse(msg)
			default:
				responses[nResponse] = rpcConn.processRequest(msg)
				if responses[nResponse] != nil {
					nResponse++
				}
			}
		}
	}
	if nResponse == 0 {
		return
	}
	var resultBytes []byte
	if responseInArray {
		resultBytes, err = json.Marshal(responses[:nResponse])
	} else {
		resultBytes, err = json.Marshal(responses[0])
	}
	if err != nil {
		return
	}
	_ = rpcConn.adapter.WriteMessage(resultBytes)
}

// MakeCall is used to make a proxy (as a normal function) to a remote procedure.
// About inName and outName, you can see details in Register.
func (rpcConn *WebsocketRPCConn) MakeCall(name string, fptr interface{}, inName []string, outName []string) {
	fobj := reflect.ValueOf(fptr).Elem()
	fType := fobj.Type()
	inConverter := newValueToJSONConverter(getAllInParamInfo(fType), inName, false)
	outParamInfo := getAllOutParamInfo(fType)
	nOut := len(outParamInfo)
	hasErrInfo := false
	if nOut > 0 && outParamInfo[nOut-1] == typeOfError {
		hasErrInfo = true
		nOut--
		outParamInfo = outParamInfo[:nOut]
	}
	outConverter := newJSONToValueConverter(outParamInfo, nil, outName)
	makeErrorResult := func(err error) []reflect.Value {
		if !hasErrInfo {
			panic(err)
		}
		result := make([]reflect.Value, nOut+1)
		for i := 0; i < nOut; i++ {
			pType := outParamInfo[i]
			if pType.Kind() == reflect.Ptr {
				result[i] = reflect.New(pType.Elem())
			} else {
				result[i] = reflect.New(pType).Elem()
			}
		}
		result[nOut] = reflect.ValueOf(err)
		return result
	}
	processorFunc := func(in []reflect.Value) []reflect.Value {
		var err error
		argsRaw, err := inConverter.tryRun(in)
		if err != nil {
			return makeErrorResult(err)
		}
		var replyRaw json.RawMessage
		err = rpcConn.CallLowLevel(name, argsRaw, &replyRaw)
		if err != nil {
			return makeErrorResult(err)
		}
		reply, err := outConverter.tryRun(nil, replyRaw)
		if err != nil {
			return makeErrorResult(err)
		}
		if hasErrInfo {
			reply = append(reply, reflect.Zero(reflect.TypeOf((*error)(nil)).Elem()))
		}
		return reply
	}
	v := reflect.MakeFunc(fType, processorFunc)
	fobj.Set(v)
}

// CallExplicitly provides a `net/rpc`-like way to call a remote procedure.
// In this way, the struct is defined explicitly by the caller
func (rpcConn *WebsocketRPCConn) CallExplicitly(name string, params interface{}, reply interface{}) error {
	paramBytes, err := json.Marshal(params)
	if err != nil {
		return err
	}
	rawParam := json.RawMessage(paramBytes)
	var rawReply json.RawMessage
	err = rpcConn.CallLowLevel(name, rawParam, &rawReply)
	if err != nil {
		return err
	}
	err = json.Unmarshal(rawReply, reply)
	return err
}

// CallLowLevel is used to call a remote rrocedure in low-level way (use json.RawMessage).
func (rpcConn *WebsocketRPCConn) CallLowLevel(name string, params json.RawMessage, reply *json.RawMessage) error {
	msg := rpcMessage{
		JSONRPC: "2.0",
		Method:  &name,
		Params:  params}
	done := make(chan *rpcMessage, 1)
	msg.ID = rpcConn.allocRequestSeq(done)
	resultBytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	err = rpcConn.adapter.WriteMessage(resultBytes)
	if err != nil {
		return err
	}
	var r *rpcMessage
	timer := time.NewTimer(rpcConn.Timeout)
	select {
	case r = <-done:
		timer.Stop()
	case <-timer.C:
		return errors.New("RPC call timed out")
	}
	if r.Error != nil {
		return r.Error
	}
	if reply != nil {
		*reply = r.Result
	}
	return nil
}

// MakeNotify is used to make a proxy (as a normal function) to send a notification.
// About inName, you can see details in Register.
func (rpcConn *WebsocketRPCConn) MakeNotify(name string, fptr interface{}, inName []string) {
	fobj := reflect.ValueOf(fptr).Elem()
	fType := fobj.Type()
	inConverter := newValueToJSONConverter(getAllInParamInfo(fType), inName, false)
	nOut := fType.NumOut()
	hasErrInfo := false
	switch nOut {
	case 0:
		break
	case 1:
		if fType.Out(0) == typeOfError {
			hasErrInfo = true
			nOut--
		} else {
			panic(errors.New("the function must have no return value or return an error"))
		}
	}
	makeErrorResult := func(err error) []reflect.Value {
		if !hasErrInfo {
			panic(err)
		}
		return []reflect.Value{reflect.ValueOf(err)}
	}
	processorFunc := func(in []reflect.Value) []reflect.Value {
		var err error
		argsRaw, err := inConverter.tryRun(in)
		if err != nil {
			return makeErrorResult(err)
		}
		err = rpcConn.NotifyLowLevel(name, argsRaw)
		if err != nil {
			return makeErrorResult(err)
		}
		if hasErrInfo {
			return []reflect.Value{reflect.Zero(reflect.TypeOf((*error)(nil)).Elem())}
		}
		return []reflect.Value{}
	}
	v := reflect.MakeFunc(fType, processorFunc)
	fobj.Set(v)
}

// NotifyExplicitly provides a `net/rpc`-like way to send a notification.
// In this way, the struct is defined explicitly by the caller
func (rpcConn *WebsocketRPCConn) NotifyExplicitly(name string, params interface{}) error {
	paramBytes, err := json.Marshal(params)
	if err != nil {
		return err
	}
	rawParam := json.RawMessage(paramBytes)
	err = rpcConn.NotifyLowLevel(name, rawParam)
	return err
}

// NotifyLowLevel is used to send a notification in low-level way (use json.RawMessage).
func (rpcConn *WebsocketRPCConn) NotifyLowLevel(name string, params json.RawMessage) error {
	msg := rpcMessage{
		JSONRPC: "2.0",
		Method:  &name,
		Params:  params}
	resultBytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	err = rpcConn.adapter.WriteMessage(resultBytes)
	return err
}

// Register is used to register a normal function for RPC.
//
// The function can have a pointer argument to receive RPC connection object
// (optional, must be the first in argument, do not provide name for this argument).
// The function can also have an error return value. (optional, must be the last out argument,
// do not provide name for this argument)
//
// If inName is nil, then you can only call this function by postion (in JSON array).
// If outName is not nil, then it will return result by name (in JSON object).
// If inName/outName is not nil, the length of them must be equal to the number of in/out arguments.
// (not including special params described above, of course)
func (rpc *WebsocketRPC) Register(name string, fobj interface{}, inName []string, outName []string) {
	if fobj == nil {
		return
	}
	fType := reflect.TypeOf(fobj)
	fValue := reflect.ValueOf(fobj)
	inConverter := newJSONToValueConverter(getAllInParamInfo(fType), typeOfPointToRPCConn, inName)
	outConverter := newValueToJSONConverter(getAllOutParamInfo(fType), outName, true)
	fLowLevel := func(rpcConn *WebsocketRPCConn, rawArgs json.RawMessage, rawReply *json.RawMessage) error {
		var err error
		args, err := inConverter.tryRun(rpcConn, rawArgs)
		if err != nil {
			return RPCInvalidParamsError
		}
		reply := fValue.Call(args)
		*rawReply, err = outConverter.tryRun(reply)
		return err
	}
	rpc.RegisterLowLevel(name, fLowLevel)
}

// RegisterExplicitly provides a `net/rpc`-like way to register a function.
// In this way, the struct is defined explicitly by the caller
//
// funcObj must have three in arguments. The first is a pointer to RPC connection,
// the second is used to receive the params (can be a pointer or not),
// and the third is used to send the result (must be a pointer).
// Moreover, the function can have no out parameters
// or have one out parameter to return error info.
func (rpc *WebsocketRPC) RegisterExplicitly(name string, fobj interface{}) error {
	if fobj == nil {
		return errors.New("nil pointer passed to RegisterExplicitly")
	}
	fType := reflect.TypeOf(fobj)
	fValue := reflect.ValueOf(fobj)
	hasErrInfoOut := false
	nIn := fType.NumIn()
	nOut := fType.NumOut()
	if nIn != 3 || nOut > 1 {
		return errors.New("cannot recognize the function")
	}
	if fType.In(0) != typeOfPointToRPCConn {
		return errors.New("first in argument must be a pointer to a RPC connection")
	}
	argType := fType.In(1)
	argIsPtr := argType.Kind() == reflect.Ptr
	if argIsPtr {
		argType = argType.Elem()
	}
	replyType := fType.In(2)
	if replyType.Kind() != reflect.Ptr {
		return errors.New("reply argument must be a pointer")
	}
	replyType = replyType.Elem()
	if nOut == 1 {
		if fType.Out(0) != typeOfError {
			return errors.New("the function must return a void or an error")
		}
		hasErrInfoOut = true
	}
	fLowLevel := func(rpcConn *WebsocketRPCConn, rawArgs json.RawMessage, rawReply *json.RawMessage) error {
		var argv reflect.Value
		var err error
		argv = reflect.New(argType)
		err = json.Unmarshal(rawArgs, argv.Interface())
		if err != nil {
			return err
		}
		if !argIsPtr {
			argv = argv.Elem()
		}
		replyv := reflect.New(replyType)
		result := fValue.Call([]reflect.Value{reflect.ValueOf(rpcConn), argv, replyv})
		if hasErrInfoOut {
			targetErr := result[0].Interface()
			if targetErr != nil {
				return targetErr.(error)
			}
		}
		rawReplyBytes, err := json.Marshal(replyv.Interface())
		if err != nil {
			return err
		}
		*rawReply = rawReplyBytes
		return nil
	}
	rpc.RegisterLowLevel(name, fLowLevel)
	return nil
}

// RegisterLowLevel is used to register a normal function for RPC in low-level way (use json.RawMessage).
func (rpc *WebsocketRPC) RegisterLowLevel(name string, method LowLevelRPCMethod) {
	if method == nil {
		return
	}
	rpc.method[name] = method
}

// Connect is a function to create a rpc connection binded to a websocket connection.
func (rpc *WebsocketRPC) Connect(conn *websocket.Conn) *WebsocketRPCConn {
	return rpc.ConnectAdapter(NewWebsocketMessageAdapter(conn))
}

// ConnectAdapter is a function to create a rpc connection binded to an adapter.
func (rpc *WebsocketRPC) ConnectAdapter(adapter MessageAdapter) *WebsocketRPCConn {
	r := WebsocketRPCConn{
		RPC:     rpc,
		adapter: adapter,
		Timeout: 10 * time.Second,
		Session: make(map[string]interface{})}
	return &r
}

// ServeConn is a function that you should call it at last to receive messages continuously.
// It will block until the connection is closed.
func (rpcConn *WebsocketRPCConn) ServeConn() {
	for {
		message, err := rpcConn.adapter.ReadMessage()
		if err != nil {
			break
		}
		go func() {
			rpcConn.processMessage(message)
		}()
	}
	// Handle all pending request
	rpcConn.pending.Range(func(key interface{}, value interface{}) bool {
		rpcConn.pending.Delete(key)
		done, _ := value.(chan *rpcMessage)
		done <- &rpcMessage{
			JSONRPC: "2.0",
			ID:      jsonNullValue,
			Error:   &RPCInternalError}
		return true
	})
}
