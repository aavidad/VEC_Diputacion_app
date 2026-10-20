package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"vec-diputacion-granada/config"
)

type errorCausaMontajeCTPrueba struct{}

func (*errorCausaMontajeCTPrueba) Error() string {
	return "postgres://persona:credencial-no-publicable@invalid/ruta-reservada"
}

func TestFalloMontajeCTConservaCausaYConstructorInterior(t *testing.T) {
	causa := &errorCausaMontajeCTPrueba{}
	interior := marcarFalloMontajeCT("firma_material_registro", causa)
	fallo := marcarFalloMontajeCT("firma_nominal", fmt.Errorf("envoltura: %w", interior))
	if !errors.Is(fallo, causa) {
		t.Fatal("se perdió la causa original")
	}
	var recuperada *errorCausaMontajeCTPrueba
	if !errors.As(fallo, &recuperada) || recuperada != causa {
		t.Fatal("no se puede recuperar el tipo original")
	}
	if got := constructorFalloMontajeCT(fallo); got != "ct.montaje.firma_material_registro" {
		t.Fatalf("constructor = %q", got)
	}
	if strings.Contains(interior.Error(), "credencial") {
		t.Fatal("el marcador incluye el mensaje de la causa")
	}
	if marcarFalloMontajeCT("firma_nominal", nil) != nil {
		t.Fatal("un resultado correcto se convirtió en fallo")
	}
}

func TestRegistroFalloMontajeCTSoloPublicaEtiquetaCerrada(t *testing.T) {
	anterior := slog.Default()
	t.Cleanup(func() { slog.SetDefault(anterior) })
	for _, constructor := range []string{"firma_material_registro", "persona", "firma_material_registro\ncredencial", ""} {
		t.Run(constructor, func(t *testing.T) {
			var salida bytes.Buffer
			slog.SetDefault(slog.New(slog.NewJSONHandler(&salida, nil)))
			fallo := marcarFalloComponenteArranque("contratacion_temporal", marcarFalloMontajeCT(constructor, &errorCausaMontajeCTPrueba{}))
			registrarFalloMontajeCT(fallo)
			var evento map[string]any
			if err := json.Unmarshal(salida.Bytes(), &evento); err != nil {
				t.Fatal(err)
			}
			esperado := "ct.montaje.sin_etiqueta"
			if constructor == "firma_material_registro" {
				esperado = "ct.montaje.firma_material_registro"
			}
			if evento["constructor"] != esperado || evento["componente"] != "contratacion_temporal" || evento["msg"] != "contratacion_temporal_no_disponible" {
				t.Fatalf("atributos del evento inesperados: %v", evento)
			}
			for _, reservado := range []string{"persona", "credencial", "postgres://", "ruta-reservada", "invalid"} {
				if strings.Contains(salida.String(), reservado) {
					t.Fatalf("el registro contiene un dato reservado: %s", reservado)
				}
			}
		})
	}
}

func TestConstructoresMontajeCTEtiquetanFalloSinPublicar(t *testing.T) {
	_, _, _, principal := nuevasRutasContratacionTemporalConReglasDesarrollo(config.Config{}, reglasEjemploDesarrollo{}, nil, nil, nil, nil,
		ConfiguracionIncorporacionDesarrollo{}, ConfiguracionIncorporacionDesarrollo{})
	_, firma := prepararMontajeFirmaVecR5CT(context.Background(), config.Config{}, configuracionFirmasCT{}, nil, dependenciasConsultasRRHHDesarrollo{}, nil, nil, nil, nil)
	_, lecturas := nuevasLecturasFirmaV2(context.Background(), nil, nil, nil)
	for _, caso := range []struct {
		err         error
		constructor string
		causa       error
	}{
		{principal, "incorporacion_config", ErrComposicionDesarrolloIncompleta},
		{firma, "firma_requisitos", errMontajeFirmaVecR5},
		{lecturas, "lecturas_requisitos", errMontajeFirmaVecR5},
	} {
		if !errors.Is(caso.err, caso.causa) || constructorFalloMontajeCT(caso.err) != "ct.montaje."+caso.constructor {
			t.Fatalf("fallo de %s perdió su causa o etiqueta", caso.constructor)
		}
	}
}
