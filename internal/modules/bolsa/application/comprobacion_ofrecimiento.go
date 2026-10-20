package application

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strings"

	"vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

var referenciaComprobacion = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9:._/-]+$`)
var huellaComprobacion = regexp.MustCompile(`^[0-9a-f]{64}$`)
var claveComprobacion = regexp.MustCompile(`^[A-Za-z0-9:_-]{8,256}$`)

type ServicioComprobacionOfrecimiento struct {
	contexto    ports.ResolutorContextoSituacionParticipacion
	autorizador ports.AutorizadorSituacionParticipacionV3
	repositorio ports.RepositorioComprobacion
}

func NuevoServicioComprobacionOfrecimiento(c ports.ResolutorContextoSituacionParticipacion, a ports.AutorizadorSituacionParticipacionV3, r ports.RepositorioComprobacion) (*ServicioComprobacionOfrecimiento, error) {
	if c == nil || a == nil || r == nil {
		return nil, ports.ErrComprobacionDependencia
	}
	return &ServicioComprobacionOfrecimiento{c, a, r}, nil
}

func (s *ServicioComprobacionOfrecimiento) Registrar(ctx context.Context, q ports.SolicitudComprobacion) (ports.ReciboComprobacion, error) {
	var cero ports.ReciboComprobacion
	if s == nil || ctx == nil || s.contexto == nil || s.autorizador == nil || s.repositorio == nil {
		return cero, ports.ErrComprobacionDependencia
	}
	q.Hechos = copiarHechosComprobacion(q.Hechos)
	if err := validarSolicitudComprobacion(q); err != nil {
		return cero, err
	}
	r, err := s.contexto.ResolverContextoSituacionParticipacion(ctx, q.ResultadoContexto.Contexto, q.Contexto.BolsaRef, q.Contexto.ParticipacionRef)
	if err != nil {
		return cero, dependenciaComprobacion(err)
	}
	if r.Validar() != nil || r.UnidadRef != q.Contexto.UnidadRef || r.AmbitoRef != q.Contexto.AmbitoRef {
		return cero, vd.ErrAutorizacionDenegada
	}
	// Esta lectura normaliza JSONB como PostgreSQL; json.Marshal no reproduce jsonb::text.
	canon, huella, err := s.repositorio.NormalizarComprobacion(ctx, q)
	if err != nil {
		return cero, dependenciaComprobacion(err)
	}
	if canon == "" || !huellaComprobacion.MatchString(huella) {
		return cero, ports.ErrComprobacionDependencia
	}
	recurso := vd.RecursoAutorizable{Referencia: q.Contexto.ParticipacionRef, ModuloID: "bolsa", Tipo: "participacion_bolsa", Ambitos: map[string]string{"unidad_ref": r.UnidadRef, "ambito_ref": r.AmbitoRef}, Atributos: map[string]string{"material_sha256": huella}}
	auth, err := vd.NuevaSolicitudAutorizacionLigadaV3(vd.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: q.Vinculo, ReferenciaMotivo: q.MotivoAutorizacion, Accion: ports.AccionComprobacionOfrecimiento, Recurso: recurso, Finalidad: ports.FinalidadComprobacionOfrecimiento, Correlacion: q.Correlacion})
	if err != nil {
		return cero, vd.ErrAutorizacionDenegada
	}
	decision, confirmacion, exportador, err := s.autorizador.EmitirMaterialAutorizacionAtestadaV3(ctx, auth, q.ResultadoContexto)
	if err != nil {
		return cero, dependenciaComprobacion(err)
	}
	if exportador == nil || decision.ValidarPara(auth) != nil {
		return cero, ports.ErrComprobacionDependencia
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !materialAutorizacionBorradorLlamamientoExacto(auth, decision, confirmacion, q.ResultadoContexto, q.MotivoAutorizacion, material, ports.AudienciaComprobacionOfrecimiento) {
		return cero, ports.ErrComprobacionDependencia
	}
	return s.repositorio.RegistrarComprobacion(ctx, ports.IntencionComprobacion{Solicitud: q, CanonPostgreSQL: canon, HuellaSHA256: huella, Material: material})
}

func copiarHechosComprobacion(origen map[string]ports.HechoComprobacion) map[string]ports.HechoComprobacion {
	if origen == nil {
		return nil
	}
	copia := make(map[string]ports.HechoComprobacion, len(origen))
	for k, v := range origen {
		v.Valor = append(json.RawMessage(nil), v.Valor...)
		copia[k] = v
	}
	return copia
}

func dependenciaComprobacion(err error) error {
	if errors.Is(err, vp.ErrDenegacionExplicitaAutorizacionLigadaV3) {
		return vd.ErrAutorizacionDenegada
	}
	if errors.Is(err, vd.ErrAutorizacionDenegada) || errors.Is(err, vd.ErrPermissionDenied) {
		return err
	}
	if errors.Is(err, ports.ErrComprobacionInvalida) || errors.Is(err, ports.ErrComprobacionConflicto) {
		return err
	}
	return ports.ErrComprobacionDependencia
}

func validarSolicitudComprobacion(q ports.SolicitudComprobacion) error {
	c := q.Contexto
	if q.ResultadoContexto.Validar() != nil || q.Vinculo.ValidarPara(q.ResultadoContexto) != nil || q.Correlacion.Validar() != nil || !vd.ReferenciaMotivoAutorizacionV2Valida(q.MotivoAutorizacion) || q.VersionEsperada < 0 || q.VersionEsperada > 999999999 || !claveComprobacion.MatchString(q.ClaveIdempotencia) || !huellaComprobacion.MatchString(c.HuellaPoliticaOfertasSHA256) {
		return ports.ErrComprobacionInvalida
	}
	for _, v := range []string{c.AmbitoRef, c.BolsaRef, c.CategoriaRef, c.ExpedienteRef, c.NecesidadRef, c.OrganizacionRef, c.ParticipacionRef, c.UnidadRef} {
		if len(v) > 256 || !referenciaComprobacion.MatchString(v) {
			return ports.ErrComprobacionInvalida
		}
	}
	if len(c.ModalidadOfrecida) > 256 || len(c.CausaOfrecida) > 256 || strings.TrimSpace(c.ModalidadOfrecida) == "" || c.OrdenActa < 1 || c.VersionBolsa < 1 || c.VersionExpediente < 1 || c.VersionInstantanea < 1 || c.VersionPoliticaOfertas < 1 {
		return ports.ErrComprobacionInvalida
	}
	if q.Hechos == nil || len(q.Hechos) > 11 {
		return ports.ErrComprobacionInvalida
	}
	for nombre, h := range q.Hechos {
		// Igual que B141: referencia legible del documento en custodia de RRHH
		// (guarda B77) o evento de contrato recibido por Bolsa.
		if !(domain.ReferenciaDocumentoCustodiaValida(h.DocumentoRef) || domain.ReferenciaEventoContratoParticipacion(h.DocumentoRef)) || !huellaComprobacion.MatchString(h.Huella) || len(h.Valor) == 0 {
			return ports.ErrComprobacionInvalida
		}
		var valor any
		if json.Unmarshal(h.Valor, &valor) != nil || valor == nil {
			return ports.ErrComprobacionInvalida
		}
		switch nombre {
		case "trabaja_en_diputacion", "es_vacante", "antecedentes_completos", "tipo_relacion", "modalidad", "causa_relacion", "inicio", "fin_previsto", "fin_inicial_programa", "fecha_inicial_baja", "prelacion":
			switch valor.(type) {
			case bool, string, float64:
			default:
				return ports.ErrComprobacionInvalida
			}
		default:
			return ports.ErrComprobacionInvalida
		}
	}
	return nil
}
