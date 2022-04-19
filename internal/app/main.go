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

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
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

func getS3File(objectKey string) (*os.File, error) {
	sess := session.Must(session.NewSession(&aws.Config{
		Credentials:      credentials.NewStaticCredentials(os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY"), ""),
		S3ForcePathStyle: aws.Bool(true),
		Region:           aws.String(endpoints.UsEast1RegionID),
		Endpoint:         aws.String("http://localstack:4566"),
	}))

	s3sess := s3.New(sess, &aws.Config{})

	_, err := s3sess.GetObject(&s3.GetObjectInput{
		Bucket: aws.String("transactions-bucket"),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		return nil, err
	}

	downloader := s3manager.NewDownloader(sess)

	file, err := os.Create("/tmp/" + objectKey)

	_, err = downloader.Download(file,
		&s3.GetObjectInput{
			Bucket: aws.String("transactions-bucket"),
			Key:    aws.String(objectKey),
		})
	if err != nil {
		return nil, err
	}

	return file, nil
}

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

func handler(s3Event events.S3Event) {
	var files []os.File

	for _, record := range s3Event.Records {
		file, err := getS3File(record.S3.Object.Key)
		if err != nil {
			log.Println(err)
			continue
		}

		files = append(files, *file)
	}

	for _, file := range files {
		transactionsSummary, err := getTransactionsSummary(&file)
		if err != nil {
			log.Println(err)
			continue
		}

		message, err := formatTransactionsEmail(*transactionsSummary)
		if err != nil {
			log.Println(err)
			continue
		}

		_, err = sendEmail(message, os.Getenv("EMAIL"))
		if err != nil {
			log.Println(err)
			continue
		}
	}

	return
}

func main() {
	lambda.Start(handler)
}
