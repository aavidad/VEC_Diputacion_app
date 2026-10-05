package administracionperfiles

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/pruebas"
)

func TestDenegarActorTransportaParV2CopiadoSinSerializarlo(t *testing.T) {
	r, v, err := pruebas.NuevoContextoRegistradoYVinculoV2(time.Now().UTC(), "per_"+strings.Repeat("a", 22), "prf_"+strings.Repeat("b", 22), domain.AuthMethodCertificate, domain.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	original := bytes.Clone(r.RepresentacionCanonica)
	aud := &auditorPrueba{}
	h := &Handler{auditor: aud}
	w := httptest.NewRecorder()
	peticion := httptest.NewRequest(http.MethodGet, "https://admin.example.test/portal/admin/personas/invalid", nil)
	h.denegarActor(w, peticion, SesionConfiable{Actor: r.Contexto, Evidencia: domain.EvidenciaSesionAdministracionPerfiles{ResultadoContexto: r, Vinculo: v}, CorrelacionRef: "correlacion_" + strings.Repeat("c", 32)}, http.StatusBadRequest, "solicitud_invalida", "consultar_persona", "per_"+strings.Repeat("d", 22))
	if w.Code != http.StatusBadRequest || aud.llamadas != 1 || aud.ultima.Evidencia.ValidarPara(aud.ultima.Actor) != nil || aud.ultima.Actor.PersonaRef != r.Contexto.PersonaRef {
		t.Fatalf("evidencia no propagada: %d %+v", w.Code, aud.ultima)
	}
	aud.ultima.Evidencia.ResultadoContexto.RepresentacionCanonica[0] ^= 1
	if !bytes.Equal(r.RepresentacionCanonica, original) {
		t.Fatal("auditor modificó V2 original")
	}
	b, err := json.Marshal(aud.ultima)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(b, []byte(`"Evidencia"`)) || bytes.Contains(b, []byte(`"Actor"`)) || bytes.Contains(b, original) {
		t.Fatal("par V2 serializado en DTO de frontera")
	}
}
