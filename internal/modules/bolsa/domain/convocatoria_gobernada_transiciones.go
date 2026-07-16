package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"
)

const vigenciaMaximaComprobacionDependencias = 15 * time.Minute

type EstadoGobiernoConvocatoria string

const (
	EstadoGobiernoConvocatoriaBorrador   EstadoGobiernoConvocatoria = "borrador"
	EstadoGobiernoConvocatoriaPublicada  EstadoGobiernoConvocatoria = "publicada"
	EstadoGobiernoConvocatoriaSustituida EstadoGobiernoConvocatoria = "sustituida"
	EstadoGobiernoConvocatoriaRetirada   EstadoGobiernoConvocatoria = "retirada"
)

func (e EstadoGobiernoConvocatoria) Valido() bool {
	return e == EstadoGobiernoConvocatoriaBorrador || e == EstadoGobiernoConvocatoriaPublicada ||
		e == EstadoGobiernoConvocatoriaSustituida || e == EstadoGobiernoConvocatoriaRetirada
}

type EvidenciaAprobacionConvocatoria struct {
	Accion                string    `json:"accion"`
	Referencia            string    `json:"referencia"`
	HuellaEvidenciaSHA256 string    `json:"huella_evidencia_sha256"`
	ConvocatoriaRef       string    `json:"convocatoria_ref"`
	HuellaContenidoSHA256 string    `json:"huella_contenido_sha256"`
	AprobadaPor           string    `json:"aprobada_por"`
	AprobadaEn            time.Time `json:"aprobada_en"`
}

type EvidenciaDependenciasConvocatoria struct {
	Referencia            string    `json:"referencia"`
	HuellaEvidenciaSHA256 string    `json:"huella_evidencia_sha256"`
	ConvocatoriaRef       string    `json:"convocatoria_ref"`
	HuellaContenidoSHA256 string    `json:"huella_contenido_sha256"`
	VerificadaEn          time.Time `json:"verificada_en"`
}

type VersionConvocatoriaGobernada struct {
	ID                       string                             `json:"id"`
	Secuencia                int                                `json:"secuencia"`
	CodigoVersionPublica     string                             `json:"codigo_version_publica"`
	Revision                 int                                `json:"revision"`
	VersionAnteriorRef       string                             `json:"version_anterior_ref,omitempty"`
	Contenido                ContenidoPublicableConvocatoria    `json:"contenido"`
	Configuracion            ConfiguracionFijadaConvocatoria    `json:"configuracion"`
	ExpedienteRef            string                             `json:"expediente_ref"`
	MotivoCreacion           string                             `json:"motivo_creacion"`
	EstadoGobierno           EstadoGobiernoConvocatoria         `json:"estado_gobierno"`
	CreadaPor                string                             `json:"creada_por"`
	CreadaEn                 time.Time                          `json:"creada_en"`
	UltimaModificacionPor    string                             `json:"ultima_modificacion_por,omitempty"`
	UltimaModificacionEn     time.Time                          `json:"ultima_modificacion_en,omitempty"`
	MotivoModificacion       string                             `json:"motivo_modificacion,omitempty"`
	PublicadaPor             string                             `json:"publicada_por,omitempty"`
	PublicadaEn              time.Time                          `json:"publicada_en,omitempty"`
	MotivoPublicacion        string                             `json:"motivo_publicacion,omitempty"`
	AprobacionPublicacion    *EvidenciaAprobacionConvocatoria   `json:"aprobacion_publicacion,omitempty"`
	ComprobacionDependencias *EvidenciaDependenciasConvocatoria `json:"comprobacion_dependencias,omitempty"`
	SustituidaPorRef         string                             `json:"sustituida_por_ref,omitempty"`
	SustituidaPor            string                             `json:"sustituida_por,omitempty"`
	SustituidaEn             time.Time                          `json:"sustituida_en,omitempty"`
	RetiradaPor              string                             `json:"retirada_por,omitempty"`
	RetiradaEn               time.Time                          `json:"retirada_en,omitempty"`
	MotivoRetirada           string                             `json:"motivo_retirada,omitempty"`
	AprobacionRetirada       *EvidenciaAprobacionConvocatoria   `json:"aprobacion_retirada,omitempty"`
}

type DatosNuevaVersionConvocatoriaGobernada struct {
	ID                   string
	CodigoVersionPublica string
	Contenido            ContenidoPublicableConvocatoria
	Configuracion        ConfiguracionFijadaConvocatoria
	ExpedienteRef        string
	Motivo               string
	ActorID              string
	Instante             time.Time
}

func NuevaVersionConvocatoriaGobernada(datos DatosNuevaVersionConvocatoriaGobernada) (VersionConvocatoriaGobernada, error) {
	version := VersionConvocatoriaGobernada{
		ID: strings.TrimSpace(datos.ID), Secuencia: 1,
		CodigoVersionPublica: strings.TrimSpace(datos.CodigoVersionPublica), Revision: 1,
		Contenido: datos.Contenido, Configuracion: datos.Configuracion,
		ExpedienteRef: strings.TrimSpace(datos.ExpedienteRef), MotivoCreacion: strings.TrimSpace(datos.Motivo),
		EstadoGobierno: EstadoGobiernoConvocatoriaBorrador,
		CreadaPor:      strings.TrimSpace(datos.ActorID), CreadaEn: instanteConvocatoriaCanonico(datos.Instante),
	}
	return version.ClonarCanonico()
}

func (v VersionConvocatoriaGobernada) Referencia() string {
	return referenciaVersionConvocatoria(v.ID, v.Secuencia)
}

func (v VersionConvocatoriaGobernada) Validar() error {
	if !referenciaConvocatoriaValida(v.ID) || v.Secuencia < 1 ||
		!claveCatalogoConvocatoriaValida(v.CodigoVersionPublica) || v.Revision < 1 ||
		!referenciaConvocatoriaValida(v.Referencia()) || v.Contenido.Validar() != nil ||
		v.Configuracion.ValidarPara(v.Contenido) != nil || !referenciaOpacaValida(v.ExpedienteRef) ||
		!textoConvocatoriaValido(v.MotivoCreacion, 8000, true) || !v.EstadoGobierno.Valido() ||
		!referenciaOpacaValida(v.CreadaPor) || !instanteUTCCanonico(v.CreadaEn) {
		return ErrVersionConvocatoriaGobernadaInvalida
	}
	if (v.Secuencia == 1 && v.VersionAnteriorRef != "") ||
		(v.Secuencia > 1 && !referenciaConvocatoriaValida(v.VersionAnteriorRef)) {
		return ErrVersionConvocatoriaGobernadaInvalida
	}
	if v.Revision == 1 {
		if v.UltimaModificacionPor != "" || !v.UltimaModificacionEn.IsZero() || v.MotivoModificacion != "" {
			return ErrVersionConvocatoriaGobernadaInvalida
		}
	} else if !referenciaOpacaValida(v.UltimaModificacionPor) ||
		!instanteUTCCanonico(v.UltimaModificacionEn) || v.UltimaModificacionEn.Before(v.CreadaEn) ||
		!textoConvocatoriaValido(v.MotivoModificacion, 8000, true) {
		return ErrVersionConvocatoriaGobernadaInvalida
	}
	switch v.EstadoGobierno {
	case EstadoGobiernoConvocatoriaBorrador:
		if v.datosPublicacionPresentes() || v.datosSustitucionPresentes() || v.datosRetiradaPresentes() {
			return ErrVersionConvocatoriaGobernadaInvalida
		}
	case EstadoGobiernoConvocatoriaPublicada:
		if !v.datosPublicacionValidos() || v.datosSustitucionPresentes() || v.datosRetiradaPresentes() {
			return ErrVersionConvocatoriaGobernadaInvalida
		}
	case EstadoGobiernoConvocatoriaSustituida:
		if !v.datosPublicacionValidos() || !v.datosSustitucionValidos() || v.datosRetiradaPresentes() {
			return ErrVersionConvocatoriaGobernadaInvalida
		}
	case EstadoGobiernoConvocatoriaRetirada:
		if !v.datosPublicacionValidos() || v.datosSustitucionPresentes() || !v.datosRetiradaValidos() {
			return ErrVersionConvocatoriaGobernadaInvalida
		}
	}
	return nil
}

func (v VersionConvocatoriaGobernada) ClonarCanonico() (VersionConvocatoriaGobernada, error) {
	clon := v
	var err error
	clon.Contenido, err = v.Contenido.ClonarCanonico()
	if err != nil {
		return VersionConvocatoriaGobernada{}, err
	}
	clon.Configuracion, err = v.Configuracion.ClonarCanonicaPara(clon.Contenido)
	if err != nil {
		return VersionConvocatoriaGobernada{}, err
	}
	clon.CreadaEn = instanteConvocatoriaCanonico(v.CreadaEn)
	clon.UltimaModificacionEn = instanteConvocatoriaCanonico(v.UltimaModificacionEn)
	clon.PublicadaEn = instanteConvocatoriaCanonico(v.PublicadaEn)
	clon.SustituidaEn = instanteConvocatoriaCanonico(v.SustituidaEn)
	clon.RetiradaEn = instanteConvocatoriaCanonico(v.RetiradaEn)
	clon.AprobacionPublicacion = clonarAprobacionConvocatoria(v.AprobacionPublicacion)
	clon.ComprobacionDependencias = clonarComprobacionDependencias(v.ComprobacionDependencias)
	clon.AprobacionRetirada = clonarAprobacionConvocatoria(v.AprobacionRetirada)
	if err := clon.Validar(); err != nil {
		return VersionConvocatoriaGobernada{}, err
	}
	return clon, nil
}

func (v VersionConvocatoriaGobernada) HuellaContenidoSHA256() (string, error) {
	if err := v.Validar(); err != nil {
		return "", err
	}
	return v.huellaContenidoSinValidar()
}

func (v VersionConvocatoriaGobernada) huellaContenidoSinValidar() (string, error) {
	contenido, err := v.Contenido.ClonarCanonico()
	if err != nil {
		return "", err
	}
	configuracion, err := v.Configuracion.ClonarCanonicaPara(contenido)
	if err != nil {
		return "", err
	}
	material := struct {
		Esquema              string                          `json:"esquema"`
		ID                   string                          `json:"id"`
		Secuencia            int                             `json:"secuencia"`
		CodigoVersionPublica string                          `json:"codigo_version_publica"`
		VersionAnteriorRef   string                          `json:"version_anterior_ref,omitempty"`
		Contenido            ContenidoPublicableConvocatoria `json:"contenido"`
		Configuracion        ConfiguracionFijadaConvocatoria `json:"configuracion"`
		ExpedienteRef        string                          `json:"expediente_ref"`
	}{"bolsa.version-convocatoria.contenido.v1", v.ID, v.Secuencia, v.CodigoVersionPublica,
		v.VersionAnteriorRef, contenido, configuracion, v.ExpedienteRef}
	bytes, err := json.Marshal(material)
	if err != nil {
		return "", ErrVersionConvocatoriaGobernadaInvalida
	}
	suma := sha256.Sum256(bytes)
	return hex.EncodeToString(suma[:]), nil
}

func (v VersionConvocatoriaGobernada) HuellaSHA256() (string, error) {
	canonica, err := v.ClonarCanonico()
	if err != nil {
		return "", err
	}
	bytes, err := json.Marshal(canonica)
	if err != nil {
		return "", ErrVersionConvocatoriaGobernadaInvalida
	}
	suma := sha256.Sum256(bytes)
	return hex.EncodeToString(suma[:]), nil
}

func clonarAprobacionConvocatoria(origen *EvidenciaAprobacionConvocatoria) *EvidenciaAprobacionConvocatoria {
	if origen == nil {
		return nil
	}
	clon := *origen
	clon.AprobadaEn = instanteConvocatoriaCanonico(origen.AprobadaEn)
	return &clon
}

func clonarComprobacionDependencias(origen *EvidenciaDependenciasConvocatoria) *EvidenciaDependenciasConvocatoria {
	if origen == nil {
		return nil
	}
	clon := *origen
	clon.VerificadaEn = instanteConvocatoriaCanonico(origen.VerificadaEn)
	return &clon
}
