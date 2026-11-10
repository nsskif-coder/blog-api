package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"blog-api/internal/auth"
	"blog-api/internal/logger"
	"blog-api/internal/middleware"
	"blog-api/internal/model"
	"blog-api/internal/storage"
)

type Handler struct {
	storage *storage.Storage
	logger  *logger.Logger
}

// создает новый обработчик
func NewHandler(s *storage.Storage, l *logger.Logger) *Handler {
	return &Handler{storage: s, logger: l}
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, model.HealthResponse{Status: "ok"})
}

// регистрация нового пользователя
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req model.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Username == "" || req.Email == "" || req.Password == "" {
		respondError(w, http.StatusBadRequest, "username, email and password are required")
		return
	}

	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	user, err := h.storage.CreateUser(req.Username, req.Email, hashedPassword)
	if err != nil {
		respondError(w, http.StatusConflict, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, user)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req model.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.storage.GetUserByEmail(req.Email)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	if !auth.CheckPasswordHash(req.Password, user.Password) {
		respondError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, err := auth.GenerateToken(user.ID, user.Username)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	response := model.LoginResponse{
		Token: token,
		User: model.User{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			CreatedAt: user.CreatedAt,
		},
	}

	respondJSON(w, http.StatusOK, response)
}

// создание поста (требуется авторизация)
func (h *Handler) CreatePost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID, _, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req model.CreatePostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Валидация
	if req.Title == "" || req.Content == "" {
		respondError(w, http.StatusBadRequest, "title and content are required")
		return
	}

	post, err := h.storage.CreatePost(userID, req.Title, req.Content)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create post")
		return
	}

	h.logger.Log("user " + strconv.Itoa(userID) + " created post " + strconv.Itoa(post.ID))

	respondJSON(w, http.StatusCreated, post)
}

//получение всех постов
func (h *Handler) GetPosts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	posts := h.storage.GetAllPosts()
	respondJSON(w, http.StatusOK, posts)
}

// получение поста по ID
func (h *Handler) GetPostByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		respondError(w, http.StatusBadRequest, "post ID is required")
		return
	}

	id, err := strconv.Atoi(parts[2])
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid post ID")
		return
	}

	post, err := h.storage.GetPostByID(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "post not found")
		return
	}

	respondJSON(w, http.StatusOK, post)
}

// создание комментария (требуется авторизация)
func (h *Handler) CreateComment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID, _, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		respondError(w, http.StatusBadRequest, "post ID is required")
		return
	}

	postID, err := strconv.Atoi(parts[2])
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid post ID")
		return
	}

	var req model.CreateCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Text == "" {
		respondError(w, http.StatusBadRequest, "text is required")
		return
	}

	comment, err := h.storage.CreateComment(postID, userID, req.Text)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.logger.Log("user " + strconv.Itoa(userID) + " created comment on post " + strconv.Itoa(postID))

	respondJSON(w, http.StatusCreated, comment)
}

func (h *Handler) GetComments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		respondError(w, http.StatusBadRequest, "post ID is required")
		return
	}

	postID, err := strconv.Atoi(parts[2])
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid post ID")
		return
	}

	comments := h.storage.GetCommentsByPostID(postID)
	respondJSON(w, http.StatusOK, comments)
}