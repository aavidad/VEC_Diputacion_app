package domain

import (
	"errors"
	"strings"
)

var ErrPreferenciasInvalidas = errors.New("usuarios preferencias invalidas")
var ErrCatalogoPreferenciasInvalido = errors.New("usuarios catalogo de preferencias invalido")

type ValoresPreferencias struct {
	Idioma            string `json:"idioma"`
	TamanoTexto       string `json:"tamano_texto"`
	AltoContraste     bool   `json:"alto_contraste"`
	Tema              string `json:"tema"`
	Inicio            string `json:"inicio"`
	Filas             int    `json:"filas"`
	AvisoCorreoTareas bool   `json:"aviso_correo_tareas"`
	AvisoCorreoPlazos bool   `json:"aviso_correo_plazos"`
}

type OpcionPreferencia struct {
	Codigo    string `json:"codigo"`
	NombreKey string `json:"nombre_key"`
}

// El catálogo puede reducir opciones; nunca ampliar el vocabulario cerrado.
type CatalogoPreferencias struct {
	VersionRef      string              `json:"version_ref"`
	Idiomas         []OpcionPreferencia `json:"idiomas"`
	TamanosTexto    []OpcionPreferencia `json:"tamanos_texto"`
	Temas           []OpcionPreferencia `json:"temas"`
	Inicios         []OpcionPreferencia `json:"inicios"`
	Filas           []int               `json:"filas"`
	Predeterminados ValoresPreferencias `json:"predeterminados"`
}

var idiomas = map[string]bool{"navegador": true, "es": true, "en": true}
var tamanos_texto = map[string]bool{"normal": true, "grande": true, "muy_grande": true}
var temas = map[string]bool{
	"sistema": true, "claro": true, "oscuro": true,
	"diputacion_granada": true, "arena": true, "salvia": true,
	"lavanda": true, "azul_sereno": true, "noche_suave": true,
}
var inicios = map[string]bool{"cuadro": true, "peticiones": true, "bolsas": true}
var filas = map[int]bool{20: true, 50: true, 100: true}

func (v ValoresPreferencias) ValidarCodigos() error {
	if !idiomas[v.Idioma] || !tamanos_texto[v.TamanoTexto] || !temas[v.Tema] || !inicios[v.Inicio] || !filas[v.Filas] {
		return ErrPreferenciasInvalidas
	}
	return nil
}

func opcionesValidas(opciones []OpcionPreferencia, permitidas map[string]bool) bool {
	if len(opciones) == 0 || len(opciones) > len(permitidas) {
		return false
	}
	vistas := make(map[string]bool, len(opciones))
	for _, opcion := range opciones {
		if !permitidas[opcion.Codigo] || vistas[opcion.Codigo] || !claveI18nValida(opcion.NombreKey) {
			return false
		}
		vistas[opcion.Codigo] = true
	}
	return true
}

func claveI18nValida(clave string) bool {
	if !strings.HasPrefix(clave, "ui.") || len(clave) > 128 {
		return false
	}
	for _, r := range clave {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '.' && r != '_' {
			return false
		}
	}
	return true
}

func versionValida(ref string) bool {
	if len(ref) < 1 || len(ref) > 96 {
		return false
	}
	for _, r := range ref {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '-' && r != '_' && r != ':' && r != '.' {
			return false
		}
	}
	return true
}

func contiene(opciones []OpcionPreferencia, codigo string) bool {
	for _, opcion := range opciones {
		if opcion.Codigo == codigo {
			return true
		}
	}
	return false
}

func (c CatalogoPreferencias) Validar() error {
	if !versionValida(c.VersionRef) || !opcionesValidas(c.Idiomas, idiomas) || !opcionesValidas(c.TamanosTexto, tamanos_texto) || !opcionesValidas(c.Temas, temas) || !opcionesValidas(c.Inicios, inicios) || len(c.Filas) == 0 || len(c.Filas) > len(filas) || c.Predeterminados.ValidarCodigos() != nil {
		return ErrCatalogoPreferenciasInvalido
	}
	vistas := map[int]bool{}
	for _, n := range c.Filas {
		if !filas[n] || vistas[n] {
			return ErrCatalogoPreferenciasInvalido
		}
		vistas[n] = true
	}
	if !contiene(c.Idiomas, c.Predeterminados.Idioma) || !contiene(c.TamanosTexto, c.Predeterminados.TamanoTexto) || !contiene(c.Temas, c.Predeterminados.Tema) || !contiene(c.Inicios, c.Predeterminados.Inicio) || !vistas[c.Predeterminados.Filas] {
		return ErrCatalogoPreferenciasInvalido
	}
	return nil
}

func (c CatalogoPreferencias) ValidarValores(v ValoresPreferencias) error {
	if c.Validar() != nil || v.ValidarCodigos() != nil || !contiene(c.Idiomas, v.Idioma) || !contiene(c.TamanosTexto, v.TamanoTexto) || !contiene(c.Temas, v.Tema) || !contiene(c.Inicios, v.Inicio) {
		return ErrPreferenciasInvalidas
	}
	for _, n := range c.Filas {
		if n == v.Filas {
			return nil
		}
	}
	return ErrPreferenciasInvalidas
}

func (c CatalogoPreferencias) Clonar() CatalogoPreferencias {
	c.Idiomas = append([]OpcionPreferencia(nil), c.Idiomas...)
	c.TamanosTexto = append([]OpcionPreferencia(nil), c.TamanosTexto...)
	c.Temas = append([]OpcionPreferencia(nil), c.Temas...)
	c.Inicios = append([]OpcionPreferencia(nil), c.Inicios...)
	c.Filas = append([]int(nil), c.Filas...)
	return c
}

func CatalogoBasePreferencias() CatalogoPreferencias {
	return CatalogoPreferencias{
		VersionRef:      "usuarios-preferencias-v1",
		Idiomas:         []OpcionPreferencia{{"navegador", "ui.usuarios.preferencias.idioma.navegador"}, {"es", "ui.usuarios.preferencias.idioma.es"}, {"en", "ui.usuarios.preferencias.idioma.en"}},
		TamanosTexto:    []OpcionPreferencia{{"normal", "ui.usuarios.preferencias.tamano_texto.normal"}, {"grande", "ui.usuarios.preferencias.tamano_texto.grande"}, {"muy_grande", "ui.usuarios.preferencias.tamano_texto.muy_grande"}},
		Temas:           []OpcionPreferencia{{"sistema", "ui.usuarios.preferencias.tema.sistema"}, {"claro", "ui.usuarios.preferencias.tema.claro"}, {"oscuro", "ui.usuarios.preferencias.tema.oscuro"}},
		Inicios:         []OpcionPreferencia{{"cuadro", "ui.usuarios.preferencias.inicio.cuadro"}, {"peticiones", "ui.usuarios.preferencias.inicio.peticiones"}, {"bolsas", "ui.usuarios.preferencias.inicio.bolsas"}},
		Filas:           []int{20, 50, 100},
		Predeterminados: ValoresPreferencias{Idioma: "navegador", TamanoTexto: "normal", Tema: "sistema", Inicio: "cuadro", Filas: 20},
	}
}
