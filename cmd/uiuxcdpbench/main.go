package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/Homiakus/UiUxMaster/internal/evidence"
	"github.com/Homiakus/UiUxMaster/internal/runtime/fastcdp"
	"github.com/Homiakus/UiUxMaster/internal/verifier"
)

type metric struct{Scenario string `json:"scenario"`;Iterations int `json:"iterations"`;P50US float64 `json:"p50_us"`;P95US float64 `json:"p95_us"`;P99US float64 `json:"p99_us"`;MeanUS float64 `json:"mean_us"`}
type report struct{Browser fastcdp.BrowserVersion `json:"browser"`;StartupUS float64 `json:"startup_us"`;AcquireUS float64 `json:"first_page_acquire_us"`;ROIBytes int `json:"roi_encoded_bytes"`;Metrics []metric `json:"metrics"`}

func main(){
	iterations:=flag.Int("iterations",20,"measured iterations per warm CDP scenario");warmup:=flag.Int("warmup",3,"warmup iterations per scenario");flag.Parse();if *iterations<1||*warmup<0{fmt.Fprintln(os.Stderr,"iterations>=1 and warmup>=0 are required");os.Exit(2)}
	ctx,cancel:=context.WithTimeout(context.Background(),45*time.Second);defer cancel();startupStart:=time.Now()
	runtime,err:=fastcdp.StartResidentRuntime(ctx,fastcdp.RuntimeConfig{Browser:fastcdp.BrowserConfig{Executable:os.Getenv("UIUX_CHROME_BIN")},Pages:fastcdp.PagePoolConfig{MaxPages:1,DiagnosticsCapacity:64,Page:fastcdp.PageSpec{URL:"about:blank",Width:640,Height:480,DPR:1}}});must(err);startup:=time.Since(startupStart)
	defer func(){closeCtx,closeCancel:=context.WithTimeout(context.Background(),5*time.Second);defer closeCancel();must(runtime.Close(closeCtx))}()
	version,err:=runtime.Version(ctx);must(err);acquireStart:=time.Now();lease,err:=runtime.Pages.Acquire(ctx);must(err);acquire:=time.Since(acquireStart);defer lease.Release();page:=lease.Page();session:=string(page.Session.SessionID)
	must(runtime.Conn.Call(ctx,session,"Runtime.evaluate",map[string]any{"expression":`document.body.style.margin="0";document.body.innerHTML='<main id="probe" style="width:320px;height:180px;display:grid;place-items:center;background:rgb(24,28,34);color:white;font:16px sans-serif"><button id="action" style="width:96px;height:36px">Inspect</button></main>';window.__UIUX_SIGNAL_RENDER__(1);`,"returnByValue":true},nil));_,err=page.Epoch.WaitAfter(ctx,0);must(err)
	viewport:=evidence.Viewport{Width:640,Height:480,DeviceScale:1,Browser:version.Product};snapshotOptions:=fastcdp.DefaultSnapshotOptions();baselineSnapshot,err:=runtime.Conn.CaptureSnapshot(ctx,session,snapshotOptions);must(err);baselineCollected:=fastcdp.CollectedEvidence{Epoch:page.Epoch.Current(),Snapshot:&baselineSnapshot};baselinePacket:=fastcdp.ToPacket(baselineCollected,fastcdp.PacketOptions{RunID:"bench",Scenario:"structural",Viewport:viewport,Browser:version,FidelityID:"blink-l2"})

	metrics:=make([]metric,0,11)
	metrics=append(metrics,measure("cdp_evaluate",*iterations,*warmup,func()error{return runtime.Conn.Call(ctx,session,"Runtime.evaluate",map[string]any{"expression":`document.getElementById("probe").clientWidth`,"returnByValue":true},nil)}))
	epoch:=page.Epoch.Current();metrics=append(metrics,measure("cdp_epoch_roundtrip",*iterations,*warmup,func()error{epoch++;previous:=epoch-1;if err:=runtime.Conn.Call(ctx,session,"Runtime.evaluate",map[string]any{"expression":fmt.Sprintf(`window.__UIUX_SIGNAL_RENDER__(%d)`,epoch),"returnByValue":true},nil);err!=nil{return err};_,err:=page.Epoch.WaitAfter(ctx,previous);return err}))
	metrics=append(metrics,measure("cdp_dom_snapshot",*iterations,*warmup,func()error{snapshot,err:=runtime.Conn.CaptureSnapshot(ctx,session,snapshotOptions);if err!=nil{return err};if len(snapshot.Documents)==0||len(snapshot.Documents[0].Nodes)==0{return fmt.Errorf("empty snapshot")};return nil}))
	metrics=append(metrics,measure("cdp_accessibility_tree",*iterations,*warmup,func()error{tree,err:=runtime.Conn.CaptureAXTree(ctx,session,0);if err!=nil{return err};if len(tree.Nodes)==0{return fmt.Errorf("empty AX tree")};return nil}))
	metrics=append(metrics,measure("cdp_font_state",*iterations,*warmup,func()error{fonts,err:=runtime.Conn.CaptureFontState(ctx,session,128);if err!=nil{return err};if fonts.Status==""{return fmt.Errorf("empty font status")};return nil}))
	metrics=append(metrics,measure("diagnostics_barrier",*iterations,*warmup,func()error{return page.Diagnostics.Barrier(ctx,runtime.Conn)}))
	metrics=append(metrics,measure("packet_projection",*iterations,*warmup,func()error{packet:=fastcdp.ToPacket(baselineCollected,fastcdp.PacketOptions{RunID:"bench",Scenario:"projection",Viewport:viewport,Browser:version,FidelityID:"blink-l2"});if len(packet.Elements)==0||len(packet.Documents)==0{return fmt.Errorf("empty packet")};return nil}))
	metrics=append(metrics,measure("deterministic_verify",*iterations,*warmup,func()error{result:=verifier.Verify(baselinePacket,verifier.DefaultPolicy());if len(result.Issues)!=0{return fmt.Errorf("unexpected verifier issues: %v",result.Issues)};return nil}))
	metrics=append(metrics,measure("structural_snapshot_packet_verify",*iterations,*warmup,func()error{snapshot,err:=runtime.Conn.CaptureSnapshot(ctx,session,snapshotOptions);if err!=nil{return err};packet:=fastcdp.ToPacket(fastcdp.CollectedEvidence{Epoch:page.Epoch.Current(),Snapshot:&snapshot},fastcdp.PacketOptions{RunID:"bench",Scenario:"structural",Viewport:viewport,Browser:version,FidelityID:"blink-l2"});result:=verifier.Verify(packet,verifier.DefaultPolicy());if len(result.Issues)!=0{return fmt.Errorf("unexpected verifier issues: %v",result.Issues)};return nil}))
	metrics=append(metrics,measure("full_nonpixel_deterministic",*iterations,*warmup,func()error{mark:=page.Diagnostics.Mark();collected,err:=page.CollectEvidence(ctx,runtime.Conn,fastcdp.EvidenceRequest{Snapshot:&snapshotOptions,Accessibility:true,Fonts:true,DiagnosticsSince:&mark});if err!=nil{return err};packet:=fastcdp.ToPacket(collected,fastcdp.PacketOptions{RunID:"bench",Scenario:"full",Viewport:viewport,Browser:version,FidelityID:"blink-l2"});result:=verifier.VerifyDeterministic(packet,verifier.DefaultPolicy());if len(result.Issues)!=0{return fmt.Errorf("unexpected verifier issues: %v",result.Issues)};if packet.Diagnostics==nil||!packet.Diagnostics.Complete||packet.AriaSnapshot==""||packet.Fonts==nil{return fmt.Errorf("incomplete full evidence")};return nil}))
	roiBytes:=0;metrics=append(metrics,measure("cdp_roi_screenshot",*iterations,*warmup,func()error{img,stats,err:=runtime.Conn.CaptureRegionRGBA(ctx,session,fastcdp.CaptureRegionOptions{X:0,Y:0,Width:320,Height:180,Scale:1,OptimizeForSpeed:true});if err!=nil{return err};if img==nil||img.Bounds().Dx()!=320||img.Bounds().Dy()!=180{return fmt.Errorf("unexpected ROI bounds")};roiBytes=stats.EncodedBytes;return nil}))
	out:=report{Browser:version,StartupUS:micros(startup),AcquireUS:micros(acquire),ROIBytes:roiBytes,Metrics:metrics};encoder:=json.NewEncoder(os.Stdout);encoder.SetIndent("","  ");must(encoder.Encode(out))
}

func measure(name string,iterations,warmup int,fn func()error)metric{for i:=0;i<warmup;i++{must(fn())};samples:=make([]time.Duration,iterations);var total time.Duration;for i:=0;i<iterations;i++{started:=time.Now();must(fn());duration:=time.Since(started);samples[i]=duration;total+=duration};sort.Slice(samples,func(i,j int)bool{return samples[i]<samples[j]});return metric{Scenario:name,Iterations:iterations,P50US:micros(percentile(samples,.50)),P95US:micros(percentile(samples,.95)),P99US:micros(percentile(samples,.99)),MeanUS:micros(total/time.Duration(iterations))}}
func percentile(samples []time.Duration,p float64)time.Duration{if len(samples)==0{return 0};index:=int(float64(len(samples)-1)*p);if index<0{index=0};if index>=len(samples){index=len(samples)-1};return samples[index]}
func micros(duration time.Duration)float64{return float64(duration.Nanoseconds())/1000}
func must(err error){if err!=nil{fmt.Fprintln(os.Stderr,err);os.Exit(1)}}
