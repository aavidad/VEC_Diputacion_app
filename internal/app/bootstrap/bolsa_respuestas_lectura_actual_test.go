package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"
	"time"

	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

func escenarioLecturaRespuestasLlamamiento(t *testing.T, ahora time.Time) (contextoSeguridadComunDesarrollo, *fuenteLecturaActualInscripcionPrueba) {
	t.Helper()
	sesion := contextoInscripcionCanalPrueba(t, ahora, false)
	v, err := sesion.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	identidad := &identidadCandidatoBolsaDesarrollo{personaRef: v.PrincipalID,
		perfilRef: v.PerfilActivoRef, candidatoRef: "can_aaaaaaaaaaaaaaaaaaaaaa"}
	instantanea, err := nuevaInstantaneaMiBolsaDesarrollo(identidad, ahora)
	if err != nil {
		t.Fatal(err)
	}
	instantanea.AsignacionPerfil.EmitidaPor = "identidad:prueba-publicada"
	instantanea.VersionRol.PublicadaPor = "seguridad:prueba-publicada"
	instantanea.ControlVigenciaVersionRol.ActualizadoPor = "seguridad:prueba-publicada"
	instantanea.AsignacionPerfil.Ambitos = []vecdomain.AmbitoPerfil{{Clave: "bolsa_ref", Valores: []string{"bolsa:prueba"}}}
	instantanea.VersionRol.Concesiones = []vecdomain.ConcesionRol{{
		Accion: accionLecturaRespuestasLlamamiento, ModuloID: "bolsa", TipoRecurso: "llamamiento_bolsa",
		Finalidades:      []string{finalidadLecturaRespuestasLlamamiento},
		CamposPermitidos: slices.Clone(camposLecturaRespuestasLlamamiento),
		GarantiaMinima:   vecdomain.AuthAssuranceHigh,
	}}
	if err := instantanea.Validar(); err != nil {
		t.Fatal(err)
	}
	return sesion, &fuenteLecturaActualInscripcionPrueba{i: instantanea}
}

func TestCapturarLecturaRespuestasLlamamientoLigaSesionPermisoYRecurso(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	sesion, fuente := escenarioLecturaRespuestasLlamamiento(t, ahora)
	ref := "llamamiento:" + "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	captura, err := CapturarLecturaRespuestasLlamamiento(context.Background(), fuente,
		sesion.Vinculo, sesion.Resultado, "bolsa:prueba", ref)
	if err != nil || len(captura.ContextoCanonico) == 0 || len(captura.VinculoCanonico) == 0 ||
		captura.BolsaRef != "bolsa:prueba" || captura.LlamamientoRef != ref ||
		!captura.ValidaHasta.After(ahora) || captura.ValidaHasta.After(time.Now().Add(30*time.Second)) {
		t.Fatalf("captura nominal ausente: %v", err)
	}
	if fuente.llamadas != 1 || fuente.principal != sesion.Resultado.Contexto.PersonaRef ||
		fuente.perfil != sesion.Resultado.Contexto.PerfilActivoRef {
		t.Fatal("la fuente no se consultó para el actor exacto")
	}
	var orden map[string]any
	if err := json.Unmarshal(captura.Orden, &orden); err != nil ||
		orden["accion"] != accionLecturaRespuestasLlamamiento ||
		orden["recurso_ref"] != ref || orden["bolsa_ref"] != "bolsa:prueba" ||
		orden["finalidad_ref"] != finalidadLecturaRespuestasLlamamiento ||
		orden["resultado"] != "obtenida" || orden["lectura_instantanea_sha256"] == "" {
		t.Fatal("orden de auditoría no ligada al permiso puntual")
	}
	if politicas, ok := orden["politicas_evaluadas"].([]any); !ok || len(politicas) != 0 {
		t.Fatal("el catálogo vacío debe serializarse como una lista vacía")
	}
}

func TestCapturarLecturaRespuestasLlamamientoDeniegaRevocacionYCampoAjeno(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	sesion, fuente := escenarioLecturaRespuestasLlamamiento(t, ahora)
	ref := "llamamiento:" + "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	fuente.i.ControlVigenciaVersionRol.Estado = vecdomain.EstadoControlVigenciaVersionRolRetirada
	fuente.i.ControlVigenciaVersionRol.ActoRef = "acto_retirada"
	fuente.i.ControlVigenciaVersionRol.MotivoCodigo = "retirada"
	if _, err := CapturarLecturaRespuestasLlamamiento(context.Background(), fuente,
		sesion.Vinculo, sesion.Resultado, "bolsa:prueba", ref); !errors.Is(err, vecdomain.ErrAutorizacionDenegada) {
		t.Fatal("rol retirado admitido")
	}
	fuente.i.ControlVigenciaVersionRol.Estado = vecdomain.EstadoControlVigenciaVersionRolHabilitada
	fuente.i.ControlVigenciaVersionRol.ActoRef = ""
	fuente.i.ControlVigenciaVersionRol.MotivoCodigo = ""
	fuente.i.VersionRol.Concesiones[0].CamposPermitidos = append(
		slices.Clone(camposLecturaRespuestasLlamamiento), "contacto_personal")
	if _, err := CapturarLecturaRespuestasLlamamiento(context.Background(), fuente,
		sesion.Vinculo, sesion.Resultado, "bolsa:prueba", ref); !errors.Is(err, vecdomain.ErrAutorizacionDenegada) {
		t.Fatal("concesión con campo ajeno admitida")
	}
}
