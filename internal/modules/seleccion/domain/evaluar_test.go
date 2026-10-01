package domain

import "testing"

func TestEmpateEnUltimaPlazaQuedaSinOrdenNiPropuesta(t *testing.T) {
	minimo := int64(5_000_000)
	config := Configuracion{Version: 1, Modalidad: "oposicion", TurnoAcceso: "libre", Destino: "plaza", Plazas: 1,
		Fases: []Fase{{Referencia: "ejercicio", Tipo: "prueba", MinimoMicropuntos: &minimo, MaximoMicropuntos: 10_000_000, Peso: 100}}}
	nota := int64(8_000_000)
	entrada := Entrada{ConvocatoriaRef: "convocatoria_sintetica", BasesVersion: 1,
		Solicitudes: []Solicitud{
			{Referencia: "a", Nombre: "María Torres Ruiz", Requisitos: []Requisito{{Referencia: "acceso", Estado: "cumple", FuenteRef: "fuente_a"}}, Notas: map[string]*int64{"ejercicio": &nota}},
			{Referencia: "b", Nombre: "José Martín López", Requisitos: []Requisito{{Referencia: "acceso", Estado: "cumple", FuenteRef: "fuente_b"}}, Notas: map[string]*int64{"ejercicio": &nota}},
		}}
	r, err := Evaluar(config, entrada, nil)
	if err != nil {
		t.Fatal(err)
	}
	if r.Estado != "indeterminado" || len(r.Causas) != 1 || r.Causas[0] != "desempate_pendiente" {
		t.Fatalf("el empate podría adjudicar la plaza: %+v", r)
	}
	for _, s := range r.Solicitudes {
		if s.Orden != nil || s.Propuesta != "pendiente" || s.Estado != "empate_pendiente" || s.TotalMicropuntos == nil || *s.TotalMicropuntos != nota {
			t.Fatalf("la solicitud empatada parece resuelta: %+v", s)
		}
		if len(s.Fases) != 1 || s.Fases[0].Peso != 100 || s.Fases[0].MinimoMicropuntos == nil || *s.Fases[0].MinimoMicropuntos != minimo {
			t.Fatalf("la puntuación no explica su umbral y ponderación: %+v", s.Fases)
		}
	}
}

func TestRequisitoPendienteNoSeConvierteEnAprobacion(t *testing.T) {
	minimo := int64(0)
	nota := int64(10_000_000)
	config := Configuracion{Version: 1, Modalidad: "oposicion", TurnoAcceso: "libre", Destino: "plaza", Plazas: 1,
		Fases: []Fase{{Referencia: "ejercicio", Tipo: "prueba", MinimoMicropuntos: &minimo, MaximoMicropuntos: nota, Peso: 100}}}
	entrada := Entrada{ConvocatoriaRef: "convocatoria_sintetica", BasesVersion: 1,
		Solicitudes: []Solicitud{{Referencia: "a", Nombre: "Carmen Vega Serrano", Requisitos: []Requisito{{Referencia: "acceso", Estado: "pendiente"}}, Notas: map[string]*int64{"ejercicio": &nota}}}}
	r, err := Evaluar(config, entrada, nil)
	if err != nil {
		t.Fatal(err)
	}
	if r.Estado != "indeterminado" || r.Solicitudes[0].Estado != "pendiente" || r.Solicitudes[0].Orden != nil || r.Solicitudes[0].Propuesta != "sin_propuesta" {
		t.Fatalf("el requisito no acreditado se convirtió en propuesta: %+v", r)
	}
}
