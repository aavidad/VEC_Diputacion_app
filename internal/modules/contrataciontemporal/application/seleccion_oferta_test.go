package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/pruebas"
)

type relojSeleccionOfertaPrueba struct{ ahora time.Time }

func (r relojSeleccionOfertaPrueba) Ahora() time.Time { return r.ahora }

type generadorCorrelacionSeleccionOfertaPrueba struct{}

func (generadorCorrelacionSeleccionOfertaPrueba) NuevaReferenciaCorrelacionAutorizacionV2(context.Context) (string, error) {
	return "correlacion_0123456789abcdef0123456789abcdef", nil
}

type preparadorSeleccionOfertaPrueba struct {
	intencion     ports.IntencionSeleccionOfertaCT
	comando       ports.ComandoOfertaBolsaCT
	preparaciones int
	consultas     int
}

func (p *preparadorSeleccionOfertaPrueba) PrepararIntencion(_ context.Context, _ string, _ uint64, _ string) (ports.IntencionSeleccionOfertaCT, error) {
	p.consultas++
	return p.intencion, nil
}
func (p *preparadorSeleccionOfertaPrueba) PrepararComando(_ context.Context, _ ports.IntencionSeleccionOfertaCT) (ports.ComandoOfertaBolsaCT, error) {
	p.preparaciones++
	return p.comando, nil
}

type autorizadorSeleccionOfertaPrueba struct {
	etapas                 []string
	denegarConfirmarUnaVez bool
}

func (a *autorizadorSeleccionOfertaPrueba) AutorizarSeleccionOfertaCT(_ context.Context, s ports.SolicitudAutorizacionSeleccionOfertaCT) (ports.AutorizacionSeleccionOfertaCT, error) {
	canon, err := ports.CanonEtapaSeleccionOfertaCT(s)
	if err != nil {
		return ports.AutorizacionSeleccionOfertaCT{}, err
	}
	a.etapas = append(a.etapas, s.Etapa)
	if s.Etapa == ports.EtapaSeleccionOfertaConfirmar && a.denegarConfirmarUnaVez {
		a.denegarConfirmarUnaVez = false
		return ports.AutorizacionSeleccionOfertaCT{}, ports.ErrAutorizacionDenegada
	}
	return ports.AutorizacionSeleccionOfertaCT{ActorRef: s.Intencion.ActorRef, PerfilRef: s.Intencion.PerfilRef, MaterialCanonico: canon}, nil
}

type registroSeleccionOfertaPrueba struct {
	estado                                        ports.EstadoSeleccionOfertaCT
	consultas, reservas, ventanas, confirmaciones int
	orden                                         []string
	denegarConfirmacionUnaVez                     bool
}

func (r *registroSeleccionOfertaPrueba) Consultar(_ context.Context, _ ports.IntencionSeleccionOfertaCT, _ ports.AutorizacionSeleccionOfertaCT) (ports.EstadoSeleccionOfertaCT, error) {
	r.consultas++
	r.orden = append(r.orden, "consultar")
	return r.estado, nil
}
func (r *registroSeleccionOfertaPrueba) Reservar(_ context.Context, _ ports.IntencionSeleccionOfertaCT, c ports.ComandoOfertaBolsaCT, _ ports.AutorizacionSeleccionOfertaCT) (ports.EstadoSeleccionOfertaCT, error) {
	r.reservas++
	r.orden = append(r.orden, "reservar")
	if r.estado.Situacion != ports.SeleccionOfertaAusente {
		return r.estado, nil
	}
	r.estado = ports.EstadoSeleccionOfertaCT{Situacion: ports.SeleccionOfertaReservada, ReservaRef: "reserva:oferta:ct:1", Comando: c}
	return r.estado, nil
}
func (r *registroSeleccionOfertaPrueba) AbrirVentana(_ context.Context, _ ports.ReservaSeleccionOfertaCT, _ ports.AutorizacionSeleccionOfertaCT) error {
	r.ventanas++
	r.orden = append(r.orden, "abrir")
	r.estado.VentanaAbierta = true
	return nil
}
func (r *registroSeleccionOfertaPrueba) ConfirmarVinculo(_ context.Context, reserva ports.ReservaSeleccionOfertaCT, o ports.OfertaPublicadaSeleccionCT, _ ports.AutorizacionSeleccionOfertaCT) (ports.ReciboSeleccionOfertaCT, error) {
	r.confirmaciones++
	r.orden = append(r.orden, "confirmar")
	if r.denegarConfirmacionUnaVez {
		r.denegarConfirmacionUnaVez = false
		return ports.ReciboSeleccionOfertaCT{}, ports.ErrAutorizacionDenegada
	}
	recibo := ports.ReciboSeleccionOfertaCT{
		Esquema: ports.EsquemaReciboSeleccionOfertaCTV1, Estado: ports.EstadoReciboSeleccionOfertaCTV1,
		OrganizacionRef: reserva.Intencion.OrganizacionRef, ExpedienteRef: reserva.Intencion.ExpedienteRef,
		VersionExpediente: reserva.Intencion.VersionExpediente, BolsaRef: o.BolsaRef,
		OfertaRef: o.OfertaRef, ReciboOfertaRef: o.ReciboOfertaRef,
		ReciboVinculoRef: "recibo:vinculo:oferta:ct:1", NumeroPlazas: o.NumeroPlazas,
		PublicadaEn: o.PublicadaEn, VenceAntesDe: o.VenceAntesDe,
	}
	r.estado = ports.EstadoSeleccionOfertaCT{Situacion: ports.SeleccionOfertaConfirmada, Comando: reserva.Comando, Recibo: recibo}
	return recibo, nil
}

type publicadorSeleccionOfertaPrueba struct {
	llamadas        int
	orden           []string
	origenVisto     string
	perderRespuesta bool
	alterarCantidad bool
	oferta          ports.OfertaPublicadaSeleccionCT
}

func (p *publicadorSeleccionOfertaPrueba) PublicarORecuperar(_ context.Context, _ ports.IntencionSeleccionOfertaCT, c ports.ComandoOfertaBolsaCT) (ports.OfertaPublicadaSeleccionCT, error) {
	p.llamadas++
	p.orden = append(p.orden, "publicar")
	p.origenVisto = string(c.OrigenCT)
	if p.oferta.OfertaRef == "" {
		p.oferta = ports.OfertaPublicadaSeleccionCT{ClaveBolsa: c.ClaveBolsa,
			BolsaRef: c.BolsaRef, OfertaRef: "oferta:seleccion:1", ReciboOfertaRef: "recibo:oferta:seleccion:1",
			NumeroPlazas: c.NumeroPlazas, Datos: c.Datos,
			PublicadaEn:  time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC),
			VenceAntesDe: time.Date(2026, 10, 19, 22, 0, 0, 0, time.UTC)}
	}
	if p.alterarCantidad {
		p.oferta.NumeroPlazas++
	}
	if p.perderRespuesta {
		p.perderRespuesta = false
		return ports.OfertaPublicadaSeleccionCT{}, errors.New("respuesta perdida")
	}
	return p.oferta, nil
}

func escenarioSeleccionOfertaPrueba(t *testing.T) (*ServicioSeleccionOfertaCT, *preparadorSeleccionOfertaPrueba, *registroSeleccionOfertaPrueba, *publicadorSeleccionOfertaPrueba, *autorizadorSeleccionOfertaPrueba, SolicitudSeleccionLlamamiento) {
	t.Helper()
	ahora := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	actor, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(ahora,
		"per_0123456789abcdefghijkl", "prf_0123456789abcdefghijkl",
		dominiovec.AuthMethodCertificate, dominiovec.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(t.Context(), generadorCorrelacionSeleccionOfertaPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	i := ports.IntencionSeleccionOfertaCT{
		OrganizacionRef: "organizacion:prueba", ExpedienteRef: "expediente:ct:prueba",
		CategoriaRef: "categoria:rpt:administrativo", CentroRef: "centro:520",
		VersionExpediente: 6, ClaveIdempotencia: "018f47a2-6b31-4c80-8a95-4d2e707c5a11",
		ActorRef: actor.Contexto.PersonaRef, PerfilRef: actor.Contexto.PerfilActivoRef,
		Vinculo: vinculo, ResultadoContexto: actor, Correlacion: correlacion,
		MotivoAutorizacion: dominiovec.ReferenciaEntradaCatalogo{CatalogoID: "motivos_autorizacion", CatalogoVersion: 1,
			CatalogoHuellaSHA256: strings.Repeat("a", 64), EntradaClave: "motivo_0123456789abcdef0123456789abcdef"},
	}
	i.HuellaIntencion = ports.HuellaIntencionSeleccionOfertaCT(i)
	if err := i.Validar(); err != nil {
		t.Fatal(err)
	}
	c := ports.ComandoOfertaBolsaCT{
		OrganizacionRef: i.OrganizacionRef, ExpedienteRef: i.ExpedienteRef,
		VersionExpediente: i.VersionExpediente, ActorRef: i.ActorRef,
		BolsaRef: "bolsa:administrativo:prueba", NecesidadRef: "necesidad:rpt:prueba",
		NumeroPlazas: 3, NumeroElegibles: 4,
		Datos: ports.DatosOfertaSeleccionCT{Categoria: "Administrativo", Centro: "Centro 520",
			FechaInicio: "2026-11-01", FechaFin: "2027-01-31", Descripcion: "Tres plazas administrativas"},
		OrigenCT:   []byte(`{"acta_ref":"acta:prueba","lectura":{"corte":"2026-10-15T12:00:00.000000Z","entradas":[1,2,3,4]}}`),
		ClaveBolsa: ports.ClaveBolsaSeleccionOfertaCT(i), InicioPlazo: ports.InicioSeleccionOfertaPublicacion,
	}
	if err := c.ValidarPara(i); err != nil {
		t.Fatal(err)
	}
	prep := &preparadorSeleccionOfertaPrueba{intencion: i, comando: c}
	reg := &registroSeleccionOfertaPrueba{estado: ports.EstadoSeleccionOfertaCT{Situacion: ports.SeleccionOfertaAusente}}
	pub := &publicadorSeleccionOfertaPrueba{}
	aut := &autorizadorSeleccionOfertaPrueba{}
	s, err := NuevoServicioSeleccionOfertaCT(prep, aut, reg, pub, relojSeleccionOfertaPrueba{ahora})
	if err != nil {
		t.Fatal(err)
	}
	q := SolicitudSeleccionLlamamiento{ExpedienteRef: i.ExpedienteRef, VersionEsperada: i.VersionExpediente, ClaveIdempotencia: i.ClaveIdempotencia}
	return s, prep, reg, pub, aut, q
}

func TestSeleccionOfertaCongelaComandoAntesDeBolsaYConfirmaVinculo(t *testing.T) {
	s, prep, reg, pub, aut, q := escenarioSeleccionOfertaPrueba(t)
	r, err := s.SeleccionarOferta(t.Context(), q)
	if err != nil || r.Estado != ports.EstadoReciboSeleccionOfertaCTV1 || r.NumeroPlazas != 3 ||
		reg.consultas != 1 || reg.reservas != 1 || reg.ventanas != 1 || reg.confirmaciones != 1 ||
		pub.llamadas != 1 || len(aut.etapas) != 4 || prep.preparaciones != 1 {
		t.Fatalf("recibo=%+v err=%v etapas=%v", r, err, aut.etapas)
	}
	if got := strings.Join(aut.etapas, ","); got != "consultar,reservar,abrir_ventana,confirmar" ||
		reg.orden[0] != "consultar" || reg.orden[1] != "reservar" || reg.orden[2] != "abrir" ||
		!strings.Contains(pub.origenVisto, `"acta_ref":"acta:prueba"`) {
		t.Fatalf("orden/autoridad/mandato: %v %v %q", reg.orden, aut.etapas, pub.origenVisto)
	}
}

func TestSeleccionOfertaReintentaComandoCongeladoTrasRespuestaPerdida(t *testing.T) {
	s, prep, reg, pub, _, q := escenarioSeleccionOfertaPrueba(t)
	pub.perderRespuesta = true
	if _, err := s.SeleccionarOferta(t.Context(), q); !errors.Is(err, ErrSeleccionOfertaIndeterminada) {
		t.Fatalf("respuesta perdida: %v", err)
	}
	congelado := pub.origenVisto
	prep.comando.OrigenCT = []byte(`{"acta_ref":"acta:nueva","lectura":{"corte":"2026-10-16T12:00:00.000000Z","entradas":[1,2,3,4]}}`)
	prep.comando.NumeroPlazas = 4
	r, err := s.SeleccionarOferta(t.Context(), q)
	if err != nil || r.NumeroPlazas != 3 || pub.origenVisto != congelado ||
		reg.reservas != 1 || reg.ventanas != 1 || prep.preparaciones != 1 || pub.llamadas != 2 {
		t.Fatalf("replay reconstruyó origen/cantidad: recibo=%+v err=%v", r, err)
	}
	// El terminal histórico se devuelve sin reabrir Bolsa ni consultar B110.
	antes := pub.llamadas
	if repetido, err := s.SeleccionarOferta(t.Context(), q); err != nil || repetido != r || pub.llamadas != antes || prep.preparaciones != 1 {
		t.Fatalf("terminal inestable: %+v %v", repetido, err)
	}
}

func TestSeleccionOfertaRecuperaTrasDenegacionDeConfirmacionPostPublicacion(t *testing.T) {
	s, prep, reg, pub, aut, q := escenarioSeleccionOfertaPrueba(t)
	aut.denegarConfirmarUnaVez = true
	if _, err := s.SeleccionarOferta(t.Context(), q); !errors.Is(err, ErrSeleccionOfertaIndeterminada) ||
		!errors.Is(err, ports.ErrAutorizacionDenegada) || pub.llamadas != 1 || reg.confirmaciones != 0 {
		t.Fatalf("fallo postpublicacion se trató como terminal: %v", err)
	}
	recibo, err := s.SeleccionarOferta(t.Context(), q)
	if err != nil || recibo.Estado != ports.EstadoReciboSeleccionOfertaCTV1 ||
		reg.reservas != 1 || reg.ventanas != 1 || reg.confirmaciones != 1 ||
		prep.preparaciones != 1 || pub.llamadas != 2 {
		t.Fatalf("no se recuperó la oferta publicada: %+v %v", recibo, err)
	}
}

func TestSeleccionOfertaRecuperaTrasFalloDeConfirmacionDurable(t *testing.T) {
	s, prep, reg, pub, _, q := escenarioSeleccionOfertaPrueba(t)
	reg.denegarConfirmacionUnaVez = true
	if _, err := s.SeleccionarOferta(t.Context(), q); !errors.Is(err, ErrSeleccionOfertaIndeterminada) ||
		!errors.Is(err, ports.ErrAutorizacionDenegada) || pub.llamadas != 1 || reg.confirmaciones != 1 {
		t.Fatalf("fallo CT postpublicacion se trató como definitivo: %v", err)
	}
	recibo, err := s.SeleccionarOferta(t.Context(), q)
	if err != nil || recibo.Estado != ports.EstadoReciboSeleccionOfertaCTV1 ||
		reg.reservas != 1 || reg.ventanas != 1 || reg.confirmaciones != 2 ||
		prep.preparaciones != 1 || pub.llamadas != 2 {
		t.Fatalf("no se recuperó vínculo CT: %+v %v", recibo, err)
	}
}

func TestSeleccionOfertaDeniegaOcupacionYRespuestaBolsaDivergente(t *testing.T) {
	s, _, reg, pub, _, q := escenarioSeleccionOfertaPrueba(t)
	reg.estado.Situacion = ports.SeleccionOfertaOcupada
	if _, err := s.SeleccionarOferta(t.Context(), q); !errors.Is(err, ErrSeleccionOfertaConcurrente) || reg.reservas != 0 || pub.llamadas != 0 {
		t.Fatalf("ocupación permitida: %v", err)
	}
	s, _, reg, pub, _, q = escenarioSeleccionOfertaPrueba(t)
	pub.alterarCantidad = true
	if _, err := s.SeleccionarOferta(t.Context(), q); !errors.Is(err, ErrResultadoSeleccionOfertaNoConfiable) || reg.confirmaciones != 0 {
		t.Fatalf("oferta con N distinto se vinculó: %v", err)
	}
}

func TestSeleccionOfertaNoPublicaUnaPlazaNiMasQueElegibles(t *testing.T) {
	for _, caso := range []struct {
		nombre            string
		plazas, elegibles int
	}{{"singular", 1, 4}, {"excede elegibles", 5, 4}, {"excede cien", 101, 101}} {
		t.Run(caso.nombre, func(t *testing.T) {
			s, prep, reg, pub, _, q := escenarioSeleccionOfertaPrueba(t)
			prep.comando.NumeroPlazas, prep.comando.NumeroElegibles = caso.plazas, caso.elegibles
			if _, err := s.SeleccionarOferta(t.Context(), q); !errors.Is(err, ErrSeleccionOfertaNoDisponible) ||
				reg.reservas != 0 || reg.ventanas != 0 || pub.llamadas != 0 {
				t.Fatalf("cantidad invalida alcanzo Bolsa: %v", err)
			}
		})
	}
}
