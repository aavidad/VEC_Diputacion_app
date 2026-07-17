package calculoexperienciaoficial

import (
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
)

func huellaPrueba(marca string) string { return strings.Repeat(marca, 64) }

func referenciaPrueba(referencia, marca string) ReferenciaExactaV1 {
	return ReferenciaExactaV1{
		Referencia: referencia, Version: 1, HuellaSHA256: huellaPrueba(marca),
	}
}

func datosClavePrueba() DatosClaveEfectoV1 {
	return DatosClaveEfectoV1{
		SujetoPseudonimizado: referenciaPrueba(
			"hmac-sha256:seudonimo_oficial_v1:"+huellaPrueba("0"), "1",
		),
		Convocatoria: referenciaPrueba("Convocatoria/2026#1", "2"),
		Reglas: VinculoReglasV1{
			Contenido: referenciaPrueba("ReglasBaremo/2026#3", "3"),
			Revision:  4, HuellaEstadoSHA256: huellaPrueba("4"),
		},
		Entrada: VinculoEntradaV1{
			Instantanea:           referenciaPrueba("EntradaExperiencia/ABC#7", "5"),
			HuellaContenidoSHA256: huellaPrueba("6"),
		},
		Motor: VinculoMotorV1{
			Contrato: "vec.bolsa.motor_experiencia.v1", Version: 1,
			HuellaContratoSHA256: huellaPrueba("7"),
		},
		HuellaPlanSHA256: huellaPrueba("8"),
		Causa: CausaGobernadaV1{
			Catalogo: referenciaPrueba("CatalogoCausas/Oficial#2", "9"),
			Clave:    "calculo_inicial_reglamentario",
		},
		Tipo: EfectoCalculoInicial,
	}
}

func clavePrueba(t *testing.T) ClaveEfectoV1 {
	t.Helper()
	clave, err := NuevaClaveEfectoV1(datosClavePrueba())
	if err != nil {
		t.Fatalf("crear clave de prueba: %v", err)
	}
	return clave
}

func intencionPrueba(t *testing.T) IntencionResultadoV1 {
	t.Helper()
	intencion, err := NuevaIntencionResultadoV1(
		clavePrueba(t), huellaPrueba("a"), ResultadoCompletado, FaseCompletado,
	)
	if err != nil {
		t.Fatalf("crear intencion de prueba: %v", err)
	}
	return intencion
}

func reciboPrueba(t *testing.T) ReciboV1 {
	t.Helper()
	intencion := intencionPrueba(t)
	indice, err := CalcularIndiceHMACSHA256(intencion.Clave(), []byte(strings.Repeat("k", 32)))
	if err != nil {
		t.Fatalf("calcular indice: %v", err)
	}
	recibo, err := NuevoReciboV1("ReciboCalculo/ABC#1", 3, indice, intencion)
	if err != nil {
		t.Fatalf("crear recibo: %v", err)
	}
	return recibo
}

func TestReferenciaExactaCompatibleConReferenciaVersionadaDeReglas(t *testing.T) {
	origen, err := reglasbaremo.NuevaReferenciaVersionada(
		"Ref.Valida_MAYUS/2026#A:1", 17, huellaPrueba("b"),
	)
	if err != nil {
		t.Fatalf("la referencia fuente debía ser válida: %v", err)
	}
	compatible := ReferenciaExactaV1{
		Referencia: origen.Referencia(), Version: origen.Version(), HuellaSHA256: origen.HuellaSHA256(),
	}
	if err := validarReferencia(compatible, "prueba"); err != nil {
		t.Fatalf("el contrato oficial rechazó una referencia fuente válida: %v", err)
	}
}
