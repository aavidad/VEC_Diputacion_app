package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

var errPropuestaDerivadaCapturada = errors.New("propuesta derivada capturada")

type capturaPropuestaDerivada struct {
	proveedor *proveedorPropuestaFormalizacionDesarrollo
	llamadas  int
	material  ports.MaterialPropuestaFormalizacion
}

type detallePropuestaDerivadaPrueba struct {
	t        *testing.T
	detalle  ports.DetalleExpedienteRRHH
	llamadas int
}

func (d *detallePropuestaDerivadaPrueba) Consultar(ctx context.Context, q ports.SolicitudDetalleRRHH) (ports.DetalleExpedienteRRHH, error) {
	d.t.Helper()
	c, ok := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	f, real := fronteraSeguridadComunDesdeContexto(ctx)
	if !ok || !real || c.ruta != httpinterno.RutaConsultaDetalleRRHH || c.consultaRRHH == nil ||
		f.descriptor.Clave != "ct-expediente-consultar" || q.VersionObservada() != 0 {
		d.t.Fatal("el detalle no recibió su subconsulta nominal")
	}
	d.llamadas++
	return d.detalle.Clonar(), nil
}

func (c *capturaPropuestaDerivada) PrepararYConfirmar(ctx context.Context, s ports.SolicitudPropuestaFormalizacion) (ports.ResultadoPropuestaFormalizacion, error) {
	c.llamadas++
	m, err := c.proveedor.PrepararPropuestaFormalizacion(ctx, s)
	if err != nil {
		return ports.ResultadoPropuestaFormalizacion{}, err
	}
	c.material = m
	recurso, err := postgresct.RecursoPropuestaFormalizacion(m)
	if err != nil || !solicitudAutorizacionLlamamientoDesarrolloValida(ctx, httpinterno.RutaPropuestaFormalizacion,
		dominiovec.DatosSolicitudAutorizacionLigadaV3{Accion: postgresct.AccionPropuestaFormalizacion,
			Finalidad: "gestionar_contratacion_temporal", ReferenciaMotivo: motivoPropuestaFormalizacionDesarrollo(),
			Recurso: recurso}) {
		return ports.ResultadoPropuestaFormalizacion{}, errors.New("confirmación derivada sin autoridad ligada")
	}
	return ports.ResultadoPropuestaFormalizacion{}, errPropuestaDerivadaCapturada
}

func escenarioPropuestaDerivadaDesarrollo(t *testing.T) (escenarioPropuestaPrueba, *capturaPropuestaDerivada, *detallePropuestaDerivadaPrueba) {
	t.Helper()
	f := nuevaPropuestaCoordinadorPrueba(t)
	padre := f.lector.expediente.Fiscalizado.Clonar()
	padre.Solicitud.NumeroPersonas = 2
	padre.Solicitud.JornadaMinutos = 420
	padre.Solicitud.PuestoSolicitado = "puesto:administrativo:01"
	if err := padre.Validar(); err != nil {
		t.Fatal("padre de dos plazas inválido:", err)
	}
	preimagen, err := json.Marshal(padre)
	if err != nil {
		t.Fatal(err)
	}
	oferta := "oferta:" + strings.Repeat("a", 64)
	hijoRef := ports.HijoRefDerivacionNombramiento(padre.OrganizacionRef, padre.Referencia, 1)
	hijo, err := domain.NuevoExpedienteDerivado(domain.AltaExpedienteDerivado{Padre: padre,
		Referencia: hijoRef, OfertaRef: oferta, NumeroPlaza: 1,
		ActorRef: "actor:rrhh:sintetico", ReciboRef: "recibo:derivacion:uno",
		CreadaEn: padre.ActualizadoEn.Add(time.Second), PreimagenPadreJSON: string(preimagen)})
	if err != nil {
		t.Fatal("hijo:", err)
	}
	f.lector.expediente = ports.ExpedienteParaSeleccion{Fiscalizado: hijo, VersionActual: 6}
	// La cabeza raw con circuito representa la exclusión CT163 del lector viejo.
	f.lector.expediente.Fiscalizado.Circuito = &domain.CircuitoAdministrativo{}
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
	d := detalleResolucionDesarrolloPrueba()
	d.Resumen.ExpedienteRef, d.Resumen.OrganizacionRef = hijoRef, f.solicitud.OrganizacionRef
	d.Resumen.NumeroVisible = padre.NumeroVisible + "-P1"
	d.Resumen.Version, d.Resumen.FaseClave, d.Resumen.EstadoClave = 6, domain.FaseFiscalizacion, domain.EstadoEnCurso
	d.InstruccionCompartida = &ports.InstruccionCompartidaRRHH{DerivacionRef: f.solicitud.DerivacionRef,
		PadreRef: padre.Referencia, NumeroVisiblePadre: padre.NumeroVisible,
		OfertaRef: oferta, NumeroPlaza: 1, ReciboDerivacionRef: hijo.InstruccionCompartida.ReciboDerivacionRef}
	lector := &detallePropuestaDerivadaPrueba{t: t, detalle: d}
	catalogo := catalogoDetalleCTPrueba(t, f.ejecutor.soporte)
	frontera, ok := catalogo.resolver(http.MethodPost, httpinterno.RutaPropuestaFormalizacion)
	if !ok {
		t.Fatal("frontera de propuesta ausente")
	}
	capacidad := f.ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	capacidad.metodo, capacidad.consultaRRHH = http.MethodPost, &contextoConsultaRRHHPeticionDesarrollo{}
	capacidad.certificadoVerificadoEn = f.ejecutor.soporte.reloj.Ahora().Add(-time.Second)
	capacidad.certificadoValidoHasta = f.ejecutor.soporte.reloj.Ahora().Add(time.Hour)
	f.ctx = context.WithValue(f.ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidad)
	f.ctx = context.WithValue(f.ctx, claveFronteraSeguridadComunDesarrollo{}, fronteraSeguridadComunDesarrollo{
		metodo: http.MethodPost, ruta: httpinterno.RutaPropuestaFormalizacion,
		superficie: superficieInternaSeguridadComunDesarrollo, catalogo: catalogo, descriptor: frontera})
	f.ejecutor.detalle, f.ejecutor.fronteras = lector, catalogo
	return f, c, lector
}

func TestPropuestaDerivadaCotejaHijoYReciboAntesDeGuardar(t *testing.T) {
	f, captura, detalle := escenarioPropuestaDerivadaDesarrollo(t)
	_, err := f.ejecutor.PrepararYConfirmar(f.ctx, f.solicitud)
	if !errors.Is(err, errPropuestaDerivadaCapturada) || captura.llamadas != 1 || detalle.llamadas != 1 || f.lector.llamadas != 0 ||
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
		otro, servicio, _ := escenarioPropuestaDerivadaDesarrollo(t)
		mutar(otro.registro.antecedente.Derivacion)
		_, err := otro.ejecutor.PrepararYConfirmar(otro.ctx, otro.solicitud)
		if !errors.Is(err, application.ErrResolucionFormalizacionNoAceptada) || servicio.llamadas != 0 {
			t.Fatalf("derivación ajena alcanzó guardado: err=%v llamadas=%d", err, servicio.llamadas)
		}
	}
}

func TestPropuestaDerivadaExigeMetadatosC4YNoReleeCircuito(t *testing.T) {
	for nombre, mutar := range map[string]func(*ports.DetalleExpedienteRRHH){
		"hijo":         func(d *ports.DetalleExpedienteRRHH) { d.Resumen.ExpedienteRef = "expediente:ajeno" },
		"organizacion": func(d *ports.DetalleExpedienteRRHH) { d.Resumen.OrganizacionRef = "organizacion:ajena" },
		"derivacion":   func(d *ports.DetalleExpedienteRRHH) { d.InstruccionCompartida.DerivacionRef = "derivacion:ajena" },
	} {
		t.Run(nombre, func(t *testing.T) {
			f, servicio, detalle := escenarioPropuestaDerivadaDesarrollo(t)
			mutar(&detalle.detalle)
			_, err := f.ejecutor.PrepararYConfirmar(f.ctx, f.solicitud)
			if !errors.Is(err, application.ErrResolucionFormalizacionNoAceptada) || servicio.llamadas != 0 ||
				f.registro.consultas != 0 || f.lector.llamadas != 0 || detalle.llamadas != 1 {
				t.Fatalf("metadato ajeno llegó al registro: err=%v servicio=%d registro=%d raw=%d", err, servicio.llamadas, f.registro.consultas, f.lector.llamadas)
			}
		})
	}
	faltaC4, servicioSinC4, detalleSinC4 := escenarioPropuestaDerivadaDesarrollo(t)
	detalleSinC4.detalle.InstruccionCompartida = nil
	_, err := faltaC4.ejecutor.PrepararYConfirmar(faltaC4.ctx, faltaC4.solicitud)
	if !errors.Is(err, application.ErrPropuestaFormalizacionNoDisponible) ||
		servicioSinC4.llamadas != 0 || faltaC4.registro.consultas != 0 {
		t.Fatal("C4 ausente no cerró solo la propuesta derivada", err)
	}
	f, servicio, detalle := escenarioPropuestaDerivadaDesarrollo(t)
	detalle.detalle.Resumen.Version = 7
	detalle.detalle.Resumen.FaseClave = domain.FaseNombramiento
	_, err = f.ejecutor.PrepararYConfirmar(f.ctx, f.solicitud)
	if !errors.Is(err, errPropuestaDerivadaCapturada) || servicio.llamadas != 1 || detalle.llamadas != 1 ||
		f.lector.llamadas != 0 || f.solicitud.VersionEsperada != 6 {
		t.Fatalf("replay no conservó v6 ante cabeza v7: err=%v raw=%d", err, f.lector.llamadas)
	}
}
