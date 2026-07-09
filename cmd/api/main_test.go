package main

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Drivers SQL falsos para exercitar o health check sem PostgreSQL:
// o database/sql só toca o driver quando Ping/Query são chamados,
// então cada driver simula um estado do banco.

type conexaoFake struct{}

func (conexaoFake) Prepare(string) (driver.Stmt, error) { return nil, errors.New("não implementado") }
func (conexaoFake) Close() error                        { return nil }
func (conexaoFake) Begin() (driver.Tx, error)           { return nil, errors.New("não implementado") }

type driverSaudavel struct{}

func (driverSaudavel) Open(string) (driver.Conn, error) { return conexaoFake{}, nil }

type driverIndisponivel struct{}

func (driverIndisponivel) Open(string) (driver.Conn, error) {
	return nil, errors.New("banco indisponível")
}

type driverLento struct{}

func (driverLento) Open(string) (driver.Conn, error) {
	time.Sleep(5 * time.Second)
	return conexaoFake{}, nil
}

func init() {
	sql.Register("fake-saudavel", driverSaudavel{})
	sql.Register("fake-indisponivel", driverIndisponivel{})
	sql.Register("fake-lento", driverLento{})
}

func abrirDB(t *testing.T, driverName string) *sql.DB {
	t.Helper()
	db, err := sql.Open(driverName, "")
	if err != nil {
		t.Fatalf("falha ao abrir driver fake: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestPingDB_Sucesso(t *testing.T) {
	if err := pingDB(abrirDB(t, "fake-saudavel")); err != nil {
		t.Errorf("esperava ping bem-sucedido, obteve: %v", err)
	}
}

func TestPingDB_BancoIndisponivel(t *testing.T) {
	if err := pingDB(abrirDB(t, "fake-indisponivel")); err == nil {
		t.Error("esperava erro com banco indisponível")
	}
}

func TestPingDB_TimeoutNaoBloqueiaHealthCheck(t *testing.T) {
	err := pingDB(abrirDB(t, "fake-lento"))
	if !errors.Is(err, http.ErrHandlerTimeout) {
		t.Errorf("esperava ErrHandlerTimeout após 2s, obteve: %v", err)
	}
}

func TestHealthReadyHandler_UP(t *testing.T) {
	handler := healthReadyHandler(abrirDB(t, "fake-saudavel"))

	rec := httptest.NewRecorder()
	handler(rec, httptest.NewRequest(http.MethodGet, "/health/ready", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("esperava 200, obteve %d", rec.Code)
	}
	var resp healthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("resposta não é JSON válido: %v", err)
	}
	if resp.Status != "UP" || resp.Components["db"].Status != "UP" {
		t.Errorf("esperava status UP geral e do db, obteve %+v", resp)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("esperava Content-Type application/json, obteve %q", ct)
	}
}

func TestHealthReadyHandler_DOWN(t *testing.T) {
	handler := healthReadyHandler(abrirDB(t, "fake-indisponivel"))

	rec := httptest.NewRecorder()
	handler(rec, httptest.NewRequest(http.MethodGet, "/health/ready", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("esperava 503 com banco fora, obteve %d", rec.Code)
	}
	var resp healthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("resposta não é JSON válido: %v", err)
	}
	if resp.Status != "DOWN" || resp.Components["db"].Status != "DOWN" {
		t.Errorf("esperava status DOWN geral e do db, obteve %+v", resp)
	}
}
