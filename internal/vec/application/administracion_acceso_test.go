package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// Doble privado de contrato. No acredita una barrera real de agrupación ni
// auditoría o persistencia y no se conecta con la composición productiva.
type autoridadAdministracionAccesoPrueba struct {
	*autoridadPerfilesLotePrueba
	conjunto                      domain.PreimagenConjuntoCuentasAdministracionAcceso
	ahora                         time.Time
	lecturas, propuestas, cierres int
	fallo                         error
	mutarPropuesta                func(*domain.PropuestaAdministracionAcceso)
	cierre                        domain.CierreAdministracionAcceso
}

func (a *autoridadAdministracionAccesoPrueba) ResolverConjuntoCuentasPersona(context.Context, domain.SolicitudPropuestaAdministracionAcceso) (domain.PreimagenConjuntoCuentasAdministracionAcceso, error) {
	a.lecturas++
	return a.conjunto, a.fallo
}

func (a *autoridadAdministracionAccesoPrueba) ProponerCambioAccesoPersona(_ context.Context, orden domain.OrdenPropuestaAdministracionAcceso) (domain.PropuestaAdministracionAcceso, error) {
	a.propuestas++
	if a.fallo != nil {
		return domain.PropuestaAdministracionAcceso{}, a.fallo
	}
	h, err := orden.Material.HuellaSHA256()
	if err != nil {
		return domain.PropuestaAdministracionAcceso{}, err
	}
	p := domain.PropuestaAdministracionAcceso{Material: orden.Material, HuellaSHA256: h, CaducaEn: a.ahora.Add(time.Hour)}
	if a.mutarPropuesta != nil {
		a.mutarPropuesta(&p)
	}
	return p, nil
}

func (a *autoridadAdministracionAccesoPrueba) CerrarCambioAccesoPersona(context.Context, domain.SolicitudCierrePropuestaAdministracionPerfiles) (domain.CierreAdministracionAcceso, error) {
	a.cierres++
	return a.cierre, a.fallo
}

func administracionAccesoAplicacionPrueba(t *testing.T) (*ServicioAdministracionPerfiles, domain.SolicitudPropuestaAdministracionAcceso, *autoridadAdministracionAccesoPrueba, catalogoPerfilesLotePrueba) {
	t.Helper()
	servicio, lote, anterior, catalogo := lotePerfilesAplicacionPrueba(t)
	a := &autoridadAdministracionAccesoPrueba{autoridadPerfilesLotePrueba: anterior, ahora: servicio.reloj.Ahora(),
		conjunto: domain.PreimagenConjuntoCuentasAdministracionAcceso{PersonaRef: lote.Cambios[0].Objetivo.PersonaRef,
			PersonaVersion: 3, RevisionContinuidad: 2,
			Cuentas: []domain.PreimagenCuentaAdministracionAcceso{
				{CuentaRef: "cta_" + strings.Repeat("a", 24), Revision: 2, Estado: domain.EstadoCuentaAccesoActiva},
				{CuentaRef: "cta_" + strings.Repeat("b", 24), Revision: 7, Estado: domain.EstadoCuentaAccesoInactiva}}}}
	servicio.actos = a
	s := domain.SolicitudPropuestaAdministracionAcceso{OperacionRef: "propuesta_admin:" + strings.Repeat("a", 32),
		Actor: lote.Actor, Evidencia: lote.Evidencia, InstantaneaAutorizacion: lote.InstantaneaAutorizacion,
		ObjetivoPersonaRef: a.conjunto.PersonaRef, EstadoNuevo: domain.EstadoCuentaAccesoInactiva,
		Motivo: lote.Motivo, CorrelacionRef: lote.CorrelacionRef}
	return servicio, s, a, catalogo
}

func TestAdministracionAccesoResuelveTodasLasCuentasEnLaFuente(t *testing.T) {
	servicio, s, a, _ := administracionAccesoAplicacionPrueba(t)
	p, err := servicio.ProponerCambioAccesoPersona(context.Background(), s)
	if err != nil || len(p.Material.Conjunto.Cuentas) != 2 || a.lecturas != 1 || a.propuestas != 1 || a.ordinarios != 0 {
		t.Fatalf("propuesta: %v llamadas=%d/%d/%d", err, a.lecturas, a.propuestas, a.ordinarios)
	}
	p.Material.Conjunto.Cuentas[0].Revision++
	if a.conjunto.Cuentas[0].Revision != 2 {
		t.Fatal("la salida comparte estado mutable con la fuente")
	}
}

func TestAdministracionAccesoDeniegaSinEnumerarCuentas(t *testing.T) {
	for _, caso := range []string{"sistemas", "auto_bloqueo", "evidencia", "motivo", "perfil_ajeno", "autoridad_anterior", "cancelada"} {
		t.Run(caso, func(t *testing.T) {
			servicio, s, a, catalogo := administracionAccesoAplicacionPrueba(t)
			ctx := context.Background()
			switch caso {
			case "sistemas":
				r := catalogo[s.InstantaneaAutorizacion.VersionRol.Referencia()]
				r.CategoriaAdmin = "sistemas"
				catalogo[r.VersionRef] = r
			case "auto_bloqueo":
				s.ObjetivoPersonaRef = s.Actor.PersonaRef
			case "evidencia":
				s.Evidencia = domain.EvidenciaSesionAdministracionPerfiles{}
			case "motivo":
				s.Motivo.EntradaClave = "motivo_cliente"
			case "perfil_ajeno":
				s.Actor.PerfilActivoRef = "prf_" + strings.Repeat("z", 24)
			case "autoridad_anterior":
				servicio.actos = a.autoridadPerfilesLotePrueba
			case "cancelada":
				cancelada, cancelar := context.WithCancel(ctx)
				cancelar()
				ctx = cancelada
			}
			p, err := servicio.ProponerCambioAccesoPersona(ctx, s)
			if err == nil || p.HuellaSHA256 != "" || a.lecturas != 0 || a.propuestas != 0 || a.ordinarios != 0 {
				t.Fatal("solicitud no autorizada alcanzó la fuente")
			}
		})
	}
}

func TestAdministracionAccesoNoAceptaConjuntoOPropuestaAlterados(t *testing.T) {
	for _, caso := range []string{"persona_ajena", "conjunto_vacio", "fallo_fuente", "material", "caducada"} {
		t.Run(caso, func(t *testing.T) {
			servicio, s, a, _ := administracionAccesoAplicacionPrueba(t)
			switch caso {
			case "persona_ajena":
				a.conjunto.PersonaRef = "per_" + strings.Repeat("z", 24)
			case "conjunto_vacio":
				a.conjunto.Cuentas = nil
			case "fallo_fuente":
				a.fallo = ports.ErrAutoridadAdministracionPerfilesNoDisponible
			case "material":
				a.mutarPropuesta = func(p *domain.PropuestaAdministracionAcceso) {
					p.Material.Conjunto.Cuentas[0].Revision++
					p.HuellaSHA256, _ = p.Material.HuellaSHA256()
				}
			case "caducada":
				a.mutarPropuesta = func(p *domain.PropuestaAdministracionAcceso) { p.CaducaEn = a.ahora }
			}
			p, err := servicio.ProponerCambioAccesoPersona(context.Background(), s)
			if err == nil || p.HuellaSHA256 != "" || a.cierres != 0 {
				t.Fatal("fuente/propuesta no coherente aceptada")
			}
		})
	}
}

func cierreAdministracionAccesoAplicacionPrueba(t *testing.T) (*ServicioAdministracionPerfiles, domain.SolicitudCierrePropuestaAdministracionPerfiles, *autoridadAdministracionAccesoPrueba, catalogoPerfilesLotePrueba) {
	t.Helper()
	servicio, propuesta, a, catalogo := administracionAccesoAplicacionPrueba(t)
	orden, err := domain.PrepararPropuestaAdministracionAcceso(propuesta, a.conjunto)
	if err != nil {
		t.Fatal(err)
	}
	// El aprobador conserva la identidad y evidencia emitidas por la autoridad
	// de pruebas común. La propuesta fue presentada por otra persona sintética.
	orden.Material.ProponentePersonaRef = "per_" + strings.Repeat("z", 24)
	orden.Material.PerfilActivoRef = "prf_" + strings.Repeat("z", 24)
	orden.Material.AsignacionPerfilRef = "asignacion:proponente_sintetico:v1"
	h, err := orden.Material.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	s := domain.SolicitudCierrePropuestaAdministracionPerfiles{OperacionRef: "cierre_admin:" + strings.Repeat("c", 32),
		PropuestaRef: orden.Material.OperacionRef, PropuestaHuellaSHA256: h,
		ProponentePersonaRef: orden.Material.ProponentePersonaRef, ObjetivoPersonaRef: a.conjunto.PersonaRef,
		Aprobador: propuesta.Actor, Evidencia: propuesta.Evidencia, InstantaneaAutorizacion: propuesta.InstantaneaAutorizacion,
		Decision: domain.DecisionAprobarPropuestaPerfil, Motivo: propuesta.Motivo, CorrelacionRef: propuesta.CorrelacionRef}
	a.cierre = domain.CierreAdministracionAcceso{OperacionRef: s.OperacionRef, Material: orden.Material,
		PropuestaHuellaSHA256: h, Decision: s.Decision, ConfirmadoEn: a.ahora, AuditoriaAccesoRef: "auditoria:acceso:actual",
		Recibo: &domain.ReciboAdministracionAcceso{ReciboRef: "recibo_admin:" + strings.Repeat("d", 32),
			ActorPersonaRef: s.Aprobador.PersonaRef, PerfilActivoRef: s.Aprobador.PerfilActivoRef,
			AsignacionPerfilRef: s.InstantaneaAutorizacion.AsignacionPerfil.Referencia(), CorrelacionRef: s.CorrelacionRef,
			Motivo: s.Motivo, AuditoriaRef: "auditoria:acceso:efecto", Cuentas: []domain.ResultadoCuentaAdministracionAcceso{
				{CuentaRef: a.conjunto.Cuentas[0].CuentaRef, RevisionPosterior: 3, EstadoPosterior: domain.EstadoCuentaAccesoInactiva, OperacionISRef: "opr_" + strings.Repeat("a", 24)},
				{CuentaRef: a.conjunto.Cuentas[1].CuentaRef, RevisionPosterior: 7, EstadoPosterior: domain.EstadoCuentaAccesoInactiva}}}}
	return servicio, s, a, catalogo
}

func TestAdministracionAccesoCierreConservaConjuntoYReciboEnReplay(t *testing.T) {
	servicio, s, a, _ := cierreAdministracionAccesoAplicacionPrueba(t)
	primero, err := servicio.CerrarCambioAccesoPersona(context.Background(), s)
	if err != nil || len(primero.Recibo.Cuentas) != 2 || a.cierres != 1 {
		t.Fatalf("cierre: %v", err)
	}
	s.CorrelacionRef = "correlacion_" + strings.Repeat("f", 32)
	s.InstantaneaAutorizacion.RevisionCatalogoPoliticas++
	replay, err := servicio.CerrarCambioAccesoPersona(context.Background(), s)
	if err != nil || replay.Recibo.CorrelacionRef != primero.Recibo.CorrelacionRef || replay.Recibo.ReciboRef != primero.Recibo.ReciboRef || a.cierres != 2 {
		t.Fatal("replay reescribe recibo o depende de correlación antigua")
	}
	replay.Recibo.Cuentas[0].RevisionPosterior++
	if a.cierre.Recibo.Cuentas[0].RevisionPosterior != 3 {
		t.Fatal("salida comparte recibo mutable con autoridad")
	}
}

func TestAdministracionAccesoCierreRechazaEfectoParcialYMaterialAjeno(t *testing.T) {
	for _, caso := range []string{"parcial", "revision", "cuenta_ajena", "asignacion_ajena", "estado", "operacionIS", "material", "misma_persona", "destinataria", "evidencia", "fallo_durable", "rechazo_con_recibo"} {
		t.Run(caso, func(t *testing.T) {
			servicio, s, a, _ := cierreAdministracionAccesoAplicacionPrueba(t)
			switch caso {
			case "parcial":
				a.cierre.Recibo.Cuentas = a.cierre.Recibo.Cuentas[:1]
			case "revision":
				a.cierre.Recibo.Cuentas[1].RevisionPosterior++
			case "cuenta_ajena":
				a.cierre.Recibo.Cuentas[0].CuentaRef = "cta_" + strings.Repeat("z", 24)
			case "asignacion_ajena":
				a.cierre.Recibo.AsignacionPerfilRef = "asignacion:ajena:v1"
			case "estado":
				a.cierre.Recibo.Cuentas[0].EstadoPosterior = domain.EstadoCuentaAccesoActiva
			case "operacionIS":
				a.cierre.Recibo.Cuentas[0].OperacionISRef = ""
			case "material":
				a.cierre.Material.Conjunto.Cuentas[0].Revision++
			case "misma_persona":
				s.ProponentePersonaRef = s.Aprobador.PersonaRef
			case "destinataria":
				s.ObjetivoPersonaRef = s.Aprobador.PersonaRef
			case "evidencia":
				s.Evidencia = domain.EvidenciaSesionAdministracionPerfiles{}
			case "fallo_durable":
				a.fallo = errors.New("CAS abortado")
			case "rechazo_con_recibo":
				s.Decision = domain.DecisionRechazarPropuestaPerfil
				a.cierre.Decision = s.Decision
			}
			cierre, err := servicio.CerrarCambioAccesoPersona(context.Background(), s)
			if err == nil || cierre.Recibo != nil {
				t.Fatal("cierre incorrecto produce recibo")
			}
		})
	}
}

func TestAdministracionAccesoReactivacionLigaTodasLasRevisiones(t *testing.T) {
	servicio, s, a, _ := cierreAdministracionAccesoAplicacionPrueba(t)
	a.cierre.Material.EstadoNuevo = domain.EstadoCuentaAccesoActiva
	h, err := a.cierre.Material.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	s.PropuestaHuellaSHA256, a.cierre.PropuestaHuellaSHA256 = h, h
	a.cierre.Recibo.Cuentas[0].EstadoPosterior = domain.EstadoCuentaAccesoActiva
	a.cierre.Recibo.Cuentas[0].RevisionPosterior, a.cierre.Recibo.Cuentas[0].OperacionISRef = 2, ""
	a.cierre.Recibo.Cuentas[1].EstadoPosterior = domain.EstadoCuentaAccesoActiva
	a.cierre.Recibo.Cuentas[1].RevisionPosterior, a.cierre.Recibo.Cuentas[1].OperacionISRef = 8, "opr_"+strings.Repeat("b", 24)
	cierre, err := servicio.CerrarCambioAccesoPersona(context.Background(), s)
	if err != nil || cierre.Recibo.Cuentas[1].RevisionPosterior != 8 {
		t.Fatalf("reactivación sin revisión nueva: %v", err)
	}
	// Una segunda cuenta cambiada no puede reutilizar el acto IS2 de la primera.
	a.cierre.Material.EstadoNuevo = domain.EstadoCuentaAccesoInactiva
	a.cierre.Material.Conjunto.Cuentas[1].Estado = domain.EstadoCuentaAccesoActiva
	s.PropuestaHuellaSHA256, _ = a.cierre.Material.HuellaSHA256()
	a.cierre.PropuestaHuellaSHA256 = s.PropuestaHuellaSHA256
	for i := range a.cierre.Recibo.Cuentas {
		a.cierre.Recibo.Cuentas[i].EstadoPosterior = domain.EstadoCuentaAccesoInactiva
		a.cierre.Recibo.Cuentas[i].RevisionPosterior = a.cierre.Material.Conjunto.Cuentas[i].Revision + 1
		a.cierre.Recibo.Cuentas[i].OperacionISRef = "opr_" + strings.Repeat("b", 24)
	}
	if c, err := servicio.CerrarCambioAccesoPersona(context.Background(), s); err == nil || c.Recibo != nil {
		t.Fatal("dos cambios comparten acto IS2")
	}
}

func TestAdministracionAccesoRechazoNoProduceReciboDeCambio(t *testing.T) {
	servicio, s, a, _ := cierreAdministracionAccesoAplicacionPrueba(t)
	s.Decision, a.cierre.Decision = domain.DecisionRechazarPropuestaPerfil, domain.DecisionRechazarPropuestaPerfil
	a.cierre.Recibo = nil
	cierre, err := servicio.CerrarCambioAccesoPersona(context.Background(), s)
	if err != nil || cierre.Recibo != nil || a.cierres != 1 {
		t.Fatal("rechazo confundido con cambio de acceso")
	}
}
