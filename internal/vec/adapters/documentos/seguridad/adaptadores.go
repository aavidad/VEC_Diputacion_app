// Package seguridad contiene adaptadores criptograficos y de infraestructura
// local. En produccion, la clave HMAC debe proceder de un gestor de secretos o
// KMS y nunca de una imagen, repositorio o variable mostrada en logs.
package seguridad

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/ports"
)

var ErrConfiguracionSeguridadInvalida = errors.New("documentos: configuracion de seguridad invalida")

type SelladorHMAC struct {
	idClave string
	clave   []byte
}

func NuevoSelladorHMAC(idClave string, clave []byte) (*SelladorHMAC, error) {
	if strings.TrimSpace(idClave) == "" || len(clave) < 32 {
		return nil, ErrConfiguracionSeguridadInvalida
	}
	copia := append([]byte(nil), clave...)
	return &SelladorHMAC{idClave: strings.TrimSpace(idClave), clave: copia}, nil
}

func (s *SelladorHMAC) SellarDatos(ctx context.Context, datos []byte) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if s == nil || len(s.clave) < 32 || strings.TrimSpace(s.idClave) == "" {
		return "", ErrConfiguracionSeguridadInvalida
	}
	mac := hmac.New(sha256.New, s.clave)
	_, _ = mac.Write(datos)
	return "hmac-sha256:" + s.idClave + ":" + hex.EncodeToString(mac.Sum(nil)), nil
}

// SellarSolicitudDocumento permite usar otra instancia de SelladorHMAC, con
// otra clave e identificador, para la huella idempotente. El metodo separado
// evita que el ensamblado de produccion reutilice una clave por accidente.
func (s *SelladorHMAC) SellarSolicitudDocumento(ctx context.Context, datos []byte) (string, error) {
	return s.SellarDatos(ctx, datos)
}

// SeudonimizarSujetoAlmacen permite dedicar una instancia y una clave
// exclusivas a la seudonimizacion tecnica del almacen. El ensamblado no debe
// reutilizar aqui el sellador de datos ni el de idempotencia.
func (s *SelladorHMAC) SeudonimizarSujetoAlmacen(
	ctx context.Context,
	solicitud ports.SolicitudSeudonimizarSujetoAlmacen,
) (string, error) {
	sujetoRef, ambitoRef, err := solicitud.RevelarParaSellado()
	if err != nil {
		return "", err
	}
	canonico := fmt.Sprintf("%d:%s\n%d:%s\n", len(sujetoRef), sujetoRef, len(ambitoRef), ambitoRef)
	return s.SellarDatos(ctx, []byte(canonico))
}

type GeneradorID struct{}

func (GeneradorID) NuevoIDDocumento() (string, error) {
	var valor [16]byte
	if _, err := rand.Read(valor[:]); err != nil {
		return "", fmt.Errorf("documentos: entropia de identificador: %w", err)
	}
	valor[6] = (valor[6] & 0x0f) | 0x40
	valor[8] = (valor[8] & 0x3f) | 0x80
	return fmt.Sprintf("documento-%x-%x-%x-%x-%x", valor[0:4], valor[4:6], valor[6:8], valor[8:10], valor[10:16]), nil
}

type RelojSistema struct{}

func (RelojSistema) Ahora() time.Time {
	return time.Now().UTC()
}
