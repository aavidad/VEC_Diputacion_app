package bootstrap

import (
	"bytes"
	"context"
	"net/http"
	"strings"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const expedienteContratacionTemporalDesarrolloRef = "expediente:ct:demo:0001"

type selloConsultasContratacionTemporalDesarrollo struct{}

type claveCapacidadConsultasContratacionTemporalDesarrollo struct{}

type capacidadConsultaContratacionTemporalDesarrollo struct {
	sello *selloConsultasContratacionTemporalDesarrollo
	ruta  string
}

// autoridadConsultasContratacionTemporalDesarrollo solo reconoce capacidades
// efimeras emitidas tras revalidar el certificado mTLS local. No representa
// autoridad corporativa ni se construye fuera del perfil de desarrollo.
type autoridadConsultasContratacionTemporalDesarrollo struct {
	sello      *selloConsultasContratacionTemporalDesarrollo
	resolvedor *resolvedorIdentidadDesarrollo
}

func nuevasRutasConsultasContratacionTemporalDesarrollo(
	cfg config.Config,
	resolvedor vechttp.DemoIdentityResolver,
) (
	[]vechttp.RutaExacta,
	*autoridadConsultasContratacionTemporalDesarrollo,
	error,
) {
	cfg = cfg.Normalize()
	resolvedorDesarrollo, esDesarrollo := resolvedor.(*resolvedorIdentidadDesarrollo)
	if !cfg.DevelopmentEnabledByDoubleKey() || validarRedLocalDesarrollo(cfg) != nil ||
		!esDesarrollo || resolvedorDesarrollo == nil {
		return nil, nil, ErrActivacionDesarrolloInvalida
	}
	origen := nuevoOrigenConsultasContratacionTemporalDesarrollo()
	cuadro, err := httpinterno.NuevoManejadorConsultaCuadroRRHH(
		&consultorCuadroContratacionTemporalDesarrollo{origen: origen},
	)
	if err != nil {
		return nil, nil, err
	}
	detalle, err := httpinterno.NuevoManejadorConsultaDetalleRRHH(
		&consultorDetalleContratacionTemporalDesarrollo{origen: origen},
	)
	if err != nil {
		return nil, nil, err
	}
	autoridad := &autoridadConsultasContratacionTemporalDesarrollo{
		sello:      &selloConsultasContratacionTemporalDesarrollo{},
		resolvedor: resolvedorDesarrollo,
	}
	return []vechttp.RutaExacta{
		{Ruta: httpinterno.RutaConsultaCuadroRRHH, Manejador: cuadro},
		{Ruta: httpinterno.RutaConsultaDetalleRRHH, Manejador: detalle},
	}, autoridad, nil
}

func (a *autoridadConsultasContratacionTemporalDesarrollo) proteger(
	siguiente http.Handler,
) http.Handler {
	return &revalidadorConsultasContratacionTemporalDesarrollo{
		siguiente: siguiente,
		autoridad: a,
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
	if capacidad.sello != a.sello || capacidad.ruta != ruta {
		return vechttp.ErrAccesoRutaExactaDenegado
	}
	return nil
}

type revalidadorConsultasContratacionTemporalDesarrollo struct {
	siguiente http.Handler
	autoridad *autoridadConsultasContratacionTemporalDesarrollo
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
	if !esRutaConsultaContratacionTemporalDesarrollo(r) {
		m.siguiente.ServeHTTP(w, r)
		return
	}
	principal, err := m.autoridad.resolvedor.ResolveDemoIdentity(
		r.Context(), peticionIdentidadConsultasContratacionTemporalDesarrollo(r),
	)
	if err == nil && principalContratacionTemporalDesarrolloValido(principal) {
		capacidad := capacidadConsultaContratacionTemporalDesarrollo{
			sello: m.autoridad.sello,
			ruta:  r.URL.Path,
		}
		r = r.WithContext(context.WithValue(
			r.Context(),
			claveCapacidadConsultasContratacionTemporalDesarrollo{},
			capacidad,
		))
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

func esRutaConsultaContratacionTemporalDesarrollo(r *http.Request) bool {
	if r == nil || r.URL == nil {
		return false
	}
	return r.URL.Path == httpinterno.RutaConsultaCuadroRRHH ||
		r.URL.Path == httpinterno.RutaConsultaDetalleRRHH
}

func principalContratacionTemporalDesarrolloValido(
	principal vecdomain.Principal,
) bool {
	return principal.Validate() == nil &&
		principal.AuthMethod == vecdomain.AuthMethodCertificate &&
		principal.AuthAssurance == vecdomain.AuthAssuranceHigh &&
		principal.Attributes["autoridad"] == AutoridadNoAutoritativa &&
		principal.Attributes["perfil_ejecucion"] == config.ExecutionProfileDevelopment
}

// origenConsultasContratacionTemporalDesarrollo es una fuente efimera,
// sintetica y no autoritativa. Solo satisface los puertos de lectura existentes
// para que la interfaz pueda demostrar el cuadro y su detalle.
type origenConsultasContratacionTemporalDesarrollo struct {
	pagina  ports.PaginaCuadroRRHH
	detalle ports.DetalleExpedienteRRHH
}

func nuevoOrigenConsultasContratacionTemporalDesarrollo() *origenConsultasContratacionTemporalDesarrollo {
	creadoEn := time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)
	actualizadoEn := time.Date(2026, 9, 2, 9, 30, 0, 0, time.UTC)
	resumen := ports.ResumenExpedienteRRHH{
		ExpedienteRef:   expedienteContratacionTemporalDesarrolloRef,
		OrganizacionRef: "organizacion:desarrollo:dipgra",
		NumeroVisible:   "2026/CT-0001",
		Version:         3,
		FlujoRef:        "flujo:ct:desarrollo",
		FlujoVersion:    1,
		FlujoHuella:     strings.Repeat("d", 64),
		FaseClave:       domain.ClaveFase("analisis"),
		EstadoClave:     domain.EstadoEnCurso,
		CentroRef:       "centro:desarrollo:001",
		CategoriaRef:    "categoria:desarrollo:c2",
		ModalidadClave:  domain.ClaveCatalogo("interinidad"),
		CreadoEn:        creadoEn,
		ActualizadoEn:   actualizadoEn,
	}
	resumenDetalle := resumen
	resumenDetalle.ModalidadClave = ""
	detalle := ports.DetalleExpedienteRRHH{
		Resumen: resumenDetalle,
		Solicitud: ports.SolicitudOperativaRRHH{
			GrupoSubgrupo: "C2",
			MotivoClave:   domain.ClaveCatalogo("sustitucion"),
			PeriodoInicio: time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC),
			PeriodoFin:    time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		},
		Hitos: []ports.HitoExpedienteRRHH{
			{
				Secuencia: 1, VersionExpediente: 1,
				AccionClave:   domain.ClaveCatalogo("alta"),
				RealizadaEn:   creadoEn,
				FaseDestino:   domain.ClaveFase("solicitud"),
				EstadoOrigen:  domain.EstadoPendiente,
				EstadoDestino: domain.EstadoEnCurso,
			},
			{
				Secuencia: 2, VersionExpediente: 2,
				AccionClave:   domain.ClaveCatalogo("analizar"),
				RealizadaEn:   creadoEn.Add(time.Hour),
				FaseOrigen:    domain.ClaveFase("solicitud"),
				FaseDestino:   domain.ClaveFase("analisis"),
				EstadoOrigen:  domain.EstadoEnCurso,
				EstadoDestino: domain.EstadoEnCurso,
			},
			{
				Secuencia: 3, VersionExpediente: 3,
				AccionClave:   domain.ClaveCatalogo("actualizar"),
				RealizadaEn:   actualizadoEn,
				FaseOrigen:    domain.ClaveFase("analisis"),
				FaseDestino:   domain.ClaveFase("analisis"),
				EstadoOrigen:  domain.EstadoEnCurso,
				EstadoDestino: domain.EstadoEnCurso,
			},
		},
	}
	return &origenConsultasContratacionTemporalDesarrollo{
		pagina: ports.PaginaCuadroRRHH{
			GeneradaEn:  actualizadoEn.Add(time.Minute),
			Expedientes: []ports.ResumenExpedienteRRHH{resumen},
		},
		detalle: detalle,
	}
}

type consultorCuadroContratacionTemporalDesarrollo struct {
	origen *origenConsultasContratacionTemporalDesarrollo
}

func (c *consultorCuadroContratacionTemporalDesarrollo) Consultar(
	ctx context.Context,
	solicitud ports.SolicitudCuadroRRHH,
) (ports.PaginaCuadroRRHH, error) {
	if ctx == nil {
		return ports.PaginaCuadroRRHH{}, application.ErrSolicitudConsultaRRHHInvalida
	}
	if err := ctx.Err(); err != nil {
		return ports.PaginaCuadroRRHH{}, err
	}
	if c == nil || c.origen == nil || len(c.origen.pagina.Expedientes) != 1 {
		return ports.PaginaCuadroRRHH{}, application.ErrConsultaRRHHNoDisponible
	}
	resumen := c.origen.pagina.Expedientes[0]
	coincide := solicitud.Cursor() == "" &&
		(solicitud.Texto() == "" || strings.HasPrefix(resumen.NumeroVisible, solicitud.Texto())) &&
		(solicitud.EstadoClave() == "" || solicitud.EstadoClave() == resumen.EstadoClave) &&
		(solicitud.FaseClave() == "" || solicitud.FaseClave() == resumen.FaseClave)
	expedientes := make([]ports.ResumenExpedienteRRHH, 0, 1)
	if coincide {
		expedientes = append(expedientes, resumen)
	}
	return ports.PaginaCuadroRRHH{
		GeneradaEn:  c.origen.pagina.GeneradaEn,
		Expedientes: expedientes,
	}, nil
}

type consultorDetalleContratacionTemporalDesarrollo struct {
	origen *origenConsultasContratacionTemporalDesarrollo
}

func (c *consultorDetalleContratacionTemporalDesarrollo) Consultar(
	ctx context.Context,
	solicitud ports.SolicitudDetalleRRHH,
) (ports.DetalleExpedienteRRHH, error) {
	if ctx == nil {
		return ports.DetalleExpedienteRRHH{}, application.ErrSolicitudConsultaRRHHInvalida
	}
	if err := ctx.Err(); err != nil {
		return ports.DetalleExpedienteRRHH{}, err
	}
	if c == nil || c.origen == nil {
		return ports.DetalleExpedienteRRHH{}, application.ErrConsultaRRHHNoDisponible
	}
	if solicitud.ExpedienteRef() != c.origen.detalle.Resumen.ExpedienteRef ||
		solicitud.VersionObservada() != 0 &&
			solicitud.VersionObservada() != c.origen.detalle.Resumen.Version {
		return ports.DetalleExpedienteRRHH{}, application.ErrConsultaRRHHNoObservable
	}
	return c.origen.detalle.Clonar(), nil
}
