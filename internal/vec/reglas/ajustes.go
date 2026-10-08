package reglas

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"maps"
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
	// ErrAjustesConflicto: otra transacción conserva el bloqueo o cambió la
	// versión. El llamador puede distinguirlo de una caída del almacén.
	ErrAjustesConflicto = errors.New("reglas: conflicto de ajustes de reglas")
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

// DatosInstantaneaRegla contiene la base y el valor resuelto con la versión
// completa de ajustes que se leyó. Incluso una regla sin cambio conserva la
// versión y huella de esa lectura. Sin versión publicada, Encontrada es false,
// VersionAjustes es 0 y la huella corresponde al canónico vacío ({}).
type DatosInstantaneaRegla struct {
	Base                Regla
	Efectiva            Regla
	CatalogoAjustesID   string
	AjustesEncontrados  bool
	VersionAjustes      int
	HuellaAjustes       string
	CanonicoAjustes     []byte
	AjustesVigenteDesde time.Time
	PreparadaEn         time.Time
}

// InstantaneaRegla no expone alias mutables. Es una preparación de lectura:
// todavía falta guardarla junto al inicio del plazo en una transacción real.
type InstantaneaRegla struct{ datos DatosInstantaneaRegla }

// Datos devuelve copias independientes de mapas, listas y bytes.
func (i InstantaneaRegla) Datos() (DatosInstantaneaRegla, error) {
	if !i.valida() {
		return DatosInstantaneaRegla{}, ErrReglasNoDisponibles
	}
	d := i.datos
	d.Base = copiarRegla(d.Base)
	d.Efectiva = copiarRegla(d.Efectiva)
	d.CanonicoAjustes = slices.Clone(d.CanonicoAjustes)
	return d, nil
}

func (i InstantaneaRegla) valida() bool {
	d := i.datos
	if d.PreparadaEn.IsZero() || d.Base.ReferenciaEntrada.Validar() != nil ||
		d.Efectiva.ReferenciaEntrada.Validar() != nil || d.Base.Clave != d.Efectiva.Clave ||
		d.CatalogoAjustesID != CatalogoAjustesDe(d.Base.ReferenciaEntrada.CatalogoID) ||
		len(d.CanonicoAjustes) == 0 || !claveCanonica(d.CatalogoAjustesID) {
		return false
	}
	suma := sha256.Sum256(d.CanonicoAjustes)
	if hex.EncodeToString(suma[:]) != d.HuellaAjustes {
		return false
	}
	if !d.AjustesEncontrados {
		return d.VersionAjustes == 0 && d.AjustesVigenteDesde.IsZero() && string(d.CanonicoAjustes) == "{}"
	}
	return d.VersionAjustes > 0 && !d.AjustesVigenteDesde.IsZero() &&
		!d.AjustesVigenteDesde.After(d.PreparadaEn)
}

func copiarRegla(r Regla) Regla {
	copia := r
	copia.Atributos = maps.Clone(r.Atributos)
	if r.Edicion != nil {
		edicion := *r.Edicion
		edicion.Campos = slices.Clone(r.Edicion.Campos)
		edicion.OpcionesUnidad = slices.Clone(r.Edicion.OpcionesUnidad)
		edicion.OpcionesComputo = slices.Clone(r.Edicion.OpcionesComputo)
		copia.Edicion = &edicion
	}
	if r.Ajuste != nil {
		ajuste := *r.Ajuste
		ajuste.Campos = maps.Clone(r.Ajuste.Campos)
		copia.Ajuste = &ajuste
	}
	return copia
}

// SolicitudCambioAjuste no admite un valor anterior del cliente. La fuente
// de ese valor es la versión previa de ajustes o la entrada base exacta.
type SolicitudCambioAjuste struct {
	ReglaClave string
	Campo      string
	Nuevo      string
}

// CambioAjustePreparado incluye el anterior que verificará CT148.
type CambioAjustePreparado struct {
	ReglaClave string
	Campo      string
	Anterior   string
	Nuevo      string
}

// DatosPreparacionAjustes es el material puro de una versión candidata. No
// publica una versión ni valida autorización; el caso de uso posterior debe
// ligar estos datos a su transacción y a una clave de idempotencia.
type DatosPreparacionAjustes struct {
	CatalogoAjustesID string
	VersionEsperada   int
	BaseVersion       int
	BaseHuellaSHA256  string
	// Nil conserva la omisión: el almacén fija entonces la fecha de publicación.
	EfectoDesde  *time.Time
	Ajustes      map[string]map[string]string
	Canonico     []byte
	HuellaSHA256 string
	Cambios      []CambioAjustePreparado
}

// PreparacionAjustes conserva copias internas para que quien inspeccione sus
// datos no pueda cambiar el material preparado.
type PreparacionAjustes struct{ datos DatosPreparacionAjustes }

func (p PreparacionAjustes) Datos() DatosPreparacionAjustes {
	d := p.datos
	d.Ajustes = copiarConjuntoAjustes(d.Ajustes)
	if d.EfectoDesde != nil {
		fecha := *d.EfectoDesde
		d.EfectoDesde = &fecha
	}
	d.Canonico = slices.Clone(d.Canonico)
	d.Cambios = slices.Clone(d.Cambios)
	return d
}

func copiarConjuntoAjustes(origen map[string]map[string]string) map[string]map[string]string {
	copia := make(map[string]map[string]string, len(origen))
	for clave, campos := range origen {
		copia[clave] = maps.Clone(campos)
	}
	return copia
}

func versionAjustesVacia(v VersionAjustes) bool {
	return v.CatalogoID == "" && v.Version == 0 && v.HuellaSHA256 == "" &&
		v.VigenteDesde.IsZero() && len(v.Ajustes) == 0
}

// PrepararCambioAjustes calcula el conjunto completo y su huella, y obtiene
// cada valor anterior exclusivamente de la versión previa o de la base.
// VersionEsperada exige la preimagen exacta: para un replay el caso de uso
// debe reutilizar el material original ligado a la clave, o recuperar esa
// preimagen histórica. Nunca debe reprocesar la cabeza actual, que cambiaría
// la huella de solicitud y perdería el recibo. Una lista vacía no sirve:
// CT148 la rechaza antes de consultar la clave de idempotencia. Los cambios
// nuevos deben variar el valor anterior; un replay usa la solicitud original
// conservada y no vuelve a pasar por esta preparación.
func PrepararCambioAjustes(
	catalogo domain.CatalogoConfigurable, instante time.Time, versionEsperada int,
	previa VersionAjustes, encontrada bool, solicitadas []SolicitudCambioAjuste,
) (PreparacionAjustes, error) {
	return PrepararAjustesSobreVersion(catalogo, instante, nil, versionEsperada, previa, encontrada, solicitadas)
}

// PrepararAjustesSobreVersion calcula una versión nueva desde la cabeza completa
// que leyó un repositorio autorizado. La cabeza puede ser futura: su número,
// contenido y huella son la preimagen del CAS, aunque todavía no rija hoy.
// La fecha explícita se normaliza a la precisión de PostgreSQL. Nil conserva
// la omisión para que el almacén fije el instante dentro de la transacción.
func PrepararAjustesSobreVersion(
	catalogo domain.CatalogoConfigurable, ahora time.Time, efectoDesde *time.Time,
	versionEsperada int, cabeza VersionAjustes, encontrada bool,
	solicitadas []SolicitudCambioAjuste,
) (PreparacionAjustes, error) {
	var vacia PreparacionAjustes
	if ahora.IsZero() || versionEsperada < 0 || versionEsperada > 9_999_998 ||
		len(solicitadas) == 0 || len(solicitadas) > maximoReglasAjustadas*4 {
		return vacia, ErrAjusteInvalido
	}
	fecha, efecto, err := normalizarEfectoAjustes(ahora, efectoDesde, false)
	if err != nil {
		return vacia, err
	}
	base, err := catalogo.ClonarCanonico()
	if err != nil || !catalogoVigenteEn(base, ahora.UTC()) || !catalogoVigenteEn(base, fecha) {
		return vacia, ErrReglasNoDisponibles
	}
	huellaBase, err := base.HuellaSHA256()
	if err != nil {
		return vacia, ErrReglasNoDisponibles
	}
	id := CatalogoAjustesDe(base.ID)
	if encontrada {
		if err := validarCabezaAjustes(cabeza, id); err != nil {
			return vacia, err
		}
	} else if !versionAjustesVacia(cabeza) {
		return vacia, ErrAjustesNoDisponibles
	}
	if cabeza.Version != versionEsperada {
		return vacia, ErrAjustesConflicto
	}
	if encontrada && fecha.Before(cabeza.VigenteDesde.UTC().Truncate(time.Microsecond)) {
		return vacia, ErrAjusteInvalido
	}
	ajustes := copiarConjuntoAjustes(cabeza.Ajustes)
	cambios := make([]CambioAjustePreparado, 0, len(solicitadas))
	tocadas := make(map[string]domain.EntradaCatalogoConfigurable)
	vistas := make(map[string]Regla)
	vistasPorClave := make(map[string]domain.EntradaCatalogoConfigurable)
	for _, entrada := range base.Entradas {
		if entrada.VigenteEn(fecha) {
			vistasPorClave[entrada.Clave] = entrada
		}
	}
	vistos := make(map[string]bool, len(solicitadas))
	for _, solicitud := range solicitadas {
		claveCampo := solicitud.ReglaClave + "\x00" + solicitud.Campo
		if vistos[claveCampo] || !claveAjusteCanonica(solicitud.ReglaClave) ||
			!campoAjustable(solicitud.Campo) || !valorAjusteCanonico(solicitud.Nuevo) {
			return vacia, ErrAjusteInvalido
		}
		vistos[claveCampo] = true
		entrada, existe := vistasPorClave[solicitud.ReglaClave]
		if !existe {
			return vacia, ErrAjusteInvalido
		}
		regla, vista := vistas[solicitud.ReglaClave]
		if !vista {
			regla, err = reglaDesdeEntrada(base, huellaBase, base.FuenteRef == MarcaPaqueteEjemplo, entrada)
			if err != nil {
				return vacia, ErrReglaInvalida
			}
			vistas[solicitud.ReglaClave] = regla
		}
		if !regla.Edicion.Admite(solicitud.Campo) {
			return vacia, ErrAjusteInvalido
		}
		anterior, ajustado := cabeza.Ajustes[solicitud.ReglaClave][solicitud.Campo]
		if !ajustado {
			anterior, ajustado = entrada.Atributos[solicitud.Campo]
		}
		if !ajustado || anterior == solicitud.Nuevo {
			return vacia, ErrAjusteInvalido
		}
		if ajustes[solicitud.ReglaClave] == nil {
			ajustes[solicitud.ReglaClave] = make(map[string]string)
		}
		ajustes[solicitud.ReglaClave][solicitud.Campo] = solicitud.Nuevo
		cambios = append(cambios, CambioAjustePreparado{
			ReglaClave: solicitud.ReglaClave, Campo: solicitud.Campo, Anterior: anterior, Nuevo: solicitud.Nuevo,
		})
		tocadas[solicitud.ReglaClave] = entrada
	}
	canonico, err := CanonicoAjustes(ajustes)
	if err != nil {
		return vacia, err
	}
	huella, err := HuellaAjustes(ajustes)
	if err != nil {
		return vacia, err
	}
	versionCandidata := VersionAjustes{CatalogoID: id, Version: cabeza.Version + 1,
		HuellaSHA256: huella, VigenteDesde: fecha, Ajustes: ajustes}
	for clave, entrada := range tocadas {
		if _, err := aplicarAjuste(base, huellaBase, base.FuenteRef == MarcaPaqueteEjemplo,
			entrada, vistas[clave], versionCandidata, ajustes[clave]); err != nil {
			return vacia, ErrAjusteInvalido
		}
	}
	slices.SortFunc(cambios, func(a, b CambioAjustePreparado) int {
		if orden := strings.Compare(a.ReglaClave, b.ReglaClave); orden != 0 {
			return orden
		}
		return strings.Compare(a.Campo, b.Campo)
	})
	return PreparacionAjustes{datos: DatosPreparacionAjustes{
		CatalogoAjustesID: id, VersionEsperada: versionEsperada,
		BaseVersion: base.Version, BaseHuellaSHA256: huellaBase,
		EfectoDesde: efecto,
		Ajustes:     ajustes, Canonico: canonico, HuellaSHA256: huella, Cambios: cambios,
	}}, nil
}

func validarCabezaAjustes(cabeza VersionAjustes, id string) error {
	if cabeza.VigenteDesde.IsZero() {
		return ErrAjustesNoDisponibles
	}
	return validarVersionAjustes(cabeza, id, cabeza.VigenteDesde)
}

func normalizarEfectoAjustes(ahora time.Time, solicitado *time.Time, repeticion bool) (time.Time, *time.Time, error) {
	if ahora.IsZero() {
		return time.Time{}, nil, ErrAjusteInvalido
	}
	actual := ahora.UTC().Truncate(time.Microsecond)
	if solicitado == nil {
		return actual, nil, nil
	}
	if solicitado.IsZero() {
		return time.Time{}, nil, ErrAjusteInvalido
	}
	fecha := solicitado.UTC().Truncate(time.Microsecond)
	if !repeticion && fecha.Before(actual) {
		return time.Time{}, nil, ErrAjusteInvalido
	}
	return fecha, &fecha, nil
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
