package bootstrap

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestOrdenOrigenCTAgregaContextoFiscalizadoSinAlterarPermiso(t *testing.T) {
	original := []byte(`{"ambito_ref":"ambito:uno","unidad_ref":"unidad:uno","accion":"bolsa.origen_ct.consultar","campos_permitidos":["entradas[].razon"],"extra":{"dato":"privado"}}`)
	copia := bytes.Clone(original)
	e := ctdomain.Expediente{OrganizacionRef: "organizacion:uno", Referencia: "expediente:uno", Version: 13,
		Analisis:   &ctdomain.AnalisisRRHH{CategoriaRef: "categoria:uno", ModalidadClave: "sustitucion", CausaClave: "baja"},
		Asignacion: &ctdomain.AsignacionUnidad{UnidadRef: "unidad:uno"}}
	d := preparacionLlamamientoDesarrollo{expediente: ctports.ExpedienteParaSeleccion{Fiscalizado: e},
		categoria: "categoria:uno", unidad: "unidad:uno",
		necesidad: referenciaPuenteLlamamientoDesarrollo("necesidad", e.OrganizacionRef, e.Referencia, "6")}
	actual, err := ordenOrigenCTConContextoComprobacion(original, d, "ambito:uno")
	if err != nil || !bytes.Equal(original, copia) {
		t.Fatalf("alteró orden original: %v", err)
	}
	var antes, despues map[string]json.RawMessage
	if json.Unmarshal(original, &antes) != nil || json.Unmarshal(actual, &despues) != nil || len(despues) != len(antes)+1 {
		t.Fatalf("orden inválida: %s", actual)
	}
	for clave, valor := range antes {
		if !reflect.DeepEqual(valor, despues[clave]) {
			t.Fatalf("modificó %s", clave)
		}
	}
	var ctx map[string]any
	if json.Unmarshal(despues["comprobacion_contexto"], &ctx) != nil || len(ctx) != 9 ||
		ctx["expediente_ref"] != e.Referencia || ctx["necesidad_ref"] != d.necesidad ||
		ctx["organizacion_ref"] != e.OrganizacionRef || ctx["categoria_ref"] != d.categoria ||
		ctx["unidad_ref"] != d.unidad || ctx["ambito_ref"] != "ambito:uno" ||
		ctx["modalidad_ofrecida"] != "sustitucion" || ctx["causa_ofrecida"] != "baja" ||
		ctx["version_expediente"] != float64(13) {
		t.Fatalf("contexto CT: %s", despues["comprobacion_contexto"])
	}
	for _, orden := range [][]byte{
		[]byte(`{"ambito_ref":"ambito:uno","ambito_ref":"ambito:uno","unidad_ref":"unidad:uno"}`),
		[]byte(`{"ambito_ref":"ambito:uno","unidad_ref":"unidad:dos"}`),
		actual,
	} {
		if _, err := ordenOrigenCTConContextoComprobacion(orden, d, "ambito:uno"); err == nil {
			t.Fatalf("aceptó orden incoherente: %s", orden)
		}
	}
}
