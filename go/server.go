package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/plaid/plaid-go/plaid"
)

var (
	PLAID_CLIENT_ID                   = ""
	PLAID_SECRET                      = ""
	PLAID_ENV                         = ""
	PLAID_PRODUCTS                    = ""
	PLAID_COUNTRY_CODES               = ""
	PLAID_REDIRECT_URI                = ""
	APP_PORT                          = ""
	client              *plaid.Client = nil
	STORE_DATA                        = false
)

var environments = map[string]plaid.Environment{
	"sandbox":     plaid.Sandbox,
	"development": plaid.Development,
	"production":  plaid.Production,
}

func init() {
	// load env vars from .env file
	err := godotenv.Load()

	// set constants from env
	PLAID_CLIENT_ID = os.Getenv("PLAID_CLIENT_ID")
	PLAID_SECRET = os.Getenv("PLAID_SECRET")

	if PLAID_CLIENT_ID == "" || PLAID_SECRET == "" {
		log.Fatal("Error: PLAID_SECRET or PLAID_CLIENT_ID is not set. Did you copy .env.example to .env and fill it out?")
	}

	PLAID_ENV = os.Getenv("PLAID_ENV")
	PLAID_PRODUCTS = os.Getenv("PLAID_PRODUCTS")
	PLAID_COUNTRY_CODES = os.Getenv("PLAID_COUNTRY_CODES")
	PLAID_REDIRECT_URI = os.Getenv("PLAID_REDIRECT_URI")
	APP_PORT = os.Getenv("APP_PORT")

	// set defaults
	if PLAID_PRODUCTS == "" {
		PLAID_PRODUCTS = "transactions"
	}
	if PLAID_COUNTRY_CODES == "" {
		PLAID_COUNTRY_CODES = "US"
	}
	if PLAID_ENV == "" {
		PLAID_ENV = "sandbox"
	}
	if APP_PORT == "" {
		APP_PORT = "8000"
	}
	if PLAID_CLIENT_ID == "" {
		log.Fatal("PLAID_CLIENT_ID is not set. Make sure to fill out the .env file")
	}
	if PLAID_SECRET == "" {
		log.Fatal("PLAID_SECRET is not set. Make sure to fill out the .env file")
	}

	// create Plaid client
	client, err = plaid.NewClient(plaid.ClientOptions{
		PLAID_CLIENT_ID,
		PLAID_SECRET,
		environments[PLAID_ENV],
		&http.Client{},
	})
	if err != nil {
		panic(fmt.Errorf("unexpected error while initializing plaid client %w", err))
	}

	t := os.Getenv("STORE_DATA")
	STORE_DATA = "true" == strings.ToLower(t) || "yes" == strings.ToLower(t)

	log.Printf("Store data: %v\n", STORE_DATA)
}

func main() {
	r := gin.Default()

	r.POST("/api/info", info)

	// For OAuth flows, the process looks as follows.
	// 1. Create a link token with the redirectURI (as white listed at https://dashboard.plaid.com/team/api).
	// 2. Once the flow succeeds, Plaid Link will redirect to redirectURI with
	// additional parameters (as required by OAuth standards and Plaid).
	// 3. Re-initialize with the link token (from step 1) and the full received redirect URI
	// from step 2.

	r.POST("/api/set_access_token", getAccessToken)
	r.POST("/api/create_link_token_for_payment", createLinkTokenForPayment)
	r.GET("/api/auth", auth)
	r.GET("/api/accounts", accounts)
	r.GET("/api/balance", balance)
	r.GET("/api/item", item)
	r.POST("/api/item", item)
	r.GET("/api/identity", identity)
	r.GET("/api/transactions", transactions)
	r.POST("/api/transactions", transactions)
	r.GET("/api/payment", payment)
	r.GET("/api/create_public_token", createPublicToken)
	r.POST("/api/create_link_token", createLinkToken)
	r.GET("/api/investment_transactions", investmentTransactions)
	r.GET("/api/holdings", holdings)
	r.GET("/api/assets", assets)
	r.GET("/api/all/transactions/csv", allTransactionsAsCsv)
	r.GET("/api/all/balances/csv", allAccountsAsCsv)

	err := r.Run(":" + APP_PORT)
	if err != nil {
		panic("unable to start server")
	}
}

// We store the access_token in memory - in production, store it in a secure
// persistent data store.
var accessToken string
var itemID string

var paymentID string

func renderError(c *gin.Context, err error) {
	if plaidError, ok := err.(plaid.Error); ok {
		// Return 200 and allow the front end to render the error.
		c.JSON(http.StatusOK, gin.H{"error": plaidError})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}

func getAccessToken(c *gin.Context) {
	publicToken := c.PostForm("public_token")
	response, err := client.ExchangePublicToken(publicToken)
	if err != nil {
		renderError(c, err)
		return
	}
	accessToken = response.AccessToken
	itemID = response.ItemID

	fmt.Println("public token: " + publicToken)
	fmt.Println("access token: " + accessToken)
	fmt.Println("item ID: " + itemID)

	c.JSON(http.StatusOK, gin.H{
		"access_token": accessToken,
		"item_id":      itemID,
	})
}

// This functionality is only relevant for the UK Payment Initiation product.
// Creates a link token configured for payment initiation. The payment
// information will be associated with the link token, and will not have to be
// passed in again when we initialize Plaid Link.
func createLinkTokenForPayment(c *gin.Context) {
	recipientCreateResp, err := client.CreatePaymentRecipient(
		"Harry Potter",
		"GB33BUKB20201555555555",
		&plaid.PaymentRecipientAddress{
			Street:     []string{"4 Privet Drive"},
			City:       "Little Whinging",
			PostalCode: "11111",
			Country:    "GB",
		})
	if err != nil {
		renderError(c, err)
		return
	}
	paymentCreateResp, err := client.CreatePayment(recipientCreateResp.RecipientID, "paymentRef", plaid.PaymentAmount{
		Currency: "GBP",
		Value:    12.34,
	})
	if err != nil {
		renderError(c, err)
		return
	}
	paymentID = paymentCreateResp.PaymentID
	fmt.Println("payment id: " + paymentID)

	linkToken, tokenCreateErr := linkTokenCreate(&plaid.PaymentInitiation{
		PaymentID: paymentID,
	})
	if tokenCreateErr != nil {
		renderError(c, tokenCreateErr)
	}
	c.JSON(http.StatusOK, gin.H{
		"link_token": linkToken,
	})
}

func auth(c *gin.Context) {
	response, err := client.GetAuth(accessToken)
	if err != nil {
		renderError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"accounts": response.Accounts,
		"numbers":  response.Numbers,
	})
}

func accounts(c *gin.Context) {
	response, err := client.GetAccounts(accessToken)
	if err != nil {
		renderError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"accounts": response.Accounts,
	})
}

func balance(c *gin.Context) {
	response, err := client.GetBalances(accessToken)
	if err != nil {
		renderError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"accounts": response.Accounts,
	})
}

func item(c *gin.Context) {
	response, err := client.GetItem(accessToken)
	if err != nil {
		renderError(c, err)
		return
	}

	institution, err := client.GetInstitutionByID(response.Item.InstitutionID)
	if err != nil {
		renderError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"item":        response.Item,
		"institution": institution.Institution,
	})
}

func identity(c *gin.Context) {
	response, err := client.GetIdentity(accessToken)
	if err != nil {
		renderError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"identity": response.Accounts,
	})
}

func transactions(c *gin.Context) {
	const iso8601TimeFormat = "2006-01-02"
	// pull transactions for the past year
	endDate := time.Now().Local().Format(iso8601TimeFormat)
	startDate := time.Now().Local().Add(-365 * 24 * time.Hour).Format(iso8601TimeFormat)

	count := 200
	offset := 0
	total := -1

	accounts := make([]plaid.Account, 0)
	transactions := make([]plaid.Transaction, 0)

	log.Printf("Start date: %s\n", startDate)
	log.Printf("End date: %s\n", endDate)

	log.Printf("%10s\t%10s\t%10s\n", "Offset", "Count", "Total")
	log.Printf("%10d\t%10d\t%10d\n", offset, count, total)

	for total < 0 || offset < total {

		options := plaid.GetTransactionsOptions{
			StartDate: startDate,
			EndDate:   endDate,
			Count:     count,
			Offset:    offset,
		}

		response, err := client.GetTransactionsWithOptions(accessToken, options)

		if err != nil {
			renderError(c, err)
			return
		}

		accounts = append(accounts, response.Accounts...)
		transactions = append(transactions, response.Transactions...)

		total = response.TotalTransactions
		offset += count

		log.Printf("%10d\t%10d\t%10d\n", offset, count, total)
	}

	if STORE_DATA {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := saveToDb(ctx, accounts, transactions); err != nil {
			renderError(c, err)
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"accounts":     accounts,
		"transactions": transactions,
	})
}

func allAccountsAsCsv(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	all, err := fetchAllAccounts(ctx)
	if err != nil {
		renderError(c, err)
		return
	}

	c.Header("Content-Type", "text/csv")

	cw := csv.NewWriter(c.Writer)
	cw.Comma = '#'

	writeCsvHeaderAccounts(cw)

	for _, a := range all {
		var rec []string
		rec = append(rec, a.AccountID)
		rec = append(rec, fmt.Sprintf("%f", a.Balances.Available))
		rec = append(rec, fmt.Sprintf("%f", a.Balances.Current))
		rec = append(rec, fmt.Sprintf("%f", a.Balances.Limit))
		rec = append(rec, a.Balances.ISOCurrencyCode)
		rec = append(rec, a.Balances.UnofficialCurrencyCode)
		rec = append(rec, a.Mask)
		rec = append(rec, a.Name)
		rec = append(rec, a.OfficialName)
		rec = append(rec, a.Subtype)
		rec = append(rec, a.Type)
		rec = append(rec, a.VerificationStatus)

		cw.Write(rec)
	}

	cw.Flush()
}

func allTransactionsAsCsv(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	all, err := fetchAllTransactions(ctx)
	if err != nil {
		renderError(c, err)
		return
	}

	c.Header("Content-Type", "text/csv")

	cw := csv.NewWriter(c.Writer)
	cw.Comma = '#'
	// write Header
	writeCsvHeaderTransactions(cw)
	// write records
	for _, t := range all {
		var rec []string
		rec = append(rec, t.AccountID)
		rec = append(rec, fmt.Sprintf("%f", t.Amount))
		rec = append(rec, t.ISOCurrencyCode)
		rec = append(rec, t.UnofficialCurrencyCode)
		rec = append(rec, strings.Join(t.Category, ","))
		rec = append(rec, t.CategoryID)
		rec = append(rec, t.Date)
		rec = append(rec, t.AuthorizedDate)

		rec = append(rec, t.Location.Address)
		rec = append(rec, t.Location.City)
		rec = append(rec, fmt.Sprintf("%f", t.Location.Lat))
		rec = append(rec, fmt.Sprintf("%f", t.Location.Lon))
		rec = append(rec, t.Location.Region)
		rec = append(rec, t.Location.StoreNumber)
		rec = append(rec, t.Location.PostalCode)
		rec = append(rec, t.Location.Country)

		rec = append(rec, t.Name)
		rec = append(rec, t.PaymentMeta.ByOrderOf)
		rec = append(rec, t.PaymentMeta.Payee)
		rec = append(rec, t.PaymentMeta.Payer)
		rec = append(rec, t.PaymentMeta.PaymentMethod)
		rec = append(rec, t.PaymentMeta.PaymentProcessor)
		rec = append(rec, t.PaymentMeta.PPDID)
		rec = append(rec, t.PaymentMeta.Reason)
		rec = append(rec, t.PaymentMeta.ReferenceNumber)

		rec = append(rec, t.PaymentChannel)
		rec = append(rec, fmt.Sprintf("%v", t.Pending))

		rec = append(rec, t.PendingTransactionID)
		rec = append(rec, t.AccountOwner)
		rec = append(rec, t.ID)
		rec = append(rec, t.Type)
		rec = append(rec, t.Code)

		cw.Write(rec)
	}

	cw.Flush()

}

func writeCsvHeaderAccounts(cw *csv.Writer) {

	var rec []string

	rec = addFieldsByJsonTag(
		rec,
		reflect.TypeOf(plaid.Account{}),
		[]string{"AccountID"},
	)

	rec = addFieldsByJsonTag(
		rec,
		reflect.TypeOf(plaid.AccountBalances{}),
		[]string{"Available", "Current", "Limit", "ISOCurrencyCode", "UnofficialCurrencyCode"},
	)

	rec = addFieldsByJsonTag(
		rec,
		reflect.TypeOf(plaid.Account{}),
		[]string{"Mask", "Name", "OfficialName", "Subtype", "Type", "VerificationStatus"},
	)

	cw.Write(rec)
}

func writeCsvHeaderTransactions(cw *csv.Writer) {
	var rec []string

	rec = addFieldsByJsonTag(
		rec,
		reflect.TypeOf(plaid.Transaction{}),
		[]string{"AccountID", "Amount", "ISOCurrencyCode", "UnofficialCurrencyCode", "Category", "CategoryID", "Date", "AuthorizedDate"},
	)

	rec = addFieldsByJsonTag(
		rec,
		reflect.TypeOf(plaid.Location{}),
		[]string{"Address", "City", "Lat", "Lon", "Region", "StoreNumber", "PostalCode", "Country"},
	)

	rec = addFieldsByJsonTag(
		rec,
		reflect.TypeOf(plaid.Transaction{}),
		[]string{"Name"},
	)

	rec = addFieldsByJsonTag(
		rec,
		reflect.TypeOf(plaid.PaymentMeta{}),
		[]string{"ByOrderOf", "Payee", "Payer", "PaymentMethod", "PaymentProcessor", "PPDID", "Reason", "ReferenceNumber"},
	)

	rec = addFieldsByJsonTag(
		rec,
		reflect.TypeOf(plaid.Transaction{}),
		[]string{"PaymentChannel", "Pending", "PendingTransactionID", "AccountOwner", "ID", "Type", "Code"},
	)

	cw.Write(rec)
}

func addFieldsByJsonTag(rec []string, tType reflect.Type, fields []string) []string {
	for _, fName := range fields {
		f, ok := tType.FieldByName(fName)
		if !ok {
			rec = append(rec, fName)
		} else {
			rec = append(rec, f.Tag.Get("json"))
		}
	}

	return rec
}

// This functionality is only relevant for the UK Payment Initiation product.
// Retrieve Payment for a specified Payment ID
func payment(c *gin.Context) {
	response, err := client.GetPayment(paymentID)
	if err != nil {
		renderError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"payment": response.Payment,
	})
}

func investmentTransactions(c *gin.Context) {
	endDate := time.Now().Local().Format("2006-01-02")
	startDate := time.Now().Local().Add(-30 * 24 * time.Hour).Format("2006-01-02")
	response, err := client.GetInvestmentTransactions(accessToken, startDate, endDate)
	fmt.Println("error", err)
	if err != nil {
		renderError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"investment_transactions": response,
	})
}

func holdings(c *gin.Context) {
	response, err := client.GetHoldings(accessToken)
	if err != nil {
		renderError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"holdings": response,
	})
}

func info(context *gin.Context) {
	context.JSON(200, map[string]interface{}{
		"item_id":      itemID,
		"access_token": accessToken,
		"products":     strings.Split(PLAID_PRODUCTS, ","),
	})
}

func createPublicToken(c *gin.Context) {
	// Create a one-time use public_token for the Item.
	// This public_token can be used to initialize Link in update mode for a user
	publicToken, err := client.CreatePublicToken(accessToken)
	if err != nil {
		renderError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"public_token": publicToken,
	})
}

func createLinkToken(c *gin.Context) {
	linkToken, err := linkTokenCreate(nil)
	if err != nil {
		renderError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"link_token": linkToken})
}

type httpError struct {
	errorCode int
	error     string
}

func (httpError *httpError) Error() string {
	return httpError.error
}

// linkTokenCreate creates a link token using the specified parameters
func linkTokenCreate(
	paymentInitiation *plaid.PaymentInitiation,
) (string, *httpError) {
	countryCodes := strings.Split(PLAID_COUNTRY_CODES, ",")
	products := strings.Split(PLAID_PRODUCTS, ",")
	redirectURI := PLAID_REDIRECT_URI
	configs := plaid.LinkTokenConfigs{
		User: &plaid.LinkTokenUser{
			// This should correspond to a unique id for the current user.
			ClientUserID: "user-id",
		},
		ClientName:        "Plaid Quickstart",
		Products:          products,
		CountryCodes:      countryCodes,
		Language:          "en",
		RedirectUri:       redirectURI,
		PaymentInitiation: paymentInitiation,
	}
	resp, err := client.CreateLinkToken(configs)
	if err != nil {
		return "", &httpError{
			errorCode: http.StatusBadRequest,
			error:     err.Error(),
		}
	}
	return resp.LinkToken, nil
}

func assets(c *gin.Context) {
	c.JSON(http.StatusBadRequest, gin.H{"error": "unfortunately the go client library does not support assets report creation yet."})
}
