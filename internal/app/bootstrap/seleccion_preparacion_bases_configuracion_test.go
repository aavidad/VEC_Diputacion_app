package bootstrap

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vec-diputacion-granada/config"
)

func configuracionPreparacionBasesPrueba() configuracionPreparacionBasesV3 {
	m := motivoCatalogoPlantillasCTDesarrollo()
	return configuracionPreparacionBasesV3{Esquema: "vec.seleccion.preparacion-bases.servidor.v1",
		Ambito:        ambitoConfiguracionPreparacionBasesV3{OrganizacionRef: "org_" + strings.Repeat("a", 16), UnidadGestionRef: "uni_" + strings.Repeat("b", 16)},
		MotivoGuardar: m, MotivoConsultar: m, MotivoIntentoDenegado: m, MotivoIntentoError: m, EscrituraFile: "escritura.dsn", LecturaFile: "lectura.dsn"}
}

func TestPreparacionBasesConfiguracionOpcionalYPrivada(t *testing.T) {
	dir := directorioTemporalFueraDeGitPrueba(t)
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{ExecutionProfile: config.ExecutionProfileDevelopment, AuthMode: config.AuthModeDevelopment,
		DevelopmentGuard: config.DevelopmentGuardAcknowledgement, DevelopmentMaterialDir: dir}
	if _, existe, err := leerConfiguracionPreparacionBasesV3(cfg); err != nil || existe {
		t.Fatalf("ausencia opcional: existe=%v err=%v", existe, err)
	}
	c := configuracionPreparacionBasesPrueba()
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(dir, nombreConfiguracionPreparacionBasesV3)
	if err := os.WriteFile(ruta, b, 0600); err != nil {
		t.Fatal(err)
	}
	obtenida, existe, err := leerConfiguracionPreparacionBasesV3(cfg)
	if err != nil || !existe || !obtenida.valida() || obtenida.Ambito != c.Ambito {
		t.Fatalf("archivo privado válido rechazado: existe=%v err=%v", existe, err)
	}
	for nombre, cuerpo := range map[string][]byte{
		"campo_desconocido": append(append([]byte(nil), b[:len(b)-1]...), []byte(`,"actor_ref":"cliente"}`)...),
		"clave_duplicada":   append(append([]byte(nil), b[:len(b)-1]...), []byte(`,"esquema":"vec.seleccion.preparacion-bases.servidor.v1"}`)...),
		"documento_extra":   append(append([]byte(nil), b...), []byte(` {}`)...),
	} {
		t.Run(nombre, func(t *testing.T) {
			if err := os.WriteFile(ruta, cuerpo, 0600); err != nil {
				t.Fatal(err)
			}
			if _, existe, err := leerConfiguracionPreparacionBasesV3(cfg); !existe || !errors.Is(err, errMontajePreparacionBasesV3) {
				t.Fatal("archivo presente inválido se trató como ausencia o configuración válida")
			}
		})
	}
	if err := os.WriteFile(ruta, b, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(ruta, 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := leerConfiguracionPreparacionBasesV3(cfg); err == nil {
		t.Fatal("archivo legible por terceros admitido")
	}
	if err := os.Remove(ruta); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "otro.json")
	if err := os.WriteFile(target, b, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, ruta); err != nil {
		t.Fatal(err)
	}
	if _, _, err := leerConfiguracionPreparacionBasesV3(cfg); err == nil {
		t.Fatal("enlace simbólico admitido")
	}
}

func TestPreparacionBasesConfiguracionNoPrestaAmbitosNiProvision(t *testing.T) {
	for nombre, mutar := range map[string]func(*configuracionPreparacionBasesV3){
		"ambito_libre":             func(c *configuracionPreparacionBasesV3) { c.Ambito.OrganizacionRef = "org_cliente" },
		"dsn_absoluto":             func(c *configuracionPreparacionBasesV3) { c.LecturaFile = "/lectura.dsn" },
		"dsn_fuera":                func(c *configuracionPreparacionBasesV3) { c.LecturaFile = "../lectura.dsn" },
		"mismo_dsn":                func(c *configuracionPreparacionBasesV3) { c.LecturaFile = c.EscrituraFile },
		"aprobacion_sin_preimagen": func(c *configuracionPreparacionBasesV3) { c.Guardar.AprobacionRef = "aprobacion:prueba" },
		"preimagen_sin_aprobacion": func(c *configuracionPreparacionBasesV3) { c.Consultar.PreimagenSHA256 = strings.Repeat("a", 64) },
	} {
		t.Run(nombre, func(t *testing.T) {
			c := configuracionPreparacionBasesPrueba()
			mutar(&c)
			if c.valida() {
				t.Fatal("configuración cruzada admitida")
			}
		})
	}
}
