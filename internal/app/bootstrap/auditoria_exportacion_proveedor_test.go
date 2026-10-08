package bootstrap

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/domain"
)

func TestExportacionAuditoriaDominioPropioYRaizExterna(t *testing.T) {
	b, err := os.ReadFile("../../../cmd/vec-auditoria-checkpoint/testdata/config.sintetica.json")
	if err != nil {
		t.Fatal(err)
	}
	var c struct {
		Politica domain.PoliticaCheckpoint `json:"politica"`
	}
	if err = json.Unmarshal(b, &c); err != nil {
		t.Fatal(err)
	}
	p, err := NuevoProveedorExportacionAuditoriaDesarrollo([32]byte{1}, [32]byte{2}, c.Politica)
	if err != nil {
		t.Fatal(err)
	}
	defer p.CerrarExportacionAuditoria()
	anterior, err := NuevoProveedorCheckpointDesarrollo([32]byte{1}, [32]byte{2}, c.Politica)
	if err != nil {
		t.Fatal(err)
	}
	defer anterior.CerrarCheckpoint()
	if p.PinExportacionAuditoria() == anterior.PinCheckpoint() {
		t.Fatal("dominios comparten raíz")
	}
	ceros := strings.Repeat("0", 64)
	m := domain.ManifiestoExportacionAuditoriaDesarrollo{
		Esquema: domain.EsquemaExportacionAuditoriaDesarrollo, Politica: c.Politica,
		Captura:   domain.CapturaExportacionAuditoria{Referencia: "captura:sintetica", AuditoriaRef: "aud_v3_sintetica", AuditoriaSHA256: ceros, CapturadaEn: "2026-10-04T01:00:00.000000Z"},
		Cobertura: domain.CoberturaCheckpoint{CadenaID: "cadena:sintetica", AnteriorSHA256: ceros, CabezaSHA256: ceros},
		Documento: domain.DocumentoExportacionAuditoria{Esquema: "vec.auditoria.verificacion.v1", Bytes: 1, SHA256: ceros},
	}
	ctx := context.Background()
	sello, err := p.SellarExportacionAuditoria(ctx, m)
	if err != nil {
		t.Fatal(err)
	}
	r, err := p.FirmarExportacionAuditoria(ctx, domain.ReciboExportacionAuditoriaDesarrollo{Manifiesto: m, TSA: sello, PinSPKISHA256: p.PinExportacionAuditoria()})
	if err != nil {
		t.Fatal(err)
	}
	der, err := p.PublicaExportacionAuditoriaDER()
	if err != nil {
		t.Fatal(err)
	}
	v, err := NuevoVerificadorExportacionAuditoriaDesarrollo(der, p.PinExportacionAuditoria(), c.Politica)
	if err != nil {
		t.Fatal(err)
	}
	if err = v.VerificarExportacionAuditoria(ctx, r); err != nil {
		t.Fatal(err)
	}
	cambios := []func(*domain.ReciboExportacionAuditoriaDesarrollo){
		func(x *domain.ReciboExportacionAuditoriaDesarrollo) {
			x.Manifiesto.Captura.AuditoriaSHA256 = strings.Repeat("1", 64)
		},
		func(x *domain.ReciboExportacionAuditoriaDesarrollo) {
			x.Manifiesto.Captura.CapturadaEn = "2026-10-04T01:00:01.000000Z"
		},
		func(x *domain.ReciboExportacionAuditoriaDesarrollo) { x.Manifiesto.Politica.ProveedorKMSVersion++ },
		func(x *domain.ReciboExportacionAuditoriaDesarrollo) { x.Manifiesto.Documento.Bytes++ },
		func(x *domain.ReciboExportacionAuditoriaDesarrollo) { x.Manifiesto.HistoricosSinFechaLigada = true },
		func(x *domain.ReciboExportacionAuditoriaDesarrollo) {
			x.TSA.HuellaPreimagenSHA256 = strings.Repeat("1", 64)
		},
	}
	for _, cambiar := range cambios {
		copia := r
		cambiar(&copia)
		if v.VerificarExportacionAuditoria(ctx, copia) == nil {
			t.Fatal("sustitución admitida")
		}
	}
	if _, err = NuevoVerificadorExportacionAuditoriaDesarrollo(der, anterior.PinCheckpoint(), c.Politica); err == nil {
		t.Fatal("pin de checkpoint admitido")
	}
	cancelado, cancelar := context.WithCancel(ctx)
	cancelar()
	if v.VerificarExportacionAuditoria(cancelado, r) == nil {
		t.Fatal("cancelación ignorada")
	}
	p.CerrarExportacionAuditoria()
	if _, err = p.FirmarExportacionAuditoria(ctx, domain.ReciboExportacionAuditoriaDesarrollo{Manifiesto: m, TSA: sello, PinSPKISHA256: p.PinExportacionAuditoria()}); err == nil {
		t.Fatal("proveedor cerrado firmó")
	}
}
