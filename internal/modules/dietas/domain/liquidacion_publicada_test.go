package domain

import (
	"errors"
	"strings"
	"testing"
)

func preparacionPublicadaPrueba(t *testing.T) *PreparacionLiquidacionPublicada {
	t.Helper()
	indice := 0
	d := DocumentoComision{GrupoDieta: 2, VersionTarifaAceptada: "tarifa:publicada:v1", TramosAceptados: []int{0},
		Lineas: []LineaDocumentoComision{{Tipo: "dieta", Grupo: 2, IndiceTramo: &indice, Fecha: "2026-10-01", Concepto: "manutencion",
			ImporteCentimos: 500, VersionTarifaRef: "tarifa:publicada:v1", Rotulo: "dietas_tarifa_publicada"}}, ManutencionCentimos: 500, TotalOrientativoCentimos: 500}
	dh, err := HuellaDatosLiquidacion(d)
	if err != nil {
		t.Fatal(err)
	}
	c := CatalogoLiquidacionPublicado{Referencia: "catalogo:liquidacion:publicado", Version: "version:uno", PublicacionRef: "publicacion:acto:uno",
		FechaPublicacion: "2026-09-20", VigenteDesde: "2026-10-01", VigenteHasta: "2026-12-31", Ambito: "granada:ordinario",
		PaisISO2: "ES", Grupo: 2, Vehiculo: "sin_vehiculo_propio", VersionTarifaRef: "tarifa:publicada:v1", RotuloClave: "dietas_tarifa_publicada",
		Normas: []NormaLiquidacionPublicada{{Referencia: "norma:bases", Articulo: "30", URL: "https://www.dipgra.es/bases"},
			{Referencia: "norma:boe", Articulo: "10", URL: "https://www.boe.es/boe"}},
		Reglas: []ReglaLiquidacionPublicada{{ReglaLiquidacionPropuesta: ReglaLiquidacionPropuesta{Referencia: "regla:manutencion:dos", Tipo: "dieta", Concepto: "manutencion", Grupo: 2, TopeCentimos: 1000}, NormaRef: "norma:bases"}}}
	c.ReglaDevengo = ReglaDevengoPublicada{Referencia: "regla:devengo:uno", NormaRef: "norma:bases", Configuracion: ConfiguracionDevengoPublicada{
		Esquema: "nacional_ordinaria_v1", Zona: "Europe/Madrid", DuracionMinimaMismoDiaHoras: 5,
		HoraSalida100AntesDe: 14, HoraSalida50AntesDe: 22, HoraRegreso50DespuesDe: 14,
		HoraRegresoMismoDiaDespuesDe: 16, DiasMaximos: 31, PorcentajeMismoDia: 50,
		PorcentajeSalidaTemprana: 100, PorcentajeSalidaMedia: 50, PorcentajeRegreso: 50,
		PorcentajeIntermedio: 100, Alojamiento: "cerrado_pendiente_rrhh"}}
	c.ReglaDevengo.HuellaSHA256, err = c.ReglaDevengo.HuellaCalculada()
	if err != nil {
		t.Fatal(err)
	}
	c.HuellaSHA256, err = c.HuellaCalculada()
	if err != nil {
		t.Fatal(err)
	}
	p, err := PrepararLiquidacionPublicada("dco_aaaaaaaaaaaaaaaaaaaaaa", 2, "2026-10-01", "08:00", "2026-10-01", "18:00", dh, d, c,
		[]RevisionLineaLiquidacion{{Indice: 0, ReglaRef: "regla:manutencion:dos", ReconocidoPropuestoCentimos: 400, MotivoCodigo: "revision_importe"}})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestPreparacionPublicadaConservaLineasYNoAdmiteEjemplo(t *testing.T) {
	p := preparacionPublicadaPrueba(t)
	s := p.Instantanea()
	if s.Totales.OriginalCentimos != 500 || s.Totales.ReconocidoPropuestoCentimos != 400 || s.Totales.RechazadoCentimos != 100 {
		t.Fatal(s.Totales)
	}
	if _, err := RecuperarPreparacionLiquidacionPublicada(s); err != nil {
		t.Fatal(err)
	}
	s.Lineas[0].ReconocidoPropuestoCentimos = 500
	if _, err := RecuperarPreparacionLiquidacionPublicada(s); err == nil {
		t.Fatal("instantanea alterada")
	}
	s = p.Instantanea()
	s.Catalogo.VersionTarifaRef = "provisional:ejemplo:20261001"
	s.Catalogo.HuellaSHA256, _ = s.Catalogo.HuellaCalculada()
	if _, err := PrepararLiquidacionPublicada(s.ComisionRef, s.ComisionVersion, s.FechaInicio, s.HoraInicio, s.FechaFin, s.HoraFin, s.DocumentoSHA256,
		s.Documento, s.Catalogo, []RevisionLineaLiquidacion{{Indice: 0, ReglaRef: s.Lineas[0].ReglaRef, ReconocidoPropuestoCentimos: 400, MotivoCodigo: "revision_importe"}}); err == nil {
		t.Fatal("tarifa provisional admitida")
	}
}

func TestPreparacionPublicadaAlojamientoPendiente(t *testing.T) {
	s := preparacionPublicadaPrueba(t).Instantanea()
	s.Documento.AlojamientoTopeCentimos = 1
	_, err := PrepararLiquidacionPublicada(s.ComisionRef, s.ComisionVersion, s.FechaInicio, s.HoraInicio, s.FechaFin, s.HoraFin, s.DocumentoSHA256,
		s.Documento, s.Catalogo, []RevisionLineaLiquidacion{{Indice: 0, ReglaRef: s.Lineas[0].ReglaRef, ReconocidoPropuestoCentimos: 400, MotivoCodigo: "revision_importe"}})
	if !errors.Is(err, ErrAlojamientoLiquidacionPendiente) {
		t.Fatal(err)
	}
}

func TestLiquidacionEconomicaExigeConstanciaSeparadaYRecuperable(t *testing.T) {
	p := preparacionPublicadaPrueba(t)
	c := ConstanciaLiquidacionEconomica{Referencia: "liquidacion:uno", ClaveIdempotencia: "clave_economia_20261014", NumeroDocumento: "VEC-D-2026-000001", VersionEntrada: 2, VersionResultante: 3,
		TitularPersonaRef: "persona:titular", DecisorPersonaRef: "persona:decisor", UnidadRef: "unidad:rrhh", Fecha: "2026-10-14T10:00:00Z",
		ReciboRef: "recibo:uno", AuditoriaRef: "auditoria:uno", EventoRef: "evento:uno"}
	l, err := NuevaLiquidacionEconomica(p, c)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := RecuperarLiquidacionEconomica(l.Instantanea()); err != nil {
		t.Fatal(err)
	}
	c.ClaveIdempotencia = "invalida"
	if _, err := NuevaLiquidacionEconomica(p, c); err == nil {
		t.Fatal("clave de idempotencia mal formada")
	}
	c.ClaveIdempotencia = "clave_economia_20261014"
	c.DecisorPersonaRef = c.TitularPersonaRef
	if _, err := NuevaLiquidacionEconomica(p, c); err == nil {
		t.Fatal("autoliquidacion")
	}
	c.DecisorPersonaRef = "persona:decisor"
	c.ReciboRef = strings.Repeat("x", 129)
	if _, err := NuevaLiquidacionEconomica(p, c); err == nil {
		t.Fatal("recibo mal formado")
	}
}
