package main

import (
	"errors"
	"io"
	"os"

	"vec-diputacion-granada/internal/modules/dietas/adapters/informeliquidacion"
	"vec-diputacion-granada/internal/modules/dietas/application/preparacionliquidacion"
)

var errArchivoInforme = errors.New("catalogo_informe_no_disponible")

// Las dos rutas las proporciona el operador local. No se resuelven URLs ni
// se muestran nombres de archivo o su contenido en los errores.
func leerArchivoInforme(ruta string) ([]byte, error) {
	if ruta == "" {
		return nil, errArchivoInforme
	}
	f, err := os.Open(ruta)
	if err != nil {
		return nil, errArchivoInforme
	}
	const limite = 65536
	datos, err := io.ReadAll(io.LimitReader(f, limite+1))
	cierre := f.Close()
	if err != nil || cierre != nil || len(datos) > limite {
		return nil, errArchivoInforme
	}
	return datos, nil
}

func ejecutarInformePreparacion(in io.Reader, out io.Writer, rutaTextos, rutaTema string) int {
	entrada, err := leerEntradaTipada[preparacionliquidacion.Entrada](in)
	if err != nil {
		return escribirFalloPreparacion(out, err)
	}
	preparacion, err := preparacionliquidacion.Preparar(entrada)
	if err != nil {
		return escribirFalloPreparacion(out, err)
	}
	datos, err := leerArchivoInforme(rutaTextos)
	if err != nil {
		return escribirFalloPreparacion(out, err)
	}
	textos, err := informeliquidacion.CargarTextos(datos)
	if err != nil {
		return escribirFalloPreparacion(out, err)
	}
	tema, err := leerArchivoInforme(rutaTema)
	if err != nil {
		return escribirFalloPreparacion(out, err)
	}
	renderizador, err := informeliquidacion.Nuevo(informeliquidacion.Configuracion{TemaCSS: string(tema)})
	if err != nil {
		return escribirFalloPreparacion(out, err)
	}
	html, err := renderizador.Renderizar(preparacion, textos)
	if err != nil {
		return escribirFalloPreparacion(out, err)
	}
	escritos, err := out.Write(html)
	if err != nil || escritos != len(html) {
		return 1
	}
	return 0
}
