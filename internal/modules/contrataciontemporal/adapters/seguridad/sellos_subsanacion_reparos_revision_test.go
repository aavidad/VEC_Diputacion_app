package seguridad

import (
	"context"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestSellosSubsanacionReparoLigaduraSemantica(t *testing.T) {
	ambito, _ := configuracionAsignacionPrueba(t, ports.DominioAmbitoIdempotenciaSubsanacionReparo, 1)
	huella, _ := configuracionAsignacionPrueba(t, ports.DominioHuellaPeticionSubsanacionReparo, 1)
	autoridad, err := NuevaAutoridadSellosSubsanacionReparoHMAC(ambito, nil, huella, nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	solicitud := solicitudAmbitoAsignacionPrueba()
	material := ports.MaterialSubsanacionReparo{
		OrganizacionRef: solicitud.OrganizacionRef, ActorRef: solicitud.ActorRef,
		PerfilRef: solicitud.PerfilRef, ClaveIdempotencia: solicitud.ClaveIdempotencia,
		ExpedienteRef: "expediente:ct:subsanacion-sintetica", VersionEsperada: 7,
		Observaciones: "Corrección sintética de documentación.",
	}
	derivar := func(m ports.MaterialSubsanacionReparo) string {
		t.Helper()
		sellos, err := autoridad.DerivarHuellaSubsanacionReparo(ctx, m)
		if err != nil || sellos.ValidarDominio(ports.DominioHuellaPeticionSubsanacionReparo) != nil {
			t.Fatalf("huella de petición inválida: %v", err)
		}
		datos, err := sellos.Datos()
		if err != nil {
			t.Fatal(err)
		}
		return datos.Activo.Valor
	}
	primera := derivar(material)
	if derivar(material) != primera {
		t.Fatal("el mismo material no conserva su huella")
	}
	cambio := material
	cambio.Observaciones = "Otra corrección sintética de documentación."
	if derivar(cambio) == primera {
		t.Fatal("cambiar observaciones conserva indebidamente la huella")
	}
	sellar := func(s ports.SolicitudSellarAmbitoIdempotencia) string {
		t.Helper()
		sellos, err := autoridad.SellarAmbitoSubsanacionReparo(ctx, s)
		if err != nil || sellos.ValidarDominio(ports.DominioAmbitoIdempotenciaSubsanacionReparo) != nil {
			t.Fatalf("ámbito inválido: %v", err)
		}
		datos, err := sellos.Datos()
		if err != nil {
			t.Fatal(err)
		}
		return datos.Activo.Valor
	}
	ambitoOriginal := sellar(solicitud)
	if ambitoOriginal == primera {
		t.Fatal("ámbito y petición no separan dominios")
	}
	solicitud.ClaveIdempotencia = "87654321-4321-4abc-8def-1234567890ab"
	material.ClaveIdempotencia = solicitud.ClaveIdempotencia
	if sellar(solicitud) == ambitoOriginal || derivar(material) != primera {
		t.Fatal("la clave de idempotencia no queda ligada exclusivamente al ámbito")
	}
}
