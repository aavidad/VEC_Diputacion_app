package domain

import (
	"strings"
	"testing"
	"time"
)

func TestBootstrapAdministracionPerfilesExigeDosPersonasAcreditadas(t *testing.T) {
	persona := func(letra string) PreimagenAdministracionPerfiles {
		return PreimagenAdministracionPerfiles{
			CuentaRef: "cta_" + strings.Repeat(letra, 22), CuentaVersion: 1,
			PersonaRef: "per_" + strings.Repeat(letra, 22), PersonaVersion: 1,
			PerfilRef:      "prf_" + strings.Repeat(letra, 22),
			VinculoRef:     "vca_" + strings.Repeat(letra, 22),
			HuellaSHA256:   strings.Repeat("a", 64),
			ProcedenciaRef: "procedencia:maestra:1", ProcedenciaVersion: 1,
			ProcedenciaHuellaSHA256: strings.Repeat("b", 64),
			VigenteHasta:            time.Date(2027, 9, 30, 0, 0, 0, 0, time.UTC),
		}
	}
	plan := PreimagenBootstrapAdministracionPerfiles{
		Primera: persona("a"), Segunda: persona("b"), HuellaPlanSHA256: strings.Repeat("c", 64),
	}
	if err := plan.Validar(); err != nil {
		t.Fatalf("dos personas: %v", err)
	}
	recibo := ReciboBootstrapAdministracionPerfiles{
		ActoRef:           "acto_admin:" + strings.Repeat("d", 32),
		ReciboRef:         "recibo_admin:" + strings.Repeat("e", 32),
		HuellaPlanSHA256:  plan.HuellaPlanSHA256,
		PrimeraPersonaRef: plan.Primera.PersonaRef,
		SegundaPersonaRef: plan.Segunda.PersonaRef,
		ConfirmadoEn:      time.Date(2026, 9, 30, 19, 0, 0, 0, time.UTC),
	}
	if err := recibo.ValidarPara(plan); err != nil {
		t.Fatalf("recibo unico: %v", err)
	}
	recibo.SegundaPersonaRef = recibo.PrimeraPersonaRef
	if err := recibo.ValidarPara(plan); err == nil {
		t.Fatal("recibo para la misma persona aceptado")
	}
	plan.Segunda.PersonaRef = plan.Primera.PersonaRef
	if err := plan.Validar(); err == nil {
		t.Fatal("dos cuentas de la misma persona aceptadas")
	}
	plan.Segunda = persona("b")
	plan.Segunda.PersonaVersion = 0
	if err := plan.Validar(); err == nil {
		t.Fatal("persona sin version acreditada aceptada")
	}
	plan.Segunda = persona("b")
	plan.Primera.RevisionContinuidad = 1
	if err := plan.Validar(); err == nil {
		t.Fatal("bootstrap sobre continuidad ya iniciada aceptado")
	}
}

func cierreAdministracionPerfilesLigadoPrueba(t *testing.T) (SolicitudCierrePropuestaAdministracionPerfiles, CierrePropuestaAdministracionPerfiles) {
	t.Helper()
	acto := solicitudActoAdministracionPerfilesPrueba(t)
	solicitud := SolicitudCierrePropuestaAdministracionPerfiles{
		OperacionRef: "cierre_admin:" + strings.Repeat("a", 32), PropuestaRef: "propuesta_admin:" + strings.Repeat("b", 32),
		PropuestaHuellaSHA256: strings.Repeat("c", 64), ProponentePersonaRef: referenciaContextoActorPrueba("per_", "p"),
		ObjetivoPersonaRef: acto.Objetivo.PersonaRef, Aprobador: acto.Actor, InstantaneaAutorizacion: acto.InstantaneaAutorizacion,
		Decision: DecisionAprobarPropuestaPerfil, Motivo: acto.Motivo, CorrelacionRef: acto.CorrelacionRef,
	}
	instante := instanteContextoActorPrueba()
	cierre := CierrePropuestaAdministracionPerfiles{
		OperacionRef: solicitud.OperacionRef, PropuestaRef: solicitud.PropuestaRef, PropuestaHuellaSHA256: solicitud.PropuestaHuellaSHA256, Decision: solicitud.Decision,
		HuellaCierreSHA256: strings.Repeat("d", 64), ConfirmadoEn: instante,
		Recibo: &ReciboAdministracionPerfiles{
			ActorPersonaRef: solicitud.Aprobador.PersonaRef, PerfilActivoRef: solicitud.Aprobador.PerfilActivoRef,
			AsignacionPerfilRef: solicitud.InstantaneaAutorizacion.AsignacionPerfil.Referencia(),
			CorrelacionRef:      solicitud.CorrelacionRef, Motivo: solicitud.Motivo,
			OperacionRef: solicitud.OperacionRef, ActoRef: "acto_admin:" + strings.Repeat("e", 32),
			ReciboRef: "recibo_admin:" + strings.Repeat("f", 32), PropuestaRef: solicitud.PropuestaRef,
			AuditoriaRef: "auditoria:admin:ligadura", ObjetivoPersonaRef: solicitud.ObjetivoPersonaRef,
			PerfilRef: acto.Objetivo.PerfilRef, VinculoRef: acto.Objetivo.VinculoRef,
			EstadoPosterior: EstadoVinculoContextoActorActivo, VersionPosterior: 1,
			HuellaAntesSHA256: strings.Repeat("0", 64), HuellaDespuesSHA256: strings.Repeat("1", 64), ConfirmadoEn: instante,
		},
	}
	if err := cierre.ValidarPara(solicitud); err != nil {
		t.Fatalf("cierre ligado: %v", err)
	}
	return solicitud, cierre
}

func TestAdministracionPerfilesCierreLigaReciboAlAprobadorYAlMaterial(t *testing.T) {
	campos := map[string]func(*CierrePropuestaAdministracionPerfiles, string){
		"actor":        func(c *CierrePropuestaAdministracionPerfiles, v string) { c.Recibo.ActorPersonaRef = v },
		"perfil":       func(c *CierrePropuestaAdministracionPerfiles, v string) { c.Recibo.PerfilActivoRef = v },
		"asignacion":   func(c *CierrePropuestaAdministracionPerfiles, v string) { c.Recibo.AsignacionPerfilRef = v },
		"correlacion":  func(c *CierrePropuestaAdministracionPerfiles, v string) { c.Recibo.CorrelacionRef = v },
		"motivo":       func(c *CierrePropuestaAdministracionPerfiles, v string) { c.Recibo.Motivo.EntradaClave = v },
		"propuesta":    func(c *CierrePropuestaAdministracionPerfiles, v string) { c.Recibo.PropuestaRef = v },
		"operacion":    func(c *CierrePropuestaAdministracionPerfiles, v string) { c.Recibo.OperacionRef = v },
		"destinataria": func(c *CierrePropuestaAdministracionPerfiles, v string) { c.Recibo.ObjetivoPersonaRef = v },
	}
	otros := map[string]string{
		"actor": referenciaContextoActorPrueba("per_", "z"), "perfil": referenciaContextoActorPrueba("prf_", "z"),
		"asignacion": "asignacion:otra:v99", "correlacion": "correlacion_invalida", "motivo": "otra_entrada",
		"propuesta": "propuesta_admin:" + strings.Repeat("9", 32), "operacion": "cierre_admin:" + strings.Repeat("9", 32),
		"destinataria": referenciaContextoActorPrueba("per_", "z"),
	}
	for campo, mutar := range campos {
		for _, valor := range []string{"", otros[campo]} {
			nombre := campo + "_ajeno"
			if valor == "" {
				nombre = campo + "_ausente"
			}
			t.Run(nombre, func(t *testing.T) {
				solicitud, cierre := cierreAdministracionPerfilesLigadoPrueba(t)
				mutar(&cierre, valor)
				if cierre.ValidarPara(solicitud) == nil {
					t.Fatal("recibo no ligado al cierre aceptado")
				}
			})
		}
	}
	for _, campo := range []string{"fecha", "huella_cierre", "huella_propuesta_ausente", "huella_propuesta_ajena", "huella_antes", "huella_despues", "recibo_ausente"} {
		t.Run(campo, func(t *testing.T) {
			solicitud, cierre := cierreAdministracionPerfilesLigadoPrueba(t)
			switch campo {
			case "fecha":
				cierre.Recibo.ConfirmadoEn = cierre.ConfirmadoEn.Add(time.Microsecond)
			case "huella_cierre":
				cierre.HuellaCierreSHA256 = ""
			case "huella_propuesta_ausente":
				cierre.PropuestaHuellaSHA256 = ""
			case "huella_propuesta_ajena":
				cierre.PropuestaHuellaSHA256 = strings.Repeat("9", 64)
			case "huella_antes":
				cierre.Recibo.HuellaAntesSHA256 = ""
			case "huella_despues":
				cierre.Recibo.HuellaDespuesSHA256 = ""
			case "recibo_ausente":
				cierre.Recibo = nil
			}
			if cierre.ValidarPara(solicitud) == nil {
				t.Fatal("recibo sin evidencia ligada aceptado")
			}
		})
	}
}

func TestAdministracionPerfilesCierreReplayConservaCorrelacionOriginal(t *testing.T) {
	solicitud, cierre := cierreAdministracionPerfilesLigadoPrueba(t)
	original := cierre.Recibo.CorrelacionRef
	solicitud.CorrelacionRef = "correlacion_" + strings.Repeat("9", 32)
	solicitud.InstantaneaAutorizacion.RevisionCatalogoPoliticas++
	if cierre.ValidarPara(solicitud) != nil || cierre.Recibo.CorrelacionRef != original {
		t.Fatal("nuevo acceso no recupera cierre original")
	}
	solicitud.CorrelacionRef = ""
	if cierre.ValidarPara(solicitud) == nil {
		t.Fatal("replay sin correlación actual aceptado")
	}
}

func TestAdministracionPerfilesRechazoConservaPropuestaInmutableSinRecibo(t *testing.T) {
	solicitud, cierre := cierreAdministracionPerfilesLigadoPrueba(t)
	solicitud.Decision = DecisionRechazarPropuestaPerfil
	cierre.Decision = solicitud.Decision
	cierre.Recibo = nil
	if cierre.ValidarPara(solicitud) != nil {
		t.Fatal("rechazo durable de propuesta exacta rechazado")
	}
	cierre.PropuestaHuellaSHA256 = strings.Repeat("9", 64)
	if cierre.ValidarPara(solicitud) == nil {
		t.Fatal("rechazo de otra huella de propuesta aceptado")
	}
}
