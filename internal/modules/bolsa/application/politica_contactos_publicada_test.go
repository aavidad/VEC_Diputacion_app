package application

import (
	"context"
	"errors"
	"strings"
	"testing"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

type fuentePoliticaContactosFalsa struct {
	f         puertosbolsa.FuentePoliticaContactos
	err       error
	consultas int
}

func (f *fuentePoliticaContactosFalsa) ObtenerPublicada(context.Context, string) (puertosbolsa.FuentePoliticaContactos, error) {
	f.consultas++
	return f.f, f.err
}

type repoPoliticaContactosFalso struct {
	actual        dominiobolsa.PoliticaContactosPublicada
	existe        bool
	publicaciones int
	recibo        puertosbolsa.ReciboPoliticaContactos
}

func (r *repoPoliticaContactosFalso) VersionActual(context.Context, string) (dominiobolsa.PoliticaContactosPublicada, bool, error) {
	return r.actual, r.existe, nil
}

func (r *repoPoliticaContactosFalso) PublicarSiVersion(_ context.Context, orden puertosbolsa.OrdenPublicarPoliticaContactos) (puertosbolsa.ReciboPoliticaContactos, error) {
	r.publicaciones++
	if r.existe {
		reutilizado := r.recibo
		reutilizado.Reutilizado = true
		return reutilizado, nil
	}
	r.actual, r.existe = orden.Politica, true
	r.recibo = puertosbolsa.ReciboPoliticaContactos{
		ReciboRef: orden.ReciboRef, BolsaRef: orden.Politica.BolsaRef,
		Version: orden.Politica.Version, HuellaSHA256: orden.Politica.HuellaSHA256,
	}
	return r.recibo, nil
}

func TestPrepararFuentePoliticaContactosRechazaCambioSinHuella(t *testing.T) {
	f := puertosbolsa.FuentePoliticaContactos{
		BolsaRef: "bolsa:administrativo:2026", CatalogoRef: "bolsa-reglas:2:b04.franja_llamadas",
		CatalogoHuellaSHA256: strings.Repeat("a", 64), TipoDia: dominiobolsa.TipoCalendarioHabilSede,
		SedeRef: "municipio:18087", Zona: "Europe/Madrid", DesdeMinuto: 540, HastaMinuto: 840,
		ControlFranja: dominiobolsa.ControlReglaImpedir, IntentosPorCiclo: 2, Ciclos: 2,
		SeparacionSegundos: 7200, ControlSeparacion: dominiobolsa.ControlReglaImpedir,
		ResultadosSinContacto: []string{dominiobolsa.ResultadoContactoNoContesta},
	}
	p := dominiobolsa.PoliticaContactosPublicada{
		Esquema: dominiobolsa.EsquemaPoliticaContactos, BolsaRef: f.BolsaRef, Version: 1,
		CatalogoRef: f.CatalogoRef, CatalogoHuellaSHA256: f.CatalogoHuellaSHA256,
		TipoDia: f.TipoDia, SedeRef: f.SedeRef, Zona: f.Zona,
		DesdeMinuto: f.DesdeMinuto, HastaMinuto: f.HastaMinuto,
		ControlFranja: f.ControlFranja, IntentosPorCiclo: f.IntentosPorCiclo,
		Ciclos: f.Ciclos, SeparacionSegundos: f.SeparacionSegundos,
		ControlSeparacion: f.ControlSeparacion, ResultadosSinContacto: f.ResultadosSinContacto,
	}
	var err error
	f.HuellaFuenteSHA256, err = p.HuellaCanonica()
	if err != nil {
		t.Fatal(err)
	}
	preparada, err := prepararFuentePoliticaContactos(f, f.BolsaRef)
	if err != nil || preparada.Validar() != nil {
		t.Fatalf("fuente integra = %+v, %v", preparada, err)
	}
	f.HastaMinuto = 900
	if _, err := prepararFuentePoliticaContactos(f, f.BolsaRef); !errors.Is(err, ErrPoliticaContactosNoDisponible) {
		t.Fatalf("politica alterada = %v", err)
	}
	f.HastaMinuto = 840
	if _, err := prepararFuentePoliticaContactos(f, "bolsa:ajena"); !errors.Is(err, ErrPoliticaContactosNoDisponible) {
		t.Fatalf("bolsa ajena = %v", err)
	}
}

func TestPublicarPoliticaConservaReplayYDeniegaAudienciaAjena(t *testing.T) {
	f := puertosbolsa.FuentePoliticaContactos{
		BolsaRef: "bolsa:administrativo:2026", CatalogoRef: "bolsa-reglas:2:b04.franja_llamadas",
		CatalogoHuellaSHA256: strings.Repeat("a", 64), TipoDia: dominiobolsa.TipoCalendarioHabilSede,
		SedeRef: "municipio:18087", Zona: "Europe/Madrid", DesdeMinuto: 540, HastaMinuto: 840,
		ControlFranja: dominiobolsa.ControlReglaImpedir, IntentosPorCiclo: 2, Ciclos: 2,
		SeparacionSegundos: 7200, ControlSeparacion: dominiobolsa.ControlReglaImpedir,
		ResultadosSinContacto: []string{dominiobolsa.ResultadoContactoNoContesta},
	}
	p := dominiobolsa.PoliticaContactosPublicada{
		Esquema: dominiobolsa.EsquemaPoliticaContactos, BolsaRef: f.BolsaRef, Version: 1,
		CatalogoRef: f.CatalogoRef, CatalogoHuellaSHA256: f.CatalogoHuellaSHA256,
		TipoDia: f.TipoDia, SedeRef: f.SedeRef, Zona: f.Zona,
		DesdeMinuto: f.DesdeMinuto, HastaMinuto: f.HastaMinuto,
		ControlFranja: f.ControlFranja, IntentosPorCiclo: f.IntentosPorCiclo,
		Ciclos: f.Ciclos, SeparacionSegundos: f.SeparacionSegundos,
		ControlSeparacion: f.ControlSeparacion, ResultadosSinContacto: f.ResultadosSinContacto,
	}
	var err error
	f.HuellaFuenteSHA256, err = p.HuellaCanonica()
	if err != nil {
		t.Fatal(err)
	}
	fuente := &fuentePoliticaContactosFalsa{f: f}
	repo := &repoPoliticaContactosFalso{}
	servicio, err := NuevoServicioPoliticaContactos(fuente, repo)
	if err != nil {
		t.Fatal(err)
	}
	solicitud := SolicitudPublicarPoliticaContactos{
		BolsaRef: f.BolsaRef, ActorRef: "per_0123456789abcdefghijkl",
		ClaveIdempotencia: "politica-contactos-001",
		ReciboRef:         "recibo:politica-contactos:" + strings.Repeat("b", 64),
		Material: materialEstructuralCalendarioPrueba(t,
			puertosbolsa.AccionPublicarPoliticaContactos,
			puertosbolsa.AudienciaPublicarPoliticaContactos, f.BolsaRef),
	}
	primero, err := servicio.Publicar(t.Context(), solicitud)
	if err != nil || primero.Reutilizado || repo.publicaciones != 1 {
		t.Fatalf("publicacion = %+v, %v", primero, err)
	}
	segundo, err := servicio.Publicar(t.Context(), solicitud)
	if err != nil || !segundo.Reutilizado || segundo.ReciboRef != primero.ReciboRef ||
		segundo.Version != primero.Version || segundo.HuellaSHA256 != primero.HuellaSHA256 {
		t.Fatalf("replay = %+v, %v", segundo, err)
	}
	solicitud.Material = materialEstructuralCalendarioPrueba(t,
		puertosbolsa.AccionPublicarPoliticaContactos,
		puertosbolsa.AudienciaEntregarCalendarioContactos, f.BolsaRef)
	antes := fuente.consultas
	if _, err := servicio.Publicar(t.Context(), solicitud); !errors.Is(err, ErrPoliticaContactosNoDisponible) ||
		fuente.consultas != antes || repo.publicaciones != 2 {
		t.Fatalf("audiencia ajena alcanzo fuente o repositorio: %v", err)
	}
}
