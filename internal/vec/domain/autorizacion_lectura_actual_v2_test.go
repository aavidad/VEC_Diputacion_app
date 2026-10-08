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
