package ledger

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"shopcore/money"
)

func sampleAccount(n int) *Account {
	return &Account{
		ID:        fmt.Sprintf("account-%03d", n),
		Code:      fmt.Sprintf("code %d;a=b,c\\d", n),
		Name:      fmt.Sprintf("name-%d", n),
		Kind:      fmt.Sprintf("kind %d;a=b,c\\d", n),
		Balance:   money.Cents(n*33 + 85),
		Status:    AccountStatusOpen,
		CreatedAt: time.Date(2023, time.January, n, 8, 0, 0, 0, time.UTC),
	}
}

func TestAccountCodecRoundTrip(t *testing.T) {
	in := sampleAccount(4)
	line := EncodeAccount(in)
	out, err := DecodeAccount(line)
	if err != nil {
		t.Fatalf("DecodeAccount(%q): %v", line, err)
	}
	if got := EncodeAccount(out); got != line {
		t.Fatalf("round trip mismatch:\n got %s\nwant %s", got, line)
	}
	if out.Code != in.Code {
		t.Errorf("Code = %q, want %q", out.Code, in.Code)
	}
	if out.Name != in.Name {
		t.Errorf("Name = %q, want %q", out.Name, in.Name)
	}
	if out.Kind != in.Kind {
		t.Errorf("Kind = %q, want %q", out.Kind, in.Kind)
	}
	if _, err := DecodeAccount("id"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("malformed input: err = %v, want ErrInvalid", err)
	}
}

func TestAccountValidate(t *testing.T) {
	if err := sampleAccount(1).Validate(); err != nil {
		t.Fatalf("valid sample rejected: %v", err)
	}
	bad := sampleAccount(1)
	bad.ID = " "
	if err := bad.Validate(); !errors.Is(err, ErrInvalid) {
		t.Fatalf("blank id: err = %v", err)
	}
	bad = sampleAccount(1)
	bad.Code = ""
	if err := bad.Validate(); !errors.Is(err, ErrInvalid) {
		t.Fatalf("missing code: err = %v", err)
	}
	bad = sampleAccount(1)
	bad.Status = AccountStatusUnknown
	if err := bad.Validate(); err == nil {
		t.Fatal("unknown status accepted")
	}
}

func TestAccountStatusTransitions(t *testing.T) {
	for _, s := range []AccountStatus{AccountStatusOpen, AccountStatusClosed} {
		got, err := ParseAccountStatus(s.String())
		if err != nil || got != s {
			t.Fatalf("Parse(%q) = %v, %v", s.String(), got, err)
		}
	}
	if !AccountStatusOpen.CanTransition(AccountStatusFrozen) {
		t.Fatal("first transition should be allowed")
	}
	if AccountStatusFrozen.CanTransition(AccountStatusOpen) {
		t.Fatal("backwards transition allowed")
	}
	if !AccountStatusClosed.Terminal() {
		t.Fatal("last status should be terminal")
	}
}

func TestSummarizeAccounts(t *testing.T) {
	items := []*Account{sampleAccount(1), nil, sampleAccount(2), sampleAccount(3)}
	items[2].Status = AccountStatusFrozen
	sum := SummarizeAccounts(items)
	if sum.Count != 3 || sum.ByStatus[AccountStatusOpen] != 2 || sum.ByStatus[AccountStatusFrozen] != 1 {
		t.Fatalf("counts = %+v", sum)
	}
	if sum.TotalBalance != 453 {
		t.Errorf("TotalBalance = %v, want 453", sum.TotalBalance)
	}
	if !sum.Earliest.Equal(items[0].CreatedAt) || !sum.Latest.Equal(items[3].CreatedAt) {
		t.Errorf("range = %v..%v", sum.Earliest, sum.Latest)
	}
	if top := TopAccountsByBalance(items[2:], 1); len(top) != 1 || top[0].ID != "account-003" {
		t.Errorf("top = %v", top)
	}
}
