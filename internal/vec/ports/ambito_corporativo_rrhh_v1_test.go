package ports

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestComprobanteAmbitoCorporativoRRHHV1ConservaVersionesYNoSeSerializa(t *testing.T) {
	d := DatosComprobanteAmbitoCorporativoRRHHV1{
		CuentaRef: "cta_" + strings.Repeat("a", 22), CuentaVersion: 3,
		PersonaRef: "per_" + strings.Repeat("b", 22), PersonaVersion: 4,
		PerfilRef: "prf_" + strings.Repeat("c", 22), PerfilVersion: 5,
		ContextoRef: "vca_" + strings.Repeat("d", 22), ContextoVersion: 6,
		VinculoCorporativoRef: "vcr_" + strings.Repeat("e", 22), VinculoCorporativoVersion: 7,
		OrganizacionRef: "org_" + strings.Repeat("f", 16), OrganizacionVersion: 8,
		OrganizacionProcedenciaRef: "prc_" + strings.Repeat("g", 22), OrganizacionProcedenciaVersion: 9,
		OrganizacionProcedenciaHuellaSHA256: strings.Repeat("a", 64), OrganizacionProcedenciaAutoridad: "autoridad_maestra_acreditada",
		VinculoProcedenciaRef: "prc_" + strings.Repeat("h", 22), VinculoProcedenciaVersion: 10,
		VinculoProcedenciaHuellaSHA256: strings.Repeat("b", 64), VinculoProcedenciaAutoridad: "autoridad_maestra_acreditada",
		Superficie: "interna_corporativa", Uso: "consulta_rrhh",
		VigenteDesde: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		VigenteHasta: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
	}
	c, err := NuevoComprobanteAmbitoCorporativoRRHHV1(d)
	if err != nil {
		t.Fatal(err)
	}
	d.OrganizacionVersion = 99
	observado, err := c.Datos()
	if err != nil || observado.OrganizacionVersion != 8 {
		t.Fatalf("version alterada: %d, %v", observado.OrganizacionVersion, err)
	}
	if _, err := json.Marshal(c); !errors.Is(err, ErrComprobanteAmbitoCorporativoRRHHV1Invalido) {
		t.Fatalf("comprobante expuesto: %v", err)
	}
	b, err := c.JSONParaSQL()
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil || len(m) != 24 || m["organizacion_version"] != float64(8) {
		t.Fatalf("material SQL incompleto: %v, %v", m, err)
	}
	d.Superficie = "exterior"
	if _, err := NuevoComprobanteAmbitoCorporativoRRHHV1(d); !errors.Is(err, ErrComprobanteAmbitoCorporativoRRHHV1Invalido) {
		t.Fatalf("superficie exterior aceptada: %v", err)
	}
}
