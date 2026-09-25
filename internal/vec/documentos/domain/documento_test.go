package domain

import (
	"strings"
	"testing"
	"time"
)

func TestDocumentoSoloAceptaReferenciasOpacasYFirmaPendiente(t *testing.T) {
	ref := func(c string) string { return "ref:" + strings.Repeat(c, 64) }
	d := Documento{
		ID: ref("1"), NumeroVEC: "VEC-2026-1", ModuloID: "dietas",
		ExpedienteRef: ref("2"), TipoRef: ref("3"), Version: 1,
		MIME: "application/pdf", HuellaSHA256: strings.Repeat("4", 64), Tamano: 3,
		ObjetoRef: "objeto:1", ObjetoVersion: "version:1", PoliticaRef: ref("5"),
		VersionPolitica: 1, HuellaPoliticaSHA256: strings.Repeat("6", 64),
		ConservacionHasta: time.Now().UTC().Add(time.Hour), Proteccion: "conservacion",
		EstadoFirma: EstadoFirmaPendienteProveedor, CreadoEn: time.Now().UTC(),
	}
	if err := d.Validar(); err != nil {
		t.Fatalf("documento valido: %v", err)
	}
	d.ExpedienteRef = "DNI:12345678Z"
	if d.Validar() == nil {
		t.Fatal("admitio identificador personal en expediente")
	}
	d.ExpedienteRef = ref("2")
	d.EstadoFirma = "firmado_verificado"
	if d.Validar() == nil {
		t.Fatal("admitio firma sin transicion acreditada")
	}
}
