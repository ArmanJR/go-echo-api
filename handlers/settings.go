package handlers

import (
	"context"
	"database/sql"
	"github.com/dgrijalva/jwt-go"
	"github.com/go-redis/redis/v8"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/labstack/echo"
	"github.com/pkg/errors"

	"github.com/armanjr/go-echo-api/models"
)

type SettingsHandler struct {
	pg     *sql.DB
	redis  *redis.Client
	secret string
}

func NewSettingsHandler(pg *sql.DB, redis *redis.Client, secret string) *SettingsHandler {
	return &SettingsHandler{
		pg:     pg,
		redis:  redis,
		secret: secret,
	}
}

// ServeSwagger serves the OpenAPI documentation.
// @Summary Serve OpenAPI documentation
// @Description Serves the OpenAPI documentation in YAML format.
// @Tags documentation
// @Produce yaml
// @Success 200 {file} swagger.yaml
// @Router /swagger.yaml [get]
func (h *SettingsHandler) ServeSwagger(c echo.Context) error {
	return c.File("docs/swagger.yaml")
}

// Hello returns a welcome message.
// @Summary Say hello
// @Description Returns a welcome message.
// @Tags greetings
// @Produce json
// @Success 200 {object} string "Hello!"
// @Router / [get]
func (h *SettingsHandler) Hello(c echo.Context) error {
	return c.JSON(http.StatusOK, echo.Map{
		"message": "Hello!",
	})
}

// SignIn godoc
// @Summary Sign in to the application
// @Description Signs in a user and returns a JWT token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param username body string true "Username"
// @Param password body string true "Password"
// @Success 200 {object} map[string]interface{} "Returns a JWT token"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Incorrect username or password"
// @Failure 500 {object} map[string]interface{} "Failed to sign token"
// @Router /signin [post]
func (h *SettingsHandler) SignIn(c echo.Context) error {
	type request struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	var req request
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	if req.Username != "admin" || req.Password != "SabziPolo" {
		return echo.NewHTTPError(http.StatusUnauthorized, "incorrect username or password")
	}
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["username"] = req.Username
	claims["exp"] = time.Now().Add(time.Hour * 72).Unix()
	signedToken, err := token.SignedString([]byte(h.secret))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to sign token")
	}
	return c.JSON(http.StatusOK, echo.Map{
		"token": signedToken,
	})
}

// GetSetting godoc
// @Summary Get a setting
// @Description Retrieves the value of a setting given a key
// @Tags settings
// @Param key path integer true "Setting key"
// @Accept  json
// @Produce  json
// @Success 200 {object} models.Setting
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /settings/{key} [get]
func (h *SettingsHandler) GetSetting(c echo.Context) error {
	key, err := strconv.Atoi(c.Param("key"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid key")
	}
	type request struct {
		Key   string `json:"key"`
		Value string `json:"value"`
		TTL   int    `json:"ttl"`
	}
	var req request
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	ctx := context.Background()

	// Check if setting exists
	var setting models.Setting
	err = h.pg.QueryRowContext(ctx, "SELECT id, key, value, created_at, updated_at FROM settings WHERE key = $1", key).Scan(&setting.ID, &setting.Key, &setting.Value, &setting.CreatedAt, &setting.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "setting not found")
		}
		return errors.Wrap(err, "failed to get setting")
	}

	// Get TTL from redis
	ttl, err := h.redis.TTL(context.Background(), setting.Key).Result()
	if err == nil {
		setting.TTL = int(ttl.Seconds())
	}

	return c.JSON(http.StatusOK, setting)
}

// GetSettings godoc
// @Summary Get all settings
// @Description Get all settings sorted by id in descending order
// @Tags settings
// @Accept json
// @Produce json
// @Success 200 {array} models.Setting
// @Failure 500 {object} ErrorResponse
// @Router /settings [get]
func (h *SettingsHandler) GetSettings(c echo.Context) error {
	ctx := context.Background()

	var settings []models.Setting

	rows, err := h.pg.QueryContext(ctx, "SELECT id, key, value, created_at, updated_at FROM settings ORDER BY id DESC")
	if err != nil {
		return errors.Wrap(err, "failed to get settings")
	}

	defer rows.Close()

	for rows.Next() {
		var setting models.Setting
		rows.Scan(&setting.ID, &setting.Key, &setting.Value, &setting.CreatedAt, &setting.UpdatedAt)
		ttl, _ := h.redis.TTL(context.Background(), setting.Key).Result()
		setting.TTL = int(ttl.Seconds())
		settings = append(settings, setting)
	}

	if err := rows.Err(); err != nil {
		return errors.Wrap(err, "failed to process settings")
	}

	return c.JSON(http.StatusOK, settings)
}

// CreateSetting godoc
// @Summary Create a new setting
// @Description Create a new setting with a key, value, and TTL
// @Tags settings
// @Accept json
// @Produce json
// @Param setting body models.Setting true "New setting"
// @Success 201 {object} models.Setting
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /settings [post]
func (h *SettingsHandler) CreateSetting(c echo.Context) error {
	// Parse request body
	req := &models.Setting{}
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "failed to parse request body"})
	}

	// Validate request body
	if req.Key == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "key is required"})
	}
	if req.Value == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "value is required"})
	}
	if req.TTL <= 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "ttl must be greater than 0"})
	}

	// Insert setting into database
	if _, err := h.pg.ExecContext(context.Background(), "INSERT INTO settings (key, value) VALUES ($1, $2)", req.Key, req.Value); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to insert setting into database"})
	}

	// Store setting in Redis cache with TTL
	if err := h.redis.Set(context.Background(), req.Key, req.Value, time.Duration(req.TTL)*time.Second).Err(); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to store setting in Redis"})
	}

	return c.JSON(http.StatusCreated, req)
}

// UpdateSetting godoc
// @Summary Update a setting
// @Description Update a setting's value or TTL by its key
// @Tags settings
// @Accept json
// @Produce json
// @Param key path string true "Key of the setting to update"
// @Param body body models.Setting true "Request body with fields to update (value or TTL)"
// @Success 200 {object} models.Setting
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /settings/{key} [put]
func (h *SettingsHandler) UpdateSetting(c echo.Context) error {
	key, err := url.QueryUnescape(c.Param("key"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid key")
	}
	// Parse request body
	req := &models.Setting{}
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "failed to parse request body"})
	}

	// Validate request body
	if req.Value == "" && req.TTL == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "at least one field (value, or ttl) is required"})
	}

	// Update setting in database
	if req.Value != "" {
		if _, err := h.pg.ExecContext(context.Background(), "UPDATE settings SET value = $1 WHERE key = $2", req.Value, key); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update setting in database"})
		}
	}

	var setting models.Setting
	err = h.pg.QueryRowContext(context.Background(), "SELECT id, key, value, created_at, updated_at FROM settings WHERE key = $1", key).Scan(&setting.ID, &setting.Key, &setting.Value, &setting.CreatedAt, &setting.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "setting not found")
		}
		return errors.Wrap(err, "failed to get setting")
	}

	// Update setting in Redis
	if req.TTL > 0 {
		if err := h.redis.Set(context.Background(), req.Key, setting.Value, time.Duration(req.TTL)*time.Second).Err(); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update setting in Redis cache"})
		}
	}

	return c.JSON(http.StatusOK, req)
}
