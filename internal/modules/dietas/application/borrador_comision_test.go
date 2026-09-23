package application

import (
	"context"
	"strings"
	"testing"
	"time"

	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
)

type repositorioBorradorPrueba struct{ llamadas int }

func (r *repositorioBorradorPrueba) CrearORecuperar(context.Context, dietasports.IdentidadEfectivaBorrador, dietasports.SolicitudCrearBorradorPropio) (dietasports.ResultadoBorradorComision, error) {
	r.llamadas++
	return dietasports.ResultadoBorradorComision{}, nil
}

func TestValidarSolicitudOperacionRechazaVariantesNoCanonicas(t *testing.T) {
	base := dietasports.SolicitudCrearBorradorPropio{ClaveIdempotencia: "clave_0123456789abcdef", FechaInicio: "2026-09-21", FechaFin: "2026-09-21", Motivo: "Visita técnica", CodigosRuta: []string{"ruta:a", "ruta:b"}}
	crear, err := NuevaSolicitudOperacionCrearBorrador(base)
	if err != nil {
		t.Fatal(err)
	}
	casos := []dietasports.SolicitudOperacionBorrador{
		{Operacion: dietasports.OperacionCrearBorrador, Crear: func() dietasports.SolicitudCrearBorradorPropio {
			x := base
			x.CodigosRuta = []string{"ruta:b", "ruta:a"}
			return x
		}()},
		{Operacion: dietasports.OperacionCrearBorrador, Crear: base, Consulta: dietasports.ConsultaBorradoresPropios{Limite: 1}},
		{Operacion: dietasports.OperacionConsultarBorrador, Referencia: "dco_" + strings.Repeat("a", 22), Consulta: dietasports.ConsultaBorradoresPropios{Limite: 1}},
		{Operacion: dietasports.OperacionConsultarBorrador, Consulta: dietasports.ConsultaBorradoresPropios{}},
		{Operacion: dietasports.OperacionConsultarBorrador, Consulta: dietasports.ConsultaBorradoresPropios{Limite: 1, Cursor: "malo"}},
		{Operacion: dietasports.OperacionConsultarBorrador, Crear: base, Consulta: dietasports.ConsultaBorradoresPropios{Limite: 1}},
	}
	if validarSolicitudOperacion(crear) != nil {
		t.Fatal("canon válido rechazado")
	}
	for i, c := range casos {
		if validarSolicitudOperacion(c) == nil {
			t.Fatalf("variante %d aceptada", i)
		}
	}
}
func (r *repositorioBorradorPrueba) ObtenerPropio(context.Context, dietasports.IdentidadEfectivaBorrador, string) (dietasports.ResultadoBorradorComision, error) {
	r.llamadas++
	return dietasports.ResultadoBorradorComision{}, nil
}
func (r *repositorioBorradorPrueba) ListarPropios(context.Context, dietasports.IdentidadEfectivaBorrador, dietasports.ConsultaBorradoresPropios) (dietasports.PaginaBorradoresPropios, error) {
	r.llamadas++
	return dietasports.PaginaBorradoresPropios{}, nil
}

func TestCrearBorradorRechazaContextoV2NoAcreditadoAntesDeRepositorio(t *testing.T) {
	repo := &repositorioBorradorPrueba{}
	servicio, err := NuevoServicioBorradorComision(repo)
	if err != nil {
		t.Fatal(err)
	}
	_, err = servicio.CrearPropio(context.Background(), dietasports.IdentidadEfectivaBorrador{}, dietasports.SolicitudCrearBorradorPropio{ClaveIdempotencia: "clave_idempotente_0001", FechaInicio: "2026-09-21", FechaFin: "2026-09-21", Motivo: "Visita"})
	if err != dietasports.ErrAccesoBorradorDenegado || repo.llamadas != 0 {
		t.Fatalf("err=%v llamadas=%d", err, repo.llamadas)
	}
}

func TestBorradorNoLlamaRepositorioConContextoTerminado(t *testing.T) {
	cancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	vencido, cancelarVencido := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancelarVencido()
	referencia := "dco_" + strings.Repeat("a", 22)
	for _, caso := range []struct {
		nombre string
		ctx    context.Context
		err    error
		llamar func(*ServicioBorradorComision, context.Context) error
	}{
		{"crear cancelado", cancelado, context.Canceled, func(s *ServicioBorradorComision, ctx context.Context) error {
			_, err := s.CrearPropio(ctx, dietasports.IdentidadEfectivaBorrador{}, dietasports.SolicitudCrearBorradorPropio{})
			return err
		}},
		{"crear vencido", vencido, context.DeadlineExceeded, func(s *ServicioBorradorComision, ctx context.Context) error {
			_, err := s.CrearPropio(ctx, dietasports.IdentidadEfectivaBorrador{}, dietasports.SolicitudCrearBorradorPropio{})
			return err
		}},
		{"detalle cancelado", cancelado, context.Canceled, func(s *ServicioBorradorComision, ctx context.Context) error {
			_, err := s.ObtenerPropio(ctx, dietasports.IdentidadEfectivaBorrador{}, referencia)
			return err
		}},
		{"detalle vencido", vencido, context.DeadlineExceeded, func(s *ServicioBorradorComision, ctx context.Context) error {
			_, err := s.ObtenerPropio(ctx, dietasports.IdentidadEfectivaBorrador{}, referencia)
			return err
		}},
		{"lista cancelada", cancelado, context.Canceled, func(s *ServicioBorradorComision, ctx context.Context) error {
			_, err := s.ListarPropios(ctx, dietasports.IdentidadEfectivaBorrador{}, dietasports.ConsultaBorradoresPropios{})
			return err
		}},
		{"lista vencida", vencido, context.DeadlineExceeded, func(s *ServicioBorradorComision, ctx context.Context) error {
			_, err := s.ListarPropios(ctx, dietasports.IdentidadEfectivaBorrador{}, dietasports.ConsultaBorradoresPropios{})
			return err
		}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			repo := &repositorioBorradorPrueba{}
			servicio, err := NuevoServicioBorradorComision(repo)
			if err != nil {
				t.Fatal(err)
			}
			if err := caso.llamar(servicio, caso.ctx); err != caso.err || repo.llamadas != 0 {
				t.Fatalf("err=%v llamadas=%d", err, repo.llamadas)
			}
		})
	}
}

func TestRelacionAcreditadaExigeIntervaloCivilCanonicoYSelloFuenteNumerico(t *testing.T) {
	base := dietasports.RelacionServicioAcreditada{RelacionRef: "rel_0123456789abcdefghijkl", PersonaRef: "per_0123456789abcdefghijkl", EmpleadoRef: "emp_0123456789abcdefghijkl", UnidadRef: "unidad:prueba", VigenteDesde: "2026-09-20", Version: 1, ProcedenciaActoRef: "acto:prueba", FuenteRef: "fuente:prueba", FuenteVersion: 1}
	if !relacionAcreditada(base) {
		t.Fatal("relacion valida rechazada")
	}
	for nombre, mutar := range map[string]func(*dietasports.RelacionServicioAcreditada){
		"inicio no canonico": func(r *dietasports.RelacionServicioAcreditada) { r.VigenteDesde = "2026-9-20" },
		"fin no canonico":    func(r *dietasports.RelacionServicioAcreditada) { r.VigenteHasta = "2026-02-30" },
		"intervalo vacio":    func(r *dietasports.RelacionServicioAcreditada) { r.VigenteHasta = r.VigenteDesde },
		"fuente cero":        func(r *dietasports.RelacionServicioAcreditada) { r.FuenteVersion = 0 },
		"version negativa":   func(r *dietasports.RelacionServicioAcreditada) { r.Version = -1 },
		"fuente negativa":    func(r *dietasports.RelacionServicioAcreditada) { r.FuenteVersion = -1 },
	} {
		t.Run(nombre, func(t *testing.T) {
			r := base
			mutar(&r)
			if relacionAcreditada(r) {
				t.Fatal("relacion invalida aceptada")
			}
		})
	}
}
