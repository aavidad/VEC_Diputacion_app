package bootstrap

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

type configuracionEnsayoGobiernoUsuarios struct {
	Fase, DirectorioMaterial, RutaConfiguracionHMAC, DSNPropietario, DSNOperador, Salida   string
	Alcance, NombreClon, HostPermitido, BasePermitida, UsuarioPropietario, UsuarioOperador string
	PuertoPermitido                                                                        uint16
}

func validarConfiguracionEnsayoGobiernoUsuarios(f configuracionEnsayoGobiernoUsuarios) error {
	switch f.Fase {
	case "preparar", "aplicar", "replay", "verificar":
	default:
		return ErrGobiernoUsuariosAdmin
	}
	if f.Alcance != "clon_sintetico_desechable" || strings.TrimSpace(f.NombreClon) == "" || !filepath.IsAbs(f.HostPermitido) || f.PuertoPermitido == 0 || f.BasePermitida == "" || f.UsuarioPropietario == "" || f.UsuarioOperador == "" || !filepath.IsAbs(f.Salida) || dentroDeRepositorioGit(f.Salida) {
		return ErrGobiernoUsuariosAdmin
	}
	for _, e := range []struct{ dsn, user string }{{f.DSNPropietario, f.UsuarioPropietario}, {f.DSNOperador, f.UsuarioOperador}} {
		c, err := pgxpool.ParseConfig(e.dsn)
		if err != nil || c.ConnConfig.Host != f.HostPermitido || c.ConnConfig.Port != f.PuertoPermitido || c.ConnConfig.Database != f.BasePermitida || c.ConnConfig.User != e.user || c.ConnConfig.TLSConfig != nil || len(c.ConnConfig.Fallbacks) != 0 {
			return ErrGobiernoUsuariosAdmin
		}
	}
	return nil
}
func escribirEnsayoGobiernoUsuarios(root *os.Root, nombre string, data []byte) error {
	if root == nil || nombre == "" || filepath.Base(nombre) != nombre {
		return ErrGobiernoUsuariosAdmin
	}
	f, err := root.OpenFile(nombre, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return ErrGobiernoUsuariosAdmin
	}
	_, err = f.Write(data)
	cerrar := f.Close()
	if err != nil || cerrar != nil {
		return ErrGobiernoUsuariosAdmin
	}
	return nil
}
func TestGobiernoUsuariosEnsayoRechazaFaseYDestinoAntesConexion(t *testing.T) {
	f := configuracionEnsayoGobiernoUsuarios{Fase: "preparar", Alcance: "clon_sintetico_desechable", NombreClon: "vec-clon-ensayo", HostPermitido: "/tmp/pg-clon", PuertoPermitido: 5432, BasePermitida: "vec_ensayo", UsuarioPropietario: "owner_ensayo", UsuarioOperador: "operador_ensayo", DSNPropietario: "host=/tmp/pg-clon port=5432 user=owner_ensayo dbname=vec_ensayo sslmode=disable", DSNOperador: "host=/tmp/pg-clon port=5432 user=operador_ensayo dbname=vec_ensayo sslmode=disable", Salida: t.TempDir()}
	if validarConfiguracionEnsayoGobiernoUsuarios(f) != nil {
		t.Fatal("destino explicito rechazado")
	}
	for _, caso := range []string{"fase", "host", "base", "port", "clon"} {
		t.Run(caso, func(t *testing.T) {
			x := f
			switch caso {
			case "fase":
				x.Fase = "otro"
			case "host":
				x.DSNPropietario = "host=cidonia port=5432 user=owner_ensayo dbname=vec_ensayo sslmode=disable"
			case "base":
				x.BasePermitida = "principal"
			case "port":
				x.PuertoPermitido = 5433
			case "clon":
				x.Alcance = "principal"
			}
			if validarConfiguracionEnsayoGobiernoUsuarios(x) == nil {
				t.Fatal("destino o fase no cerrado aceptado")
			}
		})
	}
}
func TestGobiernoUsuariosEnsayoSalidaPrivadaNoTruncaNiSigueEnlace(t *testing.T) {
	d := t.TempDir()
	if os.Chmod(d, 0700) != nil {
		t.Fatal("directorio")
	}
	root, err := AbrirRaizPrivadaDenominacionPersona(d)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if escribirEnsayoGobiernoUsuarios(root, "acta.json", []byte("original")) != nil {
		t.Fatal("salida inicial")
	}
	if escribirEnsayoGobiernoUsuarios(root, "acta.json", []byte("otro")) == nil {
		t.Fatal("acta existente truncada")
	}
	b, _ := os.ReadFile(filepath.Join(d, "acta.json"))
	if string(b) != "original" {
		t.Fatal("acta alterada")
	}
	target := filepath.Join(t.TempDir(), "fuera.json")
	if os.WriteFile(target, []byte("fuera"), 0600) != nil {
		t.Fatal("target")
	}
	if os.Symlink(target, filepath.Join(d, "alias.json")) != nil {
		t.Fatal("enlace")
	}
	if escribirEnsayoGobiernoUsuarios(root, "alias.json", []byte("secreto")) == nil {
		t.Fatal("enlace seguido")
	}
	b, _ = os.ReadFile(target)
	if string(b) != "fuera" {
		t.Fatal("target alterado")
	}
	if os.Chmod(d, 0755) != nil {
		t.Fatal("modo")
	}
	if r, err := AbrirRaizPrivadaDenominacionPersona(d); err == nil {
		r.Close()
		t.Fatal("directorio publico aceptado")
	}
	if !errors.Is(escribirEnsayoGobiernoUsuarios(nil, "x", nil), ErrGobiernoUsuariosAdmin) {
		t.Fatal("root nil")
	}
}
