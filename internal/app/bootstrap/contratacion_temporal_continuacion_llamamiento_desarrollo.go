package bootstrap

import (
	"context"
	"crypto/ed25519"
	"errors"
	"maps"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/adapters/fuentesintetica"
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
// En una renuncia la selección procede del justificante consultado; en una
// expiración confirmada, sin justificante, del antecedente que devuelve CT119.
type continuacionLigadaDesarrollo struct {
	solicitud        ports.SolicitudContinuarLlamamiento
	antecedente      ports.AntecedenteContinuacionLlamamiento
	justificante     ports.JustificanteRespuestaRecibida
	seleccion        ports.ReciboSolicitudLlamamientoBolsa
	soloRecuperacion bool
}

// antecedenteLigado exige que la selección usada sea exactamente la del
// justificante (renuncia) o la del antecedente CT119 (expiración).
func (l continuacionLigadaDesarrollo) antecedenteLigado() bool {
	if l.antecedente.EsExpiracion() {
		return l.antecedente.Seleccion != nil && *l.antecedente.Seleccion == l.seleccion &&
			l.justificante == (ports.JustificanteRespuestaRecibida{})
	}
	return l.antecedente.Seleccion == nil && l.justificante.ValidarPara(l.antecedente.Resolucion.Solicitud) == nil &&
		l.justificante.Seleccion == l.seleccion
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
	if err != nil || expediente.Fiscalizado.Validar() != nil ||
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
	if a.ValidarPara(s) != nil || !politicaAntecedenteContinuacionDesarrolloValida(a.Resolucion) {
		return vacio, ports.ErrOperacionContinuacionNoDisponible
	}
	l := continuacionLigadaDesarrollo{solicitud: s, antecedente: a, soloRecuperacion: expediente.VersionActual > 6}
	if a.EsExpiracion() {
		// Sin respuesta no hay justificante: la selección original la devolvió
		// CT119 tras consumir el permiso de continuación.
		l.seleccion = *a.Seleccion
	} else {
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
		l.justificante, l.seleccion = j, j.Seleccion
	}
	preparada, err := prepararReferenciasLlamamientoDesarrollo(expediente, a.ComandoSiguiente.SeleccionClave)
	if err != nil || preparada.operacionPropuesta != l.seleccion.OperacionRef {
		return vacio, ports.ErrOperacionContinuacionNoDisponible
	}
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

// El terminal Bolsa de la renuncia lo crea la resolución; el de la expiración
// («sin respuesta») lo crea la propia continuación, con referencia propia.
func terminalContinuacionDesarrollo(l continuacionLigadaDesarrollo) string {
	if l.antecedente.EsExpiracion() {
		s := l.antecedente.Resolucion.Solicitud
		return referenciaPuenteLlamamientoDesarrollo("operacion-expiracion-rrhh",
			s.OrganizacionRef, s.ExpedienteRef, l.seleccion.OperacionRef, s.ClaveIdempotencia)
	}
	return operacionAceptacionManualDesarrollo(aceptacionRevisadaDesarrollo{
		solicitud: l.antecedente.Resolucion.Solicitud, justificante: l.justificante, local: l.antecedente.Resolucion})
}

// politicaAntecedenteContinuacionDesarrolloValida admite la política con la que
// RRHH confirmó la resolución: la histórica sintética o una regla del catálogo
// (la expiración siempre se confirma con la regla de falta de respuesta).
func politicaAntecedenteContinuacionDesarrolloValida(r ports.ResultadoResolucionLlamamiento) bool {
	p := r.Politica
	return politicaResolucionAdmitidaDesarrollo(p.Referencia, p.Version, p.HuellaSHA256) &&
		r.Solicitud.CriterioValidacionRef == p.Referencia &&
		(r.Solicitud.Respuesta != ports.RespuestaLlamamientoExpirada || p.Referencia != criterioRevisionManualDesarrollo)
}

func antecedenteContinuacionDesarrolloValido(ctx context.Context, s ports.SolicitudResolverLlamamiento) bool {
	l, ok := ctx.Value(claveContinuacionLlamamientoDesarrollo{}).(continuacionLigadaDesarrollo)
	return ok && l.antecedente.ValidarPara(l.solicitud) == nil && l.antecedente.Resolucion.Solicitud == s &&
		politicaAntecedenteContinuacionDesarrolloValida(l.antecedente.Resolucion)
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
		!antecedenteContinuacionDesarrolloValido(ctx, l.antecedente.Resolucion.Solicitud) || !l.antecedenteLigado() {
		return vacio, ports.ErrOperacionContinuacionDenegada
	}
	a := l.antecedente.Resolucion
	apertura, existe, err := p.repositorio.BuscarOperacion(ctx, l.seleccion.OperacionRef)
	if err != nil || !existe {
		return vacio, ports.ErrOperacionContinuacionNoDisponible
	}
	fuente, err := p.fuenteResolucionLigada(a.Solicitud, l.seleccion, apertura)
	if err != nil {
		return vacio, ports.ErrOperacionContinuacionNoDisponible
	}
	terminalRef, tipoTerminal := terminalContinuacionDesarrollo(l), "renuncia_rrhh"
	esperado := puertosbolsa.ResolucionLlamamientoDesarrollo{AperturaOperacionRef: l.seleccion.OperacionRef,
		JustificanteRef: a.Solicitud.PruebaRespuestaRef, EvaluacionPlazoRef: a.EvaluacionPlazoRef,
		PoliticaRef: a.Politica.Referencia, PoliticaVersion: a.Politica.Version, PoliticaSHA256: a.Politica.HuellaSHA256, VersionEsperada: 1}
	if l.antecedente.EsExpiracion() {
		// Sin respuesta en plazo: la resolución CT confirmada por RRHH es la
		// prueba de la no aceptación que cierra el llamamiento en Bolsa.
		// Se traduce al alfabeto opaco de Bolsa, como la intención (un UUID de
		// CT puede parecer un documento personal a sus validadores).
		tipoTerminal = puertosbolsa.TipoExpiracionRRHHDesarrollo
		esperado.JustificanteRef = referenciaPuenteLlamamientoDesarrollo("resolucion-expiracion-rrhh",
			a.Solicitud.OrganizacionRef, a.Solicitud.ExpedienteRef, a.ResolucionRef)
		if err := p.cerrarSinRespuestaDesarrollo(ctx, fuente, terminalRef, esperado, l.soloRecuperacion); err != nil {
			return vacio, err
		}
	}
	terminal, existe, err := p.repositorio.BuscarOperacion(ctx, terminalRef)
	if err != nil || !existe {
		return vacio, ports.ErrOperacionContinuacionNoDisponible
	}
	canonTerminal, err := terminal.Canonico()
	if terminal.Resolucion != nil {
		esperado.ResueltaEn = terminal.Resolucion.ResueltaEn
	}
	if err != nil || terminal.OperacionRef != terminalRef || terminal.Tipo != tipoTerminal || terminal.Resolucion == nil ||
		*terminal.Resolucion != esperado || esperado.ResueltaEn.Before(a.ResueltaEn) || terminal.Llamamiento == nil || terminal.Llamamiento.LlamamientoRef != a.Solicitud.LlamamientoRef {
		return vacio, ports.ErrOperacionContinuacionNoDisponible
	}
	// La propuesta de nombramiento avanza a v7. Desde ahí solo se recupera
	// una apertura existente; nunca se ejecuta otra continuación. El servicio
	// mantiene la autorización fresca y la validación completa del replay.
	if l.soloRecuperacion {
		previa, existe, err := p.repositorio.BuscarOperacion(ctx, operacionSiguienteDesarrollo(s))
		if ctx.Err() != nil {
			return vacio, ctx.Err()
		}
		if err != nil || !existe || previa.Tipo != "propuesta" || previa.Propuesta == nil || previa.Propuesta.Continuacion == nil {
			return vacio, ports.ErrOperacionContinuacionNoDisponible
		}
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

// cerrarSinRespuestaDesarrollo registra en Bolsa el terminal «sin respuesta»
// por el mismo puente que la renuncia: servicio de integración, permiso de no
// aceptación de RRHH y registro inmutable con auditoría y outbox. Es
// idempotente: si ya existe, no pide otra autorización ni crea otro efecto; en
// modo de solo recuperación nunca lo crea.
func (p *puenteBolsaLlamamientoDesarrollo) cerrarSinRespuestaDesarrollo(ctx context.Context,
	fuente *fuentesintetica.FuenteLlamamientos, terminalRef string, resolucion puertosbolsa.ResolucionLlamamientoDesarrollo, soloRecuperacion bool,
) error {
	_, existe, err := p.repositorio.BuscarOperacion(ctx, terminalRef)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err != nil {
		return ports.ErrOperacionContinuacionNoDisponible
	}
	if existe {
		return nil
	}
	if soloRecuperacion || dependenciaEsNulaContratacionTemporalDesarrollo(p.autorizadorRenuncia) {
		return ports.ErrOperacionContinuacionNoDisponible
	}
	servicio, err := appbolsa.NuevoServicioIntegracionLlamamientosDesarrollo(fuente, p.repositorio, p.autorizadorRenuncia, p.reloj)
	if err != nil {
		return ports.ErrOperacionContinuacionNoDisponible
	}
	r, err := servicio.ExpirarLlamamiento(ctx, puertosbolsa.PeticionResolverLlamamientoDesarrollo{OperacionRef: terminalRef, Resolucion: resolucion})
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err != nil || r.Registro.Tipo != puertosbolsa.TipoExpiracionRRHHDesarrollo || r.Registro.OperacionRef != terminalRef ||
		r.Registro.EstadoLlamamiento != dominiobolsa.EstadoLlamamientoExpirado {
		return ports.ErrOperacionContinuacionNoDisponible
	}
	return nil
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
