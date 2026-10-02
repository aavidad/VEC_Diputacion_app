package ports

import (
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

func TestResultadoConsultaCircuitoRRHHAceptaDosHitosDeUnaActuacion(t *testing.T) {
	s := SolicitudConsultaCircuitoRRHH{
		AutenticacionRef: "aut_aaaaaaaaaaaaaaaaaaaaaaaa",
		SesionRef:        "ses_bbbbbbbbbbbbbbbbbbbbbbbb",
		PerfilRef:        "prf_cccccccccccccccccccccccc",
		OrganizacionRef:  "organizacion:dipgra:circuito:001",
		ExpedienteRef:    "expediente:ct:circuito:001",
		VersionObservada: 2,
	}
	d, err := domain.NuevaDefinicionCircuitoRRHH("flujo:ct:rrhh:prueba", 2, "solicitud", []domain.TransicionCircuitoRRHH{
		{Clave: "peticion_firmada", Tipo: domain.HitoPeticionFirmada, Origen: "solicitud", Destino: "autorizacion_rrhh", PerfilClave: "tecnico"},
	})
	if err != nil {
		t.Fatal(err)
	}
	instante := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	hitos := []domain.HitoCircuitoRRHH{
		{Secuencia: 1, VersionExpedienteEntrada: 1, Clave: "peticion_firmada", Tipo: domain.HitoPeticionFirmada,
			Origen: "solicitud", Destino: "autorizacion_rrhh", ReciboRef: "recibo:circuito:001", RegistradoEn: instante},
		{Secuencia: 2, VersionExpedienteEntrada: 1, Clave: "autorizacion_rrhh", Tipo: domain.HitoAutorizacionRRHH,
			Origen: "autorizacion_rrhh", Destino: "credito", ReciboRef: "recibo:circuito:001", RegistradoEn: instante},
	}
	r := ResultadoConsultaCircuitoRRHH{
		ExpedienteRef: s.ExpedienteRef, VersionExpediente: 2, Flujo: d.Flujo,
		Circuito:               domain.CircuitoAdministrativo{Definicion: d.Flujo, EstadoActual: "credito", Hitos: hitos},
		TransicionesPermitidas: []domain.TransicionCircuitoRRHH{},
	}
	if err := r.ValidarPara(s); err != nil {
		t.Fatalf("dos hitos de una actuación real: %v", err)
	}
	r.Circuito.Hitos[1].Origen = "otro_origen"
	if r.ValidarPara(s) == nil {
		t.Fatal("aceptó una historia de circuito sin continuidad")
	}
}
