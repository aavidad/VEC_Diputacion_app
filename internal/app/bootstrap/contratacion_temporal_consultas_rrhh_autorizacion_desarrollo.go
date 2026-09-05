package bootstrap

import (
	"bytes"
	"context"
	"sync"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// La raíz entrega los dos motivos resueltos por PostgreSQL. Esta pieza solo
// liga el canal y las concesiones a las autoridades existentes de VEC.
type autoridadConsultasRRHHDesarrollo struct {
	soporte   *soporteAltaContratacionTemporalDesarrollo
	delegado  puertosvec.AutorizadorSolicitudLigadaV3
	reloj     ports.Reloj
	mu        sync.Mutex
	proveedor proveedorContextoConsultaRRHHDesarrollo
}

type proveedorContextoConsultaRRHHDesarrollo interface {
	ResolverContexto(context.Context) (ports.ContextoAutorizacionAltaV3, error)
}

// Nace exclusivamente con la capacidad mTLS de una petición. No comparte
// sesiones entre peticiones y conserva también el fallo, sin repetir el alta.
type contextoConsultaRRHHPeticionDesarrollo struct {
	mu        sync.Mutex
	autoridad *autoridadConsultasRRHHDesarrollo
	contexto  ports.ContextoAutorizacionAltaV3
	err       error
}

// La raíz configura una sola fuente nominal antes de servir peticiones.
// Sin ella nunca se recurre al vínculo histórico del soporte de alta.
func (a *autoridadConsultasRRHHDesarrollo) configurarProveedorContextoConsultaRRHHDesarrollo(
	proveedor proveedorContextoConsultaRRHHDesarrollo,
) error {
	if a == nil || dependenciaEsNulaContratacionTemporalDesarrollo(proveedor) {
		return ports.ErrConsultaRRHHNoDisponible
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.proveedor != nil {
		return ports.ErrConsultaRRHHNoDisponible
	}
	a.proveedor = proveedor
	return nil
}

var (
	_ ports.AutoridadContextoConsultaRRHH     = (*autoridadConsultasRRHHDesarrollo)(nil)
	_ puertosvec.AutorizadorSolicitudLigadaV3 = (*autoridadConsultasRRHHDesarrollo)(nil)
)

func configurarAutoridadConsultasRRHHDesarrollo(
	alta *dependenciasAltaContratacionTemporalDesarrollo,
	reloj ports.Reloj,
	motivoCuadro, motivoDetalle dominiovec.ReferenciaEntradaCatalogo,
) (*autoridadConsultasRRHHDesarrollo, error) {
	if alta == nil || alta.soporte == nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(alta.autorizador) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(reloj) ||
		!dominiovec.ReferenciaMotivoAutorizacionV2Valida(motivoCuadro) ||
		!dominiovec.ReferenciaMotivoAutorizacionV2Valida(motivoDetalle) || motivoCuadro == motivoDetalle {
		return nil, ports.ErrConsultaRRHHNoDisponible
	}
	s := alta.soporte
	instante := reloj.Ahora()
	vinculo, err := s.contexto.Vinculo.Datos()
	// El identificador del certificado y el del actor registrado pertenecen
	// a fronteras distintas. capacidadValida liga el primero al canal; el
	// vínculo registrado liga el segundo al perfil, sin equipararlos.
	if err != nil || s.principalID == "" || s.certificadoSHA256 == "" || s.sello == nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(s.autoridadAsignaciones) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(s.registroDecisionesAnalisis) {
		return nil, ports.ErrConsultaRRHHNoDisponible
	}
	if _, err := ports.NuevoContextoConsultaRRHH(s.contexto, organizacionAltaContratacionTemporalDesarrollo, instante); err != nil {
		return nil, ports.ErrConsultaRRHHNoDisponible
	}
	nueva := func(rol, nombre, accion, finalidad, tipo string) (dominiovec.InstantaneaAutorizacion, error) {
		return nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(
			vinculo.PrincipalID, vinculo.PerfilActivoRef, instante, rol, nombre, rol,
			[]dominiovec.ConcesionRol{{Accion: accion, ModuloID: ports.ModuloContratacion,
				TipoRecurso: tipo, Finalidades: []string{finalidad}, GarantiaMinima: dominiovec.AuthAssuranceHigh}},
			[]dominiovec.AmbitoPerfil{
				{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}},
				{Clave: "clase_ambito", Valores: []string{string(ports.AmbitoOrganizacionRRHH)}},
				{Clave: "ambito_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}},
			})
	}
	cuadro, err := nueva("consulta_cuadro_rrhh_desarrollo", "Consulta de bandeja de desarrollo",
		ports.AccionConsultarCuadroRRHH, ports.FinalidadConsultarCuadroRRHH, ports.TipoRecursoCuadroRRHH)
	if err != nil {
		return nil, err
	}
	detalle, err := nueva("consulta_detalle_rrhh_desarrollo", "Consulta de expediente de desarrollo",
		ports.AccionConsultarDetalleRRHH, ports.FinalidadConsultarDetalleRRHH, ports.TipoRecursoExpediente)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// Solo se configura antes de servir peticiones; no reemplaza roles publicados.
	if s.instantaneaCuadroRRHH.VersionRol.RolID != "" || s.instantaneaDetalleRRHH.VersionRol.RolID != "" {
		return nil, ports.ErrConsultaRRHHNoDisponible
	}
	s.instantaneaCuadroRRHH, s.instantaneaDetalleRRHH = cuadro, detalle
	s.motivoCuadroRRHH, s.motivoDetalleRRHH = motivoCuadro, motivoDetalle
	return &autoridadConsultasRRHHDesarrollo{soporte: s, delegado: alta.autorizador, reloj: reloj}, nil
}

func rutaConsultaRRHHContratacionTemporalDesarrollo(ruta string) bool {
	return ruta == httpinterno.RutaConsultaCuadroRRHH || ruta == httpinterno.RutaConsultaDetalleRRHH
}

func (a *autoridadConsultasRRHHDesarrollo) ResolverContextoConsultaRRHH(ctx context.Context) (ports.ContextoConsultaRRHH, error) {
	contexto, err := a.contextoConsultaRRHHDesarrollo(ctx)
	if err != nil {
		return ports.ContextoConsultaRRHH{}, err
	}
	return ports.NuevoContextoConsultaRRHH(contexto, organizacionAltaContratacionTemporalDesarrollo, a.reloj.Ahora())
}

func (a *autoridadConsultasRRHHDesarrollo) contextoConsultaRRHHDesarrollo(ctx context.Context) (ports.ContextoAutorizacionAltaV3, error) {
	if a == nil || a.soporte == nil || dependenciaEsNulaContratacionTemporalDesarrollo(a.reloj) {
		return ports.ContextoAutorizacionAltaV3{}, ports.ErrConsultaRRHHNoDisponible
	}
	capacidad, valida := a.soporte.capacidadValida(ctx)
	if !valida || !rutaConsultaRRHHContratacionTemporalDesarrollo(capacidad.ruta) || capacidad.consultaRRHH == nil {
		return ports.ContextoAutorizacionAltaV3{}, ports.ErrAutorizacionDenegada
	}
	a.mu.Lock()
	proveedor := a.proveedor
	a.mu.Unlock()
	if dependenciaEsNulaContratacionTemporalDesarrollo(proveedor) {
		return ports.ContextoAutorizacionAltaV3{}, ports.ErrConsultaRRHHNoDisponible
	}
	peticion := capacidad.consultaRRHH
	peticion.mu.Lock()
	defer peticion.mu.Unlock()
	if peticion.autoridad == nil {
		peticion.autoridad = a
		peticion.contexto, peticion.err = proveedor.ResolverContexto(ctx)
		if peticion.err != nil || !a.contextoConsultaRRHHConservaActor(peticion.contexto) {
			peticion.contexto = ports.ContextoAutorizacionAltaV3{}
			peticion.err = ports.ErrAutorizacionDenegada
		} else {
			peticion.contexto.Resultado, peticion.err = peticion.contexto.Resultado.Clonar()
		}
	}
	if peticion.autoridad != a || peticion.err != nil || ctx.Err() != nil {
		return ports.ContextoAutorizacionAltaV3{}, ports.ErrAutorizacionDenegada
	}
	// Cada consumidor vuelve a comprobar la vigencia; la caché de petición no
	// prolonga una sesión vencida ni permite registrar otra para el mismo flujo.
	if _, err := ports.NuevoContextoConsultaRRHH(peticion.contexto, organizacionAltaContratacionTemporalDesarrollo, a.reloj.Ahora()); err != nil {
		return ports.ContextoAutorizacionAltaV3{}, ports.ErrAutorizacionDenegada
	}
	contexto := peticion.contexto
	var err error
	contexto.Resultado, err = contexto.Resultado.Clonar()
	return contexto, err
}

func (a *autoridadConsultasRRHHDesarrollo) contextoConsultaRRHHConservaActor(contexto ports.ContextoAutorizacionAltaV3) bool {
	base := a.soporte.contexto
	nuevo, err := contexto.Vinculo.Datos()
	anterior, errBase := base.Vinculo.Datos()
	if err != nil || errBase != nil || contexto.Vinculo.ValidarPara(contexto.Resultado) != nil ||
		nuevo.CuentaRef != anterior.CuentaRef || nuevo.CuentaOrdinariaRef != anterior.CuentaOrdinariaRef ||
		nuevo.CuentaPrivilegiada != anterior.CuentaPrivilegiada || nuevo.Superficie != anterior.Superficie ||
		contexto.Resultado.AutoridadEfectiva != base.Resultado.AutoridadEfectiva ||
		!bytes.Equal(contexto.Resultado.ManifiestoProcedenciaCanonico, base.Resultado.ManifiestoProcedenciaCanonico) {
		return false
	}
	actor, err := contexto.Resultado.Contexto.Clonar()
	if err != nil {
		return false
	}
	// Solo se normaliza una copia para comparar la identidad y sus versiones.
	// El resultado fresco y su instante autoritativo nunca se modifican.
	actor.ResueltoEn = base.Resultado.Contexto.ResueltoEn
	canon, err := actor.RepresentacionCanonicaVinculadaV2()
	return err == nil && bytes.Equal(canon, base.Resultado.RepresentacionCanonica)
}

func (a *autoridadConsultasRRHHDesarrollo) ExigirSolicitudLigadaV3(
	ctx context.Context,
	solicitud dominiovec.SolicitudAutorizacionLigadaV3,
	resultado dominiovec.ResultadoContextoActorRegistradoV2,
) (dominiovec.DecisionAutorizacionLigadaV3, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3, error) {
	contexto, err := a.contextoConsultaRRHHDesarrollo(ctx)
	if err != nil {
		return dominiovec.DecisionAutorizacionLigadaV3{}, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, err
	}
	datos, err := solicitud.Datos()
	capacidad, valida := a.soporte.capacidadValida(ctx)
	if err != nil || !valida || dependenciaEsNulaContratacionTemporalDesarrollo(a.delegado) ||
		!datos.VinculoAutenticacionActor.CoincideExactamenteCon(contexto.Vinculo) ||
		datos.VinculoAutenticacionActor.ValidarPara(resultado) != nil ||
		!a.soporte.solicitudAutorizacionConsultaRRHHDesarrolloValida(capacidad.ruta, datos) {
		return dominiovec.DecisionAutorizacionLigadaV3{}, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, ports.ErrAutorizacionDenegada
	}
	// Misma clave privada usada por el soporte V3 para preparar y publicar la
	// asignación exacta antes del registro durable. No decide ni firma aquí.
	ctx = context.WithValue(ctx, claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, datos)
	return a.delegado.ExigirSolicitudLigadaV3(ctx, solicitud, resultado)
}

func (s *soporteAltaContratacionTemporalDesarrollo) solicitudAutorizacionConsultaRRHHDesarrolloValida(
	ruta string, datos dominiovec.DatosSolicitudAutorizacionLigadaV3,
) bool {
	motivo, valido := s.motivoAutorizacionParaRuta(ruta)
	r := datos.Recurso
	if !valido || datos.ReferenciaMotivo != motivo || r.Validar() != nil ||
		r.ModuloID != ports.ModuloContratacion || len(r.Ambitos) != 3 ||
		r.Ambitos["organizacion_ref"] != organizacionAltaContratacionTemporalDesarrollo ||
		r.Ambitos["clase_ambito"] != string(ports.AmbitoOrganizacionRRHH) ||
		r.Ambitos["ambito_ref"] != organizacionAltaContratacionTemporalDesarrollo ||
		len(r.Atributos) != 2 || !huellaSHA256ValidaContratacionTemporalDesarrollo(r.Atributos["consulta_huella_sha256"]) {
		return false
	}
	switch ruta {
	case httpinterno.RutaConsultaCuadroRRHH:
		return datos.Accion == ports.AccionConsultarCuadroRRHH && datos.Finalidad == ports.FinalidadConsultarCuadroRRHH &&
			r.Tipo == ports.TipoRecursoCuadroRRHH && r.Referencia == organizacionAltaContratacionTemporalDesarrollo &&
			r.Atributos["consulta_dominio"] == ports.DominioHuellaConsultaCuadroRRHH
	case httpinterno.RutaConsultaDetalleRRHH:
		return datos.Accion == ports.AccionConsultarDetalleRRHH && datos.Finalidad == ports.FinalidadConsultarDetalleRRHH &&
			r.Tipo == ports.TipoRecursoExpediente && domain.ReferenciaOpacaValida(r.Referencia) &&
			r.Atributos["consulta_dominio"] == ports.DominioHuellaConsultaDetalleRRHH
	default:
		return false
	}
}
