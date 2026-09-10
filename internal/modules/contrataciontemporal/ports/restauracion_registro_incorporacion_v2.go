package ports

import (
	"bytes"
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	core "vec-diputacion-granada/internal/vec/domain"
)

const EsquemaEvidenciaOrdenOriginalIncorporacionV2 = "vec.contratacion-temporal.incorporacion.ejercicio.orden-original.v2"

var ErrEvidenciaOrdenOriginalIncorporacionV2 = errors.New("contratacion temporal: evidencia historica de incorporacion invalida")

// EvidenciaOrdenOriginalIncorporacionV2 es transporte para el almacén propietario,
// NO una orden, confirmación, permiso ni prueba de commit. No debe registrarse en
// logs. La restauración nominal exige lectores históricos propietarios separados.
// La solicitud se reconstruye del material/recurso completo y se coteja por hash.
// El snapshot RBAC y las autoridades históricas no se fabrican desde este DTO.
type EvidenciaOrdenOriginalIncorporacionV2 struct {
	Esquema                                                                    string
	PreparadoEn, EvaluadaEn, ConcesionRegistradaEn                             time.Time
	MaterialCanonico                                                           []byte
	MaterialSHA256, IntencionSHA256, SolicitudSHA256, ContextoOriginalRef      string
	CapacidadCanonica, DecisionCanonica, MotivoCanonico, ContextoActorCanonico []byte
	PersonaVersion, PerfilVersion                                              uint64
	PayloadVECAD3, SobreCOSESign1, EvidenciaVerificacion, RaizPublicaSPKI      []byte
	HuellaExportacionSHA256                                                    string
}

func (e EvidenciaOrdenOriginalIncorporacionV2) Copia() EvidenciaOrdenOriginalIncorporacionV2 {
	e.MaterialCanonico = bytes.Clone(e.MaterialCanonico)
	e.CapacidadCanonica = bytes.Clone(e.CapacidadCanonica)
	e.DecisionCanonica = bytes.Clone(e.DecisionCanonica)
	e.MotivoCanonico = bytes.Clone(e.MotivoCanonico)
	e.ContextoActorCanonico = bytes.Clone(e.ContextoActorCanonico)
	e.PayloadVECAD3 = bytes.Clone(e.PayloadVECAD3)
	e.SobreCOSESign1 = bytes.Clone(e.SobreCOSESign1)
	e.EvidenciaVerificacion = bytes.Clone(e.EvidenciaVerificacion)
	e.RaizPublicaSPKI = bytes.Clone(e.RaizPublicaSPKI)
	return e
}

// CapturarEvidenciaOrdenOriginalIncorporacionV2 conserva las fechas privadas
// originales sin añadir setters a Orden/Material ni emitir otra autorización.
// Debe capturarse junto al recibo/consumos del mismo commit; capturar no persiste.
func CapturarEvidenciaOrdenOriginalIncorporacionV2(ctx context.Context, o OrdenConfirmacionIncorporacionV2) (EvidenciaOrdenOriginalIncorporacionV2, error) {
	cero := EvidenciaOrdenOriginalIncorporacionV2{}
	if ctx == nil {
		return cero, ErrEvidenciaOrdenOriginalIncorporacionV2
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if !domain.InstanteUTCCanonico(o.evaluadaEn) || !domain.InstanteUTCCanonico(o.material.preparadoEn) || o.ValidarEn(o.evaluadaEn) != nil {
		return cero, ErrEvidenciaOrdenOriginalIncorporacionV2
	}
	d, err := o.material.Datos()
	if err != nil {
		return cero, ErrEvidenciaOrdenOriginalIncorporacionV2
	}
	c, err := o.autorizacion.Confirmacion.Datos()
	if err != nil {
		return cero, ErrEvidenciaOrdenOriginalIncorporacionV2
	}
	canon, err := o.material.MaterialCanonico()
	if err != nil {
		return cero, ErrEvidenciaOrdenOriginalIncorporacionV2
	}
	intencion, err := IntencionRegistroIncorporacionV2(o.material)
	if err != nil {
		return cero, ErrEvidenciaOrdenOriginalIncorporacionV2
	}
	solicitud, err := core.HuellaSHA256SolicitudAutorizacionV3(o.autorizacion.Solicitud)
	if err != nil {
		return cero, ErrEvidenciaOrdenOriginalIncorporacionV2
	}
	x := o.Exportacion()
	h, err := x.HuellaConjuntoSHA256()
	if err != nil {
		return cero, ErrEvidenciaOrdenOriginalIncorporacionV2
	}
	e := EvidenciaOrdenOriginalIncorporacionV2{
		Esquema: EsquemaEvidenciaOrdenOriginalIncorporacionV2, PreparadoEn: o.material.preparadoEn, EvaluadaEn: o.evaluadaEn, ConcesionRegistradaEn: c.RegistradaEn,
		MaterialCanonico: canon, MaterialSHA256: hashRegistroV2(canon), IntencionSHA256: hashRegistroV2(intencion), SolicitudSHA256: solicitud, ContextoOriginalRef: d.Contexto.Resultado.RegistroContextoRef,
		CapacidadCanonica: x.CapacidadCanonica(), DecisionCanonica: x.DecisionCanonica(), MotivoCanonico: x.MotivoCanonico(), ContextoActorCanonico: x.ContextoActorCanonico(),
		PersonaVersion: x.PersonaVersion(), PerfilVersion: x.PerfilVersion(), PayloadVECAD3: x.PayloadVECAD3(), SobreCOSESign1: x.SobreCOSESign1(), EvidenciaVerificacion: x.EvidenciaVerificacion(), RaizPublicaSPKI: x.RaizPublicaSPKI(), HuellaExportacionSHA256: h,
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	return e, nil
}

// CotejarOrden compara contra una orden YA nominal, nunca construye autoridad
// desde el transporte. Útil tras la futura lectura/restauración propietaria.
func (e EvidenciaOrdenOriginalIncorporacionV2) CotejarOrden(ctx context.Context, o OrdenConfirmacionIncorporacionV2) error {
	original, err := CapturarEvidenciaOrdenOriginalIncorporacionV2(ctx, o)
	if err != nil {
		return err
	}
	if e.Esquema != original.Esquema || e.PreparadoEn != original.PreparadoEn || e.EvaluadaEn != original.EvaluadaEn || e.ConcesionRegistradaEn != original.ConcesionRegistradaEn ||
		e.MaterialSHA256 != original.MaterialSHA256 || e.IntencionSHA256 != original.IntencionSHA256 || e.SolicitudSHA256 != original.SolicitudSHA256 || e.ContextoOriginalRef != original.ContextoOriginalRef ||
		e.PersonaVersion != original.PersonaVersion || e.PerfilVersion != original.PerfilVersion || e.HuellaExportacionSHA256 != original.HuellaExportacionSHA256 ||
		!bytes.Equal(e.MaterialCanonico, original.MaterialCanonico) || !bytes.Equal(e.CapacidadCanonica, original.CapacidadCanonica) || !bytes.Equal(e.DecisionCanonica, original.DecisionCanonica) ||
		!bytes.Equal(e.MotivoCanonico, original.MotivoCanonico) || !bytes.Equal(e.ContextoActorCanonico, original.ContextoActorCanonico) || !bytes.Equal(e.PayloadVECAD3, original.PayloadVECAD3) ||
		!bytes.Equal(e.SobreCOSESign1, original.SobreCOSESign1) || !bytes.Equal(e.EvidenciaVerificacion, original.EvidenciaVerificacion) || !bytes.Equal(e.RaizPublicaSPKI, original.RaizPublicaSPKI) {
		return ErrEvidenciaOrdenOriginalIncorporacionV2
	}
	return ctx.Err()
}
