package capturacopias

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func TestAyudanteProceso(t *testing.T) {
	modo := os.Getenv("CS04_HELPER")
	if modo == "" {
		return
	}
	fmt.Fprint(os.Stderr, "privado-no-imprimir")
	if modo == "espera" {
		time.Sleep(time.Minute)
		os.Exit(0)
	}
	if os.Getenv("SYNTHETIC_SECRET") != "" {
		fmt.Fprint(os.Stdout, "hereda-secreto")
		os.Exit(1)
	}
	fmt.Fprint(os.Stdout, "salida-controlada")
	os.Exit(0)
}

func TestProcesoNoHeredaSecretosNiExponeStderr(t *testing.T) {
	t.Setenv("SYNTHETIC_SECRET", "privado")
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	e := Ejecutor{Entorno: []string{"CS04_HELPER=salida", "GORACE=atexit_sleep_ms=0"}, Limite: time.Second}
	var salida bytes.Buffer
	if err := e.Ejecutar(context.Background(), Comando{exe, []string{"-test.run=^TestAyudanteProceso$"}}, &salida); err != nil {
		t.Fatal(err)
	}
	if salida.String() != "salida-controlada" || strings.Contains(salida.String(), "privado") {
		t.Fatal(salida.String())
	}
}

func TestProcesoCanceladoTerminaSinDiagnosticoPrivado(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	e := Ejecutor{Entorno: []string{"CS04_HELPER=espera"}, Limite: 50 * time.Millisecond}
	var salida bytes.Buffer
	inicio := time.Now()
	err = e.Ejecutar(context.Background(), Comando{exe, []string{"-test.run=^TestAyudanteProceso$"}}, &salida)
	if err != ErrProceso || time.Since(inicio) > 3*time.Second || salida.Len() != 0 {
		t.Fatalf("err=%v output=%q", err, salida.String())
	}
}
