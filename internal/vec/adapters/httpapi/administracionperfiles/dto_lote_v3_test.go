package administracionperfiles

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

func TestDTOLoteDistingueInicioOmitidoDeFechaProgramada(t *testing.T) {
	_, dto, _, _, _, _ := loteHTTPPrueba(t)
	c := dto.Cambios[0]
	c.InicioVigencia = "inmediato"
	c.Objetivo.VigenteDesde = time.Time{}
	b, err := json.Marshal(c)
	if err != nil || bytes.Contains(b, []byte(`"vigente_desde"`)) {
		t.Fatal("fecha inmediata suministrada por JSON")
	}
	var vuelta CambioPerfil
	if json.Unmarshal(b, &vuelta) != nil || !vuelta.Objetivo.VigenteDesde.IsZero() {
		t.Fatal("omision inmediata no reconocida")
	}
	var objeto map[string]json.RawMessage
	if json.Unmarshal(b, &objeto) != nil {
		t.Fatal("JSON generado invalido")
	}
	var objetivo map[string]json.RawMessage
	if json.Unmarshal(objeto["objetivo"], &objetivo) != nil {
		t.Fatal("objetivo invalido")
	}
	objetivo["vigente_desde"] = json.RawMessage("null")
	objeto["objetivo"], _ = json.Marshal(objetivo)
	alterado, _ := json.Marshal(objeto)
	if json.Unmarshal(alterado, &vuelta) == nil {
		t.Fatal("null suplanta omision de fecha")
	}
	c.InicioVigencia = "programado"
	c.Objetivo.VigenteDesde = c.Objetivo.VigenteHasta.Add(-time.Minute)
	b, err = json.Marshal(c)
	if err != nil || !bytes.Contains(b, []byte(`"vigente_desde"`)) || json.Unmarshal(b, &vuelta) != nil {
		t.Fatal("fecha programada no conservada")
	}
}

func TestDTOLoteBajaRechazaFechaOModoNuevo(t *testing.T) {
	_, dto, _, _, _, _ := loteHTTPPrueba(t)
	c := dto.Cambios[0]
	c.Operacion, c.InicioVigencia = "revocar", ""
	c.Objetivo.VigenteDesde, c.Objetivo.VigenteHasta = time.Time{}, time.Time{}
	b, err := json.Marshal(c)
	if err != nil || bytes.Contains(b, []byte(`"inicio_vigencia"`)) ||
		bytes.Contains(b, []byte(`"vigente_desde"`)) || bytes.Contains(b, []byte(`"vigente_hasta"`)) {
		t.Fatal("baja transporta fecha nueva")
	}
	var vuelta CambioPerfil
	if json.Unmarshal(b, &vuelta) != nil {
		t.Fatal("baja sin fecha rechazada")
	}
	var objeto map[string]json.RawMessage
	if json.Unmarshal(b, &objeto) != nil {
		t.Fatal("JSON generado invalido")
	}
	objeto["inicio_vigencia"] = json.RawMessage(`"inmediato"`)
	alterado, _ := json.Marshal(objeto)
	if json.Unmarshal(alterado, &vuelta) == nil {
		t.Fatal("baja aceptó modo de alta")
	}
}
