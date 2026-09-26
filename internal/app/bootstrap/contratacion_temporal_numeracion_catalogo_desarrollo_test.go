package bootstrap

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"vec-diputacion-granada/internal/vec/reglas"
)

func reglaNumeracionPrueba(prefijo, digitos string) map[string]reglas.Regla {
	return map[string]reglas.Regla{reglaNumeracionCT: {
		Clave: reglaNumeracionCT, Unidad: reglas.UnidadNinguna, Referencia: "vec.contratacion_temporal.reglas:1:c16.numeracion",
		Atributos: map[string]string{atributoPrefijoNumeracion: prefijo, atributoDigitosNumeracion: digitos},
	}}
}

func TestNumeracionSinCatalogoEsLaDeSiempre(t *testing.T) {
	opciones, err := nuevasOpcionesAnalisisCT(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if numeracion := opciones.numeracionVigente(); numeracion.Prefijo != "CT-" || numeracion.Digitos != 6 {
		t.Fatalf("numeración sin catálogo inesperada: %+v", numeracion)
	}
	if numeracion := (*opcionesAnalisisCTDesarrollo)(nil).numeracionVigente(); !numeracion.valida() {
		t.Fatalf("numeración predeterminada no válida: %+v", numeracion)
	}
	delEjemplo, err := nuevasOpcionesAnalisisCT(t.Context(), resolutorReglasCTPrueba(t, rutaReglasCTEjemploPrueba))
	if err != nil {
		t.Fatal(err)
	}
	numeracion := delEjemplo.numeracionVigente()
	if numeracion.Prefijo != "CT-" || numeracion.Digitos != 6 ||
		!strings.HasSuffix(numeracion.FuenteRef, ":c16.numeracion") {
		t.Fatalf("el paquete de ejemplo cambia la numeración de siempre: %+v", numeracion)
	}
}

func TestNumeracionSeCambiaEnElCatalogo(t *testing.T) {
	ruta := catalogoReglasCTModificadoPrueba(t, func(entradas []map[string]any) []map[string]any {
		for _, entrada := range entradas {
			if entrada["clave"] == reglaNumeracionCT {
				atributos := entrada["atributos"].(map[string]any)
				atributos[atributoPrefijoNumeracion] = "CTEMP-"
				atributos[atributoDigitosNumeracion] = "4"
			}
		}
		return entradas
	})
	opciones, err := nuevasOpcionesAnalisisCT(t.Context(), resolutorReglasCTPrueba(t, ruta))
	if err != nil {
		t.Fatal(err)
	}
	if numeracion := opciones.numeracionVigente(); numeracion.Prefijo != "CTEMP-" || numeracion.Digitos != 4 {
		t.Fatalf("numeración del catálogo no aplicada: %+v", numeracion)
	}
	vacio, err := numeracionDesdeReglasCT(reglaNumeracionPrueba("", "5"))
	if err != nil || vacio.Prefijo != "" {
		t.Fatalf("un prefijo vacío es válido: %+v %v", vacio, err)
	}
}

func TestNumeracionRechazaCatalogoRoto(t *testing.T) {
	for _, caso := range [][2]string{
		{"CT/", "6"}, {"CT-", "0"}, {"CT-", "10"}, {"CT-", "06"}, {"CT-", "seis"},
		{strings.Repeat("A", 21), "6"}, {"C T", "6"},
	} {
		if _, err := numeracionDesdeReglasCT(reglaNumeracionPrueba(caso[0], caso[1])); !errors.Is(err, errNumeracionNoValida) {
			t.Fatalf("se aceptó la numeración %q/%q", caso[0], caso[1])
		}
	}
	sinPrefijo := reglaNumeracionPrueba("CT-", "6")
	regla := sinPrefijo[reglaNumeracionCT]
	delete(regla.Atributos, atributoPrefijoNumeracion)
	if _, err := numeracionDesdeReglasCT(sinPrefijo); !errors.Is(err, errNumeracionNoValida) {
		t.Fatal("una numeración sin prefijo declarado debía rechazarse")
	}
	otraUnidad := reglaNumeracionPrueba("CT-", "6")
	regla = otraUnidad[reglaNumeracionCT]
	regla.Unidad = reglas.UnidadLista
	otraUnidad[reglaNumeracionCT] = regla
	if _, err := numeracionDesdeReglasCT(otraUnidad); !errors.Is(err, errNumeracionNoValida) {
		t.Fatal("una numeración con otra unidad debía rechazarse")
	}
}

// consultaNumeracionPrueba responde a la comprobación de CT-000126 y a la
// publicación, en ese orden.
type consultaNumeracionPrueba struct {
	instalada bool
	resultado string
	errores   []error
	llamadas  *[]string
}

func (c consultaNumeracionPrueba) QueryRow(_ context.Context, sql string, argumentos ...any) pgx.Row {
	*c.llamadas = append(*c.llamadas, sql)
	var err error
	if indice := len(*c.llamadas) - 1; indice < len(c.errores) {
		err = c.errores[indice]
	}
	return filaNumeracionPrueba{consulta: c, err: err, publicar: strings.Contains(sql, "publicar_numeracion_parametros_v1($1")}
}

type filaNumeracionPrueba struct {
	consulta consultaNumeracionPrueba
	err      error
	publicar bool
}

func (f filaNumeracionPrueba) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	if !f.publicar {
		*(destinos[0].(*bool)) = f.consulta.instalada
		return nil
	}
	*(destinos[0].(*string)) = f.consulta.resultado
	*(destinos[1].(*int64)) = 1
	return nil
}

func TestPublicacionNumeracionExigeLaMigracion(t *testing.T) {
	predeterminada := numeracionExpedientesPredeterminadaCT()
	for _, resultado := range []string{"publicada", "vigente"} {
		var llamadas []string
		consulta := consultaNumeracionPrueba{instalada: true, resultado: resultado, llamadas: &llamadas}
		if err := publicarNumeracionExpedientesCT(t.Context(), consulta, predeterminada); err != nil || len(llamadas) != 2 {
			t.Fatalf("%s: %v (%d llamadas)", resultado, err, len(llamadas))
		}
	}
	var llamadas []string
	sinMigracion := consultaNumeracionPrueba{llamadas: &llamadas}
	if err := publicarNumeracionExpedientesCT(t.Context(), sinMigracion, predeterminada); !errors.Is(err, errMigracionNumeracionNoInstalada) ||
		codigoFalloNumeracionExpedientesCT(err) != "migracion_ct126_no_instalada" {
		t.Fatalf("arrancaría sin CT-000126: %v", err)
	}
	llamadas = nil
	rechazada := consultaNumeracionPrueba{instalada: true, llamadas: &llamadas, errores: []error{nil, errors.New("22023")}}
	if err := publicarNumeracionExpedientesCT(t.Context(), rechazada, predeterminada); !errors.Is(err, errNumeracionNoPublicada) {
		t.Fatalf("un rechazo de la publicación no detuvo el arranque: %v", err)
	}
	llamadas = nil
	extraña := consultaNumeracionPrueba{instalada: true, resultado: "otra", llamadas: &llamadas}
	if err := publicarNumeracionExpedientesCT(t.Context(), extraña, predeterminada); !errors.Is(err, errNumeracionNoPublicada) {
		t.Fatalf("un resultado desconocido se dio por bueno: %v", err)
	}
	if err := publicarNumeracionExpedientesCT(t.Context(), extraña, numeracionExpedientesCT{Prefijo: "CT-"}); !errors.Is(err, errNumeracionNoPublicada) {
		t.Fatal("se publicó una numeración no válida")
	}
}
