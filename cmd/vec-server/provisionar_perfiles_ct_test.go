package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/app/bootstrap"
)

func TestProvisionPerfilesCTPreparaSinArrancarServidor(t *testing.T) {
	var salida, errores bytes.Buffer
	llamadas := 0
	ejecutar := func(_ context.Context, _ config.Config, s bootstrap.SolicitudProvisionPerfilesCT) (bootstrap.ResultadoProvisionPerfilesCT, error) {
		llamadas++
		if !s.Preparar || s.Aplicar || s.ManifiestoRuta != "" || s.ManifiestoSHA256 != "" || s.AprobacionRef != "" {
			t.Fatalf("solicitud de propuesta incorrecta: %+v", s)
		}
		return bootstrap.ResultadoProvisionPerfilesCT{Estado: "propuesta", Manifiesto: json.RawMessage(`{"version":1}`)}, nil
	}
	codigo := ejecutarProvisionPerfilesCT(context.Background(), []string{"--preparar"}, &salida, &errores, config.Config{}, ejecutar)
	if codigo != 0 || llamadas != 1 || errores.Len() != 0 || !strings.Contains(salida.String(), `"version":1`) {
		t.Fatalf("resultado de propuesta: codigo=%d llamadas=%d salida=%q errores=%q", codigo, llamadas, salida.String(), errores.String())
	}
}

func TestProvisionPerfilesCTAplicaSoloConHuellaYReferenciaExactas(t *testing.T) {
	contenido := []byte(`{"version":1,"aprobacion_ref":"demo:ct-1"}`)
	ruta := filepath.Join(t.TempDir(), "manifiesto.json")
	if err := os.WriteFile(ruta, contenido, 0o600); err != nil {
		t.Fatal(err)
	}
	huella := sha256.Sum256(contenido)
	huellaTexto := hex.EncodeToString(huella[:])
	args := []string{"--manifiesto", ruta, "--sha256", huellaTexto, "--aprobacion-ref", "demo:ct-1", "--aplicar"}
	llamadas := 0
	ejecutar := func(_ context.Context, _ config.Config, s bootstrap.SolicitudProvisionPerfilesCT) (bootstrap.ResultadoProvisionPerfilesCT, error) {
		llamadas++
		if !s.Aplicar || s.Preparar || s.ManifiestoRuta != ruta || s.ManifiestoSHA256 != huellaTexto || s.AprobacionRef != "demo:ct-1" {
			t.Fatalf("solicitud de aplicacion incorrecta: %+v", s)
		}
		return bootstrap.ResultadoProvisionPerfilesCT{Estado: "aplicado", ManifiestoSHA256: huellaTexto, AprobacionRef: s.AprobacionRef}, nil
	}
	var salida, errores bytes.Buffer
	if codigo := ejecutarProvisionPerfilesCT(context.Background(), args, &salida, &errores, config.Config{}, ejecutar); codigo != 0 || llamadas != 1 || errores.Len() != 0 {
		t.Fatalf("aplicacion valida: codigo=%d llamadas=%d errores=%q", codigo, llamadas, errores.String())
	}
	if !strings.Contains(salida.String(), huellaTexto) || !strings.Contains(salida.String(), "demo:ct-1") {
		t.Fatalf("recibo incompleto: %q", salida.String())
	}

	casos := [][]string{
		{"--aplicar", "--manifiesto", ruta, "--sha256", strings.Repeat("0", 64), "--aprobacion-ref", "demo:ct-1"},
		{"--aplicar", "--manifiesto", ruta, "--sha256", strings.ToUpper(huellaTexto), "--aprobacion-ref", "demo:ct-1"},
		{"--aplicar", "--manifiesto", ruta, "--sha256", huellaTexto},
		{"--aplicar", "--manifiesto", ruta, "--sha256", huellaTexto, "--aprobacion-ref", "demo:ct-1", "--preparar"},
		{"--preparar", "--manifiesto", ruta},
		{"--database-url=postgres://usuario:secreto@host/base", "--preparar"},
	}
	for _, caso := range casos {
		salida.Reset()
		errores.Reset()
		if codigo := ejecutarProvisionPerfilesCT(context.Background(), caso, &salida, &errores, config.Config{}, ejecutar); codigo != 2 || salida.Len() != 0 || strings.Contains(errores.String(), "secreto") {
			t.Errorf("entrada rechazada: codigo=%d salida=%q errores=%q", codigo, salida.String(), errores.String())
		}
	}
	if llamadas != 1 {
		t.Fatalf("el ejecutor recibio %d llamadas", llamadas)
	}
}

func TestProvisionPerfilesCTRechazaManifiestoInseguroYNoFiltraErrorProveedor(t *testing.T) {
	contenido := []byte(`{"version":1}`)
	dir := t.TempDir()
	ruta := filepath.Join(dir, "manifiesto.json")
	if err := os.WriteFile(ruta, contenido, 0o600); err != nil {
		t.Fatal(err)
	}
	huella := sha256.Sum256(contenido)
	args := func(r string) []string {
		return []string{"--aplicar", "--manifiesto", r, "--sha256", hex.EncodeToString(huella[:]), "--aprobacion-ref", "demo:ct-1"}
	}
	llamadas := 0
	ejecutar := func(context.Context, config.Config, bootstrap.SolicitudProvisionPerfilesCT) (bootstrap.ResultadoProvisionPerfilesCT, error) {
		llamadas++
		return bootstrap.ResultadoProvisionPerfilesCT{}, errors.New("postgres://usuario:secreto@host/base")
	}
	var salida, errores bytes.Buffer
	if codigo := ejecutarProvisionPerfilesCT(context.Background(), args(ruta), &salida, &errores, config.Config{}, ejecutar); codigo != 1 || llamadas != 1 || strings.Contains(errores.String(), "secreto") || salida.Len() != 0 {
		t.Fatalf("error proveedor: codigo=%d llamadas=%d salida=%q errores=%q", codigo, llamadas, salida.String(), errores.String())
	}
	if err := os.Chmod(ruta, 0o620); err != nil {
		t.Fatal(err)
	}
	salida.Reset()
	errores.Reset()
	if codigo := ejecutarProvisionPerfilesCT(context.Background(), args(ruta), &salida, &errores, config.Config{}, ejecutar); codigo != 2 || llamadas != 1 {
		t.Fatalf("fichero escribible por grupo aceptado: codigo=%d llamadas=%d", codigo, llamadas)
	}
	if err := os.Chmod(ruta, 0o600); err != nil {
		t.Fatal(err)
	}
	enlace := filepath.Join(dir, "enlace.json")
	if err := os.Symlink(ruta, enlace); err != nil {
		t.Fatal(err)
	}
	if codigo := ejecutarProvisionPerfilesCT(context.Background(), args(enlace), &salida, &errores, config.Config{}, ejecutar); codigo != 2 || llamadas != 1 {
		t.Fatalf("enlace aceptado: codigo=%d llamadas=%d", codigo, llamadas)
	}
	if err := os.WriteFile(ruta, bytes.Repeat([]byte("a"), maximoManifiestoPerfilesCT+1), 0o600); err != nil {
		t.Fatal(err)
	}
	if codigo := ejecutarProvisionPerfilesCT(context.Background(), args(ruta), &salida, &errores, config.Config{}, ejecutar); codigo != 2 || llamadas != 1 {
		t.Fatalf("manifiesto excesivo aceptado: codigo=%d llamadas=%d", codigo, llamadas)
	}
}

func TestProvisionPerfilesCTInformaIncidenciaTrasPublicacionSinFiltrarSecreto(t *testing.T) {
	contenido := []byte(`{"version":1,"aprobacion_ref":"demo:ct-1"}`)
	ruta := filepath.Join(t.TempDir(), "manifiesto.json")
	if err := os.WriteFile(ruta, contenido, 0o600); err != nil {
		t.Fatal(err)
	}
	huella := sha256.Sum256(contenido)
	huellaTexto := hex.EncodeToString(huella[:])
	args := []string{"--aplicar", "--manifiesto", ruta, "--sha256", huellaTexto, "--aprobacion-ref", "demo:ct-1"}
	ejecutar := func(context.Context, config.Config, bootstrap.SolicitudProvisionPerfilesCT) (bootstrap.ResultadoProvisionPerfilesCT, error) {
		return bootstrap.ResultadoProvisionPerfilesCT{
			Estado:     "incidencia",
			Manifiesto: json.RawMessage(`{"dato_no_autorizado":"no_emitir"}`),
			Perfiles: []bootstrap.ReciboPerfilProvisionCT{
				{Clave: "alta", PerfilRef: "perfil:alta", AsignacionRef: "asignacion:alta", Version: 1, HuellaSHA256: huellaTexto},
				{Clave: "cobertura", PerfilRef: "postgres://usuario:secreto@host/base", AsignacionRef: "asignacion:cobertura", Version: 1, HuellaSHA256: huellaTexto},
			},
		}, errors.Join(bootstrap.ErrProvisionPerfilesCTIncidenciaContexto, errors.New("postgres://usuario:secreto@host/base"))
	}
	var salida, errores bytes.Buffer
	if codigo := ejecutarProvisionPerfilesCT(context.Background(), args, &salida, &errores, config.Config{}, ejecutar); codigo != 1 || salida.Len() != 0 {
		t.Fatalf("incidencia: codigo=%d salida=%q errores=%q", codigo, salida.String(), errores.String())
	}
	var cuerpo struct {
		Error            string                              `json:"error"`
		Estado           string                              `json:"estado"`
		ManifiestoSHA256 string                              `json:"manifiesto_sha256"`
		AprobacionRef    string                              `json:"aprobacion_ref"`
		Perfiles         []bootstrap.ReciboPerfilProvisionCT `json:"perfiles"`
	}
	if err := json.Unmarshal(errores.Bytes(), &cuerpo); err != nil {
		t.Fatal(err)
	}
	if cuerpo.Error != "incidencia_contexto_tras_publicacion" || cuerpo.Estado != "incidencia" ||
		cuerpo.ManifiestoSHA256 != huellaTexto || cuerpo.AprobacionRef != "demo:ct-1" ||
		len(cuerpo.Perfiles) != 1 || cuerpo.Perfiles[0].Clave != "alta" {
		t.Fatalf("evidencia de incidencia incompleta: %+v", cuerpo)
	}
	if strings.Contains(errores.String(), "secreto") || strings.Contains(errores.String(), "no_emitir") ||
		strings.Contains(errores.String(), ruta) {
		t.Fatalf("incidencia filtra material privado: %q", errores.String())
	}
}
