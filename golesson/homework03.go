package main

import (
	"fmt"
	"log"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// User 与 Post: 一对多（一个用户有多篇文章）
type User struct {
	ID        uint `gorm:"primaryKey"`
	Name      string
	Email     string `gorm:"uniqueIndex"`
	PostCount int    `gorm:"default:0"` // 已发布文章数（由 Post.AfterCreate 维护）
	Posts     []Post
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Post 与 Comment: 一对多（一篇文章有多条评论）
type Post struct {
	ID             uint `gorm:"primaryKey"`
	Title          string
	Content        string
	UserID         uint
	CommentStatus  string `gorm:"default:''"` // 无评论 / 有评论（由 Comment 钩子维护）
	Comments       []Comment
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Comment struct {
	ID        uint `gorm:"primaryKey"`
	Content   string
	PostID    uint
	CreatedAt time.Time
	UpdatedAt time.Time
}

// AfterCreate 题目3：新建文章时自动增加该用户的文章统计。
func (p *Post) AfterCreate(tx *gorm.DB) error {
	if err := tx.Model(&User{}).Where("id = ?", p.UserID).
		UpdateColumn("post_count", gorm.Expr("post_count + ?", 1)).Error; err != nil {
		return err
	}
	return tx.Model(&Post{}).Where("id = ?", p.ID).Update("comment_status", "无评论").Error
}

// AfterCreate 新建评论时把文章状态标为「有评论」。
func (c *Comment) AfterCreate(tx *gorm.DB) error {
	return tx.Model(&Post{}).Where("id = ?", c.PostID).Update("comment_status", "有评论").Error
}

// AfterDelete 题目3：删除评论后若该文章已无评论，将文章评论状态设为「无评论」。
func (c *Comment) AfterDelete(tx *gorm.DB) error {
	var count int64
	if err := tx.Model(&Comment{}).Where("post_id = ?", c.PostID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return tx.Model(&Post{}).Where("id = ?", c.PostID).Update("comment_status", "无评论").Error
	}
	return nil
}

// backfillStats 根据实际文章/评论数量同步统计字段（兼容 AutoMigrate 前已有数据）。
func backfillStats(db *gorm.DB) error {
	var users []User
	if err := db.Find(&users).Error; err != nil {
		return err
	}
	for _, u := range users {
		var cnt int64
		if err := db.Model(&Post{}).Where("user_id = ?", u.ID).Count(&cnt).Error; err != nil {
			return err
		}
		if err := db.Model(&User{}).Where("id = ?", u.ID).Update("post_count", cnt).Error; err != nil {
			return err
		}
	}
	var posts []Post
	if err := db.Find(&posts).Error; err != nil {
		return err
	}
	for _, p := range posts {
		var c int64
		if err := db.Model(&Comment{}).Where("post_id = ?", p.ID).Count(&c).Error; err != nil {
			return err
		}
		status := "无评论"
		if c > 0 {
			status = "有评论"
		}
		if err := db.Model(&Post{}).Where("id = ?", p.ID).Update("comment_status", status).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedDemoData(db *gorm.DB) (User, error) {
	var user User
	res := db.Where("email = ?", "alice@example.com").Take(&user)
	if res.Error != nil && res.Error != gorm.ErrRecordNotFound {
		return User{}, res.Error
	}
	if res.RowsAffected == 0 {
		user = User{Name: "Alice", Email: "alice@example.com"}
		if err := db.Create(&user).Error; err != nil {
			return User{}, err
		}
	}

	var postCount int64
	if err := db.Model(&Post{}).Where("user_id = ?", user.ID).Count(&postCount).Error; err != nil {
		return User{}, err
	}
	if postCount > 0 {
		var totalComments int64
		if err := db.Model(&Comment{}).
			Joins("JOIN posts ON posts.id = comments.post_id").
			Where("posts.user_id = ?", user.ID).
			Count(&totalComments).Error; err != nil {
			return User{}, err
		}
		if totalComments == 0 {
			var p Post
			if err := db.Where("user_id = ?", user.ID).Order("id ASC").First(&p).Error; err != nil {
				return User{}, err
			}
			seedComments := []Comment{
				{Content: "demo comment A", PostID: p.ID},
				{Content: "demo comment B", PostID: p.ID},
			}
			if err := db.Create(&seedComments).Error; err != nil {
				return User{}, err
			}
		}
		return user, nil
	}

	posts := []Post{
		{Title: "Gorm Basics", Content: "Intro to models and migrations", UserID: user.ID},
		{Title: "Go Concurrency", Content: "WaitGroup and channel examples", UserID: user.ID},
	}
	if err := db.Create(&posts).Error; err != nil {
		return User{}, err
	}

	comments := []Comment{
		{Content: "Very helpful", PostID: posts[0].ID},
		{Content: "Thanks for sharing", PostID: posts[0].ID},
		{Content: "Clear explanation", PostID: posts[1].ID},
	}
	if err := db.Create(&comments).Error; err != nil {
		return User{}, err
	}

	return user, nil
}

// 查询某个用户发布的所有文章及其评论信息。
func queryPostsWithCommentsByUser(db *gorm.DB, userID uint) ([]Post, error) {
	var posts []Post
	err := db.Where("user_id = ?", userID).
		Preload("Comments").
		Find(&posts).Error
	return posts, err
}

// 查询评论数量最多的文章信息。
func queryMostCommentedPost(db *gorm.DB) (Post, int64, error) {
	type mostCommentedRow struct {
		PostID uint
		Cnt    int64
	}

	var row mostCommentedRow
	err := db.Model(&Comment{}).
		Select("post_id, COUNT(*) AS cnt").
		Group("post_id").
		Order("cnt DESC").
		Limit(1).
		Scan(&row).Error
	if err != nil {
		return Post{}, 0, err
	}
	if row.PostID == 0 {
		return Post{}, 0, nil
	}

	var post Post
	if err := db.Preload("Comments").First(&post, row.PostID).Error; err != nil {
		return Post{}, 0, err
	}
	return post, row.Cnt, nil
}

func main() {
	db, err := gorm.Open(sqlite.Open("blog.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("open db failed: %v", err)
	}

	// 按模型创建/更新数据库表结构
	if err := db.AutoMigrate(&User{}, &Post{}, &Comment{}); err != nil {
		log.Fatalf("auto migrate failed: %v", err)
	}
	if err := backfillStats(db); err != nil {
		log.Fatalf("backfill stats failed: %v", err)
	}

	user, err := seedDemoData(db)
	if err != nil {
		log.Fatalf("seed data failed: %v", err)
	}

	// 题目2-1：查询某个用户发布的所有文章及其评论。
	posts, err := queryPostsWithCommentsByUser(db, user.ID)
	if err != nil {
		log.Fatalf("query user posts failed: %v", err)
	}
	fmt.Printf("User %s has %d posts:\n", user.Name, len(posts))
	for _, p := range posts {
		fmt.Printf("- Post: %s (comments=%d)\n", p.Title, len(p.Comments))
		for _, c := range p.Comments {
			fmt.Printf("  - Comment: %s\n", c.Content)
		}
	}

	// 题目2-2：查询评论数量最多的文章。
	post, commentCount, err := queryMostCommentedPost(db)
	if err != nil {
		log.Fatalf("query most commented post failed: %v", err)
	}
	if post.ID == 0 {
		fmt.Println("No comments found yet.")
		return
	}
	fmt.Printf("Most commented post: %s (comments=%d)\n", post.Title, commentCount)

	// 题目3：钩子函数演示（Post.AfterCreate 增加用户文章数；Comment.AfterDelete 更新文章评论状态）
	fmt.Println("\n--- 题目3：钩子函数 ---")
	if err := db.First(&user, user.ID).Error; err != nil {
		log.Fatalf("reload user failed: %v", err)
	}
	fmt.Printf("User.PostCount（新建文章前）= %d\n", user.PostCount)

	hookPost := Post{
		Title:   "Hook 演示文章",
		Content: "用于验证 AfterCreate 增加 PostCount",
		UserID:  user.ID,
	}
	if err := db.Create(&hookPost).Error; err != nil {
		log.Fatalf("create hook demo post failed: %v", err)
	}
	if err := db.First(&user, user.ID).Error; err != nil {
		log.Fatalf("reload user after hook post failed: %v", err)
	}
	fmt.Printf("User.PostCount（新建文章后）= %d\n", user.PostCount)

	// 找一篇「当前仍有评论」的文章（避免上次运行已删光评论导致无法演示）
	var anchor Comment
	res := db.Table("comments").
		Joins("JOIN posts ON posts.id = comments.post_id").
		Where("posts.user_id = ?", user.ID).
		Order("comments.id ASC").
		Take(&anchor)
	if res.Error != nil {
		fmt.Println("No comments left for hook demo (skip delete demo).")
		return
	}
	var firstPost Post
	if err := db.First(&firstPost, anchor.PostID).Error; err != nil {
		log.Fatalf("load post for hook demo failed: %v", err)
	}
	var comments []Comment
	if err := db.Where("post_id = ?", firstPost.ID).Find(&comments).Error; err != nil {
		log.Fatalf("list comments failed: %v", err)
	}
	if len(comments) == 0 {
		fmt.Println("No comments to delete for hook demo.")
		return
	}
	if err := db.First(&firstPost, firstPost.ID).Error; err != nil {
		log.Fatalf("reload post failed: %v", err)
	}
	fmt.Printf("文章 %q 当前 comment_status=%q，评论数=%d\n", firstPost.Title, firstPost.CommentStatus, len(comments))

	// 删一条评论，若仍有评论则状态仍为「有评论」
	c0 := comments[0]
	if err := db.Delete(&c0).Error; err != nil {
		log.Fatalf("delete comment failed: %v", err)
	}
	if err := db.First(&firstPost, firstPost.ID).Error; err != nil {
		log.Fatalf("reload post after delete failed: %v", err)
	}
	var left int64
	_ = db.Model(&Comment{}).Where("post_id = ?", firstPost.ID).Count(&left)
	fmt.Printf("删 1 条后 comment_status=%q，剩余评论=%d\n", firstPost.CommentStatus, left)

	// 删光剩余评论（逐条 Delete 才会触发 Comment.AfterDelete）
	var restIDs []uint
	if err := db.Model(&Comment{}).Where("post_id = ?", firstPost.ID).Pluck("id", &restIDs).Error; err != nil {
		log.Fatalf("pluck comment ids failed: %v", err)
	}
	for _, id := range restIDs {
		var c Comment
		if err := db.First(&c, id).Error; err != nil {
			log.Fatalf("load comment %d failed: %v", id, err)
		}
		if err := db.Delete(&c).Error; err != nil {
			log.Fatalf("delete comment failed: %v", err)
		}
	}
	if err := db.First(&firstPost, firstPost.ID).Error; err != nil {
		log.Fatalf("reload post after delete all failed: %v", err)
	}
	fmt.Printf("删光后 comment_status=%q\n", firstPost.CommentStatus)
}
