// Package contrataciontemporal implementa la frontera anticorrupción de alta
// en Personal para el ejercicio sintético. No compone autoridades ni almacenes.
package contrataciontemporal

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/modules/personal/adapters/fuenteejercicio"
	core "vec-diputacion-granada/internal/vec/domain"
)

const (
	AccionAltaEjercicio      = "personal.alta_ejercicio.registrar"
	FinalidadAltaEjercicio   = "registrar_relacion_ocupacion_sinteticas"
	TipoRecursoAltaEjercicio = "alta_personal_ejercicio"
	AudienciaAltaEjercicio   = "vec_personal.alta_ejercicio.v1"
	EsquemaMaterialAlta      = "vec.personal.alta-ejercicio.material.v1"
)

var (
	ErrNoDisponible      = errors.New("personal: alta de ejercicio no disponible")
	ErrSolicitudInvalida = errors.New("personal: solicitud de alta de ejercicio invalida")
	ErrDenegado          = errors.New("personal: alta de ejercicio denegada")
	ErrConflicto         = errors.New("personal: alta de ejercicio en conflicto")
	ErrReciboNoConfiable = errors.New("personal: recibo de alta de ejercicio no confiable")
)

// PreparacionAlta contiene únicamente datos exactos de la fuente, no autoridad.
type PreparacionAlta struct {
	Solicitud ctports.SolicitudAltaPersonalRPT
	Fuente    fuenteejercicio.TernaEsperada
	Vinculo   fuenteejercicio.VinculoEjercicio
}

// MaterialAlta es inmutable por valor. El actor RRHH no es la persona del alta.
// Se conserva completo en el registro propio de Personal y en su idempotencia.
type MaterialAlta struct {
	Preparacion     PreparacionAlta
	OrganizacionRef string
	ActorRef        string
	PerfilRef       string
}

func (m MaterialAlta) Validar() error {
	p, v := m.Preparacion, m.Preparacion.Vinculo
	f := ctports.ReferenciaVersionadaPersonalRPT{Referencia: p.Fuente.Referencia, Version: p.Fuente.Version, HuellaSHA256: p.Fuente.HuellaSHA256}
	desde, e1 := time.Parse("2006-01-02", v.Desde)
	hasta, e2 := time.Parse("2006-01-02", v.Hasta)
	if p.Solicitud.Validar() != nil || f.Validar() != nil ||
		!ctdomain.ReferenciaOpacaValida(m.OrganizacionRef) || !ctdomain.ReferenciaOpacaValida(m.ActorRef) || !ctdomain.ReferenciaOpacaValida(m.PerfilRef) ||
		!ctdomain.ReferenciaOpacaValida(v.PersonaSinteticaRef) || !strings.HasPrefix(v.PersonaSinteticaRef, "persona:ejercicio:") || v.PersonaSinteticaRef == m.ActorRef ||
		!ctdomain.ReferenciaOpacaValida(v.CentroRef) || v.FuenteRPT != p.Solicitud.FuenteRPT || v.PuestoRef != p.Solicitud.PuestoRef || v.PlazaRef != p.Solicitud.PlazaRef ||
		e1 != nil || e2 != nil || desde.Format("2006-01-02") != v.Desde || hasta.Format("2006-01-02") != v.Hasta || hasta.Before(desde) ||
		(v.AntecedenteEjercicioRef != "" && !ctdomain.ReferenciaOpacaValida(v.AntecedenteEjercicioRef)) {
		return ErrSolicitudInvalida
	}
	return nil
}

func (m MaterialAlta) HuellaSHA256() (string, error) {
	if m.Validar() != nil {
		return "", ErrSolicitudInvalida
	}
	b, err := json.Marshal(struct {
		Esquema  string
		Material MaterialAlta
	}{EsquemaMaterialAlta, m})
	if err != nil {
		return "", ErrSolicitudInvalida
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}

// RecursoAltaEjercicio define ligaduras, nunca permisos ni concesiones.
func RecursoAltaEjercicio(m MaterialAlta) (core.RecursoAutorizable, error) {
	h, err := m.HuellaSHA256()
	if err != nil {
		return core.RecursoAutorizable{}, err
	}
	p, v := m.Preparacion, m.Preparacion.Vinculo
	hs, err := p.Solicitud.HuellaSHA256()
	if err != nil {
		return core.RecursoAutorizable{}, ErrSolicitudInvalida
	}
	r := core.RecursoAutorizable{Referencia: p.Solicitud.SolicitudRef, ModuloID: "personal", Tipo: TipoRecursoAltaEjercicio,
		Ambitos: map[string]string{"organizacion_ref": m.OrganizacionRef, "centro_ref": v.CentroRef},
		Atributos: map[string]string{"material_sha256": h, "solicitud_sha256": hs, "capacidad_ref": p.Solicitud.CapacidadRef,
			"actor_ref": m.ActorRef, "perfil_ref": m.PerfilRef, "expediente_ref": p.Solicitud.ExpedienteRef, "version_expediente": strconv.FormatUint(p.Solicitud.VersionExpediente, 10),
			"fuente_ref": p.Fuente.Referencia, "fuente_version": strconv.FormatUint(p.Fuente.Version, 10), "fuente_sha256": p.Fuente.HuellaSHA256,
			"persona_sintetica_ref": v.PersonaSinteticaRef, "desde": v.Desde, "hasta": v.Hasta,
			"rpt_ref": v.FuenteRPT.Referencia, "rpt_version": strconv.FormatUint(v.FuenteRPT.Version, 10), "rpt_sha256": v.FuenteRPT.HuellaSHA256,
			"puesto_ref": v.PuestoRef, "plaza_ref": v.PlazaRef, "tipo_validacion": "ejercicio_sintetico"}}
	if _, err := r.HuellaContextoAutorizacionSHA256(); err != nil {
		return core.RecursoAutorizable{}, ErrSolicitudInvalida
	}
	return r, nil
}

// ReciboAlta procede del commit verificado de Personal. En replay todos estos
// campos permanecen originales; no se reemplazan fecha, decisión ni referencias.
type ReciboAlta struct {
	Material               MaterialAlta
	Resultado              ctports.ResultadoAltaPersonalRPT
	RegistradoEn           time.Time
	DecisionOriginalRef    string
	AuditoriaRef           string
	OutboxRef              string
	EjercicioSintetico     bool
	FirmaOficial           bool
	EficaciaAdministrativa bool
}

type ResultadoTransaccionAlta struct {
	Recibo               ReciboAlta
	Replay               bool
	DecisionConsumidaRef string
}

func (r ResultadoTransaccionAlta) ValidarPara(o OrdenAlta, ahora time.Time) error {
	v := r.Recibo
	if o.ValidarEn(ahora) != nil || v.Material != o.material || v.Resultado.ValidarPara(o.material.Preparacion.Solicitud) != nil ||
		v.Resultado.Estado != ctports.AltaPersonalRPTConfirmada || !ctdomain.InstanteUTCCanonico(v.RegistradoEn) || v.RegistradoEn.After(ahora) ||
		!ctdomain.ReferenciaOpacaValida(v.DecisionOriginalRef) || !ctdomain.ReferenciaOpacaValida(v.AuditoriaRef) || !ctdomain.ReferenciaOpacaValida(v.OutboxRef) ||
		!v.EjercicioSintetico || v.FirmaOficial || v.EficaciaAdministrativa ||
		r.DecisionConsumidaRef != o.autorizacion.Exportacion.ResumenCapacidad().DecisionRef() {
		return ErrReciboNoConfiable
	}
	if r.Replay {
		if v.RegistradoEn.After(o.preparadaEn) || v.DecisionOriginalRef == r.DecisionConsumidaRef {
			return ErrReciboNoConfiable
		}
	} else if v.RegistradoEn.Before(o.preparadaEn) || v.DecisionOriginalRef != r.DecisionConsumidaRef {
		return ErrReciboNoConfiable
	}
	return nil
}
