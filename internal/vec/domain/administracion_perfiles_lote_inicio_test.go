package domain

import (
	"strings"
	"testing"
	"time"
)

func cambioInicioLotePrueba() CambioPerfilAdministracion {
	ahora := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	return CambioPerfilAdministracion{
		Operacion: OperacionOtorgarPerfil, InicioVigencia: InicioVigenciaLoteInmediato,
		RolVersionRef: "rol:gestion:v1",
		Objetivo: PreimagenAdministracionPerfiles{
			UnidadRef: "unidad:prueba", CuentaRef: "cta_" + strings.Repeat("a", 22), CuentaVersion: 1,
			PersonaRef: "per_" + strings.Repeat("b", 22), PersonaVersion: 1,
			PerfilRef: "prf_" + strings.Repeat("c", 22), VinculoRef: "vca_" + strings.Repeat("d", 22),
			HuellaSHA256: strings.Repeat("e", 64), ProcedenciaRef: "procedencia:maestra:1",
			ProcedenciaVersion: 1, ProcedenciaHuellaSHA256: strings.Repeat("f", 64),
			VigenteHasta: ahora.Add(time.Hour),
		},
	}
}

func TestModoInicioLoteNoFabricaFechaInmediata(t *testing.T) {
	c := cambioInicioLotePrueba()
	if c.validarLote() != nil {
		t.Fatal("alta inmediata sin fecha fue rechazada")
	}
	c.Objetivo.VigenteDesde = c.Objetivo.VigenteHasta.Add(-time.Minute)
	if c.validarLote() == nil {
		t.Fatal("alta inmediata aceptó fecha suministrada")
	}
	c.InicioVigencia = InicioVigenciaLoteProgramado
	if c.validarLote() != nil {
		t.Fatal("alta programada con fecha expresa fue rechazada")
	}
	c.Objetivo.VigenteDesde = time.Time{}
	if c.validarLote() == nil {
		t.Fatal("alta programada sin fecha fue aceptada")
	}
}

func TestBajaLoteNoRecibeModoNiFechaNueva(t *testing.T) {
	c := cambioInicioLotePrueba()
	c.Operacion, c.InicioVigencia = OperacionRevocarPerfil, ""
	c.Objetivo.PerfilVersion, c.Objetivo.VinculoVersion = 3, 3
	c.Objetivo.VigenteHasta = time.Time{}
	if c.validarLote() != nil {
		t.Fatal("baja sin fecha nueva fue rechazada")
	}
	c.InicioVigencia = InicioVigenciaLoteInmediato
	if c.validarLote() == nil {
		t.Fatal("baja aceptó modo de otorgamiento")
	}
}
