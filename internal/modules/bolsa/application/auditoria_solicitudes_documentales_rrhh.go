package application

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// ConsultorSolicitudesDocumentalesRRHH conserva el servicio existente y su
// transacción positiva. El decorador no consume una segunda autorización.
type ConsultorSolicitudesDocumentalesRRHH interface {
	ListarSolicitudesDocumentalesRRHH(context.Context, ports.SolicitudCambiarSituacionParticipacion) ([]ports.SolicitudDocumentalPendienteRRHH, error)
}

type ConsultaSolicitudesDocumentalesAuditada struct {
	consulta    ConsultorSolicitudesDocumentalesRRHH
	registrador vecports.RegistradorIntentosAuditoria
	proceso     string
}

var procesoAuditoriaDocumentales = regexp.MustCompile(`^[a-z][a-z0-9._-]{1,79}$`)

func NuevaConsultaSolicitudesDocumentalesAuditada(consulta ConsultorSolicitudesDocumentalesRRHH,
	registrador vecports.RegistradorIntentosAuditoria, proceso string,
) (*ConsultaSolicitudesDocumentalesAuditada, error) {
	if dependenciaLlamamientoNula(consulta) || dependenciaLlamamientoNula(registrador) || !procesoAuditoriaDocumentales.MatchString(proceso) {
		return nil, ports.ErrSituacionParticipacionNoDisponible
	}
	return &ConsultaSolicitudesDocumentalesAuditada{consulta: consulta, registrador: registrador, proceso: proceso}, nil
}

func (s *ConsultaSolicitudesDocumentalesAuditada) ListarSolicitudesDocumentalesRRHH(ctx context.Context,
	q ports.SolicitudCambiarSituacionParticipacion,
) ([]ports.SolicitudDocumentalPendienteRRHH, error) {
	if ctx == nil || s == nil || dependenciaLlamamientoNula(s.consulta) || dependenciaLlamamientoNula(s.registrador) ||
		!procesoAuditoriaDocumentales.MatchString(s.proceso) || q.Validar() != nil {
		return nil, falloAuditoriaConsultaDocumentales()
	}
	actor, err := q.ResultadoContexto.Clonar()
	vinculo, errVinculo := q.Vinculo.Datos()
	correlacion, errCorrelacion := q.Correlacion.ValorCanonico()
	if err != nil || errVinculo != nil || errCorrelacion != nil || vinculo.Superficie != core.SuperficieAutenticacionInternaCorporativaV1 {
		return nil, falloAuditoriaConsultaDocumentales()
	}
	// Capturar antes del servicio conserva la identidad de este intento. No
	// se vuelve a resolver el actor tras una revocación o desconexión posterior.
	items, causa := s.consulta.ListarSolicitudesDocumentalesRRHH(ctx, q)
	if ctx.Err() != nil {
		causa = ctx.Err()
	} else if causa != nil && len(items) != 0 {
		causa = ports.ErrSituacionParticipacionNoDisponible
	} else if causa == nil {
		causa = validarSolicitudesDocumentalesRRHH(items)
	}
	if causa == nil {
		return items, nil // El servicio ya confirmó la lectura y su auditoría SQL.
	}
	// El retorno anterior incluye el cierre diferido del repositorio. El
	// registrador AD169 usa su propia transacción y plazo independiente de HTTP.
	referencia, err := vecports.NuevaReferenciaIntentoAuditoria()
	if err != nil {
		return nil, falloAuditoriaConsultaDocumentales()
	}
	orden, err := vecports.NuevaOrdenIntentoAuditoria(referencia, actor, q.Vinculo, core.DatosIntentoAuditoria{
		Accion: ports.AccionConsultarSolicitudesDocumentalesRRHH, ModuloID: ports.ModuloSituacionParticipacion,
		RecursoRef: q.ParticipacionRef, FinalidadRef: ports.FinalidadCambiarSituacionParticipacion,
		Resultado: resultadoFalloConsultaDocumentales(causa), Motivo: q.MotivoAutorizacion,
		Proceso: s.proceso, Canal: string(vinculo.Superficie), CorrelacionRef: correlacion,
	})
	if err != nil {
		return nil, falloAuditoriaConsultaDocumentales()
	}
	acuse, err := s.registrador.AppendIntentoAuditoria(context.WithoutCancel(ctx), orden)
	if err != nil || acuse.ValidarPara(orden) != nil {
		return nil, falloAuditoriaConsultaDocumentales()
	}
	return nil, falloConsultaDocumentalesAuditado{causa: causa, acuse: acuse}
}

// Se conserva la forma cerrada que ya exige el transporte. Una proyección
// inválida se registra como fallo y nunca alcanza el serializador HTTP.
func validarSolicitudesDocumentalesRRHH(items []ports.SolicitudDocumentalPendienteRRHH) error {
	for _, item := range items {
		if item.SolicitudRef == "" || item.Version != 1 || item.ContenidoSHA256 == "" || item.DocumentoRef == "" ||
			item.DocumentoSHA256 == "" || item.Estado != "pendiente_rrhh" || item.ReciboRef == "" || item.RegistradaEn.IsZero() {
			return ports.ErrSituacionParticipacionNoDisponible
		}
		if item.FechaFinCausa != "" {
			fecha, err := time.Parse(time.DateOnly, item.FechaFinCausa)
			if err != nil {
				return fmt.Errorf("%w: fecha de proyección inválida: %w", ports.ErrSituacionParticipacionNoDisponible, err)
			}
			if fecha.Format(time.DateOnly) != item.FechaFinCausa {
				return ports.ErrSituacionParticipacionNoDisponible
			}
		}
	}
	return nil
}

func falloAuditoriaConsultaDocumentales() error {
	return errors.Join(ports.ErrSituacionParticipacionNoDisponible, vecports.ErrIntentoAuditoriaNoDisponible)
}

func resultadoFalloConsultaDocumentales(causa error) core.ResultadoIntentoAuditoria {
	for _, tecnico := range []error{context.Canceled, context.DeadlineExceeded, ErrCambioSituacionParticipacionNoDisponible,
		ports.ErrSituacionParticipacionNoDisponible, vecports.ErrFuenteAutorizacionNoDisponible,
		vecports.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible, vecports.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible} {
		if errors.Is(causa, tecnico) {
			return core.ResultadoIntentoAuditoriaError
		}
	}
	if errors.Is(causa, core.ErrAutorizacionDenegada) || errors.Is(causa, core.ErrPermissionDenied) {
		return core.ResultadoIntentoAuditoriaDenegado
	}
	return core.ResultadoIntentoAuditoriaError
}

type falloConsultaDocumentalesAuditado struct {
	causa error
	acuse vecports.AcuseIntentoAuditoria
}

func (falloConsultaDocumentalesAuditado) Error() string {
	return "bolsa: consulta documental fallida auditada"
}
func (e falloConsultaDocumentalesAuditado) Unwrap() error { return e.causa }

func AcuseConsultaDocumentalesFallida(err error) (vecports.AcuseIntentoAuditoria, bool) {
	var auditado falloConsultaDocumentalesAuditado
	if !errors.As(err, &auditado) {
		return vecports.AcuseIntentoAuditoria{}, false
	}
	return auditado.acuse, true
}
