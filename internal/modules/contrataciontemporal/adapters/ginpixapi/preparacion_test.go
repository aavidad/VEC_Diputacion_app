package ginpixapi

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/ginpixfichero"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const datoPersonalSinteticoPrueba = "DATO-PERSONAL-SINTETICO-NO-REGISTRAR"

func TestPrepararReutilizaO701O702O703YO705SinActivarEnvio(t *testing.T) {
	preparacion, solicitud := preparacionAPIGINPIXPrueba(t)
	cuerpo, err := preparacion.Cuerpo()
	if err != nil {
		t.Fatalf("obtener cuerpo preparado: %v", err)
	}
	modelo, _ := solicitud.Modelo()
	mapeo, _ := solicitud.Mapeo()
	carga, err := domain.AplicarMapeoGINPIX(modelo, mapeo)
	if err != nil {
		t.Fatalf("aplicar mapeo de referencia: %v", err)
	}
	esperado, err := ginpixfichero.Codificar(carga)
	if err != nil || !bytes.Equal(cuerpo, esperado) {
		t.Fatalf("la API no conserva los bytes O7-05: %v", err)
	}
	metadatos, err := preparacion.Metadatos()
	if err != nil || metadatos.ExpedienteRef != "expediente_sintetico_api_0001" ||
		metadatos.IncorporacionRef != "actuacion_incorporacion_api_0001" ||
		metadatos.VersionExpediente != 7 ||
		metadatos.CorrelacionRef != "correlacion_ginpix_api_0001" ||
		metadatos.IdempotenciaRef != "idempotencia_ginpix_api_0001" ||
		metadatos.ReciboIncorporacionRef != "recibo_incorporacion_api_0001" ||
		metadatos.ResultadoPersonalRef != "resultado_personal_api_0001" ||
		metadatos.ReciboPersonalRef != "recibo_personal_api_0001" {
		t.Fatalf("ligaduras incompletas: %+v / %v", metadatos, err)
	}
	if strings.Contains(fmt.Sprint(metadatos), datoPersonalSinteticoPrueba) {
		t.Fatal("los metadatos expusieron la carga funcional")
	}
}

func TestPreparacionClonaCuerpoYEntradasMutables(t *testing.T) {
	preparacion, solicitud := preparacionAPIGINPIXPrueba(t)
	primero, _ := preparacion.Cuerpo()
	primero[0] ^= 0xff
	segundo, err := preparacion.Cuerpo()
	if err != nil || bytes.Equal(primero, segundo) {
		t.Fatalf("el cuerpo conserva alias mutable: %v", err)
	}

	modelo, _ := solicitud.Modelo()
	publicacion := modelo.Publicacion()
	publicacion.Datos[0].Campo.Valor = "MUTADO"
	tercero, err := preparacion.Cuerpo()
	if err != nil || !bytes.Equal(segundo, tercero) {
		t.Fatalf("la preparación retuvo alias del modelo: %v", err)
	}
}

func TestPrepararDeniegaLigadurasIncorporacionAlteradas(t *testing.T) {
	solicitud, solicitudPersonal, resultadoPersonal, recibo := insumosPreparacionAPIGINPIXPrueba(t)
	cases := map[string]func(*ports.ReciboConfirmacionIncorporacion){
		"expediente": func(r *ports.ReciboConfirmacionIncorporacion) {
			r.ExpedienteRef = "expediente_sintetico_api_otro"
		},
		"incorporacion": func(r *ports.ReciboConfirmacionIncorporacion) {
			r.ActuacionRef = "actuacion_incorporacion_api_otra"
		},
		"version OCC":        func(r *ports.ReciboConfirmacionIncorporacion) { r.VersionExpediente++ },
		"recibo Personal":    func(r *ports.ReciboConfirmacionIncorporacion) { r.ReciboPersonalRef = "" },
		"resultado Personal": func(r *ports.ReciboConfirmacionIncorporacion) { r.ResultadoPersonalRef = "" },
		"version resultante": func(r *ports.ReciboConfirmacionIncorporacion) { r.VersionResultante++ },
		"documento duplicado": func(r *ports.ReciboConfirmacionIncorporacion) {
			r.Documentos = append(r.Documentos, r.Documentos[0])
		},
	}
	for nombre, alterar := range cases {
		t.Run(nombre, func(t *testing.T) {
			adulterado := recibo
			adulterado.Documentos = append([]domain.DocumentoSeguimiento(nil), recibo.Documentos...)
			alterar(&adulterado)
			if _, err := Preparar(
				solicitud,
				solicitudPersonal,
				resultadoPersonal,
				adulterado,
			); !errorsIs(err, ErrPreparacionAPIGINPIXInvalida) {
				t.Fatalf("ligadura alterada aceptada: %v", err)
			}
		})
	}
	if _, err := Preparar(
		ports.SolicitudMapeoGINPIX{},
		solicitudPersonal,
		resultadoPersonal,
		recibo,
	); !errorsIs(err, ErrPreparacionAPIGINPIXInvalida) {
		t.Fatalf("solicitud O7-03 cero aceptada: %v", err)
	}
	resultadoPersonal.IdempotenciaRef = "idempotencia_personal_api_alterada"
	if _, err := Preparar(
		solicitud,
		solicitudPersonal,
		resultadoPersonal,
		recibo,
	); !errorsIs(err, ErrPreparacionAPIGINPIXInvalida) {
		t.Fatalf("resultado O7-01 adulterado aceptado: %v", err)
	}
}

func preparacionAPIGINPIXPrueba(t *testing.T) (Preparacion, ports.SolicitudMapeoGINPIX) {
	t.Helper()
	solicitud, solicitudPersonal, resultadoPersonal, recibo := insumosPreparacionAPIGINPIXPrueba(t)
	preparacion, err := Preparar(solicitud, solicitudPersonal, resultadoPersonal, recibo)
	if err != nil {
		t.Fatalf("preparar operación API sintética: %v", err)
	}
	return preparacion, solicitud
}

func insumosPreparacionAPIGINPIXPrueba(
	t *testing.T,
) (
	ports.SolicitudMapeoGINPIX,
	ports.SolicitudAltaPersonalRPT,
	ports.ResultadoAltaPersonalRPT,
	ports.ReciboConfirmacionIncorporacion,
) {
	t.Helper()
	solicitudPersonal := ports.SolicitudAltaPersonalRPT{
		Esquema: ports.EsquemaAltaPersonalRPT, ContratoVersion: ports.VersionContratoAltaPersonalRPT,
		SolicitudRef: "solicitud_personal_api_0001", ExpedienteRef: "expediente_sintetico_api_0001",
		VersionExpediente: 7, CapacidadRef: "capacidad_personal_api_0001",
		CorrelacionRef: "correlacion_personal_api_0001", IdempotenciaRef: "idempotencia_personal_api_0001",
		FuenteRPT: ports.ReferenciaVersionadaPersonalRPT{
			Referencia: "rpt_sintetica_api_0001", Version: 3, HuellaSHA256: strings.Repeat("a", 64),
		},
		PuestoRef: "puesto_sintetico_api_0001", PlazaRef: "plaza_sintetica_api_0001",
	}
	huellaSolicitud, err := solicitudPersonal.HuellaSHA256()
	if err != nil {
		t.Fatalf("validar contrato O7-01: %v", err)
	}
	resultadoPersonal := ports.ResultadoAltaPersonalRPT{
		Esquema: ports.EsquemaAltaPersonalRPT, ContratoVersion: ports.VersionContratoAltaPersonalRPT,
		ResultadoRef: "resultado_personal_api_0001", ReciboRef: "recibo_personal_api_0001",
		SolicitudRef: solicitudPersonal.SolicitudRef, CorrelacionRef: solicitudPersonal.CorrelacionRef,
		IdempotenciaRef: solicitudPersonal.IdempotenciaRef, HuellaSolicitudSHA256: huellaSolicitud,
		Estado:      ports.AltaPersonalRPTConfirmada,
		RelacionRef: "relacion_personal_api_0001", OcupacionRef: "ocupacion_personal_api_0001",
	}
	if err := resultadoPersonal.ValidarPara(solicitudPersonal); err != nil {
		t.Fatalf("resultado O7-01 inválido: %v", err)
	}
	instante := time.Date(2026, 8, 31, 9, 30, 0, 0, time.UTC)
	recibo := ports.ReciboConfirmacionIncorporacion{
		ReciboRef: "recibo_incorporacion_api_0001", ActuacionRef: "actuacion_incorporacion_api_0001",
		CorrelacionRef: "correlacion_incorporacion_api_0001", ActorRef: "actor_rrhh_api_0001",
		OrganizacionRef: "organizacion_api_0001", UnidadRef: "unidad_rrhh_api_0001",
		ExpedienteRef: solicitudPersonal.ExpedienteRef, SolicitudPersonalRef: solicitudPersonal.SolicitudRef,
		DecisionAutorizacionRef: "decision_v3_api_0001",
		ResultadoPersonalRef:    resultadoPersonal.ResultadoRef, ReciboPersonalRef: resultadoPersonal.ReciboRef,
		RelacionRef: resultadoPersonal.RelacionRef, OcupacionRef: resultadoPersonal.OcupacionRef,
		TransicionClave: ports.TransicionConfirmarIncorporacion, MotivoClave: "incorporacion_confirmada",
		VersionExpediente: 7, VersionAnterior: 3, VersionResultante: 4,
		FechaIncorporacion: instante.Add(24 * time.Hour), FechaFinPrevista: instante.AddDate(0, 1, 0),
		ConfirmadaEn: instante,
		Documentos: []domain.DocumentoSeguimiento{{
			TipoClave: "resolucion_incorporacion", Referencia: "documento_incorporacion_api_0001",
		}},
	}
	campo, err := domain.CampoValorGINPIX(datoPersonalSinteticoPrueba)
	if err != nil {
		t.Fatal(err)
	}
	modelo, err := domain.NuevoModeloCanonicoGINPIX(domain.BorradorModeloCanonicoGINPIX{
		Esquema: domain.EsquemaModeloCanonicoGINPIXV1, VersionExpediente: 7,
		ExpedienteRef: recibo.ExpedienteRef, IncorporacionRef: recibo.ActuacionRef,
		ProcedenciaRef: "procedencia_modelo_api_0001", CorrelacionRef: "correlacion_ginpix_api_0001",
		IdempotenciaRef: "idempotencia_ginpix_api_0001",
		Datos:           []domain.DatoCanonicoGINPIX{{Clave: "codigo_puesto", Campo: campo}},
	})
	if err != nil {
		t.Fatal(err)
	}
	mapeo, err := domain.PublicarMapeoVersionadoGINPIX(domain.BorradorMapeoVersionadoGINPIX{
		Esquema: domain.EsquemaMapeoGINPIXV1, Referencia: "mapeo_api_0001", Version: 2,
		ProcedenciaRef: "procedencia_mapeo_api_0001",
		Reglas: []domain.ReglaMapeoGINPIX{{
			CampoCanonico: "codigo_puesto", CampoDestino: "puesto", Obligatorio: true,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	solicitud, err := ports.NuevaSolicitudMapeoGINPIX(modelo, mapeo)
	if err != nil {
		t.Fatal(err)
	}
	return solicitud, solicitudPersonal, resultadoPersonal, recibo
}

func errorsIs(err, objetivo error) bool {
	return errors.Is(err, objetivo)
}
