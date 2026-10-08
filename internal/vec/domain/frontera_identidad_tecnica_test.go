package domain

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestOrdenFronteraIdentidadTecnicaSoloRutasYMotivosCerrados(t *testing.T) {
	primera, err := NuevaOrdenFronteraIdentidadTecnica(MetodoInicioSesionGET, RutaSesionActual, MotivoCertificadoRequerido)
	if err != nil {
		t.Fatal(err)
	}
	segunda, err := NuevaOrdenFronteraIdentidadTecnica(MetodoInicioSesionPOST, RutaInicioSesion, MotivoServicioNoDisponible)
	if err != nil {
		t.Fatal(err)
	}
	a, err := primera.Datos()
	if err != nil || a.Resultado != "denegado" || a.CorrelacionRef == "" ||
		!strings.HasPrefix(a.CorrelacionRef, "correlacion_") || len(a.CorrelacionRef) != len("correlacion_")+32 {
		t.Fatalf("orden GET sin correlacion tecnica valida: %v", err)
	}
	b, err := segunda.Datos()
	if err != nil || b.Resultado != "error" || b.CorrelacionRef == a.CorrelacionRef {
		t.Fatalf("orden POST no separada: %v", err)
	}
	for _, caso := range []struct {
		metodo, ruta string
		motivo       MotivoFronteraIdentidadTecnica
	}{
		{MetodoInicioSesionPOST, RutaSesionActual, MotivoAccesoDenegado},
		{MetodoInicioSesionGET, RutaInicioSesion, MotivoAccesoDenegado},
		{MetodoInicioSesionGET, RutaSesionActual + "?id=1", MotivoAccesoDenegado},
		{MetodoInicioSesionGET, RutaSesionActual, "motivo_del_cliente"},
	} {
		if _, err := NuevaOrdenFronteraIdentidadTecnica(caso.metodo, caso.ruta, caso.motivo); !errors.Is(err, ErrOrdenFronteraIdentidadTecnicaInvalida) {
			t.Fatalf("orden ajena admitida: %v", err)
		}
	}
	if _, err := (OrdenFronteraIdentidadTecnica{}).Datos(); !errors.Is(err, ErrOrdenFronteraIdentidadTecnicaInvalida) {
		t.Fatalf("orden cero admitida: %v", err)
	}
	if _, err := json.Marshal(primera); !errors.Is(err, ErrOrdenFronteraIdentidadTecnicaInvalida) ||
		strings.Contains(primera.String(), a.CorrelacionRef) {
		t.Fatal("orden opaca serializable o legible")
	}
}

func TestAcuseFronteraIdentidadTecnicaExigeMaterialYFechaActual(t *testing.T) {
	const correlacion = "correlacion_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	orden := OrdenFronteraIdentidadTecnica{
		metodoEsperado: MetodoInicioSesionGET, ruta: RutaSesionActual,
		motivo: MotivoCertificadoRequerido, correlacionRef: correlacion,
	}
	const evento = "evento_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	const material = "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	ahora := time.Date(2026, 10, 8, 2, 0, 0, 0, time.UTC)
	acuse := AcuseFronteraIdentidadTecnica{
		AuditoriaRef: "aud_v3_pit_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		Secuencia:    12, MaterialSHA256: material, CorrelacionRef: correlacion,
		RegistradaEn: ahora.Add(-time.Microsecond),
	}
	if err := acuse.ValidarPara(orden, evento, material, ahora); err != nil {
		t.Fatal(err)
	}
	for _, mutar := range []func(*AcuseFronteraIdentidadTecnica){
		func(a *AcuseFronteraIdentidadTecnica) { a.AuditoriaRef = "aud_v3_pit_ajeno" },
		func(a *AcuseFronteraIdentidadTecnica) { a.MaterialSHA256 = strings.Repeat("d", 64) },
		func(a *AcuseFronteraIdentidadTecnica) { a.Secuencia = 1 << 53 },
		func(a *AcuseFronteraIdentidadTecnica) { a.CorrelacionRef = "correlacion_" + strings.Repeat("d", 32) },
		func(a *AcuseFronteraIdentidadTecnica) { a.RegistradaEn = ahora.Add(time.Microsecond) },
	} {
		alterado := acuse
		mutar(&alterado)
		if err := alterado.ValidarPara(orden, evento, material, ahora); !errors.Is(err, ErrAcuseFronteraIdentidadTecnicaInvalido) {
			t.Fatalf("acuse falso admitido: %v", err)
		}
	}
	if recurso, err := RecursoFronteraIdentidadTecnica(correlacion); err != nil ||
		recurso != "solicitud_sesion:ee9bee5664cf9be629ab1fdc52711201" {
		t.Fatalf("recurso no coincide con vector independiente: %v", err)
	}
}
