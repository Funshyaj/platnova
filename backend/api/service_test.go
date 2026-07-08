package api

import "testing"

func resetState() {
	mu.Lock()
	users = nil
	accounts = nil
	transactions = nil
	idempotencyStore = map[string]idempotentResult{}
	mu.Unlock()
}

func TestConvertAmount_SameCurrency(t *testing.T) {
	got, rate, err := ConvertAmount(100, "USD", "USD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 100 || rate != 1 {
		t.Fatalf("expected 100 @ rate 1, got %v @ %v", got, rate)
	}
}

func TestConvertAmount_USDToNGN(t *testing.T) {
	got, rate, err := ConvertAmount(10, "USD", "NGN")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 15500 {
		t.Fatalf("expected 15500, got %v", got)
	}
	if rate != 1550 {
		t.Fatalf("expected rate 1550, got %v", rate)
	}
}

func TestConvertAmount_NGNToUSD(t *testing.T) {
	got, _, err := ConvertAmount(1550, "NGN", "USD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 1 {
		t.Fatalf("expected 1, got %v", got)
	}
}

func TestConvertAmount_CrossNonUSDPair(t *testing.T) {
	// EUR -> GBP via USD: 92 EUR = 100 USD = (100 * 0.79) GBP = 79 GBP
	got, _, err := ConvertAmount(92, "EUR", "GBP")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 79 {
		t.Fatalf("expected 79, got %v", got)
	}
}

func TestConvertAmount_UnsupportedCurrency(t *testing.T) {
	if _, _, err := ConvertAmount(10, "USD", "XYZ"); err == nil {
		t.Fatal("expected error for unsupported currency")
	}
}

func TestTransfer_InsufficientFunds(t *testing.T) {
	resetState()
	from, _ := CreateAccount("", "From", "USD")
	to, _ := CreateAccount("", "To", "USD")
	_, _, _ = Deposit(from.ID, 50)

	status, body := Transfer("key-1", TransferRequest{
		FromAccountID: from.ID,
		ToAccountID:   to.ID,
		Amount:        100,
	})

	if status != 422 {
		t.Fatalf("expected 422, got %d (%v)", status, body)
	}
	refreshedFrom, _ := GetAccount(from.ID)
	if refreshedFrom.Balance != 50 {
		t.Fatalf("balance should be untouched on failed transfer, got %v", refreshedFrom.Balance)
	}
}

func TestTransfer_SuccessCrossCurrency(t *testing.T) {
	resetState()
	from, _ := CreateAccount("", "From", "USD")
	to, _ := CreateAccount("", "To", "NGN")
	_, _, _ = Deposit(from.ID, 100)

	status, body := Transfer("key-2", TransferRequest{
		FromAccountID: from.ID,
		ToAccountID:   to.ID,
		Amount:        10,
	})

	if status != 201 {
		t.Fatalf("expected 201, got %d (%v)", status, body)
	}
	result, ok := body.(TransferResult)
	if !ok {
		t.Fatalf("expected TransferResult, got %T", body)
	}
	if result.ConvertedAmt != 15500 {
		t.Fatalf("expected converted amount 15500, got %v", result.ConvertedAmt)
	}

	refreshedFrom, _ := GetAccount(from.ID)
	refreshedTo, _ := GetAccount(to.ID)
	if refreshedFrom.Balance != 90 {
		t.Fatalf("expected sender balance 90, got %v", refreshedFrom.Balance)
	}
	if refreshedTo.Balance != 15500 {
		t.Fatalf("expected receiver balance 15500, got %v", refreshedTo.Balance)
	}
}

func TestTransfer_IdempotencyReplay(t *testing.T) {
	resetState()
	from, _ := CreateAccount("", "From", "USD")
	to, _ := CreateAccount("", "To", "USD")
	_, _, _ = Deposit(from.ID, 100)

	req := TransferRequest{FromAccountID: from.ID, ToAccountID: to.ID, Amount: 30}

	status1, body1 := Transfer("same-key", req)
	status2, body2 := Transfer("same-key", req)

	if status1 != status2 {
		t.Fatalf("expected same status on replay, got %d vs %d", status1, status2)
	}
	r1 := body1.(TransferResult)
	r2 := body2.(TransferResult)
	if r1.TransferID != r2.TransferID {
		t.Fatalf("expected identical transfer id on replay, got %s vs %s", r1.TransferID, r2.TransferID)
	}

	refreshedFrom, _ := GetAccount(from.ID)
	if refreshedFrom.Balance != 70 {
		t.Fatalf("replayed request must not move money twice, expected balance 70, got %v", refreshedFrom.Balance)
	}
}

func TestTransfer_MissingIdempotencyKey(t *testing.T) {
	resetState()
	from, _ := CreateAccount("", "From", "USD")
	to, _ := CreateAccount("", "To", "USD")
	_, _, _ = Deposit(from.ID, 100)

	status, _ := Transfer("", TransferRequest{FromAccountID: from.ID, ToAccountID: to.ID, Amount: 10})
	if status != 400 {
		t.Fatalf("expected 400 for missing idempotency key, got %d", status)
	}
}

func TestAccountView_JoinsOwner(t *testing.T) {
	resetState()
	user, _ := CreateUser("Ada Lovelace", "ada@example.com")
	acc, err := CreateAccount(user.ID, "USD Wallet", "USD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	view, err := GetAccountView(acc.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if view.Owner == nil || view.Owner.ID != user.ID {
		t.Fatalf("expected joined owner %s, got %+v", user.ID, view.Owner)
	}
	if view.Owner.Email != "ada@example.com" {
		t.Fatalf("expected owner email to be joined, got %q", view.Owner.Email)
	}
}

func TestCreateAccount_UnknownUserRejected(t *testing.T) {
	resetState()
	if _, err := CreateAccount("does-not-exist", "Wallet", "USD"); err == nil {
		t.Fatal("expected error when linking account to a nonexistent user")
	}
}

func TestDeposit_RejectsNonPositiveAmount(t *testing.T) {
	resetState()
	acc, _ := CreateAccount("", "Acc", "USD")
	if _, _, err := Deposit(acc.ID, 0); err == nil {
		t.Fatal("expected error for zero amount deposit")
	}
	if _, _, err := Deposit(acc.ID, -5); err == nil {
		t.Fatal("expected error for negative amount deposit")
	}
}
