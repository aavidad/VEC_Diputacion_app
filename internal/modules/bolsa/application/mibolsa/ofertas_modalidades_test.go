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
	orden       Orden
	llamadas    int
}

func (c *consultorOfertasModalidadesPrueba) Consultar(_ context.Context, orden Orden) (bolsa.InstantaneaMiBolsa, error) {
	c.llamadas++
	c.orden = orden
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

func TestConsultaOfertasModalidades(t *testing.T) {
	orden := Orden{Motivo: core.ReferenciaEntradaCatalogo{CatalogoID: "motivos_modalidades", EntradaClave: "consultar"}}
	oferta := func(ref, modalidad, estado string, disposicion bool) bolsa.OfertaPortalCandidato {
		o := bolsa.OfertaPortalCandidato{OfertaRef: ref, Datos: dominio.DatosOferta{ModalidadClave: modalidad}, Estado: estado}
		if disposicion {
			o.Disposicion = &bolsa.DisposicionPropiaPortal{Recibo: "recibo:propio"}
		}
		return o
	}
	base := []bolsa.OfertaPortalCandidato{
		oferta("coincide", "interinidad", bolsa.EstadoOfertaPortalAbierta, false),
		oferta("otra", "eventual", bolsa.EstadoOfertaPortalAbierta, false),
		oferta("historica", "", bolsa.EstadoOfertaPortalAbierta, false),
		oferta("disposicion", "eventual", bolsa.EstadoOfertaPortalAbierta, true),
		oferta("antecedente", "eventual", bolsa.EstadoOfertaPortalResuelta, false),
	}
	casos := []struct {
		nombre    string
		version   uint64
		elegidas  []string
		esperadas []string
	}{
		{"sin preferencia", 0, nil, []string{"coincide", "otra", "historica", "disposicion", "antecedente"}},
		{"restablecida a todas", 2, nil, []string{"coincide", "otra", "historica", "disposicion", "antecedente"}},
		{"selección", 1, []string{"interinidad"}, []string{"coincide", "historica", "disposicion", "antecedente"}},
		{"vacío explícito", 1, []string{}, []string{"historica", "disposicion", "antecedente"}},
		{"historia y disposición", 2, []string{"eventual"}, []string{"otra", "historica", "disposicion", "antecedente"}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			original := append([]bolsa.OfertaPortalCandidato(nil), base...)
			consulta := &consultorOfertasModalidadesPrueba{instantanea: bolsa.InstantaneaMiBolsa{Ofertas: base}}
			lector := &lectorOfertasModalidadesPrueba{estado: bolsa.EstadoPreferenciasModalidades{Version: caso.version, ModalidadesGuardadas: caso.elegidas}}
			servicio, err := NuevoConsultaOfertasModalidades(consulta, lector)
			if err != nil {
				t.Fatal(err)
			}
			resultado, err := servicio.Consultar(t.Context(), orden)
			if err != nil {
				t.Fatal(err)
			}
			refs := make([]string, 0, len(resultado.Ofertas))
			for _, o := range resultado.Ofertas {
				refs = append(refs, o.OfertaRef)
			}
			if !reflect.DeepEqual(refs, caso.esperadas) || !reflect.DeepEqual(consulta.instantanea.Ofertas, original) ||
				consulta.llamadas != 1 || lector.llamadas != 1 || !reflect.DeepEqual(consulta.orden, orden) || !reflect.DeepEqual(lector.orden, orden) {
				t.Fatalf("refs=%v, consultas=%d, preferencias=%d", refs, consulta.llamadas, lector.llamadas)
			}
		})
	}
	t.Run("ver todas", func(t *testing.T) {
		consulta := &consultorOfertasModalidadesPrueba{instantanea: bolsa.InstantaneaMiBolsa{Ofertas: base}}
		lector := &lectorOfertasModalidadesPrueba{err: errors.New("lector no debe usarse")}
		servicio, err := NuevoConsultaOfertasModalidades(consulta, lector)
		if err != nil {
			t.Fatal(err)
		}
		resultado, err := servicio.ConsultarTodasLasOfertas(t.Context(), orden)
		if err != nil || len(resultado.Ofertas) != len(base) || consulta.llamadas != 1 || lector.llamadas != 0 || !reflect.DeepEqual(consulta.orden, orden) {
			t.Fatalf("ver todas: ofertas=%d, consultas=%d, preferencias=%d, err=%v", len(resultado.Ofertas), consulta.llamadas, lector.llamadas, err)
		}
	})
}

func TestConsultaOfertasModalidadesErrores(t *testing.T) {
	var consultaNula *consultorOfertasModalidadesPrueba
	if _, err := NuevoConsultaOfertasModalidades(consultaNula, &lectorOfertasModalidadesPrueba{}); !errors.Is(err, ErrServicioMiBolsaInvalido) {
		t.Fatalf("consultor nulo: %v", err)
	}
	if _, err := NuevoConsultaOfertasModalidades(&consultorOfertasModalidadesPrueba{}, (*lectorOfertasModalidadesPrueba)(nil)); !errors.Is(err, ErrServicioMiBolsaInvalido) {
		t.Fatalf("lector nulo: %v", err)
	}
	falloConsulta, falloPreferencias := errors.New("consulta no disponible"), errors.New("preferencias no disponibles")
	consulta := &consultorOfertasModalidadesPrueba{err: falloConsulta}
	lector := &lectorOfertasModalidadesPrueba{err: falloPreferencias}
	servicio, err := NuevoConsultaOfertasModalidades(consulta, lector)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := servicio.Consultar(t.Context(), Orden{}); !errors.Is(err, falloConsulta) || lector.llamadas != 0 {
		t.Fatalf("fallo de consulta oculto: %v", err)
	}
	consulta.err = nil
	consulta.instantanea.Ofertas = []bolsa.OfertaPortalCandidato{{OfertaRef: "oferta:1", Estado: bolsa.EstadoOfertaPortalAbierta}}
	consulta.instantanea.Participaciones = []bolsa.ParticipacionMiBolsa{{Bolsa: "bolsa:propia"}}
	resultado, err := servicio.Consultar(t.Context(), Orden{})
	if err != nil || !resultado.OfertasNoDisponibles || len(resultado.Ofertas) != 0 ||
		!reflect.DeepEqual(resultado.Participaciones, consulta.instantanea.Participaciones) || len(consulta.instantanea.Ofertas) != 1 {
		t.Fatalf("fallo parcial no preservó participaciones ni cerró ofertas: %+v, %v", resultado, err)
	}
	resultado, err = servicio.ConsultarTodasLasOfertas(t.Context(), Orden{})
	if err != nil || resultado.OfertasNoDisponibles || len(resultado.Ofertas) != 1 || lector.llamadas != 1 {
		t.Fatalf("consulta explícita de todas no recuperó la oferta: %+v, %v", resultado, err)
	}
}

func TestConsultaOfertasModalidadesSinOfertasEvitaOtraLectura(t *testing.T) {
	consulta := &consultorOfertasModalidadesPrueba{}
	lector := &lectorOfertasModalidadesPrueba{err: errors.New("preferencias no requeridas")}
	servicio, err := NuevoConsultaOfertasModalidades(consulta, lector)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := servicio.Consultar(t.Context(), Orden{})
	if err != nil || resultado.Ofertas != nil || !resultado.OfertasSegunModalidades ||
		consulta.llamadas != 1 || lector.llamadas != 0 {
		t.Fatalf("consulta vacía: preferencias=%d, error=%v", lector.llamadas, err)
	}
}
