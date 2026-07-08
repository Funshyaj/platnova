package api

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

// ---- Models ----

type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type Currency struct {
	Code     string `json:"code"`
	FullName string `json:"full_name"`
	Symbol   string `json:"symbol"`
}

// currencies is the static list of currencies the wallet service supports,
// matching the pairs available in staticRates plus USD as the base.
var currencies = []Currency{
	{Code: "USD", FullName: "US Dollar", Symbol: "$"},
	{Code: "NGN", FullName: "Naira", Symbol: "₦"},
	{Code: "EUR", FullName: "Euro", Symbol: "€"},
	{Code: "GBP", FullName: "British Pound", Symbol: "£"},
}

// Account is a wallet: a balance held in one currency, owned by a User
// (UserID empty for platform-owned accounts like the Vault).
type Account struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id,omitempty"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Currency  string    `json:"currency"`
	Balance   float64   `json:"balance"`
	CreatedAt time.Time `json:"created_at"`
}

// AccountView is an Account joined with its owning User, used for API
// responses so the frontend doesn't need a second round trip per wallet.
type AccountView struct {
	Account
	Owner *User `json:"owner,omitempty"`
}

type Transaction struct {
	ID               string    `json:"id"`
	AccountID        string    `json:"account_id"`
	Type             string    `json:"type"` // deposit | transfer_in | transfer_out
	Amount           float64   `json:"amount"`
	Currency         string    `json:"currency"`
	RelatedAccountID string    `json:"related_account_id,omitempty"`
	TransferID       string    `json:"transfer_id,omitempty"`
	Rate             float64   `json:"rate,omitempty"`
	BalanceAfter     float64   `json:"balance_after"`
	CreatedAt        time.Time `json:"created_at"`
}

// idempotentResult caches the outcome of a previously processed transfer
// so a replayed request with the same Idempotency-Key returns the same
// response instead of moving money twice.
type idempotentResult struct {
	statusCode int
	body       any
}

// ---- "Database" (in-memory arrays) ----

var (
	mu               sync.Mutex
	users            []*User
	accounts         []*Account
	transactions     []*Transaction
	idempotencyStore = map[string]idempotentResult{}
)

// staticRates is the fixed FX rate table from the assessment brief, expressed
// as USD_<CCY>: units of <CCY> per 1 USD. USD is treated as the base currency
// for converting between any two supported currencies.
var staticRates = map[string]float64{
	"USD_NGN": 1550,
	"USD_EUR": 0.92,
	"USD_GBP": 0.79,
}

func genID(prefix string) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return prefix + "_" + hex.EncodeToString(b)
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// ---- Currency conversion ----

// usdPerUnit returns how many USD one unit of currency ccy is worth.
func usdPerUnit(ccy string) (float64, error) {
	if ccy == "USD" {
		return 1, nil
	}
	rate, ok := staticRates["USD_"+ccy]
	if !ok || rate == 0 {
		return 0, errors.New("unsupported currency: " + ccy)
	}
	return 1 / rate, nil
}

// ConvertAmount converts amount from currency `from` to currency `to` using
// the static USD-based rate table, and returns the resulting amount plus the
// effective from->to rate applied.
func ConvertAmount(amount float64, from, to string) (float64, float64, error) {
	if from == to {
		return round2(amount), 1, nil
	}
	fromUSD, err := usdPerUnit(from)
	if err != nil {
		return 0, 0, err
	}
	toUSD, err := usdPerUnit(to)
	if err != nil {
		return 0, 0, err
	}
	usdAmount := amount * fromUSD
	converted := usdAmount / toUSD
	rate := fromUSD / toUSD // amount of `to` per unit of `from`
	return round2(converted), rate, nil
}

// ---- User operations ----

func CreateUser(name, email string) (*User, error) {
	if name == "" {
		return nil, errors.New("name is required")
	}
	mu.Lock()
	defer mu.Unlock()
	u := &User{ID: genID("usr"), Name: name, Email: email, CreatedAt: time.Now()}
	users = append(users, u)
	return u, nil
}

func findUserLocked(id string) (*User, error) {
	for _, u := range users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, errors.New("user not found")
}

func ListUsers() []*User {
	mu.Lock()
	defer mu.Unlock()
	out := make([]*User, len(users))
	copy(out, users)
	return out
}

// GetCurrencies returns the static list of currencies the service supports.
func GetCurrencies() []Currency {
	return currencies
}

// ---- Account operations ----

// CreateAccount opens a new wallet of `currency` for the given user, labeled
// `name` (e.g. "Savings"). userID must reference an existing User.
func CreateAccount(userID, name, currency string) (*Account, error) {
	if name == "" {
		return nil, errors.New("name is required")
	}
	if _, err := usdPerUnit(currency); err != nil {
		return nil, err
	}
	mu.Lock()
	defer mu.Unlock()
	if userID != "" {
		if _, err := findUserLocked(userID); err != nil {
			return nil, err
		}
	}
	acc := &Account{
		ID:        genID("acc"),
		UserID:    userID,
		Name:      name,
		Type:      "Personal",
		Currency:  currency,
		Balance:   0,
		CreatedAt: time.Now(),
	}
	accounts = append(accounts, acc)
	return acc, nil
}

// CreateVaultAccount seeds the platform's own reserve wallet (unowned by any
// User, Type "Vault"), used to back the dashboard's Vault Balance metric.
func CreateVaultAccount(name, currency string) (*Account, error) {
	if _, err := usdPerUnit(currency); err != nil {
		return nil, err
	}
	mu.Lock()
	defer mu.Unlock()
	acc := &Account{
		ID:        genID("acc"),
		Name:      name,
		Type:      "Vault",
		Currency:  currency,
		Balance:   0,
		CreatedAt: time.Now(),
	}
	accounts = append(accounts, acc)
	return acc, nil
}

func GetAccount(id string) (*Account, error) {
	mu.Lock()
	defer mu.Unlock()
	return findAccountLocked(id)
}

// findAccountLocked assumes mu is already held.
func findAccountLocked(id string) (*Account, error) {
	for _, a := range accounts {
		if a.ID == id {
			return a, nil
		}
	}
	return nil, errors.New("account not found")
}

func ListAccounts() []*Account {
	mu.Lock()
	defer mu.Unlock()
	out := make([]*Account, len(accounts))
	copy(out, accounts)
	return out
}

// joinOwnerLocked assumes mu is already held.
func joinOwnerLocked(a *Account) AccountView {
	view := AccountView{Account: *a}
	if a.UserID != "" {
		if owner, err := findUserLocked(a.UserID); err == nil {
			view.Owner = owner
		}
	}
	return view
}

func ListAccountViews() []AccountView {
	mu.Lock()
	defer mu.Unlock()
	out := make([]AccountView, len(accounts))
	for i, a := range accounts {
		out[i] = joinOwnerLocked(a)
	}
	return out
}

func GetAccountView(id string) (AccountView, error) {
	mu.Lock()
	defer mu.Unlock()
	a, err := findAccountLocked(id)
	if err != nil {
		return AccountView{}, err
	}
	return joinOwnerLocked(a), nil
}

func Deposit(accountID string, amount float64) (*Account, *Transaction, error) {
	if amount <= 0 {
		return nil, nil, errors.New("amount must be greater than zero")
	}
	mu.Lock()
	defer mu.Unlock()
	acc, err := findAccountLocked(accountID)
	if err != nil {
		return nil, nil, err
	}
	acc.Balance = round2(acc.Balance + amount)
	tx := &Transaction{
		ID:           genID("txn"),
		AccountID:    acc.ID,
		Type:         "deposit",
		Amount:       round2(amount),
		Currency:     acc.Currency,
		BalanceAfter: acc.Balance,
		CreatedAt:    time.Now(),
	}
	transactions = append(transactions, tx)
	return acc, tx, nil
}

// ---- Paginated ledger ----

func GetTransactions(accountID string, page, pageSize int) ([]*Transaction, int, error) {
	mu.Lock()
	defer mu.Unlock()
	if _, err := findAccountLocked(accountID); err != nil {
		return nil, 0, err
	}
	var all []*Transaction
	for _, t := range transactions {
		if t.AccountID == accountID {
			all = append(all, t)
		}
	}
	// newest first
	for i, j := 0, len(all)-1; i < j; i, j = i+1, j-1 {
		all[i], all[j] = all[j], all[i]
	}
	total := len(all)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	start := (page - 1) * pageSize
	if start >= total {
		return []*Transaction{}, total, nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return all[start:end], total, nil
}

// ---- Transfers ----

type TransferRequest struct {
	FromAccountID string  `json:"from_account_id"`
	ToAccountID   string  `json:"to_account_id"`
	Amount        float64 `json:"amount"`
}

type TransferResult struct {
	TransferID   string       `json:"transfer_id"`
	From         *Transaction `json:"from"`
	To           *Transaction `json:"to"`
	Rate         float64      `json:"rate"`
	ConvertedAmt float64      `json:"converted_amount"`
	FromAccount  AccountView  `json:"from_account"`
	ToAccount    AccountView  `json:"to_account"`
}

// ErrInsufficientFunds is returned when the source account cannot cover the
// requested transfer amount.
var ErrInsufficientFunds = errors.New("insufficient funds")

// Transfer moves `amount` (denominated in the source account's currency) from
// fromID to toID, converting to the destination currency using the static
// rate table. idempotencyKey deduplicates retried requests: a repeated key
// returns the cached result of the original attempt without moving money
// again.
func Transfer(idempotencyKey string, req TransferRequest) (int, any) {
	if idempotencyKey == "" {
		return 400, gin_H("Idempotency-Key header is required")
	}

	mu.Lock()
	defer mu.Unlock()

	if cached, ok := idempotencyStore[idempotencyKey]; ok {
		return cached.statusCode, cached.body
	}

	status, body := doTransferLocked(req)
	idempotencyStore[idempotencyKey] = idempotentResult{statusCode: status, body: body}
	return status, body
}

func doTransferLocked(req TransferRequest) (int, any) {
	if req.Amount <= 0 {
		return 400, gin_H("amount must be greater than zero")
	}
	if req.FromAccountID == "" || req.ToAccountID == "" {
		return 400, gin_H("from_account_id and to_account_id are required")
	}
	if req.FromAccountID == req.ToAccountID {
		return 400, gin_H("cannot transfer to the same account")
	}
	from, err := findAccountLocked(req.FromAccountID)
	if err != nil {
		return 404, gin_H("from_account_id not found")
	}
	to, err := findAccountLocked(req.ToAccountID)
	if err != nil {
		return 404, gin_H("to_account_id not found")
	}
	if from.Balance < req.Amount {
		return 422, gin_H(ErrInsufficientFunds.Error())
	}

	converted, rate, err := ConvertAmount(req.Amount, from.Currency, to.Currency)
	if err != nil {
		return 400, gin_H(err.Error())
	}

	from.Balance = round2(from.Balance - req.Amount)
	to.Balance = round2(to.Balance + converted)

	transferID := genID("trf")
	now := time.Now()

	outTx := &Transaction{
		ID:               genID("txn"),
		AccountID:        from.ID,
		Type:             "transfer_out",
		Amount:           round2(req.Amount),
		Currency:         from.Currency,
		RelatedAccountID: to.ID,
		TransferID:       transferID,
		Rate:             rate,
		BalanceAfter:     from.Balance,
		CreatedAt:        now,
	}
	inTx := &Transaction{
		ID:               genID("txn"),
		AccountID:        to.ID,
		Type:             "transfer_in",
		Amount:           converted,
		Currency:         to.Currency,
		RelatedAccountID: from.ID,
		TransferID:       transferID,
		Rate:             rate,
		BalanceAfter:     to.Balance,
		CreatedAt:        now,
	}
	transactions = append(transactions, outTx, inTx)

	return 201, TransferResult{
		TransferID:   transferID,
		From:         outTx,
		To:           inTx,
		Rate:         rate,
		ConvertedAmt: converted,
		FromAccount:  joinOwnerLocked(from),
		ToAccount:    joinOwnerLocked(to),
	}
}

// Rate is a structured view of one entry in the static FX table, e.g.
// Base "USD", Quote "NGN", Rate 1550 means 1 USD = 1550 NGN.
type Rate struct {
	Pair  string  `json:"pair"`
	Base  string  `json:"base"`
	Quote string  `json:"quote"`
	Rate  float64 `json:"rate"`
}

// GetRates exposes the static FX table as a structured list, useful for the
// frontend's live conversion preview and the transfer page's rate reference.
func GetRates() []Rate {
	out := make([]Rate, 0, len(staticRates))
	for pair, rate := range staticRates {
		parts := strings.SplitN(pair, "_", 2)
		out = append(out, Rate{Pair: pair, Base: parts[0], Quote: parts[1], Rate: rate})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Pair < out[j].Pair })
	return out
}

type Stats struct {
	TotalAccounts     int     `json:"total_accounts"`
	TotalTransfers    int     `json:"total_transfers"`
	TotalBalanceUSD   float64 `json:"total_balance_usd"`
	VaultBalanceUSD   float64 `json:"vault_balance_usd"`
}

// GetStats aggregates dashboard metrics server-side so the frontend has a
// single source of truth instead of re-deriving totals from paginated data.
// "Platnova Vault" is a seeded account (see main.go) treated as the
// platform's reserve balance for the Vault Balance card.
func GetStats() Stats {
	mu.Lock()
	defer mu.Unlock()

	transferIDs := map[string]struct{}{}
	for _, t := range transactions {
		if t.TransferID != "" {
			transferIDs[t.TransferID] = struct{}{}
		}
	}

	var totalUSD, vaultUSD float64
	accountCount := 0
	for _, a := range accounts {
		usdPer, err := usdPerUnit(a.Currency)
		if err != nil {
			continue
		}
		usdValue := round2(a.Balance * usdPer)
		if a.Type == "Vault" {
			vaultUSD = usdValue
			continue
		}
		accountCount++
		totalUSD += usdValue
	}

	return Stats{
		TotalAccounts:   accountCount,
		TotalTransfers:  len(transferIDs),
		TotalBalanceUSD: round2(totalUSD),
		VaultBalanceUSD: vaultUSD,
	}
}

// gin_H avoids importing gin here to keep service.go framework-agnostic;
// routes.go maps this shape directly to gin.H.
func gin_H(msg string) map[string]string {
	return map[string]string{"error": msg}
}
