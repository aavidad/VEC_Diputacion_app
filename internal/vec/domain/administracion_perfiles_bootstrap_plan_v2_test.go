package domain

import (
	"strings"
	"testing"
	"time"
)

func planBootstrapSinteticoV2() PlanBootstrapAdministracionV2 {
	h := func(c string) string { return strings.Repeat(c, 64) }
	ref := func(prefijo, c string) string { return prefijo + strings.Repeat(c, 24) }
	e := func(nombre, c string) EvidenciaBootstrapAdministracion {
		return EvidenciaBootstrapAdministracion{nombre, 1, h(c)}
	}
	p := PersonaBootstrapAdministracion{CuentaRef: ref("cta_", "a"), CuentaVersion: 2, PersonaRef: ref("per_", "a"), PersonaVersion: 3,
		PerfilRef: ref("prf_", "a"), VinculoRef: ref("vca_", "a"), PreimagenHuellaSHA256: h("a"), Procedencia: e("fuente:identidad:uno", "b"), VigenteHasta: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
		Certificado: CertificadoBootstrapAdministracion{PersonaRef: ref("per_", "a"), CuentaRef: ref("cta_", "a"), HuellaSHA256: h("c"), CAHuellaSHA256: h("d"), Acreditacion: e("cert:uno", "e")}}
	p2 := p
	p2.CuentaRef = ref("cta_", "z")
	p2.PersonaRef = ref("per_", "z")
	p2.PerfilRef = ref("prf_", "z")
	p2.VinculoRef = ref("vca_", "z")
	p2.Certificado.PersonaRef = p2.PersonaRef
	p2.Certificado.CuentaRef = p2.CuentaRef
	p2.Certificado.HuellaSHA256 = h("f")
	return PlanBootstrapAdministracionV2{Version: 2, PreparadoEn: time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC), CaducaEn: time.Date(2026, 10, 2, 13, 0, 0, 0, time.UTC),
		ControlContinuidadRevisionEsperada: 1, BootstrapEstadoEsperado: "pendiente", Rol: RolBootstrapAdministracion{"rol:administracion_perfiles:v3", h("1"), 1, h("2")},
		FuenteIdentidad: e("autoridad:identidad", "3"), FuenteCA: e("autoridad:ca", "d"), Personas: [2]PersonaBootstrapAdministracion{p, p2},
		Gobierno: GobiernoBootstrapAdministracion{AudienciaAdministrativa: "admin:ensayo:v1", PoliticaCertificadoRef: "politica:admin:ejemplo", PoliticaCertificadoHuellaSHA256: h("4"),
			Roles: []RolGobernadoBootstrapAdministracion{{VersionRef: "rol:administracion_perfiles:v3", HuellaSHA256: h("1"), Clase: ClaseControlPerfilAdministrador,
				AmbitosFijos: []AmbitoFijoBootstrapAdministracion{{Clave: "unidad", Valores: []string{"unidad:ejemplo"}}},
				VigenteDesde: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), VigenteHasta: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC), DuracionPropuestaSegundos: 300}}}}
}

func TestBootstrapV2GobiernoFinitoEnHuella(t *testing.T) {
	p := planBootstrapSinteticoV2()
	b, h, err := p.CanonicoYHuella()
	if err != nil {
		t.Fatal(err)
	}
	preimagen, err := p.Preimagen()
	if err != nil || preimagen.Validar() != nil || preimagen.HuellaPlanSHA256 != h {
		t.Fatal("preimagen divergente")
	}
	if !strings.Contains(string(b), "unidad:ejemplo") {
		t.Fatal("ámbito fuera del plan aprobado")
	}
	p.Gobierno.Roles[0].AmbitosFijos[0].Valores[0] = "unidad:otra"
	_, nuevo, err := p.CanonicoYHuella()
	if err != nil || nuevo == h {
		t.Fatal("ámbito no altera huella")
	}
	preimagen.PlanV2 = &p
	if preimagen.Validar() == nil {
		t.Fatal("configuración alterada mantiene aprobación previa")
	}
}

func TestBootstrapV2RechazaAmbitoUniversalOCatalogoIncompleto(t *testing.T) {
	for _, caso := range []string{"sin_ambito", "comodin", "duplicado", "rol_ajeno", "TTL_cero", "unidad_admin", "politica_ausente", "fuente_v1"} {
		t.Run(caso, func(t *testing.T) {
			p := planBootstrapSinteticoV2()
			switch caso {
			case "sin_ambito":
				p.Gobierno.Roles[0].AmbitosFijos = []AmbitoFijoBootstrapAdministracion{}
			case "comodin":
				p.Gobierno.Roles[0].AmbitosFijos[0].Valores = []string{"*"}
			case "duplicado":
				p.Gobierno.Roles = append(p.Gobierno.Roles, p.Gobierno.Roles[0])
			case "rol_ajeno":
				p.Rol.VersionRef = "rol:otro:v3"
			case "TTL_cero":
				p.Gobierno.Roles[0].DuracionPropuestaSegundos = 0
			case "unidad_admin":
				p.Gobierno.Roles[0].UnidadRequerida = true
			case "politica_ausente":
				p.Gobierno.PoliticaCertificadoRef = ""
			case "fuente_v1":
				p.Version = 1
			}
			if p.Validar() == nil {
				t.Fatal("configuración incompleta o universal aceptada")
			}
		})
	}
}
