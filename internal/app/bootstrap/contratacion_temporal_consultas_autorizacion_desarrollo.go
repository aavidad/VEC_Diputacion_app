package bootstrap

import (
	"bytes"
	"context"
	"net/http"
	"time"
	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

func (a *autoridadConsultasContratacionTemporalDesarrollo) proteger(
	siguiente http.Handler,
) http.Handler {
	fronteras := catalogoFronterasComunDesarrollo{}
	if a != nil && a.fronterasSeguridadComun.identidad != nil {
		fronteras = a.fronterasSeguridadComun
	}
	protegido := a.protegerConCatalogoSeguridadComun(siguiente, fronteras)
	if a != nil && a.envolverBorradorLlamamiento != nil {
		return a.envolverBorradorLlamamiento(protegido)
	}
	return protegido
}

func (a *autoridadConsultasContratacionTemporalDesarrollo) protegerConCatalogoSeguridadComun(siguiente http.Handler, fronteras catalogoFronterasComunDesarrollo) http.Handler {
	return &revalidadorConsultasContratacionTemporalDesarrollo{
		siguiente: siguiente,
		autoridad: a,
		fronteras: fronteras,
	}
}

func (a *autoridadConsultasContratacionTemporalDesarrollo) AutorizarRutaExacta(
	ctx context.Context,
	ruta string,
) error {
	if a == nil || a.sello == nil || a.resolvedor == nil || ctx == nil ||
		ctx.Err() != nil {
		return vechttp.ErrAutoridadRutaExactaNoDisponible
	}
	capacidad, existe := ctx.Value(
		claveCapacidadConsultasContratacionTemporalDesarrollo{},
	).(capacidadConsultaContratacionTemporalDesarrollo)
	if !existe {
		return vechttp.ErrAutenticacionRutaExactaRequerida
	}
	frontera, adicional := fronteraSeguridadComunDesdeContexto(ctx)
	if capacidad.sello != a.sello || capacidad.ruta != ruta || (adicional && frontera.ruta != ruta) {
		return vechttp.ErrAccesoRutaExactaDenegado
	}
	if a.noCompuesta != nil && a.noCompuesta.esRuta(ruta) &&
		!((a.llamamientoCompuesto && (ruta == httpinterno.RutaSeleccionLlamamiento || ruta == httpinterno.RutaPropuestaFormalizacion)) ||
			(a.consultasRRHHCompuestas && rutaConsultaRRHHContratacionTemporalDesarrollo(ruta)) ||
			(a.subsanacionCompuesta && ruta == httpinterno.RutaSubsanacionReparos)) {
		return a.noCompuesta.denegarRuta(ctx, ruta)
	}
	return nil
}

type revalidadorConsultasContratacionTemporalDesarrollo struct {
	siguiente http.Handler
	autoridad *autoridadConsultasContratacionTemporalDesarrollo
	fronteras catalogoFronterasComunDesarrollo
}

func (m *revalidadorConsultasContratacionTemporalDesarrollo) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if m == nil || m.siguiente == nil || m.autoridad == nil ||
		m.autoridad.sello == nil || m.autoridad.resolvedor == nil {
		http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		return
	}
	r = peticionIdentidadConsultasContratacionTemporalDesarrollo(r)
	protegidaCT := esRutaContratacionTemporalDesarrollo(r)
	fronteraComun, protegidaComun := m.fronteras.resolver(r.Method, r.URL.Path)
	if !protegidaCT && !protegidaComun {
		m.siguiente.ServeHTTP(w, r)
		return
	}
	principal, err := m.autoridad.resolvedor.ResolveDemoIdentity(
		r.Context(), r,
	)
	validoRuta := principalContratacionTemporalDesarrolloValidoParaRuta(principal, r.URL.Path)
	if protegidaComun && !protegidaCT {
		validoRuta = principalContratacionTemporalDesarrolloValido(principal)
	}
	if (rutaConsultaRRHHContratacionTemporalDesarrollo(r.URL.Path) || r.URL.Path == httpinterno.RutaEstadisticasRRHH) && len(m.autoridad.resolvedor.lectoresRRHH) != 0 {
		_, validoRuta = m.autoridad.resolvedor.lectorConsultaRRHH(principal)
	}
	if err == nil && validoRuta {
		capacidad := capacidadConsultaContratacionTemporalDesarrollo{
			sello:     m.autoridad.sello,
			ruta:      r.URL.Path,
			principal: clonarPrincipalDesarrollo(principal),
		}
		if protegidaComun || rutaContinuidadNominal(capacidad.ruta) || rutaConsultaRRHHContratacionTemporalDesarrollo(capacidad.ruta) ||
			capacidad.ruta == httpinterno.RutaIncorporacionEjercicioV2 ||
			capacidad.ruta == httpinterno.RutaFichaGINPIXV2 ||
			capacidad.ruta == httpinterno.RutaConsultaSeguimientoV2 ||
			capacidad.ruta == httpinterno.RutaResolucionFormalizacion ||
			capacidad.ruta == httpinterno.RutaSubsanacionReparos ||
			capacidad.ruta == httpinterno.RutaEstadisticasRRHH ||
			capacidad.ruta == rutaEntregaPeticionCentro ||
			rutaPeticionCentroDesarrollo(capacidad.ruta) ||
			capacidad.ruta == rutaOrganizacionContratacionTemporalDesarrollo ||
			capacidad.ruta == rutaCambiosOrganizacionContratacionTemporalDesarrollo ||
			capacidad.ruta == rutaBolsasRRHHDesarrollo || rutaBolsasCandidatosRRHHDesarrollo(capacidad.ruta) ||
			capacidad.ruta == rutaEstadisticasBolsaRRHHDesarrollo ||
			capacidad.ruta == rutaAvisosBolsaRRHHDesarrollo {
			// El resolvedor ya ha cotejado la hoja y su cadena mTLS. Revalidar
			// aquí su ventana también cubre conexiones abiertas antes de caducar.
			certificado := r.TLS.VerifiedChains[0][0]
			observado := time.Now().UTC().Truncate(time.Microsecond)
			if observado.Before(certificado.NotBefore) || !observado.Before(certificado.NotAfter) {
				m.siguiente.ServeHTTP(w, r)
				return
			}
			capacidad.certificadoVerificadoEn = observado
			capacidad.certificadoValidoHasta = certificado.NotAfter.UTC()
			capacidad.consultaRRHH = &contextoConsultaRRHHPeticionDesarrollo{}
			if rutaConsultaRRHHContratacionTemporalDesarrollo(capacidad.ruta) {
				vinculo, ok := vinculoCanalTLSCursorRRHHDesarrollo(r.TLS)
				if !ok {
					m.siguiente.ServeHTTP(w, r)
					return
				}
				capacidad.vinculoCanalTLS = vinculo
			}
		}
		ctx := context.WithValue(
			r.Context(),
			claveCapacidadConsultasContratacionTemporalDesarrollo{},
			capacidad,
		)
		ctx = context.WithValue(ctx, claveCacheSeguridadComunDesarrollo{}, &cacheSeguridadComunDesarrollo{})
		if protegidaComun {
			ctx = context.WithValue(ctx, claveFronteraSeguridadComunDesarrollo{}, fronteraSeguridadComunDesarrollo{metodo: r.Method, ruta: r.URL.Path, superficie: superficieInternaSeguridadComunDesarrollo, catalogo: m.fronteras, descriptor: fronteraComun})
		}
		r = r.WithContext(ctx)
	} else if err == nil && (protegidaCT || protegidaComun) && principalSinteticoContratacionTemporalDesarrolloValido(principal) {
		// Una identidad mTLS válida pero sin el perfil/ámbito de esta ruta está
		// autenticada. Se conserva una capacidad deliberadamente no sellada para
		// que la autoridad exacta responda 403, no el 401 de identidad ausente.
		ctx := context.WithValue(r.Context(), claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidadConsultaContratacionTemporalDesarrollo{ruta: r.URL.Path, principal: clonarPrincipalDesarrollo(principal)})
		r = r.WithContext(ctx)
	}
	m.siguiente.ServeHTTP(w, r)
}

// peticionIdentidadConsultasContratacionTemporalDesarrollo conserva la hoja
// que identifica al cliente cuando la libreria TLS entrega tambien su cadena.
// Solo normaliza certificados que ya pertenecen, en el mismo orden, a la unica
// cadena verificada por mTLS; una cadena ambigua o con extras sigue llegando al
// resolvedor sin cambios y este la rechaza.
func peticionIdentidadConsultasContratacionTemporalDesarrollo(
	peticion *http.Request,
) *http.Request {
	if peticion == nil || peticion.TLS == nil || len(peticion.TLS.PeerCertificates) <= 1 ||
		len(peticion.TLS.VerifiedChains) != 1 ||
		len(peticion.TLS.PeerCertificates) > len(peticion.TLS.VerifiedChains[0]) {
		return peticion
	}
	cadenaVerificada := peticion.TLS.VerifiedChains[0]
	for indice, certificado := range peticion.TLS.PeerCertificates {
		if certificado == nil || cadenaVerificada[indice] == nil ||
			!bytes.Equal(certificado.Raw, cadenaVerificada[indice].Raw) {
			return peticion
		}
	}
	copia := new(http.Request)
	*copia = *peticion
	estadoTLS := *peticion.TLS
	estadoTLS.PeerCertificates = estadoTLS.PeerCertificates[:1:1]
	copia.TLS = &estadoTLS
	return copia
}

func principalContratacionTemporalDesarrolloValido(
	principal vecdomain.Principal,
) bool {
	return principalSinteticoContratacionTemporalDesarrolloValido(principal) &&
		len(principal.Roles) == 1 && principal.Roles[0] == rolTecnicoRRHHContratacionTemporalDesarrollo
}

func principalIntervencionContratacionTemporalDesarrolloValido(
	principal vecdomain.Principal,
) bool {
	return principalSinteticoContratacionTemporalDesarrolloValido(principal) &&
		len(principal.Roles) == 1 && principal.Roles[0] == rolIntervencionContratacionTemporalDesarrollo
}

func principalContratacionTemporalDesarrolloValidoParaRuta(
	principal vecdomain.Principal,
	ruta string,
) bool {
	if rutaPeticionCentroDesarrollo(ruta) {
		return principalPeticionCentroDesarrolloValido(principal)
	}
	if ruta == httpinterno.RutaResultadosFiscalizacion {
		return principalIntervencionContratacionTemporalDesarrolloValido(principal)
	}
	return principalContratacionTemporalDesarrolloValido(principal)
}

func principalSinteticoContratacionTemporalDesarrolloValido(
	principal vecdomain.Principal,
) bool {
	return principal.Validate() == nil &&
		principal.AuthMethod == vecdomain.AuthMethodCertificate &&
		principal.AuthAssurance == vecdomain.AuthAssuranceHigh &&
		len(principal.Permissions) == 0 &&
		principal.Attributes["autoridad"] == AutoridadNoAutoritativa &&
		principal.Attributes["perfil_ejecucion"] == config.ExecutionProfileDevelopment
}

// origenConsultasContratacionTemporalDesarrollo es una fuente efimera,
// sintetica y no autoritativa. Solo satisface los puertos de lectura existentes
// para que la interfaz pueda demostrar el cuadro y su detalle.
