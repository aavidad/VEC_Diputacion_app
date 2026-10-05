package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	config "vec-diputacion-granada/config/auditoriacheckpoint"
	"vec-diputacion-granada/internal/app/bootstrap"
	"vec-diputacion-granada/internal/vec/domain"
)

func TestCLIVerificaPaqueteExportacionDesarrollo(t *testing.T) {
	for _, caso := range []struct {
		nombre    string
		archivo   string
		historico bool
	}{
		{"consumos_fuentes", "../vec-auditoria-verificar/testdata/union_consumos_fuentes_ad173_ad174.json", true},
		{"gobierno_usuarios", "testdata/exportacion_ad188.json", false},
		{"frontera_admin", "testdata/exportacion_ad189.json", false},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			verificarPaqueteExportacionPrueba(t, caso.archivo, caso.historico)
		})
	}
}

func verificarPaqueteExportacionPrueba(t *testing.T, archivo string, historico bool) {
	t.Helper()
	cb, err := os.ReadFile(archivo)
	if err != nil {
		t.Fatal(err)
	}
	var cabecera struct {
		Esquema    string                     `json:"esquema"`
		Manifiesto domain.CoberturaCheckpoint `json:"manifiesto"`
	}
	if err = json.Unmarshal(cb, &cabecera); err != nil {
		t.Fatal(err)
	}
	cfgBytes, err := os.ReadFile("testdata/config.sintetica.json")
	if err != nil {
		t.Fatal(err)
	}
	var cfg config.AuditoriaCheckpointOffline
	if err = json.Unmarshal(cfgBytes, &cfg); err != nil {
		t.Fatal(err)
	}
	p, err := bootstrap.NuevoProveedorExportacionAuditoriaDesarrollo([32]byte{1}, [32]byte{2}, cfg.Politica)
	if err != nil {
		t.Fatal(err)
	}
	defer p.CerrarExportacionAuditoria()
	m := domain.ManifiestoExportacionAuditoriaDesarrollo{
		Esquema: domain.EsquemaExportacionAuditoriaDesarrollo, Politica: cfg.Politica,
		Captura:   domain.CapturaExportacionAuditoria{Referencia: "captura:sintetica", AuditoriaRef: "aud_v3_sintetica", AuditoriaSHA256: strings.Repeat("a", 64), CapturadaEn: "2026-10-04T01:00:00.000000Z"},
		Documento: domain.DocumentoExportacionAuditoria{Esquema: cabecera.Esquema, Bytes: int64(len(cb)), SHA256: domain.HuellaCheckpoint(cb)}, Cobertura: cabecera.Manifiesto, HistoricosSinFechaLigada: historico,
	}
	tsa, err := p.SellarExportacionAuditoria(context.Background(), m)
	if err != nil {
		t.Fatal(err)
	}
	r, err := p.FirmarExportacionAuditoria(context.Background(), domain.ReciboExportacionAuditoriaDesarrollo{Manifiesto: m, TSA: tsa, PinSPKISHA256: p.PinExportacionAuditoria()})
	if err != nil {
		t.Fatal(err)
	}
	der, err := p.PublicaExportacionAuditoriaDER()
	if err != nil {
		t.Fatal(err)
	}
	d := t.TempDir()
	recibo, cadena, publica, conf := filepath.Join(d, "recibo.json"), filepath.Join(d, "cadena.json"), filepath.Join(d, "publica.der"), filepath.Join(d, "config.json")
	rb, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	for path, b := range map[string][]byte{recibo: rb, cadena: cb, publica: der, conf: cfgBytes} {
		if err = os.WriteFile(path, b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	args := []string{"-operacion", "verificar-exportacion", "-config", conf, "-entrada", recibo, "-cadena", cadena, "-spki", publica, "-pin-spki-sha256", p.PinExportacionAuditoria()}
	var out, log bytes.Buffer
	if run(args, &out, &log) != 0 {
		t.Fatalf("verificacion: %s", out.String())
	}
	avisoHistorico := `"historicos_sin_fecha_ligada":false`
	if historico {
		avisoHistorico = `"historicos_sin_fecha_ligada":true`
	}
	for _, want := range []string{`"estado":"verificada"`, `"origen_extraccion":"no_acreditado"`, avisoHistorico, `"firma_legal":false`, `"tiempo_independiente":false`} {
		if !strings.Contains(out.String(), want) {
			t.Fatal(out.String())
		}
	}
	if strings.Contains(log.String(), recibo) || strings.Contains(log.String(), m.Captura.Referencia) {
		t.Fatal("registro técnico expuso la captura")
	}
	pruebaFallo := func(nombre string, nuevos []string) {
		t.Helper()
		out.Reset()
		log.Reset()
		if run(nuevos, &out, &log) == 0 {
			t.Fatalf("%s admitido", nombre)
		}
	}
	pruebaFallo("secreto en verificación", append(append([]string(nil), args...), "-kms-master", "/no/necesario"))
	pruebaFallo("ancla ajena", append(append([]string(nil), args...), "-ancla", "/no/necesario"))
	sinCadena := append(append([]string(nil), args[:6]...), args[8:]...)
	pruebaFallo("sin cadena", sinCadena)
	otraRaiz := append([]string(nil), args...)
	otraRaiz[len(otraRaiz)-1] = strings.Repeat("0", 64)
	pruebaFallo("pin ajeno", otraRaiz)
	if err = os.WriteFile(cadena, append(append([]byte(nil), cb...), '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	pruebaFallo("byte añadido", args)
	if err = os.WriteFile(cadena, cb, 0600); err != nil {
		t.Fatal(err)
	}
	r.Manifiesto.Captura.AuditoriaRef = "aud_v3_otra"
	adulterado, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(recibo, adulterado, 0600); err != nil {
		t.Fatal(err)
	}
	pruebaFallo("acuse cambiado", args)
	if err = os.WriteFile(recibo, rb, 0600); err != nil {
		t.Fatal(err)
	}
	cfg.MaxBytes = int64(len(cb) + len(rb) - 1)
	limitada, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(conf, limitada, 0600); err != nil {
		t.Fatal(err)
	}
	pruebaFallo("límite acumulado", args)
}

func TestReciboExportacionExigeCamposYRechazaNulos(t *testing.T) {
	for _, b := range []string{`{}`, `{"manifiesto":null,"tsa":{},"pin_spki_sha256":"a","firma_base64":"b"}`, `{"manifiesto":{"esquema":"x"},"tsa":{},"pin_spki_sha256":"a","firma_base64":"b"}`} {
		if camposExportacionCompletos([]byte(b)) == nil {
			t.Fatal("recibo incompleto admitido")
		}
	}
}
