// Package httpinterno sirve la consulta interna de solo lectura de las reglas
// vigentes (plazos, límites y listas) de los catálogos de reglas. La
// autenticación y la autorización de la ruta las decide la frontera común de
// lectura de RRHH antes de llegar aquí; este adaptador solo valida la forma
// exacta de la petición y nunca deduce identidad ni permisos de ella.
package httpinterno

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/reglas"
)

const (
	// RutaReglasVigentes es la única ruta de la consulta.
	RutaReglasVigentes = "/api/vec/reglas/vigentes"
	// Esquema versiona el contrato de la respuesta.
	Esquema = "vec.reglas.vigentes.v1"

	maximoFuentes   = 8
	maximoRespuesta = 1 << 20
	plazoConsulta   = 10 * time.Second
)

// Estados de cada catálogo en la respuesta.
const (
	EstadoDisponible   = "disponible"
	EstadoSinCatalogo  = "sin_catalogo"
	EstadoNoDisponible = "no_disponible"
)

// ConsultaReglas es lo que el adaptador necesita de un resolutor de reglas.
type ConsultaReglas interface {
	Reglas(context.Context) ([]reglas.Regla, error)
}

// Fuente es un catálogo de reglas de un módulo. Consulta puede ser nula: el
// catálogo aparece entonces como «sin catálogo», nunca con reglas supuestas.
type Fuente struct {
	Modulo     string
	CatalogoID string
	Consulta   ConsultaReglas
}

// Manejador responde con las reglas vigentes de cada fuente, en el orden de
// composición. El fallo de un catálogo no oculta los demás.
type Manejador struct {
	fuentes []Fuente
}

// NuevoManejador copia las fuentes. Exige módulo y catálogo en cada una.
func NuevoManejador(fuentes ...Fuente) (*Manejador, error) {
	if len(fuentes) == 0 || len(fuentes) > maximoFuentes {
		return nil, errConfiguracion
	}
	copia := make([]Fuente, 0, len(fuentes))
	for _, f := range fuentes {
		if f.Modulo == "" || f.CatalogoID == "" || f.Modulo != strings.TrimSpace(f.Modulo) || f.CatalogoID != strings.TrimSpace(f.CatalogoID) {
			return nil, errConfiguracion
		}
		if nula(f.Consulta) {
			f.Consulta = nil
		}
		copia = append(copia, f)
	}
	return &Manejador{fuentes: copia}, nil
}

var errConfiguracion = errors.New("reglas httpinterno: configuracion no valida")

// Respuesta es el contrato vec.reglas.vigentes.v1.
type Respuesta struct {
	Data Datos `json:"data"`
}

type Datos struct {
	Esquema   string     `json:"esquema"`
	Catalogos []Catalogo `json:"catalogos"`
}

type Catalogo struct {
	Modulo         string       `json:"modulo"`
	CatalogoID     string       `json:"catalogo_id"`
	Estado         string       `json:"estado"`
	Version        int          `json:"version,omitempty"`
	HuellaSHA256   string       `json:"huella_sha256,omitempty"`
	PaqueteEjemplo bool         `json:"paquete_ejemplo"`
	Reglas         []ReglaVista `json:"reglas"`
}

// ReglaVista es la vista de lectura de una regla. Cantidad es cero cuando la
// unidad no la admite; Valor lleva la franja o la lista.
type ReglaVista struct {
	Clave          string `json:"clave"`
	Etiqueta       string `json:"etiqueta"`
	Descripcion    string `json:"descripcion"`
	Unidad         string `json:"unidad"`
	Cantidad       int    `json:"cantidad,omitempty"`
	Valor          string `json:"valor,omitempty"`
	Computo        string `json:"computo,omitempty"`
	Inicio         string `json:"inicio,omitempty"`
	Origen         string `json:"origen"`
	Articulo       string `json:"articulo,omitempty"`
	Norma          string `json:"norma"`
	Duda           string `json:"duda"`
	ParteEjemplo   string `json:"ejemplo_parcial,omitempty"`
	Version        int    `json:"version"`
	Referencia     string `json:"referencia"`
	PaqueteEjemplo bool   `json:"paquete_ejemplo"`
}

func (m *Manejador) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cabecerasSeguras(w)
	if r == nil || r.URL == nil {
		responderError(w, r, http.StatusBadRequest, "solicitud_invalida")
		return
	}
	if r.URL.Path != RutaReglasVigentes {
		responderError(w, r, http.StatusNotFound, "recurso_no_encontrado")
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		responderError(w, r, http.StatusMethodNotAllowed, "metodo_no_permitido")
		return
	}
	if r.URL.RawQuery != "" || r.ContentLength != 0 || len(r.TransferEncoding) != 0 || cabeceraProhibida(r.Header) {
		responderError(w, r, http.StatusBadRequest, "solicitud_invalida")
		return
	}
	if m == nil || len(m.fuentes) == 0 {
		responderError(w, r, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	ctx, cancelar := context.WithTimeout(r.Context(), plazoConsulta)
	defer cancelar()
	datos := Datos{Esquema: Esquema, Catalogos: make([]Catalogo, 0, len(m.fuentes))}
	for _, f := range m.fuentes {
		catalogo := consultar(ctx, f)
		if ctx.Err() != nil {
			responderError(w, r, http.StatusServiceUnavailable, "servicio_no_disponible")
			return
		}
		datos.Catalogos = append(datos.Catalogos, catalogo)
	}
	responder(w, r, http.StatusOK, Respuesta{Data: datos})
}

func consultar(ctx context.Context, f Fuente) Catalogo {
	catalogo := Catalogo{Modulo: f.Modulo, CatalogoID: f.CatalogoID, Estado: EstadoSinCatalogo, Reglas: []ReglaVista{}}
	if f.Consulta == nil {
		return catalogo
	}
	vigentes, err := f.Consulta.Reglas(ctx)
	switch {
	case errors.Is(err, reglas.ErrReglasNoConfiguradas):
		return catalogo
	case err != nil:
		catalogo.Estado = EstadoNoDisponible
		return catalogo
	}
	catalogo.Estado = EstadoDisponible
	for _, regla := range vigentes {
		if regla.ReferenciaEntrada.CatalogoID != f.CatalogoID {
			// Un resolutor que devuelve otro catálogo no se mezcla con este.
			return Catalogo{Modulo: f.Modulo, CatalogoID: f.CatalogoID, Estado: EstadoNoDisponible, Reglas: []ReglaVista{}}
		}
		catalogo.Version = regla.ReferenciaEntrada.CatalogoVersion
		catalogo.HuellaSHA256 = regla.HuellaCatalogo
		catalogo.PaqueteEjemplo = catalogo.PaqueteEjemplo || regla.PaqueteEjemplo
		catalogo.Reglas = append(catalogo.Reglas, ReglaVista{
			Clave: regla.Clave, Etiqueta: regla.Etiqueta, Descripcion: regla.Descripcion,
			Unidad: string(regla.Unidad), Cantidad: regla.Cantidad, Valor: regla.Valor,
			Computo: string(regla.Computo), Inicio: regla.Inicio, Origen: string(regla.Origen),
			Articulo: regla.Articulo, Norma: regla.Norma, Duda: regla.Duda, ParteEjemplo: regla.ParteEjemplo,
			Version: regla.ReferenciaEntrada.CatalogoVersion, Referencia: regla.Referencia,
			PaqueteEjemplo: regla.PaqueteEjemplo,
		})
	}
	return catalogo
}

func cabeceraProhibida(h http.Header) bool {
	for nombre := range h {
		n := strings.ToLower(nombre)
		switch {
		case n == "cookie", n == "authorization", n == "proxy-authorization", n == "forwarded", n == "remote-user",
			n == "x-remote-user", n == "x-forwarded-user", n == "x-http-method-override", n == "content-encoding",
			strings.HasPrefix(n, "x-auth-"), strings.HasPrefix(n, "x-vec-"), strings.HasPrefix(n, "x-forwarded-"),
			strings.Contains(n, "role"):
			return true
		}
	}
	return false
}

func cabecerasSeguras(w http.ResponseWriter) {
	for _, c := range []string{"Set-Cookie", "Access-Control-Allow-Origin", "Access-Control-Allow-Credentials", "Location"} {
		w.Header().Del(c)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, no-transform")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; base-uri 'none'; frame-ancestors 'none'")
}

func responderError(w http.ResponseWriter, r *http.Request, estado int, codigo string) {
	responder(w, r, estado, map[string]any{"error": map[string]string{
		"codigo": codigo, "clave_i18n": "api.reglas.error." + codigo,
	}})
}

func responder(w http.ResponseWriter, r *http.Request, estado int, valor any) {
	contenido, err := json.Marshal(valor)
	if err != nil || len(contenido) > maximoRespuesta {
		estado = http.StatusServiceUnavailable
		contenido = []byte(`{"error":{"codigo":"servicio_no_disponible","clave_i18n":"api.reglas.error.servicio_no_disponible"}}`)
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(contenido)))
	w.WriteHeader(estado)
	if r == nil || r.Method != http.MethodHead {
		_, _ = w.Write(contenido)
	}
}

func nula(dependencia any) bool {
	if dependencia == nil {
		return true
	}
	valor := reflect.ValueOf(dependencia)
	switch valor.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return valor.IsNil()
	default:
		return false
	}
}
