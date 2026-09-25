package domain

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// Correo personalizado del llamamiento de Bolsa (petición de RRHH, p. 3): cada
// persona recibe el texto de la plantilla con sus propios datos. Qué marcadores
// existen, cómo se llaman, los textos por defecto, el formato de fecha y el
// límite de tamaño proceden de un catálogo versionado y modificable. Solo el
// conjunto de fuentes de datos es cerrado: el servidor no puede sustituir un
// dato que no conoce.

var (
	ErrCatalogoCorreoLlamamientoInvalido = errors.New("bolsa: catalogo de correo de llamamiento invalido")
	ErrPlantillaCorreoLlamamiento        = errors.New("bolsa: plantilla de correo de llamamiento invalida")
	ErrDatosCorreoLlamamientoIncompletos = errors.New("bolsa: faltan datos para personalizar el correo de llamamiento")
	ErrCorreoLlamamientoExcedeLimite     = errors.New("bolsa: el correo personalizado supera el limite")
)

const EsquemaCatalogoCorreoLlamamiento = "vec.bolsa.catalogo_correo_llamamiento.v1"

// FuenteMarcadorCorreo es el dato del que se nutre un marcador. Lista cerrada.
type FuenteMarcadorCorreo string

const (
	FuenteCorreoNombre         FuenteMarcadorCorreo = "nombre"
	FuenteCorreoApellidos      FuenteMarcadorCorreo = "apellidos"
	FuenteCorreoNombreCompleto FuenteMarcadorCorreo = "nombre_completo"
	FuenteCorreoPosicion       FuenteMarcadorCorreo = "posicion"
	FuenteCorreoBolsa          FuenteMarcadorCorreo = "bolsa"
	FuenteCorreoCategoria      FuenteMarcadorCorreo = "categoria"
	FuenteCorreoReferencia     FuenteMarcadorCorreo = "referencia"
	FuenteCorreoCentro         FuenteMarcadorCorreo = "centro"
	FuenteCorreoModalidad      FuenteMarcadorCorreo = "modalidad"
	FuenteCorreoFechaInicio    FuenteMarcadorCorreo = "fecha_inicio"
	FuenteCorreoPlazo          FuenteMarcadorCorreo = "plazo"
	FuenteCorreoFechaEnvio     FuenteMarcadorCorreo = "fecha_envio"
)

// personal indica si la fuente exige consultar los datos de la persona o de su
// bolsa; las demás salen de la configuración del propio llamamiento.
func (f FuenteMarcadorCorreo) personal() bool {
	switch f {
	case FuenteCorreoNombre, FuenteCorreoApellidos, FuenteCorreoNombreCompleto, FuenteCorreoPosicion, FuenteCorreoBolsa:
		return true
	}
	return false
}

func (f FuenteMarcadorCorreo) valida() bool {
	switch f {
	case FuenteCorreoNombre, FuenteCorreoApellidos, FuenteCorreoNombreCompleto, FuenteCorreoPosicion, FuenteCorreoBolsa,
		FuenteCorreoCategoria, FuenteCorreoReferencia, FuenteCorreoCentro, FuenteCorreoModalidad, FuenteCorreoFechaInicio,
		FuenteCorreoPlazo, FuenteCorreoFechaEnvio:
		return true
	}
	return false
}

type MarcadorCorreoLlamamiento struct {
	Clave  string               `json:"clave"`
	Fuente FuenteMarcadorCorreo `json:"fuente"`
}

type TextosCorreoLlamamiento struct {
	Asunto string `json:"asunto"`
	Cuerpo string `json:"cuerpo"`
}

// CatalogoCorreoLlamamiento es inmutable tras construirse con
// CatalogoCorreoLlamamientoDesdeJSON; las lecturas devuelven copias.
type CatalogoCorreoLlamamiento struct {
	version          int
	plantillaVigente string
	plantillas       map[string]bool
	limiteCuerpo     int
	limiteAsunto     int
	formatoFecha     string
	zona             *time.Location
	idioma           string
	textos           map[string]TextosCorreoLlamamiento
	marcadores       []MarcadorCorreoLlamamiento
	porClave         map[string]FuenteMarcadorCorreo
}

type catalogoCorreoJSON struct {
	Esquema          string `json:"esquema"`
	Version          int    `json:"version"`
	PlantillaVigente string `json:"plantilla_vigente"`
	Plantillas       []struct {
		Version       string `json:"version"`
		Personalizada bool   `json:"personalizada"`
	} `json:"plantillas"`
	LimiteCaracteres       int                                `json:"limite_caracteres"`
	LimiteCaracteresAsunto int                                `json:"limite_caracteres_asunto"`
	FormatoFecha           string                             `json:"formato_fecha"`
	ZonaHoraria            string                             `json:"zona_horaria"`
	IdiomaDefecto          string                             `json:"idioma_defecto"`
	TextosDefecto          map[string]TextosCorreoLlamamiento `json:"textos_defecto"`
	Marcadores             []MarcadorCorreoLlamamiento        `json:"marcadores"`
}

var (
	patronVersionPlantillaCorreo = regexp.MustCompile(`^bolsa-llamamiento-v[1-9][0-9]{0,3}$`)
	patronClaveMarcadorCorreo    = regexp.MustCompile(`^[a-z][a-z0-9_]{0,39}$`)
	patronMarcadorEnTexto        = regexp.MustCompile(`\{([a-z][a-z0-9_]{0,39})\}`)
	patronIdiomaCorreo           = regexp.MustCompile(`^[a-z]{2}(-[A-Z]{2})?$`)
)

const (
	// El texto de la plantilla viaja en la configuración y PostgreSQL limita
	// cada campo a 4000 octetos (migración 000017); el límite del correo ya
	// sustituido es el del catálogo.
	maximoOctetosPlantillaCorreo = 4000
	limiteAsuntoPorDefecto       = 250
)

// CatalogoCorreoLlamamientoDesdeJSON valida por completo el catálogo. Un
// catálogo defectuoso deja la emisión sin componer (fail-closed).
func CatalogoCorreoLlamamientoDesdeJSON(datos []byte) (CatalogoCorreoLlamamiento, error) {
	var crudo catalogoCorreoJSON
	d := json.NewDecoder(bytes.NewReader(datos))
	d.DisallowUnknownFields()
	if len(datos) == 0 || d.Decode(&crudo) != nil || d.Decode(&struct{}{}) != io.EOF {
		return CatalogoCorreoLlamamiento{}, ErrCatalogoCorreoLlamamientoInvalido
	}
	if crudo.Esquema != EsquemaCatalogoCorreoLlamamiento || crudo.Version < 1 || len(crudo.Plantillas) < 1 || len(crudo.Plantillas) > 16 ||
		crudo.LimiteCaracteres < 200 || crudo.LimiteCaracteres > 20000 ||
		(crudo.FormatoFecha != "iso" && crudo.FormatoFecha != "dd/mm/aaaa") || !patronIdiomaCorreo.MatchString(crudo.IdiomaDefecto) ||
		len(crudo.Marcadores) < 1 || len(crudo.Marcadores) > 32 || len(crudo.TextosDefecto) < 1 || len(crudo.TextosDefecto) > 16 {
		return CatalogoCorreoLlamamiento{}, ErrCatalogoCorreoLlamamientoInvalido
	}
	limiteAsunto := crudo.LimiteCaracteresAsunto
	if limiteAsunto == 0 {
		limiteAsunto = limiteAsuntoPorDefecto
	}
	if limiteAsunto < 20 || limiteAsunto > 998 {
		return CatalogoCorreoLlamamiento{}, ErrCatalogoCorreoLlamamientoInvalido
	}
	zona, err := time.LoadLocation(crudo.ZonaHoraria)
	if strings.TrimSpace(crudo.ZonaHoraria) == "" || err != nil {
		return CatalogoCorreoLlamamiento{}, ErrCatalogoCorreoLlamamientoInvalido
	}
	c := CatalogoCorreoLlamamiento{
		version: crudo.Version, plantillaVigente: crudo.PlantillaVigente, plantillas: map[string]bool{},
		limiteCuerpo: crudo.LimiteCaracteres, limiteAsunto: limiteAsunto, formatoFecha: crudo.FormatoFecha, zona: zona,
		idioma: crudo.IdiomaDefecto, textos: map[string]TextosCorreoLlamamiento{}, porClave: map[string]FuenteMarcadorCorreo{},
	}
	for _, p := range crudo.Plantillas {
		if _, repetida := c.plantillas[p.Version]; repetida || !patronVersionPlantillaCorreo.MatchString(p.Version) {
			return CatalogoCorreoLlamamiento{}, ErrCatalogoCorreoLlamamientoInvalido
		}
		c.plantillas[p.Version] = p.Personalizada
	}
	if _, ok := c.plantillas[c.plantillaVigente]; !ok {
		return CatalogoCorreoLlamamiento{}, ErrCatalogoCorreoLlamamientoInvalido
	}
	for _, m := range crudo.Marcadores {
		if _, repetido := c.porClave[m.Clave]; repetido || !patronClaveMarcadorCorreo.MatchString(m.Clave) || !m.Fuente.valida() {
			return CatalogoCorreoLlamamiento{}, ErrCatalogoCorreoLlamamientoInvalido
		}
		c.porClave[m.Clave] = m.Fuente
		c.marcadores = append(c.marcadores, m)
	}
	for idioma, t := range crudo.TextosDefecto {
		if !patronIdiomaCorreo.MatchString(idioma) || !textoPlantillaAdmisible(t.Asunto) || !textoPlantillaAdmisible(t.Cuerpo) ||
			c.ValidarPlantilla(c.plantillaVigente, t.Asunto) != nil || c.ValidarPlantilla(c.plantillaVigente, t.Cuerpo) != nil {
			return CatalogoCorreoLlamamiento{}, ErrCatalogoCorreoLlamamientoInvalido
		}
		c.textos[idioma] = t
	}
	if _, ok := c.textos[c.idioma]; !ok {
		return CatalogoCorreoLlamamiento{}, ErrCatalogoCorreoLlamamientoInvalido
	}
	return c, nil
}

func textoPlantillaAdmisible(v string) bool {
	return utf8.ValidString(v) && strings.TrimSpace(v) == v && len(v) >= 2 && len(v) <= maximoOctetosPlantillaCorreo
}

func (c CatalogoCorreoLlamamiento) Valido() bool {
	return c.version > 0 && len(c.plantillas) > 0 && c.zona != nil && len(c.porClave) > 0
}

func (c CatalogoCorreoLlamamiento) Version() int                { return c.version }
func (c CatalogoCorreoLlamamiento) PlantillaVigente() string    { return c.plantillaVigente }
func (c CatalogoCorreoLlamamiento) LimiteCaracteres() int       { return c.limiteCuerpo }
func (c CatalogoCorreoLlamamiento) LimiteCaracteresAsunto() int { return c.limiteAsunto }
func (c CatalogoCorreoLlamamiento) IdiomaDefecto() string       { return c.idioma }
func (c CatalogoCorreoLlamamiento) Marcadores() []MarcadorCorreoLlamamiento {
	return append([]MarcadorCorreoLlamamiento(nil), c.marcadores...)
}

// TextosDefecto devuelve los textos del idioma pedido o, si no existen, los
// del idioma por defecto del catálogo.
func (c CatalogoCorreoLlamamiento) TextosDefecto(idioma string) TextosCorreoLlamamiento {
	if t, ok := c.textos[idioma]; ok {
		return t
	}
	return c.textos[c.idioma]
}

// PlantillaAdmitida indica si la versión existe y si interpreta marcadores.
func (c CatalogoCorreoLlamamiento) PlantillaAdmitida(version string) (personalizada, admitida bool) {
	personalizada, admitida = c.plantillas[version]
	return personalizada, admitida
}

// ValidarPlantilla rechaza versiones desconocidas y marcadores fuera del
// catálogo. Una plantilla literal (no personalizada) admite cualquier texto.
func (c CatalogoCorreoLlamamiento) ValidarPlantilla(version, texto string) error {
	personalizada, admitida := c.plantillas[version]
	if !admitida {
		return ErrPlantillaCorreoLlamamiento
	}
	if !personalizada {
		return nil
	}
	for _, m := range patronMarcadorEnTexto.FindAllStringSubmatch(texto, -1) {
		if _, ok := c.porClave[m[1]]; !ok {
			return ErrPlantillaCorreoLlamamiento
		}
	}
	return nil
}

// RequiereDatosPersonales indica si algún texto usa un marcador cuya fuente
// exige consultar a la persona o a su bolsa.
func (c CatalogoCorreoLlamamiento) RequiereDatosPersonales(version string, textos ...string) bool {
	if personalizada := c.plantillas[version]; !personalizada {
		return false
	}
	for _, texto := range textos {
		for _, m := range patronMarcadorEnTexto.FindAllStringSubmatch(texto, -1) {
			if c.porClave[m[1]].personal() {
				return true
			}
		}
	}
	return false
}

// ValoresCorreoLlamamiento reúne lo que puede sustituirse para una persona.
type ValoresCorreoLlamamiento struct {
	Nombre, Apellidos, Bolsa                                     string
	Posicion                                                     int
	Categoria, Referencia, Centro, Modalidad, FechaInicio, Plazo string
	FechaEnvio                                                   time.Time
}

// CorreoPersonalizado es el mensaje exacto que recibe una persona.
type CorreoPersonalizado struct {
	Asunto, Cuerpo string
}

// Personalizar sustituye los marcadores de asunto y cuerpo. Una versión
// literal devuelve los textos intactos; en ambos casos se aplica el límite.
func (c CatalogoCorreoLlamamiento) Personalizar(version, asunto, cuerpo string, v ValoresCorreoLlamamiento) (CorreoPersonalizado, error) {
	personalizada, admitida := c.plantillas[version]
	if !admitida || !c.Valido() {
		return CorreoPersonalizado{}, ErrPlantillaCorreoLlamamiento
	}
	out := CorreoPersonalizado{Asunto: asunto, Cuerpo: cuerpo}
	if personalizada {
		var err error
		if out.Asunto, err = c.sustituir(asunto, v); err != nil {
			return CorreoPersonalizado{}, err
		}
		if out.Cuerpo, err = c.sustituir(cuerpo, v); err != nil {
			return CorreoPersonalizado{}, err
		}
	}
	if strings.ContainsAny(out.Asunto, "\r\n") || utf8.RuneCountInString(out.Asunto) > c.limiteAsunto || utf8.RuneCountInString(out.Cuerpo) > c.limiteCuerpo {
		return CorreoPersonalizado{}, ErrCorreoLlamamientoExcedeLimite
	}
	return out, nil
}

func (c CatalogoCorreoLlamamiento) sustituir(texto string, v ValoresCorreoLlamamiento) (string, error) {
	var fallo error
	out := patronMarcadorEnTexto.ReplaceAllStringFunc(texto, func(marcador string) string {
		fuente, ok := c.porClave[marcador[1:len(marcador)-1]]
		if !ok {
			fallo = ErrPlantillaCorreoLlamamiento
			return ""
		}
		valor := limpiarValorCorreo(c.valor(fuente, v))
		if valor == "" && fallo == nil {
			fallo = ErrDatosCorreoLlamamientoIncompletos
		}
		return valor
	})
	if fallo != nil {
		return "", fallo
	}
	return out, nil
}

func (c CatalogoCorreoLlamamiento) valor(f FuenteMarcadorCorreo, v ValoresCorreoLlamamiento) string {
	switch f {
	case FuenteCorreoNombre:
		return v.Nombre
	case FuenteCorreoApellidos:
		return v.Apellidos
	case FuenteCorreoNombreCompleto:
		return strings.Join(strings.Fields(v.Nombre+" "+v.Apellidos), " ")
	case FuenteCorreoPosicion:
		if v.Posicion > 0 {
			return strconv.Itoa(v.Posicion)
		}
	case FuenteCorreoBolsa:
		return v.Bolsa
	case FuenteCorreoCategoria:
		return v.Categoria
	case FuenteCorreoReferencia:
		return v.Referencia
	case FuenteCorreoCentro:
		return v.Centro
	case FuenteCorreoModalidad:
		return v.Modalidad
	case FuenteCorreoFechaInicio:
		if f, err := time.Parse("2006-01-02", v.FechaInicio); err == nil {
			return c.formatearFecha(f)
		}
		return v.FechaInicio
	case FuenteCorreoPlazo:
		return v.Plazo
	case FuenteCorreoFechaEnvio:
		if !v.FechaEnvio.IsZero() {
			return c.formatearFecha(v.FechaEnvio.In(c.zona))
		}
	}
	return ""
}

func (c CatalogoCorreoLlamamiento) formatearFecha(f time.Time) string {
	if c.formatoFecha == "dd/mm/aaaa" {
		return f.Format("02/01/2006")
	}
	return f.Format("2006-01-02")
}

// limpiarValorCorreo impide que un dato sustituido inyecte saltos de línea o
// controles (cabeceras del asunto, estructura del cuerpo).
func limpiarValorCorreo(v string) string {
	return strings.Join(strings.FieldsFunc(v, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }), " ")
}
