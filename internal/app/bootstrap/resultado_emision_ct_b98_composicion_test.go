package bootstrap

import (
	"context"
	"errors"
	"testing"

	bp "vec-diputacion-granada/internal/modules/bolsa/ports"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestLectorB98RechazaAntesDeDetalleSinSesionYPreparacionCT(t *testing.T) {
	lector := &lectorResultadoEmisionCTDesdeDetalle{}
	_, err := lector.LeerResultadoEmisionCT(context.Background(), "bolsa:una", "llamamiento:uno", "recibo:uno")
	if !errors.Is(err, bp.ErrResultadoEmisionCTNoDisponible) {
		t.Fatalf("lector sin detalle nominal: %v", err)
	}
}

func TestAnclajeB113EnPropuestaExigeRutaRealYCoordenadasCT(t *testing.T) {
	f := nuevaPropuestaCoordinadorPrueba(t)
	p, err := prepararReferenciasLlamamientoDesarrollo(referenciaOrganizativaCTMontajePrueba, f.lector.expediente, f.registro.antecedente.SeleccionClave)
	if err != nil {
		t.Fatal(err)
	}
	x := bp.SelectorAceptacionNativaCT{OrganizacionRef: p.expediente.Fiscalizado.OrganizacionRef,
		ExpedienteRef: p.expediente.Fiscalizado.Referencia, ClaveCT: p.clave,
		UnidadRef: p.unidad, CategoriaRef: p.categoria, NecesidadRef: p.necesidad,
		BolsaRef: "bolsa:una", LlamamientoRef: "llamamiento:uno",
		ReciboEmisionRef: "recibo:emision:uno", ReciboRespuestaRef: "recibo:respuesta:uno"}
	a := &autoridadIncorporacionPersonalB2{perfiles: &perfilesNominalesIncorporacion{soporte: f.ejecutor.soporte}}
	ctx := context.WithValue(f.ctx, clavePreparacionLlamamientoDesarrollo{}, p)
	ctx = context.WithValue(ctx, claveIncorporacionV2Desarrollo{}, f.ejecutor.soporte.sello)
	ctx = context.WithValue(ctx, claveAnclajeNativoPropuestaCT{}, x)
	if !a.anclajeNativoPropuestaPermitido(ctx, bp.AccionConsultaAnclajeAceptacionNativaCT) ||
		a.anclajeNativoPropuestaPermitido(ctx, bp.AccionConsultaPersonaAceptacionNativaCT) {
		t.Fatal("la ruta CT no separó anclaje y persona")
	}
	x.UnidadRef = "unidad:ajena"
	ctx = context.WithValue(ctx, claveAnclajeNativoPropuestaCT{}, x)
	if a.anclajeNativoPropuestaPermitido(ctx, bp.AccionConsultaAnclajeAceptacionNativaCT) {
		t.Fatal("anclaje de otra unidad autorizado")
	}
	ctx = context.WithValue(ctx, claveAnclajeNativoPropuestaCT{}, bp.SelectorAceptacionNativaCT{})
	if a.anclajeNativoPropuestaPermitido(ctx, ct.AccionLeerPlanNominalB2) {
		t.Fatal("permiso B2 general habilitado por la propuesta")
	}
}
