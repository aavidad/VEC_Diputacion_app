package bootstrap

import (
	"context"
	"crypto/sha256"
	"errors"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/app/separacionportales"
	usuariosseguridad "vec-diputacion-granada/internal/modules/usuarios/adapters/seguridad"
)

var ErrClavesCorreosPortalExternoNoDisponibles = errors.New("bootstrap: claves de correos externos no disponibles")

const sondaCorreosPortalExternoSinClavesAjenasSQL = `SELECT vec_usuarios_correos_externo.poblacion_sin_claves_ajenas_v1()`

type consultaPoblacionCorreosPortalExterno interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func acreditarPoblacionCorreosPortalExternoSinClavesAjenas(ctx context.Context, consulta consultaPoblacionCorreosPortalExterno) error {
	if ctx == nil || consulta == nil || ctx.Err() != nil {
		return ErrClavesCorreosPortalExternoNoDisponibles
	}
	var apta bool
	if err := consulta.QueryRow(ctx, sondaCorreosPortalExternoSinClavesAjenasSQL).Scan(&apta); err != nil || !apta {
		return ErrClavesCorreosPortalExternoNoDisponibles
	}
	return nil
}

// fuenteClavesCorreosPortalExterno conserva solo claves derivadas de SU
// material; la semilla de 32 bytes se borra antes de devolverla. No ofrece
// claves retenidas del proceso combinado: cualquier referencia anterior
// deniega el arranque hasta una operación de reclaveado separada y auditada.
type fuenteClavesCorreosPortalExterno struct {
	mu     sync.RWMutex
	claves usuariosseguridad.ClavesCorreos
}

func (f *fuenteClavesCorreosPortalExterno) CargarClavesCorreos(ctx context.Context) (usuariosseguridad.ClavesCorreos, error) {
	if f == nil || ctx == nil || ctx.Err() != nil {
		return usuariosseguridad.ClavesCorreos{}, ErrClavesCorreosPortalExternoNoDisponibles
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	if f.claves.CifradoActivo.Material == ([sha256.Size]byte{}) {
		return usuariosseguridad.ClavesCorreos{}, ErrClavesCorreosPortalExternoNoDisponibles
	}
	return f.claves, nil
}

func (f *fuenteClavesCorreosPortalExterno) borrar() {
	if f != nil {
		f.mu.Lock()
		defer f.mu.Unlock()
		clear(f.claves.CifradoActivo.Material[:])
		clear(f.claves.Igualdad.Material[:])
		clear(f.claves.SemanticaActiva.Material[:])
		clear(f.claves.CodigoActivo.Material[:])
	}
}

// nuevaFuenteClavesCorreosPortalExterno es la única entrada de claves para
// «Mis correos» del proceso externo. Usa la semilla propia de Usuarios, sin
// recurrir a la ruta del KMS interno. Usuarios 000012 acredita antes que no
// quedan referencias de las claves compartidas. #145 aporta los esquemas.
func nuevaFuenteClavesCorreosPortalExterno(ctx context.Context, cfg config.Config, preflight *pgxpool.Pool) (*fuenteClavesCorreosPortalExterno, error) {
	if preflight == nil {
		return nil, ErrClavesCorreosPortalExternoNoDisponibles
	}
	return nuevaFuenteClavesCorreosPortalExternoConConsulta(ctx, cfg, preflight)
}

func nuevaFuenteClavesCorreosPortalExternoConConsulta(ctx context.Context, cfg config.Config, consulta consultaPoblacionCorreosPortalExterno) (*fuenteClavesCorreosPortalExterno, error) {
	cfg = cfg.Normalize()
	if !cfg.DevelopmentEnabledByDoubleKey() || cfg.PortalProceso != string(separacionportales.PortalExterno) ||
		separacionportales.ComprobarMaterial(separacionportales.PortalExterno, cfg.DevelopmentMaterialDir) != nil ||
		acreditarPoblacionCorreosPortalExternoSinClavesAjenas(ctx, consulta) != nil {
		return nil, ErrClavesCorreosPortalExternoNoDisponibles
	}
	semilla, err := leerSecreto32Desarrollo(cfg.DevelopmentPaths().SemillaCorreosExterna)
	if err != nil || semilla == ([sha256.Size]byte{}) {
		clear(semilla[:])
		return nil, ErrClavesCorreosPortalExternoNoDisponibles
	}
	defer clear(semilla[:])
	fuente := &fuenteClavesCorreosPortalExterno{}
	clave := func(ambito string) usuariosseguridad.ClaveCorreo {
		return usuariosseguridad.ClaveCorreo{
			Ref:      "clave:kms:desarrollo:usuarios-correos-externo-" + ambito + ":v1",
			Material: derivarClaveDesarrollo(semilla, "vec.kms.desarrollo.usuarios-correos.externo."+ambito+".v1"),
		}
	}
	fuente.claves = usuariosseguridad.ClavesCorreos{
		CifradoActivo: clave("cifrado"), Igualdad: clave("igualdad"),
		SemanticaActiva: clave("semantica"), CodigoActivo: clave("codigo"),
	}
	return fuente, nil
}
