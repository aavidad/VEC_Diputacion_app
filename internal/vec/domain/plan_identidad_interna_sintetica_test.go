package domain

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func identidadSinteticaFixture() PlanIdentidadInternaSinteticaV1 {
	t := time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC)
	hasta := t.Add(24 * time.Hour)
	e := EvidenciaFuentesInicialesAdmin{Referencia: "prc_" + strings.Repeat("a", 22), Version: 1, HuellaSHA256: strings.Repeat("a", 64)}
	return PlanIdentidadInternaSinteticaV1{Version: 1, OperacionRef: "piis_" + strings.Repeat("a", 22), PreparadoEn: t, CaducaEn: hasta, Entorno: "desarrollo", AlcanceFuente: "sintetico_declarado", Procedencia: e, Organizacion: OrganizacionIdentidadInternaSintetica{OrganizacionRef: "org_" + strings.Repeat("a", 16), VersionEsperada: 2, ProcedenciaHuellaSHA256: strings.Repeat("a", 64), VigenteHasta: hasta}, Persona: PersonaIdentidadInternaSintetica{PersonaRef: "per_" + strings.Repeat("a", 22), VigenteHasta: hasta, OperacionCuentaOrdinariaRef: "opr_" + strings.Repeat("a", 22), FuenteTitularidad: e}, FuenteHMAC: e}
}
func TestIdentidadSinteticaPrecondiciones(t *testing.T) {
	base := identidadSinteticaFixture()
	for nombre, cambiar := range map[string]func(*PlanIdentidadInternaSinteticaV1){"organizacion_nueva": func(p *PlanIdentidadInternaSinteticaV1) { p.Organizacion.VersionEsperada = 0 }, "persona_existente": func(p *PlanIdentidadInternaSinteticaV1) { p.Persona.VersionEsperada = 1 }, "produccion": func(p *PlanIdentidadInternaSinteticaV1) { p.Entorno = "produccion" }, "fuente_real": func(p *PlanIdentidadInternaSinteticaV1) { p.AlcanceFuente = "real" }, "vigencia_persona": func(p *PlanIdentidadInternaSinteticaV1) {
		p.Persona.VigenteHasta = p.Persona.VigenteHasta.Add(time.Second)
	}, "hmac_ausente": func(p *PlanIdentidadInternaSinteticaV1) { p.FuenteHMAC.Version = 0 }} {
		t.Run(nombre, func(t *testing.T) {
			p := base
			cambiar(&p)
			if p.Validar() == nil {
				t.Fatal("accepted")
			}
		})
	}
	if base.ValidarEn(base.CaducaEn) == nil {
		t.Fatal("exclusive expiry")
	}
	if base.ValidarEn(base.PreparadoEn) != nil {
		t.Fatal("valid start")
	}
}
func TestIdentidadSinteticaCanonHuella(t *testing.T) {
	p := identidadSinteticaFixture()
	b, h, e := p.CanonicoYHuella()
	if e != nil || len(h) != 64 || bytes.Contains(b, []byte("privilegiada")) {
		t.Fatal("canon")
	}
	p.Organizacion.VersionEsperada++
	_, h2, e := p.CanonicoYHuella()
	if e != nil || h == h2 {
		t.Fatal("version not bound")
	}
}

func TestIdentidadSinteticaLimitesAlineadosSQL(t *testing.T) {
	p := identidadSinteticaFixture()
	p.OperacionRef = "piis_" + strings.Repeat("a", 123)
	if p.Validar() != nil {
		t.Fatal("128 characters and 24 hours must be accepted")
	}
	p.OperacionRef += "a"
	if p.Validar() == nil {
		t.Fatal("129 characters must be rejected")
	}
	p = identidadSinteticaFixture()
	p.CaducaEn = p.CaducaEn.Add(time.Second)
	p.Persona.VigenteHasta = p.CaducaEn
	p.Organizacion.VigenteHasta = p.CaducaEn
	if p.Validar() == nil {
		t.Fatal("more than 24 hours must be rejected")
	}
}

func TestIdentidadSinteticaLimiteTotalReferencias(t *testing.T) {
	for nombre, establecer := range map[string]func(*PlanIdentidadInternaSinteticaV1, string){
		"persona":     func(p *PlanIdentidadInternaSinteticaV1, ref string) { p.Persona.PersonaRef = ref },
		"procedencia": func(p *PlanIdentidadInternaSinteticaV1, ref string) { p.Procedencia.Referencia = ref },
		"hmac":        func(p *PlanIdentidadInternaSinteticaV1, ref string) { p.FuenteHMAC.Referencia = ref },
		"titularidad": func(p *PlanIdentidadInternaSinteticaV1, ref string) { p.Persona.FuenteTitularidad.Referencia = ref },
		"cuenta":      func(p *PlanIdentidadInternaSinteticaV1, ref string) { p.Persona.OperacionCuentaOrdinariaRef = ref },
	} {
		t.Run(nombre, func(t *testing.T) {
			prefijo := "prc_"
			if nombre == "persona" {
				prefijo = "per_"
			}
			if nombre == "cuenta" {
				prefijo = "opr_"
			}
			p := identidadSinteticaFixture()
			ref := prefijo + strings.Repeat("a", 128-len(prefijo))
			establecer(&p, ref)
			if p.Validar() != nil {
				t.Fatal("128 bytes must be accepted")
			}
			establecer(&p, ref+"a")
			if p.Validar() == nil {
				t.Fatal("129 bytes must be rejected")
			}
		})
	}
}

func TestIdentidadSinteticaHuellaOrganizacion(t *testing.T) {
	p := identidadSinteticaFixture()
	_, h, e := p.CanonicoYHuella()
	if e != nil {
		t.Fatal(e)
	}
	p.Organizacion.ProcedenciaHuellaSHA256 = strings.Repeat("b", 64)
	_, otra, e := p.CanonicoYHuella()
	if e != nil || otra == h {
		t.Fatal("organization provenance not bound to plan")
	}
	for _, invalida := range []string{"", strings.Repeat("0", 64), strings.Repeat("A", 64), strings.Repeat("a", 63)} {
		p.Organizacion.ProcedenciaHuellaSHA256 = invalida
		if p.Validar() == nil {
			t.Fatal("invalid organization provenance hash accepted")
		}
	}
}
