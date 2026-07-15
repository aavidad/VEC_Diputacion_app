package application

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type OrdenActivarCodigoCotejo struct {
	Principal        domain.Principal
	PerfilActivo     string
	RepresentadoRef  string
	Finalidad        string
	CodigoID         string
	RepresentacionID string
	ActivacionRef    string
	Motivo           string
	CorrelacionRef   string
}

func (s *ServicioCotejo) ActivarCodigoCotejo(ctx context.Context, orden OrdenActivarCodigoCotejo) (domain.CodigoCotejo, error) {
	if err := ctx.Err(); err != nil {
		return domain.CodigoCotejo{}, err
	}
	if err := validarContextoCotejo(orden.Principal, orden.PerfilActivo, orden.Finalidad,
		orden.Motivo, orden.CorrelacionRef, domain.AuthAssuranceHigh); err != nil {
		return domain.CodigoCotejo{}, err
	}
	if strings.TrimSpace(orden.CodigoID) == "" || strings.TrimSpace(orden.RepresentacionID) == "" ||
		strings.TrimSpace(orden.ActivacionRef) == "" {
		return domain.CodigoCotejo{}, ErrOrdenCotejoInvalida
	}
	anterior, err := s.codigos.ObtenerCodigoCotejo(ctx, strings.TrimSpace(orden.CodigoID))
	if err != nil {
		return domain.CodigoCotejo{}, err
	}
	documento, expedienteRef, err := s.documentoVinculadoCodigoCotejo(ctx, anterior)
	if err != nil {
		return domain.CodigoCotejo{}, err
	}
	if anterior.Estado != domain.EstadoCodigoCotejoReservado ||
		!estadoDocumentoPermiteActivarCotejo(documento.Estado, anterior.Politica) {
		return domain.CodigoCotejo{}, domain.ErrTransicionCodigoCotejo
	}
	decision, err := exigirDecisionAutorizacion(ctx, s.autorizador, s.reloj,
		orden.Principal, orden.PerfilActivo, AccionActivarCodigoCotejo,
		recursoCodigoCotejo(anterior, expedienteRef, nil), orden.Finalidad, orden.CorrelacionRef, orden.Motivo,
		usoCamposDecisionNoAplicables)
	if err != nil {
		return domain.CodigoCotejo{}, err
	}
	evidencia, err := s.evidenciasEmision.ObtenerEvidenciaEmisionDocumento(ctx, ports.SolicitudObtenerEvidenciaEmisionDocumento{
		Documento:        anterior.Documento,
		RepresentacionID: strings.TrimSpace(orden.RepresentacionID),
		SolicitanteID:    strings.TrimSpace(orden.Principal.ID),
		AutorizacionRef:  decision.DecisionRef,
		Finalidad:        strings.TrimSpace(orden.Finalidad),
		CorrelacionRef:   strings.TrimSpace(orden.CorrelacionRef),
	})
	if err != nil {
		return domain.CodigoCotejo{}, err
	}
	if evidencia.Validar() != nil || evidencia.Documento != anterior.Documento ||
		evidencia.VersionEmitida.RepresentacionID != strings.TrimSpace(orden.RepresentacionID) {
		return domain.CodigoCotejo{}, domain.ErrEvidenciaEmisionInvalida
	}
	representaciones, err := s.documentos.ListarRepresentacionesDocumento(ctx, anterior.Documento)
	if err != nil {
		return domain.CodigoCotejo{}, err
	}
	representacion, encontrada := buscarRepresentacionCotejo(representaciones, orden.RepresentacionID)
	if !encontrada || !representacionAptaParaCotejo(representacion, documento, evidencia, anterior.Politica) {
		return domain.CodigoCotejo{}, domain.ErrEvidenciaEmisionInvalida
	}
	huellaAnterior, err := anterior.HuellaEstadoSHA256()
	if err != nil {
		return domain.CodigoCotejo{}, err
	}
	ahora := s.reloj.Ahora().UTC()
	activado, err := anterior.Activar(orden.Principal.ID, orden.ActivacionRef, orden.Motivo, evidencia, ahora)
	if err != nil {
		return domain.CodigoCotejo{}, err
	}
	huellaNueva, err := activado.HuellaEstadoSHA256()
	if err != nil {
		return domain.CodigoCotejo{}, err
	}
	traza := trazaCodigoCotejo(orden.Principal, orden.PerfilActivo, orden.RepresentadoRef,
		decision.DecisionRef, orden.Finalidad, eventoCodigoCotejoActivado, activado,
		orden.Motivo, orden.CorrelacionRef, huellaAnterior, huellaNueva, ahora)
	traza.RuleRef = evidencia.EvidenciaRef
	traza.Metadata["representacion_id"] = evidencia.VersionEmitida.RepresentacionID
	traza.Metadata["firmas"] = strconv.Itoa(len(evidencia.VersionEmitida.FirmaRefs))
	traza.Metadata["sellos_tiempo"] = strconv.Itoa(len(evidencia.VersionEmitida.SelloTiempoRefs))
	traza.Metadata["registro"] = strconv.FormatBool(strings.TrimSpace(evidencia.VersionEmitida.RegistroRef) != "")
	evento := eventoCodigoCotejo(eventoCodigoCotejoActivado, activado, orden.Principal.ID, huellaNueva, ahora)
	evento.Payload["representacion_id"] = evidencia.VersionEmitida.RepresentacionID
	if err := s.codigos.ConfirmarActivacionCodigoCotejo(ctx, huellaAnterior, activado, traza, evento); err != nil {
		return domain.CodigoCotejo{}, err
	}
	return activado, nil
}

type OrdenRetirarCodigoCotejo struct {
	Principal       domain.Principal
	PerfilActivo    string
	RepresentadoRef string
	Finalidad       string
	CodigoID        string
	RetiradaRef     string
	Motivo          string
	CorrelacionRef  string
}

func (s *ServicioCotejo) RetirarCodigoCotejo(ctx context.Context, orden OrdenRetirarCodigoCotejo) (domain.CodigoCotejo, error) {
	if err := ctx.Err(); err != nil {
		return domain.CodigoCotejo{}, err
	}
	if err := validarContextoCotejo(orden.Principal, orden.PerfilActivo, orden.Finalidad,
		orden.Motivo, orden.CorrelacionRef, domain.AuthAssuranceHigh); err != nil {
		return domain.CodigoCotejo{}, err
	}
	if strings.TrimSpace(orden.CodigoID) == "" || strings.TrimSpace(orden.RetiradaRef) == "" {
		return domain.CodigoCotejo{}, ErrOrdenCotejoInvalida
	}
	anterior, err := s.codigos.ObtenerCodigoCotejo(ctx, strings.TrimSpace(orden.CodigoID))
	if err != nil {
		return domain.CodigoCotejo{}, err
	}
	_, expedienteRef, err := s.documentoVinculadoCodigoCotejo(ctx, anterior)
	if err != nil {
		return domain.CodigoCotejo{}, err
	}
	decision, err := exigirDecisionAutorizacion(ctx, s.autorizador, s.reloj,
		orden.Principal, orden.PerfilActivo, AccionRetirarCodigoCotejo,
		recursoCodigoCotejo(anterior, expedienteRef, nil), orden.Finalidad, orden.CorrelacionRef, orden.Motivo,
		usoCamposDecisionNoAplicables)
	if err != nil {
		return domain.CodigoCotejo{}, err
	}
	huellaAnterior, err := anterior.HuellaEstadoSHA256()
	if err != nil {
		return domain.CodigoCotejo{}, err
	}
	ahora := s.reloj.Ahora().UTC()
	retirado, err := anterior.Retirar(orden.Principal.ID, orden.RetiradaRef, orden.Motivo, ahora)
	if err != nil {
		return domain.CodigoCotejo{}, err
	}
	huellaNueva, err := retirado.HuellaEstadoSHA256()
	if err != nil {
		return domain.CodigoCotejo{}, err
	}
	traza := trazaCodigoCotejo(orden.Principal, orden.PerfilActivo, orden.RepresentadoRef,
		decision.DecisionRef, orden.Finalidad, eventoCodigoCotejoRetirado, retirado,
		orden.Motivo, orden.CorrelacionRef, huellaAnterior, huellaNueva, ahora)
	traza.RuleRef = strings.TrimSpace(orden.RetiradaRef)
	evento := eventoCodigoCotejo(eventoCodigoCotejoRetirado, retirado, orden.Principal.ID, huellaNueva, ahora)
	if err := s.codigos.ConfirmarRetiradaCodigoCotejo(ctx, huellaAnterior, retirado, traza, evento); err != nil {
		return domain.CodigoCotejo{}, err
	}
	return retirado, nil
}

type OrdenSustituirCodigoCotejo struct {
	Principal       domain.Principal
	PerfilActivo    string
	RepresentadoRef string
	Finalidad       string
	CodigoID        string
	SustitutoID     string
	SustitucionRef  string
	Motivo          string
	CorrelacionRef  string
}

func (s *ServicioCotejo) SustituirCodigoCotejo(ctx context.Context, orden OrdenSustituirCodigoCotejo) (domain.CodigoCotejo, error) {
	if err := ctx.Err(); err != nil {
		return domain.CodigoCotejo{}, err
	}
	if err := validarContextoCotejo(orden.Principal, orden.PerfilActivo, orden.Finalidad,
		orden.Motivo, orden.CorrelacionRef, domain.AuthAssuranceHigh); err != nil {
		return domain.CodigoCotejo{}, err
	}
	if strings.TrimSpace(orden.CodigoID) == "" || strings.TrimSpace(orden.SustitutoID) == "" ||
		orden.CodigoID == orden.SustitutoID || strings.TrimSpace(orden.SustitucionRef) == "" {
		return domain.CodigoCotejo{}, ErrOrdenCotejoInvalida
	}
	anterior, err := s.codigos.ObtenerCodigoCotejo(ctx, strings.TrimSpace(orden.CodigoID))
	if err != nil {
		return domain.CodigoCotejo{}, err
	}
	sustituto, err := s.codigos.ObtenerCodigoCotejo(ctx, strings.TrimSpace(orden.SustitutoID))
	if err != nil {
		return domain.CodigoCotejo{}, err
	}
	_, expedienteRef, err := s.documentoVinculadoCodigoCotejo(ctx, anterior)
	if err != nil {
		return domain.CodigoCotejo{}, err
	}
	_, expedienteSustituto, err := s.documentoVinculadoCodigoCotejo(ctx, sustituto)
	if err != nil {
		return domain.CodigoCotejo{}, err
	}
	if anterior.Estado != domain.EstadoCodigoCotejoActivo || sustituto.Estado != domain.EstadoCodigoCotejoActivo ||
		anterior.Documento == sustituto.Documento || expedienteRef != expedienteSustituto ||
		anterior.ModuloID != sustituto.ModuloID || anterior.TipoDocumental != sustituto.TipoDocumental ||
		anterior.Clasificacion != sustituto.Clasificacion || anterior.Organo != sustituto.Organo {
		return domain.CodigoCotejo{}, domain.ErrTransicionCodigoCotejo
	}
	decision, err := exigirDecisionAutorizacion(ctx, s.autorizador, s.reloj,
		orden.Principal, orden.PerfilActivo, AccionSustituirCodigoCotejo,
		recursoCodigoCotejo(anterior, expedienteRef, map[string]string{"sustituto_ref": sustituto.Referencia()}),
		orden.Finalidad, orden.CorrelacionRef, orden.Motivo, usoCamposDecisionNoAplicables)
	if err != nil {
		return domain.CodigoCotejo{}, err
	}
	ahora := s.reloj.Ahora().UTC()
	if !sustituto.DisponibleEn(ahora) {
		return domain.CodigoCotejo{}, domain.ErrTransicionCodigoCotejo
	}
	huellaAnterior, err := anterior.HuellaEstadoSHA256()
	if err != nil {
		return domain.CodigoCotejo{}, err
	}
	sustituido, err := anterior.Sustituir(orden.Principal.ID, orden.SustitucionRef,
		orden.Motivo, sustituto.Referencia(), ahora)
	if err != nil {
		return domain.CodigoCotejo{}, err
	}
	huellaNueva, err := sustituido.HuellaEstadoSHA256()
	if err != nil {
		return domain.CodigoCotejo{}, err
	}
	traza := trazaCodigoCotejo(orden.Principal, orden.PerfilActivo, orden.RepresentadoRef,
		decision.DecisionRef, orden.Finalidad, eventoCodigoCotejoSustituido, sustituido,
		orden.Motivo, orden.CorrelacionRef, huellaAnterior, huellaNueva, ahora)
	traza.RuleRef = strings.TrimSpace(orden.SustitucionRef)
	traza.Metadata["sustituto_ref"] = sustituto.Referencia()
	evento := eventoCodigoCotejo(eventoCodigoCotejoSustituido, sustituido, orden.Principal.ID, huellaNueva, ahora)
	evento.Payload["sustituto_ref"] = sustituto.Referencia()
	if err := s.codigos.ConfirmarSustitucionCodigoCotejo(ctx, huellaAnterior, sustituido, traza, evento); err != nil {
		return domain.CodigoCotejo{}, err
	}
	return sustituido, nil
}

func (s *ServicioCotejo) documentoVinculadoCodigoCotejo(ctx context.Context, codigo domain.CodigoCotejo) (domain.DocumentoLogico, string, error) {
	if err := codigo.Validar(); err != nil {
		return domain.DocumentoLogico{}, "", err
	}
	documento, err := s.documentos.ObtenerDocumentoLogico(ctx, codigo.Documento)
	if err != nil {
		return domain.DocumentoLogico{}, "", err
	}
	expedienteRef, err := referenciaExpedientePrincipal(documento.Relaciones)
	if err != nil {
		return domain.DocumentoLogico{}, "", err
	}
	if documento.Referencia() != codigo.Documento || documento.ModuloID != codigo.ModuloID ||
		documento.TipoDocumental != codigo.TipoDocumental || documento.Clasificacion != codigo.Clasificacion ||
		documento.ENI.Organo != codigo.Organo || expedienteRef != codigo.ExpedienteRef {
		return domain.DocumentoLogico{}, "", errors.Join(ErrResultadoCotejoInvalido, domain.ErrDocumentoLogicoInvalido)
	}
	return documento, expedienteRef, nil
}

func estadoDocumentoPermiteActivarCotejo(estado domain.EstadoDocumentoLogico, politica domain.AplicacionPoliticaCotejo) bool {
	if politica.RequiereRegistro {
		return estado == domain.EstadoDocumentoLogicoRegistrado
	}
	if politica.RequiereFirma {
		return estado == domain.EstadoDocumentoLogicoFirmado || estado == domain.EstadoDocumentoLogicoRegistrado
	}
	return estado == domain.EstadoDocumentoLogicoCerrado || estado == domain.EstadoDocumentoLogicoFirmado ||
		estado == domain.EstadoDocumentoLogicoRegistrado
}

func buscarRepresentacionCotejo(representaciones []domain.RepresentacionDocumento, id string) (domain.RepresentacionDocumento, bool) {
	id = strings.TrimSpace(id)
	for _, representacion := range representaciones {
		if representacion.ID == id {
			return representacion, true
		}
	}
	return domain.RepresentacionDocumento{}, false
}

func representacionAptaParaCotejo(
	representacion domain.RepresentacionDocumento,
	documento domain.DocumentoLogico,
	evidencia domain.EvidenciaEmisionDocumento,
	politica domain.AplicacionPoliticaCotejo,
) bool {
	version := evidencia.VersionEmitida
	if representacion.ValidarPertenencia(documento) != nil || representacion.EstadoTecnico != domain.EstadoRepresentacionDisponible ||
		(representacion.EstadoAntivirus != domain.EstadoAntivirusLimpio && representacion.EstadoAntivirus != domain.EstadoAntivirusNoAplica) ||
		representacion.Tipo == domain.TipoRepresentacionTrabajo ||
		(politica.RequiereFirma && representacion.Tipo != domain.TipoRepresentacionFirma && representacion.Tipo != domain.TipoRepresentacionPreservacion) ||
		version.RepresentacionID != representacion.ID || version.ReferenciaContenido != representacion.ReferenciaContenido ||
		version.HuellaContenidoSHA256 != representacion.HuellaContenidoSHA256 || version.MIME != representacion.MIME ||
		version.Tamano != representacion.Tamano || version.EmitidaEn.Before(representacion.GeneradaEn) {
		return false
	}
	return true
}

func recursoCodigoCotejo(codigo domain.CodigoCotejo, expedienteRef string, atributos map[string]string) domain.RecursoAutorizable {
	atributosCopia := map[string]string{
		"documento_ref":   referenciaDocumentoCotejo(codigo.Documento),
		"tipo_documental": codigo.TipoDocumental,
		"estado":          string(codigo.Estado),
	}
	for clave, valor := range atributos {
		atributosCopia[clave] = valor
	}
	return domain.RecursoAutorizable{
		Referencia: codigo.Referencia(),
		ModuloID:   codigo.ModuloID,
		Tipo:       "codigo_cotejo",
		Ambitos: map[string]string{
			"expediente":    expedienteRef,
			"clasificacion": codigo.Clasificacion,
		},
		Atributos: atributosCopia,
	}
}
