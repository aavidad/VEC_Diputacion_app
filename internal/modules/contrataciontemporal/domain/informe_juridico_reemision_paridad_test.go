package domain

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

// Los tres agregados de testdata los escribió PostgreSQL 18 al recorrer
// CT92 → CT123 → CT93 sobre el volcado sintético de la principal
// (probar_ct123_informe_nuevo_tras_subsanacion_pg18.sh): v7 subsanado, v8 con
// el informe nuevo y v9 fiscalizado favorable. El dominio Go debe restaurarlos
// y producir exactamente la misma proyección que la base.

func cargarAgregadoPrueba(t *testing.T, nombre string) (Expediente, any) {
	t.Helper()
	contenido, err := os.ReadFile("testdata/" + nombre)
	if err != nil {
		t.Fatal(err)
	}
	var e Expediente
	var generico any
	if err := json.Unmarshal(contenido, &e); err != nil || json.Unmarshal(contenido, &generico) != nil {
		t.Fatalf("%s: %v", nombre, err)
	}
	if err := e.Validar(); err != nil {
		t.Fatalf("%s no valida en Go: %v", nombre, err)
	}
	return e, generico
}

func mismaProyeccionJSON(t *testing.T, e Expediente, esperado any) {
	t.Helper()
	contenido, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	var obtenido any
	if err := json.Unmarshal(contenido, &obtenido); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(obtenido, esperado) {
		t.Fatalf("la proyección Go difiere de la de PostgreSQL:\nGo:  %s", contenido)
	}
}

func TestParidadInformeNuevoTrasSubsanacionConPostgreSQL(t *testing.T) {
	v7, _ := cargarAgregadoPrueba(t, "expediente_informe_nuevo_v7.json")
	v8, v8JSON := cargarAgregadoPrueba(t, "expediente_informe_nuevo_v8.json")
	v9, v9JSON := cargarAgregadoPrueba(t, "expediente_informe_nuevo_v9.json")
	if !v7.RefiscalizacionEsperaInformeNuevo() || v8.RefiscalizacionEsperaInformeNuevo() || !v8.InformeReemitidoTrasSubsanacion() {
		t.Fatal("estados de espera del informe nuevo incoherentes con la base")
	}

	informe := *v8.InformeJuridico
	informe.Sustituye, informe.ActuacionRegistro = nil, nil
	ultima := v8.Actuaciones[len(v8.Actuaciones)-1]
	obtenido, err := v7.EmitirInformeJuridico(v7.Version, informe, DatosActuacion{
		AccionClave: ultima.AccionClave, ActorRef: ultima.ActorRef, UnidadRef: ultima.UnidadRef,
		ReciboRef: ultima.ReciboRef, RealizadaEn: ultima.RealizadaEn, DocumentosRef: ultima.DocumentosRef,
	})
	if err != nil {
		t.Fatalf("emitir el informe nuevo sobre v7: %v", err)
	}
	mismaProyeccionJSON(t, obtenido, v8JSON)

	ultima = v9.Actuaciones[len(v9.Actuaciones)-1]
	fiscalizado, err := v8.RegistrarFiscalizacion(v8.Version, DatosRegistrarFiscalizacion{
		FiscalizacionRef: v9.Fiscalizacion.FiscalizacionRef, Resultado: v9.Fiscalizacion.Resultado,
		UnidadFiscalizadoraRef: v9.Fiscalizacion.UnidadFiscalizadoraRef, FiscalizadaEn: v9.Fiscalizacion.FiscalizadaEn,
	}, DatosActuacion{
		AccionClave: ultima.AccionClave, ActorRef: ultima.ActorRef, UnidadRef: ultima.UnidadRef, ReciboRef: ultima.ReciboRef,
		RealizadaEn: ultima.RealizadaEn, FaseDestino: ultima.FaseDestino, EstadoDestino: ultima.EstadoDestino,
		DocumentosRef: ultima.DocumentosRef, RetornoRef: ultima.RetornoRef,
	})
	if err != nil {
		t.Fatalf("nueva fiscalización sobre v8: %v", err)
	}
	mismaProyeccionJSON(t, fiscalizado, v9JSON)
	if fiscalizado.Fiscalizacion.InformeJuridicoRef != v8.InformeJuridico.InformeRef {
		t.Fatal("la nueva fiscalización no cita el informe nuevo")
	}
}
