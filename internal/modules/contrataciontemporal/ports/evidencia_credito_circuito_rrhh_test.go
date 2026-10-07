package ports

import (
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

func TestEvidenciaCreditoCircuitoExigeDefinicionYHuellaDocumental(t *testing.T) {
	definicion, err := domain.NuevaDefinicionCircuitoRRHH(
		"flujo:ct:rrhh:prueba-credito", 2, "solicitud",
		[]domain.TransicionCircuitoRRHH{{
			Clave: "contratacion_temporal.circuito.credito_comprobado",
			Tipo:  domain.HitoCreditoComprobado, Origen: "solicitud", Destino: "oferta",
			RequiereDocumento: true, PerfilClave: "tecnico_rrhh",
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitud := SolicitudEvidenciaCreditoCircuitoRRHH{
		OrganizacionRef: "organizacion:prueba:credito",
		ExpedienteRef:   "expediente:prueba:credito", VersionEntrada: 2,
		Flujo: definicion.Flujo, CreditoRef: "recibo:prueba:credito",
		ActorRef: "actor:rrhh:prueba", PerfilRef: "perfil:rrhh:prueba",
		DocumentoRef: "documento:prueba:credito",
	}
	evidencia := EvidenciaCreditoCircuitoRRHH{
		Solicitud: solicitud, Definicion: definicion,
		DocumentoVersion:      1,
		HuellaDocumentoSHA256: strings.Repeat("a", 64),
	}
	if err := evidencia.ValidarPara(solicitud); err != nil {
		t.Fatalf("evidencia íntegra: %v", err)
	}
	alterada := evidencia
	alterada.HuellaDocumentoSHA256 = ""
	if alterada.ValidarPara(solicitud) == nil {
		t.Fatal("se aceptó un documento sin huella")
	}
	alterada = evidencia
	alterada.Solicitud.DocumentoRef = "documento:prueba:ajeno"
	if alterada.ValidarPara(solicitud) == nil {
		t.Fatal("se aceptó una huella de otro documento")
	}
	alterada = evidencia
	alterada.Definicion.Flujo.HuellaSHA256 = strings.Repeat("b", 64)
	if alterada.ValidarPara(solicitud) == nil {
		t.Fatal("se aceptó una definición con otra huella")
	}
}
