package scan

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type (
	nilClient struct{}
	AxInput   struct{}
	BxInput   struct{}
	CxInput   struct{}
	DxInput   struct{}
	AxRequest struct{}
	BxRequest struct{}
	CxRequest struct{}
	DxRequest struct{}
	AxOutput  struct{}
	BxOutput  struct{}
	CxOutput  struct{}
	DxOutput  struct{}
)

func newNilClient() *nilClient { return nil }

func (*nilClient) AxRequest(*AxInput) AxRequest { return AxRequest{} }
func (*nilClient) BxRequest(*BxInput) BxRequest { return BxRequest{} }
func (*nilClient) CxRequest(*CxInput) CxRequest { return CxRequest{} }
func (*nilClient) DxRequest(*DxInput) DxRequest { return DxRequest{} }

func (AxRequest) Send() *AxOutput { return nil }
func (BxRequest) Send() *BxOutput { return nil }
func (CxRequest) Send() *CxOutput { return nil }
func (DxRequest) Send() *DxOutput { return nil }

type tree struct{ *Ctx }

func (tree) B(*AxOutput) []BxInput { return nil }
func (tree) C(*BxOutput) []CxInput { return nil }
func (tree) D(*BxOutput) []DxInput { return nil }

type diamond struct{ *Ctx }

func (diamond) A() []AxInput                     { return nil }
func (diamond) B(*AxOutput) []BxInput            { return nil }
func (diamond) C(*AxOutput) []CxInput            { return nil }
func (diamond) D(*BxOutput, *CxOutput) []DxInput { return nil }

type multi struct{ *Ctx }

func (multi) C(*AxOutput) []CxInput   { return nil }
func (multi) DxB(*BxOutput) []DxInput { return nil }
func (multi) DxC(*CxOutput) []DxInput { return nil }

func TestSvc(t *testing.T) {
	var reg registry
	register := func(name string, newFunc interface{}, iface svcIface, roots ...interface{}) *svc {
		reg.register(name, name, newNilClient, iface, roots)
		return reg.reg[name]
	}
	tests := []*struct {
		svc  *svc
		next map[string][]string
	}{{
		svc: register("tree", newNilClient, tree{}, []AxInput{}),
		next: map[string][]string{
			"":   {"Ax"},
			"Ax": {"Bx"},
			"Bx": {"Cx", "Dx"},
		},
	}, {
		svc: register("diamond", newNilClient, diamond{}),
		next: map[string][]string{
			"":   {"Ax"},
			"Ax": {"Bx", "Cx"},
			"Bx": {"Dx"},
			"Cx": {"Dx"},
		},
	}, {
		svc: register("multi", newNilClient, multi{}, []AxInput{}, []BxInput{}),
		next: map[string][]string{
			"":   {"Ax", "Bx"},
			"Ax": {"Cx"},
			"Bx": {"Dx"},
			"Cx": {"Dx"},
		},
	}}
	reg.get()
	for _, tc := range tests {
		assert.Contains(t, reg.reg, tc.svc.name)
		assert.Len(t, tc.svc.api, 4, "svc=%s", tc.svc.name)
		assert.Equal(t, tc.next, tc.svc.next, "name=%s", tc.svc.name)
	}
}

type depErr struct{ *Ctx }

func (depErr) B(*AxOutput) []BxInput { return nil }
func (depErr) C(*BxOutput) []CxInput { return nil }
func (depErr) D(*CxOutput) []DxInput { return nil }

type cycleErr struct{ *Ctx }

func (cycleErr) B(*DxOutput) []BxInput { return nil }
func (cycleErr) C(*BxOutput) []CxInput { return nil }
func (cycleErr) D(*CxOutput) []DxInput { return nil }

func TestSvcPanic(t *testing.T) {
	assert.PanicsWithValue(t, "scan: unsatisfied dependency: *scan.AxOutput -> []scan.BxInput", func() {
		var r registry
		r.register("test1", "depErr", newNilClient, depErr{}, nil)
		r.get()
	})
	assert.PanicsWithValue(t, "scan: test2 dependency cycle: Bx,Cx,Dx", func() {
		var r registry
		r.register("test2", "cycleErr", newNilClient, cycleErr{}, []interface{}{[]AxInput{}})
		r.get()
	})
}
