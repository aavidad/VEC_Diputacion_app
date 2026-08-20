package ports

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

const huellaPersonalRPTPrueba = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func solicitudAltaPersonalRPTPrueba() SolicitudAltaPersonalRPT {
	return SolicitudAltaPersonalRPT{
		Esquema:           EsquemaAltaPersonalRPT,
		ContratoVersion:   VersionContratoAltaPersonalRPT,
		SolicitudRef:      "solicitud:personal:0001",
		ExpedienteRef:     "expediente:temporal:0001",
		VersionExpediente: 7,
		CapacidadRef:      "capacidad:personal:0001",
		CorrelacionRef:    "correlacion:personal:0001",
		IdempotenciaRef:   "idempotencia:personal:0001",
		FuenteRPT: ReferenciaVersionadaPersonalRPT{
			Referencia:   "rpt:publicada:0001",
			Version:      3,
			HuellaSHA256: huellaPersonalRPTPrueba,
		},
		PuestoRef: "puesto:rpt:0001",
		PlazaRef:  "plaza:rpt:0001",
	}
}

func resultadoAltaPersonalRPTPrueba(
	t *testing.T,
	solicitud SolicitudAltaPersonalRPT,
) ResultadoAltaPersonalRPT {
	t.Helper()
	huella, err := solicitud.HuellaSHA256()
	if err != nil {
		t.Fatalf("crear huella de solicitud: %v", err)
	}
	return ResultadoAltaPersonalRPT{
		Esquema:               EsquemaAltaPersonalRPT,
		ContratoVersion:       VersionContratoAltaPersonalRPT,
		ResultadoRef:          "resultado:personal:0001",
		ReciboRef:             "recibo:personal:0001",
		SolicitudRef:          solicitud.SolicitudRef,
		CorrelacionRef:        solicitud.CorrelacionRef,
		IdempotenciaRef:       solicitud.IdempotenciaRef,
		HuellaSolicitudSHA256: huella,
		Estado:                AltaPersonalRPTConfirmada,
		RelacionRef:           "relacion:juridica:0001",
		OcupacionRef:          "ocupacion:personal:0001",
	}
}

func TestO701SolicitudPersonalRPTCanonicaDeterminista(t *testing.T) {
	solicitud := solicitudAltaPersonalRPTPrueba()
	primero, err := solicitud.MaterialCanonico()
	if err != nil {
		t.Fatalf("material válido rechazado: %v", err)
	}
	segundo, err := solicitud.MaterialCanonico()
	if err != nil || !bytes.Equal(primero, segundo) {
		t.Fatalf("material no determinista: %v", err)
	}
	primero[0] ^= 0xff
	tercero, err := solicitud.MaterialCanonico()
	if err != nil || bytes.Equal(primero, tercero) {
		t.Fatalf("el material no se devolvió mediante copia defensiva: %v", err)
	}
	huella, err := solicitud.HuellaSHA256()
	if err != nil || len(huella) != 64 {
		t.Fatalf("huella canónica inválida: %q %v", huella, err)
	}
	mutada := solicitud
	mutada.PlazaRef = "plaza:rpt:0002"
	huellaMutada, err := mutada.HuellaSHA256()
	if err != nil || huellaMutada == huella {
		t.Fatalf("la huella no ligó la plaza: %v", err)
	}
	if strings.Contains(string(tercero), "nombre") ||
		strings.Contains(string(tercero), "dni") {
		t.Fatal("el contrato canónico contiene datos personales no autorizados")
	}
}

func TestO701SolicitudPersonalRPTRechazaAusenciaYEsquemaDesconocido(t *testing.T) {
	casos := map[string]func(*SolicitudAltaPersonalRPT){
		"esquema desconocido": func(s *SolicitudAltaPersonalRPT) { s.Esquema = "v2" },
		"sin expediente":      func(s *SolicitudAltaPersonalRPT) { s.ExpedienteRef = "" },
		"sin capacidad":       func(s *SolicitudAltaPersonalRPT) { s.CapacidadRef = "" },
		"sin idempotencia":    func(s *SolicitudAltaPersonalRPT) { s.IdempotenciaRef = "" },
		"sin version":         func(s *SolicitudAltaPersonalRPT) { s.VersionExpediente = 0 },
		"rpt sin huella":      func(s *SolicitudAltaPersonalRPT) { s.FuenteRPT.HuellaSHA256 = "" },
		"sin puesto":          func(s *SolicitudAltaPersonalRPT) { s.PuestoRef = "" },
		"sin plaza":           func(s *SolicitudAltaPersonalRPT) { s.PlazaRef = "" },
	}
	for nombre, mutar := range casos {
		t.Run(nombre, func(t *testing.T) {
			solicitud := solicitudAltaPersonalRPTPrueba()
			mutar(&solicitud)
			if err := solicitud.Validar(); !errors.Is(
				err,
				ErrSolicitudAltaPersonalRPTInvalida,
			) {
				t.Fatalf("solicitud inválida aceptada: %v", err)
			}
			if _, err := solicitud.MaterialCanonico(); !errors.Is(
				err,
				ErrSolicitudAltaPersonalRPTInvalida,
			) {
				t.Fatalf("solicitud inválida serializada: %v", err)
			}
		})
	}
}

func TestO701ResultadoPersonalRPTConfirmaORechazaDeFormaCerrada(t *testing.T) {
	solicitud := solicitudAltaPersonalRPTPrueba()
	confirmado := resultadoAltaPersonalRPTPrueba(t, solicitud)
	if err := confirmado.ValidarPara(solicitud); err != nil {
		t.Fatalf("confirmación válida rechazada: %v", err)
	}

	rechazado := confirmado
	rechazado.Estado = AltaPersonalRPTRechazada
	rechazado.RelacionRef = ""
	rechazado.OcupacionRef = ""
	rechazado.MotivoRechazo = ReferenciaVersionadaPersonalRPT{
		Referencia:   "catalogo:rechazos-personal:incompatible",
		Version:      2,
		HuellaSHA256: huellaPersonalRPTPrueba,
	}
	if err := rechazado.ValidarPara(solicitud); err != nil {
		t.Fatalf("rechazo gobernado válido rechazado: %v", err)
	}

	casos := map[string]func(*ResultadoAltaPersonalRPT){
		"esquema desconocido": func(r *ResultadoAltaPersonalRPT) {
			r.Esquema = "vec.contratacion-temporal.personal-rpt.alta.v2"
		},
		"discordancia": func(r *ResultadoAltaPersonalRPT) {
			r.CorrelacionRef = "correlacion:personal:otra"
		},
		"huella forjada": func(r *ResultadoAltaPersonalRPT) {
			r.HuellaSolicitudSHA256 = strings.Repeat("b", 64)
		},
		"confirmacion sin relacion": func(r *ResultadoAltaPersonalRPT) {
			r.RelacionRef = ""
		},
		"confirmacion con rechazo": func(r *ResultadoAltaPersonalRPT) {
			r.MotivoRechazo = rechazado.MotivoRechazo
		},
		"estado desconocido": func(r *ResultadoAltaPersonalRPT) {
			r.Estado = "pendiente"
		},
	}
	for nombre, mutar := range casos {
		t.Run(nombre, func(t *testing.T) {
			resultado := confirmado
			mutar(&resultado)
			if err := resultado.ValidarPara(solicitud); !errors.Is(
				err,
				ErrResultadoAltaPersonalRPTInvalido,
			) {
				t.Fatalf("resultado discordante aceptado: %v", err)
			}
		})
	}

	rechazoConOcupacion := rechazado
	rechazoConOcupacion.OcupacionRef = "ocupacion:personal:forjada"
	if err := rechazoConOcupacion.ValidarPara(solicitud); !errors.Is(
		err,
		ErrResultadoAltaPersonalRPTInvalido,
	) {
		t.Fatalf("rechazo con ocupación aceptado: %v", err)
	}
	rechazoSinMotivo := rechazado
	rechazoSinMotivo.MotivoRechazo = ReferenciaVersionadaPersonalRPT{}
	if err := rechazoSinMotivo.ValidarPara(solicitud); !errors.Is(
		err,
		ErrResultadoAltaPersonalRPTInvalido,
	) {
		t.Fatalf("rechazo sin motivo gobernado aceptado: %v", err)
	}
}
