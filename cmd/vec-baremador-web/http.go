package main

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"os"
	"regexp"
	"strings"
	"syscall"

	"vec-diputacion-granada/internal/modules/bolsa/adapters/simuladorlocal"
	provision "vec-diputacion-granada/internal/modules/provision/adapters/simulacion"
)

const entradaWeb = "/portal-empleado/modulos/bolsa/baremo/"
const entradaProvision = "/portal-empleado/modulos/provision/"
const entradaSeleccion = "/portal-empleado/modulos/seleccion/"

var recursos = []string{
	"/portal-empleado/modulos/bolsa/baremo/index.html",
	"/portal-empleado/modulos/bolsa/baremo/baremo.css",
	"/portal-empleado/modulos/bolsa/baremo/montaje.js",
	"/portal-empleado/modulos/bolsa/baremo-editor.js",
	"/portal-empleado/modulos/bolsa/baremo-cliente.js",
	"/portal-empleado/modulos/bolsa/baremo-vista.js",
	"/portal-empleado/modulos/bolsa/concursos-cliente.js",
	"/portal-empleado/modulos/bolsa/concursos-editor.js",
	"/portal-empleado/modulos/bolsa/concursos-vista.js",
	"/portal-empleado/modulos/bolsa/concursos-montaje.js",
	"/portal-empleado/modulos/provision/index.html",
	"/portal-empleado/modulos/provision/entrada.js",
	"/portal-empleado/modulos/provision/modelo.js",
	"/portal-empleado/modulos/provision/vista.js",
	"/portal-empleado/modulos/provision/montaje.js",
	"/portal-empleado/modulos/provision/cliente-local.js",
	"/portal-empleado/modulos/provision/ensayos-modelo.js",
	"/portal-empleado/modulos/provision/ensayos-vista.js",
	"/portal-empleado/modulos/provision/ensayos-cliente.js",
	"/portal-empleado/modulos/seleccion/index.html",
	"/portal-empleado/modulos/seleccion/seleccion.css",
	"/portal-empleado/modulos/seleccion/entrada.js",
	"/portal-empleado/modulos/seleccion/cliente.js",
	"/portal-empleado/modulos/seleccion/configuracion.js",
	"/portal-empleado/modulos/seleccion/estado.js",
	"/portal-empleado/modulos/seleccion/dom.js",
	"/portal-empleado/modulos/seleccion/formulario.js",
	"/portal-empleado/modulos/seleccion/resultado.js",
	"/portal-empleado/modulos/seleccion/montaje.js",
	"/portal-empleado/portal.css", "/portal-empleado/portal-componentes.css",
	"/portal-empleado/portal-patrones.css", "/portal-empleado/portal-flujos.css", "/portal-empleado/portal-modulos.css",
	"/comun/tema-vec.css", "/comun/textos.js", "/comun/idioma.js",
	"/textos/idiomas.json",
	"/catalogos/baremo-jornada-v1.json",
	"/catalogos/baremo-restos-v1.json",
}

var codigoIdioma = regexp.MustCompile(`^[a-z]{2,3}(?:-[a-z0-9]{2,8})*$`)

type recurso struct {
	contenido []byte
	tipo      string
}

// La raíz la fija el operador al arrancar. Se cargan solo ficheros regulares
// autorizados, sin escapes por enlaces, y no se sirve el directorio por HTTP.
func cargarRecursos(dir string) (map[string]recurso, error) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	salida := make(map[string]recurso, len(recursos)+1)
	cargar := func(ruta string) error {
		f, err := root.OpenFile(strings.TrimPrefix(ruta, "/"), os.O_RDONLY|syscall.O_NONBLOCK, 0)
		if err != nil {
			return err
		}
		st, statErr := f.Stat()
		if statErr != nil || !st.Mode().IsRegular() || st.Size() == 0 || st.Size() > 1024*1024 {
			if err := f.Close(); err != nil {
				return errors.New("recurso_invalido")
			}
			return errors.New("recurso_invalido")
		}
		b, readErr := io.ReadAll(io.LimitReader(f, 1024*1024+1))
		closeErr := f.Close()
		if readErr != nil || closeErr != nil || len(b) > 1024*1024 {
			return errors.New("recurso_invalido")
		}
		ext := ruta[strings.LastIndex(ruta, "."):]
		salida[ruta] = recurso{b, mime.TypeByExtension(ext)}
		return nil
	}
	for _, ruta := range recursos {
		if err := cargar(ruta); err != nil {
			if ruta == "/catalogos/baremo-restos-v1.json" && errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}
	}
	// Los idiomas proceden del catálogo común de datos. Cada uno añade solo
	// el catálogo de este editor a la lista cerrada de recursos.
	var indice struct {
		Idiomas []struct {
			Codigo string `json:"codigo"`
		} `json:"idiomas"`
	}
	if err := json.Unmarshal(salida["/textos/idiomas.json"].contenido, &indice); err != nil || len(indice.Idiomas) == 0 || len(indice.Idiomas) > 8 {
		return nil, errors.New("indice_idiomas_invalido")
	}
	vistos := make(map[string]bool)
	for _, idioma := range indice.Idiomas {
		if len(idioma.Codigo) > 32 || !codigoIdioma.MatchString(idioma.Codigo) || vistos[idioma.Codigo] {
			return nil, errors.New("indice_idiomas_invalido")
		}
		vistos[idioma.Codigo] = true
		for _, catalogo := range []string{"baremo-bolsa", "baremo-concursos", "provision", "seleccion"} {
			if err := cargar("/textos/" + idioma.Codigo + "/" + catalogo + ".json"); err != nil {
				return nil, err
			}
		}
	}
	salida[entradaWeb] = salida[entradaWeb+"index.html"]
	salida[entradaProvision] = salida[entradaProvision+"index.html"]
	salida[entradaSeleccion] = salida[entradaSeleccion+"index.html"]
	return salida, nil
}

func nuevoHandler(host string, assets map[string]recurso) http.Handler {
	origen := "http://" + host
	ocupadas := make(chan struct{}, 2)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self'; connect-src 'self'; img-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'")
		if r.Host != host || len(r.Header.Values("Origin")) > 1 || (r.Header.Get("Origin") != "" && r.Header.Get("Origin") != origen) || r.Header.Get("Sec-Fetch-Site") == "cross-site" || r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" {
			responderError(w, http.StatusForbidden, "origen_no_admitido")
			return
		}
		if r.URL.RawQuery != "" && (r.URL.Path == "/simular" || strings.HasPrefix(r.URL.Path, "/api/provision/") || strings.HasPrefix(r.URL.Path, "/api/seleccion/")) {
			responderError(w, http.StatusBadRequest, "solicitud_invalida")
			return
		}
		switch r.URL.Path {
		case "/favicon.ico":
			if !metodo(w, r, http.MethodGet) {
				return
			}
			w.WriteHeader(http.StatusNoContent)
		case "/":
			if !metodo(w, r, http.MethodGet) {
				return
			}
			http.Redirect(w, r, entradaWeb, http.StatusFound)
		case "/ejemplos":
			if !metodo(w, r, http.MethodGet) {
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(struct {
				Ejemplos []simuladorlocal.Ejemplo `json:"ejemplos"`
			}{(simuladorlocal.Motor{}).Ejemplos()})
		case "/api/provision/v1/configuracion-local":
			if !metodo(w, r, http.MethodGet) {
				return
			}
			ejemplos, err := provision.Ejemplos()
			if err != nil {
				responderError(w, http.StatusServiceUnavailable, "configuracion_no_disponible")
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(struct {
				Ejemplos []provision.Ejemplo `json:"ejemplos"`
			}{ejemplos})
		case "/api/seleccion/v1/ensayos":
			if !metodo(w, r, http.MethodGet) {
				return
			}
			configurarSeleccionLocal(w)
		case rutaProcesosLocales:
			if !metodo(w, r, http.MethodGet) {
				return
			}
			configurarProcesosLocales(w)
		case rutaAdjudicacionesLocales, rutaCiclosLocales:
			if !metodo(w, r, http.MethodGet) {
				return
			}
			if r.URL.Path == rutaAdjudicacionesLocales {
				configurarAdjudicacionesLocales(w)
			} else {
				configurarCiclosLocales(w)
			}
		case "/simular", "/api/provision/v1/simulaciones", rutaSimulacionProceso, rutaSimularAdjudicacion, rutaSimularCiclo, "/api/seleccion/v1/simulaciones":
			if !metodo(w, r, http.MethodPost) {
				return
			}
			if r.Header.Get("Origin") != origen {
				responderError(w, http.StatusForbidden, "origen_no_admitido")
				return
			}
			tipo, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if err != nil || tipo != "application/json" || r.Header.Get("Content-Encoding") != "" {
				responderError(w, http.StatusUnsupportedMediaType, "tipo_no_admitido")
				return
			}
			select {
			case ocupadas <- struct{}{}:
				defer func() { <-ocupadas }()
			default:
				responderError(w, http.StatusTooManyRequests, "simulacion_ocupada")
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, simuladorlocal.MaximoBytes)
			defer r.Body.Close()
			b, err := io.ReadAll(r.Body)
			if err != nil {
				var grande *http.MaxBytesError
				estado := http.StatusBadRequest
				if errors.As(err, &grande) {
					estado = http.StatusRequestEntityTooLarge
				}
				responderError(w, estado, "solicitud_invalida")
				return
			}
			if r.URL.Path == "/api/seleccion/v1/simulaciones" {
				simularSeleccionLocal(w, b)
				return
			}
			if r.URL.Path == "/api/provision/v1/simulaciones" {
				simularConcursos(w, b)
				return
			}
			if r.URL.Path == rutaSimulacionProceso {
				simularProcesoLocal(w, b)
				return
			}
			if r.URL.Path == rutaSimularAdjudicacion {
				simularAdjudicacionLocal(w, b)
				return
			}
			if r.URL.Path == rutaSimularCiclo {
				simularCicloLocal(w, b)
				return
			}
			s, err := simuladorlocal.Decodificar(b)
			if err != nil {
				responderError(w, http.StatusBadRequest, "solicitud_invalida")
				return
			}
			resultado, err := (simuladorlocal.Motor{}).Simular(s)
			if err != nil {
				responderError(w, http.StatusUnprocessableEntity, "reglas_invalidas")
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(resultado)
		default:
			asset, ok := assets[r.URL.Path]
			if !ok {
				responderError(w, http.StatusNotFound, "ruta_no_admitida")
				return
			}
			if !metodo(w, r, http.MethodGet) {
				return
			}
			w.Header().Set("Content-Type", asset.tipo)
			_, _ = w.Write(asset.contenido)
		}
	})
}

func metodo(w http.ResponseWriter, r *http.Request, permitido string) bool {
	if r.Method == permitido {
		return true
	}
	w.Header().Set("Allow", permitido)
	responderError(w, http.StatusMethodNotAllowed, "metodo_no_admitido")
	return false
}

func responderError(w http.ResponseWriter, estado int, codigo string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(estado)
	_ = json.NewEncoder(w).Encode(struct {
		Error string `json:"error"`
	}{codigo})
}
