package main

import (
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const jwtSecret = "change-this-secret-in-real-project"

type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Username  string    `gorm:"uniqueIndex;size:64;not null" json:"username"`
	Password  string    `gorm:"size:255;not null" json:"-"`
	Email     string    `gorm:"uniqueIndex;size:128;not null" json:"email"`
	Posts     []Post    `json:"-"`
	Comments  []Comment `json:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Post struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Title     string    `gorm:"size:200;not null" json:"title"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	User      User      `json:"author,omitempty"`
	Comments  []Comment `json:"comments,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Comment struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	PostID    uint      `gorm:"index;not null" json:"post_id"`
	User      User      `json:"author,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Claims struct {
	UserID uint `json:"user_id"`
	jwt.RegisteredClaims
}

type APIError struct {
	Message string `json:"message"`
}

func respondError(c *gin.Context, code int, msg string) {
	c.JSON(code, APIError{Message: msg})
}

func issueToken(userID uint) (string, error) {
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtSecret))
}

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			respondError(c, http.StatusUnauthorized, "missing or invalid Authorization header")
			c.Abort()
			return
		}

		rawToken := strings.TrimPrefix(header, "Bearer ")
		parsed, err := jwt.ParseWithClaims(rawToken, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(jwtSecret), nil
		})
		if err != nil || !parsed.Valid {
			respondError(c, http.StatusUnauthorized, "invalid token")
			c.Abort()
			return
		}

		claims, ok := parsed.Claims.(*Claims)
		if !ok {
			respondError(c, http.StatusUnauthorized, "invalid token claims")
			c.Abort()
			return
		}

		c.Set("userID", claims.UserID)
		c.Next()
	}
}

func getUserID(c *gin.Context) uint {
	v, _ := c.Get("userID")
	userID, _ := v.(uint)
	return userID
}

func main() {
	db, err := gorm.Open(sqlite.Open("blog_homework04.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect db: %v", err)
	}
	if err := db.AutoMigrate(&User{}, &Post{}, &Comment{}); err != nil {
		log.Fatalf("failed to migrate: %v", err)
	}

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	r.POST("/register", func(c *gin.Context) {
		var req struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required,min=6"`
			Email    string `json:"email" binding:"required,email"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			respondError(c, http.StatusBadRequest, err.Error())
			return
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			respondError(c, http.StatusInternalServerError, "failed to hash password")
			return
		}

		user := User{
			Username: req.Username,
			Password: string(hash),
			Email:    req.Email,
		}
		if err := db.Create(&user).Error; err != nil {
			respondError(c, http.StatusBadRequest, "username or email already exists")
			return
		}

		c.JSON(http.StatusCreated, gin.H{"id": user.ID, "username": user.Username, "email": user.Email})
	})

	r.POST("/login", func(c *gin.Context) {
		var req struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			respondError(c, http.StatusBadRequest, err.Error())
			return
		}

		var user User
		if err := db.Where("username = ?", req.Username).First(&user).Error; err != nil {
			respondError(c, http.StatusUnauthorized, "invalid username or password")
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
			respondError(c, http.StatusUnauthorized, "invalid username or password")
			return
		}

		token, err := issueToken(user.ID)
		if err != nil {
			respondError(c, http.StatusInternalServerError, "failed to generate token")
			return
		}
		c.JSON(http.StatusOK, gin.H{"token": token})
	})

	r.GET("/posts", func(c *gin.Context) {
		var posts []Post
		if err := db.Preload("User").Find(&posts).Error; err != nil {
			respondError(c, http.StatusInternalServerError, "failed to query posts")
			return
		}
		c.JSON(http.StatusOK, posts)
	})

	r.GET("/posts/:id", func(c *gin.Context) {
		var post Post
		if err := db.Preload("User").Preload("Comments").Preload("Comments.User").First(&post, c.Param("id")).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				respondError(c, http.StatusNotFound, "post not found")
				return
			}
			respondError(c, http.StatusInternalServerError, "failed to query post")
			return
		}
		c.JSON(http.StatusOK, post)
	})

	auth := r.Group("/")
	auth.Use(authMiddleware())

	auth.POST("/posts", func(c *gin.Context) {
		var req struct {
			Title   string `json:"title" binding:"required"`
			Content string `json:"content" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			respondError(c, http.StatusBadRequest, err.Error())
			return
		}
		post := Post{
			Title:   req.Title,
			Content: req.Content,
			UserID:  getUserID(c),
		}
		if err := db.Create(&post).Error; err != nil {
			respondError(c, http.StatusInternalServerError, "failed to create post")
			return
		}
		c.JSON(http.StatusCreated, post)
	})

	auth.PUT("/posts/:id", func(c *gin.Context) {
		var post Post
		if err := db.First(&post, c.Param("id")).Error; err != nil {
			respondError(c, http.StatusNotFound, "post not found")
			return
		}
		if post.UserID != getUserID(c) {
			respondError(c, http.StatusForbidden, "you can only update your own post")
			return
		}

		var req struct {
			Title   string `json:"title" binding:"required"`
			Content string `json:"content" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			respondError(c, http.StatusBadRequest, err.Error())
			return
		}

		post.Title = req.Title
		post.Content = req.Content
		if err := db.Save(&post).Error; err != nil {
			respondError(c, http.StatusInternalServerError, "failed to update post")
			return
		}
		c.JSON(http.StatusOK, post)
	})

	auth.DELETE("/posts/:id", func(c *gin.Context) {
		var post Post
		if err := db.First(&post, c.Param("id")).Error; err != nil {
			respondError(c, http.StatusNotFound, "post not found")
			return
		}
		if post.UserID != getUserID(c) {
			respondError(c, http.StatusForbidden, "you can only delete your own post")
			return
		}
		if err := db.Delete(&post).Error; err != nil {
			respondError(c, http.StatusInternalServerError, "failed to delete post")
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "post deleted"})
	})

	auth.POST("/posts/:id/comments", func(c *gin.Context) {
		var post Post
		if err := db.First(&post, c.Param("id")).Error; err != nil {
			respondError(c, http.StatusNotFound, "post not found")
			return
		}

		var req struct {
			Content string `json:"content" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			respondError(c, http.StatusBadRequest, err.Error())
			return
		}

		comment := Comment{
			Content: req.Content,
			UserID:  getUserID(c),
			PostID:  post.ID,
		}
		if err := db.Create(&comment).Error; err != nil {
			respondError(c, http.StatusInternalServerError, "failed to create comment")
			return
		}
		c.JSON(http.StatusCreated, comment)
	})

	r.GET("/posts/:id/comments", func(c *gin.Context) {
		var comments []Comment
		if err := db.Where("post_id = ?", c.Param("id")).Preload("User").Find(&comments).Error; err != nil {
			respondError(c, http.StatusInternalServerError, "failed to query comments")
			return
		}
		c.JSON(http.StatusOK, comments)
	})

	log.Println("server started at http://localhost:8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
