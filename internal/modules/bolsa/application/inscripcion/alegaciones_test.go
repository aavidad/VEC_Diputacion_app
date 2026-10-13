package inscripcion

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestAlegacionDosObjetosExactosYMaterialLigado(t *testing.T) {
	clave := "clave_alegacion_20261013"
	persona := "per_" + strings.Repeat("a", 22)
	decision := ObjetoAlegacion{Tipo: "decision_inscripcion", SolicitudRef: "solicitud_inscripcion_" + strings.Repeat("b", 64), ActoRef: "decision_real_3", VersionActo: 3}
	participacion := ObjetoAlegacion{Tipo: "acto_participacion", ParticipacionRef: "participacion_legacy_1", ActoRef: "acta_real_7", VersionActo: 7}
	for _, objeto := range []ObjetoAlegacion{decision, participacion} {
		p := PresentacionAlegacion{Objeto: objeto, Fundamento: "Alego sobre el acto indicado.", ClaveIdempotencia: clave}
		material, huella, err := MaterialAlegacion(p)
		if err != nil || len(huella) != 64 || !bytes.Contains(material, []byte(objeto.ActoRef)) {
			t.Fatalf("material del acto real inválido: %v", err)
		}
		var m map[string]any
		if err := json.Unmarshal(material, &m); err != nil || len(m) != 4 || m["esquema"] != EsquemaMaterialAlegacion {
			t.Fatalf("material contiene datos no contratados: %v", err)
		}
		recurso, err := RecursoPresentacionAlegacion(p, huella, map[string]string{"candidato_ref": "can_" + strings.Repeat("c", 22)})
		if err != nil || !bytes.Contains(recurso, []byte(huella)) || bytes.Contains(recurso, []byte(persona)) {
			t.Fatalf("recurso sin vínculo exacto: %v", err)
		}
	}
	ref, err := ReferenciaAlegacion(persona, clave)
	repetida, err2 := ReferenciaAlegacion(persona, clave)
	otra, err3 := ReferenciaAlegacion(persona, clave+"_otra")
	if err != nil || err2 != nil || err3 != nil || ref != repetida || ref == otra {
		t.Fatal("la referencia idempotente no distingue la operación")
	}
	decision.ParticipacionRef = participacion.ParticipacionRef
	if decision.Validar() == nil {
		t.Fatal("se aceptaron dos objetos para una misma alegación")
	}
	participacion.ActoRef = ""
	if participacion.Validar() == nil {
		t.Fatal("se aceptó una participación sin acta")
	}
}
