package httpseguridad

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

type registroMemoria struct {
	mu               sync.Mutex
	asercionesUsadas map[string]struct{}
	sesiones         map[string]ConfirmacionAltaSesion
	cuentas          map[string]string
	revocadas        map[string]struct{}
	cuentasInactivas map[string]struct{}
	contador         uint64
	ultimaConsulta   ConsultaSesionActiva
}

func nuevoRegistroMemoria() *registroMemoria {
	return &registroMemoria{asercionesUsadas: make(map[string]struct{}),
		sesiones: make(map[string]ConfirmacionAltaSesion), cuentas: make(map[string]string),
		revocadas: make(map[string]struct{}), cuentasInactivas: make(map[string]struct{})}
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
	if _, existe := r.sesiones[alta.SesionID]; existe {
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
		CuentaRef: cuentaRef, CuentaOrdinariaRef: cuentaOrdinariaRef, AltaConfirmada: alta,
	}
	r.asercionesUsadas[alta.AsercionID] = struct{}{}
	r.sesiones[alta.SesionID] = confirmacion
	return confirmacion, nil
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
	if _, revocada := r.revocadas[consulta.SesionID]; revocada {
		return errors.New("sesion revocada")
	}
	if _, inactiva := r.cuentasInactivas[consulta.CuentaID]; inactiva {
		return errors.New("cuenta inactiva")
	}
	if _, inactiva := r.cuentasInactivas[consulta.CuentaOrdinariaID]; consulta.CuentaOrdinariaID != "" && inactiva {
		return errors.New("cuenta ordinaria inactiva")
	}
	confirmacion, existe := r.sesiones[consulta.SesionID]
	alta := confirmacion.AltaConfirmada
	if !existe || confirmacion.AutenticacionRef != consulta.AutenticacionRef ||
		confirmacion.AsercionRef != consulta.AsercionRef || confirmacion.SesionRef != consulta.SesionRef ||
		confirmacion.ControlSesionRef != consulta.ControlSesionRef || confirmacion.CuentaRef != consulta.CuentaRef ||
		confirmacion.CuentaOrdinariaRef != consulta.CuentaOrdinariaRef || alta.AsercionID != consulta.AsercionID ||
		alta.SesionID != consulta.SesionID || alta.SujetoID != consulta.SujetoID || alta.CuentaID != consulta.CuentaID ||
		alta.CuentaOrdinariaID != consulta.CuentaOrdinariaID || alta.CuentaPrivilegiada != consulta.CuentaPrivilegiada ||
		alta.Superficie != consulta.Superficie || !alta.EmitidaEn.Equal(consulta.EmitidaEn) ||
		!alta.ExpiraEn.Equal(consulta.ExpiraEn) || alta.PoliticaRef != consulta.PoliticaRef ||
		alta.HuellaPolitica != consulta.HuellaPolitica {
		return errors.New("sesion distinta")
	}
	return nil
}

func (r *registroMemoria) revocar(sesionID string) {
	r.mu.Lock()
	r.revocadas[sesionID] = struct{}{}
	r.mu.Unlock()
}

func (r *registroMemoria) inactivar(cuentaID string) {
	r.mu.Lock()
	r.cuentasInactivas[cuentaID] = struct{}{}
	r.mu.Unlock()
}
