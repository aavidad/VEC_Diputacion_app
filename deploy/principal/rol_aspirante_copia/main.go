// Command rol_aspirante_copia prepara, sin red ni base de datos, el material
// de los dos actos que publican en una copia local la versión del rol del
// aspirante: el plan de admisión del catálogo con las fuentes nuevas y el
// cuerpo de cada propuesta con la huella del plan calculada por el dominio.
//
// Lee un JSON por la entrada estándar y escribe otro por la salida. No conoce
// claves, certificados ni DSN: el guion de la copia le pasa las lecturas de
// solo lectura y presenta el resultado por las vías gobernadas (AUT58 y las
// rutas de propuesta y cierre de vec-admin).
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

const maximoEntrada = 32 << 20

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "uso: rol_aspirante_copia catalogo|propuesta < entrada.json")
		os.Exit(2)
	}
	salida, err := ejecutar(os.Args[1], os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "rol_aspirante_copia:", err)
		os.Exit(1)
	}
	if _, err := os.Stdout.Write(append(salida, '\n')); err != nil {
		os.Exit(1)
	}
}

func ejecutar(orden string, entrada io.Reader) ([]byte, error) {
	datos, err := io.ReadAll(io.LimitReader(entrada, maximoEntrada+1))
	if err != nil || len(datos) > maximoEntrada {
		return nil, errors.New("entrada ilegible o excesiva")
	}
	switch orden {
	case "catalogo":
		var e entradaCatalogo
		if err := decodificarExacto(datos, &e); err != nil {
			return nil, err
		}
		s, err := prepararCatalogo(e)
		if err != nil {
			return nil, err
		}
		return json.Marshal(s)
	case "propuesta":
		var e entradaPropuesta
		if err := decodificarExacto(datos, &e); err != nil {
			return nil, err
		}
		s, err := prepararPropuesta(e)
		if err != nil {
			return nil, err
		}
		return json.Marshal(s)
	}
	return nil, errors.New("orden desconocida")
}

func decodificarExacto(datos []byte, destino any) error {
	dec := json.NewDecoder(bytes.NewReader(datos))
	dec.DisallowUnknownFields()
	if err := dec.Decode(destino); err != nil {
		return fmt.Errorf("JSON inválido: %w", err)
	}
	if err := dec.Decode(new(any)); !errors.Is(err, io.EOF) {
		return errors.New("JSON con datos sobrantes")
	}
	return nil
}
