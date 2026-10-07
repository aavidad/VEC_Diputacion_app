package ports_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/pruebas"
)

func TestIntentoAuditoriaExigeIdentidadLigadaYCanalAcreditado(t *testing.T) {
	ahora := time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC)
	resultadoA, vinculoA, err := pruebas.NuevoContextoRegistradoYVinculoV2(
		ahora, "per_0123456789abcdefghijkl", "prf_0123456789abcdefghijkl",
		domain.AuthMethodCertificate, domain.AuthAssuranceHigh,
	)
	if err != nil {
		t.Fatal(err)
	}
	resultadoB, _, err := pruebas.NuevoContextoRegistradoYVinculoV2(
		ahora, "per_1123456789abcdefghijkl", "prf_0123456789abcdefghijkl",
		domain.AuthMethodCertificate, domain.AuthAssuranceHigh,
	)
	if err != nil {
		t.Fatal(err)
	}
	datos := domain.DatosIntentoAuditoria{
		Accion: "administracion.perfiles.consultar", ModuloID: "administracion",
		RecursoRef: "persona:11111111111111111111111111111111", FinalidadRef: "administracion_perfiles",
		Resultado: domain.ResultadoIntentoAuditoriaDenegado,
		Motivo: domain.ReferenciaEntradaCatalogo{
			CatalogoID: "motivos_auditoria", CatalogoVersion: 1,
			CatalogoHuellaSHA256: strings.Repeat("a", 64), EntradaClave: "acceso_denegado",
		},
		Proceso: "vec-server", Canal: "interna_corporativa",
		CorrelacionRef: "correlacion_11111111111111111111111111111111",
	}
	const intento = "intento_11111111111111111111111111111111"
	if _, err := ports.NuevaOrdenIntentoAuditoria(intento, resultadoB, vinculoA, datos); !errors.Is(err, ports.ErrOrdenIntentoAuditoriaInvalida) {
		t.Fatalf("identidad distinta aceptada: %v", err)
	}
	datos.Canal = "administracion_privilegiada"
	if _, err := ports.NuevaOrdenIntentoAuditoria(intento, resultadoA, vinculoA, datos); !errors.Is(err, ports.ErrOrdenIntentoAuditoriaInvalida) {
		t.Fatalf("canal ajeno al vinculo aceptado: %v", err)
	}
	datos.Canal = "interna_corporativa"
	orden, err := ports.NuevaOrdenIntentoAuditoria(intento, resultadoA, vinculoA, datos)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := json.Marshal(orden); !errors.Is(err, ports.ErrOrdenIntentoAuditoriaInvalida) {
		t.Fatalf("orden sensible serializable: %v", err)
	}
	if salida := fmt.Sprintf("%d", orden); salida != orden.String() {
		t.Fatalf("formato no textual expone orden: %q", salida)
	}
	func() {
		defer func() {
			if !errors.Is(errorDePanicoIntentoAuditoria(recover()), ports.ErrOrdenIntentoAuditoriaInvalida) {
				t.Fatal("el fallo del escritor expuso error o contenido privado")
			}
		}()
		orden.Format(estadoFormatoIntentoAuditoriaFallido{}, 'd')
	}()
}

type estadoFormatoIntentoAuditoriaFallido struct{ fmt.State }

func (estadoFormatoIntentoAuditoriaFallido) Write([]byte) (int, error) {
	return 0, errors.New("fallo_sintetico_del_destino")
}
func errorDePanicoIntentoAuditoria(valor any) error {
	err, _ := valor.(error)
	return err
}
