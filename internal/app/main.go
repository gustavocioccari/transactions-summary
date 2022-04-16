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
