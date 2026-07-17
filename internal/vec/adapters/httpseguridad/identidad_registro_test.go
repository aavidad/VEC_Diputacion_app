package httpseguridad

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

type registroMemoria struct {
	mu               sync.Mutex
	asercionesUsadas map[string]struct{}
	sesiones         map[string]ConfirmacionAltaSesion
	sesionesID       map[string]struct{}
	cuentas          map[string]string
	revocadas        map[string]struct{}
	cuentasInactivas map[string]struct{}
	contador         uint64
	ultimaConsulta   ConsultaSesionActiva
}

func nuevoRegistroMemoria() *registroMemoria {
	return &registroMemoria{asercionesUsadas: make(map[string]struct{}),
		sesiones: make(map[string]ConfirmacionAltaSesion), cuentas: make(map[string]string),
		sesionesID: make(map[string]struct{}),
		revocadas:  make(map[string]struct{}), cuentasInactivas: make(map[string]struct{})}
}

func (r *registroMemoria) ConsumirAsercionYRegistrar(_ context.Context, alta AltaSesionAtomica) (ConfirmacionAltaSesion, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, usada := r.asercionesUsadas[alta.AsercionID]; usada {
		return ConfirmacionAltaSesion{}, errors.New("reproduccion")
	}
	if _, inactiva := r.cuentasInactivas[alta.CuentaID]; inactiva {
		return ConfirmacionAltaSesion{}, errors.New("cuenta inactiva")
	}
	if _, inactiva := r.cuentasInactivas[alta.CuentaOrdinariaID]; alta.CuentaOrdinariaID != "" && inactiva {
		return ConfirmacionAltaSesion{}, errors.New("cuenta ordinaria inactiva")
	}
	if _, existe := r.sesionesID[alta.SesionID]; existe {
		return ConfirmacionAltaSesion{}, errors.New("sesion duplicada")
	}
	cuentaRef := r.referenciaCuenta(alta.CuentaID)
	cuentaOrdinariaRef := cuentaRef
	if alta.CuentaPrivilegiada {
		cuentaOrdinariaRef = r.referenciaCuenta(alta.CuentaOrdinariaID)
	}
	confirmacion := ConfirmacionAltaSesion{
		AutenticacionRef: r.emitirReferencia("aut_"), AsercionRef: r.emitirReferencia("ase_"),
		SesionRef: r.emitirReferencia("ses_"), ControlSesionRef: r.emitirReferencia("cse_"),
		ControlSesionRevision: 1, ControlSesionEstado: EstadoControlSesionActiva,
		CuentaRef: cuentaRef, CuentaOrdinariaRef: cuentaOrdinariaRef,
		SesionRevalidadaEn: alta.SesionEmitidaEn, SesionValidaHasta: alta.AsercionExpiraEn,
		AltaConfirmada: alta,
	}
	confirmacion.ControlSesionHuellaSHA256 = huellaControlSesionPrueba(confirmacion)
	r.asercionesUsadas[alta.AsercionID] = struct{}{}
	r.sesionesID[alta.SesionID] = struct{}{}
	r.sesiones[confirmacion.SesionRef] = confirmacion
	return confirmacion, nil
}

func huellaControlSesionPrueba(confirmacion ConfirmacionAltaSesion) string {
	suma := sha256.Sum256([]byte(strings.Join([]string{
		confirmacion.ControlSesionRef, strconv.FormatUint(confirmacion.ControlSesionRevision, 10),
		confirmacion.SesionRef, string(confirmacion.ControlSesionEstado),
		confirmacion.SesionRevalidadaEn.Format(time.RFC3339Nano),
		confirmacion.SesionValidaHasta.Format(time.RFC3339Nano),
	}, "\x00")))
	return hex.EncodeToString(suma[:])
}

func (r *registroMemoria) emitirReferencia(prefijo string) string {
	r.contador++
	return fmt.Sprintf("%s%022d", prefijo, r.contador)
}

func (r *registroMemoria) referenciaCuenta(id string) string {
	if referencia := r.cuentas[id]; referencia != "" {
		return referencia
	}
	referencia := r.emitirReferencia("cta_")
	r.cuentas[id] = referencia
	return referencia
}

func (r *registroMemoria) ComprobarSesionYCuentaActivas(_ context.Context, consulta ConsultaSesionActiva) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ultimaConsulta = consulta
	if _, revocada := r.revocadas[consulta.SesionRef]; revocada {
		return errors.New("sesion revocada")
	}
	confirmacion, existe := r.sesiones[consulta.SesionRef]
	alta := confirmacion.AltaConfirmada
	if _, inactiva := r.cuentasInactivas[alta.CuentaID]; inactiva {
		return errors.New("cuenta inactiva")
	}
	if _, inactiva := r.cuentasInactivas[alta.CuentaOrdinariaID]; alta.CuentaOrdinariaID != "" && inactiva {
		return errors.New("cuenta ordinaria inactiva")
	}
	if !existe || confirmacion.AutenticacionRef != consulta.AutenticacionRef ||
		alta.AutenticacionHuellaSHA256 != consulta.AutenticacionHuellaSHA256 ||
		confirmacion.AsercionRef != consulta.AsercionRef || confirmacion.CuentaRef != consulta.CuentaRef ||
		confirmacion.CuentaOrdinariaRef != consulta.CuentaOrdinariaRef ||
		alta.CuentaPrivilegiada != consulta.CuentaPrivilegiada || alta.Superficie != consulta.Superficie ||
		alta.MetodoObservado != consulta.MetodoObservado || alta.GarantiaObservada != consulta.GarantiaObservada ||
		alta.PoliticaGarantiaRef != consulta.PoliticaGarantiaRef ||
		alta.PoliticaGarantiaHuellaSHA256 != consulta.PoliticaGarantiaHuellaSHA256 ||
		!alta.AutenticacionVerificadaEn.Equal(consulta.AutenticacionVerificadaEn) ||
		!alta.SesionEmitidaEn.Equal(consulta.SesionEmitidaEn) ||
		confirmacion.ControlSesionRef != consulta.ControlSesionRef ||
		confirmacion.ControlSesionRevision != consulta.ControlSesionRevision ||
		confirmacion.ControlSesionEstado != consulta.ControlSesionEstado ||
		confirmacion.ControlSesionHuellaSHA256 != consulta.ControlSesionHuellaSHA256 ||
		!confirmacion.SesionRevalidadaEn.Equal(consulta.SesionRevalidadaEn) ||
		!confirmacion.SesionValidaHasta.Equal(consulta.SesionValidaHasta) {
		return errors.New("sesion distinta")
	}
	return nil
}

func (r *registroMemoria) revocar(sesionID string) {
	r.mu.Lock()
	for sesionRef, confirmacion := range r.sesiones {
		if confirmacion.AltaConfirmada.SesionID == sesionID {
			r.revocadas[sesionRef] = struct{}{}
		}
	}
	r.mu.Unlock()
}

func (r *registroMemoria) inactivar(cuentaID string) {
	r.mu.Lock()
	r.cuentasInactivas[cuentaID] = struct{}{}
	r.mu.Unlock()
}
