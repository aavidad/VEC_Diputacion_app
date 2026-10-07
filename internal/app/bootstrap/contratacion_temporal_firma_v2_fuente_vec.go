package bootstrap

import (
	"context"
	"net/http"
	"sync"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/firmaemisorv2"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application/consultafirmasv2"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

// El contenedor pertenece a una sola petición. No se serializa ni admite
// material del cliente como selección de persona, cuenta o perfil.
type claveCompetenciaFirmaVecV2 struct{}

type contenedorCompetenciaFirmaVecV2 struct {
	peticion  *http.Request
	mu        sync.Mutex
	evidencia ports.EvidenciaCompetenciaFirmante
	presente  bool
	ambigua   bool
	capsula   *capsulaSesionFirmanteV2
	fallida   bool
}

// PrepararPeticionFirmaVecV2 lo usa exclusivamente el montaje de la ruta
// registro-vec. La frontera mTLS se acredita después con la autoridad 3a;
// esta preparación sola no concede acceso.
func prepararPeticionFirmaVecV2(r *http.Request) (*http.Request, error) {
	if r == nil || r.URL == nil || r.Context().Err() != nil || r.Method != http.MethodPost ||
		r.URL.Path != httpinterno.RutaRegistroFirmaVec || r.URL.RawPath != "" || r.URL.RawQuery != "" ||
		r.URL.ForceQuery || r.URL.EscapedPath() != r.URL.Path {
		return nil, ports.ErrFirmaDocumentoDenegada
	}
	r = peticionIdentidadConsultasContratacionTemporalDesarrollo(r)
	c := &contenedorCompetenciaFirmaVecV2{}
	peticion := r.WithContext(context.WithValue(r.Context(), claveCompetenciaFirmaVecV2{}, c))
	c.peticion = peticion
	return peticion, nil
}

func competenciaFirmaVecV2DesdeContexto(ctx context.Context) (*contenedorCompetenciaFirmaVecV2, bool) {
	if ctx == nil || ctx.Err() != nil {
		return nil, false
	}
	c, ok := ctx.Value(claveCompetenciaFirmaVecV2{}).(*contenedorCompetenciaFirmaVecV2)
	return c, ok && c != nil && c.peticion != nil
}

func (c *contenedorCompetenciaFirmaVecV2) guardar(e ports.EvidenciaCompetenciaFirmante) error {
	if c == nil {
		return ports.ErrCompetenciaFirmanteNoAcreditada
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ambigua || (c.presente && c.evidencia != e) {
		c.ambigua = true
		return ports.ErrCompetenciaFirmanteNoDisponible
	}
	c.evidencia, c.presente = e, true
	return nil
}

func (c *contenedorCompetenciaFirmaVecV2) leer() (ports.EvidenciaCompetenciaFirmante, *http.Request, bool) {
	if c == nil {
		return ports.EvidenciaCompetenciaFirmante{}, nil, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.presente || c.ambigua || c.peticion == nil {
		return ports.EvidenciaCompetenciaFirmante{}, nil, false
	}
	return c.evidencia, c.peticion, true
}

func (c *contenedorCompetenciaFirmaVecV2) resolver(ctx context.Context, a *autoridadSesionFirmanteV2) (firmaemisorv2.ContextoActorFirmaV2, error) {
	var cero firmaemisorv2.ContextoActorFirmaV2
	if c == nil || a == nil || ctx == nil || ctx.Err() != nil {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.presente || c.ambigua || c.fallida || c.peticion == nil || c.peticion.Context().Err() != nil {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	var vinculo core.VinculoAutenticacionActorV2
	var resultado core.ResultadoContextoActorRegistradoV2
	var err error
	if c.capsula == nil {
		e := c.evidencia
		seleccion := ports.SeleccionFirmanteV2{PersonaRef: e.FirmantePrincipalRef,
			CuentaRef: e.CuentaFirmanteRef, PerfilActivoRef: e.PerfilActivoFirmanteRef, RolID: e.RolIDFirmante,
			VinculoCertificado: ports.ReferenciaVersionadaFirmanteV2{Referencia: e.VinculoCredencialFirmanteRef,
				Version: e.VinculoCredencialFirmanteRevision, HuellaSHA256: e.VinculoCredencialFirmanteHuella}}
		c.capsula, err = a.acreditar(c.peticion, seleccion, e.FirmantePrincipalRef, e.Solicitud.CertificadoHuella)
		if err == nil {
			vinculo, resultado, err = a.abrirConContexto(ctx, c.peticion, c.capsula)
		}
	} else {
		vinculo, resultado, err = a.revalidarConContexto(ctx, c.peticion, c.capsula)
	}
	if err != nil {
		c.fallida = true
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	return firmaemisorv2.ContextoActorFirmaV2{Vinculo: vinculo, Resultado: resultado,
		CertificadoCanalSHA256: c.evidencia.Solicitud.CertificadoHuella}, nil
}

// La fuente nominal consume sólo el contenedor sellado de registro-vec. Cada
// invocación del emisor vuelve a consultar la sesión común mediante 3a.
type fuenteNominalFirmaVecV2 struct {
	autoridad *autoridadSesionFirmanteV2
}

var _ firmaemisorv2.FuenteContextoActorFirmaV2 = (*fuenteNominalFirmaVecV2)(nil)
var _ consultafirmasv2.FuenteContexto = (*fuenteNominalFirmaVecV2)(nil)

func (f *fuenteNominalFirmaVecV2) RevalidarContextoActorFirmaV2(ctx context.Context) (firmaemisorv2.ContextoActorFirmaV2, error) {
	var cero firmaemisorv2.ContextoActorFirmaV2
	if f == nil || f.autoridad == nil {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	c, ok := competenciaFirmaVecV2DesdeContexto(ctx)
	if !ok {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	return c.resolver(ctx, f.autoridad)
}

func (f *fuenteNominalFirmaVecV2) ResolverContextoConsultaFirmasR5V2(ctx context.Context) (consultafirmasv2.Contexto, error) {
	actor, err := f.RevalidarContextoActorFirmaV2(ctx)
	if err != nil {
		return consultafirmasv2.Contexto{}, err
	}
	c, ok := competenciaFirmaVecV2DesdeContexto(ctx)
	e, _, presente := c.leer()
	if !ok || !presente || !domain.ReferenciaOpacaValida(e.Solicitud.OrganizacionRef) {
		return consultafirmasv2.Contexto{}, ports.ErrFirmaDocumentoDenegada
	}
	return consultafirmasv2.Contexto{OrganizacionRef: e.Solicitud.OrganizacionRef,
		FirmantePrincipalCandidatoRef: actor.Resultado.Contexto.PersonaRef}, nil
}

// Sólo los ámbitos publicados del perfil activo llegan al emisor. El PDP
// decide cada acción y CT176 vuelve a comprobarla al confirmar la firma.
func nuevoEmisorFirmaVecV2CT(a *autoridadSesionFirmanteV2,
	consulta, firmaVec *emisorMaterialRenovableCTDesarrollo, autorizacion vp.FuenteAutorizacion,
	reloj vp.Reloj, motivo core.ReferenciaEntradaCatalogo,
) (*firmaemisorv2.Emisor, *fuenteNominalFirmaVecV2, error) {
	if a == nil || consulta == nil || firmaVec == nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(autorizacion) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(reloj) || !core.ReferenciaMotivoAutorizacionV2Valida(motivo) {
		return nil, nil, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	fuente := &fuenteNominalFirmaVecV2{autoridad: a}
	emisores := &emisorFirmasR5V2CTDesarrollo{porAccion: map[string]*emisorMaterialRenovableCTDesarrollo{
		ports.AccionConsultarFirmasR5V2: consulta, ports.AccionRegistrarFirmaVec: firmaVec,
	}}
	emisor, err := firmaemisorv2.NuevoEmisorConAmbitos(fuente, emisores, motivo, reloj, autorizacion)
	if err != nil {
		return nil, nil, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	return emisor, fuente, nil
}

// La vía VEC exige organización y unidad del firmante en la asignación real.
// Nunca admite el perfil fijo de RRHH de la vía externa ni otra acción.
func solicitudAutorizacionFirmaVecV2CTValida(datos core.DatosSolicitudAutorizacionLigadaV3,
	motivo core.ReferenciaEntradaCatalogo) bool {
	r := datos.Recurso
	if !core.ReferenciaMotivoAutorizacionV2Valida(motivo) || datos.ReferenciaMotivo != motivo ||
		datos.Finalidad != ports.FinalidadFirmaDocumento ||
		r.Validar() != nil || r.ModuloID != ports.ModuloContratacion || len(r.Ambitos) != 2 ||
		r.Ambitos["organizacion_ref"] == "" || r.Ambitos["unidad_ref"] == "" ||
		!domain.HuellaSHA256FirmaValida(r.Atributos["material_sha256"]) {
		return false
	}
	switch datos.Accion {
	case ports.AccionRegistrarFirmaVec:
		clave := r.Referencia
		if len(clave) <= len(ports.PrefijoRecursoFirmaVec) || clave[:len(ports.PrefijoRecursoFirmaVec)] != ports.PrefijoRecursoFirmaVec {
			return false
		}
		descriptor := domain.HuellaSHA256FirmaValida(r.Atributos["descriptor_firma_sha256"])
		plan := domain.HuellaSHA256FirmaValida(r.Atributos["plan_firma_sha256"])
		return r.Tipo == ports.TipoRecursoFirmaVec && ports.ClaveIdempotenciaFirmaValida(clave[len(ports.PrefijoRecursoFirmaVec):]) &&
			len(r.Atributos) == 2 && descriptor != plan
	case ports.AccionConsultarFirmasR5V2:
		return r.Tipo == ports.TipoRecursoConsultaFirmasR5 && domain.ReferenciaOpacaValida(r.Referencia) && len(r.Atributos) == 1
	}
	return false
}

// fuenteCompetenciaFirmaVecV2 envuelve la fuente del plan que consulta AUT56.
// La vía externa conserva su fuente y no escribe en este contenedor. En la vía
// VEC el resultado central queda ligado al certificado que verificó CA25.
type fuenteCompetenciaFirmaVecV2 struct {
	delegada ports.FuenteCompetenciaFirmante
}

var _ ports.FuenteCompetenciaFirmante = (*fuenteCompetenciaFirmaVecV2)(nil)

func nuevaFuenteCompetenciaFirmaVecV2(delegada ports.FuenteCompetenciaFirmante) (*fuenteCompetenciaFirmaVecV2, error) {
	if dependenciaEsNulaContratacionTemporalDesarrollo(delegada) {
		return nil, ports.ErrCompetenciaFirmanteNoDisponible
	}
	return &fuenteCompetenciaFirmaVecV2{delegada: delegada}, nil
}

func (f *fuenteCompetenciaFirmaVecV2) AcreditarCompetenciaFirmante(ctx context.Context,
	q ports.SolicitudCompetenciaFirmante,
) (ports.EvidenciaCompetenciaFirmante, error) {
	var cero ports.EvidenciaCompetenciaFirmante
	if f == nil || dependenciaEsNulaContratacionTemporalDesarrollo(f.delegada) || ctx == nil || ctx.Err() != nil {
		return cero, ports.ErrCompetenciaFirmanteNoDisponible
	}
	c, vec := competenciaFirmaVecV2DesdeContexto(ctx)
	if vec && (!domain.HuellaSHA256FirmaValida(q.CertificadoHuella) ||
		q.FirmanteRef != "ref:"+q.CertificadoHuella) {
		return cero, ports.ErrCompetenciaFirmanteNoAcreditada
	}
	e, err := f.delegada.AcreditarCompetenciaFirmante(ctx, q)
	if err != nil || !vec {
		return e, err
	}
	if e.Solicitud != q || !e.Vigente || !domain.ReferenciaOpacaValida(e.FirmantePrincipalRef) ||
		!domain.ReferenciaOpacaValida(e.CuentaFirmanteRef) || !domain.ReferenciaOpacaValida(e.PerfilActivoFirmanteRef) ||
		!domain.ReferenciaOpacaValida(e.VinculoCredencialFirmanteRef) ||
		e.VinculoCredencialFirmanteRevision == 0 || !domain.HuellaSHA256FirmaValida(e.VinculoCredencialFirmanteHuella) ||
		e.RolIDFirmante == "" || e.RolIDFirmante == rolFirmaExternaRegistroCTDesarrollo {
		return cero, ports.ErrCompetenciaFirmanteNoAcreditada
	}
	if err := c.guardar(e); err != nil {
		return cero, err
	}
	return e, nil
}
