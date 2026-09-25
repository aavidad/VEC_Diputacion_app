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

// componerSupervisionServidor crea el emisor de incidencias técnicas del
// servidor (JSON Lines hacia destino, recogido fuera del proceso), lo inyecta
// en el middleware común de respuestas 5xx y pánicos y sanea el ErrorLog de
// net/http hacia registro. Si el emisor no puede crearse, el servidor conserva
// la contención y el saneamiento y el fallo queda en registro con texto fijo:
// la supervisión nunca impide ni retrasa el arranque. Devuelve la función
// idempotente que vacía y cierra el emisor.
func componerSupervisionServidor(srv *http.Server, destino, registro io.Writer) func() {
	emisor, err := observabilidad.NuevoEmisorJSONLines(observabilidad.OpcionesEmisor{
		Destino:        destino,
		Entorno:        os.Getenv(envEntornoSupervision),
		VersionBinario: revisionCompilada(),
	})
	if err != nil {
		escribirRegistroFijo(registro, "vec-server: emisor de incidencias tecnicas no disponible\n")
		server.SupervisarServidor(srv, nil, registro)
		return func() {}
	}
	server.SupervisarServidor(srv, emisor, registro)
	var una sync.Once
	cerrar := func() {
		una.Do(func() {
			ctx, cancelar := context.WithTimeout(context.Background(), plazoCierreSupervision)
			defer cancelar()
			if emisor.Cerrar(ctx) != nil {
				escribirRegistroFijo(registro, "vec-server: incidencias tecnicas pendientes sin vaciar al cerrar\n")
			}
		})
	}
	srv.RegisterOnShutdown(cerrar)
	return cerrar
}

// escribirRegistroFijo es el último destino de un mensaje fijo de la propia
// supervisión: si también falla no queda otro canal al que informar.
func escribirRegistroFijo(registro io.Writer, mensaje string) {
	if registro != nil {
		_, _ = io.WriteString(registro, mensaje)
	}
}
