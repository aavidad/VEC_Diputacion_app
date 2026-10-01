package main

import (
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"vec-diputacion-granada/internal/modules/provision/adapters/simulacion"
	"vec-diputacion-granada/internal/modules/provision/application"
	"vec-diputacion-granada/internal/modules/provision/domain"
	"vec-diputacion-granada/internal/modules/provision/ports"
)

func main() { os.Exit(ejecutar(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)) }

// La petición estricta entra por stdin. No hay rutas de archivo, red ni base.
func ejecutar(args []string, entrada io.Reader, salida, diagnostico io.Writer) int {
	flags := flag.NewFlagSet("vec-ensayar-provision", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	ejemplo := flags.Bool("ejemplo", false, "")
	emitirEntrada := flags.Bool("emitir-entrada", false, "")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 || (*emitirEntrada && !*ejemplo) {
		return fallar(diagnostico, "argumentos_invalidos", "ciclo")
	}
	p, err := obtenerPeticion(*ejemplo, entrada)
	if err != nil {
		return fallarNominal(diagnostico, err)
	}
	if *emitirEntrada {
		return emitir(salida, diagnostico, p)
	}
	ciclo, err := application.EnsayarCiclo(p)
	if err != nil {
		return fallarNominal(diagnostico, err)
	}
	return emitir(salida, diagnostico, ciclo)
}

func obtenerPeticion(ejemplo bool, entrada io.Reader) (ports.PeticionCiclo, error) {
	if ejemplo {
		return simulacion.EjemploCiclo()
	}
	return simulacion.DecodificarCiclo(entrada)
}

func emitir(salida, diagnostico io.Writer, v any) int {
	if err := json.NewEncoder(salida).Encode(v); err != nil {
		return fallar(diagnostico, "salida_fallida", "ciclo")
	}
	return 0
}

func fallarNominal(w io.Writer, err error) int {
	var nominal *domain.Error
	if errors.As(err, &nominal) {
		// El motor puede usar un ID aportado como Campo. El diagnóstico del
		// transporte conserva sólo el código nominal y un campo fijo.
		return fallar(w, nominal.Codigo, "ciclo")
	}
	return fallar(w, "ensayo_fallido", "ciclo")
}

func fallar(w io.Writer, codigo, campo string) int {
	_ = json.NewEncoder(w).Encode(domain.Error{Codigo: codigo, Campo: campo})
	return 2
}
