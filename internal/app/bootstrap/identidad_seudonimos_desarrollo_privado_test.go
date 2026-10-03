package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	postgresidentidad "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
)

func TestSeudonimosPrivadosCincoPropositosYDominioSeparado(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "idempotencia"), 0700); err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(dir, "idempotencia", "configuracion.json")
	if err := os.WriteFile(ruta, configuracionIdempotenciaJSONPrueba(2, 1), 0600); err != nil {
		t.Fatal(err)
	}
	m := materialIdempotenciaDeterministaPrueba(2, 1)
	for _, g := range m.generaciones {
		for dominio, secreto := range map[string][]byte{"localizador": g.localizador.material[:], "huella-solicitud": g.huellaSolicitud.material[:]} {
			if err := os.WriteFile(rutaClaveIdempotenciaDesarrollo(dir, g.generacion, dominio), secreto, 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
	cfg := ConfiguracionSeudonimosSesionPrivada{DirectorioMaterial: dir, RutaConfiguracionHMAC: ruta, EspacioIdentidad: "https://admin.example.invalid/identidad", DominioRef: "idh_admin_prueba", EspacioClave: "vec.identidad.admin.prueba", DominioHMAC: "vec.identidad.admin.hmac.v1", IncluirCuentaOrdinaria: true}
	s, cerrar, err := NuevoSeudonimizadorSesionDesdeArchivo(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer cerrar()
	ids := postgresidentidad.IdentificadoresAlta{EspacioIdentidad: cfg.EspacioIdentidad, AsercionID: "mismo", SesionID: "mismo", SujetoID: "mismo", CuentaID: "mismo", CuentaOrdinariaID: "mismo"}
	a, err := s.SeudonimizarAlta(context.Background(), ids)
	if err != nil {
		t.Fatal(err)
	}
	vistos := map[[32]byte]bool{}
	for _, h := range [][32]byte{a.AsercionIDHMAC, a.SesionIDHMAC, a.SujetoIDHMAC, a.CuentaIDHMAC, a.CuentaOrdinariaIDHMAC} {
		if h == [32]byte{} || vistos[h] {
			t.Fatal("propositos no separados")
		}
		vistos[h] = true
	}
	if a.EspacioIdentidad != cfg.EspacioIdentidad || a.DominioRef != cfg.DominioRef || a.ClaveID != cfg.EspacioClave+".g2" {
		t.Fatal("coordenadas sustituidas")
	}
	ids.AsercionID = "nueva"
	b, err := s.SeudonimizarAlta(context.Background(), ids)
	if err != nil || a.CuentaIDHMAC != b.CuentaIDHMAC || a.CuentaOrdinariaIDHMAC != b.CuentaOrdinariaIDHMAC || a.AsercionIDHMAC == b.AsercionIDHMAC {
		t.Fatal("identidad inestable")
	}
	ids.CuentaOrdinariaID = ""
	if _, err := s.SeudonimizarAlta(context.Background(), ids); err == nil {
		t.Fatal("cuenta ordinaria ausente aceptada")
	}
	ids.CuentaOrdinariaID = "mismo"
	cerrar()
	if _, err := s.SeudonimizarAlta(context.Background(), ids); err == nil {
		t.Fatal("material cerrado reutilizado")
	}
}
