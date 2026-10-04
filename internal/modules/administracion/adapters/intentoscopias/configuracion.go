// Package intentoscopias connects failed ADMIN attempts to the common audit authority.
package intentoscopias

import (
	"reflect"
	"time"

	http "vec-diputacion-granada/internal/modules/administracion/adapters/httpcopias"
	p "vec-diputacion-granada/internal/modules/administracion/ports/httpcopias"
	d "vec-diputacion-granada/internal/vec/domain"
	v "vec-diputacion-granada/internal/vec/ports"
)

// Operacion contains server-governed references, never request attributes.
type Operacion struct {
	Accion, FinalidadRef, RecursoFallback string
}

type Configuracion struct {
	Motivos     map[string]d.ReferenciaEntradaCatalogo
	Operaciones map[p.Operacion]Operacion
	// FronteraNominal records unknown routes/methods after nominal resolution.
	FronteraNominal Operacion
	Proceso         string
	Canal           string
	Plazo           time.Duration
}

var acciones = [...]p.Operacion{p.Consultar, p.Lanzar, p.ConfigurarCalendario, p.ConfigurarRetencion, p.Proponer, p.Revisar, p.Ejecutar}
var codigos = [...]string{"autenticacion_requerida", "acceso_denegado", "solicitud_invalida", "conflicto_estado", "recurso_no_encontrado", "servicio_no_disponible", "metodo_no_permitido"}

type Auditor struct {
	config    Configuracion
	fuente    FuenteIntento
	registro  v.RegistradorIntentosAuditoria
	noNominal http.AuditorFrontera
}

// Nuevo requires a separately configured non-nominal boundary for pre-session
// failures. It supplies no default process, catalogue, identity or permission.
func Nuevo(c Configuracion, f FuenteIntento, r v.RegistradorIntentosAuditoria, noNominal http.AuditorFrontera) (*Auditor, error) {
	if ausente(f) || ausente(r) || noNominal == nil || c.Plazo <= 0 || c.Plazo > 30*time.Second || c.Canal != string(d.SuperficieAutenticacionAdministracionPrivilegiadaV1) || len(c.Motivos) != len(codigos) || len(c.Operaciones) != len(acciones) {
		return nil, p.ErrNoDisponible
	}
	copyConfig := Configuracion{Proceso: c.Proceso, Canal: c.Canal, Plazo: c.Plazo, FronteraNominal: c.FronteraNominal, Motivos: make(map[string]d.ReferenciaEntradaCatalogo, len(codigos)), Operaciones: make(map[p.Operacion]Operacion, len(acciones))}
	for _, codigo := range codigos {
		motivo, existe := c.Motivos[codigo]
		if !existe {
			return nil, p.ErrNoDisponible
		}
		copyConfig.Motivos[codigo] = motivo
	}
	for _, accion := range acciones {
		op, existe := c.Operaciones[accion]
		if !existe {
			return nil, p.ErrNoDisponible
		}
		if !operacionValida(op, copyConfig) {
			return nil, p.ErrNoDisponible
		}
		copyConfig.Operaciones[accion] = op
	}
	if !operacionValida(c.FronteraNominal, copyConfig) {
		return nil, p.ErrNoDisponible
	}
	return &Auditor{config: copyConfig, fuente: f, registro: r, noNominal: noNominal}, nil
}

func operacionValida(op Operacion, c Configuracion) bool {
	// Only structural configuration validation. This sentinel is never emitted.
	for _, motivo := range c.Motivos {
		datos := d.DatosIntentoAuditoria{Accion: op.Accion, ModuloID: "administracion", RecursoRef: op.RecursoFallback, FinalidadRef: op.FinalidadRef, Resultado: d.ResultadoIntentoAuditoriaError, Motivo: motivo, Proceso: c.Proceso, Canal: c.Canal, CorrelacionRef: "correlacion_00000000000000000000000000000000"}
		if datos.Validar() != nil {
			return false
		}
	}
	return true
}

func ausente(x any) bool {
	if x == nil {
		return true
	}
	r := reflect.ValueOf(x)
	switch r.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Func, reflect.Map, reflect.Slice, reflect.Chan:
		return r.IsNil()
	}
	return false
}
