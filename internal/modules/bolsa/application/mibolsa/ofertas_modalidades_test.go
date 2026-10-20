package mibolsa

import (
	"context"
	"errors"
	"reflect"
	"testing"

	dominio "vec-diputacion-granada/internal/modules/bolsa/domain"
	bolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

type consultorOfertasModalidadesPrueba struct {
	instantanea bolsa.InstantaneaMiBolsa
	err         error
	llamadas    int
}

func (c *consultorOfertasModalidadesPrueba) Consultar(context.Context, Orden) (bolsa.InstantaneaMiBolsa, error) {
	c.llamadas++
	return c.instantanea, c.err
}

type lectorOfertasModalidadesPrueba struct {
	estado   bolsa.EstadoPreferenciasModalidades
	err      error
	orden    Orden
	llamadas int
}

func (l *lectorOfertasModalidadesPrueba) ConsultarPreferenciasModalidades(_ context.Context, orden Orden) (bolsa.EstadoPreferenciasModalidades, error) {
	l.llamadas++
	l.orden = orden
	return l.estado, l.err
}

func ofertaModalidadPrueba(ref, modalidad, estado string, respondida bool) bolsa.OfertaPortalCandidato {
	o := bolsa.OfertaPortalCandidato{OfertaRef: ref, Datos: dominio.DatosOferta{ModalidadClave: modalidad}, Estado: estado}
	if respondida {
		o.Disposicion = &bolsa.DisposicionPropiaPortal{Recibo: "recibo:propio"}
	}
	return o
}

func referenciasOfertas(ofertas []bolsa.OfertaPortalCandidato) []string {
	refs := make([]string, 0, len(ofertas))
	for _, o := range ofertas {
		refs = append(refs, o.OfertaRef)
	}
	return refs
}

func instantaneaOfertasPrueba() bolsa.InstantaneaMiBolsa {
	return bolsa.InstantaneaMiBolsa{
		Participaciones: []bolsa.ParticipacionMiBolsa{{Bolsa: "bolsa:auxiliar", OrdenInicial: 4, TotalInstantanea: 9}},
		Ofertas: []bolsa.OfertaPortalCandidato{
			ofertaModalidadPrueba("vacante", "vacante", bolsa.EstadoOfertaPortalAbierta, false),
			ofertaModalidadPrueba("sustitucion", "sustitucion", bolsa.EstadoOfertaPortalAbierta, false),
			ofertaModalidadPrueba("sin-modalidad", "", bolsa.EstadoOfertaPortalAbierta, false),
			ofertaModalidadPrueba("respondida", "sustitucion", bolsa.EstadoOfertaPortalAbierta, true),
			ofertaModalidadPrueba("resuelta", "sustitucion", bolsa.EstadoOfertaPortalResuelta, false),
		},
	}
}

func TestConsultaOfertasModalidadesAplicaLaEleccion(t *testing.T) {
	todas := []string{"vacante", "sustitucion", "sin-modalidad", "respondida", "resuelta"}
	casos := []struct {
		nombre    string
		version   uint64
		elegidas  []string
		esperadas []string
	}{
		{"sin elección guardada", 0, nil, todas},
		{"todas guardada", 2, nil, todas},
		{"solo vacante", 1, []string{"vacante"}, []string{"vacante", "sin-modalidad", "respondida", "resuelta"}},
		{"ninguna", 1, []string{}, []string{"sin-modalidad", "respondida", "resuelta"}},
	}
	orden := Orden{Motivo: core.ReferenciaEntradaCatalogo{CatalogoID: "motivos", EntradaClave: "consultar"}}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			consulta := &consultorOfertasModalidadesPrueba{instantanea: instantaneaOfertasPrueba()}
			lector := &lectorOfertasModalidadesPrueba{estado: bolsa.EstadoPreferenciasModalidades{Version: caso.version, ModalidadesGuardadas: caso.elegidas}}
			s, err := NuevaConsultaOfertasModalidades(consulta, lector)
			if err != nil {
				t.Fatal(err)
			}
			got, err := s.Consultar(context.Background(), orden)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(referenciasOfertas(got.Ofertas), caso.esperadas) {
				t.Fatalf("ofertas=%v, esperadas=%v", referenciasOfertas(got.Ofertas), caso.esperadas)
			}
			if !got.OfertasSegunModalidades || lector.llamadas != 1 || !reflect.DeepEqual(lector.orden, orden) {
				t.Fatalf("indicador=%v llamadas=%d", got.OfertasSegunModalidades, lector.llamadas)
			}
			// La posición y el resto de la instantánea no cambian.
			if !reflect.DeepEqual(got.Participaciones, instantaneaOfertasPrueba().Participaciones) {
				t.Fatal("la elección alteró las participaciones")
			}
		})
	}
}

func TestConsultaOfertasModalidadesSinLecturaMuestraTodas(t *testing.T) {
	for _, causa := range []error{core.ErrAutorizacionDenegada, bolsa.ErrPreferenciasNoDisponibles, errors.New("fallo interno con datos")} {
		consulta := &consultorOfertasModalidadesPrueba{instantanea: instantaneaOfertasPrueba()}
		lector := &lectorOfertasModalidadesPrueba{err: causa}
		s, _ := NuevaConsultaOfertasModalidades(consulta, lector)
		got, err := s.Consultar(context.Background(), Orden{})
		if err != nil {
			t.Fatalf("%v: %v", causa, err)
		}
		if got.OfertasSegunModalidades || len(got.Ofertas) != 5 {
			t.Fatalf("%v: indicador=%v ofertas=%d", causa, got.OfertasSegunModalidades, len(got.Ofertas))
		}
	}
}

func TestConsultaOfertasModalidadesNoLeeSinOfertasFiltrables(t *testing.T) {
	sinFiltrables := bolsa.InstantaneaMiBolsa{Ofertas: []bolsa.OfertaPortalCandidato{
		ofertaModalidadPrueba("sin-modalidad", "", bolsa.EstadoOfertaPortalAbierta, false),
		ofertaModalidadPrueba("respondida", "vacante", bolsa.EstadoOfertaPortalAbierta, true),
	}}
	for _, inst := range []bolsa.InstantaneaMiBolsa{{}, sinFiltrables} {
		lector := &lectorOfertasModalidadesPrueba{}
		s, _ := NuevaConsultaOfertasModalidades(&consultorOfertasModalidadesPrueba{instantanea: inst}, lector)
		got, err := s.Consultar(context.Background(), Orden{})
		if err != nil || lector.llamadas != 0 || got.OfertasSegunModalidades || len(got.Ofertas) != len(inst.Ofertas) {
			t.Fatalf("err=%v llamadas=%d", err, lector.llamadas)
		}
	}
}

func TestConsultaOfertasModalidadesPropagaErroresYCancelacion(t *testing.T) {
	fallo := errors.New("consulta propia")
	lector := &lectorOfertasModalidadesPrueba{}
	s, _ := NuevaConsultaOfertasModalidades(&consultorOfertasModalidadesPrueba{err: fallo}, lector)
	if _, err := s.Consultar(context.Background(), Orden{}); !errors.Is(err, fallo) || lector.llamadas != 0 {
		t.Fatalf("err=%v llamadas=%d", err, lector.llamadas)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	s, _ = NuevaConsultaOfertasModalidades(&consultorOfertasModalidadesPrueba{instantanea: instantaneaOfertasPrueba()},
		&lectorOfertasModalidadesPrueba{err: context.Canceled})
	if _, err := s.Consultar(ctx, Orden{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v", err)
	}
	if _, err := NuevaConsultaOfertasModalidades(nil, lector); !errors.Is(err, ErrServicioMiBolsaInvalido) {
		t.Fatal("aceptó una consulta nula")
	}
	var nilLector *lectorOfertasModalidadesPrueba
	if _, err := NuevaConsultaOfertasModalidades(&consultorOfertasModalidadesPrueba{}, nilLector); !errors.Is(err, ErrServicioMiBolsaInvalido) {
		t.Fatal("aceptó un lector nulo con tipo")
	}
}

func TestConsultaOfertasModalidadesVerTodasNoLeeLaEleccion(t *testing.T) {
	consulta := &consultorOfertasModalidadesPrueba{instantanea: instantaneaOfertasPrueba()}
	lector := &lectorOfertasModalidadesPrueba{estado: bolsa.EstadoPreferenciasModalidades{Version: 1, ModalidadesGuardadas: []string{}}}
	s, _ := NuevaConsultaOfertasModalidades(consulta, lector)
	got, err := s.ConsultarTodasOfertas(context.Background(), Orden{})
	if err != nil || lector.llamadas != 0 || consulta.llamadas != 1 || got.OfertasSegunModalidades || len(got.Ofertas) != 5 {
		t.Fatalf("err=%v lector=%d consulta=%d indicador=%v ofertas=%d", err, lector.llamadas, consulta.llamadas, got.OfertasSegunModalidades, len(got.Ofertas))
	}
	var nulo *ConsultaOfertasModalidades
	if _, err := nulo.ConsultarTodasOfertas(context.Background(), Orden{}); !errors.Is(err, ErrServicioMiBolsaInvalido) {
		t.Fatalf("nulo: %v", err)
	}
}
