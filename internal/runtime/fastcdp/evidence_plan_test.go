package fastcdp

import (
	"testing"

	"github.com/Homiakus/UiUxMaster/internal/evidenceplan"
)

func TestRequestFromSignalsKeepsQuickStructuralNonPixel(t *testing.T){
	mark:=DiagnosticMark{Sequence:7}
	req:=RequestFromSignals(evidenceplan.Signals{Intent:evidenceplan.IntentQuickStructural},PlannedRequestOptions{RequireAfter:3,WaitForNewEpoch:true,DiagnosticsSince:&mark})
	if req.Snapshot==nil||req.Accessibility||req.Fonts||req.Region!=nil||req.DiagnosticsSince==nil{t.Fatalf("request = %#v",req)}
}
func TestRequestFromSignalsMapsVisualROI(t *testing.T){
	mark:=DiagnosticMark{}
	req:=RequestFromSignals(evidenceplan.Signals{Intent:evidenceplan.IntentVisualRegion,Region:&evidenceplan.Region{X:10,Y:20,Width:200,Height:100}},PlannedRequestOptions{DiagnosticsSince:&mark})
	if req.Snapshot==nil||!req.Accessibility||!req.Fonts||req.Region==nil{t.Fatalf("request = %#v",req)}
	if req.Region.X!=10||req.Region.Y!=20||req.Region.Width!=200||req.Region.Height!=100||req.Region.Scale!=1{t.Fatalf("region = %#v",req.Region)}
}
