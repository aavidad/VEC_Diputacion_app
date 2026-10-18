package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	maximoOfertasConsulta = 100
	// maximaSecuenciaPlaza acota la historia de una plaza (la migración
	// B58 admite secuencias 1..10000).
	maximaSecuenciaPlaza = 10000
)

var idCalendarioOfertaV5 = regexp.MustCompile(`^[A-Za-z0-9:_-]+$`)

// ServicioOfertasPublicadas coordina la publicación de ofertas (art. 8.1 del
// Reglamento), su consulta por RRHH y la confirmación de la propuesta de
// adjudicación. El plazo lo fija la regla b10 del catálogo; la propuesta y
// todas las guardas de estado viven en el almacén de Bolsa.
type ServicioOfertasPublicadas struct {
	contextoBolsa puertosbolsa.ResolutorContextoContactoParticipacion
	autorizador   puertosbolsa.AutorizadorSituacionParticipacionV3
	repositorio   puertosbolsa.RepositorioOfertasPublicadas
	plazos        puertosbolsa.CalculadoraPlazoOferta
	reloj         func() time.Time
}

func NuevoServicioOfertasPublicadas(cb puertosbolsa.ResolutorContextoContactoParticipacion, a puertosbolsa.AutorizadorSituacionParticipacionV3, r puertosbolsa.RepositorioOfertasPublicadas, p puertosbolsa.CalculadoraPlazoOferta, reloj func() time.Time) (*ServicioOfertasPublicadas, error) {
	if cb == nil || a == nil || r == nil || p == nil || reloj == nil {
		return nil, puertosbolsa.ErrOfertaNoDisponible
	}
	return &ServicioOfertasPublicadas{cb, a, r, p, reloj}, nil
}

// PublicarOferta calcula el plazo con la regla vigente y reserva la oferta
// consumiendo una autorización de emisión de llamamiento sobre la bolsa.
func (s *ServicioOfertasPublicadas) PublicarOferta(ctx context.Context, q puertosbolsa.SolicitudPublicarOferta) (puertosbolsa.OfertaPublicada, error) {
	if ctx == nil || s == nil || q.ResultadoContexto.Validar() != nil || q.Vinculo.ValidarPara(q.ResultadoContexto) != nil ||
		!referenciaOfertaValida(q.BolsaRef) || !claveOfertaValida(q.ClaveIdempotencia) || q.Datos.Validar() != nil ||
		(q.Datos.ModalidadClave != "" && q.OrigenCT == nil) ||
		!dominiobolsa.NumeroPlazasValido(q.NumeroPlazas) || q.Correlacion.Validar() != nil || !dominiovec.ReferenciaMotivoAutorizacionV2Valida(q.MotivoAutorizacion) {
		return puertosbolsa.OfertaPublicada{}, puertosbolsa.ErrOfertaInvalida
	}
	if q.OrigenCT != nil && (q.Notificacion != (dominiobolsa.NotificacionOferta{}) ||
		q.OrigenCT.ValidarPara(q.BolsaRef, q.NumeroPlazas) != nil) {
		return puertosbolsa.OfertaPublicada{}, puertosbolsa.ErrOfertaInvalida
	}
	ahora := s.reloj().UTC().Truncate(time.Microsecond)
	if q.Notificacion != (dominiobolsa.NotificacionOferta{}) {
		if q.Notificacion.ValidarPara(ahora) != nil {
			return puertosbolsa.OfertaPublicada{}, puertosbolsa.ErrOfertaInvalida
		}
		q.Notificacion.NotificadaEn = q.Notificacion.NotificadaEn.UTC()
	}
	if recuperador, ok := s.repositorio.(puertosbolsa.RecuperadorOfertaPublicada); ok {
		huella := huellaMaterialRecuperarOfertaV1(q.BolsaRef, q.ClaveIdempotencia,
			q.ResultadoContexto.Contexto.PersonaRef, q.NumeroPlazas, q.Datos, q.OrigenCT, q.Notificacion)
		emision, err := s.materialEmision(ctx, q.Vinculo, q.ResultadoContexto, q.BolsaRef,
			q.Correlacion, q.MotivoAutorizacion, huella)
		if err != nil {
			return puertosbolsa.OfertaPublicada{}, err
		}
		recuperada, encontrada, err := recuperador.RecuperarOferta(ctx, puertosbolsa.ComandoRecuperarOferta{
			BolsaRef: q.BolsaRef, ActorRef: emision.ActorRef,
			UnidadRef: emision.UnidadRef, AmbitoRef: emision.AmbitoRef,
			ClaveIdempotencia: q.ClaveIdempotencia, Datos: q.Datos,
			Notificacion: q.Notificacion, NumeroPlazas: q.NumeroPlazas,
			OrigenCT: q.OrigenCT, Material: emision.Material,
		})
		if err != nil {
			return puertosbolsa.OfertaPublicada{}, err
		}
		if encontrada {
			if !ofertaRecuperadaCorresponde(recuperada, q) {
				return puertosbolsa.OfertaPublicada{}, puertosbolsa.ErrOfertaConflicto
			}
			return recuperada, nil
		}
	}
	// Las operaciones CT históricas sin claves se recuperan con su intención
	// original; una publicación CT nueva exige modalidad y causa gobernadas.
	if q.OrigenCT != nil && q.Datos.ModalidadClave == "" {
		return puertosbolsa.OfertaPublicada{}, puertosbolsa.ErrOfertaInvalida
	}
	inicio, err := s.inicioPlazoDisposicion(ctx, q.BolsaRef)
	if err != nil {
		return puertosbolsa.OfertaPublicada{}, err
	}
	if q.OrigenCT != nil && inicio != dominiobolsa.InicioPlazoOfertaPublicacion {
		return puertosbolsa.OfertaPublicada{}, puertosbolsa.ErrOfertaInvalida
	}
	comienzo := ahora
	switch inicio {
	case dominiobolsa.InicioPlazoOfertaPublicacion:
		if q.Notificacion != (dominiobolsa.NotificacionOferta{}) {
			return puertosbolsa.OfertaPublicada{}, puertosbolsa.ErrOfertaInvalida
		}
	case dominiobolsa.InicioPlazoOfertaNotificacion:
		if q.Notificacion == (dominiobolsa.NotificacionOferta{}) {
			return puertosbolsa.OfertaPublicada{}, puertosbolsa.ErrOfertaInvalida
		}
		comienzo = q.Notificacion.NotificadaEn
	default:
		return puertosbolsa.OfertaPublicada{}, puertosbolsa.ErrPlazoOfertaNoConfigurado
	}
	var plazo puertosbolsa.PlazoOferta
	var vence time.Time
	if porBolsa, ok := s.plazos.(puertosbolsa.CalculadoraPlazoOfertaPorBolsa); ok {
		plazo, vence, err = porBolsa.PlazoDisposicionBolsa(ctx, q.BolsaRef, comienzo)
	} else {
		plazo, vence, err = s.plazos.PlazoDisposicion(ctx, comienzo)
	}
	if err != nil {
		if errors.Is(err, puertosbolsa.ErrPlazoOfertaNoConfigurado) {
			return puertosbolsa.OfertaPublicada{}, err
		}
		return puertosbolsa.OfertaPublicada{}, puertosbolsa.ErrOfertaNoDisponible
	}
	if !vence.After(comienzo) || (plazo.Inicio != "" && plazo.Inicio != inicio) {
		return puertosbolsa.OfertaPublicada{}, puertosbolsa.ErrOfertaNoDisponible
	}
	if inicio == dominiobolsa.InicioPlazoOfertaPublicacion && !calendariosOfertaV5Validos(plazo.Calendarios) {
		return puertosbolsa.OfertaPublicada{}, puertosbolsa.ErrOfertaNoDisponible
	}
	vence = vence.UTC().Truncate(time.Microsecond)
	if inicio == dominiobolsa.InicioPlazoOfertaPublicacion {
		plazo.Inicio = inicio
		plazo.Notificacion = nil
	} else {
		notificacion := q.Notificacion
		plazo.Notificacion = &notificacion
	}
	materialHash := huellaMaterialPlazoOfertaConPlazas(q.BolsaRef, ahora, vence, plazo, q.NumeroPlazas)
	if inicio == dominiobolsa.InicioPlazoOfertaPublicacion {
		materialHash = huellaMaterialPublicarOferta(q.BolsaRef, q.ClaveIdempotencia,
			q.ResultadoContexto.Contexto.PersonaRef, q.NumeroPlazas, q.Datos, q.OrigenCT,
			ahora, vence, plazo)
	}
	emision, err := s.materialEmision(ctx, q.Vinculo, q.ResultadoContexto, q.BolsaRef, q.Correlacion, q.MotivoAutorizacion, materialHash)
	if err != nil {
		return puertosbolsa.OfertaPublicada{}, err
	}
	sufijo := huellaOferta(q.BolsaRef, q.ClaveIdempotencia)
	oferta, err := s.repositorio.Publicar(ctx, puertosbolsa.ComandoPublicarOferta{
		OfertaRef: "oferta:" + sufijo, ReciboRef: "recibo:oferta:" + sufijo, BolsaRef: q.BolsaRef, ActorRef: emision.ActorRef,
		UnidadRef: emision.UnidadRef, AmbitoRef: emision.AmbitoRef,
		ClaveIdempotencia: q.ClaveIdempotencia, Datos: q.Datos, NumeroPlazas: q.NumeroPlazas, OrigenCT: q.OrigenCT,
		Plazo: plazo, PublicadaEn: ahora,
		VenceAntesDe: vence, Material: emision.Material,
	})
	if err != nil {
		return puertosbolsa.OfertaPublicada{}, err
	}
	if oferta.BolsaRef != q.BolsaRef || oferta.Datos != q.Datos || oferta.NumeroPlazas != q.NumeroPlazas ||
		(inicio == dominiobolsa.InicioPlazoOfertaPublicacion &&
			(oferta.Plazo.Inicio != inicio || oferta.Plazo.Notificacion != nil)) ||
		(inicio == dominiobolsa.InicioPlazoOfertaNotificacion &&
			(oferta.Plazo.Notificacion == nil || *oferta.Plazo.Notificacion != q.Notificacion)) {
		return puertosbolsa.OfertaPublicada{}, puertosbolsa.ErrOfertaConflicto
	}
	return oferta, nil
}

// ResolverOferta registra un acto sobre una plaza. El almacén recalcula en ese
// mismo instante la propuesta y el estado de la plaza; si difieren de lo que
// RRHH vio (persona, tipo o secuencia), lo rechaza.
func (s *ServicioOfertasPublicadas) ResolverOferta(ctx context.Context, q puertosbolsa.SolicitudResolverOferta) (puertosbolsa.OfertaPublicada, error) {
	if ctx == nil || s == nil || q.ResultadoContexto.Validar() != nil || q.Vinculo.ValidarPara(q.ResultadoContexto) != nil ||
		!referenciaOfertaValida(q.BolsaRef) || !strings.HasPrefix(q.OfertaRef, "oferta:") || !referenciaOfertaValida(q.OfertaRef) ||
		!dominiobolsa.NumeroPlazasValido(q.NumeroDePlaza) || !dominiobolsa.ActoPlazaValido(q.Tipo, q.ParticipacionRef != "") ||
		q.SecuenciaEsperada < 0 || q.SecuenciaEsperada >= maximaSecuenciaPlaza ||
		(q.ParticipacionRef != "" && !referenciaOfertaValida(q.ParticipacionRef)) || !claveOfertaValida(q.ClaveIdempotencia) ||
		q.Correlacion.Validar() != nil || !dominiovec.ReferenciaMotivoAutorizacionV2Valida(q.MotivoAutorizacion) {
		return puertosbolsa.OfertaPublicada{}, puertosbolsa.ErrOfertaInvalida
	}
	emision, err := s.materialEmision(ctx, q.Vinculo, q.ResultadoContexto, q.BolsaRef, q.Correlacion, q.MotivoAutorizacion, "")
	if err != nil {
		return puertosbolsa.OfertaPublicada{}, err
	}
	return s.repositorio.Resolver(ctx, puertosbolsa.ComandoResolverOferta{
		OfertaRef: q.OfertaRef, ReciboRef: "recibo:plaza-oferta:" + huellaOferta(q.OfertaRef, q.ClaveIdempotencia),
		BolsaRef: q.BolsaRef, NumeroDePlaza: q.NumeroDePlaza, Tipo: q.Tipo, SecuenciaEsperada: q.SecuenciaEsperada,
		ParticipacionRef: q.ParticipacionRef, ActorRef: emision.ActorRef,
		ClaveIdempotencia: q.ClaveIdempotencia, Material: emision.Material,
	})
}

// ConsultarOfertas devuelve las ofertas de una bolsa admitida para la sesión
// de RRHH, con disposiciones, propuesta y resolución en el instante actual.
func (s *ServicioOfertasPublicadas) ConsultarOfertas(ctx context.Context, q puertosbolsa.SolicitudConsultarOfertas) ([]puertosbolsa.OfertaPublicada, error) {
	if ctx == nil || s == nil || q.ContextoActor.PersonaRef == "" || !referenciaOfertaValida(q.BolsaRef) || q.Limite < 1 || q.Limite > maximoOfertasConsulta {
		return nil, puertosbolsa.ErrOfertaInvalida
	}
	resuelto, err := s.contextoBolsa.ResolverContextoContactosBolsa(ctx, q.ContextoActor, q.BolsaRef)
	if err != nil || resuelto.Validar() != nil {
		return nil, errorDependenciaOferta(err)
	}
	return s.repositorio.Listar(ctx, q.BolsaRef, s.reloj().UTC().Truncate(time.Microsecond), q.Limite)
}

// InicioPlazoOferta devuelve el inicio que RRHH debe usar al preparar el
// formulario. La consulta comprueba la misma bolsa y sesión que el listado.
func (s *ServicioOfertasPublicadas) InicioPlazoOferta(ctx context.Context, q puertosbolsa.SolicitudConsultarOfertas) (string, error) {
	if ctx == nil || s == nil || q.ContextoActor.PersonaRef == "" || !referenciaOfertaValida(q.BolsaRef) {
		return "", puertosbolsa.ErrOfertaInvalida
	}
	resuelto, err := s.contextoBolsa.ResolverContextoContactosBolsa(ctx, q.ContextoActor, q.BolsaRef)
	if err != nil || resuelto.Validar() != nil {
		return "", errorDependenciaOferta(err)
	}
	return s.inicioPlazoDisposicion(ctx, q.BolsaRef)
}

func (s *ServicioOfertasPublicadas) inicioPlazoDisposicion(ctx context.Context, bolsa string) (string, error) {
	calculadora, ok := s.plazos.(puertosbolsa.CalculadoraInicioPlazoOferta)
	if !ok {
		return dominiobolsa.InicioPlazoOfertaNotificacion, nil
	}
	inicio, err := calculadora.InicioPlazoDisposicion(ctx, bolsa)
	if err != nil {
		if errors.Is(err, puertosbolsa.ErrPlazoOfertaNoConfigurado) {
			return "", err
		}
		return "", errorDependenciaOferta(err)
	}
	if inicio != dominiobolsa.InicioPlazoOfertaNotificacion && inicio != dominiobolsa.InicioPlazoOfertaPublicacion {
		return "", puertosbolsa.ErrPlazoOfertaNoConfigurado
	}
	return inicio, nil
}

type emisionOfertaAtestada struct {
	Material                       puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
	ActorRef, UnidadRef, AmbitoRef string
}

func (s *ServicioOfertasPublicadas) materialEmision(ctx context.Context, vinculo dominiovec.VinculoAutenticacionActorV2, resultado dominiovec.ResultadoContextoActorRegistradoV2, bolsa string, correlacion dominiovec.ReferenciaCorrelacionAutorizacionV2, motivo dominiovec.ReferenciaEntradaCatalogo, materialHash string) (emisionOfertaAtestada, error) {
	actor := resultado.Contexto
	resuelto, err := s.contextoBolsa.ResolverContextoContactosBolsa(ctx, actor, bolsa)
	if err != nil || resuelto.Validar() != nil {
		return emisionOfertaAtestada{}, errorDependenciaOferta(err)
	}
	recurso := dominiovec.RecursoAutorizable{Referencia: bolsa, ModuloID: puertosbolsa.ModuloBorradorLlamamiento, Tipo: puertosbolsa.TipoRecursoEmision, Ambitos: map[string]string{"unidad_ref": resuelto.UnidadRef, "ambito_ref": resuelto.AmbitoRef}}
	if materialHash != "" {
		recurso.Atributos = map[string]string{"material_sha256": materialHash}
	}
	auth, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(dominiovec.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: vinculo, ReferenciaMotivo: motivo, Accion: puertosbolsa.AccionEmitirLlamamiento, Recurso: recurso, Finalidad: puertosbolsa.FinalidadEmitirLlamamiento, Correlacion: correlacion})
	if err != nil {
		return emisionOfertaAtestada{}, dominiovec.ErrAutorizacionDenegada
	}
	decision, confirmacion, exportador, err := s.autorizador.EmitirMaterialAutorizacionAtestadaV3(ctx, auth, resultado)
	if err != nil || exportador == nil || decision.ValidarPara(auth) != nil {
		return emisionOfertaAtestada{}, errorDependenciaOferta(err)
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !materialAutorizacionBorradorLlamamientoExacto(auth, decision, confirmacion, resultado, motivo, material, puertosbolsa.AudienciaEmitirLlamamiento) {
		return emisionOfertaAtestada{}, puertosbolsa.ErrOfertaNoDisponible
	}
	return emisionOfertaAtestada{Material: material, ActorRef: actor.PersonaRef, UnidadRef: resuelto.UnidadRef, AmbitoRef: resuelto.AmbitoRef}, nil
}

const formatoInstanteMaterialOferta = "2006-01-02T15:04:05.000000Z"

// Esta preimagen no incluye textos personales y tiene los mismos campos y
// separadores que publicar_oferta_v2 en Bolsa B47.
func huellaMaterialPlazoOferta(bolsa string, publicada, vence time.Time, p puertosbolsa.PlazoOferta) string {
	h := sha256.Sum256([]byte(strings.Join(camposMaterialPlazoOferta(bolsa, publicada, vence, p), "\x1f")))
	return hex.EncodeToString(h[:])
}

func camposMaterialPlazoOferta(bolsa string, publicada, vence time.Time, p puertosbolsa.PlazoOferta) []string {
	campos := []string{bolsa, publicada.UTC().Format(formatoInstanteMaterialOferta),
		vence.UTC().Format(formatoInstanteMaterialOferta), p.ReglaRef, p.HuellaCatalogo,
		p.Unidad, fmt.Sprint(p.Cantidad), p.Computo, p.MunicipioSede, p.UltimoDia,
		fmt.Sprint(p.PoliticaVersion), strings.Join(p.Calendarios, "\x1e")}
	if p.Unidad == "horas_naturales" {
		campos = append(campos, p.AperturaEn, p.VenceEn)
	}
	return campos
}

// huellaMaterialPlazoOfertaConPlazas añade al final el número de plazas; es la
// preimagen de publicar_oferta_v3 en Bolsa B58, de modo que la autorización de
// publicación queda ligada a cuántas plazas se ofrecen.
func huellaMaterialPlazoOfertaConPlazas(bolsa string, publicada, vence time.Time, p puertosbolsa.PlazoOferta, numeroPlazas int) string {
	campos := camposMaterialPlazoOferta(bolsa, publicada, vence, p)
	campos = append(campos, fmt.Sprint(numeroPlazas))
	if n := p.Notificacion; n != nil {
		campos = append(campos, n.NotificadaEn.UTC().Format(formatoInstanteMaterialOferta), n.ReferenciaCorreo, n.HuellaCorreoSHA256, n.Fuente)
	}
	h := sha256.Sum256([]byte(strings.Join(campos, "\x1f")))
	return hex.EncodeToString(h[:])
}

// El canon v5 usa longitud de bytes UTF-8 por campo. SQL calcula los mismos
// bytes sin depender de la representación textual de jsonb. La lectura B110
// se verifica completa en Bolsa; estos campos ligan su identidad y corte.
func camposIntencionOfertaV5(prefijo, bolsa, clave, actor string, numero int, datos dominiobolsa.DatosOferta, origen *puertosbolsa.OrigenOfertaCT) []string {
	campos := []string{prefijo, bolsa, clave, actor, fmt.Sprint(numero),
		datos.Categoria, datos.Centro, datos.FechaInicio, datos.FechaFin, datos.Descripcion}
	if origen == nil {
		return append(campos, "sin_origen")
	}
	return append(campos, "con_origen",
		origen.OrganizacionRef, origen.UnidadRef, origen.ExpedienteRef,
		fmt.Sprint(origen.VersionExpediente), origen.ClaveCT, origen.NecesidadRef,
		origen.ActaRef, origen.CategoriaRef, origen.BolsaRef,
		fmt.Sprint(origen.VersionBolsa), origen.HuellaBolsaSHA256,
		origen.InstantaneaRef, fmt.Sprint(origen.VersionInstantanea),
		origen.HuellaInstantaneaSHA256, origen.PoliticaRef,
		fmt.Sprint(origen.VersionPolitica), origen.HuellaPoliticaSHA256,
		origen.CorteCanonico(), fmt.Sprint(origen.NumeroElegibles()))
}

func huellaCamposOfertaV5(campos []string) string {
	h := sha256.New()
	for _, campo := range campos {
		_, _ = h.Write([]byte(strconv.Itoa(len([]byte(campo))) + ":" + campo))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// El formato v6 liga las claves gobernadas al canon v5 sin cambiar las
// preimágenes de ofertas anteriores. Usa el mismo framing de bytes UTF-8.
func huellaCamposOferta(datos dominiobolsa.DatosOferta, campos []string) string {
	v5 := huellaCamposOfertaV5(campos)
	if datos.ModalidadClave == "" {
		return v5
	}
	return huellaCamposOfertaV5([]string{"vec.bolsa.oferta.canon.v6", v5, datos.ModalidadClave, datos.CausaClave})
}

func calendariosOfertaV5Validos(ids []string) bool {
	if len(ids) < 1 || len(ids) > 16 {
		return false
	}
	for _, id := range ids {
		if len(id) > 256 || !idCalendarioOfertaV5.MatchString(id) {
			return false
		}
	}
	return true
}

func camposNotificacionOfertaV5(campos []string, n dominiobolsa.NotificacionOferta) []string {
	if n == (dominiobolsa.NotificacionOferta{}) {
		return append(campos, "sin_notificacion")
	}
	return append(campos, "con_notificacion",
		n.NotificadaEn.UTC().Format(formatoInstanteMaterialOferta),
		n.ReferenciaCorreo, n.HuellaCorreoSHA256, n.Fuente)
}

func huellaMaterialRecuperarOfertaV1(bolsa, clave, actor string, numero int, datos dominiobolsa.DatosOferta, origen *puertosbolsa.OrigenOfertaCT, notificacion dominiobolsa.NotificacionOferta) string {
	campos := camposIntencionOfertaV5("vec.bolsa.oferta.recuperar.v1", bolsa, clave, actor, numero, datos, origen)
	return huellaCamposOferta(datos, camposNotificacionOfertaV5(campos, notificacion))
}

func huellaMaterialPublicarOferta(bolsa, clave, actor string, numero int, datos dominiobolsa.DatosOferta, origen *puertosbolsa.OrigenOfertaCT, publicada, vence time.Time, plazo puertosbolsa.PlazoOferta) string {
	campos := camposIntencionOfertaV5("vec.bolsa.oferta.publicar.v1", bolsa, clave, actor, numero, datos, origen)
	campos = camposNotificacionOfertaV5(campos, dominiobolsa.NotificacionOferta{})
	campos = append(campos, publicada.UTC().Format(formatoInstanteMaterialOferta),
		vence.UTC().Format(formatoInstanteMaterialOferta), plazo.ReglaRef,
		plazo.HuellaCatalogo, plazo.Unidad, fmt.Sprint(plazo.Cantidad),
		plazo.Computo, plazo.MunicipioSede, plazo.UltimoDia,
		fmt.Sprint(plazo.PoliticaVersion), strings.Join(plazo.Calendarios, "\x1e"),
		plazo.AperturaEn, plazo.VenceEn)
	return huellaCamposOferta(datos, campos)
}

func ofertaRecuperadaCorresponde(o puertosbolsa.OfertaPublicada, q puertosbolsa.SolicitudPublicarOferta) bool {
	if o.OfertaRef == "" || o.ReciboRef == "" || o.BolsaRef != q.BolsaRef ||
		o.Datos != q.Datos || o.NumeroPlazas != q.NumeroPlazas ||
		o.PublicadaEn.IsZero() || !o.VenceAntesDe.After(o.PublicadaEn) {
		return false
	}
	if q.Notificacion == (dominiobolsa.NotificacionOferta{}) {
		return o.Plazo.Inicio == dominiobolsa.InicioPlazoOfertaPublicacion && o.Plazo.Notificacion == nil
	}
	return (o.Plazo.Inicio == "" || o.Plazo.Inicio == dominiobolsa.InicioPlazoOfertaNotificacion) &&
		o.Plazo.Notificacion != nil && *o.Plazo.Notificacion == q.Notificacion
}

func errorDependenciaOferta(err error) error {
	if errors.Is(err, dominiovec.ErrAutorizacionDenegada) || errors.Is(err, dominiovec.ErrPermissionDenied) {
		return err
	}
	return puertosbolsa.ErrOfertaNoDisponible
}

func huellaOferta(base, clave string) string {
	h := sha256.Sum256([]byte(base + "\x1f" + clave))
	return hex.EncodeToString(h[:])
}

func referenciaOfertaValida(ref string) bool {
	return ref != "" && len(ref) <= 256 && strings.TrimSpace(ref) == ref
}

func claveOfertaValida(clave string) bool {
	return len(clave) >= 8 && len(clave) <= 256 && strings.TrimSpace(clave) == clave
}
