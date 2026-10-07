package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/seleccion/application"
	"vec-diputacion-granada/internal/modules/seleccion/domain"
)

const maximoEntrada = 1024 * 1024

func ejecutar(ctx context.Context, args []string, entrada io.Reader, salida, diagnostico io.Writer) int {
	if len(args) != 0 || entrada == nil || salida == nil {
		return informar(diagnostico, "seleccion.calificaciones_ejercicio.entrada_invalida")
	}
	datos, err := io.ReadAll(io.LimitReader(entrada, maximoEntrada+1))
	if err != nil || len(datos) == 0 || len(datos) > maximoEntrada || !utf8.Valid(datos) {
		return informar(diagnostico, "seleccion.calificaciones_ejercicio.entrada_invalida")
	}
	var material domain.MaterialCalificacionesEjercicio
	if decodificar(datos, &material) != nil {
		return informar(diagnostico, "seleccion.calificaciones_ejercicio.entrada_invalida")
	}
	registro, err := application.PrepararMaterialCalificacionesEjercicio(ctx, material)
	if err != nil {
		return informar(diagnostico, "seleccion.calificaciones_ejercicio.material_invalido")
	}
	if json.NewEncoder(salida).Encode(registro) != nil {
		return informar(diagnostico, "seleccion.calificaciones_ejercicio.salida_no_disponible")
	}
	return 0
}

func decodificar(datos []byte, v any) error {
	d := json.NewDecoder(bytes.NewReader(datos))
	if err := clavesUnicas(d, 0); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return domain.ErrCalificacionesEjercicio
	}
	d = json.NewDecoder(bytes.NewReader(datos))
	d.DisallowUnknownFields()
	return d.Decode(v)
}

func clavesUnicas(d *json.Decoder, nivel int) error {
	if nivel > 32 {
		return domain.ErrCalificacionesEjercicio
	}
	t, err := d.Token()
	if err != nil {
		return domain.ErrCalificacionesEjercicio
	}
	inicio, ok := t.(json.Delim)
	if !ok {
		return nil
	}
	if inicio != '{' && inicio != '[' {
		return domain.ErrCalificacionesEjercicio
	}
	vistas := map[string]bool{}
	for d.More() {
		if inicio == '{' {
			k, err := d.Token()
			clave, ok := k.(string)
			if err != nil || !ok || clave == "" || strings.Trim(clave, "abcdefghijklmnopqrstuvwxyz_0123456789") != "" || vistas[clave] {
				return domain.ErrCalificacionesEjercicio
			}
			vistas[clave] = true
		}
		if err := clavesUnicas(d, nivel+1); err != nil {
			return err
		}
	}
	fin, err := d.Token()
	if err != nil || inicio == '{' && fin != json.Delim('}') || inicio == '[' && fin != json.Delim(']') {
		return domain.ErrCalificacionesEjercicio
	}
	return nil
}

func informar(w io.Writer, clave string) int {
	if w == nil || json.NewEncoder(w).Encode(struct {
		Error string `json:"error"`
	}{clave}) != nil {
		return 2
	}
	return 1
}

func main() {
	ctx, cancelar := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelar()
	os.Exit(ejecutar(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
