package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

type relojUsoAutorizacionPrueba struct{ ahora time.Time }

func (r relojUsoAutorizacionPrueba) Ahora() time.Time { return r.ahora }

type autorizadorUsoAutorizacionPrueba struct {
	ahora        time.Time
	campos       []string
	obligaciones []string
	alterar      func(*domain.DecisionAutorizacion)
	invocado     bool
}

func (a *autorizadorUsoAutorizacionPrueba) Exigir(
	_ context.Context,
	solicitud domain.SolicitudAutorizacion,
) (domain.DecisionAutorizacion, error) {
	a.invocado = true
	decision := completarDecisionAutorizacionPrueba(solicitud, domain.DecisionAutorizacion{
		DecisionRef:            "decision:uso:uno",
		Concedida:              true,
		Codigo:                 "concedida",
		PrincipalID:            solicitud.Principal.ID,
		PerfilActivoRef:        solicitud.PerfilActivoRef,
		Accion:                 solicitud.Accion,
		RecursoRef:             solicitud.Recurso.Referencia,
		Finalidad:              solicitud.Finalidad,
		CorrelacionRef:         solicitud.CorrelacionRef,
		AsignacionRef:          "asignacion:uso:v1",
		AsignacionHuellaSHA256: strings.Repeat("a", 64),
		VersionRolRef:          "rol:uso:v1",
		VersionRolHuellaSHA256: strings.Repeat("b", 64),
		GarantiaMinima:         domain.AuthAssuranceHigh,
		CamposPermitidos:       append([]string(nil), a.campos...),
		Obligaciones:           append([]string(nil), a.obligaciones...),
		EmitidaEn:              a.ahora.Add(-time.Second),
		ValidaHasta:            a.ahora.Add(time.Minute),
	})
	if a.alterar != nil {
		a.alterar(&decision)
	}
	return decision, nil
}

func TestExigirDecisionAutorizacionDeniegaUsoDeCamposNoDeclarado(t *testing.T) {
	ahora := time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)
	autorizador := &autorizadorUsoAutorizacionPrueba{ahora: ahora}
	_, err := exigirDecisionAutorizacion(
		context.Background(), autorizador, relojUsoAutorizacionPrueba{ahora: ahora},
		principalUsoAutorizacionPrueba(), perfilAutorizacionPrueba("uso"), "bolsa.expediente.leer",
		recursoUsoAutorizacionPrueba(), "tramitar_bolsa", "correlacion:uso", "Consulta",
		usoCamposDecisionNoDeclarado,
	)
	if !errors.Is(err, domain.ErrAutorizacionDenegada) {
		t.Fatalf("uso de campos no declarado: error = %v", err)
	}
	if autorizador.invocado {
		t.Fatal("una configuracion local invalida no debe consultar al autorizador")
	}
}

func TestExigirDecisionAutorizacionDeniegaContextoNuloOCancelado(t *testing.T) {
	ahora := time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)
	for _, caso := range []struct {
		nombre string
		ctx    context.Context
	}{
		{nombre: "nulo", ctx: nil},
		{nombre: "cancelado", ctx: contextoCanceladoAutorizacionPrueba()},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			autorizador := &autorizadorUsoAutorizacionPrueba{ahora: ahora}
			_, err := exigirDecisionAutorizacion(
				caso.ctx, autorizador, relojUsoAutorizacionPrueba{ahora: ahora},
				principalUsoAutorizacionPrueba(), perfilAutorizacionPrueba("uso"), "bolsa.expediente.leer",
				recursoUsoAutorizacionPrueba(), "tramitar_bolsa", "correlacion:uso", "Consulta",
				usoCamposDecisionConsumidos,
			)
			if !errors.Is(err, domain.ErrAutorizacionDenegada) || autorizador.invocado {
				t.Fatalf("contexto %s no fallo cerrado: invocado=%t err=%v", caso.nombre, autorizador.invocado, err)
			}
		})
	}
}

func TestExigirDecisionAutorizacionNoNormalizaDatosParaConceder(t *testing.T) {
	ahora := time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)
	perfilValido := perfilAutorizacionPrueba("uso")
	casos := []struct {
		nombre         string
		perfil         string
		accion         string
		finalidad      string
		correlacion    string
		motivo         string
		alterarRecurso func(*domain.RecursoAutorizable)
	}{
		{nombre: "perfil", perfil: " " + perfilValido, accion: "bolsa.expediente.leer", finalidad: "tramitar_bolsa", correlacion: "correlacion:uso", motivo: "Consulta"},
		{nombre: "accion", perfil: perfilValido, accion: "bolsa.expediente.leer ", finalidad: "tramitar_bolsa", correlacion: "correlacion:uso", motivo: "Consulta"},
		{nombre: "finalidad", perfil: perfilValido, accion: "bolsa.expediente.leer", finalidad: " tramitar_bolsa", correlacion: "correlacion:uso", motivo: "Consulta"},
		{nombre: "correlacion", perfil: perfilValido, accion: "bolsa.expediente.leer", finalidad: "tramitar_bolsa", correlacion: "correlacion:uso ", motivo: "Consulta"},
		{nombre: "motivo", perfil: perfilValido, accion: "bolsa.expediente.leer", finalidad: "tramitar_bolsa", correlacion: "correlacion:uso", motivo: " Consulta"},
		{
			nombre: "recurso", perfil: perfilValido, accion: "bolsa.expediente.leer",
			finalidad: "tramitar_bolsa", correlacion: "correlacion:uso", motivo: "Consulta",
			alterarRecurso: func(recurso *domain.RecursoAutorizable) { recurso.Referencia += " " },
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			autorizador := &autorizadorUsoAutorizacionPrueba{ahora: ahora}
			recurso := recursoUsoAutorizacionPrueba()
			if caso.alterarRecurso != nil {
				caso.alterarRecurso(&recurso)
			}
			_, err := exigirDecisionAutorizacion(
				context.Background(), autorizador, relojUsoAutorizacionPrueba{ahora: ahora},
				principalUsoAutorizacionPrueba(), caso.perfil, caso.accion, recurso,
				caso.finalidad, caso.correlacion, caso.motivo, usoCamposDecisionNoAplicables,
			)
			if !errors.Is(err, domain.ErrAutorizacionDenegada) {
				t.Fatalf("dato no canonico aceptado: %v", err)
			}
			if autorizador.invocado {
				t.Fatal("una solicitud no canonica no debe llegar al autorizador")
			}
		})
	}
}

func TestExigirDecisionAutorizacionDeniegaRestriccionesIgnoradas(t *testing.T) {
	ahora := time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)
	casos := []struct {
		nombre       string
		campos       []string
		obligaciones []string
		usoCampos    usoCamposDecisionAutorizacion
	}{
		{
			nombre:    "campos en operacion atomica",
			campos:    []string{"estado"},
			usoCampos: usoCamposDecisionNoAplicables,
		},
		{
			nombre:       "obligacion no implementada",
			obligaciones: []string{"doble_control"},
			usoCampos:    usoCamposDecisionConsumidos,
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			autorizador := &autorizadorUsoAutorizacionPrueba{
				ahora: ahora, campos: caso.campos, obligaciones: caso.obligaciones,
			}
			_, err := exigirDecisionAutorizacion(
				context.Background(), autorizador, relojUsoAutorizacionPrueba{ahora: ahora},
				principalUsoAutorizacionPrueba(), perfilAutorizacionPrueba("uso"), "bolsa.expediente.leer",
				recursoUsoAutorizacionPrueba(), "tramitar_bolsa", "correlacion:uso", "Consulta",
				caso.usoCampos,
			)
			if !errors.Is(err, domain.ErrAutorizacionDenegada) {
				t.Fatalf("restriccion ignorada: error = %v", err)
			}
		})
	}
}

func TestExigirDecisionAutorizacionAdmiteCamposSoloSiElCasoLosConsume(t *testing.T) {
	ahora := time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)
	autorizador := &autorizadorUsoAutorizacionPrueba{ahora: ahora, campos: []string{"estado"}}
	decision, err := exigirDecisionAutorizacion(
		context.Background(), autorizador, relojUsoAutorizacionPrueba{ahora: ahora},
		principalUsoAutorizacionPrueba(), perfilAutorizacionPrueba("uso"), "bolsa.expediente.leer",
		recursoUsoAutorizacionPrueba(), "tramitar_bolsa", "correlacion:uso", "Consulta",
		usoCamposDecisionConsumidos,
	)
	if err != nil {
		t.Fatalf("campos consumidos: %v", err)
	}
	if len(decision.CamposPermitidos) != 1 || decision.CamposPermitidos[0] != "estado" {
		t.Fatalf("decision inesperada: %+v", decision.CamposPermitidos)
	}
}

func TestExigirDecisionAutorizacionLigaModuloTipoYContextoDelRecurso(t *testing.T) {
	ahora := time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)
	casos := []struct {
		nombre  string
		alterar func(*domain.DecisionAutorizacion)
	}{
		{"modulo", func(d *domain.DecisionAutorizacion) { d.ModuloID = "cronos" }},
		{"tipo", func(d *domain.DecisionAutorizacion) { d.TipoRecurso = "fichaje" }},
		{"contexto", func(d *domain.DecisionAutorizacion) { d.ContextoRecursoHuellaSHA256 = strings.Repeat("f", 64) }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			autorizador := &autorizadorUsoAutorizacionPrueba{ahora: ahora, alterar: caso.alterar}
			_, err := exigirDecisionAutorizacion(
				context.Background(), autorizador, relojUsoAutorizacionPrueba{ahora: ahora},
				principalUsoAutorizacionPrueba(), perfilAutorizacionPrueba("uso"), "bolsa.expediente.leer",
				recursoUsoAutorizacionPrueba(), "tramitar_bolsa", "correlacion:uso", "Consulta",
				usoCamposDecisionConsumidos,
			)
			if !errors.Is(err, domain.ErrAutorizacionDenegada) || !errors.Is(err, domain.ErrDecisionAutorizacionInvalida) {
				t.Fatalf("decision no vinculada aceptada: %v", err)
			}
		})
	}
}

func principalUsoAutorizacionPrueba() domain.Principal {
	return domain.Principal{
		ID:            personaAutorizacionPrueba("uso"),
		AuthMethod:    domain.AuthMethodCertificate,
		AuthAssurance: domain.AuthAssuranceHigh,
	}
}

func recursoUsoAutorizacionPrueba() domain.RecursoAutorizable {
	return domain.RecursoAutorizable{
		Referencia: "expediente:uso",
		ModuloID:   "bolsa",
		Tipo:       "expediente",
	}
}
