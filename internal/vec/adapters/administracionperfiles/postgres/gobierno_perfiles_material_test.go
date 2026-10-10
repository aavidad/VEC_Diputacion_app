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

func escenarioRecursoGobiernoRol(t *testing.T) (Efecto, domain.AsignacionPerfil) {
	t.Helper()
	ahora := time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC)
	motivo := domain.ReferenciaEntradaCatalogo{CatalogoID: "motivos_administracion", CatalogoVersion: 1,
		CatalogoHuellaSHA256: strings.Repeat("a", 64), EntradaClave: "motivo_" + strings.Repeat("b", 32)}
	concesion := domain.ConcesionRol{Accion: "bolsa.carga_convoca.confirmar", ModuloID: "bolsa", TipoRecurso: "carga_convoca",
		Finalidades: []string{"cargar_bolsa"}, GarantiaMinima: domain.AuthAssuranceHigh}
	plan := domain.PlanGobiernoPerfil{Operacion: domain.OperacionCrearPerfilGobernado,
		CatalogoRef: "catalogo:acciones", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("c", 64),
		VersionRolObjetivoRef: "rol:nuevo_sintetico:v1", DefinicionNueva: &domain.DefinicionVersionPerfilGobernado{
			RolID: "nuevo_sintetico", Version: 1, Nombre: "Nuevo sintético", Concesiones: []domain.ConcesionRol{concesion}},
		Selecciones: []domain.SeleccionAccionAdministracionV1{{EntradaRef: "entrada:sintetica", EntradaVersion: 1,
			EntradaHuellaSHA256: strings.Repeat("d", 64)}}, Motivo: motivo}
	m := domain.MaterialPropuestaGobiernoPerfil{OperacionRef: "propuesta_admin:" + strings.Repeat("e", 32),
		ProponentePersonaRef: "per_" + strings.Repeat("f", 24), PerfilActivoRef: "prf_" + strings.Repeat("g", 24),
		AsignacionPerfilRef: "asignacion:admin:v5", Plan: plan}
	canon, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	hm, err := m.HuellaSHA256()
	if err != nil {
		t.Fatalf("material prueba inválido: %v", err)
	}
	hp, err := plan.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(propuestaGobiernoRolEnvelope{Esquema: "administracion_gobierno_rol_nuevo_propuesta_v1",
		MaterialCanon: string(canon), MaterialSHA256: hm, PlanSHA256: hp,
		CorrelacionRef: "correlacion_" + strings.Repeat("1", 32)})
	if err != nil {
		t.Fatal(err)
	}
	asignacion := domain.AsignacionPerfil{AsignacionID: "admin", Version: 5,
		PerfilActivoRef: "prf_" + strings.Repeat("g", 24), PrincipalID: "per_" + strings.Repeat("f", 24),
		VersionRolRef: "rol:administracion_perfiles:v8", Estado: domain.EstadoAsignacionPerfilActiva,
		Ambitos: []domain.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{"organizacion:uno"}},
			{Clave: "unidad_ref", Valores: []string{"unidad:uno"}}},
		EmitidaPor: "mantenimiento_operador:prueba", EmitidaEn: ahora,
		VigenteDesde: ahora, VigenteHasta: ahora.Add(24 * time.Hour)}
	if err := asignacion.Validar(); err != nil {
		t.Fatalf("asignación prueba inválida: %v", err)
	}
	return Efecto{Accion: AccionGobiernoRolProponer, Audiencia: AudienciaGobiernoRolProponer,
		Referencia: plan.VersionRolObjetivoRef, Material: b,
		CorrelacionAccesoRef: "correlacion_" + strings.Repeat("1", 32)}, asignacion
}

func TestRecursoGobiernoRolNuevoLigaPlanYAsignacionSinAmpliarAmbito(t *testing.T) {
	e, a := escenarioRecursoGobiernoRol(t)
	r, err := RecursoGobiernoRolNuevo(e, a)
	if err != nil || r.Referencia != "rol:nuevo_sintetico:v1" || r.Tipo != "definicion_rol" ||
		r.Ambitos["organizacion_ref"] != "organizacion:uno" || r.Ambitos["unidad_ref"] != "unidad:uno" {
		t.Fatalf("recurso exacto: %+v %v", r, err)
	}
	h := sha256.Sum256(e.Material)
	if r.Atributos[atributoMaterialGobiernoRol] != hex.EncodeToString(h[:]) {
		t.Fatal("material no ligado al contexto V3")
	}
	for _, caso := range []string{"rol_ajeno", "material_mutado", "ambito_multiple", "accion_asignacion"} {
		t.Run(caso, func(t *testing.T) {
			otroE, otraA := e, a
			switch caso {
			case "rol_ajeno":
				otroE.Referencia = "rol:otro:v1"
			case "material_mutado":
				var p propuestaGobiernoRolEnvelope
				if err := json.Unmarshal(e.Material, &p); err != nil {
					t.Fatal(err)
				}
				p.MaterialSHA256 = strings.Repeat("0", 64)
				otroE.Material, _ = json.Marshal(p)
			case "ambito_multiple":
				otraA.Ambitos = []domain.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{"organizacion:uno", "organizacion:dos"}},
					{Clave: "unidad_ref", Valores: []string{"unidad:uno"}}}
			case "accion_asignacion":
				otroE.Accion = "administracion.perfiles.otorgar"
			}
			if r, err := RecursoGobiernoRolNuevo(otroE, otraA); err == nil || r.Referencia != "" {
				t.Fatal("variante no publicada alcanzó recurso V3")
			}
		})
	}
}
