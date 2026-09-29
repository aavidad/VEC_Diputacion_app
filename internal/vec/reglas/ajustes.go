package reglas

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"slices"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

// Ajustes de reglas: valores que RRHH fija desde la aplicación sobre el
// catálogo base (docs/estudio_requisitos/plazos_configurables_2026-09-30.md).
//
// El catálogo base no cambia. Los ajustes forman su propio catálogo
// versionado, <catálogo base>.ajustes, de solo adición: cada versión lleva
// todos los ajustes vigentes, quién la publicó, cuándo y por qué. El
// resolutor aplica sobre cada regla base el ajuste vigente. Solo se ajustan
// los campos que la regla base declara en su atributo «editable», con las
// opciones y límites que ella misma fija; nada de eso está en el código.

// SufijoCatalogoAjustes forma el identificador del catálogo de ajustes.
const SufijoCatalogoAjustes = ".ajustes"

// Atributos de la regla base que declaran qué se puede ajustar.
const (
	AtributoEditable        = "editable"
	AtributoOpcionesUnidad  = "opciones_unidad"
	AtributoOpcionesComputo = "opciones_computo"
	AtributoCantidadMinima  = "cantidad_minima"
	AtributoCantidadMaxima  = "cantidad_maxima"
)

// Campos ajustables. Los estructurales (fases, inicio, listas y catálogos
// enlazados) no se ajustan: cambian qué permisos se necesitan.
const (
	CampoCantidad        = "cantidad"
	CampoCantidadUrgente = AtributoCantidadUrgente
	CampoUnidad          = "unidad"
	CampoComputo         = "computo"
)

const (
	maximoReglasAjustadas = 64
	maximoBytesAjustes    = 16 * 1024
)

var (
	// ErrAjustesNoDisponibles: la versión de ajustes no se pudo leer o no es
	// coherente. Nunca se sustituye por los valores base.
	ErrAjustesNoDisponibles = errors.New("reglas: ajustes de reglas no disponibles")
	// ErrAjusteInvalido: el ajuste toca un campo no editable, sale de las
	// opciones o límites de la regla o produce una regla no válida.
	ErrAjusteInvalido = errors.New("reglas: ajuste de regla no valido")
)

// Edicion es lo que la regla base admite ajustar. Nil si no admite nada.
type Edicion struct {
	Campos          []string
	OpcionesUnidad  []Unidad
	OpcionesComputo []Computo
	CantidadMinima  int
	CantidadMaxima  int
}

// Admite indica si el campo es ajustable.
func (e *Edicion) Admite(campo string) bool {
	return e != nil && slices.Contains(e.Campos, campo)
}

// VersionAjustes es una versión publicada del catálogo de ajustes. Ajustes
// lleva, por clave de regla, los campos ajustados y su valor canónico. Quién
// la publicó y por qué quedan en el historial del almacén, no aquí: no viajan
// con las reglas a sus consumidores.
type VersionAjustes struct {
	CatalogoID   string
	Version      int
	HuellaSHA256 string
	VigenteDesde time.Time
	Ajustes      map[string]map[string]string
}

// ConsultaAjustes es el puerto hacia el almacén de ajustes. Devuelve la versión
// vigente en el instante (la de mayor número con VigenteDesde no posterior) y
// encontrada=false si todavía no hay ninguna.
type ConsultaAjustes interface {
	AjustesVigentesEn(ctx context.Context, catalogoAjustesID string, instante time.Time) (version VersionAjustes, encontrada bool, err error)
}

// Ajuste describe el ajuste aplicado a una regla.
type Ajuste struct {
	Version      int
	VigenteDesde time.Time
	// Campos son los valores ajustados de esta regla.
	Campos map[string]string
	// BaseReferencia es la entrada del catálogo base sobre la que se aplica.
	BaseReferencia domain.ReferenciaEntradaCatalogo
	// HuellaAjustes es la huella de la versión de ajustes.
	HuellaAjustes string
}

// CatalogoAjustesDe devuelve el identificador del catálogo de ajustes de un
// catálogo base.
func CatalogoAjustesDe(catalogoBaseID string) string { return catalogoBaseID + SufijoCatalogoAjustes }

// HuellaAjustes calcula la huella SHA-256 de la forma canónica de unos
// ajustes (JSON con claves ordenadas, sin espacios).
func HuellaAjustes(ajustes map[string]map[string]string) (string, error) {
	canonico, err := CanonicoAjustes(ajustes)
	if err != nil {
		return "", err
	}
	suma := sha256.Sum256(canonico)
	return hex.EncodeToString(suma[:]), nil
}

// CanonicoAjustes devuelve la forma canónica de unos ajustes. encoding/json
// ordena las claves de los mapas, así que la salida es determinista.
func CanonicoAjustes(ajustes map[string]map[string]string) ([]byte, error) {
	if ajustes == nil {
		ajustes = map[string]map[string]string{}
	}
	if len(ajustes) > maximoReglasAjustadas {
		return nil, ErrAjusteInvalido
	}
	for clave, campos := range ajustes {
		if !claveAjusteCanonica(clave) || len(campos) == 0 || len(campos) > 4 {
			return nil, ErrAjusteInvalido
		}
		for campo, valor := range campos {
			if !campoAjustable(campo) || !valorAjusteCanonico(valor) {
				return nil, ErrAjusteInvalido
			}
		}
	}
	canonico, err := json.Marshal(ajustes)
	if err != nil || len(canonico) > maximoBytesAjustes {
		return nil, ErrAjusteInvalido
	}
	return canonico, nil
}

// claveAjusteCanonica coincide con el máximo de CT148. La clave también debe
// pertenecer al alfabeto de las entradas del catálogo base.
func claveAjusteCanonica(clave string) bool {
	if len(clave) < 2 || len(clave) > 80 || !claveCanonica(clave) {
		return false
	}
	for i := 0; i < len(clave); i++ {
		c := clave[i]
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '.' && c != '_' && c != '-' {
			return false
		}
	}
	return true
}

// valorAjusteCanonico comparte el alfabeto acotado de la persistencia: las
// cantidades, unidades y cómputos son claves, nunca texto libre.
func valorAjusteCanonico(valor string) bool {
	if valor == "" || len(valor) > 64 {
		return false
	}
	for i := 0; i < len(valor); i++ {
		c := valor[i]
		if c < 'a' || c > 'z' {
			if c < '0' || c > '9' {
				if c != '_' {
					return false
				}
			}
		}
	}
	return true
}

func campoAjustable(campo string) bool {
	return campo == CampoCantidad || campo == CampoCantidadUrgente || campo == CampoUnidad || campo == CampoComputo
}

// validarVersionAjustes comprueba forma y huella de la versión leída.
func validarVersionAjustes(v VersionAjustes, catalogoAjustesID string, instante time.Time) error {
	if v.CatalogoID != catalogoAjustesID || v.Version < 1 || v.VigenteDesde.IsZero() || v.VigenteDesde.After(instante) {
		return ErrAjustesNoDisponibles
	}
	huella, err := HuellaAjustes(v.Ajustes)
	if err != nil || huella != v.HuellaSHA256 {
		return ErrAjustesNoDisponibles
	}
	return nil
}

// edicionDesdeAtributos lee la declaración de campos ajustables de una regla
// ya validada. Una declaración incoherente invalida la regla.
func edicionDesdeAtributos(regla Regla, a map[string]string) (*Edicion, error) {
	texto, declarada := a[AtributoEditable]
	_, conUnidades := a[AtributoOpcionesUnidad]
	_, conComputos := a[AtributoOpcionesComputo]
	_, conMinima := a[AtributoCantidadMinima]
	_, conMaxima := a[AtributoCantidadMaxima]
	if !declarada {
		if conUnidades || conComputos || conMinima || conMaxima {
			return nil, ErrReglaInvalida
		}
		return nil, nil
	}
	// Una regla del Reglamento no se ajusta desde la aplicación: su valor lo
	// fija una norma publicada.
	if regla.Origen == OrigenReglamento || regla.ParteEjemplo != "" {
		return nil, ErrReglaInvalida
	}
	e := &Edicion{}
	for _, campo := range strings.Split(texto, ",") {
		if !campoAjustable(campo) || slices.Contains(e.Campos, campo) {
			return nil, ErrReglaInvalida
		}
		e.Campos = append(e.Campos, campo)
	}
	conCantidad := e.Admite(CampoCantidad) || e.Admite(CampoCantidadUrgente)
	if conCantidad != (conMinima && conMaxima) || conMinima != conMaxima ||
		e.Admite(CampoUnidad) != conUnidades || e.Admite(CampoComputo) != conComputos {
		return nil, ErrReglaInvalida
	}
	if conCantidad {
		minima, errMin := enteroCanonico(a[AtributoCantidadMinima])
		maxima, errMax := enteroCanonico(a[AtributoCantidadMaxima])
		if errMin != nil || errMax != nil || minima > maxima || !regla.Unidad.conCantidad() {
			return nil, ErrReglaInvalida
		}
		e.CantidadMinima, e.CantidadMaxima = minima, maxima
	}
	if conUnidades {
		for _, u := range strings.Split(a[AtributoOpcionesUnidad], ",") {
			unidad := Unidad(u)
			if unidad.EsPlazo() != regla.Unidad.EsPlazo() || !unidad.conCantidad() || slices.Contains(e.OpcionesUnidad, unidad) {
				return nil, ErrReglaInvalida
			}
			e.OpcionesUnidad = append(e.OpcionesUnidad, unidad)
		}
	}
	if conComputos {
		for _, c := range strings.Split(a[AtributoOpcionesComputo], ",") {
			computo := Computo(c)
			if (computo != ComputoAdministrativo && computo != ComputoCivil) || slices.Contains(e.OpcionesComputo, computo) {
				return nil, ErrReglaInvalida
			}
			e.OpcionesComputo = append(e.OpcionesComputo, computo)
		}
	}
	if err := e.validarValores(regla); err != nil {
		return nil, ErrReglaInvalida
	}
	return e, nil
}

// validarValores comprueba que los valores de la regla estén dentro de sus
// propias opciones y límites.
func (e *Edicion) validarValores(regla Regla) error {
	if e.Admite(CampoCantidad) && (regla.Cantidad < e.CantidadMinima || regla.Cantidad > e.CantidadMaxima) {
		return ErrAjusteInvalido
	}
	if texto, ok := regla.Atributos[AtributoCantidadUrgente]; ok && e.Admite(CampoCantidadUrgente) {
		urgente, err := enteroCanonico(texto)
		if err != nil || urgente < e.CantidadMinima || urgente > e.CantidadMaxima || urgente > regla.Cantidad {
			return ErrAjusteInvalido
		}
	}
	if e.Admite(CampoUnidad) && !slices.Contains(e.OpcionesUnidad, regla.Unidad) {
		return ErrAjusteInvalido
	}
	if e.Admite(CampoComputo) && !slices.Contains(e.OpcionesComputo, regla.Computo) {
		return ErrAjusteInvalido
	}
	return nil
}

func enteroCanonico(texto string) (int, error) {
	valor, err := strconv.Atoi(texto)
	if err != nil || valor < 1 || valor > maximoCantidadRegla || strconv.Itoa(valor) != texto {
		return 0, ErrReglaInvalida
	}
	return valor, nil
}

// aplicarAjuste devuelve la regla con el ajuste aplicado, validada con las
// mismas comprobaciones que la base y dentro de sus opciones y límites.
func aplicarAjuste(
	catalogo domain.CatalogoConfigurable, huellaBase string, ejemplo bool,
	entrada domain.EntradaCatalogoConfigurable, base Regla, v VersionAjustes, campos map[string]string,
) (Regla, error) {
	if base.Edicion == nil || len(campos) == 0 {
		return Regla{}, ErrAjusteInvalido
	}
	atributos := make(map[string]string, len(entrada.Atributos))
	for clave, valor := range entrada.Atributos {
		atributos[clave] = valor
	}
	for campo, valor := range campos {
		if !base.Edicion.Admite(campo) {
			return Regla{}, ErrAjusteInvalido
		}
		atributos[campo] = valor
	}
	entrada.Atributos = atributos
	regla, err := reglaDesdeEntrada(catalogo, huellaBase, ejemplo, entrada)
	if err != nil {
		return Regla{}, ErrAjusteInvalido
	}
	// La edición es la de la base: un ajuste no amplía sus propias opciones.
	regla.Edicion = base.Edicion
	if err := regla.Edicion.validarValores(regla); err != nil {
		return Regla{}, ErrAjusteInvalido
	}
	copiaCampos := make(map[string]string, len(campos))
	for campo, valor := range campos {
		copiaCampos[campo] = valor
	}
	regla.Ajuste = &Ajuste{
		Version: v.Version, VigenteDesde: v.VigenteDesde.UTC(), Campos: copiaCampos, BaseReferencia: base.ReferenciaEntrada, HuellaAjustes: v.HuellaSHA256,
	}
	// La regla ajustada cita el ajuste. Su huella combina la de la base y la
	// de los ajustes: identifica exactamente el valor que la gobernó.
	regla.ReferenciaEntrada = domain.ReferenciaEntradaCatalogo{
		CatalogoID: v.CatalogoID, CatalogoVersion: v.Version,
		CatalogoHuellaSHA256: huellaEfectiva(huellaBase, v.HuellaSHA256), EntradaClave: entrada.Clave,
	}
	if regla.ReferenciaEntrada.Validar() != nil {
		return Regla{}, ErrAjusteInvalido
	}
	regla.Referencia = regla.ReferenciaEntrada.Referencia()
	regla.HuellaCatalogo = regla.ReferenciaEntrada.CatalogoHuellaSHA256
	return regla, nil
}

func huellaEfectiva(huellaBase, huellaAjustes string) string {
	suma := sha256.Sum256([]byte("vec.reglas.ajuste.v1\x1f" + huellaBase + "\x1f" + huellaAjustes))
	return hex.EncodeToString(suma[:])
}
