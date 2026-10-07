// Package calculomeritos valora hechos sintéticos según reglas propias de Bolsa.
// No admite fórmulas ejecutables ni toma decisiones de admisión administrativa.
package calculomeritos

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"

	"vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
	"vec-diputacion-granada/internal/shared/baremacion"
)

const (
	EsquemaReglas    = "vec.bolsa.reglas_meritos.v1"
	EsquemaEntrada   = "vec.bolsa.entrada_meritos.v1"
	EsquemaResultado = "vec.bolsa.resultado_meritos.v1"
	MaximoBytes      = 1024 * 1024
)

var patronClave = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)
var patronReferencia = regexp.MustCompile(`^[a-z]{3}_[0-9a-f]{32}$`)

// Error solo expone un código nominal; nunca conserva documentos o rutas.
type Error struct{ Codigo string }

func (e *Error) Error() string  { return "meritos:" + e.Codigo }
func fallo(codigo string) error { return &Error{Codigo: codigo} }

// Dependencia fija una fuente y su versión exacta, sin buscar la vigente.
type Dependencia struct {
	Referencia   string `json:"referencia"`
	Version      uint64 `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
}

func (d Dependencia) validar() error {
	if !patronReferencia.MatchString(d.Referencia) {
		return fallo("referencia_no_opaca")
	}
	_, err := reglasbaremo.NuevaReferenciaVersionada(d.Referencia, d.Version, d.HuellaSHA256)
	if err != nil {
		return fallo("dependencia_invalida")
	}
	return nil
}

// Conjunto configura la política de duplicados y el momento de redondeo de
// forma explícita. En V1 se bloquean duplicados y se redondea por regla.
type Conjunto struct {
	Esquema             string                `json:"esquema"`
	Referencia          string                `json:"referencia"`
	Version             uint64                `json:"version"`
	ConvocatoriaRef     string                `json:"convocatoria_ref"`
	ExpedienteRef       string                `json:"expediente_ref"`
	Bases               Dependencia           `json:"bases"`
	FechaCorteInclusiva baremacion.FechaCivil `json:"fecha_corte_inclusiva"`
	Duplicados          string                `json:"duplicados"`
	MomentoRedondeo     string                `json:"momento_redondeo"`
	SeleccionElementos  string                `json:"seleccion_elementos"`
	MaximoTotal         *baremacion.Puntos    `json:"maximo_total"`
	Secciones           []Seccion             `json:"secciones"`
	Reglas              []Regla               `json:"reglas"`
}

type Seccion struct {
	Clave        string             `json:"clave"`
	Definicion   Dependencia        `json:"definicion"`
	MaximoPuntos *baremacion.Puntos `json:"maximo_puntos"`
}

// Familia/unidad quedan cerradas: formación/hora, titulación/titulo y otros/unidad.
// Clase procede de un catálogo versionado y no de nombres libres de cursos.
type Regla struct {
	Clave           string                  `json:"clave"`
	Definicion      Dependencia             `json:"definicion"`
	SeccionClave    string                  `json:"seccion_clave"`
	Familia         string                  `json:"familia"`
	Clase           string                  `json:"clase"`
	Catalogo        Dependencia             `json:"catalogo"`
	Unidad          string                  `json:"unidad"`
	MinimoUnidades  *baremacion.Racional    `json:"minimo_unidades"`
	MaximoUnidades  *baremacion.Racional    `json:"maximo_unidades"`
	MaximoElementos *uint32                 `json:"maximo_elementos"`
	PuntosPorUnidad *baremacion.Puntos      `json:"puntos_por_unidad"`
	MaximoPuntos    *baremacion.Puntos      `json:"maximo_puntos"`
	Redondeo        baremacion.ModoRedondeo `json:"redondeo"`
}

// Entrada guarda referencias opacas de hechos y evidencias, sin personas ni
// contenido documental. Una cantidad ausente es un impedimento de negocio.
type Entrada struct {
	Esquema    string   `json:"esquema"`
	Referencia string   `json:"referencia"`
	Version    uint64   `json:"version"`
	Meritos    []Merito `json:"meritos"`
}

type Merito struct {
	Referencia     string                 `json:"referencia"`
	Hecho          Dependencia            `json:"hecho"`
	Evidencia      Dependencia            `json:"evidencia"`
	Familia        string                 `json:"familia"`
	Clase          string                 `json:"clase"`
	Catalogo       Dependencia            `json:"catalogo"`
	Uso            string                 `json:"uso"`
	Estado         string                 `json:"estado"`
	Aplicabilidad  string                 `json:"aplicabilidad"`
	FechaObtencion *baremacion.FechaCivil `json:"fecha_obtencion"`
	VigenteHasta   *baremacion.FechaCivil `json:"vigente_hasta"`
	Unidades       *baremacion.Racional   `json:"unidades"`
}

func familiaUnidad(familia, unidad string) bool {
	return familia == "formacion" && unidad == "hora" || familia == "titulacion" && unidad == "titulo" || familia == "otros" && unidad == "unidad"
}
func opaca(ref, prefijo string) bool {
	if len(ref) != len(prefijo)+32 || ref[:len(prefijo)] != prefijo {
		return false
	}
	for _, c := range ref[len(prefijo):] {
		if c < '0' || c > '9' {
			if c < 'a' || c > 'f' {
				return false
			}
		}
	}
	return true
}
func puntosPositivos(p *baremacion.Puntos) bool {
	return p != nil && p.EsValido() && p.Micropuntos() > 0
}
func racionalPositivo(r *baremacion.Racional) bool {
	return r != nil && r.EsValido() && r.Numerador() > 0
}

func validarConsistenciaDependencias(dependencias []Dependencia) error {
	type identidad struct {
		ref     string
		version uint64
	}
	huellas := make(map[identidad]string)
	for _, d := range dependencias {
		key := identidad{d.Referencia, d.Version}
		if h, existe := huellas[key]; existe && h != d.HuellaSHA256 {
			return fallo("dependencia_contradictoria")
		}
		huellas[key] = d.HuellaSHA256
	}
	return nil
}

func (c Conjunto) Validar() error {
	if c.Esquema != EsquemaReglas || c.Duplicados != "bloquear" || c.MomentoRedondeo != "por_regla" || c.SeleccionElementos != "mayor_unidad" || !c.FechaCorteInclusiva.EsValida() || !puntosPositivos(c.MaximoTotal) {
		return fallo("politica_incompleta")
	}
	if _, err := reglasbaremo.NuevaIdentidadConjuntoReglasBaremo(c.Referencia, c.Version, c.ConvocatoriaRef, c.ExpedienteRef); err != nil {
		return fallo("identidad_invalida")
	}
	if err := c.Bases.validar(); err != nil {
		return err
	}
	if len(c.Secciones) == 0 || len(c.Secciones) > 32 || len(c.Reglas) == 0 || len(c.Reglas) > 128 {
		return fallo("volumen_invalido")
	}
	dependencias := []Dependencia{c.Bases}
	secciones := make(map[string]bool)
	refs := map[string]bool{c.Referencia: true, c.ConvocatoriaRef: true, c.ExpedienteRef: true}
	registrar := func(ref string) bool {
		if refs[ref] {
			return false
		}
		refs[ref] = true
		return true
	}
	if !registrar(c.Bases.Referencia) {
		return fallo("referencia_duplicada")
	}
	for i, s := range c.Secciones {
		dependencias = append(dependencias, s.Definicion)
		if !patronClave.MatchString(s.Clave) || !puntosPositivos(s.MaximoPuntos) || s.Definicion.validar() != nil || !registrar(s.Definicion.Referencia) {
			return fallo("seccion_invalida")
		}
		if i > 0 && c.Secciones[i-1].Clave >= s.Clave {
			return fallo("orden_no_canonico")
		}
		secciones[s.Clave] = true
	}
	parejas := make(map[string]bool)
	usadas := make(map[string]bool)
	for i, r := range c.Reglas {
		dependencias = append(dependencias, r.Definicion, r.Catalogo)
		pareja := r.Familia + ":" + r.Clase
		if !patronClave.MatchString(r.Clave) || !patronClave.MatchString(r.Clase) || !secciones[r.SeccionClave] || !familiaUnidad(r.Familia, r.Unidad) || !r.Redondeo.EsValido() || !puntosPositivos(r.PuntosPorUnidad) || !puntosPositivos(r.MaximoPuntos) || r.MinimoUnidades == nil || !r.MinimoUnidades.EsValido() || r.MinimoUnidades.Numerador() < 0 || !racionalPositivo(r.MaximoUnidades) || r.MaximoElementos == nil || *r.MaximoElementos == 0 || *r.MaximoElementos > 2048 {
			return fallo("regla_incompleta")
		}
		if r.Definicion.validar() != nil || r.Catalogo.validar() != nil || !registrar(r.Definicion.Referencia) {
			return fallo("dependencia_invalida")
		}
		cmp, _ := r.MinimoUnidades.Comparar(*r.MaximoUnidades)
		if cmp > 0 {
			return fallo("limites_incompatibles")
		}
		if i > 0 && c.Reglas[i-1].Clave >= r.Clave || parejas[pareja] {
			return fallo("reglas_ambiguas")
		}
		parejas[pareja] = true
		usadas[r.SeccionClave] = true
	}
	if err := validarConsistenciaDependencias(dependencias); err != nil {
		return err
	}
	for clave := range secciones {
		if !usadas[clave] {
			return fallo("seccion_sin_reglas")
		}
	}
	return nil
}
func (e Entrada) Validar() error {
	if e.Esquema != EsquemaEntrada || !opaca(e.Referencia, "ent_") || e.Version == 0 || e.Version > 1_000_000_000 || e.Meritos == nil || len(e.Meritos) > 2048 {
		return fallo("entrada_invalida")
	}
	dependencias := make([]Dependencia, 0, len(e.Meritos)*3)
	for i, m := range e.Meritos {
		dependencias = append(dependencias, m.Hecho, m.Evidencia, m.Catalogo)
		if !opaca(m.Referencia, "mer_") || !patronClave.MatchString(m.Clase) || (m.Familia != "formacion" && m.Familia != "titulacion" && m.Familia != "otros") || m.Hecho.validar() != nil || m.Evidencia.validar() != nil || m.Catalogo.validar() != nil {
			return fallo("merito_invalido")
		}
		if i > 0 && e.Meritos[i-1].Referencia >= m.Referencia {
			return fallo("orden_no_canonico")
		}
		if m.Uso != "requisito" && m.Uso != "merito" || m.Estado != "acreditado" && m.Estado != "declarado" && m.Estado != "pendiente" && m.Estado != "rechazado" || m.Aplicabilidad != "si" && m.Aplicabilidad != "no" && m.Aplicabilidad != "pendiente" {
			return fallo("estado_invalido")
		}
		if m.FechaObtencion != nil && !m.FechaObtencion.EsValida() || m.VigenteHasta != nil && !m.VigenteHasta.EsValida() || m.Unidades != nil && (!m.Unidades.EsValido() || m.Unidades.Numerador() <= 0) {
			return fallo("dato_invalido")
		}
		if m.FechaObtencion != nil && m.VigenteHasta != nil {
			cmp, _ := m.FechaObtencion.Comparar(*m.VigenteHasta)
			if cmp > 0 {
				return fallo("vigencia_invalida")
			}
		}
		if m.Familia == "titulacion" && m.Unidades != nil && m.Unidades.String() != "1/1" {
			return fallo("unidad_titulo_invalida")
		}
	}
	return validarConsistenciaDependencias(dependencias)
}

func (c Conjunto) RepresentacionCanonica() ([]byte, error) {
	if err := c.Validar(); err != nil {
		return nil, err
	}
	return json.Marshal(c)
}
func (e Entrada) RepresentacionCanonica() ([]byte, error) {
	if err := e.Validar(); err != nil {
		return nil, err
	}
	return json.Marshal(e)
}
func HuellaSHA256(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }
func restaurar(b []byte, huella string, destino any) error {
	if len(b) == 0 || len(b) > MaximoBytes || HuellaSHA256(b) != huella {
		return fallo("huella_o_volumen_invalido")
	}
	decoder := json.NewDecoder(bytes.NewReader(b))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destino); err != nil {
		return fallo("json_invalido")
	}
	canonico, err := json.Marshal(destino)
	if err != nil || !bytes.Equal(b, canonico) {
		return fallo("representacion_no_canonica")
	}
	return nil
}
func RestaurarConjunto(b []byte, huella string) (Conjunto, error) {
	var c Conjunto
	if err := restaurar(b, huella, &c); err != nil {
		return Conjunto{}, err
	}
	if err := c.Validar(); err != nil {
		return Conjunto{}, err
	}
	return c, nil
}
func RestaurarEntrada(b []byte, huella string) (Entrada, error) {
	var e Entrada
	if err := restaurar(b, huella, &e); err != nil {
		return Entrada{}, err
	}
	if err := e.Validar(); err != nil {
		return Entrada{}, err
	}
	return e, nil
}

var _ error = (*Error)(nil)
