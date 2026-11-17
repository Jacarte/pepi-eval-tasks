package warehouse

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func sampleTransfer(n int) *Transfer {
	return &Transfer{
		ID:        fmt.Sprintf("transfer-%03d", n),
		FromBin:   fmt.Sprintf("frombin %d;a=b,c\\d", n),
		ToBin:     fmt.Sprintf("tobin-%d", n),
		SKU:       fmt.Sprintf("sku %d;a=b,c\\d", n),
		Qty:       n*18 + 259,
		Status:    TransferStatusPlanned,
		CreatedAt: time.Date(2023, time.January, n, 8, 0, 0, 0, time.UTC),
	}
}

func TestTransferCodecRoundTrip(t *testing.T) {
	in := sampleTransfer(4)
	line := EncodeTransfer(in)
	out, err := DecodeTransfer(line)
	if err != nil {
		t.Fatalf("DecodeTransfer(%q): %v", line, err)
	}
	if got := EncodeTransfer(out); got != line {
		t.Fatalf("round trip mismatch:\n got %s\nwant %s", got, line)
	}
	if out.FromBin != in.FromBin {
		t.Errorf("FromBin = %q, want %q", out.FromBin, in.FromBin)
	}
	if out.ToBin != in.ToBin {
		t.Errorf("ToBin = %q, want %q", out.ToBin, in.ToBin)
	}
	if out.SKU != in.SKU {
		t.Errorf("SKU = %q, want %q", out.SKU, in.SKU)
	}
	if _, err := DecodeTransfer("id"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("malformed input: err = %v, want ErrInvalid", err)
	}
}

func TestMemoryTransferStore(t *testing.T) {
	ctx := context.Background()
	st := NewMemoryTransferStore()
	for n := 1; n <= 3; n++ {
		if err := st.Store(ctx, sampleTransfer(n)); err != nil {
			t.Fatalf("Store(%d): %v", n, err)
		}
	}
	got, err := st.Load(ctx, "transfer-002")
	if err != nil {
		t.Fatal(err)
	}
	got.Status = TransferStatusDone
	again, _ := st.Load(ctx, "transfer-002")
	if again.Status != TransferStatusPlanned {
		t.Fatal("store handed out a shared pointer")
	}
	if err := st.Store(ctx, got); err != nil {
		t.Fatal(err)
	}
	list, err := st.Scan(ctx, TransferFilter{Status: TransferStatusPlanned})
	if err != nil || len(list) != 2 || list[0].ID != "transfer-001" {
		t.Fatalf("Scan by status = %v, %v", list, err)
	}
	if err := st.Drop(ctx, "transfer-001"); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Load(ctx, "transfer-001"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("after delete: err = %v, want ErrNotFound", err)
	}
	if st.Len() != 2 {
		t.Fatalf("Len = %d, want 2", st.Len())
	}
}

func TestTransferStatusTransitions(t *testing.T) {
	for _, s := range []TransferStatus{TransferStatusPlanned, TransferStatusDone} {
		got, err := ParseTransferStatus(s.String())
		if err != nil || got != s {
			t.Fatalf("Parse(%q) = %v, %v", s.String(), got, err)
		}
	}
	if !TransferStatusPlanned.CanTransition(TransferStatusMoving) {
		t.Fatal("first transition should be allowed")
	}
	if TransferStatusMoving.CanTransition(TransferStatusPlanned) {
		t.Fatal("backwards transition allowed")
	}
	if !TransferStatusDone.Terminal() {
		t.Fatal("last status should be terminal")
	}
}

func TestSummarizeTransfers(t *testing.T) {
	items := []*Transfer{sampleTransfer(1), nil, sampleTransfer(2), sampleTransfer(3)}
	items[2].Status = TransferStatusMoving
	sum := SummarizeTransfers(items)
	if sum.Count != 3 || sum.ByStatus[TransferStatusPlanned] != 2 || sum.ByStatus[TransferStatusMoving] != 1 {
		t.Fatalf("counts = %+v", sum)
	}
	if sum.TotalQty != 885 {
		t.Errorf("TotalQty = %v, want 885", sum.TotalQty)
	}
	if !sum.Earliest.Equal(items[0].CreatedAt) || !sum.Latest.Equal(items[3].CreatedAt) {
		t.Errorf("range = %v..%v", sum.Earliest, sum.Latest)
	}
	if top := TopTransfersByQty(items[2:], 1); len(top) != 1 || top[0].ID != "transfer-003" {
		t.Errorf("top = %v", top)
	}
}
