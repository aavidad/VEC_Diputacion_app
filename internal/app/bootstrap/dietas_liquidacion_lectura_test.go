package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	dietascomp "vec-diputacion-granada/internal/modules/dietas/adapters/composicion"
	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type identidadLecturaLiquidacionPrueba struct {
	sesion contextoSeguridadComunDesarrollo
}

func (i identidadLecturaLiquidacionPrueba) ResolverIdentidadRegistradaBorrador(context.Context) (dietascomp.IdentidadRegistradaBorrador, error) {
	return dietascomp.IdentidadRegistradaBorrador{Contexto: i.sesion.Resultado, Vinculo: i.sesion.Vinculo}, nil
}

func TestCapturaLecturaLiquidacionLigaPermisoActualY33Claves(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	sesion, fuente := escenarioLecturaRespuestasLlamamiento(t, ahora)
	persona := sesion.Resultado.Contexto.PersonaRef
	fuente.i.AsignacionPerfil.Ambitos = []vecdomain.AmbitoPerfil{
		{Clave: "persona_ref", Valores: []string{persona}},
		{Clave: "unidad_ref", Valores: []string{"unidad:prueba"}},
	}
	fuente.i.VersionRol.Concesiones = append(fuente.i.VersionRol.Concesiones, vecdomain.ConcesionRol{
		Accion: "dietas.liquidacion_economica.consultar", ModuloID: "dietas", TipoRecurso: "liquidacion_economica",
		Finalidades:      []string{"consultar_liquidacion_economica_dietas"},
		CamposPermitidos: slices.Clone(camposLiquidacionEconomica), GarantiaMinima: vecdomain.AuthAssuranceHigh,
	})
	if err := fuente.i.Validar(); err != nil {
		t.Fatal(err)
	}
	c := &capturadorLecturaLiquidacion{identidad: identidadLecturaLiquidacionPrueba{sesion}, fuente: fuente}
	ref := "dco_" + strings.Repeat("a", 22)
	captura, err := c.CapturarLecturaLiquidacionEconomica(context.Background(), ref)
	if err != nil {
		t.Fatal(err)
	}
	var orden map[string]json.RawMessage
	if err := json.Unmarshal(captura.Orden, &orden); err != nil {
		t.Fatal(err)
	}
	if len(orden) != 33 || orden["unidad_ref"] != nil || orden["titular_persona_ref"] != nil {
		t.Fatalf("claves: %d", len(orden))
	}
	var accion, motivo, recurso string
	if json.Unmarshal(orden["accion"], &accion) != nil || json.Unmarshal(orden["motivo_ref"], &motivo) != nil ||
		json.Unmarshal(orden["recurso_ref"], &recurso) != nil || accion != "dietas.liquidacion_economica.consultar" ||
		motivo != motivoAuditoriaLecturaLiquidacionEconomica || recurso != ref || len(captura.ContextoCanonico) == 0 || len(captura.VinculoCanonico) == 0 {
		t.Fatal("captura desalineada")
	}
	fuente.i.AsignacionPerfil.Ambitos = append(fuente.i.AsignacionPerfil.Ambitos,
		vecdomain.AmbitoPerfil{Clave: "empleado_ref", Valores: []string{"emp_" + strings.Repeat("b", 22)}})
	if _, err := c.CapturarLecturaLiquidacionEconomica(context.Background(), ref); !errors.Is(err, dietasports.ErrAccesoCircuitoDenegado) {
		t.Fatalf("empleado sin vínculo registrado admitido: %v", err)
	}
	fuente.i.AsignacionPerfil.Ambitos = fuente.i.AsignacionPerfil.Ambitos[:2]
	fuente.i.VersionRol.Concesiones = fuente.i.VersionRol.Concesiones[:len(fuente.i.VersionRol.Concesiones)-1]
	if _, err := c.CapturarLecturaLiquidacionEconomica(context.Background(), ref); !errors.Is(err, dietasports.ErrAccesoCircuitoDenegado) {
		t.Fatalf("permiso retirado: %v", err)
	}
}

func TestCapturaLecturaLiquidacionCotejaEmpleadoCanonicoYAmbitosExactos(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	sesion, fuente := escenarioLecturaRespuestasLlamamiento(t, ahora)
	registrado := resultadoConVinculoContextoPrueba(t, sesion.Resultado, 1,
		vecdomain.TipoReferenciaContextoActorEmpleado, "emp_", "dietas-empleado")
	datos, err := sesion.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	autenticacion := vecdomain.AutenticacionRevalidadaV1{
		AutenticacionRef: datos.AutenticacionRef, AutenticacionHuellaSHA256: datos.AutenticacionHuellaSHA256,
		AsercionRef: datos.AsercionRef, SesionRef: datos.SesionRef, ControlSesionRef: datos.ControlSesionRef,
		ControlSesionRevision: datos.ControlSesionRevision, ControlSesionHuellaSHA256: datos.ControlSesionHuellaSHA256,
		CuentaRef: datos.CuentaRef, CuentaOrdinariaRef: datos.CuentaOrdinariaRef,
		CuentaPrivilegiada: datos.CuentaPrivilegiada, Superficie: datos.Superficie,
		MetodoObservado: datos.MetodoObservado, GarantiaObservada: datos.GarantiaObservada,
		PoliticaGarantiaRef:          datos.PoliticaGarantiaRef,
		PoliticaGarantiaHuellaSHA256: datos.PoliticaGarantiaHuellaSHA256,
		AutenticacionVerificadaEn:    datos.AutenticacionVerificadaEn, SesionEmitidaEn: datos.SesionEmitidaEn,
		SesionValidaHasta: datos.SesionValidaHasta, SesionRevalidadaEn: datos.SesionRevalidadaEn,
	}
	vinculo, resuelto, err := vecdomain.CrearVinculoAutenticacionActorV2ConResultado(context.Background(),
		revalidadorAutenticacionAltaContratacionTemporalDesarrollo{valor: autenticacion},
		vecdomain.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: datos.AutenticacionRef, SesionRef: datos.SesionRef},
		resolutorContextoAltaContratacionTemporalDesarrollo{valor: registrado},
		vecdomain.SolicitudContextoActor{Cuenta: vecdomain.CuentaAutenticadaContextoActor{
			CuentaRef: datos.CuentaRef, Metodo: vecdomain.AuthMethodCertificate, Garantia: vecdomain.AuthAssuranceHigh},
			PerfilActivoRef: datos.PerfilActivoRef}, relojFijoAltaContratacionTemporalDesarrollo{ahora: ahora})
	if err != nil {
		t.Fatal(err)
	}
	sesion.Vinculo, sesion.Resultado = vinculo, resuelto
	empleado := registrado.Contexto.Instantanea.Vinculos[0].Referencia
	persona := registrado.Contexto.PersonaRef
	fuente.i.AsignacionPerfil.Ambitos = []vecdomain.AmbitoPerfil{
		{Clave: "persona_ref", Valores: []string{persona}},
		{Clave: "unidad_ref", Valores: []string{"unidad:prueba"}},
		{Clave: "empleado_ref", Valores: []string{empleado}},
	}
	fuente.i.VersionRol.Concesiones = append(fuente.i.VersionRol.Concesiones, vecdomain.ConcesionRol{
		Accion: "dietas.liquidacion_economica.consultar", ModuloID: "dietas", TipoRecurso: "liquidacion_economica",
		Finalidades:      []string{"consultar_liquidacion_economica_dietas"},
		CamposPermitidos: slices.Clone(camposLiquidacionEconomica), GarantiaMinima: vecdomain.AuthAssuranceHigh,
	})
	c := &capturadorLecturaLiquidacion{identidad: identidadLecturaLiquidacionPrueba{sesion}, fuente: fuente}
	ref := "dco_" + strings.Repeat("a", 22)
	if _, err := c.CapturarLecturaLiquidacionEconomica(context.Background(), ref); err != nil {
		t.Fatalf("el perfil del empleado canónico quedó denegado: %v", err)
	}
	fuente.i.AsignacionPerfil.Ambitos[2].Valores = []string{"emp_" + strings.Repeat("b", 22)}
	if _, err := c.CapturarLecturaLiquidacionEconomica(context.Background(), ref); !errors.Is(err, dietasports.ErrAccesoCircuitoDenegado) {
		t.Fatalf("el ámbito de otro empleado se aceptó: %v", err)
	}
	fuente.i.AsignacionPerfil.Ambitos[2].Valores = []string{empleado}
	fuente.i.AsignacionPerfil.Ambitos = append(fuente.i.AsignacionPerfil.Ambitos,
		vecdomain.AmbitoPerfil{Clave: "ambito_ref", Valores: []string{"ambito:ajeno"}})
	if _, err := c.CapturarLecturaLiquidacionEconomica(context.Background(), ref); !errors.Is(err, dietasports.ErrAccesoCircuitoDenegado) {
		t.Fatalf("una dimensión extra se aceptó: %v", err)
	}
	fuente.i.AsignacionPerfil.Ambitos = fuente.i.AsignacionPerfil.Ambitos[:3]
	fuente.i.VersionRol.Concesiones = fuente.i.VersionRol.Concesiones[:len(fuente.i.VersionRol.Concesiones)-1]
	if _, err := c.CapturarLecturaLiquidacionEconomica(context.Background(), ref); !errors.Is(err, dietasports.ErrAccesoCircuitoDenegado) {
		t.Fatalf("el permiso retirado se aceptó: %v", err)
	}
}
