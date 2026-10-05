package domain

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func planBootstrapSinteticoV3() PlanBootstrapAdministracionV3 {
	v2 := planBootstrapSinteticoV2()
	p := PlanBootstrapAdministracionV3{Version: 3, PreparadoEn: v2.PreparadoEn, CaducaEn: v2.CaducaEn,
		ControlContinuidadRevisionEsperada: v2.ControlContinuidadRevisionEsperada, BootstrapEstadoEsperado: v2.BootstrapEstadoEsperado,
		Rol: v2.Rol, FuenteIdentidad: v2.FuenteIdentidad, FuenteCA: v2.FuenteCA,
		FuenteRepartoAprobado: EvidenciaBootstrapAdministracion{Referencia: "fuente:reparto:sintetico", Version: 1, HuellaSHA256: strings.Repeat("9", 64)}}
	p.Rol.VersionRef = "rol:administracion_perfiles:v4"
	ambitos := func() []AmbitoBootstrapAdministracionV3 {
		return []AmbitoBootstrapAdministracionV3{{Dimension: "organizacion_ref", Valores: []string{"organizacion:sintetica"},
			Fuente: EvidenciaBootstrapAdministracion{Referencia: "fuente:organizacion:sintetica", Version: 1, HuellaSHA256: strings.Repeat("8", 64)}}}
	}
	for i, persona := range v2.Personas {
		p.Personas[i] = PersonaBootstrapAdministracionV3{CuentaRef: persona.CuentaRef, CuentaVersion: persona.CuentaVersion,
			PersonaRef: persona.PersonaRef, PersonaVersion: persona.PersonaVersion, PerfilRef: persona.PerfilRef, VinculoRef: persona.VinculoRef,
			PreimagenHuellaSHA256: persona.PreimagenHuellaSHA256, Procedencia: persona.Procedencia, VigenteHasta: persona.VigenteHasta,
			Certificado: persona.Certificado, Ambitos: ambitos(), Sistemas: []AsignacionSistemasBootstrapAdministracionV3{}}
		for _, s := range persona.Sistemas {
			p.Personas[i].Sistemas = append(p.Personas[i].Sistemas, AsignacionSistemasBootstrapAdministracionV3{
				Rol: s.Rol, PerfilRef: s.PerfilRef, VinculoRef: s.VinculoRef, VigenteHasta: s.VigenteHasta, Ambitos: ambitos()})
		}
	}
	g := v2.Gobierno
	p.Gobierno = GobiernoBootstrapAdministracionV3{AudienciaSelectorADMIN: g.AudienciaSelectorADMIN, AudienciaAdministrativa: g.AudienciaAdministrativa,
		PoliticaCertificadoRef: g.PoliticaCertificadoRef, PoliticaCertificadoHuellaSHA256: g.PoliticaCertificadoHuellaSHA256, Motivos: g.Motivos}
	for i, r := range g.Roles {
		version := uint64(1)
		rolID := "operador_plataforma"
		if i == 0 {
			r.VersionRef, version = "rol:administracion_perfiles:v4", 4
			rolID = "administracion_perfiles"
		}
		p.Gobierno.Roles = append(p.Gobierno.Roles, RolGobernadoBootstrapAdministracionV3{VersionRef: r.VersionRef, HuellaSHA256: r.HuellaSHA256,
			Clase: r.Clase, CategoriaAdmin: r.CategoriaAdmin, UnidadRequerida: false,
			AmbitosFijos: []AmbitoFijoBootstrapAdministracion{{Clave: "organizacion_ref", Valores: []string{"organizacion:sintetica"}}},
			VigenteDesde: r.VigenteDesde, VigenteHasta: r.VigenteHasta, DuracionPropuestaSegundos: r.DuracionPropuestaSegundos,
			FuenteCategoria:   EvidenciaBootstrapAdministracion{Referencia: r.VersionRef, Version: version, HuellaSHA256: r.HuellaSHA256},
			DimensionesAmbito: []string{"organizacion_ref"}, RolID: rolID})
	}
	return p
}

func TestBootstrapV3HuellaCubreFuentesYAmbitosSinCambiarV2(t *testing.T) {
	p := planBootstrapSinteticoV3()
	b, huella, err := p.CanonicoYHuella()
	if err != nil {
		t.Fatal(err)
	}
	suma := sha256.Sum256(b)
	if huella != hex.EncodeToString(suma[:]) || bytes.Contains(b, []byte(`"plan_v2"`)) {
		t.Fatal("canon o huella divergente")
	}
	for _, cambiar := range []func(*PlanBootstrapAdministracionV3){
		func(p *PlanBootstrapAdministracionV3) { p.FuenteRepartoAprobado.Version++ },
		func(p *PlanBootstrapAdministracionV3) { p.Personas[0].Ambitos[0].Fuente.Version++ },
		func(p *PlanBootstrapAdministracionV3) { p.Personas[0].Sistemas[0].Ambitos[0].Fuente.Version++ },
	} {
		otro := planBootstrapSinteticoV3()
		cambiar(&otro)
		_, h, err := otro.CanonicoYHuella()
		if err != nil || h == huella {
			t.Fatal("fuente cambiada no altera aprobación necesaria")
		}
	}
	if planBootstrapSinteticoV2().Validar() != nil {
		t.Fatal("V2 dejó de ser compatible")
	}
}

func TestBootstrapV3RechazaFuentesIncompletasYAsignacionesFueraDelKit(t *testing.T) {
	casos := map[string]func(*PlanBootstrapAdministracionV3){
		"fuente_reparto_ausente":    func(p *PlanBootstrapAdministracionV3) { p.FuenteRepartoAprobado.HuellaSHA256 = "" },
		"sha_no_resuelto":           func(p *PlanBootstrapAdministracionV3) { p.Rol.HuellaSHA256 = "" },
		"organizacion_sin_fuente":   func(p *PlanBootstrapAdministracionV3) { p.Personas[0].Ambitos[0].Fuente.Version = 0 },
		"sin_organizacion":          func(p *PlanBootstrapAdministracionV3) { p.Personas[0].Ambitos = nil },
		"ambito_legacy":             func(p *PlanBootstrapAdministracionV3) { p.Personas[0].Ambitos[0].Dimension = "administracion" },
		"organizacion_fuera_fuente": func(p *PlanBootstrapAdministracionV3) { p.Personas[0].Ambitos[0].Valores[0] = "organizacion:otra" },
		"comodin":                   func(p *PlanBootstrapAdministracionV3) { p.Personas[0].Ambitos[0].Valores[0] = "*" },
		"categoria_sin_fuente":      func(p *PlanBootstrapAdministracionV3) { p.Gobierno.Roles[1].FuenteCategoria.HuellaSHA256 = "" },
		"rol_id_ajeno": func(p *PlanBootstrapAdministracionV3) {
			p.Gobierno.Roles[0].RolID = "operador_plataforma"
		},
		"otra_categoria": func(p *PlanBootstrapAdministracionV3) { p.Gobierno.Roles[1].CategoriaAdmin = "aplicacion" },
		"rol_extra": func(p *PlanBootstrapAdministracionV3) {
			p.Gobierno.Roles = append(p.Gobierno.Roles, p.Gobierno.Roles[1])
		},
		"sin_sistemas": func(p *PlanBootstrapAdministracionV3) {
			p.Personas[0].Sistemas = []AsignacionSistemasBootstrapAdministracionV3{}
		},
		"dos_sistemas":      func(p *PlanBootstrapAdministracionV3) { p.Personas[1].Sistemas = p.Personas[0].Sistemas },
		"perfil_compartido": func(p *PlanBootstrapAdministracionV3) { p.Personas[0].Sistemas[0].PerfilRef = p.Personas[1].PerfilRef },
		"certificado_otra_persona": func(p *PlanBootstrapAdministracionV3) {
			p.Personas[0].Certificado.PersonaRef = p.Personas[1].PersonaRef
		},
	}
	for nombre, cambiar := range casos {
		t.Run(nombre, func(t *testing.T) {
			p := planBootstrapSinteticoV3()
			cambiar(&p)
			if p.Validar() == nil {
				t.Fatal("material fuera del reparto y de las fuentes aceptado")
			}
		})
	}
}

func TestBootstrapV3UnidadSoloSegunLimiteGobernadoYConFuente(t *testing.T) {
	p := planBootstrapSinteticoV3()
	unidad := AmbitoBootstrapAdministracionV3{Dimension: "unidad_ref", Valores: []string{"unidad:sintetica"},
		Fuente: EvidenciaBootstrapAdministracion{Referencia: "fuente:unidad:sintetica", Version: 1, HuellaSHA256: strings.Repeat("7", 64)}}
	p.Personas[0].Ambitos = append(p.Personas[0].Ambitos, unidad)
	if p.Validar() == nil {
		t.Fatal("unidad sin dimensión publicada admitida")
	}
	p.Gobierno.Roles[0].DimensionesAmbito = append(p.Gobierno.Roles[0].DimensionesAmbito, "unidad_ref")
	p.Gobierno.Roles[0].AmbitosFijos = append(p.Gobierno.Roles[0].AmbitosFijos, AmbitoFijoBootstrapAdministracion{Clave: "unidad_ref", Valores: []string{"unidad:sintetica"}})
	if p.Validar() != nil {
		t.Fatal("unidad opcional acreditada rechazada")
	}
	p.Gobierno.Roles[0].UnidadRequerida = true
	if p.Validar() == nil {
		t.Fatal("segunda asignación sin unidad obligatoria admitida")
	}
	p.Personas[1].Ambitos = append(p.Personas[1].Ambitos, unidad)
	if p.Validar() != nil {
		t.Fatal("unidad obligatoria acreditada rechazada")
	}
	p.Personas[1].Ambitos[1].Fuente.HuellaSHA256 = ""
	if p.Validar() == nil {
		t.Fatal("unidad sin fuente admitida")
	}
}

func TestBootstrapV3CamposCanonicosPactados(t *testing.T) {
	b, _, err := planBootstrapSinteticoV3().CanonicoYHuella()
	if err != nil {
		t.Fatal(err)
	}
	var documento map[string]json.RawMessage
	if json.Unmarshal(b, &documento) != nil || len(documento) != 11 || documento["fuente_reparto_aprobado"] == nil {
		t.Fatal("ABI incompleta")
	}
	var otro PlanBootstrapAdministracionV3
	if json.Unmarshal(b, &otro) != nil || !reflect.DeepEqual(planBootstrapSinteticoV3(), otro) {
		t.Fatal("el canon no conserva su material")
	}
}

func TestBootstrapV3VersionesProcedenDelCatalogoAportado(t *testing.T) {
	p := planBootstrapSinteticoV3()
	p.Rol.VersionRef = "rol:administracion_perfiles:v9"
	p.Gobierno.Roles[0].VersionRef = p.Rol.VersionRef
	p.Gobierno.Roles[0].FuenteCategoria.Referencia = p.Rol.VersionRef
	p.Gobierno.Roles[0].FuenteCategoria.Version = 9
	p.Personas[0].Sistemas[0].Rol.VersionRef = "rol:operador_plataforma:v2"
	p.Gobierno.Roles[1].VersionRef = p.Personas[0].Sistemas[0].Rol.VersionRef
	p.Gobierno.Roles[1].FuenteCategoria.Referencia = p.Gobierno.Roles[1].VersionRef
	p.Gobierno.Roles[1].FuenteCategoria.Version = 2
	if p.Validar() != nil {
		t.Fatal("versión del catálogo rechazada por una constante de implementación")
	}
	p.Gobierno.Roles[1].VersionRef = "rol:operador_plataforma:v3"
	if p.Validar() == nil {
		t.Fatal("referencia que no figura en el catálogo admitida")
	}
}
