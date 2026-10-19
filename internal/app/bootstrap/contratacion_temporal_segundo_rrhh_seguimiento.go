package bootstrap

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	postgresidentidad "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// seleccionSoporteSeguimientoRRHH elige desde la capacidad sellada por la
// frontera mTLS. El segundo técnico sólo alcanza la consulta previa del
// expediente exacto y la no incorporación.
// Cada soporte conserva su comprobación nominal de sujeto, hoja y ruta.
type seleccionSoporteSeguimientoRRHH struct {
	original  *soporteAltaContratacionTemporalDesarrollo
	adicional *soporteAltaContratacionTemporalDesarrollo
}

func (s seleccionSoporteSeguimientoRRHH) elegir(ctx context.Context) (*soporteAltaContratacionTemporalDesarrollo, bool) {
	if s.original == nil || contextoInterfazNulo(ctx) || ctx.Err() != nil {
		return nil, false
	}
	capacidad, ok := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	if !ok || capacidad.sello == nil || capacidad.principal.ID == "" {
		return nil, false
	}
	if capacidad.principal.ID == s.original.principalID &&
		capacidad.principal.Attributes["certificate_sha256"] == s.original.certificadoSHA256 {
		if _, valido := s.original.capacidadValida(ctx); valido {
			return s.original, true
		}
		return nil, false
	}
	if s.adicional != nil && rutaSegundoRRHHNoIncorporacion(capacidad.ruta, capacidad.metodo) &&
		capacidad.principal.ID == s.adicional.principalID &&
		capacidad.principal.Attributes["certificate_sha256"] == s.adicional.certificadoSHA256 {
		if _, valido := s.adicional.capacidadValida(ctx); valido {
			return s.adicional, true
		}
	}
	return nil, false
}

func rutaSegundoRRHHNoIncorporacion(ruta, metodo string) bool {
	return metodo == http.MethodPost &&
		(ruta == httpinterno.RutaSeguimientoCese || ruta == httpinterno.RutaNoIncorporaciones)
}

func (s seleccionSoporteSeguimientoRRHH) ResolverContextoAutorizacionAltaV3(ctx context.Context, solicitud ports.SolicitudResolverContextoAutorizacionAltaV3) (ports.ContextoAutorizacionAltaV3, error) {
	soporte, ok := s.elegir(ctx)
	if !ok {
		return ports.ContextoAutorizacionAltaV3{}, ports.ErrAutorizacionDenegada
	}
	return soporte.ResolverContextoAutorizacionAltaV3(ctx, solicitud)
}

// La hoja adicional no concede nada por sí sola. Solo se consume una
// asignación ya publicada al perfil sintético y al actor exactos: la semilla
// (solo organización) o la misma asignación ya ceñida por una operación
// anterior a un expediente en nombramiento, como hace la técnica original.
func instantaneaSegundoRRHHNoIncorporacionValida(i vecdomain.InstantaneaAutorizacion, principal, perfil string, ahora time.Time) bool {
	if i.Validar() != nil || i.AsignacionPerfil.PrincipalID != principal ||
		i.AsignacionPerfil.PerfilActivoRef != perfil ||
		i.AsignacionPerfil.Estado != vecdomain.EstadoAsignacionPerfilActiva ||
		!i.AsignacionPerfil.VigenteEn(ahora) ||
		i.ControlVigenciaVersionRol.Estado != vecdomain.EstadoControlVigenciaVersionRolHabilitada ||
		len(i.AsignacionPerfil.Ambitos) < 1 || len(i.AsignacionPerfil.Ambitos) > 4 {
		return false
	}
	ambitos := map[string]string{}
	for _, ambito := range i.AsignacionPerfil.Ambitos {
		if len(ambito.Valores) != 1 || ambitos[ambito.Clave] != "" {
			return false
		}
		ambitos[ambito.Clave] = ambito.Valores[0]
	}
	for clave, valor := range ambitos {
		switch clave {
		case "organizacion_ref":
			if valor != organizacionAltaContratacionTemporalDesarrollo {
				return false
			}
		case "expediente_ref":
			if !domain.ReferenciaOpacaValida(valor) {
				return false
			}
		case "fase_previa":
			if valor != string(domain.FaseNombramiento) {
				return false
			}
		case "estado_previo":
			if valor != string(domain.EstadoEnCurso) {
				return false
			}
		default:
			return false
		}
	}
	if ambitos["organizacion_ref"] == "" {
		return false
	}
	lectura, noIncorporacion := false, false
	for _, concesion := range i.VersionRol.Concesiones {
		if concesion.Accion == string(domain.AccionRegistrarNoIncorporacion) &&
			concesion.ModuloID == ports.ModuloContratacion &&
			concesion.TipoRecurso == ports.TipoRecursoNoIncorporacion {
			for _, finalidad := range concesion.Finalidades {
				if finalidad == ports.FinalidadRegistrarNoIncorporacion {
					noIncorporacion = true
				}
			}
		}
		if concesion.Accion == accionConsultarSeguimientoCeseDesarrollo &&
			concesion.ModuloID == ports.ModuloContratacion &&
			concesion.TipoRecurso == "seguimiento_contratacion_temporal" {
			for _, finalidad := range concesion.Finalidades {
				if finalidad == "gestionar_contratacion_temporal" {
					lectura = true
				}
			}
		}
	}
	return lectura && noIncorporacion
}

// baseSegundoRRHH devuelve la instantánea con el ámbito de organización: cada
// operación la vuelve a ceñir al expediente exacto antes de decidir.
func baseSegundoRRHH(i vecdomain.InstantaneaAutorizacion) vecdomain.InstantaneaAutorizacion {
	base := clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(i)
	base.AsignacionPerfil.Ambitos = []vecdomain.AmbitoPerfil{{Clave: "organizacion_ref",
		Valores: []string{organizacionAltaContratacionTemporalDesarrollo}}}
	return base
}

// La consulta no tiene consumidor que registre la decisión junto con un
// efecto. Publica primero la asignación ceñida al expediente y exige el
// registro V3 durable antes de permitir que el manejador entregue el estado.
// La confirmación c22 conserva su consumidor transaccional ya existente.
type registroLecturaSegundoRRHH struct {
	soporte *soporteAltaContratacionTemporalDesarrollo
	durable registroDecisionesAnalisisContratacionTemporalDesarrollo
}

func (r registroLecturaSegundoRRHH) RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(
	ctx context.Context, orden puertosvec.OrdenRegistroConcesionCandidataAutorizacionLigadaV3,
) (time.Time, error) {
	if r.soporte == nil || r.durable == nil {
		return time.Time{}, puertosvec.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible
	}
	capacidad, valida := r.soporte.capacidadValida(ctx)
	if !valida || !rutaSegundoRRHHNoIncorporacion(capacidad.ruta, capacidad.metodo) {
		return time.Time{}, puertosvec.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible
	}
	if capacidad.ruta == httpinterno.RutaNoIncorporaciones {
		return r.soporte.RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(ctx, orden)
	}
	datos, err := orden.Datos()
	solicitud, errSolicitud := datos.Solicitud.Datos()
	if err != nil || errSolicitud != nil || solicitud.Accion != accionConsultarSeguimientoCeseDesarrollo ||
		solicitud.Finalidad != "gestionar_contratacion_temporal" ||
		solicitud.ReferenciaMotivo != motivoSeguimientoCeseDesarrollo(httpinterno.RutaSeguimientoCese) {
		return time.Time{}, puertosvec.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible
	}
	if _, ok := r.soporte.ambitosSeguimientoCese(capacidad.ruta, solicitud); !ok {
		return time.Time{}, puertosvec.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible
	}
	instantanea, ok := r.soporte.instantaneaParaContexto(ctx, capacidad.ruta)
	if !ok || instantanea.Validar() != nil || r.soporte.autoridadAsignaciones == nil ||
		publicarInstantaneaAsignacionCTSegunRuta(ctx, capacidad.ruta, r.soporte.autoridadAsignaciones, instantanea) != nil {
		return time.Time{}, puertosvec.ErrInstantaneaAutorizacionObsoleta
	}
	return r.durable.RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(ctx, orden)
}

func (r registroLecturaSegundoRRHH) RegistrarDenegacionAutorizacionLigadaV3(
	ctx context.Context, orden puertosvec.OrdenRegistroDenegacionAutorizacionLigadaV3,
) error {
	if r.soporte == nil || r.durable == nil {
		return puertosvec.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible
	}
	capacidad, valida := r.soporte.capacidadValida(ctx)
	if !valida || !rutaSegundoRRHHNoIncorporacion(capacidad.ruta, capacidad.metodo) {
		return puertosvec.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible
	}
	if capacidad.ruta == httpinterno.RutaNoIncorporaciones {
		return r.soporte.RegistrarDenegacionAutorizacionLigadaV3(ctx, orden)
	}
	datos, err := orden.Datos()
	solicitud, errSolicitud := datos.Solicitud.Datos()
	if err != nil || errSolicitud != nil || solicitud.Accion != accionConsultarSeguimientoCeseDesarrollo ||
		solicitud.Finalidad != "gestionar_contratacion_temporal" ||
		solicitud.ReferenciaMotivo != motivoSeguimientoCeseDesarrollo(httpinterno.RutaSeguimientoCese) {
		return puertosvec.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible
	}
	return r.durable.RegistrarDenegacionAutorizacionLigadaV3(ctx, orden)
}

// Preparación local: publica solo ContextoActor sintético mediante el productor
// existente; cuenta/alias se preparan con el circuito nominal ya instalado.
// La asignación V3 debe estar publicada antes y nunca se crea aquí.
func componerSegundoSoporteNoIncorporacion(ctx context.Context, dependencias *DependenciasCT,
	alta *dependenciasAltaContratacionTemporalDesarrollo, reloj relojContratacionTemporalDesarrollo,
) (*soporteAltaContratacionTemporalDesarrollo, autorizadorLigadoContratacionTemporalDesarrollo) {
	if dependencias == nil || dependencias.resolvedor == nil || dependencias.resolvedor.segundoTecnicoRRHH == nil {
		return nil, nil
	}
	if alta == nil || alta.soporte == nil || alta.postgresql.gobierno == nil {
		avisoSegundoTecnicoRRHH("dependencias_alta")
		return nil, nil
	}
	base, ok := alta.soporte.sesionOperativa.(*proveedorSesionConsultaRRHHDesarrollo)
	if !ok || base == nil || base.resolutor == nil || base.registro == nil || base.revalidador == nil {
		slog.Warn("segundo técnico de RRHH sin sesión nominal; no se activa no incorporación")
		return nil, nil
	}
	principal := dependencias.resolvedor.segundoTecnicoRRHH.principal
	contexto, err := nuevoContextoSinteticoContratacionTemporalDesarrollo(principal, reloj.Ahora())
	if err != nil || contexto.Resultado.Validar() != nil {
		registrarFalloSegundoTecnicoRRHH("contexto_sintetico", err)
		return nil, nil
	}
	vinculo, err := contexto.Vinculo.Datos()
	if err != nil || vinculo.PrincipalID != contexto.Resultado.Contexto.PersonaRef {
		registrarFalloSegundoTecnicoRRHH("vinculo_contexto", err)
		return nil, nil
	}
	perfil := vinculo.PerfilActivoRef
	publicada, existe, err := leerInstantaneaPublicadaPostgreSQLDesarrollo(ctx, alta.postgresql.gobierno, perfil)
	if err != nil || !existe || publicada.actoAsignacion != actoAsignacionCTDesarrollo ||
		!instantaneaSegundoRRHHNoIncorporacionValida(publicada.instantanea, vinculo.PrincipalID, perfil, reloj.Ahora()) {
		slog.Warn("segundo técnico de RRHH sin asignación de lectura y no incorporación consumible")
		return nil, nil
	}
	alta.soporte.mu.Lock()
	registro := alta.soporte.registroDecisionesAnalisis
	alta.soporte.mu.Unlock()
	if registro == nil {
		avisoSegundoTecnicoRRHH("registro_decisiones")
		return nil, nil
	}
	soporte := &soporteAltaContratacionTemporalDesarrollo{
		sello: alta.soporte.sello, principalID: principal.ID,
		certificadoSHA256: principal.Attributes["certificate_sha256"],
		contexto:          contexto, reloj: reloj,
		seguimientoCese:            &soporteSeguimientoCeseDesarrollo{instantanea: baseSegundoRRHH(publicada.instantanea)},
		concesiones:                make(map[string]struct{}),
		instantaneasPorSolicitud:   make(map[string]vecdomain.InstantaneaAutorizacion),
		registroDecisionesAnalisis: registro,
	}
	// Igual que la técnica original y los lectores de RRHH: cada operación
	// ciñe la asignación ya sembrada al expediente exacto antes de decidir.
	soporte.autoridadAsignaciones = &autoridadPostgreSQLContratacionTemporalDesarrollo{pool: alta.postgresql.gobierno, soporte: soporte}
	if err := publicarResultadoContextoPostgreSQLDesarrollo(ctx, alta.postgresql.gobierno,
		contexto.Resultado, operacionContextoContratacionTemporalDesarrollo(soporte)); err != nil {
		slog.Warn("segundo técnico de RRHH sin contexto registrado; no se activa no incorporación")
		return nil, nil
	}
	derivador := dependencias.derivador
	if derivador == nil || !derivador.valido() {
		avisoSegundoTecnicoRRHH("derivador")
		return nil, nil
	}
	seudonimos, err := (&seudonimizadorSesionDesarrollo{derivador: derivador}).SeudonimizarAlta(ctx,
		postgresidentidad.IdentificadoresAlta{EspacioIdentidad: espacioIdentidadSesionDesarrollo,
			AsercionID: "preparacion-cuenta", SesionID: "preparacion-alias",
			CuentaID: "desarrollo:" + vinculo.CuentaRef, SujetoID: soporte.principalID})
	if err != nil || prepararCuentaNominalConsultasDesarrollo(ctx, alta.postgresql.gobierno, soporte, seudonimos) != nil {
		slog.Warn("segundo técnico de RRHH sin cuenta nominal; no se activa no incorporación")
		return nil, nil
	}
	esperado, err := contextoEsperadoRegistradoDesarrollo(ctx, base.resolutor, soporte)
	if err != nil {
		registrarFalloSegundoTecnicoRRHH("contexto_registrado", err)
		return nil, nil
	}
	soporte.contextoEsperadoRegistrado = esperado
	sesion, err := nuevoProveedorSesionConsultaRRHHConCatalogoDesarrollo(soporte, base.registro,
		base.revalidador, reloj, base.resolutor, base.fronteras)
	if err != nil {
		registrarFalloSegundoTecnicoRRHH("sesion_nominal", err)
		return nil, nil
	}
	soporte.sesionOperativa = sesion
	registroLectura := registroLecturaSegundoRRHH{soporte: soporte, durable: registro}
	autorizador, err := aplicacionvec.NuevoServicioAutorizacionSolicitudLigadaV3(soporte, registroLectura, registroLectura, soporte,
		reloj, seguridadvec.GeneradorReferenciasCriptograficas{},
		aplicacionvec.ConfiguracionServicioAutorizacion{VigenciaDecision: 90 * time.Second})
	if err != nil {
		registrarFalloSegundoTecnicoRRHH("autorizador", err)
		return nil, nil
	}
	return soporte, autorizador
}

// avisoSegundoTecnicoRRHH deja en el registro técnico la etapa que impide
// activar la segunda hoja, sin datos personales ni detalles internos.
func avisoSegundoTecnicoRRHH(etapa string) {
	slog.Warn("segundo técnico de RRHH no activado; se conserva el técnico original", "etapa", etapa)
}

// registrarFalloSegundoTecnicoRRHH registra además el error que impide activarlo.
func registrarFalloSegundoTecnicoRRHH(etapa string, err error) {
	slog.Warn("segundo técnico de RRHH no activado; se conserva el técnico original", "etapa", etapa, "error", err)
}
