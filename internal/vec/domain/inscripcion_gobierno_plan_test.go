package domain

import (
	"strings"
	"testing"
	"time"
)

func planEmpleadoInscripcionPrueba(t *testing.T) (PlanVersionInscripcion, CatalogoAccionesAdministracionV1, time.Time) {
	t.Helper()
	ahora := time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC)
	desde, hasta := ahora.Add(-time.Hour), ahora.Add(time.Hour)
	_, acciones, _ := contratoPerfilInscripcion(PerfilVersionInscripcionEmpleado)
	concesiones := make([]ConcesionRol, 0, len(acciones))
	entradas := make([]EntradaAccionAdministracionV1, 0, len(acciones))
	selecciones := make([]SeleccionAccionAdministracionV1, 0, len(acciones))
	for i, accion := range acciones {
		finalidad, tipo := "consulta_convocatoria_abierta", "convocatoria_inscripcion"
		switch i {
		case 2, 3:
			finalidad, tipo = "consulta_inscripcion_propia", "solicitud_inscripcion"
		case 4:
			finalidad, tipo = "presentar_inscripcion", "inscripcion_convocatoria"
		}
		concesionExterna := ConcesionRol{Accion: accion, ModuloID: "bolsa", TipoRecurso: tipo,
			Finalidades: []string{finalidad}, GarantiaMinima: AuthAssuranceHigh}
		if i < 4 {
			concesionExterna.CamposPermitidos = []string{"resumen_inscripcion"}
		}
		externa := EntradaAccionAdministracionV1{Referencia: "accion:externa:" + string(rune('a'+i)), Version: 1,
			FuenteRef: "fuente:bolsa:inscripcion", FuenteVersion: 1, FuenteHuellaSHA256: strings.Repeat("b", 64),
			Concesion: concesionExterna, DimensionesAmbito: []string{"candidato_ref"}, ClaseControl: "ordinario",
			VigenteDesde: desde, VigenteHasta: hasta}
		entradas = append(entradas, externa)
		concesion := concesionExterna
		concesion.TipoRecurso += "_empleado"
		concesiones = append(concesiones, concesion)
		entrada := EntradaAccionAdministracionV1{Referencia: "accion:empleado:" + string(rune('a'+i)), Version: 1,
			FuenteRef: "fuente:bolsa:inscripcion", FuenteVersion: 1, FuenteHuellaSHA256: strings.Repeat("b", 64),
			Concesion: concesion, DimensionesAmbito: []string{"empleado_ref"}, ClaseControl: "ordinario",
			VigenteDesde: desde, VigenteHasta: hasta}
		he, err := entrada.HuellaSHA256()
		if err != nil {
			t.Fatal(err)
		}
		entradas = append(entradas, entrada)
		selecciones = append(selecciones, SeleccionAccionAdministracionV1{EntradaRef: entrada.Referencia,
			EntradaVersion: entrada.Version, EntradaHuellaSHA256: he})
	}
	catalogo := CatalogoAccionesAdministracionV1{Referencia: "catalogo:inscripcion:sintetico", Version: 1,
		FuenteRef: "fuente:inscripcion:sintetica", FuenteVersion: 1, FuenteHuellaSHA256: strings.Repeat("a", 64),
		VigenteDesde: desde, VigenteHasta: hasta, Entradas: entradas}
	hc, err := catalogo.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	empleadoRef := "emp_" + strings.Repeat("e", 24)
	plan := PlanVersionInscripcion{Operacion: OperacionCrearPerfilGobernado,
		PerfilObjetivo: PerfilVersionInscripcionEmpleado, CatalogoRef: catalogo.Referencia,
		CatalogoVersion: catalogo.Version, CatalogoHuellaSHA256: hc,
		VersionRolObjetivoRef: "rol:" + RolInscripcionEmpleado + ":v1",
		DefinicionNueva: DefinicionVersionPerfilGobernado{RolID: RolInscripcionEmpleado, Version: 1,
			Nombre: "Bolsa empleado sintético", Concesiones: concesiones}, Selecciones: selecciones,
		Asignaciones: []CambioAsignacionInscripcion{{Almacen: "normal", Modo: "alta",
			AsignacionID: "asg_" + strings.Repeat("1", 24), PrincipalID: "per_" + strings.Repeat("2", 24),
			PerfilActivoRef: "prf_" + strings.Repeat("3", 24),
			Ambitos:         []AmbitoPerfil{{Clave: "empleado_ref", Valores: []string{empleadoRef}}},
			VigenteDesde:    desde, VigenteHasta: hasta, EmpleadoRef: empleadoRef,
			ProyeccionEmpleado: &ProyeccionEmpleadoInscripcion{ProyeccionRef: "pep_" + strings.Repeat("4", 24),
				Version: 1, ProcedenciaRef: "fuente:personal:sintetica", ProcedenciaVersion: 1,
				ProcedenciaHuellaSHA256: strings.Repeat("c", 64)}}},
		Motivo: ReferenciaEntradaCatalogo{CatalogoID: "motivos_administracion", CatalogoVersion: 1,
			CatalogoHuellaSHA256: strings.Repeat("d", 64), EntradaClave: "motivo_" + strings.Repeat("f", 32)}}
	return plan, catalogo, ahora
}

func TestPlanVersionInscripcionExigeFuenteYAmbitoEmpleadoExactos(t *testing.T) {
	plan, catalogo, ahora := planEmpleadoInscripcionPrueba(t)
	if err := plan.ValidarContraCatalogo(catalogo, ahora); err != nil {
		t.Fatalf("plan administrado válido: %v", err)
	}
	if _, err := plan.HuellaSHA256(); err != nil {
		t.Fatal(err)
	}

	alterado := plan
	alterado.DefinicionNueva.Concesiones = append([]ConcesionRol(nil), plan.DefinicionNueva.Concesiones...)
	alterado.DefinicionNueva.Concesiones[0].CamposPermitidos = []string{"expediente_personal_completo"}
	if alterado.ValidarContraCatalogo(catalogo, ahora) == nil {
		t.Fatal("se aceptó una ampliación de campos ajena a la fuente")
	}

	alterado = plan
	alterado.Asignaciones = append([]CambioAsignacionInscripcion(nil), plan.Asignaciones...)
	alterado.Asignaciones[0].Ambitos = []AmbitoPerfil{{Clave: "categoria_ref", Valores: []string{"categoria:sintetica"}}}
	if alterado.ValidarContraCatalogo(catalogo, ahora) == nil {
		t.Fatal("se aceptó categoría de convocatoria como ámbito personal")
	}
}
