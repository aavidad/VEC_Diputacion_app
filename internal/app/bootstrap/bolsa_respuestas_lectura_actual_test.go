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
	instantanea, err := nuevaInstantaneaAutorizacionBorradorLlamamientoBolsaDesarrolloVersion(
		v.PrincipalID, v.PerfilActivoRef, "unidad:prueba", "ambito:prueba", ahora, 5)
	if err != nil {
		t.Fatal(err)
	}
	instantanea.AsignacionPerfil.EmitidaPor = "identidad:prueba-publicada"
	instantanea.VersionRol.PublicadaPor = "seguridad:prueba-publicada"
	instantanea.ControlVigenciaVersionRol.ActualizadoPor = "seguridad:prueba-publicada"
	instantanea.VersionRol.Concesiones = append(instantanea.VersionRol.Concesiones, vecdomain.ConcesionRol{
		Accion: accionLecturaRespuestasLlamamiento, ModuloID: "bolsa", TipoRecurso: "llamamiento_bolsa",
		Finalidades:      []string{finalidadLecturaRespuestasLlamamiento},
		CamposPermitidos: slices.Clone(camposLecturaRespuestasLlamamiento),
		GarantiaMinima:   vecdomain.AuthAssuranceHigh,
	})
	if err := instantanea.Validar(); err != nil {
		t.Fatal(err)
	}
	return sesion, &fuenteLecturaActualInscripcionPrueba{i: instantanea}
}

func TestCapturarLecturaRespuestasLlamamientoLigaSesionPermisoYRecurso(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	sesion, fuente := escenarioLecturaRespuestasLlamamiento(t, ahora)
	ref := "llamamiento:" + "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	for n, bolsaRef := range []string{"bolsa:prueba", "bolsa:otra"} {
		captura, err := CapturarLecturaRespuestasLlamamiento(context.Background(), fuente,
			sesion.Vinculo, sesion.Resultado, "unidad:prueba", "ambito:prueba", bolsaRef, ref)
		if err != nil || len(captura.ContextoCanonico) == 0 || len(captura.VinculoCanonico) == 0 ||
			captura.BolsaRef != bolsaRef || captura.LlamamientoRef != ref ||
			!captura.ValidaHasta.After(ahora) || captura.ValidaHasta.After(time.Now().Add(30*time.Second)) {
			t.Fatalf("captura nominal ausente para %s: %v", bolsaRef, err)
		}
		if fuente.llamadas != n+1 || fuente.principal != sesion.Resultado.Contexto.PersonaRef ||
			fuente.perfil != sesion.Resultado.Contexto.PerfilActivoRef {
			t.Fatal("la fuente no se consultó para el actor exacto")
		}
		var orden map[string]any
		if err := json.Unmarshal(captura.Orden, &orden); err != nil ||
			orden["accion"] != accionLecturaRespuestasLlamamiento ||
			orden["recurso_ref"] != ref || orden["bolsa_ref"] != bolsaRef ||
			orden["unidad_ref"] != "unidad:prueba" || orden["ambito_ref"] != "ambito:prueba" ||
			orden["finalidad_ref"] != finalidadLecturaRespuestasLlamamiento ||
			orden["resultado"] != "obtenida" || orden["lectura_instantanea_sha256"] == "" {
			t.Fatal("orden de auditoría no ligada al permiso puntual")
		}
		if politicas, ok := orden["politicas_evaluadas"].([]any); !ok || len(politicas) != 0 {
			t.Fatal("el catálogo vacío debe serializarse como una lista vacía")
		}
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
		sesion.Vinculo, sesion.Resultado, "unidad:prueba", "ambito:prueba", "bolsa:prueba", ref); !errors.Is(err, vecdomain.ErrAutorizacionDenegada) {
		t.Fatal("rol retirado admitido")
	}
	fuente.i.ControlVigenciaVersionRol.Estado = vecdomain.EstadoControlVigenciaVersionRolHabilitada
	fuente.i.ControlVigenciaVersionRol.ActoRef = ""
	fuente.i.ControlVigenciaVersionRol.MotivoCodigo = ""
	fuente.i.VersionRol.Concesiones[len(fuente.i.VersionRol.Concesiones)-1].CamposPermitidos = append(
		slices.Clone(camposLecturaRespuestasLlamamiento), "contacto_personal")
	if _, err := CapturarLecturaRespuestasLlamamiento(context.Background(), fuente,
		sesion.Vinculo, sesion.Resultado, "unidad:prueba", "ambito:prueba", "bolsa:prueba", ref); !errors.Is(err, vecdomain.ErrAutorizacionDenegada) {
		t.Fatal("concesión con campo ajeno admitida")
	}
}

func TestCapturarLecturaRespuestasLlamamientoDeniegaAmbitoAjenoYRolAusente(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	sesion, fuente := escenarioLecturaRespuestasLlamamiento(t, ahora)
	ref := "llamamiento:" + "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if _, err := CapturarLecturaRespuestasLlamamiento(context.Background(), fuente,
		sesion.Vinculo, sesion.Resultado, "unidad:ajena", "ambito:prueba", "bolsa:prueba", ref); !errors.Is(err, vecdomain.ErrAutorizacionDenegada) {
		t.Fatal("unidad ajena admitida")
	}
	if _, err := CapturarLecturaRespuestasLlamamiento(context.Background(), fuente,
		sesion.Vinculo, sesion.Resultado, "unidad:prueba", "ambito:ajeno", "bolsa:prueba", ref); !errors.Is(err, vecdomain.ErrAutorizacionDenegada) {
		t.Fatal("ámbito ajeno admitido")
	}
	fuente.i.VersionRol.Concesiones = fuente.i.VersionRol.Concesiones[:len(fuente.i.VersionRol.Concesiones)-1]
	if _, err := CapturarLecturaRespuestasLlamamiento(context.Background(), fuente,
		sesion.Vinculo, sesion.Resultado, "unidad:prueba", "ambito:prueba", "bolsa:prueba", ref); !errors.Is(err, vecdomain.ErrAutorizacionDenegada) {
		t.Fatal("perfil sin concesión B103 admitido")
	}
	fuente.i.VersionRol.Concesiones = append(fuente.i.VersionRol.Concesiones, vecdomain.ConcesionRol{
		Accion: accionLecturaRespuestasLlamamiento, ModuloID: "bolsa", TipoRecurso: "llamamiento_bolsa",
		Finalidades:      []string{finalidadLecturaRespuestasLlamamiento},
		CamposPermitidos: slices.Clone(camposLecturaRespuestasLlamamiento),
		GarantiaMinima:   vecdomain.AuthAssuranceHigh,
	})
	fuente.i.VersionRol.RolID = "otro_rol"
	fuente.i.AsignacionPerfil.VersionRolRef = fuente.i.VersionRol.Referencia()
	fuente.i.ControlVigenciaVersionRol.VersionRolRef = fuente.i.VersionRol.Referencia()
	if _, err := CapturarLecturaRespuestasLlamamiento(context.Background(), fuente,
		sesion.Vinculo, sesion.Resultado, "unidad:prueba", "ambito:prueba", "bolsa:prueba", ref); !errors.Is(err, vecdomain.ErrAutorizacionDenegada) {
		t.Fatal("otro rol con concesión copiada admitido")
	}
}
