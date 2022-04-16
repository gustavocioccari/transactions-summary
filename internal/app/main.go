package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Transaction struct {
	ID                string
	Date              time.Time
	TransactionAmount float64
}

type TransactionsByMonth struct {
	January   int `json:"January,omitempty"`
	February  int `json:"February,omitempty"`
	March     int `json:"March,omitempty"`
	April     int `json:"April,omitempty"`
	May       int `json:"May,omitempty"`
	June      int `json:"June,omitempty"`
	July      int `json:"July,omitempty"`
	August    int `json:"August,omitempty"`
	September int `json:"September,omitempty"`
	October   int `json:"October,omitempty"`
	November  int `json:"November,omitempty"`
	December  int `json:"December,omitempty"`
}

type TransactionsSummary struct {
	TotalBalance        float64
	AverageDebitAmount  float64
	AverageCreditAmount float64
	TransactionsByMonth TransactionsByMonth
}

type TransactionType string

const (
	Credit TransactionType = "CREDIT"
	Debit  TransactionType = "DEBIT"
)

func getCsvRows(file *os.File) ([][]string, error) {
	reader := csv.NewReader(file)
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	rowsWithouIndex := rows[1:]
	return rowsWithouIndex, nil
}

func parseTransaction(rows [][]string) ([]Transaction, error) {
	transactions := make([]Transaction, len(rows))

	for index, row := range rows {
		date, err := time.Parse("1/2", row[1])
		if err != nil {
			return nil, err
		}

		transactionAmount, err := strconv.ParseFloat(row[2], 64)
		if err != nil {
			return nil, err
		}

		transactions[index].ID = row[0]
		transactions[index].Date = date
		transactions[index].TransactionAmount = transactionAmount
	}

	return transactions, nil
}

func getTotalBalance(transactions []Transaction) float64 {
	var totalBalance float64

	for _, value := range transactions {
		totalBalance += value.TransactionAmount
	}

	return totalBalance
}

func getAverageAmount(transactions []Transaction, transactionType TransactionType) float64 {
	var sum float64
	var numberOfTransactions float64

	for _, value := range transactions {
		if (transactionType == Credit && value.TransactionAmount > 0.0) ||
			(transactionType == Debit && value.TransactionAmount < 0.0) {
			sum += value.TransactionAmount
			numberOfTransactions += 1
		}
	}
	averageAmount := sum / numberOfTransactions

	return averageAmount
}

func countTransactionsByMonth(transactions []Transaction) (*TransactionsByMonth, error) {
	numberOfTransactionsByMonth := make(map[string]float64, 0)

	for _, transaction := range transactions {
		numberOfTransactionsByMonth[transaction.Date.Month().String()] += 1
	}

	jsonTransactionsByMonth, err := json.Marshal(numberOfTransactionsByMonth)
	if err != nil {
		return nil, err
	}

	transactionsByMonth := TransactionsByMonth{}
	if err := json.Unmarshal(jsonTransactionsByMonth, &transactionsByMonth); err != nil {
		return nil, err
	}

	return &transactionsByMonth, nil
}

func getTransactionsSummary(file *os.File) (*TransactionsSummary, error) {
	rows, err := getCsvRows(file)
	if err != nil {
		return nil, err
	}

	transactions, err := parseTransaction(rows)
	if err != nil {
		return nil, err
	}

	totalBalance := getTotalBalance(transactions)
	averageDebitAmount := getAverageAmount(transactions, Debit)
	averageCreditAmount := getAverageAmount(transactions, Credit)
	transactionsByMonth, err := countTransactionsByMonth(transactions)
	if err != nil {
		return nil, err
	}

	transactionsSummary := TransactionsSummary{
		TotalBalance:        totalBalance,
		AverageDebitAmount:  averageDebitAmount,
		AverageCreditAmount: averageCreditAmount,
		TransactionsByMonth: *transactionsByMonth,
	}

	return &transactionsSummary, nil
}

func formatTransactionsEmail(transactionsSummary TransactionsSummary) (string, error) {
	jsonTransactionsByMonth, err := json.MarshalIndent(transactionsSummary.TransactionsByMonth, "", "\t\t")
	if err != nil {
		return "err", err
	}

	transactionsByMonth := strings.ReplaceAll(string(jsonTransactionsByMonth), "{", "")
	transactionsByMonth = strings.ReplaceAll(transactionsByMonth, "}", "")
	transactionsByMonth = strings.ReplaceAll(transactionsByMonth, `"`, "")
	transactionsByMonth = strings.Replace(transactionsByMonth, "\n", "", 1)

	message := fmt.Sprintf("Subject: Stori transaction summary \n\n"+
		`
Hi customer, here's your transactions summary:
	Total Balance: %.2f
	Average debit amount: %.2f
	Average credit amount: %.2f
	Number of transactions by month:
%s
	`, transactionsSummary.TotalBalance, transactionsSummary.AverageDebitAmount, transactionsSummary.AverageCreditAmount, transactionsByMonth)

	return message, nil
}

func sendEmail(message string, toAddress string) (response bool, err error) {
	fromAddress := os.Getenv("EMAIL")
	fromEmailPassword := os.Getenv("PASSWORD")
	smtpServer := os.Getenv("SMTP_SERVER")
	smptPort := os.Getenv("SMTP_PORT")

	var auth = smtp.PlainAuth("", fromAddress, fromEmailPassword, smtpServer)
	err = smtp.SendMail(smtpServer+":"+smptPort, auth, fromAddress, []string{toAddress}, []byte(message))
	if err != nil {
		return false, err
	}

	return true, nil
}

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

func main() {
	file, err := os.Open("transactions.csv")
	if err != nil {
		log.Fatal(err)
		return
	}

	transactionsSummary, err := getTransactionsSummary(file)
	if err != nil {
		log.Fatal(err)
		return
	}

	log.Println("summary", transactionsSummary)

	message, err := formatTransactionsEmail(*transactionsSummary)
	if err != nil {
		log.Fatal(err)
		return
	}

	_, err = sendEmail(message, os.Getenv("EMAIL"))
	if err != nil {
		log.Fatal(err)
		return
	}
}
