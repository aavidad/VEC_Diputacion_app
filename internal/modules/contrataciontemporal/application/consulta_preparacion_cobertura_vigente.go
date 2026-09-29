package application

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const TiempoMaximoConsultaPreparacionCoberturaVigente = 5 * time.Second

// La consulta previa al alta no recibe expediente, vía, fecha ni actor libres
// del canal. El contexto V3 resuelve el perfil activo antes de cualquier lectura.
type SolicitudConsultarPreparacionCoberturaVigente struct {
	AutenticacionRef string
	SesionRef        string
	PerfilRef        string
	OrganizacionRef  string
}

func (SolicitudConsultarPreparacionCoberturaVigente) String() string {
	return redaccionSolicitudProponerCobertura
}
func (s SolicitudConsultarPreparacionCoberturaVigente) GoString() string { return s.String() }

// AutorizarConsultaPreparacionCoberturaVigente representa exclusivamente la
// acción V3 de lectura previa al alta. La concesión de proponer o decidir una
// cobertura no satisface este puerto.
type AutorizadorConsultaPreparacionCoberturaVigente interface {
	AutorizarConsultaPreparacionCoberturaVigente(
		context.Context,
		ports.SolicitudResolverContextoAutorizacionAltaV3,
		ports.ContextoAutorizacionAltaV3,
		string,
		time.Time,
	) error
}

// FuenteCatalogoCoberturaVigente devuelve la publicación durable actual del
// propietario CT. Nunca recibe una referencia de expediente del navegador.
type FuenteCatalogoCoberturaVigente interface {
	ConsultarCatalogoViasCoberturaVigente(context.Context, string) (domain.CatalogoViasCobertura, error)
}

type ServicioConsultaPreparacionCoberturaVigente struct {
	contextos   ports.ResolutorContextoAutorizacionAltaV3
	autorizador AutorizadorConsultaPreparacionCoberturaVigente
	fuente      FuenteCatalogoCoberturaVigente
	reloj       cobertura.RelojGobiernoOperacionCobertura
}

func NuevoServicioConsultaPreparacionCoberturaVigente(
	contextos ports.ResolutorContextoAutorizacionAltaV3,
	autorizador AutorizadorConsultaPreparacionCoberturaVigente,
	fuente FuenteCatalogoCoberturaVigente,
	reloj cobertura.RelojGobiernoOperacionCobertura,
) (*ServicioConsultaPreparacionCoberturaVigente, error) {
	if dependenciaNula(contextos) || dependenciaNula(autorizador) ||
		dependenciaNula(fuente) || dependenciaNula(reloj) {
		return nil, ErrServicioPresentacionPropuestaCoberturaInvalido
	}
	return &ServicioConsultaPreparacionCoberturaVigente{
		contextos: contextos, autorizador: autorizador,
		fuente: fuente, reloj: reloj,
	}, nil
}

// ResultadoConsultaPreparacionCoberturaVigenteParaAdaptador es una instantánea
// mínima. El adaptador no puede construirla ni recibir la publicación completa.
type ResultadoConsultaPreparacionCoberturaVigenteParaAdaptador struct {
	datos PreparacionCatalogoPropuestaCobertura
	sello string
}

func (r ResultadoConsultaPreparacionCoberturaVigenteParaAdaptador) DatosParaAdaptador() (PreparacionCatalogoPropuestaCobertura, bool) {
	if r.sello == "" || r.sello != r.datos.Identidad.HuellaSHA256 ||
		!preparacionCatalogoVigenteValida(r.datos) {
		return PreparacionCatalogoPropuestaCobertura{}, false
	}
	return *copiarPreparacionCatalogoCobertura(&r.datos), true
}

func (s *ServicioConsultaPreparacionCoberturaVigente) ConsultarParaAdaptador(
	ctx context.Context,
	peticion SolicitudConsultarPreparacionCoberturaVigente,
) (ResultadoConsultaPreparacionCoberturaVigenteParaAdaptador, error) {
	vacio := ResultadoConsultaPreparacionCoberturaVigenteParaAdaptador{}
	if ctx == nil || s == nil || dependenciaNula(s.contextos) ||
		dependenciaNula(s.autorizador) || dependenciaNula(s.fuente) ||
		dependenciaNula(s.reloj) || !domain.ReferenciaOpacaValida(peticion.OrganizacionRef) {
		return vacio, ErrSolicitudProponerCoberturaInvalida
	}
	solicitudContexto := ports.SolicitudResolverContextoAutorizacionAltaV3{
		AutenticacionRef: peticion.AutenticacionRef,
		SesionRef:        peticion.SesionRef,
		PerfilRef:        peticion.PerfilRef,
	}
	if solicitudContexto.Validar() != nil {
		return vacio, ErrSolicitudProponerCoberturaInvalida
	}
	operacion, cancelar := context.WithTimeout(ctx, TiempoMaximoConsultaPreparacionCoberturaVigente)
	defer cancelar()
	contexto, err := s.contextos.ResolverContextoAutorizacionAltaV3(operacion, solicitudContexto)
	if err != nil {
		return vacio, normalizarErrorConsultaPreparacionCobertura(operacion, err)
	}
	inicio, err := s.ahora(operacion)
	if err != nil {
		return vacio, normalizarErrorConsultaPreparacionCobertura(operacion, err)
	}
	if contexto.ValidarPara(solicitudContexto, inicio) != nil {
		return vacio, ErrPresentacionPropuestaCoberturaDenegada
	}
	if err := s.autorizar(operacion, solicitudContexto, contexto, peticion.OrganizacionRef, inicio); err != nil {
		return vacio, err
	}
	catalogo, err := s.fuente.ConsultarCatalogoViasCoberturaVigente(operacion, peticion.OrganizacionRef)
	if err != nil {
		return vacio, normalizarErrorConsultaPreparacionCobertura(operacion, err)
	}
	final, err := s.ahora(operacion)
	if err != nil {
		return vacio, normalizarErrorConsultaPreparacionCobertura(operacion, err)
	}
	if contexto.ValidarPara(solicitudContexto, final) != nil {
		return vacio, ErrPresentacionPropuestaCoberturaDenegada
	}
	if err := s.autorizar(operacion, solicitudContexto, contexto, peticion.OrganizacionRef, final); err != nil {
		return vacio, err
	}
	if catalogo.Validar() != nil || catalogo.Canon() != domain.CanonHuellaCatalogoCoberturaV2() ||
		catalogo.PublicadoEn().After(final) || !catalogo.VigenteEn(final) {
		return vacio, ErrPresentacionPropuestaCoberturaNoDisponible
	}
	publicacion := catalogo.Publicacion()
	datos := PreparacionCatalogoPropuestaCobertura{
		Identidad: catalogo.Identidad(), Canon: catalogo.Canon(),
		EsEjemplo: publicacion.EsEjemplo,
		Vias:      make([]PreparacionViaPropuestaCobertura, 0, len(publicacion.Vias)),
	}
	for _, via := range publicacion.Vias {
		datos.Vias = append(datos.Vias, PreparacionViaPropuestaCobertura{
			Clave: via.Clave, Orden: via.Orden,
			Documentos: append([]domain.ElementoPreparacionViaCobertura(nil), via.Documentos...),
			Datos:      append([]domain.ElementoPreparacionViaCobertura(nil), via.Datos...),
		})
	}
	if !preparacionCatalogoVigenteValida(datos) {
		return vacio, ErrPresentacionPropuestaCoberturaNoDisponible
	}
	return ResultadoConsultaPreparacionCoberturaVigenteParaAdaptador{
		datos: datos, sello: datos.Identidad.HuellaSHA256,
	}, nil
}

func (s *ServicioConsultaPreparacionCoberturaVigente) ahora(ctx context.Context) (time.Time, error) {
	instante, err := s.reloj.AhoraGobiernoOperacionCobertura(ctx)
	if err != nil || !domain.InstanteUTCCanonico(instante) {
		return time.Time{}, ErrPresentacionPropuestaCoberturaNoDisponible
	}
	return instante, nil
}

func (s *ServicioConsultaPreparacionCoberturaVigente) autorizar(
	ctx context.Context,
	solicitud ports.SolicitudResolverContextoAutorizacionAltaV3,
	contexto ports.ContextoAutorizacionAltaV3,
	organizacion string,
	instante time.Time,
) error {
	err := s.autorizador.AutorizarConsultaPreparacionCoberturaVigente(ctx, solicitud, contexto, organizacion, instante)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, ErrPreparacionCatalogoCoberturaNoDisponiblePerfil) {
		return ErrPreparacionCatalogoCoberturaNoDisponiblePerfil
	}
	if errors.Is(err, ErrPresentacionPropuestaCoberturaDenegada) {
		return ErrPresentacionPropuestaCoberturaDenegada
	}
	if err != nil {
		return ErrPresentacionPropuestaCoberturaNoDisponible
	}
	return nil
}

func normalizarErrorConsultaPreparacionCobertura(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, ErrPresentacionPropuestaCoberturaDenegada) || errors.Is(err, ports.ErrAutorizacionDenegada) {
		return ErrPresentacionPropuestaCoberturaDenegada
	}
	return ErrPresentacionPropuestaCoberturaNoDisponible
}

func preparacionCatalogoVigenteValida(datos PreparacionCatalogoPropuestaCobertura) bool {
	if datos.Canon != domain.CanonHuellaCatalogoCoberturaV2() || datos.Identidad.Validar() != nil ||
		len(datos.Vias) == 0 || len(datos.Vias) > 64 {
		return false
	}
	claves := make(map[domain.ClaveCatalogo]struct{}, len(datos.Vias))
	ordenes := make(map[uint16]struct{}, len(datos.Vias))
	total := 0
	for _, via := range datos.Vias {
		if !via.Clave.Valida() || via.Orden == 0 ||
			len(via.Documentos) > 32 || len(via.Datos) > 32 ||
			!elementosPreparacionAdaptadorValidos(via.Documentos) ||
			!elementosPreparacionAdaptadorValidos(via.Datos) {
			return false
		}
		if _, existe := claves[via.Clave]; existe {
			return false
		}
		if _, existe := ordenes[via.Orden]; existe {
			return false
		}
		claves[via.Clave] = struct{}{}
		ordenes[via.Orden] = struct{}{}
		total += len(via.Documentos) + len(via.Datos)
	}
	return total > 0 && total <= 512
}
