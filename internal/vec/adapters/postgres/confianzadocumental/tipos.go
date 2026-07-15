package confianzadocumental

import (
	"fmt"
	"io"
	"log/slog"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// AudienciaCOSEDocumental es cerrada porque cada valor separa un protocolo de
// seguridad distinto mediante external_aad. No se aceptan cadenas libres.
type AudienciaCOSEDocumental string

const (
	AudienciaCOSEReciboComponenteDocumental AudienciaCOSEDocumental = "recibo_componente"
	AudienciaCOSETokenCercadoDocumental     AudienciaCOSEDocumental = "token_cercado"
	AudienciaCOSEEvidenciaDocumental        AudienciaCOSEDocumental = "evidencia"
	AudienciaCOSEReconciliacionDocumental   AudienciaCOSEDocumental = "reconciliacion"
	AudienciaCOSEEscrituraAlmacenDocumental AudienciaCOSEDocumental = "escritura_almacen"
	AudienciaCOSEAtestacionAutorizacionPDP  AudienciaCOSEDocumental = "atestacion_autorizacion_pdp"
)

func (a AudienciaCOSEDocumental) valida() bool {
	switch a {
	case AudienciaCOSEReciboComponenteDocumental,
		AudienciaCOSETokenCercadoDocumental,
		AudienciaCOSEEvidenciaDocumental,
		AudienciaCOSEReconciliacionDocumental,
		AudienciaCOSEEscrituraAlmacenDocumental,
		AudienciaCOSEAtestacionAutorizacionPDP:
		return true
	default:
		return false
	}
}

// SolicitudVerificacionCOSESign1 fija los bytes exactos que deben estar
// firmados y una audiencia cerrada. No contiene un instante aportado por el
// llamador: Servicio lo obtiene de su reloj interno confiable.
type SolicitudVerificacionCOSESign1 struct {
	payloadEsperado []byte
	audiencia       AudienciaCOSEDocumental
}

func NuevaSolicitudVerificacionCOSESign1(
	payloadEsperado []byte,
	audiencia AudienciaCOSEDocumental,
) (SolicitudVerificacionCOSESign1, error) {
	solicitud := SolicitudVerificacionCOSESign1{
		payloadEsperado: append([]byte(nil), payloadEsperado...),
		audiencia:       audiencia,
	}
	if solicitud.Validar() != nil {
		return SolicitudVerificacionCOSESign1{}, ErrSolicitudVerificacionCOSESign1Invalida
	}
	return solicitud, nil
}

func (s SolicitudVerificacionCOSESign1) Validar() error {
	limite, audienciaValida := limitePayloadPorAudiencia(s.audiencia)
	if !audienciaValida || len(s.payloadEsperado) == 0 || len(s.payloadEsperado) > limite {
		return ErrSolicitudVerificacionCOSESign1Invalida
	}
	return nil
}

func limitePayloadPorAudiencia(audiencia AudienciaCOSEDocumental) (int, bool) {
	if !audiencia.valida() {
		return 0, false
	}
	if audiencia == AudienciaCOSEAtestacionAutorizacionPDP {
		return domain.TamanoMaximoMensajeAtestacionAutorizacionV1, true
	}
	return maximoBytesPayloadDocumentalV4, true
}

func (s SolicitudVerificacionCOSESign1) PayloadEsperado() ([]byte, error) {
	if s.Validar() != nil {
		return nil, ErrSolicitudVerificacionCOSESign1Invalida
	}
	return append([]byte(nil), s.payloadEsperado...), nil
}

func (s SolicitudVerificacionCOSESign1) Audiencia() (AudienciaCOSEDocumental, error) {
	if s.Validar() != nil {
		return "", ErrSolicitudVerificacionCOSESign1Invalida
	}
	return s.audiencia, nil
}

// AADExterno devuelve la vinculacion canonica que debe usar el firmante.
func (s SolicitudVerificacionCOSESign1) AADExterno() ([]byte, error) {
	if s.Validar() != nil {
		return nil, ErrSolicitudVerificacionCOSESign1Invalida
	}
	return []byte(prefijoAADDocumentalV4 + string(s.audiencia)), nil
}

func (SolicitudVerificacionCOSESign1) String() string {
	return "[SOLICITUD-VERIFICACION-COSE-SIGN1-DOCUMENTAL-REDACTADA]"
}

func (s SolicitudVerificacionCOSESign1) GoString() string { return s.String() }
func (s SolicitudVerificacionCOSESign1) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, s.String())
}
func (s SolicitudVerificacionCOSESign1) LogValue() slog.Value {
	return slog.StringValue(s.String())
}
func (SolicitudVerificacionCOSESign1) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionAutoridadCOSESign1Prohibida
}
func (*SolicitudVerificacionCOSESign1) UnmarshalJSON([]byte) error {
	return ErrSerializacionAutoridadCOSESign1Prohibida
}
func (SolicitudVerificacionCOSESign1) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionAutoridadCOSESign1Prohibida
}
func (*SolicitudVerificacionCOSESign1) UnmarshalText([]byte) error {
	return ErrSerializacionAutoridadCOSESign1Prohibida
}
func (SolicitudVerificacionCOSESign1) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionAutoridadCOSESign1Prohibida
}
func (*SolicitudVerificacionCOSESign1) UnmarshalBinary([]byte) error {
	return ErrSerializacionAutoridadCOSESign1Prohibida
}

// PruebaCOSESign1DocumentalVerificada es autoridad local opaca. No existe
// constructor publico y nunca conserva el payload ni el sobre firmado.
type PruebaCOSESign1DocumentalVerificada struct {
	marca                     string
	algoritmo                 AlgoritmoCOSEDocumental
	claveID                   []byte
	huellaClaveSHA256         string
	estadoConfianza           EstadoConfianzaClaveDocumental
	audiencia                 AudienciaCOSEDocumental
	huellaPayloadSHA256       string
	huellaSobreSHA256         string
	verificadaEn              time.Time
	raizValidaDesde           time.Time
	raizValidaHasta           time.Time
	revisionConfianza         string
	huellaConfiguracionSHA256 string
	configuracionPublicadaEn  time.Time
	configuracionExpiraEn     time.Time
}

type comprobacionCriptograficaSatisfactoria struct {
	algoritmo                 AlgoritmoCOSEDocumental
	claveID                   []byte
	huellaClaveSHA256         string
	estadoConfianza           EstadoConfianzaClaveDocumental
	audiencia                 AudienciaCOSEDocumental
	huellaPayloadSHA256       string
	huellaSobreSHA256         string
	verificadaEn              time.Time
	raizValidaDesde           time.Time
	raizValidaHasta           time.Time
	revisionConfianza         string
	huellaConfiguracionSHA256 string
	configuracionPublicadaEn  time.Time
	configuracionExpiraEn     time.Time
}

func nuevaPruebaCOSESign1DocumentalVerificada(
	comprobacion comprobacionCriptograficaSatisfactoria,
) (PruebaCOSESign1DocumentalVerificada, error) {
	prueba := PruebaCOSESign1DocumentalVerificada{
		marca:                     marcaPruebaCOSESign1VerificadaV4,
		algoritmo:                 comprobacion.algoritmo,
		claveID:                   append([]byte(nil), comprobacion.claveID...),
		huellaClaveSHA256:         comprobacion.huellaClaveSHA256,
		estadoConfianza:           comprobacion.estadoConfianza,
		audiencia:                 comprobacion.audiencia,
		huellaPayloadSHA256:       comprobacion.huellaPayloadSHA256,
		huellaSobreSHA256:         comprobacion.huellaSobreSHA256,
		verificadaEn:              comprobacion.verificadaEn,
		raizValidaDesde:           comprobacion.raizValidaDesde,
		raizValidaHasta:           comprobacion.raizValidaHasta,
		revisionConfianza:         comprobacion.revisionConfianza,
		huellaConfiguracionSHA256: comprobacion.huellaConfiguracionSHA256,
		configuracionPublicadaEn:  comprobacion.configuracionPublicadaEn,
		configuracionExpiraEn:     comprobacion.configuracionExpiraEn,
	}
	if prueba.Validar() != nil {
		return PruebaCOSESign1DocumentalVerificada{}, ErrPruebaCOSESign1VerificadaInvalida
	}
	return prueba, nil
}

func (p PruebaCOSESign1DocumentalVerificada) Validar() error {
	if p.marca != marcaPruebaCOSESign1VerificadaV4 || !p.algoritmo.valido() ||
		!claveIDDocumentalValida(p.claveID) || !huellaSHA256DocumentalValida(p.huellaClaveSHA256) ||
		p.estadoConfianza != EstadoConfianzaClaveDocumentalActiva || !p.audiencia.valida() ||
		!huellaSHA256DocumentalValida(p.huellaPayloadSHA256) ||
		!huellaSHA256DocumentalValida(p.huellaSobreSHA256) ||
		!instanteCanonicoDocumental(p.verificadaEn) || !instanteCanonicoDocumental(p.raizValidaDesde) ||
		!instanteCanonicoDocumental(p.raizValidaHasta) ||
		!p.raizValidaHasta.After(p.raizValidaDesde) || p.verificadaEn.Before(p.raizValidaDesde) ||
		!p.verificadaEn.Before(p.raizValidaHasta) ||
		!referenciaConfiguracionDocumentalValida(p.revisionConfianza) ||
		!huellaSHA256DocumentalValida(p.huellaConfiguracionSHA256) ||
		!instanteCanonicoDocumental(p.configuracionPublicadaEn) ||
		!instanteCanonicoDocumental(p.configuracionExpiraEn) ||
		!p.configuracionExpiraEn.After(p.configuracionPublicadaEn) ||
		p.configuracionExpiraEn.Sub(p.configuracionPublicadaEn) >
			maximaVigenciaConfiguracionConfianzaV4 ||
		p.verificadaEn.Before(p.configuracionPublicadaEn) ||
		!p.verificadaEn.Before(p.configuracionExpiraEn) {
		return ErrPruebaCOSESign1VerificadaInvalida
	}
	return nil
}

func (p PruebaCOSESign1DocumentalVerificada) ValidarPara(
	solicitud SolicitudVerificacionCOSESign1,
	sobre ports.SobreCriptograficoDocumentalCrudoV4,
) error {
	huellaSobre, err := sobre.HuellaSHA256()
	if p.Validar() != nil || solicitud.Validar() != nil || err != nil ||
		p.audiencia != solicitud.audiencia ||
		p.huellaPayloadSHA256 != huellaBytesDocumentales(solicitud.payloadEsperado) ||
		p.huellaSobreSHA256 != huellaSobre {
		return ErrPruebaCOSESign1VerificadaInvalida
	}
	return nil
}

func (p PruebaCOSESign1DocumentalVerificada) Algoritmo() (AlgoritmoCOSEDocumental, error) {
	if p.Validar() != nil {
		return "", ErrPruebaCOSESign1VerificadaInvalida
	}
	return p.algoritmo, nil
}

func (p PruebaCOSESign1DocumentalVerificada) ClaveID() ([]byte, error) {
	if p.Validar() != nil {
		return nil, ErrPruebaCOSESign1VerificadaInvalida
	}
	return append([]byte(nil), p.claveID...), nil
}

func (p PruebaCOSESign1DocumentalVerificada) Audiencia() (AudienciaCOSEDocumental, error) {
	if p.Validar() != nil {
		return "", ErrPruebaCOSESign1VerificadaInvalida
	}
	return p.audiencia, nil
}

func (p PruebaCOSESign1DocumentalVerificada) HuellaPayloadSHA256() (string, error) {
	if p.Validar() != nil {
		return "", ErrPruebaCOSESign1VerificadaInvalida
	}
	return p.huellaPayloadSHA256, nil
}

func (p PruebaCOSESign1DocumentalVerificada) HuellaSobreSHA256() (string, error) {
	if p.Validar() != nil {
		return "", ErrPruebaCOSESign1VerificadaInvalida
	}
	return p.huellaSobreSHA256, nil
}

func (p PruebaCOSESign1DocumentalVerificada) VerificadaEn() (time.Time, error) {
	if p.Validar() != nil {
		return time.Time{}, ErrPruebaCOSESign1VerificadaInvalida
	}
	return p.verificadaEn, nil
}

func (p PruebaCOSESign1DocumentalVerificada) RevisionConfianza() (string, error) {
	if p.Validar() != nil {
		return "", ErrPruebaCOSESign1VerificadaInvalida
	}
	return p.revisionConfianza, nil
}

func (p PruebaCOSESign1DocumentalVerificada) HuellaConfiguracionConfianzaSHA256() (string, error) {
	if p.Validar() != nil {
		return "", ErrPruebaCOSESign1VerificadaInvalida
	}
	return p.huellaConfiguracionSHA256, nil
}

func (PruebaCOSESign1DocumentalVerificada) String() string {
	return "[AUTORIDAD-COSE-SIGN1-DOCUMENTAL-VERIFICADA-REDACTADA]"
}

func (p PruebaCOSESign1DocumentalVerificada) GoString() string { return p.String() }
func (p PruebaCOSESign1DocumentalVerificada) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, p.String())
}
func (p PruebaCOSESign1DocumentalVerificada) LogValue() slog.Value {
	return slog.StringValue(p.String())
}
func (PruebaCOSESign1DocumentalVerificada) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionAutoridadCOSESign1Prohibida
}
func (*PruebaCOSESign1DocumentalVerificada) UnmarshalJSON([]byte) error {
	return ErrSerializacionAutoridadCOSESign1Prohibida
}
func (PruebaCOSESign1DocumentalVerificada) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionAutoridadCOSESign1Prohibida
}
func (*PruebaCOSESign1DocumentalVerificada) UnmarshalText([]byte) error {
	return ErrSerializacionAutoridadCOSESign1Prohibida
}
func (PruebaCOSESign1DocumentalVerificada) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionAutoridadCOSESign1Prohibida
}
func (*PruebaCOSESign1DocumentalVerificada) UnmarshalBinary([]byte) error {
	return ErrSerializacionAutoridadCOSESign1Prohibida
}
