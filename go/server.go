package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/csv"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	plaid "github.com/plaid/plaid-go/plaid"
)

var (
	PLAID_CLIENT_ID                      = ""
	PLAID_SECRET                         = ""
	PLAID_ENV                            = ""
	PLAID_PRODUCTS                       = ""
	PLAID_COUNTRY_CODES                  = ""
	PLAID_REDIRECT_URI                   = ""
	APP_PORT                             = ""
	client              *plaid.APIClient = nil
	STORE_DATA                           = false
)

var environments = map[string]plaid.Environment{
	"sandbox":     plaid.Sandbox,
	"development": plaid.Development,
	"production":  plaid.Production,
}

func init() {
	// load env vars from .env file
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error when loading environment variables from .env file %w", err)
	}

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
	configuration := plaid.NewConfiguration()
	configuration.AddDefaultHeader("PLAID-CLIENT-ID", PLAID_CLIENT_ID)
	configuration.AddDefaultHeader("PLAID-SECRET", PLAID_SECRET)
	configuration.UseEnvironment(environments[PLAID_ENV])
	client = plaid.NewAPIClient(configuration)

	t := strings.ToLower(os.Getenv("STORE_DATA"))
	STORE_DATA = t == "true" || t == "yes"

	log.Printf("Store data: %v\n", STORE_DATA)

}

func main() {
	r := gin.Default()

	r.Use(firebaseAuth())

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
	r.GET("/api/transfer", transfer)

	err := r.Run(":" + APP_PORT)
	if err != nil {
		panic("unable to start server")
	}
}

// We store the access_token in memory - in production, store it in a secure
// persistent data store.
//var accessToken string
var itemID string

var paymentID string

// The transfer_id is only relevant for the Transfer ACH product.
// We store the transfer_id in memory - in production, store it in a secure
// persistent data store
var transferID string

func accessToken(c *gin.Context) string {
	user, err := users.FindByUID(c.Request.Context(), c.GetString("uid"))
	if err != nil {
		return "not found"
	}

	return user.AccessToken
}

func renderError(c *gin.Context, originalErr error) {
	if plaidError, err := plaid.ToPlaidError(originalErr); err == nil {
		// Return 200 and allow the front end to render the error.
		c.JSON(http.StatusOK, gin.H{"error": plaidError})
		return
	}

	c.JSON(http.StatusInternalServerError, gin.H{"error": originalErr.Error()})
}

func getAccessToken(c *gin.Context) {
	publicToken := c.PostForm("public_token")
	ctx := context.Background()

	// exchange the public_token for an access_token
	exchangePublicTokenResp, _, err := client.PlaidApi.ItemPublicTokenExchange(ctx).ItemPublicTokenExchangeRequest(
		*plaid.NewItemPublicTokenExchangeRequest(publicToken),
	).Execute()
	if err != nil {
		renderError(c, err)
		return
	}

	accessToken := exchangePublicTokenResp.GetAccessToken()
	itemID = exchangePublicTokenResp.GetItemId()
	if itemExists(strings.Split(PLAID_PRODUCTS, ","), "transfer") {
		transferID, err = authorizeAndCreateTransfer(ctx, client, accessToken)
		if err != nil {
			log.Println(err)
		}
	}

	fmt.Println("public token: " + publicToken)
	fmt.Println("access token: " + accessToken)
	fmt.Println("item ID: " + itemID)

	// save the access token in persistent storage
	user := User{
		UID:             c.GetString("uid"),
		AccessToken:     accessToken,
		ItemID:          itemID,
		TokenReceivedAt: time.Now(),
	}

	if err := users.Save(c.Request.Context(), user); err != nil {
		renderError(c, err)
	}

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
	ctx := context.Background()

	// Create payment recipient
	paymentRecipientRequest := plaid.NewPaymentInitiationRecipientCreateRequest("Harry Potter")
	paymentRecipientRequest.SetIban("GB33BUKB20201555555555")
	paymentRecipientRequest.SetAddress(*plaid.NewPaymentInitiationAddress(
		[]string{"4 Privet Drive"},
		"Little Whinging",
		"11111",
		"GB",
	))
	paymentRecipientCreateResp, _, err := client.PlaidApi.PaymentInitiationRecipientCreate(ctx).PaymentInitiationRecipientCreateRequest(*paymentRecipientRequest).Execute()
	if err != nil {
		renderError(c, err)
		return
	}

	// Create payment
	paymentCreateRequest := plaid.NewPaymentInitiationPaymentCreateRequest(
		paymentRecipientCreateResp.GetRecipientId(),
		"paymentRef",
		*plaid.NewPaymentAmount("GBP", 1.34),
	)
	paymentCreateResp, _, err := client.PlaidApi.PaymentInitiationPaymentCreate(ctx).PaymentInitiationPaymentCreateRequest(*paymentCreateRequest).Execute()
	if err != nil {
		renderError(c, err)
		return
	}

	paymentID = paymentCreateResp.GetPaymentId()
	fmt.Println("payment id: " + paymentID)

	linkTokenCreateReqPaymentInitiation := plaid.NewLinkTokenCreateRequestPaymentInitiation(paymentID)
	linkToken, err := linkTokenCreate(linkTokenCreateReqPaymentInitiation)
	if err != nil {
		renderError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"link_token": linkToken,
	})
}

func auth(c *gin.Context) {
	ctx := context.Background()

	authGetResp, _, err := client.PlaidApi.AuthGet(ctx).AuthGetRequest(
		*plaid.NewAuthGetRequest(accessToken(c)),
	).Execute()

	if err != nil {
		renderError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"accounts": authGetResp.GetAccounts(),
		"numbers":  authGetResp.GetNumbers(),
	})
}

func accounts(c *gin.Context) {
	ctx := context.Background()

	accountsGetResp, _, err := client.PlaidApi.AccountsGet(ctx).AccountsGetRequest(
		*plaid.NewAccountsGetRequest(accessToken(c)),
	).Execute()

	if err != nil {
		renderError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"accounts": accountsGetResp.GetAccounts(),
	})
}

func balance(c *gin.Context) {
	ctx := context.Background()

	balancesGetResp, _, err := client.PlaidApi.AccountsBalanceGet(ctx).AccountsBalanceGetRequest(
		*plaid.NewAccountsBalanceGetRequest(accessToken(c)),
	).Execute()

	if err != nil {
		renderError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"accounts": balancesGetResp.GetAccounts(),
	})
}

func item(c *gin.Context) {
	ctx := context.Background()

	itemGetResp, _, err := client.PlaidApi.ItemGet(ctx).ItemGetRequest(
		*plaid.NewItemGetRequest(accessToken(c)),
	).Execute()

	if err != nil {
		renderError(c, err)
		return
	}

	institutionGetByIdResp, _, err := client.PlaidApi.InstitutionsGetById(ctx).InstitutionsGetByIdRequest(
		*plaid.NewInstitutionsGetByIdRequest(
			*itemGetResp.GetItem().InstitutionId.Get(),
			convertCountryCodes(strings.Split(PLAID_COUNTRY_CODES, ",")),
		),
	).Execute()

	if err != nil {
		renderError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"item":        itemGetResp.GetItem(),
		"institution": institutionGetByIdResp.GetInstitution(),
	})
}

func identity(c *gin.Context) {
	ctx := context.Background()

	identityGetResp, _, err := client.PlaidApi.IdentityGet(ctx).IdentityGetRequest(
		*plaid.NewIdentityGetRequest(accessToken(c)),
	).Execute()
	if err != nil {
		renderError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"identity": identityGetResp.GetAccounts(),
	})
}

func transactions(c *gin.Context) {
	const iso8601TimeFormat = "2006-01-02"
	// pull transactions for the past year
	endDate := time.Now().Local().Format(iso8601TimeFormat)
	startDate := time.Now().Local().Add(-365 * 2 * 24 * time.Hour).Format(iso8601TimeFormat)

	count := int32(200)
	offset := int32(0)
	total := int32(-1)

	accounts := make([]plaid.AccountBase, 0)
	transactions := make([]plaid.Transaction, 0)

	log.Printf("Start date: %s\n", startDate)
	log.Printf("End date: %s\n", endDate)

	log.Printf("%10s\t%10s\t%10s\n", "Offset", "Count", "Total")
	log.Printf("%10d\t%10d\t%10d\n", offset, count, total)

	ctx := context.Background()

	accessToken := accessToken(c)

	for total < 0 || offset < total {

		transGetReq := *plaid.NewTransactionsGetRequest(
			accessToken,
			startDate,
			endDate,
		)

		options := *plaid.NewTransactionsGetRequestOptions()
		options.Count = &count
		options.Offset = &offset

		transGetReq.Options = &options

		response, _, err := client.PlaidApi.TransactionsGet(ctx).TransactionsGetRequest(
			transGetReq,
		).Execute()
		//			response, err := client.GetTransactionsWithOptions(accessToken, options)

		if err != nil {
			renderError(c, err)
			return
		}

		if total < 0 {
			// save the accounts only once
			accounts = append(accounts, response.Accounts...)
		}

		transactions = append(transactions, response.Transactions...)

		total = response.TotalTransactions
		offset += count

		log.Printf("%10d\t%10d\t%10d\n", offset, count, total)
	}

	if STORE_DATA {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		user, err := users.FindByUID(ctx, c.GetString("uid"))
		if err != nil {
			renderError(c, err)
			return
		}

		if err := saveToDb(ctx, user, accounts, transactions); err != nil {
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
	ctx := c.Request.Context()

	user, err := users.FindByUID(ctx, c.GetString("uid"))
	if err != nil {
		renderError(c, err)
		return
	}

	all, err := accountsRepo.ListAllForUser(ctx, user)
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
		rec = append(rec, a.AccountId)
		rec = append(rec, fmt.Sprintf("%f", a.Balances.GetAvailable()))
		rec = append(rec, fmt.Sprintf("%f", a.Balances.GetCurrent()))
		rec = append(rec, fmt.Sprintf("%f", a.Balances.GetLimit()))
		rec = append(rec, a.Balances.GetIsoCurrencyCode())
		rec = append(rec, a.Balances.GetUnofficialCurrencyCode())
		rec = append(rec, a.GetMask())
		rec = append(rec, a.Name)
		rec = append(rec, a.GetOfficialName())
		rec = append(rec, string(a.GetSubtype()))
		rec = append(rec, string(a.GetType()))
		rec = append(rec, a.GetVerificationStatus())

		cw.Write(rec)
	}

	cw.Flush()
}

func allTransactionsAsCsv(c *gin.Context) {
	ctx := c.Request.Context()

	user, err := users.FindByUID(ctx, c.GetString("uid"))
	if err != nil {
		renderError(c, err)
		return
	}

	all, err := transactionsRepo.ListAllForUser(ctx, user)
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
		rec = append(rec, t.AccountId)
		rec = append(rec, fmt.Sprintf("%f", t.Amount))
		rec = append(rec, t.GetIsoCurrencyCode())
		rec = append(rec, t.GetUnofficialCurrencyCode())
		rec = append(rec, strings.Join(t.Category, ","))
		rec = append(rec, t.GetCategoryId())
		rec = append(rec, t.Date)
		rec = append(rec, t.GetAuthorizedDate())

		rec = append(rec, t.Location.GetAddress())
		rec = append(rec, t.Location.GetCity())
		rec = append(rec, fmt.Sprintf("%f", t.Location.GetLat()))
		rec = append(rec, fmt.Sprintf("%f", t.Location.GetLon()))
		rec = append(rec, t.Location.GetRegion())
		rec = append(rec, t.Location.GetStoreNumber())
		rec = append(rec, t.Location.GetPostalCode())
		rec = append(rec, t.Location.GetCountry())

		rec = append(rec, t.Name)
		pm := t.GetPaymentMeta()
		rec = append(rec, *pm.ByOrderOf.Get())
		rec = append(rec, pm.GetPayee())
		rec = append(rec, pm.GetPayer())
		rec = append(rec, pm.GetPaymentMethod())
		rec = append(rec, pm.GetPaymentProcessor())
		rec = append(rec, pm.GetPpdId())
		rec = append(rec, pm.GetReason())
		rec = append(rec, pm.GetReferenceNumber())

		rec = append(rec, t.PaymentChannel)
		rec = append(rec, fmt.Sprintf("%v", t.Pending))

		rec = append(rec, t.GetPendingTransactionId())
		rec = append(rec, t.GetAccountOwner())
		rec = append(rec, t.TransactionId)
		rec = append(rec, *t.TransactionType)
		rec = append(rec, string(t.GetTransactionCode()))

		cw.Write(rec)
	}

	cw.Flush()

}

func writeCsvHeaderAccounts(cw *csv.Writer) {

	var rec []string

	rec = addFieldsByJsonTag(
		rec,
		reflect.TypeOf(plaid.AccountBase{}),
		[]string{"AccountId"},
	)

	rec = addFieldsByJsonTag(
		rec,
		reflect.TypeOf(plaid.AccountBalance{}),
		[]string{"Available", "Current", "Limit", "IsoCurrencyCode", "UnofficialCurrencyCode"},
	)

	rec = addFieldsByJsonTag(
		rec,
		reflect.TypeOf(plaid.AccountBase{}),
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
	ctx := context.Background()

	paymentGetResp, _, err := client.PlaidApi.PaymentInitiationPaymentGet(ctx).PaymentInitiationPaymentGetRequest(
		*plaid.NewPaymentInitiationPaymentGetRequest(paymentID),
	).Execute()

	if err != nil {
		renderError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"payment": paymentGetResp,
	})
}

// This functionality is only relevant for the ACH Transfer product.
// Retrieve Transfer for a specified Transfer ID
func transfer(c *gin.Context) {
	ctx := context.Background()

	transferGetResp, _, err := client.PlaidApi.TransferGet(ctx).TransferGetRequest(
		*plaid.NewTransferGetRequest(transferID),
	).Execute()
	if err != nil {
		renderError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"transfer": transferGetResp.GetTransfer(),
	})
}

func investmentTransactions(c *gin.Context) {
	ctx := context.Background()

	endDate := time.Now().Local().Format("2006-01-02")
	startDate := time.Now().Local().Add(-30 * 24 * time.Hour).Format("2006-01-02")

	request := plaid.NewInvestmentsTransactionsGetRequest(accessToken(c), startDate, endDate)
	invTxResp, _, err := client.PlaidApi.InvestmentsTransactionsGet(ctx).InvestmentsTransactionsGetRequest(*request).Execute()

	if err != nil {
		renderError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"investment_transactions": invTxResp,
	})
}

func holdings(c *gin.Context) {
	ctx := context.Background()

	holdingsGetResp, _, err := client.PlaidApi.InvestmentsHoldingsGet(ctx).InvestmentsHoldingsGetRequest(
		*plaid.NewInvestmentsHoldingsGetRequest(accessToken(c)),
	).Execute()
	if err != nil {
		renderError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"holdings": holdingsGetResp,
	})
}

func info(context *gin.Context) {
	context.JSON(http.StatusOK, map[string]interface{}{
		"item_id":      itemID,
		"access_token": accessToken(context),
		"products":     strings.Split(PLAID_PRODUCTS, ","),
	})
}

func createPublicToken(c *gin.Context) {
	ctx := context.Background()

	// Create a one-time use public_token for the Item.
	// This public_token can be used to initialize Link in update mode for a user
	publicTokenCreateResp, _, err := client.PlaidApi.ItemCreatePublicToken(ctx).ItemPublicTokenCreateRequest(
		*plaid.NewItemPublicTokenCreateRequest(accessToken(c)),
	).Execute()

	if err != nil {
		renderError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"public_token": publicTokenCreateResp.GetPublicToken(),
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

func convertCountryCodes(countryCodeStrs []string) []plaid.CountryCode {
	countryCodes := []plaid.CountryCode{}

	for _, countryCodeStr := range countryCodeStrs {
		countryCodes = append(countryCodes, plaid.CountryCode(countryCodeStr))
	}

	return countryCodes
}

func convertProducts(productStrs []string) []plaid.Products {
	products := []plaid.Products{}

	for _, productStr := range productStrs {
		products = append(products, plaid.Products(productStr))
	}

	return products
}

// linkTokenCreate creates a link token using the specified parameters
func linkTokenCreate(
	paymentInitiation *plaid.LinkTokenCreateRequestPaymentInitiation,
) (string, error) {
	ctx := context.Background()
	countryCodes := convertCountryCodes(strings.Split(PLAID_COUNTRY_CODES, ","))
	products := convertProducts(strings.Split(PLAID_PRODUCTS, ","))
	redirectURI := PLAID_REDIRECT_URI

	user := plaid.LinkTokenCreateRequestUser{
		ClientUserId: time.Now().String(),
	}

	request := plaid.NewLinkTokenCreateRequest(
		"Plaid Quickstart",
		"en",
		countryCodes,
		user,
	)

	request.SetProducts(products)

	if redirectURI != "" {
		request.SetRedirectUri(redirectURI)
	}

	if paymentInitiation != nil {
		request.SetPaymentInitiation(*paymentInitiation)
	}

	linkTokenCreateResp, _, err := client.PlaidApi.LinkTokenCreate(ctx).LinkTokenCreateRequest(*request).Execute()

	if err != nil {
		return "", err
	}

	return linkTokenCreateResp.GetLinkToken(), nil
}

func assets(c *gin.Context) {
	ctx := context.Background()

	// create the asset report
	assetReportCreateResp, _, err := client.PlaidApi.AssetReportCreate(ctx).AssetReportCreateRequest(
		*plaid.NewAssetReportCreateRequest([]string{accessToken(c)}, 10),
	).Execute()
	if err != nil {
		renderError(c, err)
		return
	}

	assetReportToken := assetReportCreateResp.GetAssetReportToken()

	// get the asset report
	assetReportGetResp, err := pollForAssetReport(ctx, client, assetReportToken)
	if err != nil {
		renderError(c, err)
		return
	}

	// get it as a pdf
	pdfRequest := plaid.NewAssetReportPDFGetRequest(assetReportToken)
	pdfFile, _, err := client.PlaidApi.AssetReportPdfGet(ctx).AssetReportPDFGetRequest(*pdfRequest).Execute()
	if err != nil {
		renderError(c, err)
		return
	}

	reader := bufio.NewReader(pdfFile)
	content, err := ioutil.ReadAll(reader)
	if err != nil {
		renderError(c, err)
		return
	}

	// convert pdf to base64
	encodedPdf := base64.StdEncoding.EncodeToString(content)

	c.JSON(http.StatusOK, gin.H{
		"json": assetReportGetResp.GetReport(),
		"pdf":  encodedPdf,
	})
}

func pollForAssetReport(ctx context.Context, client *plaid.APIClient, assetReportToken string) (*plaid.AssetReportGetResponse, error) {
	numRetries := 20
	request := plaid.NewAssetReportGetRequest(assetReportToken)

	for i := 0; i < numRetries; i++ {
		response, _, err := client.PlaidApi.AssetReportGet(ctx).AssetReportGetRequest(*request).Execute()
		if err != nil {
			plaidErr, err := plaid.ToPlaidError(err)
			if plaidErr.ErrorCode == "PRODUCT_NOT_READY" {
				time.Sleep(1 * time.Second)
				continue
			} else {
				return nil, err
			}
		} else {
			return &response, nil
		}
	}
	return nil, errors.New("timed out when polling for an asset report")
}

// This is a helper function to authorize and create a Transfer after successful
// exchange of a public_token for an access_token. The transfer_id is then used
// to obtain the data about that particular Transfer.
func authorizeAndCreateTransfer(ctx context.Context, client *plaid.APIClient, accessToken string) (string, error) {
	// We call /accounts/get to obtain first account_id - in production,
	// account_id's should be persisted in a data store and retrieved
	// from there.
	accountsGetResp, _, _ := client.PlaidApi.AccountsGet(ctx).AccountsGetRequest(
		*plaid.NewAccountsGetRequest(accessToken),
	).Execute()

	accountID := accountsGetResp.GetAccounts()[0].AccountId

	transferAuthorizationCreateUser := plaid.NewTransferUserInRequest("FirstName LastName")
	transferAuthorizationCreateRequest := plaid.NewTransferAuthorizationCreateRequest(
		accessToken,
		accountID,
		"credit",
		"ach",
		"1.34",
		"ppd",
		*transferAuthorizationCreateUser,
	)
	transferAuthorizationCreateResp, _, err := client.PlaidApi.TransferAuthorizationCreate(ctx).TransferAuthorizationCreateRequest(*transferAuthorizationCreateRequest).Execute()
	if err != nil {
		return "", err
	}
	authorizationID := transferAuthorizationCreateResp.GetAuthorization().Id

	transferCreateRequest := plaid.NewTransferCreateRequest(
		"1223abc456xyz7890001",
		accessToken,
		accountID,
		authorizationID,
		"credit",
		"ach",
		"1.34",
		"Payment",
		"ppd",
		*transferAuthorizationCreateUser,
	)
	transferCreateResp, _, err := client.PlaidApi.TransferCreate(ctx).TransferCreateRequest(*transferCreateRequest).Execute()
	if err != nil {
		return "", err
	}

	return transferCreateResp.GetTransfer().Id, nil
}

// Helper function to determine if Transfer is in Plaid product array
func itemExists(array []string, product string) bool {
	for _, item := range array {
		if item == product {
			return true
		}
	}

	return false
}
