package application_test

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	app "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func referenciaSeguimientoV2Prueba(etiqueta string) string {
	suma := sha256.Sum256([]byte(etiqueta))
	return "ref:" + fmt.Sprintf("%x", suma)
}

func proyeccionSeguimientoV2Original(t *testing.T) (ports.ReciboIncorporacionAplicacionV2, domain.PublicacionDefinicionSeguimiento, domain.EstadoPersistidoSeguimiento) {
	t.Helper()
	contenido, err := os.ReadFile("../adapters/seguimientoejercicio/testdata/definicion-ejercicio.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Publicacion domain.PublicacionDefinicionSeguimiento `json:"publicacion"`
	}
	if err := json.Unmarshal(contenido, &fixture); err != nil {
		t.Fatal(err)
	}
	definicion, err := domain.RestaurarDefinicionSeguimiento(fixture.Publicacion)
	if err != nil {
		t.Fatal(err)
	}
	registradaEn := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	periodo := domain.IntervaloSeguimiento{Desde: registradaEn, Hasta: registradaEn.AddDate(0, 1, 0)}
	antes, err := domain.NuevoSeguimiento(definicion, domain.AltaSeguimiento{
		Referencia: referenciaSeguimientoV2Prueba("seguimiento"), OrganizacionRef: referenciaSeguimientoV2Prueba("organizacion"),
		ExpedienteRef: referenciaSeguimientoV2Prueba("expediente"), RelacionRef: referenciaSeguimientoV2Prueba("relacion"),
		PeriodoPrevisto: periodo, CreadoEn: registradaEn.Add(-time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	despues, err := antes.Aplicar(definicion, 0, domain.DatosTransicionSeguimiento{
		ActuacionRef: referenciaSeguimientoV2Prueba("actuacion"), TransicionClave: ports.TransicionConfirmarIncorporacion,
		MotivoClave: "ejercicio_incorporacion", ActorRef: referenciaSeguimientoV2Prueba("actor"), UnidadRef: referenciaSeguimientoV2Prueba("unidad"),
		EfectivoEn: periodo.Desde, RegistradaEn: registradaEn,
		Documentos: []domain.DocumentoSeguimiento{{TipoClave: "resolucion_ejercicio", Referencia: referenciaSeguimientoV2Prueba("documento")}},
		Periodo:    &periodo, ReciboRef: referenciaSeguimientoV2Prueba("recibo"), CorrelacionRef: referenciaSeguimientoV2Prueba("correlacion"),
	})
	if err != nil {
		t.Fatal(err)
	}
	return ports.ReciboIncorporacionAplicacionV2{
		Esquema:       "vec.contratacion-temporal.incorporacion-ejercicio.recibo.v2",
		ExpedienteRef: referenciaSeguimientoV2Prueba("expediente"), SolicitudPersonalRef: referenciaSeguimientoV2Prueba("solicitud"),
		RelacionRef: referenciaSeguimientoV2Prueba("relacion"), ReciboRef: referenciaSeguimientoV2Prueba("recibo"), ActuacionRef: referenciaSeguimientoV2Prueba("actuacion"),
		RegistradaEn: registradaEn, Periodo: periodo, VersionSolicitudPersonal: 1, VersionActualExpediente: 1,
		SeguimientoRef: referenciaSeguimientoV2Prueba("seguimiento"), VersionSeguimientoAnterior: 0, VersionSeguimientoResultante: 1,
		AuditoriaRef: referenciaSeguimientoV2Prueba("auditoria"), OutboxRef: referenciaSeguimientoV2Prueba("outbox"), EjercicioSintetico: true,
	}, fixture.Publicacion, despues.Estado()
}

func TestProyectarSeguimientoIncorporacionV2OriginalCrucesCopiasYDatosLimpios(t *testing.T) {
	recibo, publicacion, estado := proyeccionSeguimientoV2Original(t)
	vista, err := app.ProyectarSeguimientoIncorporacionV2(recibo, publicacion, estado)
	if err != nil {
		t.Fatal(err)
	}
	if vista.Esquema != "vec.contratacion-temporal.seguimiento-incorporacion.v2" ||
		vista.Alcance != "original_incorporacion" ||
		vista.ExpedienteRef != recibo.ExpedienteRef ||
		vista.VersionExpediente != recibo.VersionActualExpediente ||
		vista.ReciboIncorporacionRef != recibo.ReciboRef ||
		vista.SeguimientoRef != recibo.SeguimientoRef ||
		vista.VersionSeguimiento != recibo.VersionSeguimientoResultante ||
		vista.EstadoClave != estado.EstadoActual || vista.Periodo != recibo.Periodo ||
		!vista.RegistradoEn.Equal(recibo.RegistradaEn) ||
		vista.EjercicioSintetico != recibo.EjercicioSintetico ||
		vista.FirmaOficial || vista.EficaciaAdministrativa {
		t.Fatal("proyeccion nominal no conserva solo los datos publicables originales")
	}
	if len(vista.Actuaciones) != len(estado.Actuaciones) || len(vista.Actuaciones) == 0 || len(vista.Actuaciones[0].Documentos) == 0 ||
		!reflect.DeepEqual(vista.Actuaciones[0].Documentos, estado.Actuaciones[0].Documentos) {
		t.Fatal("actuaciones o documentos publicables incompletos")
	}

	vista.Actuaciones[0].Documentos[0].Referencia = "documento:alterado"
	vista2, err := app.ProyectarSeguimientoIncorporacionV2(recibo, publicacion, estado)
	if err != nil || vista2.Actuaciones[0].Documentos[0].Referencia == "documento:alterado" {
		t.Fatal("la vista comparte datos mutables con otra proyeccion")
	}
	estadoSinDocumentos := estado
	estadoSinDocumentos.Actuaciones = append([]domain.ActuacionSeguimiento(nil), estado.Actuaciones...)
	estadoSinDocumentos.Actuaciones[len(estadoSinDocumentos.Actuaciones)-1].Documentos = nil
	vistaSinDocumentos, err := app.ProyectarSeguimientoIncorporacionV2(recibo, publicacion, estadoSinDocumentos)
	if !errors.Is(err, app.ErrProyeccionSeguimientoIncorporacionV2Invalida) ||
		!reflect.DeepEqual(vistaSinDocumentos, ports.VistaSeguimientoIncorporacionV2{}) {
		t.Fatal("historia documental adulterada aceptada")
	}

	for nombre, alterar := range map[string]func(*ports.ReciboIncorporacionAplicacionV2, *domain.EstadoPersistidoSeguimiento){
		"expediente": func(r *ports.ReciboIncorporacionAplicacionV2, _ *domain.EstadoPersistidoSeguimiento) {
			r.ExpedienteRef = "expediente:cruzado"
		},
		"relacion": func(r *ports.ReciboIncorporacionAplicacionV2, _ *domain.EstadoPersistidoSeguimiento) {
			r.RelacionRef = "relacion:cruzada"
		},
		"seguimiento": func(r *ports.ReciboIncorporacionAplicacionV2, _ *domain.EstadoPersistidoSeguimiento) {
			r.SeguimientoRef = "seguimiento:cruzado"
		},
		"version": func(r *ports.ReciboIncorporacionAplicacionV2, _ *domain.EstadoPersistidoSeguimiento) {
			r.VersionSeguimientoResultante++
		},
		"periodo": func(r *ports.ReciboIncorporacionAplicacionV2, _ *domain.EstadoPersistidoSeguimiento) {
			r.Periodo.Hasta = r.Periodo.Hasta.AddDate(0, 0, 1)
		},
		"actuacion": func(r *ports.ReciboIncorporacionAplicacionV2, _ *domain.EstadoPersistidoSeguimiento) {
			r.ActuacionRef = "actuacion:cruzada"
		},
		"recibo": func(r *ports.ReciboIncorporacionAplicacionV2, _ *domain.EstadoPersistidoSeguimiento) {
			r.ReciboRef = "recibo:cruzado"
		},
		"registro": func(r *ports.ReciboIncorporacionAplicacionV2, _ *domain.EstadoPersistidoSeguimiento) {
			r.RegistradaEn = r.RegistradaEn.AddDate(0, 0, 1)
		},
		"firma": func(r *ports.ReciboIncorporacionAplicacionV2, _ *domain.EstadoPersistidoSeguimiento) {
			r.FirmaOficial = true
		},
		"hito_actuacion": func(_ *ports.ReciboIncorporacionAplicacionV2, e *domain.EstadoPersistidoSeguimiento) {
			e.Actuaciones[len(e.Actuaciones)-1].ActuacionRef = "actuacion:hito:cruzada"
		},
		"hito_recibo": func(_ *ports.ReciboIncorporacionAplicacionV2, e *domain.EstadoPersistidoSeguimiento) {
			e.Actuaciones[len(e.Actuaciones)-1].ReciboRef = "recibo:hito:cruzado"
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			r := recibo
			e := estado
			e.Actuaciones = append([]domain.ActuacionSeguimiento(nil), estado.Actuaciones...)
			alterar(&r, &e)
			vista, err := app.ProyectarSeguimientoIncorporacionV2(r, publicacion, e)
			if !errors.Is(err, app.ErrProyeccionSeguimientoIncorporacionV2Invalida) || !reflect.DeepEqual(vista, ports.VistaSeguimientoIncorporacionV2{}) {
				t.Fatal("cruce recibo/estado aceptado")
			}
		})
	}
}
