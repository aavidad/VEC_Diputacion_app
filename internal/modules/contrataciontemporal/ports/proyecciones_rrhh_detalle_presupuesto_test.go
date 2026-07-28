package ports

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestPresupuestoEntradaDetalleRRHHAdmiteBordeExactoYRechazaAdyacente(
	t *testing.T,
) {
	t.Parallel()
	resumen := ResumenExpedienteRRHH{}
	base, cabe := presupuestoEntradaDetalleRRHHMinimizada(
		resumen, SolicitudOperativaRRHH{},
		nil, ReferenciaHitoAnalisisRRHH{},
		nil, ReferenciaHitoCoberturaRRHH{},
		nil, ReferenciaHitoAsignacionRRHH{}, nil,
	)
	if !cabe || base >= limiteBytesEntradaDetalleRRHHMinimizada {
		t.Fatalf("presupuesto base inesperado: %d, %t", base, cabe)
	}
	relleno := limiteBytesEntradaDetalleRRHHMinimizada - base
	resumen.NumeroVisible = strings.Repeat("a", int(relleno))
	exacto, cabe := presupuestoEntradaDetalleRRHHMinimizada(
		resumen, SolicitudOperativaRRHH{},
		nil, ReferenciaHitoAnalisisRRHH{},
		nil, ReferenciaHitoCoberturaRRHH{},
		nil, ReferenciaHitoAsignacionRRHH{}, nil,
	)
	if !cabe || exacto != limiteBytesEntradaDetalleRRHHMinimizada {
		t.Fatalf("borde exacto rechazado: %d, %t", exacto, cabe)
	}
	resumen.NumeroVisible += "a"
	adyacente, cabe := presupuestoEntradaDetalleRRHHMinimizada(
		resumen, SolicitudOperativaRRHH{},
		nil, ReferenciaHitoAnalisisRRHH{},
		nil, ReferenciaHitoCoberturaRRHH{},
		nil, ReferenciaHitoAsignacionRRHH{}, nil,
	)
	if cabe || adyacente != limiteBytesEntradaDetalleRRHHMinimizada+1 {
		t.Fatalf("borde +1 aceptado: %d, %t", adyacente, cabe)
	}
}

func TestMedidorCadenaDetalleRRHHCubreUTF8YEscapesJSON(t *testing.T) {
	t.Parallel()
	for nombre, caso := range map[string]struct {
		valor    string
		esperado uint64
	}{
		"vacia":           {"", 2},
		"ascii":           {"a", 3},
		"acentuada":       {"á", 4},
		"emoji":           {"🙂", 6},
		"comilla":         {`"`, 4},
		"barra_inversa":   {`\`, 4},
		"salto":           {"\n", 4},
		"escape_html":     {"<", 8},
		"separador_linea": {"\u2028", 8},
		"utf8_invalido":   {string([]byte{0xff}), 8},
	} {
		medidor := nuevoMedidorPresupuestoDetalleRRHH()
		medidor.cadena(caso.valor)
		obtenido := limiteBytesEntradaDetalleRRHHMinimizada -
			medidor.restante
		if medidor.excedido || obtenido != caso.esperado {
			t.Fatalf(
				"%s: tamaño=%d, esperado=%d, excedido=%t",
				nombre, obtenido, caso.esperado, medidor.excedido,
			)
		}
	}
}

func TestPresupuestoEntradaDetalleRRHHRechazaAntesDeClonar(t *testing.T) {
	hitos := make(
		[]HitoExpedienteRRHH,
		limiteMaximoHitosPorPresupuestoRRHH+1,
	)
	ejecutar := func() {
		_, err := NuevaEntradaDetalleExpedienteRRHHMinimizada(
			ResumenExpedienteRRHH{}, SolicitudOperativaRRHH{},
			nil, ReferenciaHitoAnalisisRRHH{},
			nil, ReferenciaHitoCoberturaRRHH{},
			nil, ReferenciaHitoAsignacionRRHH{}, hitos,
		)
		if err != ErrResultadoConsultaRRHHNoConfiable {
			panic("entrada fuera de presupuesto no rechazada")
		}
	}
	ejecutar()
	if asignaciones := testing.AllocsPerRun(100, ejecutar); asignaciones != 0 {
		t.Fatalf(
			"el rechazo reservó o clonó memoria: %.2f asignaciones",
			asignaciones,
		)
	}
}

func TestPresupuestoDetalleRRHHRechazaComprobacionesEnTiempoConstante(
	t *testing.T,
) {
	comprobaciones := make([]ComprobacionOperativaRRHH, 250_000)
	cobertura := &CoberturaOperativaRRHH{
		Comprobaciones: comprobaciones,
	}
	ejecutar := func() {
		_, err := NuevaEntradaDetalleExpedienteRRHHMinimizada(
			ResumenExpedienteRRHH{}, SolicitudOperativaRRHH{},
			nil, ReferenciaHitoAnalisisRRHH{},
			cobertura, ReferenciaHitoCoberturaRRHH{},
			nil, ReferenciaHitoAsignacionRRHH{}, nil,
		)
		if err != ErrResultadoConsultaRRHHNoConfiable {
			panic("cardinalidad de comprobaciones no rechazada")
		}
	}
	ejecutar()
	if asignaciones := testing.AllocsPerRun(1_000, ejecutar); asignaciones != 0 {
		t.Fatalf(
			"el rechazo O(1) reservó o clonó memoria: %.2f asignaciones",
			asignaciones,
		)
	}
}

func TestPresupuestoEntradaDetalleRRHHNoDesbordaConMaximos(t *testing.T) {
	t.Parallel()
	maximo := strings.Repeat("a", 1<<20)
	resumen := ResumenExpedienteRRHH{
		ExpedienteRef: maximo, OrganizacionRef: maximo,
		NumeroVisible: maximo, FlujoRef: maximo, FlujoHuella: maximo,
		CentroRef: maximo, CategoriaRef: maximo,
		ModalidadClave: "a", UnidadRef: maximo,
	}
	tamano, cabe := presupuestoEntradaDetalleRRHHMinimizada(
		resumen, SolicitudOperativaRRHH{GrupoSubgrupo: maximo},
		&AnalisisOperativoRRHH{
			CategoriaRef: maximo, FuenteCosteRef: maximo,
			CostePrevisto: &ImporteOperativoRRHH{Moneda: maximo},
		},
		ReferenciaHitoAnalisisRRHH{},
		&CoberturaOperativaRRHH{
			ProcedimientoRef: maximo, BolsaRef: maximo,
		},
		ReferenciaHitoCoberturaRRHH{},
		&AsignacionOperativaRRHH{UnidadRef: maximo},
		ReferenciaHitoAsignacionRRHH{}, nil,
	)
	if cabe || tamano != limiteBytesEntradaDetalleRRHHMinimizada+1 {
		t.Fatalf("máximos desbordados aceptados: %d, %t", tamano, cabe)
	}
}

func TestPresupuestoEntradaDetalleRRHHCubreSerializacionCerrada(t *testing.T) {
	t.Parallel()
	instante := time.Date(2026, 7, 28, 19, 0, 0, 123_456_000, time.UTC)
	resumen := ResumenExpedienteRRHH{
		ExpedienteRef: "expediente:á<&", OrganizacionRef: "organizacion:🙂",
		NumeroVisible: "2026/CT-\"01", Version: 4,
		FlujoRef: "flujo:rrhh:01", FlujoVersion: 1,
		FlujoHuella: strings.Repeat("a", 64),
		FaseClave:   "fase\u2028final", EstadoClave: "en_curso",
		CentroRef: "centro:rrhh:01", CategoriaRef: "categoria:rrhh:01",
		ModalidadClave: "interinidad", UnidadRef: "unidad:rrhh:01",
		CreadoEn: instante, ActualizadoEn: instante,
	}
	solicitud := SolicitudOperativaRRHH{
		GrupoSubgrupo: "C2", MotivoClave: "sustitucion",
		PeriodoInicio: instante, PeriodoFin: instante,
	}
	analisis := &AnalisisOperativoRRHH{
		ModalidadClave: "interinidad", CategoriaRef: "categoria:rrhh:01",
		CausaClave: "sustitucion", PeriodoInicio: instante,
		PeriodoFin: instante, PorcentajeJornada: 10_000,
		ResultadoRC: "no_requerida",
		CostePrevisto: &ImporteOperativoRRHH{
			Centimos: -9_223_372_036_854_775_808, Moneda: "E<&",
		},
		FuenteCosteRef: "fuente:coste:01",
	}
	cobertura := &CoberturaOperativaRRHH{
		ViaClave: "bolsa", ProcedimientoRef: "procedimiento:rrhh:01",
		BolsaRef: "bolsa:rrhh:01",
		Comprobaciones: []ComprobacionOperativaRRHH{{
			Clave: "disponibilidad", Resultado: "afirmativa",
		}},
	}
	asignacion := &AsignacionOperativaRRHH{
		UnidadRef: "unidad:rrhh:01", AsignadaEn: instante,
		MotivoClave: "necesidad_servicio",
	}
	hitos := []HitoExpedienteRRHH{{
		Secuencia: 1, VersionExpediente: 1,
		AccionClave: "actuacion\nuno", RealizadaEn: instante,
		FaseDestino: "solicitud", EstadoOrigen: "pendiente",
		EstadoDestino: "en_curso",
	}}
	estimado, cabe := presupuestoEntradaDetalleRRHHMinimizada(
		resumen, solicitud,
		analisis, ReferenciaHitoAnalisisRRHH{},
		cobertura, ReferenciaHitoCoberturaRRHH{},
		asignacion, ReferenciaHitoAsignacionRRHH{}, hitos,
	)
	if !cabe {
		t.Fatal("fixture cerrada excedió el presupuesto")
	}
	contenido, err := json.Marshal(DetalleExpedienteRRHH{
		Resumen: resumen, Solicitud: solicitud, Analisis: analisis,
		Cobertura: cobertura, Asignacion: asignacion, Hitos: hitos,
	})
	if err != nil {
		t.Fatal(err)
	}
	if uint64(len(contenido)) > estimado {
		t.Fatalf(
			"la cota %d quedó por debajo de la salida %d",
			estimado, len(contenido),
		)
	}
}
