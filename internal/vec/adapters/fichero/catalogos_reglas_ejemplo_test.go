package fichero

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

// Los paquetes de reglas de ejemplo solo existen para desarrollo y la
// presentación a RRHH. Esta prueba los carga con el adaptador real y fija el
// contrato mínimo de atributos que consume el resolutor de reglas.
func TestPaquetesReglasEjemploCarganConAdaptadorReal(t *testing.T) {
	casos := []struct {
		ruta, id, modulo string
		minimoEntradas   int
		motivos          bool
	}{
		{"../../../../data/demo/reglas/bolsa_reglas.ejemplo.demo.json", "vec.bolsa.reglas", "bolsa", 26, false},
		{"../../../../data/demo/reglas/ct_reglas.ejemplo.demo.json", "vec.contratacion_temporal.reglas", "contratacion_temporal", 8, false},
		{"../../../../data/demo/reglas/ct_circuito_firma.ejemplo.demo.json", "vec.contratacion_temporal.circuito_firma", "contratacion_temporal", 11, false},
		{"../../../../data/demo/reglas/ct_motivos_rectificacion.demo.json", "motivos_rectificacion_analisis", "contratacion_temporal", 6, true},
	}
	for _, caso := range casos {
		t.Run(caso.id, func(t *testing.T) {
			consulta, err := NuevaConsultaCatalogos(caso.ruta)
			if err != nil {
				t.Fatalf("el adaptador rechaza el paquete: %v", err)
			}
			metadatos, err := consulta.ObtenerMetadatosFuenteCatalogos(context.Background())
			if err != nil || !metadatos.Demostracion {
				t.Fatalf("el paquete debe declararse de demostración: %+v %v", metadatos, err)
			}
			catalogo, err := consulta.ObtenerCatalogo(context.Background(), caso.id, 1)
			if err != nil {
				t.Fatal(err)
			}
			if catalogo.ModuloID != caso.modulo || catalogo.FuenteRef != "paquete:ejemplo:vec:v1" ||
				catalogo.Estado != domain.EstadoCatalogoPublicado || len(catalogo.Entradas) < caso.minimoEntradas {
				t.Fatalf("catálogo inesperado: modulo=%s fuente=%s estado=%s entradas=%d",
					catalogo.ModuloID, catalogo.FuenteRef, catalogo.Estado, len(catalogo.Entradas))
			}
			instante := time.Date(2026, 9, 28, 9, 0, 0, 0, time.UTC)
			for _, entrada := range catalogo.Entradas {
				if !entrada.VigenteEn(instante) {
					t.Errorf("%s no está vigente el día de la presentación", entrada.Clave)
				}
				if caso.motivos {
					comprobarMotivoEjemplo(t, entrada)
					continue
				}
				comprobarReglaEjemplo(t, entrada)
			}
		})
	}
}

func comprobarMotivoEjemplo(t *testing.T, entrada domain.EntradaCatalogoConfigurable) {
	t.Helper()
	if entrada.Atributos["clave_i18n"] != "contratacion_temporal.analisis.rectificacion."+entrada.Clave ||
		entrada.Atributos["origen"] != "ejemplo" {
		t.Errorf("motivo %s sin clave i18n o sin marca de ejemplo", entrada.Clave)
	}
}

func comprobarReglaEjemplo(t *testing.T, entrada domain.EntradaCatalogoConfigurable) {
	t.Helper()
	a := entrada.Atributos
	for _, obligatorio := range []string{"origen", "norma", "duda", "unidad"} {
		if strings.TrimSpace(a[obligatorio]) == "" {
			t.Errorf("%s sin atributo %s", entrada.Clave, obligatorio)
		}
	}
	switch a["origen"] {
	case "reglamento":
		if a["articulo"] == "" {
			t.Errorf("%s de origen reglamento sin artículo", entrada.Clave)
		}
	case "ejemplo":
		if _, existe := a["articulo"]; existe {
			t.Errorf("%s de ejemplo no puede citar un artículo del Reglamento", entrada.Clave)
		}
	default:
		t.Errorf("%s con origen desconocido %q", entrada.Clave, a["origen"])
	}
	cantidad, conCantidad := a["cantidad"]
	switch a["unidad"] {
	case "dias_habiles", "dias_naturales", "meses", "anios":
		if n, err := strconv.Atoi(cantidad); err != nil || n < 1 {
			t.Errorf("%s: plazo sin cantidad positiva", entrada.Clave)
		}
		if conInicio := a["inicio"] != ""; conInicio != (a["computo"] == "administrativo" || a["computo"] == "civil") {
			t.Errorf("%s: inicio y cómputo deben declararse juntos", entrada.Clave)
		}
	case "intentos", "procesos", "horas", "minutos_semanales":
		if n, err := strconv.Atoi(cantidad); err != nil || n < 1 {
			t.Errorf("%s: sin cantidad positiva", entrada.Clave)
		}
	case "franja_horaria", "lista":
		if a["valor"] == "" || conCantidad {
			t.Errorf("%s: %s exige valor y no cantidad", entrada.Clave, a["unidad"])
		}
	case "ninguna":
		if conCantidad {
			t.Errorf("%s: sin unidad no admite cantidad", entrada.Clave)
		}
	default:
		t.Errorf("%s: unidad desconocida %q", entrada.Clave, a["unidad"])
	}
}
