package confianzaatestacion

import (
	"bytes"
	"context"
	"testing"
	"time"

	dominiocontratacion "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	puertoscontratacion "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type registroConcesionCapacidadV3Prueba struct {
	registradaEn time.Time
}

func (r registroConcesionCapacidadV3Prueba) RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(
	context.Context,
	puertosvec.OrdenRegistroConcesionCandidataAutorizacionLigadaV3,
) (time.Time, error) {
	return r.registradaEn, nil
}

func TestCapacidadAtestacionV3EsCompatibleConOrdenAltaO204Real(
	t *testing.T,
) {
	escenario, prueba := escenarioYPruebaConfianzaV3(t)
	datosSolicitud, err := escenario.solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	ambito := datosSolicitud.Recurso.Referencia
	huellaPeticion := datosSolicitud.Recurso.
		Atributos[puertoscontratacion.AtributoHuellaPeticionHMACActiva]
	ambitos, err := puertoscontratacion.NuevaColeccionSellosHMAC(
		ambito,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	huellas, err := puertoscontratacion.NuevaColeccionSellosHMAC(
		huellaPeticion,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	ordenRegistro, err :=
		puertosvec.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(
			escenario.solicitud,
			escenario.decision,
			escenario.motivo,
			escenario.resultado,
		)
	if err != nil {
		t.Fatal(err)
	}
	confirmacion, err :=
		puertosvec.RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(
			context.Background(),
			registroConcesionCapacidadV3Prueba{
				registradaEn: escenario.ahora,
			},
			ordenRegistro,
		)
	if err != nil {
		t.Fatal(err)
	}
	vinculo, err := datosSolicitud.VinculoAutenticacionActor.Datos()
	if err != nil {
		t.Fatal(err)
	}
	flujo := dominiocontratacion.ReferenciaFlujo{
		DefinicionRef: datosSolicitud.Recurso.Atributos["flujo_ref"],
		Version:       1,
		HuellaSHA256: datosSolicitud.Recurso.
			Atributos["flujo_huella_sha256"],
	}
	solicitudCentro := dominiocontratacion.SolicitudCentro{
		CentroRef:     datosSolicitud.Recurso.Ambitos["centro_ref"],
		ContactoRef:   "persona:contacto-centro-001",
		CategoriaRef:  datosSolicitud.Recurso.Ambitos["categoria_ref"],
		GrupoSubgrupo: "C2",
		MotivoClave:   "sustitucion_temporal",
		Detalle:       "Necesidad temporal validada para prueba de contrato.",
		Periodo: dominiocontratacion.PeriodoPrevisto{
			Inicio: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
			Fin:    time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC),
		},
		DocumentosAdjuntos: []string{},
	}
	preparacion := puertoscontratacion.PreparacionAlta{
		ReservaRef: "reserva:ct-alta:001",
		Referencias: puertoscontratacion.ReferenciasAlta{
			ExpedienteRef: "expediente:ct:001",
			NumeroVisible: "2026/CT-001",
			ReciboRef:     "recibo:ct-alta:001",
		},
		AmbitoIdempotenciaHMAC: ambito,
		HuellaPeticionHMAC:     huellaPeticion,
		OrganizacionRef: datosSolicitud.Recurso.
			Ambitos["organizacion_ref"],
		ActorRef:  vinculo.PrincipalID,
		PerfilRef: vinculo.PerfilActivoRef,
		Estado:    puertoscontratacion.PreparacionReservada,
	}
	expediente, err := dominiocontratacion.NuevoExpediente(
		dominiocontratacion.AltaExpediente{
			Referencia:      preparacion.Referencias.ExpedienteRef,
			OrganizacionRef: preparacion.OrganizacionRef,
			NumeroVisible:   preparacion.Referencias.NumeroVisible,
			Flujo:           flujo,
			FaseInicial:     "solicitud_registrada",
			Solicitud:       solicitudCentro,
			Actuacion: dominiocontratacion.DatosActuacion{
				AccionClave:   "registrar_solicitud",
				ActorRef:      vinculo.PrincipalID,
				UnidadRef:     "unidad:recursos-humanos",
				ReciboRef:     preparacion.Referencias.ReciboRef,
				RealizadaEn:   escenario.ahora,
				FaseDestino:   "solicitud_registrada",
				EstadoDestino: dominiocontratacion.EstadoEnCurso,
				DocumentosRef: []string{},
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := puertoscontratacion.NuevaOrdenConfirmarAlta(
		puertoscontratacion.DatosOrdenConfirmarAlta{
			Expediente:              expediente,
			SolicitudAutorizacionV3: escenario.solicitud,
			DecisionAutorizacionV3:  escenario.decision,
			ConfirmacionRegistroV3:  confirmacion,
			AmbitosIdempotenciaHMAC: ambitos,
			HuellasPeticionHMAC:     huellas,
			Preparacion:             preparacion,
		},
	); err != nil {
		t.Fatalf("NuevaOrdenConfirmarAlta rechazó el perfil O2-04: %v", err)
	}

	clave := claveCapacidadAtestacionV3Prueba(
		t,
		EstadoClaveHMACCapacidadAtestacionV3Emision,
		time.Time{},
		bytes.Repeat([]byte{0x58}, 32),
	)
	emisor, err := nuevoEmisorCapacidadesAtestacionAutorizacionV3(
		clave,
		&relojConfianzaAtestacionV3Prueba{
			ahora: escenario.ahora.Add(time.Microsecond),
		},
		bytes.NewReader(bytes.Repeat([]byte{0x98}, 32)),
	)
	if err != nil {
		t.Fatal(err)
	}
	capacidad, err := emisor.Emitir(
		context.Background(),
		escenario.solicitud,
		escenario.decision,
		escenario.motivo,
		escenario.resultado,
		escenario.atestacion,
		prueba,
	)
	if err != nil {
		t.Fatalf("el emisor rechazó la solicitud O2-04 real: %v", err)
	}
	exportacion, err := capacidad.ExportacionCanonicaParaConsumidor()
	if err != nil {
		t.Fatal(err)
	}
	documento, err := interpretarExportacionCapacidadV3(exportacion)
	if err != nil {
		t.Fatal(err)
	}
	huellaContexto, err := datosSolicitud.Recurso.
		HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	if documento.EfectoRef != ambito ||
		documento.HuellaEfectoSHA256 != huellaContexto {
		t.Fatalf("efecto no derivado del recurso O2-04: %+v", documento)
	}
}
