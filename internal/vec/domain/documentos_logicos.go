package domain

import (
	"errors"
	"sort"
	"strings"
	"time"
)

var (
	ErrReferenciaDocumentoInvalida          = errors.New("vec: referencia documental invalida")
	ErrRelacionDocumentoInvalida            = errors.New("vec: relacion documental invalida")
	ErrRelacionDocumentoDuplicada           = errors.New("vec: relacion documental duplicada")
	ErrRequisitoRelacionDocumentoInvalido   = errors.New("vec: requisito de relacion documental invalido")
	ErrRequisitoRelacionDocumentoIncumplido = errors.New("vec: requisito de relacion documental incumplido")
	ErrDocumentoLogicoInvalido              = errors.New("vec: documento logico invalido")
	ErrRepresentacionDocumentoInvalida      = errors.New("vec: representacion documental invalida")
	ErrSolicitudRepresentacionDuplicada     = errors.New("vec: solicitud de representacion documental duplicada")
)

const maximoCaracteresReferenciaDocumental = 512

// ReferenciaDocumento identifica una version concreta del contenido de un
// documento logico. Los cambios de estado administrativo no incrementan esta
// version; una modificacion del contenido si debe crear una nueva.
type ReferenciaDocumento struct {
	ID      string `json:"id"`
	Version int    `json:"version"`
}

func (r ReferenciaDocumento) Validar() error {
	if !referenciaDocumentalValida(r.ID) || r.Version < 1 {
		return ErrReferenciaDocumentoInvalida
	}
	return nil
}

// ReferenciaPlantillaDocumento fija la version publicada exacta y su huella.
// De este modo no existe una dependencia implicita de «la ultima plantilla».
type ReferenciaPlantillaDocumento struct {
	ID           string `json:"id"`
	Version      int    `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
}

func (r ReferenciaPlantillaDocumento) Validar() error {
	if !esClaveDocumentalCanonica(r.ID) || r.Version < 1 || !esSHA256(r.HuellaSHA256) {
		return ErrReferenciaDocumentoInvalida
	}
	return nil
}

// TipoRelacionDocumento es extensible deliberadamente. Las constantes
// proporcionan un vocabulario comun, pero un modulo puede registrar nuevas
// claves canonicas sin obligar a modificar el nucleo documental.
type TipoRelacionDocumento string

const (
	TipoRelacionPersona     TipoRelacionDocumento = "persona"
	TipoRelacionProceso     TipoRelacionDocumento = "proceso"
	TipoRelacionLlamamiento TipoRelacionDocumento = "llamamiento"
	TipoRelacionContrato    TipoRelacionDocumento = "contrato"
	TipoRelacionExpediente  TipoRelacionDocumento = "expediente"
)

func (t TipoRelacionDocumento) Valido() bool {
	return esClaveDocumentalCanonica(string(t))
}

// RelacionDocumento vincula el documento con entidades mediante referencias
// internas opacas. Referencia no debe contener nombres, DNI, correo ni ningun
// otro dato descriptivo de la persona o del expediente.
type RelacionDocumento struct {
	Tipo       TipoRelacionDocumento `json:"tipo"`
	Referencia string                `json:"referencia"`
	Rol        string                `json:"rol"`
}

func (r RelacionDocumento) Validar() error {
	if !r.Tipo.Valido() || !referenciaDocumentalValida(r.Referencia) || !esClaveDocumentalCanonica(r.Rol) {
		return ErrRelacionDocumentoInvalida
	}
	return nil
}

// RequisitoRelacionDocumento permite que cada plantilla o flujo declare sus
// cardinalidades sin fijarlas en codigo. Rol vacio significa cualquier rol.
// Maximo cero significa que no existe limite superior.
type RequisitoRelacionDocumento struct {
	Tipo   TipoRelacionDocumento `json:"tipo"`
	Rol    string                `json:"rol,omitempty"`
	Minimo int                   `json:"minimo"`
	Maximo int                   `json:"maximo,omitempty"`
}

func (r RequisitoRelacionDocumento) Validar() error {
	if !r.Tipo.Valido() || (r.Rol != "" && !esClaveDocumentalCanonica(r.Rol)) ||
		r.Minimo < 1 || r.Maximo < 0 || (r.Maximo > 0 && r.Maximo < r.Minimo) {
		return ErrRequisitoRelacionDocumentoInvalido
	}
	return nil
}

// CanonizarRelacionesDocumento valida, clona y ordena las relaciones. Dos
// relaciones con el mismo tipo, rol y referencia son un error, no dos pruebas
// distintas de la misma vinculacion.
func CanonizarRelacionesDocumento(relaciones []RelacionDocumento) ([]RelacionDocumento, error) {
	canonicas := append([]RelacionDocumento(nil), relaciones...)
	for indice := range canonicas {
		canonicas[indice].Referencia = strings.TrimSpace(canonicas[indice].Referencia)
		if err := canonicas[indice].Validar(); err != nil {
			return nil, err
		}
	}
	sort.Slice(canonicas, func(i, j int) bool {
		if canonicas[i].Tipo != canonicas[j].Tipo {
			return canonicas[i].Tipo < canonicas[j].Tipo
		}
		if canonicas[i].Rol != canonicas[j].Rol {
			return canonicas[i].Rol < canonicas[j].Rol
		}
		return canonicas[i].Referencia < canonicas[j].Referencia
	})
	for indice := 1; indice < len(canonicas); indice++ {
		if canonicas[indice] == canonicas[indice-1] {
			return nil, ErrRelacionDocumentoDuplicada
		}
	}
	return canonicas, nil
}

// ValidarRequisitosRelacionesDocumento comprueba las cardinalidades declaradas
// sin revelar referencias concretas en los errores.
func ValidarRequisitosRelacionesDocumento(relaciones []RelacionDocumento, requisitos []RequisitoRelacionDocumento) error {
	canonicas, err := CanonizarRelacionesDocumento(relaciones)
	if err != nil {
		return err
	}
	vistos := make(map[string]struct{}, len(requisitos))
	for _, requisito := range requisitos {
		if err := requisito.Validar(); err != nil {
			return err
		}
		clave := string(requisito.Tipo) + "\x00" + requisito.Rol
		if _, existe := vistos[clave]; existe {
			return ErrRequisitoRelacionDocumentoInvalido
		}
		vistos[clave] = struct{}{}

		coincidencias := 0
		for _, relacion := range canonicas {
			if relacion.Tipo == requisito.Tipo && (requisito.Rol == "" || relacion.Rol == requisito.Rol) {
				coincidencias++
			}
		}
		if coincidencias < requisito.Minimo || (requisito.Maximo > 0 && coincidencias > requisito.Maximo) {
			return ErrRequisitoRelacionDocumentoIncumplido
		}
	}
	return nil
}

// EstadoDocumentoLogico expresa el avance administrativo. Nunca se deduce del
// formato: DOCX y PDF pueden representar el mismo documento en el mismo estado.
type EstadoDocumentoLogico string

const (
	EstadoDocumentoLogicoBorrador       EstadoDocumentoLogico = "borrador"
	EstadoDocumentoLogicoEnRevision     EstadoDocumentoLogico = "en_revision"
	EstadoDocumentoLogicoCerrado        EstadoDocumentoLogico = "cerrado"
	EstadoDocumentoLogicoPendienteFirma EstadoDocumentoLogico = "pendiente_firma"
	EstadoDocumentoLogicoFirmado        EstadoDocumentoLogico = "firmado"
	EstadoDocumentoLogicoRegistrado     EstadoDocumentoLogico = "registrado"
	EstadoDocumentoLogicoAnulado        EstadoDocumentoLogico = "anulado"
)

func (e EstadoDocumentoLogico) Valido() bool {
	switch e {
	case EstadoDocumentoLogicoBorrador, EstadoDocumentoLogicoEnRevision,
		EstadoDocumentoLogicoCerrado, EstadoDocumentoLogicoPendienteFirma,
		EstadoDocumentoLogicoFirmado, EstadoDocumentoLogicoRegistrado,
		EstadoDocumentoLogicoAnulado:
		return true
	default:
		return false
	}
}

// DocumentoLogico agrupa todas las representaciones de una misma version de
// contenido. HuellaDatosHMAC protege los valores fusionados y
// HuellaFuenteHMAC identifica la fuente semantica comun a sus representaciones.
type DocumentoLogico struct {
	ID               string                       `json:"id"`
	Version          int                          `json:"version"`
	Revision         int                          `json:"revision"`
	VersionAnterior  *ReferenciaDocumento         `json:"version_anterior,omitempty"`
	Plantilla        ReferenciaPlantillaDocumento `json:"plantilla"`
	ModuloID         string                       `json:"modulo_id"`
	TipoDocumental   string                       `json:"tipo_documental"`
	Clasificacion    string                       `json:"clasificacion"`
	Relaciones       []RelacionDocumento          `json:"relaciones"`
	Estado           EstadoDocumentoLogico        `json:"estado"`
	HuellaDatosHMAC  string                       `json:"huella_datos_hmac"`
	HuellaFuenteHMAC string                       `json:"huella_fuente_hmac"`
	CreadoPor        string                       `json:"creado_por"`
	CreadoEn         time.Time                    `json:"creado_en"`
	CorrelacionRef   string                       `json:"correlacion_ref"`
	Motivo           string                       `json:"motivo"`
	ENI              MetadatosENI                 `json:"eni"`
}

func (d DocumentoLogico) Referencia() ReferenciaDocumento {
	return ReferenciaDocumento{ID: d.ID, Version: d.Version}
}

func (d DocumentoLogico) Validar() error {
	if err := d.Referencia().Validar(); err != nil {
		return ErrDocumentoLogicoInvalido
	}
	if d.Revision < 1 || d.Plantilla.Validar() != nil || !esClaveDocumentalCanonica(d.ModuloID) ||
		!esClaveDocumentalCanonica(d.TipoDocumental) || !esClaveDocumentalCanonica(d.Clasificacion) ||
		len(d.Relaciones) == 0 || !d.Estado.Valido() ||
		!esHuellaHMACSHA256(d.HuellaDatosHMAC) || !esHuellaHMACSHA256(d.HuellaFuenteHMAC) ||
		!referenciaDocumentalValida(d.CreadoPor) || d.CreadoEn.IsZero() ||
		!referenciaDocumentalValida(d.CorrelacionRef) || !textoDocumentalNoVacioValido(d.Motivo) ||
		d.ENI.Validar() != nil {
		return ErrDocumentoLogicoInvalido
	}
	if _, err := CanonizarRelacionesDocumento(d.Relaciones); err != nil {
		return err
	}
	if d.Version == 1 {
		if d.VersionAnterior != nil {
			return ErrDocumentoLogicoInvalido
		}
	} else {
		if d.VersionAnterior == nil || d.VersionAnterior.Validar() != nil ||
			d.VersionAnterior.ID != d.ID || d.VersionAnterior.Version != d.Version-1 {
			return ErrDocumentoLogicoInvalido
		}
	}
	return nil
}

// ClonarCanonico devuelve una copia independiente con relaciones ordenadas.
func (d DocumentoLogico) ClonarCanonico() (DocumentoLogico, error) {
	canonico := d
	if d.VersionAnterior != nil {
		anterior := *d.VersionAnterior
		canonico.VersionAnterior = &anterior
	}
	relaciones, err := CanonizarRelacionesDocumento(d.Relaciones)
	if err != nil {
		return DocumentoLogico{}, err
	}
	canonico.Relaciones = relaciones
	if err := canonico.Validar(); err != nil {
		return DocumentoLogico{}, err
	}
	return canonico, nil
}

// TipoRepresentacionDocumento describe el uso del artefacto, no su validez
// administrativa. Una representacion firmada o de preservacion deriva siempre
// de otra representacion inmutable.
type TipoRepresentacionDocumento string

const (
	TipoRepresentacionTrabajo       TipoRepresentacionDocumento = "trabajo"
	TipoRepresentacionVisualizacion TipoRepresentacionDocumento = "visualizacion"
	TipoRepresentacionFirma         TipoRepresentacionDocumento = "firma"
	TipoRepresentacionPreservacion  TipoRepresentacionDocumento = "preservacion"
)

func (t TipoRepresentacionDocumento) Valido() bool {
	switch t {
	case TipoRepresentacionTrabajo, TipoRepresentacionVisualizacion,
		TipoRepresentacionFirma, TipoRepresentacionPreservacion:
		return true
	default:
		return false
	}
}

// EstadoRepresentacionDocumento solo describe disponibilidad tecnica.
type EstadoRepresentacionDocumento string

const (
	EstadoRepresentacionPendiente  EstadoRepresentacionDocumento = "pendiente"
	EstadoRepresentacionDisponible EstadoRepresentacionDocumento = "disponible"
	EstadoRepresentacionCuarentena EstadoRepresentacionDocumento = "cuarentena"
	EstadoRepresentacionRechazada  EstadoRepresentacionDocumento = "rechazada"
	EstadoRepresentacionRetirada   EstadoRepresentacionDocumento = "retirada"
)

func (e EstadoRepresentacionDocumento) Valido() bool {
	switch e {
	case EstadoRepresentacionPendiente, EstadoRepresentacionDisponible,
		EstadoRepresentacionCuarentena, EstadoRepresentacionRechazada,
		EstadoRepresentacionRetirada:
		return true
	default:
		return false
	}
}

// RepresentacionDocumento identifica los bytes exactos de un DOCX, PDF u otro
// adaptador futuro. Su SHA-256 es propia; HuellaFuenteHMAC debe coincidir con la
// del documento logico al que pertenece.
type RepresentacionDocumento struct {
	ID                    string                        `json:"id"`
	Documento             ReferenciaDocumento           `json:"documento"`
	Tipo                  TipoRepresentacionDocumento   `json:"tipo"`
	Formato               FormatoDocumento              `json:"formato"`
	MIME                  string                        `json:"mime"`
	NombreFichero         string                        `json:"nombre_fichero"`
	Tamano                int64                         `json:"tamano"`
	HuellaContenidoSHA256 string                        `json:"huella_contenido_sha256"`
	HuellaFuenteHMAC      string                        `json:"huella_fuente_hmac"`
	ReferenciaContenido   string                        `json:"referencia_contenido"`
	EstadoTecnico         EstadoRepresentacionDocumento `json:"estado_tecnico"`
	EstadoAntivirus       EstadoAntivirusDocumento      `json:"estado_antivirus"`
	GeneradaPor           string                        `json:"generada_por"`
	GeneradaEn            time.Time                     `json:"generada_en"`
	DerivadaDeRef         string                        `json:"derivada_de_ref,omitempty"`
}

func (r RepresentacionDocumento) Validar() error {
	if !referenciaDocumentalValida(r.ID) || r.Documento.Validar() != nil || !r.Tipo.Valido() ||
		!r.Formato.Valido() || r.MIME != r.Formato.MIME() || !textoDocumentalNoVacioValido(r.NombreFichero) ||
		r.Tamano < 1 || !esSHA256(r.HuellaContenidoSHA256) || !esHuellaHMACSHA256(r.HuellaFuenteHMAC) ||
		!referenciaDocumentalValida(r.ReferenciaContenido) || !r.EstadoTecnico.Valido() ||
		!r.EstadoAntivirus.Valido() || !referenciaDocumentalValida(r.GeneradaPor) || r.GeneradaEn.IsZero() {
		return ErrRepresentacionDocumentoInvalida
	}
	if r.DerivadaDeRef != "" && (!referenciaDocumentalValida(r.DerivadaDeRef) || r.DerivadaDeRef == r.ID) {
		return ErrRepresentacionDocumentoInvalida
	}
	if (r.Tipo == TipoRepresentacionFirma || r.Tipo == TipoRepresentacionPreservacion) && r.DerivadaDeRef == "" {
		return ErrRepresentacionDocumentoInvalida
	}
	return nil
}

func (r RepresentacionDocumento) ValidarPertenencia(documento DocumentoLogico) error {
	if err := r.Validar(); err != nil {
		return err
	}
	if err := documento.Validar(); err != nil {
		return err
	}
	if r.Documento != documento.Referencia() || r.HuellaFuenteHMAC != documento.HuellaFuenteHMAC {
		return ErrRepresentacionDocumentoInvalida
	}
	return nil
}

// SolicitudRepresentacionDocumento permite pedir varias salidas de una unica
// fuente, por ejemplo DOCX de trabajo y PDF de visualizacion.
type SolicitudRepresentacionDocumento struct {
	Tipo    TipoRepresentacionDocumento `json:"tipo"`
	Formato FormatoDocumento            `json:"formato"`
}

func (s SolicitudRepresentacionDocumento) Validar() error {
	if !s.Tipo.Valido() || !s.Formato.Valido() ||
		s.Tipo == TipoRepresentacionFirma || s.Tipo == TipoRepresentacionPreservacion {
		return ErrRepresentacionDocumentoInvalida
	}
	return nil
}

func CanonizarSolicitudesRepresentacionDocumento(solicitudes []SolicitudRepresentacionDocumento) ([]SolicitudRepresentacionDocumento, error) {
	canonicas := append([]SolicitudRepresentacionDocumento(nil), solicitudes...)
	for _, solicitud := range canonicas {
		if err := solicitud.Validar(); err != nil {
			return nil, err
		}
	}
	sort.Slice(canonicas, func(i, j int) bool {
		if canonicas[i].Tipo != canonicas[j].Tipo {
			return canonicas[i].Tipo < canonicas[j].Tipo
		}
		return canonicas[i].Formato < canonicas[j].Formato
	})
	for indice := 1; indice < len(canonicas); indice++ {
		if canonicas[indice] == canonicas[indice-1] {
			return nil, ErrSolicitudRepresentacionDuplicada
		}
	}
	return canonicas, nil
}

// ResultadoGeneracionDocumento devuelve un unico documento logico y todas sus
// representaciones. Repetida indica una respuesta idempotente ya confirmada.
type ResultadoGeneracionDocumento struct {
	Documento        DocumentoLogico           `json:"documento"`
	Representaciones []RepresentacionDocumento `json:"representaciones"`
	Repetida         bool                      `json:"repetida"`
}

func (r ResultadoGeneracionDocumento) Validar() error {
	if err := r.Documento.Validar(); err != nil {
		return err
	}
	if len(r.Representaciones) == 0 {
		return ErrRepresentacionDocumentoInvalida
	}
	vistos := make(map[string]struct{}, len(r.Representaciones))
	tiposFormatos := make(map[string]struct{}, len(r.Representaciones))
	for _, representacion := range r.Representaciones {
		if err := representacion.ValidarPertenencia(r.Documento); err != nil {
			return err
		}
		if _, existe := vistos[representacion.ID]; existe {
			return ErrRepresentacionDocumentoInvalida
		}
		vistos[representacion.ID] = struct{}{}
		claveTipoFormato := string(representacion.Tipo) + "\x00" + string(representacion.Formato)
		if _, existe := tiposFormatos[claveTipoFormato]; existe {
			return ErrRepresentacionDocumentoInvalida
		}
		tiposFormatos[claveTipoFormato] = struct{}{}
	}
	return nil
}

func (r ResultadoGeneracionDocumento) ClonarCanonico() (ResultadoGeneracionDocumento, error) {
	canonico := r
	documento, err := r.Documento.ClonarCanonico()
	if err != nil {
		return ResultadoGeneracionDocumento{}, err
	}
	canonico.Documento = documento
	canonico.Representaciones = append([]RepresentacionDocumento(nil), r.Representaciones...)
	sort.Slice(canonico.Representaciones, func(i, j int) bool {
		return canonico.Representaciones[i].ID < canonico.Representaciones[j].ID
	})
	if err := canonico.Validar(); err != nil {
		return ResultadoGeneracionDocumento{}, err
	}
	return canonico, nil
}

func referenciaDocumentalValida(valor string) bool {
	return valor == strings.TrimSpace(valor) && len(valor) > 0 &&
		len(valor) <= maximoCaracteresReferenciaDocumental && textoDocumentalValido(valor)
}

func textoDocumentalNoVacioValido(valor string) bool {
	return valor == strings.TrimSpace(valor) && valor != "" &&
		len(valor) <= maximoCaracteresReferenciaDocumental && textoDocumentalValido(valor)
}

func esHuellaHMACSHA256(valor string) bool {
	partes := strings.Split(valor, ":")
	return len(partes) == 3 && partes[0] == "hmac-sha256" &&
		esClaveDocumentalCanonica(partes[1]) && esSHA256(partes[2])
}
