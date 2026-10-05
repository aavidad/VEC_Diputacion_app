package adminperfiles

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestVinculoSesionADMINContextosNoMezclanSesionMismaCuenta(t *testing.T) {
	now := time.Date(2026, 10, 4, 15, 0, 0, 0, time.UTC)
	v := VinculoSesionADMIN{Referencia: "vis_" + strings.Repeat("a", 32), Version: 1, HuellaSHA256: strings.Repeat("a", 64), AutenticacionRef: "aut_" + strings.Repeat("a", 22), SesionRef: "ses_" + strings.Repeat("a", 22), PersonaRef: "per_" + strings.Repeat("a", 22), CuentaRef: "cta_" + strings.Repeat("a", 22), CuentaOrdinariaRef: "cta_" + strings.Repeat("b", 22), PerfilActivoRef: "prf_" + strings.Repeat("a", 22), CertificadoSHA256: strings.Repeat("a", 64), CASHA256: strings.Repeat("b", 64), VinculoCertificadoRef: "vca_" + strings.Repeat("a", 22), VinculoCertificadoVersion: 1, PoliticaRef: "pga_" + strings.Repeat("a", 22), PoliticaSHA256: strings.Repeat("a", 64), SeleccionRevision: 1, ControlSesionRef: "cse_" + strings.Repeat("a", 22), ControlSesionRevision: 1, ControlSesionSHA256: strings.Repeat("a", 64), VinculadaEn: now, VigenteHasta: now.Add(time.Minute), FuenteRef: "fuente:ensayo:ids", FuenteSHA256: strings.Repeat("a", 64)}
	a, e := ContextoConVinculoSesionADMIN(context.Background(), v)
	if e != nil {
		t.Fatal(e)
	}
	w := v
	w.Referencia = "vis_" + strings.Repeat("b", 32)
	w.SesionRef = "ses_" + strings.Repeat("b", 22)
	b, e := ContextoConVinculoSesionADMIN(context.Background(), w)
	if e != nil {
		t.Fatal(e)
	}
	got, e := VinculoSesionADMINDeContexto(a)
	if e != nil || got != v {
		t.Fatal("vinculo_mezclado")
	}
	got, e = VinculoSesionADMINDeContexto(b)
	if e != nil || got != w {
		t.Fatal("vinculo_mezclado")
	}
	if _, e = ContextoConVinculoSesionADMIN(a, w); e == nil {
		t.Fatal("vinculo_sustituido")
	}
	if _, e = VinculoSesionADMINDeContexto(context.Background()); e == nil {
		t.Fatal("vinculo_ausente_aceptado")
	}
	raw, e := json.Marshal(v)
	if e != nil || strings.Contains(string(raw), v.CuentaRef) {
		t.Fatal("datos_exportados")
	}
}
