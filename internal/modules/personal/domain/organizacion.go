package domain

import (
	"errors"
	"strings"
	"time"

	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const (
	prefijoReferenciaOrganizacion = "org_"
	minimoSufijoOrganizacion      = 16
	maximoSufijoOrganizacion      = 80
)

var ErrReferenciaOrganizacionInvalida error = errors.New("personal: referencia de organizacion invalida")

var (
	ErrCambioOrganizacionInvalido      = errors.New("personal: cambio de organizacion invalido")
	ErrRevisionOrganizacionEnConflicto = errors.New("personal: revision de organizacion en conflicto")
)

type CambioUnidadOrganizativa struct {
	Clave, Etiqueta, Tipo, AdscripcionClave string
}

const idEstructuraOrganizativa = "estructura-organizativa-dipgra"

func ValidarEstructuraOrganizativa(c vecdomain.CatalogoConfigurable) error {
	if len(c.Entradas) > 1000 {
		return ErrCambioOrganizacionInvalido
	}
	claves := make(map[string]struct{}, len(c.Entradas))
	padres := make(map[string]string, len(c.Entradas))
	for _, e := range c.Entradas {
		t := e.Atributos["tipo"]
		if (t != "delegacion" && t != "centro" && t != "puesto_responsabilidad") || strings.TrimSpace(e.Clave) != e.Clave || e.Clave == "" || strings.TrimSpace(e.Etiqueta) != e.Etiqueta || e.Etiqueta == "" {
			return ErrCambioOrganizacionInvalido
		}
		if _, ok := claves[e.Clave]; ok {
			return ErrCambioOrganizacionInvalido
		}
		claves[e.Clave] = struct{}{}
		if e.Atributos["adscripcion_clave"] == e.Clave {
			return ErrCambioOrganizacionInvalido
		}
		if p := e.Atributos["adscripcion_clave"]; p != "" {
			padres[e.Clave] = p
		}
	}
	for clave, padre := range padres {
		if _, ok := claves[padre]; !ok {
			return ErrCambioOrganizacionInvalido
		}
		vistos := map[string]bool{}
		for actual := clave; actual != ""; actual = padres[actual] {
			if vistos[actual] {
				return ErrCambioOrganizacionInvalido
			}
			vistos[actual] = true
		}
	}
	return nil
}

func PrepararCambioEstructuraOrganizativa(actual vecdomain.CatalogoConfigurable, revisionEsperada int, huellaEsperada, actorID, motivo string, cambio CambioUnidadOrganizativa, ahora time.Time) (vecdomain.CatalogoConfigurable, error) {
	if actual.Estado != vecdomain.EstadoCatalogoBorrador || actual.ID != idEstructuraOrganizativa || actual.Version < 1 {
		return vecdomain.CatalogoConfigurable{}, ErrCambioOrganizacionInvalido
	}
	if revisionEsperada != actual.Revision {
		return vecdomain.CatalogoConfigurable{}, ErrRevisionOrganizacionEnConflicto
	}
	h, err := actual.HuellaSHA256()
	if err != nil || h != huellaEsperada {
		return vecdomain.CatalogoConfigurable{}, ErrRevisionOrganizacionEnConflicto
	}
	if strings.TrimSpace(cambio.Etiqueta) != cambio.Etiqueta || cambio.Etiqueta == "" || cambio.Tipo == "" {
		return vecdomain.CatalogoConfigurable{}, ErrCambioOrganizacionInvalido
	}
	entradas, err := actual.ClonarCanonico()
	if err != nil {
		return vecdomain.CatalogoConfigurable{}, ErrCambioOrganizacionInvalido
	}
	idx := -1
	for i := range entradas.Entradas {
		if entradas.Entradas[i].Clave == cambio.Clave {
			idx = i
			break
		}
	}
	if idx < 0 && !claveLocalOrganizacionValida(cambio.Clave) {
		return vecdomain.CatalogoConfigurable{}, ErrCambioOrganizacionInvalido
	}
	if idx < 0 {
		entradas.Entradas = append(entradas.Entradas, vecdomain.EntradaCatalogoConfigurable{Clave: cambio.Clave, Etiqueta: cambio.Etiqueta, Orden: len(entradas.Entradas), VigenteDesde: ahora.UTC(), Atributos: map[string]string{"tipo": cambio.Tipo, "modificada_localmente": "si", "adscripcion_clave": cambio.AdscripcionClave}})
	} else {
		e := entradas.Entradas[idx]
		e.Etiqueta, e.Atributos["tipo"], e.Atributos["modificada_localmente"] = cambio.Etiqueta, cambio.Tipo, "si"
		if cambio.AdscripcionClave != "" {
			e.Atributos["adscripcion_clave"] = cambio.AdscripcionClave
		} else {
			delete(e.Atributos, "adscripcion_clave")
		}
		entradas.Entradas[idx] = e
	}
	if err := ValidarEstructuraOrganizativa(entradas); err != nil {
		return vecdomain.CatalogoConfigurable{}, err
	}
	resultado, err := actual.ActualizarBorrador(revisionEsperada, actorID, actual.Nombre, actual.Descripcion, actual.FuenteRef, motivo, entradas.Entradas, ahora)
	if err != nil {
		return vecdomain.CatalogoConfigurable{}, ErrCambioOrganizacionInvalido
	}
	return resultado, nil
}

func claveLocalOrganizacionValida(v string) bool {
	if len(v) != len("local-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx") || v[:6] != "local-" {
		return false
	}
	for i, c := range v[6:] {
		if c == '-' {
			if i != 8 && i != 13 && i != 18 && i != 23 {
				return false
			}
			continue
		}
		if !(c >= '0' && c <= '9') && !(c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

// ReferenciaOrganizacion conserva el identificador nominal canonico de una
// organizacion. Su forma no acredita existencia, actividad, vigencia,
// procedencia ni autorizacion.
type ReferenciaOrganizacion struct {
	referencia string
}

// NuevaReferenciaOrganizacion admite exclusivamente la representacion
// canonica recibida, sin normalizarla ni completarla.
func NuevaReferenciaOrganizacion(referencia string) (ReferenciaOrganizacion, error) {
	valor := ReferenciaOrganizacion{referencia: referencia}
	if err := valor.Validar(); err != nil {
		return ReferenciaOrganizacion{}, err
	}
	return valor, nil
}

// Validar reacredita la forma canonica del valor nominal.
func (r ReferenciaOrganizacion) Validar() error {
	if !referenciaOrganizacionValida(r.referencia) {
		return ErrReferenciaOrganizacionInvalida
	}
	return nil
}

// Referencia devuelve los bytes exactos recibidos al construir un valor
// valido.
func (r ReferenciaOrganizacion) Referencia() (string, error) {
	if err := r.Validar(); err != nil {
		return "", err
	}
	return r.referencia, nil
}

func referenciaOrganizacionValida(referencia string) bool {
	longitudSufijo := len(referencia) - len(prefijoReferenciaOrganizacion)
	if longitudSufijo < minimoSufijoOrganizacion || longitudSufijo > maximoSufijoOrganizacion {
		return false
	}
	if referencia[:len(prefijoReferenciaOrganizacion)] != prefijoReferenciaOrganizacion {
		return false
	}
	for indice := len(prefijoReferenciaOrganizacion); indice < len(referencia); indice++ {
		caracter := referencia[indice]
		if (caracter < 'a' || caracter > 'z') && (caracter < '0' || caracter > '9') {
			return false
		}
	}
	return true
}
