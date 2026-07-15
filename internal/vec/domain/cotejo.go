package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	ErrPoliticaCotejoInvalida       = errors.New("vec: politica de cotejo invalida")
	ErrPoliticaCotejoNoPublicada    = errors.New("vec: politica de cotejo no publicada")
	ErrCodigoCotejoInvalido         = errors.New("vec: codigo de cotejo invalido")
	ErrTransicionCodigoCotejo       = errors.New("vec: transicion de codigo de cotejo invalida")
	ErrVersionCotejoInvalida        = errors.New("vec: version documental de cotejo invalida")
	ErrEvidenciaEmisionInvalida     = errors.New("vec: evidencia de emision invalida")
	ErrDocumentoNoAdmitidoPorCotejo = errors.New("vec: documento no admitido por la politica de cotejo")
)

const (
	maximoValoresPoliticaCotejo = 256
	minimoEntropiaCodigoCotejo  = 128
	maximoDiasActivacionCotejo  = 365
	maximoDiasCotejo            = 36_600
)

const (
	AccionPoliticaCotejoBorradorCreada      = "vec.documentos.cotejo.politica.borrador.creado"
	AccionPoliticaCotejoBorradorActualizada = "vec.documentos.cotejo.politica.borrador.actualizado"
	AccionPoliticaCotejoPublicada           = "vec.documentos.cotejo.politica.publicada"
	AccionPoliticaCotejoRetirada            = "vec.documentos.cotejo.politica.retirada"
	AccionCodigoCotejoReservado             = "vec.documentos.cotejo.codigo.reservado"
	AccionCodigoCotejoActivado              = "vec.documentos.cotejo.codigo.activado"
	AccionCodigoCotejoRetirado              = "vec.documentos.cotejo.codigo.retirado"
	AccionCodigoCotejoSustituido            = "vec.documentos.cotejo.codigo.sustituido"
	AccionConsultaPublicaCotejo             = "vec.documentos.cotejo.consulta.publica"
	AccionConsultaProtegidaCotejo           = "vec.documentos.cotejo.consulta.protegida"
)

// ClaseAccesoCotejo es una frontera de seguridad estable. Las politicas
// versionadas deciden que documentos usan cada clase y que campos se muestran.
type ClaseAccesoCotejo string

const (
	ClaseAccesoCotejoPublico   ClaseAccesoCotejo = "publico"
	ClaseAccesoCotejoProtegido ClaseAccesoCotejo = "protegido"
	ClaseAccesoCotejoInterno   ClaseAccesoCotejo = "interno"
)

func (c ClaseAccesoCotejo) Valida() bool {
	return c == ClaseAccesoCotejoPublico || c == ClaseAccesoCotejoProtegido || c == ClaseAccesoCotejoInterno
}

// CampoPublicoCotejo es una lista cerrada por privacidad, no un catalogo de
// negocio. Incorporar un campo nuevo requiere revisar de forma expresa que no
// permita identificar a una persona o expediente.
type CampoPublicoCotejo string

const (
	CampoPublicoCotejoOrgano         CampoPublicoCotejo = "organo"
	CampoPublicoCotejoTipoDocumental CampoPublicoCotejo = "tipo_documental"
	CampoPublicoCotejoFechaEmision   CampoPublicoCotejo = "fecha_emision"
	CampoPublicoCotejoHuellaSHA256   CampoPublicoCotejo = "huella_sha256"
)

func (c CampoPublicoCotejo) Valido() bool {
	switch c {
	case CampoPublicoCotejoOrgano, CampoPublicoCotejoTipoDocumental,
		CampoPublicoCotejoFechaEmision, CampoPublicoCotejoHuellaSHA256:
		return true
	default:
		return false
	}
}

type EstadoPoliticaCotejo string

const (
	EstadoPoliticaCotejoBorrador  EstadoPoliticaCotejo = "borrador"
	EstadoPoliticaCotejoPublicada EstadoPoliticaCotejo = "publicada"
	EstadoPoliticaCotejoRetirada  EstadoPoliticaCotejo = "retirada"
)

func (e EstadoPoliticaCotejo) Valido() bool {
	return e == EstadoPoliticaCotejoBorrador || e == EstadoPoliticaCotejoPublicada || e == EstadoPoliticaCotejoRetirada
}

// PoliticaCotejo es una version gobernada e inmutable tras su publicacion. Las
// listas son datos de configuracion: agregar un modulo, tipo o clasificacion no
// obliga a recompilar el nucleo.
type PoliticaCotejo struct {
	ID                       string               `json:"id"`
	Version                  int                  `json:"version"`
	Revision                 int                  `json:"revision"`
	VersionAnteriorRef       string               `json:"version_anterior_ref,omitempty"`
	Nombre                   string               `json:"nombre"`
	Descripcion              string               `json:"descripcion"`
	Modulos                  []string             `json:"modulos"`
	TiposDocumentales        []string             `json:"tipos_documentales"`
	Clasificaciones          []string             `json:"clasificaciones"`
	ClaseAcceso              ClaseAccesoCotejo    `json:"clase_acceso"`
	CamposPublicos           []CampoPublicoCotejo `json:"campos_publicos,omitempty"`
	PermiteDescargaDocumento bool                 `json:"permite_descarga_documento"`
	RequiereTitularidad      bool                 `json:"requiere_titularidad"`
	RolesTitularidad         []string             `json:"roles_titularidad,omitempty"`
	RequiereFirma            bool                 `json:"requiere_firma"`
	RequiereSelloTiempo      bool                 `json:"requiere_sello_tiempo"`
	RequiereRegistro         bool                 `json:"requiere_registro"`
	GarantiaMinima           AuthAssurance        `json:"garantia_minima"`
	DiasPlazoActivacion      int                  `json:"dias_plazo_activacion"`
	DiasDisponibilidad       int                  `json:"dias_disponibilidad"`
	Estado                   EstadoPoliticaCotejo `json:"estado"`
	FuenteRef                string               `json:"fuente_ref"`
	MotivoCreacion           string               `json:"motivo_creacion"`
	CreadaPor                string               `json:"creada_por"`
	CreadaEn                 time.Time            `json:"creada_en"`
	ActualizadaPor           string               `json:"actualizada_por,omitempty"`
	ActualizadaEn            time.Time            `json:"actualizada_en,omitempty"`
	MotivoActualizacion      string               `json:"motivo_actualizacion,omitempty"`
	PublicadaPor             string               `json:"publicada_por,omitempty"`
	PublicadaEn              time.Time            `json:"publicada_en,omitempty"`
	AprobacionRef            string               `json:"aprobacion_ref,omitempty"`
	MotivoPublicacion        string               `json:"motivo_publicacion,omitempty"`
	RetiradaPor              string               `json:"retirada_por,omitempty"`
	RetiradaEn               time.Time            `json:"retirada_en,omitempty"`
	RetiradaAprobacionRef    string               `json:"retirada_aprobacion_ref,omitempty"`
	MotivoRetirada           string               `json:"motivo_retirada,omitempty"`
}

func (p PoliticaCotejo) Referencia() string {
	return "politica-cotejo:" + strings.TrimSpace(p.ID) + ":v" + strconv.Itoa(p.Version)
}

func (p PoliticaCotejo) Validar() error {
	if !esClaveDocumentalCanonica(p.ID) || p.Version < 1 || p.Revision < 1 ||
		!textoDocumentalNoVacioValido(p.Nombre) || !textoDocumentalNoVacioValido(p.Descripcion) ||
		!p.ClaseAcceso.Valida() || !p.GarantiaMinima.Valida() ||
		p.DiasPlazoActivacion < 1 || p.DiasPlazoActivacion > maximoDiasActivacionCotejo ||
		p.DiasDisponibilidad < 1 || p.DiasDisponibilidad > maximoDiasCotejo ||
		!p.Estado.Valido() || !referenciaDocumentalValida(p.FuenteRef) ||
		!textoDocumentalNoVacioValido(p.MotivoCreacion) || !referenciaDocumentalValida(p.CreadaPor) || p.CreadaEn.IsZero() {
		return ErrPoliticaCotejoInvalida
	}
	if (p.Version == 1 && strings.TrimSpace(p.VersionAnteriorRef) != "") ||
		(p.Version > 1 && p.VersionAnteriorRef != "politica-cotejo:"+p.ID+":v"+strconv.Itoa(p.Version-1)) {
		return ErrPoliticaCotejoInvalida
	}
	if _, err := canonizarClavesCotejo(p.Modulos); err != nil {
		return err
	}
	if _, err := canonizarClavesCotejo(p.TiposDocumentales); err != nil {
		return err
	}
	if _, err := canonizarClavesCotejo(p.Clasificaciones); err != nil {
		return err
	}
	if _, err := canonizarCamposPublicosCotejo(p.CamposPublicos); err != nil {
		return err
	}
	if p.RequiereTitularidad {
		if _, err := canonizarClavesCotejo(p.RolesTitularidad); err != nil {
			return err
		}
	} else if len(p.RolesTitularidad) != 0 {
		return ErrPoliticaCotejoInvalida
	}
	if p.ClaseAcceso == ClaseAccesoCotejoInterno && len(p.CamposPublicos) != 0 {
		return ErrPoliticaCotejoInvalida
	}
	actualizacionVacia := strings.TrimSpace(p.ActualizadaPor) == "" && p.ActualizadaEn.IsZero() &&
		strings.TrimSpace(p.MotivoActualizacion) == ""
	actualizacionValida := referenciaDocumentalValida(p.ActualizadaPor) && !p.ActualizadaEn.IsZero() &&
		!p.ActualizadaEn.Before(p.CreadaEn) && textoDocumentalNoVacioValido(p.MotivoActualizacion)
	if !actualizacionVacia && !actualizacionValida {
		return ErrPoliticaCotejoInvalida
	}
	switch p.Estado {
	case EstadoPoliticaCotejoBorrador:
		if (p.Revision == 1 && !actualizacionVacia) || (p.Revision > 1 && !actualizacionValida) ||
			strings.TrimSpace(p.PublicadaPor) != "" || !p.PublicadaEn.IsZero() ||
			strings.TrimSpace(p.AprobacionRef) != "" || strings.TrimSpace(p.MotivoPublicacion) != "" ||
			strings.TrimSpace(p.RetiradaPor) != "" || !p.RetiradaEn.IsZero() ||
			strings.TrimSpace(p.RetiradaAprobacionRef) != "" || strings.TrimSpace(p.MotivoRetirada) != "" {
			return ErrPoliticaCotejoInvalida
		}
	case EstadoPoliticaCotejoPublicada:
		if (actualizacionVacia && p.Revision != 2) || (actualizacionValida && p.Revision < 3) ||
			(actualizacionValida && p.PublicadaEn.Before(p.ActualizadaEn)) ||
			!referenciaDocumentalValida(p.PublicadaPor) || p.PublicadaEn.IsZero() || p.PublicadaEn.Before(p.CreadaEn) ||
			!referenciaDocumentalValida(p.AprobacionRef) || !textoDocumentalNoVacioValido(p.MotivoPublicacion) ||
			strings.TrimSpace(p.RetiradaPor) != "" || !p.RetiradaEn.IsZero() ||
			strings.TrimSpace(p.RetiradaAprobacionRef) != "" || strings.TrimSpace(p.MotivoRetirada) != "" {
			return ErrPoliticaCotejoInvalida
		}
	case EstadoPoliticaCotejoRetirada:
		if (actualizacionVacia && p.Revision != 3) || (actualizacionValida && p.Revision < 4) ||
			(actualizacionValida && p.PublicadaEn.Before(p.ActualizadaEn)) ||
			!referenciaDocumentalValida(p.PublicadaPor) || p.PublicadaEn.IsZero() ||
			!referenciaDocumentalValida(p.AprobacionRef) || !textoDocumentalNoVacioValido(p.MotivoPublicacion) ||
			!referenciaDocumentalValida(p.RetiradaPor) || p.RetiradaEn.IsZero() || p.RetiradaEn.Before(p.PublicadaEn) ||
			!referenciaDocumentalValida(p.RetiradaAprobacionRef) || !textoDocumentalNoVacioValido(p.MotivoRetirada) {
			return ErrPoliticaCotejoInvalida
		}
	}
	return nil
}

// ActualizarBorrador copia solo configuracion editable. Identidad, version,
// autoria inicial y estado se conservan para impedir que una edicion se haga
// pasar por una politica distinta o ya publicada.
func (p PoliticaCotejo) ActualizarBorrador(propuesta PoliticaCotejo, actor, motivo string, fecha time.Time) (PoliticaCotejo, error) {
	if err := p.Validar(); err != nil || p.Estado != EstadoPoliticaCotejoBorrador ||
		!referenciaDocumentalValida(actor) || !textoDocumentalNoVacioValido(motivo) || fecha.IsZero() ||
		fecha.Before(p.CreadaEn) || (!p.ActualizadaEn.IsZero() && fecha.Before(p.ActualizadaEn)) {
		return PoliticaCotejo{}, ErrPoliticaCotejoInvalida
	}
	actualizada := p
	actualizada.Revision++
	actualizada.Nombre = propuesta.Nombre
	actualizada.Descripcion = propuesta.Descripcion
	actualizada.Modulos = append([]string(nil), propuesta.Modulos...)
	actualizada.TiposDocumentales = append([]string(nil), propuesta.TiposDocumentales...)
	actualizada.Clasificaciones = append([]string(nil), propuesta.Clasificaciones...)
	actualizada.ClaseAcceso = propuesta.ClaseAcceso
	actualizada.CamposPublicos = append([]CampoPublicoCotejo(nil), propuesta.CamposPublicos...)
	actualizada.PermiteDescargaDocumento = propuesta.PermiteDescargaDocumento
	actualizada.RequiereTitularidad = propuesta.RequiereTitularidad
	actualizada.RolesTitularidad = append([]string(nil), propuesta.RolesTitularidad...)
	actualizada.RequiereFirma = propuesta.RequiereFirma
	actualizada.RequiereSelloTiempo = propuesta.RequiereSelloTiempo
	actualizada.RequiereRegistro = propuesta.RequiereRegistro
	actualizada.GarantiaMinima = propuesta.GarantiaMinima
	actualizada.DiasPlazoActivacion = propuesta.DiasPlazoActivacion
	actualizada.DiasDisponibilidad = propuesta.DiasDisponibilidad
	actualizada.FuenteRef = propuesta.FuenteRef
	actualizada.ActualizadaPor = strings.TrimSpace(actor)
	actualizada.ActualizadaEn = fecha.UTC()
	actualizada.MotivoActualizacion = strings.TrimSpace(motivo)
	return actualizada.ClonarCanonica()
}

func (p PoliticaCotejo) ClonarCanonica() (PoliticaCotejo, error) {
	canonico := p
	var err error
	if canonico.Modulos, err = canonizarClavesCotejo(p.Modulos); err != nil {
		return PoliticaCotejo{}, err
	}
	if canonico.TiposDocumentales, err = canonizarClavesCotejo(p.TiposDocumentales); err != nil {
		return PoliticaCotejo{}, err
	}
	if canonico.Clasificaciones, err = canonizarClavesCotejo(p.Clasificaciones); err != nil {
		return PoliticaCotejo{}, err
	}
	if canonico.CamposPublicos, err = canonizarCamposPublicosCotejo(p.CamposPublicos); err != nil {
		return PoliticaCotejo{}, err
	}
	if p.RequiereTitularidad {
		if canonico.RolesTitularidad, err = canonizarClavesCotejo(p.RolesTitularidad); err != nil {
			return PoliticaCotejo{}, err
		}
	} else {
		canonico.RolesTitularidad = nil
	}
	if err := canonico.Validar(); err != nil {
		return PoliticaCotejo{}, err
	}
	return canonico, nil
}

func (p PoliticaCotejo) HuellaSHA256() (string, error) {
	canonico, err := p.ClonarCanonica()
	if err != nil {
		return "", err
	}
	contenido, err := json.Marshal(canonico)
	if err != nil {
		return "", err
	}
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:]), nil
}

func (p PoliticaCotejo) Publicar(actor, aprobacionRef, motivo string, fecha time.Time) (PoliticaCotejo, error) {
	if err := p.Validar(); err != nil || p.Estado != EstadoPoliticaCotejoBorrador ||
		!referenciaDocumentalValida(actor) || !referenciaDocumentalValida(aprobacionRef) ||
		!textoDocumentalNoVacioValido(motivo) || fecha.IsZero() || fecha.Before(p.CreadaEn) {
		return PoliticaCotejo{}, ErrPoliticaCotejoInvalida
	}
	publicada := p
	publicada.Revision++
	publicada.Estado = EstadoPoliticaCotejoPublicada
	publicada.PublicadaPor = strings.TrimSpace(actor)
	publicada.PublicadaEn = fecha.UTC()
	publicada.AprobacionRef = strings.TrimSpace(aprobacionRef)
	publicada.MotivoPublicacion = strings.TrimSpace(motivo)
	return publicada.ClonarCanonica()
}

func (p PoliticaCotejo) Retirar(actor, aprobacionRef, motivo string, fecha time.Time) (PoliticaCotejo, error) {
	if err := p.Validar(); err != nil || p.Estado != EstadoPoliticaCotejoPublicada ||
		!referenciaDocumentalValida(actor) || !referenciaDocumentalValida(aprobacionRef) ||
		!textoDocumentalNoVacioValido(motivo) || fecha.IsZero() || fecha.Before(p.PublicadaEn) {
		return PoliticaCotejo{}, ErrPoliticaCotejoInvalida
	}
	retirada := p
	retirada.Revision++
	retirada.Estado = EstadoPoliticaCotejoRetirada
	retirada.RetiradaPor = strings.TrimSpace(actor)
	retirada.RetiradaEn = fecha.UTC()
	retirada.RetiradaAprobacionRef = strings.TrimSpace(aprobacionRef)
	retirada.MotivoRetirada = strings.TrimSpace(motivo)
	return retirada.ClonarCanonica()
}

func (p PoliticaCotejo) Admite(documento DocumentoLogico) bool {
	if p.Validar() != nil || p.Estado != EstadoPoliticaCotejoPublicada || documento.Validar() != nil {
		return false
	}
	return contieneCadenaCotejo(p.Modulos, documento.ModuloID) &&
		contieneCadenaCotejo(p.TiposDocumentales, documento.TipoDocumental) &&
		contieneCadenaCotejo(p.Clasificaciones, documento.Clasificacion)
}

type ReferenciaPoliticaCotejo struct {
	ID           string `json:"id"`
	Version      int    `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
}

func (r ReferenciaPoliticaCotejo) Validar() error {
	if !esClaveDocumentalCanonica(r.ID) || r.Version < 1 || !esSHA256(r.HuellaSHA256) {
		return ErrPoliticaCotejoInvalida
	}
	return nil
}

type AplicacionPoliticaCotejo struct {
	Referencia               ReferenciaPoliticaCotejo `json:"referencia"`
	ClaseAcceso              ClaseAccesoCotejo        `json:"clase_acceso"`
	CamposPublicos           []CampoPublicoCotejo     `json:"campos_publicos,omitempty"`
	PermiteDescargaDocumento bool                     `json:"permite_descarga_documento"`
	RequiereTitularidad      bool                     `json:"requiere_titularidad"`
	RolesTitularidad         []string                 `json:"roles_titularidad,omitempty"`
	RequiereFirma            bool                     `json:"requiere_firma"`
	RequiereSelloTiempo      bool                     `json:"requiere_sello_tiempo"`
	RequiereRegistro         bool                     `json:"requiere_registro"`
	GarantiaMinima           AuthAssurance            `json:"garantia_minima"`
	DiasPlazoActivacion      int                      `json:"dias_plazo_activacion"`
	DiasDisponibilidad       int                      `json:"dias_disponibilidad"`
}

func (p PoliticaCotejo) Aplicacion() (AplicacionPoliticaCotejo, error) {
	if err := p.Validar(); err != nil || p.Estado != EstadoPoliticaCotejoPublicada {
		return AplicacionPoliticaCotejo{}, ErrPoliticaCotejoNoPublicada
	}
	huella, err := p.HuellaSHA256()
	if err != nil {
		return AplicacionPoliticaCotejo{}, err
	}
	campos, err := canonizarCamposPublicosCotejo(p.CamposPublicos)
	if err != nil {
		return AplicacionPoliticaCotejo{}, err
	}
	var rolesTitularidad []string
	if p.RequiereTitularidad {
		rolesTitularidad, err = canonizarClavesCotejo(p.RolesTitularidad)
		if err != nil {
			return AplicacionPoliticaCotejo{}, err
		}
	}
	return AplicacionPoliticaCotejo{
		Referencia:               ReferenciaPoliticaCotejo{ID: p.ID, Version: p.Version, HuellaSHA256: huella},
		ClaseAcceso:              p.ClaseAcceso,
		CamposPublicos:           campos,
		PermiteDescargaDocumento: p.PermiteDescargaDocumento,
		RequiereTitularidad:      p.RequiereTitularidad,
		RolesTitularidad:         rolesTitularidad,
		RequiereFirma:            p.RequiereFirma,
		RequiereSelloTiempo:      p.RequiereSelloTiempo,
		RequiereRegistro:         p.RequiereRegistro,
		GarantiaMinima:           p.GarantiaMinima,
		DiasPlazoActivacion:      p.DiasPlazoActivacion,
		DiasDisponibilidad:       p.DiasDisponibilidad,
	}, nil
}

func (a AplicacionPoliticaCotejo) Validar() error {
	if a.Referencia.Validar() != nil || !a.ClaseAcceso.Valida() || !a.GarantiaMinima.Valida() ||
		a.DiasPlazoActivacion < 1 || a.DiasPlazoActivacion > maximoDiasActivacionCotejo ||
		a.DiasDisponibilidad < 1 || a.DiasDisponibilidad > maximoDiasCotejo {
		return ErrPoliticaCotejoInvalida
	}
	if _, err := canonizarCamposPublicosCotejo(a.CamposPublicos); err != nil {
		return err
	}
	if a.RequiereTitularidad {
		if _, err := canonizarClavesCotejo(a.RolesTitularidad); err != nil {
			return err
		}
	} else if len(a.RolesTitularidad) != 0 {
		return ErrPoliticaCotejoInvalida
	}
	if a.ClaseAcceso == ClaseAccesoCotejoInterno && len(a.CamposPublicos) != 0 {
		return ErrPoliticaCotejoInvalida
	}
	return nil
}

type EstadoCodigoCotejo string

const (
	EstadoCodigoCotejoReservado  EstadoCodigoCotejo = "reservado"
	EstadoCodigoCotejoActivo     EstadoCodigoCotejo = "activo"
	EstadoCodigoCotejoRetirado   EstadoCodigoCotejo = "retirado"
	EstadoCodigoCotejoSustituido EstadoCodigoCotejo = "sustituido"
)

func (e EstadoCodigoCotejo) Valido() bool {
	return e == EstadoCodigoCotejoReservado || e == EstadoCodigoCotejoActivo ||
		e == EstadoCodigoCotejoRetirado || e == EstadoCodigoCotejoSustituido
}

// VersionEmitidaCotejo enlaza el CSV con los bytes exactos. Nunca se rellena
// con valores declarados por un cliente: procede de la fuente de evidencias de
// emision y del repositorio documental del servidor.
type VersionEmitidaCotejo struct {
	RepresentacionID      string    `json:"representacion_id"`
	ReferenciaContenido   string    `json:"referencia_contenido"`
	HuellaContenidoSHA256 string    `json:"huella_contenido_sha256"`
	MIME                  string    `json:"mime"`
	Tamano                int64     `json:"tamano"`
	FirmaRefs             []string  `json:"firma_refs,omitempty"`
	SelloTiempoRefs       []string  `json:"sello_tiempo_refs,omitempty"`
	ValidacionFirmaRef    string    `json:"validacion_firma_ref"`
	RegistroRef           string    `json:"registro_ref"`
	EmitidaEn             time.Time `json:"emitida_en"`
}

func (v VersionEmitidaCotejo) Validar() error {
	if !referenciaDocumentalValida(v.RepresentacionID) || !referenciaDocumentalValida(v.ReferenciaContenido) ||
		!esSHA256(v.HuellaContenidoSHA256) || strings.TrimSpace(v.MIME) == "" || v.Tamano < 1 ||
		(strings.TrimSpace(v.ValidacionFirmaRef) != "" && !referenciaDocumentalValida(v.ValidacionFirmaRef)) ||
		(strings.TrimSpace(v.RegistroRef) != "" && !referenciaDocumentalValida(v.RegistroRef)) || v.EmitidaEn.IsZero() {
		return ErrVersionCotejoInvalida
	}
	if _, err := canonizarReferenciasCotejo(v.FirmaRefs); err != nil {
		return err
	}
	if _, err := canonizarReferenciasCotejo(v.SelloTiempoRefs); err != nil {
		return err
	}
	if len(v.FirmaRefs) > 0 && strings.TrimSpace(v.ValidacionFirmaRef) == "" {
		return ErrVersionCotejoInvalida
	}
	return nil
}

func (v VersionEmitidaCotejo) clonarCanonica() (VersionEmitidaCotejo, error) {
	canonico := v
	var err error
	if canonico.FirmaRefs, err = canonizarReferenciasCotejo(v.FirmaRefs); err != nil {
		return VersionEmitidaCotejo{}, err
	}
	if canonico.SelloTiempoRefs, err = canonizarReferenciasCotejo(v.SelloTiempoRefs); err != nil {
		return VersionEmitidaCotejo{}, err
	}
	if err := canonico.Validar(); err != nil {
		return VersionEmitidaCotejo{}, err
	}
	return canonico, nil
}

type EvidenciaEmisionDocumento struct {
	Documento      ReferenciaDocumento  `json:"documento"`
	VersionEmitida VersionEmitidaCotejo `json:"version_emitida"`
	Apta           bool                 `json:"apta"`
	EvidenciaRef   string               `json:"evidencia_ref"`
}

func (e EvidenciaEmisionDocumento) Validar() error {
	if e.Documento.Validar() != nil || e.VersionEmitida.Validar() != nil || !e.Apta ||
		!referenciaDocumentalValida(e.EvidenciaRef) {
		return ErrEvidenciaEmisionInvalida
	}
	return nil
}

// CodigoCotejo nunca expone ni serializa el valor del CSV. IndiceCodigoHMAC
// permite buscarlo sin conservarlo en claro; ProteccionRef apunta a su custodia
// cifrada para recuperar el mismo valor en reintentos internos autorizados.
type CodigoCotejo struct {
	ID                  string                   `json:"id"`
	Revision            int                      `json:"revision"`
	Documento           ReferenciaDocumento      `json:"documento"`
	ModuloID            string                   `json:"modulo_id"`
	TipoDocumental      string                   `json:"tipo_documental"`
	Clasificacion       string                   `json:"clasificacion"`
	Organo              string                   `json:"organo"`
	ExpedienteRef       string                   `json:"-"`
	IndiceCodigoHMAC    string                   `json:"-"`
	ProteccionRef       string                   `json:"-"`
	VersionGenerador    string                   `json:"version_generador"`
	EntropiaBits        int                      `json:"entropia_bits"`
	Politica            AplicacionPoliticaCotejo `json:"politica"`
	Estado              EstadoCodigoCotejo       `json:"estado"`
	ReservadoPor        string                   `json:"reservado_por"`
	ReservadoEn         time.Time                `json:"reservado_en"`
	ReservaExpiraEn     time.Time                `json:"reserva_expira_en"`
	MotivoReserva       string                   `json:"motivo_reserva"`
	CorrelacionRef      string                   `json:"correlacion_ref"`
	VersionEmitida      *VersionEmitidaCotejo    `json:"version_emitida,omitempty"`
	ActivadoPor         string                   `json:"activado_por,omitempty"`
	ActivadoEn          time.Time                `json:"activado_en,omitempty"`
	ActivacionRef       string                   `json:"activacion_ref,omitempty"`
	EvidenciaEmisionRef string                   `json:"evidencia_emision_ref,omitempty"`
	MotivoActivacion    string                   `json:"motivo_activacion,omitempty"`
	DisponibleDesde     time.Time                `json:"disponible_desde,omitempty"`
	DisponibleHasta     time.Time                `json:"disponible_hasta,omitempty"`
	RetiradoPor         string                   `json:"retirado_por,omitempty"`
	RetiradoEn          time.Time                `json:"retirado_en,omitempty"`
	RetiradaRef         string                   `json:"retirada_ref,omitempty"`
	MotivoRetirada      string                   `json:"motivo_retirada,omitempty"`
	SustituidoPorRef    string                   `json:"sustituido_por_ref,omitempty"`
}

func (c CodigoCotejo) Referencia() string {
	return "cotejo:" + strings.TrimSpace(c.ID)
}

func (c CodigoCotejo) Validar() error {
	if !referenciaDocumentalValida(c.ID) || c.Revision < 1 || c.Documento.Validar() != nil ||
		!esClaveDocumentalCanonica(c.ModuloID) || !esClaveDocumentalCanonica(c.TipoDocumental) ||
		!esClaveDocumentalCanonica(c.Clasificacion) || strings.TrimSpace(c.Organo) == "" ||
		!referenciaDocumentalValida(c.ExpedienteRef) || !esHuellaHMACSHA256(c.IndiceCodigoHMAC) ||
		!referenciaDocumentalValida(c.ProteccionRef) || !referenciaDocumentalValida(c.VersionGenerador) ||
		c.EntropiaBits < minimoEntropiaCodigoCotejo || c.Politica.Validar() != nil || !c.Estado.Valido() ||
		!referenciaDocumentalValida(c.ReservadoPor) || c.ReservadoEn.IsZero() ||
		!c.ReservaExpiraEn.After(c.ReservadoEn) || !textoDocumentalNoVacioValido(c.MotivoReserva) ||
		!referenciaDocumentalValida(c.CorrelacionRef) {
		return ErrCodigoCotejoInvalido
	}
	switch c.Estado {
	case EstadoCodigoCotejoReservado:
		if c.Revision != 1 || c.VersionEmitida != nil || strings.TrimSpace(c.ActivadoPor) != "" ||
			!c.ActivadoEn.IsZero() || strings.TrimSpace(c.ActivacionRef) != "" ||
			strings.TrimSpace(c.EvidenciaEmisionRef) != "" || strings.TrimSpace(c.MotivoActivacion) != "" ||
			!c.DisponibleDesde.IsZero() || !c.DisponibleHasta.IsZero() ||
			strings.TrimSpace(c.RetiradoPor) != "" || !c.RetiradoEn.IsZero() ||
			strings.TrimSpace(c.RetiradaRef) != "" || strings.TrimSpace(c.MotivoRetirada) != "" ||
			strings.TrimSpace(c.SustituidoPorRef) != "" {
			return ErrCodigoCotejoInvalido
		}
	case EstadoCodigoCotejoActivo:
		if !c.datosActivacionValidos() || strings.TrimSpace(c.RetiradoPor) != "" || !c.RetiradoEn.IsZero() ||
			strings.TrimSpace(c.RetiradaRef) != "" || strings.TrimSpace(c.MotivoRetirada) != "" ||
			strings.TrimSpace(c.SustituidoPorRef) != "" {
			return ErrCodigoCotejoInvalido
		}
	case EstadoCodigoCotejoRetirado:
		if !c.datosActivacionValidos() || !c.datosRetiradaValidos() || strings.TrimSpace(c.SustituidoPorRef) != "" {
			return ErrCodigoCotejoInvalido
		}
	case EstadoCodigoCotejoSustituido:
		if !c.datosActivacionValidos() || !c.datosRetiradaValidos() || !referenciaDocumentalValida(c.SustituidoPorRef) ||
			c.SustituidoPorRef == c.Referencia() {
			return ErrCodigoCotejoInvalido
		}
	}
	return nil
}

func (c CodigoCotejo) datosActivacionValidos() bool {
	return c.VersionEmitida != nil && c.VersionEmitida.Validar() == nil &&
		referenciaDocumentalValida(c.ActivadoPor) && !c.ActivadoEn.IsZero() &&
		referenciaDocumentalValida(c.ActivacionRef) && referenciaDocumentalValida(c.EvidenciaEmisionRef) &&
		textoDocumentalNoVacioValido(c.MotivoActivacion) && !c.ActivadoEn.Before(c.ReservadoEn) &&
		!c.DisponibleDesde.IsZero() && c.DisponibleHasta.After(c.DisponibleDesde) &&
		!c.DisponibleDesde.Before(c.ActivadoEn) &&
		!c.ActivadoEn.Before(c.VersionEmitida.EmitidaEn)
}

func (c CodigoCotejo) datosRetiradaValidos() bool {
	return referenciaDocumentalValida(c.RetiradoPor) && !c.RetiradoEn.IsZero() &&
		!c.RetiradoEn.Before(c.ActivadoEn) && referenciaDocumentalValida(c.RetiradaRef) &&
		textoDocumentalNoVacioValido(c.MotivoRetirada)
}

func (c CodigoCotejo) ClonarCanonico() (CodigoCotejo, error) {
	canonico := c
	campos, err := canonizarCamposPublicosCotejo(c.Politica.CamposPublicos)
	if err != nil {
		return CodigoCotejo{}, err
	}
	canonico.Politica.CamposPublicos = campos
	if c.Politica.RequiereTitularidad {
		roles, err := canonizarClavesCotejo(c.Politica.RolesTitularidad)
		if err != nil {
			return CodigoCotejo{}, err
		}
		canonico.Politica.RolesTitularidad = roles
	} else {
		canonico.Politica.RolesTitularidad = nil
	}
	if c.VersionEmitida != nil {
		version, err := c.VersionEmitida.clonarCanonica()
		if err != nil {
			return CodigoCotejo{}, err
		}
		canonico.VersionEmitida = &version
	}
	if err := canonico.Validar(); err != nil {
		return CodigoCotejo{}, err
	}
	return canonico, nil
}

func (c CodigoCotejo) HuellaEstadoSHA256() (string, error) {
	canonico, err := c.ClonarCanonico()
	if err != nil {
		return "", err
	}
	// Los campos ocultos al JSON de salida siguen formando parte de la huella
	// de integridad. Asi una sustitucion del indice, la custodia del secreto o
	// el expediente no puede pasar inadvertida en la auditoria.
	contenido, err := json.Marshal(struct {
		CodigoCotejo
		ExpedienteRef    string `json:"expediente_ref"`
		IndiceCodigoHMAC string `json:"indice_codigo_hmac"`
		ProteccionRef    string `json:"proteccion_ref"`
	}{
		CodigoCotejo:     canonico,
		ExpedienteRef:    canonico.ExpedienteRef,
		IndiceCodigoHMAC: canonico.IndiceCodigoHMAC,
		ProteccionRef:    canonico.ProteccionRef,
	})
	if err != nil {
		return "", err
	}
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:]), nil
}

func (c CodigoCotejo) Activar(actor, activacionRef, motivo string, evidencia EvidenciaEmisionDocumento, fecha time.Time) (CodigoCotejo, error) {
	if err := c.Validar(); err != nil || c.Estado != EstadoCodigoCotejoReservado ||
		!referenciaDocumentalValida(actor) || !referenciaDocumentalValida(activacionRef) ||
		!textoDocumentalNoVacioValido(motivo) ||
		evidencia.Validar() != nil || evidencia.Documento != c.Documento || fecha.IsZero() ||
		fecha.Before(c.ReservadoEn) || fecha.After(c.ReservaExpiraEn) ||
		evidencia.VersionEmitida.EmitidaEn.After(fecha) || !c.Politica.admiteEvidencia(evidencia.VersionEmitida) {
		return CodigoCotejo{}, ErrTransicionCodigoCotejo
	}
	version, err := evidencia.VersionEmitida.clonarCanonica()
	if err != nil {
		return CodigoCotejo{}, err
	}
	activo := c
	activo.Revision++
	activo.Estado = EstadoCodigoCotejoActivo
	activo.VersionEmitida = &version
	activo.ActivadoPor = strings.TrimSpace(actor)
	activo.ActivadoEn = fecha.UTC()
	activo.ActivacionRef = strings.TrimSpace(activacionRef)
	activo.EvidenciaEmisionRef = evidencia.EvidenciaRef
	activo.MotivoActivacion = strings.TrimSpace(motivo)
	activo.DisponibleDesde = fecha.UTC()
	activo.DisponibleHasta = fecha.UTC().AddDate(0, 0, c.Politica.DiasDisponibilidad)
	return activo.ClonarCanonico()
}

func (a AplicacionPoliticaCotejo) admiteEvidencia(version VersionEmitidaCotejo) bool {
	return (!a.RequiereFirma || (len(version.FirmaRefs) > 0 && strings.TrimSpace(version.ValidacionFirmaRef) != "")) &&
		(!a.RequiereSelloTiempo || len(version.SelloTiempoRefs) > 0) &&
		(!a.RequiereRegistro || strings.TrimSpace(version.RegistroRef) != "")
}

func (c CodigoCotejo) Retirar(actor, retiradaRef, motivo string, fecha time.Time) (CodigoCotejo, error) {
	return c.finalizar(actor, retiradaRef, motivo, "", EstadoCodigoCotejoRetirado, fecha)
}

func (c CodigoCotejo) Sustituir(actor, retiradaRef, motivo, sustituidoPorRef string, fecha time.Time) (CodigoCotejo, error) {
	if !referenciaDocumentalValida(sustituidoPorRef) || sustituidoPorRef == c.Referencia() {
		return CodigoCotejo{}, ErrTransicionCodigoCotejo
	}
	return c.finalizar(actor, retiradaRef, motivo, strings.TrimSpace(sustituidoPorRef), EstadoCodigoCotejoSustituido, fecha)
}

func (c CodigoCotejo) finalizar(actor, retiradaRef, motivo, sustituidoPorRef string, estado EstadoCodigoCotejo, fecha time.Time) (CodigoCotejo, error) {
	if err := c.Validar(); err != nil || c.Estado != EstadoCodigoCotejoActivo ||
		!referenciaDocumentalValida(actor) || !referenciaDocumentalValida(retiradaRef) ||
		!textoDocumentalNoVacioValido(motivo) || fecha.IsZero() || fecha.Before(c.ActivadoEn) ||
		(estado != EstadoCodigoCotejoRetirado && estado != EstadoCodigoCotejoSustituido) {
		return CodigoCotejo{}, ErrTransicionCodigoCotejo
	}
	finalizado := c
	finalizado.Revision++
	finalizado.Estado = estado
	finalizado.RetiradoPor = strings.TrimSpace(actor)
	finalizado.RetiradoEn = fecha.UTC()
	finalizado.RetiradaRef = strings.TrimSpace(retiradaRef)
	finalizado.MotivoRetirada = strings.TrimSpace(motivo)
	finalizado.SustituidoPorRef = sustituidoPorRef
	return finalizado.ClonarCanonico()
}

func (c CodigoCotejo) DisponibleEn(instante time.Time) bool {
	instante = instante.UTC()
	return c.Estado == EstadoCodigoCotejoActivo && !instante.Before(c.DisponibleDesde) && instante.Before(c.DisponibleHasta)
}

// NormalizarValorCodigoCotejo elimina separadores de lectura. No admite otros
// caracteres para evitar distintas representaciones del mismo secreto.
func NormalizarValorCodigoCotejo(valor string) (string, error) {
	// Solo el espacio ASCII y el guion son separadores admitidos. TrimSpace
	// aceptaria saltos de linea y otros controles, creando representaciones
	// ambiguas en cabeceras, logs o protocolos de transporte.
	valor = strings.ToUpper(valor)
	var canonico strings.Builder
	canonico.Grow(len(valor))
	for _, caracter := range valor {
		switch caracter {
		case '-', ' ':
			continue
		}
		if !strings.ContainsRune("23456789ABCDEFGHJKLMNPQRSTUVWXYZ", caracter) {
			return "", ErrCodigoCotejoInvalido
		}
		canonico.WriteRune(caracter)
	}
	if canonico.Len() < 26 || canonico.Len() > 96 {
		return "", ErrCodigoCotejoInvalido
	}
	return canonico.String(), nil
}

func canonizarClavesCotejo(valores []string) ([]string, error) {
	if len(valores) == 0 || len(valores) > maximoValoresPoliticaCotejo {
		return nil, ErrPoliticaCotejoInvalida
	}
	canonicas := append([]string(nil), valores...)
	for indice := range canonicas {
		canonicas[indice] = strings.TrimSpace(canonicas[indice])
		if !esClaveDocumentalCanonica(canonicas[indice]) {
			return nil, ErrPoliticaCotejoInvalida
		}
	}
	sort.Strings(canonicas)
	for indice := 1; indice < len(canonicas); indice++ {
		if canonicas[indice] == canonicas[indice-1] {
			return nil, ErrPoliticaCotejoInvalida
		}
	}
	return canonicas, nil
}

func canonizarCamposPublicosCotejo(campos []CampoPublicoCotejo) ([]CampoPublicoCotejo, error) {
	if len(campos) > len([]CampoPublicoCotejo{
		CampoPublicoCotejoOrgano,
		CampoPublicoCotejoTipoDocumental,
		CampoPublicoCotejoFechaEmision,
		CampoPublicoCotejoHuellaSHA256,
	}) {
		return nil, ErrPoliticaCotejoInvalida
	}
	canonicos := append([]CampoPublicoCotejo(nil), campos...)
	for _, campo := range canonicos {
		if !campo.Valido() {
			return nil, ErrPoliticaCotejoInvalida
		}
	}
	sort.Slice(canonicos, func(i, j int) bool { return canonicos[i] < canonicos[j] })
	for indice := 1; indice < len(canonicos); indice++ {
		if canonicos[indice] == canonicos[indice-1] {
			return nil, ErrPoliticaCotejoInvalida
		}
	}
	return canonicos, nil
}

func canonizarReferenciasCotejo(referencias []string) ([]string, error) {
	if len(referencias) > maximoValoresPoliticaCotejo {
		return nil, ErrVersionCotejoInvalida
	}
	canonicas := append([]string(nil), referencias...)
	for indice := range canonicas {
		canonicas[indice] = strings.TrimSpace(canonicas[indice])
		if !referenciaDocumentalValida(canonicas[indice]) {
			return nil, ErrVersionCotejoInvalida
		}
	}
	sort.Strings(canonicas)
	for indice := 1; indice < len(canonicas); indice++ {
		if canonicas[indice] == canonicas[indice-1] {
			return nil, ErrVersionCotejoInvalida
		}
	}
	return canonicas, nil
}

func contieneCadenaCotejo(valores []string, buscado string) bool {
	buscado = strings.TrimSpace(buscado)
	for _, valor := range valores {
		if strings.TrimSpace(valor) == buscado {
			return true
		}
	}
	return false
}

func (c CodigoCotejo) TieneCampoPublico(campo CampoPublicoCotejo) bool {
	for _, permitido := range c.Politica.CamposPublicos {
		if permitido == campo {
			return true
		}
	}
	return false
}
