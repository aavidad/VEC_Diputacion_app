package domain

import "testing"

func TestPlantillaVersionadaOrdenaBloquesYConservaLimites(t *testing.T) {
	p := Plantilla{Esquema: "vec.certificados.plantilla-ensayo.v1", ID: "servicios", Version: 2, Estado: "ensayo", Bloques: []string{"persona", "corte", "fuente", "plantilla", "criterio", "limite"}}
	if p.Validar("servicios", 2) != nil {
		t.Fatal("catalogue order rejected")
	}
	p.Bloques[5] = "persona"
	if p.Validar("servicios", 2) == nil {
		t.Fatal("safety block could be omitted")
	}
}

func TestFuenteDeEnsayoYFormaV1NoMezclanCampos(t *testing.T) {
	dias := int64(10)
	base := func(esquema string) FuenteServicios {
		return FuenteServicios{Esquema: esquema, Sintetica: true, ProcedenciaRef: "ensayo:x", Nombre: "Elena Martín Robles",
			Corte: Corte{VigenteEn: "2026-10-01", ConocidoEn: "2026-10-01T08:00:00Z"}}
	}
	ensayo := base(EsquemaFuenteEnsayo)
	ensayo.Servicios = []Servicio{{Inicio: "2024-01-01", Fin: "2024-01-10", Clase: "A", Dias: &dias, Estado: "reconocido"}}
	if ensayo.ValidarEnsayo() != nil {
		t.Fatal("ensayo válido rechazado")
	}
	v1 := base(EsquemaFuentePersonalV1Ensayo)
	v1.Cobertura = "completa"
	v1.Servicios = []Servicio{{Inicio: "2024-01-01", Clase: "c", ClaseVersion: 1, Estado: "reconocido", Certeza: "acreditado", ServicioRef: "s:1", ActoRef: "a:1"}}
	if v1.ValidarEnsayo() != nil {
		t.Fatal("V1 válida rechazada")
	}
	cambios := []func(){
		func() { ensayo.Cobertura = "completa" },
		func() { ensayo.Servicios[0].Certeza = "acreditado" },
		func() { ensayo.Servicios[0].Fin = "" },
		func() { ensayo.Servicios[0].Dias = nil },
		func() { v1.Servicios[0].Dias = &dias },
		func() { v1.Cobertura = "" },
		func() { v1.Servicios[0].Certeza = "" },
		func() { v1.Servicios[0].ActoRef = "" },
		func() { v1.Servicios[0].Inicio = "2026-10-02" },
	}
	for i, cambiar := range cambios {
		guardaE, guardaV := ensayo, v1
		guardaE.Servicios, guardaV.Servicios = append([]Servicio{}, ensayo.Servicios...), append([]Servicio{}, v1.Servicios...)
		cambiar()
		if ensayo.ValidarEnsayo() == nil && v1.ValidarEnsayo() == nil {
			t.Errorf("caso %d aceptado", i)
		}
		ensayo, v1 = guardaE, guardaV
	}
}
