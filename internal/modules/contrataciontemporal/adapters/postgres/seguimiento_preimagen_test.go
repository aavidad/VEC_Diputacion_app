package postgres

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestSeguimientoConservaPreimagenSQLConMicrosegundoFinalCero(t *testing.T) {
	civil, _ := expedienteAltaCivilSQLPrueba(t)
	var expediente map[string]json.RawMessage
	if err := json.Unmarshal(civil, &expediente); err != nil {
		t.Fatal(err)
	}
	const instante = `"2026-07-24T08:00:00.415640Z"`
	for _, campo := range []string{"creado_en", "actualizado_en"} {
		expediente[campo] = json.RawMessage(instante)
	}
	var actuaciones []map[string]json.RawMessage
	if err := json.Unmarshal(expediente["actuaciones"], &actuaciones); err != nil || len(actuaciones) != 1 {
		t.Fatal("actuación original no disponible", err)
	}
	actuaciones[0]["realizada_en"] = json.RawMessage(instante)
	expediente["actuaciones"], _ = json.Marshal(actuaciones)
	crudo, _ := json.Marshal(expediente)
	respuesta, _ := json.Marshal(struct {
		Esquema    string          `json:"esquema"`
		Resultado  string          `json:"resultado"`
		Expediente json.RawMessage `json:"expediente"`
	}{operacionesSeguimientoSQL[ports.OperacionRegistrarCese].esquemaResultado, "preparada", crudo})
	preparada, err := decodificarRespuestaSeguimiento(respuesta, operacionesSeguimientoSQL[ports.OperacionRegistrarCese].esquemaResultado)
	if err != nil || preparada.Expediente == nil || preparada.Expediente.Validar() != nil ||
		!preparada.Expediente.ActualizadoEn.Equal(time.Date(2026, 7, 24, 8, 0, 0, 415640000, time.UTC)) {
		t.Fatalf("expediente tipado alterado: %v", err)
	}
	if !bytes.Contains(preparada.ExpedienteRaw, []byte(instante)) {
		t.Fatal("se perdió la representación original de SQL")
	}
	respuesta[0] = 'X'
	if !json.Valid(preparada.ExpedienteRaw) {
		t.Fatal("el original comparte memoria con el búfer de respuesta")
	}
	preimagen, err := preimagenOriginalSeguimiento(ports.PreparacionOperacionSeguimiento{
		Expediente: *preparada.Expediente, ExpedienteAnteriorJSON: preparada.ExpedienteRaw,
	}, preparada.Expediente.OrganizacionRef, preparada.Expediente.Referencia)
	if err != nil {
		t.Fatal(err)
	}
	carga, err := json.Marshal(map[string]any{"expediente_anterior": preimagen})
	if err != nil || !bytes.Contains(carga, []byte(instante)) ||
		bytes.Contains(carga, []byte(`"actualizado_en":"2026-07-24T08:00:00.41564Z"`)) {
		t.Fatal("confirmación perdió los bytes temporales originales", err)
	}
	var sobre map[string]json.RawMessage
	if err := json.Unmarshal(carga, &sobre); err != nil {
		t.Fatal(err)
	}
	var anterior struct {
		CreadoEn      string `json:"creado_en"`
		ActualizadoEn string `json:"actualizado_en"`
		Actuaciones   []struct {
			RealizadaEn string `json:"realizada_en"`
		} `json:"actuaciones"`
	}
	if err := json.Unmarshal(sobre["expediente_anterior"], &anterior); err != nil ||
		anterior.CreadoEn != "2026-07-24T08:00:00.415640Z" ||
		anterior.ActualizadoEn != "2026-07-24T08:00:00.415640Z" ||
		len(anterior.Actuaciones) != 1 || anterior.Actuaciones[0].RealizadaEn != "2026-07-24T08:00:00.415640Z" {
		t.Fatal("valores JSONB de la preimagen no coinciden con SQL", err)
	}
	instanteNuevo := time.Date(2026, 7, 24, 8, 0, 1, 123456000, time.UTC)
	siguiente := *preparada.Expediente
	siguiente.Version++
	siguiente.ActualizadoEn = instanteNuevo
	siguiente.Actuaciones = append(append(siguiente.Actuaciones[:0:0], siguiente.Actuaciones...), siguiente.Actuaciones[0])
	siguiente.Actuaciones[1].Secuencia = 2
	siguiente.Actuaciones[1].VersionExpediente = 2
	siguiente.Actuaciones[1].RealizadaEn = instanteNuevo
	proyectada, err := postimagenOriginalSeguimiento(ports.PreparacionOperacionSeguimiento{
		Expediente: *preparada.Expediente, ExpedienteAnteriorJSON: preparada.ExpedienteRaw,
	}, siguiente, instanteNuevo)
	if err != nil || !bytes.Contains(proyectada, []byte(`"inicio":"2026-09-01"`)) ||
		!bytes.Contains(proyectada, []byte(`"fecha":""`)) ||
		!bytes.Contains(proyectada, []byte(instante)) || !bytes.Contains(proyectada, []byte(`"version":2`)) {
		t.Fatal("postimagen perdió la forma física del agregado SQL", err)
	}
	if tipada, err := json.Marshal(siguiente); err != nil || bytes.Contains(tipada, []byte(`"inicio":"2026-09-01"`)) {
		t.Fatal("la muestra no distingue DTO de agregado físico", err)
	}
	for _, invalida := range []json.RawMessage{nil, []byte(`{"referencia":`), []byte(`{"referencia":"otro","version":1}`)} {
		_, err := preimagenOriginalSeguimiento(ports.PreparacionOperacionSeguimiento{
			Expediente: *preparada.Expediente, ExpedienteAnteriorJSON: invalida,
		}, preparada.Expediente.OrganizacionRef, preparada.Expediente.Referencia)
		if !errors.Is(err, ports.ErrOperacionSeguimientoInvalida) {
			t.Fatalf("preimagen inválida aceptada: %v", err)
		}
	}
}

func TestSeguimientoOtraOperacionConservaProyeccionTipada(t *testing.T) {
	expediente := expedienteInicialAnalisisPostgreSQLPrueba(t)
	orden := ports.OrdenConfirmarOperacionSeguimiento{
		Operacion:   ports.OperacionCerrarExpediente,
		Preparacion: ports.PreparacionOperacionSeguimiento{Expediente: expediente},
		Siguiente:   expediente,
	}
	anterior, siguiente, err := proyeccionesConfirmacionSeguimiento(orden, expediente.OrganizacionRef, expediente.Referencia)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := anterior.(domain.Expediente); !ok {
		t.Fatal("cierre dejó de enviar expediente tipado anterior")
	}
	if _, ok := siguiente.(domain.Expediente); !ok {
		t.Fatal("cierre dejó de enviar expediente tipado siguiente")
	}
}
