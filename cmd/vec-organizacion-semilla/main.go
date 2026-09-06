// vec-organizacion-semilla prepara SQL de inicialización; no abre conexiones.
package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/vec/adapters/fichero"
)

func main() {
	archivo := flag.String("archivo", "", "paquete de organización preparatoria validado; no datos personales")
	flag.Parse()
	if *archivo == "" || flag.NArg() != 0 {
		flag.Usage()
		os.Exit(2)
	}
	if err := prepararSemilla(os.Stdout, *archivo); err != nil {
		fmt.Fprintln(os.Stderr, "No se pudo validar la semilla de organización; no se ha escrito en la base de datos.")
		os.Exit(1)
	}
}

func prepararSemilla(salida io.Writer, archivo string) error {
	lector, err := fichero.NuevaConsultaCatalogos(archivo)
	if err != nil {
		return err
	}
	catalogo, err := lector.ObtenerCatalogo(context.Background(), "estructura-organizativa-dipgra", 1)
	if err != nil {
		return err
	}
	if catalogo.Revision != 1 || catalogo.Estado != "borrador" {
		return fmt.Errorf("semilla no inicial")
	}
	if err := personaldomain.ValidarEstructuraOrganizativa(catalogo); err != nil {
		return err
	}
	canonico, err := catalogo.ClonarCanonico()
	if err != nil {
		return err
	}
	b, err := json.Marshal(canonico)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(salida, "\\set ON_ERROR_STOP on\nBEGIN;\nSET LOCAL ROLE vec_contratacion_temporal_propietario;\nSELECT vec_contratacion_temporal.inicializar_organizacion_preparatoria_v1(convert_from(decode('%s','base64'),'UTF8'));\nCOMMIT;\n", base64.StdEncoding.EncodeToString(b))
	return err
}
