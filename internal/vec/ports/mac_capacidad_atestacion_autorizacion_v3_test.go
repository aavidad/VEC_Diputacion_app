package ports

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"testing"
	"time"
)

type calculadorMACCapacidadV3NuloPrueba struct{}

func (*calculadorMACCapacidadV3NuloPrueba) PerfilEmisionMACCapacidadAtestacionAutorizacionV3(
	context.Context,
) (PerfilEmisionMACCapacidadAtestacionAutorizacionV3, error) {
	panic("no debe invocarse")
}

func (*calculadorMACCapacidadV3NuloPrueba) CalcularMACCapacidadAtestacionAutorizacionV3(
	context.Context,
	SolicitudCalculoMACCapacidadAtestacionAutorizacionV3,
) (ResultadoCalculoMACCapacidadAtestacionAutorizacionV3, error) {
	panic("no debe invocarse")
}

func perfilMACCapacidadV3Prueba(
	t *testing.T,
) PerfilEmisionMACCapacidadAtestacionAutorizacionV3 {
	t.Helper()
	perfil, err := NuevoPerfilEmisionMACCapacidadAtestacionAutorizacionV3(
		DatosPerfilEmisionMACCapacidadAtestacionAutorizacionV3{
			ClaveID: "clave:capacidad:cuadro:v1", ClaveVersion: 1,
			RevisionGobierno: 3, HuellaGobiernoSHA256: strings.Repeat("7", 64),
			EmisorID:         "emisor:capacidad:cuadro:v1",
			AudienciaConsumo: "vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1",
			ValidaDesde:      time.Date(2026, 7, 29, 20, 0, 0, 0, time.UTC),
			ValidaHasta:      time.Date(2026, 7, 29, 22, 0, 0, 0, time.UTC),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return perfil
}

func solicitudMACCapacidadV3Prueba(
	t *testing.T,
) SolicitudCalculoMACCapacidadAtestacionAutorizacionV3 {
	t.Helper()
	solicitud, err := NuevaSolicitudCalculoMACCapacidadAtestacionAutorizacionV3(
		perfilMACCapacidadV3Prueba(t),
		[]byte("4:vec.\n1:3\n"),
		time.Date(2026, 7, 29, 21, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatal(err)
	}
	return solicitud
}

func TestContratoMACCapacidadV3LigaPerfilPreimagenYResultado(t *testing.T) {
	perfil := perfilMACCapacidadV3Prueba(t)
	datos, err := perfil.Datos()
	if err != nil || datos.ClaveVersion != 1 ||
		datos.AudienciaConsumo !=
			"vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1" {
		t.Fatalf("perfil inesperado: %+v, %v", datos, err)
	}
	preimagen := []byte("4:vec.\n1:3\n")
	solicitud, err := NuevaSolicitudCalculoMACCapacidadAtestacionAutorizacionV3(
		perfil, preimagen,
		time.Date(2026, 7, 29, 21, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatal(err)
	}
	preimagen[0] ^= 1
	perfilSolicitud, copia, instante, huella, err := solicitud.MaterialParaCalculador()
	if err != nil || perfilSolicitud != perfil ||
		string(copia) != "4:vec.\n1:3\n" || huella == "" ||
		!instante.Equal(time.Date(2026, 7, 29, 21, 0, 0, 0, time.UTC)) {
		t.Fatalf("material no ligado: %q %q %v", copia, huella, err)
	}
	copia[0] ^= 1
	_, segunda, _, _, _ := solicitud.MaterialParaCalculador()
	if string(segunda) != "4:vec.\n1:3\n" {
		t.Fatal("la preimagen interna fue expuesta")
	}

	mac := bytes.Repeat([]byte{0x5a}, TamanoMACCapacidadAtestacionAutorizacionV3)
	resultado, err := NuevoResultadoCalculoMACCapacidadAtestacionAutorizacionV3(
		solicitud, mac,
	)
	if err != nil {
		t.Fatal(err)
	}
	mac[0] ^= 1
	obtenida, err := resultado.MACPara(solicitud)
	if err != nil || obtenida[0] != 0x5a {
		t.Fatalf("MAC no ligada: %x, %v", obtenida, err)
	}
	obtenida[0] ^= 1
	segundaMAC, _ := resultado.MACPara(solicitud)
	if segundaMAC[0] != 0x5a {
		t.Fatal("la MAC interna fue expuesta")
	}

	otraSolicitud, _ := NuevaSolicitudCalculoMACCapacidadAtestacionAutorizacionV3(
		perfil, []byte("4:otra\n"),
		time.Date(2026, 7, 29, 21, 0, 0, 0, time.UTC),
	)
	if !errors.Is(
		resultado.ValidarPara(otraSolicitud),
		ErrCalculoMACCapacidadAtestacionAutorizacionV3NoDisponible,
	) {
		t.Fatal("resultado aceptado para otra preimagen")
	}
}

func TestContratoMACCapacidadV3RechazaCerosLimitesYPerfilesInvalidos(
	t *testing.T,
) {
	var perfilCero PerfilEmisionMACCapacidadAtestacionAutorizacionV3
	var solicitudCero SolicitudCalculoMACCapacidadAtestacionAutorizacionV3
	var resultadoCero ResultadoCalculoMACCapacidadAtestacionAutorizacionV3
	if perfilCero.Validar() == nil || solicitudCero.Validar() == nil ||
		resultadoCero.ValidarPara(solicitudCero) == nil {
		t.Fatal("valor cero aceptado")
	}
	base, _ := perfilMACCapacidadV3Prueba(t).Datos()
	casos := map[string]func(*DatosPerfilEmisionMACCapacidadAtestacionAutorizacionV3){
		"clave": func(d *DatosPerfilEmisionMACCapacidadAtestacionAutorizacionV3) {
			d.ClaveID = ""
		},
		"version": func(d *DatosPerfilEmisionMACCapacidadAtestacionAutorizacionV3) {
			d.ClaveVersion = 0
		},
		"version_fuera_json": func(d *DatosPerfilEmisionMACCapacidadAtestacionAutorizacionV3) {
			d.ClaveVersion = versionMaximaExactaMACCapacidadAtestacionV3 + 1
		},
		"revision": func(d *DatosPerfilEmisionMACCapacidadAtestacionAutorizacionV3) {
			d.RevisionGobierno = 0
		},
		"huella": func(d *DatosPerfilEmisionMACCapacidadAtestacionAutorizacionV3) {
			d.HuellaGobiernoSHA256 = "no"
		},
		"emisor": func(d *DatosPerfilEmisionMACCapacidadAtestacionAutorizacionV3) {
			d.EmisorID = ""
		},
		"audiencia": func(d *DatosPerfilEmisionMACCapacidadAtestacionAutorizacionV3) {
			d.AudienciaConsumo = ""
		},
		"desde": func(d *DatosPerfilEmisionMACCapacidadAtestacionAutorizacionV3) {
			d.ValidaDesde = time.Time{}
		},
		"ventana": func(d *DatosPerfilEmisionMACCapacidadAtestacionAutorizacionV3) {
			d.ValidaHasta = d.ValidaDesde
		},
	}
	for nombre, mutar := range casos {
		t.Run(nombre, func(t *testing.T) {
			datos := base
			mutar(&datos)
			perfil, err := NuevoPerfilEmisionMACCapacidadAtestacionAutorizacionV3(datos)
			if perfil.Validar() == nil || !errors.Is(
				err,
				ErrCalculoMACCapacidadAtestacionAutorizacionV3NoDisponible,
			) {
				t.Fatalf("perfil inválido aceptado: %v", err)
			}
		})
	}

	perfil := perfilMACCapacidadV3Prueba(t)
	instante := time.Date(2026, 7, 29, 21, 0, 0, 0, time.UTC)
	preimagenGrande := bytes.Repeat(
		[]byte{1},
		TamanoMaximoPreimagenMACCapacidadAtestacionV3+1,
	)
	for nombre, preimagen := range map[string][]byte{
		"nula": nil, "vacia": {}, "ceros": make([]byte, 32),
		"grande": preimagenGrande,
	} {
		t.Run("preimagen_"+nombre, func(t *testing.T) {
			solicitud, err := NuevaSolicitudCalculoMACCapacidadAtestacionAutorizacionV3(
				perfil, preimagen, instante,
			)
			if solicitud.Validar() == nil || err == nil {
				t.Fatalf("preimagen inválida aceptada: %v", err)
			}
		})
	}
	for nombre, cuando := range map[string]time.Time{
		"antes":           base.ValidaDesde.Add(-time.Microsecond),
		"limite_final":    base.ValidaHasta,
		"no_utc":          instante.In(time.FixedZone("otra", 3600)),
		"no_microsegundo": instante.Add(time.Nanosecond),
	} {
		t.Run("instante_"+nombre, func(t *testing.T) {
			_, err := NuevaSolicitudCalculoMACCapacidadAtestacionAutorizacionV3(
				perfil, []byte("preimagen"), cuando,
			)
			if err == nil {
				t.Fatal("instante inválido aceptado")
			}
		})
	}

	solicitud := solicitudMACCapacidadV3Prueba(t)
	for nombre, mac := range map[string][]byte{
		"nula": nil, "corta": make([]byte, 31),
		"larga": make([]byte, 33), "ceros": make([]byte, 32),
	} {
		t.Run("mac_"+nombre, func(t *testing.T) {
			resultado, err := NuevoResultadoCalculoMACCapacidadAtestacionAutorizacionV3(
				solicitud, mac,
			)
			if resultado.ValidarPara(solicitud) == nil || err == nil {
				t.Fatalf("MAC inválida aceptada: %v", err)
			}
		})
	}
}

func TestContratoMACCapacidadV3EsOpacoYNoConcedeAutoridad(t *testing.T) {
	perfil := perfilMACCapacidadV3Prueba(t)
	solicitud := solicitudMACCapacidadV3Prueba(t)
	resultado, _ := NuevoResultadoCalculoMACCapacidadAtestacionAutorizacionV3(
		solicitud, bytes.Repeat([]byte{0x6b}, 32),
	)
	for nombre, valor := range map[string]any{
		"perfil": perfil, "solicitud": solicitud, "resultado": resultado,
	} {
		t.Run(nombre, func(t *testing.T) {
			if _, err := json.Marshal(valor); err == nil {
				t.Fatal("JSON genérico permitido")
			}
			if _, err := xml.Marshal(valor); err == nil {
				t.Fatal("XML genérico permitido")
			}
			bloqueadores := []error{}
			if codificador, ok := valor.(interface {
				MarshalText() ([]byte, error)
			}); ok {
				_, err := codificador.MarshalText()
				bloqueadores = append(bloqueadores, err)
			}
			if codificador, ok := valor.(interface {
				MarshalBinary() ([]byte, error)
			}); ok {
				_, err := codificador.MarshalBinary()
				bloqueadores = append(bloqueadores, err)
			}
			if codificador, ok := valor.(interface {
				GobEncode() ([]byte, error)
			}); ok {
				_, err := codificador.GobEncode()
				bloqueadores = append(bloqueadores, err)
			}
			if codificador, ok := valor.(interface {
				MarshalCBOR() ([]byte, error)
			}); ok {
				_, err := codificador.MarshalCBOR()
				bloqueadores = append(bloqueadores, err)
			}
			if codificador, ok := valor.(interface {
				MarshalYAML() (any, error)
			}); ok {
				_, err := codificador.MarshalYAML()
				bloqueadores = append(bloqueadores, err)
			}
			if len(bloqueadores) != 5 {
				t.Fatalf("faltan bloqueadores de codec: %d", len(bloqueadores))
			}
			for _, err := range bloqueadores {
				if !errors.Is(
					err,
					ErrCalculoMACCapacidadAtestacionAutorizacionV3NoDisponible,
				) {
					t.Fatalf("codec no bloqueado: %v", err)
				}
			}
			for _, formato := range []string{"%v", "%+v", "%#v", "%s", "%q", "%x"} {
				texto := fmt.Sprintf(formato, valor)
				if strings.Contains(texto, "consultar_cuadro") ||
					strings.Contains(texto, "4:vec.") ||
					strings.Contains(texto, "6b6b6b") {
					t.Fatalf("formato %s filtró contenido: %s", formato, texto)
				}
			}
			var registro bytes.Buffer
			slog.New(slog.NewJSONHandler(&registro, nil)).Info("prueba", "valor", valor)
			if strings.Contains(registro.String(), "consultar_cuadro") ||
				strings.Contains(registro.String(), "4:vec.") {
				t.Fatalf("slog filtró contenido: %s", registro.String())
			}
		})
	}

	tipoCalculador := reflect.TypeOf((*CalculadorMACCapacidadAtestacionAutorizacionV3)(nil)).Elem()
	if tipoCalculador.NumMethod() != 2 {
		t.Fatalf("superficie del puerto inesperada: %s", tipoCalculador)
	}
	for indice := 0; indice < tipoCalculador.NumMethod(); indice++ {
		metodo := tipoCalculador.Method(indice)
		for parametro := 0; parametro < metodo.Type.NumIn(); parametro++ {
			tipo := metodo.Type.In(parametro)
			if tipo.Kind() == reflect.String || tipo.Kind() == reflect.Uint64 ||
				tipo.Kind() == reflect.Slice {
				t.Fatalf("selector libre en %s: %s", metodo.Name, tipo)
			}
		}
	}
	for _, valor := range []any{perfil, solicitud, resultado} {
		if reflect.TypeOf(valor).Implements(tipoCalculador) {
			t.Fatalf("%T obtuvo autoridad de cálculo", valor)
		}
		if _, autoridad := valor.(ExportadorCapacidadAtestacionAutorizacionV3); autoridad {
			t.Fatalf("%T obtuvo autoridad exportable", valor)
		}
	}

	var nulo *calculadorMACCapacidadV3NuloPrueba
	if !CalculadorMACCapacidadAtestacionAutorizacionV3Nulo(nil) ||
		!CalculadorMACCapacidadAtestacionAutorizacionV3Nulo(nulo) {
		t.Fatal("calculador nulo o nulo tipado aceptado")
	}
	if CalculadorMACCapacidadAtestacionAutorizacionV3Nulo(
		&calculadorMACCapacidadV3NuloPrueba{},
	) {
		t.Fatal("calculador no nulo rechazado")
	}
}
