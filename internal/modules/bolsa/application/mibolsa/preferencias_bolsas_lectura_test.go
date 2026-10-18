package mibolsa

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	bolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type lectorPreferenciasAplicacionPrueba struct {
	llamadas int
	captura  bolsa.CapturaLecturaPreferencias
	err      error
}

func (l *lectorPreferenciasAplicacionPrueba) LeerPreferenciasBolsas(_ context.Context, c bolsa.CapturaLecturaPreferencias) (bolsa.EstadoPreferenciasBolsas, error) {
	l.llamadas++
	l.captura = c
	if l.err != nil {
		return bolsa.EstadoPreferenciasBolsas{}, l.err
	}
	return bolsa.EstadoPreferenciasBolsas{BolsasActuales: []string{"bolsa:propia"}}, nil
}

type auditorPreferenciasAplicacionPrueba struct {
	llamadas     int
	captura      bolsa.CapturaLecturaPreferencias
	err          error
	contextoVivo bool
}

func (a *auditorPreferenciasAplicacionPrueba) RegistrarIntentoLecturaPreferenciasBolsas(ctx context.Context, c bolsa.CapturaLecturaPreferencias) error {
	a.llamadas++
	a.captura = c
	a.contextoVivo = ctx.Err() == nil
	return a.err
}

func TestLecturaPreferenciasPropiaRevocadaYAjena(t *testing.T) {
	e := nuevoEntorno(t)
	i := &e.fuente.instantanea
	i.VersionRol.RolID = "candidato_bolsa_portal_historial_propio_desarrollo"
	i.AsignacionPerfil.VersionRolRef = i.VersionRol.Referencia()
	i.ControlVigenciaVersionRol.VersionRolRef = i.VersionRol.Referencia()
	c := &i.VersionRol.Concesiones[0]
	c.Accion = bolsa.AccionConsultarPreferenciasBolsas
	c.ModuloID = bolsa.ModuloMiBolsa
	c.TipoRecurso = bolsa.TipoRecursoMiBolsa
	c.Finalidades = []string{bolsa.FinalidadPreferenciasBolsas}
	c.CamposPermitidos = bolsa.CamposLecturaPreferenciasBolsas()
	c.GarantiaMinima = dominiovec.AuthAssuranceHigh
	lector := new(lectorPreferenciasAplicacionPrueba)
	auditor := new(auditorPreferenciasAplicacionPrueba)
	servicio, err := NuevaLecturaPreferenciasBolsas(lector, auditor, e.fuente, &relojAutorizacionServicioPrueba{ahora: e.ahora})
	if err != nil {
		t.Fatal(err)
	}
	estado, err := servicio.ConsultarPreferencias(t.Context(), e.orden)
	if err != nil || lector.llamadas != 1 || len(estado.BolsasActuales) != 1 ||
		lector.captura.CandidatoRef() != referenciaServicioContextoActorPrueba("can_", "c") ||
		len(lector.captura.ContextoCanonico()) == 0 || len(lector.captura.VinculoCanonico()) == 0 {
		t.Fatalf("lectura propia: %v", err)
	}
	var orden map[string]any
	if json.Unmarshal(lector.captura.OrdenJSON(), &orden) != nil || len(orden) != 35 ||
		orden["recurso_ref"] != "mi-bolsa:"+lector.captura.CandidatoRef() ||
		orden["accion"] != bolsa.AccionConsultarPreferenciasBolsas {
		t.Fatalf("captura incorrecta: %v", orden)
	}
	c.Accion = "bolsa.otra_lectura"
	_, err = servicio.ConsultarPreferencias(t.Context(), e.orden)
	if !errors.Is(err, dominiovec.ErrAutorizacionDenegada) || lector.llamadas != 1 || auditor.llamadas != 1 {
		t.Fatalf("revocación no cerró GET: %v", err)
	}
	var intento map[string]string
	if json.Unmarshal(auditor.captura.OrdenJSON(), &intento) != nil || intento["resultado"] != "denegado" ||
		len(intento) != 16 || intento["recurso_ref"] == "mi-bolsa:"+auditor.captura.CandidatoRef() ||
		len(intento["recurso_ref"]) != len("mi-bolsa:")+64 || !auditor.contextoVivo {
		t.Fatalf("intento nominal inválido: %v", intento)
	}
	auditor.err = bolsa.ErrPreferenciasNoDisponibles
	_, err = servicio.ConsultarPreferencias(t.Context(), e.orden)
	if !errors.Is(err, bolsa.ErrPreferenciasNoDisponibles) || errors.Is(err, dominiovec.ErrAutorizacionDenegada) || lector.llamadas != 1 {
		t.Fatalf("auditor caído entregó denegación sin asiento: %v", err)
	}
	auditor.err = nil
	c.Accion = bolsa.AccionConsultarPreferenciasBolsas
	e.fuente.err = bolsa.ErrPreferenciasNoDisponibles
	_, err = servicio.ConsultarPreferencias(t.Context(), e.orden)
	if !errors.Is(err, bolsa.ErrPreferenciasNoDisponibles) || errors.Is(err, dominiovec.ErrAutorizacionDenegada) || lector.llamadas != 1 {
		t.Fatalf("fuente caída confundida con denegación: %v", err)
	}
	if json.Unmarshal(auditor.captura.OrdenJSON(), &intento) != nil || intento["resultado"] != "error" {
		t.Fatalf("error técnico sin asiento: %v", intento)
	}
	e.fuente.err = nil
	lector.err = bolsa.ErrPreferenciasNoDisponibles
	_, err = servicio.ConsultarPreferencias(t.Context(), e.orden)
	if !errors.Is(err, bolsa.ErrPreferenciasNoDisponibles) || lector.llamadas != 2 || auditor.llamadas != 4 {
		t.Fatalf("lectura técnica sin asiento: %v, consultas=%d auditorías=%d", err, lector.llamadas, auditor.llamadas)
	}
	lector.err = nil
	ajena := e.orden
	ajena.Vinculo = dominiovec.VinculoAutenticacionActorV2{}
	_, err = servicio.ConsultarPreferencias(t.Context(), ajena)
	if !errors.Is(err, dominiovec.ErrAutorizacionDenegada) || lector.llamadas != 2 || auditor.llamadas != 4 {
		t.Fatalf("actor ajeno alcanzó lectura: %v", err)
	}
}

func TestLecturaPreferenciasInternaConSesionYCanalPropios(t *testing.T) {
	e := nuevoEntorno(t)
	autenticacion := e.autenticacion
	autenticacion.Superficie = dominiovec.SuperficieAutenticacionInternaCorporativaV1
	vinculo, err := dominiovec.CrearVinculoAutenticacionActorV2(t.Context(),
		&revalidadorVinculoAplicacionAdversarial{resultado: autenticacion},
		dominiovec.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: autenticacion.AutenticacionRef, SesionRef: autenticacion.SesionRef},
		resolutorContextoAutorizacionV3Prueba{resultado: e.resultado}, solicitudServicioContextoActorPrueba(),
		&relojAutorizacionServicioPrueba{ahora: e.ahora})
	if err != nil {
		t.Fatal(err)
	}
	e.orden.Vinculo = vinculo
	i := &e.fuente.instantanea
	i.VersionRol.RolID = "candidato_bolsa_portal_historial_propio_desarrollo"
	i.AsignacionPerfil.VersionRolRef = i.VersionRol.Referencia()
	i.ControlVigenciaVersionRol.VersionRolRef = i.VersionRol.Referencia()
	c := &i.VersionRol.Concesiones[0]
	c.Accion = bolsa.AccionConsultarPreferenciasBolsas
	c.ModuloID = bolsa.ModuloMiBolsa
	c.TipoRecurso = bolsa.TipoRecursoMiBolsa
	c.Finalidades = []string{bolsa.FinalidadPreferenciasBolsas}
	c.CamposPermitidos = bolsa.CamposLecturaPreferenciasBolsas()
	c.GarantiaMinima = dominiovec.AuthAssuranceHigh
	lector := new(lectorPreferenciasAplicacionPrueba)
	auditor := new(auditorPreferenciasAplicacionPrueba)
	servicio, err := NuevaLecturaPreferenciasBolsasInterna(lector, auditor, e.fuente, &relojAutorizacionServicioPrueba{ahora: e.ahora})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := servicio.ConsultarPreferencias(t.Context(), e.orden); err != nil || lector.llamadas != 1 {
		t.Fatalf("lectura interna propia: %v", err)
	}
	var captura map[string]any
	if err := json.Unmarshal(lector.captura.OrdenJSON(), &captura); err != nil || captura["canal"] != string(dominiovec.SuperficieAutenticacionInternaCorporativaV1) {
		t.Fatalf("canal interno no ligado al vinculo: %v", captura)
	}
	externa, err := NuevaLecturaPreferenciasBolsas(new(lectorPreferenciasAplicacionPrueba), auditor, e.fuente, &relojAutorizacionServicioPrueba{ahora: e.ahora})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := externa.ConsultarPreferencias(t.Context(), e.orden); !errors.Is(err, dominiovec.ErrAutorizacionDenegada) {
		t.Fatalf("sesion interna entro en la autoridad externa: %v", err)
	}
}

func TestFalloTrasCancelacionConservaPlazoDeAuditoria(t *testing.T) {
	e := nuevoEntorno(t)
	ctx, cancelar := context.WithCancel(t.Context())
	e.fuente.despues = cancelar
	auditor := new(auditorPreferenciasAplicacionPrueba)
	servicio, err := NuevaLecturaPreferenciasBolsas(new(lectorPreferenciasAplicacionPrueba), auditor, e.fuente, &relojAutorizacionServicioPrueba{ahora: e.ahora})
	if err != nil {
		t.Fatal(err)
	}
	_, err = servicio.ConsultarPreferencias(ctx, e.orden)
	if !errors.Is(err, bolsa.ErrPreferenciasNoDisponibles) || auditor.llamadas != 1 || !auditor.contextoVivo {
		t.Fatalf("fallo tras cancelar perdió auditoría: %v, llamadas=%d, contexto=%t", err, auditor.llamadas, auditor.contextoVivo)
	}
}
