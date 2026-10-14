package domain

import (
	"errors"
	"testing"
	"time"
)

func TestLiquidacionPublicadaMuestraDiferenciaYExigeDecision(t *testing.T) {
	base := preparacionPublicadaPrueba(t).Instantanea()
	for _, caso := range []struct {
		nombre, motivo  string
		base, calculado int64
	}{
		{"tarifa superior", "diferencia_tarifa_publicada", 2400, 1200},
		{"tarifa inferior", "diferencia_tarifa_publicada", 800, 400},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			c := base.Catalogo
			c.Reglas[0].TopeCentimos = caso.base
			var err error
			c.HuellaSHA256, err = c.HuellaCalculada()
			if err != nil {
				t.Fatal(err)
			}
			preparar := func(revisiones []RevisionLineaLiquidacion) (*PreparacionLiquidacionPublicada, error) {
				return PrepararLiquidacionPublicada(base.ComisionRef, base.ComisionVersion,
					base.FechaInicio, base.HoraInicio, base.FechaFin, base.HoraFin,
					base.DocumentoSHA256, base.Documento, c, revisiones)
			}
			vista, err := preparar(nil)
			if err != nil {
				t.Fatal(err)
			}
			s := vista.Instantanea()
			if !s.DecisionPendiente || s.Lineas[0].OriginalCentimos != 500 ||
				s.Lineas[0].CalculadoPublicadoCentimos != caso.calculado ||
				s.Lineas[0].DiferenciaCentimos != caso.calculado-500 ||
				s.Totales.DiferenciaCentimos != caso.calculado-500 {
				t.Fatalf("vista sin diferencia comprobable: %+v", s)
			}
			constancia := ConstanciaLiquidacionEconomica{Referencia: "liquidacion:uno", ClaveIdempotencia: "clave_economia_20261014", NumeroDocumento: "VEC-D-2026-000001",
				VersionEntrada: 2, VersionResultante: 3, TitularPersonaRef: "persona:titular",
				DecisorPersonaRef: "persona:decisor", UnidadRef: "unidad:rrhh", Fecha: "2026-10-14T10:00:00Z",
				ReciboRef: "recibo:uno", AuditoriaRef: "auditoria:uno", EventoRef: "evento:uno"}
			if _, err := NuevaLiquidacionEconomica(vista, constancia); !errors.Is(err, ErrLiquidacionPublicada) {
				t.Fatalf("preview registrable: %v", err)
			}
			revision := []RevisionLineaLiquidacion{{Indice: 0, ReglaRef: s.Lineas[0].ReglaRef,
				ReconocidoPropuestoCentimos: caso.calculado, MotivoCodigo: caso.motivo}}
			decidida, err := preparar(revision)
			if err != nil {
				t.Fatal(err)
			}
			linea := decidida.Instantanea().Lineas[0]
			if decidida.DecisionPendiente() || linea.ReconocidoPropuestoCentimos != caso.calculado ||
				linea.RechazadoCentimos != 0 {
				t.Fatalf("decisión alterada: %+v", linea)
			}
			revision[0].ReconocidoPropuestoCentimos = caso.calculado + 1
			if _, err := preparar(revision); !errors.Is(err, ErrLiquidacionPublicada) {
				t.Fatalf("importe superior a publicado: %v", err)
			}
			revision[0].ReconocidoPropuestoCentimos, revision[0].MotivoCodigo = caso.calculado, ""
			if _, err := preparar(revision); !errors.Is(err, ErrLiquidacionPublicada) {
				t.Fatalf("diferencia sin motivo: %v", err)
			}
		})
	}
}

func TestReglaPublicadaCambiaTramoSoloConHuellaNueva(t *testing.T) {
	base := preparacionPublicadaPrueba(t).Instantanea()
	c := base.Catalogo
	c.VersionTarifaRef = "tarifa:Provisional:2026"
	c.HuellaSHA256, _ = c.HuellaCalculada()
	if c.ValidarForma() == nil {
		t.Fatal("tarifa provisional rotulada publicada")
	}
	c = base.Catalogo
	c.ReglaDevengo.Configuracion.PorcentajeMismoDia = 75
	c.HuellaSHA256, _ = c.HuellaCalculada()
	if c.ValidarForma() == nil {
		t.Fatal("se aceptó una regla alterada conservando huella")
	}
	c.ReglaDevengo.HuellaSHA256, _ = c.ReglaDevengo.HuellaCalculada()
	c.HuellaSHA256, _ = c.HuellaCalculada()
	inicio, _ := ResolverInstanteCivil(base.FechaInicio, base.HoraInicio, mustZonaPublicada(t))
	fin, _ := ResolverInstanteCivil(base.FechaFin, base.HoraFin, mustZonaPublicada(t))
	r, err := CalcularTramosNacionalesPublicados(inicio, fin, c)
	if err != nil || len(r.Tramos) != 1 || r.Tramos[0].Porcentaje != 75 || r.Tramos[0].ImporteCentimos != 750 {
		t.Fatalf("regla publicada no aplicada: %+v %v", r, err)
	}
}

func TestLiquidacionPublicadaRecalculaKilometrosExactosSinConsultarRuta(t *testing.T) {
	base := preparacionPublicadaPrueba(t).Instantanea()
	c := base.Catalogo
	c.Vehiculo = "propio"
	c.Reglas = append(c.Reglas, ReglaLiquidacionPublicada{ReglaLiquidacionPropuesta: ReglaLiquidacionPropuesta{
		Referencia: "regla:kilometraje", Tipo: "kilometraje", Concepto: "vehiculo_propio", CentimosPorKM: 40}, NormaRef: "norma:boe"})
	c.HuellaSHA256, _ = c.HuellaCalculada()
	d := DocumentoComision{VehiculoPropio: true, GrupoDieta: 2, VersionTarifaAceptada: c.VersionTarifaRef,
		Lineas: []LineaDocumentoComision{{Tipo: "kilometraje", RutaIndice: 1,
			OrigenCodigo: "18087", DestinoCodigo: "18003", VersionGrafo: "grafo:interno:uno",
			KilometrosBase: "11.0000", AjusteKilometros: "1.0000", MotivoAjuste: "Desvío justificado",
			Kilometros: "12.0000", ImporteCentimos: 312, VersionTarifaRef: c.VersionTarifaRef, Rotulo: c.RotuloClave}},
		KilometrajeCentimos: 312, TotalOrientativoCentimos: 312}
	lineas, totales, err := CalcularBasesLiquidacionPublicada(base.FechaInicio, base.HoraInicio,
		base.FechaFin, base.HoraFin, d, c)
	if err != nil || len(lineas) != 1 || lineas[0].CalculadoPublicadoCentimos != 480 ||
		lineas[0].DiferenciaCentimos != 168 || totales.CalculadoPublicadoCentimos != 480 {
		t.Fatalf("kilometraje publicado: %+v %+v %v", lineas, totales, err)
	}
	d.Lineas[0].Kilometros = "13.0000"
	if _, _, err := CalcularBasesLiquidacionPublicada(base.FechaInicio, base.HoraInicio,
		base.FechaFin, base.HoraFin, d, c); !errors.Is(err, ErrLiquidacionPublicada) {
		t.Fatalf("ruta alterada admitida: %v", err)
	}
}

func mustZonaPublicada(t *testing.T) *time.Location {
	t.Helper()
	z, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatal(err)
	}
	return z
}
