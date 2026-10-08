package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/ports"
)

type montajeCapacidadesRecursoPrueba struct {
	vista   ports.VistaMontajeCapacidadesRecurso
	err     error
	veces   int
	despues func()
}

func (m *montajeCapacidadesRecursoPrueba) VistaMontada(_ context.Context, superficie, perfil string) (ports.VistaMontajeCapacidadesRecurso, error) {
	m.veces++
	if m.despues != nil {
		m.despues()
	}
	if m.vista.Superficie != superficie || m.vista.PerfilActivoRef != perfil {
		return ports.VistaMontajeCapacidadesRecurso{}, ports.ErrFuenteAutorizacionNoDisponible
	}
	return m.vista, m.err
}

func escenarioCapacidadesRecursoAplicacionPrueba(t *testing.T) (
	*entornoAutorizacionSolicitudV3Prueba,
	ports.LoteRecursosLeidosCapacidad,
	*montajeCapacidadesRecursoPrueba,
) {
	t.Helper()
	e := nuevoEntornoAutorizacionSolicitudV3Prueba(t)
	datos, err := e.solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	vinculo, err := datos.VinculoAutenticacionActor.Datos()
	if err != nil {
		t.Fatal(err)
	}
	r := ports.RecursoLeidoCapacidad{RutaID: "bolsa.expediente.detalle", Metodo: "GET",
		Accion: datos.Accion, Finalidad: datos.Finalidad, Recurso: datos.Recurso}
	lote := ports.LoteRecursosLeidosCapacidad{
		Vinculo: datos.VinculoAutenticacionActor, Resultado: e.resultado,
		Superficie: "interna", Recursos: []ports.RecursoLeidoCapacidad{r},
	}
	montaje := &montajeCapacidadesRecursoPrueba{vista: ports.VistaMontajeCapacidadesRecurso{
		Superficie: "interna", PerfilActivoRef: vinculo.PerfilActivoRef, Completa: true,
		Rutas: []ports.RutaMontadaCapacidadRecurso{{RutaID: r.RutaID, Metodo: r.Metodo,
			ModuloID: r.Recurso.ModuloID, TipoRecurso: r.Recurso.Tipo,
			Accion: r.Accion, Finalidad: r.Finalidad}},
	}}
	return e, lote, montaje
}

func TestCapacidadesRecursoUnaInstantaneaExactaParaLote(t *testing.T) {
	e, lote, montaje := escenarioCapacidadesRecursoAplicacionPrueba(t)
	noAmbito := lote.Recursos[0]
	noAmbito.Recurso = clonarRecursoCapacidad(noAmbito.Recurso)
	noAmbito.Recurso.Ambitos["unidad"] = "ajena"
	sinRuta := lote.Recursos[0]
	sinRuta.RutaID = "ruta.no.montada"
	lote.Recursos = append(lote.Recursos, noAmbito, sinRuta)
	p, err := NuevoProyectorCapacidadesRecurso(e.fuente, montaje, &relojAutorizacionServicioPrueba{ahora: e.ahora})
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := p.Proyectar(context.Background(), lote)
	if err != nil || len(resultado.Resultados) != 3 ||
		resultado.Resultados[0].Estado != CapacidadRecursoDisponible ||
		resultado.Resultados[1].Estado != CapacidadRecursoNoAutorizado ||
		resultado.Resultados[2].Estado != CapacidadRecursoSinMontaje ||
		resultado.Revisiones.Asignacion != e.instantanea.AsignacionPerfil.Version ||
		!resultado.VigenteHasta.After(e.ahora) {
		t.Fatalf("proyeccion incorrecta: %+v, %v", resultado, err)
	}
	if e.fuente.invocaciones != 1 || montaje.veces != 1 || e.concesiones.invocaciones != 0 || e.denegaciones.invocaciones != 0 {
		t.Fatalf("lote hizo consultas o registros por tarjeta: fuente=%d montaje=%d concesiones=%d denegaciones=%d",
			e.fuente.invocaciones, montaje.veces, e.concesiones.invocaciones, e.denegaciones.invocaciones)
	}
}

func TestCapacidadesRecursoCoberturaDesconocidaYFuenteCaidaSonIndisponibles(t *testing.T) {
	e, lote, montaje := escenarioCapacidadesRecursoAplicacionPrueba(t)
	p, err := NuevoProyectorCapacidadesRecurso(e.fuente, montaje, &relojAutorizacionServicioPrueba{ahora: e.ahora})
	if err != nil {
		t.Fatal(err)
	}
	montaje.vista.Completa = false
	r, err := p.Proyectar(context.Background(), lote)
	if err != nil || r.Resultados[0].Estado != CapacidadRecursoIndisponible || e.fuente.invocaciones != 0 {
		t.Fatalf("cobertura desconocida se tomo por denegacion: %+v, %v", r, err)
	}
	montaje.vista.Completa = true
	montaje.err = ports.ErrFuenteAutorizacionNoDisponible
	r, err = p.Proyectar(context.Background(), lote)
	if !errors.Is(err, ports.ErrFuenteAutorizacionNoDisponible) || len(r.Resultados) != 0 || e.fuente.invocaciones != 0 {
		t.Fatalf("fallo de montaje oculto o proyección parcial: %+v, %v", r, err)
	}
	montaje.err = nil
	e.fuente.err = ports.ErrFuenteAutorizacionNoDisponible
	r, err = p.Proyectar(context.Background(), lote)
	if !errors.Is(err, ports.ErrFuenteAutorizacionNoDisponible) || len(r.Resultados) != 0 {
		t.Fatalf("fuente caida se oculto o publicó proyección parcial: %+v, %v", r, err)
	}
	e.fuente.err = ports.ErrAsignacionPerfilNoEncontrada
	r, err = p.Proyectar(context.Background(), lote)
	if err != nil || len(r.Resultados) != 1 || r.Resultados[0].Estado != CapacidadRecursoNoAutorizado {
		t.Fatalf("asignación inexistente no se distinguió del fallo de fuente: %+v, %v", r, err)
	}
	e.fuente.err = errors.Join(ports.ErrAsignacionPerfilNoEncontrada, ports.ErrFuenteAutorizacionNoDisponible)
	r, err = p.Proyectar(context.Background(), lote)
	if !errors.Is(err, ports.ErrFuenteAutorizacionNoDisponible) || len(r.Resultados) != 0 {
		t.Fatalf("fallo real compuesto se tomó por ausencia legítima: %+v, %v", r, err)
	}
}

func TestCapacidadesRecursoCaducidadYCancelacionNoPublicanDisponible(t *testing.T) {
	e, lote, montaje := escenarioCapacidadesRecursoAplicacionPrueba(t)
	reloj := &relojAutorizacionServicioPrueba{ahora: e.ahora}
	p, err := NuevoProyectorCapacidadesRecurso(e.fuente, montaje, reloj)
	if err != nil {
		t.Fatal(err)
	}
	e.fuente.despues = func() { reloj.ahora = e.ahora.Add(2 * time.Hour) }
	r, err := p.Proyectar(context.Background(), lote)
	if err != nil || r.Resultados[0].Estado != CapacidadRecursoNoAutorizado {
		t.Fatalf("resultado caducado publicado: %+v, %v", r, err)
	}
	e.fuente.despues = nil
	reloj.ahora = e.ahora
	ctx, cancelar := context.WithCancel(context.Background())
	montaje.despues = cancelar
	if _, err := p.Proyectar(ctx, lote); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelacion ocultada: %v", err)
	}
}

func TestCapacidadesRecursoNoMezclaPerfilDeOtraSesion(t *testing.T) {
	e, lote, montaje := escenarioCapacidadesRecursoAplicacionPrueba(t)
	e.fuente.instantanea.AsignacionPerfil.PrincipalID = "persona:ajena"
	p, err := NuevoProyectorCapacidadesRecurso(e.fuente, montaje, &relojAutorizacionServicioPrueba{ahora: e.ahora})
	if err != nil {
		t.Fatal(err)
	}
	r, err := p.Proyectar(context.Background(), lote)
	if err != nil || r.Resultados[0].Estado != CapacidadRecursoNoAutorizado {
		t.Fatalf("perfil ajeno aceptado: %+v, %v", r, err)
	}
}
