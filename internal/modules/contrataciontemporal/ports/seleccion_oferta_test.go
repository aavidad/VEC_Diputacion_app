package ports

import "testing"

// Vector independiente del canon SQL CT220. Cambiar cualquiera de los bytes
// haría que un reintento dejara de reconocer la reserva original.
func TestClavesSeleccionOfertaCTCompartenCanonConPersistencia(t *testing.T) {
	i := IntencionSeleccionOfertaCT{
		OrganizacionRef: "organizacion:prueba", ExpedienteRef: "expediente:ct:prueba",
		VersionExpediente: 6, ClaveIdempotencia: "018f47a2-6b31-4c80-8a95-4d2e707c5a11",
		ActorRef: "per_0123456789abcdefghijkl", PerfilRef: "prf_0123456789abcdefghijkl",
	}
	if h := HuellaIntencionSeleccionOfertaCT(i); h != "0afd8fffc7ec8d9efa6d3c6e9d7a779d1bcbaf9719efa31ed0af6532c648e0c4" {
		t.Fatalf("huella intención distinta: %s", h)
	}
	if k := ClaveBolsaSeleccionOfertaCT(i); k != "ct-oferta:7e96a8659bc0c5f07febdd6c0888514d235e4ad075983ceb46571099b8976d03" {
		t.Fatalf("clave Bolsa distinta: %s", k)
	}
}
