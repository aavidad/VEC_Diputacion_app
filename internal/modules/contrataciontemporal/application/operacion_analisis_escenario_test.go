package application

import (
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func nuevoEscenarioOperacionAnalisisSaneado(
	t *testing.T,
	operacion ports.TipoOperacionAnalisis,
	marcaActor string,
) escenarioOperacionAnalisisSaneado {
	t.Helper()
	instante := time.Date(2026, time.July, 23, 12, 0, 0, 0, time.UTC)
	contexto := contextoAutorizacionAltaV3PruebaConMarcas(
		t,
		instante,
		marcaActor,
		marcaActor,
	)
	vinculo, err := contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	funcionales := datosFuncionalesAnalisisSinteticos()
	expediente := expedienteInicialSintetico(t, instante, funcionales)
	if operacion == ports.OperacionRectificarAnalisis {
		solicitudArtefacto := ports.SolicitudPrepararArtefactoAnalisis{
			ArtefactoRef:      "artefacto:analisis-previo-sintetico",
			OrganizacionRef:   expediente.OrganizacionRef,
			ExpedienteRef:     expediente.Referencia,
			VersionExpediente: expediente.Version,
			DatosFuncionales:  funcionales,
			SolicitadaEn:      instante.Add(-20 * time.Minute),
		}
		datos := datosArtefactoAnalisisSintetico(solicitudArtefacto)
		datos.PreparadoEn = solicitudArtefacto.SolicitadaEn
		datos.ValidadaEn = solicitudArtefacto.SolicitadaEn
		datos.CalculadoEn = solicitudArtefacto.SolicitadaEn
		artefacto, err := ports.NuevoArtefactoAnalisisPreparado(
			solicitudArtefacto,
			datos,
		)
		if err != nil {
			t.Fatal(err)
		}
		analisis, err := ports.DerivarAnalisisDesdeArtefacto(
			solicitudArtefacto,
			artefacto,
		)
		if err != nil {
			t.Fatal(err)
		}
		expediente, err = expediente.RegistrarAnalisis(
			expediente.Version,
			analisis,
			domain.DatosActuacion{
				AccionClave: domain.ClaveCatalogo(
					ports.AccionRegistrarAnalisis,
				),
				ActorRef:      "actor:sintetico-anterior-001",
				UnidadRef:     "unidad:rrhh-sintetica-001",
				ReciboRef:     "recibo:analisis-previo-sintetico",
				RealizadaEn:   instante.Add(-15 * time.Minute),
				FaseDestino:   expediente.FaseActual,
				EstadoDestino: expediente.EstadoActual,
			},
		)
		if err != nil {
			t.Fatal(err)
		}
	}
	motivoAutorizacion := dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID:           "motivos_autorizacion",
		CatalogoVersion:      2,
		CatalogoHuellaSHA256: strings.Repeat("5", 64),
		EntradaClave:         "motivo_55555555555555555555555555555555",
	}
	motivoRectificacion := domain.ClaveCatalogo(
		"contratacion_temporal.analisis.rectificacion.ajuste_sintetico",
	)
	return escenarioOperacionAnalisisSaneado{
		instante:            instante,
		contexto:            contexto,
		expediente:          expediente,
		funcionales:         funcionales,
		motivoAutorizacion:  motivoAutorizacion,
		motivoRectificacion: motivoRectificacion,
		registrar: SolicitudRegistrarAnalisis{
			AutenticacionRef:  vinculo.AutenticacionRef,
			SesionRef:         vinculo.SesionRef,
			PerfilRef:         vinculo.PerfilActivoRef,
			OrganizacionRef:   expediente.OrganizacionRef,
			ExpedienteRef:     expediente.Referencia,
			VersionEsperada:   expediente.Version,
			ClaveIdempotencia: "11111111-2222-4333-8444-555555555555",
			ArtefactoRef:      "artefacto:analisis-sintetico-001",
			DatosFuncionales:  funcionales,
		},
		rectificar: SolicitudRectificarAnalisis{
			AutenticacionRef:         vinculo.AutenticacionRef,
			SesionRef:                vinculo.SesionRef,
			PerfilRef:                vinculo.PerfilActivoRef,
			OrganizacionRef:          expediente.OrganizacionRef,
			ExpedienteRef:            expediente.Referencia,
			VersionEsperada:          expediente.Version,
			ClaveIdempotencia:        "66666666-7777-4888-8999-aaaaaaaaaaaa",
			ArtefactoRef:             "artefacto:rectificacion-sintetica-001",
			DatosFuncionales:         funcionales,
			MotivoRectificacionClave: motivoRectificacion,
		},
	}
}

func datosFuncionalesAnalisisSinteticos() ports.DatosFuncionalesOperacionAnalisis {
	return ports.DatosFuncionalesOperacionAnalisis{
		ModalidadClave: "modalidad.sintetica",
		CategoriaRef:   "categoria:sintetica-001",
		GrupoSubgrupo:  "C2",
		CausaClave:     "causa.sintetica",
		Periodo: domain.PeriodoPrevisto{
			Inicio: time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC),
			Fin:    time.Date(2026, time.October, 31, 0, 0, 0, 0, time.UTC),
		},
		PorcentajeJornada: domain.JornadaCompletaDiezmilesimas,
		EntradaRC: domain.VinculoEntradaRC{
			Referencia:   "entrada:rc-sintetica-001",
			HuellaSHA256: strings.Repeat("6", 64),
		},
	}
}

func expedienteInicialSintetico(
	t *testing.T,
	instante time.Time,
	funcionales ports.DatosFuncionalesOperacionAnalisis,
) domain.Expediente {
	t.Helper()
	expediente, err := domain.NuevoExpediente(domain.AltaExpediente{
		Referencia:      "expediente:temporal-sintetico-001",
		OrganizacionRef: "organizacion:sintetica-001",
		NumeroVisible:   "2026/SINT-0001",
		Flujo: domain.ReferenciaFlujo{
			DefinicionRef: "flujo:temporal-sintetico-001",
			Version:       1,
			HuellaSHA256:  strings.Repeat("7", 64),
		},
		FaseInicial: "recepcion_sintetica",
		Solicitud: domain.SolicitudCentro{
			CentroRef:     "centro:sintetico-001",
			ContactoRef:   "contacto:sintetico-001",
			CategoriaRef:  funcionales.CategoriaRef,
			GrupoSubgrupo: funcionales.GrupoSubgrupo,
			MotivoClave:   "motivo.sintetico",
			Detalle:       "Contenido funcional completamente sintético.",
			Periodo:       funcionales.Periodo,
			RC:            domain.DeclaracionRC{Existe: false},
			DocumentosAdjuntos: []string{
				"documento:sintetico-001",
			},
		},
		Actuacion: domain.DatosActuacion{
			AccionClave:   "solicitud.sintetica_registrada",
			ActorRef:      "actor:sintetico-inicial-001",
			UnidadRef:     "unidad:sintetica-inicial-001",
			ReciboRef:     "recibo:alta-sintetica-001",
			RealizadaEn:   instante.Add(-30 * time.Minute),
			FaseDestino:   "recepcion_sintetica",
			EstadoDestino: domain.EstadoEnCurso,
			DocumentosRef: []string{"documento:sintetico-001"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return expediente
}

func datosArtefactoAnalisisSintetico(
	solicitud ports.SolicitudPrepararArtefactoAnalisis,
) ports.DatosArtefactoAnalisis {
	fechaRC := time.Date(2026, time.July, 22, 0, 0, 0, 0, time.UTC)
	importeRC := domain.Importe{Centimos: 5_000_000, Moneda: "EUR"}
	coste := domain.Importe{Centimos: 4_000_000, Moneda: "EUR"}
	return ports.DatosArtefactoAnalisis{
		ArtefactoRef:          solicitud.ArtefactoRef,
		ArtefactoHuellaSHA256: strings.Repeat("8", 64),
		OrganizacionRef:       solicitud.OrganizacionRef,
		ExpedienteRef:         solicitud.ExpedienteRef,
		VersionExpediente:     solicitud.VersionExpediente,
		DatosFuncionales:      solicitud.DatosFuncionales,
		ResultadoRC:           domain.RCValidada,
		FuenteRCRef:           "fuente:rc-sintetica-001",
		ReciboRCRef:           "recibo:rc-sintetico-001",
		ValidadaEn:            solicitud.SolicitadaEn,
		FechaRC:               &fechaRC,
		NumeroRC:              "rc:sintetica-001",
		ImporteRC:             &importeRC,
		DocumentoRCRef:        "documento:rc-sintetico-001",
		CostePrevisto:         &coste,
		FuenteCosteRef:        "fuente:coste-sintetica-001",
		ReciboCosteRef:        "recibo:coste-sintetico-001",
		CalculadoEn:           solicitud.SolicitadaEn,
		PreparadoEn:           solicitud.SolicitadaEn,
	}
}
