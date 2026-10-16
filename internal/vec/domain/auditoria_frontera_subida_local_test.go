package domain

import "testing"

func TestAuditoriaFronteraSubidaLocalUsaRutasCanonicas(t *testing.T) {
	base := OrdenAuditoriaFronteraRutaExacta{
		CorrelacionRef: "corr_no_disponible",
		Motivo:         MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida,
		Superficie:     SuperficieAuditoriaFronteraRutaExactaDocumentosSubidaLocal,
		Ruta:           "/api/vec/meritos/cartera/comprobantes",
	}
	if err := base.Validar(); err != nil {
		t.Fatal(err)
	}
	base.Ruta = "/api/vec/meritos/cartera/comprobantes/estado"
	if err := base.Validar(); err != nil {
		t.Fatal(err)
	}
	base.Ruta = "/api/vec/meritos/cartera/comprobantes/{comprobante_ref}/original"
	if err := base.Validar(); err != nil {
		t.Fatal(err)
	}
	base.Ruta = "/api/vec/meritos/cartera/comprobantes/ref%3Adoc/original"
	if err := base.Validar(); err == nil {
		t.Fatal("aceptó referencia del documento en auditoría")
	}
	base.Ruta = "/api/vec/meritos/cartera/comprobantes"
	base.Motivo = MotivoAuditoriaFronteraRutaExactaAccesoDenegado
	base.ActorRef = "docactor_" + "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if err := base.Validar(); err != nil {
		t.Fatal(err)
	}
	base.ActorRef = "per_identidad"
	if err := base.Validar(); err == nil {
		t.Fatal("aceptó persona en vez de seudónimo")
	}
}
