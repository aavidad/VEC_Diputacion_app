package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"time"

	"vec-diputacion-granada/internal/modules/usuarios/adapters/seguridad"
	"vec-diputacion-granada/internal/modules/usuarios/ports"
)

type fuenteEfimera struct{ claves seguridad.ClavesCorreos }

func (f *fuenteEfimera) CargarClavesCorreos(context.Context) (seguridad.ClavesCorreos, error) {
	return f.claves, nil
}

func (f *fuenteEfimera) borrar() {
	clear(f.claves.CifradoActivo.Material[:])
	clear(f.claves.Igualdad.Material[:])
	clear(f.claves.SemanticaActiva.Material[:])
	clear(f.claves.CodigoActivo.Material[:])
}

// Estos dominios son los protocolos ya publicados del KMS de desarrollo.
// La fuente combinada deriva primero la envoltura; la externa usa SU semilla.
func fuenteDesdeSemilla(semilla [32]byte, externa bool) *fuenteEfimera {
	defer clear(semilla[:])
	ref, dominio := "clave:kms:desarrollo:usuarios-correos-", "vec.kms.desarrollo.usuarios-correos."
	if externa {
		ref, dominio = ref+"externo-", dominio+"externo."
	} else {
		semilla = derivar(semilla, "vec.kms.desarrollo.envoltura.v1")
	}
	clave := func(ambito string) seguridad.ClaveCorreo {
		return seguridad.ClaveCorreo{Ref: ref + ambito + ":v1", Material: derivar(semilla, dominio+ambito+".v1")}
	}
	return &fuenteEfimera{seguridad.ClavesCorreos{CifradoActivo: clave("cifrado"), Igualdad: clave("igualdad"), SemanticaActiva: clave("semantica"), CodigoActivo: clave("codigo")}}
}

func derivar(semilla [32]byte, dominio string) [32]byte {
	defer clear(semilla[:])
	mac := hmac.New(sha256.New, semilla[:])
	_, _ = mac.Write([]byte(dominio))
	var resultado [32]byte
	copy(resultado[:], mac.Sum(nil))
	return resultado
}

type direccionDB struct {
	CorreoRef     string `json:"correo_ref"`
	Version       uint64 `json:"version_sobre"`
	ClaveSobre    string `json:"clave_sobre_ref"`
	ClaveIgualdad string `json:"clave_igualdad_ref"`
	Nonce         string `json:"nonce"`
	Cifrado       string `json:"cifrado"`
	Igualdad      string `json:"huella_igualdad"`
}

type direccionNueva struct {
	CorreoRef string `json:"correo_ref"`
	Version   uint64 `json:"version_sobre"`
	Nonce     string `json:"nonce_hex"`
	Cifrado   string `json:"cifrado_hex"`
	Igualdad  string `json:"huella_igualdad_hex"`
}

type materialNuevo struct {
	ClaveSobre     string           `json:"clave_sobre_ref"`
	ClaveIgualdad  string           `json:"clave_igualdad_ref"`
	ClaveCodigo    string           `json:"clave_codigo_ref"`
	ClaveSemantica string           `json:"clave_semantica_ref"`
	Direcciones    []direccionNueva `json:"direcciones"`
}

func convertir(ctx context.Context, persona string, filas []direccionDB, vieja, propia *fuenteEfimera) (materialNuevo, error) {
	m := materialNuevo{ClaveSobre: propia.claves.CifradoActivo.Ref, ClaveIgualdad: propia.claves.Igualdad.Ref,
		ClaveCodigo: propia.claves.CodigoActivo.Ref, ClaveSemantica: propia.claves.SemanticaActiva.Ref, Direcciones: []direccionNueva{}}
	origen, _ := seguridad.NuevoAdaptadorCorreos(vieja, time.Now)
	destino, _ := seguridad.NuevoAdaptadorCorreos(propia, time.Now)
	for _, fila := range filas {
		nonce, e1 := bytea(fila.Nonce)
		cifrado, e2 := bytea(fila.Cifrado)
		igualdad, e3 := bytea(fila.Igualdad)
		if e1 != nil || e2 != nil || e3 != nil || fila.ClaveSobre != vieja.claves.CifradoActivo.Ref || fila.ClaveIgualdad != vieja.claves.Igualdad.Ref {
			return materialNuevo{}, errCripto
		}
		sobre := ports.SobreDireccionCorreo{CorreoRef: fila.CorreoRef, Version: fila.Version, ClaveRef: fila.ClaveSobre,
			ClaveIgualdadRef: fila.ClaveIgualdad, Nonce: nonce, Cifrado: cifrado, HuellaIgualdad: igualdad}
		err := origen.ConDireccionCorreoDescifrada(ctx, persona, sobre, func(claro []byte) error {
			// Verifica también la huella vieja, además de la autenticidad AEAD.
			comprobacion, err := origen.CifrarDireccionCorreo(ctx, persona, fila.CorreoRef, fila.Version, claro)
			if err != nil || subtle.ConstantTimeCompare(comprobacion.HuellaIgualdad, igualdad) != 1 {
				return errCripto
			}
			nuevo, err := destino.CifrarDireccionCorreo(ctx, persona, fila.CorreoRef, fila.Version, claro)
			if err != nil {
				return errCripto
			}
			// Comprobación de ida y vuelta con la clave propia antes del CAS.
			if err := destino.ConDireccionCorreoDescifrada(ctx, persona, nuevo, func(recuperado []byte) error {
				if subtle.ConstantTimeCompare(claro, recuperado) != 1 {
					return errCripto
				}
				return nil
			}); err != nil {
				return errCripto
			}
			m.Direcciones = append(m.Direcciones, direccionNueva{fila.CorreoRef, fila.Version, hex.EncodeToString(nuevo.Nonce), hex.EncodeToString(nuevo.Cifrado), hex.EncodeToString(nuevo.HuellaIgualdad)})
			return nil
		})
		if err != nil {
			return materialNuevo{}, errCripto
		}
	}
	return m, nil
}

func bytea(s string) ([]byte, error) {
	if len(s) < 2 || s[:2] != `\x` {
		return nil, errCripto
	}
	return hex.DecodeString(s[2:])
}
