package ports

import (
	"fmt"
	"io"
	"log/slog"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
)

type ReciboConsumoVerificacionConvocatoria struct {
	bloqueoSerializacionGobiernoConvocatoria
	TokenConsumoRef        string
	AtestacionRef          string
	HuellaAtestacionSHA256 string
}

func (r ReciboConsumoVerificacionConvocatoria) validar() bool {
	return referenciaGobiernoConvocatoriaValida(r.TokenConsumoRef) &&
		referenciaGobiernoConvocatoriaValida(r.AtestacionRef) &&
		huellaGobiernoConvocatoriaValida(r.HuellaAtestacionSHA256) &&
		r.TokenConsumoRef != r.AtestacionRef
}

// ReciboGobiernoConvocatoria es la prueba minima devuelta tras COMMIT. Liga
// el estado confirmado con la decision consumida, la intencion idempotente,
// el registro de auditoria y el evento outbox. No contiene atributos personales
// directos; es interno y PrincipalRef sigue siendo una referencia seudonima.
type ReciboGobiernoConvocatoria struct {
	bloqueoSerializacionGobiernoConvocatoria
	TransaccionRef                     string
	Accion                             string
	EstadoPrincipal                    ReferenciaEstadoVersionConvocatoria
	EstadoRelacionado                  *ReferenciaEstadoVersionConvocatoria
	PrincipalRef                       string
	AutorizacionRef                    string
	HuellaAutorizacionSHA256           string
	AtestacionAutorizacionRef          string
	HuellaAtestacionAutorizacionSHA256 string
	ConsumoAutorizacionRef             string
	IndiceIdempotenciaHMACSHA256       string
	AtestacionIdempotenciaRef          string
	HuellaAtestacionIdempotenciaSHA256 string
	HuellaIntencionSHA256              string
	AuditoriaRef                       string
	HuellaAuditoriaSHA256              string
	EventoOutboxRef                    string
	HuellaEventoOutboxSHA256           string
	ConsumoMotivo                      *ReciboConsumoVerificacionConvocatoria
	ConsumoDependencias                *ReciboConsumoVerificacionConvocatoria
	ConsumoAprobacion                  *ReciboConsumoVerificacionConvocatoria
	ConfirmadaEn                       time.Time
}

func (ReciboGobiernoConvocatoria) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionGobiernoConvocatoriaProhibida
}

func (ReciboGobiernoConvocatoria) String() string {
	return "[RECIBO-GOBIERNO-CONVOCATORIA-INTERNO]"
}

func (r ReciboGobiernoConvocatoria) GoString() string { return r.String() }
func (r ReciboGobiernoConvocatoria) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, r.String())
}
func (r ReciboGobiernoConvocatoria) LogValue() slog.Value {
	return slog.StringValue(r.String())
}

func (r ReciboGobiernoConvocatoria) ValidarPara(
	preparacion PreparacionTransaccionGobiernoConvocatoria,
	versionConfirmada dominiobolsa.VersionConvocatoriaGobernada,
) error {
	datosAutorizacion, errAutorizacion := preparacion.Autorizacion.Datos()
	datosIdempotencia, errIdempotencia := preparacion.Idempotencia.Datos()
	datosSellado, errSellado := preparacion.SelladoMotivo.DatosParaConsumo()
	huellaIntencion, errHuella := preparacion.Material.HuellaSHA256()
	if preparacion.validarEn(versionConfirmada, r.ConfirmadaEn) != nil || errAutorizacion != nil ||
		errIdempotencia != nil || errSellado != nil || errHuella != nil ||
		!referenciaGobiernoConvocatoriaValida(r.TransaccionRef) ||
		r.Accion != preparacion.Material.Accion ||
		r.EstadoPrincipal != preparacion.Material.EstadoPrincipalNuevo ||
		!estadoRelacionadoReciboCoincide(r.EstadoRelacionado, preparacion.Material.EstadoRelacionadoNuevo) ||
		r.PrincipalRef != datosAutorizacion.Decision.PrincipalID ||
		r.AutorizacionRef != datosAutorizacion.Decision.DecisionRef ||
		r.HuellaAutorizacionSHA256 != datosAutorizacion.HuellaDecisionSHA256 ||
		!referenciaGobiernoConvocatoriaValida(r.AtestacionAutorizacionRef) ||
		!huellaGobiernoConvocatoriaValida(r.HuellaAtestacionAutorizacionSHA256) ||
		!referenciaGobiernoConvocatoriaValida(r.ConsumoAutorizacionRef) ||
		r.IndiceIdempotenciaHMACSHA256 != datosIdempotencia.IndiceOperacionHMACSHA256 ||
		r.AtestacionIdempotenciaRef != datosIdempotencia.AtestacionRef ||
		r.HuellaAtestacionIdempotenciaSHA256 != datosIdempotencia.HuellaAtestacionSHA256 ||
		r.HuellaIntencionSHA256 != huellaIntencion ||
		r.ConsumoMotivo == nil || *r.ConsumoMotivo != reciboConsumoMotivo(datosSellado) ||
		!referenciaGobiernoConvocatoriaValida(r.AuditoriaRef) ||
		!huellaGobiernoConvocatoriaValida(r.HuellaAuditoriaSHA256) ||
		!referenciaGobiernoConvocatoriaValida(r.EventoOutboxRef) ||
		!huellaGobiernoConvocatoriaValida(r.HuellaEventoOutboxSHA256) ||
		!referenciasReciboGobiernoConvocatoriaDistintas(r) ||
		!instanteGobiernoConvocatoriaCanonico(r.ConfirmadaEn) {
		return ErrReciboGobiernoConvocatoriaInvalido
	}
	if !consumosReciboValidosParaAccion(r) {
		return ErrReciboGobiernoConvocatoriaInvalido
	}
	return nil
}

func referenciasReciboGobiernoConvocatoriaDistintas(r ReciboGobiernoConvocatoria) bool {
	referencias := []string{
		r.TransaccionRef, r.AutorizacionRef, r.AtestacionAutorizacionRef,
		r.ConsumoAutorizacionRef, r.AtestacionIdempotenciaRef,
		r.AuditoriaRef, r.EventoOutboxRef,
	}
	if r.ConsumoDependencias != nil {
		referencias = append(referencias,
			r.ConsumoDependencias.TokenConsumoRef, r.ConsumoDependencias.AtestacionRef,
		)
	}
	if r.ConsumoMotivo != nil {
		referencias = append(referencias,
			r.ConsumoMotivo.TokenConsumoRef, r.ConsumoMotivo.AtestacionRef,
		)
	}
	if r.ConsumoAprobacion != nil {
		referencias = append(referencias,
			r.ConsumoAprobacion.TokenConsumoRef, r.ConsumoAprobacion.AtestacionRef,
		)
	}
	return referenciasGobiernoConvocatoriaDistintas(referencias...)
}

func consumosReciboValidosParaAccion(r ReciboGobiernoConvocatoria) bool {
	publicacion := r.Accion == AccionPublicarVersionConvocatoria ||
		r.Accion == AccionPublicarYSustituirConvocatoria ||
		r.Accion == AccionPublicarTrasRetiradaConvocatoria
	if publicacion {
		return r.ConsumoDependencias != nil && r.ConsumoDependencias.validar() &&
			r.ConsumoAprobacion != nil && r.ConsumoAprobacion.validar() &&
			r.ConsumoDependencias.TokenConsumoRef != r.ConsumoAprobacion.TokenConsumoRef
	}
	if r.Accion == AccionRetirarVersionConvocatoria {
		return r.ConsumoDependencias == nil && r.ConsumoAprobacion != nil && r.ConsumoAprobacion.validar()
	}
	return r.ConsumoDependencias == nil && r.ConsumoAprobacion == nil
}

func estadoRelacionadoReciboCoincide(
	recibido, esperado *ReferenciaEstadoVersionConvocatoria,
) bool {
	if recibido == nil || esperado == nil {
		return recibido == nil && esperado == nil
	}
	return *recibido == *esperado
}

func reciboConsumoDependencias(
	datos DatosAtestacionDependenciasConvocatoria,
) ReciboConsumoVerificacionConvocatoria {
	return ReciboConsumoVerificacionConvocatoria{
		TokenConsumoRef: datos.TokenConsumoRef, AtestacionRef: datos.AtestacionRef,
		HuellaAtestacionSHA256: datos.HuellaAtestacionSHA256,
	}
}

func reciboConsumoMotivo(
	datos DatosAtestacionSelladoMotivoConvocatoria,
) ReciboConsumoVerificacionConvocatoria {
	return ReciboConsumoVerificacionConvocatoria{
		TokenConsumoRef: datos.TokenConsumoRef, AtestacionRef: datos.AtestacionRef,
		HuellaAtestacionSHA256: datos.HuellaAtestacionSHA256,
	}
}

func reciboConsumoAprobacion(
	datos DatosAtestacionAprobacionConvocatoria,
) ReciboConsumoVerificacionConvocatoria {
	return ReciboConsumoVerificacionConvocatoria{
		TokenConsumoRef: datos.TokenConsumoRef, AtestacionRef: datos.AtestacionRef,
		HuellaAtestacionSHA256: datos.HuellaAtestacionSHA256,
	}
}
