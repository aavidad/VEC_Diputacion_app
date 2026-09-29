package bootstrap

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/app/separacionportales"
	postgresidentidad "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
	core "vec-diputacion-granada/internal/vec/domain"
)

// espacioSeudonimosPortalExterno es el espacio de la clave con la que el
// proceso externo seudonimiza cuentas y sujetos de sesión. Distinto del
// interno: un alias del externo nunca coincide con uno interno y el lado
// interno solo registra alias de este espacio.
const espacioSeudonimosPortalExterno = "vec.identidad.desarrollo.externo"

var ErrSeudonimosPortalExternoInvalidos = errors.New("bootstrap: seudonimos del portal externo no validos")

// seudonimoCuentaPortalExterno es lo que el proceso externo entrega para que
// el lado interno registre el alias de una de sus cuentas: la cuenta y dos
// huellas HMAC. No contiene la clave ni los identificadores en claro.
type seudonimoCuentaPortalExterno struct {
	CuentaRef    string `json:"cuenta_ref"`
	Esquema      string `json:"esquema"`
	DominioRef   string `json:"dominio_ref"`
	ClaveID      string `json:"clave_id"`
	ClaveVersion uint64 `json:"clave_version"`
	CuentaHMAC   string `json:"cuenta_id_hmac"`
	SujetoHMAC   string `json:"sujeto_id_hmac"`
}

type seudonimosPortalExterno struct {
	Version int                            `json:"version"`
	Cuentas []seudonimoCuentaPortalExterno `json:"cuentas"`
}

// ExportarSeudonimosPortalExterno se ejecuta en el proceso externo, sin
// conexiones: calcula con su propia clave los alias de las cuentas del Área
// personal que tiene en su material. El resultado lo registra después el
// lado interno con PrepararMaterialPortalExterno.
func ExportarSeudonimosPortalExterno(cfg config.Config) ([]byte, error) {
	cfg = cfg.Normalize()
	portal, err := separacionportales.Parsear(cfg.PortalProceso)
	if err != nil || portal != separacionportales.PortalExterno || !cfg.DevelopmentEnabledByDoubleKey() {
		return nil, ErrSeudonimosPortalExternoInvalidos
	}
	if err := separacionportales.ComprobarMaterial(portal, cfg.DevelopmentMaterialDir); err != nil {
		return nil, err
	}
	raiz := cfg.DevelopmentMaterialDir
	idempotencia, err := cargarMaterialIdempotenciaDesarrollo(raiz, filepath.Join(raiz, config.DevelopmentIdempotencyHMACConfigRelativePath))
	if err != nil {
		return nil, ErrSeudonimosPortalExternoInvalidos
	}
	derivador, err := nuevoDerivadorIdentidadOperacionDesarrollo(&idempotencia)
	idempotencia.borrar()
	if err != nil {
		return nil, ErrSeudonimosPortalExternoInvalidos
	}
	defer derivador.borrar()
	derivador.espacioSeudonimos = espacioSeudonimosPortalExterno
	seudonimizador := &seudonimizadorSesionDesarrollo{derivador: derivador}
	material, err := cargarMaterialPortalExterno(cfg)
	if err != nil || material.identidad == nil || material.identidad.candidatoBolsa == nil {
		return nil, ErrSeudonimosPortalExternoInvalidos
	}
	cuentas := []struct{ ref, sujeto string }{{
		ref:    material.identidad.candidatoBolsa.cuentaRef,
		sujeto: material.identidad.candidatoBolsa.identidad.principal.ID,
	}}
	rutaPreferencias := filepath.Join(raiz, "identidad", nombreConfiguracionPreferencias(core.SuperficieAutenticacionExternaPersonalV1))
	if _, errFichero := os.Lstat(rutaPreferencias); errFichero == nil {
		c, err := leerConfiguracionUsuariosPreferenciasDesarrollo(cfg, core.SuperficieAutenticacionExternaPersonalV1)
		if err != nil {
			return nil, ErrSeudonimosPortalExternoInvalidos
		}
		for _, cuenta := range c.Cuentas {
			cuentas = append(cuentas, struct{ ref, sujeto string }{cuenta.CuentaRef, cuenta.Sujeto})
		}
	} else if !errors.Is(errFichero, os.ErrNotExist) {
		return nil, ErrSeudonimosPortalExternoInvalidos
	}
	vistas := map[string]string{}
	resultado := seudonimosPortalExterno{Version: 1}
	for _, cuenta := range cuentas {
		if sujeto, repetida := vistas[cuenta.ref]; repetida {
			if sujeto != cuenta.sujeto {
				return nil, ErrSeudonimosPortalExternoInvalidos
			}
			continue
		}
		vistas[cuenta.ref] = cuenta.sujeto
		// Los mismos identificadores que usa resolverSesion en cada petición.
		s, err := seudonimizador.SeudonimizarAlta(context.Background(), postgresidentidad.IdentificadoresAlta{
			EspacioIdentidad: espacioIdentidadSesionDesarrollo,
			AsercionID:       "preparacion-cuenta", SesionID: "preparacion-alias",
			CuentaID: "desarrollo:" + cuenta.ref, SujetoID: cuenta.sujeto,
		})
		if err != nil {
			return nil, ErrSeudonimosPortalExternoInvalidos
		}
		resultado.Cuentas = append(resultado.Cuentas, seudonimoCuentaPortalExterno{
			CuentaRef: cuenta.ref, Esquema: s.Esquema, DominioRef: s.DominioRef,
			ClaveID: s.ClaveID, ClaveVersion: s.ClaveVersion,
			CuentaHMAC: hex.EncodeToString(s.CuentaIDHMAC[:]), SujetoHMAC: hex.EncodeToString(s.SujetoIDHMAC[:]),
		})
	}
	return json.MarshalIndent(resultado, "", "  ")
}

// leerSeudonimosPortalExterno valida lo que entrega el proceso externo antes
// de registrarlo: solo alias del espacio externo, con huellas bien formadas.
func leerSeudonimosPortalExterno(contenido []byte) (seudonimosPortalExterno, error) {
	var s seudonimosPortalExterno
	if len(contenido) == 0 || len(contenido) > 256<<10 || validarClavesJSONUnicas(contenido) != nil {
		return s, ErrSeudonimosPortalExternoInvalidos
	}
	d := json.NewDecoder(bytes.NewReader(contenido))
	d.DisallowUnknownFields()
	var sobra any
	// El material admite hasta 64 preferencias y una cuenta Bolsa distinta.
	if d.Decode(&s) != nil || !errors.Is(d.Decode(&sobra), io.EOF) || s.Version != 1 || len(s.Cuentas) == 0 || len(s.Cuentas) > 65 {
		return seudonimosPortalExterno{}, ErrSeudonimosPortalExternoInvalidos
	}
	for _, c := range s.Cuentas {
		cuenta, errC := hex.DecodeString(c.CuentaHMAC)
		sujeto, errS := hex.DecodeString(c.SujetoHMAC)
		if !strings.HasPrefix(c.CuentaRef, "cta_") || c.Esquema != postgresidentidad.EsquemaHMACSHA256V1 ||
			c.DominioRef != dominioIdentidadSesionDesarrollo || !strings.HasPrefix(c.ClaveID, espacioSeudonimosPortalExterno+".g") ||
			c.ClaveVersion < 1 || errC != nil || errS != nil || len(cuenta) != 32 || len(sujeto) != 32 || bytes.Equal(cuenta, sujeto) {
			return seudonimosPortalExterno{}, ErrSeudonimosPortalExternoInvalidos
		}
	}
	return s, nil
}

const registrarAliasPortalExternoSQL = `SELECT vec_identidad_sesiones_v1.registrar_alias_hmac_cuenta_v1(
    $1::text, $2::text, $3::text, $4::text, $5::text, $6::bigint, $7::bytea, $8::bytea)`

// cuentaPrivilegiadaOVinculadaSQL es verdadero si la cuenta es privilegiada
// o si es la cuenta ordinaria de una persona que además tiene una
// privilegiada. Si la cuenta no existe no devuelve fila y se rechaza.
const cuentaPrivilegiadaOVinculadaSQL = `SELECT c.cuenta_privilegiada OR EXISTS(
 SELECT 1 FROM vec_identidad_sesiones_v1.cuenta p WHERE p.cuenta_ordinaria_ref=c.cuenta_ref)
 FROM vec_identidad_sesiones_v1.cuenta c WHERE c.cuenta_ref=$1::text`

// registrarSeudonimosPortalExterno registra, con el rol de gobierno y de
// forma idempotente, los alias que calculó el proceso externo. La cuenta
// debe existir y estar activa; si no, no se registra nada.
func registrarSeudonimosPortalExterno(ctx context.Context, gobierno *pgxpool.Pool, s seudonimosPortalExterno) error {
	if ctx == nil || gobierno == nil {
		return ErrSeudonimosPortalExternoInvalidos
	}
	tx, err := gobierno.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return ErrSeudonimosPortalExternoInvalidos
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	if _, err = tx.Exec(ctx, configurarCuentaNominalDesarrolloSQL); err != nil {
		return ErrSeudonimosPortalExternoInvalidos
	}
	for _, c := range s.Cuentas {
		// Nunca una cuenta privilegiada, aunque figure en la lista positiva.
		var privilegiada bool
		if err := tx.QueryRow(ctx, cuentaPrivilegiadaOVinculadaSQL, c.CuentaRef).Scan(&privilegiada); err != nil || privilegiada {
			return ErrSeudonimosPortalExternoInvalidos
		}
		cuenta, _ := hex.DecodeString(c.CuentaHMAC)
		sujeto, _ := hex.DecodeString(c.SujetoHMAC)
		operacion := referenciaAltaContratacionTemporalDesarrollo("opr_externo_alias_",
			c.CuentaRef+"\x00"+c.ClaveID+"\x00"+c.CuentaHMAC+"\x00"+c.SujetoHMAC)
		var registrada *string
		if err := tx.QueryRow(ctx, registrarAliasPortalExternoSQL, operacion, c.CuentaRef, c.Esquema, c.DominioRef,
			c.ClaveID, int64(c.ClaveVersion), cuenta, sujeto).Scan(&registrada); err != nil || registrada == nil || *registrada != c.CuentaRef {
			return ErrSeudonimosPortalExternoInvalidos
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return ErrSeudonimosPortalExternoInvalidos
	}
	return nil
}
