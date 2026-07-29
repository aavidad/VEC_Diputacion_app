package ports_test

import (
	"bytes"
	"crypto/sha256"
	"encoding"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type datosReciboCuadroV2Prueba struct {
	contexto  ports.ContextoConsultaRRHH
	capacidad ports.CapacidadConsultaRRHH
	orden     ports.OrdenConsultaCuadroRRHH
	pagina    ports.PaginaCuadroRRHH

	accesoRef, anterior, huella, vinculo, alcance string
	secuencia                                     uint64
	registradaEn                                  time.Time
	auditoriaRef, auditoriaHuella, consumoHuella  string
	contenidoHuella, resultadoHuella              string
	cursorHuella                                  string
	generadaEn                                    time.Time
	expedienteRef                                 string
	version                                       uint64
	total                                         uint16
}

func TestResultadoRegistradorAccesoRRHHV2RespetaGenesisYContratoSQL(
	t *testing.T,
) {
	t.Parallel()
	ahora := instantePuertosRRHH()
	acceso := "acceso:rrhh:" + strings.Repeat("a", 32)
	noCero := strings.Repeat("b", 64)
	cero := strings.Repeat("0", 64)
	resto := []string{
		strings.Repeat("c", 64),
		strings.Repeat("d", 64),
		strings.Repeat("e", 64),
	}
	construirConEsquema := func(
		esquema string,
		referencia string,
		secuencia uint64,
		anterior string,
	) error {
		_, err := ports.NuevoResultadoRegistradorAccesoRRHHV2(
			esquema,
			referencia, secuencia, anterior,
			resto[0], resto[1], resto[2], ahora,
		)
		return err
	}
	construir := func(
		referencia string,
		secuencia uint64,
		anterior string,
	) error {
		return construirConEsquema(
			ports.EsquemaResultadoRegistradorAccesoRRHHV2,
			referencia, secuencia, anterior,
		)
	}
	if err := construir(acceso, 1, cero); err != nil {
		t.Fatalf("primer acceso SQL válido rechazado: %v", err)
	}
	if err := construir(acceso, 2, noCero); err != nil {
		t.Fatalf("acceso posterior SQL válido rechazado: %v", err)
	}
	casosInvalidos := map[string]error{
		"genesis_no_cero":      construir(acceso, 1, noCero),
		"posterior_cero":       construir(acceso, 2, cero),
		"secuencia_fuera_json": construir(acceso, 9_007_199_254_740_992, noCero),
		"acceso_corto":         construir("acceso:rrhh:a", 1, cero),
		"acceso_mayusculas":    construir("acceso:rrhh:"+strings.Repeat("A", 32), 1, cero),
		"acceso_sufijo_no_hex": construir("acceso:rrhh:"+strings.Repeat("z", 32), 1, cero),
		"esquema_vacio": construirConEsquema(
			"", acceso, 1, cero,
		),
		"esquema_v1": construirConEsquema(
			"vec.contratacion-temporal.recibo-acceso-rrhh.o4-05.v1",
			acceso, 1, cero,
		),
		"esquema_mayusculas": construirConEsquema(
			strings.ToUpper(ports.EsquemaResultadoRegistradorAccesoRRHHV2),
			acceso, 1, cero,
		),
		"esquema_otro": construirConEsquema(
			"vec.contratacion-temporal.recibo-acceso-rrhh.o4-06.v2",
			acceso, 1, cero,
		),
	}
	for nombre, err := range casosInvalidos {
		if !errors.Is(err, ports.ErrResultadoConsultaRRHHNoConfiable) {
			t.Errorf("%s aceptado: %v", nombre, err)
		}
	}
}

func TestReciboLecturaRRHHV2ValidaCuadroYRechazaV1EnEjecucion(
	t *testing.T,
) {
	t.Parallel()
	datos := prepararReciboCuadroV2Prueba(t)
	recibo := construirReciboCuadroV2Prueba(t, datos, "")
	datos.pagina.Lectura = recibo
	if err := datos.pagina.ValidarParaEjecucionInterna(datos.orden); err != nil {
		t.Fatalf("recibo V2 válido rechazado: %v", err)
	}

	v1, err := ports.NuevoReciboLecturaRRHH(
		"lectura:rrhh:historica", "auditoria:rrhh:historica",
		datos.contexto, datos.capacidad, "", 0, datos.total,
		datos.registradaEn,
	)
	if err != nil {
		t.Fatalf("recibo histórico V1: %v", err)
	}
	paginaV1 := datos.pagina
	paginaV1.Lectura = v1
	if err := paginaV1.ValidarPara(datos.orden); err != nil {
		t.Fatalf("compatibilidad histórica V1 rota: %v", err)
	}
	if err := paginaV1.ValidarParaEjecucionInterna(datos.orden); !errors.Is(
		err, ports.ErrResultadoConsultaRRHHNoConfiable,
	) {
		t.Fatalf("V1 entró en ejecución interna: %v", err)
	}
}

func TestReciboLecturaRRHHV2LigaCadenaConsumoIdentidadYResultado(
	t *testing.T,
) {
	t.Parallel()
	base := prepararReciboCuadroV2Prueba(t)
	sello := selloCanonReciboV2Prueba(base)
	mutaciones := map[string]func(*datosReciboCuadroV2Prueba){
		"secuencia": func(d *datosReciboCuadroV2Prueba) { d.secuencia++ },
		"anterior": func(d *datosReciboCuadroV2Prueba) {
			d.anterior = strings.Repeat("3", 64)
		},
		"huella_cadena": func(d *datosReciboCuadroV2Prueba) {
			d.huella = strings.Repeat("4", 64)
		},
		"vinculo_identidad": func(d *datosReciboCuadroV2Prueba) {
			d.vinculo = strings.Repeat("5", 64)
		},
		"alcance": func(d *datosReciboCuadroV2Prueba) {
			d.alcance = strings.Repeat("6", 64)
		},
		"auditoria": func(d *datosReciboCuadroV2Prueba) {
			d.auditoriaHuella = strings.Repeat("7", 64)
		},
		"consumo": func(d *datosReciboCuadroV2Prueba) {
			d.consumoHuella = strings.Repeat("8", 64)
		},
		"contenido": func(d *datosReciboCuadroV2Prueba) {
			d.contenidoHuella = strings.Repeat("9", 64)
		},
		"resultado": func(d *datosReciboCuadroV2Prueba) {
			d.resultadoHuella = strings.Repeat("a", 64)
		},
		"cursor": func(d *datosReciboCuadroV2Prueba) {
			d.cursorHuella = strings.Repeat("b", 64)
		},
		"generada": func(d *datosReciboCuadroV2Prueba) {
			d.generadaEn = d.generadaEn.Add(time.Microsecond)
		},
		"registrada": func(d *datosReciboCuadroV2Prueba) {
			d.registradaEn = d.registradaEn.Add(time.Microsecond)
		},
		"total": func(d *datosReciboCuadroV2Prueba) { d.total++ },
	}
	for nombre, mutar := range mutaciones {
		datos := base
		mutar(&datos)
		if recibo := construirReciboCuadroV2SinFallar(datos, sello); recibo != nil {
			t.Errorf("%s no quedó ligado por el sello", nombre)
		}
	}

	autoridadDistinta := autoridadContextoPuertosRRHH(
		t, instantePuertosRRHH(), "b", "b",
	)
	contextoDistinto, err := ports.NuevoContextoConsultaRRHH(
		autoridadDistinta, base.contexto.OrganizacionRef(),
		instantePuertosRRHH(),
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitud, err := ports.NuevaSolicitudCuadroRRHH("", "", "", 20, "")
	if err != nil {
		t.Fatal(err)
	}
	capacidadDistinta := capacidadCuadroPuertosRRHH(
		t, contextoDistinto, solicitud,
		instantePuertosRRHH(),
	)
	registro, evidencia := nominalesReciboCuadroV2Prueba(t, base)
	if _, err := ports.NuevoReciboLecturaRRHHV2(
		contextoDistinto, capacidadDistinta, registro, evidencia, sello,
	); !errors.Is(err, ports.ErrResultadoConsultaRRHHNoConfiable) {
		t.Fatalf("contexto/capacidad distintos reutilizaron recibo: %v", err)
	}
}

func TestReciboLecturaRRHHV2ImpideCruceEntrePaginas(
	t *testing.T,
) {
	t.Parallel()
	datos := prepararReciboCuadroV2Prueba(t)
	datos.pagina.Lectura = construirReciboCuadroV2Prueba(t, datos, "")
	otra := datos.pagina
	otra.Expedientes = append(
		[]ports.ResumenExpedienteRRHH(nil), datos.pagina.Expedientes...,
	)
	otra.Expedientes[0].NumeroVisible = "2026/CT-OTRA"
	if err := otra.ValidarPara(datos.orden); err != nil {
		t.Fatalf("el cruce debía detectarlo el canon, no la validación V1: %v", err)
	}
	if err := otra.ValidarParaEjecucionInterna(datos.orden); !errors.Is(
		err, ports.ErrResultadoConsultaRRHHNoConfiable,
	) {
		t.Fatalf("recibo reutilizado en otra página: %v", err)
	}
}

func TestReciboLecturaRRHHV2ValidaDetalleYCruzaSuCanon(
	t *testing.T,
) {
	t.Parallel()
	datosDetalle := datosDetalleMinimizadoPrueba(3)
	entrada := construirEntradaDetalleMinimizadaPrueba(t, datosDetalle)
	instante := instantePuertosRRHH()
	_, contexto := autoridadYContextoPuertosRRHH(t, instante)
	solicitud, err := ports.NuevaSolicitudDetalleRRHH(
		datosDetalle.resumen.ExpedienteRef,
		datosDetalle.resumen.Version,
	)
	if err != nil {
		t.Fatal(err)
	}
	capacidad := capacidadDetallePuertosRRHH(
		t, contexto, solicitud, instante,
	)
	orden, err := ports.NuevaOrdenConsultaDetalleRRHH(
		contexto, capacidad, solicitud, instante,
	)
	if err != nil {
		t.Fatal(err)
	}
	generadaEn := instante.Add(time.Second)
	contenido, err := entrada.ExportarContenidoCanonicoParaSQL(generadaEn)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := contenido.ExportarResultadoCanonicoParaSQL()
	if err != nil {
		t.Fatal(err)
	}
	consumo := strings.Repeat("d", 64)
	sumaAcceso := sha256.Sum256([]byte("acceso:rrhh:" + consumo))
	base := datosReciboCuadroV2Prueba{
		contexto: contexto, capacidad: capacidad,
		accesoRef: "acceso:rrhh:" + hex.EncodeToString(sumaAcceso[:])[:32],
		secuencia: 1, anterior: strings.Repeat("0", 64),
		huella: strings.Repeat("2", 64), vinculo: strings.Repeat("3", 64),
		alcance: "", registradaEn: generadaEn.Add(time.Second),
		auditoriaRef:    "auditoria:vec:detalle:00000001",
		auditoriaHuella: strings.Repeat("a", 64), consumoHuella: consumo,
		contenidoHuella: resultado.ContenidoHuellaSHA256(),
		resultadoHuella: resultado.HuellaSHA256(),
		cursorHuella:    "", generadaEn: generadaEn,
		expedienteRef: datosDetalle.resumen.ExpedienteRef,
		version:       datosDetalle.resumen.Version, total: 1,
	}
	registro, evidencia := nominalesReciboCuadroV2Prueba(t, base)
	recibo, err := ports.NuevoReciboLecturaRRHHV2(
		contexto, capacidad, registro, evidencia, selloCanonReciboV2Prueba(base),
	)
	if err != nil {
		t.Fatalf("recibo V2 de detalle: %v", err)
	}
	detalle, err := ports.NuevoDetalleExpedienteRRHHMinimizado(entrada, recibo)
	if err != nil {
		t.Fatalf("detalle V2: %v", err)
	}
	if err := detalle.ValidarParaEjecucionInterna(orden); err != nil {
		t.Fatalf("detalle V2 válido rechazado: %v", err)
	}

	v1, err := ports.NuevoReciboLecturaRRHH(
		"lectura:rrhh:detalle:historica", "auditoria:rrhh:detalle:historica",
		contexto, capacidad, base.expedienteRef, base.version, 1,
		base.registradaEn,
	)
	if err != nil {
		t.Fatal(err)
	}
	detalleV1, err := ports.NuevoDetalleExpedienteRRHHMinimizado(entrada, v1)
	if err != nil {
		t.Fatal(err)
	}
	if err := detalleV1.ValidarPara(orden); err != nil {
		t.Fatalf("compatibilidad histórica detalle V1 rota: %v", err)
	}
	if err := detalleV1.ValidarParaEjecucionInterna(orden); !errors.Is(
		err, ports.ErrResultadoConsultaRRHHNoConfiable,
	) {
		t.Fatalf("detalle V1 entró en ejecución interna: %v", err)
	}

	mutacionesDetalle := map[string]func(*ports.DetalleExpedienteRRHH){
		"expediente": func(d *ports.DetalleExpedienteRRHH) {
			d.Resumen.ExpedienteRef = "expediente:rrhh:distinto"
		},
		"version": func(d *ports.DetalleExpedienteRRHH) {
			d.Resumen.Version++
		},
		"hito": func(d *ports.DetalleExpedienteRRHH) {
			d.Hitos[0].AccionClave = "actuacion.detalle.mutada"
		},
		"bloque": func(d *ports.DetalleExpedienteRRHH) {
			d.Analisis.CategoriaRef = "categoria:rrhh:mutada"
		},
	}
	for nombre, mutar := range mutacionesDetalle {
		copia := detalle.Clonar()
		mutar(&copia)
		if err := copia.ValidarParaEjecucionInterna(orden); !errors.Is(
			err, ports.ErrResultadoConsultaRRHHNoConfiable,
		) {
			t.Errorf("detalle mutado en %s aceptado: %v", nombre, err)
		}
	}

	sello := selloCanonReciboV2Prueba(base)
	mutacionesEvidencia := map[string]func(*datosReciboCuadroV2Prueba){
		"expediente": func(d *datosReciboCuadroV2Prueba) {
			d.expedienteRef = "expediente:rrhh:distinto"
		},
		"version": func(d *datosReciboCuadroV2Prueba) { d.version++ },
		"generada": func(d *datosReciboCuadroV2Prueba) {
			d.generadaEn = d.generadaEn.Add(time.Microsecond)
		},
		"registrada": func(d *datosReciboCuadroV2Prueba) {
			d.registradaEn = d.registradaEn.Add(time.Microsecond)
		},
		"contenido": func(d *datosReciboCuadroV2Prueba) {
			d.contenidoHuella = strings.Repeat("8", 64)
		},
		"resultado": func(d *datosReciboCuadroV2Prueba) {
			d.resultadoHuella = strings.Repeat("9", 64)
		},
	}
	for nombre, mutar := range mutacionesEvidencia {
		copia := base
		mutar(&copia)
		if recibo := construirReciboCuadroV2SinFallar(copia, sello); recibo != nil {
			t.Errorf("evidencia detalle mutada en %s aceptada", nombre)
		}
	}
	conCursor := base
	conCursor.cursorHuella = strings.Repeat("b", 64)
	if recibo := construirReciboCuadroV2SinFallar(conCursor, sello); recibo != nil {
		t.Error("detalle aceptó cursor")
	}
	registro, evidencia = nominalesReciboCuadroV2Prueba(t, base)
	if _, err := ports.NuevoReciboLecturaRRHHV2(
		contexto, capacidad, registro, evidencia, strings.Repeat("f", 64),
	); !errors.Is(err, ports.ErrResultadoConsultaRRHHNoConfiable) {
		t.Fatalf("sello mutado aceptado: %v", err)
	}
}

func TestDetalleV2RechazaResultadoAnteriorALaOrdenYAceptaElBorde(
	t *testing.T,
) {
	t.Parallel()
	t0 := instantePuertosRRHH()
	datosDetalle := datosDetalleMinimizadoPrueba(0)
	entrada := construirEntradaDetalleMinimizadaPrueba(t, datosDetalle)
	_, contexto := autoridadYContextoPuertosRRHH(t, t0)
	solicitud, err := ports.NuevaSolicitudDetalleRRHH(
		datosDetalle.resumen.ExpedienteRef,
		datosDetalle.resumen.Version,
	)
	if err != nil {
		t.Fatal(err)
	}
	capacidad := capacidadDetallePuertosRRHH(
		t, contexto, solicitud, t0,
	)
	instanteOrden := t0.Add(time.Second)
	orden, err := ports.NuevaOrdenConsultaDetalleRRHH(
		contexto, capacidad, solicitud, instanteOrden,
	)
	if err != nil {
		t.Fatal(err)
	}

	construirDetalle := func(generadaEn time.Time) ports.DetalleExpedienteRRHH {
		t.Helper()
		contenido, err := entrada.ExportarContenidoCanonicoParaSQL(generadaEn)
		if err != nil {
			t.Fatal(err)
		}
		resultado, err := contenido.ExportarResultadoCanonicoParaSQL()
		if err != nil {
			t.Fatal(err)
		}
		consumo := strings.Repeat("e", 64)
		sumaAcceso := sha256.Sum256([]byte("acceso:rrhh:" + consumo))
		datos := datosReciboCuadroV2Prueba{
			contexto: contexto, capacidad: capacidad,
			accesoRef: "acceso:rrhh:" +
				hex.EncodeToString(sumaAcceso[:])[:32],
			secuencia: 1, anterior: strings.Repeat("0", 64),
			huella:  strings.Repeat("2", 64),
			vinculo: strings.Repeat("3", 64), alcance: "",
			registradaEn:    t0.Add(2 * time.Second),
			auditoriaRef:    "auditoria:vec:detalle:temporal:0001",
			auditoriaHuella: strings.Repeat("a", 64),
			consumoHuella:   consumo,
			contenidoHuella: resultado.ContenidoHuellaSHA256(),
			resultadoHuella: resultado.HuellaSHA256(),
			generadaEn:      generadaEn,
			expedienteRef:   datosDetalle.resumen.ExpedienteRef,
			version:         datosDetalle.resumen.Version,
			total:           1,
		}
		registro, evidencia := nominalesReciboCuadroV2Prueba(t, datos)
		recibo, err := ports.NuevoReciboLecturaRRHHV2(
			contexto, capacidad, registro, evidencia,
			selloCanonReciboV2Prueba(datos),
		)
		if err != nil {
			t.Fatal(err)
		}
		detalle, err := ports.NuevoDetalleExpedienteRRHHMinimizado(
			entrada, recibo,
		)
		if err != nil {
			t.Fatal(err)
		}
		return detalle
	}

	anterior := construirDetalle(t0)
	if err := anterior.ValidarPara(orden); err != nil {
		t.Fatalf("precondición histórica no reproducida: %v", err)
	}
	if err := anterior.ValidarParaEjecucionInterna(orden); !errors.Is(
		err, ports.ErrResultadoConsultaRRHHNoConfiable,
	) {
		t.Fatalf("detalle generado antes de la orden aceptado: %v", err)
	}
	enElBorde := construirDetalle(instanteOrden)
	if err := enElBorde.ValidarParaEjecucionInterna(orden); err != nil {
		t.Fatalf("detalle generado exactamente en la orden rechazado: %v", err)
	}
}

func TestEvidenciasReciboRRHHV2SonOpacas(t *testing.T) {
	t.Parallel()
	datos := prepararReciboCuadroV2Prueba(t)
	registro, evidencia := nominalesReciboCuadroV2Prueba(t, datos)
	recibo := construirReciboCuadroV2Prueba(t, datos, "")
	sensibles := []string{
		datos.accesoRef, datos.consumoHuella, datos.vinculo,
		datos.auditoriaRef, datos.cursorHuella,
	}
	for nombre, valor := range map[string]any{
		"registrador": registro,
		"evidencia":   evidencia,
		"recibo":      recibo,
	} {
		if _, err := json.Marshal(valor); !errors.Is(
			err, ports.ErrMaterialConsultaRRHHSensible,
		) {
			t.Errorf("%s serializable como JSON: %v", nombre, err)
		}
		if _, err := valor.(encoding.TextMarshaler).MarshalText(); !errors.Is(
			err, ports.ErrMaterialConsultaRRHHSensible,
		) {
			t.Errorf("%s serializable como texto: %v", nombre, err)
		}
		if _, err := valor.(encoding.BinaryMarshaler).MarshalBinary(); !errors.Is(
			err, ports.ErrMaterialConsultaRRHHSensible,
		) {
			t.Errorf("%s serializable como binario: %v", nombre, err)
		}
		var binario bytes.Buffer
		if err := gob.NewEncoder(&binario).Encode(valor); !errors.Is(
			err, ports.ErrMaterialConsultaRRHHSensible,
		) {
			t.Errorf("%s serializable como gob: %v", nombre, err)
		}
		if _, err := xml.Marshal(valor); !errors.Is(
			err, ports.ErrMaterialConsultaRRHHSensible,
		) {
			t.Errorf("%s serializable como XML: %v", nombre, err)
		}
		var bitacora bytes.Buffer
		slog.New(slog.NewJSONHandler(&bitacora, nil)).Info(
			"prueba", "valor", valor,
		)
		representaciones := []string{
			fmt.Sprintf("%v", valor),
			fmt.Sprintf("%#v", valor),
			bitacora.String(),
		}
		for _, representacion := range representaciones {
			for _, sensible := range sensibles {
				if strings.Contains(representacion, sensible) {
					t.Errorf("%s filtró material en %q", nombre, representacion)
				}
			}
		}
		tipo := reflect.TypeOf(valor)
		for i := 0; i < tipo.NumField(); i++ {
			if tipo.Field(i).PkgPath == "" {
				t.Errorf("%s expone el campo %s", nombre, tipo.Field(i).Name)
			}
		}
	}
}

func prepararReciboCuadroV2Prueba(
	t *testing.T,
) datosReciboCuadroV2Prueba {
	t.Helper()
	instante := instantePuertosRRHH()
	_, contexto := autoridadYContextoPuertosRRHH(t, instante)
	solicitud, err := ports.NuevaSolicitudCuadroRRHH("", "", "", 20, "")
	if err != nil {
		t.Fatal(err)
	}
	capacidad := capacidadCuadroPuertosRRHH(
		t, contexto, solicitud, instante,
	)
	orden, err := ports.NuevaOrdenConsultaCuadroRRHH(
		contexto, capacidad, solicitud, instante,
	)
	if err != nil {
		t.Fatal(err)
	}
	pagina := paginaContenidoCuadroRRHHPrueba(cursorResultadoRRHHPrueba())
	contenido, err := pagina.ExportarContenidoCanonicoParaSQL()
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := contenido.ExportarResultadoCanonicoParaSQL()
	if err != nil {
		t.Fatal(err)
	}
	consumo := strings.Repeat("c", 64)
	sumaAcceso := sha256.Sum256([]byte("acceso:rrhh:" + consumo))
	return datosReciboCuadroV2Prueba{
		contexto: contexto, capacidad: capacidad, orden: orden, pagina: pagina,
		accesoRef: "acceso:rrhh:" + hex.EncodeToString(sumaAcceso[:])[:32],
		secuencia: 2, anterior: strings.Repeat("1", 64),
		huella: strings.Repeat("2", 64), vinculo: strings.Repeat("3", 64),
		alcance:         strings.Repeat("4", 64),
		registradaEn:    pagina.GeneradaEn.Add(time.Second),
		auditoriaRef:    "auditoria:vec:0000000000000001",
		auditoriaHuella: strings.Repeat("a", 64), consumoHuella: consumo,
		contenidoHuella: resultado.ContenidoHuellaSHA256(),
		resultadoHuella: resultado.HuellaSHA256(),
		cursorHuella:    resultado.CursorHuellaSHA256(),
		generadaEn:      resultado.GeneradaEn(),
		total:           resultado.Total(),
	}
}

func nominalesReciboCuadroV2Prueba(
	t *testing.T,
	d datosReciboCuadroV2Prueba,
) (ports.ResultadoRegistradorAccesoRRHHV2, ports.EvidenciaConsumoResultadoRRHHV2) {
	t.Helper()
	registro, err := ports.NuevoResultadoRegistradorAccesoRRHHV2(
		ports.EsquemaResultadoRegistradorAccesoRRHHV2,
		d.accesoRef, d.secuencia, d.anterior, d.huella,
		d.vinculo, d.alcance, d.registradaEn,
	)
	if err != nil {
		t.Fatalf("resultado registrador: %v", err)
	}
	evidencia, err := ports.NuevaEvidenciaConsumoResultadoRRHHV2(
		d.auditoriaRef, d.auditoriaHuella, d.consumoHuella,
		d.contenidoHuella, d.resultadoHuella, d.cursorHuella,
		d.generadaEn, d.expedienteRef, d.version, d.total,
	)
	if err != nil {
		t.Fatalf("evidencia consumo/resultado: %v", err)
	}
	return registro, evidencia
}

func construirReciboCuadroV2Prueba(
	t *testing.T,
	d datosReciboCuadroV2Prueba,
	sello string,
) ports.ReciboLecturaRRHH {
	t.Helper()
	if sello == "" {
		sello = selloCanonReciboV2Prueba(d)
	}
	registro, evidencia := nominalesReciboCuadroV2Prueba(t, d)
	recibo, err := ports.NuevoReciboLecturaRRHHV2(
		d.contexto, d.capacidad, registro, evidencia, sello,
	)
	if err != nil {
		t.Fatalf("recibo V2: %v", err)
	}
	return recibo
}

func construirReciboCuadroV2SinFallar(
	d datosReciboCuadroV2Prueba,
	sello string,
) *ports.ReciboLecturaRRHH {
	registro, err := ports.NuevoResultadoRegistradorAccesoRRHHV2(
		ports.EsquemaResultadoRegistradorAccesoRRHHV2,
		d.accesoRef, d.secuencia, d.anterior, d.huella,
		d.vinculo, d.alcance, d.registradaEn,
	)
	if err != nil {
		return nil
	}
	evidencia, err := ports.NuevaEvidenciaConsumoResultadoRRHHV2(
		d.auditoriaRef, d.auditoriaHuella, d.consumoHuella,
		d.contenidoHuella, d.resultadoHuella, d.cursorHuella,
		d.generadaEn, d.expedienteRef, d.version, d.total,
	)
	if err != nil {
		return nil
	}
	recibo, err := ports.NuevoReciboLecturaRRHHV2(
		d.contexto, d.capacidad, registro, evidencia, sello,
	)
	if err != nil {
		return nil
	}
	return &recibo
}

func selloCanonReciboV2Prueba(
	d datosReciboCuadroV2Prueba,
) string {
	var canon []byte
	canon = append(canon, []byte("VEC-CT-RECIBO-LECTURA-RRHH-V2\n")...)
	texto := func(valor string) {
		canon = append(canon, strconv.Itoa(len(valor))...)
		canon = append(canon, ':')
		canon = append(canon, valor...)
		canon = append(canon, '\n')
	}
	entero := func(valor uint64) { texto(strconv.FormatUint(valor, 10)) }
	instante := func(valor time.Time) {
		texto(valor.Format("2006-01-02T15:04:05.000000Z"))
	}
	texto(ports.EsquemaResultadoRegistradorAccesoRRHHV2)
	texto(d.accesoRef)
	entero(d.secuencia)
	texto(d.anterior)
	texto(d.huella)
	texto(d.vinculo)
	texto(d.alcance)
	instante(d.registradaEn)
	texto(d.auditoriaRef)
	texto(d.auditoriaHuella)
	texto(d.consumoHuella)
	texto(d.capacidad.DecisionRef())
	texto(d.capacidad.DecisionHuellaSHA256())
	texto(d.capacidad.CapacidadHuellaSHA256())
	texto(d.capacidad.MaterialHuellaSHA256())
	texto(d.capacidad.ConsultaHuellaSHA256())
	texto(d.capacidad.CorrelacionRef())
	texto(d.contexto.AutenticacionRef())
	texto(strings.Repeat("1", 64))
	texto(d.contexto.SesionRef())
	texto(d.contexto.ControlSesionRef())
	entero(d.contexto.ControlSesionRevision())
	texto(d.contexto.ControlSesionHuellaSHA256())
	texto(d.contexto.ActorRef())
	texto(d.contexto.PerfilRef())
	entero(d.contexto.PerfilVersion())
	texto(d.contexto.OrganizacionRef())
	texto(string(d.capacidad.ClaseAmbito()))
	texto(d.capacidad.AmbitoRef())
	texto(d.capacidad.Accion())
	texto(d.capacidad.Finalidad())
	texto(d.expedienteRef)
	entero(d.version)
	entero(uint64(d.total))
	texto(d.contenidoHuella)
	texto(d.resultadoHuella)
	texto(d.cursorHuella)
	instante(d.generadaEn)
	suma := sha256.Sum256(canon)
	clear(canon)
	return hex.EncodeToString(suma[:])
}
