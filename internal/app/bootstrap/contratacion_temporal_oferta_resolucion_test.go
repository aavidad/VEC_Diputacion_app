package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	bolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	seguridad "vec-diputacion-granada/internal/vec/adapters/seguridad"
	core "vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/pruebas"
)

func datosResolucionOfertaCTPrueba(t *testing.T) (ct.IntencionSeleccionOfertaCT, ct.ComandoOfertaBolsaCT) {
	t.Helper()
	ahora := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	actor, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(ahora,
		"per_0123456789abcdefghijkl", "prf_0123456789abcdefghijkl",
		core.AuthMethodCertificate, core.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	correlacion, err := core.GenerarReferenciaCorrelacionAutorizacionV2(t.Context(), seguridad.GeneradorReferenciasCriptograficas{})
	if err != nil {
		t.Fatal(err)
	}
	i := ct.IntencionSeleccionOfertaCT{
		OrganizacionRef: "organizacion:prueba", ExpedienteRef: "expediente:ct:prueba",
		CategoriaRef: "categoria:rpt:administrativo", CentroRef: "centro:520",
		UnidadGestoraRef:  "unidad:rrhh:prueba",
		VersionExpediente: 6, ClaveIdempotencia: "018f47a2-6b31-4c80-8a95-4d2e707c5a11",
		ActorRef: actor.Contexto.PersonaRef, PerfilRef: actor.Contexto.PerfilActivoRef,
		Vinculo: vinculo, ResultadoContexto: actor, Correlacion: correlacion,
		MotivoAutorizacion: core.ReferenciaEntradaCatalogo{CatalogoID: "motivos_autorizacion", CatalogoVersion: 1,
			CatalogoHuellaSHA256: strings.Repeat("a", 64), EntradaClave: "motivo_0123456789abcdef0123456789abcdef"},
	}
	i.HuellaIntencion = ct.HuellaIntencionSeleccionOfertaCT(i)
	if err := i.Validar(); err != nil {
		t.Fatal(err)
	}
	c := ct.ComandoOfertaBolsaCT{
		OrganizacionRef: i.OrganizacionRef, ExpedienteRef: i.ExpedienteRef,
		VersionExpediente: i.VersionExpediente, ActorRef: i.ActorRef,
		BolsaRef: "bolsa:administrativo:prueba", NecesidadRef: "necesidad:rpt:prueba",
		NumeroPlazas: 3, NumeroElegibles: 4,
		Datos: ct.DatosOfertaSeleccionCT{Categoria: "Administrativo", Centro: "Centro 520",
			FechaInicio: "2026-11-01", FechaFin: "2027-01-31", Descripcion: "Tres plazas administrativas"},
		OrigenCT:   []byte(`{"acta_ref":"acta:prueba","lectura":{"corte":"2026-10-15T12:00:00.000000Z","entradas":[1,2,3,4]}}`),
		ClaveBolsa: ct.ClaveBolsaSeleccionOfertaCT(i), InicioPlazo: ct.InicioSeleccionOfertaPublicacion,
	}
	if err := c.ValidarPara(i); err != nil {
		t.Fatal(err)
	}
	return i, c
}

func cierreResolucionOfertaCTPrueba(i ct.IntencionSeleccionOfertaCT, c ct.ComandoOfertaBolsaCT) ct.ReciboCierreOfertaCT {
	return ct.ReciboCierreOfertaCT{
		Esquema: ct.EsquemaCierreOfertaBolsaCTV1, Estado: ct.EstadoCierreOfertaBolsaCTV1,
		OrganizacionRef: i.OrganizacionRef, ExpedienteRef: i.ExpedienteRef,
		VersionExpediente: i.VersionExpediente, ClaveCT: i.ClaveIdempotencia,
		ActorRef: i.ActorRef, BolsaRef: c.BolsaRef, ClaveBolsa: c.ClaveBolsa,
		NumeroPlazas: c.NumeroPlazas, DatosSHA256: strings.Repeat("a", 64),
		OrigenSHA256: strings.Repeat("b", 64), ComandoSHA256: strings.Repeat("c", 64),
		ReciboCierreRef: "recibo:bolsa:cierre:1", DecisionRef: "decision:bolsa:cierre:1",
		AuditoriaRef: "auditoria:bolsa:cierre:1", EventoRef: "evento:bolsa:cierre:1",
		CerradaEn: time.Date(2026, 10, 15, 12, 6, 0, 123456000, time.UTC),
	}
}

func TestProyectarResolucionOfertaCTPublicadaOCerrada(t *testing.T) {
	i, c := datosResolucionOfertaCTPrueba(t)
	publicada := time.Date(2026, 10, 15, 12, 4, 0, 123456000, time.UTC)
	oferta := bolsa.OfertaPublicada{
		OfertaRef: "oferta:bolsa:1", ReciboRef: "recibo:oferta:1", BolsaRef: c.BolsaRef,
		Datos: dominiobolsa.DatosOferta{Categoria: c.Datos.Categoria, Centro: c.Datos.Centro,
			FechaInicio: c.Datos.FechaInicio, FechaFin: c.Datos.FechaFin, Descripcion: c.Datos.Descripcion},
		NumeroPlazas: c.NumeroPlazas, PublicadaEn: publicada, VenceAntesDe: publicada.Add(48 * time.Hour),
	}
	salida, err := proyectarResultadoResolucionOfertaCT(i, c,
		bolsa.ResultadoResolverIntencionOfertaCT{Situacion: bolsa.ResultadoIntencionOfertaPublicada, Oferta: &oferta})
	if err != nil || salida.Situacion != ct.CierreOfertaPublicada || salida.Oferta == nil || salida.Cierre != nil ||
		salida.Oferta.ReciboOfertaRef != oferta.ReciboRef || salida.Oferta.NumeroPlazas != c.NumeroPlazas {
		t.Fatalf("publicación no proyectada desde recibo Bolsa: %+v, %v", salida, err)
	}
	cierre := cierreResolucionOfertaCTPrueba(i, c)
	jsonCierre, err := json.Marshal(cierre)
	if err != nil || !bytes.Contains(jsonCierre, []byte(`"cerrada_en":"2026-10-15T12:06:00.123456Z"`)) {
		t.Fatalf("recibo B121 sin UTC microsegundos: %s, %v", jsonCierre, err)
	}
	salida, err = proyectarResultadoResolucionOfertaCT(i, c,
		bolsa.ResultadoResolverIntencionOfertaCT{Situacion: bolsa.ResultadoIntencionOfertaCerrada, Cierre: jsonCierre})
	if err != nil || salida.Situacion != ct.CierreOfertaCerrada || salida.Oferta != nil || salida.Cierre == nil ||
		salida.Cierre.ReciboCierreRef != cierre.ReciboCierreRef || !salida.Cierre.CerradaEn.Equal(cierre.CerradaEn) {
		t.Fatalf("cierre no proyectado desde recibo Bolsa: %+v, %v", salida, err)
	}
}

func TestProyectarResolucionOfertaCTRechazaRespuestaB121Incoherente(t *testing.T) {
	i, c := datosResolucionOfertaCTPrueba(t)
	publicada := time.Date(2026, 10, 15, 12, 4, 0, 0, time.UTC)
	oferta := bolsa.OfertaPublicada{
		OfertaRef: "oferta:bolsa:1", ReciboRef: "recibo:oferta:1", BolsaRef: c.BolsaRef,
		Datos: dominiobolsa.DatosOferta{Categoria: c.Datos.Categoria, Centro: c.Datos.Centro,
			FechaInicio: c.Datos.FechaInicio, FechaFin: c.Datos.FechaFin, Descripcion: c.Datos.Descripcion},
		NumeroPlazas: c.NumeroPlazas, PublicadaEn: publicada, VenceAntesDe: publicada.Add(48 * time.Hour),
	}
	cierre := cierreResolucionOfertaCTPrueba(i, c)
	jsonCierre, err := json.Marshal(cierre)
	if err != nil {
		t.Fatal(err)
	}
	for _, caso := range []struct {
		nombre string
		r      bolsa.ResultadoResolverIntencionOfertaCT
	}{
		{"doble terminal", bolsa.ResultadoResolverIntencionOfertaCT{Situacion: bolsa.ResultadoIntencionOfertaPublicada, Oferta: &oferta, Cierre: jsonCierre}},
		{"oferta de otra bolsa", func() bolsa.ResultadoResolverIntencionOfertaCT {
			otra := oferta
			otra.BolsaRef = "bolsa:ajena"
			return bolsa.ResultadoResolverIntencionOfertaCT{Situacion: bolsa.ResultadoIntencionOfertaPublicada, Oferta: &otra}
		}()},
		{"cierre otro actor", func() bolsa.ResultadoResolverIntencionOfertaCT {
			otro := cierre
			otro.ActorRef = "per_abcdefghij0123456789kl"
			b, _ := json.Marshal(otro)
			return bolsa.ResultadoResolverIntencionOfertaCT{Situacion: bolsa.ResultadoIntencionOfertaCerrada, Cierre: b}
		}()},
		{"cierre de otra bolsa", func() bolsa.ResultadoResolverIntencionOfertaCT {
			otro := cierre
			otro.BolsaRef = "bolsa:ajena"
			b, _ := json.Marshal(otro)
			return bolsa.ResultadoResolverIntencionOfertaCT{Situacion: bolsa.ResultadoIntencionOfertaCerrada, Cierre: b}
		}()},
		{"cierre campo desconocido", bolsa.ResultadoResolverIntencionOfertaCT{Situacion: bolsa.ResultadoIntencionOfertaCerrada,
			Cierre: append(append([]byte(nil), jsonCierre[:len(jsonCierre)-1]...), []byte(`,"extra":"ajeno"}`)...)}},
		{"cierre instante no canónico", bolsa.ResultadoResolverIntencionOfertaCT{Situacion: bolsa.ResultadoIntencionOfertaCerrada,
			Cierre: bytes.Replace(jsonCierre, []byte(".123456Z"), []byte(".123456001Z"), 1)}},
		{"estado desconocido", bolsa.ResultadoResolverIntencionOfertaCT{Situacion: "ausente"}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			if _, err := proyectarResultadoResolucionOfertaCT(i, c, caso.r); err == nil {
				t.Fatal("respuesta B121 incoherente aceptada")
			}
		})
	}
}

func TestResolverOfertaCTSinCapturaRealFallaCerrado(t *testing.T) {
	i, c := datosResolucionOfertaCTPrueba(t)
	_, err := (publicadorOfertaBolsaCTDesarrollo{}).Resolver(context.Background(), i, c)
	if !errors.Is(err, ct.ErrIntegracionBolsaNoDisponible) {
		t.Fatalf("resolución sin puente nativo admitida: %v", err)
	}
}
