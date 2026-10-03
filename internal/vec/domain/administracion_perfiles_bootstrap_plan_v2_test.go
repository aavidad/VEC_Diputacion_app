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
	plan := PlanBootstrapAdministracionV2{Version: 2, PreparadoEn: time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC), CaducaEn: time.Date(2026, 10, 2, 13, 0, 0, 0, time.UTC),
		ControlContinuidadRevisionEsperada: 1, BootstrapEstadoEsperado: "pendiente", Rol: RolBootstrapAdministracion{"rol:administracion_perfiles:v3", h("1"), 1, h("2")},
		FuenteIdentidad: e("autoridad:identidad", "3"), FuenteCA: e("autoridad:ca", "d"), Personas: [2]PersonaBootstrapAdministracion{p, p2},
		Gobierno: GobiernoBootstrapAdministracion{AudienciaAdministrativa: "admin:ensayo:v1", PoliticaCertificadoRef: "politica:admin:ejemplo", PoliticaCertificadoHuellaSHA256: h("4"),
			Roles: []RolGobernadoBootstrapAdministracion{{VersionRef: "rol:administracion_perfiles:v3", HuellaSHA256: h("1"), Clase: ClaseControlPerfilAdministrador,
				AmbitosFijos: []AmbitoFijoBootstrapAdministracion{{Clave: "unidad", Valores: []string{"unidad:ejemplo"}}},
				VigenteDesde: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), VigenteHasta: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC), DuracionPropuestaSegundos: 300}}}}
	plan.Personas[0].Sistemas = []AsignacionSistemasBootstrapAdministracion{{Rol: RolBootstrapAdministracion{"rol:operador_plataforma:v1", h("5"), 1, h("6")}, PerfilRef: ref("prf_", "s"), VinculoRef: ref("vca_", "s"), VigenteHasta: plan.Personas[0].VigenteHasta}}
	plan.Personas[1].Sistemas = []AsignacionSistemasBootstrapAdministracion{}
	plan.Gobierno.AudienciaSelectorADMIN = "admin:selector:ensayo:v1"
	plan.Gobierno.Motivos = []MotivoGobernadoBootstrapAdministracion{}
	plan.Gobierno.Roles[0].CategoriaAdmin = "aplicacion"
	plan.Gobierno.Roles[0].AmbitosFijos = []AmbitoFijoBootstrapAdministracion{{Clave: "administracion", Valores: []string{"perfiles"}}}
	rolSys := plan.Gobierno.Roles[0]
	rolSys.VersionRef = "rol:operador_plataforma:v1"
	rolSys.CategoriaAdmin = "sistemas"
	rolSys.HuellaSHA256 = h("5")
	rolSys.AmbitosFijos = []AmbitoFijoBootstrapAdministracion{{Clave: "administracion", Valores: []string{"sistemas"}}}
	plan.Gobierno.Roles = append(plan.Gobierno.Roles, rolSys)
	return plan
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
	if !strings.Contains(string(b), "perfiles") {
		t.Fatal("ámbito fuera del plan aprobado")
	}
	p.Gobierno.Motivos = []MotivoGobernadoBootstrapAdministracion{{Accion: "administracion.perfiles.consultar", TipoRecurso: "perfil", Finalidad: "gestion_perfiles", Referencia: ReferenciaEntradaCatalogo{CatalogoID: "admin.motivos", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("a", 64), EntradaClave: "motivo_" + strings.Repeat("a", 32)}}}
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

func TestBootstrapKitDosAplicacionYUnSistemas(t *testing.T) {
	for _, caso := range []string{"sin_sistemas", "dos_sistemas", "perfil_reutilizado", "rol_otra_familia", "scope_admin_universal", "motivo_fuera_accion"} {
		t.Run(caso, func(t *testing.T) {
			p := planBootstrapSinteticoV2()
			switch caso {
			case "sin_sistemas":
				p.Personas[0].Sistemas = []AsignacionSistemasBootstrapAdministracion{}
			case "dos_sistemas":
				p.Personas[1].Sistemas = p.Personas[0].Sistemas
			case "perfil_reutilizado":
				p.Personas[0].Sistemas[0].PerfilRef = p.Personas[1].PerfilRef
			case "rol_otra_familia":
				p.Personas[0].Sistemas[0].Rol.VersionRef = "rol:rrhh:v1"
			case "scope_admin_universal":
				p.Gobierno.Roles[0].AmbitosFijos[0].Valores = []string{"*"}
			case "motivo_fuera_accion":
				p.Gobierno.Motivos = []MotivoGobernadoBootstrapAdministracion{{Accion: "bolsa.otorgar", TipoRecurso: "perfil", Finalidad: "gestion_perfiles"}}
			}
			if p.Validar() == nil {
				t.Fatal("kit o binding no admitido")
			}
		})
	}
}

func TestBootstrapMotivosCompatiblesConConsumidorV3(t *testing.T) {
	for _, caso := range []string{"valido", "clave_legible", "huella_cero", "version_fuera_rango"} {
		t.Run(caso, func(t *testing.T) {
			p := planBootstrapSinteticoV2()
			m := ReferenciaEntradaCatalogo{CatalogoID: "admin.motivos", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("a", 64), EntradaClave: "motivo_" + strings.Repeat("b", 32)}
			switch caso {
			case "clave_legible":
				m.EntradaClave = "consulta"
			case "huella_cero":
				m.CatalogoHuellaSHA256 = strings.Repeat("0", 64)
			case "version_fuera_rango":
				m.CatalogoVersion = 1 << 31
			}
			p.Gobierno.Motivos = []MotivoGobernadoBootstrapAdministracion{{Accion: "administracion.perfiles.consultar", TipoRecurso: "perfil", Finalidad: "gestion_perfiles", Referencia: m}}
			if (p.Validar() == nil) != (caso == "valido") {
				t.Fatal("motivo incompatible con V3 aceptado")
			}
		})
	}
}

func TestSelectorYVigenciaSistemasCubiertosPorPlan(t *testing.T) {
	p := planBootstrapSinteticoV2()
	_, h, err := p.CanonicoYHuella()
	if err != nil {
		t.Fatal(err)
	}
	p.Gobierno.AudienciaSelectorADMIN = "admin:selector:otro:v1"
	_, otro, err := p.CanonicoYHuella()
	if err != nil || otro == h {
		t.Fatal("audiencia de selección no cambia la aprobación")
	}
	p.Gobierno.AudienciaSelectorADMIN = ""
	if p.Validar() == nil {
		t.Fatal("selector sin audiencia admitido")
	}
	p = planBootstrapSinteticoV2()
	p.Personas[0].Sistemas[0].VigenteHasta = p.Personas[0].VigenteHasta.Add(time.Hour)
	if p.Validar() == nil {
		t.Fatal("Sistemas supera vigencia de Aplicación de su persona")
	}
}
