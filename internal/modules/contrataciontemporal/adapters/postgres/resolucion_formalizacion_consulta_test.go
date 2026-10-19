package postgres

import (
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// La preparación de la resolución lee de CT69 exactamente las columnas que esa
// función devuelve y en el orden de sus destinos: el cierre del detalle y la
// preparación, sin la cifra de personas de CT225.
func TestPreparacionResolucionLeeColumnasDeCT69(t *testing.T) {
	columnas, funcion := columnasSelectConsultaRRHH(t, consultaPreparacionResolucionRRHHPostgreSQL)
	if funcion != "vec_contratacion_temporal.consultar_preparacion_resolucion_v1" {
		t.Fatalf("lee de %q", funcion)
	}
	if len(columnas) != len(destinosDetalleConsultaRRHH(&salidaDetalleConsultaRRHH{}))+1 ||
		columnas[len(columnas)-1] != "preparacion::text" {
		t.Fatalf("columnas = %v", columnas)
	}
	b, err := os.ReadFile("../../../../../deploy/postgresql/contratacion_temporal/migraciones/000069_resolucion_formalizacion_manual_ejercicio.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	up := string(b)
	i := strings.Index(up, "CREATE FUNCTION vec_contratacion_temporal.consultar_preparacion_resolucion_v1(")
	if i < 0 {
		t.Fatal("sin preparación en CT69")
	}
	tabla := regexp.MustCompile(`(?s)RETURNS TABLE\s*\((.*?)\)\s*LANGUAGE`).FindStringSubmatch(up[i:])
	if tabla == nil {
		t.Fatal("CT69 sin RETURNS TABLE reconocible")
	}
	var devueltas []string
	for _, c := range strings.Split(tabla[1], ",") {
		devueltas = append(devueltas, strings.Fields(c)[0])
	}
	var pedidas []string
	for _, c := range columnas {
		pedidas = append(pedidas, strings.TrimSuffix(c, "::text"))
	}
	if !reflect.DeepEqual(pedidas, devueltas) {
		t.Fatalf("columnas pedidas %v, CT69 devuelve %v", pedidas, devueltas)
	}
	if n := strings.Count(consultaPreparacionResolucionRRHHPostgreSQL, "$"); n !=
		len(argumentosSQLDetalleConsultaRRHH("", "", "", ports.SolicitudDetalleRRHH{}, argumentosMaterialConsultaRRHH{})) {
		t.Fatalf("parámetros = %d", n)
	}
}
