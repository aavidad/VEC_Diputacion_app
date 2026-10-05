// Package temas valida material local; no instala ni autoriza operaciones ADMIN.
package temas

import (
	"encoding/json"
	"errors"
	"math"
	"regexp"
	"strings"
	"unicode"
)

const MaxBytes = 64 * 1024
const MaxProfundidad = 8
const MaxTokens = 64
const MaxIdiomas = 32
const MaxNombre = 100
const MaxPares = 128

// MaxVersion conserva el rango de un entero de 32 bits en contratos futuros.
const MaxVersion = 2_147_483_647

var ErrPaquete = errors.New("temas.paquetes.error.paquete")
var ErrPolitica = errors.New("temas.paquetes.error.politica")
var ErrContraste = errors.New("temas.paquetes.error.contraste")

var identificador = regexp.MustCompile(`^[a-z][a-z0-9]*(?:[_.-][a-z0-9]+)*$`)
var token = regexp.MustCompile(`^--portal-[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`)
var clave = regexp.MustCompile(`^ui\.temas\.[a-z][a-z0-9_.-]*$`)
var color = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
var idioma = regexp.MustCompile(`^[a-z]{2,3}(?:-[a-z0-9]{2,8})*$`)

type Paquete struct {
	Esquema          int                          `json:"esquema"`
	TemaID           string                       `json:"tema_id"`
	Version          int                          `json:"version"`
	SistemaDisenoRef string                       `json:"sistema_diseno_ref"`
	PoliticaRef      string                       `json:"politica_ref"`
	NombreKey        string                       `json:"nombre_key"`
	Textos           map[string]map[string]string `json:"textos"`
	Variantes        Variantes                    `json:"variantes"`
}

type Variantes struct {
	Clara  map[string]string `json:"clara"`
	Oscura map[string]string `json:"oscura"`
}

type Limites struct {
	Bytes   int `json:"bytes"`
	Tokens  int `json:"tokens"`
	Idiomas int `json:"idiomas"`
	Nombre  int `json:"nombre"`
}

type ParContraste struct {
	PrimerPlano string  `json:"primer_plano"`
	Fondo       string  `json:"fondo"`
	Tipo        string  `json:"tipo"`
	Minimo      float64 `json:"minimo"`
}

// Politica es una entrada de plataforma verificada por huella antes de leer el
// paquete. Sus listas nunca proceden del propio paquete.
type Politica struct {
	Esquema           int               `json:"esquema"`
	PoliticaRef       string            `json:"politica_ref"`
	SistemaDisenoRef  string            `json:"sistema_diseno_ref"`
	Tokens            []string          `json:"tokens"`
	IdiomasRequeridos []string          `json:"idiomas_requeridos"`
	ValoresFijos      map[string]string `json:"valores_fijos"`
	Contrastes        []ParContraste    `json:"contrastes"`
	Limites           Limites           `json:"limites"`
}

func refValida(s string) bool {
	return len(s) > 0 && len(s) <= 96 && identificador.MatchString(s)
}

func (p Politica) Validar() error {
	if p.Esquema != 1 || !refValida(p.PoliticaRef) || !refValida(p.SistemaDisenoRef) ||
		p.Limites.Bytes < 1 || p.Limites.Bytes > MaxBytes || p.Limites.Tokens < 1 || p.Limites.Tokens > MaxTokens ||
		p.Limites.Idiomas < 1 || p.Limites.Idiomas > MaxIdiomas || p.Limites.Nombre < 1 || p.Limites.Nombre > MaxNombre ||
		len(p.Tokens) == 0 || len(p.Tokens) > p.Limites.Tokens || len(p.IdiomasRequeridos) == 0 || len(p.IdiomasRequeridos) > p.Limites.Idiomas ||
		len(p.Contrastes) == 0 || len(p.Contrastes) > MaxPares {
		return ErrPolitica
	}
	permitidos := map[string]bool{}
	for _, t := range p.Tokens {
		if len(t) > 64 || !token.MatchString(t) || permitidos[t] {
			return ErrPolitica
		}
		permitidos[t] = true
	}
	idiomas := map[string]bool{}
	for _, i := range p.IdiomasRequeridos {
		if len(i) > 16 || !idioma.MatchString(i) || idiomas[i] {
			return ErrPolitica
		}
		idiomas[i] = true
	}
	if !permitidos["--portal-fondo-logo"] || !color.MatchString(p.ValoresFijos["--portal-fondo-logo"]) {
		return ErrPolitica
	}
	for t, v := range p.ValoresFijos {
		if !permitidos[t] || !color.MatchString(v) {
			return ErrPolitica
		}
	}
	pares := map[string]bool{}
	texto, componente := false, false
	for _, par := range p.Contrastes {
		minimo := 0.0
		switch par.Tipo {
		case "texto":
			minimo = 4.5
			texto = true
		case "componente":
			minimo = 3
			componente = true
		default:
			return ErrPolitica
		}
		id := par.PrimerPlano + "/" + par.Fondo
		if !permitidos[par.PrimerPlano] || !permitidos[par.Fondo] || par.PrimerPlano == par.Fondo || pares[id] || math.IsNaN(par.Minimo) || math.IsInf(par.Minimo, 0) || !(par.Minimo >= minimo && par.Minimo <= 21) {
			return ErrPolitica
		}
		pares[id] = true
	}
	if !texto || !componente {
		return ErrPolitica
	}
	return nil
}

func nombreValido(s string, max int) bool {
	if s == "" || len([]rune(s)) > max || strings.TrimSpace(s) != s || strings.ContainsAny(s, "<>{}\\") || strings.Contains(s, "://") {
		return false
	}
	for _, r := range s {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) || r == '\u2028' || r == '\u2029' || r == '\ufffd' {
			return false
		}
	}
	return true
}

// Normalizar copia y convierte únicamente colores opacos. No interpreta CSS.
func (p Paquete) Normalizar(politica Politica) (Paquete, error) {
	if politica.Validar() != nil {
		return Paquete{}, ErrPolitica
	}
	if p.Esquema != 1 || !refValida(p.TemaID) || p.Version < 1 || p.Version > MaxVersion || p.SistemaDisenoRef != politica.SistemaDisenoRef || p.PoliticaRef != politica.PoliticaRef ||
		len(p.NombreKey) > 128 || !clave.MatchString(p.NombreKey) || p.NombreKey != "ui.temas."+p.TemaID+".nombre" || len(p.Textos) != len(politica.IdiomasRequeridos) {
		return Paquete{}, ErrPaquete
	}
	normal := p
	normal.Textos = make(map[string]map[string]string, len(p.Textos))
	for _, i := range politica.IdiomasRequeridos {
		nombres := p.Textos[i]
		if len(nombres) != 1 || !nombreValido(nombres[p.NombreKey], politica.Limites.Nombre) {
			return Paquete{}, ErrPaquete
		}
		normal.Textos[i] = map[string]string{p.NombreKey: nombres[p.NombreKey]}
	}
	variantes := []*map[string]string{&normal.Variantes.Clara, &normal.Variantes.Oscura}
	for _, variante := range variantes {
		original := *variante
		if len(original) != len(politica.Tokens) {
			return Paquete{}, ErrPaquete
		}
		colores := make(map[string]string, len(original))
		for _, t := range politica.Tokens {
			v := original[t]
			if !color.MatchString(v) {
				return Paquete{}, ErrPaquete
			}
			colores[t] = strings.ToLower(v)
			if fijo, existe := politica.ValoresFijos[t]; existe && colores[t] != strings.ToLower(fijo) {
				return Paquete{}, ErrPaquete
			}
		}
		for _, par := range politica.Contrastes {
			if Contraste(colores[par.PrimerPlano], colores[par.Fondo]) < par.Minimo {
				return Paquete{}, ErrContraste
			}
		}
		*variante = colores
	}
	return normal, nil
}

// Canonico tiene orden de propiedades fijo y mapas ordenados por encoding/json.
func (p Paquete) Canonico() ([]byte, error) { return json.Marshal(p) }
