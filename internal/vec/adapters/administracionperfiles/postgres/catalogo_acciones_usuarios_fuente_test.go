package postgres

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

func TestFuenteUsuariosSoloPreferenciasEImagenPropias(t *testing.T) {
	ruta := filepath.Join("..", "..", "..", "..", "..", "deploy", "principal", "fuentes", "usuarios_preferencias_imagen_propias_v1.json")
	b, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	var fuente fuentePaqueteCatalogoAccionesV2
	if err := json.Unmarshal(b, &fuente); err != nil || fuente.ModuloID != "usuarios" || len(fuente.Entradas) != 4 {
		t.Fatal("fuente de Usuarios incompleta", err)
	}
	canon, huella, err := canonEntradasFuenteV2(fuente.Entradas)
	if err != nil || string(canon) != fuente.EntradasCanon || huella != fuente.HuellaSHA256 {
		t.Fatal("fuente de Usuarios divergente", err)
	}
	acciones := []string{"vec.preferencias.consultar", "vec.preferencias.actualizar", "vec.imagen.consultar", "vec.imagen.actualizar"}
	for i, e := range fuente.Entradas {
		tipo, finalidad := "preferencias_persona", "finalidad:usuarios:preferencias-propias:v1"
		campos := []string{"catalogo", "valores", "version"}
		if i == 1 {
			campos = []string{"valores", "version"}
		}
		if i >= 2 {
			tipo, finalidad = "imagen_persona", "finalidad:usuarios:imagen-propia:v1"
			campos = []string{"foto", "icono", "modo", "paleta", "version"}
		}
		c := e.Concesion
		if e.FuenteRef != fuente.Referencia || e.FuenteVersion != fuente.Version || e.FuenteHuellaSHA256 != huella ||
			c.Accion != acciones[i] || c.ModuloID != "usuarios" || c.TipoRecurso != tipo || c.GarantiaMinima != domain.AuthAssuranceHigh ||
			!reflect.DeepEqual(c.Finalidades, []string{finalidad}) || !reflect.DeepEqual(c.CamposPermitidos, campos) || len(c.Obligaciones) != 0 ||
			!reflect.DeepEqual(e.DimensionesAmbito, []string{"persona_ref"}) || e.ClaseControl != string(domain.ClaseControlPerfilOrdinario) ||
			e.VigenteHasta.Sub(e.VigenteDesde) != 90*24*time.Hour {
			t.Fatalf("entrada %d de Usuarios amplía su contrato propio", i)
		}
	}
}
