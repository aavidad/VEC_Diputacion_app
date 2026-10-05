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

// txPreparacionPrueba guarda la sentencia y el número de argumentos.
type txPreparacionPrueba struct {
	txFalsa
	sql  string
	args int
}

func (t *txPreparacionPrueba) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	t.sql, t.args = sql, len(args)
	return t.txFalsa.QueryRow(ctx, sql, args...)
}

type poolPreparacionPrueba struct {
	tx        *txPreparacionPrueba
	comienzos int
}

func (p *poolPreparacionPrueba) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	p.comienzos++
	return p.tx, nil
}
func (p *poolPreparacionPrueba) QueryRow(context.Context, string, ...any) pgx.Row {
	return filaFalsa{err: errors.New("no_usada")}
}

// emisorPreparacionPrueba entrega material estructural para el recurso que
// recibe y conserva ese recurso para comprobar su atributo.
type emisorPreparacionPrueba struct {
	t       *testing.T
	s       domain.SolicitudLoteAdministracionPerfiles
	ahora   time.Time
	recurso domain.RecursoAutorizable
	efecto  Efecto
}

func (e *emisorPreparacionPrueba) EmitirLoteOrdinario(_ context.Context, _ domain.ContextoActor,
	_ domain.EvidenciaSesionAdministracionPerfiles, _ domain.InstantaneaAutorizacion,
	recurso domain.RecursoAutorizable, efecto Efecto) (ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	e.recurso, e.efecto = recurso, efecto
	return materialLoteEstructuralPrueba(e.t, e.s, recurso, efecto, e.ahora), nil
}

func solicitudPreparacionPrueba(t *testing.T) (domain.SolicitudPreparacionLoteAdministracionPerfiles, domain.SolicitudLoteAdministracionPerfiles, time.Time) {
	t.Helper()
	lote, _, _ := solicitudLoteOrdinarioPrueba(t)
	// Sesión ADMIN privilegiada (como en las pruebas de auditoría del lote) y
	// asignación del actor alineada con ella.
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	lote.Actor, lote.Evidencia = sesionFronteraPrueba(t, ahora)
	lote.InstantaneaAutorizacion.AsignacionPerfil.PrincipalID = lote.Actor.PersonaRef
	lote.InstantaneaAutorizacion.AsignacionPerfil.PerfilActivoRef = lote.Actor.PerfilActivoRef
	lote.InstantaneaAutorizacion.AsignacionPerfil.VigenteDesde = ahora.Add(-time.Hour)
	lote.InstantaneaAutorizacion.AsignacionPerfil.VigenteHasta = ahora.Add(24 * time.Hour)
	s := domain.SolicitudPreparacionLoteAdministracionPerfiles{OperacionRef: "prep_admin:" + strings.Repeat("c", 32),
		OrganizacionRef: "org_prueba", UnidadRef: "unidad:prueba", PersonaRef: lote.Cambios[0].Objetivo.PersonaRef,
		Actor: lote.Actor, Evidencia: lote.Evidencia, InstantaneaAutorizacion: lote.InstantaneaAutorizacion,
		CorrelacionRef: lote.CorrelacionRef}
	if err := s.Validar(); err != nil {
		t.Fatal(err)
	}
	return s, lote, ahora
}

func respuestaPreparacionPrueba(s domain.SolicitudPreparacionLoteAdministracionPerfiles, ahora time.Time) map[string]any {
	instante := func(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05.000000Z") }
	return map[string]any{"operacion_ref": s.OperacionRef, "auditoria_ref": "aud_v3_" + strings.Repeat("d", 32),
		"preparada_en": instante(ahora), "persona_ref": s.PersonaRef, "persona_version": 1,
		"cuenta_ref": "cta_" + strings.Repeat("e", 36), "cuenta_version": 1,
		"procedencia_ref": "prc_" + strings.Repeat("f", 32), "procedencia_version": 1,
		"procedencia_huella_sha256": strings.Repeat("a", 64), "organizacion_ref": s.OrganizacionRef, "unidad_ref": s.UnidadRef,
		"altas": []any{map[string]any{"rol_version_ref": "rol:tecnico_rrhh_desarrollo:v1", "nombre": "Técnico",
			"unidad_requerida": false, "vigente_hasta_maxima": instante(ahora.Add(400 * 24 * time.Hour)),
			"duracion_propuesta_segundos": 86400, "perfil_ref": "prf_" + strings.Repeat("1", 32),
			"vinculo_ref": "vca_" + strings.Repeat("2", 32), "huella_sha256": strings.Repeat("3", 64)}},
		"bajas": []any{map[string]any{"rol_version_ref": "rol:llamamiento_desarrollo:v1", "nombre": "Llamamientos",
			"perfil_ref": "prf_" + strings.Repeat("4", 32), "vinculo_ref": "vca_" + strings.Repeat("5", 32),
			"perfil_version": 1, "vinculo_version": 2, "vigente_desde": "2026-01-01T00:00:00Z",
			"vigente_hasta": "2036-01-01T00:00:00Z", "huella_sha256": strings.Repeat("6", 64)}},
		"truncado": false}
}

func autoridadPreparacionPrueba(t *testing.T, cambiar func(map[string]any)) (*AutoridadLoteOrdinario, *poolPreparacionPrueba,
	*emisorPreparacionPrueba, *registroFronteraPrueba, domain.SolicitudPreparacionLoteAdministracionPerfiles) {
	t.Helper()
	s, lote, ahora := solicitudPreparacionPrueba(t)
	respuesta := respuestaPreparacionPrueba(s, ahora)
	if cambiar != nil {
		cambiar(respuesta)
	}
	b := mustJSON(t, respuesta)
	pool := &poolPreparacionPrueba{tx: &txPreparacionPrueba{txFalsa: txFalsa{fila: filaFalsa{dato: b}}}}
	emisor := &emisorPreparacionPrueba{t: t, s: lote, ahora: ahora}
	reg := &registroFronteraPrueba{ahora: ahora}
	a := auditorLotePrueba(reg)
	a.pool, a.emisor, a.proveedor, a.reloj, a.organizacion = pool, emisor, fuenteLoteValidaPrueba{}, relojFijo(ahora), "org_prueba"
	return a, pool, emisor, reg, s
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestPreparacionLoteConsumeConAtributoPropioYDevuelveOpciones(t *testing.T) {
	a, pool, emisor, reg, s := autoridadPreparacionPrueba(t, nil)
	p, err := a.PrepararLoteOrdinario(context.Background(), s)
	if err != nil {
		t.Fatal(err)
	}
	if pool.comienzos != 1 || pool.tx.commits != 1 || pool.tx.sql != prepararLoteOrdinarioSQL || pool.tx.args != 11 || reg.llamadas != 0 {
		t.Fatal("la preparación no usó una transacción con la fachada AUT50")
	}
	_, huella, _ := s.CanonicoYHuella()
	if len(emisor.recurso.Atributos) != 1 || emisor.recurso.Atributos[AtributoPreparacionLote] != huella ||
		emisor.recurso.Ambitos["unidad_ref"] != s.UnidadRef || emisor.efecto.Referencia != s.PersonaRef {
		t.Fatal("recurso de la preparación sin su atributo propio")
	}
	if len(p.Altas) != 1 || len(p.Bajas) != 1 || p.Altas[0].DuracionPropuesta != 24*time.Hour ||
		p.Bajas[0].VinculoVersion != 2 || p.Bajas[0].Nombre != "Llamamientos" || p.CuentaRef == "" || p.ValidarPara(s) != nil {
		t.Fatal("preparación mal traducida")
	}
}

func TestPreparacionLoteRechazaRespuestaAjenaYAudita(t *testing.T) {
	for nombre, cambiar := range map[string]func(map[string]any){
		"otra_persona":    func(r map[string]any) { r["persona_ref"] = "per_" + strings.Repeat("z", 32) },
		"otra_unidad":     func(r map[string]any) { r["unidad_ref"] = "unidad:otra" },
		"campo_extra":     func(r map[string]any) { r["extra"] = true },
		"sin_truncado":    func(r map[string]any) { delete(r, "truncado") },
		"baja_sin_nombre": func(r map[string]any) { delete(r["bajas"].([]any)[0].(map[string]any), "nombre") },
		"duracion_cero":   func(r map[string]any) { r["altas"].([]any)[0].(map[string]any)["duracion_propuesta_segundos"] = 0 },
		"vinculo_repetido": func(r map[string]any) {
			r["bajas"].([]any)[0].(map[string]any)["vinculo_ref"] = "vca_" + strings.Repeat("2", 32)
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			a, pool, _, reg, s := autoridadPreparacionPrueba(t, cambiar)
			p, err := a.PrepararLoteOrdinario(context.Background(), s)
			if err == nil || p.OperacionRef != "" || pool.tx.commits != 0 || pool.tx.rollbacks != 1 || reg.llamadas != 1 ||
				!errors.Is(err, ports.ErrAutoridadAdministracionPerfilesNoDisponible) {
				t.Fatal("respuesta ajena aceptada o sin auditar")
			}
		})
	}
}

func TestPreparacionLoteOtraOrganizacionNoLlegaAlEmisor(t *testing.T) {
	a, pool, emisor, reg, s := autoridadPreparacionPrueba(t, nil)
	s.OrganizacionRef = "org_otra"
	_, err := a.PrepararLoteOrdinario(context.Background(), s)
	if !errors.Is(err, domain.ErrActoAdministracionPerfilesInvalido) || pool.comienzos != 0 ||
		emisor.efecto.Referencia != "" || reg.llamadas != 1 || reg.ordenes[0].Datos.Resultado != domain.ResultadoIntentoAuditoriaDenegado ||
		reg.ordenes[0].Datos.Accion != AccionPreparacionLote {
		t.Fatal("organización ajena no se rechazó antes de emitir")
	}
}
