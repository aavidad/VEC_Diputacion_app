package postgres

import (
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

func TestRecursoVersionarRolBolsaCierreNoAceptaReferenciaNiAmbitoAjeno(t *testing.T) {
	base, _, _, _ := contratoV2Prueba(t)
	base.InstantaneaAutorizacion.AsignacionPerfil.Ambitos = []domain.AmbitoPerfil{
		{Clave: "organizacion_ref", Valores: []string{"organizacion:uno"}},
		{Clave: "unidad_ref", Valores: []string{"unidad:uno"}},
	}
	asignacion := base.InstantaneaAutorizacion.AsignacionPerfil
	s := domain.SolicitudCierreVersionarRolBolsa{
		OperacionRef:          "cierre_admin:" + strings.Repeat("4", 32),
		PropuestaRef:          "propuesta_admin:" + strings.Repeat("1", 32),
		PropuestaHuellaSHA256: strings.Repeat("a", 64),
		Aprobador:             base.Actor, Evidencia: base.Evidencia,
		InstantaneaAutorizacion: base.InstantaneaAutorizacion,
		Decision:                domain.DecisionAprobarPropuestaPerfil,
		Motivo: domain.ReferenciaEntradaCatalogo{CatalogoID: "motivos_administracion",
			CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("c", 64),
			EntradaClave: "motivo_" + strings.Repeat("a", 32)},
		CorrelacionRef: "correlacion_" + strings.Repeat("5", 32),
	}
	if s.Validar() != nil {
		t.Fatal("fixture de cierre inválido")
	}
	e, err := materialCierreVersionarRolBolsa(s)
	if err != nil {
		t.Fatal(err)
	}
	r, err := RecursoVersionarRolBolsa(e, asignacion)
	if err != nil || r.Referencia != s.PropuestaRef || r.Tipo != "propuesta_definicion_rol" ||
		r.Ambitos["organizacion_ref"] != "organizacion:uno" || r.Ambitos["unidad_ref"] != "unidad:uno" {
		t.Fatalf("recurso B1: %+v %v", r, err)
	}
	for _, caso := range []string{"referencia", "accion", "ambito"} {
		t.Run(caso, func(t *testing.T) {
			x, a := e, asignacion
			switch caso {
			case "referencia":
				x.Referencia = "propuesta_admin:" + strings.Repeat("f", 32)
			case "accion":
				x.Accion = AccionGobiernoRolAprobar
			case "ambito":
				a.Ambitos[0].Valores = []string{"organizacion:uno", "organizacion:dos"}
			}
			if _, err := RecursoVersionarRolBolsa(x, a); err == nil {
				t.Fatal("material o ámbito ajeno aceptado")
			}
		})
	}
}

type emisorVersionarRolBolsaPrueba struct {
	t     *testing.T
	ahora time.Time
}

func (e emisorVersionarRolBolsaPrueba) EmitirVersionarRolBolsa(_ context.Context,
	actor domain.ContextoActor, evidencia domain.EvidenciaSesionAdministracionPerfiles,
	instantanea domain.InstantaneaAutorizacion, efecto Efecto,
) (ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	e.t.Helper()
	recurso, err := RecursoVersionarRolBolsa(efecto, instantanea.AsignacionPerfil)
	if err != nil {
		e.t.Fatal(err)
	}
	h, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		e.t.Fatal(err)
	}
	resumen, err := ports.NuevoResumenCapacidadAtestacionAutorizacionV3(
		"decision:version_bolsa:prueba", strings.Repeat("a", 64), strings.Repeat("b", 64),
		evidencia.ResultadoContexto.RegistroContextoRef, evidencia.ResultadoContexto.HuellaSHA256,
		efecto.Accion, recurso.Referencia, h, efecto.Audiencia, e.ahora, e.ahora.Add(4*time.Second))
	if err != nil {
		e.t.Fatal(err)
	}
	raiz, err := x509.MarshalPKIXPublicKey(ed25519.PublicKey(make([]byte, ed25519.PublicKeySize)))
	if err != nil {
		e.t.Fatal(err)
	}
	return ports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(make([]byte, 512), resumen,
		[]byte("decision"), []byte("motivo"), []byte("contexto"), actor.Instantanea.PersonaVersion,
		actor.Instantanea.PerfilVersion, []byte("payload"), []byte("sobre"), []byte("evidencia"), raiz)
}

func TestVersionarRolBolsaResultadoIncompletoRevierteAntesDeCommit(t *testing.T) {
	ahora := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	base, _, _, _ := contratoV2Prueba(t)
	_, asignacion := escenarioRecursoGobiernoRol(t)
	base.InstantaneaAutorizacion.AsignacionPerfil.Ambitos = asignacion.Ambitos
	s := domain.SolicitudCierreVersionarRolBolsa{
		OperacionRef:          "cierre_admin:" + strings.Repeat("4", 32),
		PropuestaRef:          "propuesta_admin:" + strings.Repeat("1", 32),
		PropuestaHuellaSHA256: strings.Repeat("a", 64),
		Aprobador:             base.Actor, Evidencia: base.Evidencia,
		InstantaneaAutorizacion: base.InstantaneaAutorizacion,
		Decision:                domain.DecisionAprobarPropuestaPerfil,
		Motivo: domain.ReferenciaEntradaCatalogo{CatalogoID: "motivos_administracion",
			CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("c", 64),
			EntradaClave: "motivo_" + strings.Repeat("a", 32)},
		CorrelacionRef: "correlacion_" + strings.Repeat("5", 32),
	}
	if s.Validar() != nil {
		t.Fatal("fixture de cierre inválido")
	}
	e, err := materialCierreVersionarRolBolsa(s)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := RecursoVersionarRolBolsa(e, s.InstantaneaAutorizacion.AsignacionPerfil); err != nil {
		t.Fatalf("fixture de recurso inválido: %v", err)
	}
	b, err := json.Marshal(map[string]any{"estado": "permitido", "replay": false,
		"operacion_ref": s.OperacionRef, "decision": s.Decision,
		"material_canon": "{}", "propuesta_huella_sha256": s.PropuestaHuellaSHA256,
		"confirmado_en": ahora, "auditoria_acceso_ref": "aud_v3_" + strings.Repeat("a", 32),
		"recibo": map[string]any{}})
	if err != nil {
		t.Fatal(err)
	}
	tx := &txGobiernoReferenciaPrueba{fila: filaFalsa{dato: b}}
	a := &AutoridadVersionarRolBolsa{pool: &poolGobiernoReferenciaPrueba{tx: tx},
		catalogo: fuenteCatalogoGobiernoReferenciaPrueba{},
		emisor:   emisorVersionarRolBolsaPrueba{t: t, ahora: ahora}, reloj: relojFijo(ahora)}
	resultado, err := a.CerrarVersionarRolBolsa(context.Background(), s)
	if !errors.Is(err, ports.ErrAutoridadAdministracionPerfilesNoDisponible) ||
		resultado.Recibo != nil || tx.consultas != 1 || tx.commits != 0 || tx.rollbacks != 1 {
		t.Fatalf("respuesta incompleta confirmada: resultado=%+v err=%v consultas=%d commits=%d rollbacks=%d",
			resultado, err, tx.consultas, tx.commits, tx.rollbacks)
	}
}

func TestVersionarRolBolsaDenegacionSQLConfirmaAuditoria(t *testing.T) {
	ahora := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	base, _, _, _ := contratoV2Prueba(t)
	_, asignacion := escenarioRecursoGobiernoRol(t)
	base.InstantaneaAutorizacion.AsignacionPerfil.Ambitos = asignacion.Ambitos
	s := domain.SolicitudCierreVersionarRolBolsa{
		OperacionRef:          "cierre_admin:" + strings.Repeat("4", 32),
		PropuestaRef:          "propuesta_admin:" + strings.Repeat("1", 32),
		PropuestaHuellaSHA256: strings.Repeat("a", 64),
		Aprobador:             base.Actor, Evidencia: base.Evidencia,
		InstantaneaAutorizacion: base.InstantaneaAutorizacion,
		Decision:                domain.DecisionAprobarPropuestaPerfil,
		Motivo: domain.ReferenciaEntradaCatalogo{CatalogoID: "motivos_administracion",
			CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("c", 64),
			EntradaClave: "motivo_" + strings.Repeat("a", 32)},
		CorrelacionRef: "correlacion_" + strings.Repeat("5", 32),
	}
	e, err := materialCierreVersionarRolBolsa(s)
	if err != nil {
		t.Fatal(err)
	}
	for _, caso := range []struct {
		estado, codigo string
		denegado       bool
	}{
		{"denegado", "version_rol_bolsa_denegado", true},
		{"error", "version_rol_bolsa_error", false},
	} {
		t.Run(caso.estado, func(t *testing.T) {
			b, err := json.Marshal(map[string]any{"estado": caso.estado,
				"codigo":            caso.codigo,
				"auditoria_intento": map[string]any{"auditoria_ref": "aud_v3_" + strings.Repeat("a", 32)}})
			if err != nil {
				t.Fatal(err)
			}
			tx := &txGobiernoReferenciaPrueba{fila: filaFalsa{dato: b}}
			a := &AutoridadVersionarRolBolsa{pool: &poolGobiernoReferenciaPrueba{tx: tx},
				catalogo: fuenteCatalogoGobiernoReferenciaPrueba{},
				emisor:   emisorVersionarRolBolsaPrueba{t: t, ahora: ahora}, reloj: relojFijo(ahora)}
			err = a.ejecutar(context.Background(), s.Aprobador, s.Evidencia, s.InstantaneaAutorizacion,
				e, cerrarVersionarRolBolsaSQL, func([]byte) error { t.Fatal("fallo validado como éxito"); return nil })
			if !errors.Is(err, ports.ErrGobiernoRolIntentoAuditado) ||
				errors.Is(err, domain.ErrAutorizacionDenegada) != caso.denegado ||
				tx.consultas != 1 || tx.commits != 1 {
				t.Fatalf("fallo perdió auditoría: err=%v consultas=%d commits=%d", err, tx.consultas, tx.commits)
			}
		})
	}
}
