package postgres

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

func TestRecursoGobiernoInscripcionLigaAprobadorYAsignacion(t *testing.T) {
	desde := time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC)
	asignacion := domain.AsignacionPerfil{
		AsignacionID: "bootstrap_" + strings.Repeat("a", 32), Version: 7,
		PerfilActivoRef: "prf_" + strings.Repeat("b", 24), PrincipalID: "per_" + strings.Repeat("c", 24),
		VersionRolRef: "rol:administracion_perfiles:v10", Estado: domain.EstadoAsignacionPerfilActiva,
		Ambitos: []domain.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{"organizacion:sintetica"}},
			{Clave: "unidad_ref", Valores: []string{"unidad:rrhh:sintetica"}}},
		VigenteDesde: desde, VigenteHasta: desde.Add(48 * time.Hour),
		EmitidaPor: "actor:admin:sintetico", EmitidaEn: desde,
	}
	correlacion := "correlacion_" + strings.Repeat("d", 32)
	propuesta := "propuesta_admin:" + strings.Repeat("e", 32)
	sobre := cierreGobiernoInscripcionEnvelope{
		Esquema:      "administracion_version_inscripcion_cierre_v1",
		OperacionRef: "cierre_admin:" + strings.Repeat("f", 32),
		PropuestaRef: propuesta, PropuestaHuellaSHA256: strings.Repeat("1", 64),
		Decision:        domain.DecisionAprobarPropuestaPerfil,
		ActorPersonaRef: asignacion.PrincipalID, ActorPerfilRef: asignacion.PerfilActivoRef,
		AsignacionRef: asignacion.Referencia(), CorrelacionRef: correlacion,
		Motivo: domain.ReferenciaEntradaCatalogo{CatalogoID: "motivos_administracion", CatalogoVersion: 1,
			CatalogoHuellaSHA256: strings.Repeat("2", 64), EntradaClave: "motivo_" + strings.Repeat("3", 32)},
	}
	b, err := json.Marshal(sobre)
	if err != nil {
		t.Fatal(err)
	}
	e := EfectoGobiernoInscripcion{Accion: AccionGobiernoInscripcionAprobar,
		Audiencia: AudienciaGobiernoInscripcionAprobar, RecursoRef: propuesta,
		CorrelacionRef: correlacion, Material: b}
	recurso, err := RecursoGobiernoInscripcion(e, asignacion)
	if err != nil || recurso.Referencia != propuesta || recurso.Tipo != "propuesta_definicion_rol" ||
		len(recurso.Ambitos) != 2 || recurso.Atributos["material_sha256"] == "" {
		t.Fatalf("recurso cerrado: %+v %v", recurso, err)
	}
	contexto := `{"ambitos":{"organizacion_ref":"organizacion:sintetica","unidad_ref":"unidad:rrhh:sintetica"},"atributos":{"material_sha256":"` +
		recurso.Atributos["material_sha256"] + `"}}`
	hCtx := sha256.Sum256([]byte(contexto))
	hDominio, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil || hDominio != hex.EncodeToString(hCtx[:]) {
		t.Fatal("la huella V3 no coincide con el canon de contexto SQL")
	}
	t.Logf("material_sha256=%s contexto_sha256=%s", recurso.Atributos["material_sha256"], hDominio)

	sobre.ActorPersonaRef = "per_" + strings.Repeat("9", 24)
	e.Material, _ = json.Marshal(sobre)
	if _, err := RecursoGobiernoInscripcion(e, asignacion); err == nil {
		t.Fatal("se aceptó otro aprobador para la asignación ADMIN")
	}
	sobre.ActorPersonaRef = asignacion.PrincipalID
	e.Material, _ = json.Marshal(sobre)
	asignacion.Ambitos = append(asignacion.Ambitos, domain.AmbitoPerfil{Clave: "ambito_ref", Valores: []string{"otro"}})
	if _, err := RecursoGobiernoInscripcion(e, asignacion); err == nil {
		t.Fatal("se aceptó ampliar el ámbito del recurso ADMIN")
	}
}
