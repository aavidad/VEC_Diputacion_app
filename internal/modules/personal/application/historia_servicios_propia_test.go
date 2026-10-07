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

type autorizadorHistoriaPrueba struct {
	t        *testing.T
	prestada bool
	llamadas int
}

func (a *autorizadorHistoriaPrueba) AutorizarHistoriaServiciosPropia(_ context.Context, m domain.MaterialHistoriaServiciosPropia) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	a.llamadas++
	base, _ := domain.NuevoMaterialFichaPropia(domain.SolicitudFichaPropia{Actor: m.Actor(), Corte: domain.CorteEmpleadoB2{VigenteEn: m.Corte().Desde, ConocidoEn: m.Corte().ConocidoEn}})
	b := atestacionFichaPropiaPrueba(a.t, base, domain.AccionFichaPropia)
	if a.prestada {
		return b, nil
	}
	// Doble estructural: no acredita firma, consumo, SQL ni COMMIT real.
	x := b.ResumenCapacidad()
	h, _ := m.HuellaSHA256()
	r, e := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3(x.DecisionRef(), strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_prueba", strings.Repeat("c", 64), domain.AccionHistoriaServiciosPropia, m.EmpleadoRef(), h, domain.AudienciaHistoriaServiciosPropia, x.EmitidaEn(), x.ExpiraEn())
	if e != nil {
		a.t.Fatal(e)
	}
	return vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(b.CapacidadCanonica(), r, b.DecisionCanonica(), b.MotivoCanonico(), b.ContextoActorCanonico(), b.PersonaVersion(), b.PerfilVersion(), b.PayloadVECAD3(), b.SobreCOSESign1(), b.EvidenciaVerificacion(), b.RaizPublicaSPKI())
}

type repositorioHistoriaPrueba struct {
	llamadas int
	cerrado  bool
	err      error
	alterar  string
}

func (r *repositorioHistoriaPrueba) ConsultarHistoriaServiciosPropia(_ context.Context, o ports.OrdenHistoriaServiciosPropia) (ports.ResultadoHistoriaServiciosPropia, error) {
	r.llamadas++
	r.cerrado = true
	if r.err != nil {
		return ports.ResultadoHistoriaServiciosPropia{}, r.err
	}
	x := o.Autorizacion.ResumenCapacidad()
	h := domain.HistoriaServiciosPropia{EmpleadoRef: o.Material.EmpleadoRef(), Corte: o.Material.Corte(), Cobertura: "parcial", Revisiones: []domain.RevisionServicioPropio{}}
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
	return ports.ResultadoHistoriaServiciosPropia{Historia: h, Evidencia: e}, nil
}

type intentosHistoriaPrueba struct {
	repo        *repositorioHistoriaPrueba
	intents     []ports.IntentoHistoriaServiciosPropia
	err         error
	antesCerrar bool
	cancelada   bool
}

func (i *intentosHistoriaPrueba) VerificarRegistroHistoriaServiciosPropia(context.Context) error {
	return nil
}
func (i *intentosHistoriaPrueba) RegistrarIntentoHistoriaServiciosPropia(ctx context.Context, in ports.IntentoHistoriaServiciosPropia) error {
	i.intents = append(i.intents, in)
	i.antesCerrar = i.repo.llamadas > 0 && !i.repo.cerrado
	i.cancelada = ctx.Err() != nil
	return i.err
}
func solicitudHistoriaPrueba(t *testing.T) domain.SolicitudHistoriaServiciosPropia {
	b := solicitudFichaPropiaPrueba(t, "pep_")
	return domain.SolicitudHistoriaServiciosPropia{Actor: b.Actor, Corte: domain.CorteHistoriaServiciosPropia{Desde: "2020-01-01", Hasta: "2027-01-01", ConocidoEn: b.Corte.ConocidoEn}}
}
func TestHistoriaServiciosPropiaRequierePermisoPropioYEvidenciaLigada(t *testing.T) {
	for _, caso := range []string{"permitida", "prestada", "decision", "corte", "recibo"} {
		t.Run(caso, func(t *testing.T) {
			a := &autorizadorHistoriaPrueba{t: t, prestada: caso == "prestada"}
			r := &repositorioHistoriaPrueba{alterar: caso}
			i := &intentosHistoriaPrueba{repo: r}
			s, _ := NuevoServicioHistoriaServiciosPropia(a, r, i)
			out, e := s.Consultar(context.Background(), solicitudHistoriaPrueba(t))
			if caso == "permitida" {
				if e != nil || out.Historia.Cobertura != "parcial" || len(i.intents) != 0 {
					t.Fatal(e)
				}
				return
			}
			if !errors.Is(e, domain.ErrHistoriaServiciosPropiaNoDisponible) || out.Historia.Revisiones != nil || len(i.intents) != 1 || i.antesCerrar {
				t.Fatal("fallo no cerrado", e)
			}
			if caso == "prestada" && r.llamadas != 0 {
				t.Fatal("lectura con permiso prestado")
			}
		})
	}
}
func TestHistoriaServiciosPropiaFalloTrasCerrarYAcuseObligatorio(t *testing.T) {
	for _, falloAcuse := range []bool{false, true} {
		a := &autorizadorHistoriaPrueba{t: t}
		r := &repositorioHistoriaPrueba{err: domain.ErrHistoriaServiciosPropiaDenegada}
		i := &intentosHistoriaPrueba{repo: r}
		if falloAcuse {
			i.err = errors.New("privado")
		}
		s, _ := NuevoServicioHistoriaServiciosPropia(a, r, i)
		out, e := s.Consultar(context.Background(), solicitudHistoriaPrueba(t))
		esperado := domain.ErrHistoriaServiciosPropiaDenegada
		if falloAcuse {
			esperado = domain.ErrHistoriaServiciosPropiaNoDisponible
		}
		if !errors.Is(e, esperado) || out.Historia.Revisiones != nil || i.antesCerrar || len(i.intents) != 1 || i.intents[0].Motivo != "denegado" {
			t.Fatal(e)
		}
	}
	a := &autorizadorHistoriaPrueba{t: t}
	r := &repositorioHistoriaPrueba{err: domain.ErrHistoriaServiciosPropiaExcedeLimite}
	i := &intentosHistoriaPrueba{repo: r}
	s, _ := NuevoServicioHistoriaServiciosPropia(a, r, i)
	if _, e := s.Consultar(context.Background(), solicitudHistoriaPrueba(t)); !errors.Is(e, domain.ErrHistoriaServiciosPropiaExcedeLimite) {
		t.Fatal("límite indistinguible", e)
	}
	if _, e := NuevoServicioHistoriaServiciosPropia(a, r, nil); !errors.Is(e, domain.ErrHistoriaServiciosPropiaNoDisponible) {
		t.Fatal("intentos opcionales")
	}
}
func TestHistoriaServiciosPropiaCanceladaAuditaSinFuenteNiDatos(t *testing.T) {
	a := &autorizadorHistoriaPrueba{t: t}
	r := &repositorioHistoriaPrueba{}
	i := &intentosHistoriaPrueba{repo: r}
	s, _ := NuevoServicioHistoriaServiciosPropia(a, r, i)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := s.Consultar(ctx, solicitudHistoriaPrueba(t)); !errors.Is(e, context.Canceled) || a.llamadas != 0 || r.llamadas != 0 || len(i.intents) != 1 || i.cancelada {
		t.Fatal("cancelación sin intento", e)
	}
}
