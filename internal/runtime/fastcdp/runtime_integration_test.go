package fastcdp

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Homiakus/UiUxMaster/internal/evidence"
	"github.com/Homiakus/UiUxMaster/internal/verifier"
)

func TestResidentRuntimeIntegration(t *testing.T) {
	if os.Getenv("UIUX_FASTCDP_INTEGRATION") != "1" {
		t.Skip("set UIUX_FASTCDP_INTEGRATION=1 to run against a real Chromium binary")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	runtime, err := StartResidentRuntime(ctx, RuntimeConfig{
		Browser: BrowserConfig{Executable: os.Getenv("UIUX_CHROME_BIN")},
		Pages: PagePoolConfig{MaxPages:1, DiagnosticsCapacity:64, Page:PageSpec{URL:"about:blank",Width:320,Height:200,DPR:1}},
	})
	if err != nil { t.Fatal(err) }
	defer func(){ closeCtx,closeCancel:=context.WithTimeout(context.Background(),5*time.Second);defer closeCancel();if err:=runtime.Close(closeCtx);err!=nil{t.Errorf("close runtime: %v",err)} }()

	version,err:=runtime.Version(ctx);if err!=nil{t.Fatal(err)};if version.Product==""{t.Fatal("browser product is empty")}
	lease,err:=runtime.Pages.Acquire(ctx);if err!=nil{t.Fatal(err)};defer lease.Release();page:=lease.Page()
	if page.Diagnostics==nil{t.Fatal("warm page diagnostics observer is missing")}
	before:=page.Epoch.Current();mark:=page.Diagnostics.Mark()

	// Defects deliberately span independent evidence planes: geometry, target
	// sizing, computed accessibility name and asynchronous runtime diagnostics.
	expression:=`document.documentElement.style.background="white";
document.body.style.margin="0";
document.body.innerHTML='<main id="wide" style="width:360px;height:120px;background:#eee"><div id="clip" style="width:100px;height:50px;overflow:hidden"><button id="tiny" aria-label="" style="display:block;width:20px;height:20px;margin-left:90px"></button></div></main>';
console.error("uiux-probe-error");
window.__UIUX_SIGNAL_RENDER__(1);`
	if err:=runtime.Conn.Call(ctx,string(page.Session.SessionID),"Runtime.evaluate",map[string]any{"expression":expression,"returnByValue":true},nil);err!=nil{t.Fatal(err)}

	collected,err:=page.CollectEvidence(ctx,runtime.Conn,EvidenceRequest{
		RequireAfter:before,WaitForNewEpoch:true,
		Snapshot:ptrSnapshotOptions(DefaultSnapshotOptions()),
		Accessibility:true,Fonts:true,DiagnosticsSince:&mark,
	})
	if err!=nil{t.Fatal(err)}
	if collected.Epoch!=1{t.Fatalf("epoch = %d, want 1",collected.Epoch)}
	if collected.Snapshot==nil||len(collected.Snapshot.Documents)==0{t.Fatalf("empty DOM snapshot: %#v",collected.Snapshot)}
	if collected.Accessibility==nil||len(collected.Accessibility.Nodes)==0{t.Fatalf("empty AX tree: %#v",collected.Accessibility)}
	if collected.Fonts==nil{t.Fatal("font state was not collected")}
	if collected.Diagnostics==nil||!collected.Diagnostics.Complete{t.Fatalf("incomplete diagnostics: %#v",collected.Diagnostics)}
	if collected.RGBA!=nil{t.Fatal("full deterministic L2 pass unexpectedly captured pixels")}

	packet:=ToPacket(collected,PacketOptions{
		RunID:"chromium-integration",Scenario:"full-deterministic-defects",
		Viewport:evidence.Viewport{Width:320,Height:200,DeviceScale:1},Browser:version,FidelityID:"blink-l2",
	})
	if packet.Epoch!=1||packet.Renderer.Tier!="L2"||packet.Pixels!=nil||len(packet.Elements)==0||len(packet.Accessibility)==0||packet.Fonts==nil||packet.AriaSnapshot==""{t.Fatalf("incomplete canonical packet: %#v",packet)}
	if packet.Diagnostics==nil||!packet.Diagnostics.Complete{t.Fatalf("packet diagnostics = %#v",packet.Diagnostics)}

	verifier.ApplyDeterministic(&packet,verifier.DefaultPolicy())
	assertIssueCode(t,packet.RuntimeIssues,verifier.CodeViewportHorizontalOverflow)
	assertIssueCode(t,packet.RuntimeIssues,verifier.CodeInteractiveClipped)
	assertIssueCode(t,packet.RuntimeIssues,verifier.CodeTargetTooSmall)
	assertIssueCode(t,packet.RuntimeIssues,verifier.CodeA11yNameMissing)
	assertIssueCode(t,packet.RuntimeIssues,string(DiagnosticConsole))

	// Pixel path remains independently covered and intentionally more expensive.
	img,stats,err:=runtime.Conn.CaptureRegionRGBA(ctx,string(page.Session.SessionID),CaptureRegionOptions{X:0,Y:0,Width:320,Height:200,Scale:1,OptimizeForSpeed:true})
	if err!=nil{t.Fatal(err)}
	if img==nil||img.Bounds().Dx()!=320||img.Bounds().Dy()!=200||stats.EncodedBytes==0{t.Fatalf("invalid ROI capture: image=%v stats=%#v",img,stats)}
}

func ptrSnapshotOptions(value SnapshotOptions)*SnapshotOptions{return &value}
func assertIssueCode(t *testing.T,issues []evidence.RuntimeIssue,code string){t.Helper();for _,issue:=range issues{if issue.Code==code{return}};t.Fatalf("missing issue %q: %#v",code,issues)}
