package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	dietaspg "vec-diputacion-granada/internal/modules/dietas/adapters/postgres"
	personalpg "vec-diputacion-granada/internal/modules/personal/adapters/postgres"
)

// ErrComprobacionArranqueDietas envuelve cualquier rechazo de la comprobación
// previa. El texto añadido nombra solo la etapa; nunca DSN, usuarios ni datos.
var ErrComprobacionArranqueDietas = errors.New("bootstrap: comprobacion previa de Dietas fallida")

// dsnsMaterialDietasComprobacion recoge solo las conexiones de los manifiestos
// privados. La validación completa (cuentas, motivos, material) sigue siendo
// del arranque; aquí solo se acreditan las identidades PostgreSQL.
type dsnsMaterialDietasComprobacion struct {
	DSNRegistroIdentidad     string `json:"dsn_registro_identidad"`
	DSNRevalidacionIdentidad string `json:"dsn_revalidacion_identidad"`
	DSNContexto              string `json:"dsn_contexto"`
	DSNFuenteAutorizacion    string `json:"dsn_fuente_autorizacion"`
	DSNRegistroAutorizacion  string `json:"dsn_registro_autorizacion"`
	DSNMotivos               string `json:"dsn_motivos"`
	DSNAuditoriaFrontera     string `json:"dsn_auditoria_frontera"`
	DSNConsumo               string `json:"dsn_consumo"`
}

// ComprobarArranqueDietasSoloLectura ejecuta, sin arrancar el servidor ni
// escribir en la base, las mismas acreditaciones PostgreSQL que la composición
// de Dietas exige al arrancar con VEC_DIETAS_BORRADORES_ENABLED=true:
//
//   - selector, doble llave y separación de las cuatro URL de entorno;
//   - configuración OSRM (misma composición, sin llamar a OSRM);
//   - pools propios (borradores y relaciones de Personal) con su identidad;
//   - postimagen exacta de Personal (acreditarPostimagenPersonalDietas);
//   - pools de asignación y auditoría de Personal, sus funciones y preflight;
//   - las siete identidades de identidad/dietas-comisiones.json y, si existe,
//     las siete de identidad/dietas-rutas.json.
//
// Cada sonda es un SELECT sobre catálogo o sobre la propia sesión; el pool de
// relaciones abre además sus transacciones en solo lectura. Sirve para
// comprobar una activación antes de reiniciar vec-server.
func ComprobarArranqueDietasSoloLectura(ctx context.Context, cfg config.Config) error {
	if ctx == nil {
		return ErrComprobacionArranqueDietas
	}
	activo, err := cfg.DietasBorradoresDesarrolloActivos()
	if err != nil {
		return fmt.Errorf("%w: selector o entorno de Dietas: %w", ErrComprobacionArranqueDietas, err)
	}
	if !activo {
		return fmt.Errorf("%w: VEC_DIETAS_BORRADORES_ENABLED no es true", ErrComprobacionArranqueDietas)
	}
	// Misma composición del conector cartográfico que el arranque; no llama
	// a OSRM, solo valida URL, ámbito, límites, CIDR y versión del grafo.
	if calculo, err := nuevoCasoUsoCalculoRutas(cfg.Normalize()); err != nil || calculo == nil {
		return fmt.Errorf("%w: cartografia VEC_OSRM_* incompleta o invalida", ErrComprobacionArranqueDietas)
	}
	comisiones, err := leerDSNsMaterialDietasComprobacion(filepath.Join(cfg.DevelopmentMaterialDir, "identidad", "dietas-comisiones.json"), 256<<10, false)
	if err != nil {
		return fmt.Errorf("%w: identidad/dietas-comisiones.json: %w", ErrComprobacionArranqueDietas, err)
	}
	rutas, err := leerDSNsMaterialDietasComprobacion(filepath.Join(cfg.DevelopmentMaterialDir, "identidad", "dietas-rutas.json"), 128<<10, true)
	if err != nil {
		return fmt.Errorf("%w: identidad/dietas-rutas.json: %w", ErrComprobacionArranqueDietas, err)
	}
	sonda, cancelar := context.WithTimeout(ctx, 60*time.Second)
	defer cancelar()
	var abiertos []*pgxpool.Pool
	defer func() {
		for _, p := range abiertos {
			p.Close()
		}
	}()
	etapa := func(nombre string, causa error) error {
		if causa == nil {
			return fmt.Errorf("%w: %s", ErrComprobacionArranqueDietas, nombre)
		}
		return fmt.Errorf("%w: %s: %w", ErrComprobacionArranqueDietas, nombre, causa)
	}

	propios, err := nuevosPoolsPostgreSQLDietasDesarrollo(sonda, cfg)
	if err != nil {
		return etapa("pools VEC_DIETAS_BORRADORES/PERSONAL_RELACIONES", err)
	}
	defer propios.Cerrar()
	if err := acreditarPostimagenPersonalDietas(sonda, propios.Personal()); err != nil {
		return etapa("postimagen Personal", err)
	}
	usuarios := map[string]bool{}
	var topologiaDietas topologiaPostgreSQLDietasDesarrollo
	for i, pool := range []*pgxpool.Pool{propios.Dietas(), propios.Personal()} {
		usuario, topologia, err := acreditarPoolPostgreSQLDietasDesarrollo(sonda, pool, perfilesPoolPostgreSQLDietasDesarrollo[i].rol)
		if err != nil || usuarios[usuario] {
			return etapa("identidad de los pools propios", err)
		}
		usuarios[usuario] = true
		if i == 0 {
			topologiaDietas = topologia
		}
	}
	personal := cfg.DietasBorradoresPostgreSQL
	dsnAsignacion, err := personal.DSNAsignacionPersonal()
	if err != nil {
		return etapa("VEC_DIETAS_PERSONAL_ASIGNACION_DATABASE_URL", err)
	}
	dsnAuditoria, err := personal.DSNAuditoriaPersonal()
	if err != nil {
		return etapa("VEC_DIETAS_PERSONAL_AUDITORIA_FRONTERA_DATABASE_URL", err)
	}
	for i, entrada := range []struct{ nombre, dsn string }{
		{"VEC_DIETAS_PERSONAL_ASIGNACION_DATABASE_URL", dsnAsignacion},
		{"VEC_DIETAS_PERSONAL_AUDITORIA_FRONTERA_DATABASE_URL", dsnAuditoria},
	} {
		pool, usuario, topologia, err := abrirPoolPersonalAsignacionDietas(sonda, entrada.dsn, perfilesPersonalAsignacionDietas[i])
		if err != nil {
			return etapa(entrada.nombre, err)
		}
		abiertos = append(abiertos, pool)
		if usuarios[usuario] || !topologiaDietas.igual(topologia) {
			return etapa(entrada.nombre+": login repetido o base distinta", nil)
		}
		usuarios[usuario] = true
		if i == 0 {
			if err := acreditarFuncionesAsignacionPersonalDietas(sonda, pool); err != nil {
				return etapa(entrada.nombre+": funciones de asignacion", err)
			}
			continue
		}
		registrador, err := personalpg.NuevoRegistradorAuditoriaFronteraAsignacionPostgreSQL(pool)
		if err != nil || registrador.Preflight(sonda) != nil {
			return etapa(entrada.nombre+": preflight de auditoria", err)
		}
	}

	comprobarManifiesto := func(fichero string, m *dsnsMaterialDietasComprobacion, consumo bool) error {
		entradas := []struct{ clave, dsn, rol string }{
			{"dsn_registro_identidad", m.DSNRegistroIdentidad, "vec_identidad_sesiones_v1_registrador"},
			{"dsn_revalidacion_identidad", m.DSNRevalidacionIdentidad, "vec_identidad_sesiones_v1_revalidador"},
			{"dsn_contexto", m.DSNContexto, "vec_contexto_actor_v1_runtime"},
			{"dsn_fuente_autorizacion", m.DSNFuenteAutorizacion, "vec_autorizacion_fuente"},
			{"dsn_registro_autorizacion", m.DSNRegistroAutorizacion, "vec_autorizacion_registro"},
			{"dsn_motivos", m.DSNMotivos, "vec_autorizacion_motivos_evaluador"},
		}
		if consumo {
			entradas = append(entradas, struct{ clave, dsn, rol string }{"dsn_consumo", m.DSNConsumo, "vec_dietas_ejecutor"})
		}
		vistos := map[string]bool{}
		for _, entrada := range entradas {
			pool, usuario, err := abrirPoolRutasDietas(sonda, entrada.dsn, entrada.rol)
			if err != nil {
				return etapa(fichero+" "+entrada.clave, err)
			}
			abiertos = append(abiertos, pool)
			if vistos[usuario] {
				return etapa(fichero+" "+entrada.clave+": login repetido", nil)
			}
			vistos[usuario] = true
		}
		if consumo {
			return nil
		}
		// La auditoría de frontera de Dietas no puede compartir login con
		// ningún pool de la composición de comisiones (arranque idéntico).
		pool, usuario, err := abrirPoolAuditoriaFronteraDietasDesarrollo(sonda, m.DSNAuditoriaFrontera)
		if err != nil {
			return etapa(fichero+" dsn_auditoria_frontera", err)
		}
		abiertos = append(abiertos, pool)
		if vistos[usuario] || usuarios[usuario] {
			return etapa(fichero+" dsn_auditoria_frontera: login repetido", nil)
		}
		registrador, err := dietaspg.NuevoRegistradorAuditoriaFronteraComisionPostgreSQL(pool)
		if err != nil || registrador.Preflight(sonda) != nil {
			return etapa(fichero+" dsn_auditoria_frontera: preflight de auditoria", err)
		}
		return nil
	}
	if err := comprobarManifiesto("dietas-comisiones.json", comisiones, false); err != nil {
		return err
	}
	if rutas != nil {
		if err := comprobarManifiesto("dietas-rutas.json", rutas, true); err != nil {
			return err
		}
	}
	return nil
}

// leerDSNsMaterialDietasComprobacion aplica las mismas protecciones de lectura
// que el arranque (fichero regular, sin enlace, 0600, tamaño acotado, claves
// únicas). Con opcional=true, un fichero ausente devuelve nil sin error.
func leerDSNsMaterialDietasComprobacion(ruta string, limite int64, opcional bool) (*dsnsMaterialDietasComprobacion, error) {
	if opcional {
		if _, err := os.Lstat(ruta); errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
	}
	contenido, err := leerFicheroMaterialSeguro(ruta, limite)
	if err != nil {
		return nil, errors.New("ausente, enlace, permisos distintos de 0600 o tamano invalido")
	}
	defer borrarBytes(contenido)
	if validarClavesJSONUnicas(contenido) != nil {
		return nil, errors.New("JSON con claves repetidas o invalido")
	}
	var m dsnsMaterialDietasComprobacion
	if err := json.NewDecoder(bytes.NewReader(contenido)).Decode(&m); err != nil {
		return nil, errors.New("JSON invalido")
	}
	return &m, nil
}
