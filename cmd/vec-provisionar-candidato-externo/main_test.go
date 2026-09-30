package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/app/bootstrap"
)

func materialCLIPrueba(t *testing.T) (string, string, config.Config) {
	t.Helper()
	c := func(prefijo string) bootstrap.ComponenteSnapshotContextoExterno {
		return bootstrap.ComponenteSnapshotContextoExterno{
			Referencia: prefijo + "sintetico_1234567890123456", Version: 1, ProcedenciaRef: "prc_sintetica_1234567890123456", ProcedenciaVersion: 1,
			ProcedenciaHuellaSHA256: strings.Repeat("a", 64), ProcedenciaAutoridad: "autoridad_maestra_acreditada", Estado: "activo", VigenteDesde: "2026-01-01T00:00:00.000000Z", VigenteHasta: "2027-01-01T00:00:00.000000Z"}
	}
	f := bootstrap.FuenteProvisionCandidatoExterno{Version: 1, Snapshot: bootstrap.SnapshotContextoExterno{ProvisionRef: "pce_sintetico_1234567890123456", Poblacion: "candidato", Estado: "activo", Cuenta: c("cta_"), Persona: c("per_"), Perfil: c("prf_"), Contexto: c("vca_"), VinculoCandidato: &bootstrap.VinculoSnapshotCandidatoExterno{ComponenteSnapshotContextoExterno: c("vin_"), CandidatoRef: "can_sintetico_1234567890123456"}}}
	b, e := json.Marshal(f)
	if e != nil {
		t.Fatal(e)
	}
	dir := t.TempDir()
	ruta := filepath.Join(dir, "fuente.json")
	dsn := filepath.Join(dir, "dsn")
	if os.WriteFile(ruta, b, 0o600) != nil || os.WriteFile(dsn, []byte("secreto-sintetico-no-es-DSN"), 0o600) != nil {
		t.Fatal("material")
	}
	cfg := config.Config{PortalProceso: config.ValorPortalProcesoInterno, ExecutionProfile: config.ExecutionProfileDevelopment, AuthMode: config.AuthModeDevelopment, DevelopmentGuard: config.DevelopmentGuardAcknowledgement}
	return ruta, dsn, cfg
}

func TestProvisionCandidatoCLIPlanNoEjecutaNiRevelaFuente(t *testing.T) {
	ruta, _, cfg := materialCLIPrueba(t)
	var out, errout bytes.Buffer
	dep := dependencias{reloj: time.Now, preparar: func(_ context.Context, _ config.Config, dsn string, p bootstrap.PlanProvisionCandidatoExterno) (bootstrap.PlanProvisionCandidatoExterno, error) {
		if dsn != "" {
			t.Fatal("plan puro abrió material de conexión")
		}
		return p, nil
	}, ejecutar: func(context.Context, config.Config, string, bootstrap.PlanProvisionCandidatoExterno, string, string) (bootstrap.ResumenProvisionCandidatoExterno, error) {
		t.Fatal("plan publicó")
		return bootstrap.ResumenProvisionCandidatoExterno{}, nil
	}}
	if estado := ejecutar(context.Background(), []string{"--fuente", ruta, "--fase", "autorizacion"}, cfg, &out, &errout, dep); estado != 0 {
		t.Fatal(estado, errout.String())
	}
	if strings.Contains(out.String(), "sintetic") || strings.Contains(out.String(), ruta) || errout.Len() != 0 {
		t.Fatal("identidad o ruta en stdout")
	}
	var r bootstrap.ResumenProvisionCandidatoExterno
	if json.Unmarshal(out.Bytes(), &r) != nil || len(r.HuellaSHA256) != 64 || len(r.PreimagenSHA256) != 64 {
		t.Fatal("plan incompleto")
	}
}

func TestProvisionCandidatoCLIFalloNoReintentaNiImprimeErrorPrivado(t *testing.T) {
	ruta, dsn, cfg := materialCLIPrueba(t)
	var out, errout bytes.Buffer
	llamadas := 0
	f, e := bootstrap.CargarFuenteProvisionCandidatoExterno(ruta)
	if e != nil {
		t.Fatal(e)
	}
	p, e := bootstrap.PrepararProvisionCandidatoExterno(cfg, f, "autorizacion", time.Now())
	if e != nil {
		t.Fatal(e)
	}
	dep := dependencias{reloj: time.Now, preparar: func(_ context.Context, _ config.Config, _ string, p bootstrap.PlanProvisionCandidatoExterno) (bootstrap.PlanProvisionCandidatoExterno, error) {
		return p, nil
	}, ejecutar: func(_ context.Context, _ config.Config, _ string, _ bootstrap.PlanProvisionCandidatoExterno, a, h string) (bootstrap.ResumenProvisionCandidatoExterno, error) {
		llamadas++
		if a != p.Resumen().HuellaSHA256 || h != p.Resumen().PreimagenSHA256 {
			t.Fatal("aprobación cambiada")
		}
		return bootstrap.ResumenProvisionCandidatoExterno{}, errors.New("dato-privado-DSN")
	}}
	args := []string{"--fuente", ruta, "--fase", "autorizacion", "--dsn-archivo", dsn, "--aprobar", p.Resumen().HuellaSHA256, "--preimagen", p.Resumen().PreimagenSHA256}
	if estado := ejecutar(context.Background(), args, cfg, &out, &errout, dep); estado != 1 || llamadas != 1 || out.Len() != 0 || strings.Contains(errout.String(), "dato-privado") {
		t.Fatal("reintento o fuga de error")
	}
	if estado := ejecutar(context.Background(), []string{"--fuente", ruta, "--fase", "autorizacion", "--aprobar", "incompleta"}, cfg, io.Discard, io.Discard, dep); estado != 2 || llamadas != 1 {
		t.Fatal("aprobación incompleta alcanzó dependencia")
	}
}
