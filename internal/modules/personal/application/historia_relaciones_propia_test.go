package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type autorizadorHistoriaRelacionesPrueba struct {
	t        *testing.T
	prestada bool
	llamadas int
}

func (a *autorizadorHistoriaRelacionesPrueba) AutorizarHistoriaRelacionesPropia(_ context.Context, m domain.MaterialHistoriaRelacionesPropia) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	a.llamadas++
	base, _ := domain.NuevoMaterialFichaPropia(domain.SolicitudFichaPropia{Actor: m.Actor(), Corte: domain.CorteEmpleadoB2{VigenteEn: m.Corte().Desde, ConocidoEn: m.Corte().ConocidoEn}})
	b := atestacionFichaPropiaPrueba(a.t, base, domain.AccionFichaPropia)
	if a.prestada {
		return b, nil
	}
	// Doble estructural: no acredita firma, consumo, SQL ni COMMIT real.
	x := b.ResumenCapacidad()
	h, _ := m.HuellaSHA256()
	r, e := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3(x.DecisionRef(), strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_prueba", strings.Repeat("c", 64), domain.AccionHistoriaRelacionesPropia, m.EmpleadoRef(), h, domain.AudienciaHistoriaRelacionesPropia, x.EmitidaEn(), x.ExpiraEn())
	if e != nil {
		a.t.Fatal(e)
	}
	return vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(b.CapacidadCanonica(), r, b.DecisionCanonica(), b.MotivoCanonico(), b.ContextoActorCanonico(), b.PersonaVersion(), b.PerfilVersion(), b.PayloadVECAD3(), b.SobreCOSESign1(), b.EvidenciaVerificacion(), b.RaizPublicaSPKI())
}

type repositorioHistoriaRelacionesPrueba struct {
	llamadas int
	cerrado  bool
	err      error
	alterar  string
}

func (r *repositorioHistoriaRelacionesPrueba) ConsultarHistoriaRelacionesPropia(_ context.Context, o ports.OrdenHistoriaRelacionesPropia) (ports.ResultadoHistoriaRelacionesPropia, error) {
	r.llamadas++
	r.cerrado = true
	if r.err != nil {
		return ports.ResultadoHistoriaRelacionesPropia{}, r.err
	}
	x := o.Autorizacion.ResumenCapacidad()
	h := domain.HistoriaRelacionesPropia{EmpleadoRef: o.Material.EmpleadoRef(), Corte: o.Material.Corte(), Cobertura: "parcial", Revisiones: []domain.RevisionRelacionPropia{}}
	e := ports.EvidenciaRegistroEmpleadoB2{ReciboRef: "aud_v3_" + strings.Repeat("d", 32), DecisionRef: x.DecisionRef(), EfectoRef: x.EfectoRef(), ConsumoHuellaSHA256: strings.Repeat("d", 64), AuditoriaRef: "aud_v3_" + strings.Repeat("d", 32), ConsultadaEn: x.EmitidaEn().Add(time.Microsecond)}
	if r.alterar == "recibo" {
		e.ReciboRef = "fichapropia:0f0e0d0c-0b0a-4908-8706-050403020100"
	}
	if r.alterar == "decision" {
		e.DecisionRef = "dec_ajena"
	}
	if r.alterar == "corte" {
		h.Corte.Hasta = "2028-01-01"
	}
	return ports.ResultadoHistoriaRelacionesPropia{Historia: h, Evidencia: e}, nil
}

type intentosHistoriaRelacionesPrueba struct {
	repo        *repositorioHistoriaRelacionesPrueba
	intents     []ports.IntentoHistoriaRelacionesPropia
	err         error
	antesCerrar bool
	cancelada   bool
}

func (i *intentosHistoriaRelacionesPrueba) VerificarRegistroHistoriaRelacionesPropia(context.Context) error {
	return nil
}
func (i *intentosHistoriaRelacionesPrueba) RegistrarIntentoHistoriaRelacionesPropia(ctx context.Context, in ports.IntentoHistoriaRelacionesPropia) error {
	i.intents = append(i.intents, in)
	i.antesCerrar = i.repo.llamadas > 0 && !i.repo.cerrado
	i.cancelada = ctx.Err() != nil
	return i.err
}
func solicitudHistoriaRelacionesPrueba(t *testing.T) domain.SolicitudHistoriaRelacionesPropia {
	b := solicitudFichaPropiaPrueba(t, "pep_")
	return domain.SolicitudHistoriaRelacionesPropia{Actor: b.Actor, Corte: domain.CorteHistoriaRelacionesPropia{Desde: "2020-01-01", Hasta: "2027-01-01", ConocidoEn: b.Corte.ConocidoEn}}
}
func TestHistoriaRelacionesPropiaRequierePermisoPropioYEvidenciaLigada(t *testing.T) {
	for _, caso := range []string{"permitida", "prestada", "decision", "corte", "recibo"} {
		t.Run(caso, func(t *testing.T) {
			a := &autorizadorHistoriaRelacionesPrueba{t: t, prestada: caso == "prestada"}
			r := &repositorioHistoriaRelacionesPrueba{alterar: caso}
			i := &intentosHistoriaRelacionesPrueba{repo: r}
			s, _ := NuevoServicioHistoriaRelacionesPropia(a, r, i)
			out, e := s.Consultar(context.Background(), solicitudHistoriaRelacionesPrueba(t))
			if caso == "permitida" {
				if e != nil || out.Historia.Cobertura != "parcial" || len(i.intents) != 0 {
					t.Fatal(e)
				}
				return
			}
			if !errors.Is(e, domain.ErrHistoriaRelacionesPropiaNoDisponible) || out.Historia.Revisiones != nil || len(i.intents) != 1 || i.antesCerrar {
				t.Fatal("fallo no cerrado", e)
			}
			if caso == "prestada" && r.llamadas != 0 {
				t.Fatal("lectura con permiso prestado")
			}
		})
	}
}
func TestHistoriaRelacionesPropiaFalloTrasCerrarYAcuseObligatorio(t *testing.T) {
	for _, falloAcuse := range []bool{false, true} {
		a := &autorizadorHistoriaRelacionesPrueba{t: t}
		r := &repositorioHistoriaRelacionesPrueba{err: domain.ErrHistoriaRelacionesPropiaDenegada}
		i := &intentosHistoriaRelacionesPrueba{repo: r}
		if falloAcuse {
			i.err = errors.New("privado")
		}
		s, _ := NuevoServicioHistoriaRelacionesPropia(a, r, i)
		out, e := s.Consultar(context.Background(), solicitudHistoriaRelacionesPrueba(t))
		esperado := domain.ErrHistoriaRelacionesPropiaDenegada
		if falloAcuse {
			esperado = domain.ErrHistoriaRelacionesPropiaNoDisponible
		}
		if !errors.Is(e, esperado) || out.Historia.Revisiones != nil || i.antesCerrar || len(i.intents) != 1 || i.intents[0].Motivo != "denegado" {
			t.Fatal(e)
		}
	}
	a := &autorizadorHistoriaRelacionesPrueba{t: t}
	r := &repositorioHistoriaRelacionesPrueba{err: domain.ErrHistoriaRelacionesPropiaExcedeLimite}
	i := &intentosHistoriaRelacionesPrueba{repo: r}
	s, _ := NuevoServicioHistoriaRelacionesPropia(a, r, i)
	if _, e := s.Consultar(context.Background(), solicitudHistoriaRelacionesPrueba(t)); !errors.Is(e, domain.ErrHistoriaRelacionesPropiaExcedeLimite) {
		t.Fatal("límite indistinguible", e)
	}
	if _, e := NuevoServicioHistoriaRelacionesPropia(a, r, nil); !errors.Is(e, domain.ErrHistoriaRelacionesPropiaNoDisponible) {
		t.Fatal("intentos opcionales")
	}
}
func TestHistoriaRelacionesPropiaCanceladaAuditaSinFuenteNiDatos(t *testing.T) {
	a := &autorizadorHistoriaRelacionesPrueba{t: t}
	r := &repositorioHistoriaRelacionesPrueba{}
	i := &intentosHistoriaRelacionesPrueba{repo: r}
	s, _ := NuevoServicioHistoriaRelacionesPropia(a, r, i)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := s.Consultar(ctx, solicitudHistoriaRelacionesPrueba(t)); !errors.Is(e, context.Canceled) || a.llamadas != 0 || r.llamadas != 0 || len(i.intents) != 1 || i.cancelada {
		t.Fatal("cancelación sin intento", e)
	}
}
