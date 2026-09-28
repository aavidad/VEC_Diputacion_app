package postgres

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestConsultaReciboRespuestaVinculaOrganizacionExpedienteYComunicacion(t *testing.T) {
	base := ports.SolicitudConsultaReciboRespuesta{
		OrganizacionRef: "organizacion:prueba", ExpedienteRef: "expediente:prueba", ComunicacionRef: "comunicacion:prueba",
	}
	obtener := func(s ports.SolicitudConsultaReciboRespuesta) string {
		t.Helper()
		r, err := RecursoConsultaReciboRespuesta(s)
		if err != nil {
			t.Fatal(err)
		}
		if r.Referencia != s.ComunicacionRef || r.Ambitos["organizacion_ref"] != s.OrganizacionRef ||
			r.Ambitos["expediente_ref"] != s.ExpedienteRef || len(r.Ambitos) != 2 {
			t.Fatalf("ámbito incorrecto: %+v", r)
		}
		h, err := r.HuellaContextoAutorizacionSHA256()
		if err != nil {
			t.Fatal(err)
		}
		return h
	}
	h := obtener(base)
	material, err := json.Marshal(base)
	if err != nil {
		t.Fatal(err)
	}
	hMaterial := sha256.Sum256(material)
	canon := fmt.Sprintf(`{"ambitos":{"expediente_ref":"%s","organizacion_ref":"%s"},"atributos":{"material_sha256":"%s"}}`,
		base.ExpedienteRef, base.OrganizacionRef, hex.EncodeToString(hMaterial[:]))
	hCanon := sha256.Sum256([]byte(canon))
	if h != hex.EncodeToString(hCanon[:]) {
		t.Fatalf("huella Go distinta del contexto canónico SQL")
	}
	for _, alterada := range []ports.SolicitudConsultaReciboRespuesta{
		{OrganizacionRef: "organizacion:otra", ExpedienteRef: base.ExpedienteRef, ComunicacionRef: base.ComunicacionRef},
		{OrganizacionRef: base.OrganizacionRef, ExpedienteRef: "expediente:otro", ComunicacionRef: base.ComunicacionRef},
		{OrganizacionRef: base.OrganizacionRef, ExpedienteRef: base.ExpedienteRef, ComunicacionRef: "comunicacion:otra"},
	} {
		if obtener(alterada) == h {
			t.Fatalf("cambio de ámbito conservó huella: %+v", alterada)
		}
	}
}
