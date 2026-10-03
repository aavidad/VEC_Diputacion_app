package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type poolCatalogoPrueba struct {
	poolFalso
	roles map[string][]byte
}

func (p *poolCatalogoPrueba) QueryRow(_ context.Context, consulta string, argumentos ...any) pgx.Row {
	if consulta != catalogoSQL {
		return filaFalsa{err: errors.New("consulta no admitida")}
	}
	return filaFalsa{dato: p.roles[argumentos[0].(string)]}
}

func contratoV2Prueba(t *testing.T) (domain.SolicitudActoAdministracionPerfiles, ports.RolAdministrable, *poolCatalogoPrueba, domain.ReciboAdministracionPerfiles) {
	t.Helper()
	ahora := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	actor, evidencia := actorYEvidenciaPrueba(t, ahora)
	version := domain.VersionRol{RolID: "administracion_perfiles", Version: 1, Nombre: "Administración de perfiles",
		Estado: domain.EstadoVersionRolPublicada, PublicadaPor: "responsable-seguridad", PublicadaEn: ahora.Add(-2 * time.Hour),
		Concesiones: []domain.ConcesionRol{{Accion: "administracion.perfiles.otorgar", ModuloID: "administracion",
			TipoRecurso: "perfil", Finalidades: []string{"administrar_perfiles"}, GarantiaMinima: domain.AuthAssuranceHigh}}}
	huellaCatalogo, err := domain.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		t.Fatal(err)
	}
	instantanea := domain.InstantaneaAutorizacion{VersionRol: version,
		AsignacionPerfil: domain.AsignacionPerfil{AsignacionID: "asig-admin", Version: 1,
			PerfilActivoRef: actor.PerfilActivoRef, PrincipalID: actor.PersonaRef, VersionRolRef: version.Referencia(),
			Estado: domain.EstadoAsignacionPerfilActiva, Ambitos: []domain.AmbitoPerfil{{Clave: "unidad", Valores: []string{"unidad:prueba"}}},
			VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour), EmitidaPor: version.PublicadaPor, EmitidaEn: version.PublicadaEn},
		ControlVigenciaVersionRol: domain.ControlVigenciaVersionRol{VersionRolRef: version.Referencia(), Revision: 1,
			Estado: domain.EstadoControlVigenciaVersionRolHabilitada, ActualizadoPor: version.PublicadaPor, ActualizadoEn: version.PublicadaEn},
		RevisionCatalogoPoliticas: 1, CatalogoPoliticasHuellaSHA256: huellaCatalogo}
	s := domain.SolicitudActoAdministracionPerfiles{OperacionRef: "acto_admin:" + strings.Repeat("a", 32), Actor: actor, Evidencia: evidencia,
		InstantaneaAutorizacion: instantanea, Operacion: domain.OperacionOtorgarPerfil, Clase: domain.ClaseControlPerfilOrdinario,
		RolVersionRef: "rol:cronos_rrhh:v1", CorrelacionRef: "correlacion_" + strings.Repeat("b", 32), ReferenciaActo: "resolucion:prueba",
		Motivo: domain.ReferenciaEntradaCatalogo{CatalogoID: "motivos_admin", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("c", 64), EntradaClave: "provision"},
		Objetivo: domain.PreimagenAdministracionPerfiles{UnidadRef: "unidad:prueba", CentroRef: "centro:prueba",
			CuentaRef: "cta_" + strings.Repeat("d", 22), CuentaVersion: 1, PersonaRef: "per_" + strings.Repeat("e", 22), PersonaVersion: 1,
			PerfilRef: "prf_" + strings.Repeat("f", 22), VinculoRef: "vca_" + strings.Repeat("g", 22), HuellaSHA256: strings.Repeat("d", 64),
			ProcedenciaRef: "procedencia:maestra:prueba", ProcedenciaVersion: 1, ProcedenciaHuellaSHA256: strings.Repeat("e", 64),
			VigenteDesde: ahora, VigenteHasta: ahora.Add(time.Hour)}}
	if err := s.Validar(); err != nil {
		t.Fatal(err)
	}
	huellaRol, err := version.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	categoria, unidad := "aplicacion", true
	admin := rolJSON{VersionRef: version.Referencia(), Clase: domain.ClaseControlPerfilAdministrador, CategoriaAdmin: &categoria,
		HuellaSHA256: huellaRol, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour), UnidadRequerida: &unidad}
	rol := ports.RolAdministrable{VersionRef: s.RolVersionRef, Clase: domain.ClaseControlPerfilOrdinario, HuellaSHA256: strings.Repeat("f", 64),
		VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour), UnidadRequerida: true}
	objetivo := rolJSON{VersionRef: rol.VersionRef, Clase: rol.Clase, HuellaSHA256: rol.HuellaSHA256,
		VigenteDesde: rol.VigenteDesde, VigenteHasta: rol.VigenteHasta, UnidadRequerida: &unidad}
	adminBytes, _ := json.Marshal(admin)
	rolBytes, _ := json.Marshal(objetivo)
	pool := &poolCatalogoPrueba{roles: map[string][]byte{version.Referencia(): adminBytes, rol.VersionRef: rolBytes}}
	p := s.Objetivo
	r := domain.ReciboAdministracionPerfiles{OperacionRef: s.OperacionRef, ActoRef: s.OperacionRef,
		ReciboRef: "recibo_admin:" + strings.Repeat("a", 32), AuditoriaRef: "auditoria:prueba", ActorPersonaRef: s.Actor.PersonaRef,
		PerfilActivoRef: s.Actor.PerfilActivoRef, AsignacionPerfilRef: s.InstantaneaAutorizacion.AsignacionPerfil.Referencia(), CorrelacionRef: s.CorrelacionRef,
		ObjetivoPersonaRef: p.PersonaRef, PerfilRef: p.PerfilRef, VinculoRef: p.VinculoRef, EstadoPosterior: domain.EstadoVinculoContextoActorActivo,
		VersionPosterior: 1, HuellaAntesSHA256: p.HuellaSHA256, HuellaDespuesSHA256: strings.Repeat("f", 64), ConfirmadoEn: ahora,
		UnidadRef: p.UnidadRef, CentroRef: p.CentroRef, RolVersionRef: s.RolVersionRef, VigenteDesde: p.VigenteDesde, VigenteHasta: p.VigenteHasta,
		Motivo: s.Motivo, ReferenciaActo: s.ReferenciaActo}
	if err := r.ValidarPara(s); err != nil {
		t.Fatal(err)
	}
	return s, rol, pool, r
}

func TestMaterialV2ConservaInicioCentroYAsignacionSinCorrelacionDeAcceso(t *testing.T) {
	s, rol, _, _ := contratoV2Prueba(t)
	original, err := materialActo(s, rol, false)
	if err != nil {
		t.Fatal(err)
	}
	s.CorrelacionRef = "correlacion_" + strings.Repeat("9", 32)
	s.InstantaneaAutorizacion.RevisionCatalogoPoliticas++
	replay, err := materialActo(s, rol, false)
	if err != nil || string(replay.Material) != string(original.Material) || replay.CorrelacionAccesoRef == original.CorrelacionAccesoRef {
		t.Fatal("acceso renovado cambia efecto")
	}
	var x actoJSON
	if decodificar(original.Material, &x) != nil || x.Esquema != "administracion_perfiles_acto_v2" ||
		x.Objetivo.CentroRef != s.Objetivo.CentroRef || !x.Objetivo.VigenteDesde.Equal(s.Objetivo.VigenteDesde) ||
		x.AsignacionRef != s.InstantaneaAutorizacion.AsignacionPerfil.Referencia() || strings.Contains(string(original.Material), "correlacion") {
		t.Fatal("material omitió datos o mezcló acceso")
	}
	s.Objetivo.VigenteDesde = s.Objetivo.VigenteDesde.Add(time.Second)
	cambiado, _ := materialActo(s, rol, false)
	if string(cambiado.Material) == string(original.Material) {
		t.Fatal("inicio nuevo mantiene material anterior")
	}
}

func TestProyeccionViejaOCruzadaHaceRollbackAntesDelCommit(t *testing.T) {
	for _, campo := range []string{"completo", "actor_persona_ref", "perfil_activo_ref", "asignacion_perfil_ref", "correlacion_ref", "rol_version_ref", "centro_ref", "vigente_desde", "vigente_hasta", "motivo", "actor_ajeno", "commit_incierto"} {
		t.Run(campo, func(t *testing.T) {
			s, rol, pool, recibo := contratoV2Prueba(t)
			// Proyección SQL simulada: los datos siempre salen de la respuesta, no
			// se completan desde la solicitud. No acredita PostgreSQL ni V3 reales.
			x := proyeccionReciboPrueba(recibo)
			if campo != "completo" && campo != "actor_ajeno" && campo != "commit_incierto" {
				delete(x, campo)
			}
			if campo == "actor_ajeno" {
				x["actor_persona_ref"] = s.Objetivo.PersonaRef
			}
			b, _ := json.Marshal(x)
			tx := &txFalsa{fila: filaFalsa{dato: b}}
			if campo == "commit_incierto" {
				tx.falloCommit = errors.New("conexión perdida")
			}
			pool.tx = tx
			e, _ := materialActo(s, rol, false)
			a := &Autoridad{pool: pool, emisor: &emisorFalso{material: materialSintetico(t, e, recibo.ConfirmadoEn)}, reloj: relojFijo(recibo.ConfirmadoEn)}
			resultado, err := a.AplicarActoOrdinario(context.Background(), s)
			if campo == "completo" {
				if err != nil || resultado != recibo || tx.commits != 1 {
					t.Fatalf("proyección completa: %v", err)
				}
			} else if !errors.Is(err, ports.ErrAutoridadAdministracionPerfilesNoDisponible) || resultado.ReciboRef != "" || tx.commits != map[bool]int{true: 1, false: 0}[campo == "commit_incierto"] {
				t.Fatalf("salida parcial/ajena confirmada: %s %v", campo, err)
			}
			if tx.rollbacks != 1 {
				t.Fatal("falta limpieza transaccional")
			}
		})
	}
}

func proyeccionReciboPrueba(r domain.ReciboAdministracionPerfiles) map[string]any {
	return map[string]any{"operacion_ref": r.OperacionRef, "acto_ref": r.ActoRef, "recibo_ref": r.ReciboRef, "propuesta_ref": r.PropuestaRef,
		"auditoria_ref": r.AuditoriaRef, "actor_persona_ref": r.ActorPersonaRef, "perfil_activo_ref": r.PerfilActivoRef,
		"asignacion_perfil_ref": r.AsignacionPerfilRef, "correlacion_ref": r.CorrelacionRef, "rol_version_ref": r.RolVersionRef,
		"objetivo_persona_ref": r.ObjetivoPersonaRef, "perfil_ref": r.PerfilRef, "vinculo_ref": r.VinculoRef,
		"estado_posterior": r.EstadoPosterior, "version_posterior": r.VersionPosterior, "huella_antes_sha256": r.HuellaAntesSHA256,
		"huella_despues_sha256": r.HuellaDespuesSHA256, "confirmado_en": r.ConfirmadoEn, "unidad_ref": r.UnidadRef,
		"centro_ref": r.CentroRef, "vigente_desde": r.VigenteDesde, "vigente_hasta": r.VigenteHasta, "motivo": r.Motivo, "referencia_acto": r.ReferenciaActo}
}

func TestRevocacionConservaVentanaHistoricaCompletaAntesDeCommit(t *testing.T) {
	for _, ruta := range []string{"ordinario", "cierre"} {
		for _, periodo := range []string{"historia", "ausente", "fin_antes", "submicro", "zona"} {
			t.Run(ruta+"_"+periodo, func(t *testing.T) {
				s, rol, pool, recibo := contratoV2Prueba(t)
				s.Operacion = domain.OperacionRevocarPerfil
				s.Objetivo.PerfilVersion, s.Objetivo.VinculoVersion = 1, 2
				s.Objetivo.VigenteDesde, s.Objetivo.VigenteHasta = time.Time{}, time.Time{}
				recibo.EstadoPosterior, recibo.VersionPosterior = domain.EstadoVinculoContextoActorRevocado, 3
				if err := s.Validar(); err != nil {
					t.Fatal(err)
				}
				cierre := domain.SolicitudCierrePropuestaAdministracionPerfiles{
					OperacionRef: "cierre_admin:" + strings.Repeat("a", 32), PropuestaRef: "propuesta_admin:" + strings.Repeat("b", 32),
					PropuestaHuellaSHA256: strings.Repeat("c", 64), ProponentePersonaRef: "per_" + strings.Repeat("x", 22),
					ObjetivoPersonaRef: s.Objetivo.PersonaRef, Aprobador: s.Actor, Evidencia: s.Evidencia, InstantaneaAutorizacion: s.InstantaneaAutorizacion,
					Decision: domain.DecisionAprobarPropuestaPerfil, Motivo: s.Motivo, CorrelacionRef: s.CorrelacionRef}
				if ruta == "cierre" {
					recibo.OperacionRef, recibo.PropuestaRef = cierre.OperacionRef, cierre.PropuestaRef
				}
				x := proyeccionReciboPrueba(recibo)
				switch periodo {
				case "ausente":
					delete(x, "vigente_desde")
					delete(x, "vigente_hasta")
				case "fin_antes":
					x["vigente_hasta"] = recibo.VigenteDesde.Add(-time.Second)
				case "submicro":
					x["vigente_desde"] = recibo.VigenteDesde.Add(time.Nanosecond)
				case "zona":
					x["vigente_desde"] = recibo.VigenteDesde.In(time.FixedZone("zona", 3600))
				}
				var salida any = x
				e, _ := materialActo(s, rol, false)
				if ruta == "cierre" {
					salida = map[string]any{"operacion_ref": cierre.OperacionRef, "propuesta_ref": cierre.PropuestaRef, "propuesta_huella_sha256": cierre.PropuestaHuellaSHA256,
						"decision": cierre.Decision, "huella_cierre_sha256": strings.Repeat("d", 64), "confirmado_en": recibo.ConfirmadoEn, "recibo": x}
					e, _ = materialCierre(cierre)
				}
				b, _ := json.Marshal(salida)
				tx := &txFalsa{fila: filaFalsa{dato: b}}
				pool.tx = tx
				a := &Autoridad{pool: pool, emisor: &emisorFalso{material: materialSintetico(t, e, recibo.ConfirmadoEn)}, reloj: relojFijo(recibo.ConfirmadoEn)}
				var err error
				if ruta == "cierre" {
					_, err = a.CerrarPropuestaSensible(context.Background(), cierre)
				} else {
					_, err = a.AplicarActoOrdinario(context.Background(), s)
				}
				if periodo == "historia" {
					if err != nil || tx.commits != 1 {
						t.Fatalf("ventana histórica completa perdida: %v", err)
					}
				} else if !errors.Is(err, ports.ErrAutoridadAdministracionPerfilesNoDisponible) || tx.commits != 0 {
					t.Fatalf("ventana incompleta/ajena confirmada: %v", err)
				}
			})
		}
	}
}

func TestCierreExigeHuellaOriginalAntesDeCommitYConservaAccesoSeparado(t *testing.T) {
	for _, completa := range []bool{true, false} {
		s, _, pool, recibo := contratoV2Prueba(t)
		cierre := domain.SolicitudCierrePropuestaAdministracionPerfiles{
			OperacionRef: "cierre_admin:" + strings.Repeat("a", 32), PropuestaRef: "propuesta_admin:" + strings.Repeat("b", 32),
			PropuestaHuellaSHA256: strings.Repeat("c", 64), ProponentePersonaRef: "per_" + strings.Repeat("x", 22),
			ObjetivoPersonaRef: s.Objetivo.PersonaRef, Aprobador: s.Actor, Evidencia: s.Evidencia, InstantaneaAutorizacion: s.InstantaneaAutorizacion,
			Decision: domain.DecisionRechazarPropuestaPerfil, Motivo: s.Motivo, CorrelacionRef: s.CorrelacionRef}
		if err := cierre.Validar(); err != nil {
			t.Fatal(err)
		}
		x := map[string]any{"operacion_ref": cierre.OperacionRef, "propuesta_ref": cierre.PropuestaRef, "decision": cierre.Decision,
			"huella_cierre_sha256": strings.Repeat("d", 64), "confirmado_en": recibo.ConfirmadoEn}
		if completa {
			x["propuesta_huella_sha256"] = cierre.PropuestaHuellaSHA256
		}
		b, _ := json.Marshal(x)
		tx := &txFalsa{fila: filaFalsa{dato: b}}
		pool.tx = tx
		e, _ := materialCierre(cierre)
		cierre.CorrelacionRef = "correlacion_" + strings.Repeat("9", 32)
		replay, _ := materialCierre(cierre)
		if string(e.Material) != string(replay.Material) || e.CorrelacionAccesoRef == replay.CorrelacionAccesoRef {
			t.Fatal("cierre mezcla orden y acceso")
		}
		a := &Autoridad{pool: pool, emisor: &emisorFalso{material: materialSintetico(t, e, recibo.ConfirmadoEn)}, reloj: relojFijo(recibo.ConfirmadoEn)}
		r, err := a.CerrarPropuestaSensible(context.Background(), cierre)
		if completa {
			if err != nil || r.PropuestaHuellaSHA256 != cierre.PropuestaHuellaSHA256 || tx.commits != 1 {
				t.Fatalf("cierre completo: %v", err)
			}
		} else if !errors.Is(err, ports.ErrAutoridadAdministracionPerfilesNoDisponible) || r.OperacionRef != "" || tx.commits != 0 {
			t.Fatalf("cierre parcial confirmado: %v", err)
		}
	}
}

func TestActoSinEvidenciaActualNoEmiteNiAbreTransaccion(t *testing.T) {
	s, _, pool, recibo := contratoV2Prueba(t)
	s.Evidencia = domain.EvidenciaSesionAdministracionPerfiles{}
	a := &Autoridad{pool: pool, emisor: &emisorFalso{}, reloj: relojFijo(recibo.ConfirmadoEn)}
	if _, err := a.AplicarActoOrdinario(context.Background(), s); err == nil || pool.comienzos != 0 {
		t.Fatal("acto sin evidencia actual abrió transacción")
	}
}

func TestCatalogoAdministradorExigeCategoriaExplicitaAplicacion(t *testing.T) {
	for _, categoria := range []string{"ausente", "sistemas", "aplicacion"} {
		t.Run(categoria, func(t *testing.T) {
			s, _, pool, recibo := contratoV2Prueba(t)
			ref := s.InstantaneaAutorizacion.VersionRol.Referencia()
			var x map[string]any
			_ = json.Unmarshal(pool.roles[ref], &x)
			if categoria == "ausente" {
				delete(x, "categoria_admin")
			} else {
				x["categoria_admin"] = categoria
			}
			pool.roles[ref], _ = json.Marshal(x)
			a := &Autoridad{pool: pool, emisor: &emisorFalso{}, reloj: relojFijo(recibo.ConfirmadoEn)}
			err := a.validarAdministrador(context.Background(), s.InstantaneaAutorizacion)
			if (err == nil) != (categoria == "aplicacion") || pool.comienzos != 0 {
				t.Fatalf("categoría no acreditada admitida: %v", err)
			}
		})
	}
}

func TestLoteSinFachadaCentralDevuelveNoDisponibleSinEfecto(t *testing.T) {
	pool := &poolFalso{}
	a := &Autoridad{pool: pool, emisor: &emisorFalso{}, reloj: relojFijo(time.Now())}
	recibo, err := a.AplicarLoteOrdinario(context.Background(), domain.SolicitudLoteAdministracionPerfiles{})
	if !errors.Is(err, ports.ErrAutoridadAdministracionPerfilesNoDisponible) || recibo.ReciboRef != "" || pool.comienzos != 0 {
		t.Fatal("lote abrió efecto sin fachada")
	}
}
