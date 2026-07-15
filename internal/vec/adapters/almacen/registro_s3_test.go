package almacen_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"vec-diputacion-granada/internal/vec/adapters/almacen"
	s3compatible "vec-diputacion-granada/internal/vec/adapters/almacen/s3"
	"vec-diputacion-granada/internal/vec/ports"
)

func TestRegistroS3CompatibleSeSeleccionaPorIdentificadorSinAcoplarElNucleo(t *testing.T) {
	registro := almacen.NuevoRegistroConectoresAlmacen()
	if err := almacen.RegistrarS3Compatible(registro, "ceph_documental"); err != nil {
		t.Fatalf("registrar S3 compatible: %v", err)
	}
	if obtenidos := registro.Listar(); !reflect.DeepEqual(obtenidos, []string{"ceph_documental"}) {
		t.Fatalf("conectores registrados: %v", obtenidos)
	}
	_, err := registro.Crear(context.Background(), "ceph_documental", almacen.ConfiguracionConectorAlmacen{
		"conector_id": "otro_conector",
	}, ports.RequisitosAlmacenObjetos{})
	if !errors.Is(err, s3compatible.ErrConfiguracionInvalida) {
		t.Fatalf("la fabrica acepto una identidad divergente: %v", err)
	}
}

func TestRegistroS3CompatibleRechazaRegistroNulo(t *testing.T) {
	if err := almacen.RegistrarS3Compatible(nil, "ceph_documental"); !errors.Is(err, almacen.ErrFabricaConectorAlmacenInvalida) {
		t.Fatalf("registro nulo: %v", err)
	}
}
