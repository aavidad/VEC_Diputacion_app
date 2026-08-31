package ports

import (
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

func TestCTLITEO706AClaveOperacionGINPIXLigaTodosLosEjes(t *testing.T) {
	base := datosClaveOperacionGINPIXPrueba()
	referencia := referenciaOperacionGINPIX(base)
	if !referenciaClaveOperacionGINPIXValida(referencia) ||
		referencia != referenciaOperacionGINPIX(base) {
		t.Fatalf("clave no canonica: %q", referencia)
	}

	mutaciones := map[string]func(*DatosOperacionGINPIX){
		"orden":                func(d *DatosOperacionGINPIX) { d.OrdenHuellaSHA256 = strings.Repeat("b", 64) },
		"version expediente":   func(d *DatosOperacionGINPIX) { d.VersionExpediente++ },
		"expediente":           func(d *DatosOperacionGINPIX) { d.ExpedienteRef += "-otro" },
		"incorporacion":        func(d *DatosOperacionGINPIX) { d.IncorporacionRef += "-otra" },
		"recibo incorporacion": func(d *DatosOperacionGINPIX) { d.ReciboIncorporacionRef += "-otro" },
		"resultado Personal":   func(d *DatosOperacionGINPIX) { d.ResultadoPersonalRef += "-otro" },
		"recibo Personal":      func(d *DatosOperacionGINPIX) { d.ReciboPersonalRef += "-otro" },
		"correlacion":          func(d *DatosOperacionGINPIX) { d.CorrelacionRef += "-otra" },
		"idempotencia":         func(d *DatosOperacionGINPIX) { d.IdempotenciaRef += "-otra" },
		"procedencia modelo":   func(d *DatosOperacionGINPIX) { d.ProcedenciaModeloRef += "-otra" },
		"modelo":               func(d *DatosOperacionGINPIX) { d.ModeloHuellaSHA256 = strings.Repeat("c", 64) },
		"mapeo":                func(d *DatosOperacionGINPIX) { d.MapeoRef += "-otro" },
		"version mapeo":        func(d *DatosOperacionGINPIX) { d.MapeoVersion++ },
		"procedencia mapeo":    func(d *DatosOperacionGINPIX) { d.ProcedenciaMapeoRef += "-otra" },
		"huella mapeo":         func(d *DatosOperacionGINPIX) { d.MapeoHuellaSHA256 = strings.Repeat("d", 64) },
		"carga":                func(d *DatosOperacionGINPIX) { d.CargaHuellaSHA256 = strings.Repeat("e", 64) },
	}
	for nombre, mutar := range mutaciones {
		t.Run(nombre, func(t *testing.T) {
			variante := base
			mutar(&variante)
			if obtenida := referenciaOperacionGINPIX(variante); obtenida == referencia {
				t.Fatalf("el eje no altera la clave: %q", obtenida)
			}
		})
	}
}

func TestCTLITEO706AHuellaOrdenIncluyeReciboCompletoYDocumentos(t *testing.T) {
	instante := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)
	evidencia := EvidenciaOrdenConfirmarIncorporacion{EvaluadaEn: instante}
	recibo := ReciboConfirmacionIncorporacion{
		ReciboRef: "recibo-incorporacion", ActuacionRef: "actuacion-incorporacion",
		CorrelacionRef: "correlacion", ActorRef: "actor", OrganizacionRef: "organizacion",
		UnidadRef: "unidad", ExpedienteRef: "expediente", SolicitudPersonalRef: "solicitud-personal",
		DecisionAutorizacionRef: "decision", ResultadoPersonalRef: "resultado-personal",
		ReciboPersonalRef: "recibo-personal", RelacionRef: "relacion", OcupacionRef: "ocupacion",
		TransicionClave: TransicionConfirmarIncorporacion, MotivoClave: "incorporacion_confirmada",
		VersionExpediente: 7, VersionAnterior: 2, VersionResultante: 3,
		FechaIncorporacion: instante.Add(24 * time.Hour), FechaFinPrevista: instante.AddDate(0, 1, 0),
		ConfirmadaEn: instante,
		Documentos:   []domain.DocumentoSeguimiento{{TipoClave: "resolucion", Referencia: "documento-1"}},
	}
	huella := huellaOrdenOperacionGINPIX(evidencia, recibo)
	if !huellaOperacionGINPIXValida(huella) ||
		huella != huellaOrdenOperacionGINPIX(evidencia, recibo) {
		t.Fatalf("huella de orden no canonica: %q", huella)
	}
	variante := recibo
	variante.Documentos = append([]domain.DocumentoSeguimiento(nil), recibo.Documentos...)
	variante.Documentos[0].Referencia = "documento-2"
	if huellaOrdenOperacionGINPIX(evidencia, variante) == huella {
		t.Fatal("un documento variante conserva la huella de orden")
	}
	evidencia.EvaluadaEn = evidencia.EvaluadaEn.Add(time.Nanosecond)
	if huellaOrdenOperacionGINPIX(evidencia, recibo) == huella {
		t.Fatal("una orden evaluada en otro instante conserva la huella")
	}
}

func TestCTLITEO706AEstadosYValoresCeroFallanCerrados(t *testing.T) {
	var solicitud SolicitudOperacionGINPIX
	if !errors.Is(solicitud.Validar(), ErrSolicitudOperacionGINPIXInvalida) {
		t.Fatal("solicitud cero aceptada")
	}
	if _, err := solicitud.Datos(); !errors.Is(err, ErrSolicitudOperacionGINPIXInvalida) {
		t.Fatalf("solicitud cero expuso datos: %v", err)
	}
	if err := (ReservaOperacionGINPIX{}).ValidarPara(solicitud); !errors.Is(
		err,
		ErrReservaOperacionGINPIXInvalida,
	) {
		t.Fatalf("reserva cero aceptada: %v", err)
	}
	if err := (ReciboExternoOperacionGINPIX{}).ValidarPara(solicitud); !errors.Is(
		err,
		ErrReciboExternoOperacionGINPIXInvalido,
	) {
		t.Fatalf("recibo externo cero aceptado: %v", err)
	}
	if err := (ResultadoOperacionGINPIX{}).ValidarPara(solicitud); !errors.Is(
		err,
		ErrResultadoOperacionGINPIXInvalido,
	) {
		t.Fatalf("resultado cero aceptado: %v", err)
	}
}

func datosClaveOperacionGINPIXPrueba() DatosOperacionGINPIX {
	return DatosOperacionGINPIX{
		OrdenHuellaSHA256: strings.Repeat("a", 64), VersionExpediente: 7,
		ExpedienteRef: "expediente-sintetico", IncorporacionRef: "incorporacion-sintetica",
		ReciboIncorporacionRef: "recibo-incorporacion-sintetico",
		ResultadoPersonalRef:   "resultado-personal-sintetico", ReciboPersonalRef: "recibo-personal-sintetico",
		CorrelacionRef: "correlacion-sintetica", IdempotenciaRef: "idempotencia-sintetica",
		ProcedenciaModeloRef: "procedencia-modelo-sintetica", ModeloHuellaSHA256: strings.Repeat("1", 64),
		MapeoRef: "mapeo-sintetico", MapeoVersion: 3,
		ProcedenciaMapeoRef: "procedencia-mapeo-sintetica", MapeoHuellaSHA256: strings.Repeat("2", 64),
		CargaHuellaSHA256: strings.Repeat("3", 64),
	}
}
