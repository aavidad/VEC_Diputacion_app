package domain

import (
	"crypto/subtle"
	"errors"
	"regexp"
	"strings"
	"time"
)

var ErrRegistroPropioInvalido = errors.New("vec: registro propio invalido")

var referenciaRegistroPropio = regexp.MustCompile(`^[a-z]{3}_[A-Za-z0-9_-]{22,128}$`)
var huellaRegistroPropio = regexp.MustCompile(`^[0-9a-f]{64}$`)

func referenciaRegistroConPrefijo(valor, prefijo string) bool {
	return strings.HasPrefix(valor, prefijo) && referenciaRegistroPropio.MatchString(valor)
}

const EstadoRegistroPropioPendienteContacto = "pendiente_contacto"

func ReferenciaSujetoRegistroPropioValida(ref string) bool {
	return referenciaRegistroConPrefijo(ref, "suj_")
}

// AcreditacionInstitucionalRegistroPropioV1 procede exclusivamente de una
// autoridad institucional inyectada. Su estructura valida no acredita por si
// sola la credencial: el productor debe revalidarla al crear el efecto.
type AcreditacionInstitucionalRegistroPropioV1 struct {
	CredencialRef        string
	SujetoRef            string
	DominioHMACRef       string
	ClaveHMACID          string
	ClaveHMACVersion     uint64
	CuentaHuellaHMAC     []byte
	SujetoHuellaHMAC     []byte
	EquivalenciaRef      string
	ProcedenciaRef       string
	ProcedenciaVersion   uint64
	ProcedenciaSHA256    string
	ProcedenciaAutoridad string
	VigenteDesde         time.Time
	VigenteHasta         time.Time
}

func (a AcreditacionInstitucionalRegistroPropioV1) ValidarEn(ahora time.Time) error {
	if !referenciaRegistroPropio.MatchString(a.CredencialRef) ||
		!referenciaRegistroConPrefijo(a.SujetoRef, "suj_") ||
		!referenciaRegistroConPrefijo(a.DominioHMACRef, "idh_") || a.ClaveHMACID == "" || a.ClaveHMACVersion == 0 ||
		len(a.CuentaHuellaHMAC) != 32 || len(a.SujetoHuellaHMAC) != 32 ||
		subtle.ConstantTimeCompare(a.CuentaHuellaHMAC, a.SujetoHuellaHMAC) == 1 ||
		!referenciaRegistroPropio.MatchString(a.EquivalenciaRef) ||
		!referenciaRegistroConPrefijo(a.ProcedenciaRef, "prc_") ||
		a.ProcedenciaVersion == 0 || !huellaRegistroPropio.MatchString(a.ProcedenciaSHA256) ||
		a.ProcedenciaAutoridad != "autoridad_maestra_acreditada" || a.VigenteDesde.IsZero() || a.VigenteHasta.IsZero() ||
		!a.VigenteDesde.Before(a.VigenteHasta) || ahora.Before(a.VigenteDesde) || !ahora.Before(a.VigenteHasta) {
		return ErrRegistroPropioInvalido
	}
	return nil
}

// MismaAcreditacion exige revalidacion de la misma identidad y procedencia.
// Las referencias son opacas; nunca se usan nombres, DNI o correo como clave.
func (a AcreditacionInstitucionalRegistroPropioV1) MismaAcreditacion(otra AcreditacionInstitucionalRegistroPropioV1) bool {
	x := []byte(a.CredencialRef + "\x00" + a.SujetoRef + "\x00" + a.DominioHMACRef + "\x00" + a.ClaveHMACID + "\x00" + a.EquivalenciaRef + "\x00" + a.ProcedenciaRef + "\x00" + a.ProcedenciaSHA256)
	y := []byte(otra.CredencialRef + "\x00" + otra.SujetoRef + "\x00" + otra.DominioHMACRef + "\x00" + otra.ClaveHMACID + "\x00" + otra.EquivalenciaRef + "\x00" + otra.ProcedenciaRef + "\x00" + otra.ProcedenciaSHA256)
	return subtle.ConstantTimeCompare(x, y) == 1 && a.ProcedenciaVersion == otra.ProcedenciaVersion &&
		a.ProcedenciaAutoridad == otra.ProcedenciaAutoridad && a.ClaveHMACVersion == otra.ClaveHMACVersion &&
		subtle.ConstantTimeCompare(a.CuentaHuellaHMAC, otra.CuentaHuellaHMAC) == 1 &&
		subtle.ConstantTimeCompare(a.SujetoHuellaHMAC, otra.SujetoHuellaHMAC) == 1 &&
		a.VigenteDesde.Equal(otra.VigenteDesde) && a.VigenteHasta.Equal(otra.VigenteHasta)
}

type ReciboRegistroPropioV1 struct {
	OperacionRef       string
	ReciboRef          string
	Estado             string
	CuentaRef          string
	CuentaVersion      uint64
	PersonaRef         string
	PersonaVersion     uint64
	PerfilRef          string
	PerfilVersion      uint64
	VinculoRef         string
	VinculoVersion     uint64
	ProcedenciaRef     string
	ProcedenciaVersion uint64
	ProcedenciaSHA256  string
}

func (r ReciboRegistroPropioV1) ValidarPendiente() error {
	if !referenciaRegistroConPrefijo(r.OperacionRef, "opr_") || !referenciaRegistroConPrefijo(r.ReciboRef, "rpr_") ||
		r.Estado != EstadoRegistroPropioPendienteContacto || !referenciaRegistroConPrefijo(r.CuentaRef, "cta_") ||
		r.CuentaVersion == 0 || !referenciaRegistroConPrefijo(r.PersonaRef, "per_") || r.PersonaVersion == 0 ||
		!referenciaRegistroConPrefijo(r.PerfilRef, "prf_") || r.PerfilVersion == 0 ||
		!referenciaRegistroConPrefijo(r.VinculoRef, "vca_") || r.VinculoVersion == 0 ||
		!referenciaRegistroConPrefijo(r.ProcedenciaRef, "prc_") || r.ProcedenciaVersion == 0 ||
		!huellaRegistroPropio.MatchString(r.ProcedenciaSHA256) {
		return ErrRegistroPropioInvalido
	}
	return nil
}
