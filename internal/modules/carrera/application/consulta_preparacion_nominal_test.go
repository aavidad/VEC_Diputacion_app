package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
	careerports "vec-diputacion-granada/internal/modules/carrera/ports"
	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/pruebas"
)

type raizNominalPrueba struct {
	q   personalports.ConsultaAntecedentesCarreraV1
	o   vecports.OrdenRegistroConcesionCandidataAutorizacionLigadaV3
	c   vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	err error
}

func (r raizNominalPrueba) ResolverConsultaPreparacionCarrera(context.Context) (personalports.ConsultaAntecedentesCarreraV1, vecports.OrdenRegistroConcesionCandidataAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, error) {
	return r.q, r.o, r.c, r.err
}

type lectorNominalPrueba struct {
	r        personalports.ResultadoAntecedentesCarreraV1
	err      error
	llamadas int
}

func (l *lectorNominalPrueba) ConsultarAntecedentesCarrera(_ context.Context, q personalports.ConsultaAntecedentesCarreraV1) (personalports.ResultadoAntecedentesCarreraV1, error) {
	l.llamadas++
	return l.r, l.err
}
func entornoNominal(t *testing.T) (raizNominalPrueba, *lectorNominalPrueba, time.Time) {
	t.Helper()
	ahora := time.Date(2026, 10, 2, 8, 0, 0, 0, time.UTC)
	persona, perfil := "per_0123456789abcdefghijkl", "prf_0123456789abcdefghijkl"
	ctx, _, err := pruebas.NuevoContextoRegistradoYVinculoV2(ahora, persona, perfil, core.AuthMethodCertificate, core.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	q := personalports.ConsultaAntecedentesCarreraV1{Actor: ctx.Contexto, EmpleadoRef: "emp_0123456789abcdefghijkl", OrganismoRef: "organismo:prueba", Corte: personaldomain.CorteEmpleadoB2{VigenteEn: personaldomain.FechaCivil("2026-10-02"), ConocidoEn: ahora}}
	recurso, err := RecursoConsultaPreparacionCarrera(q)
	if err != nil {
		t.Fatal(err)
	}
	concesion, err := pruebas.NuevaConcesionV3Prueba(pruebas.DatosConcesionV3Prueba{Instante: ahora, PersonaRef: persona, PerfilRef: perfil, Accion: careerports.AccionConsultaPreparacionCarrera, Recurso: recurso, Finalidad: careerports.FinalidadConsultaPreparacionCarrera, Campos: CamposConsultaPreparacionCarrera(), DecisionRef: "decision:carrera:prueba"})
	if err != nil {
		t.Fatal(err)
	}
	datos, err := concesion.Solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	orden, err := vecports.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(concesion.Solicitud, concesion.Decision, datos.ReferenciaMotivo, ctx)
	if err != nil {
		t.Fatal(err)
	}
	nivel := 22
	r := personalports.ResultadoAntecedentesCarreraV1{EmpleadoRef: q.EmpleadoRef, OrganismoRef: q.OrganismoRef, Version: 2, Corte: q.Corte, Cobertura: personalports.CoberturaPersonalParcialV1, Puestos: []personalports.PuestoAntecedenteCarreraV1{{PuestoRef: "puesto:prueba", PuestoVersion: "1", RelacionRef: "relacion:prueba", Periodo: personalports.PeriodoPersonalNominalV1{Desde: "2026-01-01", Hasta: ""}, Procedencia: personalports.ProcedenciaPersonalNominalV1{ActoRef: "acto:prueba", FuenteRef: "fuente:prueba", FuenteVersion: "1", Certeza: personalports.CertezaPersonalPendienteV1}, Nivel: &nivel}}, Situaciones: []personalports.SituacionAntecedenteCarreraV1{{SituacionRef: "dato:no:exponer"}}, Evidencia: personalports.EvidenciaRegistroEmpleadoB2{ReciboRef: "recibo:prueba", DecisionRef: "decision:fuente", EfectoRef: q.EmpleadoRef, ConsumoHuellaSHA256: strings.Repeat("1", 64), AuditoriaRef: "auditoria:prueba", ConsultadaEn: ahora.Add(2 * time.Second)}}
	return raizNominalPrueba{q: q, o: orden, c: concesion.Confirmacion}, &lectorNominalPrueba{r: r}, ahora.Add(2 * time.Second)
}
func TestPreparacionNominalConsumeConcesionComunYReciboBSinConvertirASintetico(t *testing.T) {
	raiz, lector, ahora := entornoNominal(t)
	s, err := NuevoServicioConsultaPreparacionNominal(raiz, lector, func() time.Time { return ahora })
	if err != nil {
		t.Fatal(err)
	}
	r, err := s.Consultar(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if lector.llamadas != 1 || r.Estado != "pendiente" || len(r.Pendientes) != 3 || r.Antecedentes.Evidencia.ReciboRef != "recibo:prueba" || r.Antecedentes.Situaciones != nil {
		t.Fatal("pierde recibo, inventa decisión o expone datos innecesarios")
	}
	*lector.r.Puestos[0].Nivel = 99
	if *r.Antecedentes.Puestos[0].Nivel != 22 {
		t.Fatal("comparte dato mutable con la fuente")
	}
}
func TestPreparacionNominalAusenciaEvidenciaAjenaYRespuestaIncongruenteFallanSinDatos(t *testing.T) {
	for _, tipo := range []string{"confirmacion", "empleado", "organismo", "corte", "respuesta", "recibo", "hash", "caida", "caducada"} {
		t.Run(tipo, func(t *testing.T) {
			r, l, ahora := entornoNominal(t)
			switch tipo {
			case "confirmacion":
				r.c = vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}
			case "empleado":
				r.q.EmpleadoRef = "emp_abcdefghijkl0123456789"
			case "organismo":
				r.q.OrganismoRef = "organismo:ajeno"
			case "corte":
				r.q.Corte.VigenteEn = "2026-10-03"
			case "respuesta":
				l.r.OrganismoRef = "organismo:ajeno"
			case "recibo":
				l.r.Evidencia.ReciboRef = ""
			case "hash":
				l.r.Evidencia.ConsumoHuellaSHA256 = strings.Repeat("z", 64)
			case "caida":
				l.err = errors.New("contenido privado")
			case "caducada":
				ahora = ahora.Add(time.Hour)
			}
			s, _ := NuevoServicioConsultaPreparacionNominal(r, l, func() time.Time { return ahora })
			out, err := s.Consultar(context.Background())
			if !errors.Is(err, ErrPreparacionNominalNoDisponible) || out.Antecedentes.EmpleadoRef != "" {
				t.Fatal("datos ante fallo o evidencia inconsistente")
			}
			if tipo != "respuesta" && tipo != "recibo" && tipo != "hash" && tipo != "caida" && l.llamadas != 0 {
				t.Fatal("consulta fuente sin autorización propia válida")
			}
		})
	}
	if _, err := NuevoServicioConsultaPreparacionNominal(nil, nil, time.Now); !errors.Is(err, ErrPreparacionNominalNoDisponible) {
		t.Fatal("acepta dependencias ausentes")
	}
}
func TestPreparacionNominalSoloDenegacionRegistradaSeDistingue(t *testing.T) {
	r, l, ahora := entornoNominal(t)
	r.err = vecports.ErrDenegacionExplicitaAutorizacionLigadaV3
	s, _ := NuevoServicioConsultaPreparacionNominal(r, l, func() time.Time { return ahora })
	out, err := s.Consultar(context.Background())
	if !errors.Is(err, ErrPreparacionNominalDenegada) || out.Antecedentes.EmpleadoRef != "" || l.llamadas != 0 {
		t.Fatal("denegación no cerrada")
	}
	r.err = errors.Join(r.err, vecports.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible)
	s, _ = NuevoServicioConsultaPreparacionNominal(r, l, func() time.Time { return ahora })
	if _, err = s.Consultar(context.Background()); !errors.Is(err, ErrPreparacionNominalNoDisponible) {
		t.Fatal("denegación no registrada parece definitiva")
	}
}

func TestPreparacionNominalRechazaRecursoDistintoAunqueCompartaAmbitosYAtributos(t *testing.T) {
	for _, campo := range []string{"referencia", "modulo", "tipo"} {
		t.Run(campo, func(t *testing.T) {
			r, l, ahora := entornoNominal(t)
			instante := ahora.Add(-2 * time.Second)
			recurso, _ := RecursoConsultaPreparacionCarrera(r.q)
			switch campo {
			case "referencia":
				recurso.Referencia = "emp_abcdefghijkl0123456789"
			case "modulo":
				recurso.ModuloID = "otro_modulo"
			case "tipo":
				recurso.Tipo = "otro_recurso"
			}
			con, err := pruebas.NuevaConcesionV3Prueba(pruebas.DatosConcesionV3Prueba{Instante: instante, PersonaRef: r.q.Actor.PersonaRef, PerfilRef: r.q.Actor.PerfilActivoRef, Accion: careerports.AccionConsultaPreparacionCarrera, Recurso: recurso, Finalidad: careerports.FinalidadConsultaPreparacionCarrera, Campos: CamposConsultaPreparacionCarrera(), DecisionRef: "decision:carrera:ajena"})
			if err != nil {
				t.Fatal(err)
			}
			ctx, _, err := pruebas.NuevoContextoRegistradoYVinculoV2(instante, r.q.Actor.PersonaRef, r.q.Actor.PerfilActivoRef, core.AuthMethodCertificate, core.AuthAssuranceHigh)
			if err != nil {
				t.Fatal(err)
			}
			sol, _ := con.Solicitud.Datos()
			r.o, err = vecports.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(con.Solicitud, con.Decision, sol.ReferenciaMotivo, ctx)
			if err != nil {
				t.Fatal(err)
			}
			r.c = con.Confirmacion
			s, _ := NuevoServicioConsultaPreparacionNominal(r, l, func() time.Time { return ahora })
			out, err := s.Consultar(context.Background())
			if !errors.Is(err, ErrPreparacionNominalNoDisponible) || l.llamadas != 0 || out.Antecedentes.EmpleadoRef != "" {
				t.Fatal("autoriza recurso diferente por igualdad de ámbitos")
			}
		})
	}
}
func TestPreparacionNominalRelojSeNormalizaYExpiracionDuranteLecturaDescartaDatos(t *testing.T) {
	r, l, ahora := entornoNominal(t)
	reloj := ahora.In(time.FixedZone("zona-prueba", 3600)).Add(123 * time.Nanosecond)
	s, _ := NuevoServicioConsultaPreparacionNominal(r, l, func() time.Time { return reloj })
	if _, err := s.Consultar(context.Background()); err != nil {
		t.Fatal(err)
	}
	llamadas := 0
	s, _ = NuevoServicioConsultaPreparacionNominal(r, l, func() time.Time {
		llamadas++
		if llamadas > 1 {
			return ahora.Add(time.Hour)
		}
		return ahora
	})
	out, err := s.Consultar(context.Background())
	if !errors.Is(err, ErrPreparacionNominalNoDisponible) || out.Antecedentes.EmpleadoRef != "" {
		t.Fatal("presenta datos tras caducar la autorización")
	}
}
