package confianzaatestacion

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	puertoscontratacion "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const audienciaConsumoCapacidadV3Prueba = "vec_contratacion_temporal.confirmar_alta_atestada_v1"

type lectorCapacidadV3Prueba struct {
	contenido []byte
	cancelar  context.CancelFunc
	err       error
}

func (l *lectorCapacidadV3Prueba) Read(destino []byte) (int, error) {
	if l.cancelar != nil {
		l.cancelar()
	}
	if l.err != nil {
		return 0, l.err
	}
	if len(l.contenido) == 0 {
		return 0, io.EOF
	}
	n := copy(destino, l.contenido)
	l.contenido = l.contenido[n:]
	return n, nil
}

func TestCapacidadAtestacionV3CruzaProcesoConMACYVigenciaCincoSegundos(
	t *testing.T,
) {
	escenario, prueba := escenarioYPruebaConfianzaV3(t)
	emitidaEn := escenario.ahora.Add(time.Microsecond)
	claveEmision := claveCapacidadAtestacionV3Prueba(
		t,
		EstadoClaveHMACCapacidadAtestacionV3Emision,
		time.Time{},
		bytes.Repeat([]byte{0x51}, 32),
	)
	emisor, err := nuevoEmisorCapacidadesAtestacionAutorizacionV3(
		claveEmision,
		&relojConfianzaAtestacionV3Prueba{ahora: emitidaEn},
		bytes.NewReader(bytes.Repeat([]byte{0x91}, 32)),
	)
	if err != nil {
		t.Fatal(err)
	}
	capacidad, err := emisor.Emitir(
		context.Background(),
		escenario.solicitud,
		escenario.decision,
		escenario.motivo,
		escenario.resultado,
		escenario.atestacion,
		prueba,
	)
	if err != nil {
		t.Fatal(err)
	}
	exportacion, err := capacidad.ExportacionCanonicaParaConsumidor()
	if err != nil {
		t.Fatal(err)
	}
	documento, err := interpretarExportacionCapacidadV3(exportacion)
	if err != nil {
		t.Fatal(err)
	}
	datosSolicitud, _ := escenario.solicitud.Datos()
	atributosO204 := []string{
		"flujo_ref",
		"flujo_version",
		"flujo_huella_sha256",
		puertoscontratacion.AtributoHuellaPeticionHMACActiva,
	}
	if len(datosSolicitud.Recurso.Ambitos) != 3 ||
		len(datosSolicitud.Recurso.Atributos) != len(atributosO204) {
		t.Fatalf(
			"la solicitud no conserva el perfil cerrado O2-04: %+v",
			datosSolicitud.Recurso,
		)
	}
	for _, atributo := range atributosO204 {
		if _, existe := datosSolicitud.Recurso.Atributos[atributo]; !existe {
			t.Fatalf("falta atributo O2-04 %q", atributo)
		}
	}
	huellaEfecto, err := datosSolicitud.Recurso.
		HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	if documento.Operacion != datosSolicitud.Accion ||
		documento.EfectoRef != datosSolicitud.Recurso.Referencia ||
		documento.HuellaEfectoSHA256 != huellaEfecto ||
		documento.AudienciaDespliegue != audienciaConfianzaAtestacionV3Prueba ||
		documento.AudienciaConsumo != audienciaConsumoCapacidadV3Prueba ||
		documento.ConfiguracionSecuencia !=
			escenario.configuracion.secuencia ||
		documento.RaizVersion != escenario.raiz.version ||
		documento.Suite != SuiteAtestacionAutorizacionV3COSEEdDSA {
		t.Fatalf("ligaduras incompletas: %+v", documento)
	}
	expiraEn, _ := parsearInstanteCapacidadV3(documento.ExpiraEn)
	if expiraEn.Sub(emitidaEn) !=
		VigenciaMaximaCapacidadAtestacionAutorizacionV3 {
		t.Fatalf("vigencia = %s", expiraEn.Sub(emitidaEn))
	}

	claveVerificacion := claveCapacidadAtestacionV3Prueba(
		t,
		EstadoClaveHMACCapacidadAtestacionV3Verificacion,
		time.Time{},
		bytes.Repeat([]byte{0x51}, 32),
	)
	verificador, err := NuevoVerificadorCapacidadesAtestacionAutorizacionV3(
		&relojConfianzaAtestacionV3Prueba{
			ahora: emitidaEn.Add(time.Microsecond),
		},
		claveVerificacion,
	)
	if err != nil ||
		verificador.VerificarExportacionCanonica(
			context.Background(),
			exportacion,
		) != nil {
		t.Fatalf("salto de proceso rechazado: %v", err)
	}
	exportacion[0] ^= 1
	segunda, _ := capacidad.ExportacionCanonicaParaConsumidor()
	if bytes.Equal(exportacion, segunda) {
		t.Fatal("la exportacion expuso el buffer interno")
	}

	verificador.reloj = &relojConfianzaAtestacionV3Prueba{ahora: expiraEn}
	if err := verificador.Verificar(
		context.Background(),
		capacidad,
	); !errors.Is(err, ErrCapacidadAtestacionV3Invalida) {
		t.Fatalf("capacidad expirada aceptada: %v", err)
	}
}

func TestCapacidadAtestacionV3RechazaMatrizDeCrucesYAlteraciones(
	t *testing.T,
) {
	escenario, prueba := escenarioYPruebaConfianzaV3(t)
	emitidaEn := escenario.ahora.Add(time.Microsecond)
	material := bytes.Repeat([]byte{0x52}, 32)
	claveEmision := claveCapacidadAtestacionV3Prueba(
		t, EstadoClaveHMACCapacidadAtestacionV3Emision, time.Time{}, material,
	)
	emisor, err := nuevoEmisorCapacidadesAtestacionAutorizacionV3(
		claveEmision,
		&relojConfianzaAtestacionV3Prueba{ahora: emitidaEn},
		bytes.NewReader(bytes.Repeat([]byte{0x92}, 32)),
	)
	if err != nil {
		t.Fatal(err)
	}
	capacidad, err := emisor.Emitir(
		context.Background(),
		escenario.solicitud,
		escenario.decision,
		escenario.motivo,
		escenario.resultado,
		escenario.atestacion,
		prueba,
	)
	if err != nil {
		t.Fatal(err)
	}
	exportacion, _ := capacidad.ExportacionCanonicaParaConsumidor()
	documento, _ := interpretarExportacionCapacidadV3(exportacion)
	claveVerificacion := claveCapacidadAtestacionV3Prueba(
		t, EstadoClaveHMACCapacidadAtestacionV3Verificacion, time.Time{}, material,
	)
	verificador, _ := NuevoVerificadorCapacidadesAtestacionAutorizacionV3(
		&relojConfianzaAtestacionV3Prueba{ahora: emitidaEn.Add(time.Microsecond)},
		claveVerificacion,
	)
	mutaciones := map[string]func(*capacidadAtestacionAutorizacionV3JSON){
		"decision": func(c *capacidadAtestacionAutorizacionV3JSON) {
			c.DecisionRef = "dec_otra23456789abcdef0123456789abcdef"
		},
		"motivo": func(c *capacidadAtestacionAutorizacionV3JSON) {
			c.HuellaMotivoSHA256 = strings.Repeat("1", 64)
		},
		"payload": func(c *capacidadAtestacionAutorizacionV3JSON) {
			c.HuellaPayloadVECAD3SHA256 = strings.Repeat("2", 64)
		},
		"contexto": func(c *capacidadAtestacionAutorizacionV3JSON) {
			c.ContextoRef = "rca_otra23456789abcdefghijklmn"
		},
		"audiencia": func(c *capacidadAtestacionAutorizacionV3JSON) {
			c.AudienciaConsumo = "vec_consumidor_otro.confirmar"
		},
		"operacion": func(c *capacidadAtestacionAutorizacionV3JSON) {
			c.Operacion = "contratacion_temporal.solicitud.retirar"
		},
		"efecto": func(c *capacidadAtestacionAutorizacionV3JSON) {
			c.EfectoRef = "hmac-sha256:ambito.v1:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
		},
		"huella_efecto": func(c *capacidadAtestacionAutorizacionV3JSON) {
			c.HuellaEfectoSHA256 = strings.Repeat("3", 64)
		},
		"secuencia_configuracion": func(
			c *capacidadAtestacionAutorizacionV3JSON,
		) {
			c.ConfiguracionSecuencia++
		},
		"version_raiz": func(c *capacidadAtestacionAutorizacionV3JSON) {
			c.RaizVersion++
		},
		"mac": func(c *capacidadAtestacionAutorizacionV3JSON) {
			c.MACSHA256 = strings.Repeat("4", 64)
		},
	}
	for nombre, mutar := range mutaciones {
		t.Run(nombre, func(t *testing.T) {
			cruzado := documento
			mutar(&cruzado)
			contenido, err := json.Marshal(cruzado)
			if err != nil {
				t.Fatal(err)
			}
			if err := verificador.VerificarExportacionCanonica(
				context.Background(),
				contenido,
			); !errors.Is(err, ErrCapacidadAtestacionV3Invalida) {
				t.Fatalf("cruce aceptado: %v", err)
			}
		})
	}
	mutacionesConMACValida := map[string]func(
		*capacidadAtestacionAutorizacionV3JSON,
	){
		"expira_despues_configuracion": func(
			c *capacidadAtestacionAutorizacionV3JSON,
		) {
			c.ConfiguracionExpiraEn = emitidaEn.Add(time.Second).
				Format(time.RFC3339Nano)
		},
		"expira_despues_raiz": func(
			c *capacidadAtestacionAutorizacionV3JSON,
		) {
			c.RaizValidaHasta = emitidaEn.Add(time.Second).
				Format(time.RFC3339Nano)
		},
		"vigencia_superior_cinco_segundos": func(
			c *capacidadAtestacionAutorizacionV3JSON,
		) {
			c.ExpiraEn = emitidaEn.
				Add(VigenciaMaximaCapacidadAtestacionAutorizacionV3 + time.Nanosecond).
				Format(time.RFC3339Nano)
		},
		"todavia_no_emitida": func(
			c *capacidadAtestacionAutorizacionV3JSON,
		) {
			nuevaEmision := emitidaEn.Add(time.Second)
			c.EmitidaEn = nuevaEmision.Format(time.RFC3339Nano)
			c.ExpiraEn = nuevaEmision.
				Add(VigenciaMaximaCapacidadAtestacionAutorizacionV3).
				Format(time.RFC3339Nano)
		},
		"secuencia_configuracion_nula": func(
			c *capacidadAtestacionAutorizacionV3JSON,
		) {
			c.ConfiguracionSecuencia = 0
		},
		"version_raiz_nula": func(
			c *capacidadAtestacionAutorizacionV3JSON,
		) {
			c.RaizVersion = 0
		},
	}
	for nombre, mutar := range mutacionesConMACValida {
		t.Run(nombre, func(t *testing.T) {
			cruzado := documento
			mutar(&cruzado)
			cruzado.MACSHA256 = calcularMACCapacidadAtestacionV3(
				cruzado,
				material,
			)
			contenido, err := json.Marshal(cruzado)
			if err != nil {
				t.Fatal(err)
			}
			if err := verificador.VerificarExportacionCanonica(
				context.Background(),
				contenido,
			); !errors.Is(err, ErrCapacidadAtestacionV3Invalida) {
				t.Fatalf("ventana inconsistente con MAC valida aceptada: %v", err)
			}
		})
	}
}

func TestCapacidadAtestacionV3LimitaEnterosAlRangoExactoJSON(t *testing.T) {
	contenido, err := os.ReadFile(
		filepath.Join("testdata", "capacidad_v3_canonica_o2_05.json"),
	)
	if err != nil {
		t.Fatal(err)
	}
	base, err := interpretarExportacionCapacidadV3(bytes.TrimSpace(contenido))
	if err != nil {
		t.Fatal(err)
	}
	campos := map[string]func(*capacidadAtestacionAutorizacionV3JSON, uint64){
		"clave_version": func(c *capacidadAtestacionAutorizacionV3JSON, v uint64) {
			c.ClaveVersion = v
		},
		"revision_gobierno": func(c *capacidadAtestacionAutorizacionV3JSON, v uint64) {
			c.RevisionGobierno = v
		},
		"configuracion_secuencia": func(c *capacidadAtestacionAutorizacionV3JSON, v uint64) {
			c.ConfiguracionSecuencia = v
		},
		"raiz_version": func(c *capacidadAtestacionAutorizacionV3JSON, v uint64) {
			c.RaizVersion = v
		},
	}
	for nombre, asignar := range campos {
		t.Run(nombre, func(t *testing.T) {
			enLimite := base
			asignar(&enLimite, maximoEnteroExactoJSONCapacidadV3)
			if enLimite.validarEstructura() != nil {
				t.Fatal("se rechazó el límite exacto 2^53-1")
			}
			fuera := base
			asignar(&fuera, maximoEnteroExactoJSONCapacidadV3+1)
			if !errors.Is(
				fuera.validarEstructura(),
				ErrCapacidadAtestacionV3Invalida,
			) {
				t.Fatal("se aceptó 2^53")
			}
		})
	}
}

func TestCapacidadAtestacionV3GobiernaRotacionRevocacionYClave(t *testing.T) {
	escenario, prueba := escenarioYPruebaConfianzaV3(t)
	emitidaEn := escenario.ahora.Add(time.Microsecond)
	material := bytes.Repeat([]byte{0x53}, 32)
	claveEmision := claveCapacidadAtestacionV3Prueba(
		t, EstadoClaveHMACCapacidadAtestacionV3Emision, time.Time{}, material,
	)
	emisor, _ := nuevoEmisorCapacidadesAtestacionAutorizacionV3(
		claveEmision,
		&relojConfianzaAtestacionV3Prueba{ahora: emitidaEn},
		bytes.NewReader(bytes.Repeat([]byte{0x93}, 32)),
	)
	capacidad, err := emisor.Emitir(
		context.Background(),
		escenario.solicitud,
		escenario.decision,
		escenario.motivo,
		escenario.resultado,
		escenario.atestacion,
		prueba,
	)
	if err != nil {
		t.Fatal(err)
	}
	instanteConsumo := emitidaEn.Add(time.Microsecond)
	retenida := claveCapacidadAtestacionV3Prueba(
		t, EstadoClaveHMACCapacidadAtestacionV3Verificacion, time.Time{}, material,
	)
	verificador, _ := NuevoVerificadorCapacidadesAtestacionAutorizacionV3(
		&relojConfianzaAtestacionV3Prueba{ahora: instanteConsumo},
		retenida,
	)
	if err := verificador.Verificar(context.Background(), capacidad); err != nil {
		t.Fatalf("clave retenida no verifico emision historica: %v", err)
	}

	otraClave := claveCapacidadAtestacionV3Prueba(
		t,
		EstadoClaveHMACCapacidadAtestacionV3Verificacion,
		time.Time{},
		bytes.Repeat([]byte{0x54}, 32),
	)
	verificadorAjeno, _ := NuevoVerificadorCapacidadesAtestacionAutorizacionV3(
		&relojConfianzaAtestacionV3Prueba{ahora: instanteConsumo},
		otraClave,
	)
	if err := verificadorAjeno.Verificar(
		context.Background(),
		capacidad,
	); !errors.Is(err, ErrCapacidadAtestacionV3Invalida) {
		t.Fatalf("otra clave aceptada: %v", err)
	}

	revocada := claveCapacidadAtestacionV3Prueba(
		t,
		EstadoClaveHMACCapacidadAtestacionV3Revocada,
		escenario.ahora.Add(-time.Minute),
		material,
	)
	verificadorRevocado, _ := NuevoVerificadorCapacidadesAtestacionAutorizacionV3(
		&relojConfianzaAtestacionV3Prueba{ahora: instanteConsumo},
		revocada,
	)
	if err := verificadorRevocado.Verificar(
		context.Background(),
		capacidad,
	); !errors.Is(err, ErrCapacidadAtestacionV3Invalida) {
		t.Fatalf("clave revocada aceptada: %v", err)
	}
}

func TestCapacidadAtestacionV3RespetaCancelacionYFallaCerrado(t *testing.T) {
	escenario, prueba := escenarioYPruebaConfianzaV3(t)
	emitidaEn := escenario.ahora.Add(time.Microsecond)
	clave := claveCapacidadAtestacionV3Prueba(
		t,
		EstadoClaveHMACCapacidadAtestacionV3Emision,
		time.Time{},
		bytes.Repeat([]byte{0x55}, 32),
	)
	reloj := &relojConfianzaAtestacionV3Prueba{ahora: emitidaEn}
	emisor, _ := nuevoEmisorCapacidadesAtestacionAutorizacionV3(
		clave,
		reloj,
		bytes.NewReader(bytes.Repeat([]byte{0x95}, 32)),
	)
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := emisor.Emitir(
		ctx,
		escenario.solicitud,
		escenario.decision,
		escenario.motivo,
		escenario.resultado,
		escenario.atestacion,
		prueba,
	); !errors.Is(err, context.Canceled) || reloj.invocaciones != 0 {
		t.Fatalf("cancelacion previa alcanzo reloj: %v", err)
	}

	ctx, cancelar = context.WithCancel(context.Background())
	emisor.entropia = &lectorCapacidadV3Prueba{
		contenido: bytes.Repeat([]byte{0x96}, 32),
		cancelar:  cancelar,
	}
	if _, err := emisor.Emitir(
		ctx,
		escenario.solicitud,
		escenario.decision,
		escenario.motivo,
		escenario.resultado,
		escenario.atestacion,
		prueba,
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelacion tras entropia ignorada: %v", err)
	}

	emisor.entropia = bytes.NewReader(make([]byte, 32))
	if _, err := emisor.Emitir(
		context.Background(),
		escenario.solicitud,
		escenario.decision,
		escenario.motivo,
		escenario.resultado,
		escenario.atestacion,
		prueba,
	); !errors.Is(err, ErrCapacidadAtestacionV3NoDisponible) {
		t.Fatalf("nonce nulo aceptado: %v", err)
	}

	emisor.entropia = bytes.NewReader(bytes.Repeat([]byte{0x99}, 32))
	capacidad, err := emisor.Emitir(
		context.Background(),
		escenario.solicitud,
		escenario.decision,
		escenario.motivo,
		escenario.resultado,
		escenario.atestacion,
		prueba,
	)
	if err != nil {
		t.Fatal(err)
	}
	claveVerificacion := claveCapacidadAtestacionV3Prueba(
		t,
		EstadoClaveHMACCapacidadAtestacionV3Verificacion,
		time.Time{},
		bytes.Repeat([]byte{0x55}, 32),
	)
	relojVerificacion := &relojConfianzaAtestacionV3Prueba{
		ahora: emitidaEn.Add(time.Microsecond),
	}
	verificador, err := NuevoVerificadorCapacidadesAtestacionAutorizacionV3(
		relojVerificacion,
		claveVerificacion,
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancelar = context.WithCancel(context.Background())
	cancelar()
	if err := verificador.Verificar(
		ctx,
		capacidad,
	); !errors.Is(err, context.Canceled) ||
		relojVerificacion.invocaciones != 0 {
		t.Fatalf("verificacion cancelada alcanzo reloj: %v", err)
	}
	ctx, cancelar = context.WithCancel(context.Background())
	relojVerificacion.cancelar = cancelar
	if err := verificador.Verificar(
		ctx,
		capacidad,
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelacion tras reloj de verificacion ignorada: %v", err)
	}
}

func TestCapacidadAtestacionV3UnicaExportacionYPreimagenCerrada(t *testing.T) {
	if obtenida := preimagenMACCapacidadAtestacionV3([]string{"á", "b"}); !bytes.Equal(obtenida, []byte("2:á\n1:b\n")) {
		t.Fatalf("encuadre incompatible: %q", obtenida)
	}
	var capacidad CapacidadBreveAtestacionAutorizacionV3
	if _, err := json.Marshal(capacidad); !errors.Is(
		err,
		ErrSerializacionCapacidadAtestacionV3Prohibida,
	) {
		t.Fatalf("JSON generico permitido: %v", err)
	}
	const marca = "[CAPACIDAD-ATESTACION-AUTORIZACION-V3-REDACTADA]"
	for _, texto := range []string{
		fmt.Sprint(capacidad),
		fmt.Sprintf("%+v", capacidad),
		fmt.Sprintf("%#v", capacidad),
	} {
		if !strings.Contains(texto, marca) {
			t.Fatalf("formato no redactado: %q", texto)
		}
	}
	var bitacora bytes.Buffer
	slog.New(slog.NewTextHandler(&bitacora, nil)).Info("capacidad", "valor", capacidad)
	if !strings.Contains(bitacora.String(), marca) {
		t.Fatalf("slog no redacto: %s", bitacora.String())
	}
}

func TestCapacidadAtestacionV3RechazaJSONNoCanonicoDuplicadoOAbierto(t *testing.T) {
	escenario, prueba := escenarioYPruebaConfianzaV3(t)
	clave := claveCapacidadAtestacionV3Prueba(
		t,
		EstadoClaveHMACCapacidadAtestacionV3Emision,
		time.Time{},
		bytes.Repeat([]byte{0x56}, 32),
	)
	emisor, _ := nuevoEmisorCapacidadesAtestacionAutorizacionV3(
		clave,
		&relojConfianzaAtestacionV3Prueba{ahora: escenario.ahora.Add(time.Microsecond)},
		bytes.NewReader(bytes.Repeat([]byte{0x97}, 32)),
	)
	capacidad, err := emisor.Emitir(
		context.Background(),
		escenario.solicitud,
		escenario.decision,
		escenario.motivo,
		escenario.resultado,
		escenario.atestacion,
		prueba,
	)
	if err != nil {
		t.Fatal(err)
	}
	contenido, _ := capacidad.ExportacionCanonicaParaConsumidor()
	duplicado := bytes.Replace(
		contenido,
		[]byte(`{"esquema":`),
		[]byte(`{"esquema":"vec.autorizacion.capacidad-registro-consumo-atestado.v3","esquema":`),
		1,
	)
	desconocido := append(append([]byte(nil), contenido[:len(contenido)-1]...), []byte(`,"libre":"no"}`)...)
	reordenado := append([]byte(" \n"), contenido...)
	for nombre, caso := range map[string][]byte{
		"duplicado":   duplicado,
		"desconocido": desconocido,
		"reordenado":  reordenado,
		"trailing":    append(append([]byte(nil), contenido...), []byte(`{}`)...),
	} {
		t.Run(nombre, func(t *testing.T) {
			if _, err := interpretarExportacionCapacidadV3(caso); !errors.Is(
				err,
				ErrCapacidadAtestacionV3Invalida,
			) {
				t.Fatalf("JSON no canonico aceptado: %v", err)
			}
		})
	}
}

func TestComponentesAtestacionV3MantienenSeparacionDeCredenciales(t *testing.T) {
	tipoVerificadorCOSE := reflect.TypeOf((*ServicioConfianzaAtestacionAutorizacionV3)(nil))
	tipoEmisor := reflect.TypeOf((*EmisorCapacidadesAtestacionAutorizacionV3)(nil))
	tipoConsumidor := reflect.TypeOf((*VerificadorCapacidadesAtestacionAutorizacionV3)(nil))
	if _, existe := tipoVerificadorCOSE.MethodByName("Emitir"); existe {
		t.Fatal("el verificador COSE puede emitir capacidades")
	}
	if _, existe := tipoEmisor.MethodByName("VerificarExportacionCanonica"); existe {
		t.Fatal("el emisor posee la autoridad del consumidor")
	}
	if _, existe := tipoConsumidor.MethodByName("Emitir"); existe {
		t.Fatal("el consumidor puede emitir capacidades")
	}
}

func escenarioYPruebaConfianzaV3(
	t *testing.T,
) (
	escenarioConfianzaAtestacionV3Prueba,
	PruebaConfianzaAtestacionAutorizacionV3,
) {
	t.Helper()
	escenario := nuevoEscenarioConfianzaAtestacionV3Prueba(t)
	prueba, err := escenario.servicio.Verificar(
		context.Background(),
		escenario.solicitud,
		escenario.decision,
		escenario.motivo,
		escenario.resultado,
		escenario.atestacion,
	)
	if err != nil {
		t.Fatal(err)
	}
	return escenario, prueba
}

func claveCapacidadAtestacionV3Prueba(
	t *testing.T,
	estado EstadoClaveHMACCapacidadAtestacionV3,
	revocadaEn time.Time,
	material []byte,
) ClaveHMACCapacidadAtestacionV3 {
	t.Helper()
	ahora := time.Date(2026, 7, 23, 9, 0, 0, 0, time.UTC)
	clave, err := NuevaClaveHMACCapacidadAtestacionAutorizacionV3(
		"clave:capacidad:atestacion:v3:1",
		1,
		material,
		"emisor:capacidad:atestacion:v3:1",
		audienciaConsumoCapacidadV3Prueba,
		estado,
		ahora.Add(-time.Hour),
		ahora.Add(time.Hour),
		revocadaEn,
		7,
		strings.Repeat("7", 64),
	)
	if err != nil {
		t.Fatal(err)
	}
	return clave
}
