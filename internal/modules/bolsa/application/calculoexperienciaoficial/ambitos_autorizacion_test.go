package calculoexperienciaoficial

import (
	"testing"
	"time"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func TestAsignacionPerfilRealCubreSoloSujetoYConvocatoriaExactos(t *testing.T) {
	escenario := nuevoEscenarioServicioPrueba(t, perfilInternoAlto, false)
	lectura, err := recursoLectura(escenario.datosOrden.Selector)
	if err != nil {
		t.Fatal(err)
	}
	preparado, err := prepararCalculo(escenario.datosOrden, escenario.fuente.resultado)
	if err != nil {
		t.Fatal(err)
	}
	escritura, err := recursoEscritura(preparado.intencion)
	if err != nil {
		t.Fatal(err)
	}
	asignacion := dominiovec.AsignacionPerfil{
		AsignacionID: "asignacion_calculo_oficial", Version: 1,
		PerfilActivoRef: escenario.datosOrden.ContextoActor.PerfilActivoRef,
		PrincipalID:     escenario.datosOrden.ContextoActor.Principal.ID,
		VersionRolRef:   "rol:calculo:oficial:v1",
		Estado:          dominiovec.EstadoAsignacionPerfilActiva,
		Ambitos: []dominiovec.AmbitoPerfil{
			{Clave: "sujeto_ref", Valores: []string{
				escenario.datosOrden.Selector.SujetoPseudonimo.Referencia(),
			}},
			{Clave: "convocatoria_ref", Valores: []string{
				escenario.datosOrden.Selector.Convocatoria.Referencia(),
			}},
		},
		EmitidaPor: "principal:seguridad", EmitidaEn: escenario.ahora.Add(-2 * time.Hour),
		VigenteDesde: escenario.ahora.Add(-time.Hour),
		VigenteHasta: escenario.ahora.Add(time.Hour),
	}
	if asignacion.Validar() != nil || !asignacion.Cubre(lectura) || !asignacion.Cubre(escritura) {
		t.Fatal("la asignacion exacta no cubre ambos recursos")
	}

	casos := map[string]dominiovec.AsignacionPerfil{
		"sin_asignacion": {},
		"falta_convocatoria": copiarAsignacionConAmbitos(
			asignacion, asignacion.Ambitos[:1],
		),
		"sujeto_ajeno": copiarAsignacionConAmbitos(asignacion, []dominiovec.AmbitoPerfil{
			{Clave: "sujeto_ref", Valores: []string{
				"hmac-sha256:seudonimo_ajeno_v1:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			}},
			asignacion.Ambitos[1],
		}),
		"convocatoria_ajena": copiarAsignacionConAmbitos(asignacion, []dominiovec.AmbitoPerfil{
			asignacion.Ambitos[0],
			{Clave: "convocatoria_ref", Valores: []string{"convocatoria:ajena:v1"}},
		}),
		"comodin": copiarAsignacionConAmbitos(asignacion, []dominiovec.AmbitoPerfil{
			{Clave: "sujeto_ref", Valores: []string{"*"}}, asignacion.Ambitos[1],
		}),
	}
	for nombre, candidata := range casos {
		t.Run(nombre, func(t *testing.T) {
			if candidata.Cubre(lectura) || candidata.Cubre(escritura) {
				t.Fatal("una asignacion no exacta cubrio un recurso oficial")
			}
		})
	}
}

func copiarAsignacionConAmbitos(
	base dominiovec.AsignacionPerfil,
	ambitos []dominiovec.AmbitoPerfil,
) dominiovec.AsignacionPerfil {
	base.Ambitos = append([]dominiovec.AmbitoPerfil(nil), ambitos...)
	return base
}
