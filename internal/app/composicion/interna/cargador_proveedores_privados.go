package interna

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/app/composicion/internagobierno"
	inc "vec-diputacion-granada/internal/app/incorporacionejercicio"
	"vec-diputacion-granada/internal/vec/adapters/seudonimizacionpkcs11"
)

// El directorio es aprovisionado fuera de Git. Ningun error de esta frontera
// incluye rutas, LOGIN o DSN del material privado.
const EnvMaterialInterno = "VEC_INTERNO_MATERIAL_DIR"

var ErrMaterialSeguimientoNoDisponible = errors.New("composicion interna: material de seguimiento no disponible")

const limiteMaterialPoolsSeguimiento = 64 << 10

type documentoPoolsSeguimiento struct {
	Version int `json:"version"`
	Pools   struct {
		AltaPersonal          entradaPoolSeguimiento `json:"alta_personal"`
		RegistroCT            entradaPoolSeguimiento `json:"registro_ct"`
		InicialCT             entradaPoolSeguimiento `json:"inicial_ct"`
		LocalizadorCT         entradaPoolSeguimiento `json:"localizador_ct"`
		LocalizadorPersonal   entradaPoolSeguimiento `json:"localizador_personal"`
		LecturaPersonal       entradaPoolSeguimiento `json:"lectura_personal"`
		HistoriaRegistroCT    entradaPoolSeguimiento `json:"historia_registro_ct"`
		HistoriaAutenticacion entradaPoolSeguimiento `json:"historia_autenticacion"`
		HistoriaContexto      entradaPoolSeguimiento `json:"historia_contexto"`
		HistoriaEvaluacion    entradaPoolSeguimiento `json:"historia_evaluacion"`
		HistoriaConcesion     entradaPoolSeguimiento `json:"historia_concesion"`
	} `json:"pools"`
}

type entradaPoolSeguimiento struct {
	Login string `json:"login"`
	DSN   string `json:"dsn"`
}

// cargarMaterialPoolsSeguimiento lee exactamente el contrato versionado de
// aprovisionamiento. AbrirPoolsSeguimiento verifica después TLS, LOGIN y ACL.
func cargarMaterialPoolsSeguimiento(directorio string) (MaterialPoolsSeguimiento, error) {
	var vacio MaterialPoolsSeguimiento
	if !filepath.IsAbs(directorio) || filepath.Clean(directorio) != directorio || strings.TrimSpace(directorio) != directorio {
		return vacio, ErrMaterialSeguimientoNoDisponible
	}
	infoDir, err := os.Lstat(directorio)
	if err != nil || !infoDir.IsDir() || infoDir.Mode().Perm() != 0700 {
		return vacio, ErrMaterialSeguimientoNoDisponible
	}
	raiz, err := os.OpenRoot(directorio)
	if err != nil {
		return vacio, ErrMaterialSeguimientoNoDisponible
	}
	defer raiz.Close()
	actualDir, err := raiz.Stat(".")
	if err != nil || !os.SameFile(infoDir, actualDir) || actualDir.Mode().Perm() != 0700 {
		return vacio, ErrMaterialSeguimientoNoDisponible
	}
	info, err := raiz.Lstat("pools.json")
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 || info.Size() <= 0 || info.Size() > limiteMaterialPoolsSeguimiento {
		return vacio, ErrMaterialSeguimientoNoDisponible
	}
	archivo, err := raiz.Open("pools.json")
	if err != nil {
		return vacio, ErrMaterialSeguimientoNoDisponible
	}
	defer archivo.Close()
	actual, err := archivo.Stat()
	if err != nil || !os.SameFile(info, actual) || !actual.Mode().IsRegular() || actual.Mode().Perm() != 0600 {
		return vacio, ErrMaterialSeguimientoNoDisponible
	}
	contenido, err := io.ReadAll(io.LimitReader(archivo, limiteMaterialPoolsSeguimiento+1))
	if err != nil || len(contenido) == 0 || len(contenido) > limiteMaterialPoolsSeguimiento {
		clear(contenido)
		return vacio, ErrMaterialSeguimientoNoDisponible
	}
	defer clear(contenido)
	if rechazarClavesDuplicadasPools(contenido) != nil {
		return vacio, ErrMaterialSeguimientoNoDisponible
	}
	var documento documentoPoolsSeguimiento
	lector := json.NewDecoder(bytes.NewReader(contenido))
	lector.DisallowUnknownFields()
	if lector.Decode(&documento) != nil || lector.Decode(new(any)) != io.EOF || documento.Version != 1 {
		return vacio, ErrMaterialSeguimientoNoDisponible
	}
	p := documento.Pools
	entradas := []entradaPoolSeguimiento{p.AltaPersonal, p.RegistroCT, p.InicialCT, p.LocalizadorCT,
		p.LocalizadorPersonal, p.LecturaPersonal, p.HistoriaRegistroCT, p.HistoriaAutenticacion,
		p.HistoriaContexto, p.HistoriaEvaluacion, p.HistoriaConcesion}
	for _, entrada := range entradas {
		if entrada.Login == "" || entrada.DSN == "" || strings.TrimSpace(entrada.Login) != entrada.Login ||
			strings.TrimSpace(entrada.DSN) != entrada.DSN {
			return vacio, ErrMaterialSeguimientoNoDisponible
		}
	}
	return MaterialPoolsSeguimiento{
		AltaPersonal:          MaterialPoolSeguimiento{DSN: p.AltaPersonal.DSN, Login: p.AltaPersonal.Login},
		RegistroCT:            MaterialPoolSeguimiento{DSN: p.RegistroCT.DSN, Login: p.RegistroCT.Login},
		InicialCT:             MaterialPoolSeguimiento{DSN: p.InicialCT.DSN, Login: p.InicialCT.Login},
		LocalizadorCT:         MaterialPoolSeguimiento{DSN: p.LocalizadorCT.DSN, Login: p.LocalizadorCT.Login},
		LocalizadorPersonal:   MaterialPoolSeguimiento{DSN: p.LocalizadorPersonal.DSN, Login: p.LocalizadorPersonal.Login},
		LecturaPersonal:       MaterialPoolSeguimiento{DSN: p.LecturaPersonal.DSN, Login: p.LecturaPersonal.Login},
		HistoriaRegistroCT:    MaterialPoolSeguimiento{DSN: p.HistoriaRegistroCT.DSN, Login: p.HistoriaRegistroCT.Login},
		HistoriaAutenticacion: MaterialPoolSeguimiento{DSN: p.HistoriaAutenticacion.DSN, Login: p.HistoriaAutenticacion.Login},
		HistoriaContexto:      MaterialPoolSeguimiento{DSN: p.HistoriaContexto.DSN, Login: p.HistoriaContexto.Login},
		HistoriaEvaluacion:    MaterialPoolSeguimiento{DSN: p.HistoriaEvaluacion.DSN, Login: p.HistoriaEvaluacion.Login},
		HistoriaConcesion:     MaterialPoolSeguimiento{DSN: p.HistoriaConcesion.DSN, Login: p.HistoriaConcesion.Login},
	}, nil
}

// La forma JSON es pequeña y cerrada; detectar duplicados impide que dos
// intérpretes den distinto significado a un mismo material de seguridad.
func rechazarClavesDuplicadasPools(contenido []byte) error {
	lector := json.NewDecoder(bytes.NewReader(contenido))
	var recorrer func() error
	recorrer = func() error {
		token, err := lector.Token()
		if err != nil {
			return err
		}
		apertura, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch apertura {
		case '{':
			vistas := map[string]bool{}
			for lector.More() {
				clave, err := lector.Token()
				if err != nil {
					return err
				}
				nombre, ok := clave.(string)
				if !ok || vistas[nombre] {
					return ErrMaterialSeguimientoNoDisponible
				}
				vistas[nombre] = true
				if err := recorrer(); err != nil {
					return err
				}
			}
		case '[':
			for lector.More() {
				if err := recorrer(); err != nil {
					return err
				}
			}
		default:
			return ErrMaterialSeguimientoNoDisponible
		}
		_, err = lector.Token()
		return err
	}
	if err := recorrer(); err != nil {
		return err
	}
	if _, err := lector.Token(); err != io.EOF {
		return ErrMaterialSeguimientoNoDisponible
	}
	return nil
}

// cargarPoolsConsultaSeguimiento devuelve propiedad de los once pools solo
// cuando todas sus sondas han pasado. No afirma que el resto de proveedores
// (sesiones, F1, V3, detalle y auditoría) esté ya compuesto.
func cargarPoolsConsultaSeguimiento(ctx context.Context) (inc.ConfiguracionServidorV2PostgreSQL, error) {
	var vacio inc.ConfiguracionServidorV2PostgreSQL
	if ctx == nil || ctx.Err() != nil {
		return vacio, ErrMaterialSeguimientoNoDisponible
	}
	material, err := cargarMaterialPoolsSeguimiento(os.Getenv(EnvMaterialInterno))
	if err != nil {
		return vacio, err
	}
	pools, err := AbrirPoolsSeguimiento(ctx, material)
	if err != nil {
		return vacio, ErrMaterialSeguimientoNoDisponible
	}
	return pools, nil
}

// PoolsIdentidadInterna mantiene separados registro, revalidacion, contexto y
// auditoria. Sus constructores propietarios acreditan después la ACL nominal.
type PoolsIdentidadInterna struct {
	Registro, Revalidacion, Contexto, Auditoria *pgxpool.Pool
}

func (p PoolsIdentidadInterna) Cerrar() {
	cerrarPoolsSeguimiento([]*pgxpool.Pool{p.Registro, p.Revalidacion, p.Contexto, p.Auditoria})
}

type documentoPoolsIdentidad struct {
	Version int `json:"version"`
	Pools   struct {
		Registro     entradaPoolSeguimiento `json:"registro"`
		Revalidacion entradaPoolSeguimiento `json:"revalidacion"`
		Contexto     entradaPoolSeguimiento `json:"contexto"`
		Auditoria    entradaPoolSeguimiento `json:"auditoria"`
	} `json:"pools"`
}

func leerDocumentoIdentidadPrivado(base, nombre string) ([]byte, error) {
	if !filepath.IsAbs(base) || filepath.Clean(base) != base || strings.TrimSpace(base) != base {
		return nil, ErrMaterialSeguimientoNoDisponible
	}
	infoBase, err := os.Lstat(base)
	if err != nil || !infoBase.IsDir() || infoBase.Mode().Perm() != 0700 {
		return nil, ErrMaterialSeguimientoNoDisponible
	}
	raiz, err := os.OpenRoot(base)
	if err != nil {
		return nil, ErrMaterialSeguimientoNoDisponible
	}
	defer raiz.Close()
	actualBase, err := raiz.Stat(".")
	if err != nil || !os.SameFile(infoBase, actualBase) || actualBase.Mode().Perm() != 0700 {
		return nil, ErrMaterialSeguimientoNoDisponible
	}
	infoIdentidad, err := raiz.Lstat("identidad")
	if err != nil || !infoIdentidad.IsDir() || infoIdentidad.Mode().Perm() != 0700 {
		return nil, ErrMaterialSeguimientoNoDisponible
	}
	sub, err := raiz.OpenRoot("identidad")
	if err != nil {
		return nil, ErrMaterialSeguimientoNoDisponible
	}
	defer sub.Close()
	actualIdentidad, err := sub.Stat(".")
	if err != nil || !os.SameFile(infoIdentidad, actualIdentidad) || actualIdentidad.Mode().Perm() != 0700 {
		return nil, ErrMaterialSeguimientoNoDisponible
	}
	info, err := sub.Lstat(nombre)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 || info.Size() <= 0 || info.Size() > limiteMaterialPoolsSeguimiento {
		return nil, ErrMaterialSeguimientoNoDisponible
	}
	f, err := sub.Open(nombre)
	if err != nil {
		return nil, ErrMaterialSeguimientoNoDisponible
	}
	defer f.Close()
	actual, err := f.Stat()
	if err != nil || !os.SameFile(info, actual) || !actual.Mode().IsRegular() || actual.Mode().Perm() != 0600 {
		return nil, ErrMaterialSeguimientoNoDisponible
	}
	contenido, err := io.ReadAll(io.LimitReader(f, limiteMaterialPoolsSeguimiento+1))
	if err != nil || len(contenido) == 0 || len(contenido) > limiteMaterialPoolsSeguimiento {
		clear(contenido)
		return nil, ErrMaterialSeguimientoNoDisponible
	}
	if rechazarClavesDuplicadasPools(contenido) != nil {
		clear(contenido)
		return nil, ErrMaterialSeguimientoNoDisponible
	}
	return contenido, nil
}

func cargarMaterialPoolsIdentidad(base string, ct MaterialPoolsSeguimiento) (documentoPoolsIdentidad, error) {
	var vacio documentoPoolsIdentidad
	contenido, err := leerDocumentoIdentidadPrivado(base, "pools.json")
	if err != nil {
		return vacio, err
	}
	defer clear(contenido)
	lector := json.NewDecoder(bytes.NewReader(contenido))
	lector.DisallowUnknownFields()
	var documento documentoPoolsIdentidad
	if lector.Decode(&documento) != nil || lector.Decode(new(any)) != io.EOF || documento.Version != 1 {
		return vacio, ErrMaterialSeguimientoNoDisponible
	}
	reservados := map[string]bool{}
	for _, perfil := range perfilesPoolsSeguimiento(ct) {
		if perfil.material.Login == "" || reservados[perfil.material.Login] {
			return vacio, ErrMaterialSeguimientoNoDisponible
		}
		reservados[perfil.material.Login] = true
	}
	for _, p := range []entradaPoolSeguimiento{documento.Pools.Registro, documento.Pools.Revalidacion, documento.Pools.Contexto, documento.Pools.Auditoria} {
		if p.Login == "" || p.DSN == "" || strings.TrimSpace(p.Login) != p.Login || strings.TrimSpace(p.DSN) != p.DSN || reservados[p.Login] {
			return vacio, ErrMaterialSeguimientoNoDisponible
		}
		reservados[p.Login] = true
	}
	return documento, nil
}

func abrirPoolsIdentidadInterna(ctx context.Context, base string, ct MaterialPoolsSeguimiento) (PoolsIdentidadInterna, error) {
	var vacio PoolsIdentidadInterna
	if ctx == nil || ctx.Err() != nil {
		return vacio, ErrMaterialSeguimientoNoDisponible
	}
	m, err := cargarMaterialPoolsIdentidad(base, ct)
	if err != nil {
		return vacio, err
	}
	perfiles := []struct {
		material     entradaPoolSeguimiento
		rol, funcion string
		hereda       bool
	}{
		{m.Pools.Registro, "vec_identidad_sesiones_v1_registrador", "vec_identidad_sesiones_v1.registrar_sesion_v1(text,text,text,text,bigint,bytea,bytea,bytea,bytea,bytea,boolean,text,text,text,text,timestamptz,timestamptz,timestamptz,text,text)", false},
		{m.Pools.Revalidacion, "vec_identidad_sesiones_v1_revalidador", "vec_identidad_sesiones_v1.coincide_politica_certificado_desarrollo_v1(text,text,timestamptz)", false},
		{m.Pools.Contexto, "vec_contexto_actor_v1_runtime", "vec_contexto_actor_v1.acreditar_runtime_contexto_actor_v1()", false},
		{m.Pools.Auditoria, "vec_contratacion_temporal_registrador_frontera", "vec_contratacion_temporal.registrar_auditoria_frontera_ruta_exacta_v1(text,text,text,text,text)", true},
	}
	abiertos := make([]*pgxpool.Pool, 0, 4)
	defer func() {
		if err != nil {
			cerrarPoolsSeguimiento(abiertos)
		}
	}()
	for _, perfil := range perfiles {
		var configuracion *pgxpool.Config
		configuracion, err = configurarPoolIdentidadInterna(perfil.material, perfil.rol, perfil.funcion, perfil.hereda)
		if err != nil {
			return vacio, ErrMaterialSeguimientoNoDisponible
		}
		var pool *pgxpool.Pool
		pool, err = pgxpool.NewWithConfig(ctx, configuracion)
		if err != nil {
			return vacio, ErrMaterialSeguimientoNoDisponible
		}
		abiertos = append(abiertos, pool)
		sonda, cancelar := context.WithTimeout(ctx, plazoSondaPoolSeguimiento)
		err = pool.Ping(sonda)
		cancelar()
		if err != nil {
			return vacio, ErrMaterialSeguimientoNoDisponible
		}
	}
	return PoolsIdentidadInterna{Registro: abiertos[0], Revalidacion: abiertos[1], Contexto: abiertos[2], Auditoria: abiertos[3]}, nil
}

func configurarPoolIdentidadInterna(material entradaPoolSeguimiento, rol, funcion string, hereda bool) (*pgxpool.Config, error) {
	c, err := pgxpool.ParseConfig(material.DSN)
	if err != nil || c == nil || c.ConnConfig == nil || c.ConnConfig.User != material.Login ||
		!tlsPoolSeguimientoVerificado(&c.ConnConfig.Config) {
		return nil, ErrMaterialSeguimientoNoDisponible
	}
	c.MaxConns, c.MinConns, c.MinIdleConns = 2, 0, 0
	c.ConnConfig.ConnectTimeout = plazoConexionPoolSeguimiento
	c.PingTimeout = plazoSondaPoolSeguimiento
	c.MaxConnLifetime, c.MaxConnIdleTime = 30*time.Minute, 5*time.Minute
	if c.ConnConfig.RuntimeParams == nil {
		c.ConnConfig.RuntimeParams = map[string]string{}
	}
	for k, v := range map[string]string{
		"application_name": "vec-interno-identidad", "timezone": "UTC", "search_path": "pg_catalog,pg_temp",
		"default_transaction_isolation": "serializable", "statement_timeout": "15s", "lock_timeout": "3s",
		"idle_in_transaction_session_timeout": "15s",
	} {
		c.ConnConfig.RuntimeParams[k] = v
	}
	c.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		sonda, cancelar := context.WithTimeout(ctx, plazoSondaPoolSeguimiento)
		defer cancelar()
		var usuario, efectivo string
		var loginValido, aclValida bool
		err := conn.QueryRow(sonda, consultaACLPoolSeguimiento, rol, funcion, hereda).
			Scan(&usuario, &efectivo, &loginValido, &aclValida)
		if err != nil || usuario != material.Login || efectivo != usuario ||
			!loginValido || !aclValida {
			return ErrMaterialSeguimientoNoDisponible
		}
		return nil
	}
	return c, nil
}

type documentoHMACIdentidad struct {
	Version          int    `json:"version"`
	Modulo           string `json:"modulo"`
	TokenLabel       string `json:"token_label"`
	TokenSerial      string `json:"token_serial"`
	ObjetoIDHex      string `json:"objeto_id_hex"`
	ClaveID          string `json:"clave_id"`
	ClaveVersion     uint64 `json:"clave_version"`
	DominioRef       string `json:"dominio_ref"`
	EspacioIdentidad string `json:"espacio_identidad"`
	PINFichero       string `json:"pin_fichero"`
}

func cargarConfiguracionHMACIdentidad(base string) (seudonimizacionpkcs11.Configuracion, error) {
	var vacio seudonimizacionpkcs11.Configuracion
	contenido, err := leerDocumentoIdentidadPrivado(base, "hmac.json")
	if err != nil {
		return vacio, err
	}
	defer clear(contenido)
	lector := json.NewDecoder(bytes.NewReader(contenido))
	lector.DisallowUnknownFields()
	var d documentoHMACIdentidad
	if lector.Decode(&d) != nil || lector.Decode(new(any)) != io.EOF || d.Version != 1 {
		return vacio, ErrMaterialSeguimientoNoDisponible
	}
	objetoID, err := hex.DecodeString(d.ObjetoIDHex)
	if err != nil || len(objetoID) == 0 || len(objetoID) > 64 || hex.EncodeToString(objetoID) != d.ObjetoIDHex ||
		!filepath.IsAbs(d.Modulo) || filepath.Clean(d.Modulo) != d.Modulo ||
		!textoTecnicoMaterial(d.TokenLabel, 32) || !textoTecnicoMaterial(d.TokenSerial, 32) ||
		!textoTecnicoMaterial(d.ClaveID, 128) || d.ClaveVersion == 0 || d.ClaveVersion > 1<<63-1 ||
		!dominioHMACMaterialValido(d.DominioRef) || !espacioIdentidadMaterialValido(d.EspacioIdentidad) ||
		d.PINFichero != filepath.Join(base, "identidad", "hmac.pin") {
		return vacio, ErrMaterialSeguimientoNoDisponible
	}
	info, err := os.Lstat(d.PINFichero)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 || info.Size() < 1 || info.Size() > 256 {
		return vacio, ErrMaterialSeguimientoNoDisponible
	}
	propietario, ok := info.Sys().(*syscall.Stat_t)
	if !ok || propietario.Uid != uint32(os.Geteuid()) && propietario.Uid != 0 {
		return vacio, ErrMaterialSeguimientoNoDisponible
	}
	return seudonimizacionpkcs11.Configuracion{Modulo: d.Modulo, TokenLabel: d.TokenLabel, TokenSerial: d.TokenSerial,
		ObjetoID: objetoID, ClaveID: d.ClaveID, ClaveVersion: d.ClaveVersion, DominioRef: d.DominioRef,
		EspacioIdentidad: d.EspacioIdentidad, PINFichero: d.PINFichero}, nil
}

func textoTecnicoMaterial(v string, max int) bool {
	if v == "" || len(v) > max || strings.TrimSpace(v) != v {
		return false
	}
	for _, r := range v {
		if r <= 0x20 || r >= 0x7f {
			return false
		}
	}
	return true
}

func dominioHMACMaterialValido(v string) bool {
	if !strings.HasPrefix(v, "idh_") || len(v) < 26 || len(v) > 132 {
		return false
	}
	for _, r := range v[4:] {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-') {
			return false
		}
	}
	return true
}

func espacioIdentidadMaterialValido(v string) bool {
	if !textoTecnicoMaterial(v, 512) {
		return false
	}
	u, err := url.Parse(v)
	return err == nil && strings.EqualFold(u.Scheme, "https") && u.Host != "" && u.User == nil && u.RawQuery == "" && u.Fragment == ""
}

type documentoContextosIdentidad struct {
	Version   int `json:"version"`
	Contextos []struct {
		CuentaRef       string `json:"cuenta_ref"`
		PerfilRef       string `json:"perfil_ref"`
		OrganizacionRef string `json:"organizacion_ref"`
		UnidadRef       string `json:"unidad_ref"`
	} `json:"contextos"`
}

// El selector privado no atribuye persona, rol ni permiso. La fuente F1
// nominal y V3 revalidan esas autoridades vivas en cada petición.
func cargarContextosNominalesIdentidad(base string) (map[string]internagobierno.ContextoNominal, error) {
	contenido, err := leerDocumentoIdentidadPrivado(base, "contextos.json")
	if err != nil {
		return nil, err
	}
	defer clear(contenido)
	lector := json.NewDecoder(bytes.NewReader(contenido))
	lector.DisallowUnknownFields()
	var d documentoContextosIdentidad
	if lector.Decode(&d) != nil || lector.Decode(new(any)) != io.EOF || d.Version != 1 ||
		len(d.Contextos) == 0 || len(d.Contextos) > 1024 {
		return nil, ErrMaterialSeguimientoNoDisponible
	}
	porCuenta := make(map[string]internagobierno.ContextoNominal, len(d.Contextos))
	for _, c := range d.Contextos {
		if c.CuentaRef == "" || c.PerfilRef == "" || c.OrganizacionRef == "" || c.UnidadRef == "" ||
			strings.TrimSpace(c.CuentaRef) != c.CuentaRef || strings.TrimSpace(c.PerfilRef) != c.PerfilRef ||
			strings.TrimSpace(c.OrganizacionRef) != c.OrganizacionRef || strings.TrimSpace(c.UnidadRef) != c.UnidadRef ||
			porCuenta[c.CuentaRef] != (internagobierno.ContextoNominal{}) {
			return nil, ErrMaterialSeguimientoNoDisponible
		}
		porCuenta[c.CuentaRef] = internagobierno.ContextoNominal{
			PerfilActivoRef: c.PerfilRef, OrganizacionRef: c.OrganizacionRef, UnidadRef: c.UnidadRef,
		}
	}
	return porCuenta, nil
}
