package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"time"

	dom "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// PreparacionRaizIncorporacionV2 es un candidato técnico, NO persistencia ni
// autorización. Sólo CT75 puede insertarlo tras consumir sus permisos actuales.
// El envoltorio no se guarda como evidencia ni cambia el recibo/historia V2.
type PreparacionRaizIncorporacionV2 struct {
	seguimiento string
	estado      []byte
	version     uint64
}

type ResolverRaizInicialIncorporacionV2 interface {
	ResolverRaizInicialIncorporacionV2(context.Context, ct.OrdenConfirmacionIncorporacionV2) (PreparacionRaizIncorporacionV2, error)
}

func RaizIncorporacionV2Existente(ref string) (PreparacionRaizIncorporacionV2, error) {
	if !refSeguimientoRegistroV2(ref) {
		return PreparacionRaizIncorporacionV2{}, ct.ErrRegistroIncorporacionV2
	}
	return PreparacionRaizIncorporacionV2{seguimiento: ref}, nil
}

func NuevaPreparacionRaizIncorporacionV2(def dom.DefinicionSeguimiento, referencia string, periodo dom.IntervaloSeguimiento, orden ct.OrdenConfirmacionIncorporacionV2, ahora time.Time) (PreparacionRaizIncorporacionV2, error) {
	z := PreparacionRaizIncorporacionV2{}
	if orden.ValidarEn(ahora) != nil || !def.VigenteEn(ahora) {
		return z, ct.ErrRegistroIncorporacionV2
	}
	m, err := orden.Material().Datos()
	if err != nil || m.Confirmacion.VersionSeguimientoEsperada != 0 || m.Confirmacion.PeriodoIncorporacion != periodo || m.Personal.Resultado != m.Confirmacion.ResultadoPersonal || m.Personal.Solicitud != m.Confirmacion.SolicitudPersonal {
		return z, ct.ErrRegistroIncorporacionV2
	}
	// Ancla durable del estado cero: fecha del ORIGINAL Personal, no Now ni
	// fecha del reintento. Relación sólo de ese original acreditado por la orden.
	s, err := dom.NuevoSeguimiento(def, dom.AltaSeguimiento{Referencia: referencia, OrganizacionRef: m.Preparacion.OrganizacionRef, ExpedienteRef: m.Personal.Solicitud.ExpedienteRef, RelacionRef: m.Personal.Resultado.RelacionRef, PeriodoPrevisto: periodo, CreadoEn: m.Personal.RegistradoEn})
	if err != nil {
		return z, ct.ErrRegistroIncorporacionV2
	}
	snapshot, err := PrepararSnapshotSeguimientoPersistido(def, s)
	if err != nil || len(snapshot.EstadoJSON) > 128<<10 {
		return z, ct.ErrRegistroIncorporacionV2
	}
	return PreparacionRaizIncorporacionV2{seguimiento: referencia, estado: bytes.Clone(snapshot.EstadoJSON), version: m.VersionActualExpediente}, nil
}

func (r PreparacionRaizIncorporacionV2) envolverEvidencia(original []byte) ([]byte, error) {
	if !refSeguimientoRegistroV2(r.seguimiento) || len(original) == 0 {
		return nil, ct.ErrRegistroIncorporacionV2
	}
	if len(r.estado) == 0 {
		return bytes.Clone(original), nil
	}
	if r.version == 0 || len(r.estado) > 128<<10 {
		return nil, ct.ErrRegistroIncorporacionV2
	}
	b, err := json.Marshal(struct {
		Esquema string          `json:"esquema"`
		Orden   json.RawMessage `json:"orden"`
		Raiz    struct {
			Estado  json.RawMessage `json:"estado"`
			Version uint64          `json:"version_expediente"`
		} `json:"raiz_inicial"`
	}{Esquema: "vec.ct.incorporacion.raiz-inicial.v1", Orden: original, Raiz: struct {
		Estado  json.RawMessage `json:"estado"`
		Version uint64          `json:"version_expediente"`
	}{r.estado, r.version}})
	if err != nil || len(b) > MaximoBytesEvidenciaRegistroIncorporacionV2 {
		return nil, ct.ErrRegistroIncorporacionV2
	}
	return b, nil
}
