package application

import (
	"context"
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/documentos/ports"
)

// autorizadorExternaPrueba devuelve la autorización fija del escenario y
// registra la preimagen que le pide el servicio.
type autorizadorExternaPrueba struct {
	autorizacion ports.AutorizacionV3
	err          error
	preimagen    []byte
	documento    string
	expediente   string
	llamadas     int
}

func (a *autorizadorExternaPrueba) AutorizarRegistroExterno(_ context.Context, preimagen []byte, documento, expediente string) (ports.AutorizacionV3, error) {
	a.llamadas++
	a.preimagen, a.documento, a.expediente = append([]byte(nil), preimagen...), documento, expediente
	return a.autorizacion, a.err
}

func TestRegistrarExternoAutorizadoPideV3TrasResolverLaPolitica(t *testing.T) {
	servicio, repo, in := escenarioExterna(t)
	autorizador := &autorizadorExternaPrueba{autorizacion: in.Autorizacion}
	in.Autorizacion = ports.AutorizacionV3{}
	d, err := servicio.RegistrarExternoAutorizado(context.Background(), in, autorizador)
	if err != nil || repo.llamadas != 1 || autorizador.llamadas != 1 {
		t.Fatalf("registro autorizado: %v repo=%d autorizador=%d", err, repo.llamadas, autorizador.llamadas)
	}
	esperada, _ := repo.recibido.PreimagenExterna()
	if string(autorizador.preimagen) != string(esperada) || autorizador.documento != in.ID || autorizador.expediente != in.ExpedienteRef ||
		!strings.Contains(string(esperada), `"conservacion_hasta"`) || d.CustodiaExternaRef != in.Custodia {
		t.Fatalf("la concesión no se ligó a la preimagen confirmada: %s", autorizador.preimagen)
	}
}

func TestRegistrarExternoAutorizadoNoConfirmaSinConcesionLigada(t *testing.T) {
	casos := map[string]func(*autorizadorExternaPrueba, *ports.AltaExterna){
		"denegada": func(a *autorizadorExternaPrueba, _ *ports.AltaExterna) {
			a.err = ports.ErrAccesoDenegado
		},
		"otro recurso": func(a *autorizadorExternaPrueba, _ *ports.AltaExterna) {
			a.autorizacion.RecursoRef = "ref:" + strings.Repeat("7", 64)
		},
		"otro ambito": func(a *autorizadorExternaPrueba, _ *ports.AltaExterna) {
			a.autorizacion.AmbitoRef = "ref:" + strings.Repeat("6", 64)
		},
		"accion de listado": func(a *autorizadorExternaPrueba, _ *ports.AltaExterna) {
			a.autorizacion.Accion = ports.AccionListar
		},
		"preimagen de otra referencia": func(_ *autorizadorExternaPrueba, in *ports.AltaExterna) {
			in.Custodia.Referencia = "justificante:0002"
		},
	}
	for nombre, alterar := range casos {
		servicio, repo, in := escenarioExterna(t)
		autorizador := &autorizadorExternaPrueba{autorizacion: in.Autorizacion}
		in.Autorizacion = ports.AutorizacionV3{}
		alterar(autorizador, &in)
		if _, err := servicio.RegistrarExternoAutorizado(context.Background(), in, autorizador); err == nil || repo.llamadas != 0 {
			t.Errorf("%s: err=%v llamadas=%d", nombre, err, repo.llamadas)
		}
	}
}

func TestRegistrarExternoAutorizadoRechazaAutorizacionAportadaOAutorizadorNulo(t *testing.T) {
	servicio, repo, in := escenarioExterna(t)
	autorizador := &autorizadorExternaPrueba{autorizacion: in.Autorizacion}
	if _, err := servicio.RegistrarExternoAutorizado(context.Background(), in, autorizador); !errors.Is(err, ports.ErrSolicitudInvalida) {
		t.Fatalf("autorización aportada por el llamante aceptada: %v", err)
	}
	in.Autorizacion = ports.AutorizacionV3{}
	if _, err := servicio.RegistrarExternoAutorizado(context.Background(), in, nil); !errors.Is(err, ports.ErrSolicitudInvalida) {
		t.Fatalf("autorizador nulo aceptado: %v", err)
	}
	if repo.llamadas != 0 || autorizador.llamadas != 0 {
		t.Fatalf("efecto sin autorizador válido: repo=%d autorizador=%d", repo.llamadas, autorizador.llamadas)
	}
}
