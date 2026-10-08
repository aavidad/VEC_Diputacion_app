package domain

import (
	"strings"
	"testing"
)

func nota(v int64) *int64 { return &v }

func materialEjercicio() MaterialCalificacionesEjercicio {
	return MaterialCalificacionesEjercicio{
		Esquema: EsquemaCalificacionesEjercicio, ConvocatoriaRef: "convocatoria_sintetica", BasesVersion: 2,
		BasesHuellaSHA256: strings.Repeat("a", 64),
		Configuracion: Configuracion{Version: 2, Modalidad: "oposicion", TurnoAcceso: "libre", Destino: "plaza", Plazas: 1,
			Fases: []Fase{{Referencia: "ejercicio_1", Tipo: "prueba", MinimoMicropuntos: nota(5_000_000), MaximoMicropuntos: 10_000_000, Peso: 100}}, Desempates: []string{"ejercicio_1"}},
		EjercicioRef: "cuestionario_1", FaseRef: "ejercicio_1", Revision: 1,
		Notas: []NotaEjercicio{{SolicitudRef: "aspirante_b", PuntosMicropuntos: nil}, {SolicitudRef: "aspirante_a", PuntosMicropuntos: nota(7_000_000), FuenteRef: "correccion_a"}},
	}
}

func TestPrepararCalificacionesEjercicioConservaPendienteYHuella(t *testing.T) {
	m := materialEjercicio()
	r, err := PrepararCalificacionesEjercicio(m)
	if err != nil {
		t.Fatal(err)
	}
	if r.Aprobada || r.Publicada || r.Notas[0].SolicitudRef != "aspirante_a" || r.Notas[1].Estado != "pendiente" || len(r.HuellaMaterialSHA256) != 64 {
		t.Fatalf("resultado inesperado: %+v", r)
	}
	m.Notas[0], m.Notas[1] = m.Notas[1], m.Notas[0]
	reordenado, err := PrepararCalificacionesEjercicio(m)
	if err != nil || reordenado.HuellaMaterialSHA256 != r.HuellaMaterialSHA256 {
		t.Fatalf("huella depende del orden de entrada: %v", err)
	}
	m.Configuracion.Fases[0].MinimoMicropuntos = nota(6_000_000)
	otra, err := PrepararCalificacionesEjercicio(m)
	if err != nil || otra.HuellaMaterialSHA256 == r.HuellaMaterialSHA256 {
		t.Fatalf("cambio de reglas no cambia huella: %v", err)
	}
}

func TestPrepararCalificacionesEjercicioRechazaFuentesYRevisionInvalidas(t *testing.T) {
	casos := []struct {
		nombre  string
		cambiar func(*MaterialCalificacionesEjercicio)
	}{
		{"nota fuera de rango", func(m *MaterialCalificacionesEjercicio) { m.Notas[1].PuntosMicropuntos = nota(10_000_001) }},
		{"nota sin fuente", func(m *MaterialCalificacionesEjercicio) { m.Notas[1].FuenteRef = "" }},
		{"fuente sin nota", func(m *MaterialCalificacionesEjercicio) { m.Notas[0].FuenteRef = "correccion_b" }},
		{"duplicada", func(m *MaterialCalificacionesEjercicio) { m.Notas[1].SolicitudRef = m.Notas[0].SolicitudRef }},
		{"revision sin antecedente", func(m *MaterialCalificacionesEjercicio) { m.Revision = 2 }},
		{"antecedente inesperado", func(m *MaterialCalificacionesEjercicio) { m.AntecedenteSHA256 = strings.Repeat("b", 64) }},
		{"fase de meritos", func(m *MaterialCalificacionesEjercicio) {
			m.Configuracion.Modalidad = "concurso"
			m.Configuracion.Fases[0].Tipo = "meritos"
		}},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			m := materialEjercicio()
			c.cambiar(&m)
			if _, err := PrepararCalificacionesEjercicio(m); err == nil {
				t.Fatal("aceptó material inválido")
			}
		})
	}
}
