// Package modulos proyecta una decisión operativa del catálogo central. Esta
// decisión sólo restringe acceso: nunca concede permisos ni modifica módulos.
package modulos

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	administracion "vec-diputacion-granada/internal/modules/administracion"
	vec "vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrConfiguracion = errors.New("administracion: configuracion de modulos invalida")
	ErrCatalogo      = errors.New("administracion: catalogo de modulos no disponible")
	ErrDesactivado   = errors.New("administracion: modulo desactivado")
	ErrConflicto     = errors.New("administracion: decision de modulos cambiada")
	ErrCambio        = errors.New("administracion: preparacion de modulos invalida")
)

type Configuracion struct {
	CatalogoID string
	Gobernados []string
}

func (c Configuracion) Validar() error {
	if c.CatalogoID == "" || len(c.CatalogoID) > 128 || strings.TrimSpace(c.CatalogoID) != c.CatalogoID || len(c.Gobernados) == 0 || len(c.Gobernados) > 128 {
		return ErrConfiguracion
	}
	vistos := map[string]bool{}
	for _, id := range c.Gobernados {
		if id == "" || id == administracion.ModuleID || len(id) > 128 || strings.TrimSpace(id) != id || vistos[id] {
			return ErrConfiguracion
		}
		vistos[id] = true
	}
	return nil
}

type Modulo struct {
	ID         string `json:"modulo_id"`
	NombreKey  string `json:"nombre_key"`
	Habilitado bool   `json:"habilitado"`
	Gobernado  bool   `json:"gobernado"`
}

type Estado struct {
	Version             int      `json:"version"`
	HuellaSHA256        string   `json:"huella_sha256"`
	Modulos             []Modulo `json:"modulos"`
	EscrituraDisponible bool     `json:"escritura_disponible"`
}

// Proyectar valida la publicación completa antes de ofrecerla al menú o a una
// operación. ADMIN y módulos no gobernados conservan su autorización propia.
func Proyectar(cfg Configuracion, catalogo vec.CatalogoConfigurable, registrados []vec.ModuleManifest, ahora time.Time) (Estado, error) {
	if cfg.Validar() != nil || ahora.IsZero() || len(registrados) == 0 || len(registrados) > 128 {
		return Estado{}, ErrConfiguracion
	}
	if catalogo.ID != cfg.CatalogoID || catalogo.ModuloID != administracion.ModuleID || catalogo.Estado != vec.EstadoCatalogoPublicado || catalogo.PublicadoEn.After(ahora) || catalogo.Validar() != nil {
		return Estado{}, ErrCatalogo
	}
	registradosPorID := map[string]bool{}
	for _, m := range registrados {
		if m.Validate() != nil || registradosPorID[m.ID] {
			return Estado{}, ErrConfiguracion
		}
		registradosPorID[m.ID] = true
	}
	gobernados := map[string]bool{}
	for _, id := range cfg.Gobernados {
		if !registradosPorID[id] {
			return Estado{}, ErrConfiguracion
		}
		gobernados[id] = true
	}
	activos := map[string]bool{}
	for _, e := range catalogo.Entradas {
		if !gobernados[e.Clave] || len(e.Atributos) != 1 || (e.Atributos["habilitado"] != "true" && e.Atributos["habilitado"] != "false") {
			return Estado{}, ErrCatalogo
		}
		// Una entrada fuera de vigencia permanece cerrada; no reaparece una
		// publicación anterior para reemplazar esta decisión.
		activos[e.Clave] = e.VigenteEn(ahora) && e.Atributos["habilitado"] == "true"
	}
	huella, err := catalogo.HuellaSHA256()
	if err != nil {
		return Estado{}, ErrCatalogo
	}
	estado := Estado{Version: catalogo.Version, HuellaSHA256: huella, Modulos: make([]Modulo, 0, len(registrados))}
	for _, m := range registrados {
		estado.Modulos = append(estado.Modulos, Modulo{ID: m.ID, NombreKey: m.NameKey, Habilitado: !gobernados[m.ID] || activos[m.ID], Gobernado: gobernados[m.ID]})
	}
	return estado, nil
}

func (e Estado) ExigirHabilitado(id string) error {
	for _, m := range e.Modulos {
		if m.ID == id && m.Habilitado {
			return nil
		}
	}
	return ErrDesactivado
}

type Cambio struct {
	ModuloID        string `json:"modulo_id"`
	Habilitado      bool   `json:"habilitado"`
	VersionEsperada int    `json:"version_esperada"`
	HuellaEsperada  string `json:"huella_esperada"`
	Motivo          string `json:"motivo"`
}

type Preparacion struct {
	Cambio             Cambio                            `json:"cambio"`
	CatalogoReferencia string                            `json:"catalogo_referencia"`
	VersionSiguiente   int                               `json:"version_siguiente"`
	Entradas           []vec.EntradaCatalogoConfigurable `json:"entradas"`
	Publicado          bool                              `json:"publicado"`
}

// Preparar devuelve entradas para una NUEVA versión del gobierno central.
// No crea, publica ni escribe un catálogo, y no es un recibo de activación.
func Preparar(cfg Configuracion, catalogo vec.CatalogoConfigurable, registrados []vec.ModuleManifest, ahora time.Time, cambio Cambio) (Preparacion, error) {
	estado, err := Proyectar(cfg, catalogo, registrados, ahora)
	if err != nil {
		return Preparacion{}, err
	}
	if cambio.VersionEsperada != estado.Version || cambio.HuellaEsperada != estado.HuellaSHA256 {
		return Preparacion{}, ErrConflicto
	}
	if cambio.Motivo == "" || strings.TrimSpace(cambio.Motivo) != cambio.Motivo || len(cambio.Motivo) > 2048 || !textoValido(cambio.Motivo) || catalogo.Version >= 1_000_000 {
		return Preparacion{}, ErrCambio
	}
	gobernado := false
	for _, m := range estado.Modulos {
		if m.ID == cambio.ModuloID {
			gobernado = m.Gobernado
		}
	}
	if !gobernado {
		return Preparacion{}, ErrCambio
	}
	canonico, err := catalogo.ClonarCanonico()
	if err != nil {
		return Preparacion{}, err
	}
	encontrado := false
	for i, e := range canonico.Entradas {
		if e.Clave == cambio.ModuloID {
			// Una entrada caducada necesita una revisión de vigencia por el gobierno
			// de catálogos, no un cambio que afirme activar sin tener efecto.
			if !e.VigenteEn(ahora) {
				return Preparacion{}, ErrCambio
			}
			valor := "false"
			if cambio.Habilitado {
				valor = "true"
			}
			if e.Atributos["habilitado"] == valor {
				return Preparacion{}, ErrCambio
			}
			canonico.Entradas[i].Atributos["habilitado"] = valor
			encontrado = true
		}
	}
	if !encontrado {
		return Preparacion{}, ErrCambio
	}
	return Preparacion{Cambio: cambio, CatalogoReferencia: catalogo.Referencia(), VersionSiguiente: catalogo.Version + 1, Entradas: canonico.Entradas}, nil
}

func textoValido(s string) bool {
	if !utf8.ValidString(s) {
		return false
	}
	for _, r := range s {
		if (r < 0x20 && r != '\n' && r != '\r' && r != '\t') || r == 0x7f {
			return false
		}
	}
	return true
}
