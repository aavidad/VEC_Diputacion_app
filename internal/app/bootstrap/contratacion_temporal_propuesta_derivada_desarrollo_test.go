package bootstrap

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

var errPropuestaDerivadaCapturada = errors.New("propuesta derivada capturada")

type capturaPropuestaDerivada struct {
	proveedor *proveedorPropuestaFormalizacionDesarrollo
	llamadas  int
	material  ports.MaterialPropuestaFormalizacion
}

func (c *capturaPropuestaDerivada) PrepararYConfirmar(ctx context.Context, s ports.SolicitudPropuestaFormalizacion) (ports.ResultadoPropuestaFormalizacion, error) {
	c.llamadas++
	m, err := c.proveedor.PrepararPropuestaFormalizacion(ctx, s)
	if err != nil {
		return ports.ResultadoPropuestaFormalizacion{}, err
	}
	c.material = m
	return ports.ResultadoPropuestaFormalizacion{}, errPropuestaDerivadaCapturada
}

func escenarioPropuestaDerivadaDesarrollo(t *testing.T) (escenarioPropuestaPrueba, *capturaPropuestaDerivada) {
	t.Helper()
	f := nuevaPropuestaCoordinadorPrueba(t)
	padre := f.lector.expediente.Fiscalizado.Clonar()
	padre.Solicitud.NumeroPersonas = 2
	padre.Solicitud.JornadaMinutos = 420
	padre.Solicitud.PuestoSolicitado = "puesto:administrativo:01"
	if err := padre.Validar(); err != nil {
		t.Fatal("padre de dos plazas inválido:", err)
	}
	oferta := "oferta:" + strings.Repeat("a", 64)
	hijoRef := ports.HijoRefDerivacionNombramiento(padre.OrganizacionRef, padre.Referencia, 1)
	hijo, err := domain.NuevoExpedienteDerivado(domain.AltaExpedienteDerivado{Padre: padre,
		Referencia: hijoRef, OfertaRef: oferta, NumeroPlaza: 1,
		ActorRef: "actor:rrhh:sintetico", ReciboRef: "recibo:derivacion:uno",
		CreadaEn: padre.ActualizadoEn.Add(time.Second)})
	if err != nil {
		t.Fatal("hijo:", err)
	}
	f.lector.expediente = ports.ExpedienteParaSeleccion{Fiscalizado: hijo, VersionActual: 6}
	f.solicitud.ExpedienteRef, f.solicitud.DerivacionRef = hijoRef, "derivacion:ct234:uno"
	f.solicitud.LlamamientoRef, f.solicitud.ResolucionLlamamientoAceptadaRef, f.solicitud.ReciboResolucionAceptadaRef = "", "", ""
	f.solicitud.VersionEsperada = 6
	evidencia := ports.EvidenciaAceptacionDerivada{DerivacionRef: f.solicitud.DerivacionRef,
		PadreRef: padre.Referencia, HijoRef: hijoRef, OfertaRef: oferta, NumeroPlaza: 1,
		AceptacionRef: "aceptacion-plaza:uno", ReciboDerivacionRef: hijo.InstruccionCompartida.ReciboDerivacionRef}
	f.registro.antecedente = ports.AntecedentePropuestaFormalizacion{Derivacion: &evidencia}
	if f.registro.antecedente.ValidarPara(f.solicitud) != nil {
		t.Fatal("antecedente derivado inválido")
	}
	c := &capturaPropuestaDerivada{proveedor: f.registro.proveedor}
	f.ejecutor.servicio = c
	f.ejecutor.bolsa = nil // El hijo consume el antecedente CT234, sin otra lectura de Bolsa.
	return f, c
}

func TestPropuestaDerivadaCotejaHijoYReciboAntesDeGuardar(t *testing.T) {
	f, captura := escenarioPropuestaDerivadaDesarrollo(t)
	_, err := f.ejecutor.PrepararYConfirmar(f.ctx, f.solicitud)
	if !errors.Is(err, errPropuestaDerivadaCapturada) || captura.llamadas != 1 ||
		captura.material.Validar() != nil || captura.material.AceptacionDerivada == nil ||
		captura.material.AceptacionBolsa != nil || captura.material.AceptacionBolsaNativa != nil ||
		*captura.material.AceptacionDerivada != *f.registro.antecedente.Derivacion {
		t.Fatalf("material derivado no llegó íntegro: err=%v material=%+v", err, captura.material)
	}
	for _, mutar := range []func(*ports.EvidenciaAceptacionDerivada){
		func(e *ports.EvidenciaAceptacionDerivada) { e.HijoRef = "expediente:hijo:ajeno" },
		func(e *ports.EvidenciaAceptacionDerivada) { e.DerivacionRef = "derivacion:ct234:ajena" },
		func(e *ports.EvidenciaAceptacionDerivada) { e.OfertaRef = "oferta:" + strings.Repeat("b", 64) },
		func(e *ports.EvidenciaAceptacionDerivada) { e.ReciboDerivacionRef = "recibo:derivacion:ajena" },
	} {
		otro, servicio := escenarioPropuestaDerivadaDesarrollo(t)
		mutar(otro.registro.antecedente.Derivacion)
		_, err := otro.ejecutor.PrepararYConfirmar(otro.ctx, otro.solicitud)
		if !errors.Is(err, application.ErrResolucionFormalizacionNoAceptada) || servicio.llamadas != 0 {
			t.Fatalf("derivación ajena alcanzó guardado: err=%v llamadas=%d", err, servicio.llamadas)
		}
	}
}
