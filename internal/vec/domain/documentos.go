package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	ErrPlantillaDocumentoInvalida = errors.New("vec: plantilla documental invalida")
	ErrPlantillaNoPublicada       = errors.New("vec: plantilla documental no publicada")
	ErrCampoPlantillaInvalido     = errors.New("vec: campo de plantilla invalido")
	ErrCampoPlantillaFaltante     = errors.New("vec: falta un campo obligatorio de la plantilla")
	ErrCampoPlantillaDesconocido  = errors.New("vec: dato no declarado por la plantilla")
	ErrFormatoDocumentoInvalido   = errors.New("vec: formato documental invalido")
	ErrDocumentoInvalido          = errors.New("vec: documento invalido")
	ErrGarantiaInsuficiente       = errors.New("vec: nivel de garantia insuficiente")
	ErrContenidoFusionadoExcesivo = errors.New("vec: el contenido fusionado supera el limite seguro")
)

const (
	maximoCamposPlantilla     = 256
	maximoParrafosPlantilla   = 2_000
	maximoCaracteresPlantilla = 2 * 1024 * 1024
	maximoCaracteresDato      = 256 * 1024
	maximoMarcadoresPlantilla = 10_000
	maximoBytesFusionados     = 16 * 1024 * 1024
)

var (
	claveDocumentalValida = regexp.MustCompile(`^[a-z][a-z0-9._-]{0,127}$`)
	marcadorPlantilla     = regexp.MustCompile(`\{\{[ \t]*([a-z][a-z0-9._-]{0,127})[ \t]*\}\}`)
)

// FormatoDocumento identifica un formato de salida que debe proporcionar un
// adaptador. DOCX es el formato Word editable; el binario historico DOC no se
// genera por seguridad e interoperabilidad.
type FormatoDocumento string

const (
	FormatoDocumentoPDF  FormatoDocumento = "pdf"
	FormatoDocumentoDOCX FormatoDocumento = "docx"
)

func (f FormatoDocumento) Valido() bool {
	return f == FormatoDocumentoPDF || f == FormatoDocumentoDOCX
}

func (f FormatoDocumento) MIME() string {
	switch f {
	case FormatoDocumentoPDF:
		return "application/pdf"
	case FormatoDocumentoDOCX:
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	default:
		return ""
	}
}

func (f FormatoDocumento) Extension() string {
	if !f.Valido() {
		return ""
	}
	return "." + string(f)
}

// EstadoPlantillaDocumento gobierna el ciclo de vida de una version. Una
// version publicada es inmutable; cualquier cambio crea una version nueva.
type EstadoPlantillaDocumento string

const (
	EstadoPlantillaBorrador  EstadoPlantillaDocumento = "borrador"
	EstadoPlantillaPublicada EstadoPlantillaDocumento = "publicada"
	EstadoPlantillaRetirada  EstadoPlantillaDocumento = "retirada"
)

func (e EstadoPlantillaDocumento) Valido() bool {
	return e == EstadoPlantillaBorrador || e == EstadoPlantillaPublicada || e == EstadoPlantillaRetirada
}

// CampoPlantillaDocumento declara de forma cerrada un dato de fusion. El
// servicio rechaza datos adicionales para evitar filtraciones accidentales.
type CampoPlantillaDocumento struct {
	Clave       string `json:"clave"`
	Etiqueta    string `json:"etiqueta"`
	Obligatorio bool   `json:"obligatorio"`
	Sensible    bool   `json:"sensible,omitempty"`
}

func (c CampoPlantillaDocumento) Validar() error {
	if !esClaveDocumentalCanonica(c.Clave) || strings.TrimSpace(c.Etiqueta) == "" {
		return fmt.Errorf("%w: %q", ErrCampoPlantillaInvalido, c.Clave)
	}
	return nil
}

// PlantillaDocumento es una version reproducible y gobernada de una
// plantilla. No existe la operacion "ultima version" para generar un acto: el
// caso de uso exige ID y version explicitos.
type PlantillaDocumento struct {
	ID                string                    `json:"id"`
	Version           int                       `json:"version"`
	ModuloID          string                    `json:"modulo_id"`
	TipoDocumental    string                    `json:"tipo_documental"`
	Nombre            string                    `json:"nombre"`
	Titulo            string                    `json:"titulo"`
	Parrafos          []string                  `json:"parrafos"`
	Campos            []CampoPlantillaDocumento `json:"campos"`
	Formatos          []FormatoDocumento        `json:"formatos"`
	PermisoGenerar    string                    `json:"permiso_generar"`
	GarantiaMinima    AuthAssurance             `json:"garantia_minima"`
	Estado            EstadoPlantillaDocumento  `json:"estado"`
	CreadaPor         string                    `json:"creada_por"`
	CreadaEn          time.Time                 `json:"creada_en"`
	PublicadaPor      string                    `json:"publicada_por,omitempty"`
	PublicadaEn       time.Time                 `json:"publicada_en,omitempty"`
	AprobacionRef     string                    `json:"aprobacion_ref,omitempty"`
	MotivoPublicacion string                    `json:"motivo_publicacion,omitempty"`
}

func (p PlantillaDocumento) Validar() error {
	if !esClaveDocumentalCanonica(p.ID) ||
		p.Version < 1 ||
		!esClaveDocumentalCanonica(p.ModuloID) ||
		!esClaveDocumentalCanonica(p.TipoDocumental) ||
		strings.TrimSpace(p.Nombre) == "" ||
		strings.TrimSpace(p.Titulo) == "" ||
		len(p.Parrafos) == 0 ||
		len(p.Parrafos) > maximoParrafosPlantilla ||
		len(p.Campos) > maximoCamposPlantilla ||
		strings.TrimSpace(p.PermisoGenerar) == "" || p.PermisoGenerar != strings.TrimSpace(p.PermisoGenerar) ||
		strings.ContainsRune(p.PermisoGenerar, '*') ||
		!p.GarantiaMinima.Valida() ||
		!p.Estado.Valido() ||
		strings.TrimSpace(p.CreadaPor) == "" ||
		p.CreadaEn.IsZero() {
		return ErrPlantillaDocumentoInvalida
	}

	total := len(p.Titulo)
	for _, parrafo := range p.Parrafos {
		total += len(parrafo)
		if !textoDocumentalValido(parrafo) {
			return ErrPlantillaDocumentoInvalida
		}
	}
	if !textoDocumentalValido(p.Titulo) || total > maximoCaracteresPlantilla {
		return ErrPlantillaDocumentoInvalida
	}

	campos := make(map[string]CampoPlantillaDocumento, len(p.Campos))
	for _, campo := range p.Campos {
		if err := campo.Validar(); err != nil {
			return err
		}
		clave := strings.TrimSpace(campo.Clave)
		if _, repetido := campos[clave]; repetido {
			return fmt.Errorf("%w: campo repetido %q", ErrPlantillaDocumentoInvalida, clave)
		}
		campos[clave] = campo
	}

	usados := make(map[string]struct{}, len(campos))
	for _, texto := range append([]string{p.Titulo}, p.Parrafos...) {
		marcadores, err := extraerMarcadores(texto)
		if err != nil {
			return err
		}
		for _, clave := range marcadores {
			if _, existe := campos[clave]; !existe {
				return fmt.Errorf("%w: %q", ErrCampoPlantillaDesconocido, clave)
			}
			usados[clave] = struct{}{}
		}
	}
	for clave := range campos {
		if _, usado := usados[clave]; !usado {
			return fmt.Errorf("%w: campo declarado sin marcador %q", ErrPlantillaDocumentoInvalida, clave)
		}
	}

	if len(p.Formatos) == 0 {
		return ErrFormatoDocumentoInvalido
	}
	formatos := make(map[FormatoDocumento]struct{}, len(p.Formatos))
	for _, formato := range p.Formatos {
		if !formato.Valido() {
			return ErrFormatoDocumentoInvalido
		}
		if _, repetido := formatos[formato]; repetido {
			return ErrFormatoDocumentoInvalido
		}
		formatos[formato] = struct{}{}
	}

	switch p.Estado {
	case EstadoPlantillaBorrador:
		if strings.TrimSpace(p.PublicadaPor) != "" || !p.PublicadaEn.IsZero() ||
			strings.TrimSpace(p.AprobacionRef) != "" || strings.TrimSpace(p.MotivoPublicacion) != "" {
			return ErrPlantillaDocumentoInvalida
		}
	case EstadoPlantillaPublicada:
		if strings.TrimSpace(p.PublicadaPor) == "" || p.PublicadaEn.IsZero() ||
			strings.TrimSpace(p.AprobacionRef) == "" || strings.TrimSpace(p.MotivoPublicacion) == "" ||
			p.PublicadaEn.Before(p.CreadaEn) || p.PublicadaPor == p.CreadaPor {
			return ErrPlantillaDocumentoInvalida
		}
	}
	return nil
}

// Publicar aplica segregacion minima: quien creo el borrador no puede
// publicarlo y debe existir una referencia de aprobacion del flujo gobernado.
func (p PlantillaDocumento) Publicar(actor, aprobacionRef, motivo string, fecha time.Time) (PlantillaDocumento, error) {
	if err := p.Validar(); err != nil {
		return PlantillaDocumento{}, err
	}
	if p.Estado != EstadoPlantillaBorrador || strings.TrimSpace(actor) == "" ||
		strings.TrimSpace(actor) == strings.TrimSpace(p.CreadaPor) || strings.TrimSpace(aprobacionRef) == "" ||
		strings.TrimSpace(motivo) == "" || fecha.IsZero() || fecha.Before(p.CreadaEn) {
		return PlantillaDocumento{}, ErrPlantillaDocumentoInvalida
	}
	publicada := p
	publicada.Estado = EstadoPlantillaPublicada
	publicada.PublicadaPor = strings.TrimSpace(actor)
	publicada.PublicadaEn = fecha.UTC()
	publicada.AprobacionRef = strings.TrimSpace(aprobacionRef)
	publicada.MotivoPublicacion = strings.TrimSpace(motivo)
	if err := publicada.Validar(); err != nil {
		return PlantillaDocumento{}, err
	}
	return publicada, nil
}

func (p PlantillaDocumento) HuellaSHA256() (string, error) {
	if err := p.Validar(); err != nil {
		return "", err
	}
	contenido, err := json.Marshal(p)
	if err != nil {
		return "", fmt.Errorf("%w: serializar", ErrPlantillaDocumentoInvalida)
	}
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:]), nil
}

func (p PlantillaDocumento) AdmiteFormato(formato FormatoDocumento) bool {
	for _, permitido := range p.Formatos {
		if permitido == formato {
			return true
		}
	}
	return false
}

// ContenidoDocumento es el modelo neutral que consumen los renderizadores.
type ContenidoDocumento struct {
	Titulo   string   `json:"titulo"`
	Parrafos []string `json:"parrafos"`
}

// Fusionar aplica una sustitucion literal y cerrada. Los valores nunca se
// reinterpretan como plantilla ni se incorporan a logs o trazas.
func (p PlantillaDocumento) Fusionar(datos map[string]string) (ContenidoDocumento, error) {
	if err := p.Validar(); err != nil {
		return ContenidoDocumento{}, err
	}
	if p.Estado != EstadoPlantillaPublicada {
		return ContenidoDocumento{}, ErrPlantillaNoPublicada
	}

	declarados := make(map[string]CampoPlantillaDocumento, len(p.Campos))
	for _, campo := range p.Campos {
		declarados[campo.Clave] = campo
	}
	for clave, valor := range datos {
		if _, existe := declarados[clave]; !existe {
			return ContenidoDocumento{}, fmt.Errorf("%w: %q", ErrCampoPlantillaDesconocido, clave)
		}
		if !textoDocumentalValido(valor) || len(valor) > maximoCaracteresDato {
			return ContenidoDocumento{}, fmt.Errorf("%w: %q", ErrCampoPlantillaInvalido, clave)
		}
	}
	for clave, campo := range declarados {
		if campo.Obligatorio && strings.TrimSpace(datos[clave]) == "" {
			return ContenidoDocumento{}, fmt.Errorf("%w: %q", ErrCampoPlantillaFaltante, clave)
		}
	}

	totalFusionado := 0
	fusionar := func(texto string) (string, error) {
		var salida strings.Builder
		restante := texto
		for {
			indices := marcadorPlantilla.FindStringSubmatchIndex(restante)
			if indices == nil {
				if !reservarBytesFusionados(&totalFusionado, len(restante)) {
					return "", ErrContenidoFusionadoExcesivo
				}
				salida.WriteString(restante)
				return salida.String(), nil
			}
			prefijo := restante[:indices[0]]
			valor := datos[restante[indices[2]:indices[3]]]
			if !reservarBytesFusionados(&totalFusionado, len(prefijo)) ||
				!reservarBytesFusionados(&totalFusionado, len(valor)) {
				return "", ErrContenidoFusionadoExcesivo
			}
			salida.WriteString(prefijo)
			salida.WriteString(valor)
			restante = restante[indices[1]:]
		}
	}
	titulo, err := fusionar(p.Titulo)
	if err != nil {
		return ContenidoDocumento{}, err
	}
	resultado := ContenidoDocumento{Titulo: titulo, Parrafos: make([]string, len(p.Parrafos))}
	for indice, parrafo := range p.Parrafos {
		resultado.Parrafos[indice], err = fusionar(parrafo)
		if err != nil {
			return ContenidoDocumento{}, err
		}
	}
	return resultado, nil
}

type EstadoDocumento string

const (
	EstadoDocumentoBorrador       EstadoDocumento = "borrador"
	EstadoDocumentoGenerado       EstadoDocumento = "generado"
	EstadoDocumentoPendienteFirma EstadoDocumento = "pendiente_firma"
	EstadoDocumentoFirmado        EstadoDocumento = "firmado"
	EstadoDocumentoRegistrado     EstadoDocumento = "registrado"
	EstadoDocumentoAnulado        EstadoDocumento = "anulado"
)

func (e EstadoDocumento) Valido() bool {
	switch e {
	case EstadoDocumentoBorrador, EstadoDocumentoGenerado, EstadoDocumentoPendienteFirma,
		EstadoDocumentoFirmado, EstadoDocumentoRegistrado, EstadoDocumentoAnulado:
		return true
	default:
		return false
	}
}

type EstadoAntivirusDocumento string

const (
	EstadoAntivirusPendiente EstadoAntivirusDocumento = "pendiente"
	EstadoAntivirusLimpio    EstadoAntivirusDocumento = "limpio"
	EstadoAntivirusRechazado EstadoAntivirusDocumento = "rechazado"
	EstadoAntivirusError     EstadoAntivirusDocumento = "error"
	EstadoAntivirusNoAplica  EstadoAntivirusDocumento = "no_aplica_generado"
)

func (e EstadoAntivirusDocumento) Valido() bool {
	switch e {
	case EstadoAntivirusPendiente, EstadoAntivirusLimpio, EstadoAntivirusRechazado,
		EstadoAntivirusError, EstadoAntivirusNoAplica:
		return true
	default:
		return false
	}
}

// MetadatosENI conserva el minimo transversal. Los perfiles completos y las
// normas tecnicas aplicables se validaran en el adaptador de expediente ENI.
type MetadatosENI struct {
	Identificador     string    `json:"identificador"`
	Organo            string    `json:"organo"`
	Origen            string    `json:"origen"`
	EstadoElaboracion string    `json:"estado_elaboracion"`
	TipoDocumental    string    `json:"tipo_documental"`
	FechaCaptura      time.Time `json:"fecha_captura"`
}

func (m MetadatosENI) Validar() error {
	if strings.TrimSpace(m.Identificador) == "" || strings.TrimSpace(m.Organo) == "" ||
		strings.TrimSpace(m.Origen) == "" || strings.TrimSpace(m.EstadoElaboracion) == "" ||
		strings.TrimSpace(m.TipoDocumental) == "" || m.FechaCaptura.IsZero() {
		return ErrDocumentoInvalido
	}
	return nil
}

// DocumentoGenerado es la identidad permanente del artefacto. La referencia
// de contenido es opaca: una URL temporal nunca identifica un documento.
type DocumentoGenerado struct {
	ID                  string                   `json:"id"`
	Version             int                      `json:"version"`
	PlantillaID         string                   `json:"plantilla_id"`
	PlantillaVersion    int                      `json:"plantilla_version"`
	ModuloID            string                   `json:"modulo_id"`
	TipoDocumental      string                   `json:"tipo_documental"`
	ExpedienteRef       string                   `json:"expediente_ref"`
	Formato             FormatoDocumento         `json:"formato"`
	MIME                string                   `json:"mime"`
	NombreFichero       string                   `json:"nombre_fichero"`
	Tamano              int64                    `json:"tamano"`
	HuellaSHA256        string                   `json:"huella_sha256"`
	HuellaDatosHMAC     string                   `json:"huella_datos_hmac"`
	ReferenciaContenido string                   `json:"referencia_contenido"`
	Estado              EstadoDocumento          `json:"estado"`
	EstadoAntivirus     EstadoAntivirusDocumento `json:"estado_antivirus"`
	GeneradoPor         string                   `json:"generado_por"`
	GeneradoEn          time.Time                `json:"generado_en"`
	CorrelacionRef      string                   `json:"correlacion_ref"`
	Motivo              string                   `json:"motivo"`
	ENI                 MetadatosENI             `json:"eni"`
	FirmaRefs           []string                 `json:"firma_refs,omitempty"`
	RegistroRef         string                   `json:"registro_ref,omitempty"`
	CSV                 string                   `json:"csv,omitempty"`
}

func (d DocumentoGenerado) Validar() error {
	if strings.TrimSpace(d.ID) == "" || d.Version < 1 ||
		!esClaveDocumentalCanonica(d.PlantillaID) || d.PlantillaVersion < 1 ||
		!esClaveDocumentalCanonica(d.ModuloID) ||
		!esClaveDocumentalCanonica(d.TipoDocumental) ||
		strings.TrimSpace(d.ExpedienteRef) == "" || !d.Formato.Valido() ||
		d.MIME != d.Formato.MIME() || strings.TrimSpace(d.NombreFichero) == "" ||
		d.Tamano <= 0 || !esSHA256(d.HuellaSHA256) || strings.TrimSpace(d.HuellaDatosHMAC) == "" ||
		strings.TrimSpace(d.ReferenciaContenido) == "" || !d.Estado.Valido() ||
		!d.EstadoAntivirus.Valido() || strings.TrimSpace(d.GeneradoPor) == "" ||
		d.GeneradoEn.IsZero() || strings.TrimSpace(d.CorrelacionRef) == "" ||
		strings.TrimSpace(d.Motivo) == "" {
		return ErrDocumentoInvalido
	}
	return d.ENI.Validar()
}

func extraerMarcadores(texto string) ([]string, error) {
	claves := make([]string, 0)
	restante := texto
	for {
		indices := marcadorPlantilla.FindStringSubmatchIndex(restante)
		if indices == nil {
			if strings.Contains(restante, "{{") || strings.Contains(restante, "}}") {
				return nil, ErrPlantillaDocumentoInvalida
			}
			return claves, nil
		}
		if prefijo := restante[:indices[0]]; strings.Contains(prefijo, "{{") || strings.Contains(prefijo, "}}") {
			return nil, ErrPlantillaDocumentoInvalida
		}
		if len(claves) >= maximoMarcadoresPlantilla {
			return nil, ErrPlantillaDocumentoInvalida
		}
		claves = append(claves, restante[indices[2]:indices[3]])
		restante = restante[indices[1]:]
	}
}

func reservarBytesFusionados(total *int, incremento int) bool {
	if incremento < 0 || *total > maximoBytesFusionados-incremento {
		return false
	}
	*total += incremento
	return true
}

func esClaveDocumentalCanonica(valor string) bool {
	return valor == strings.TrimSpace(valor) && claveDocumentalValida.MatchString(valor)
}

func textoDocumentalValido(texto string) bool {
	if !utf8.ValidString(texto) {
		return false
	}
	for _, caracter := range texto {
		if caracter == '\t' || caracter == '\n' || caracter == '\r' {
			continue
		}
		if caracter < 0x20 || caracter == 0x7f {
			return false
		}
	}
	return true
}

func esSHA256(valor string) bool {
	if len(valor) != 64 {
		return false
	}
	for _, caracter := range valor {
		if (caracter < '0' || caracter > '9') && (caracter < 'a' || caracter > 'f') {
			return false
		}
	}
	return true
}
