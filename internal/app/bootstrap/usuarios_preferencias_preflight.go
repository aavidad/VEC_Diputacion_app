package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/config"
	usuariospg "vec-diputacion-granada/internal/modules/usuarios/adapters/postgres"
	core "vec-diputacion-granada/internal/vec/domain"
)

func leerConfiguracionUsuariosPreferenciasDesarrollo(cfg config.Config, superficie core.SuperficieAutenticacionActorV1) (configuracionUsuariosPreferenciasDesarrollo, error) {
	vacia := configuracionUsuariosPreferenciasDesarrollo{}
	nombre := nombreConfiguracionPreferencias(superficie)
	if nombre == "" {
		return vacia, errComposicionUsuariosPreferencias
	}
	b, err := leerFicheroMaterialSeguro(filepath.Join(cfg.DevelopmentMaterialDir, "identidad", nombre), 128<<10)
	if err != nil || validarClavesJSONUnicas(b) != nil {
		return vacia, errComposicionUsuariosPreferencias
	}
	defer borrarBytes(b)
	var c configuracionUsuariosPreferenciasDesarrollo
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	var extra any
	if d.Decode(&c) != nil || !errors.Is(d.Decode(&extra), io.EOF) || c.Version != 1 || c.Autoridad != AutoridadNoAutoritativa || c.Superficie != superficie ||
		len(c.Cuentas) == 0 || len(c.Cuentas) > 64 || !core.ReferenciaMotivoAutorizacionV2Valida(c.MotivoConsulta) ||
		!core.ReferenciaMotivoAutorizacionV2Valida(c.MotivoActualizacion) || c.MotivoConsulta.CatalogoID != c.MotivoActualizacion.CatalogoID {
		return vacia, errComposicionUsuariosPreferencias
	}
	return c, nil
}

// Los cuatro roles de Usuarios y las seis identidades de infraestructura por
// superficie se comprueban antes de publicar cualquiera de las cuatro claves
// V3. Esta sonda no conserva pools: el montaje vuelve a abrirlos y los posee.
func preflightSQLPreferenciasUsuariosDesarrollo(cfg config.Config) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var pools []*pgxpool.Pool
	defer func() {
		for _, p := range pools {
			p.Close()
		}
	}()
	logins := map[string]bool{}
	for _, superficie := range []core.SuperficieAutenticacionActorV1{core.SuperficieAutenticacionInternaCorporativaV1, core.SuperficieAutenticacionExternaPersonalV1} {
		c, err := leerConfiguracionUsuariosPreferenciasDesarrollo(cfg, superficie)
		if err != nil {
			return errComposicionUsuariosPreferencias
		}
		entradas := []struct{ dsn, rol string }{
			{c.DSNRegistroIdentidad, "vec_identidad_sesiones_v1_registrador"}, {c.DSNRevalidacionIdentidad, "vec_identidad_sesiones_v1_revalidador"},
			{c.DSNContexto, "vec_contexto_actor_v1_runtime"}, {c.DSNFuenteAutorizacion, "vec_autorizacion_fuente"},
			{c.DSNRegistroAutorizacion, "vec_autorizacion_registro"}, {c.DSNMotivos, "vec_autorizacion_motivos_evaluador"},
		}
		for _, entrada := range entradas {
			p, login, e := abrirPoolRutasDietas(ctx, entrada.dsn, entrada.rol)
			if e != nil || login == "" || logins[login] {
				if p != nil {
					p.Close()
				}
				return errComposicionUsuariosPreferencias
			}
			pools = append(pools, p)
			logins[login] = true
		}
		ejecutor, loginE, err := abrirPoolUsuariosPreferencias(ctx, c.DSNUsuarios, rolEjecutorPreferencias(string(superficie)))
		if err != nil || loginE == "" || logins[loginE] {
			if ejecutor != nil {
				ejecutor.Close()
			}
			return errComposicionUsuariosPreferencias
		}
		pools = append(pools, ejecutor)
		logins[loginE] = true
		registrador, loginR, err := abrirPoolUsuariosPreferencias(ctx, c.DSNUsuariosFrontera, rolRegistradorPreferencias(string(superficie)))
		if err != nil || loginR == "" || logins[loginR] {
			if registrador != nil {
				registrador.Close()
			}
			return errComposicionUsuariosPreferencias
		}
		pools = append(pools, registrador)
		logins[loginR] = true
		if _, err = usuariospg.NuevoRegistroPreferenciasPostgreSQL(ctx, ejecutor, superficie); err != nil {
			return errComposicionUsuariosPreferencias
		}
		if _, err = usuariospg.NuevoRegistradorDenegacionPreferenciasPostgreSQL(ctx, registrador, superficie); err != nil {
			return errComposicionUsuariosPreferencias
		}
	}
	return nil
}
