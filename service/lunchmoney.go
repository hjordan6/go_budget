package service

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/hjordan6/go_budget/models"
	"gorm.io/gorm"
)

// lunchMoneyBaseURL is the stable v1 API root. Transactions are fetched with a
// Bearer access token the user pastes into the app's Settings screen.
const lunchMoneyBaseURL = "https://dev.lunchmoney.app/v1"

// syncWindowDays is how far back each sync looks. Rows already staged are left
// untouched (see SyncUser), so a rolling window is enough to catch new activity.
const syncWindowDays = 30

// LMTransaction is one transaction as returned by the Lunch Money v1 API. Only
// the fields we care about are decoded.
type LMTransaction struct {
	ID       int64  `json:"id"`
	Date     string `json:"date"`
	Payee    string `json:"payee"`
	Amount   string `json:"amount"` // numeric string; positive = expense in LM's convention
	Currency string `json:"currency"`
	Notes    string `json:"notes"`
	Status   string `json:"status"`
}

type lmTransactionsResponse struct {
	Transactions []LMTransaction `json:"transactions"`
}

// AppAmount converts a Lunch Money amount into this app's signed convention
// (negative = expense). Lunch Money reports expenses as positive numbers and
// income as negative, so we flip the sign.
func (t LMTransaction) AppAmount() (float64, error) {
	v, err := strconv.ParseFloat(t.Amount, 64)
	if err != nil {
		return 0, err
	}
	return -v, nil
}

// FetchLunchMoneyTransactions pulls transactions in [start, end] for the given
// access token.
func FetchLunchMoneyTransactions(token string, start, end time.Time) ([]LMTransaction, error) {
	q := url.Values{}
	q.Set("start_date", start.Format("2006-01-02"))
	q.Set("end_date", end.Format("2006-01-02"))
	endpoint := fmt.Sprintf("%s/transactions?%s", lunchMoneyBaseURL, q.Encode())

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("lunch money API returned status %d", resp.StatusCode)
	}

	var out lmTransactionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Transactions, nil
}

// SyncAllLunchMoney runs a sync for every user who has connected a token. It logs
// and continues past individual failures so one bad token doesn't block others.
func SyncAllLunchMoney(db *gorm.DB) {
	var users []models.User
	if err := db.Where("lunch_money_token <> ''").Find(&users).Error; err != nil {
		log.Printf("lunchmoney sync: load users: %v", err)
		return
	}
	for _, u := range users {
		added, err := SyncUserLunchMoney(db, u)
		if err != nil {
			log.Printf("lunchmoney sync: user %d: %v", u.ID, err)
			continue
		}
		if added > 0 {
			log.Printf("lunchmoney sync: user %d: %d new transactions staged", u.ID, added)
		}
	}
}

// SyncUserLunchMoney fetches the user's recent transactions and stages any that
// aren't already present, returning the number of newly staged rows. Existing
// rows are never overwritten, so user edits and review decisions are preserved.
func SyncUserLunchMoney(db *gorm.DB, u models.User) (int, error) {
	if u.LunchMoneyToken == "" {
		return 0, nil
	}
	end := time.Now()
	start := end.AddDate(0, 0, -syncWindowDays)

	lmTxns, err := FetchLunchMoneyTransactions(u.LunchMoneyToken, start, end)
	if err != nil {
		return 0, err
	}

	added := 0
	for _, lt := range lmTxns {
		amount, err := lt.AppAmount()
		if err != nil {
			log.Printf("lunchmoney sync: user %d txn %d: bad amount %q", u.ID, lt.ID, lt.Amount)
			continue
		}
		row := models.LunchMoneyTransaction{
			UserID:       u.ID,
			LunchMoneyID: lt.ID,
			Date:         lt.Date,
			Payee:        lt.Payee,
			Amount:       amount,
			Currency:     lt.Currency,
			Notes:        lt.Notes,
			Status:       models.LMStatusPending,
		}
		// Insert only when this (user, lunch_money_id) pair is new. Attrs are
		// applied on create; on a hit we leave the existing row as-is.
		res := db.Where(models.LunchMoneyTransaction{UserID: u.ID, LunchMoneyID: lt.ID}).
			Attrs(row).FirstOrCreate(&models.LunchMoneyTransaction{})
		if res.Error != nil {
			return added, res.Error
		}
		added += int(res.RowsAffected)
	}
	return added, nil
}

// StartLunchMoneySync kicks off a background goroutine that syncs shortly after
// startup and then hourly. It returns immediately.
func StartLunchMoneySync(db *gorm.DB) {
	go func() {
		time.Sleep(10 * time.Second) // let the server settle before the first pull
		SyncAllLunchMoney(db)
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			SyncAllLunchMoney(db)
		}
	}()
}
