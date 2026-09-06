package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	postgresbolsa "vec-diputacion-granada/internal/modules/bolsa/adapters/postgres"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type clavePropuestaFormalizacionDesarrollo struct{}
type claveMaterialPropuestaFormalizacionDesarrollo struct{}

type propuestaFormalizacionLigadaDesarrollo struct {
	antecedente ports.AntecedentePropuestaFormalizacion
	material    ports.MaterialPropuestaFormalizacion
}

type registroPropuestaFormalizacionDesarrollo interface {
	ports.TransaccionPropuestaFormalizacion
	LeerAntecedente(context.Context, ports.SolicitudPropuestaFormalizacion) (ports.AntecedentePropuestaFormalizacion, error)
}

type ejecutorPropuestaFormalizacionDesarrollo struct {
	soporte       *soporteAltaContratacionTemporalDesarrollo
	lector        ports.LectorExpedienteSeleccionLlamamiento
	registro      registroPropuestaFormalizacionDesarrollo
	bolsa         puertosbolsa.RepositorioLlamamientoDesarrollo
	publicaciones publicacionesPropuestaDesarrollo
	servicio      httpinterno.EjecutorPropuestaFormalizacion
}

var (
	_ httpinterno.EjecutorPropuestaFormalizacion          = (*ejecutorPropuestaFormalizacionDesarrollo)(nil)
	_ httpinterno.AutoridadServidorPropuestaFormalizacion = (*ejecutorPropuestaFormalizacionDesarrollo)(nil)
)

// Reutiliza los pools, la ruta, el puerto y el caso de uso existentes. No
// arranca otro servidor ni una segunda línea de formalización.
func nuevasDependenciasPropuestaFormalizacionDesarrollo(alta *dependenciasAltaContratacionTemporalDesarrollo, reloj relojContratacionTemporalDesarrollo) (*ejecutorPropuestaFormalizacionDesarrollo, error) {
	if alta == nil || alta.soporte == nil || alta.postgresql.bolsa == nil || alta.postgresql.ejecucion == nil {
		return nil, application.ErrPropuestaFormalizacionNoDisponible
	}
	publicaciones, err := cargarPublicacionesPropuestaDesarrollo()
	if err != nil {
		return nil, application.ErrPropuestaFormalizacionNoDisponible
	}
	lector, err := postgresct.NuevoLectorExpedienteSeleccionLlamamientoPostgreSQL(alta.postgresql.ejecucion)
	if err != nil {
		return nil, err
	}
	bolsa, err := postgresbolsa.NuevoRepositorioIntegracionLlamamientosDesarrollo(alta.postgresql.bolsa)
	if err != nil {
		return nil, err
	}
	proveedor := &proveedorPropuestaFormalizacionDesarrollo{
		soporte: alta.soporte, reloj: reloj, publicaciones: publicaciones,
		autorizador: &autorizadorLlamamientoDesarrollo{alta: alta, material: alta.postgresql.proveedorMaterial, propuestaFormalizacion: true},
	}
	registro, err := postgresct.NuevoRegistroPropuestaFormalizacionPostgreSQL(alta.postgresql.ejecucion, proveedor)
	if err != nil {
		return nil, err
	}
	servicio, err := application.NuevoServicioPropuestaFormalizacion(registro)
	if err != nil {
		return nil, err
	}
	return &ejecutorPropuestaFormalizacionDesarrollo{soporte: alta.soporte, lector: lector,
		registro: registro, bolsa: bolsa, publicaciones: publicaciones, servicio: servicio}, nil
}

func (e *ejecutorPropuestaFormalizacionDesarrollo) ResolverContextoPropuestaFormalizacion(ctx context.Context) (httpinterno.ContextoServidorPropuestaFormalizacion, error) {
	vacio := httpinterno.ContextoServidorPropuestaFormalizacion{}
	if contextoInterfazNulo(ctx) || e == nil || e.soporte == nil {
		return vacio, application.ErrPropuestaFormalizacionDenegada
	}
	c, valida := e.soporte.capacidadValida(ctx)
	if !valida || c.ruta != httpinterno.RutaPropuestaFormalizacion {
		return vacio, application.ErrPropuestaFormalizacionDenegada
	}
	return httpinterno.ContextoServidorPropuestaFormalizacion{OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo}, nil
}

func (e *ejecutorPropuestaFormalizacionDesarrollo) PrepararYConfirmar(ctx context.Context, s ports.SolicitudPropuestaFormalizacion) (ports.ResultadoPropuestaFormalizacion, error) {
	vacio := ports.ResultadoPropuestaFormalizacion{}
	organizacion, err := e.ResolverContextoPropuestaFormalizacion(ctx)
	if err != nil {
		return vacio, err
	}
	if s.OrganizacionRef != organizacion.OrganizacionRef || !e.publicaciones.admite(s) {
		return vacio, application.ErrSolicitudPropuestaFormalizacionInvalida
	}
	if dependenciaEsNulaContratacionTemporalDesarrollo(e.lector) || dependenciaEsNulaContratacionTemporalDesarrollo(e.registro) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(e.bolsa) || dependenciaEsNulaContratacionTemporalDesarrollo(e.servicio) {
		return vacio, application.ErrPropuestaFormalizacionNoDisponible
	}
	expediente, err := e.lector.LeerExpedienteParaSeleccion(ctx, s.OrganizacionRef, s.ExpedienteRef, 6)
	if ctx.Err() != nil {
		return vacio, ctx.Err()
	}
	// Se recupera el antecedente v6, no se bloquea un replay porque la propuesta
	// ya haya avanzado el expediente. PostgreSQL decide OCC o recuperación.
	if err != nil || !expedienteComunicacionLlamamientoDesarrolloValido(expediente,
		ports.SolicitudRegistrarComunicacionLlamamiento{OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef}) {
		return vacio, application.ErrPropuestaFormalizacionNoDisponible
	}
	ctx = context.WithValue(ctx, clavePreparacionLlamamientoDesarrollo{}, preparacionLlamamientoDesarrollo{expediente: expediente})
	ctx = context.WithValue(ctx, claveMaterialPropuestaFormalizacionDesarrollo{}, ports.MaterialPropuestaFormalizacion{Etapa: "consulta", Solicitud: s.Clonar()})
	a, err := e.registro.LeerAntecedente(ctx, s)
	if ctx.Err() != nil {
		return vacio, ctx.Err()
	}
	if err != nil {
		return vacio, errorPreparacionPropuestaDesarrollo(err)
	}
	if a.ValidarPara(s) != nil || a.Resolucion.Politica != politicaManualDesarrollo() ||
		!consultaJustificanteLigadaAlExpedienteDesarrollo(expediente, a.Resolucion.Solicitud) {
		return vacio, application.ErrResolucionFormalizacionNoAceptada
	}
	p, err := prepararReferenciasLlamamientoDesarrollo(expediente, a.SeleccionClave)
	if err != nil || p.operacionPropuesta != a.Justificante.Seleccion.OperacionRef {
		return vacio, application.ErrResolucionFormalizacionNoAceptada
	}
	ctx = context.WithValue(ctx, clavePreparacionLlamamientoDesarrollo{}, p)
	// Lectura por el repositorio propietario Bolsa. No se consulta su tabla
	// desde CT ni se vuelve a ejecutar una aceptación para fabricar el requisito.
	b, err := acreditarAceptacionBolsaPropuestaDesarrollo(ctx, e.bolsa, a, p)
	if err != nil {
		return vacio, err
	}
	if b.ValidarPara(a) != nil {
		return vacio, application.ErrResolucionFormalizacionNoAceptada
	}
	m := ports.MaterialPropuestaFormalizacion{Etapa: "confirmacion", Solicitud: s.Clonar(), AceptacionBolsa: &b}
	ctx = context.WithValue(ctx, clavePropuestaFormalizacionDesarrollo{}, propuestaFormalizacionLigadaDesarrollo{antecedente: a, material: m})
	ctx = context.WithValue(ctx, claveMaterialPropuestaFormalizacionDesarrollo{}, m)
	return e.servicio.PrepararYConfirmar(ctx, s)
}

func acreditarAceptacionBolsaPropuestaDesarrollo(ctx context.Context, repo puertosbolsa.RepositorioLlamamientoDesarrollo,
	a ports.AntecedentePropuestaFormalizacion, p preparacionLlamamientoDesarrollo) (ports.EvidenciaAceptacionBolsaPropuesta, error) {
	vacio := ports.EvidenciaAceptacionBolsaPropuesta{}
	l := aceptacionRevisadaDesarrollo{solicitud: a.Resolucion.Solicitud, justificante: a.Justificante, local: a.Resolucion}
	operacion := operacionAceptacionManualDesarrollo(l)
	b, existe, err := repo.BuscarOperacion(ctx, operacion)
	if ctx.Err() != nil {
		return vacio, ctx.Err()
	}
	if err != nil {
		return vacio, application.ErrPropuestaFormalizacionNoDisponible
	}
	canon, err := b.Canonico()
	j, r, s := a.Justificante.Seleccion, a.Resolucion, a.Resolucion.Solicitud
	aperturaRef, propuestaRef := j.OperacionRef, j.Propuesta.Referencia
	if c := a.Justificante.Continuacion; c != nil {
		aperturaRef, propuestaRef = c.ReciboBolsa.OperacionRef, c.ReciboBolsa.PropuestaRef
	}
	if !existe || err != nil || b.Tipo != "aceptacion_rrhh" || b.OperacionRef != operacion ||
		b.EstadoLlamamiento != dominiobolsa.EstadoLlamamientoAceptado || b.Resolucion == nil || b.Llamamiento == nil || b.Propuesta == nil ||
		b.NecesidadRef != j.Necesidad.Referencia || b.CategoriaRef != p.categoria || b.UnidadRef != p.unidad ||
		b.OrdenOperacionRef != p.operacionOrden || b.Propuesta.PropuestaRef != propuestaRef ||
		b.Llamamiento.LlamamientoRef != s.LlamamientoRef || b.Llamamiento.Version != 2 ||
		b.Resolucion.AperturaOperacionRef != aperturaRef || b.Resolucion.JustificanteRef != s.PruebaRespuestaRef ||
		b.Resolucion.EvaluacionPlazoRef != r.EvaluacionPlazoRef || b.Resolucion.PoliticaRef != r.Politica.Referencia ||
		b.Resolucion.PoliticaVersion != r.Politica.Version || b.Resolucion.PoliticaSHA256 != r.Politica.HuellaSHA256 ||
		b.Resolucion.ResueltaEn.Before(r.ResueltaEn) {
		return vacio, application.ErrResolucionFormalizacionNoAceptada
	}
	if a.Justificante.Continuacion != nil {
		if err := acreditarCadenaSucesoraPropuestaDesarrollo(ctx, repo, a, b, p); err != nil {
			return vacio, err
		}
	}
	h := sha256.Sum256(canon)
	return ports.EvidenciaAceptacionBolsaPropuesta{OperacionRef: operacion, AperturaOperacionRef: aperturaRef,
		LlamamientoRef: s.LlamamientoRef, JustificanteRef: s.PruebaRespuestaRef, EvaluacionPlazoRef: r.EvaluacionPlazoRef,
		Politica:       ports.SnapshotGobernadoFormalizacion{Referencia: r.Politica.Referencia, Version: r.Politica.Version, HuellaSHA256: r.Politica.HuellaSHA256},
		RegistroSHA256: hex.EncodeToString(h[:]), ResueltaEn: b.Resolucion.ResueltaEn}, nil
}

// Solo recupera originales del repositorio propietario. La continuación fija
// la apertura; no se transforma Seleccion ni se vuelve a aceptar en Bolsa.
func acreditarCadenaSucesoraPropuestaDesarrollo(ctx context.Context, repo puertosbolsa.RepositorioLlamamientoDesarrollo,
	a ports.AntecedentePropuestaFormalizacion, aceptacion puertosbolsa.RegistroLlamamientoDesarrollo, p preparacionLlamamientoDesarrollo,
) error {
	j, c := a.Justificante.Seleccion, a.Justificante.Continuacion
	if c == nil || a.Justificante.ValidarPara(a.Resolucion.Solicitud) != nil {
		return application.ErrResolucionFormalizacionNoAceptada
	}
	refs := [3]string{j.OperacionRef, c.ReciboBolsa.OperacionRef, c.ReciboBolsa.TerminalOperacionRef}
	var registros [3]puertosbolsa.RegistroLlamamientoDesarrollo
	var canones [3][]byte
	for i, ref := range refs {
		r, existe, err := repo.BuscarOperacion(ctx, ref)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil {
			return application.ErrPropuestaFormalizacionNoDisponible
		}
		canon, err := r.Canonico()
		if !existe || err != nil || r.OperacionRef != ref {
			return application.ErrResolucionFormalizacionNoAceptada
		}
		registros[i], canones[i] = r, canon
	}
	raiz, apertura, terminal := registros[0], registros[1], registros[2]
	if raiz.Tipo != "propuesta" || raiz.Propuesta == nil || raiz.Propuesta.Continuacion != nil || raiz.Llamamiento == nil ||
		raiz.OperacionRef != p.operacionPropuesta || raiz.OrdenOperacionRef != p.operacionOrden ||
		raiz.Llamamiento.LlamamientoRef != j.LlamamientoRef ||
		j.Resultado != referenciaVersionadaPuenteLlamamientoDesarrollo(j.ReciboRef, 1, huellaPuenteLlamamientoDesarrollo(canones[0])) ||
		j.Propuesta != referenciaVersionadaPuenteLlamamientoDesarrollo(raiz.Propuesta.PropuestaRef, 1, raiz.Propuesta.HuellaContenidoSHA256) ||
		j.Orden != referenciaVersionadaPuenteLlamamientoDesarrollo(raiz.Instantanea.InstantaneaRef, raiz.Instantanea.Version, raiz.Instantanea.HuellaContenidoSHA256) ||
		uint32(raiz.Propuesta.OrdenSeleccionado) != j.OrdenSeleccionado ||
		apertura.Tipo != "propuesta" || apertura.Propuesta == nil || apertura.Propuesta.Continuacion == nil || apertura.Llamamiento == nil ||
		apertura.Llamamiento.LlamamientoRef != c.ReciboBolsa.LlamamientoRef || apertura.Propuesta.PropuestaRef != c.ReciboBolsa.PropuestaRef ||
		huellaPuenteLlamamientoDesarrollo(canones[1]) != c.ReciboBolsa.RegistroSHA256 ||
		apertura.Propuesta.GeneradaEn.After(c.ReciboBolsa.ConfirmadaEn) {
		return application.ErrResolucionFormalizacionNoAceptada
	}
	ante := apertura.Propuesta.Continuacion
	if terminal.Tipo != "renuncia_rrhh" || terminal.Resolucion == nil || terminal.Llamamiento == nil ||
		terminal.Resolucion.AperturaOperacionRef != raiz.OperacionRef || terminal.Llamamiento.LlamamientoRef != j.LlamamientoRef ||
		ante.TerminalOperacionRef != terminal.OperacionRef || ante.TerminalSHA256 != huellaPuenteLlamamientoDesarrollo(canones[2]) ||
		ante.PropuestaRef != raiz.Propuesta.PropuestaRef || ante.PropuestaSHA256 != raiz.Propuesta.HuellaContenidoSHA256 ||
		ante.OrdenAnterior != raiz.Propuesta.OrdenSeleccionado || apertura.Propuesta.OrdenSeleccionado <= ante.OrdenAnterior ||
		ante.IntencionRef != intencionSiguienteBolsaDesarrollo(c.Solicitud) ||
		terminal.Resolucion.ResueltaEn.Before(j.ConfirmadaEn) || terminal.Resolucion.ResueltaEn.After(apertura.Propuesta.GeneradaEn) {
		return application.ErrResolucionFormalizacionNoAceptada
	}
	// Igualdad de origen/firma/instantánea/orden íntegros, no una nueva fuente.
	base := apertura
	base.OperacionRef, base.Propuesta, base.Llamamiento = raiz.OperacionRef, raiz.Propuesta, raiz.Llamamiento
	canonBase, err := base.Canonico()
	if err != nil || !bytes.Equal(canonBase, canones[0]) {
		return application.ErrResolucionFormalizacionNoAceptada
	}
	// El terminal actual debe conservar toda su apertura, incluido el enlace
	// de continuación; solamente cambia el estado/version y su resolución.
	esperada := apertura
	datos := *apertura.Llamamiento
	datos.Version = 2
	esperada.OperacionRef, esperada.Tipo, esperada.EstadoLlamamiento = aceptacion.OperacionRef, "aceptacion_rrhh", dominiobolsa.EstadoLlamamientoAceptado
	esperada.Llamamiento, esperada.Resolucion = &datos, aceptacion.Resolucion
	canonEsperado, err := esperada.Canonico()
	canonAceptacion, errAceptacion := aceptacion.Canonico()
	if err != nil || errAceptacion != nil || !bytes.Equal(canonEsperado, canonAceptacion) {
		return application.ErrResolucionFormalizacionNoAceptada
	}
	return nil
}

func errorPreparacionPropuestaDesarrollo(err error) error {
	// Mantiene la clasificación única del caso de uso, sin errores SQL ni datos.
	switch {
	case errors.Is(err, context.Canceled):
		return context.Canceled
	case errors.Is(err, context.DeadlineExceeded):
		return context.DeadlineExceeded
	case errors.Is(err, ports.ErrOperacionPropuestaFormalizacionDenegada):
		return application.ErrPropuestaFormalizacionDenegada
	case errors.Is(err, ports.ErrResolucionLlamamientoNoAceptada):
		return application.ErrResolucionFormalizacionNoAceptada
	case errors.Is(err, ports.ErrVersionPropuestaFormalizacionEnConflicto):
		return application.ErrVersionPropuestaFormalizacionEnConflicto
	case errors.Is(err, ports.ErrClavePropuestaFormalizacionUsada):
		return application.ErrClavePropuestaFormalizacionEnColision
	}
	return application.ErrPropuestaFormalizacionNoDisponible
}
