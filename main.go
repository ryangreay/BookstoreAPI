package main

import (
	"net/http"
	"strconv"

	"database/sql"

	"os"

	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

var connStr = os.Getenv("DB_URL")

type Book struct {
	Pkbookid   int     `json:"Pkbookid"`
	Bookname   string  `json:"Bookname"`
	Authorname string  `json:"Authorname"`
	Price      float64 `json:"Price"`
}

func main() {
	// Set the router as the default one shipped with Gin
	router := gin.Default()

	// Serve frontend static files
	//UNUSED
	router.Use(static.Serve("/", static.LocalFile("./views", true)))

	// Setup route group for the API
	//UNUSED
	api := router.Group("/api")
	{
		api.GET("/", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "success",
			})
		})
	}

	router.GET("/bookstore/books", books)

	router.POST("/auth/createuser", createUser)
	router.POST("/auth/loginuser", loginUser)
	router.POST("/auth/signoutuser", signOutUser)
	router.DELETE("/auth/deleteuser", deleteUser)

	router.GET("/user/getcash", getCash)
	router.POST("/user/addcash", addCash)
	router.GET("/user/books", userBooks)
	router.POST("/user/addbook", addBook)
	router.DELETE("/user/returnbook", returnBook)

	// Start and run the server
	//read port from env variable in fly.toml
	router.Run(":8080")
}

//Authentication routes /auth/...

// pass in username and password, return an accesstoken
func createUser(c *gin.Context) {
	username := c.Query("username")
	password := c.Query("password")

	//create a guid
	uuid := uuid.NewString()

	//create a user in the database
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	defer db.Close()

	//check if the user already exists in the database
	rows, err := db.Query("SELECT * FROM users WHERE username = $1", username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	defer rows.Close()

	//check if the user already exists in the database
	if rows.Next() {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "User already exists",
		})
		return
	}

	//insert the user into the database
	_, err = db.Exec("INSERT INTO users (username, password, accesstoken, cash) VALUES ($1, $2, $3, 0)", username, password, uuid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"accesstoken": uuid,
	})
}

// pass in username and password, return an accesstoken
func loginUser(c *gin.Context) {
	username := c.Query("username")
	password := c.Query("password")

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	defer db.Close()

	//check if the user exists in the database
	rows, err := db.Query("SELECT * FROM users WHERE username = $1 AND password = $2", username, password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	defer rows.Close()

	//check if the user exists in the database
	if !rows.Next() {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "Invalid username or password",
		})
		return
	}

	//write new accesstoken to the database
	token := uuid.NewString()
	_, err = db.Exec("UPDATE users SET accesstoken = $1 WHERE username = $2 AND password = $3", token, username, password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"accesstoken": token,
	})
}

// pass in accesstoken, return a success or failure message
func signOutUser(c *gin.Context) {
	token := c.GetHeader("Access-Token")

	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "Unauthorized",
		})
		return
	}

	//query the database for the accesstoken
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT * FROM users WHERE accesstoken = $1", token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	defer rows.Close()

	//check if the user exists in the database
	if !rows.Next() {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "Unauthorized",
		})
		return
	}

	//update the accesstoken to null
	_, err = db.Exec("UPDATE users SET accesstoken = null WHERE accesstoken = $1", token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Signed out successfully",
	})
}

// pass in accesstoken, return a success or failure message
func deleteUser(c *gin.Context) {
	token := c.GetHeader("Access-Token")

	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "Unauthorized",
		})
		return
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT * FROM users WHERE accesstoken = $1", token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	defer rows.Close()

	//check if the user exists in the database
	if !rows.Next() {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "Unauthorized",
		})
		return
	}

	//delete the user from the database
	_, err = db.Exec("DELETE FROM users WHERE accesstoken = $1", token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Deleted user successfully",
	})
}

//User routes /user/...

// pass in accesstoken, return the user's cash amount
func getCash(c *gin.Context) {
	token := c.GetHeader("Access-Token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "Unauthorized",
		})
		return
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT cash FROM users WHERE accesstoken = $1", token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	defer rows.Close()

	//check if the user exists in the database
	if !rows.Next() {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "Unauthorized",
		})
		return
	}

	var cash float64
	err = rows.Scan(&cash)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"cash": cash,
	})
}

// pass in accesstoken, cashamount, return a success or failure message
func addCash(c *gin.Context) {
	token := c.GetHeader("Access-Token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "Unauthorized",
		})
		return
	}

	cashamount, err := strconv.ParseFloat(c.Query("cashamount"), 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid cash amount",
		})
		return
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	defer db.Close()

	_, err = db.Exec("UPDATE users SET cash = cash + $1 WHERE accesstoken = $2", cashamount, token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Cash added successfully",
	})
}

// pass in accesstoken, return the user's books
func userBooks(c *gin.Context) {
	token := c.GetHeader("Access-Token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "Unauthorized",
		})
		return
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT b.* FROM books b INNER JOIN userbooks ub ON ub.fkbookid = b.pkbookid INNER JOIN users u ON u.pkuserid = ub.fkuserid WHERE u.accesstoken = $1", token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	defer rows.Close()

	var books []Book
	for rows.Next() {
		var book Book
		err = rows.Scan(&book.Pkbookid, &book.Bookname, &book.Authorname, &book.Price)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
			return
		}
		books = append(books, book)
	}

	//if books is empty, return an empty array
	if len(books) == 0 {
		c.JSON(http.StatusOK, []Book{})
		return
	}

	c.JSON(http.StatusOK, books)
}

// pass in accesstoken, and book id, return a success or failure message
func addBook(c *gin.Context) {
	token := c.GetHeader("Access-Token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "Unauthorized",
		})
		return
	}

	bookId := c.Query("bookId")

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT pkuserid FROM users WHERE accesstoken = $1", token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	defer rows.Close()

	//check if the user exists in the database
	if !rows.Next() {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "Unauthorized",
		})
		return
	}

	var userId int
	err = rows.Scan(&userId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}

	//check if the user already owns the book
	rows, err = db.Query("SELECT * FROM userbooks WHERE fkuserid = $1 AND fkbookid = $2", userId, bookId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	defer rows.Close()

	//check if the user already owns the book
	if rows.Next() {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "User already owns the book",
		})
		return
	}

	//check if the user has enough cash to buy the book
	rows, err = db.Query("SELECT cash, (SELECT price FROM books WHERE pkbookid = $1) as price FROM users WHERE pkuserid = $2", bookId, userId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	defer rows.Close()

	//check if the user has enough cash to buy the book
	var cash float64 = 0
	var price float64 = 0
	if rows.Next() {
		err = rows.Scan(&cash, &price)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
			return
		}
	}

	if cash < price {
		c.JSON(http.StatusPaymentRequired, gin.H{
			"message": "Insufficient funds",
		})
		return
	}

	_, err = db.Exec("UPDATE users set cash = cash - (SELECT price FROM books WHERE pkbookid = $1) WHERE pkuserid = $2", bookId, userId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}

	_, err = db.Exec("INSERT INTO userbooks (fkuserid, fkbookid) VALUES ($2, $1)", bookId, userId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Book added successfully",
	})
}

// pass in accesstoken, and book id, return a success or failure message
func returnBook(c *gin.Context) {
	token := c.GetHeader("Access-Token")

	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "Unauthorized",
		})
		return
	}

	bookId := c.Query("bookId")

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT pkuserid FROM users WHERE accesstoken = $1", token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	defer rows.Close()

	//check if the user exists in the database
	if !rows.Next() {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "Unauthorized",
		})
		return
	}

	var userId int
	err = rows.Scan(&userId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}

	_, err = db.Exec("DELETE FROM userbooks WHERE fkuserid = $1 AND fkbookid = $2", userId, bookId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}

	_, err = db.Exec("UPDATE users set cash = cash + (SELECT price FROM books WHERE pkbookid = $1) WHERE pkuserid = $2", bookId, userId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Book returned successfully",
	})
}

//bookstore routes /bookstore/...

// return all books in the bookstore
func books(c *gin.Context) {
	c.Header("Content-Type", "application/json")
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT * FROM books")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	defer rows.Close()

	var books []Book
	for rows.Next() {
		var book Book
		err = rows.Scan(&book.Pkbookid, &book.Bookname, &book.Authorname, &book.Price)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
			return
		}
		books = append(books, book)
	}

	c.JSON(http.StatusOK, books)
}
