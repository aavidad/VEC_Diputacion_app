package ports

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestPaginaComunicacionesExpedienteSeleccionPendienteConservaForma(t *testing.T) {
	c := ConsultaComunicacionesExpediente{ExpedienteRef: "expediente:ct231", Limite: 2}
	fecha := time.Date(2026, 10, 16, 4, 4, 0, 123456000, time.UTC)
	pendiente := ComunicacionExpediente{
		OrganizacionRef: "organizacion:ct231", ExpedienteRef: c.ExpedienteRef,
		LlamamientoRef: "llamamiento:ct231", Estado: EstadoSeleccionConfirmadaPendienteAviso,
		Version: 1, RegistradaEn: fecha, AntecedenteTipo: "seleccion_confirmada",
		ReciboAntecedenteRef: "recibo:ct231",
	}
	comunicada := ComunicacionExpediente{
		OrganizacionRef: pendiente.OrganizacionRef, ExpedienteRef: c.ExpedienteRef,
		LlamamientoRef: "llamamiento:anterior", ComunicacionRef: "comunicacion:ct231",
		Version: 2, Estado: "registrada_localmente", EstadoRespuesta: "sin_respuesta",
		RegistradaEn: fecha.Add(-time.Minute), ReciboComunicacionRef: "recibo:comunicacion:ct231",
		AntecedenteTipo: "seleccion_confirmada", ReciboAntecedenteRef: "recibo:anterior",
	}
	pagina := PaginaComunicacionesExpediente{ExpedienteRef: c.ExpedienteRef,
		Comunicaciones:  []ComunicacionExpediente{comunicada, pendiente},
		SiguienteCursor: pendiente.LlamamientoRef + "#" + strings.Repeat("a", 64)}
	if err := pagina.ValidarPara(c); err != nil {
		t.Fatalf("página mixta válida: %v", err)
	}
	for nombre, cambiar := range map[string]func(*PaginaComunicacionesExpediente){
		"ambito":                func(p *PaginaComunicacionesExpediente) { p.Comunicaciones[1].ExpedienteRef = "expediente:ajeno" },
		"version_ct54":          func(p *PaginaComunicacionesExpediente) { p.Comunicaciones[1].Version = 2 },
		"comunicacion_ficticia": func(p *PaginaComunicacionesExpediente) { p.Comunicaciones[1].ComunicacionRef = "comunicacion:falsa" },
		"recibo_ausente":        func(p *PaginaComunicacionesExpediente) { p.Comunicaciones[1].ReciboAntecedenteRef = "" },
		"otra_pendiente":        func(p *PaginaComunicacionesExpediente) { p.Comunicaciones[0] = p.Comunicaciones[1] },
		"cursor_ajeno": func(p *PaginaComunicacionesExpediente) {
			p.SiguienteCursor = comunicada.ComunicacionRef + "#" + strings.Repeat("a", 64)
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			mutada := pagina
			mutada.Comunicaciones = append([]ComunicacionExpediente(nil), pagina.Comunicaciones...)
			cambiar(&mutada)
			if mutada.ValidarPara(c) == nil {
				t.Fatal("aceptó fila o cursor incompatible")
			}
		})
	}
	for nombre, fila := range map[string]string{
		"extra":      `{"organizacion_ref":"organizacion:ct231","expediente_ref":"expediente:ct231","llamamiento_ref":"llamamiento:ct231","estado":"seleccion_confirmada_pendiente_aviso","registrada_en":"2026-10-16T04:04:00Z","version":1,"antecedente_tipo":"seleccion_confirmada","recibo_antecedente_ref":"recibo:ct231","seleccion_confirmada":{}}`,
		"campo_ct54": `{"organizacion_ref":"organizacion:ct231","expediente_ref":"expediente:ct231","llamamiento_ref":"llamamiento:ct231","estado":"seleccion_confirmada_pendiente_aviso","registrada_en":"2026-10-16T04:04:00Z","version":1,"antecedente_tipo":"seleccion_confirmada","recibo_antecedente_ref":"recibo:ct231","comunicacion_ref":"comunicacion:falsa"}`,
	} {
		t.Run(nombre, func(t *testing.T) {
			var decodificada ComunicacionExpediente
			if json.Unmarshal([]byte(fila), &decodificada) == nil {
				t.Fatal("aceptó forma ajena al contrato")
			}
		})
	}
	for nombre, fila := range map[string]ComunicacionExpediente{"pendiente": pendiente, "comunicacion": comunicada} {
		t.Run("json_"+nombre, func(t *testing.T) {
			contenido, err := json.Marshal(fila)
			if err != nil {
				t.Fatal(err)
			}
			var campos map[string]json.RawMessage
			if json.Unmarshal(contenido, &campos) != nil ||
				(len(campos) != 8 && nombre == "pendiente") ||
				(len(campos) != 11 && nombre == "comunicacion") {
				t.Fatalf("forma pública distinta: %s", contenido)
			}
			var recuperada ComunicacionExpediente
			if err := json.Unmarshal(contenido, &recuperada); err != nil || recuperada != fila {
				t.Fatalf("fila no recuperable: %v", err)
			}
		})
	}
}
