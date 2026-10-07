package domain

import (
	"strings"
	"testing"
	"time"
)

func solicitudActoAdministracionPerfilesPrueba(t *testing.T) SolicitudActoAdministracionPerfiles {
	t.Helper()
	instante := instanteContextoActorPrueba()
	cuenta := solicitudContextoActorPrueba().Cuenta
	actor, err := NuevoContextoActor(cuenta, instantaneaContextoActorPrueba(instante), instante)
	if err != nil {
		t.Fatal(err)
	}
	version := versionRolValidaPrueba()
	asignacion := asignacionPerfilValidaPrueba()
	asignacion.PerfilActivoRef = actor.PerfilActivoRef
	asignacion.PrincipalID = actor.PersonaRef
	politicas := []PoliticaRestrictiva{politicaRestrictivaValidaPrueba("minimizacion")}
	huellaCatalogo, err := HuellaCatalogoPoliticasAutorizacion(politicas)
	if err != nil {
		t.Fatal(err)
	}
	instantanea := InstantaneaAutorizacion{
		AsignacionPerfil: asignacion,
		VersionRol:       version,
		ControlVigenciaVersionRol: ControlVigenciaVersionRol{
			VersionRolRef: version.Referencia(), Revision: 1,
			Estado:         EstadoControlVigenciaVersionRolHabilitada,
			ActualizadoPor: version.PublicadaPor, ActualizadoEn: version.PublicadaEn,
		},
		Politicas: politicas, RevisionCatalogoPoliticas: 3,
		CatalogoPoliticasHuellaSHA256: huellaCatalogo,
	}
	if err := instantanea.Validar(); err != nil {
		t.Fatal(err)
	}
	return SolicitudActoAdministracionPerfiles{
		OperacionRef: "acto_admin:" + strings.Repeat("a", 32),
		Actor:        actor, InstantaneaAutorizacion: instantanea,
		Operacion: OperacionOtorgarPerfil, Clase: ClaseControlPerfilOrdinario,
		RolVersionRef: "rol:tecnico_bolsa:v1",
		Objetivo: PreimagenAdministracionPerfiles{
			UnidadRef: "unidad:prueba",
			CuentaRef: referenciaContextoActorPrueba("cta_", "b"), CuentaVersion: 2,
			PersonaRef: referenciaContextoActorPrueba("per_", "s"), PersonaVersion: 3,
			PerfilRef:      referenciaContextoActorPrueba("prf_", "x"),
			VinculoRef:     referenciaContextoActorPrueba("vca_", "y"),
			HuellaSHA256:   strings.Repeat("c", 64),
			ProcedenciaRef: "procedencia:maestra:1", ProcedenciaVersion: 1,
			ProcedenciaHuellaSHA256: strings.Repeat("f", 64),
			VigenteDesde:            instante,
			VigenteHasta:            instante.Add(24 * time.Hour),
		},
		Motivo: ReferenciaEntradaCatalogo{
			CatalogoID: "motivos_administracion", CatalogoVersion: 1,
			CatalogoHuellaSHA256: strings.Repeat("d", 64), EntradaClave: "provision_nominal",
		},
		CorrelacionRef: "correlacion_" + strings.Repeat("e", 32),
	}
}

func TestEvidenciaSesionAdministracionPerfilesConservaParRegistrado(t *testing.T) {
	ahora := instanteVinculoAutenticacionActorV2Prueba()
	vinculo, resultado, _ := vinculoAutenticacionActorV2Prueba(t, ahora)
	e := EvidenciaSesionAdministracionPerfiles{ResultadoContexto: resultado, Vinculo: vinculo}
	if err := e.ValidarPara(resultado.Contexto); err != nil {
		t.Fatal(err)
	}
	if err := (EvidenciaSesionAdministracionPerfiles{}).ValidarPara(resultado.Contexto); err == nil {
		t.Fatal("ausencia de evidencia admitida")
	}
	otro := resultado.Contexto
	otro.PerfilActivoRef = "prf_" + strings.Repeat("f", 22)
	otro.Instantanea.PerfilActivoRef = otro.PerfilActivoRef
	if err := otro.Validar(); err != nil {
		t.Fatal(err)
	}
	if err := e.ValidarPara(otro); err == nil {
		t.Fatal("evidencia del perfil original aceptada para otro actor")
	}
}

func TestAdministracionPerfilesSeparaActoOrdinarioDePropuestaSensible(t *testing.T) {
	s := solicitudActoAdministracionPerfilesPrueba(t)
	if err := s.Validar(); err != nil {
		t.Fatalf("acto ordinario: %v", err)
	}
	s.Clase = ClaseControlPerfilAdministrador
	if err := s.Validar(); err == nil {
		t.Fatal("propuesta sin revision de continuidad aceptada")
	}
	s.Objetivo.RevisionContinuidad = 1
	s.OperacionRef = "propuesta_admin:" + strings.Repeat("a", 32)
	if err := s.Validar(); err != nil {
		t.Fatalf("propuesta sensible: %v", err)
	}
	s.Objetivo.PerfilVersion = 1
	if err := s.Validar(); err == nil {
		t.Fatal("otorgamiento que recicla perfil aceptado")
	}
}

func TestAdministracionPerfilesRevocacionExigeCASCompleto(t *testing.T) {
	s := solicitudActoAdministracionPerfilesPrueba(t)
	s.Operacion = OperacionRevocarPerfil
	if err := s.Validar(); err == nil {
		t.Fatal("revocacion sin perfil ni vinculo aceptada")
	}
	s.Objetivo.PerfilVersion = 2
	s.Objetivo.VinculoVersion = 3
	s.Objetivo.VigenteDesde = time.Time{}
	s.Objetivo.VigenteHasta = time.Time{}
	if err := s.Validar(); err != nil {
		t.Fatalf("revocacion con CAS: %v", err)
	}
	s.Objetivo.VinculoVersion = 0
	if err := s.Validar(); err == nil {
		t.Fatal("revocacion sin version de vinculo aceptada")
	}
}

func TestAdministracionPerfilesDobleControlPorPersona(t *testing.T) {
	s := solicitudActoAdministracionPerfilesPrueba(t)
	cierre := SolicitudCierrePropuestaAdministracionPerfiles{
		OperacionRef:          "cierre_admin:" + strings.Repeat("b", 32),
		PropuestaRef:          "propuesta_admin:" + strings.Repeat("a", 32),
		PropuestaHuellaSHA256: strings.Repeat("c", 64),
		ProponentePersonaRef:  referenciaContextoActorPrueba("per_", "p"),
		ObjetivoPersonaRef:    s.Objetivo.PersonaRef,
		Aprobador:             s.Actor, InstantaneaAutorizacion: s.InstantaneaAutorizacion,
		Decision: DecisionAprobarPropuestaPerfil, Motivo: s.Motivo,
		CorrelacionRef: s.CorrelacionRef,
	}
	if err := cierre.Validar(); err != nil {
		t.Fatalf("tres personas distintas: %v", err)
	}
	cierre.ProponentePersonaRef = cierre.Aprobador.PersonaRef
	if err := cierre.Validar(); err == nil {
		t.Fatal("autoaprobacion aceptada")
	}
	cierre.ProponentePersonaRef = referenciaContextoActorPrueba("per_", "p")
	cierre.ObjetivoPersonaRef = cierre.Aprobador.PersonaRef
	if err := cierre.Validar(); err == nil {
		t.Fatal("aprobacion del afectado aceptada")
	}
	cierre.ObjetivoPersonaRef = s.Objetivo.PersonaRef
	confirmado := instanteContextoActorPrueba()
	resultado := CierrePropuestaAdministracionPerfiles{
		PropuestaHuellaSHA256: cierre.PropuestaHuellaSHA256,
		OperacionRef:          cierre.OperacionRef,
		PropuestaRef:          cierre.PropuestaRef,
		Decision:              DecisionAprobarPropuestaPerfil,
		HuellaCierreSHA256:    strings.Repeat("d", 64),
		ConfirmadoEn:          confirmado,
		Recibo: &ReciboAdministracionPerfiles{
			ActorPersonaRef:     cierre.Aprobador.PersonaRef,
			PerfilActivoRef:     cierre.Aprobador.PerfilActivoRef,
			AsignacionPerfilRef: cierre.InstantaneaAutorizacion.AsignacionPerfil.Referencia(),
			CorrelacionRef:      cierre.CorrelacionRef, Motivo: cierre.Motivo,
			OperacionRef:        cierre.OperacionRef,
			ActoRef:             "acto_admin:" + strings.Repeat("f", 32),
			ReciboRef:           "recibo_admin:" + strings.Repeat("e", 32),
			PropuestaRef:        cierre.PropuestaRef,
			AuditoriaRef:        "auditoria:admin:1",
			ObjetivoPersonaRef:  cierre.ObjetivoPersonaRef,
			PerfilRef:           s.Objetivo.PerfilRef,
			VinculoRef:          s.Objetivo.VinculoRef,
			EstadoPosterior:     EstadoVinculoContextoActorActivo,
			VersionPosterior:    1,
			HuellaAntesSHA256:   strings.Repeat("a", 64),
			HuellaDespuesSHA256: strings.Repeat("b", 64),
			ConfirmadoEn:        confirmado,
		},
	}
	if err := resultado.ValidarPara(cierre); err != nil {
		t.Fatalf("recibo de la persona objetivo: %v", err)
	}
	resultado.Recibo.ObjetivoPersonaRef = referenciaContextoActorPrueba("per_", "z")
	if err := resultado.ValidarPara(cierre); err == nil {
		t.Fatal("recibo de otra persona aceptado")
	}
}

func TestReciboAdministracionPerfilesNoConfundePropuestaConEfecto(t *testing.T) {
	r := ReciboAdministracionPerfiles{}
	if err := r.Validar(); err == nil {
		t.Fatal("recibo vacio aceptado")
	}
	r = ReciboAdministracionPerfiles{
		OperacionRef:       "acto_admin:" + strings.Repeat("a", 32),
		ActoRef:            "acto_admin:" + strings.Repeat("a", 32),
		ReciboRef:          "recibo_admin:" + strings.Repeat("b", 32),
		AuditoriaRef:       "auditoria:admin:1",
		ObjetivoPersonaRef: referenciaContextoActorPrueba("per_", "s"),
		PerfilRef:          referenciaContextoActorPrueba("prf_", "x"),
		VinculoRef:         referenciaContextoActorPrueba("vca_", "y"),
		EstadoPosterior:    EstadoVinculoContextoActorActivo, VersionPosterior: 1,
		HuellaAntesSHA256: strings.Repeat("c", 64), HuellaDespuesSHA256: strings.Repeat("d", 64),
		ConfirmadoEn: time.Date(2026, 9, 30, 19, 0, 0, 0, time.UTC),
	}
	if err := r.Validar(); err != nil {
		t.Fatalf("recibo durable: %v", err)
	}
	r.ReciboRef = ""
	if err := r.Validar(); err == nil {
		t.Fatal("recibo sin referencia durable aceptado")
	}
}

func TestReferenciaActoOpcionalNoSustituyeProcedencia(t *testing.T) {
	s := solicitudActoAdministracionPerfilesPrueba(t)
	s.ReferenciaActo = "Resolución 2026/123"
	s.Objetivo.UnidadRef = "unidad:prueba"
	if err := s.Validar(); err != nil {
		t.Fatal(err)
	}
	for _, ref := range []string{"acto\n123", " acto:123 ", strings.Repeat("á", 257)} {
		s.ReferenciaActo = ref
		if s.Validar() == nil {
			t.Fatal("referencia no canónica admitida")
		}
	}
	s.ReferenciaActo = ""
	s.Objetivo.ProcedenciaRef = ""
	if s.Validar() == nil {
		t.Fatal("referencia opcional sustituye procedencia obligatoria")
	}
}
func TestUnidadAsignacionNoAdmiteComodin(t *testing.T) {
	s := solicitudActoAdministracionPerfilesPrueba(t)
	s.Objetivo.UnidadRef = "*"
	if s.Validar() == nil {
		t.Fatal("ámbito global admitido")
	}
}

func TestCierreBajaADMINPropiaReconstituyeClaseEnAutoridad(t *testing.T) {
	s := solicitudActoAdministracionPerfilesPrueba(t)
	cierre := SolicitudCierrePropuestaAdministracionPerfiles{OperacionRef: "cierre_admin:" + strings.Repeat("b", 32), PropuestaRef: "propuesta_admin:" + strings.Repeat("a", 32), PropuestaHuellaSHA256: strings.Repeat("c", 64), ProponentePersonaRef: s.Objetivo.PersonaRef, ObjetivoPersonaRef: s.Objetivo.PersonaRef, Aprobador: s.Actor, InstantaneaAutorizacion: s.InstantaneaAutorizacion, Decision: DecisionAprobarPropuestaPerfil, Motivo: s.Motivo, CorrelacionRef: s.CorrelacionRef}
	if cierre.Validar() != nil {
		t.Fatal("igualdad proponente/afectado impedía baja propia antes de recuperar propuesta")
	}
	cierre.Aprobador.PersonaRef = s.Objetivo.PersonaRef
	if cierre.Validar() == nil {
		t.Fatal("autoaprobación admitida")
	}
}

func TestContinuidadAdministradoresBajaPropiaConDobleControl(t *testing.T) {
	s := solicitudActoAdministracionPerfilesPrueba(t)
	actorA, autorizacionA := s.Actor, s.InstantaneaAutorizacion
	instante := instanteContextoActorPrueba()
	instantaneaB := instantaneaContextoActorPrueba(instante)
	instantaneaB.CuentaRef = s.Objetivo.CuentaRef
	instantaneaB.CuentaVersion = s.Objetivo.CuentaVersion
	instantaneaB.PersonaRef = s.Objetivo.PersonaRef
	instantaneaB.PersonaVersion = s.Objetivo.PersonaVersion
	instantaneaB.PerfilActivoRef = s.Objetivo.PerfilRef
	instantaneaB.VinculoRef = s.Objetivo.VinculoRef
	cuentaB := solicitudContextoActorPrueba().Cuenta
	cuentaB.CuentaRef = instantaneaB.CuentaRef
	actorB, err := NuevoContextoActor(cuentaB, instantaneaB, instante)
	if err != nil {
		t.Fatal(err)
	}
	s.Actor = actorB
	s.InstantaneaAutorizacion.AsignacionPerfil.PrincipalID = actorB.PersonaRef
	s.InstantaneaAutorizacion.AsignacionPerfil.PerfilActivoRef = actorB.PerfilActivoRef
	s.Clase = ClaseControlPerfilAdministrador
	s.Operacion = OperacionRevocarPerfil
	s.OperacionRef = "propuesta_admin:" + strings.Repeat("a", 32)
	s.RolVersionRef = "rol:administracion_perfiles:v2"
	s.Objetivo.PerfilVersion = actorB.Instantanea.PerfilVersion
	s.Objetivo.VinculoVersion = actorB.Instantanea.VinculoVersion
	s.Objetivo.RevisionContinuidad = 1
	s.Objetivo.VigenteDesde = time.Time{}
	s.Objetivo.VigenteHasta = time.Time{}
	if err := s.Validar(); err != nil {
		t.Fatalf("B propone su propia baja: %v", err)
	}
	cierre := SolicitudCierrePropuestaAdministracionPerfiles{
		OperacionRef:          "cierre_admin:" + strings.Repeat("b", 32),
		PropuestaRef:          s.OperacionRef,
		PropuestaHuellaSHA256: strings.Repeat("c", 64),
		ProponentePersonaRef:  actorB.PersonaRef,
		ObjetivoPersonaRef:    actorB.PersonaRef,
		Aprobador:             actorA, InstantaneaAutorizacion: autorizacionA,
		Decision: DecisionAprobarPropuestaPerfil, Motivo: s.Motivo,
		CorrelacionRef: s.CorrelacionRef,
	}
	if err := cierre.Validar(); err != nil {
		t.Fatalf("A aprueba baja de B: %v", err)
	}
	if err := (ContinuidadAdministradores{EfectivosAntes: 2, EfectivosDespues: 1}).ValidarBaja(); err != nil {
		t.Fatalf("2 a 1 debe conservar continuidad: %v", err)
	}
	if (ContinuidadAdministradores{EfectivosAntes: 2, EfectivosDespues: 1}).AdmiteNuevoActoSensible() {
		t.Fatal("un solo administrador permite nuevo doble control")
	}
	if err := (ContinuidadAdministradores{EfectivosAntes: 1, EfectivosDespues: 0}).ValidarBaja(); err == nil {
		t.Fatal("1 a 0 aceptado")
	}
	cierre.Aprobador = actorB
	cierre.InstantaneaAutorizacion = s.InstantaneaAutorizacion
	if err := cierre.Validar(); err == nil {
		t.Fatal("B aprobo su propia baja")
	}
	s.Operacion = OperacionOtorgarPerfil
	s.Objetivo.PerfilVersion = 0
	s.Objetivo.VinculoVersion = 0
	s.Objetivo.VigenteHasta = instante.Add(24 * time.Hour)
	if err := s.Validar(); err == nil {
		t.Fatal("autoalta de administrador aceptada")
	}
	s.Operacion = OperacionRevocarPerfil
	s.Clase = ClaseControlPerfilIntervencion
	s.Objetivo.PerfilVersion = actorB.Instantanea.PerfilVersion
	s.Objetivo.VinculoVersion = actorB.Instantanea.VinculoVersion
	s.Objetivo.VigenteDesde = time.Time{}
	s.Objetivo.VigenteHasta = time.Time{}
	if err := s.Validar(); err == nil {
		t.Fatal("autobaja de Intervencion aceptada sin fuente")
	}
}
