package almacen_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/almacen"
	"vec-diputacion-granada/internal/vec/adapters/memory"
	"vec-diputacion-granada/internal/vec/ports"
)

func TestRegistroSeleccionaConectorPorConfiguracionSinCambiarElNucleo(t *testing.T) {
	registro := almacen.NuevoRegistroConectoresAlmacen()
	reloj := relojFijoAlmacen{ahora: time.Date(2026, time.July, 14, 22, 30, 0, 0, time.UTC)}
	for _, identificador := range []string{"ceph_local", "s3_corporativo"} {
		identificador := identificador
		if err := registro.Registrar(identificador, func(_ context.Context, configuracion almacen.ConfiguracionConectorAlmacen) (ports.AlmacenObjetos, error) {
			configuracion["alterada_por_fabrica"] = "si"
			return memory.NuevoAlmacenObjetosMemoria(identificador, 8*1024*1024, reloj)
		}); err != nil {
			t.Fatalf("Registrar(%s) error = %v", identificador, err)
		}
	}
	if obtenidos := registro.Listar(); !reflect.DeepEqual(obtenidos, []string{"ceph_local", "s3_corporativo"}) {
		t.Fatalf("Listar() = %v", obtenidos)
	}
	configuracion := almacen.ConfiguracionConectorAlmacen{"endpoint": "https://objetos.interno.invalid"}
	conector, err := registro.Crear(context.Background(), "s3_corporativo", configuracion, requisitosPrueba())
	if err != nil {
		t.Fatalf("Crear() error = %v", err)
	}
	capacidades, err := conector.Capacidades(context.Background())
	if err != nil || capacidades.ConectorID != "s3_corporativo" {
		t.Fatalf("conector seleccionado = %+v, %v", capacidades, err)
	}
	if _, alterada := configuracion["alterada_por_fabrica"]; alterada {
		t.Fatal("la fabrica recibio el mapa de configuracion original")
	}
}

func TestRegistroRechazaDuplicadosAusentesYCapacidadesInsuficientes(t *testing.T) {
	registro := almacen.NuevoRegistroConectoresAlmacen()
	reloj := relojFijoAlmacen{ahora: time.Date(2026, time.July, 14, 22, 30, 0, 0, time.UTC)}
	fabrica := func(context.Context, almacen.ConfiguracionConectorAlmacen) (ports.AlmacenObjetos, error) {
		return memory.NuevoAlmacenObjetosMemoria("memoria_pruebas", 1024, reloj)
	}
	if err := registro.Registrar("memoria_pruebas", fabrica); err != nil {
		t.Fatalf("Registrar() error = %v", err)
	}
	if err := registro.Registrar("memoria_pruebas", fabrica); !errors.Is(err, almacen.ErrConectorAlmacenYaRegistrado) {
		t.Fatalf("duplicado: error = %v", err)
	}
	if _, err := registro.Crear(context.Background(), "no_instalado", nil, requisitosPrueba()); !errors.Is(err, almacen.ErrConectorAlmacenNoRegistrado) {
		t.Fatalf("ausente: error = %v", err)
	}
	if _, err := registro.Crear(context.Background(), "memoria_pruebas", nil, ports.RequisitosAlmacenObjetos{
		CifradoEnReposo:  true,
		CifradoPorObjeto: true,
	}); !errors.Is(err, ports.ErrCapacidadAlmacenNoDisponible) {
		t.Fatalf("capacidades insuficientes: error = %v", err)
	}
}

func TestRegistroFallaCerradoConContextoRegistroOConectorNulos(t *testing.T) {
	var registroNulo *almacen.RegistroConectoresAlmacen
	if err := registroNulo.Registrar("memoria_pruebas", func(context.Context, almacen.ConfiguracionConectorAlmacen) (ports.AlmacenObjetos, error) {
		return nil, nil
	}); !errors.Is(err, almacen.ErrRegistroConectoresInvalido) {
		t.Fatalf("registro nulo aceptado al registrar: %v", err)
	}
	if _, err := registroNulo.Crear(context.Background(), "memoria_pruebas", nil, ports.RequisitosAlmacenObjetos{}); !errors.Is(err, almacen.ErrRegistroConectoresInvalido) {
		t.Fatalf("registro nulo aceptado al crear: %v", err)
	}
	if obtenidos := registroNulo.Listar(); obtenidos != nil {
		t.Fatalf("registro nulo devolvio conectores: %v", obtenidos)
	}

	registro := almacen.NuevoRegistroConectoresAlmacen()
	if err := registro.Registrar("memoria_pruebas", func(context.Context, almacen.ConfiguracionConectorAlmacen) (ports.AlmacenObjetos, error) {
		var conectorNulo *memory.AlmacenObjetosMemoria
		return conectorNulo, nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := registro.Crear(nil, "memoria_pruebas", nil, ports.RequisitosAlmacenObjetos{}); !errors.Is(err, ports.ErrSolicitudAlmacenInvalida) {
		t.Fatalf("contexto nulo aceptado: %v", err)
	}
	if _, err := registro.Crear(context.Background(), "memoria_pruebas", nil, ports.RequisitosAlmacenObjetos{}); !errors.Is(err, almacen.ErrFabricaConectorAlmacenInvalida) {
		t.Fatalf("conector tipado nulo aceptado: %v", err)
	}
}
