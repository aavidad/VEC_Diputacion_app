package httpinterno

import (
	"encoding/json"
	"strings"
	"testing"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestConsultaRRHHDetalleContratoProyectaObservaciones(t *testing.T) {
	t.Parallel()

	// Con observaciones
	analisis := ports.AnalisisOperativoRRHH{
		ModalidadClave: "sustitucion",
		Observaciones:  "Observaciones operativas del análisis de RRHH",
	}
	proyectado := proyectarAnalisisRRHH(analisis)
	if proyectado.Observaciones != analisis.Observaciones {
		t.Fatalf("esperado %q, obtenido %q", analisis.Observaciones, proyectado.Observaciones)
	}

	bytesJSON, err := json.Marshal(proyectado)
	if err != nil {
		t.Fatal(err)
	}
	var deserializado analisisRRHHJSON
	if err := json.Unmarshal(bytesJSON, &deserializado); err != nil {
		t.Fatal(err)
	}
	if deserializado.Observaciones != analisis.Observaciones {
		t.Fatalf("deserializado: esperado %q, obtenido %q", analisis.Observaciones, deserializado.Observaciones)
	}

	// Sin observaciones (debe omitirse o estar vacía)
	analisisVacio := ports.AnalisisOperativoRRHH{
		ModalidadClave: "sustitucion",
	}
	proyectadoVacio := proyectarAnalisisRRHH(analisisVacio)
	if proyectadoVacio.Observaciones != "" {
		t.Fatalf("esperado vacío, obtenido %q", proyectadoVacio.Observaciones)
	}
	bytesVacio, err := json.Marshal(proyectadoVacio)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(bytesVacio), "observaciones") {
		t.Fatalf("se esperaba omisión de observaciones vacías: %s", string(bytesVacio))
	}
}

// CT243: el detalle publica el grupo y la entrada de retención elegidos para
// precargar la rectificación; si faltan (versión antigua), no aparecen.
func TestConsultaRRHHDetalleContratoProyectaGrupoYEntradaRC(t *testing.T) {
	t.Parallel()
	con, err := json.Marshal(proyectarAnalisisRRHH(ports.AnalisisOperativoRRHH{
		ModalidadClave: "sustitucion", GrupoSubgrupo: "C1", EntradaRCRef: "rc:desarrollo:001",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(con), `"grupo_subgrupo":"C1"`) ||
		!strings.Contains(string(con), `"entrada_rc_ref":"rc:desarrollo:001"`) {
		t.Fatalf("faltan grupo o entrada: %s", con)
	}
	sin, err := json.Marshal(proyectarAnalisisRRHH(ports.AnalisisOperativoRRHH{ModalidadClave: "sustitucion"}))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(sin), "grupo_subgrupo") || strings.Contains(string(sin), "entrada_rc_ref") {
		t.Fatalf("se esperaba omisión: %s", sin)
	}
}
