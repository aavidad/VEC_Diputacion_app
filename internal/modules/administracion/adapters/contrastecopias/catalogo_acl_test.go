package contrastecopias

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"testing"

	domain "vec-diputacion-granada/internal/modules/administracion/domain/contrastecopias"
)

// El responsable prepara un contenedor propio, sin red y con datos sintéticos.
// Este ensayo no utiliza conexión de la aplicación ni bases conservadas.
func TestContrasteACLPG18(t *testing.T) {
	name := os.Getenv("VEC_CS06_CONTENEDOR_ACL_ENSAYO")
	if name == "" {
		t.Skip("ensayo ACL PostgreSQL aislado no solicitado")
	}
	if !baseAdmitida.MatchString(name) {
		t.Fatal("referencia de recurso de ensayo no admitida")
	}
	x := dockerEnsayo{name: name}
	guard := exclusionACLEnsayo{x}
	if _, err := guard.ComprobarExclusion(context.Background()); err != nil {
		t.Fatal("recurso sintético propio no comprobado")
	}
	sqlEnsayo(t, x, `CREATE ROLE cs06_acl_sintetico;`)
	l, err := Nuevo(limitesPrueba())
	if err != nil {
		t.Fatal(err)
	}
	capture := func() domain.Snapshot {
		t.Helper()
		s, err := l.CapturarEjecutor(context.Background(), x, "postgres", guard)
		if err != nil || !s.Completo || len(domain.Validar(s)) != 0 {
			t.Fatalf("captura ACL no completa: %v %v", err, s.Motivos)
		}
		return s
	}
	for _, scenario := range []struct{ name, change, restore string }{
		{"permiso_parametro", `GRANT SET ON PARAMETER statement_timeout TO cs06_acl_sintetico WITH GRANT OPTION`, `REVOKE SET ON PARAMETER statement_timeout FROM cs06_acl_sintetico`},
		{"permiso_lenguaje", `REVOKE USAGE ON LANGUAGE plpgsql FROM PUBLIC`, `GRANT USAGE ON LANGUAGE plpgsql TO PUBLIC`},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			before := capture()
			sqlEnsayo(t, x, scenario.change)
			result := domain.Comparar(before, capture())
			if result.Estado != domain.Diferente {
				t.Fatalf("cambio de permiso omitido: %s", result.Estado)
			}
			found := false
			for _, reason := range result.Razones {
				if reason.Clase == "acl" && reason.Codigo == "contenido_diferente" {
					found = true
				}
			}
			if !found {
				t.Fatal("huella ACL no detecta el cambio de permiso")
			}
			sqlEnsayo(t, x, scenario.restore)
			if domain.Comparar(before, capture()).Estado != domain.Igual {
				t.Fatal("retirada del cambio no recupera la evidencia ACL")
			}
		})
	}
	t.Run("propietario_lenguaje_acl_vacia", func(t *testing.T) {
		sqlEnsayo(t, x, `REVOKE ALL ON LANGUAGE plpgsql FROM PUBLIC; REVOKE ALL ON LANGUAGE plpgsql FROM postgres;`)
		tx := &transporteEjecutor{exec: x, base: "postgres", maxBytes: 4096, timeout: 1000}
		var empty bool
		if err := tx.QueryRow(context.Background(), `SELECT cardinality(lanacl)=0 FROM pg_language WHERE lanname='plpgsql'`).Scan(&empty); err != nil || !empty {
			t.Fatal("la regresión necesita una ACL de lenguaje realmente vacía")
		}
		before := capture()
		sqlEnsayo(t, x, `ALTER LANGUAGE plpgsql OWNER TO cs06_acl_sintetico`)
		result := domain.Comparar(before, capture())
		if result.Estado != domain.Diferente {
			t.Fatalf("propietario con ACL vacía omitido: %s", result.Estado)
		}
		found := false
		for _, reason := range result.Razones {
			if reason.Clase == "acl" && reason.Codigo == "contenido_diferente" {
				found = true
			}
		}
		if !found {
			t.Fatal("huella ACL no detecta el cambio de propietario del lenguaje")
		}
		sqlEnsayo(t, x, `ALTER LANGUAGE plpgsql OWNER TO postgres`)
		if domain.Comparar(before, capture()).Estado != domain.Igual {
			t.Fatal("reversión del propietario no recupera la evidencia ACL")
		}
	})
}

type exclusionACLEnsayo struct{ dockerEnsayo }

func (x exclusionACLEnsayo) ComprobarExclusion(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "/usr/bin/docker", "inspect", x.name).Output()
	if err != nil {
		return "", errCaptura
	}
	var info []struct {
		Id, Image  string
		Config     struct{ Labels map[string]string }
		HostConfig struct{ NetworkMode string }
		State      struct{ Running bool }
	}
	if json.Unmarshal(out, &info) != nil || len(info) != 1 {
		return "", errCaptura
	}
	i := info[0]
	if !i.State.Running || i.HostConfig.NetworkMode != "none" || i.Config.Labels["vec.cs06.owner"] != "contraste-acl" {
		return "", errCaptura
	}
	// La ventana es secuencial y pertenece al responsable de este único ensayo.
	b, _ := json.Marshal([]string{i.Id, i.Image, i.HostConfig.NetworkMode})
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}
