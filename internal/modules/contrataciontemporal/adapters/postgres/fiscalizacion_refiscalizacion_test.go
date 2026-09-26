package postgres

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestFuncionesFiscalizacionSeleccionanSoloLosDosOrigenesPermitidos(t *testing.T) {
	pruebas := []struct {
		nombre     string
		version    uint64
		preparar   string
		confirmar  string
		debeFallar bool
	}{
		{
			nombre:    "inicial_v5",
			version:   5,
			preparar:  funcionPrepararFiscalizacion,
			confirmar: funcionConfirmarFiscalizacion,
		},
		{
			nombre:    "corregida_v7",
			version:   7,
			preparar:  funcionPrepararFiscalizacionV2,
			confirmar: funcionConfirmarFiscalizacionV2,
		},
		{
			nombre:    "corregida_o_modificada_posterior",
			version:   8,
			preparar:  funcionPrepararFiscalizacionV2,
			confirmar: funcionConfirmarFiscalizacionV2,
		},
		{nombre: "version_ajena_v6", version: 6, debeFallar: true},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			preparar, errPreparar := funcionPrepararFiscalizacionParaVersion(prueba.version)
			confirmar, errConfirmar := funcionConfirmarFiscalizacionParaVersion(prueba.version)
			if prueba.debeFallar {
				if errPreparar == nil || errConfirmar == nil {
					t.Fatal("la versión ajena no fue denegada")
				}
				return
			}
			if errPreparar != nil || errConfirmar != nil ||
				preparar != prueba.preparar || confirmar != prueba.confirmar {
				t.Fatalf("selector inesperado: preparar=%q (%v), confirmar=%q (%v)", preparar, errPreparar, confirmar, errConfirmar)
			}
		})
	}
}

func TestConfirmacionFiscalizacionTerminalNoReintentaOtros40001(t *testing.T) {
	terminal := &pgconn.PgError{
		Code:    "40001",
		Message: "recuperar preparación de fiscalización confirmada",
	}
	if !confirmacionFiscalizacionYaRecuperable(terminal) {
		t.Fatal("no reconoció el conflicto terminal de recuperación")
	}
	for _, causa := range []error{
		&pgconn.PgError{Code: "40001", Message: "serialización ordinaria"},
		&pgconn.PgError{Code: "40P01", Message: terminal.Message},
		errors.New("40001 recuperar preparación de fiscalización confirmada"),
	} {
		if confirmacionFiscalizacionYaRecuperable(causa) {
			t.Fatalf("conflicto no terminal marcado como recuperable: %v", causa)
		}
	}
}

// CT123: el rechazo de la base por falta de informe nuevo se reconoce solo
// con su código y mensaje exactos.
func TestConfirmacionFiscalizacionReconoceInformeNuevoPendiente(t *testing.T) {
	if !confirmacionFiscalizacionEsperaInformeNuevo(&pgconn.PgError{
		Code: "55000", Message: "informe jurídico nuevo tras subsanación pendiente",
	}) {
		t.Fatal("no reconoció el rechazo de CT123")
	}
	for _, causa := range []error{
		&pgconn.PgError{Code: "55000", Message: "otra causa"},
		&pgconn.PgError{Code: "42501", Message: "informe jurídico nuevo tras subsanación pendiente"},
		errors.New("55000 informe jurídico nuevo tras subsanación pendiente"),
	} {
		if confirmacionFiscalizacionEsperaInformeNuevo(causa) {
			t.Fatalf("rechazo ajeno tomado por CT123: %v", causa)
		}
	}
}
