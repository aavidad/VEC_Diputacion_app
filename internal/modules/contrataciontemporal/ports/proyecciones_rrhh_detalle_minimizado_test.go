package ports_test

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestDetalleRRHHMinimizadoAdmiteSoloCombinacionesProgresivas(t *testing.T) {
	t.Parallel()
	for _, bloques := range []int{0, 1, 2, 3} {
		bloques := bloques
		t.Run(fmt.Sprintf("%d_bloques", bloques), func(t *testing.T) {
			t.Parallel()
			entrada, lectura := entradaDetalleMinimizadaPrueba(t, bloques)
			detalle, err := ports.NuevoDetalleExpedienteRRHHMinimizado(
				entrada, lectura,
			)
			if err != nil {
				t.Fatalf("construir combinación %d: %v", bloques, err)
			}
			if (detalle.Analisis != nil) != (bloques >= 1) ||
				(detalle.Cobertura != nil) != (bloques >= 2) ||
				(detalle.Asignacion != nil) != (bloques >= 3) ||
				detalle.Resumen.Version != uint64(bloques+1) ||
				len(detalle.Hitos) != bloques+1 {
				t.Fatalf("proyección incoherente: %#v", detalle)
			}
			contenido, err := json.Marshal(detalle)
			if err != nil {
				t.Fatalf("serializar proyección minimizada: %v", err)
			}
			if len(contenido) > 256*1024 {
				t.Fatalf("salida fuera del límite protegido: %d", len(contenido))
			}
			for _, prohibido := range []string{
				"lectura:rrhh:minimizada", "auditoria:rrhh:minimizada",
				"decision:rrhh:minimizada", "sesion:rrhh:minimizada",
			} {
				if strings.Contains(string(contenido), prohibido) {
					t.Fatalf("la salida filtró recibo %q: %s", prohibido, contenido)
				}
			}
		})
	}
}

func TestDetalleRRHHMinimizadoRechazaBloquesYVinculosIncoherentes(t *testing.T) {
	t.Parallel()
	datos := datosDetalleMinimizadoPrueba(3)
	refAnalisis := referenciaAnalisisMinimizadaPrueba(t, 2)
	refCobertura := referenciaCoberturaMinimizadaPrueba(t, 3)
	refAsignacion := referenciaAsignacionMinimizadaPrueba(t, 4)

	casosEntrada := map[string]func() (ports.EntradaDetalleExpedienteRRHHMinimizada, error){
		"analisis_sin_vinculo": func() (ports.EntradaDetalleExpedienteRRHHMinimizada, error) {
			return ports.NuevaEntradaDetalleExpedienteRRHHMinimizada(
				datos.resumen, datos.solicitud, datos.analisis,
				ports.ReferenciaHitoAnalisisRRHH{},
				datos.cobertura, refCobertura,
				datos.asignacion, refAsignacion, datos.hitos,
			)
		},
		"vinculo_analisis_sin_bloque": func() (ports.EntradaDetalleExpedienteRRHHMinimizada, error) {
			return ports.NuevaEntradaDetalleExpedienteRRHHMinimizada(
				datos.resumen, datos.solicitud, nil, refAnalisis,
				nil, ports.ReferenciaHitoCoberturaRRHH{},
				nil, ports.ReferenciaHitoAsignacionRRHH{}, datos.hitos,
			)
		},
		"asignacion_sin_vinculo": func() (ports.EntradaDetalleExpedienteRRHHMinimizada, error) {
			return ports.NuevaEntradaDetalleExpedienteRRHHMinimizada(
				datos.resumen, datos.solicitud, datos.analisis, refAnalisis,
				datos.cobertura, refCobertura, datos.asignacion,
				ports.ReferenciaHitoAsignacionRRHH{}, datos.hitos,
			)
		},
	}
	for nombre, construir := range casosEntrada {
		if _, err := construir(); !errors.Is(
			err, ports.ErrResultadoConsultaRRHHNoConfiable,
		) {
			t.Fatalf("%s aceptado: %v", nombre, err)
		}
	}

	analisisCruzado := referenciaAnalisisMinimizadaPrueba(t, 3)
	coberturaCruzada := referenciaCoberturaMinimizadaPrueba(t, 2)
	entrada, err := ports.NuevaEntradaDetalleExpedienteRRHHMinimizada(
		datos.resumen, datos.solicitud, datos.analisis, analisisCruzado,
		datos.cobertura, coberturaCruzada,
		datos.asignacion, refAsignacion, datos.hitos,
	)
	if err != nil {
		t.Fatalf("preparar cruce nominal: %v", err)
	}
	lectura := reciboDetalleMinimizadoPrueba(
		t, datos.resumen.ExpedienteRef, datos.resumen.Version,
		datos.resumen.ActualizadoEn,
	)
	if _, err = ports.NuevoDetalleExpedienteRRHHMinimizado(
		entrada, lectura,
	); !errors.Is(err, ports.ErrResultadoConsultaRRHHNoConfiable) {
		t.Fatalf("vínculos de análisis/cobertura cruzados aceptados: %v", err)
	}

	for nombre, mutar := range map[string]func(*datosDetalleMinimizado){
		"cobertura_sin_analisis": func(d *datosDetalleMinimizado) {
			d.analisis = nil
			d.resumen.ModalidadClave = ""
		},
		"asignacion_sin_cobertura": func(d *datosDetalleMinimizado) {
			d.cobertura = nil
		},
	} {
		datos := datosDetalleMinimizadoPrueba(3)
		mutar(&datos)
		entrada, err := nuevaEntradaDetalleMinimizadaPrueba(t, datos)
		if err != nil {
			t.Fatalf("preparar %s: %v", nombre, err)
		}
		lectura := reciboDetalleMinimizadoPrueba(
			t, datos.resumen.ExpedienteRef, datos.resumen.Version,
			datos.resumen.ActualizadoEn,
		)
		if _, err = ports.NuevoDetalleExpedienteRRHHMinimizado(
			entrada, lectura,
		); !errors.Is(err, ports.ErrResultadoConsultaRRHHNoConfiable) {
			t.Fatalf("%s aceptado: %v", nombre, err)
		}
	}
}

func TestReferenciasHitoDetalleRRHHMinimizadoRechazanLimites(t *testing.T) {
	t.Parallel()
	const fueraDelEnteroJSONSeguro = uint64(9_007_199_254_740_992)
	for nombre, construir := range map[string]func(uint64) error{
		"analisis": func(secuencia uint64) error {
			_, err := ports.NuevaReferenciaHitoAnalisisRRHH(secuencia)
			return err
		},
		"cobertura": func(secuencia uint64) error {
			_, err := ports.NuevaReferenciaHitoCoberturaRRHH(secuencia)
			return err
		},
		"asignacion": func(secuencia uint64) error {
			_, err := ports.NuevaReferenciaHitoAsignacionRRHH(secuencia)
			return err
		},
	} {
		for _, secuencia := range []uint64{0, 1, fueraDelEnteroJSONSeguro} {
			if err := construir(secuencia); !errors.Is(
				err, ports.ErrResultadoConsultaRRHHNoConfiable,
			) {
				t.Fatalf("%s aceptó secuencia %d: %v", nombre, secuencia, err)
			}
		}
		if err := construir(2); err != nil {
			t.Fatalf("%s rechazó la primera secuencia válida: %v", nombre, err)
		}
	}
}

func TestDetalleRRHHMinimizadoExigeOrdenEstrictoDeBloques(t *testing.T) {
	t.Parallel()
	for nombre, secuencias := range map[string][3]uint64{
		"analisis_tras_cobertura":    {3, 2, 4},
		"cobertura_tras_asignacion":  {2, 4, 3},
		"analisis_igual_cobertura":   {2, 2, 4},
		"cobertura_igual_asignacion": {2, 3, 3},
	} {
		datos := datosDetalleMinimizadoPrueba(3)
		entrada, err := ports.NuevaEntradaDetalleExpedienteRRHHMinimizada(
			datos.resumen, datos.solicitud,
			datos.analisis,
			referenciaAnalisisMinimizadaPrueba(t, secuencias[0]),
			datos.cobertura,
			referenciaCoberturaMinimizadaPrueba(t, secuencias[1]),
			datos.asignacion,
			referenciaAsignacionMinimizadaPrueba(t, secuencias[2]),
			datos.hitos,
		)
		if err != nil {
			t.Fatalf("preparar %s: %v", nombre, err)
		}
		lectura := reciboDetalleMinimizadoPrueba(
			t, datos.resumen.ExpedienteRef, datos.resumen.Version,
			datos.resumen.ActualizadoEn,
		)
		if _, err = ports.NuevoDetalleExpedienteRRHHMinimizado(
			entrada, lectura,
		); !errors.Is(err, ports.ErrResultadoConsultaRRHHNoConfiable) {
			t.Fatalf("%s aceptado: %v", nombre, err)
		}
	}
}

func TestDetalleRRHHMinimizadoRechazaFasesVersionesYReciboCruzados(t *testing.T) {
	t.Parallel()
	for nombre, mutar := range map[string]func(*datosDetalleMinimizado){
		"version_hito": func(d *datosDetalleMinimizado) {
			d.hitos[1].VersionExpediente++
		},
		"secuencia_hito": func(d *datosDetalleMinimizado) {
			d.hitos[2].Secuencia++
		},
		"fase_origen": func(d *datosDetalleMinimizado) {
			d.hitos[2].FaseOrigen = "fase:ajena"
		},
		"estado_origen": func(d *datosDetalleMinimizado) {
			d.hitos[2].EstadoOrigen = domain.EstadoPendiente
		},
		"fase_final": func(d *datosDetalleMinimizado) {
			d.resumen.FaseClave = "fase:ajena"
		},
	} {
		datos := datosDetalleMinimizadoPrueba(3)
		mutar(&datos)
		entrada, err := nuevaEntradaDetalleMinimizadaPrueba(t, datos)
		if errors.Is(err, ports.ErrResultadoConsultaRRHHNoConfiable) {
			continue
		}
		if err != nil {
			t.Fatalf("%s produjo error inesperado: %v", nombre, err)
		}
		lectura := reciboDetalleMinimizadoPrueba(
			t, datos.resumen.ExpedienteRef, datos.resumen.Version,
			datos.resumen.ActualizadoEn,
		)
		if _, err = ports.NuevoDetalleExpedienteRRHHMinimizado(
			entrada, lectura,
		); !errors.Is(err, ports.ErrResultadoConsultaRRHHNoConfiable) {
			t.Fatalf("%s aceptado: %v", nombre, err)
		}
	}

	datos := datosDetalleMinimizadoPrueba(3)
	entrada := construirEntradaDetalleMinimizadaPrueba(t, datos)
	lecturaOtraVersion := reciboDetalleMinimizadoPrueba(
		t, datos.resumen.ExpedienteRef, datos.resumen.Version-1,
		datos.resumen.ActualizadoEn,
	)
	if _, err := ports.NuevoDetalleExpedienteRRHHMinimizado(
		entrada, lecturaOtraVersion,
	); !errors.Is(err, ports.ErrResultadoConsultaRRHHNoConfiable) {
		t.Fatalf("recibo de otra versión aceptado: %v", err)
	}
}

func TestDetalleRRHHMinimizadoRealizaCopiasDefensivas(t *testing.T) {
	t.Parallel()
	datos := datosDetalleMinimizadoPrueba(3)
	costeOriginal := datos.analisis.CostePrevisto.Centimos
	comprobacionOriginal := datos.cobertura.Comprobaciones[0].Clave
	hitoOriginal := datos.hitos[0].AccionClave
	entrada := construirEntradaDetalleMinimizadaPrueba(t, datos)

	datos.analisis.CostePrevisto.Centimos++
	datos.cobertura.Comprobaciones[0].Clave = "comprobacion:alterada"
	datos.hitos[0].AccionClave = "actuacion:alterada"
	lectura := reciboDetalleMinimizadoPrueba(
		t, datos.resumen.ExpedienteRef, datos.resumen.Version,
		datos.resumen.ActualizadoEn,
	)
	detalle, err := ports.NuevoDetalleExpedienteRRHHMinimizado(
		entrada, lectura,
	)
	if err != nil {
		t.Fatal(err)
	}
	if detalle.Analisis.CostePrevisto.Centimos != costeOriginal ||
		detalle.Cobertura.Comprobaciones[0].Clave != comprobacionOriginal ||
		detalle.Hitos[0].AccionClave != hitoOriginal {
		t.Fatal("la entrada conservó alias mutables del adaptador")
	}

	clon := detalle.Clonar()
	clon.Analisis.CostePrevisto.Centimos++
	clon.Cobertura.Comprobaciones[0].Clave = "comprobacion:clon"
	clon.Hitos[0].AccionClave = "actuacion:clon"
	if detalle.Analisis.CostePrevisto.Centimos != costeOriginal ||
		detalle.Cobertura.Comprobaciones[0].Clave != comprobacionOriginal ||
		detalle.Hitos[0].AccionClave != hitoOriginal {
		t.Fatal("Clonar dejó alias mutables en la proyección")
	}
}

func TestEntradaYReferenciasDetalleRRHHMinimizadoSonOpacas(t *testing.T) {
	t.Parallel()
	entrada, lectura := entradaDetalleMinimizadaPrueba(t, 3)
	referencia := referenciaAnalisisMinimizadaPrueba(t, 2)
	for nombre, valor := range map[string]any{
		"entrada": entrada, "referencia": referencia, "recibo": lectura,
	} {
		comprobarOpacidadDetalleMinimizado(t, nombre, valor)
	}
}

func comprobarOpacidadDetalleMinimizado(
	t *testing.T,
	nombre string,
	valor any,
) {
	t.Helper()
	for codec, err := range map[string]error{
		"json": func() error { _, err := json.Marshal(valor); return err }(),
		"xml":  func() error { _, err := xml.Marshal(valor); return err }(),
		"gob": func() error {
			var salida bytes.Buffer
			return gob.NewEncoder(&salida).Encode(valor)
		}(),
	} {
		if !errors.Is(err, ports.ErrMaterialConsultaRRHHSensible) {
			t.Fatalf("%s/%s no quedó bloqueado: %v", nombre, codec, err)
		}
	}
	var bitacora bytes.Buffer
	slog.New(slog.NewJSONHandler(&bitacora, nil)).Info(
		"material", "valor", valor,
	)
	formato := fmt.Sprintf("%v %+v %#v", valor, valor, valor)
	if !strings.Contains(formato, "MATERIAL-CONSULTA-RRHH-OPACO") ||
		!strings.Contains(bitacora.String(), "MATERIAL-CONSULTA-RRHH-OPACO") {
		t.Fatalf("%s no se redacta: fmt=%q slog=%q", nombre, formato, bitacora.String())
	}
	for _, sensible := range []string{
		"expediente:rrhh:minimizado", "2026/CT-MIN",
		"lectura:rrhh:minimizada", "auditoria:rrhh:minimizada",
	} {
		if strings.Contains(formato, sensible) ||
			strings.Contains(bitacora.String(), sensible) {
			t.Fatalf("%s filtró %q", nombre, sensible)
		}
	}
}

type datosDetalleMinimizado struct {
	resumen    ports.ResumenExpedienteRRHH
	solicitud  ports.SolicitudOperativaRRHH
	analisis   *ports.AnalisisOperativoRRHH
	cobertura  *ports.CoberturaOperativaRRHH
	asignacion *ports.AsignacionOperativaRRHH
	hitos      []ports.HitoExpedienteRRHH
}

func entradaDetalleMinimizadaPrueba(
	t *testing.T,
	bloques int,
) (
	ports.EntradaDetalleExpedienteRRHHMinimizada,
	ports.ReciboLecturaRRHH,
) {
	t.Helper()
	datos := datosDetalleMinimizadoPrueba(bloques)
	entrada := construirEntradaDetalleMinimizadaPrueba(t, datos)
	lectura := reciboDetalleMinimizadoPrueba(
		t, datos.resumen.ExpedienteRef, datos.resumen.Version,
		datos.resumen.ActualizadoEn,
	)
	return entrada, lectura
}

func construirEntradaDetalleMinimizadaPrueba(
	t *testing.T,
	datos datosDetalleMinimizado,
) ports.EntradaDetalleExpedienteRRHHMinimizada {
	t.Helper()
	entrada, err := nuevaEntradaDetalleMinimizadaPrueba(t, datos)
	if err != nil {
		t.Fatalf("crear entrada minimizada: %v", err)
	}
	return entrada
}

func nuevaEntradaDetalleMinimizadaPrueba(
	t *testing.T,
	datos datosDetalleMinimizado,
) (ports.EntradaDetalleExpedienteRRHHMinimizada, error) {
	t.Helper()
	var refAnalisis ports.ReferenciaHitoAnalisisRRHH
	var refCobertura ports.ReferenciaHitoCoberturaRRHH
	var refAsignacion ports.ReferenciaHitoAsignacionRRHH
	if datos.analisis != nil {
		refAnalisis = referenciaAnalisisMinimizadaPrueba(t, 2)
	}
	if datos.cobertura != nil {
		refCobertura = referenciaCoberturaMinimizadaPrueba(t, 3)
	}
	if datos.asignacion != nil {
		refAsignacion = referenciaAsignacionMinimizadaPrueba(
			t, uint64(len(datos.hitos)),
		)
	}
	return ports.NuevaEntradaDetalleExpedienteRRHHMinimizada(
		datos.resumen, datos.solicitud, datos.analisis, refAnalisis,
		datos.cobertura, refCobertura,
		datos.asignacion, refAsignacion, datos.hitos,
	)
}

func datosDetalleMinimizadoPrueba(bloques int) datosDetalleMinimizado {
	ahora := instantePuertosRRHH()
	version := uint64(bloques + 1)
	fases := []domain.ClaveFase{
		"solicitud", "gestion_bolsa", "asignacion_unidad", "unidad_gestora",
	}
	hitos := make([]ports.HitoExpedienteRRHH, version)
	for indice := range hitos {
		secuencia := uint64(indice + 1)
		faseOrigen := domain.ClaveFase("")
		estadoOrigen := domain.EstadoPendiente
		if indice > 0 {
			faseOrigen = fases[indice-1]
			estadoOrigen = domain.EstadoEnCurso
		}
		hitos[indice] = ports.HitoExpedienteRRHH{
			Secuencia: secuencia, VersionExpediente: secuencia,
			AccionClave: domain.ClaveCatalogo(fmt.Sprintf(
				"actuacion.minimizada.%d", secuencia,
			)),
			RealizadaEn: ahora.Add(time.Duration(indice-bloques) * time.Minute),
			FaseOrigen:  faseOrigen, FaseDestino: fases[indice],
			EstadoOrigen: estadoOrigen, EstadoDestino: domain.EstadoEnCurso,
		}
	}
	inicioBase := ahora.AddDate(0, 1, 0)
	inicio := time.Date(
		inicioBase.Year(), inicioBase.Month(), inicioBase.Day(),
		0, 0, 0, 0, time.UTC,
	)
	datos := datosDetalleMinimizado{
		resumen: ports.ResumenExpedienteRRHH{
			ExpedienteRef:   "expediente:rrhh:minimizado",
			OrganizacionRef: "organizacion:diputacion-granada",
			NumeroVisible:   "2026/CT-MIN", Version: version,
			FlujoRef: "flujo:rrhh:minimizado", FlujoVersion: 1,
			FlujoHuella: strings.Repeat("a", 64),
			FaseClave:   fases[bloques], EstadoClave: domain.EstadoEnCurso,
			CentroRef:    "centro:rrhh:minimizado",
			CategoriaRef: "categoria:rrhh:minimizada",
			CreadoEn:     hitos[0].RealizadaEn, ActualizadoEn: ahora,
		},
		solicitud: ports.SolicitudOperativaRRHH{
			GrupoSubgrupo: "C2", MotivoClave: "sustitucion",
			PeriodoInicio: inicio, PeriodoFin: inicio.AddDate(0, 1, 0),
		},
		hitos: hitos,
	}
	if bloques >= 1 {
		datos.resumen.ModalidadClave = "interinidad"
		datos.analisis = &ports.AnalisisOperativoRRHH{
			ModalidadClave: "interinidad",
			CategoriaRef:   datos.resumen.CategoriaRef,
			CausaClave:     "sustitucion", PeriodoInicio: inicio,
			PeriodoFin:        inicio.AddDate(0, 1, 0),
			PorcentajeJornada: 10_000, ResultadoRC: domain.RCNoRequerida,
			CostePrevisto: &ports.ImporteOperativoRRHH{
				Centimos: 125_000, Moneda: "EUR",
			},
			FuenteCosteRef: "fuente:coste:minimizada",
		}
	}
	if bloques >= 2 {
		datos.cobertura = &ports.CoberturaOperativaRRHH{
			ViaClave: "bolsa", ProcedimientoRef: "procedimiento:rrhh:minimizado",
			BolsaRef: "bolsa:rrhh:minimizada",
			Comprobaciones: []ports.ComprobacionOperativaRRHH{{
				Clave:     "disponibilidad",
				Resultado: domain.ComprobacionAfirmativa,
			}},
		}
	}
	if bloques >= 3 {
		datos.resumen.UnidadRef = "unidad:rrhh:minimizada"
		datos.asignacion = &ports.AsignacionOperativaRRHH{
			UnidadRef:  datos.resumen.UnidadRef,
			AsignadaEn: hitos[len(hitos)-1].RealizadaEn,
		}
	}
	return datos
}

func reciboDetalleMinimizadoPrueba(
	t *testing.T,
	expedienteRef string,
	version uint64,
	registradaEn time.Time,
) ports.ReciboLecturaRRHH {
	t.Helper()
	autoridad, contexto := autoridadYContextoPuertosRRHH(t, registradaEn)
	solicitud, err := ports.NuevaSolicitudDetalleRRHH(expedienteRef, version)
	if err != nil {
		t.Fatal(err)
	}
	capacidad := capacidadDetallePuertosRRHH(
		t, autoridad, contexto, solicitud, registradaEn,
	)
	recibo, err := ports.NuevoReciboLecturaRRHH(
		"lectura:rrhh:minimizada", "auditoria:rrhh:minimizada",
		contexto, capacidad, expedienteRef, version, 1, registradaEn,
	)
	if err != nil {
		t.Fatal(err)
	}
	return recibo
}

func referenciaAnalisisMinimizadaPrueba(
	t *testing.T,
	secuencia uint64,
) ports.ReferenciaHitoAnalisisRRHH {
	t.Helper()
	referencia, err := ports.NuevaReferenciaHitoAnalisisRRHH(secuencia)
	if err != nil {
		t.Fatal(err)
	}
	return referencia
}

func referenciaCoberturaMinimizadaPrueba(
	t *testing.T,
	secuencia uint64,
) ports.ReferenciaHitoCoberturaRRHH {
	t.Helper()
	referencia, err := ports.NuevaReferenciaHitoCoberturaRRHH(secuencia)
	if err != nil {
		t.Fatal(err)
	}
	return referencia
}

func referenciaAsignacionMinimizadaPrueba(
	t *testing.T,
	secuencia uint64,
) ports.ReferenciaHitoAsignacionRRHH {
	t.Helper()
	referencia, err := ports.NuevaReferenciaHitoAsignacionRRHH(secuencia)
	if err != nil {
		t.Fatal(err)
	}
	return referencia
}
