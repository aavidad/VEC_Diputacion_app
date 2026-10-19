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
)

// seleccionSoporteSeguimientoRRHH elige desde la capacidad sellada por la
// frontera mTLS. El segundo técnico sólo alcanza la no incorporación.
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
	if s.adicional != nil && capacidad.ruta == httpinterno.RutaNoIncorporaciones && capacidad.metodo == http.MethodPost &&
		capacidad.principal.ID == s.adicional.principalID &&
		capacidad.principal.Attributes["certificate_sha256"] == s.adicional.certificadoSHA256 {
		if _, valido := s.adicional.capacidadValida(ctx); valido {
			return s.adicional, true
		}
	}
	return nil, false
}

func (s seleccionSoporteSeguimientoRRHH) ResolverContextoAutorizacionAltaV3(ctx context.Context, solicitud ports.SolicitudResolverContextoAutorizacionAltaV3) (ports.ContextoAutorizacionAltaV3, error) {
	soporte, ok := s.elegir(ctx)
	if !ok {
		return ports.ContextoAutorizacionAltaV3{}, ports.ErrAutorizacionDenegada
	}
	return soporte.ResolverContextoAutorizacionAltaV3(ctx, solicitud)
}

// La hoja adicional no concede nada por sí sola. Solo se consume una
// asignación ya publicada al perfil sintético y al actor exactos.
func instantaneaSegundoRRHHNoIncorporacionValida(i vecdomain.InstantaneaAutorizacion, principal, perfil string, ahora time.Time) bool {
	if i.Validar() != nil || i.AsignacionPerfil.PrincipalID != principal ||
		i.AsignacionPerfil.PerfilActivoRef != perfil ||
		i.AsignacionPerfil.Estado != vecdomain.EstadoAsignacionPerfilActiva ||
		!i.AsignacionPerfil.VigenteEn(ahora) ||
		i.ControlVigenciaVersionRol.Estado != vecdomain.EstadoControlVigenciaVersionRolHabilitada ||
		len(i.AsignacionPerfil.Ambitos) < 1 || len(i.AsignacionPerfil.Ambitos) > 2 {
		return false
	}
	ambitos := map[string]string{}
	for _, ambito := range i.AsignacionPerfil.Ambitos {
		if len(ambito.Valores) != 1 || ambitos[ambito.Clave] != "" {
			return false
		}
		ambitos[ambito.Clave] = ambito.Valores[0]
	}
	if ambitos["organizacion_ref"] != organizacionAltaContratacionTemporalDesarrollo ||
		(len(ambitos) == 2 && !domain.ReferenciaOpacaValida(ambitos["expediente_ref"])) {
		return false
	}
	for _, concesion := range i.VersionRol.Concesiones {
		if concesion.Accion == string(domain.AccionRegistrarNoIncorporacion) &&
			concesion.ModuloID == ports.ModuloContratacion &&
			concesion.TipoRecurso == ports.TipoRecursoNoIncorporacion {
			for _, finalidad := range concesion.Finalidades {
				if finalidad == ports.FinalidadRegistrarNoIncorporacion {
					return true
				}
			}
		}
	}
	return false
}

func expedienteAdmitidoSegundoRRHH(i vecdomain.InstantaneaAutorizacion, expediente string) bool {
	for _, ambito := range i.AsignacionPerfil.Ambitos {
		if ambito.Clave == "expediente_ref" {
			return len(ambito.Valores) == 1 && ambito.Valores[0] == expediente
		}
	}
	return true
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
		avisoSegundoTecnicoRRHH("contexto_sintetico")
		return nil, nil
	}
	vinculo, err := contexto.Vinculo.Datos()
	if err != nil || vinculo.PrincipalID != contexto.Resultado.Contexto.PersonaRef {
		avisoSegundoTecnicoRRHH("vinculo_contexto")
		return nil, nil
	}
	perfil := vinculo.PerfilActivoRef
	publicada, existe, err := leerInstantaneaPublicadaPostgreSQLDesarrollo(ctx, alta.postgresql.gobierno, perfil)
	if err != nil || !existe || !instantaneaSegundoRRHHNoIncorporacionValida(publicada.instantanea, vinculo.PrincipalID, perfil, reloj.Ahora()) {
		slog.Warn("segundo técnico de RRHH sin asignación c22 consumible; no se activa no incorporación")
		return nil, nil
	}
	soporte := &soporteAltaContratacionTemporalDesarrollo{
		sello: alta.soporte.sello, principalID: principal.ID,
		certificadoSHA256: principal.Attributes["certificate_sha256"],
		contexto:          contexto, reloj: reloj,
		seguimientoCese:          &soporteSeguimientoCeseDesarrollo{instantanea: publicada.instantanea},
		concesiones:              make(map[string]struct{}),
		instantaneasPorSolicitud: make(map[string]vecdomain.InstantaneaAutorizacion),
	}
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
		avisoSegundoTecnicoRRHH("contexto_registrado")
		return nil, nil
	}
	soporte.contextoEsperadoRegistrado = esperado
	sesion, err := nuevoProveedorSesionConsultaRRHHConCatalogoDesarrollo(soporte, base.registro,
		base.revalidador, reloj, base.resolutor, base.fronteras)
	if err != nil {
		avisoSegundoTecnicoRRHH("sesion_nominal")
		return nil, nil
	}
	soporte.sesionOperativa = sesion
	autorizador, err := aplicacionvec.NuevoServicioAutorizacionSolicitudLigadaV3(soporte, soporte, soporte, soporte,
		reloj, seguridadvec.GeneradorReferenciasCriptograficas{},
		aplicacionvec.ConfiguracionServicioAutorizacion{VigenciaDecision: 90 * time.Second})
	if err != nil {
		avisoSegundoTecnicoRRHH("autorizador")
		return nil, nil
	}
	return soporte, autorizador
}

// avisoSegundoTecnicoRRHH deja en el registro técnico la etapa que impide
// activar la segunda hoja, sin datos personales ni detalles internos.
func avisoSegundoTecnicoRRHH(etapa string) {
	slog.Warn("segundo técnico de RRHH no activado; se conserva el técnico original", "etapa", etapa)
}
