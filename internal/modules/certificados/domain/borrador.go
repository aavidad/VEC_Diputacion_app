// Package domain describes preparation for review, without administrative issuance.
package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"vec-diputacion-granada/internal/shared/i18n"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrEntrada      = errors.New("certificados.entrada_invalida")
	ErrCatalogo     = errors.New("certificados.catalogo_invalido")
	ErrNoDisponible = errors.New("certificados.no_disponible")
)

const MaxServicios = 200

const (
	EsquemaFuenteEnsayo           = "vec.certificados.fuente-servicios.ensayo.v1"
	EsquemaFuentePersonalV1Ensayo = "vec.certificados.fuente-servicios.personal-v1.ensayo"
	CertezaAcreditado             = "acreditado"
	CoberturaCompleta             = "completa"
)

var (
	certezas         = map[string]bool{CertezaAcreditado: true, "pendiente": true, "no_acreditado": true}
	coberturas       = map[string]bool{CoberturaCompleta: true, "parcial": true, "no_acreditada": true}
	referenciaFuente = regexp.MustCompile(`^[A-Za-z0-9:._-]{1,256}$`)
)

var referencia = regexp.MustCompile(`^ensayo:[a-z0-9][a-z0-9._-]{0,99}$`)

type Corte struct {
	VigenteEn  string `json:"vigente_en"`
	ConocidoEn string `json:"conocido_en"`
}

// Servicio es un periodo de la fuente. En la muestra con forma del contrato
// V1 de Personal, Fin vacío es un periodo abierto, Dias no llega (V1 no los
// trae) y Certeza, ServicioRef, ActoRef y ClaseVersion conservan su procedencia.
type Servicio struct {
	Inicio       string `json:"inicio"`
	Fin          string `json:"fin"`
	Clase        string `json:"clase"`
	ClaseVersion int64  `json:"clase_version,omitempty"`
	// EnCursoAlCorte: el fin previsto es posterior a la fecha de referencia;
	// Fin se recorta a esa fecha y el borrador lo dice.
	EnCursoAlCorte bool   `json:"en_curso_al_corte,omitempty"`
	Dias           *int64 `json:"dias,omitempty"`
	Estado         string `json:"estado"`
	Certeza        string `json:"certeza,omitempty"`
	ServicioRef    string `json:"servicio_ref,omitempty"`
	ActoRef        string `json:"acto_ref,omitempty"`
}

// SustentaCertificacion: solo un servicio reconocido y con procedencia
// acreditada puede sustentar un certificado (contrato V1 de Personal).
func (s Servicio) SustentaCertificacion() bool {
	return s.Estado == "reconocido" && s.Certeza == CertezaAcreditado
}

// FuenteServicios is a snapshot supplied by its owner. The present consumer
// accepts only explicit rehearsal sources; it grants no permission for real data.
type FuenteServicios struct {
	Esquema        string `json:"esquema"`
	Sintetica      bool   `json:"sintetica"`
	ProcedenciaRef string `json:"procedencia_ref"`
	Nombre         string `json:"nombre"`
	Corte          Corte  `json:"corte"`
	// Cobertura solo llega con la forma V1: completa, parcial o no_acreditada.
	Cobertura string     `json:"cobertura,omitempty"`
	Servicios []Servicio `json:"servicios"`
}

// ConFormaPersonalV1 indica si la fuente procede del traductor del contrato V1.
func (f FuenteServicios) ConFormaPersonalV1() bool { return f.Esquema == EsquemaFuentePersonalV1Ensayo }

func IdiomaValido(s string) bool {
	return regexp.MustCompile(`^[a-z]{2,3}(-[A-Za-z0-9]{2,8})*$`).MatchString(s)
}

func FechaValida(s string) bool {
	t, e := time.Parse("2006-01-02", s)
	return e == nil && len(s) == 10 && t.Format("2006-01-02") == s
}
func TextoValido(s string, max int) bool {
	if strings.TrimSpace(s) == "" || !utf8.ValidString(s) || utf8.RuneCountInString(s) > max {
		return false
	}
	for _, r := range s {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return false
		}
	}
	return true
}
func (f FuenteServicios) ValidarEnsayo() error {
	v1 := f.ConFormaPersonalV1()
	if (f.Esquema != EsquemaFuenteEnsayo && !v1) || !f.Sintetica || v1 != coberturas[f.Cobertura] || (!v1 && f.Cobertura != "") ||
		!referencia.MatchString(f.ProcedenciaRef) || !TextoValido(f.Nombre, 120) ||
		!FechaValida(f.Corte.VigenteEn) || f.Servicios == nil || len(f.Servicios) > MaxServicios {
		return ErrEntrada
	}
	_, e := time.Parse(time.RFC3339Nano, f.Corte.ConocidoEn)
	if e != nil || !strings.HasSuffix(f.Corte.ConocidoEn, "Z") {
		return ErrEntrada
	}
	for _, s := range f.Servicios {
		abierto := v1 && s.Fin == ""
		if !FechaValida(s.Inicio) || (!abierto && (!FechaValida(s.Fin) || s.Fin < s.Inicio || s.Fin > f.Corte.VigenteEn)) ||
			s.Inicio > f.Corte.VigenteEn || !TextoValido(s.Clase, 120) ||
			(s.Estado != "declarado" && s.Estado != "comprobado" && s.Estado != "reconocido") {
			return ErrEntrada
		}
		if s.EnCursoAlCorte && (!v1 || s.Fin != f.Corte.VigenteEn) {
			return ErrEntrada
		}
		if v1 {
			// V1 no trae días: no se inventan ni se calculan.
			if s.Dias != nil || !certezas[s.Certeza] || s.ClaseVersion < 1 || !referenciaFuente.MatchString(s.ServicioRef) || !referenciaFuente.MatchString(s.ActoRef) {
				return ErrEntrada
			}
		} else if s.Dias == nil || *s.Dias < 0 || *s.Dias > 200000 || s.Certeza != "" || s.ClaseVersion != 0 || s.ServicioRef != "" || s.ActoRef != "" {
			return ErrEntrada
		}
	}
	return nil
}

// Plantilla defines ordered message keys, never executable expressions. Its
// version is for rehearsal review; it is not a published administrative template.
type Plantilla struct {
	Esquema string   `json:"esquema"`
	ID      string   `json:"id"`
	Version int      `json:"version"`
	Estado  string   `json:"estado"`
	Bloques []string `json:"bloques"`
}

func (p Plantilla) Validar(id string, version int) error {
	if p.Esquema != "vec.certificados.plantilla-ensayo.v1" || p.ID != "servicios" || p.ID != id ||
		p.Version != version || version < 1 || p.Estado != "ensayo" ||
		len(p.Bloques) != 6 {
		return ErrCatalogo
	}
	obligatorios := map[string]bool{"limite": false, "persona": false, "corte": false, "fuente": false, "plantilla": false, "criterio": false}
	for _, clave := range p.Bloques {
		visto, existe := obligatorios[clave]
		if !existe || visto {
			return ErrCatalogo
		}
		obligatorios[clave] = true
	}
	return nil
}

type Textos struct {
	Esquema         string            `json:"esquema"`
	Idioma          string            `json:"idioma"`
	FormatoFecha    string            `json:"formato_fecha"`
	FormatoInstante string            `json:"formato_instante"`
	Mensajes        map[string]string `json:"mensajes"`
}

var claves = []string{"titulo", "limite", "persona", "corte", "fuente", "plantilla", "criterio",
	"declarado", "comprobado", "reconocido", "servicio", "vacio", "revision", "sin_servicios",
	"cli_indice_idiomas", "cli_uso", "cli_fuente", "cli_plantilla", "cli_version", "cli_idioma", "cli_textos", "cli_salida", "cli_ensayo", "cli_ok",
	"error_argumentos", "error_entrada", "error_catalogo", "error_salida", "error_no_disponible",
	"servicio_v1", "servicio_v1_abierto", "servicio_v1_al_corte", "criterio_v1", "certeza_acreditado", "certeza_pendiente", "certeza_no_acreditado",
	"sustenta_si", "sustenta_no", "cobertura_completa", "cobertura_parcial", "cobertura_no_acreditada"}

func (t Textos) Validar() error {
	if t.Esquema != "vec.certificados.textos.v1" || !IdiomaValido(t.Idioma) || len(t.Mensajes) != len(claves) || !TextoValido(t.FormatoFecha, 80) || !TextoValido(t.FormatoInstante, 80) {
		return ErrCatalogo
	}
	for _, k := range claves {
		if !TextoValido(t.Mensajes[k], 1200) {
			return ErrCatalogo
		}
	}
	return nil
}

// Mensaje performs literal replacement. Inserted values are never re-evaluated.
func (t Textos) Mensaje(clave string, valores ...string) string {
	if len(valores)%2 != 0 {
		return ""
	}
	reemplazos := make([]string, 0, len(valores))
	for i := 0; i < len(valores); i += 2 {
		reemplazos = append(reemplazos, "{{"+valores[i]+"}}", valores[i+1])
	}
	catalogo, e := i18n.New(t.Idioma, map[string]map[string]string{t.Idioma: t.Mensajes})
	if e != nil {
		return ""
	}
	return strings.NewReplacer(reemplazos...).Replace(catalogo.T(t.Idioma, clave))
}
func Huella(v any) (string, error) {
	b, e := json.Marshal(v)
	if e != nil {
		return "", ErrEntrada
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}

type VersionPlantilla struct {
	ID           string `json:"id"`
	Version      int    `json:"version"`
	Estado       string `json:"estado"`
	HuellaSHA256 string `json:"huella_sha256"`
}
type Grupo struct {
	Estado    string     `json:"estado"`
	Servicios []Servicio `json:"servicios"`
}
type Borrador struct {
	Esquema            string                       `json:"esquema"`
	Estado             string                       `json:"estado"`
	Modo               string                       `json:"modo"`
	Idioma             string                       `json:"idioma"`
	Plantilla          VersionPlantilla             `json:"plantilla"`
	TextosHuellaSHA256 string                       `json:"textos_huella_sha256"`
	FuenteHuellaSHA256 string                       `json:"fuente_huella_sha256"`
	Fuente             FuenteServicios              `json:"fuente"`
	Grupos             []Grupo                      `json:"grupos"`
	Contenido          vecdomain.ContenidoDocumento `json:"contenido"`
}
