package main

import (
	"context"
	"embed"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	"demo/pkg/db"

	"github.com/google/uuid"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/marketplace/metering/v1"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/marketplace/pim/v1/saas"
	ycsdk "github.com/yandex-cloud/go-sdk"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Embed the entire directory.
//
//go:embed templates
var templates embed.FS

type Server struct {
	repo *db.Repo
	sdk  *ycsdk.SDK
}

func NewServer(repo *db.Repo, sdk *ycsdk.SDK) *Server {
	return &Server{
		repo: repo,
		sdk:  sdk,
	}
}

func (s Server) registerPostHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	login := r.FormValue("login")
	password := r.FormValue("password")

	user, err := s.repo.CreateUser(r.Context(), login, password)
	if err != nil {
		log.Printf("Could not create user: %s\n", err)
		http.Error(w, "Could not create user", http.StatusInternalServerError)
		return
	}
	session, err := s.repo.CreateSession(r.Context(), *user)
	if err != nil {
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
	})

	w.WriteHeader(http.StatusSeeOther)
}

func (s Server) registerHandler(w http.ResponseWriter, r *http.Request) {
	t, _ := template.ParseFS(templates, "templates/register.tpl.html")
	user, token := s.reqVariables(r)

	err := t.Execute(w, map[string]any{
		"user":  user,
		"token": token,
	})
	if err != nil {
		http.Error(w, "Could not render template", http.StatusInternalServerError)
		return
	}
}

func (s Server) loginPostHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	t, _ := template.ParseFS(templates, "templates/login.tpl.html")

	login := r.FormValue("login")
	password := r.FormValue("password")

	_, token := s.reqVariables(r)

	user, err := s.repo.GetUser(r.Context(), login)
	if err != nil {
		log.Printf("Could not get user: %s\n", err)
		_ = t.Execute(w, map[string]any{
			"error": "Invalid login or password",
			"token": token,
		})
		return
	}

	if !user.CheckPassword(password) {
		_ = t.Execute(w, map[string]any{
			"error": "Invalid login or password",
			"token": token,
		})
		return
	}

	session, err := s.repo.CreateSession(r.Context(), *user)
	if err != nil {
		http.Error(w, "Could not create session", http.StatusInternalServerError)
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
	})

	w.WriteHeader(http.StatusSeeOther)
}
func (s Server) loginHandler(w http.ResponseWriter, r *http.Request) {
	t, _ := template.ParseFS(templates, "templates/login.tpl.html")

	user, token := s.reqVariables(r)

	err := t.Execute(w, map[string]any{
		"user":  user,
		"token": token,
	})
	if err != nil {
		http.Error(w, "Could not render template", http.StatusInternalServerError)
		return
	}
}

func (s Server) logoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		http.Error(w, "Could not get session", http.StatusInternalServerError)
		return
	}

	err = s.repo.DeleteSession(r.Context(), cookie.Value)
	if err != nil {
		http.Error(w, "Could not delete session", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(-time.Hour),
	})

	w.Header().Set("Location", "/")
	w.WriteHeader(http.StatusSeeOther)
}

func (s Server) indexHandler(w http.ResponseWriter, r *http.Request) {
	index, err := template.ParseFS(templates, "templates/index.tpl.html")
	app, err := template.ParseFiles("templates/app.tpl.html")
	if err != nil {
		http.Error(w, "Could not load template", http.StatusInternalServerError)
		return
	}

	user, token := s.reqVariables(r)

	if user != nil && user.ProductInstanceId != "" {
		err = app.Execute(w, map[string]any{
			"user": user,
		})
	} else {
		err = index.Execute(w, map[string]any{
			"user":  user,
			"token": token,
		})
	}

	if err != nil {
		http.Error(w, "Could not render template", http.StatusInternalServerError)
		return
	}
}

func (s Server) bindPostHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	user, token := s.reqVariables(r)
	if user == nil || token == "" {
		http.Error(w, "User not logged in or token missing", http.StatusBadRequest)
		return
	}
	op, err := s.sdk.WrapOperation(s.sdk.Marketplace().PIM().ProductInstance().Claim(r.Context(), &saas.ClaimProductInstanceRequest{
		Token: token,
	}))
	// Wait for the operation to complete
	err = op.Wait(ctx)
	if err != nil {
		return
	}
	resp, err := op.Response()
	if err != nil {
		log.Printf("Could not claim product instance: %s\n", err)
		http.Error(w, "Invalid token", http.StatusBadRequest)
		return
	}
	pim, ok := resp.(*saas.ProductInstance)
	if !ok {
		log.Printf("Invalid response type: %T\n", resp)
		http.Error(w, "Invalid token", http.StatusBadRequest)
		return
	}
	err = s.repo.UpdateUser(r.Context(), user.Login, pim.Id)
	w.Header().Set("Location", "/")
	w.WriteHeader(http.StatusSeeOther)
}

func (s Server) reportPostHandler(w http.ResponseWriter, r *http.Request) {
	// Get the user and token from the request
	user, _ := s.reqVariables(r)

	// Read body and parse it as form
	if user == nil || user.ProductInstanceId == "" {
		http.Error(w, "User not logged in or product instance not bound", http.StatusBadRequest)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	// Get amount from body
	amountValue := r.FormValue("amount")
	if amountValue == "" {
		http.Error(w, "Amount is required", http.StatusBadRequest)
		return
	}
	amount, err := strconv.Atoi(amountValue)
	if err != nil || amount <= 0 {
		http.Error(w, "Invalid amount", http.StatusBadRequest)
		return
	}

	inst, err := s.sdk.Marketplace().PIM().ProductInstance().Get(r.Context(), &saas.GetProductInstanceRequest{
		ProductInstanceId: user.ProductInstanceId,
	})
	if err != nil {
		log.Fatal("Instance get error: " + err.Error())
	}
	if inst == nil {
		http.Error(w, "Product instance not found", http.StatusNotFound)
		return
	}

	recordId, err := uuid.NewUUID()
	if err != nil {
		log.Printf("Could not generate UUID for usage record: %s\n", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	// This is a hardcoded SKU ID for the demo. In a real application, you would retrieve this from the product instance or configuration.
	const skuId = "dn2e395635hoblkrj1at"
	resp, err := s.sdk.Marketplace().Metering().ProductUsage().Write(r.Context(), &metering.WriteUsageRequest{
		DryRun:            false,
		ProductInstanceId: user.ProductInstanceId,
		UsageRecords: []*metering.UsageRecord{
			{
				Uuid:     recordId.String(),
				SkuId:    skuId,
				Quantity: int64(amount),
				Timestamp: &timestamppb.Timestamp{
					Seconds: time.Now().Unix(),
					Nanos:   0,
				},
			},
		},
	})
	if err != nil {
		return
	}
	if len(resp.Rejected) > 0 {
		log.Printf("Usage record rejected: %v\n", resp.Rejected)
		http.Error(w, "Usage record rejected", http.StatusBadRequest)
		return
	}

	w.Header().Set("Location", "/")
	w.WriteHeader(http.StatusSeeOther)
}

func (s Server) reqVariables(r *http.Request) (*db.User, string) {
	var user *db.User
	cookie, err := r.Cookie("session_id")
	if err == nil && cookie.Value != "" {
		session, err := s.repo.GetSessionWithUser(cookie.Value)
		if err == nil {
			user = &session.User
		}
	}

	// get token from search query
	token := r.URL.Query().Get("token")
	return user, token
}
