package postgres

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestCanonConfirmacionAltaCoincideConOrdenCongeladoO205(t *testing.T) {
	expediente := expedienteConfirmacionPrueba(t)
	ambitos, err := ports.NuevaColeccionSellosHMAC(
		"hmac-sha256:vec.contratacion-temporal.ambito-idempotencia/v2:"+
			strings.Repeat("b", 64),
		[]string{
			"hmac-sha256:vec.contratacion-temporal.ambito-idempotencia/v1:" +
				strings.Repeat("a", 64),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	huellas, err := ports.NuevaColeccionSellosHMAC(
		"hmac-sha256:vec.contratacion-temporal.huella-peticion/v2:"+
			strings.Repeat("d", 64),
		[]string{
			"hmac-sha256:vec.contratacion-temporal.huella-peticion/v1:" +
				strings.Repeat("c", 64),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	candidatura := ports.CandidaturaAlta{
		ReservaRef: "reserva:alta-o2-06-001",
		Referencias: ports.ReferenciasAlta{
			ExpedienteRef: expediente.Referencia,
			NumeroVisible: expediente.NumeroVisible,
			ReciboRef:     expediente.Actuaciones[0].ReciboRef,
		},
		AmbitoIdempotenciaHMAC: selloActivoPrueba(t, ambitos),
		HuellaPeticionHMAC:     selloActivoPrueba(t, huellas),
		OrganizacionRef:        expediente.OrganizacionRef,
		ActorRef:               expediente.Actuaciones[0].ActorRef,
		PerfilRef:              "perfil:tecnica-rrhh-001",
	}
	solicitud := ports.SolicitudProyectarEfectoAlta{
		Expediente:  expediente,
		Candidatura: candidatura,
	}
	evidencia := ports.EvidenciaOrdenConfirmarAlta{
		Expediente:              expediente,
		Candidatura:             candidatura,
		AmbitosIdempotenciaHMAC: ambitos,
		HuellasPeticionHMAC:     huellas,
	}
	alta, err := canonEfectoAltaV2(solicitud)
	if err != nil {
		t.Fatal(err)
	}
	suma := sha256.Sum256(alta)
	huella := hex.EncodeToString(suma[:])
	const huellaCongelada = "087ca760e24465f2633e5532d86ec08e04ad4b15c011a1cb8e17c270aee9f324"
	if huella != huellaCongelada {
		t.Fatalf("huella canónica O2-05=%s; contenido=%s", huella, alta)
	}
	sellos, err := canonSellosAltaV1(evidencia)
	if err != nil {
		t.Fatal(err)
	}
	esperados := `{"esquema":"vec.contratacion-temporal.sellos-hmac.v1",` +
		`"activo":{"generacion":2,` +
		`"ambito_hmac":"hmac-sha256:vec.contratacion-temporal.` +
		`ambito-idempotencia/v2:` + strings.Repeat("b", 64) + `",` +
		`"huella_hmac":"hmac-sha256:vec.contratacion-temporal.` +
		`huella-peticion/v2:` + strings.Repeat("d", 64) + `"},` +
		`"retenidos":[{"generacion":1,` +
		`"ambito_hmac":"hmac-sha256:vec.contratacion-temporal.` +
		`ambito-idempotencia/v1:` + strings.Repeat("a", 64) + `",` +
		`"huella_hmac":"hmac-sha256:vec.contratacion-temporal.` +
		`huella-peticion/v1:` + strings.Repeat("c", 64) + `"}]}`
	if string(sellos) != esperados {
		t.Fatalf("sellos fuera de contrato: %s", sellos)
	}
}

func selloActivoPrueba(
	t *testing.T,
	coleccion ports.ColeccionSellosHMAC,
) string {
	t.Helper()
	datos, err := coleccion.Datos()
	if err != nil {
		t.Fatal(err)
	}
	return datos.Activo.Valor
}
