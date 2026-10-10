package domain

import (
	"encoding/json"
	"errors"
	"maps"
	"reflect"
	"testing"
	"time"
)

func TestLecturaActualAutorizacionV3ConservaRevisionVentanaYNoTieneEfectos(t *testing.T) {
	vinculo, resultado, instantanea, consulta, ahora := escenarioCapacidadesInformativasV3Prueba(t)
	antes, err := json.Marshal(instantanea)
	if err != nil {
		t.Fatal(err)
	}
	evaluada, err := EvaluarLecturaActualAutorizacionV3(
		vinculo, resultado, instantanea, consulta.Accion, consulta.Recurso,
		consulta.Finalidad, []string{"estado"}, ahora,
	)
	if err != nil || !evaluada.Permitido || evaluada.RevisionCatalogoPoliticas != instantanea.RevisionCatalogoPoliticas ||
		evaluada.RevisionControlVersionRol != instantanea.ControlVigenciaVersionRol.Revision ||
		evaluada.AsignacionRef != instantanea.AsignacionPerfil.Referencia() ||
		evaluada.VersionRolRef != instantanea.VersionRol.Referencia() ||
		evaluada.ControlVersionRolRef != instantanea.ControlVigenciaVersionRol.VersionRolRef ||
		evaluada.AsignacionVersion != instantanea.AsignacionPerfil.Version ||
		evaluada.VersionRolVersion != instantanea.VersionRol.Version ||
		evaluada.RevisionPermisos == 0 ||
		!huellaSHA256AutorizacionV3NoNula(evaluada.HuellaInstantaneaSHA256) ||
		!huellaSHA256AutorizacionV3NoNula(evaluada.AsignacionHuellaSHA256) ||
		!huellaSHA256AutorizacionV3NoNula(evaluada.VersionRolHuellaSHA256) ||
		!huellaSHA256AutorizacionV3NoNula(evaluada.ControlVersionRolHuellaSHA256) ||
		evaluada.CatalogoPoliticasHuellaSHA256 != instantanea.CatalogoPoliticasHuellaSHA256 ||
		len(evaluada.PoliticasEvaluadas) != len(instantanea.Politicas) ||
		!evaluada.EvaluadaEn.Equal(ahora) || !evaluada.ValidaHasta.After(ahora) ||
		evaluada.ValidaHasta.After(ahora.Add(VigenciaMaximaDecisionAutorizacion)) ||
		!reflect.DeepEqual(evaluada.CamposPermitidos, []string{"estado"}) ||
		!reflect.DeepEqual(evaluada.Obligaciones, []string{"auditar_acceso", "registrar_revision"}) {
		t.Fatalf("evaluacion incompleta: %+v, err=%v", evaluada, err)
	}
	despues, err := json.Marshal(instantanea)
	if err != nil || !reflect.DeepEqual(antes, despues) {
		t.Fatal("la evaluacion modificó la instantanea")
	}
	evaluada.CamposPermitidos[0] = "alterado"
	if instantanea.VersionRol.Concesiones[0].CamposPermitidos[0] == "alterado" {
		t.Fatal("el resultado comparte campos con la concesion")
	}
	for n, politica := range evaluada.PoliticasEvaluadas {
		if n > 0 && politica.Referencia <= evaluada.PoliticasEvaluadas[n-1].Referencia {
			t.Fatal("las politicas evaluadas no están ordenadas")
		}
		coincide := false
		for _, original := range instantanea.Politicas {
			huella, err := original.HuellaSHA256()
			if err == nil && politica.Referencia == original.Referencia() && politica.HuellaSHA256 == huella {
				coincide = true
			}
		}
		if !coincide {
			t.Fatalf("politica evaluada incompleta: %+v", politica)
		}
	}
}

func TestLecturaActualAutorizacionV3DeniegaCambiosDeAutoridadYAlcance(t *testing.T) {
	vinculo, resultado, instantanea, consulta, ahora := escenarioCapacidadesInformativasV3Prueba(t)
	type caso struct {
		nombre   string
		preparar func(*InstantaneaAutorizacion, *ConsultaCapacidadInformativaV3, *[]string, *time.Time)
	}
	casos := []caso{
		{"perfil cambiado", func(i *InstantaneaAutorizacion, _ *ConsultaCapacidadInformativaV3, _ *[]string, _ *time.Time) {
			i.AsignacionPerfil.PerfilActivoRef = "perfil_ajeno"
		}},
		{"perfil revocado", func(i *InstantaneaAutorizacion, _ *ConsultaCapacidadInformativaV3, _ *[]string, _ *time.Time) {
			i.AsignacionPerfil.Estado = EstadoAsignacionPerfilRevocada
			i.AsignacionPerfil.RevocadaPor = "usr_seguridad_0123456789"
			i.AsignacionPerfil.RevocadaEn = ahora
			i.AsignacionPerfil.RevocacionRef = "revocacion_prueba"
		}},
		{"rol retirado", func(i *InstantaneaAutorizacion, _ *ConsultaCapacidadInformativaV3, _ *[]string, _ *time.Time) {
			i.ControlVigenciaVersionRol.Estado = EstadoControlVigenciaVersionRolRetirada
			i.ControlVigenciaVersionRol.ActoRef = "acto_retirada"
			i.ControlVigenciaVersionRol.MotivoCodigo = "retirada"
		}},
		{"permiso ausente", func(i *InstantaneaAutorizacion, _ *ConsultaCapacidadInformativaV3, _ *[]string, _ *time.Time) {
			i.VersionRol.Concesiones = append([]ConcesionRol(nil), i.VersionRol.Concesiones...)
			i.VersionRol.Concesiones[0].Accion = "otra_accion"
		}},
		{"campo extra", func(_ *InstantaneaAutorizacion, _ *ConsultaCapacidadInformativaV3, c *[]string, _ *time.Time) {
			*c = []string{"estado", "dato_personal_extra"}
		}},
		{"finalidad ajena", func(_ *InstantaneaAutorizacion, c *ConsultaCapacidadInformativaV3, _ *[]string, _ *time.Time) {
			c.Finalidad = "otra_finalidad"
		}},
		{"ambito ajeno", func(_ *InstantaneaAutorizacion, c *ConsultaCapacidadInformativaV3, _ *[]string, _ *time.Time) {
			c.Recurso.Ambitos = maps.Clone(c.Recurso.Ambitos)
			c.Recurso.Ambitos["unidad"] = "ajena"
		}},
		{"vinculo vencido", func(_ *InstantaneaAutorizacion, _ *ConsultaCapacidadInformativaV3, _ *[]string, instante *time.Time) {
			*instante = resultado.Contexto.Instantanea.VigenteHasta
		}},
	}
	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			i, c, campos, instante := instantanea, consulta, []string{"estado"}, ahora
			tc.preparar(&i, &c, &campos, &instante)
			evaluada, err := EvaluarLecturaActualAutorizacionV3(
				vinculo, resultado, i, c.Accion, c.Recurso, c.Finalidad, campos, instante,
			)
			if !errors.Is(err, ErrAutorizacionDenegada) || evaluada.Permitido ||
				evaluada.RevisionCatalogoPoliticas != 0 || !evaluada.ValidaHasta.IsZero() {
				t.Fatalf("lectura concedida: %+v, err=%v", evaluada, err)
			}
		})
	}
}

func TestLecturaActualAutorizacionV3LigaContenidoAunConRevisionIgual(t *testing.T) {
	vinculo, resultado, instantanea, consulta, ahora := escenarioCapacidadesInformativasV3Prueba(t)
	evaluar := func(i InstantaneaAutorizacion) (ResultadoLecturaActualAutorizacionV3, error) {
		return EvaluarLecturaActualAutorizacionV3(
			vinculo, resultado, i, consulta.Accion, consulta.Recurso, consulta.Finalidad, []string{"estado"}, ahora,
		)
	}
	base, err := evaluar(instantanea)
	if err != nil {
		t.Fatal(err)
	}
	variantes := []struct {
		nombre   string
		preparar func(*InstantaneaAutorizacion)
		huella   func(ResultadoLecturaActualAutorizacionV3) string
	}{
		{"asignacion", func(i *InstantaneaAutorizacion) { i.AsignacionPerfil.EmitidaPor = "otra_autoridad" },
			func(r ResultadoLecturaActualAutorizacionV3) string { return r.AsignacionHuellaSHA256 }},
		{"rol", func(i *InstantaneaAutorizacion) { i.VersionRol.Nombre = "Otro nombre" },
			func(r ResultadoLecturaActualAutorizacionV3) string { return r.VersionRolHuellaSHA256 }},
		{"control", func(i *InstantaneaAutorizacion) { i.ControlVigenciaVersionRol.ActualizadoPor = "otra_autoridad" },
			func(r ResultadoLecturaActualAutorizacionV3) string { return r.ControlVersionRolHuellaSHA256 }},
		{"politica", func(i *InstantaneaAutorizacion) {
			i.Politicas = append([]PoliticaRestrictiva(nil), i.Politicas...)
			i.Politicas[0].Nombre = "Otra minimizacion"
			i.CatalogoPoliticasHuellaSHA256, _ = HuellaCatalogoPoliticasAutorizacion(i.Politicas)
		}, func(r ResultadoLecturaActualAutorizacionV3) string { return r.CatalogoPoliticasHuellaSHA256 }},
	}
	for _, variante := range variantes {
		t.Run(variante.nombre, func(t *testing.T) {
			i := instantanea
			variante.preparar(&i)
			si, err := evaluar(i)
			if err != nil || !si.Permitido || si.AsignacionRef != base.AsignacionRef ||
				si.VersionRolRef != base.VersionRolRef ||
				si.RevisionControlVersionRol != base.RevisionControlVersionRol ||
				si.RevisionCatalogoPoliticas != base.RevisionCatalogoPoliticas ||
				variante.huella(si) == variante.huella(base) ||
				si.HuellaInstantaneaSHA256 == base.HuellaInstantaneaSHA256 ||
				si.RevisionPermisos == base.RevisionPermisos {
				t.Fatalf("contenido no ligado a huella exacta: %+v, err=%v", si, err)
			}
		})
	}
	alterada := instantanea
	alterada.Politicas = append([]PoliticaRestrictiva(nil), alterada.Politicas...)
	alterada.Politicas[0].Nombre = "Otra minimizacion"
	if salida, err := evaluar(alterada); !errors.Is(err, ErrAutorizacionDenegada) || salida.Permitido {
		t.Fatalf("catalogo con huella incongruente aceptado: %+v, %v", salida, err)
	}
}
