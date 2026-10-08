package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func planFuentesFixture() PlanFuentesInicialesAdminV1 {
	ahora := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	e := EvidenciaFuentesInicialesAdmin{Referencia: "prc_" + strings.Repeat("e", 22), Version: 1, HuellaSHA256: strings.Repeat("a", 64)}
	persona := func(id string) PersonaFuentesInicialesAdmin {
		return PersonaFuentesInicialesAdmin{PersonaRef: "per_" + strings.Repeat(id, 22), VigenteHasta: ahora.Add(24 * time.Hour), OperacionCuentaOrdinariaRef: "opr_" + strings.Repeat(id, 22) + "o", OperacionCuentaPrivilegiadaRef: "opr_" + strings.Repeat(id, 22) + "p", FuenteTitularidad: e}
	}
	return PlanFuentesInicialesAdminV1{Version: 1, OperacionRef: "pfi_" + strings.Repeat("z", 22), PreparadoEn: ahora, CaducaEn: ahora.Add(time.Hour), Entorno: "desarrollo", AlcanceFuente: "sintetico_declarado", Procedencia: e,
		Organizacion: OrganizacionFuentesInicialesAdmin{OrganizacionRef: "org_" + strings.Repeat("a", 16), VigenteHasta: ahora.Add(48 * time.Hour)}, Personas: [2]PersonaFuentesInicialesAdmin{persona("a"), persona("b")}, FuenteHMAC: e,
		PoliticaADMIN: PoliticaFuentesInicialesAdmin{PoliticaRef: "pga_" + strings.Repeat("p", 22), HostADMIN: "admin.example.invalid", CAHuellaSHA256: strings.Repeat("c", 64), HuellaAprobacionSHA256: strings.Repeat("d", 64), MaximaEdadRevocacionSegundos: 60, VigenteHasta: ahora.Add(24 * time.Hour)}}
}
func TestPlanFuentesInicialesCanon(t *testing.T) {
	p := planFuentesFixture()
	b, h, err := p.CanonicoYHuella()
	if err != nil {
		t.Fatal(err)
	}
	esperado, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(esperado)
	if h != "e65773fb60cfc69f6d3b504205f05169ea1053617af4c79a6a29e7ef60d67a88" {
		t.Fatal("vector V1 divergente")
	}
	if string(b) != string(esperado) || h != hex.EncodeToString(digest[:]) {
		t.Fatal("canon divergente")
	}
	b2, h2, err := p.CanonicoYHuella()
	if err != nil || string(b2) != string(b) || h2 != h {
		t.Fatal("canon inestable")
	}
	p.PoliticaADMIN.CAHuellaSHA256 = strings.Repeat("e", 64)
	_, nuevo, err := p.CanonicoYHuella()
	if err != nil || nuevo == h {
		t.Fatal("no compromete politica")
	}
}
func TestPlanFuentesInicialesRechaza(t *testing.T) {
	casos := map[string]func(*PlanFuentesInicialesAdminV1){
		"produccion":             func(p *PlanFuentesInicialesAdminV1) { p.Entorno = "produccion" },
		"alcance":                func(p *PlanFuentesInicialesAdminV1) { p.AlcanceFuente = "real" },
		"version":                func(p *PlanFuentesInicialesAdminV1) { p.Version = 2 },
		"organizacion_existente": func(p *PlanFuentesInicialesAdminV1) { p.Organizacion.VersionEsperada = 1 },
		"persona_existente":      func(p *PlanFuentesInicialesAdminV1) { p.Personas[0].VersionEsperada = 1 },
		"version_fuente":         func(p *PlanFuentesInicialesAdminV1) { p.FuenteHMAC.Version = 2 },
		"huella":                 func(p *PlanFuentesInicialesAdminV1) { p.FuenteHMAC.HuellaSHA256 = strings.Repeat("A", 64) },
		"persona_duplicada":      func(p *PlanFuentesInicialesAdminV1) { p.Personas[1].PersonaRef = p.Personas[0].PersonaRef },
		"desorden":               func(p *PlanFuentesInicialesAdminV1) { p.Personas[0], p.Personas[1] = p.Personas[1], p.Personas[0] },
		"operacion_duplicada": func(p *PlanFuentesInicialesAdminV1) {
			p.Personas[1].OperacionCuentaOrdinariaRef = p.Personas[0].OperacionCuentaPrivilegiadaRef
		},
		"cuenta_cliente": func(p *PlanFuentesInicialesAdminV1) {
			p.Personas[0].OperacionCuentaOrdinariaRef = "cta_" + strings.Repeat("a", 22)
		},
		"fecha_fraccion": func(p *PlanFuentesInicialesAdminV1) { p.PreparadoEn = p.PreparadoEn.Add(time.Nanosecond) },
		"fecha_zona":     func(p *PlanFuentesInicialesAdminV1) { p.PreparadoEn = p.PreparadoEn.In(time.FixedZone("offset", 3600)) },
		"fecha_overflow": func(p *PlanFuentesInicialesAdminV1) {
			p.PoliticaADMIN.VigenteHasta = time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)
		},
		"caducidad": func(p *PlanFuentesInicialesAdminV1) { p.CaducaEn = p.PreparadoEn },
		"vigencia":  func(p *PlanFuentesInicialesAdminV1) { p.Personas[0].VigenteHasta = p.CaducaEn.Add(-time.Second) },
		"org_formato": func(p *PlanFuentesInicialesAdminV1) {
			p.Organizacion.OrganizacionRef = "org_" + strings.Repeat("-", 22)
		},
		"huella_cero":     func(p *PlanFuentesInicialesAdminV1) { p.FuenteHMAC.HuellaSHA256 = strings.Repeat("0", 64) },
		"ca_cero":         func(p *PlanFuentesInicialesAdminV1) { p.PoliticaADMIN.CAHuellaSHA256 = strings.Repeat("0", 64) },
		"operacion_larga": func(p *PlanFuentesInicialesAdminV1) { p.OperacionRef = "pfi_" + strings.Repeat("a", 125) },
		"host_corto":      func(p *PlanFuentesInicialesAdminV1) { p.PoliticaADMIN.HostADMIN = "a.b" },
		"operacion_procedencia": func(p *PlanFuentesInicialesAdminV1) {
			p.Personas[0].OperacionCuentaOrdinariaRef = "prc_" + strings.Repeat("a", 22)
		},
		"host_url":            func(p *PlanFuentesInicialesAdminV1) { p.PoliticaADMIN.HostADMIN = "https://admin.example.invalid" },
		"revocacion_cero":     func(p *PlanFuentesInicialesAdminV1) { p.PoliticaADMIN.MaximaEdadRevocacionSegundos = 0 },
		"revocacion_overflow": func(p *PlanFuentesInicialesAdminV1) { p.PoliticaADMIN.MaximaEdadRevocacionSegundos = 1 << 63 },
	}
	for nombre, cambiar := range casos {
		t.Run(nombre, func(t *testing.T) {
			p := planFuentesFixture()
			cambiar(&p)
			if p.Validar() == nil {
				t.Fatal("acepta invalido")
			}
			if b, h, e := p.CanonicoYHuella(); e == nil || b != nil || h != "" {
				t.Fatal("canon de invalido")
			}
		})
	}
}
func TestPlanFuentesInicialesReloj(t *testing.T) {
	p := planFuentesFixture()
	if p.ValidarEn(p.PreparadoEn) != nil || p.ValidarEn(p.CaducaEn.Add(-time.Second)) != nil {
		t.Fatal("vigencia valida denegada")
	}
	for _, ahora := range []time.Time{{}, p.PreparadoEn.Add(-time.Second), p.CaducaEn, p.CaducaEn.Add(time.Second)} {
		if p.ValidarEn(ahora) == nil {
			t.Fatal("vigencia invalida aceptada")
		}
	}
}
