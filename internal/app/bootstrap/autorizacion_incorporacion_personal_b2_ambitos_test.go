package bootstrap

import (
	"strings"
	"testing"
	"time"

	personal "vec-diputacion-granada/internal/modules/personal/domain"
	core "vec-diputacion-granada/internal/vec/domain"
)

// perfilesB2AmbitosPrueba compone los perfiles nominales B2 de la
// configuración de prueba y devuelve también el actor y el organismo.
func perfilesB2AmbitosPrueba(t *testing.T) (*perfilesNominalesIncorporacion, core.ContextoActor, string) {
	t.Helper()
	alta, consultas, _ := escenarioConsultasRRHHDesarrolloPrueba(t)
	soporte := alta.soporte
	vinculo, err := soporte.contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	refs := ReferenciasCTIncorporacionDesarrollo{
		PrincipalV3Ref: vinculo.PrincipalID, PerfilV3Ref: vinculo.PerfilActivoRef,
		OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo, UnidadRef: "unidad:desarrollo:rrhh", ActorRef: vinculo.PrincipalID,
	}
	detalle, err := nuevoPerfilNominalIncorporacion(soporte, refs, claveIncorporacionDetalle, nil, soporte.reloj.Ahora())
	if err != nil {
		t.Fatal(err)
	}
	perfiles := &perfilesNominalesIncorporacion{soporte: soporte, consultas: consultas, detalle: detalle}
	config := configuracionB2PuraPrueba().PersonalB2
	if err := extenderPerfilesNominalesB2(perfiles, refs, config, soporte.reloj.Ahora()); err != nil {
		t.Fatal(err)
	}
	actor, err := soporte.contexto.Resultado.Contexto.Clonar()
	if err != nil {
		t.Fatal(err)
	}
	return perfiles, actor, config.OrganismoRef
}

type recursoPersonalB2Prueba struct {
	accion  string
	clave   string // atributo que lleva el objetivo concreto
	recurso func(organismo, objetivo string) core.RecursoAutorizable
}

// recursosPersonalB2Prueba construye, con los constructores reales del
// dominio, los recursos de Personal cuyo objetivo cambia en cada petición.
func recursosPersonalB2Prueba(t *testing.T, actor core.ContextoActor) []recursoPersonalB2Prueba {
	t.Helper()
	fallo := func(err error) {
		if err != nil {
			t.Helper()
			t.Fatal(err)
		}
	}
	plan := func(op string) func(string, string) core.RecursoAutorizable {
		return func(org, obj string) core.RecursoAutorizable {
			m, err := personal.NuevoMaterialConsultarPlanIncorporacionCT(personal.ConsultaPlanIncorporacionCT{PlanRef: obj, OrganismoRef: org, Actor: actor}, op)
			fallo(err)
			return m.Recurso()
		}
	}
	procedencia := personal.ProcedenciaActoEmpleadoB2{ActoRef: "acto:alta", FuenteRef: "fuente:rrhh", FuenteVersion: 1, FuenteHuellaSHA256: strings.Repeat("a", 64), IdempotenciaRef: "11111111-1111-4111-8111-111111111111"}
	catalogo := func(op string) func(string, string) core.RecursoAutorizable {
		return func(org, obj string) core.RecursoAutorizable {
			s := personal.SolicitudCambioCatalogoEmpleadoB2{Operacion: op, OrganismoRef: org, Tipo: "regimen", Ref: obj, Version: 1, Revision: 1,
				Denominacion: "Funcionario interino", VigenteDesde: personal.FechaCivil("2026-01-01"), IdempotenciaRef: "11111111-2222-4333-8444-555555555555", Actor: actor}
			if op == "retirar" {
				s.Revision, s.HuellaSHA256 = 2, strings.Repeat("f", 64)
			} else {
				s.HuellaSHA256 = personal.HuellaPublicacionCatalogoEmpleadoB2(s)
			}
			m, err := personal.NuevoMaterialCambioCatalogoEmpleadoB2(s)
			fallo(err)
			return m.Recurso()
		}
	}
	accionCatalogo := func(op string) string {
		a, ok := accionCatalogoEmpleadoB2(op)
		if !ok {
			t.Fatalf("sin acción de catálogo %s", op)
		}
		return a
	}
	return []recursoPersonalB2Prueba{
		{"personal.plan_incorporacion_ct.preparar", "objetivo_ref", func(org, obj string) core.RecursoAutorizable {
			d := datosPlanB2AmbitosPrueba(org)
			d.IdempotenciaRef = obj
			m, err := personal.NuevoMaterialPrepararPlanIncorporacionCT(personal.SolicitudPlanIncorporacionCT{DatosPlanIncorporacionCT: d, Actor: actor})
			fallo(err)
			return m.Recurso()
		}},
		{"personal.plan_incorporacion_ct.consultar", "objetivo_ref", plan("consultar")},
		{"personal.plan_incorporacion_ct.ejecutar", "objetivo_ref", plan("ejecutar")},
		{"personal.plan_incorporacion_ct.confirmar", "objetivo_ref", plan("confirmar")},
		{"personal.plan_incorporacion_ct.seleccionar", "objetivo_ref", func(org, obj string) core.RecursoAutorizable {
			m, err := personal.NuevoMaterialSeleccionPlanIncorporacionCT(personal.SolicitudSeleccionPlanIncorporacionCT{
				SelectorOrganizacionPlanCT: personal.SelectorOrganizacionPlanCT{PlazaRef: obj, PuestoRef: "puesto:10000000-0000-4000-8000-000000000003", Desde: "2026-10-01"},
				OrganismoRef:               org, Actor: actor})
			fallo(err)
			return m.Recurso()
		}},
		{personal.AccionAltaEmpleadoB2, "objetivo_ref", func(org, obj string) core.RecursoAutorizable {
			m, err := personal.NuevoMaterialAltaEmpleadoB2(personal.SolicitudAltaEmpleadoB2{PersonaRef: obj, OrganismoRef: org, UnidadRef: "unidad:uno",
				Regimen: personal.EntradaCatalogoEmpleadoB2{Ref: "regimen:funcionario", Version: 1}, Modalidad: personal.EntradaCatalogoEmpleadoB2{Ref: "modalidad:interino", Version: 1},
				VigenteDesde: "2026-09-20", Procedencia: procedencia, Actor: actor})
			fallo(err)
			return m.Recurso()
		}},
		{personal.AccionHechoEmpleadoB2, "objetivo_ref", func(org, obj string) core.RecursoAutorizable {
			m, err := personal.NuevoMaterialHechoEmpleadoB2(personal.SolicitudHechoEmpleadoB2{Tipo: "relacion", EmpleadoRef: obj, OrganismoRef: org, RevisionEsperada: 1,
				UnidadRef: "unidad:uno", Regimen: personal.EntradaCatalogoEmpleadoB2{Ref: "regimen:funcionario", Version: 1}, Modalidad: personal.EntradaCatalogoEmpleadoB2{Ref: "modalidad:uno", Version: 1}, Estado: "vigente",
				VigenteDesde: "2026-09-20", Procedencia: procedencia, Actor: actor})
			fallo(err)
			return m.Recurso()
		}},
		{personal.AccionFichaEmpleadoB2, "empleado_ref", func(org, obj string) core.RecursoAutorizable {
			m, err := personal.NuevoMaterialFichaEmpleadoB2(personal.SolicitudFichaEmpleadoB2{EmpleadoRef: obj, OrganismoRef: org, Corte: personal.CorteEmpleadoB2{VigenteEn: "2026-10-01", ConocidoEn: time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)}, Actor: actor})
			fallo(err)
			return m.Recurso()
		}},
		{accionCatalogo("publicar"), "objetivo_ref", catalogo("publicar")},
		{accionCatalogo("retirar"), "objetivo_ref", catalogo("retirar")},
	}
}

func datosPlanB2AmbitosPrueba(org string) personal.DatosPlanIncorporacionCT {
	return personal.DatosPlanIncorporacionCT{IdempotenciaRef: "10000000-0000-4000-8000-000000000001", OrigenCTRef: "ct:plan", OrigenCTReciboRef: "ct:recibo", OrigenCTHuellaSHA256: strings.Repeat("a", 64), ExpedienteRef: "exp:uno", ExpedienteVersion: 7, OrganismoRef: org, UnidadRef: "uni:uno", PersonaRef: "per_" + strings.Repeat("p", 24), PersonaVersion: 1, FuenteBolsaRef: "bolsa:persona", FuenteBolsaVersion: 2, FuenteBolsaReciboRef: "bolsa:recibo", FuenteBolsaHuellaSHA256: strings.Repeat("b", 64), Regimen: personal.EntradaCatalogoEmpleadoB2{Ref: "reg:uno", Version: 1}, Modalidad: personal.EntradaCatalogoEmpleadoB2{Ref: "mod:uno", Version: 2}, Desde: "2026-10-01", PlazaRef: "plaza:10000000-0000-4000-8000-000000000002", PuestoRef: "puesto:10000000-0000-4000-8000-000000000003", ClaseOcupacion: "temporal", VersionPlantillaRef: "plantilla:uno", VersionRPTRef: "rpt:uno", RevisionPlaza: 1, RevisionPuesto: 2, FuenteOrganizacionRef: "organizacion:uno", FuenteOrganizacionHuellaSHA256: strings.Repeat("c", 64), CatalogoRPTID: "rpt:catalogo", CatalogoRPTModulo: "contrataciontemporal", CatalogoRPTCategoria: "categoria:uno", CatalogoRPTVersion: 1, CatalogoRPTHuellaSHA256: strings.Repeat("d", 64), VinculoCTReciboRef: "vinculo:recibo", Procedencia: personal.ProcedenciaActoEmpleadoB2{ActoRef: "acto:incorporacion", FuenteRef: "fuente:ct", FuenteVersion: 7, FuenteHuellaSHA256: strings.Repeat("e", 64), IdempotenciaRef: "10000000-0000-4000-8000-000000000004"}}
}

// objetivosB2AmbitosPrueba da dos objetivos válidos distintos por acción.
func objetivosB2AmbitosPrueba(accion string) (string, string) {
	switch {
	case strings.HasSuffix(accion, ".preparar"):
		return "10000000-0000-4000-8000-0000000000a1", "10000000-0000-4000-8000-0000000000a2"
	case strings.HasSuffix(accion, ".seleccionar"):
		return "plaza:10000000-0000-4000-8000-0000000000b1", "plaza:10000000-0000-4000-8000-0000000000b2"
	case accion == personal.AccionAltaEmpleadoB2:
		return "per_" + strings.Repeat("x", 24), "per_" + strings.Repeat("y", 24)
	case accion == personal.AccionHechoEmpleadoB2 || accion == personal.AccionFichaEmpleadoB2:
		return "emp_" + strings.Repeat("x", 24), "emp_" + strings.Repeat("y", 24)
	case strings.Contains(accion, ".catalogo."):
		return "regimen:funcionario-interino", "regimen:laboral-temporal"
	}
	return "plan:uno", "plan:dos"
}

// Consenso B2 (10/10): el perfil fijo de Personal tiene competencia sobre su
// organismo; el objetivo concreto (plan, plaza, persona, empleado o entrada)
// va en los atributos firmados de la decisión y lo coteja la SQL de Personal
// (Personal41 a Personal44). Cubre sigue estricto.
func TestIncorporacionB2PersonalCompetenciaPorOrganismo(t *testing.T) {
	perfiles, actor, org := perfilesB2AmbitosPrueba(t)
	const ajeno = "organismo:ajeno"
	if ajeno == org {
		t.Fatal("organismo ajeno igual al configurado")
	}
	for _, c := range recursosPersonalB2Prueba(t, actor) {
		perfil := perfiles.b2[c.accion]
		if perfil == nil {
			t.Fatalf("%s sin perfil nominal", c.accion)
		}
		uno, dos := objetivosB2AmbitosPrueba(c.accion)
		r := c.recurso(org, uno)
		if len(r.Ambitos) != 1 || r.Ambitos["organismo_ref"] != org || r.Atributos[c.clave] == "" {
			t.Fatalf("%s: ámbitos %v, atributos %v", c.accion, r.Ambitos, r.Atributos)
		}
		// Condición 4: permitido en su organismo, para cualquier objetivo.
		if !perfil.plantilla.AsignacionPerfil.Cubre(r) || !perfil.plantilla.AsignacionPerfil.Cubre(c.recurso(org, dos)) {
			t.Fatalf("%s: el perfil del organismo no cubre su recurso", c.accion)
		}
		// Denegado entre organismos.
		if perfil.plantilla.AsignacionPerfil.Cubre(c.recurso(ajeno, uno)) {
			t.Fatalf("%s: el perfil cubre un organismo ajeno", c.accion)
		}
		// Denegado si el recurso aún lleva el objetivo como dimensión de ámbito.
		viejo := r
		viejo.Ambitos = map[string]string{"organismo_ref": org, c.clave: r.Atributos[c.clave]}
		if perfil.plantilla.AsignacionPerfil.Cubre(viejo) {
			t.Fatalf("%s: el perfil cubre un recurso con el objetivo en ámbitos", c.accion)
		}
		// Sustituir el objetivo o el material firmado cambia la huella del
		// recurso que coteja la SQL: la decisión de uno no vale para otro.
		h1, err := r.HuellaContextoAutorizacionSHA256()
		if err != nil {
			t.Fatal(err)
		}
		h2, err2 := c.recurso(org, dos).HuellaContextoAutorizacionSHA256()
		otroMaterial := r
		otroMaterial.Atributos = map[string]string{}
		for k, v := range r.Atributos {
			otroMaterial.Atributos[k] = v
		}
		otroMaterial.Atributos["material_sha256"] = strings.Repeat("0", 64)
		h3, err3 := otroMaterial.HuellaContextoAutorizacionSHA256()
		soloAmbito := r
		soloAmbito.Atributos = map[string]string{}
		for k, v := range r.Atributos {
			if k != c.clave {
				soloAmbito.Atributos[k] = v
			}
		}
		h4, err4 := soloAmbito.HuellaContextoAutorizacionSHA256()
		if err2 != nil || err3 != nil || err4 != nil {
			t.Fatalf("%s: huella del recurso: %v %v %v", c.accion, err2, err3, err4)
		}
		if h1 == h2 || h1 == h3 || h1 == h4 || h1 == "" {
			t.Fatalf("%s: objetivo o material no ligados a la huella", c.accion)
		}
	}
}

// Condición 5: consultar el catálogo y gobernarlo siguen en perfiles
// distintos; la lectura conserva sus valores cerrados de objetivo_ref.
func TestIncorporacionB2PersonalConsultaYGobiernoCatalogoSeparados(t *testing.T) {
	perfiles, actor, org := perfilesB2AmbitosPrueba(t)
	publicar, _ := accionCatalogoEmpleadoB2("publicar")
	lectura := perfiles.b2[personal.AccionConsultarCatalogoEmpleadoB2]
	gobierno := perfiles.b2[publicar]
	if lectura == nil || gobierno == nil || lectura == gobierno {
		t.Fatal("consulta y gobierno del catálogo comparten perfil")
	}
	m, err := personal.NuevoMaterialConsultaCatalogoEmpleadoB2(personal.SolicitudConsultaCatalogoEmpleadoB2{OrganismoRef: org, Tipo: "regimen", Estado: "publicada", Limite: 100, Actor: actor})
	if err != nil {
		t.Fatal(err)
	}
	r := m.Recurso()
	if r.Ambitos["objetivo_ref"] != org+":regimen" || len(r.Ambitos) != 2 {
		t.Fatalf("la consulta del catálogo cambió de ámbitos: %v", r.Ambitos)
	}
	if !lectura.plantilla.AsignacionPerfil.Cubre(r) || gobierno.plantilla.AsignacionPerfil.Cubre(r) {
		t.Fatal("la consulta del catálogo salió de su perfil cerrado")
	}
	for _, c := range recursosPersonalB2Prueba(t, actor) {
		if c.accion != publicar {
			continue
		}
		uno, _ := objetivosB2AmbitosPrueba(c.accion)
		if lectura.plantilla.AsignacionPerfil.Cubre(c.recurso(org, uno)) {
			t.Fatal("el perfil de consulta cubre una publicación")
		}
	}
}
