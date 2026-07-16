package ports

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"strconv"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
)

const (
	EsquemaManifiestoProbatorioBaremacion   = "vec.bolsa.manifiesto_probatorio"
	FinalidadManifiestoProbatorioBaremacion = "decision_tecnica_baremacion"
	VersionManifiestoProbatorioBaremacion   = 2
)

// TipoEvidenciaProbatoriaBaremacion es un catalogo cerrado. Una evidencia no
// puede incorporarse al manifiesto bajo una etiqueta libre o ambigua.
type TipoEvidenciaProbatoriaBaremacion string

const (
	EvidenciaEstadoBaseBaremacion                 TipoEvidenciaProbatoriaBaremacion = "estado_base"
	EvidenciaCalculoOficialBaremacion             TipoEvidenciaProbatoriaBaremacion = "calculo_oficial"
	EvidenciaCriterioPublicadoBaremacion          TipoEvidenciaProbatoriaBaremacion = "criterio_publicado"
	EvidenciaDocumentoMeritoBaremacion            TipoEvidenciaProbatoriaBaremacion = "documento_merito"
	EvidenciaRepresentacionBaremacion             TipoEvidenciaProbatoriaBaremacion = "representacion_documento"
	EvidenciaContenidoDecisionBaremacion          TipoEvidenciaProbatoriaBaremacion = "contenido_decision"
	EvidenciaPoliticaFirmaBaremacion              TipoEvidenciaProbatoriaBaremacion = "politica_firma"
	EvidenciaDocumentoCanonicoBaremacion          TipoEvidenciaProbatoriaBaremacion = "documento_canonico"
	EvidenciaCustodiaFirmableBaremacion           TipoEvidenciaProbatoriaBaremacion = "custodia_firmable"
	EvidenciaPreparacionFirmaBaremacion           TipoEvidenciaProbatoriaBaremacion = "preparacion_firma"
	EvidenciaConsultaFirmaBaremacion              TipoEvidenciaProbatoriaBaremacion = "consulta_firma"
	EvidenciaValidacionInicialBaremacion          TipoEvidenciaProbatoriaBaremacion = "validacion_firma_inicial"
	EvidenciaSelloTiempoBaremacion                TipoEvidenciaProbatoriaBaremacion = "sello_tiempo"
	EvidenciaVinculoRevisionSelladaBaremacion     TipoEvidenciaProbatoriaBaremacion = "vinculo_revision_sellada"
	EvidenciaValidacionDocumentoSelladoBaremacion TipoEvidenciaProbatoriaBaremacion = "validacion_documento_sellado"
	EvidenciaAumentoLongevidadBaremacion          TipoEvidenciaProbatoriaBaremacion = "aumento_longevidad"
	EvidenciaVinculoRevisionLongevaBaremacion     TipoEvidenciaProbatoriaBaremacion = "vinculo_revision_longeva"
	EvidenciaValidacionFinalBaremacion            TipoEvidenciaProbatoriaBaremacion = "validacion_firma_final"
	EvidenciaRecuperacionFirmadoBaremacion        TipoEvidenciaProbatoriaBaremacion = "recuperacion_documento_firmado"
	EvidenciaCustodiaFirmadoBaremacion            TipoEvidenciaProbatoriaBaremacion = "custodia_documento_firmado"
	EvidenciaRetencionFirmadoBaremacion           TipoEvidenciaProbatoriaBaremacion = "retencion_documento_firmado"
)

func (t TipoEvidenciaProbatoriaBaremacion) valida() bool {
	switch t {
	case EvidenciaEstadoBaseBaremacion, EvidenciaCalculoOficialBaremacion,
		EvidenciaCriterioPublicadoBaremacion, EvidenciaDocumentoMeritoBaremacion,
		EvidenciaRepresentacionBaremacion, EvidenciaContenidoDecisionBaremacion,
		EvidenciaPoliticaFirmaBaremacion, EvidenciaDocumentoCanonicoBaremacion,
		EvidenciaCustodiaFirmableBaremacion, EvidenciaPreparacionFirmaBaremacion,
		EvidenciaConsultaFirmaBaremacion, EvidenciaValidacionInicialBaremacion,
		EvidenciaSelloTiempoBaremacion, EvidenciaVinculoRevisionSelladaBaremacion,
		EvidenciaValidacionDocumentoSelladoBaremacion, EvidenciaAumentoLongevidadBaremacion,
		EvidenciaVinculoRevisionLongevaBaremacion,
		EvidenciaValidacionFinalBaremacion, EvidenciaRecuperacionFirmadoBaremacion,
		EvidenciaCustodiaFirmadoBaremacion, EvidenciaRetencionFirmadoBaremacion:
		return true
	default:
		return false
	}
}

// AutorizacionProbatoriaBaremacion conserva el permiso positivo exacto usado
// por cada PEP. La secuencia impide reordenar o omitir pasos silenciosamente.
type AutorizacionProbatoriaBaremacion struct {
	Secuencia       uint32
	Accion          AccionOperacionBaremacion
	ClaseRecurso    ClaseRecursoOperacionBaremacion
	RecursoRef      string
	AutorizacionRef string
}

func (a AutorizacionProbatoriaBaremacion) Validar() error {
	especificacion, existe := especificacionesAccionBaremacion[a.Accion]
	if a.Secuencia < 1 || !existe || a.ClaseRecurso != especificacion.clase ||
		!referenciaValida(a.RecursoRef, 512) || !referenciaValida(a.AutorizacionRef, 512) {
		return ErrSolicitudBaremacionInvalida
	}
	return nil
}

type EvidenciaProbatoriaBaremacion struct {
	Secuencia             uint32
	Tipo                  TipoEvidenciaProbatoriaBaremacion
	Referencia            string
	HuellaEvidenciaSHA256 string
}

func (e EvidenciaProbatoriaBaremacion) Validar() error {
	if e.Secuencia < 1 || !e.Tipo.valida() || !referenciaValida(e.Referencia, 512) ||
		!huellaSHA256Valida(e.HuellaEvidenciaSHA256) {
		return ErrSolicitudBaremacionInvalida
	}
	return nil
}

// ManifiestoProbatorioBaremacion es el indice sellado de capacidades y
// evidencias que sostienen una decision. No acepta mapas ni extensiones libres.
type ManifiestoProbatorioBaremacion struct {
	Esquema                   string
	Finalidad                 string
	VersionEsquema            int
	Referencia                string
	ProcesoRef                string
	SolicitudRef              string
	SujetoRef                 string
	BaremacionMeritoRef       string
	DecisionRef               string
	VersionBase               uint64
	HuellaVersionBaseSHA256   string
	Autorizaciones            []AutorizacionProbatoriaBaremacion
	Evidencias                []EvidenciaProbatoriaBaremacion
	CreadoEn                  time.Time
	HuellaManifiestoSHA256    string
	SelloManifiestoHMACSHA256 string
}

func (m ManifiestoProbatorioBaremacion) Clonar() ManifiestoProbatorioBaremacion {
	clon := m
	clon.Autorizaciones = append([]AutorizacionProbatoriaBaremacion(nil), m.Autorizaciones...)
	clon.Evidencias = append([]EvidenciaProbatoriaBaremacion(nil), m.Evidencias...)
	clon.CreadoEn = m.CreadoEn.UTC()
	return clon
}

func (m ManifiestoProbatorioBaremacion) Validar() error {
	if m.validarContenidoConHuella() != nil || !huellaHMACSHA256Valida(m.SelloManifiestoHMACSHA256) {
		return ErrSolicitudBaremacionInvalida
	}
	return nil
}

func (m ManifiestoProbatorioBaremacion) validarContenidoConHuella() error {
	if m.validarContenido() != nil || !huellaSHA256Valida(m.HuellaManifiestoSHA256) {
		return ErrSolicitudBaremacionInvalida
	}
	huella, err := m.huellaCalculada()
	if err != nil || huella != m.HuellaManifiestoSHA256 {
		return ErrSolicitudBaremacionInvalida
	}
	return nil
}

func (m ManifiestoProbatorioBaremacion) validarContenido() error {
	if m.Esquema != EsquemaManifiestoProbatorioBaremacion ||
		m.Finalidad != FinalidadManifiestoProbatorioBaremacion ||
		m.VersionEsquema != VersionManifiestoProbatorioBaremacion ||
		!referenciaValida(m.Referencia, 512) || !referenciaValida(m.ProcesoRef, 512) ||
		!referenciaValida(m.SolicitudRef, 512) || !referenciaValida(m.SujetoRef, 512) ||
		!referenciaValida(m.BaremacionMeritoRef, 512) || !referenciaValida(m.DecisionRef, 512) ||
		m.VersionBase < 1 || !huellaSHA256Valida(m.HuellaVersionBaseSHA256) ||
		len(m.Autorizaciones) == 0 || len(m.Autorizaciones) > 4096 ||
		len(m.Evidencias) == 0 || len(m.Evidencias) > 4096 || m.CreadoEn.IsZero() ||
		m.CreadoEn.Location() != time.UTC {
		return ErrSolicitudBaremacionInvalida
	}
	autorizaciones := make(map[string]struct{}, len(m.Autorizaciones))
	for indice, autorizacion := range m.Autorizaciones {
		if autorizacion.Validar() != nil || autorizacion.Secuencia != uint32(indice+1) {
			return ErrSolicitudBaremacionInvalida
		}
		if _, repetida := autorizaciones[autorizacion.AutorizacionRef]; repetida {
			return ErrSolicitudBaremacionInvalida
		}
		autorizaciones[autorizacion.AutorizacionRef] = struct{}{}
	}
	evidencias := make(map[string]struct{}, len(m.Evidencias))
	for indice, evidencia := range m.Evidencias {
		if evidencia.Validar() != nil || evidencia.Secuencia != uint32(indice+1) {
			return ErrSolicitudBaremacionInvalida
		}
		clave := string(evidencia.Tipo) + "\x00" + evidencia.Referencia
		if _, repetida := evidencias[clave]; repetida {
			return ErrSolicitudBaremacionInvalida
		}
		evidencias[clave] = struct{}{}
	}
	if _, err := m.validarCoberturaCanonica(); err != nil {
		return err
	}
	return nil
}

func (m ManifiestoProbatorioBaremacion) ValidarPara(
	version ReferenciaVersionBaremacion,
	contenido dominiobolsa.ContenidoDecisionTecnica,
) error {
	_, err := m.validarCoberturaPara(version, contenido)
	return err
}

// PrepararSellado calcula la huella cerrada y devuelve exactamente los bytes
// que debe autenticar el sellador institucional.
func (m ManifiestoProbatorioBaremacion) PrepararSellado() (ManifiestoProbatorioBaremacion, CargaProtegida, error) {
	if m.HuellaManifiestoSHA256 != "" || m.SelloManifiestoHMACSHA256 != "" || m.CreadoEn.IsZero() ||
		m.CreadoEn.Location() != time.UTC {
		return ManifiestoProbatorioBaremacion{}, CargaProtegida{}, ErrSolicitudBaremacionInvalida
	}
	if _, err := m.validarCoberturaCanonica(); err != nil {
		return ManifiestoProbatorioBaremacion{}, CargaProtegida{}, err
	}
	huella, err := m.huellaCalculada()
	if err != nil {
		return ManifiestoProbatorioBaremacion{}, CargaProtegida{}, err
	}
	preparado := m.Clonar()
	preparado.HuellaManifiestoSHA256 = huella
	carga, err := RepresentacionCanonicaManifiestoProbatorioBaremacion(preparado)
	if err != nil {
		return ManifiestoProbatorioBaremacion{}, CargaProtegida{}, err
	}
	return preparado, carga, nil
}

// RepresentacionCanonicaManifiestoProbatorioBaremacion reconstruye los bytes
// exactos que autentica el sellador. Admite el manifiesto preparado o ya
// sellado porque el propio sello nunca forma parte de la carga autenticada.
// La finalidad criptografica exclusiva encierra el material funcional y evita
// reutilizar el HMAC valido de otro contrato aunque comparta campos.
func RepresentacionCanonicaManifiestoProbatorioBaremacion(
	manifiesto ManifiestoProbatorioBaremacion,
) (CargaProtegida, error) {
	if manifiesto.validarContenidoConHuella() != nil ||
		(manifiesto.SelloManifiestoHMACSHA256 != "" &&
			!huellaHMACSHA256Valida(manifiesto.SelloManifiestoHMACSHA256)) {
		return CargaProtegida{}, ErrSolicitudBaremacionInvalida
	}
	material, err := manifiesto.materialCanonico(true)
	if err != nil {
		return CargaProtegida{}, ErrSolicitudBaremacionInvalida
	}
	return cargaPartesCanonicas([]string{
		string(FinalidadSelloManifiestoProbatorioBaremacionV2),
		string(material),
	})
}

func (m ManifiestoProbatorioBaremacion) IncorporarSello(sello string) (ManifiestoProbatorioBaremacion, error) {
	if !huellaSHA256Valida(m.HuellaManifiestoSHA256) || m.SelloManifiestoHMACSHA256 != "" ||
		!huellaHMACSHA256Valida(sello) {
		return ManifiestoProbatorioBaremacion{}, ErrSolicitudBaremacionInvalida
	}
	resultado := m.Clonar()
	resultado.SelloManifiestoHMACSHA256 = sello
	if err := resultado.Validar(); err != nil {
		return ManifiestoProbatorioBaremacion{}, err
	}
	return resultado, nil
}

func (m ManifiestoProbatorioBaremacion) huellaCalculada() (string, error) {
	material, err := m.materialCanonico(false)
	if err != nil {
		return "", err
	}
	huella := sha256.Sum256(material)
	return hex.EncodeToString(huella[:]), nil
}

func (m ManifiestoProbatorioBaremacion) materialCanonico(incluirHuella bool) ([]byte, error) {
	if m.Esquema != EsquemaManifiestoProbatorioBaremacion ||
		m.Finalidad != FinalidadManifiestoProbatorioBaremacion ||
		m.VersionEsquema != VersionManifiestoProbatorioBaremacion ||
		!referenciaValida(m.Referencia, 512) || !referenciaValida(m.ProcesoRef, 512) ||
		!referenciaValida(m.SolicitudRef, 512) || !referenciaValida(m.SujetoRef, 512) ||
		!referenciaValida(m.BaremacionMeritoRef, 512) || !referenciaValida(m.DecisionRef, 512) ||
		m.VersionBase < 1 || !huellaSHA256Valida(m.HuellaVersionBaseSHA256) ||
		len(m.Autorizaciones) == 0 || len(m.Autorizaciones) > 4096 ||
		len(m.Evidencias) == 0 || len(m.Evidencias) > 4096 || m.CreadoEn.IsZero() ||
		m.CreadoEn.Location() != time.UTC {
		return nil, ErrSolicitudBaremacionInvalida
	}
	var destino bytes.Buffer
	escribir := func(valor string) {
		var longitud [8]byte
		binary.BigEndian.PutUint64(longitud[:], uint64(len(valor)))
		_, _ = destino.Write(longitud[:])
		_, _ = destino.WriteString(valor)
	}
	for _, valor := range []string{
		m.Esquema, m.Finalidad, strconv.Itoa(m.VersionEsquema),
		m.Referencia, m.ProcesoRef, m.SolicitudRef, m.SujetoRef, m.BaremacionMeritoRef,
		m.DecisionRef, strconv.FormatUint(m.VersionBase, 10), m.HuellaVersionBaseSHA256,
		m.CreadoEn.Format(time.RFC3339Nano),
	} {
		escribir(valor)
	}
	escribir(strconv.Itoa(len(m.Autorizaciones)))
	for indice, autorizacion := range m.Autorizaciones {
		if autorizacion.Validar() != nil || autorizacion.Secuencia != uint32(indice+1) {
			return nil, ErrSolicitudBaremacionInvalida
		}
		escribir(strconv.FormatUint(uint64(autorizacion.Secuencia), 10))
		escribir(string(autorizacion.Accion))
		escribir(string(autorizacion.ClaseRecurso))
		escribir(autorizacion.RecursoRef)
		escribir(autorizacion.AutorizacionRef)
	}
	escribir(strconv.Itoa(len(m.Evidencias)))
	for indice, evidencia := range m.Evidencias {
		if evidencia.Validar() != nil || evidencia.Secuencia != uint32(indice+1) {
			return nil, ErrSolicitudBaremacionInvalida
		}
		escribir(strconv.FormatUint(uint64(evidencia.Secuencia), 10))
		escribir(string(evidencia.Tipo))
		escribir(evidencia.Referencia)
		escribir(evidencia.HuellaEvidenciaSHA256)
	}
	if incluirHuella {
		if !huellaSHA256Valida(m.HuellaManifiestoSHA256) {
			return nil, ErrSolicitudBaremacionInvalida
		}
		escribir(m.HuellaManifiestoSHA256)
	}
	return destino.Bytes(), nil
}
