package incorporacionejercicio

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	dom "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	pd "vec-diputacion-granada/internal/modules/personal/domain"
	pp "vec-diputacion-granada/internal/modules/personal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

// actosCeseB2Prueba imita la función SQL 000019 en lo que importa al cese:
// la clave de idempotencia devuelve el recibo original si el material es el
// mismo, la revisión esperada se compara con la última y cada escritura
// añade una revisión a la ficha compartida.
type actosCeseB2Prueba struct {
	ficha       *fichaConsumidorB2Prueba
	recibos     map[string]pp.ReciboActoRegistroEmpleadoB2
	materiales  map[string]pd.SolicitudHechoEmpleadoB2
	escrituras  int
	llamadas    int
	ultima      pd.SolicitudHechoEmpleadoB2
	falloAntes  error
	falloTras   error
	instante    time.Time
	secuenciaID int
}

func (a *actosCeseB2Prueba) RegistrarHecho(_ context.Context, s pd.SolicitudHechoEmpleadoB2) (pp.ResultadoHechoEmpleadoB2, error) {
	a.llamadas++
	a.ultima = s
	if a.falloAntes != nil {
		return pp.ResultadoHechoEmpleadoB2{}, a.falloAntes
	}
	clave := s.Procedencia.IdempotenciaRef
	if previo, ok := a.recibos[clave]; ok {
		if !reflect.DeepEqual(a.materiales[clave], s) {
			return pp.ResultadoHechoEmpleadoB2{}, pd.ErrRegistroEmpleadoB2Conflicto
		}
		return pp.ResultadoHechoEmpleadoB2{Recibo: previo}, nil
	}
	var ultima *pd.RelacionRegistroEmpleadoB2
	for i := range a.ficha.ficha.Relaciones {
		r := &a.ficha.ficha.Relaciones[i]
		if r.RelacionRef == s.RelacionRef && (ultima == nil || r.Traza.Version > ultima.Traza.Version) {
			ultima = r
		}
	}
	if ultima == nil || s.EmpleadoRef != a.ficha.ficha.EmpleadoRef {
		return pp.ResultadoHechoEmpleadoB2{}, pd.ErrRegistroEmpleadoB2NoEncontrado
	}
	if ultima.Traza.Version != s.RelacionVersionEsperada || s.RevisionEsperada != s.RelacionVersionEsperada+1 {
		return pp.ResultadoHechoEmpleadoB2{}, pd.ErrRegistroEmpleadoB2Conflicto
	}
	nueva := *ultima
	nueva.Estado = s.Estado
	nueva.Traza.Version = s.RevisionEsperada
	nueva.Traza.Desde, nueva.Traza.Hasta = s.VigenteDesde, s.VigenteHasta
	nueva.Traza.ActoRef, nueva.Traza.FuenteRef, nueva.Traza.FuenteVersion = s.Procedencia.ActoRef, s.Procedencia.FuenteRef, s.Procedencia.FuenteVersion
	a.ficha.ficha.Relaciones = append(a.ficha.ficha.Relaciones, nueva)
	a.secuenciaID++
	rec := pp.ReciboActoRegistroEmpleadoB2{ReciboRef: "perrec_" + strings.Repeat(string(rune('a'+a.secuenciaID)), 32), EmpleadoRef: s.EmpleadoRef,
		RelacionRef: s.RelacionRef, HechoRef: s.RelacionRef, Tipo: "relacion", Version: s.RevisionEsperada, RegistradoEn: a.instante, EfectoRef: s.EmpleadoRef}
	a.recibos[clave], a.materiales[clave] = rec, s
	a.escrituras++
	if a.falloTras != nil {
		// La transacción de Personal confirmó, pero la respuesta no llegó.
		return pp.ResultadoHechoEmpleadoB2{}, a.falloTras
	}
	return pp.ResultadoHechoEmpleadoB2{Recibo: rec}, nil
}

type origenCeseB2Prueba struct {
	origen     ct.OrigenIncorporacionPersonalB2
	encontrado bool
	lecturas   int
}

func (o *origenCeseB2Prueba) LeerOrigenIncorporacionB2(context.Context, string, string) (ct.OrigenIncorporacionPersonalB2, bool, error) {
	o.lecturas++
	return o.origen, o.encontrado, nil
}

type actoresCeseB2Prueba struct{ actor core.ContextoActor }

func (a actoresCeseB2Prueba) ActorLecturaHechosB2(context.Context) (core.ContextoActor, error) {
	return a.actor, nil
}
func (a actoresCeseB2Prueba) ActorHechoB2(context.Context) (core.ContextoActor, error) {
	return a.actor, nil
}

type escenarioCeseB2 struct {
	cese      *CesePersonalB2
	contrato  ContratoPlanNominal
	contratos int
	origen    *origenCeseB2Prueba
	ficha     *fichaConsumidorB2Prueba
	actos     *actosCeseB2Prueba
	solicitud SolicitudCesePersonalB2
	actor     core.ContextoActor
}

const (
	empleadoCeseB2     = "emp_eeeeeeeeeeeeeeeeeeeeeeee"
	relacionCeseB2     = "rel_rrrrrrrrrrrrrrrrrrrrrrrr"
	origenReciboCeseB2 = ct.PrefijoReciboOrigenIncorporacionPersonalB2 + "5f0c2a4e-8d1b-4c7a-9e3f-1a2b3c4d5e6f"
)

func nuevoEscenarioCeseB2(t *testing.T, regla ReglaFechaCesePersonalB2) *escenarioCeseB2 {
	t.Helper()
	ahora := time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC)
	ctxActor, _, _ := autoridadFixtureContexto(t, ahora, "p", "c")
	actor := ctxActor.Resultado.Contexto
	e := &escenarioCeseB2{contrato: contratoB2Prueba(), actor: actor}
	d := e.contrato.DatosPersonal
	snapshot := func(tipo, ref string) *pd.SnapshotEntradaCatalogoEmpleadoB2 {
		return &pd.SnapshotEntradaCatalogoEmpleadoB2{OrganismoRef: d.OrganismoRef, Tipo: tipo, Ref: ref, Version: 1, Revision: 1,
			Estado: "publicada", Denominacion: "Entrada sintética", VigenteDesde: d.Desde, HuellaSHA256: strings.Repeat("a", 64)}
	}
	e.ficha = &fichaConsumidorB2Prueba{ficha: pd.FichaEmpleadoB2{EmpleadoRef: empleadoCeseB2, PersonaRef: e.contrato.PersonaRef,
		OrganismoRef: d.OrganismoRef, Version: 2, Relaciones: []pd.RelacionRegistroEmpleadoB2{{RelacionRef: relacionCeseB2,
			OrganismoRef: d.OrganismoRef, UnidadRef: d.UnidadRef, RegimenRef: d.Regimen.Ref, ModalidadRef: d.Modalidad.Ref, Estado: "vigente",
			Traza: pd.TrazaEmpleadoB2{Desde: d.Desde, Hasta: "2027-03-31", RegistradaEn: ahora.Add(-time.Hour), Version: 1,
				ActoRef: d.Procedencia.ActoRef, FuenteRef: d.Procedencia.FuenteRef, FuenteVersion: d.Procedencia.FuenteVersion},
			CatalogoSnapshot: pd.SnapshotCatalogoEmpleadoB2{Regimen: snapshot("regimen", d.Regimen.Ref), Modalidad: snapshot("modalidad", d.Modalidad.Ref)}}}}}
	e.origen = &origenCeseB2Prueba{encontrado: true, origen: ct.OrigenIncorporacionPersonalB2{Protocolo: ct.ProtocoloIncorporacionPersonalB2, ReciboRef: origenReciboCeseB2,
		Confirmacion: ct.ConfirmacionOrigenIncorporacionB2{OrganizacionRef: e.contrato.OrganizacionRef, ExpedienteRef: e.contrato.ExpedienteRef,
			Hechos: ct.HechosPersonalIncorporacionB2{EmpleadoRef: empleadoCeseB2, RelacionRef: relacionCeseB2, RelacionVersion: 1}}}}
	e.actos = &actosCeseB2Prueba{ficha: e.ficha, recibos: map[string]pp.ReciboActoRegistroEmpleadoB2{},
		materiales: map[string]pd.SolicitudHechoEmpleadoB2{}, instante: ahora}
	e.solicitud = SolicitudCesePersonalB2{Recibo: ct.ReciboOperacionSeguimiento{Operacion: ct.OperacionRegistrarCese,
		OrganizacionRef: e.contrato.OrganizacionRef, ExpedienteRef: e.contrato.ExpedienteRef, VersionAnterior: 9, VersionResultante: 10,
		FaseResultante: dom.FaseNombramiento, EstadoResultante: dom.EstadoEnCurso, ReciboRef: "recibo:ct-cese:0123abcd",
		AuditoriaRef: "auditoria:cese", EventoRef: "evento:cese", ActorRef: actor.Principal.ID, RegistradaEn: ahora,
		CausaClave: "fin_necesidad", FechaEfecto: "2026-12-15"},
		JustificanteRef: "documento:cese", JustificanteSHA256: strings.Repeat("c", 64), IncorporacionReciboRef: origenReciboCeseB2}
	cese, err := NuevoCesePersonalB2(ConfiguracionCesePersonalB2{
		Contratos: contratoNominalPrueba(func(context.Context, string, string) (ContratoPlanNominal, error) {
			e.contratos++
			return e.contrato, nil
		}),
		Origen: e.origen, Ficha: e.ficha, Actos: e.actos, Actores: actoresCeseB2Prueba{actor: actor},
		Reloj: autoridadRelojDoble{instante: ahora}, Fecha: regla})
	if err != nil {
		t.Fatal(err)
	}
	e.cese = cese
	return e
}

func (e *escenarioCeseB2) finalizar() (ResultadoCesePersonalB2, error) {
	return e.cese.FinalizarRelacionPersonalB2(context.Background(), e.solicitud)
}

func TestCesePersonalB2FinalizaRelacionConRevisionEsperadaYReglaDeFecha(t *testing.T) {
	for _, caso := range []struct {
		regla ReglaFechaCesePersonalB2
		hasta pd.FechaCivil
	}{{CeseUltimoDiaTrabajado, "2026-12-16"}, {CesePrimerDiaSinRelacion, "2026-12-15"}} {
		t.Run(string(caso.regla), func(t *testing.T) {
			e := nuevoEscenarioCeseB2(t, caso.regla)
			r, err := e.finalizar()
			if err != nil {
				t.Fatal(err)
			}
			s := e.actos.ultima
			if !r.Aplica || r.VigenteHasta != caso.hasta || r.Recibo.Version != 2 || r.RelacionRef != relacionCeseB2 || e.actos.escrituras != 1 {
				t.Fatalf("resultado inesperado: %+v", r)
			}
			if s.Tipo != "relacion" || s.Estado != "finalizada" || s.RelacionVersionEsperada != 1 || s.RevisionEsperada != 2 ||
				s.VigenteDesde != e.contrato.DatosPersonal.Desde || s.VigenteHasta != caso.hasta || s.EmpleadoRef != empleadoCeseB2 ||
				s.OrganismoRef != e.contrato.DatosPersonal.OrganismoRef || s.UnidadRef != e.contrato.DatosPersonal.UnidadRef ||
				s.Regimen != e.contrato.DatosPersonal.Regimen || s.Modalidad != e.contrato.DatosPersonal.Modalidad {
				t.Fatalf("solicitud B2 inesperada: %+v", s)
			}
			p := s.Procedencia
			if p.ActoRef != e.solicitud.Recibo.ReciboRef || p.FuenteRef != e.solicitud.Recibo.ReciboRef || p.FuenteVersion != 10 ||
				p.Validar() != nil || p.IdempotenciaRef != r.IdempotenciaRef {
				t.Fatalf("procedencia no ligada al recibo del cese: %+v", p)
			}
		})
	}
}

func TestCesePersonalB2RepetirMismoCeseDevuelveMismoReciboSinOtraRevision(t *testing.T) {
	e := nuevoEscenarioCeseB2(t, CeseUltimoDiaTrabajado)
	primero, err := e.finalizar()
	if err != nil {
		t.Fatal(err)
	}
	segundo, err := e.finalizar()
	if err != nil {
		t.Fatal(err)
	}
	if segundo.Recibo != primero.Recibo || segundo.IdempotenciaRef != primero.IdempotenciaRef || e.actos.escrituras != 1 || len(e.ficha.ficha.Relaciones) != 2 {
		t.Fatalf("el reintento duplicó o cambió el recibo: %+v / %+v, escrituras=%d", primero, segundo, e.actos.escrituras)
	}
	otro := nuevoEscenarioCeseB2(t, CeseUltimoDiaTrabajado)
	otro.solicitud.Recibo.ReciboRef = "recibo:ct-cese:ffff0000"
	r, err := otro.finalizar()
	if err != nil || r.IdempotenciaRef == primero.IdempotenciaRef {
		t.Fatalf("un cese distinto debe usar otra clave: %v %s", err, r.IdempotenciaRef)
	}
}

func TestCesePersonalB2FalloTrasConfirmarPersonalYReintentoDejaUnSoloRecibo(t *testing.T) {
	e := nuevoEscenarioCeseB2(t, CesePrimerDiaSinRelacion)
	e.actos.falloTras = pd.ErrRegistroEmpleadoB2NoDisponible
	if _, err := e.finalizar(); err == nil {
		t.Fatal("se dio por hecho el fin sin recibo de Personal")
	}
	e.actos.falloTras = nil
	r, err := e.finalizar()
	if err != nil {
		t.Fatal(err)
	}
	if e.actos.escrituras != 1 || len(e.ficha.ficha.Relaciones) != 2 || r.Recibo.Version != 2 || e.actos.llamadas != 2 {
		t.Fatalf("el reintento duplicó la revisión: escrituras=%d revisiones=%d", e.actos.escrituras, len(e.ficha.ficha.Relaciones))
	}
	if e.actos.ultima.RelacionVersionEsperada != 1 || e.actos.ultima.RevisionEsperada != 2 {
		t.Fatalf("el reintento no repitió la solicitud original: %+v", e.actos.ultima)
	}
}

func TestCesePersonalB2RechazaRelacionNoVigenteORevisionDivergenteSinEfecto(t *testing.T) {
	for _, caso := range []struct {
		nombre  string
		cambiar func(*escenarioCeseB2)
	}{
		{"finalizada_por_otro_acto", func(e *escenarioCeseB2) {
			r := e.ficha.ficha.Relaciones[0]
			r.Estado, r.Traza.Version, r.Traza.Hasta, r.Traza.ActoRef, r.Traza.FuenteRef = "finalizada", 2, "2026-11-01", "acto:otro", "fuente:otra"
			e.ficha.ficha.Relaciones = append(e.ficha.ficha.Relaciones, r)
		}},
		{"suspendida", func(e *escenarioCeseB2) { e.ficha.ficha.Relaciones[0].Estado = "suspendida" }},
		{"revision_anterior_al_origen", func(e *escenarioCeseB2) { e.origen.origen.Confirmacion.Hechos.RelacionVersion = 3 }},
		{"fin_antes_del_inicio", func(e *escenarioCeseB2) { e.solicitud.Recibo.FechaEfecto = "2026-09-01" }},
		{"cese_alarga_relacion_prevista", func(e *escenarioCeseB2) { e.solicitud.Recibo.FechaEfecto = "2027-04-30" }},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			e := nuevoEscenarioCeseB2(t, CeseUltimoDiaTrabajado)
			caso.cambiar(e)
			_, err := e.finalizar()
			if !errors.Is(err, ct.ErrConflictoIncorporacionAplicacion) || e.actos.llamadas != 0 {
				t.Fatalf("se aceptó o escribió: %v, llamadas=%d", err, e.actos.llamadas)
			}
		})
	}
	// La revisión cambia entre la lectura y la escritura: Personal rechaza por CAS.
	e := nuevoEscenarioCeseB2(t, CeseUltimoDiaTrabajado)
	e.actos.falloAntes = pd.ErrRegistroEmpleadoB2Conflicto
	if _, err := e.finalizar(); !errors.Is(err, ct.ErrConflictoIncorporacionAplicacion) || e.actos.escrituras != 0 {
		t.Fatalf("revisión divergente no se rechazó: %v", err)
	}
}

func TestCesePersonalB2RechazaRelacionDeOtroEmpleadoOOrganismo(t *testing.T) {
	for _, caso := range []struct {
		nombre  string
		cambiar func(*escenarioCeseB2)
		want    error
	}{
		{"otra_persona", func(e *escenarioCeseB2) { e.ficha.ficha.PersonaRef = "per_zzzzzzzzzzzzzzzzzzzzzzzz" }, ct.ErrConflictoIncorporacionAplicacion},
		{"otro_organismo_en_ficha", func(e *escenarioCeseB2) { e.ficha.ficha.OrganismoRef = "organismo:otro" }, ct.ErrConflictoIncorporacionAplicacion},
		{"relacion_de_otro_organismo", func(e *escenarioCeseB2) { e.ficha.ficha.Relaciones[0].OrganismoRef = "organismo:otro" }, ct.ErrConflictoIncorporacionAplicacion},
		{"relacion_ausente", func(e *escenarioCeseB2) {
			e.origen.origen.Confirmacion.Hechos.RelacionRef = "rel_xxxxxxxxxxxxxxxxxxxxxxxx"
		}, ct.ErrConflictoIncorporacionAplicacion},
		{"origen_de_otro_expediente", func(e *escenarioCeseB2) { e.origen.origen.Confirmacion.ExpedienteRef = "expediente:otro" }, ct.ErrConflictoIncorporacionAplicacion},
		{"otro_actor_rrhh", func(e *escenarioCeseB2) { e.solicitud.Recibo.ActorRef = "principal:otro" }, ct.ErrDenegadaIncorporacionAplicacion},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			e := nuevoEscenarioCeseB2(t, CeseUltimoDiaTrabajado)
			caso.cambiar(e)
			_, err := e.finalizar()
			if !errors.Is(err, caso.want) || e.actos.llamadas != 0 {
				t.Fatalf("se aceptó una relación ajena: %v, llamadas=%d", err, e.actos.llamadas)
			}
		})
	}
}

// Solo el recibo de incorporación que CT asocia al expediente decide si
// aplica; sin origen B2 no se lee nada con permisos B2 ni se escribe.
func TestCesePersonalB2NoAplicaSinOrigenB2EnCT(t *testing.T) {
	for _, incorporacion := range []string{"", "recibo:ct-incorporacion-v2:0123abcd", ct.PrefijoReciboOrigenIncorporacionPersonalB2} {
		e := nuevoEscenarioCeseB2(t, CeseUltimoDiaTrabajado)
		e.solicitud.IncorporacionReciboRef = incorporacion
		r, err := e.finalizar()
		if err != nil || r.Aplica || e.contratos != 0 || e.origen.lecturas != 0 || e.ficha.lecturas != 0 || e.actos.llamadas != 0 {
			t.Fatalf("incorporación %q tocó B2: %v %+v", incorporacion, err, r)
		}
	}
}

// Con origen B2 en CT, cualquier incoherencia con el plan o el origen
// guardados se rechaza en lugar de tratarse como «no aplica».
func TestCesePersonalB2OrigenB2IncoherenteNoSeIgnora(t *testing.T) {
	for _, caso := range []struct {
		nombre  string
		cambiar func(*escenarioCeseB2)
		want    error
	}{
		{"plan_ejercicio_v2", func(e *escenarioCeseB2) { e.contrato.Protocolo = ProtocoloEjercicioV2 }, ct.ErrConflictoIncorporacionAplicacion},
		{"sin_plan_nominal", func(e *escenarioCeseB2) {
			e.cese.c.Contratos = contratoNominalPrueba(func(context.Context, string, string) (ContratoPlanNominal, error) {
				return ContratoPlanNominal{}, ct.ErrPlanNominalB2NoEncontrado
			})
		}, ct.ErrPreparacionIncorporacionPendiente},
		{"origen_sin_confirmar", func(e *escenarioCeseB2) { e.origen.encontrado = false }, ct.ErrPreparacionIncorporacionPendiente},
		{"origen_con_otro_recibo", func(e *escenarioCeseB2) {
			e.origen.origen.ReciboRef = ct.PrefijoReciboOrigenIncorporacionPersonalB2 + "00000000-0000-4000-8000-000000000000"
		}, ct.ErrConflictoIncorporacionAplicacion},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			e := nuevoEscenarioCeseB2(t, CeseUltimoDiaTrabajado)
			caso.cambiar(e)
			if _, err := e.finalizar(); !errors.Is(err, caso.want) || e.actos.llamadas != 0 {
				t.Fatalf("origen B2 incoherente: %v, llamadas=%d", err, e.actos.llamadas)
			}
		})
	}
}

// La lectura puede no ver aún la revisión «finalizada» (reloj de la aplicación
// por detrás del de la base): se repite la solicitud original y Personal la
// resuelve como repetición de la misma clave, sin otra revisión.
func TestCesePersonalB2LecturaQueNoVeLaRevisionFinalSeResuelvePorRepeticion(t *testing.T) {
	e := nuevoEscenarioCeseB2(t, CeseUltimoDiaTrabajado)
	primero, err := e.finalizar()
	if err != nil {
		t.Fatal(err)
	}
	original := e.actos.ultima
	e.ficha.ficha.Relaciones = e.ficha.ficha.Relaciones[:1]
	segundo, err := e.finalizar()
	if err != nil {
		t.Fatal(err)
	}
	if segundo.Recibo != primero.Recibo || e.actos.escrituras != 1 || !reflect.DeepEqual(e.actos.ultima, original) {
		t.Fatalf("la lectura atrasada produjo otra solicitud o revisión: %+v / %+v", primero, segundo)
	}
}

func TestCesePersonalB2ExigeReglaDeFechaYDependencias(t *testing.T) {
	e := nuevoEscenarioCeseB2(t, CeseUltimoDiaTrabajado)
	c := e.cese.c
	for _, regla := range []ReglaFechaCesePersonalB2{"", "fecha_de_efectos"} {
		c.Fecha = regla
		if _, err := NuevoCesePersonalB2(c); !errors.Is(err, ct.ErrComposicionIncorporacionAplicacion) {
			t.Fatalf("regla %q admitida: %v", regla, err)
		}
	}
	c.Fecha, c.Actos = CeseUltimoDiaTrabajado, nil
	if _, err := NuevoCesePersonalB2(c); !errors.Is(err, ct.ErrComposicionIncorporacionAplicacion) {
		t.Fatalf("composición sin actos admitida: %v", err)
	}
	if _, err := CeseUltimoDiaTrabajado.Hasta("2026-02-30"); err == nil {
		t.Fatal("fecha civil inexistente admitida")
	}
	if h, err := CeseUltimoDiaTrabajado.Hasta("2026-12-31"); err != nil || h != "2027-01-01" {
		t.Fatalf("cambio de año: %v %s", err, h)
	}
}
