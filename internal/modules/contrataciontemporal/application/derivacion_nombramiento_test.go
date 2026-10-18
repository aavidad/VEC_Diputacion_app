package application

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vp "vec-diputacion-granada/internal/vec/ports"
)

type preparadorDerivacionAplicacionPrueba struct {
	solicitud ports.SolicitudDerivacionNombramiento
	llamadas  int
}

func (p *preparadorDerivacionAplicacionPrueba) PrepararDerivacionNombramiento(_ context.Context, _ ports.EntradaDerivacionNombramiento) (ports.SolicitudDerivacionNombramiento, error) {
	p.llamadas++
	return p.solicitud, nil
}

type autorizadorDerivacionAplicacionPrueba struct {
	fallo    error
	llamadas int
}

func (a *autorizadorDerivacionAplicacionPrueba) AutorizarDerivacionNombramiento(context.Context, ports.SolicitudDerivacionNombramiento) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	a.llamadas++
	return vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, a.fallo
}

type transaccionDerivacionAplicacionPrueba struct {
	resultado ports.ResultadoDerivacionNombramiento
	llamadas  int
}

func (x *transaccionDerivacionAplicacionPrueba) ConfirmarDerivacionNombramiento(context.Context, ports.SolicitudDerivacionNombramiento, vp.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ResultadoDerivacionNombramiento, error) {
	x.llamadas++
	return x.resultado, nil
}

func escenarioDerivacionAplicacionPrueba(t *testing.T) (ports.SolicitudDerivacionNombramiento, ports.ResultadoDerivacionNombramiento) {
	t.Helper()
	b, err := os.ReadFile("../domain/testdata/expediente_nombramiento_v7.json")
	if err != nil {
		t.Fatal(err)
	}
	var padre domain.Expediente
	if err = json.Unmarshal(b, &padre); err != nil {
		t.Fatal(err)
	}
	padre.Actuaciones = padre.Actuaciones[:6]
	padre.Version = 6
	padre.FaseActual = domain.FaseFiscalizacion
	padre.ActualizadoEn = padre.Actuaciones[5].RealizadaEn
	padre.Solicitud.NumeroPersonas = 2
	padre.Solicitud.JornadaMinutos = 420
	padre.Solicitud.PuestoSolicitado = "puesto:administrativo:01"
	if err = padre.Validar(); err != nil {
		t.Fatalf("fixture padre: %v", err)
	}
	b, err = json.Marshal(padre)
	if err != nil {
		t.Fatal(err)
	}
	s := ports.SolicitudDerivacionNombramiento{
		Esquema: ports.EsquemaSolicitudDerivacionNombramiento, OrganizacionRef: padre.OrganizacionRef,
		PadreRef: padre.Referencia, VersionPadre: 6, HijoRef: ports.HijoRefDerivacionNombramiento(padre.OrganizacionRef, padre.Referencia, 1),
		OfertaRef: "oferta:" + strings.Repeat("a", 64), NumeroPlaza: 1, SecuenciaPlaza: 1,
		ParticipacionRef: "participacion:ct:sintetica:01", ReciboAdjudicacionRef: "recibo:adjudicacion:sintetica:01",
		ClaveIdempotencia: "a1234567-1234-4123-8123-123456789abc", UnidadRef: padre.Asignacion.UnidadRef, PadreCanonico: string(b),
	}
	instante := padre.ActualizadoEn.Add(time.Minute).Truncate(time.Second)
	r := ports.ResultadoDerivacionNombramiento{
		Esquema: ports.EsquemaResultadoDerivacionNombramiento, Estado: "confirmado", PadreRef: s.PadreRef, HijoRef: s.HijoRef,
		NumeroVisible: padre.NumeroVisible + "-P" + strconv.Itoa(int(s.NumeroPlaza)), NumeroPlaza: s.NumeroPlaza,
		TotalPersonas: 2, Version: 6, AceptacionRef: "aceptacion:ct:sintetica:01",
		ReciboRef: "recibo:derivacion:sintetica:01", AuditoriaRef: "auditoria:ct:sintetica:01", ConfirmadaEn: instante,
	}
	r.Expediente, err = domain.NuevoExpedienteDerivado(domain.AltaExpedienteDerivado{
		Padre: padre, Referencia: s.HijoRef, OfertaRef: s.OfertaRef, NumeroPlaza: 1,
		ActorRef: "actor:rrhh:sintetico:01", ReciboRef: r.ReciboRef, CreadaEn: instante,
	})
	if err != nil || r.ValidarPara(s) != nil {
		t.Fatalf("fixture hijo: %v", err)
	}
	return s, r
}

func TestServicioDerivacionNombramientoCortaFronteras(t *testing.T) {
	s, r := escenarioDerivacionAplicacionPrueba(t)
	denegada := errors.New("denegacion de prueba")
	for _, tc := range []struct {
		nombre                          string
		preparar                        func(*ports.SolicitudDerivacionNombramiento)
		falloAutorizacion               error
		alterar                         func(*ports.ResultadoDerivacionNombramiento)
		esperado                        error
		llamadasAutorizador, llamadasTX int
	}{
		{"preparador cambia entrada", func(x *ports.SolicitudDerivacionNombramiento) {
			x.ClaveIdempotencia = "b1234567-1234-4123-8123-123456789abc"
		}, nil, nil, ports.ErrDerivacionNombramientoNoConfiable, 0, 0},
		{"autorizador deniega", nil, denegada, nil, denegada, 1, 0},
		{"resultado manipulado", nil, nil, func(x *ports.ResultadoDerivacionNombramiento) { x.ReciboRef = "recibo:otro:sintetico:01" }, ports.ErrDerivacionNombramientoNoConfiable, 1, 1},
		{"resultado coherente", nil, nil, nil, nil, 1, 1},
	} {
		t.Run(tc.nombre, func(t *testing.T) {
			p := &preparadorDerivacionAplicacionPrueba{solicitud: s}
			if tc.preparar != nil {
				tc.preparar(&p.solicitud)
			}
			a := &autorizadorDerivacionAplicacionPrueba{fallo: tc.falloAutorizacion}
			x := &transaccionDerivacionAplicacionPrueba{resultado: r}
			if tc.alterar != nil {
				tc.alterar(&x.resultado)
			}
			servicio, err := NuevoServicioDerivacionNombramiento(p, a, x)
			if err != nil {
				t.Fatal(err)
			}
			obtenido, err := servicio.Derivar(t.Context(), s.Entrada())
			if !errors.Is(err, tc.esperado) || p.llamadas != 1 || a.llamadas != tc.llamadasAutorizador || x.llamadas != tc.llamadasTX {
				t.Fatalf("error=%v, llamadas=%d/%d/%d", err, p.llamadas, a.llamadas, x.llamadas)
			}
			if tc.esperado == nil && obtenido.ReciboRef != r.ReciboRef {
				t.Fatal("no devolvió el recibo coherente")
			}
		})
	}
}
