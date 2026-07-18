package bootstrap

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"vec-diputacion-granada/config"
)

func configuracionPerfilDesarrollo(t *testing.T) config.Config {
	t.Helper()
	return config.Config{
		ExecutionProfile:       config.ExecutionProfileDevelopment,
		AuthMode:               config.AuthModeDevelopment,
		DevelopmentGuard:       config.DevelopmentGuardAcknowledgement,
		DevelopmentMaterialDir: t.TempDir(),
	}
}

func proveedoresPerfilDesarrollo(t *testing.T) []DescriptorProveedorSeguridad {
	t.Helper()
	proveedores := make([]DescriptorProveedorSeguridad, 0, len(tiposProveedorDesarrollo))
	for _, dato := range []struct {
		tipo       TipoProveedorSeguridad
		referencia string
	}{
		{ProveedorIdentidad, "identidad-mtls-local-v1"},
		{ProveedorIdempotencia, referenciaProveedorIdempotenciaDesarrollo},
		{ProveedorKMS, "kms-fichero-local-v2"},
		{ProveedorTSA, "tsa-determinista-local-v1"},
		{ProveedorTLS, "tls-ca-local-v1"},
	} {
		proveedor, err := NuevoDescriptorProveedorDesarrollo(dato.tipo, dato.referencia)
		if err != nil {
			t.Fatalf("descriptor %s: %v", dato.tipo, err)
		}
		proveedores = append(proveedores, proveedor)
	}
	return proveedores
}

func TestPrepararPerfilDesarrolloMarcaActosYEmiteAvisoSinSecretos(t *testing.T) {
	cfg := configuracionPerfilDesarrollo(t)
	proveedores := proveedoresPerfilDesarrollo(t)
	var registro bytes.Buffer

	metadatos, err := PrepararPerfilEjecucion(cfg, proveedores, &registro)
	if err != nil {
		t.Fatalf("PrepararPerfilEjecucion: %v", err)
	}
	datos := metadatos.Datos()
	if datos.Autoridad != AutoridadNoAutoritativa || datos.PerfilEjecucion != config.ExecutionProfileDevelopment ||
		datos.MigrableAProduccion || !datos.DescartableAlCambiarPerfil ||
		len(datos.Proveedores) != len(tiposProveedorDesarrollo) {
		t.Fatalf("marca no autoritativa incompleta: %+v", datos)
	}
	serializados, err := json.Marshal(metadatos)
	if err != nil {
		t.Fatalf("serializar marca: %v", err)
	}
	if !bytes.Contains(serializados, []byte(`"autoridad":"no_autoritativo"`)) ||
		!bytes.Contains(serializados, []byte(`"migrable_a_produccion":false`)) {
		t.Fatalf("marca estructural ausente: %s", serializados)
	}

	aviso := registro.String()
	for _, esperado := range []string{
		"ADVERTENCIA", "credenciales_no_autoritativas", "identidad", "idempotencia_hmac", "kms", "tsa", "tls",
	} {
		if !strings.Contains(aviso, esperado) {
			t.Fatalf("aviso de arranque no contiene %q: %s", esperado, aviso)
		}
	}
	if strings.Contains(aviso, cfg.DevelopmentMaterialDir) || strings.Contains(aviso, "kms-fichero-local-v2") ||
		strings.Contains(aviso, referenciaProveedorIdempotenciaDesarrollo) {
		t.Fatalf("el aviso filtro ruta o referencia interna: %s", aviso)
	}
}

func TestMarcaNoAutoritativaNoPuedeMutarseMedianteDatos(t *testing.T) {
	metadatos, err := PrepararPerfilEjecucion(
		configuracionPerfilDesarrollo(t),
		proveedoresPerfilDesarrollo(t),
		io.Discard,
	)
	if err != nil {
		t.Fatal(err)
	}
	copia := metadatos.Datos()
	copia.Autoridad = "autoritativo"
	copia.Proveedores[0].Referencia = "alterada"
	segunda := metadatos.Datos()
	if segunda.Autoridad != AutoridadNoAutoritativa || segunda.Proveedores[0].Referencia == "alterada" {
		t.Fatalf("la copia modifico la marca privada: %+v", segunda)
	}
}

func TestProduccionRechazaCualquierProveedorDesarrollo(t *testing.T) {
	for _, tipo := range tiposProveedorDesarrollo {
		t.Run(string(tipo), func(t *testing.T) {
			proveedor, err := NuevoDescriptorProveedorDesarrollo(tipo, "proveedor-local-v1")
			if err != nil {
				t.Fatal(err)
			}
			_, err = PrepararPerfilEjecucion(
				config.Config{}, []DescriptorProveedorSeguridad{proveedor}, io.Discard,
			)
			if !errors.Is(err, ErrProveedorDesarrolloEnProduccion) {
				t.Fatalf("%s: error = %v; se esperaba rechazo del proveedor local", tipo, err)
			}
		})
	}
}

func TestPerfilDesarrolloFallaCerradoSiFaltaLlaveProveedorORuido(t *testing.T) {
	cfg := configuracionPerfilDesarrollo(t)
	proveedores := proveedoresPerfilDesarrollo(t)

	sinGuarda := cfg
	sinGuarda.DevelopmentGuard = ""
	if _, err := PrepararPerfilEjecucion(sinGuarda, proveedores, io.Discard); !errors.Is(err, ErrActivacionDesarrolloInvalida) {
		t.Fatalf("sin guarda: %v", err)
	}
	if _, err := PrepararPerfilEjecucion(
		cfg, proveedores[:len(proveedores)-1], io.Discard,
	); !errors.Is(err, ErrComposicionDesarrolloIncompleta) {
		t.Fatalf("sin TLS: %v", err)
	}
	if _, err := PrepararPerfilEjecucion(cfg, proveedores, nil); !errors.Is(err, ErrRegistroArranqueDesarrollo) {
		t.Fatalf("sin registro ruidoso: %v", err)
	}
	if _, err := PrepararPerfilEjecucion(cfg, proveedores, escritorConError{}); !errors.Is(err, ErrRegistroArranqueDesarrollo) {
		t.Fatalf("registro averiado: %v", err)
	}
}

func TestRaizIntegradaNoAceptaModoDesarrolloSinComposicionCompleta(t *testing.T) {
	cfg := configuracionPerfilDesarrollo(t)
	if _, err := NewDemoAPIWithConfig(cfg); !errors.Is(err, ErrComposicionDesarrolloIncompleta) {
		t.Fatalf("raiz integrada = %v; debe exigir proveedores reales inyectados", err)
	}
}

type escritorConError struct{}

func (escritorConError) Write([]byte) (int, error) {
	return 0, errors.New("registro no disponible")
}
