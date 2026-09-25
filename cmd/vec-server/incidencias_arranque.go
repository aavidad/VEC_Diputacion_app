package main

import (
	"context"
	"io"
	"net/http"
	"os"
	"runtime/debug"
	"sync"
	"time"

	"vec-diputacion-granada/internal/app/server"
	"vec-diputacion-granada/internal/vec/adapters/observabilidad"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// envEntornoSupervision declara el entorno de las incidencias técnicas. Solo
// admite la lista cerrada del dominio; cualquier otro valor pasa a
// "desconocido".
const envEntornoSupervision = "VEC_ENTORNO"

// plazoRegistroFalloArranque acota lo que el proceso espera, antes de salir,
// a que la incidencia se escriba: un destino bloqueado no retiene la salida.
const plazoRegistroFalloArranque = 2 * time.Second

// registrarFalloArranque es el primer consumidor del emisor de incidencias
// técnicas (M1): declara ARRANQUE_FALLIDO en JSON Lines saneado justo antes de
// la salida con error ya existente, sin alterarla. El emisor solo se crea en
// este camino de fallo, de modo que el arranque correcto no paga ningún coste.
// Nunca se incluye el error original: su texto puede contener rutas, DSN o
// datos de configuración.
func registrarFalloArranque(destino io.Writer, componente domain.ComponenteIncidenciaTecnica, etapa domain.EtapaIncidenciaTecnica) {
	emisor, err := observabilidad.NuevoEmisorJSONLines(observabilidad.OpcionesEmisor{
		Destino:        destino,
		Capacidad:      1,
		Entorno:        os.Getenv(envEntornoSupervision),
		VersionBinario: revisionCompilada(),
	})
	if err != nil {
		escribirRegistroFijo(os.Stderr, "vec-server: emisor de incidencias tecnicas no disponible\n")
		return
	}
	emisor.Emitir(domain.SolicitudIncidenciaTecnica{Codigo: domain.IncidenciaArranqueFallido, Componente: componente, Etapa: etapa})
	ctx, cancelar := context.WithTimeout(context.Background(), plazoRegistroFalloArranque)
	defer cancelar()
	_ = emisor.Cerrar(ctx)
}

// revisionCompilada devuelve la revisión VCS incrustada por la cadena de
// compilación, o vacío; el dominio la normaliza a su formato cerrado.
func revisionCompilada() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	for _, ajuste := range info.Settings {
		if ajuste.Key == "vcs.revision" {
			return ajuste.Value
		}
	}
	return ""
}

// plazoCierreSupervision acota la espera para vaciar las incidencias
// pendientes al terminar el servidor.
const plazoCierreSupervision = 2 * time.Second

// crearEmisorServidor crea el emisor de incidencias técnicas del servidor
// (JSON Lines hacia destino, recogido fuera del proceso) antes de componer la
// aplicación, para inyectarlo por constructor en los adaptadores. Nunca
// devuelve nil: si el emisor no puede crearse, el fallo queda en registro con
// texto fijo y se usa el emisor nulo, de modo que la supervisión nunca impide
// ni retrasa el arranque. Devuelve también la función que vacía y cierra el
// emisor.
func crearEmisorServidor(destino, registro io.Writer) (ports.EmisorIncidenciasTecnicas, func()) {
	emisor, err := observabilidad.NuevoEmisorJSONLines(observabilidad.OpcionesEmisor{
		Destino:        destino,
		Entorno:        os.Getenv(envEntornoSupervision),
		VersionBinario: revisionCompilada(),
	})
	if err != nil {
		escribirRegistroFijo(registro, "vec-server: emisor de incidencias tecnicas no disponible\n")
		return ports.EmisorIncidenciasTecnicasNulo{}, func() {}
	}
	return emisor, func() {
		ctx, cancelar := context.WithTimeout(context.Background(), plazoCierreSupervision)
		defer cancelar()
		if emisor.Cerrar(ctx) != nil {
			escribirRegistroFijo(registro, "vec-server: incidencias tecnicas pendientes sin vaciar al cerrar\n")
		}
	}
}

// componerSupervisionServidor inyecta el emisor en el middleware común de
// respuestas 5xx y pánicos y sanea el ErrorLog de net/http hacia registro.
// Devuelve la función idempotente que vuelca el último grupo del ErrorLog y
// después vacía y cierra el emisor; se registra también en el cierre
// ordenado del servidor.
func componerSupervisionServidor(srv *http.Server, emisor ports.EmisorIncidenciasTecnicas, cerrarEmisor func(), registro io.Writer) func() {
	volcarErrorLog := server.SupervisarServidor(srv, emisor, registro)
	var una sync.Once
	cerrar := func() {
		una.Do(func() {
			volcarErrorLog()
			if cerrarEmisor != nil {
				cerrarEmisor()
			}
		})
	}
	if srv != nil {
		srv.RegisterOnShutdown(cerrar)
	}
	return cerrar
}

// escribirRegistroFijo es el último destino de un mensaje fijo de la propia
// supervisión: si también falla no queda otro canal al que informar.
func escribirRegistroFijo(registro io.Writer, mensaje string) {
	if registro != nil {
		_, _ = io.WriteString(registro, mensaje)
	}
}
