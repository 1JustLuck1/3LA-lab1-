package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB
var jwtKey = []byte("bq3X7Z8k9y2A1B4C5D6E7F8G9H0I1J2K3L4M5N6O7P8Q9R0S")

type User struct {
	ID       uint   `gorm:"primaryKey"`
	Username string `gorm:"unique"`
	Password string
	Role     string `gorm:"default:'user'"`
}

type Book struct {
	ID    uint   `gorm:"primaryKey"`
	Name  string `gorm:"unique"`
	Genre string
	Price float64
}

type Order struct {
	ID        uint `gorm:"primaryKey"`
	UserID    uint
	CreatedAt time.Time
}

type OrderItem struct {
	ID       uint `gorm:"primaryKey"`
	OrderID  uint
	BookID   uint
	Quantity int
}

type Genre struct {
	ID   uint   `gorm:"primaryKey"`
	Name string `gorm:"unique"`
}

type OrderInfo struct {
	OrderItem
	Book
}

func main() {
	dsn := "host=db user=postgres password=2025 dbname=mydb port=5432 sslmode=disable"
	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	//Автомиграция
	db.AutoMigrate(&User{}, &Book{}, &Order{}, &OrderItem{}, &Genre{})

	// Заполнение тестовыми данными
	// db.Create(&User{Username: "user1", Password: "111"})
	// db.Create(&User{Username: "user2", Password: "222"})

	// db.Create(&Genre{Name: "Роман"})
	// db.Create(&Genre{Name: "Повесть"})
	// db.Create(&Genre{Name: "Комедия"})

	// db.Create(&Book{Name: "Война и Мир", Genre: "Роман", Price: 109.99})
	// db.Create(&Book{Name: "Герой нашего времени", Genre: "Роман", Price: 99.99})
	// db.Create(&Book{Name: "Собачье сердце", Genre: "Повесть", Price: 79.99})
	// db.Create(&Book{Name: "Горе от ума", Genre: "Комедия", Price: 89.99})
	// db.Create(&Book{Name: "Преступление и наказание", Genre: "Роман", Price: 99.99})
	// db.Create(&Book{Name: "Капитанская дочка", Genre: "Повесть", Price: 59.99})

	r := gin.Default()
	r.Use(cors.Default())
	// Регистрация
	r.POST("/register", func(c *gin.Context) {
		var user User
		if err := c.ShouldBindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := db.Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "User registered successfully"})
	})
	// Авторизация
	r.POST("/login", func(c *gin.Context) {
		var user User
		if err := c.ShouldBindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		var foundUser User
		if err := db.Where("username = ? AND password = ?", user.Username, user.Password).First(&foundUser).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"user_id": foundUser.ID,
			"exp":     time.Now().Add(time.Minute * 5).Unix(),
		})
		tokenString, err := token.SignedString(jwtKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate token"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"token": tokenString})
	})
	//API для поиска книг
	r.GET("/search", func(c *gin.Context) {
		query := strings.ToLower(c.Query("query"))
		if query == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Пустой запрос"})
			return
		}
		var products []Book
		result := db.Where("LOWER(name) LIKE ?", "%"+query+"%").Find(&products)
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
			return
		}
		c.JSON(http.StatusOK, products)
	})
	//Получение списка всех книг
	r.GET("/books", func(c *gin.Context) {
		var books []Book
		result := db.Find(&books)
		if result.Error != nil {
			c.JSON(500, gin.H{"error": result.Error.Error()})
			return
		}
		c.JSON(200, books)
	})

	//Маршруты для авторизованных пользователей
	auth := r.Group("/auth")
	auth.Use(authMiddleware)
	{
		//Заказы пользователя
		auth.GET("/orders", func(c *gin.Context) {
			userID := c.GetUint("user_id")
			var orders []Order
			if err := db.Where("user_id = ?", userID).Find(&orders).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch orders"})
				return
			}

			c.JSON(http.StatusOK, orders)
		})
		//Детали конкретного заказа
		auth.GET("/orderDetail/:order_id", func(c *gin.Context) {
			orderID := c.Param("order_id")
			var orderItems OrderItem
			if err := db.Where("order_id = ?", orderID).Find(&orderItems).Error; err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "Ошибка при получении состава заказа!"})
				return
			}
			var book Book
			if err := db.Where("id = ?", orderItems.BookID).Find(&book).Error; err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "Ошибка при получении состава заказа!"})
				return
			}
			orderDetails := OrderInfo{
				OrderItem: orderItems,
				Book:      book,
			}
			// fmt.Println("!--- ORDERITTEMS ON SERVER: ", orderItems)
			c.JSON(http.StatusOK, orderDetails)
		})
		//Создание нового заказа
		auth.POST("/createOrder", func(c *gin.Context) {
			var request struct {
				BookName string `json:"bookName"`
				Quantity int    `json:"quantity"`
			}

			if err := c.ShouldBindJSON(&request); err != nil {
				fmt.Println(request)
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			fmt.Println(request)

			userID, exists := c.Get("user_id")
			if !exists {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
				return
			}

			var book Book
			if err := db.Where("name = ?", request.BookName).First(&book).Error; err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "Book not found"})
				return
			}

			order := Order{
				UserID: userID.(uint),
			}
			if err := db.Create(&order).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order"})
				return
			}

			var neworder Order
			if err := db.Where("user_id = ?", userID).Last(&neworder).Error; err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "Book not found"})
				return
			}

			ordrItem := OrderItem{
				OrderID:  neworder.ID,
				BookID:   book.ID,
				Quantity: request.Quantity,
			}
			if err := db.Create(&ordrItem).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order"})
				return
			}

			c.JSON(http.StatusOK, gin.H{"message": "Order created successfully"})
		})
		//Удаление существующего заказа
		auth.DELETE("/deleteOrder/:order_id", func(c *gin.Context) {
			orderID := c.Param("order_id")

			userID, exists := c.Get("user_id")
			if !exists {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
				return
			}

			if err := db.Where("id = ? AND user_id = ?", orderID, userID).Delete(&Order{}).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete order"})
				return
			}

			if err := db.Where("order_id = ?", orderID).Delete(&OrderItem{}).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete order"})
				return
			}

			c.JSON(http.StatusOK, gin.H{"message": "Order deleted successfully"})
		})
	}
	r.Run(":8080")
}

// Middleware для проверки JWT
func authMiddleware(c *gin.Context) {
	tokenString := c.GetHeader("Authorization")
	if tokenString == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		c.Abort()
		return
	}
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		c.Abort()
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
		c.Abort()
		return
	}

	userID, ok := claims["user_id"].(float64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user_id in token"})
		c.Abort()
		return
	}
	c.Set("user_id", uint(userID))
	c.Next()
}
