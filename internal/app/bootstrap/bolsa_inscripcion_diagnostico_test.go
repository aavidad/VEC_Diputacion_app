package bootstrap

import (
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/shared/telemetria"
)

func TestInscripcionRRHHArranqueIdentificaDependenciaAusente(t *testing.T) {
	cfg := configInscripcionEncendidaPrueba(t)
	cfg.DevelopmentMaterialDir = t.TempDir()
	identidad := &resolvedorIdentidadDesarrollo{}
	derivador := nuevoDerivadorIdempotenciaPrueba(t, 2, 1)
	proveedor := &proveedorMaterialAltaContratacionTemporalDesarrollo{}
	for _, caso := range []struct {
		fase      string
		identidad *resolvedorIdentidadDesarrollo
		derivador *derivadorIdentidadOperacionDesarrollo
		material  materialInscripcionRRHHDesarrollo
	}{
		{"resolvedor", nil, nil, materialInscripcionRRHHDesarrollo{}},
		{"idempotencia", identidad, nil, materialInscripcionRRHHDesarrollo{}},
		{"material_revisar", identidad, derivador, materialInscripcionRRHHDesarrollo{}},
		{"material_incorporar", identidad, derivador, materialInscripcionRRHHDesarrollo{revisar: proveedor}},
		{"fichero_privado", identidad, derivador, materialInscripcionRRHHDesarrollo{revisar: proveedor, incorporar: proveedor}},
	} {
		t.Run(caso.fase, func(t *testing.T) {
			_, err := nuevaInscripcionRRHHDesarrollo(cfg, caso.identidad, caso.derivador, caso.material)
			err = marcarFalloComponenteArranque("inscripcion_bolsa_rrhh", err)
			if !errors.Is(err, errMontajeInscripcionBolsa) || ComponenteFalloArranque(err) != "inscripcion_rrhh_"+caso.fase {
				t.Fatalf("guarda esperado=%s actual=%s: %v", caso.fase, ComponenteFalloArranque(err), err)
			}
			mensaje := telemetria.MensajeErrorArranque(err)
			if strings.Contains(mensaje, "no catalogado") || strings.Contains(mensaje, cfg.DevelopmentMaterialDir) {
				t.Fatalf("diagnóstico genérico o ruta privada: %s", mensaje)
			}
		})
	}
}

func TestInscripcionRRHHCatalogoPreservaCausaSinDatosPrivados(t *testing.T) {
	privado := errors.New("postgres://usuario:clave@destino/privada certificado cuenta")
	err := marcarFalloComponenteArranque("inscripcion_rrhh_cuentas_lectores", errors.Join(errMontajeInscripcionBolsa, privado))
	if !errors.Is(err, privado) || !errors.Is(err, errMontajeInscripcionBolsa) {
		t.Fatal("se perdió la causa interna")
	}
	if mensaje := telemetria.MensajeErrorArranque(err); strings.Contains(mensaje, privado.Error()) ||
		!strings.Contains(mensaje, errMontajeInscripcionBolsa.Error()) {
		t.Fatalf("diagnóstico no seguro: %s", mensaje)
	}
}
