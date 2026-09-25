package main

import (
	"context"
	"fmt"
	"io"

	"vec-diputacion-granada/config"
)

// ejecutarComprobacionDietas implementa «vec-server comprobar-dietas»: con el
// mismo entorno que usará el arranque, acredita en solo lectura las conexiones
// y la postimagen de Personal que exige Dietas, sin abrir el puerto HTTP. Así
// una activación se valida antes de reiniciar el servicio. Código 0 si todo
// está acreditado, 1 si no, 2 si los argumentos son inválidos. La salida nunca
// contiene DSN, contraseñas ni usuarios: solo la etapa que falló.
func ejecutarComprobacionDietas(ctx context.Context, argumentos []string, salida, errores io.Writer, cfg config.Config, comprobar func(context.Context, config.Config) error) int {
	if len(argumentos) != 0 || comprobar == nil {
		fmt.Fprintln(errores, "uso: vec-server comprobar-dietas (sin argumentos; usa el entorno de arranque)")
		return 2
	}
	if err := comprobar(ctx, cfg); err != nil {
		fmt.Fprintf(errores, "comprobar-dietas: NO: %v\n", err)
		return 1
	}
	fmt.Fprintln(salida, "comprobar-dietas: OK (solo lectura; vec-server puede arrancar con Dietas activado)")
	return 0
}
