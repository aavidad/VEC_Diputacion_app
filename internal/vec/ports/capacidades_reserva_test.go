package ports_test

import (
	"bytes"
	"encoding"
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/ports"
)

type capacidadReservaOpacaPrueba interface {
	fmt.Stringer
	fmt.GoStringer
	fmt.Formatter
	slog.LogValuer
	json.Marshaler
	encoding.TextMarshaler
	encoding.BinaryMarshaler
	Valido() bool
	HuellaSHA256() (string, error)
	CoincideConHuellaSHA256(string) bool
}

type casoCapacidadReservaOpaca struct {
	nombre              string
	nueva               func() (capacidadReservaOpacaPrueba, error)
	cero                func() capacidadReservaOpacaPrueba
	punteroCero         func() any
	errorReserva        error
	errorSerializacion  error
	textoRedactado      string
	envolverEnResultado func(capacidadReservaOpacaPrueba) any
}

type contenedorTokenCobroAlta struct {
	Token ports.TokenReservaOrdenCobro `json:"token" xml:"token"`
}

type contenedorTokenCobroDevolucion struct {
	Token ports.TokenReservaDevolucionCobro `json:"token" xml:"token"`
}

func TestCapacidadesReservaSonNominalesOpacasYNoSerializables(t *testing.T) {
	casos := []casoCapacidadReservaOpaca{
		{
			nombre: "generacion_documental",
			nueva: func() (capacidadReservaOpacaPrueba, error) {
				return ports.NuevoTokenReservaGeneracionDocumento()
			},
			cero: func() capacidadReservaOpacaPrueba {
				return ports.TokenReservaGeneracionDocumento{}
			},
			punteroCero:        func() any { return &ports.TokenReservaGeneracionDocumento{} },
			errorReserva:       ports.ErrReservaDocumentoNoValida,
			errorSerializacion: ports.ErrSerializacionTokenReservaDocumentoProhibida,
			textoRedactado:     "[TOKEN-RESERVA-GENERACION-DOCUMENTO-REDACTADO]",
			envolverEnResultado: func(capacidad capacidadReservaOpacaPrueba) any {
				return ports.ReservaGeneracionDocumento{
					Token: capacidad.(ports.TokenReservaGeneracionDocumento),
				}
			},
		},
		{
			nombre: "emision_cotejo",
			nueva: func() (capacidadReservaOpacaPrueba, error) {
				return ports.NuevoTokenReservaEmisionCodigoCotejo()
			},
			cero: func() capacidadReservaOpacaPrueba {
				return ports.TokenReservaEmisionCodigoCotejo{}
			},
			punteroCero:        func() any { return &ports.TokenReservaEmisionCodigoCotejo{} },
			errorReserva:       ports.ErrReservaCodigoCotejoNoValida,
			errorSerializacion: ports.ErrSerializacionTokenReservaCotejoProhibida,
			textoRedactado:     "[TOKEN-RESERVA-EMISION-COTEJO-REDACTADO]",
			envolverEnResultado: func(capacidad capacidadReservaOpacaPrueba) any {
				return ports.ReservaEmisionCodigoCotejo{
					Token: capacidad.(ports.TokenReservaEmisionCodigoCotejo),
				}
			},
		},
		{
			nombre: "alta_cobro",
			nueva: func() (capacidadReservaOpacaPrueba, error) {
				return ports.NuevoTokenReservaOrdenCobro()
			},
			cero: func() capacidadReservaOpacaPrueba {
				return ports.TokenReservaOrdenCobro{}
			},
			punteroCero:        func() any { return &ports.TokenReservaOrdenCobro{} },
			errorReserva:       ports.ErrReservaOrdenCobroInvalida,
			errorSerializacion: ports.ErrSerializacionTokenReservaCobroProhibida,
			textoRedactado:     "[TOKEN-RESERVA-ALTA-COBRO-REDACTADO]",
			envolverEnResultado: func(capacidad capacidadReservaOpacaPrueba) any {
				return contenedorTokenCobroAlta{
					Token: capacidad.(ports.TokenReservaOrdenCobro),
				}
			},
		},
		{
			nombre: "devolucion_cobro",
			nueva: func() (capacidadReservaOpacaPrueba, error) {
				return ports.NuevoTokenReservaDevolucionCobro()
			},
			cero: func() capacidadReservaOpacaPrueba {
				return ports.TokenReservaDevolucionCobro{}
			},
			punteroCero:        func() any { return &ports.TokenReservaDevolucionCobro{} },
			errorReserva:       ports.ErrReservaOrdenCobroInvalida,
			errorSerializacion: ports.ErrSerializacionTokenReservaCobroProhibida,
			textoRedactado:     "[TOKEN-RESERVA-DEVOLUCION-COBRO-REDACTADO]",
			envolverEnResultado: func(capacidad capacidadReservaOpacaPrueba) any {
				return contenedorTokenCobroDevolucion{
					Token: capacidad.(ports.TokenReservaDevolucionCobro),
				}
			},
		},
	}

	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			probarCapacidadReservaOpaca(t, caso)
		})
	}
}

func probarCapacidadReservaOpaca(t *testing.T, caso casoCapacidadReservaOpaca) {
	t.Helper()
	capacidad, err := caso.nueva()
	if err != nil || !capacidad.Valido() {
		t.Fatalf("crear capacidad: valida=%t error=%v", capacidad != nil && capacidad.Valido(), err)
	}
	huella, err := capacidad.HuellaSHA256()
	if err != nil || len(huella) != 64 || huella != strings.ToLower(huella) {
		t.Fatalf("huella persistible = %q, %v", huella, err)
	}
	copia := capacidad
	huellaCopia, err := copia.HuellaSHA256()
	if err != nil || huellaCopia != huella {
		t.Fatalf("la copia inmutable perdio su autoridad: huella=%q error=%v", huellaCopia, err)
	}
	if !capacidad.CoincideConHuellaSHA256(huella) ||
		capacidad.CoincideConHuellaSHA256(strings.ToUpper(huella)) ||
		capacidad.CoincideConHuellaSHA256(strings.Repeat("0", 64)) ||
		capacidad.CoincideConHuellaSHA256("malformada") {
		t.Fatal("la comparacion canonica de la huella no fallo en cerrado")
	}

	distinta, err := caso.nueva()
	if err != nil {
		t.Fatalf("crear segunda capacidad: %v", err)
	}
	huellaDistinta, err := distinta.HuellaSHA256()
	if err != nil || huellaDistinta == huella || capacidad.CoincideConHuellaSHA256(huellaDistinta) {
		t.Fatalf("capacidades no independientes: primera=%q segunda=%q error=%v", huella, huellaDistinta, err)
	}

	cero := caso.cero()
	if cero.Valido() || cero.CoincideConHuellaSHA256(huella) {
		t.Fatal("el valor cero adquirio autoridad")
	}
	if _, err := cero.HuellaSHA256(); !errors.Is(err, caso.errorReserva) {
		t.Fatalf("huella del valor cero: error=%v", err)
	}

	for _, formato := range []string{"%v", "%+v", "%#v", "%s", "%q", "%x"} {
		salida := fmt.Sprintf(formato, capacidad)
		if salida != caso.textoRedactado || strings.Contains(salida, huella) {
			t.Fatalf("formato %q filtro capacidad: %q", formato, salida)
		}
	}

	var registro bytes.Buffer
	registrador := slog.New(slog.NewJSONHandler(&registro, nil))
	registrador.Info("prueba", "capacidad", capacidad)
	if !strings.Contains(registro.String(), caso.textoRedactado) || strings.Contains(registro.String(), huella) {
		t.Fatalf("slog no redacto la capacidad: %s", registro.String())
	}

	if _, err := json.Marshal(capacidad); !errors.Is(err, caso.errorSerializacion) {
		t.Fatalf("MarshalJSON(): error=%v", err)
	}
	if _, err := capacidad.MarshalText(); !errors.Is(err, caso.errorSerializacion) {
		t.Fatalf("MarshalText(): error=%v", err)
	}
	if _, err := capacidad.MarshalBinary(); !errors.Is(err, caso.errorSerializacion) {
		t.Fatalf("MarshalBinary(): error=%v", err)
	}
	if _, err := json.Marshal(caso.envolverEnResultado(capacidad)); !errors.Is(err, caso.errorSerializacion) {
		t.Fatalf("serializacion del resultado contenedor: error=%v", err)
	}
	var binario bytes.Buffer
	if err := gob.NewEncoder(&binario).Encode(caso.envolverEnResultado(capacidad)); !errors.Is(err, caso.errorSerializacion) {
		t.Fatalf("serializacion gob del resultado contenedor: error=%v", err)
	}
	if _, err := xml.Marshal(caso.envolverEnResultado(capacidad)); !errors.Is(err, caso.errorSerializacion) {
		t.Fatalf("serializacion XML del resultado contenedor: error=%v", err)
	}

	puntero := caso.punteroCero()
	if err := json.Unmarshal([]byte(`"inyectada"`), puntero); !errors.Is(err, caso.errorSerializacion) {
		t.Fatalf("UnmarshalJSON(): error=%v", err)
	}
	if err := puntero.(encoding.TextUnmarshaler).UnmarshalText([]byte("inyectada")); !errors.Is(err, caso.errorSerializacion) {
		t.Fatalf("UnmarshalText(): error=%v", err)
	}
	if err := puntero.(encoding.BinaryUnmarshaler).UnmarshalBinary([]byte("inyectada")); !errors.Is(err, caso.errorSerializacion) {
		t.Fatalf("UnmarshalBinary(): error=%v", err)
	}
	if err := xml.Unmarshal([]byte(`<token>inyectada</token>`), puntero); !errors.Is(err, caso.errorSerializacion) {
		t.Fatalf("UnmarshalXML(): error=%v", err)
	}

	valor := reflect.ValueOf(capacidad)
	if valor.Kind() != reflect.Struct || valor.NumField() != 1 {
		t.Fatalf("representacion interna inesperada: %s con %d campos", valor.Kind(), valor.NumField())
	}
	campo := valor.Field(0)
	if campo.Kind() != reflect.Func || campo.CanInterface() || campo.CanSet() {
		t.Fatalf(
			"el secreto quedo en una superficie reflectible: kind=%s interface=%t set=%t",
			campo.Kind(), campo.CanInterface(), campo.CanSet(),
		)
	}
	funcionInvocable := true
	func() {
		defer func() {
			if recover() != nil {
				funcionInvocable = false
			}
		}()
		campo.Call([]reflect.Value{reflect.Zero(campo.Type().In(0))})
	}()
	if funcionInvocable {
		t.Fatal("reflect pudo invocar el cierre privado")
	}

	tipo := valor.Type()
	for indice := 0; indice < tipo.NumMethod(); indice++ {
		nombre := strings.ToLower(tipo.Method(indice).Name)
		if strings.Contains(nombre, "revelar") || strings.Contains(nombre, "material") || nombre == "bytes" {
			t.Fatalf("la API publica revela material mediante %s", tipo.Method(indice).Name)
		}
	}
}

func TestCapacidadesReservaNoColisionanEnUnaMuestraConcurrente(t *testing.T) {
	const cantidad = 128
	for nombre, nueva := range map[string]func() (capacidadReservaOpacaPrueba, error){
		"generacion_documental": func() (capacidadReservaOpacaPrueba, error) {
			return ports.NuevoTokenReservaGeneracionDocumento()
		},
		"emision_cotejo": func() (capacidadReservaOpacaPrueba, error) {
			return ports.NuevoTokenReservaEmisionCodigoCotejo()
		},
		"alta_cobro": func() (capacidadReservaOpacaPrueba, error) {
			return ports.NuevoTokenReservaOrdenCobro()
		},
		"devolucion_cobro": func() (capacidadReservaOpacaPrueba, error) {
			return ports.NuevoTokenReservaDevolucionCobro()
		},
	} {
		nombre, nueva := nombre, nueva
		t.Run(nombre, func(t *testing.T) {
			huellas := make(chan string, cantidad)
			errores := make(chan error, cantidad)
			for indice := 0; indice < cantidad; indice++ {
				go func() {
					capacidad, err := nueva()
					if err != nil {
						errores <- err
						return
					}
					huella, err := capacidad.HuellaSHA256()
					if err != nil {
						errores <- err
						return
					}
					huellas <- huella
				}()
			}
			vistas := make(map[string]struct{}, cantidad)
			for indice := 0; indice < cantidad; indice++ {
				select {
				case err := <-errores:
					t.Fatalf("generar capacidad concurrente: %v", err)
				case huella := <-huellas:
					if _, existe := vistas[huella]; existe {
						t.Fatalf("huella duplicada: %s", huella)
					}
					vistas[huella] = struct{}{}
				}
			}
		})
	}
}
