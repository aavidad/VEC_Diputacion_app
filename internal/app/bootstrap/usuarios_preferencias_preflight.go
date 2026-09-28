package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"time"

	"vec-diputacion-granada/config"
	usuariospg "vec-diputacion-granada/internal/modules/usuarios/adapters/postgres"
)

// Antes de publicar las dos claves de gobierno V3, comprobar que las dos
// funciones nominales de Usuarios están presentes con LOGIN separados.
// Los pools efímeros se cierran antes de componer los definitivos.
func preflightSQLPreferenciasUsuariosDesarrollo(cfg config.Config) error {
	b, err := leerFicheroMaterialSeguro(filepath.Join(cfg.DevelopmentMaterialDir, "identidad", "usuarios-preferencias.json"), 128<<10)
	if err != nil || validarClavesJSONUnicas(b) != nil {
		return errComposicionUsuariosPreferencias
	}
	defer borrarBytes(b)
	var c configuracionUsuariosPreferenciasDesarrollo
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	var extra any
	if d.Decode(&c) != nil || !errors.Is(d.Decode(&extra), io.EOF) || c.Version != 1 || c.Autoridad != AutoridadNoAutoritativa || c.DSNUsuarios == "" || c.DSNUsuariosFrontera == "" {
		return errComposicionUsuariosPreferencias
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	ejecutor, loginE, err := abrirPoolUsuariosPreferencias(ctx, c.DSNUsuarios, "vec_usuarios_ejecutor")
	if err != nil {
		return errComposicionUsuariosPreferencias
	}
	defer ejecutor.Close()
	registrador, loginR, err := abrirPoolUsuariosPreferencias(ctx, c.DSNUsuariosFrontera, "vec_usuarios_registrador_frontera")
	if err != nil {
		return errComposicionUsuariosPreferencias
	}
	defer registrador.Close()
	if loginE == loginR {
		return errComposicionUsuariosPreferencias
	}
	if _, err = usuariospg.NuevoRegistroPreferenciasPostgreSQL(ctx, ejecutor); err != nil {
		return errComposicionUsuariosPreferencias
	}
	if _, err = usuariospg.NuevoRegistradorDenegacionPreferenciasPostgreSQL(ctx, registrador); err != nil {
		return errComposicionUsuariosPreferencias
	}
	return nil
}
