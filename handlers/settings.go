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

// CreateSetting creates a new setting in the database and Redis cache
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

// UpdateSetting updates an existing setting in the database and Redis cache
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
