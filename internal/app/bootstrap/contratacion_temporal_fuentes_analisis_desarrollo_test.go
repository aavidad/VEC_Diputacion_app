package bootstrap

import (
	"context"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestFuentesAnalisisDesarrolloPreparanCincoModalidadesAtestadas(t *testing.T) {
	derivador := nuevoDerivadorIdempotenciaPrueba(t, 2, 1)
	capacidad, err :=
		nuevoPreparadorFuentesAnalisisContratacionTemporalDesarrollo(
			derivador,
			relojContratacionTemporalDesarrollo{},
		)
	if err != nil {
		t.Fatal(err)
	}
	for indice, modalidad := range modalidadesAnalisisContratacionTemporalDesarrollo {
		t.Run(string(modalidad), func(t *testing.T) {
			solicitud := solicitudPrepararArtefactoAnalisisDesarrolloPrueba(
				modalidad,
				"expediente:ct:desarrollo:analisis:"+numeroDecimal(uint32(indice+1)),
			)
			artefacto, err := capacidad.PrepararArtefactoAnalisis(
				t.Context(),
				solicitud,
			)
			if err != nil {
				t.Fatal(err)
			}
			datos, err := artefacto.DatosPara(solicitud)
			if err != nil {
				t.Fatal(err)
			}
			if datos.ResultadoRC != domain.RCValidada ||
				datos.NumeroRC != numeroRCAnalisisDesarrollo ||
				datos.DocumentoRCRef != documentoRCAnalisisDesarrollo ||
				datos.ImporteRC == nil ||
				datos.ImporteRC.Centimos != centimosRCAnalisisDesarrollo ||
				datos.CostePrevisto == nil ||
				datos.CostePrevisto.Centimos != centimosCosteAnalisisDesarrollo ||
				datos.DatosFuncionales.ModalidadClave != modalidad ||
				datos.DatosFuncionales.EntradaRC !=
					solicitud.DatosFuncionales.EntradaRC {
				t.Fatalf("artefacto inesperado: %+v", datos)
			}
			if datos.FuenteRCRef != autoridadFuenteRCAnalisisDesarrollo ||
				datos.FuenteCosteRef != autoridadCosteAnalisisDesarrollo ||
				datos.AutoridadVerificadorRC.AutoridadRef !=
					autoridadVerificadorAnalisisDesarrollo ||
				datos.AutoridadPublicadorRC.AutoridadRef !=
					autoridadPublicadorAnalisisDesarrollo {
				t.Fatalf("autoridades inesperadas: %+v", datos)
			}
		})
	}
}

func TestFuentesAnalisisDesarrolloRechazanCatalogoManipulado(t *testing.T) {
	derivador := nuevoDerivadorIdempotenciaPrueba(t, 2, 1)
	capacidad, err :=
		nuevoPreparadorFuentesAnalisisContratacionTemporalDesarrollo(
			derivador,
			relojContratacionTemporalDesarrollo{},
		)
	if err != nil {
		t.Fatal(err)
	}
	base := solicitudPrepararArtefactoAnalisisDesarrolloPrueba(
		"sustitucion",
		"expediente:ct:desarrollo:analisis:rechazos",
	)
	casos := []struct {
		nombre  string
		alterar func(*ports.SolicitudPrepararArtefactoAnalisis)
	}{
		{
			nombre: "modalidad",
			alterar: func(s *ports.SolicitudPrepararArtefactoAnalisis) {
				s.DatosFuncionales.ModalidadClave = "otra_modalidad"
			},
		},
		{
			nombre: "categoria",
			alterar: func(s *ports.SolicitudPrepararArtefactoAnalisis) {
				s.DatosFuncionales.CategoriaRef = "categoria:desarrollo:otra"
			},
		},
		{
			nombre: "grupo",
			alterar: func(s *ports.SolicitudPrepararArtefactoAnalisis) {
				s.DatosFuncionales.GrupoSubgrupo = "A1"
			},
		},
		{
			nombre: "causa",
			alterar: func(s *ports.SolicitudPrepararArtefactoAnalisis) {
				s.DatosFuncionales.CausaClave = "otra_causa"
			},
		},
		{
			nombre: "referencia RC",
			alterar: func(s *ports.SolicitudPrepararArtefactoAnalisis) {
				s.DatosFuncionales.EntradaRC.Referencia = "rc:desarrollo:otra"
			},
		},
		{
			nombre: "huella RC",
			alterar: func(s *ports.SolicitudPrepararArtefactoAnalisis) {
				s.DatosFuncionales.EntradaRC.HuellaSHA256 =
					huellaAltaContratacionTemporalDesarrollo("otra-rc")
			},
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			solicitud := base
			caso.alterar(&solicitud)
			if _, err := capacidad.PrepararArtefactoAnalisis(
				t.Context(),
				solicitud,
			); err == nil {
				t.Fatal("la entrada no gobernada fue aceptada")
			}
		})
	}
}

func TestMaterialFuentesAnalisisDesarrolloEsEstableYSeparado(t *testing.T) {
	primero := nuevoDerivadorIdempotenciaPrueba(t, 2, 1)
	segundo := nuevoDerivadorIdempotenciaPrueba(t, 2, 1)
	semillaRC, generacionRC, err := derivarSemillaAutoridadAnalisisDesarrollo(
		primero,
		"fuente-rc",
	)
	if err != nil {
		t.Fatal(err)
	}
	defer borrarBytes(semillaRC[:])
	semillaReinicio, generacionReinicio, err :=
		derivarSemillaAutoridadAnalisisDesarrollo(segundo, "fuente-rc")
	if err != nil {
		t.Fatal(err)
	}
	defer borrarBytes(semillaReinicio[:])
	semillaCoste, generacionCoste, err :=
		derivarSemillaAutoridadAnalisisDesarrollo(primero, "calculo-coste")
	if err != nil {
		t.Fatal(err)
	}
	defer borrarBytes(semillaCoste[:])
	if semillaRC != semillaReinicio ||
		generacionRC != generacionReinicio ||
		generacionRC != generacionCoste {
		t.Fatal("el material estable cambio entre reconstrucciones")
	}
	if semillaRC == semillaCoste {
		t.Fatal("fuente RC y calculador comparten clave")
	}
	claveActiva, err := derivarClaveRespuestaAnalisisDesarrollo(
		primero,
		generacionRC,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer borrarBytes(claveActiva[:])
	claveRetenida, err := derivarClaveRespuestaAnalisisDesarrollo(primero, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer borrarBytes(claveRetenida[:])
	if claveActiva == claveRetenida {
		t.Fatal("dos generaciones comparten clave de respuesta")
	}
}

func TestDependenciasAnalisisDesarrolloFallanCerradasSinAlta(t *testing.T) {
	derivador := nuevoDerivadorIdempotenciaPrueba(t, 2, 1)
	if servicio, err := nuevasDependenciasAnalisisContratacionTemporalDesarrollo(
		nil,
		derivador,
		relojContratacionTemporalDesarrollo{},
	); err == nil || servicio != nil {
		t.Fatalf("servicio=%v error=%v", servicio, err)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := (generadorPeticionFuenteAnalisisDesarrollo{}).
		NuevaReferenciaPeticionFuenteAnalisis(
			ctx,
			ports.TipoPeticionValidacionRC,
		); err == nil {
		t.Fatal("el generador ignoro la cancelacion")
	}
}

func solicitudPrepararArtefactoAnalisisDesarrolloPrueba(
	modalidad domain.ClaveCatalogo,
	expedienteRef string,
) ports.SolicitudPrepararArtefactoAnalisis {
	return ports.SolicitudPrepararArtefactoAnalisis{
		ArtefactoRef:      artefactoAnalisisContratacionTemporalDesarrollo,
		OrganizacionRef:   organizacionAltaContratacionTemporalDesarrollo,
		ExpedienteRef:     expedienteRef,
		VersionExpediente: 1,
		DatosFuncionales: ports.DatosFuncionalesOperacionAnalisis{
			ModalidadClave: modalidad,
			CategoriaRef:   categoriaAltaContratacionTemporalDesarrollo,
			GrupoSubgrupo:  grupoSubgrupoAltaContratacionTemporalDesarrollo,
			CausaClave:     causaAnalisisContratacionTemporalDesarrollo,
			Periodo: domain.PeriodoPrevisto{
				Inicio: time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC),
				Fin:    time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
			},
			PorcentajeJornada: domain.JornadaCompletaDiezmilesimas,
			EntradaRC: domain.VinculoEntradaRC{
				Referencia:   entradaRCAnalisisContratacionTemporalDesarrollo,
				HuellaSHA256: huellaEntradaRCAnalisisContratacionTemporalDesarrollo,
			},
		},
		SolicitadaEn: time.Now().UTC().Truncate(time.Microsecond),
	}
}
