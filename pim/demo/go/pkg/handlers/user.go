package handlers

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"demo/pkg/db"
	"demo/pkg/logging" // Import the new logging package
	"demo/pkg/metering"

	"github.com/go-chi/chi/v5/middleware" // Added for middleware.GetReqID
	"github.com/yandex-cloud/go-genproto/yandex/cloud/marketplace/pim/v1/saas"
	ycsdk "github.com/yandex-cloud/go-sdk"
	"google.golang.org/grpc/status"
)

// Embed the entire directory.
//
//go:embed templates
var templates embed.FS

// Server holds the dependencies for the HTTP server.
type Server struct {
	repo           db.Repository // Changed to use Repository interface
	sdk            *ycsdk.SDK
	logger         *slog.Logger
	meteringClient *metering.Client
}

// NewServer creates a new Server instance.
func NewServer(repo db.Repository, sdk *ycsdk.SDK, logger *slog.Logger, meteringClient *metering.Client) *Server { // Changed to use Repository interface
	return &Server{
		repo:           repo,
		sdk:            sdk,
		logger:         logger,
		meteringClient: meteringClient,
	}
}

// logError logs an error with context (request ID if available).
func (s Server) logError(r *http.Request, msg string, err error, details ...any) {
	args := []any{slog.String("msg", msg)}
	if err != nil {
		args = append(args, slog.String("error", err.Error()))
	}
	// Add request ID from context if available, using chi's GetReqID
	if reqID := middleware.GetReqID(r.Context()); reqID != "" {
		args = append(args, slog.String(string(logging.RequestIDKey), reqID))
	}
	args = append(args, details...)
	// Use GetLoggerFromContext from the logging package, though s.logger should already have request ID if middleware is used
	logging.GetLoggerFromContext(r.Context()).ErrorContext(r.Context(), "request_error", args...)
}

// handleError is a helper to send standard HTTP error responses and log the error.
func (s Server) handleError(w http.ResponseWriter, r *http.Request, httpStatus int, userMessage string, err error, details ...any) {
	s.logError(r, userMessage, err, details...)
	http.Error(w, userMessage, httpStatus)
}

func (s Server) RegisterPostHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.handleError(w, r, http.StatusMethodNotAllowed, "Invalid request method", nil)
		return
	}

	login := r.FormValue("login")
	password := r.FormValue("password")

	if login == "" || password == "" {
		s.handleError(w, r, http.StatusBadRequest, "Login and password are required", nil)
		return
	}

	user, err := s.repo.CreateUser(r.Context(), login, password)
	if err != nil {
		s.handleError(w, r, http.StatusInternalServerError, "Could not create user", err, slog.String("login", login))
		return
	}
	session, err := s.repo.CreateSession(r.Context(), *user)
	if err != nil {
		s.handleError(w, r, http.StatusInternalServerError, "Could not create session after registration", err, slog.String("login", login))
		return
	}

	token := r.URL.Query().Get("token")
	location := "/"
	if token != "" {
		location = "/?token=" + token
	}
	w.Header().Set("Location", location)
	// Save session to cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    session.ID,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/", // Added Path attribute
	})

	w.WriteHeader(http.StatusSeeOther)
	s.logger.InfoContext(r.Context(), "User registered successfully", slog.String("login", login))
}

func (s Server) RegisterGetHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFS(templates, "templates/register.tpl.html")
	if err != nil {
		s.handleError(w, r, http.StatusInternalServerError, "Could not parse register template", err)
		return
	}
	user, token := s.reqVariables(r)

	err = t.Execute(w, map[string]any{
		"user":  user,
		"token": token,
	})
	if err != nil {
		s.handleError(w, r, http.StatusInternalServerError, "Could not render register template", err)
		return
	}
}

func (s Server) LoginPostHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.handleError(w, r, http.StatusMethodNotAllowed, "Invalid request method", nil)
		return
	}
	t, err := template.ParseFS(templates, "templates/login.tpl.html")
	if err != nil {
		s.handleError(w, r, http.StatusInternalServerError, "Could not parse login template", err)
		return
	}

	login := r.FormValue("login")
	password := r.FormValue("password")

	_, token := s.reqVariables(r)

	user, err := s.repo.GetUser(r.Context(), login)
	if err != nil {
		s.logError(r, "Failed to get user during login", err, slog.String("login", login))
		_ = t.Execute(w, map[string]any{
			"error": "Invalid login or password",
			"token": token,
		})
		return
	}

	if !user.CheckPassword(password) {
		s.logger.WarnContext(r.Context(), "Invalid password attempt", slog.String("login", login))
		_ = t.Execute(w, map[string]any{
			"error": "Invalid login or password",
			"token": token,
		})
		return
	}

	session, err := s.repo.CreateSession(r.Context(), *user)
	if err != nil {
		s.handleError(w, r, http.StatusInternalServerError, "Could not create session", err, slog.String("login", user.Login))
		return
	}

	location := "/"
	if token != "" {
		location = "/?token=" + token
	}
	w.Header().Set("Location", location)
	// Save session to cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    session.ID,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/", // Added Path attribute
	})

	w.WriteHeader(http.StatusSeeOther)
	s.logger.InfoContext(r.Context(), "User logged in successfully", slog.String("login", user.Login))
}

func (s Server) LoginGetHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFS(templates, "templates/login.tpl.html")
	if err != nil {
		s.handleError(w, r, http.StatusInternalServerError, "Could not parse login template", err)
		return
	}

	user, token := s.reqVariables(r)

	err = t.Execute(w, map[string]any{
		"user":  user,
		"token": token,
	})
	if err != nil {
		s.handleError(w, r, http.StatusInternalServerError, "Could not render login template", err)
		return
	}
}

func (s Server) LogoutGetHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		s.handleError(w, r, http.StatusBadRequest, "No session cookie found", err) // Changed to Bad Request
		return
	}

	err = s.repo.DeleteSession(r.Context(), cookie.Value)
	if err != nil {
		s.handleError(w, r, http.StatusInternalServerError, "Could not delete session", err, slog.String("session_id", cookie.Value))
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(-time.Hour),
		Path:     "/", // Added Path attribute
	})

	w.Header().Set("Location", "/")
	w.WriteHeader(http.StatusSeeOther)
	s.logger.InfoContext(r.Context(), "User logged out successfully", slog.String("session_id", cookie.Value))
}

func (s Server) IndexGetHandler(w http.ResponseWriter, r *http.Request) {
	// It's generally better to parse templates once at startup, but for simplicity here, we parse them per request.
	indexTpl, err := template.ParseFS(templates, "templates/index.tpl.html")
	if err != nil {
		s.handleError(w, r, http.StatusInternalServerError, "Could not parse index template", err)
		return
	}
	appTpl, err := template.ParseFS(templates, "templates/app.tpl.html") // Changed to ParseFS
	if err != nil {
		s.handleError(w, r, http.StatusInternalServerError, "Could not parse app template", err)
		return
	}

	user, token := s.reqVariables(r)

	if user != nil && user.ProductInstanceId != "" {
		err = appTpl.Execute(w, map[string]any{
			"user": user,
		})
	} else {
		err = indexTpl.Execute(w, map[string]any{
			"user":  user,
			"token": token,
		})
	}

	if err != nil {
		s.handleError(w, r, http.StatusInternalServerError, "Could not render template", err)
		return
	}
}

func (s Server) BindPostHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second) // Increased timeout
	defer cancel()

	user, token := s.reqVariables(r)
	if user == nil {
		s.handleError(w, r, http.StatusUnauthorized, "User not logged in", nil) // Changed to Unauthorized
		return
	}
	if token == "" {
		s.handleError(w, r, http.StatusBadRequest, "Token missing", nil)
		return
	}

	op, err := s.sdk.WrapOperation(s.sdk.Marketplace().PIM().ProductInstance().Claim(ctx, &saas.ClaimProductInstanceRequest{
		Token: token,
	}))
	if err != nil {
		// Check for gRPC error codes if possible
		st, _ := status.FromError(err)
		s.handleError(w, r, http.StatusInternalServerError, "Failed to initiate product instance claim", err, slog.String("user_login", user.Login), slog.String("grpc_status_code", st.Code().String()))
		return
	}

	// Wait for the operation to complete
	if err = op.Wait(ctx); err != nil {
		st, _ := status.FromError(err)
		s.handleError(w, r, http.StatusInternalServerError, "Waiting for product instance claim operation failed", err, slog.String("user_login", user.Login), slog.String("operation_id", op.Id()), slog.String("grpc_status_code", st.Code().String()))
		return
	}

	resp, err := op.Response()
	if err != nil {
		s.handleError(w, r, http.StatusInternalServerError, "Could not get response from product instance claim operation", err, slog.String("user_login", user.Login), slog.String("operation_id", op.Id()))
		return
	}

	pim, ok := resp.(*saas.ProductInstance)
	if !ok {
		s.handleError(w, r, http.StatusInternalServerError, "Invalid response type from product instance claim", nil, slog.String("user_login", user.Login), slog.String("response_type", fmt.Sprintf("%T", resp)))
		return
	}

	err = s.repo.UpdateUser(r.Context(), user.Login, pim.Id)
	if err != nil {
		s.handleError(w, r, http.StatusInternalServerError, "Failed to update user with product instance ID", err, slog.String("user_login", user.Login), slog.String("product_instance_id", pim.Id))
		return
	}

	w.Header().Set("Location", "/")
	w.WriteHeader(http.StatusSeeOther)
	s.logger.InfoContext(r.Context(), "Product instance bound successfully", slog.String("user_login", user.Login), slog.String("product_instance_id", pim.Id))
}

func (s Server) ReportPostHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second) // Added timeout
	defer cancel()

	user, _ := s.reqVariables(r)

	if user == nil {
		s.handleError(w, r, http.StatusUnauthorized, "User not logged in", nil)
		return
	}
	if user.ProductInstanceId == "" {
		s.handleError(w, r, http.StatusBadRequest, "Product instance not bound for user", nil, slog.String("user_login", user.Login))
		return
	}
	if r.Method != http.MethodPost {
		s.handleError(w, r, http.StatusMethodNotAllowed, "Invalid request method", nil)
		return
	}

	amountValue := r.FormValue("amount")
	if amountValue == "" {
		s.handleError(w, r, http.StatusBadRequest, "Amount is required", nil)
		return
	}
	amount, err := strconv.Atoi(amountValue)
	if err != nil || amount <= 0 {
		s.handleError(w, r, http.StatusBadRequest, "Invalid amount", err, slog.String("amount_value", amountValue))
		return
	}

	usageParams := metering.ReportUsageParameters{
		ProductInstanceID: user.ProductInstanceId,
		Amount:            amount,
		UserID:            user.Login, // Pass user login for logging context
	}

	usageRecordID, err := s.meteringClient.ReportUsage(ctx, usageParams)
	if err != nil {
		s.handleError(w, r, http.StatusInternalServerError, "Failed to report usage", err,
			slog.String("user_login", user.Login),
			slog.String("product_instance_id", user.ProductInstanceId),
			slog.Int("amount", amount),
		)
		return
	}

	w.Header().Set("Location", "/")
	w.WriteHeader(http.StatusSeeOther)
	s.logger.InfoContext(r.Context(), "Usage reported successfully",
		slog.String("user_login", user.Login),
		slog.String("product_instance_id", user.ProductInstanceId),
		slog.Int("amount", amount),
		slog.String("usage_record_id", usageRecordID),
	)
}

// reqVariables extracts user from session and token from query parameters.
// It logs if a session cookie is present but fetching the session fails.
func (s Server) reqVariables(r *http.Request) (*db.User, string) {
	var user *db.User
	cookie, err := r.Cookie("session_id")
	if err == nil && cookie.Value != "" {
		session, sessionErr := s.repo.GetSessionWithUser(r.Context(), cookie.Value)
		if sessionErr != nil {
			// Log the error but don't fail the request, as user might not be logged in.
			loggerFromCtx := logging.GetLoggerFromContext(r.Context())
			loggerFromCtx.Error("Failed to get session with user from cookie", slog.String("error", sessionErr.Error()), slog.String("session_id_from_cookie", cookie.Value))
		} else if session != nil {
			user = &session.User
		}
	}

	// get token from search query
	token := r.URL.Query().Get("token")
	return user, token
}
