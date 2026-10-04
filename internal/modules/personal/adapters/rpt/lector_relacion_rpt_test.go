package rpt

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type lectorSeleccionadaRPTPrueba struct {
	consultas []ports.ConsultaRelacionParaRPTV1
	resultado ports.ResultadoRelacionParaRPTV1
	err       error
	mutar     func(*ports.ConsultaRelacionParaRPTV1)
}

func (l *lectorSeleccionadaRPTPrueba) ConsultarRelacionParaRPT(_ context.Context, q ports.ConsultaRelacionParaRPTV1) (ports.ResultadoRelacionParaRPTV1, error) {
	l.consultas = append(l.consultas, q)
	if l.mutar != nil {
		l.mutar(&q)
	}
	return l.resultado, l.err
}

func actorSeleccionadaRPTPrueba(t *testing.T) vecdomain.ContextoActor {
	t.Helper()
	z := strings.Repeat("a", 24)
	ahora := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	cuenta := vecdomain.CuentaAutenticadaContextoActor{CuentaRef: "cta_" + z, Metodo: vecdomain.AuthMethodCertificate, Garantia: vecdomain.AuthAssuranceHigh}
	i := vecdomain.InstantaneaContextoActor{VinculoRef: "vca_" + z, VinculoVersion: 1, CuentaRef: cuenta.CuentaRef, CuentaVersion: 1, PersonaRef: "per_" + z, PersonaVersion: 1, PerfilActivoRef: "prf_" + z, PerfilVersion: 1, Estado: vecdomain.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour), Vinculos: []vecdomain.VinculoReferenciaContextoActor{{VinculoRef: "pep_" + z, Version: 1, Tipo: vecdomain.TipoReferenciaContextoActorEmpleado, Referencia: "emp_" + z, Estado: vecdomain.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour)}}}
	a, err := vecdomain.NuevoContextoActor(cuenta, i, ahora)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

// La preparación se produce mediante la función real de PR356; la ficha es
// sintética y no acredita autorización B2 ni lectura PostgreSQL.
func preparacionSeleccionadaRPTPrueba(t *testing.T) (domain.FichaEmpleadoB2, domain.PreparacionRelacionParaRPT) {
	t.Helper()
	f := domain.FichaEmpleadoB2{EmpleadoRef: "emp_" + strings.Repeat("e", 24), PersonaRef: "per_" + strings.Repeat("p", 24), OrganismoRef: "organismo:sintetico", Version: 7, Corte: domain.CorteEmpleadoB2{VigenteEn: "2026-10-01", ConocidoEn: time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)}}
	anterior := domain.RelacionRegistroEmpleadoB2{RelacionRef: "rel_" + strings.Repeat("r", 24), OrganismoRef: f.OrganismoRef, Estado: "vigente", UnidadDenominacion: "dato nominal excluido", Traza: domain.TrazaEmpleadoB2{Desde: "2024-01-01", RegistradaEn: f.Corte.ConocidoEn.Add(-time.Hour), Version: 1, ActoRef: "acto:sintetico", FuenteRef: "fuente:sintetica", FuenteVersion: 1}}
	ultima := anterior
	ultima.Estado, ultima.Traza.Version, ultima.Traza.RegistradaEn, ultima.Traza.Hasta = "finalizada", 2, f.Corte.ConocidoEn, f.Corte.VigenteEn
	otra := anterior
	otra.RelacionRef, otra.Estado = "rel_"+strings.Repeat("o", 24), "suspendida"
	f.Relaciones = []domain.RelacionRegistroEmpleadoB2{anterior, otra, ultima}
	f.Ocupaciones = []domain.OcupacionEmpleadoB2{{PuestoRef: "puesto:excluido", PlazaRef: "plaza:excluida"}}
	p, err := domain.PrepararRelacionParaRPT(f)
	if err != nil {
		t.Fatal(err)
	}
	return f, p
}

func TestPuenteRelacionRPTUsaPreparacionRealYVersionDeTraza(t *testing.T) {
	f, p := preparacionSeleccionadaRPTPrueba(t)
	a := actorSeleccionadaRPTPrueba(t)
	if p.VersionFicha == p.Relaciones[0].Traza.Version || p.Relaciones[0].Estado != "finalizada" || p.Relaciones[0].EnIntervalo {
		t.Fatal("fixture sin revisión finalizada distinta de versión de ficha")
	}
	esperado := ports.ResultadoRelacionParaRPTV1{Relacion: ports.RelacionParaRPTV1{RelacionRef: p.Relaciones[0].RelacionRef, Estado: "finalizada"}, Cobertura: ports.CoberturaPersonalNoAcreditadaV1, Evidencia: ports.EvidenciaRegistroEmpleadoB2{DecisionRef: "decision:rpt_propia", ReciboRef: "recibo:rpt_propio"}}
	lector := &lectorSeleccionadaRPTPrueba{resultado: esperado}
	puente, err := NuevoLectorRelacionSeleccionadaRPT(lector, registroPuenteRPTPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	r, err := puente.ConsultarSeleccionada(context.Background(), a, p, f.OrganismoRef, p.Relaciones[0].RelacionRef)
	if err != nil || !reflect.DeepEqual(r, esperado) || len(lector.consultas) != 1 {
		t.Fatalf("consulta no delegada una vez: %+v, %v", r, err)
	}
	q := lector.consultas[0]
	if !reflect.DeepEqual(q.Actor, a) || q.EmpleadoRef != p.EmpleadoRef || q.OrganismoRef != f.OrganismoRef || q.RelacionRef != p.Relaciones[0].RelacionRef || q.VersionEsperada != 2 || q.Corte != p.Corte {
		t.Fatalf("selector o actor alterado: %+v", q)
	}
	if q.Actor.Instantanea.Vinculos[0].Referencia == q.EmpleadoRef {
		t.Fatal("el actor debe conservar otro empleado")
	}
	material, err := domain.NuevoMaterialLectorRelacionRPT(domain.SolicitudLectorRelacionRPT{Actor: q.Actor, EmpleadoRef: q.EmpleadoRef, RelacionRef: q.RelacionRef, OrganismoRef: q.OrganismoRef, VersionEsperada: q.VersionEsperada, Corte: q.Corte})
	if err != nil {
		t.Fatal(err)
	}
	if material.Recurso().Referencia != q.RelacionRef || material.Recurso().Tipo != domain.TipoRecursoLectorRelacionRPT || material.Recurso().Atributos["operacion"] != "relacion_para_rpt" {
		t.Fatal("selector no corresponde a consulta nominal RPT")
	}
	b, err := json.Marshal(q)
	if err != nil {
		t.Fatal(err)
	}
	for _, excluido := range []string{"dato nominal excluido", "puesto:excluido", "plaza:excluida", "acto:sintetico", "fuente:sintetica", "version_ficha", "preparacion", "Ficha", "Autorizacion", "Evidencia"} {
		if strings.Contains(string(b), excluido) {
			t.Fatalf("la consulta copió datos o autoridad B2: %s", excluido)
		}
	}
	if f.Version != 7 || len(f.Relaciones) != 3 || f.Relaciones[0].Estado != "vigente" {
		t.Fatal("se alteró la historia fuente")
	}
}

func TestPuenteRelacionRPTRechazaSeleccionAusenteDuplicadaOPreparacionInvalida(t *testing.T) {
	casos := map[string]func(*domain.PreparacionRelacionParaRPT, *string){
		"ausente": func(_ *domain.PreparacionRelacionParaRPT, seleccion *string) {
			*seleccion = "rel_" + strings.Repeat("x", 24)
		},
		"duplicada": func(p *domain.PreparacionRelacionParaRPT, _ *string) {
			p.Relaciones = append(p.Relaciones, p.Relaciones[0])
		},
		"sin_relaciones": func(p *domain.PreparacionRelacionParaRPT, _ *string) { p.Relaciones = nil },
		"esquema":        func(p *domain.PreparacionRelacionParaRPT, _ *string) { p.Esquema = "vec.personal.ficha.v1" },
		"uso":            func(p *domain.PreparacionRelacionParaRPT, _ *string) { p.Uso = "ocupacion" },
		"cobertura":      func(p *domain.PreparacionRelacionParaRPT, _ *string) { p.Cobertura = "completa" },
		"estado_rpt":     func(p *domain.PreparacionRelacionParaRPT, _ *string) { p.EstadoRPT = "ocupada" },
		"corte":          func(p *domain.PreparacionRelacionParaRPT, _ *string) { p.Corte.VigenteEn = "2026-02-30" },
		"empleado":       func(p *domain.PreparacionRelacionParaRPT, _ *string) { p.EmpleadoRef = "" },
		"version_traza":  func(p *domain.PreparacionRelacionParaRPT, _ *string) { p.Relaciones[0].Traza.Version = 0 },
		"traza_futura": func(p *domain.PreparacionRelacionParaRPT, _ *string) {
			p.Relaciones[0].Traza.RegistradaEn = p.Corte.ConocidoEn.Add(time.Second)
		},
		"sin_fuente": func(p *domain.PreparacionRelacionParaRPT, _ *string) { p.Relaciones[0].Traza.FuenteRef = "" },
		"exceso": func(p *domain.PreparacionRelacionParaRPT, _ *string) {
			p.Relaciones = make([]domain.RelacionPreparacionParaRPT, 201)
		},
	}
	for nombre, mutar := range casos {
		t.Run(nombre, func(t *testing.T) {
			f, p := preparacionSeleccionadaRPTPrueba(t)
			seleccion := p.Relaciones[0].RelacionRef
			mutar(&p, &seleccion)
			lector := &lectorSeleccionadaRPTPrueba{resultado: ports.ResultadoRelacionParaRPTV1{Relacion: ports.RelacionParaRPTV1{RelacionRef: seleccion}}}
			puente, err := NuevoLectorRelacionSeleccionadaRPT(lector, registroPuenteRPTPrueba{})
			if err != nil {
				t.Fatal(err)
			}
			r, err := puente.ConsultarSeleccionada(context.Background(), actorSeleccionadaRPTPrueba(t), p, f.OrganismoRef, seleccion)
			if !errors.Is(err, domain.ErrLectorRelacionRPTInvalido) || len(lector.consultas) != 0 || !reflect.DeepEqual(r, ports.ResultadoRelacionParaRPTV1{}) {
				t.Fatalf("preparación inválida pasó al lector: %+v, %v", r, err)
			}
		})
	}
}

func TestPuenteRelacionRPTConservaErrorDelLectorPropio(t *testing.T) {
	f, p := preparacionSeleccionadaRPTPrueba(t)
	for _, esperado := range []error{domain.ErrLectorRelacionRPTDenegado, domain.ErrLectorRelacionRPTNoDisponible, context.Canceled} {
		lector := &lectorSeleccionadaRPTPrueba{err: esperado}
		puente, err := NuevoLectorRelacionSeleccionadaRPT(lector, registroPuenteRPTPrueba{})
		if err != nil {
			t.Fatal(err)
		}
		r, err := puente.ConsultarSeleccionada(context.Background(), actorSeleccionadaRPTPrueba(t), p, f.OrganismoRef, p.Relaciones[0].RelacionRef)
		if !errors.Is(err, esperado) || len(lector.consultas) != 1 || !reflect.DeepEqual(r, ports.ResultadoRelacionParaRPTV1{}) {
			t.Fatalf("fallo del lector convertido en preparación: %+v, %v", r, err)
		}
	}
}

func TestPuenteRelacionRPTClonaActorSinMutarOriginal(t *testing.T) {
	f, p := preparacionSeleccionadaRPTPrueba(t)
	a := actorSeleccionadaRPTPrueba(t)
	original, err := a.Clonar()
	if err != nil {
		t.Fatal(err)
	}
	lector := &lectorSeleccionadaRPTPrueba{mutar: func(q *ports.ConsultaRelacionParaRPTV1) {
		q.Actor.Instantanea.Vinculos[0].Referencia = "emp_" + strings.Repeat("x", 24)
	}}
	puente, err := NuevoLectorRelacionSeleccionadaRPT(lector, registroPuenteRPTPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := puente.ConsultarSeleccionada(context.Background(), a, p, f.OrganismoRef, p.Relaciones[0].RelacionRef); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a, original) {
		t.Fatal("lector modificó el contexto original del actor")
	}
}

func TestPuenteRelacionRPTDeniegaSinLectorOActorValido(t *testing.T) {
	var nulo *lectorSeleccionadaRPTPrueba
	for _, lector := range []ports.LectorRelacionParaRPTV1{nil, nulo} {
		if p, err := NuevoLectorRelacionSeleccionadaRPT(lector, registroPuenteRPTPrueba{}); p != nil || !errors.Is(err, domain.ErrLectorRelacionRPTNoDisponible) {
			t.Fatal("puente sin lector compuesto")
		}
	}
	f, p := preparacionSeleccionadaRPTPrueba(t)
	lector := &lectorSeleccionadaRPTPrueba{}
	puente, err := NuevoLectorRelacionSeleccionadaRPT(lector, registroPuenteRPTPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	r, err := puente.ConsultarSeleccionada(context.Background(), vecdomain.ContextoActor{}, p, f.OrganismoRef, p.Relaciones[0].RelacionRef)
	if !errors.Is(err, domain.ErrLectorRelacionRPTInvalido) || len(lector.consultas) != 0 || !reflect.DeepEqual(r, ports.ResultadoRelacionParaRPTV1{}) {
		t.Fatal("actor inválido pasó al lector")
	}
}

type registroPuenteRPTPrueba struct{}

func (registroPuenteRPTPrueba) VerificarRegistroRelacionRPT(context.Context) error { return nil }
func (registroPuenteRPTPrueba) RegistrarIntentoRelacionRPT(context.Context, ports.IntentoLectorRelacionRPT) error {
	return nil
}

type registroRechazoPuenteRPTPrueba struct {
	writes int
	fallo  bool
}

func (r *registroRechazoPuenteRPTPrueba) VerificarRegistroRelacionRPT(context.Context) error {
	return nil
}
func (r *registroRechazoPuenteRPTPrueba) RegistrarIntentoRelacionRPT(context.Context, ports.IntentoLectorRelacionRPT) error {
	r.writes++
	if r.fallo {
		return domain.ErrLectorRelacionRPTNoDisponible
	}
	return nil
}
func TestPuenteRPTSeleccionInvalidaTambienRegistra(t *testing.T) {
	fuente := &lectorSeleccionadaRPTPrueba{}
	registro := &registroRechazoPuenteRPTPrueba{}
	l, err := NuevoLectorRelacionSeleccionadaRPT(fuente, registro)
	if err != nil {
		t.Fatal(err)
	}
	_, err = l.ConsultarSeleccionada(context.Background(), vecdomain.ContextoActor{}, domain.PreparacionRelacionParaRPT{}, "organismo:dipgra", "rel_invalida")
	if !errors.Is(err, domain.ErrLectorRelacionRPTInvalido) || registro.writes != 1 || len(fuente.consultas) != 0 {
		t.Fatal("selección rechazada sin registro", err)
	}
	registro.fallo = true
	_, err = l.ConsultarSeleccionada(context.Background(), vecdomain.ContextoActor{}, domain.PreparacionRelacionParaRPT{}, "organismo:dipgra", "rel_invalida")
	if !errors.Is(err, domain.ErrLectorRelacionRPTNoDisponible) || registro.writes != 2 || len(fuente.consultas) != 0 {
		t.Fatal("registro fallido no cierra puente", err)
	}
}
