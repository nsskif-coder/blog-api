package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"blog-api/internal/model"
)

type Storage struct {
	mu       sync.RWMutex
	dataDir  string
	users    []model.User
	posts    []model.Post
	comments []model.Comment
	
	nextUserID    int
	nextPostID    int
	nextCommentID int
}

func NewStorage(dataDir string) (*Storage, error) {
	s := &Storage{dataDir: dataDir}
	if err := s.loadFromFiles(); err != nil {
		return nil, fmt.Errorf("failed to load data: %w", err)
	}
	return s, nil
}

func (s *Storage) loadFromFiles() error {
	s.users = make([]model.User, 0)
	s.posts = make([]model.Post, 0)
	s.comments = make([]model.Comment, 0)

	s.loadJSON("users.json", &s.users)
	s.loadJSON("posts.json", &s.posts)
	s.loadJSON("comments.json", &s.comments)

	s.nextUserID = len(s.users) + 1
	s.nextPostID = len(s.posts) + 1
	s.nextCommentID = len(s.comments) + 1
	return nil
}

func (s *Storage) loadJSON(filename string, v interface{}) error {
	filePath := s.dataDir + "/" + filename
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err 
	}
	return json.Unmarshal(data, v)
}

func (s *Storage) saveToFiles() error {
	s.saveJSON("users.json", s.users)
	s.saveJSON("posts.json", s.posts)
	s.saveJSON("comments.json", s.comments)
	return nil
}

func (s *Storage) saveJSON(filename string, v interface{}) error {
	filePath := s.dataDir + "/" + filename
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0644)
}

func (s *Storage) CreateUser(username, email, password string) (*model.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, u := range s.users {
		if u.Email == email {
			return nil, fmt.Errorf("user with this email already exists")
		}
		if u.Username == username {
			return nil, fmt.Errorf("user with this username already exists")
		}
	}

	user := model.User{
		ID:        s.nextUserID,
		Username:  username,
		Email:     email,
		Password:  password, 
		CreatedAt: time.Now(),
	}

	s.users = append(s.users, user)
	s.nextUserID++
	s.saveToFiles()
	return &user, nil
}

func (s *Storage) GetUserByEmail(email string) (*model.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, u := range s.users {
		if u.Email == email {
			return &u, nil
		}
	}
	return nil, fmt.Errorf("user not found")
}

func (s *Storage) CreatePost(authorID int, title, content string) (*model.Post, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	post := model.Post{
		ID:        s.nextPostID,
		AuthorID:  authorID,
		Title:     title,
		Content:   content,
		CreatedAt: time.Now(),
	}

	s.posts = append(s.posts, post)
	s.nextPostID++
	s.saveToFiles()
	return &post, nil
}

func (s *Storage) GetAllPosts() []model.Post {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]model.Post, len(s.posts))
	copy(result, s.posts)
	return result
}

func (s *Storage) GetPostByID(id int) (*model.Post, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, p := range s.posts {
		if p.ID == id {
			return &p, nil
		}
	}
	return nil, fmt.Errorf("post not found")
}

func (s *Storage) CreateComment(postID, authorID int, text string) (*model.Comment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Проверка есть ли пост
	postExists := false
	for _, p := range s.posts {
		if p.ID == postID {
			postExists = true
			break
		}
	}
	if !postExists {
		return nil, fmt.Errorf("post not found")
	}

	comment := model.Comment{
		ID:        s.nextCommentID,
		PostID:    postID,
		AuthorID:  authorID,
		Text:      text,
		CreatedAt: time.Now(),
	}

	s.comments = append(s.comments, comment)
	s.nextCommentID++
	s.saveToFiles()
	return &comment, nil
}

func (s *Storage) GetCommentsByPostID(postID int) []model.Comment {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []model.Comment
	for _, c := range s.comments {
		if c.PostID == postID {
			result = append(result, c)
		}
	}
	return result
}