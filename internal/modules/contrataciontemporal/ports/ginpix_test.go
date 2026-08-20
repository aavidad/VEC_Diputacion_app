package ports

import (
	"bytes"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

type mapeadorGINPIXSintetico struct{}

func (mapeadorGINPIXSintetico) Mapear(
	solicitud SolicitudMapeoGINPIX,
) (domain.CargaMapeadaGINPIX, error) {
	modelo, err := solicitud.Modelo()
	if err != nil {
		return domain.CargaMapeadaGINPIX{}, err
	}
	mapeo, err := solicitud.Mapeo()
	if err != nil {
		return domain.CargaMapeadaGINPIX{}, err
	}
	return domain.AplicarMapeoGINPIX(modelo, mapeo)
}

var _ MapeadorGINPIX = mapeadorGINPIXSintetico{}

func TestSolicitudMapeoGINPIXRetieneCopiasYNoActivaEnvio(t *testing.T) {
	modelo := modeloGINPIXSinteticoPrueba(t)
	mapeo := mapeoGINPIXSinteticoPrueba(t)
	solicitud, err := NuevaSolicitudMapeoGINPIX(modelo, mapeo)
	if err != nil || solicitud.Validar() != nil {
		t.Fatalf("crear solicitud neutral: %v", err)
	}

	publicacionModelo := modelo.Publicacion()
	publicacionModelo.Datos[0].Campo.Valor = "MUTADO"
	publicacionMapeo := mapeo.Publicacion()
	publicacionMapeo.Reglas[0].CampoDestino = "mutado"

	modeloRetenido, err := solicitud.Modelo()
	if err != nil {
		t.Fatalf("recuperar modelo retenido: %v", err)
	}
	mapeoRetenido, err := solicitud.Mapeo()
	if err != nil {
		t.Fatalf("recuperar mapeo retenido: %v", err)
	}
	serialModelo, _ := modeloRetenido.SerializarCanonico()
	serialModeloOriginal, _ := modelo.SerializarCanonico()
	serialMapeo, _ := mapeoRetenido.SerializarCanonico()
	serialMapeoOriginal, _ := mapeo.SerializarCanonico()
	if !bytes.Equal(serialModelo, serialModeloOriginal) ||
		!bytes.Equal(serialMapeo, serialMapeoOriginal) {
		t.Fatal("la solicitud expuso estado mutable")
	}

	carga, err := (mapeadorGINPIXSintetico{}).Mapear(solicitud)
	if err != nil {
		t.Fatalf("transformación pura sintética: %v", err)
	}
	datos := carga.Datos()
	if datos.CorrelacionRef != "correlacion_sintetica_ginpix_ports_01" ||
		datos.IdempotenciaRef != "idempotencia_sintetica_ginpix_ports_01" ||
		datos.MapeoVersion != 2 || len(datos.Campos) != 1 ||
		datos.Campos[0].Campo.Valor != "PUESTO-SINT-01" {
		t.Fatalf("resultado neutral perdió ligaduras: %+v", datos)
	}
}

func TestSolicitudMapeoGINPIXDeniegaValoresCero(t *testing.T) {
	if _, err := NuevaSolicitudMapeoGINPIX(
		domain.ModeloCanonicoGINPIX{},
		mapeoGINPIXSinteticoPrueba(t),
	); !errors.Is(err, ErrSolicitudMapeoGINPIXInvalida) {
		t.Fatalf("modelo cero aceptado: %v", err)
	}
	if _, err := NuevaSolicitudMapeoGINPIX(
		modeloGINPIXSinteticoPrueba(t),
		domain.MapeoVersionadoGINPIX{},
	); !errors.Is(err, ErrSolicitudMapeoGINPIXInvalida) {
		t.Fatalf("mapeo cero aceptado: %v", err)
	}
	var solicitud SolicitudMapeoGINPIX
	if !errors.Is(solicitud.Validar(), ErrSolicitudMapeoGINPIXInvalida) {
		t.Fatal("solicitud cero aceptada")
	}
	if _, err := solicitud.Modelo(); !errors.Is(err, ErrSolicitudMapeoGINPIXInvalida) {
		t.Fatalf("solicitud cero expuso modelo: %v", err)
	}
	if _, err := solicitud.Mapeo(); !errors.Is(err, ErrSolicitudMapeoGINPIXInvalida) {
		t.Fatalf("solicitud cero expuso mapeo: %v", err)
	}
}

func modeloGINPIXSinteticoPrueba(t *testing.T) domain.ModeloCanonicoGINPIX {
	t.Helper()
	campo, err := domain.CampoValorGINPIX("PUESTO-SINT-01")
	if err != nil {
		t.Fatalf("crear campo sintético: %v", err)
	}
	modelo, err := domain.NuevoModeloCanonicoGINPIX(
		domain.BorradorModeloCanonicoGINPIX{
			Esquema:           domain.EsquemaModeloCanonicoGINPIXV1,
			VersionExpediente: 4,
			ExpedienteRef:     "expediente_sintetico_ginpix_ports_01",
			IncorporacionRef:  "incorporacion_sintetica_ginpix_ports_01",
			ProcedenciaRef:    "procedencia_sintetica_ginpix_ports_01",
			CorrelacionRef:    "correlacion_sintetica_ginpix_ports_01",
			IdempotenciaRef:   "idempotencia_sintetica_ginpix_ports_01",
			Datos: []domain.DatoCanonicoGINPIX{
				{Clave: "codigo_puesto", Campo: campo},
			},
		},
	)
	if err != nil {
		t.Fatalf("crear modelo sintético: %v", err)
	}
	return modelo
}

func mapeoGINPIXSinteticoPrueba(t *testing.T) domain.MapeoVersionadoGINPIX {
	t.Helper()
	mapeo, err := domain.PublicarMapeoVersionadoGINPIX(
		domain.BorradorMapeoVersionadoGINPIX{
			Esquema:        domain.EsquemaMapeoGINPIXV1,
			Referencia:     "mapeo_sintetico_ginpix_ports_01",
			Version:        2,
			ProcedenciaRef: "procedencia_sintetica_mapeo_ports_01",
			Reglas: []domain.ReglaMapeoGINPIX{
				{
					CampoCanonico: "codigo_puesto",
					CampoDestino:  "puesto",
					Obligatorio:   true,
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("crear mapeo sintético: %v", err)
	}
	return mapeo
}
