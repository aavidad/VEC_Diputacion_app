package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type solicitudPersonalAnotacionDoble struct {
	referencia string
	err        error
	llamadas   int
}

func (d *solicitudPersonalAnotacionDoble) ResolverSolicitudPersonalAnotacionAdministrativa(
	context.Context, string, string,
) (string, error) {
	d.llamadas++
	return d.referencia, d.err
}

type selladorAmbitoAnotacionDoble struct {
	coleccion ports.ColeccionSellosHMAC
	err       error
}

func (d selladorAmbitoAnotacionDoble) SellarAmbitoAnotacionAdministrativa(
	context.Context, ports.SolicitudSellarAmbitoIdempotencia,
) (ports.ColeccionSellosHMAC, error) {
	return d.coleccion, d.err
}

type derivadorHuellaAnotacionDoble struct {
	coleccion ports.ColeccionSellosHMAC
	err       error
}

func (d derivadorHuellaAnotacionDoble) DerivarHuellaAnotacionAdministrativa(
	context.Context, ports.MaterialAnotacionAdministrativa,
) (ports.ColeccionSellosHMAC, error) {
	return d.coleccion, d.err
}

type preparadorAnotacionDoble struct {
	preparacion ports.PreparacionAnotacionAdministrativa
	err         error
	llamadas    int
}

func (d *preparadorAnotacionDoble) PrepararAnotacionAdministrativa(
	context.Context, ports.SolicitudPrepararAnotacionAdministrativa,
) (ports.PreparacionAnotacionAdministrativa, error) {
	d.llamadas++
	return d.preparacion, d.err
}

type politicaAnotacionDoble struct {
	politica ports.PoliticaAnotacionAdministrativa
	err      error
	llamadas int
}

func (d *politicaAnotacionDoble) ResolverPoliticaAnotacionAdministrativa(
	context.Context, ports.SolicitudResolverPoliticaAnotacionAdministrativa,
) (ports.PoliticaAnotacionAdministrativa, error) {
	d.llamadas++
	return d.politica, d.err
}

type transaccionAnotacionDoble struct {
	recibo   ports.ReciboAnotacionAdministrativa
	err      error
	llamadas int
	orden    ports.OrdenConfirmarAnotacionAdministrativa
}

func (d *transaccionAnotacionDoble) ConfirmarAnotacionAdministrativa(
	_ context.Context, orden ports.OrdenConfirmarAnotacionAdministrativa,
) (ports.ReciboAnotacionAdministrativa, error) {
	d.llamadas++
	d.orden = orden
	return d.recibo, d.err
}

type recuperadorMaterialAnotacionDoble struct {
	material  ports.MaterialAnotacionAdministrativa
	err       error
	llamadas  int
	solicitud ports.SolicitudRecuperarMaterialAnotacionAdministrativa
}

func (d *recuperadorMaterialAnotacionDoble) RecuperarMaterialAnotacionAdministrativaAutorizada(
	_ context.Context, solicitud ports.SolicitudRecuperarMaterialAnotacionAdministrativa,
) (ports.MaterialAnotacionAdministrativa, error) {
	d.llamadas++
	d.solicitud = solicitud
	return d.material, d.err
}

type escenarioAnotacionAdministrativa struct {
	instante    time.Time
	solicitud   SolicitudRegistrarAnotacionAdministrativa
	preparacion ports.PreparacionAnotacionAdministrativa
	recibo      ports.ReciboAnotacionAdministrativa
	contexto    ports.ContextoAutorizacionAltaV3
	motivo      dominiovec.ReferenciaEntradaCatalogo
}

type dependenciasAnotacionAdministrativa struct {
	contextos     *resolutorContextoDoble
	solicitudes   *solicitudPersonalAnotacionDoble
	preparador    *preparadorAnotacionDoble
	politicas     *politicaAnotacionDoble
	correlaciones *generadorReferenciasDoble
	autorizador   *autorizadorV3Doble
	transaccion   *transaccionAnotacionDoble
}

func nuevoEscenarioAnotacionAdministrativa(t *testing.T, confirmada bool) escenarioAnotacionAdministrativa {
	t.Helper()
	entorno := nuevoEntornoConsultaRRHH(t)
	configurarDetalleCompletoRRHHPrueba(t, entorno)
	instante := entorno.ahora
	contexto := contextoAutorizacionAltaV3Prueba(t, instante)
	vinculo, err := contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	solicitud := SolicitudRegistrarAnotacionAdministrativa{
		AutenticacionRef: vinculo.AutenticacionRef, SesionRef: vinculo.SesionRef,
		PerfilRef: vinculo.PerfilActivoRef, OrganizacionRef: entorno.expediente.OrganizacionRef,
		ExpedienteRef: entorno.expediente.Referencia, VersionEsperada: entorno.expediente.Version,
		ClaveIdempotencia: "11111111-2222-4333-8444-555555555555",
		Observaciones:     "Anotación administrativa sintética sobre la incorporación acreditada.",
	}
	material := ports.MaterialAnotacionAdministrativa{
		OrganizacionRef: solicitud.OrganizacionRef, ExpedienteRef: solicitud.ExpedienteRef,
		SolicitudPersonalRef: "solicitud-personal:anotacion-sintetica-001",
		VersionEsperada:      solicitud.VersionEsperada, ClaveIdempotencia: solicitud.ClaveIdempotencia,
		Observaciones: solicitud.Observaciones, ActorRef: vinculo.PrincipalID, PerfilRef: vinculo.PerfilActivoRef,
	}
	vinculoSeguimiento := domain.VinculoSeguimientoOriginal{
		SeguimientoRef: "seguimiento:incorporacion-sintetica-001", VersionSeguimiento: 1,
		HuellaRaizSeguimientoSHA256: strings.Repeat("a", 64),
	}
	recibo := ports.ReciboAnotacionAdministrativa{
		Operacion:       ports.OperacionRegistrarAnotacionAdministrativa,
		OrganizacionRef: material.OrganizacionRef, ExpedienteRef: material.ExpedienteRef,
		VersionAnterior: material.VersionEsperada, VersionResultante: material.VersionEsperada + 1,
		FaseResultante: entorno.expediente.FaseActual, EstadoResultante: entorno.expediente.EstadoActual,
		SeguimientoOriginal: vinculoSeguimiento, ReciboRef: "recibo:anotacion-sintetica-001",
		AuditoriaRef: "auditoria:anotacion-sintetica-001", EventoRef: "evento:anotacion-sintetica-001",
		ActorRef: material.ActorRef, RegistradaEn: instante,
	}
	preparacion := ports.PreparacionAnotacionAdministrativa{
		Material:               material,
		AmbitoIdempotenciaHMAC: selloHMACRegistroPrueba(ports.DominioAmbitoIdempotenciaAnotacionAdministrativa+"/v1", "b"),
		HuellaPeticionHMAC:     selloHMACRegistroPrueba(ports.DominioHuellaPeticionAnotacionAdministrativa+"/v1", "c"),
		Expediente:             entorno.expediente, SeguimientoOriginal: vinculoSeguimiento,
		Referencias:                    ports.ReferenciasEfectoAnotacionAdministrativa{ReciboRef: recibo.ReciboRef, EventoRef: recibo.EventoRef},
		Estado:                         ports.PreparacionAnotacionAdministrativaPreparada,
		HuellaEstadoSeguimientoSHA256:  strings.Repeat("d", 64),
		ReciboIncorporacionOriginalRef: "recibo:incorporacion-sintetica-001",
	}
	if confirmada {
		preparacion.Estado = ports.PreparacionAnotacionAdministrativaConfirmada
		preparacion.ReciboConfirmado = &recibo
	}
	return escenarioAnotacionAdministrativa{
		instante: instante, solicitud: solicitud, preparacion: preparacion, recibo: recibo, contexto: contexto,
		motivo: dominiovec.ReferenciaEntradaCatalogo{CatalogoID: "motivos_autorizacion", CatalogoVersion: 1,
			CatalogoHuellaSHA256: strings.Repeat("e", 64), EntradaClave: "motivo_11111111111111111111111111111111"},
	}
}

func construirServicioAnotacionAdministrativa(t *testing.T, e escenarioAnotacionAdministrativa) (*ServicioAnotacionesAdministrativas, *dependenciasAnotacionAdministrativa) {
	t.Helper()
	ambitos, err := ports.NuevaColeccionSellosHMAC(e.preparacion.AmbitoIdempotenciaHMAC, nil)
	if err != nil {
		t.Fatal(err)
	}
	huellas, err := ports.NuevaColeccionSellosHMAC(e.preparacion.HuellaPeticionHMAC, nil)
	if err != nil {
		t.Fatal(err)
	}
	d := &dependenciasAnotacionAdministrativa{
		contextos:   &resolutorContextoDoble{contexto: e.contexto},
		solicitudes: &solicitudPersonalAnotacionDoble{referencia: e.preparacion.Material.SolicitudPersonalRef},
		preparador:  &preparadorAnotacionDoble{preparacion: e.preparacion},
		politicas: &politicaAnotacionDoble{politica: ports.PoliticaAnotacionAdministrativa{
			MotivoAutorizacion: e.motivo, DefinicionRef: "politica:anotacion-administrativa-sintetica-001",
			DefinicionVersion: 1, DefinicionHuellaSHA256: strings.Repeat("f", 64),
			EvaluadaEn: e.instante, ValidaHasta: e.instante.Add(time.Minute),
		}},
		correlaciones: &generadorReferenciasDoble{correlacion: "correlacion_11111111111111111111111111111111"},
		autorizador:   &autorizadorV3Doble{t: t, instante: e.instante, motivo: e.motivo},
		transaccion:   &transaccionAnotacionDoble{recibo: e.recibo},
	}
	servicio, err := NuevoServicioAnotacionesAdministrativas(d.contextos, d.solicitudes,
		selladorAmbitoAnotacionDoble{coleccion: ambitos}, derivadorHuellaAnotacionDoble{coleccion: huellas},
		d.preparador, d.politicas, d.correlaciones, d.autorizador, &relojMutable{instante: e.instante}, d.transaccion)
	if err != nil {
		t.Fatal(err)
	}
	return servicio, d
}

func TestAnotacionAdministrativaConfirmaUnEfectoAutorizado(t *testing.T) {
	e := nuevoEscenarioAnotacionAdministrativa(t, false)
	servicio, d := construirServicioAnotacionAdministrativa(t, e)
	recibo, err := servicio.Registrar(context.Background(), e.solicitud)
	if err != nil {
		t.Fatalf("registrar anotación: %v", err)
	}
	if recibo != e.recibo || d.autorizador.llamadas != 1 || d.transaccion.llamadas != 1 ||
		d.transaccion.orden.ExpedienteSiguiente.Version != e.preparacion.Expediente.Version+1 {
		t.Fatalf("efecto o autorización inesperados: recibo=%#v v3=%d tx=%d orden=%#v", recibo, d.autorizador.llamadas, d.transaccion.llamadas, d.transaccion.orden)
	}
}

func TestAnotacionAdministrativaReplayReautorizaYConfirmaMismoRecibo(t *testing.T) {
	e := nuevoEscenarioAnotacionAdministrativa(t, true)
	servicio, d := construirServicioAnotacionAdministrativa(t, e)
	recibo, err := servicio.Registrar(context.Background(), e.solicitud)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if recibo != e.recibo || d.politicas.llamadas != 1 || d.autorizador.llamadas != 1 || d.transaccion.llamadas != 1 ||
		d.transaccion.orden.ExpedienteSiguiente.Version != 0 {
		t.Fatalf("replay sin reacreditación transaccional: recibo=%#v política=%d v3=%d tx=%d", recibo, d.politicas.llamadas, d.autorizador.llamadas, d.transaccion.llamadas)
	}
}

func TestAnotacionAdministrativaRechazaPreparacionCruzadaSinConfirmar(t *testing.T) {
	for _, caso := range []struct {
		nombre string
		mutar  func(*escenarioAnotacionAdministrativa)
	}{
		{"observaciones", func(e *escenarioAnotacionAdministrativa) {
			e.preparacion.Material.Observaciones = "Otra anotación sintética."
		}},
		{"clave", func(e *escenarioAnotacionAdministrativa) {
			e.preparacion.Material.ClaveIdempotencia = "66666666-7777-4888-8999-aaaaaaaaaaaa"
		}},
		{"vinculo", func(e *escenarioAnotacionAdministrativa) { e.preparacion.SeguimientoOriginal.VersionSeguimiento = 2 }},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			e := nuevoEscenarioAnotacionAdministrativa(t, false)
			caso.mutar(&e)
			servicio, d := construirServicioAnotacionAdministrativa(t, e)
			_, err := servicio.Registrar(context.Background(), e.solicitud)
			if !errors.Is(err, ErrResultadoAnotacionAdministrativaNoConfiable) || d.politicas.llamadas != 0 || d.autorizador.llamadas != 0 || d.transaccion.llamadas != 0 {
				t.Fatalf("preparación cruzada alcanzó efectos: err=%v política=%d v3=%d tx=%d", err, d.politicas.llamadas, d.autorizador.llamadas, d.transaccion.llamadas)
			}
		})
	}
}

func TestAnotacionAdministrativaDeniegaYRespetaCancelacion(t *testing.T) {
	t.Run("denegada", func(t *testing.T) {
		e := nuevoEscenarioAnotacionAdministrativa(t, false)
		servicio, d := construirServicioAnotacionAdministrativa(t, e)
		d.autorizador.decisionDenegada = true
		_, err := servicio.Registrar(context.Background(), e.solicitud)
		if !errors.Is(err, ErrAnotacionAdministrativaDenegada) || d.autorizador.llamadas != 1 || d.transaccion.llamadas != 0 {
			t.Fatalf("denegación produjo efecto: err=%v v3=%d tx=%d", err, d.autorizador.llamadas, d.transaccion.llamadas)
		}
	})
	t.Run("cancelada", func(t *testing.T) {
		e := nuevoEscenarioAnotacionAdministrativa(t, false)
		servicio, d := construirServicioAnotacionAdministrativa(t, e)
		ctx, cancelar := context.WithCancel(context.Background())
		cancelar()
		_, err := servicio.Registrar(ctx, e.solicitud)
		if !errors.Is(err, context.Canceled) || d.solicitudes.llamadas != 0 || d.autorizador.llamadas != 0 || d.transaccion.llamadas != 0 {
			t.Fatalf("cancelación inició efectos: err=%v solicitud=%d v3=%d tx=%d", err, d.solicitudes.llamadas, d.autorizador.llamadas, d.transaccion.llamadas)
		}
	})
}

func TestAnotacionAdministrativaRechazaReciboTransaccionalMalVinculado(t *testing.T) {
	e := nuevoEscenarioAnotacionAdministrativa(t, false)
	servicio, d := construirServicioAnotacionAdministrativa(t, e)
	d.transaccion.recibo.EventoRef = "evento:ajeno-sintetico-001"
	_, err := servicio.Registrar(context.Background(), e.solicitud)
	if !errors.Is(err, ErrResultadoAnotacionAdministrativaNoConfiable) || d.transaccion.llamadas != 1 {
		t.Fatalf("recibo cruzado aceptado: err=%v tx=%d", err, d.transaccion.llamadas)
	}
}

func solicitudRecuperarAnotacionAdministrativa(e escenarioAnotacionAdministrativa) SolicitudRegistrarAnotacionAdministrativa {
	return SolicitudRegistrarAnotacionAdministrativa{
		AutenticacionRef:  e.solicitud.AutenticacionRef,
		SesionRef:         e.solicitud.SesionRef,
		PerfilRef:         e.solicitud.PerfilRef,
		OrganizacionRef:   e.solicitud.OrganizacionRef,
		ExpedienteRef:     e.solicitud.ExpedienteRef,
		ClaveIdempotencia: e.solicitud.ClaveIdempotencia,
	}
}

func TestRecuperarAnotacionAdministrativaExigeFuenteAutorizada(t *testing.T) {
	e := nuevoEscenarioAnotacionAdministrativa(t, true)
	servicio, d := construirServicioAnotacionAdministrativa(t, e)
	entrada := solicitudRecuperarAnotacionAdministrativa(e)
	if _, err := servicio.RecuperarAnotacionAdministrativa(context.Background(), entrada); !errors.Is(err, ports.ErrPersistenciaAnotacionAdministrativaNoDisponible) {
		t.Fatalf("recuperación sin fuente = %v", err)
	}
	fuente := &recuperadorMaterialAnotacionDoble{err: ports.ErrAutorizacionDenegada}
	servicio, err := servicio.ConRecuperacionAutorizada(fuente)
	if err != nil {
		t.Fatal(err)
	}
	_, err = servicio.RecuperarAnotacionAdministrativa(context.Background(), entrada)
	if !errors.Is(err, ErrAnotacionAdministrativaDenegada) || fuente.llamadas != 1 ||
		d.preparador.llamadas != 0 || d.autorizador.llamadas != 0 || d.transaccion.llamadas != 0 {
		t.Fatalf("lectura denegada produjo material o efecto: err=%v fuente=%d preparar=%d v3=%d tx=%d", err, fuente.llamadas, d.preparador.llamadas, d.autorizador.llamadas, d.transaccion.llamadas)
	}
}

func TestRecuperarAnotacionAdministrativaReautorizaMaterialOriginal(t *testing.T) {
	e := nuevoEscenarioAnotacionAdministrativa(t, true)
	servicio, d := construirServicioAnotacionAdministrativa(t, e)
	fuente := &recuperadorMaterialAnotacionDoble{material: e.preparacion.Material}
	servicio, err := servicio.ConRecuperacionAutorizada(fuente)
	if err != nil {
		t.Fatal(err)
	}
	entrada := solicitudRecuperarAnotacionAdministrativa(e)
	recibo, err := servicio.RecuperarAnotacionAdministrativa(context.Background(), entrada)
	if err != nil {
		t.Fatalf("recuperar: %v", err)
	}
	if recibo != e.recibo || fuente.llamadas != 1 || fuente.solicitud.VersionEsperada != 0 ||
		fuente.solicitud.OrganizacionRef != entrada.OrganizacionRef || fuente.solicitud.ExpedienteRef != entrada.ExpedienteRef ||
		fuente.solicitud.ClaveIdempotencia != entrada.ClaveIdempotencia || d.politicas.llamadas != 1 ||
		d.autorizador.llamadas != 1 || d.transaccion.llamadas != 1 || d.transaccion.orden.Material != e.preparacion.Material {
		t.Fatalf("recuperación no preservó material ni reacreditó replay: recibo=%#v fuente=%#v política=%d v3=%d tx=%d", recibo, fuente.solicitud, d.politicas.llamadas, d.autorizador.llamadas, d.transaccion.llamadas)
	}
}

func TestRecuperarAnotacionAdministrativaRechazaFuenteCruzadaONueva(t *testing.T) {
	for _, caso := range []struct {
		nombre string
		mutar  func(*escenarioAnotacionAdministrativa, *ports.MaterialAnotacionAdministrativa)
	}{
		{"preparacion_nueva", func(e *escenarioAnotacionAdministrativa, _ *ports.MaterialAnotacionAdministrativa) {
			e.preparacion.Estado = ports.PreparacionAnotacionAdministrativaPreparada
		}},
		{"organizacion", func(_ *escenarioAnotacionAdministrativa, m *ports.MaterialAnotacionAdministrativa) {
			m.OrganizacionRef = "organizacion:ajena-sintetica-001"
		}},
		{"expediente", func(_ *escenarioAnotacionAdministrativa, m *ports.MaterialAnotacionAdministrativa) {
			m.ExpedienteRef = "expediente:ajeno-sintetico-001"
		}},
		{"actor", func(_ *escenarioAnotacionAdministrativa, m *ports.MaterialAnotacionAdministrativa) {
			m.ActorRef = "persona:ajena-sintetica-001"
		}},
		{"perfil", func(_ *escenarioAnotacionAdministrativa, m *ports.MaterialAnotacionAdministrativa) {
			m.PerfilRef = "perfil:ajeno-sintetico-001"
		}},
		{"clave", func(_ *escenarioAnotacionAdministrativa, m *ports.MaterialAnotacionAdministrativa) {
			m.ClaveIdempotencia = "66666666-7777-4888-8999-aaaaaaaaaaaa"
		}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			e := nuevoEscenarioAnotacionAdministrativa(t, true)
			material := e.preparacion.Material
			caso.mutar(&e, &material)
			servicio, d := construirServicioAnotacionAdministrativa(t, e)
			fuente := &recuperadorMaterialAnotacionDoble{material: material}
			servicio, err := servicio.ConRecuperacionAutorizada(fuente)
			if err != nil {
				t.Fatal(err)
			}
			_, err = servicio.RecuperarAnotacionAdministrativa(context.Background(), solicitudRecuperarAnotacionAdministrativa(e))
			if !errors.Is(err, ErrResultadoAnotacionAdministrativaNoConfiable) || fuente.llamadas != 1 ||
				d.politicas.llamadas != 0 || d.autorizador.llamadas != 0 || d.transaccion.llamadas != 0 {
				t.Fatalf("material o preparación no recuperable produjo efecto: err=%v política=%d v3=%d tx=%d", err, d.politicas.llamadas, d.autorizador.llamadas, d.transaccion.llamadas)
			}
		})
	}
}
