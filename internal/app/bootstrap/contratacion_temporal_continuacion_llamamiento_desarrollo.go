package bootstrap

import (
	"context"
	"crypto/ed25519"
	"errors"
	"maps"
	"time"

	appbolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type claveContinuacionLlamamientoDesarrollo struct{}
type claveMaterialContinuacionDesarrollo struct{}

// Solo la composición instala estos antecedentes después de recuperarlos con
// permisos nominales nuevos. No son entradas del navegador ni otra cola.
type continuacionLigadaDesarrollo struct {
	solicitud    ports.SolicitudContinuarLlamamiento
	antecedente  ports.AntecedenteContinuacionLlamamiento
	justificante ports.JustificanteRespuestaRecibida
}

type continuadorBolsaDesarrollo interface {
	AbrirSiguienteRRHH(context.Context, ports.SolicitudContinuarLlamamiento) (ports.ReciboBolsaContinuacion, error)
}

var _ httpinterno.EjecutorContinuacionLlamamiento = (*ejecutorComunicacionLlamamientoDesarrollo)(nil)

func (e *ejecutorComunicacionLlamamientoDesarrollo) Continuar(ctx context.Context, s ports.SolicitudContinuarLlamamiento) (ports.ResultadoContinuacionLlamamiento, error) {
	vacio := ports.ResultadoContinuacionLlamamiento{}
	if contextoInterfazNulo(ctx) || e == nil || e.soporte == nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(e.lector) || dependenciaEsNulaContratacionTemporalDesarrollo(e.lectorJustificante) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(e.continuaciones) || dependenciaEsNulaContratacionTemporalDesarrollo(e.continuador) {
		return vacio, ports.ErrOperacionContinuacionNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	if s.Validar() != nil {
		return vacio, ports.ErrOperacionContinuacionInvalida
	}
	c, valida := e.soporte.capacidadValida(ctx)
	if !valida || c.ruta != httpinterno.RutaContinuacionLlamamiento || s.OrganizacionRef != organizacionAltaContratacionTemporalDesarrollo {
		return vacio, ports.ErrOperacionContinuacionDenegada
	}
	expediente, err := e.lector.LeerExpedienteParaSeleccion(ctx, s.OrganizacionRef, s.ExpedienteRef, 6)
	if ctx.Err() != nil {
		return vacio, ctx.Err()
	}
	if err != nil || expediente.Fiscalizado.Validar() != nil || expediente.VersionActual != 6 ||
		!expedienteComunicacionLlamamientoDesarrolloValido(expediente, ports.SolicitudRegistrarComunicacionLlamamiento{OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef}) {
		return vacio, ports.ErrOperacionContinuacionNoDisponible
	}
	ctx = context.WithValue(ctx, clavePreparacionLlamamientoDesarrollo{}, preparacionLlamamientoDesarrollo{expediente: expediente})
	ctx = context.WithValue(ctx, claveMaterialContinuacionDesarrollo{}, ports.MaterialContinuacionLlamamiento{Etapa: "consulta", Solicitud: s})
	a, err := e.continuaciones.LeerAntecedente(ctx, s)
	if ctx.Err() != nil {
		return vacio, ctx.Err()
	}
	if err != nil {
		return vacio, err
	}
	if a.ValidarPara(s) != nil || a.Resolucion.Politica != politicaManualDesarrollo() ||
		a.Resolucion.Solicitud.CriterioValidacionRef != criterioRevisionManualDesarrollo {
		return vacio, ports.ErrOperacionContinuacionNoDisponible
	}
	l := continuacionLigadaDesarrollo{solicitud: s, antecedente: a}
	ctx = context.WithValue(ctx, claveContinuacionLlamamientoDesarrollo{}, l)
	ctx = context.WithValue(ctx, claveConsultaJustificanteRespuestaDesarrollo{}, a.Resolucion.Solicitud)
	j, err := e.lectorJustificante.ConsultarJustificanteRespuestaRecibida(ctx, a.Resolucion.Solicitud)
	if ctx.Err() != nil {
		return vacio, ctx.Err()
	}
	if errors.Is(err, ports.ErrOperacionRespuestaRecibidaDenegada) {
		return vacio, ports.ErrOperacionContinuacionDenegada
	}
	if err != nil || j.ValidarPara(a.Resolucion.Solicitud) != nil || a.Resolucion.ResueltaEn.Before(j.Respuesta.RegistradaEn) {
		return vacio, ports.ErrOperacionContinuacionNoDisponible
	}
	preparada, err := prepararReferenciasLlamamientoDesarrollo(expediente, a.ComandoSiguiente.SeleccionClave)
	if err != nil || preparada.operacionPropuesta != j.Seleccion.OperacionRef {
		return vacio, ports.ErrOperacionContinuacionNoDisponible
	}
	l.justificante = j
	ctx = context.WithValue(ctx, claveContinuacionLlamamientoDesarrollo{}, l)
	b, err := e.continuador.AbrirSiguienteRRHH(ctx, s)
	if ctx.Err() != nil {
		return vacio, ctx.Err()
	}
	if err != nil {
		return vacio, err
	}
	if b.ValidarPara(s) != nil || b.OperacionRef != operacionSiguienteDesarrollo(s) ||
		b.TerminalOperacionRef != terminalContinuacionDesarrollo(l) || b.LlamamientoRef == a.Resolucion.Solicitud.LlamamientoRef ||
		b.ConfirmadaEn.Before(a.Resolucion.ResueltaEn) {
		return vacio, ports.ErrOperacionContinuacionNoDisponible
	}
	// Bolsa ya hizo COMMIT. Un fallo posterior conserva ese efecto; no se
	// compensa ni se devuelve éxito parcial. El reintento usa la misma intención.
	ctx = context.WithValue(ctx, claveMaterialContinuacionDesarrollo{}, ports.MaterialContinuacionLlamamiento{Etapa: "confirmacion", Solicitud: s, ReciboBolsa: &b})
	r, err := e.continuaciones.Confirmar(ctx, s, b)
	if ctx.Err() != nil {
		return vacio, ctx.Err()
	}
	if err != nil {
		return vacio, err
	}
	if r.ValidarPara(s) != nil || r.ReciboBolsa != b || r.LlamamientoAnteriorRef != a.Resolucion.Solicitud.LlamamientoRef {
		return vacio, ports.ErrOperacionContinuacionNoDisponible
	}
	return r, nil
}

func operacionSiguienteDesarrollo(s ports.SolicitudContinuarLlamamiento) string {
	return referenciaPuenteLlamamientoDesarrollo("operacion-siguiente-rrhh", s.OrganizacionRef, s.ExpedienteRef, s.IntencionRef)
}

// Cada módulo conserva su referencia nativa. Un UUID válido de CT puede
// coincidir accidentalmente con la detección de documentos personales de
// Bolsa. Se traduce aquí a su alfabeto opaco, sin relajar esos validadores ni
// cambiar la intención CT original. No es una intención nueva ni otro efecto.
func intencionSiguienteBolsaDesarrollo(s ports.SolicitudContinuarLlamamiento) string {
	return referenciaPuenteLlamamientoDesarrollo("intencion-siguiente-bolsa", s.OrganizacionRef, s.ExpedienteRef, s.ResolucionRef, s.IntencionRef)
}

func terminalContinuacionDesarrollo(l continuacionLigadaDesarrollo) string {
	return operacionAceptacionManualDesarrollo(aceptacionRevisadaDesarrollo{
		solicitud: l.antecedente.Resolucion.Solicitud, justificante: l.justificante, local: l.antecedente.Resolucion})
}

func antecedenteContinuacionDesarrolloValido(ctx context.Context, s ports.SolicitudResolverLlamamiento) bool {
	l, ok := ctx.Value(claveContinuacionLlamamientoDesarrollo{}).(continuacionLigadaDesarrollo)
	return ok && l.antecedente.ValidarPara(l.solicitud) == nil && l.antecedente.Resolucion.Solicitud == s &&
		l.antecedente.Resolucion.Politica == politicaManualDesarrollo() && s.CriterioValidacionRef == criterioRevisionManualDesarrollo
}

func (p *puenteBolsaLlamamientoDesarrollo) AbrirSiguienteRRHH(ctx context.Context, s ports.SolicitudContinuarLlamamiento) (ports.ReciboBolsaContinuacion, error) {
	vacio := ports.ReciboBolsaContinuacion{}
	if contextoInterfazNulo(ctx) || p == nil || p.alta == nil || p.alta.soporte == nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(p.autorizadorSiguiente) || dependenciaEsNulaContratacionTemporalDesarrollo(p.repositorio) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(p.reloj) || len(p.privadaFuente) != ed25519.PrivateKeySize {
		return vacio, ports.ErrOperacionContinuacionNoDisponible
	}
	if ctx.Err() != nil {
		return vacio, ctx.Err()
	}
	c, valida := p.alta.soporte.capacidadValida(ctx)
	l, ok := ctx.Value(claveContinuacionLlamamientoDesarrollo{}).(continuacionLigadaDesarrollo)
	if !valida || c.ruta != httpinterno.RutaContinuacionLlamamiento || !ok || l.solicitud != s ||
		!antecedenteContinuacionDesarrolloValido(ctx, l.antecedente.Resolucion.Solicitud) || l.justificante.ValidarPara(l.antecedente.Resolucion.Solicitud) != nil {
		return vacio, ports.ErrOperacionContinuacionDenegada
	}
	a := l.antecedente.Resolucion
	apertura, existe, err := p.repositorio.BuscarOperacion(ctx, l.justificante.Seleccion.OperacionRef)
	if err != nil || !existe {
		return vacio, ports.ErrOperacionContinuacionNoDisponible
	}
	fuente, err := p.fuenteResolucionLigada(a.Solicitud, l.justificante.Seleccion, apertura)
	if err != nil {
		return vacio, ports.ErrOperacionContinuacionNoDisponible
	}
	terminalRef := terminalContinuacionDesarrollo(l)
	terminal, existe, err := p.repositorio.BuscarOperacion(ctx, terminalRef)
	if err != nil || !existe {
		return vacio, ports.ErrOperacionContinuacionNoDisponible
	}
	canonTerminal, err := terminal.Canonico()
	esperado := puertosbolsa.ResolucionLlamamientoDesarrollo{AperturaOperacionRef: l.justificante.Seleccion.OperacionRef,
		JustificanteRef: a.Solicitud.PruebaRespuestaRef, EvaluacionPlazoRef: a.EvaluacionPlazoRef,
		PoliticaRef: a.Politica.Referencia, PoliticaVersion: a.Politica.Version, PoliticaSHA256: a.Politica.HuellaSHA256, VersionEsperada: 1}
	if terminal.Resolucion != nil {
		esperado.ResueltaEn = terminal.Resolucion.ResueltaEn
	}
	if err != nil || terminal.OperacionRef != terminalRef || terminal.Tipo != "renuncia_rrhh" || terminal.Resolucion == nil ||
		*terminal.Resolucion != esperado || esperado.ResueltaEn.Before(a.ResueltaEn) || terminal.Llamamiento == nil || terminal.Llamamiento.LlamamientoRef != a.Solicitud.LlamamientoRef {
		return vacio, ports.ErrOperacionContinuacionNoDisponible
	}
	servicio, err := appbolsa.NuevoServicioIntegracionLlamamientosDesarrollo(fuente, p.repositorio, p.autorizadorSiguiente, p.reloj)
	if err != nil {
		return vacio, ports.ErrOperacionContinuacionNoDisponible
	}
	r, err := servicio.SolicitarSiguienteLlamamiento(ctx, puertosbolsa.PeticionSiguienteLlamamientoDesarrollo{
		OperacionRef: operacionSiguienteDesarrollo(s), IntencionRef: intencionSiguienteBolsaDesarrollo(s), TerminalOperacionRef: terminalRef})
	if ctx.Err() != nil {
		return vacio, ctx.Err()
	}
	if err != nil {
		return vacio, ports.ErrOperacionContinuacionNoDisponible
	}
	canon, err := r.Registro.Canonico()
	if err != nil || r.Registro.Tipo != "propuesta" || r.Registro.OperacionRef != operacionSiguienteDesarrollo(s) ||
		r.Registro.Propuesta == nil || r.Registro.Propuesta.Continuacion == nil || r.Registro.Llamamiento == nil ||
		r.Registro.EstadoLlamamiento != dominiobolsa.EstadoLlamamientoAbierto {
		return vacio, ports.ErrOperacionContinuacionNoDisponible
	}
	ante := r.Registro.Propuesta.Continuacion
	if ante.TerminalOperacionRef != terminalRef || ante.TerminalSHA256 != huellaPuenteLlamamientoDesarrollo(canonTerminal) ||
		ante.IntencionRef != intencionSiguienteBolsaDesarrollo(s) || ante.PropuestaRef != apertura.Propuesta.PropuestaRef ||
		ante.PropuestaSHA256 != apertura.Propuesta.HuellaContenidoSHA256 || ante.OrdenAnterior != apertura.Propuesta.OrdenSeleccionado ||
		r.Registro.Llamamiento.LlamamientoRef == a.Solicitud.LlamamientoRef || r.ConfirmadaEn.Before(a.ResueltaEn) {
		return vacio, ports.ErrOperacionContinuacionNoDisponible
	}
	b := ports.ReciboBolsaContinuacion{IntencionRef: s.IntencionRef, TerminalOperacionRef: terminalRef,
		OperacionRef: r.Registro.OperacionRef, LlamamientoRef: r.Registro.Llamamiento.LlamamientoRef, PropuestaRef: r.Registro.Propuesta.PropuestaRef,
		ReciboRef: r.ReciboRef, AuditoriaRef: r.AuditoriaRef, EventoRef: r.EventoRef, RegistroSHA256: huellaPuenteLlamamientoDesarrollo(canon), ConfirmadaEn: r.ConfirmadaEn}
	if b.ValidarPara(s) != nil {
		return vacio, ports.ErrOperacionContinuacionNoDisponible
	}
	return b, nil
}

type proveedorContinuacionLlamamientoDesarrollo struct {
	soporte     *soporteAltaContratacionTemporalDesarrollo
	autorizador autorizacionComunicacionLlamamientoDesarrollo
	reloj       ports.Reloj
}

func (p *proveedorContinuacionLlamamientoDesarrollo) AutorizarContinuacionLlamamiento(ctx context.Context, m ports.MaterialContinuacionLlamamiento) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacio := puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	if contextoInterfazNulo(ctx) || p == nil || p.soporte == nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(p.autorizador) || dependenciaEsNulaContratacionTemporalDesarrollo(p.reloj) {
		return vacio, ports.ErrOperacionContinuacionDenegada
	}
	if ctx.Err() != nil {
		return vacio, ctx.Err()
	}
	c, valida := p.soporte.capacidadValida(ctx)
	esperado, existe := ctx.Value(claveMaterialContinuacionDesarrollo{}).(ports.MaterialContinuacionLlamamiento)
	r, err := postgresct.RecursoContinuacionLlamamiento(m)
	e, errEsperado := postgresct.RecursoContinuacionLlamamiento(esperado)
	if !valida || c.ruta != httpinterno.RutaContinuacionLlamamiento || !existe || err != nil || errEsperado != nil ||
		r.Referencia != e.Referencia || !maps.Equal(r.Ambitos, e.Ambitos) || !maps.Equal(r.Atributos, e.Atributos) {
		return vacio, ports.ErrOperacionContinuacionDenegada
	}
	if _, _, vigente := ventanaAutoridadSinteticaContratacionTemporalDesarrollo(p.reloj.Ahora()); !vigente {
		return vacio, ports.ErrOperacionContinuacionDenegada
	}
	a, err := p.autorizador.AutorizarOperacion(ctx, postgresct.AccionContinuacionLlamamiento, r)
	if ctx.Err() != nil {
		return vacio, ctx.Err()
	}
	if err != nil || a.ValidarEstructura() != nil {
		return vacio, ports.ErrOperacionContinuacionDenegada
	}
	resumen, ahora := a.ResumenCapacidad(), p.reloj.Ahora()
	if ahora.Before(resumen.EmitidaEn()) || !ahora.Before(resumen.ExpiraEn()) || resumen.ExpiraEn().Sub(resumen.EmitidaEn()) > 5*time.Minute {
		return vacio, ports.ErrOperacionContinuacionDenegada
	}
	return a, nil
}
