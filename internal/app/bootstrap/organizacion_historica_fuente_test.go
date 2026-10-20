package bootstrap

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"slices"
	"testing"

	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// La fuente de F1-103 debe describir exactamente la concesión que Personal
// consume (AD51 y Personal10): acción, recurso, finalidad, seis campos, sin
// obligaciones y con las dimensiones del selector histórico.
func TestFuenteOrganizacionHistoricaRRHHCoherenteConPersonal(t *testing.T) {
	contenido, err := os.ReadFile("../../../deploy/principal/fuentes/personal_organizacion_historica_rrhh_v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var fuente struct {
		ModuloID   string                                    `json:"modulo_id"`
		Referencia string                                    `json:"referencia"`
		Version    int                                       `json:"version"`
		Huella     string                                    `json:"huella_sha256"`
		Canon      string                                    `json:"entradas_canon"`
		Entradas   []vecdomain.EntradaAccionAdministracionV1 `json:"entradas"`
	}
	if err := json.Unmarshal(contenido, &fuente); err != nil {
		t.Fatal(err)
	}
	huella := sha256.Sum256([]byte(fuente.Canon))
	if hex.EncodeToString(huella[:]) != fuente.Huella || len(fuente.Entradas) != 1 {
		t.Fatal("huella o número de entradas de la fuente incorrectos")
	}
	e := fuente.Entradas[0]
	// El canon es la serialización Go de las entradas sin su propia huella.
	sinHuella := map[string]any{}
	crudo, _ := json.Marshal(fuente.Entradas[0])
	if err := json.Unmarshal(crudo, &sinHuella); err != nil {
		t.Fatal(err)
	}
	delete(sinHuella, "fuente_huella_sha256")
	var canon []map[string]any
	if err := json.Unmarshal([]byte(fuente.Canon), &canon); err != nil || len(canon) != 1 {
		t.Fatal("canon ilegible")
	}
	a, _ := json.Marshal(canon[0])
	b, _ := json.Marshal(sinHuella)
	if !bytes.Equal(a, b) {
		t.Fatal("el canon no coincide con la entrada completa")
	}
	c := e.Concesion
	if e.Validar() != nil || fuente.ModuloID != "personal" || e.FuenteRef != fuente.Referencia ||
		e.FuenteVersion != fuente.Version || e.FuenteHuellaSHA256 != fuente.Huella ||
		c.Accion != personaldomain.AccionConsultaOrganizacionHistorica || c.ModuloID != "personal" ||
		c.TipoRecurso != "organizacion_historica" ||
		!slices.Equal(c.Finalidades, []string{"consultar_organizacion_historica"}) ||
		!slices.Equal(c.CamposPermitidos, []string{"dotaciones", "plazas", "puestos_individuales", "puestos_tipo", "unidades", "vinculos"}) ||
		len(c.Obligaciones) != 0 || c.GarantiaMinima != vecdomain.AuthAssuranceSubstantial ||
		!slices.Equal(e.DimensionesAmbito, []string{"organismo_ref", "unidad_clave"}) ||
		e.ClaseControl != string(vecdomain.ClaseControlPerfilOrdinario) {
		t.Fatal("la entrada no coincide con la consulta de Organización histórica de Personal")
	}
}
