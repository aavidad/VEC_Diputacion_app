// Command vec-preparar-material-interno compone y valida el inventario
// privado personal_b2_v3.json (formato 4) de vec-interno a partir del
// material privado existente y lo coteja con el gobierno V3 publicado.
//
// No publica gobierno, raíz, configuración, claves ni permisos; no amplía el
// rol de preflight y no lee secretos del gobierno. Las claves B2 se obtienen
// con bootstrap.DerivarClavesPersonalB2V3DesdeMaterialDesarrollo, que recorre
// la misma ruta de código que vec-server desde su material privado de
// idempotencia; la clave base CT sólo existe en memoria durante esa llamada.
// Las claves se escriben sólo si sus huellas, versiones, revisiones y
// vigencias coinciden con lo publicado.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// variableDSN nombra la variable de entorno alternativa al fichero 0600 con
// el DSN del gobierno. Se retira del entorno del proceso para que no pase a
// procesos hijos, pero eso no borra la copia inicial que conserva el núcleo
// (visible en /proc/<pid>/environ para el mismo usuario y root) ni la de la
// shell que la exportó. Se recomienda -dsn-archivo.
const variableDSN = "VEC_PREPARAR_MATERIAL_GOBIERNO_DSN"

var errUso = errors.New("uso: vec-preparar-material-interno -inventario-ct RUTA/ct_v3.json -material-idempotencia RUTA/idempotencia -motivos RUTA -salida DIRECTORIO_NUEVO [-dsn-archivo RUTA]")

type dependencias struct {
	abrirGobierno  func(context.Context, string) (fuenteGobierno, error)
	reloj          func() time.Time
	antesDeActivar func() error
	// sincronizarPadre sólo se sustituye en pruebas; nil usa fsync(2).
	sincronizarPadre func(*os.File) error
}

func main() {
	ctx, parar := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	dsn, hayDSN := os.LookupEnv(variableDSN)
	_ = os.Unsetenv(variableDSN)
	codigo := ejecutar(ctx, os.Args[1:], dsn, hayDSN, os.Stdout, os.Stderr, dependencias{
		abrirGobierno: abrirGobiernoPostgreSQL,
		reloj:         time.Now,
	})
	parar()
	os.Exit(codigo)
}

// ejecutar devuelve 0 si el material quedó activado, 2 ante un uso incorrecto
// y 1 ante cualquier otro fallo. Los mensajes son fijos: nunca incluyen
// secretos, DSN, rutas privadas ni errores de bibliotecas.
func ejecutar(ctx context.Context, args []string, dsnEntorno string, hayDSNEntorno bool, salida, errores io.Writer, d dependencias) int {
	banderas := flag.NewFlagSet("vec-preparar-material-interno", flag.ContinueOnError)
	banderas.SetOutput(io.Discard)
	var o opciones
	banderas.StringVar(&o.inventarioCT, "inventario-ct", "", "ruta absoluta de ct_v3.json existente")
	banderas.StringVar(&o.idempotencia, "material-idempotencia", "", "subdirectorio idempotencia del material de desarrollo de vec-server (0700)")
	banderas.StringVar(&o.motivos, "motivos", "", "fichero JSON 0600 con los ocho motivos B2")
	banderas.StringVar(&o.salida, "salida", "", "directorio nuevo (inexistente o vacío, 0700)")
	banderas.StringVar(&o.dsnArchivo, "dsn-archivo", "", "fichero 0600 con el DSN del LOGIN de gobierno de vec-server")
	if err := banderas.Parse(args); err != nil || banderas.NArg() != 0 || o.inventarioCT == "" || o.idempotencia == "" || o.motivos == "" || o.salida == "" {
		fmt.Fprintln(errores, errUso)
		return 2
	}
	if (o.dsnArchivo == "") == !hayDSNEntorno {
		fmt.Fprintln(errores, errDSN)
		return 2
	}
	p := preparacion{opciones: o, dsnEntorno: dsnEntorno, dep: d}
	sincronizado, err := p.preparar(ctx)
	if err != nil {
		fmt.Fprintln(errores, mensajeSeguro(err))
		return 1
	}
	if !sincronizado {
		// rename(2) ya se hizo: el material está activado y visible; sólo
		// falta confirmar que la entrada del directorio padre es durable ante
		// un corte de energía. No es un fallo de preparación.
		fmt.Fprintln(salida, "material Personal B2 activado; fsync del padre no confirmado")
		return 0
	}
	fmt.Fprintln(salida, "material Personal B2 preparado y validado: personal_b2_v3.json (formato 4) y 8 claves de capacidad")
	return 0
}

// mensajeSeguro sólo deja pasar los errores propios de esta herramienta.
func mensajeSeguro(err error) string {
	var propio errorPropio
	if errors.As(err, &propio) {
		return propio.Error()
	}
	return "preparación rechazada"
}

// errorPropio es un mensaje fijo sin datos variables.
type errorPropio string

func (e errorPropio) Error() string { return string(e) }
