package main

import (
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"syscall"

	"vec-diputacion-granada/internal/modules/bolsa/application/simulacionbaremo"
)

const maximoBytesArchivo = 16 * 1024 * 1024

func main() { os.Exit(ejecutar(os.Args[1:], os.Stdout, os.Stderr, simulacionbaremo.Servicio{})) }

func ejecutar(args []string, salida, diagnostico io.Writer, servicio simulacionbaremo.Simulador) int {
	flags := flag.NewFlagSet("vec-baremador", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	reglas := flags.String("reglas", "", "")
	huellaReglas := flags.String("reglas-sha256", "", "")
	entrada := flags.String("entrada", "", "")
	huellaEntrada := flags.String("entrada-sha256", "", "")
	limite := flags.Int64("limite-bytes", maximoBytesArchivo, "")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return diagnosticar(diagnostico, "uso", "reglas,reglas-sha256,entrada,entrada-sha256,limite-bytes", 0)
		}
		return diagnosticar(diagnostico, "argumentos_invalidos", "", 2)
	}
	if flags.NArg() != 0 || *reglas == "" || *entrada == "" || *huellaReglas == "" || *huellaEntrada == "" || *limite <= 0 || *limite > maximoBytesArchivo {
		return diagnosticar(diagnostico, "argumentos_invalidos", "", 2)
	}
	contenidoReglas, err := leerArchivoLimitado(*reglas, *limite)
	if err != nil {
		return diagnosticar(diagnostico, "archivo_invalido", "reglas", 2)
	}
	contenidoEntrada, err := leerArchivoLimitado(*entrada, *limite)
	if err != nil {
		return diagnosticar(diagnostico, "archivo_invalido", "entrada", 2)
	}
	simulacion, err := servicio.Simular(simulacionbaremo.Solicitud{
		ConjuntoCanonico: contenidoReglas, HuellaConjuntoSHA256: *huellaReglas,
		EntradaCanonica: contenidoEntrada, HuellaEntradaSHA256: *huellaEntrada,
	})
	if err != nil {
		fase := "simulacion"
		var fallo *simulacionbaremo.Error
		if errors.As(err, &fallo) {
			fase = fallo.Fase
		}
		return diagnosticar(diagnostico, "simulacion_fallida", fase, 2)
	}
	contenido := simulacion.RepresentacionCanonica()
	if len(contenido) == 0 {
		return diagnosticar(diagnostico, "simulacion_fallida", "resultado", 2)
	}
	n, err := salida.Write(contenido)
	if err != nil || n != len(contenido) {
		return diagnosticar(diagnostico, "salida_fallida", "", 2)
	}
	return 0
}

// O_NONBLOCK impide quedar esperando al abrir un FIFO. Solo se admiten
// archivos regulares y la lectura tiene límite incluso si cambian de tamaño.
func leerArchivoLimitado(ruta string, limite int64) (contenido []byte, err error) {
	if limite <= 0 || limite > maximoBytesArchivo {
		return nil, errors.New("limite_invalido")
	}
	// La ruta la elige el operador local y se abre con sus permisos del SO.
	// No procede de HTTP ni hay un directorio de recursos autorizado implícito.
	fichero, err := os.OpenFile(ruta, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cierre := fichero.Close(); cierre != nil && err == nil {
			contenido = nil
			err = cierre
		}
	}()
	estado, err := fichero.Stat()
	if err != nil {
		return nil, err
	}
	if !estado.Mode().IsRegular() || estado.Size() <= 0 || estado.Size() > limite {
		return nil, errors.New("archivo_invalido")
	}
	contenido, err = io.ReadAll(io.LimitReader(fichero, limite+1))
	if err != nil {
		return nil, err
	}
	if len(contenido) == 0 || int64(len(contenido)) > limite {
		return nil, errors.New("archivo_invalido")
	}
	return contenido, nil
}

func diagnosticar(salida io.Writer, codigo, fase string, retorno int) int {
	contenido, err := json.Marshal(struct {
		Codigo string `json:"codigo"`
		Fase   string `json:"fase,omitempty"`
	}{codigo, fase})
	if err != nil {
		return 2
	}
	n, err := salida.Write(contenido)
	if err != nil || n != len(contenido) {
		return 2
	}
	return retorno
}
