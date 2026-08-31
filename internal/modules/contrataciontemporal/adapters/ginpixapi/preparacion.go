// Package ginpixapi adapta cargas GINPIX ya preparadas a un transporte HTTP
// inyectado. No compone dependencias ni abre conexiones por si mismo.
package ginpixapi

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"strings"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/ginpixfichero"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	EsquemaReciboAPIGINPIXV1 = "vec.dipgra.contratacion-temporal.ginpix.api.recibo.v1"
	VersionReciboAPIGINPIXV1 = uint64(1)
)

var (
	ErrPreparacionAPIGINPIXInvalida = errors.New(
		"contratacion temporal: preparacion api ginpix invalida",
	)
	ErrReciboExternoGINPIXInvalido = errors.New(
		"contratacion temporal: recibo externo ginpix invalido",
	)
)

// MetadatosOperacion conserva solamente ligaduras y evidencias opacas. Los
// campos funcionales permanecen dentro del cuerpo canónico O7-05.
type MetadatosOperacion struct {
	VersionExpediente      uint64
	ExpedienteRef          string
	IncorporacionRef       string
	ProcedenciaModeloRef   string
	CorrelacionRef         string
	IdempotenciaRef        string
	ModeloHuellaSHA256     string
	MapeoRef               string
	MapeoVersion           uint64
	ProcedenciaMapeoRef    string
	MapeoHuellaSHA256      string
	CargaHuellaSHA256      string
	CuerpoHuellaSHA256     string
	ReciboIncorporacionRef string
	ResultadoPersonalRef   string
	ReciboPersonalRef      string
}

type datosPreparacion struct {
	cuerpo        []byte
	metadatos     MetadatosOperacion
	orden         ports.OrdenConfirmarIncorporacion
	incorporacion ports.ReciboConfirmacionIncorporacion
}

// Preparacion es un sobre inmutable. Preparar no autentica, no envia y no
// acredita ningun efecto fuera del proceso.
type Preparacion struct{ datos *datosPreparacion }

// Preparar coteja la orden autentica y el recibo O7-02, la transformación
// O7-03 y los bytes exactos O7-05. La orden conserva el contexto y la decisión
// V3 completos; este adaptador no los reconstruye desde campos copiables.
func Preparar(
	solicitud ports.SolicitudMapeoGINPIX,
	orden ports.OrdenConfirmarIncorporacion,
	incorporacion ports.ReciboConfirmacionIncorporacion,
) (Preparacion, error) {
	if _, err := orden.Datos(); err != nil || incorporacion.ValidarPara(orden) != nil {
		return Preparacion{}, ErrPreparacionAPIGINPIXInvalida
	}
	modelo, errModelo := solicitud.Modelo()
	mapeo, errMapeo := solicitud.Mapeo()
	if errModelo != nil || errMapeo != nil {
		return Preparacion{}, ErrPreparacionAPIGINPIXInvalida
	}
	carga, err := domain.AplicarMapeoGINPIX(modelo, mapeo)
	if err != nil || carga.Validar() != nil {
		return Preparacion{}, ErrPreparacionAPIGINPIXInvalida
	}
	datosCarga := carga.Datos()
	if !evidenciaIncorporacionLigada(
		orden,
		incorporacion,
		datosCarga,
	) {
		return Preparacion{}, ErrPreparacionAPIGINPIXInvalida
	}

	exportacion, err := ginpixfichero.PrepararExportacion(carga)
	if err != nil {
		return Preparacion{}, ErrPreparacionAPIGINPIXInvalida
	}
	cuerpo, errCuerpo := exportacion.Contenido()
	huellaCuerpo, errHuella := exportacion.HuellaSHA256()
	metadatosFichero, errMetadatos := exportacion.Metadatos()
	if errCuerpo != nil || errHuella != nil || errMetadatos != nil ||
		len(cuerpo) == 0 || len(cuerpo) > ginpixfichero.MaximoBytesFicheroGINPIX {
		return Preparacion{}, ErrPreparacionAPIGINPIXInvalida
	}
	metadatos := MetadatosOperacion{
		VersionExpediente:      metadatosFichero.VersionExpediente,
		ExpedienteRef:          metadatosFichero.ExpedienteRef,
		IncorporacionRef:       metadatosFichero.IncorporacionRef,
		ProcedenciaModeloRef:   metadatosFichero.ProcedenciaModeloRef,
		CorrelacionRef:         metadatosFichero.CorrelacionRef,
		IdempotenciaRef:        metadatosFichero.IdempotenciaRef,
		ModeloHuellaSHA256:     datosCarga.ModeloHuellaSHA256,
		MapeoRef:               metadatosFichero.MapeoRef,
		MapeoVersion:           metadatosFichero.MapeoVersion,
		ProcedenciaMapeoRef:    metadatosFichero.ProcedenciaMapeoRef,
		MapeoHuellaSHA256:      datosCarga.MapeoHuellaSHA256,
		CargaHuellaSHA256:      datosCarga.HuellaSHA256,
		CuerpoHuellaSHA256:     huellaCuerpo,
		ReciboIncorporacionRef: incorporacion.ReciboRef,
		ResultadoPersonalRef:   incorporacion.ResultadoPersonalRef,
		ReciboPersonalRef:      incorporacion.ReciboPersonalRef,
	}
	incorporacion = clonarReciboConfirmacionIncorporacion(incorporacion)
	preparacion := Preparacion{datos: &datosPreparacion{
		cuerpo: append([]byte(nil), cuerpo...), metadatos: metadatos,
		orden: orden, incorporacion: incorporacion,
	}}
	if preparacion.Validar() != nil {
		return Preparacion{}, ErrPreparacionAPIGINPIXInvalida
	}
	return preparacion, nil
}

func (p Preparacion) Validar() error {
	if p.datos == nil || len(p.datos.cuerpo) == 0 ||
		len(p.datos.cuerpo) > ginpixfichero.MaximoBytesFicheroGINPIX ||
		!p.datos.metadatos.validar() ||
		p.datos.incorporacion.ValidarPara(p.datos.orden) != nil ||
		!p.datos.metadatos.ligadosA(p.datos.incorporacion) {
		return ErrPreparacionAPIGINPIXInvalida
	}
	suma := sha256.Sum256(p.datos.cuerpo)
	if !huellasIguales(p.datos.metadatos.CuerpoHuellaSHA256, hex.EncodeToString(suma[:])) {
		return ErrPreparacionAPIGINPIXInvalida
	}
	return nil
}

func (p Preparacion) Cuerpo() ([]byte, error) {
	if p.Validar() != nil {
		return nil, ErrPreparacionAPIGINPIXInvalida
	}
	return append([]byte(nil), p.datos.cuerpo...), nil
}

func (p Preparacion) Metadatos() (MetadatosOperacion, error) {
	if p.Validar() != nil {
		return MetadatosOperacion{}, ErrPreparacionAPIGINPIXInvalida
	}
	return p.datos.metadatos, nil
}

func (m MetadatosOperacion) validar() bool {
	return m.VersionExpediente > 0 && m.MapeoVersion > 0 &&
		domain.ReferenciaOpacaValida(m.ExpedienteRef) &&
		domain.ReferenciaOpacaValida(m.IncorporacionRef) &&
		domain.ReferenciaOpacaValida(m.ProcedenciaModeloRef) &&
		domain.ReferenciaOpacaValida(m.CorrelacionRef) &&
		domain.ReferenciaOpacaValida(m.IdempotenciaRef) &&
		domain.ReferenciaOpacaValida(m.MapeoRef) &&
		domain.ReferenciaOpacaValida(m.ProcedenciaMapeoRef) &&
		domain.ReferenciaOpacaValida(m.ReciboIncorporacionRef) &&
		domain.ReferenciaOpacaValida(m.ResultadoPersonalRef) &&
		domain.ReferenciaOpacaValida(m.ReciboPersonalRef) &&
		huellaValida(m.ModeloHuellaSHA256) && huellaValida(m.MapeoHuellaSHA256) &&
		huellaValida(m.CargaHuellaSHA256) && huellaValida(m.CuerpoHuellaSHA256)
}

func (m MetadatosOperacion) ligadosA(r ports.ReciboConfirmacionIncorporacion) bool {
	return m.VersionExpediente == r.VersionExpediente &&
		m.ExpedienteRef == r.ExpedienteRef && m.IncorporacionRef == r.ActuacionRef &&
		m.ReciboIncorporacionRef == r.ReciboRef &&
		m.ResultadoPersonalRef == r.ResultadoPersonalRef &&
		m.ReciboPersonalRef == r.ReciboPersonalRef
}

func evidenciaIncorporacionLigada(
	orden ports.OrdenConfirmarIncorporacion,
	r ports.ReciboConfirmacionIncorporacion,
	carga domain.DatosCargaMapeadaGINPIX,
) bool {
	evidencia, err := orden.Datos()
	if err != nil || r.ValidarPara(orden) != nil {
		return false
	}
	datos := evidencia.Confirmacion
	return datos.SolicitudPersonal.ExpedienteRef == carga.ExpedienteRef &&
		datos.SolicitudPersonal.VersionExpediente == carga.VersionExpediente &&
		r.ExpedienteRef == carga.ExpedienteRef && r.ActuacionRef == carga.IncorporacionRef &&
		r.VersionExpediente == carga.VersionExpediente
}

func clonarReciboConfirmacionIncorporacion(
	r ports.ReciboConfirmacionIncorporacion,
) ports.ReciboConfirmacionIncorporacion {
	r.Documentos = append([]domain.DocumentoSeguimiento(nil), r.Documentos...)
	return r
}

// DatosReciboExterno contiene solo referencias opacas, versiones y huellas.
type DatosReciboExterno struct {
	Esquema                      string `json:"esquema"`
	Version                      uint64 `json:"version"`
	ReciboExternoRef             string `json:"recibo_externo_ref"`
	EvidenciaExternaRef          string `json:"evidencia_externa_ref"`
	EvidenciaExternaHuellaSHA256 string `json:"evidencia_externa_huella_sha256"`
	VersionExpediente            uint64 `json:"version_expediente"`
	ExpedienteRef                string `json:"expediente_ref"`
	IncorporacionRef             string `json:"incorporacion_ref"`
	CorrelacionRef               string `json:"correlacion_ref"`
	IdempotenciaRef              string `json:"idempotencia_ref"`
	ModeloHuellaSHA256           string `json:"modelo_huella_sha256"`
	MapeoRef                     string `json:"mapeo_ref"`
	MapeoVersion                 uint64 `json:"mapeo_version"`
	MapeoHuellaSHA256            string `json:"mapeo_huella_sha256"`
	CargaHuellaSHA256            string `json:"carga_huella_sha256"`
	CuerpoHuellaSHA256           string `json:"cuerpo_huella_sha256"`
	ReciboIncorporacionRef       string `json:"recibo_incorporacion_ref"`
	ResultadoPersonalRef         string `json:"resultado_personal_ref"`
	ReciboPersonalRef            string `json:"recibo_personal_ref"`
}

// ReciboExterno solo existe tras validar el contrato completo de respuesta.
type ReciboExterno struct{ datos *DatosReciboExterno }

func (r ReciboExterno) Datos() (DatosReciboExterno, error) {
	if r.datos == nil || !r.datos.validar() {
		return DatosReciboExterno{}, ErrReciboExternoGINPIXInvalido
	}
	return *r.datos, nil
}

func nuevoReciboExterno(datos DatosReciboExterno, preparacion Preparacion) (ReciboExterno, error) {
	metadatos, err := preparacion.Metadatos()
	if err != nil || !datos.validarPara(metadatos) {
		return ReciboExterno{}, ErrReciboExternoGINPIXInvalido
	}
	copia := datos
	return ReciboExterno{datos: &copia}, nil
}

func (d DatosReciboExterno) validar() bool {
	return d.Esquema == EsquemaReciboAPIGINPIXV1 && d.Version == VersionReciboAPIGINPIXV1 &&
		d.VersionExpediente > 0 && d.MapeoVersion > 0 &&
		domain.ReferenciaOpacaValida(d.ReciboExternoRef) &&
		domain.ReferenciaOpacaValida(d.EvidenciaExternaRef) &&
		domain.ReferenciaOpacaValida(d.ExpedienteRef) &&
		domain.ReferenciaOpacaValida(d.IncorporacionRef) &&
		domain.ReferenciaOpacaValida(d.CorrelacionRef) &&
		domain.ReferenciaOpacaValida(d.IdempotenciaRef) &&
		domain.ReferenciaOpacaValida(d.MapeoRef) &&
		domain.ReferenciaOpacaValida(d.ReciboIncorporacionRef) &&
		domain.ReferenciaOpacaValida(d.ResultadoPersonalRef) &&
		domain.ReferenciaOpacaValida(d.ReciboPersonalRef) &&
		huellaValida(d.EvidenciaExternaHuellaSHA256) &&
		huellaValida(d.ModeloHuellaSHA256) && huellaValida(d.MapeoHuellaSHA256) &&
		huellaValida(d.CargaHuellaSHA256) && huellaValida(d.CuerpoHuellaSHA256)
}

func (d DatosReciboExterno) validarPara(m MetadatosOperacion) bool {
	return d.validar() && d.VersionExpediente == m.VersionExpediente &&
		d.ExpedienteRef == m.ExpedienteRef && d.IncorporacionRef == m.IncorporacionRef &&
		d.CorrelacionRef == m.CorrelacionRef && d.IdempotenciaRef == m.IdempotenciaRef &&
		d.ModeloHuellaSHA256 == m.ModeloHuellaSHA256 && d.MapeoRef == m.MapeoRef &&
		d.MapeoVersion == m.MapeoVersion && d.MapeoHuellaSHA256 == m.MapeoHuellaSHA256 &&
		d.CargaHuellaSHA256 == m.CargaHuellaSHA256 &&
		d.CuerpoHuellaSHA256 == m.CuerpoHuellaSHA256 &&
		d.ReciboIncorporacionRef == m.ReciboIncorporacionRef &&
		d.ResultadoPersonalRef == m.ResultadoPersonalRef &&
		d.ReciboPersonalRef == m.ReciboPersonalRef
}

func huellaValida(valor string) bool {
	if len(valor) != sha256.Size*2 || valor == strings.Repeat("0", sha256.Size*2) {
		return false
	}
	_, err := hex.DecodeString(valor)
	return err == nil && valor == strings.ToLower(valor)
}

func huellasIguales(primera, segunda string) bool {
	primeraBytes, errPrimera := hex.DecodeString(primera)
	segundaBytes, errSegunda := hex.DecodeString(segunda)
	return errPrimera == nil && errSegunda == nil && len(primeraBytes) == sha256.Size &&
		len(segundaBytes) == sha256.Size && subtle.ConstantTimeCompare(primeraBytes, segundaBytes) == 1
}
