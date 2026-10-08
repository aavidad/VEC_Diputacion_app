// Package auditoriafirma registra fallos posperfil en la auditoría común.
package auditoriafirma

import (
	"context"
	"errors"
	"reflect"

	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// FabricaOrden obtiene contexto registrado, vínculo y correlación desde la
// frontera confiable de la petición. No recibe identidades del material CT.
// El motivo, proceso, canal y finalidad proceden del catálogo/configuración.
type FabricaOrden interface {
	CrearOrdenIntentoFirma(context.Context, string, string, string, domain.ResultadoIntentoAuditoria) (ports.OrdenIntentoAuditoria, error)
}

type Registro struct {
	registro ct.RegistroFirmasVerificadasV2
	intentos ports.RegistradorIntentosAuditoria
	fabrica  FabricaOrden
}

var _ ct.RegistroFirmasVerificadasV2 = (*Registro)(nil)

func Nuevo(registro ct.RegistroFirmasVerificadasV2, intentos ports.RegistradorIntentosAuditoria, fabrica FabricaOrden) (*Registro, error) {
	if nulo(registro) || nulo(intentos) || nulo(fabrica) {
		return nil, ct.ErrRegistroFirmaDocumentoNoDisponible
	}
	return &Registro{registro, intentos, fabrica}, nil
}

// El repositorio original retorna sólo después de cerrar su transacción.
// Los permisos positivos siguen auditándose dentro de esa misma transacción.
func (r *Registro) RegistrarFirmaVerificadaV2(ctx context.Context, m ct.MaterialFirmaVerificadaV2, c ct.CapacidadFirmaVerificadaV2) (ct.ReciboFirmaDocumento, error) {
	if r == nil || ctx == nil {
		return ct.ReciboFirmaDocumento{}, ct.ErrRegistroFirmaDocumentoNoDisponible
	}
	intento, err := ports.NuevaReferenciaIntentoAuditoria()
	if err != nil {
		return ct.ReciboFirmaDocumento{}, ct.ErrRegistroFirmaDocumentoNoDisponible
	}
	recibo, err := r.registro.RegistrarFirmaVerificadaV2(ctx, m, c)
	if err == nil {
		return recibo, nil
	}
	accion := ct.AccionRegistrarFirmaExterna
	if m.Via == ct.ViaFirmaCertificadoVEC {
		accion = ct.AccionRegistrarFirmaVec
	}
	return ct.ReciboFirmaDocumento{}, r.auditar(ctx, intento, accion, m.RecursoRef(), err)
}

func (r *Registro) ConsultarFirmasAutorizadasV2(ctx context.Context, m ct.MaterialConsultaFirmasR5V2, c ct.CapacidadConsultaFirmasR5V2) (ct.LecturaFirmasR5V2, error) {
	if r == nil || ctx == nil {
		return ct.LecturaFirmasR5V2{}, ct.ErrRegistroFirmaDocumentoNoDisponible
	}
	intento, err := ports.NuevaReferenciaIntentoAuditoria()
	if err != nil {
		return ct.LecturaFirmasR5V2{}, ct.ErrRegistroFirmaDocumentoNoDisponible
	}
	lectura, err := r.registro.ConsultarFirmasAutorizadasV2(ctx, m, c)
	if err == nil {
		return lectura, nil
	}
	if lecturaAuditadaSinResultado(err) {
		return ct.LecturaFirmasR5V2{}, err
	}
	return ct.LecturaFirmasR5V2{}, r.auditar(ctx, intento, ct.AccionConsultarFirmasR5V2, m.ExpedienteRef, err)
}

// lecturaAuditadaSinResultado reconoce el «no encontrado» de una lectura
// autorizada. El lector sólo lo devuelve después de confirmar la transacción
// que consumió la decisión y escribió su auditoría de consumo: esa fila ya
// registra el acceso. Otro intento «error» lo duplicaría y contaría como fallo
// técnico lo que es una respuesta 404.
func lecturaAuditadaSinResultado(err error) bool {
	return errors.Is(err, ct.ErrExpedienteConsultaFirmasNoEncontrado)
}

func (r *Registro) auditar(ctx context.Context, intento, accion, recurso string, fallo error) error {
	resultado := domain.ResultadoIntentoAuditoriaError
	if errors.Is(fallo, ct.ErrFirmaDocumentoDenegada) {
		resultado = domain.ResultadoIntentoAuditoriaDenegado
	}
	orden, err := r.fabrica.CrearOrdenIntentoFirma(ctx, intento, accion, recurso, resultado)
	if err != nil {
		return ct.ErrRegistroFirmaDocumentoNoDisponible
	}
	d, err := orden.Datos()
	if err != nil || d.IntentoRef != intento || d.Datos.Accion != accion || d.Datos.ModuloID != ct.ModuloContratacion ||
		d.Datos.RecursoRef != recurso || d.Datos.Resultado != resultado {
		return ct.ErrRegistroFirmaDocumentoNoDisponible
	}
	acuse, err := r.intentos.AppendIntentoAuditoria(ctx, orden)
	if err != nil || acuse.ValidarPara(orden) != nil {
		return ct.ErrRegistroFirmaDocumentoNoDisponible
	}
	return fallo
}

func nulo(v any) bool {
	if v == nil {
		return true
	}
	x := reflect.ValueOf(v)
	switch x.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Func, reflect.Map, reflect.Slice, reflect.Chan:
		return x.IsNil()
	}
	return false
}
