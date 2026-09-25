package application

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func solicitudFichaPropiaPrueba(t *testing.T, prefijo string) domain.SolicitudFichaPropia {
	t.Helper()
	z := strings.Repeat("a", 24)
	ahora := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	cuenta := vecdomain.CuentaAutenticadaContextoActor{CuentaRef: "cta_" + z, Metodo: vecdomain.AuthMethodCertificate, Garantia: vecdomain.AuthAssuranceHigh}
	instantanea := vecdomain.InstantaneaContextoActor{VinculoRef: "vca_" + z, VinculoVersion: 1, CuentaRef: cuenta.CuentaRef, CuentaVersion: 1, PersonaRef: "per_" + z, PersonaVersion: 1, PerfilActivoRef: "prf_" + z, PerfilVersion: 1, Estado: vecdomain.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour),
		Vinculos: []vecdomain.VinculoReferenciaContextoActor{{VinculoRef: prefijo + z, Version: 1, Tipo: vecdomain.TipoReferenciaContextoActorEmpleado, Referencia: "emp_" + z, Estado: vecdomain.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour)}}}
	actor, err := vecdomain.NuevoContextoActor(cuenta, instantanea, ahora)
	if err != nil {
		t.Fatal(err)
	}
	return domain.SolicitudFichaPropia{Corte: domain.CorteEmpleadoB2{VigenteEn: "2026-09-25", ConocidoEn: ahora.Add(-time.Second)}, Actor: actor}
}

func atestacionFichaPropiaPrueba(t *testing.T, m domain.MaterialFichaPropia, accion string) vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	t.Helper()
	ahora := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	h, _ := m.HuellaSHA256()
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("dec_prueba", strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_prueba", strings.Repeat("c", 64), accion, m.EmpleadoRef(), h, domain.AudienciaFichaPropia, ahora, ahora.Add(3*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	raiz, _ := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	actor := m.Actor()
	canon, _ := actor.RepresentacionCanonicaVinculadaV2()
	a, err := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), resumen, []byte("d"), []byte("m"), canon, 1, 1, []byte("p"), []byte("s"), []byte("e"), raiz)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

type autorizadorFichaPropiaPrueba struct {
	accion   string
	err      error
	t        *testing.T
	llamadas int
}

func (a *autorizadorFichaPropiaPrueba) AutorizarFichaPropia(_ context.Context, m domain.MaterialFichaPropia) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	a.llamadas++
	if a.err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, a.err
	}
	return atestacionFichaPropiaPrueba(a.t, m, a.accion), nil
}

type repositorioFichaPropiaPrueba struct {
	decision string
	err      error
	llamadas int
}

func (r *repositorioFichaPropiaPrueba) ConsultarFichaPropia(_ context.Context, o ports.OrdenFichaPropia) (ports.ResultadoFichaPropia, error) {
	r.llamadas++
	if r.err != nil {
		return ports.ResultadoFichaPropia{}, r.err
	}
	x := o.Autorizacion.ResumenCapacidad()
	decision := x.DecisionRef()
	if r.decision != "" {
		decision = r.decision
	}
	return ports.ResultadoFichaPropia{
		Ficha:     domain.FichaPropia{Corte: o.Material.Corte(), Relaciones: []domain.RelacionFichaPropia{{Inicio: "2026-01-01", Estado: "vigente", Regimen: "Laboral"}}, Servicios: []domain.ServicioFichaPropia{}},
		Evidencia: ports.EvidenciaRegistroEmpleadoB2{ReciboRef: "fichapropia:x", DecisionRef: decision, EfectoRef: x.EfectoRef(), ConsumoHuellaSHA256: strings.Repeat("d", 64), AuditoriaRef: "auditoria:x", ConsultadaEn: x.EmitidaEn().Add(time.Microsecond)},
	}, nil
}

func TestServicioFichaPropiaConsultaConConcesionNominal(t *testing.T) {
	a := &autorizadorFichaPropiaPrueba{accion: domain.AccionFichaPropia, t: t}
	r := &repositorioFichaPropiaPrueba{}
	s, err := NuevoServicioFichaPropia(a, r)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := s.Consultar(context.Background(), solicitudFichaPropiaPrueba(t, "pep_"))
	if err != nil || len(resultado.Ficha.Relaciones) != 1 || a.llamadas != 1 || r.llamadas != 1 {
		t.Fatalf("consulta propia no servida: %v", err)
	}
}

func TestServicioFichaPropiaDeniegaSinLlegarAlRepositorio(t *testing.T) {
	casos := map[string]struct {
		prefijo string
		a       *autorizadorFichaPropiaPrueba
		err     error
	}{
		"sin_proyeccion": {"vin_", &autorizadorFichaPropiaPrueba{accion: domain.AccionFichaPropia}, domain.ErrFichaPropiaSinEmpleado},
		"pdp_deniega":    {"pep_", &autorizadorFichaPropiaPrueba{err: domain.ErrFichaPropiaDenegada}, domain.ErrFichaPropiaDenegada},
		"pdp_caido":      {"pep_", &autorizadorFichaPropiaPrueba{err: errors.New("detalle interno")}, domain.ErrFichaPropiaNoDisponible},
		"otra_operacion": {"pep_", &autorizadorFichaPropiaPrueba{accion: domain.AccionFichaEmpleadoB2}, domain.ErrFichaPropiaNoDisponible},
	}
	for nombre, caso := range casos {
		t.Run(nombre, func(t *testing.T) {
			caso.a.t = t
			r := &repositorioFichaPropiaPrueba{}
			s, _ := NuevoServicioFichaPropia(caso.a, r)
			_, err := s.Consultar(context.Background(), solicitudFichaPropiaPrueba(t, caso.prefijo))
			if !errors.Is(err, caso.err) || r.llamadas != 0 || strings.Contains(err.Error(), "interno") {
				t.Fatalf("error %v, se esperaba %v", err, caso.err)
			}
		})
	}
}

func TestServicioFichaPropiaRechazaEvidenciaAjena(t *testing.T) {
	a := &autorizadorFichaPropiaPrueba{accion: domain.AccionFichaPropia, t: t}
	s, _ := NuevoServicioFichaPropia(a, &repositorioFichaPropiaPrueba{decision: "dec_otra"})
	if _, err := s.Consultar(context.Background(), solicitudFichaPropiaPrueba(t, "pep_")); !errors.Is(err, domain.ErrFichaPropiaNoDisponible) {
		t.Fatal("evidencia de otra decisión aceptada", err)
	}
	if _, err := NuevoServicioFichaPropia(nil, &repositorioFichaPropiaPrueba{}); err == nil {
		t.Fatal("servicio sin autorizador compuesto")
	}
}
