package apiserver

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/USER/go-and-compose/storage"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"

	"net/http"
	"time"
)

var defaultStopTimeOut = time.Second * 30

type APIServer struct {
	addr    string
	storage *storage.Storage
}
type Endpoint struct {
	handler EndpointFunc
}
type EndpointFunc func(w http.ResponseWriter, req *http.Request) error

func (e Endpoint) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if err := e.handler(w, req); err != nil {
		logrus.WithError(err).Error("could not process request")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
	}
}

func NewAPIServer(addr string, storage *storage.Storage) (*APIServer, error) {
	if addr == "" {
		return nil, errors.New("addr cannot be blank")
	}

	return &APIServer{
		addr:    addr,
		storage: storage,
	}, nil
}

// Start starts a server with a stop channel
func (s *APIServer) Start(stop <-chan struct{}) error {
	srv := &http.Server{
		Addr:    s.addr,
		Handler: s.router(),
	}

	go func() {
		logrus.WithField("addr", srv.Addr).Info("starting server")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logrus.Fatalf("listen: %s\n", err)
		}
	}()
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), defaultStopTimeOut)
	defer cancel()

	logrus.WithField("timeout", defaultStopTimeOut).Info("stopping server")
	return srv.Shutdown(ctx)
}

func (s *APIServer) router() http.Handler {
	router := mux.NewRouter()

	router.HandleFunc("/", s.defaultRoute)
	router.Methods("POST").Path("/tokens").Handler(Endpoint{s.issueTokens})
	router.Methods("POST").Path("/tokens/refresh").Handler(Endpoint{s.refreshToken})

	return router
}

func (s *APIServer) defaultRoute(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("Hello World"))
}

func (s *APIServer) issueTokens(w http.ResponseWriter, req *http.Request) error {
	var input struct {
		UserID string `json:"user_id"`
	}

	if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
		logrus.WithError(err).Error("Error decoding request body")
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return nil
	}

	userID := input.UserID
	ip := req.RemoteAddr

	if userID == "" {
		http.Error(w, "missing user_id", http.StatusBadRequest)
		return nil
	}

	accessToken, err := generateAccessToken(userID, ip)
	if err != nil {
		return err
	}

	refreshTokenRaw := fmt.Sprintf("%s:%s", userID, time.Now().String())
	refreshToken := base64.StdEncoding.EncodeToString([]byte(refreshTokenRaw))

	hashedRefreshTokenBytes, err := bcrypt.GenerateFromPassword([]byte(refreshToken), bcrypt.DefaultCost)
	if err != nil {
		logrus.WithError(err).Error("Error generating hashed refresh token")
		return err
	}

	hashedRefreshToken := string(hashedRefreshTokenBytes) // Преобразуем в строку

	if err := s.storage.StoreRefreshToken(req.Context(), userID, hashedRefreshToken); err != nil {
		return err
	}

	response := map[string]string{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	}

	return json.NewEncoder(w).Encode(response)
}

func (s *APIServer) refreshToken(w http.ResponseWriter, req *http.Request) error {
	var input struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}

	if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
		return err
	}

	claims, err := parseAccessToken(input.AccessToken)
	if err != nil {
		http.Error(w, "invalid access token", http.StatusUnauthorized)
		return nil
	}

	hashedToken, err := s.storage.GetRefreshToken(req.Context(), claims.UserID)
	if err != nil {
		http.Error(w, "invalid refresh token", http.StatusUnauthorized)
		return nil
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hashedToken), []byte(input.RefreshToken)); err != nil {
		http.Error(w, "invalid refresh token", http.StatusUnauthorized)
		return nil
	}

	newAccessToken, err := generateAccessToken(claims.UserID, req.RemoteAddr)
	if err != nil {
		return err
	}

	if claims.IP != req.RemoteAddr {
		// Send email to user about the IP change (mock implementation)
		fmt.Printf("Warning: IP changed for user %s. Sending email...\n", claims.UserID)
	}

	response := map[string]string{
		"access_token": newAccessToken,
	}

	return json.NewEncoder(w).Encode(response)
}
