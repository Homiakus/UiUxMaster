package fastcdp

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const defaultDiagnosticCapacity = 256

type DiagnosticKind string

const (
	DiagnosticRuntimeException DiagnosticKind = "runtime.exception"
	DiagnosticConsole          DiagnosticKind = "runtime.console"
	DiagnosticLog              DiagnosticKind = "browser.log"
	DiagnosticNetworkFailed    DiagnosticKind = "network.failed"
	DiagnosticHTTPError        DiagnosticKind = "network.http_error"
)

type DiagnosticEvent struct {
	Sequence   uint64
	Kind       DiagnosticKind
	Level      string
	Message    string
	URL        string
	RequestID  string
	Resource   string
	StatusCode int
	Canceled   bool
}

type DiagnosticMark struct {
	Sequence uint64
	Dropped  map[string]uint64
}

type DiagnosticSnapshot struct {
	Events         []DiagnosticEvent
	Complete       bool
	DroppedMethods []string
}

type diagnosticSubscription struct {
	method    string
	sub       *EventSubscription
	processed atomic.Uint64
}

type DiagnosticsObserver struct {
	session  string
	capacity int
	seq            atomic.Uint64
	evictedThrough atomic.Uint64
	mu             sync.RWMutex
	ring           []DiagnosticEvent
	subs []*diagnosticSubscription
	stop chan struct{}
	done sync.WaitGroup
	once sync.Once
}

func NewDiagnosticsObserver(conn *Connection, session SessionID, capacity int) (*DiagnosticsObserver, error) {
	if conn == nil || session == "" { return nil, fmt.Errorf("fastcdp: diagnostics observer requires connection and session") }
	if capacity <= 0 { capacity = defaultDiagnosticCapacity }
	o := &DiagnosticsObserver{session:string(session), capacity:capacity, stop:make(chan struct{})}
	for _, method := range []string{"Runtime.exceptionThrown","Runtime.consoleAPICalled","Log.entryAdded","Network.loadingFailed","Network.responseReceived"} {
		item := &diagnosticSubscription{method:method, sub:conn.SubscribeObserved(method,capacity)}
		o.subs = append(o.subs,item)
		o.done.Add(1)
		go o.consume(item)
	}
	return o,nil
}

func (o *DiagnosticsObserver) consume(item *diagnosticSubscription) {
	defer o.done.Done()
	for {
		select {
		case <-o.stop: return
		case event := <-item.sub.Events:
			if event.SessionID == o.session {
				if diagnostic,ok:=decodeDiagnostic(item.method,event.Params); ok { o.append(diagnostic) }
			}
			item.processed.Add(1)
		}
	}
}

func (o *DiagnosticsObserver) append(event DiagnosticEvent) {
	event.Sequence=o.seq.Add(1)
	o.mu.Lock()
	if len(o.ring)<o.capacity { o.ring=append(o.ring,event) } else { o.evictedThrough.Store(o.ring[0].Sequence); copy(o.ring,o.ring[1:]); o.ring[len(o.ring)-1]=event }
	o.mu.Unlock()
}

func (o *DiagnosticsObserver) Mark() DiagnosticMark {
	if o==nil { return DiagnosticMark{} }
	mark:=DiagnosticMark{Sequence:o.seq.Load(),Dropped:make(map[string]uint64,len(o.subs))}
	for _,item:=range o.subs { mark.Dropped[item.method]=item.sub.Dropped() }
	return mark
}

// Barrier establishes a causal cut through the page session. When the no-op
// Runtime.evaluate response is observed, all earlier CDP events have already
// passed through Connection.publish. The observer then waits until its consumer
// goroutines have processed every successfully delivered event up to that cut.
func (o *DiagnosticsObserver) Barrier(ctx context.Context, conn *Connection) error {
	if o==nil { return nil }
	if conn==nil { return fmt.Errorf("fastcdp: diagnostics barrier requires connection") }
	if err:=conn.Call(ctx,o.session,"Runtime.evaluate",map[string]any{"expression":"0","returnByValue":false},nil); err!=nil { return fmt.Errorf("fastcdp: diagnostics barrier: %w",err) }
	targets:=make([]uint64,len(o.subs)); for i,item:=range o.subs { targets[i]=item.sub.Delivered() }
	complete:=func() bool { for i,item:=range o.subs { if item.processed.Load()<targets[i] { return false } }; return true }
	if complete(){ return nil }
	ticker:=time.NewTicker(100*time.Microsecond); defer ticker.Stop()
	for { select { case <-ctx.Done(): return ctx.Err(); case <-ticker.C: if complete(){ return nil } } }
}

func (o *DiagnosticsObserver) SnapshotSince(mark DiagnosticMark) DiagnosticSnapshot {
	if o==nil { return DiagnosticSnapshot{Complete:true} }
	o.mu.RLock(); out:=make([]DiagnosticEvent,0); for _,event:=range o.ring { if event.Sequence>mark.Sequence { out=append(out,event) } }; o.mu.RUnlock()
	result:=DiagnosticSnapshot{Events:out,Complete:true}
	if mark.Sequence<o.evictedThrough.Load(){ result.Complete=false; result.DroppedMethods=append(result.DroppedMethods,"observer.ring") }
	for _,item:=range o.subs { if item.sub.Dropped()>mark.Dropped[item.method] { result.Complete=false; result.DroppedMethods=append(result.DroppedMethods,item.method) } }
	sort.Strings(result.DroppedMethods); return result
}

func (o *DiagnosticsObserver) Close(){ if o==nil{return}; o.once.Do(func(){ close(o.stop); for _,item:=range o.subs{item.sub.Close()}; o.done.Wait() }) }

func (c *Connection) EnableDiagnosticDomains(ctx context.Context, session SessionID) error {
	for _,method:=range []string{"Network.enable","Log.enable"}{ if err:=c.Call(ctx,string(session),method,nil,nil); err!=nil{return fmt.Errorf("fastcdp: %s: %w",method,err)} }; return nil
}

func decodeDiagnostic(method string, params json.RawMessage) (DiagnosticEvent,bool) {
	switch method {
	case "Runtime.exceptionThrown":
		var payload struct{ ExceptionDetails struct{ Text string `json:"text"`; URL string `json:"url"`; Exception struct{ Description string `json:"description"`; Value any `json:"value"` } `json:"exception"` } `json:"exceptionDetails"` }
		if json.Unmarshal(params,&payload)!=nil{return DiagnosticEvent{},false}; message:=firstNonEmpty(payload.ExceptionDetails.Exception.Description,stringify(payload.ExceptionDetails.Exception.Value),payload.ExceptionDetails.Text); return DiagnosticEvent{Kind:DiagnosticRuntimeException,Level:"error",Message:message,URL:payload.ExceptionDetails.URL},message!=""
	case "Runtime.consoleAPICalled":
		var payload struct{ Type string `json:"type"`; Args []struct{Value any `json:"value"`; Description string `json:"description"`} `json:"args"` }
		if json.Unmarshal(params,&payload)!=nil{return DiagnosticEvent{},false}; level:=strings.ToLower(payload.Type); if level!="error"&&level!="warning"&&level!="assert"{return DiagnosticEvent{},false}; parts:=make([]string,0,len(payload.Args)); for _,arg:=range payload.Args{if text:=firstNonEmpty(stringify(arg.Value),arg.Description);text!=""{parts=append(parts,text)}}; return DiagnosticEvent{Kind:DiagnosticConsole,Level:level,Message:strings.Join(parts," ")},true
	case "Log.entryAdded":
		var payload struct{Entry struct{Source string `json:"source"`; Level string `json:"level"`; Text string `json:"text"`; URL string `json:"url"`} `json:"entry"`}; if json.Unmarshal(params,&payload)!=nil||payload.Entry.Text==""{return DiagnosticEvent{},false}; level:=strings.ToLower(payload.Entry.Level); if level!="error"&&level!="warning"{return DiagnosticEvent{},false}; return DiagnosticEvent{Kind:DiagnosticLog,Level:level,Message:payload.Entry.Text,URL:payload.Entry.URL,Resource:payload.Entry.Source},true
	case "Network.loadingFailed":
		var payload struct{RequestID string `json:"requestId"`; Type string `json:"type"`; ErrorText string `json:"errorText"`; Canceled bool `json:"canceled"`}; if json.Unmarshal(params,&payload)!=nil||payload.ErrorText==""{return DiagnosticEvent{},false}; return DiagnosticEvent{Kind:DiagnosticNetworkFailed,Level:"error",Message:payload.ErrorText,RequestID:payload.RequestID,Resource:payload.Type,Canceled:payload.Canceled},true
	case "Network.responseReceived":
		var payload struct{RequestID string `json:"requestId"`; Type string `json:"type"`; Response struct{URL string `json:"url"`; Status float64 `json:"status"`; StatusText string `json:"statusText"`} `json:"response"`}; if json.Unmarshal(params,&payload)!=nil||payload.Response.Status<400{return DiagnosticEvent{},false}; status:=int(payload.Response.Status); message:="HTTP "+strconv.Itoa(status); if payload.Response.StatusText!=""{message+=" "+payload.Response.StatusText}; return DiagnosticEvent{Kind:DiagnosticHTTPError,Level:"error",Message:message,URL:payload.Response.URL,RequestID:payload.RequestID,Resource:payload.Type,StatusCode:status},true
	}
	return DiagnosticEvent{},false
}
func firstNonEmpty(values ...string)string{for _,value:=range values{if value=strings.TrimSpace(value);value!=""{return value}};return ""}
func stringify(value any)string{if value==nil{return ""};if text,ok:=value.(string);ok{return text};payload,err:=json.Marshal(value);if err!=nil{return fmt.Sprint(value)};return string(payload)}
