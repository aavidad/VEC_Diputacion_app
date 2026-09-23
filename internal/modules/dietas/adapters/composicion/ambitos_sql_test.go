package composicion

import (
	"os"
	"reflect"
	"regexp"
	"sort"
	"testing"

	dietasapp "vec-diputacion-granada/internal/modules/dietas/application"
	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
)

// El SQL de Dietas recalcula contexto_recurso_huella_sha256 con los mismos
// ámbitos que el RecursoAutorizable de Go. Si divergen, toda operación real se
// deniega (las pruebas SQL con AD3 simulado no lo ven): se fija aquí.
func TestAmbitosDelRecursoCoincidenConLaHuellaSQL(t *testing.T) {
	base, _ := identidadYMaterialR15(t)
	relacion := relacionR15(base.Contexto.Contexto.PersonaRef, "a")
	sello := selloRelacion(relacion, base.FechaReferencia)
	s := dietasports.SolicitudOperacionBorrador{Operacion: dietasports.OperacionCrearBorrador, Crear: dietasports.SolicitudCrearBorradorPropio{ClaveIdempotencia: "clave_0123456789abcdef", FechaInicio: "2026-09-21", FechaFin: "2026-09-21", Motivo: "Visita técnica", CodigosRuta: []string{}}}
	efecto, err := dietasapp.ConstruirEfectoAutorizacionBorrador(base.Contexto, traducirRelacion(relacion), sello, s)
	if err != nil {
		t.Fatal(err)
	}
	var enGo []string
	for clave := range efecto.Recurso.Ambitos {
		enGo = append(enGo, clave)
	}
	sort.Strings(enGo)

	sql, err := os.ReadFile("../../../../../deploy/postgresql/dietas_borradores/migraciones/000001_borrador_comision_durable.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	linea := regexp.MustCompile(`(?m)^\s*amb:=.*$`).FindAll(sql, -1)
	if len(linea) != 1 {
		t.Fatalf("se esperaba una construcción amb:= en 000001, hay %d", len(linea))
	}
	var enSQL []string
	for _, m := range regexp.MustCompile(`"(\w+)":'\|\|\(i->'(\w+)'\)`).FindAllSubmatch(linea[0], -1) {
		if string(m[1]) != string(m[2]) {
			t.Fatalf("clave %s toma el valor de %s", m[1], m[2])
		}
		enSQL = append(enSQL, string(m[1]))
	}
	if !reflect.DeepEqual(enGo, enSQL) {
		t.Fatalf("ámbitos Go %v y SQL %v difieren", enGo, enSQL)
	}
}
