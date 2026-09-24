package domain

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func reglaTramosPrueba() ReglaDevengoProvisional {
	return ReglaDevengoProvisional{ReglaRef: "provisional:regla:nacional-ordinaria:20260923", VersionTarifaRef: "provisional:rd462:20260923", PaisISO2: "ES", Variante: "nacional_ordinaria", HuellaSHA256: strings.Repeat("a", 64), Configuracion: ConfiguracionDevengoProvisional{Regla: "nacional_ordinaria_provisional_v1", Zona: "Europe/Madrid", DuracionMinimaMismoDiaHoras: 5, HoraSalida100AntesDe: 14, HoraSalida50AntesDe: 22, HoraRegreso50DespuesDe: 14, HoraRegresoMismoDiaDespuesDe: 16, DiasMaximos: 31, Alojamiento: "tope_pendiente_justificante", Liquidable: false, PorcentajeMismoDia: 50, PorcentajeSalidaTemprana: 100, PorcentajeSalidaMedia: 50, PorcentajeRegreso: 50, PorcentajeIntermedio: 100, PorcentajeAlojamientoTope: 100}}
}

func tarifaTramosPrueba() TarifaNacionalProvisional {
	return TarifaNacionalProvisional{
		VersionRef: "provisional:rd462:20260923", Rotulo: RotuloTarifaProvisional,
		PaisISO2: "ES", Grupo: 2, VigenteDesde: "2026-09-23",
		ManutencionCentimos: 3740, AlojamientoTopeCentimos: 6597,
	}
}

func instanteTramosPrueba(t *testing.T, zona *time.Location, fechaHora string) time.Time {
	t.Helper()
	v, err := time.ParseInLocation("2006-01-02 15:04", fechaHora, zona)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestTramosProvisionalesDiaUnicoExigeCincoHorasYComida(t *testing.T) {
	zona, _ := time.LoadLocation("Europe/Madrid")
	calcular := func(inicio, fin string) CalculoDietasProvisional {
		t.Helper()
		r, err := CalcularTramosNacionalesProvisionales(instanteTramosPrueba(t, zona, inicio), instanteTramosPrueba(t, zona, fin), zona, tarifaTramosPrueba(), reglaTramosPrueba())
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	corto := calcular("2026-09-23 12:00", "2026-09-23 16:30")
	if len(corto.Tramos) != 0 || corto.TotalMaximoOrientativo != 0 {
		t.Fatalf("cuatro horas y media no generan manutención: %+v", corto)
	}
	comida := calcular("2026-09-23 11:00", "2026-09-23 17:00")
	if len(comida.Tramos) != 1 || comida.Tramos[0].Tipo != "manutencion" || comida.Tramos[0].Porcentaje != 50 || comida.TotalMaximoOrientativo != 1870 || comida.Rotulo != RotuloTarifaProvisional {
		t.Fatalf("media manutención provisional incompatible: %+v", comida)
	}
	limite := calcular("2026-09-23 14:00", "2026-09-23 20:00")
	if len(limite.Tramos) != 0 {
		t.Fatalf("inicio exactamente a las 14 h no cumple antes de las 14: %+v", limite)
	}
}

func TestTramosProvisionalesVariosDiasSeparanManutencionYTopeDeAlojamiento(t *testing.T) {
	zona, _ := time.LoadLocation("Europe/Madrid")
	r, err := CalcularTramosNacionalesProvisionales(
		instanteTramosPrueba(t, zona, "2026-09-23 13:00"),
		instanteTramosPrueba(t, zona, "2026-09-25 15:00"), zona, tarifaTramosPrueba(), reglaTramosPrueba(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Tramos) != 5 || r.ManutencionCentimos != 3740+3740+1870 || r.AlojamientoTopeCentimos != 2*6597 || r.TotalMaximoOrientativo != 22544 {
		t.Fatalf("tres días con dos noches deben separar importes máximos: %+v", r)
	}
	for _, tramo := range r.Tramos {
		if tramo.VersionTarifaRef != r.VersionTarifaRef || tramo.Rotulo != RotuloTarifaProvisional {
			t.Fatalf("tramo sin versión provisional: %+v", tramo)
		}
	}
}

func TestTramosProvisionalesCambioHorarioCuentaUnaNoche(t *testing.T) {
	zona, _ := time.LoadLocation("Europe/Madrid")
	r, err := CalcularTramosNacionalesProvisionales(
		instanteTramosPrueba(t, zona, "2026-10-24 20:00"),
		instanteTramosPrueba(t, zona, "2026-10-25 09:00"), zona, tarifaTramosPrueba(), reglaTramosPrueba(),
	)
	if err != nil || r.AlojamientoTopeCentimos != 6597 || r.ManutencionCentimos != 1870 || len(r.Tramos) != 2 {
		t.Fatalf("cambio horario alteró día natural o porcentajes: %+v %v", r, err)
	}
}

func TestTramosProvisionalesUnaNocheYRedondeoCentimos(t *testing.T) {
	zona, _ := time.LoadLocation("Europe/Madrid")
	tarifa := tarifaTramosPrueba()
	tarifa.Grupo, tarifa.ManutencionCentimos = 3, 2821
	r, err := CalcularTramosNacionalesProvisionales(
		instanteTramosPrueba(t, zona, "2026-09-23 15:00"),
		instanteTramosPrueba(t, zona, "2026-09-24 14:01"), zona, tarifa, reglaTramosPrueba(),
	)
	if err != nil || len(r.Tramos) != 3 || r.ManutencionCentimos != 2822 || r.AlojamientoTopeCentimos != 6597 || r.Tramos[1].Tipo != "alojamiento_tope_pendiente_justificante" {
		t.Fatalf("una noche debe mostrar dos medias dietas y tope separado: %+v %v", r, err)
	}
}

func TestTramosProvisionalesDenieganTarifaAjenaOVencida(t *testing.T) {
	zona, _ := time.LoadLocation("Europe/Madrid")
	i := instanteTramosPrueba(t, zona, "2026-09-23 09:00")
	f := instanteTramosPrueba(t, zona, "2026-09-23 17:00")
	casos := []TarifaNacionalProvisional{tarifaTramosPrueba(), tarifaTramosPrueba(), tarifaTramosPrueba(), tarifaTramosPrueba()}
	casos[0].PaisISO2 = "FR"
	casos[1].Rotulo = "definitiva"
	casos[2].VersionRef = "legal:20260923"
	casos[3].VigenteHasta = "2026-09-23"
	for _, tarifa := range casos {
		if _, err := CalcularTramosNacionalesProvisionales(i, f, zona, tarifa, reglaTramosPrueba()); !errors.Is(err, ErrTramosProvisionalesNoDisponibles) {
			t.Fatalf("tarifa no gobernada admitida: %+v, err=%v", tarifa, err)
		}
	}
}

func TestReglaCatalogadaNuevaVersionCambiaTramoYPreservaInstantaneaAnterior(t *testing.T) {
	zona, _ := time.LoadLocation("Europe/Madrid")
	inicio := instanteTramosPrueba(t, zona, "2026-09-23 11:00")
	fin := instanteTramosPrueba(t, zona, "2026-09-23 17:00")
	tarifa := tarifaTramosPrueba()
	regla1 := reglaTramosPrueba()
	anterior, err := CalcularTramosNacionalesProvisionales(inicio, fin, zona, tarifa, regla1)
	if err != nil || len(anterior.Tramos) != 1 || anterior.Tramos[0].ImporteCentimos != 1870 {
		t.Fatalf("regla inicial: %+v %v", anterior, err)
	}
	tarifa.VersionRef = "provisional:rd462:20260924"
	regla2 := regla1
	regla2.ReglaRef = "provisional:regla:nacional-ordinaria:20260924"
	regla2.VersionTarifaRef = tarifa.VersionRef
	regla2.HuellaSHA256 = strings.Repeat("b", 64)
	regla2.Configuracion.PorcentajeMismoDia = 75
	nuevo, err := CalcularTramosNacionalesProvisionales(inicio, fin, zona, tarifa, regla2)
	if err != nil || len(nuevo.Tramos) != 1 || nuevo.Tramos[0].Porcentaje != 75 || nuevo.Tramos[0].ImporteCentimos != 2805 || nuevo.ReglaRef != regla2.ReglaRef || nuevo.ReglaHuellaSHA256 != regla2.HuellaSHA256 {
		t.Fatalf("regla nueva: %+v %v", nuevo, err)
	}
	if anterior.Tramos[0].ImporteCentimos != 1870 || anterior.ReglaRef != regla1.ReglaRef || anterior.VersionTarifaRef != regla1.VersionTarifaRef {
		t.Fatal("instantánea anterior alterada por versión nueva")
	}
	regla2.Configuracion.PorcentajeMismoDia = 101
	if _, err := CalcularTramosNacionalesProvisionales(inicio, fin, zona, tarifa, regla2); !errors.Is(err, ErrTramosProvisionalesNoDisponibles) {
		t.Fatalf("porcentaje fuera del esquema aceptado: %v", err)
	}
}
