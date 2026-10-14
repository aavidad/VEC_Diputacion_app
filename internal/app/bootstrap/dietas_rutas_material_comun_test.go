package bootstrap

import (
	"context"
	"testing"
	"time"

	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

func TestAmbitoRutasAdmiteNombreVisibleYRevocaAlCambiarLimites(t *testing.T) {
	valor := ambitoTecnicoRutasDietas("Granada provincia + 15 km", "-4.0,36.0,-2.0,38.0")
	if err := (vecdomain.AmbitoPerfil{Clave: "ambito_ref", Valores: []string{valor}}).Validar(); err != nil {
		t.Fatalf("el ámbito no es asignable: %v", err)
	}
	if valor == ambitoTecnicoRutasDietas("Granada provincia + 15 km", "-4.0,36.0,-2.0,38.1") {
		t.Fatal("cambiar límites conservó la misma concesión territorial")
	}
}

func TestMaterialRutasGobernadoPorCTConservaAudienciaNominal(t *testing.T) {
	descriptor := descriptorMaterialRutasDietasDesarrollo()
	if descriptor.Audiencia != dietasports.AudienciaAccesoRutasDietas ||
		!audienciaConsumoGobiernoPostgreSQLContratacionTemporalDesarrolloEsPropia(descriptor.Audiencia) {
		t.Fatal("la audiencia de rutas no pertenece al gobierno nominal común")
	}
	for _, accion := range []string{dietasports.AccionConsultarCatalogoRutasDietas, dietasports.AccionSolicitarCalculoRutasDietas} {
		t.Run(accion, func(t *testing.T) {
			escenario := nuevoEscenarioMaterialRutasDietasPrueba(t, accion, time.Now().UTC().Truncate(time.Microsecond))
			material, err := nuevoMaterialAtestacionDietasDesarrollo(escenario.datos, audienciaAtestacionContratacionTemporalDesarrollo, audienciaConsumoRutasDietasDesarrollo)
			if err != nil {
				t.Fatal(err)
			}
			defer material.borrarCopiasEfimeras()
			comun, err := nuevoProveedorMaterialAutorizacionBaseDesarrollo(materialAtestacionContratacionTemporalDesarrollo{
				claveID: material.claveID, privada: material.privada, raiz: material.raiz,
				configuracion: material.configuracion, capacidad: material.capacidad,
			}, relojContratacionTemporalDesarrollo{})
			if err != nil {
				t.Fatal(err)
			}
			proveedor := &proveedorMaterialRutasCTDesarrollo{comun: comun}
			exportado, err := proveedor.ProveerMaterialAccesoRutas(context.Background(), escenario.solicitud, escenario.decision, escenario.confirmacion, escenario.resultado)
			if err != nil {
				t.Fatal(err)
			}
			if exportado.ValidarEstructura() != nil || exportado.ResumenCapacidad().AudienciaConsumo() != dietasports.AudienciaAccesoRutasDietas {
				t.Fatal("la capacidad no conserva la audiencia nominal de Rutas")
			}
		})
	}
}

func TestMaterialRutasComunRechazaProveedorAusente(t *testing.T) {
	escenario := nuevoEscenarioMaterialRutasDietasPrueba(t, dietasports.AccionConsultarCatalogoRutasDietas, time.Now().UTC().Truncate(time.Microsecond))
	proveedor := &proveedorMaterialRutasCTDesarrollo{}
	if _, err := proveedor.ProveerMaterialAccesoRutas(context.Background(), escenario.solicitud, escenario.decision, escenario.confirmacion, escenario.resultado); err == nil {
		t.Fatal("rutas emitió material sin proveedor común")
	}
}
