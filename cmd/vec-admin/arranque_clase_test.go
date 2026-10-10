package main

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/app/administracion"
)

// El registro de un fallo de composición lleva etapa y clase cerradas, nunca
// la causa ni los valores de configuración que la provocaron.
func TestRegistrarClaseFalloSinCausa(t *testing.T) {
	var salida bytes.Buffer
	anterior := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&salida, nil)))
	defer slog.SetDefault(anterior)

	ruta := "/ruta/privada/no-debe-salir.crt"
	_, err := administracion.NuevoServidor(administracion.Configuracion{Entorno: "cidonia", CAAdministracion: ruta,
		RetiradaEn: time.Now().Add(time.Hour).UTC()})
	registrarClaseFallo("servidor", err)
	linea := salida.String()
	if !strings.Contains(linea, "etapa=servidor") || !strings.Contains(linea, "clase=rutas_tls") || strings.Contains(linea, ruta) {
		t.Fatalf("registro sin clase o con datos privados: %q", linea)
	}

	salida.Reset()
	registrarClaseFallo("servidor", fmt.Errorf("%w: %s", administracion.ErrConfiguracion, ruta))
	registrarClaseFallo("servidor", errors.New(ruta))
	if salida.Len() != 0 {
		t.Fatalf("registró una clase inexistente: %q", salida.String())
	}
}
