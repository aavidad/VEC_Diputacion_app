package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/config"
	usuariospg "vec-diputacion-granada/internal/modules/usuarios/adapters/postgres"
	contextopg "vec-diputacion-granada/internal/vec/adapters/contextoactor/postgres"
	identidadpg "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
	vecpg "vec-diputacion-granada/internal/vec/adapters/postgres"
	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecapp "vec-diputacion-granada/internal/vec/application"
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

func configuracionesPreferenciasSeparadas(interna, externa configuracionUsuariosPreferenciasDesarrollo) bool {
	if interna.Superficie != core.SuperficieAutenticacionInternaCorporativaV1 || externa.Superficie != core.SuperficieAutenticacionExternaPersonalV1 {
		return false
	}
	certificados := map[string]bool{}
	cuentas := map[string]bool{}
	perfiles := map[string]bool{}
	for _, c := range interna.Cuentas {
		b, err := hex.DecodeString(c.CertificadoSHA256)
		if err != nil || len(b) != sha256.Size || hex.EncodeToString(b) != c.CertificadoSHA256 || c.Sujeto == "" || c.CuentaRef == "" || c.PerfilRef == "" {
			return false
		}
		certificados[c.CertificadoSHA256] = true
		cuentas[c.CuentaRef] = true
		perfiles[c.PerfilRef] = true
	}
	for _, c := range externa.Cuentas {
		b, err := hex.DecodeString(c.CertificadoSHA256)
		if err != nil || len(b) != sha256.Size || hex.EncodeToString(b) != c.CertificadoSHA256 || c.Sujeto == "" || c.CuentaRef == "" || c.PerfilRef == "" ||
			certificados[c.CertificadoSHA256] || cuentas[c.CuentaRef] || perfiles[c.PerfilRef] {
			return false
		}
	}
	return true
}

// Los cuatro roles de Usuarios y las seis identidades de infraestructura por
// superficie se comprueban antes de publicar cualquiera de las cuatro claves
// V3. Esta sonda no conserva pools: el montaje vuelve a abrirlos y los posee.
func preflightSQLPreferenciasUsuariosDesarrollo(cfg config.Config, derivador *derivadorIdentidadOperacionDesarrollo) error {
	if derivador == nil || !derivador.valido() {
		return errComposicionUsuariosPreferencias
	}
	cInterna, err := leerConfiguracionUsuariosPreferenciasDesarrollo(cfg, core.SuperficieAutenticacionInternaCorporativaV1)
	if err != nil {
		return errComposicionUsuariosPreferencias
	}
	cExterna, err := leerConfiguracionUsuariosPreferenciasDesarrollo(cfg, core.SuperficieAutenticacionExternaPersonalV1)
	if err != nil || !configuracionesPreferenciasSeparadas(cInterna, cExterna) {
		return errComposicionUsuariosPreferencias
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var pools []*pgxpool.Pool
	defer func() {
		for _, p := range pools {
			p.Close()
		}
	}()
	logins := map[string]bool{}
	for _, c := range []configuracionUsuariosPreferenciasDesarrollo{cInterna, cExterna} {
		superficie := c.Superficie
		entradas := []struct{ dsn, rol string }{
			{c.DSNRegistroIdentidad, "vec_identidad_sesiones_v1_registrador"}, {c.DSNRevalidacionIdentidad, "vec_identidad_sesiones_v1_revalidador"},
			{c.DSNContexto, "vec_contexto_actor_v1_runtime"}, {c.DSNFuenteAutorizacion, "vec_autorizacion_fuente"},
			{c.DSNRegistroAutorizacion, "vec_autorizacion_registro"}, {c.DSNMotivos, "vec_autorizacion_motivos_evaluador"},
		}
		inicio := len(pools)
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
		// La sonda genérica de membresía no acredita las funciones/ACL de
		// sesión y Contexto. Sus constructores ejecutan esa acreditación SQL
		// antes de que el gobierno publique ninguna de las cuatro audiencias.
		if _, err = identidadpg.NuevoRegistroSesionesPostgreSQL(ctx, pools[inicio], pools[inicio+1],
			&seudonimizadorSesionDesarrollo{derivador: derivador}, espacioIdentidadSesionDesarrollo, dominioIdentidadSesionDesarrollo); err != nil {
			return errComposicionUsuariosPreferencias
		}
		if _, err = identidadpg.NuevoRevalidadorAutenticacionActorPostgreSQL(ctx, pools[inicio+1]); err != nil {
			return errComposicionUsuariosPreferencias
		}
		resolutor, err := contextopg.NuevoResolutorRegistroContextoActorPostgreSQLV2(ctx, pools[inicio+2])
		if err != nil {
			return errComposicionUsuariosPreferencias
		}
		reloj := relojRutasDietas{}
		servicioContexto, err := vecapp.NuevoServicioContextoActorProductivoV2(resolutor, contextopg.NuevoGeneradorOperacionContextoActorV2Criptografico(), reloj)
		if err != nil {
			return errComposicionUsuariosPreferencias
		}
		if _, err = vecapp.NuevaAutoridadContextoActorRegistradoV2(servicioContexto); err != nil {
			return errComposicionUsuariosPreferencias
		}
		fuente, err := vecpg.NuevoAlmacenAutorizacion(pools[inicio+3])
		if err != nil {
			return errComposicionUsuariosPreferencias
		}
		registroAutorizacion, err := vecpg.NuevoAlmacenAutorizacion(pools[inicio+4])
		if err != nil {
			return errComposicionUsuariosPreferencias
		}
		motivos, err := vecpg.NuevoValidadorReferenciaMotivoPostgreSQLV2(pools[inicio+5], c.MotivoConsulta.CatalogoID)
		if err != nil {
			return errComposicionUsuariosPreferencias
		}
		if _, err = vecapp.NuevoServicioAutorizacionSolicitudLigadaV3(fuente, registroAutorizacion, registroAutorizacion, motivos, reloj,
			seguridad.GeneradorReferenciasCriptograficas{}, vecapp.ConfiguracionServicioAutorizacion{VigenciaDecision: 30 * time.Second}); err != nil {
			return errComposicionUsuariosPreferencias
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
