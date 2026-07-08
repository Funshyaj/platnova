package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes wires every endpoint from the assessment brief (plus the
// additions documented in the README: GET /accounts, GET /rates, GET /stats,
// GET /users, GET /currencies) to the service.go business logic.
func RegisterRoutes(r *gin.Engine) {
	r.Use(corsMiddleware())

	r.GET("/accounts", listAccountsHandler)
	r.POST("/accounts", createAccountHandler)
	r.GET("/accounts/:id", getAccountHandler)
	r.POST("/accounts/:id/deposit", depositHandler)
	r.GET("/accounts/:id/transactions", getTransactionsHandler)
	r.POST("/transfers", transferHandler)
	r.GET("/rates", ratesHandler)
	r.GET("/stats", statsHandler)
	r.GET("/users", listUsersHandler)
	r.POST("/users", createUserHandler)
	r.GET("/currencies", currenciesHandler)
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Idempotency-Key")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func listAccountsHandler(c *gin.Context) {
	c.JSON(http.StatusOK, ListAccountViews())
}

// createAccountHandler keeps the brief's exact contract ({name, currency})
// while adding an optional user_id to tie the new wallet to an existing
// User. If user_id is omitted, the wallet is created unowned (like the
// platform Vault).
func createAccountHandler(c *gin.Context) {
	var body struct {
		Name     string `json:"name"`
		Currency string `json:"currency"`
		UserID   string `json:"user_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	acc, err := CreateAccount(body.UserID, body.Name, body.Currency)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, acc)
}

func getAccountHandler(c *gin.Context) {
	view, err := GetAccountView(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, view)
}

func depositHandler(c *gin.Context) {
	var body struct {
		Amount float64 `json:"amount"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	acc, tx, err := Deposit(c.Param("id"), body.Amount)
	if err != nil {
		status := http.StatusBadRequest
		if err.Error() == "account not found" {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"account": acc, "transaction": tx})
}

func getTransactionsHandler(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	txs, total, err := GetTransactions(c.Param("id"), page, pageSize)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data":      txs,
		"page":      page,
		"page_size": pageSize,
		"total":     total,
	})
}

func transferHandler(c *gin.Context) {
	var req TransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	key := c.GetHeader("Idempotency-Key")
	status, body := Transfer(key, req)
	c.JSON(status, body)
}

func ratesHandler(c *gin.Context) {
	c.JSON(http.StatusOK, GetRates())
}

func statsHandler(c *gin.Context) {
	c.JSON(http.StatusOK, GetStats())
}

func listUsersHandler(c *gin.Context) {
	c.JSON(http.StatusOK, ListUsers())
}

func createUserHandler(c *gin.Context) {
	var body struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	u, err := CreateUser(body.Name, body.Email)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, u)
}

func currenciesHandler(c *gin.Context) {
	c.JSON(http.StatusOK, GetCurrencies())
}
