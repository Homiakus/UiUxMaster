package fastcdp

import (
	"testing"
	"time"

	"github.com/Homiakus/UiUxMaster/internal/evidence"
)

func TestToPacketProjectsAccessibilityFontsAndDiagnostics(t *testing.T) {
	packet := ToPacket(CollectedEvidence{
		Epoch: 3,
		Accessibility: &AXTree{Nodes: []AXNode{{ID:"ax-1",BackendDOMNodeID:42,Role:"button",Name:"Save",Properties:map[string]string{"focusable":"true"}}}},
		Fonts: &FontState{Status:"loaded",Total:1,Faces:[]FontFaceState{{Family:"Inter",Weight:"400",Status:"loaded"}}},
		Diagnostics: &DiagnosticSnapshot{Complete:true,Events:[]DiagnosticEvent{{Kind:DiagnosticConsole,Level:"error",Message:"boom"}}},
		Timing: EvidenceTiming{Accessibility:time.Millisecond,Fonts:2*time.Millisecond,Diagnostics:3*time.Millisecond,Total:4*time.Millisecond},
	}, PacketOptions{Browser: BrowserVersion{Product:"Chrome/152"}})
	if len(packet.Accessibility)!=1||packet.Accessibility[0].BackendNodeID!=42||packet.AriaSnapshot==""{t.Fatalf("accessibility = %#v snapshot=%q",packet.Accessibility,packet.AriaSnapshot)}
	if packet.Fonts==nil||packet.Fonts.Status!="loaded"||len(packet.Fonts.Faces)!=1{t.Fatalf("fonts = %#v",packet.Fonts)}
	if packet.Diagnostics==nil||!packet.Diagnostics.Complete{t.Fatalf("diagnostics = %#v",packet.Diagnostics)}
	if len(packet.RuntimeIssues)!=1||packet.RuntimeIssues[0].Code!=string(DiagnosticConsole)||packet.RuntimeIssues[0].Severity!=evidence.SeverityHigh{t.Fatalf("runtime issues = %#v",packet.RuntimeIssues)}
	if packet.Latency.AccessibilityMS!=1||packet.Latency.FontsMS!=2||packet.Latency.DiagnosticsMS!=3{t.Fatalf("latency = %#v",packet.Latency)}
}

func TestToPacketPreservesIncompleteDiagnostics(t *testing.T) {
	packet:=ToPacket(CollectedEvidence{Diagnostics:&DiagnosticSnapshot{Complete:false,DroppedMethods:[]string{"Runtime.consoleAPICalled"}}},PacketOptions{})
	if packet.Diagnostics==nil||packet.Diagnostics.Complete||len(packet.Diagnostics.DroppedMethods)!=1{t.Fatalf("diagnostics = %#v",packet.Diagnostics)}
}
