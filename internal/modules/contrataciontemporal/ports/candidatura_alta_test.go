package ports

import (
	"errors"
	"testing"
)

func solicitudCandidaturaAltaPrueba(
	t *testing.T,
) SolicitudResolverCandidaturaAlta {
	t.Helper()
	ambitoV2 := selloGeneracionalPrueba(
		"vec.contratacion-temporal.ambito-idempotencia",
		2,
		"b",
	)
	ambitoV1 := selloGeneracionalPrueba(
		"vec.contratacion-temporal.ambito-idempotencia",
		1,
		"a",
	)
	huellaV2 := selloGeneracionalPrueba(
		"vec.contratacion-temporal.huella-peticion",
		2,
		"d",
	)
	huellaV1 := selloGeneracionalPrueba(
		"vec.contratacion-temporal.huella-peticion",
		1,
		"c",
	)
	ambitos, err := NuevaColeccionSellosHMAC(ambitoV2, []string{ambitoV1})
	if err != nil {
		t.Fatal(err)
	}
	huellas, err := NuevaColeccionSellosHMAC(huellaV2, []string{huellaV1})
	if err != nil {
		t.Fatal(err)
	}
	return SolicitudResolverCandidaturaAlta{
		AmbitosIdempotenciaHMAC: ambitos,
		HuellasPeticionHMAC:     huellas,
		OrganizacionRef:         "organizacion:diputacion-granada",
		ActorRef:                "actor:responsable-centro-001",
		PerfilRef:               "perfil:responsable-centro",
		Propuesta: CandidaturaAlta{
			ReservaRef: "reserva:alta-candidata-002",
			Referencias: ReferenciasAlta{
				ExpedienteRef: "expediente:ct-2026-0002",
				NumeroVisible: "2026/CT-0002",
				ReciboRef:     "recibo:alta-candidata-002",
			},
			AmbitoIdempotenciaHMAC: ambitoV2,
			HuellaPeticionHMAC:     huellaV2,
			OrganizacionRef:        "organizacion:diputacion-granada",
			ActorRef:               "actor:responsable-centro-001",
			PerfilRef:              "perfil:responsable-centro",
		},
	}
}

func TestSolicitudCandidaturaAltaAdmiteReplayDeGeneracionRetenida(t *testing.T) {
	solicitud := solicitudCandidaturaAltaPrueba(t)
	if err := solicitud.Validar(); err != nil {
		t.Fatal(err)
	}
	ambitos, err := solicitud.AmbitosIdempotenciaHMAC.Datos()
	if err != nil {
		t.Fatal(err)
	}
	huellas, err := solicitud.HuellasPeticionHMAC.Datos()
	if err != nil {
		t.Fatal(err)
	}
	recuperada := solicitud.Propuesta
	recuperada.AmbitoIdempotenciaHMAC = ambitos.Retenidos[0].Valor
	recuperada.HuellaPeticionHMAC = huellas.Retenidos[0].Valor
	if err := solicitud.ValidarResultado(recuperada); err != nil {
		t.Fatalf("replay retenido rechazado: %v", err)
	}
}

func TestSolicitudCandidaturaAltaNoSeparaParesNiContexto(t *testing.T) {
	solicitud := solicitudCandidaturaAltaPrueba(t)
	ambitos, _ := solicitud.AmbitosIdempotenciaHMAC.Datos()
	huellas, _ := solicitud.HuellasPeticionHMAC.Datos()
	casos := map[string]CandidaturaAlta{
		"generaciones cruzadas": func() CandidaturaAlta {
			candidatura := solicitud.Propuesta
			candidatura.AmbitoIdempotenciaHMAC = ambitos.Retenidos[0].Valor
			candidatura.HuellaPeticionHMAC = huellas.Activo.Valor
			return candidatura
		}(),
		"organización cruzada": func() CandidaturaAlta {
			candidatura := solicitud.Propuesta
			candidatura.OrganizacionRef = "organizacion:ajena"
			return candidatura
		}(),
	}
	for nombre, candidatura := range casos {
		t.Run(nombre, func(t *testing.T) {
			if err := solicitud.ValidarResultado(candidatura); !errors.Is(
				err,
				ErrPreparacionAltaInvalida,
			) {
				t.Fatalf("resultado cruzado aceptado: %v", err)
			}
		})
	}
}

func TestSolicitudCandidaturaAltaExigePropuestaDelParActivo(t *testing.T) {
	solicitud := solicitudCandidaturaAltaPrueba(t)
	ambitos, _ := solicitud.AmbitosIdempotenciaHMAC.Datos()
	solicitud.Propuesta.AmbitoIdempotenciaHMAC =
		ambitos.Retenidos[0].Valor
	if err := solicitud.Validar(); !errors.Is(
		err,
		ErrPreparacionAltaInvalida,
	) {
		t.Fatalf("propuesta no activa aceptada: %v", err)
	}
}
