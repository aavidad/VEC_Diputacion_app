package domain

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"testing"
	"time"
)

// El expediente de prueba es el agregado sintético v7 (nombramiento) tal como
// lo guarda PostgreSQL. Las pruebas comprueban además que la proyección Go
// coincide con la que CT115/CT116 construyen en SQL mezclando el agregado.
func expedienteNombramientoPrueba(t *testing.T) (Expediente, map[string]any) {
	t.Helper()
	contenido, err := os.ReadFile("testdata/expediente_nombramiento_v7.json")
	if err != nil {
		t.Fatal(err)
	}
	var e Expediente
	dec := json.NewDecoder(bytes.NewReader(contenido))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&e); err != nil || e.Validar() != nil {
		t.Fatalf("fixture inválido: %v %v", err, e.Validar())
	}
	var mapa map[string]any
	if err := json.Unmarshal(contenido, &mapa); err != nil {
		t.Fatal(err)
	}
	return e, mapa
}

func mapaJSON(t *testing.T, v any) map[string]any {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func instantePrueba(e Expediente) time.Time {
	return e.ActualizadoEn.Add(90*time.Second + 123456*time.Microsecond)
}

func TestRegistrarCeseAnadeActuacionComoCT115(t *testing.T) {
	e, original := expedienteNombramientoPrueba(t)
	instante := instantePrueba(e)
	datos := DatosCese{CausaClave: "fin_sustitucion", FechaEfecto: time.Date(2027, 2, 15, 0, 0, 0, 0, time.UTC),
		JustificanteTipo: "comunicacion_reincorporacion", JustificanteRef: "documento:ct:justificante:1",
		JustificanteSHA256: "3259ad24878afcc3e4cf6ad860377c7d84f6b2b5152dc61146686a8e69ee895b", Observaciones: "Reincorporación"}
	act := DatosActuacion{AccionClave: AccionCesarNombramiento, ActorRef: "per_actor", UnidadRef: e.Asignacion.UnidadRef,
		ReciboRef: "recibo:cese:1", RealizadaEn: instante, FaseDestino: FaseNombramiento, EstadoDestino: EstadoEnCurso,
		Observaciones: datos.Observaciones, DocumentosRef: []string{datos.JustificanteRef}}
	siguiente, err := e.RegistrarCese(e.Version, datos, act)
	if err != nil {
		t.Fatal(err)
	}
	esperado := original
	esperado["version"] = float64(e.Version + 1)
	esperado["actualizado_en"] = instante.Format(time.RFC3339Nano)
	acts := append([]any(nil), original["actuaciones"].([]any)...)
	acts = append(acts, mapaJSON(t, siguiente.Actuaciones[len(siguiente.Actuaciones)-1]))
	esperado["actuaciones"] = acts
	if !reflect.DeepEqual(mapaJSON(t, siguiente), esperado) {
		t.Fatal("la proyección Go del cese no coincide con la mezcla de CT115")
	}
	if _, err := siguiente.RegistrarCese(siguiente.Version, datos, act); err == nil {
		t.Fatal("un segundo cese debe rechazarse")
	}
	malo := act
	malo.DocumentosRef = []string{"documento:otro"}
	if _, err := e.RegistrarCese(e.Version, datos, malo); err == nil {
		t.Fatal("el justificante de la actuación debe ser el del cese")
	}
	if _, err := e.RegistrarCese(e.Version-1, datos, act); err != ErrVersionEnConflicto {
		t.Fatalf("versión esperada: %v", err)
	}
}

func TestCerrarTrasCeseExigeCeseYGINPIXSegunCondiciones(t *testing.T) {
	e, _ := expedienteNombramientoPrueba(t)
	instante := instantePrueba(e)
	confirmada := time.Date(2027, 2, 16, 0, 0, 0, 0, time.UTC)
	datos := DatosCierreExpediente{Condiciones: []string{CondicionCeseRegistrado, CondicionGINPIXConfirmado}, GINPIXNumero: "GX-1", GINPIXConfirmadaEn: &confirmada}
	cierre := DatosActuacion{AccionClave: AccionCerrarExpediente, ActorRef: "per_actor", UnidadRef: e.Asignacion.UnidadRef,
		ReciboRef: "recibo:cierre:1", RealizadaEn: instante, FaseDestino: FaseNombramiento, EstadoDestino: EstadoCompletado,
		DocumentosRef: []string{"ginpix:GX-1"}}
	if _, err := e.CerrarTrasCese(e.Version, datos, cierre); err == nil {
		t.Fatal("sin cese no hay cierre")
	}
	cesado, err := e.RegistrarCese(e.Version, DatosCese{CausaClave: "renuncia", FechaEfecto: confirmada, JustificanteTipo: "escrito_renuncia",
		JustificanteRef: "documento:renuncia", JustificanteSHA256: "3259ad24878afcc3e4cf6ad860377c7d84f6b2b5152dc61146686a8e69ee895b"},
		DatosActuacion{AccionClave: AccionCesarNombramiento, ActorRef: "per_actor", UnidadRef: e.Asignacion.UnidadRef, ReciboRef: "recibo:cese:2",
			RealizadaEn: instante, FaseDestino: FaseNombramiento, EstadoDestino: EstadoEnCurso, DocumentosRef: []string{"documento:renuncia"}})
	if err != nil {
		t.Fatal(err)
	}
	cerrado, err := cesado.CerrarTrasCese(cesado.Version, datos, cierre)
	if err != nil || cerrado.EstadoActual != EstadoCompletado {
		t.Fatalf("cierre: %v", err)
	}
	sinGINPIX := DatosCierreExpediente{Condiciones: []string{CondicionCeseRegistrado, CondicionGINPIXConfirmado}}
	if sinGINPIX.Validar() == nil {
		t.Fatal("la regla exige GINPIX y falta el número")
	}
	soloCese := DatosCierreExpediente{Condiciones: []string{CondicionCeseRegistrado}}
	cierre.DocumentosRef = nil
	if _, err := cesado.CerrarTrasCese(cesado.Version, soloCese, cierre); err != nil {
		t.Fatalf("la regla solo exige el cese: %v", err)
	}
	if CondicionesCierreValidas([]string{CondicionGINPIXConfirmado}) {
		t.Fatal("el cese es siempre condición de cierre")
	}
}

func TestModificarTrasNombramientoCreaAnalisisYVuelveAFiscalizacion(t *testing.T) {
	e, original := expedienteNombramientoPrueba(t)
	instante := instantePrueba(e)
	datos := DatosModificacionTrasNombramiento{MotivoClave: "cambio_jornada", Periodo: e.Analisis.Periodo, Jornada: 5000,
		Coste: Importe{Moneda: "EUR", Centimos: 2000000}, FuenteCoste: "autoridad:ct:desarrollo:calculo-coste",
		FaseRetorno: FaseFiscalizacion, Observaciones: "Reducción de jornada"}
	act := DatosActuacion{AccionClave: AccionModificarTrasNombramiento, ActorRef: "per_actor", UnidadRef: e.Asignacion.UnidadRef,
		ReciboRef: "recibo:mod:1", RealizadaEn: instante, FaseDestino: FaseFiscalizacion, EstadoDestino: EstadoEnCurso,
		Observaciones: datos.Observaciones}
	siguiente, err := e.ModificarTrasNombramiento(e.Version, datos, act)
	if err != nil {
		t.Fatal(err)
	}
	if siguiente.Fiscalizacion != nil || siguiente.FaseActual != FaseFiscalizacion || siguiente.Analisis.PorcentajeJornada != 5000 ||
		siguiente.InformeJuridico == nil {
		t.Fatal("proyección de la modificación inesperada")
	}
	// Misma construcción que proyeccion_modificacion_ct116.
	esperado := original
	delete(esperado, "fiscalizacion")
	analisis := esperado["analisis"].(map[string]any)
	analisis["porcentaje_jornada"] = float64(5000)
	analisis["coste_previsto"] = map[string]any{"moneda": "EUR", "centimos": float64(2000000)}
	analisis["fuente_coste_ref"] = datos.FuenteCoste
	analisis["actuacion_registro"] = map[string]any{"secuencia": float64(e.Version + 1), "version_expediente": float64(e.Version + 1),
		"accion_clave": string(AccionModificarTrasNombramiento), "fase_destino": string(FaseFiscalizacion), "recibo_ref": "recibo:mod:1"}
	esperado["version"] = float64(e.Version + 1)
	esperado["fase_actual"] = string(FaseFiscalizacion)
	esperado["actualizado_en"] = instante.Format(time.RFC3339Nano)
	acts := append([]any(nil), original["actuaciones"].([]any)...)
	esperado["actuaciones"] = append(acts, mapaJSON(t, siguiente.Actuaciones[len(siguiente.Actuaciones)-1]))
	if !reflect.DeepEqual(mapaJSON(t, siguiente), esperado) {
		t.Fatal("la proyección Go de la modificación no coincide con CT116")
	}
	sinCambio := datos
	sinCambio.Jornada = e.Analisis.PorcentajeJornada
	if _, err := e.ModificarTrasNombramiento(e.Version, sinCambio, act); err == nil {
		t.Fatal("una modificación sin cambios debe rechazarse")
	}
	caro := datos
	caro.Coste.Centimos = e.Analisis.ValidacionRC.Importe.Centimos + 1
	if _, err := e.ModificarTrasNombramiento(e.Version, caro, act); err == nil {
		t.Fatal("un coste por encima de la retención de crédito debe rechazarse")
	}
}
