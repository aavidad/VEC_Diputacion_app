package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// AccionPreparacionLote nombra en la auditoría común los intentos fallidos de
// preparar. La decisión que se consume es la de la acción del lote; el
// registro distingue así preparar de aplicar.
const AccionPreparacionLote = "administracion.perfiles.preparar_lote_ordinario"

const prepararLoteOrdinarioSQL = `SELECT vec_autorizacion.preparar_lote_ordinario_admin_v1($1::text,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,$8::bytea,$9::bytea,$10::bytea,$11::bytea)`

// La preparación (AUT50) devuelve como mucho 64 altas y 64 bajas; el tope del
// documento deja margen para nombres largos sin admitir respuestas ajenas.
const maximoPreparacionLoteBytes = 128 * 1024

var _ ports.PreparadorLotesAdministracionPerfiles = (*AutoridadLoteOrdinario)(nil)

type altaPreparacionJSON struct {
	RolVersionRef      string    `json:"rol_version_ref"`
	Nombre             string    `json:"nombre"`
	UnidadRequerida    *bool     `json:"unidad_requerida"`
	VigenteHastaMaxima time.Time `json:"vigente_hasta_maxima"`
	DuracionSegundos   int64     `json:"duracion_propuesta_segundos"`
	PerfilRef          string    `json:"perfil_ref"`
	VinculoRef         string    `json:"vinculo_ref"`
	HuellaSHA256       string    `json:"huella_sha256"`
}

type bajaPreparacionJSON struct {
	RolVersionRef  string    `json:"rol_version_ref"`
	Nombre         *string   `json:"nombre"`
	PerfilRef      string    `json:"perfil_ref"`
	VinculoRef     string    `json:"vinculo_ref"`
	PerfilVersion  uint64    `json:"perfil_version"`
	VinculoVersion uint64    `json:"vinculo_version"`
	VigenteDesde   time.Time `json:"vigente_desde"`
	VigenteHasta   time.Time `json:"vigente_hasta"`
	HuellaSHA256   string    `json:"huella_sha256"`
}

type preparacionLoteJSON struct {
	OperacionRef            string                `json:"operacion_ref"`
	AuditoriaRef            string                `json:"auditoria_ref"`
	PreparadaEn             time.Time             `json:"preparada_en"`
	PersonaRef              string                `json:"persona_ref"`
	PersonaVersion          uint64                `json:"persona_version"`
	CuentaRef               string                `json:"cuenta_ref"`
	CuentaVersion           uint64                `json:"cuenta_version"`
	ProcedenciaRef          string                `json:"procedencia_ref"`
	ProcedenciaVersion      uint64                `json:"procedencia_version"`
	ProcedenciaHuellaSHA256 string                `json:"procedencia_huella_sha256"`
	OrganizacionRef         string                `json:"organizacion_ref"`
	UnidadRef               string                `json:"unidad_ref"`
	Altas                   []altaPreparacionJSON `json:"altas"`
	Bajas                   []bajaPreparacionJSON `json:"bajas"`
	Truncado                *bool                 `json:"truncado"`
}

func (x preparacionLoteJSON) dominio() (domain.PreparacionLoteAdministracionPerfiles, error) {
	if x.Altas == nil || x.Bajas == nil || x.Truncado == nil {
		return domain.PreparacionLoteAdministracionPerfiles{}, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	p := domain.PreparacionLoteAdministracionPerfiles{OperacionRef: x.OperacionRef, AuditoriaRef: x.AuditoriaRef,
		PreparadaEn: x.PreparadaEn, PersonaRef: x.PersonaRef, PersonaVersion: x.PersonaVersion,
		CuentaRef: x.CuentaRef, CuentaVersion: x.CuentaVersion, ProcedenciaRef: x.ProcedenciaRef,
		ProcedenciaVersion: x.ProcedenciaVersion, ProcedenciaHuellaSHA256: x.ProcedenciaHuellaSHA256,
		OrganizacionRef: x.OrganizacionRef, UnidadRef: x.UnidadRef, Truncado: *x.Truncado,
		Altas: make([]domain.AltaPosibleLoteAdministracion, 0, len(x.Altas)),
		Bajas: make([]domain.BajaPosibleLoteAdministracion, 0, len(x.Bajas))}
	for _, a := range x.Altas {
		// Una duración fuera de rango no se recorta: invalida la respuesta.
		if a.UnidadRequerida == nil || a.DuracionSegundos <= 0 || a.DuracionSegundos > int64(10*365*24*3600) {
			return domain.PreparacionLoteAdministracionPerfiles{}, ports.ErrAutoridadAdministracionPerfilesNoDisponible
		}
		p.Altas = append(p.Altas, domain.AltaPosibleLoteAdministracion{RolVersionRef: a.RolVersionRef, Nombre: a.Nombre,
			UnidadRequerida: *a.UnidadRequerida, VigenteHastaMaxima: a.VigenteHastaMaxima.UTC(),
			DuracionPropuesta: time.Duration(a.DuracionSegundos) * time.Second,
			PerfilRef:         a.PerfilRef, VinculoRef: a.VinculoRef, HuellaSHA256: a.HuellaSHA256})
	}
	for _, b := range x.Bajas {
		if b.Nombre == nil {
			return domain.PreparacionLoteAdministracionPerfiles{}, ports.ErrAutoridadAdministracionPerfilesNoDisponible
		}
		p.Bajas = append(p.Bajas, domain.BajaPosibleLoteAdministracion{RolVersionRef: b.RolVersionRef, Nombre: *b.Nombre,
			PerfilRef: b.PerfilRef, VinculoRef: b.VinculoRef, PerfilVersion: b.PerfilVersion,
			VinculoVersion: b.VinculoVersion, VigenteDesde: b.VigenteDesde.UTC(), VigenteHasta: b.VigenteHasta.UTC(),
			HuellaSHA256: b.HuellaSHA256})
	}
	p.PreparadaEn = p.PreparadaEn.UTC()
	return p, nil
}

// PrepararLoteOrdinario audita todo fallo con sesión V2 válida, igual que el
// lote. El éxito queda registrado por AUT50 y la auditoría nominal de AD190.
func (a *AutoridadLoteOrdinario) PrepararLoteOrdinario(ctx context.Context, s domain.SolicitudPreparacionLoteAdministracionPerfiles) (domain.PreparacionLoteAdministracionPerfiles, error) {
	var vacia domain.PreparacionLoteAdministracionPerfiles
	if a == nil || ctx == nil || ausente(a.registrador) || a.auditoria.validar() != nil {
		return vacia, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	p, err := a.prepararLoteOrdinario(ctx, s)
	if err == nil {
		return p, nil
	}
	if a.registrarFalloConsumo(ctx, AccionPreparacionLote, s.Actor, s.Evidencia, s.CorrelacionRef, s.PersonaRef, err) != nil ||
		errors.Is(err, errCommitLoteIndeterminado) {
		return vacia, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	return vacia, err
}

func (a *AutoridadLoteOrdinario) prepararLoteOrdinario(ctx context.Context, s domain.SolicitudPreparacionLoteAdministracionPerfiles) (domain.PreparacionLoteAdministracionPerfiles, error) {
	var vacia domain.PreparacionLoteAdministracionPerfiles
	if ctx.Err() != nil || ausente(a.pool) || ausente(a.emisor) || ausente(a.reloj) {
		return vacia, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	ahora := a.reloj.Ahora()
	if s.Validar() != nil || s.Evidencia.ValidarEn(s.Actor, ahora) != nil || s.OrganizacionRef != a.organizacion ||
		s.InstantaneaAutorizacion.VersionRol.RolID != "administracion_perfiles" ||
		!domain.VersionRolAplicacionAdmitida(s.InstantaneaAutorizacion.VersionRol.Referencia()) ||
		s.InstantaneaAutorizacion.VersionRol.Estado != domain.EstadoVersionRolPublicada ||
		s.InstantaneaAutorizacion.ControlVigenciaVersionRol.Estado != domain.EstadoControlVigenciaVersionRolHabilitada ||
		!s.InstantaneaAutorizacion.AsignacionPerfil.VigenteEn(ahora) {
		return vacia, domain.ErrActoAdministracionPerfilesInvalido
	}
	canonico, huella, err := s.CanonicoYHuella()
	if err != nil {
		return vacia, domain.ErrActoAdministracionPerfilesInvalido
	}
	actor, err := s.Actor.Clonar()
	if err != nil {
		return vacia, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	resultado, err := s.Evidencia.ResultadoContexto.Clonar()
	if err != nil {
		return vacia, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	evidencia := domain.EvidenciaSesionAdministracionPerfiles{ResultadoContexto: resultado, Vinculo: s.Evidencia.Vinculo}
	if evidencia.ValidarPara(actor) != nil {
		return vacia, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	var instantanea domain.InstantaneaAutorizacion
	b, err := json.Marshal(s.InstantaneaAutorizacion)
	if err != nil || json.Unmarshal(b, &instantanea) != nil || instantanea.Validar() != nil {
		return vacia, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	efecto := Efecto{Accion: accionLoteOrdinario, Audiencia: audienciaLoteOrdinario, Referencia: s.PersonaRef,
		Material: canonico, CorrelacionAccesoRef: s.CorrelacionRef}
	// Mismo canon que recurso_preparacion_lote_admin_v1: el atributo distinto
	// impide usar una decisión de preparación para aplicar un lote y al revés.
	recurso := domain.RecursoAutorizable{Referencia: s.PersonaRef, ModuloID: "administracion", Tipo: "persona",
		Ambitos:   map[string]string{"organizacion_ref": a.organizacion, "unidad_ref": s.UnidadRef},
		Atributos: map[string]string{AtributoPreparacionLote: huella}}
	if recurso.Validar() != nil {
		return vacia, domain.ErrActoAdministracionPerfilesInvalido
	}
	var preparacion domain.PreparacionLoteAdministracionPerfiles
	err = a.ejecutarConsumoLote(ctx, actor, evidencia, instantanea, recurso, efecto, prepararLoteOrdinarioSQL,
		[]any{string(canonico)}, func(bruto []byte) error {
			if len(bruto) > maximoPreparacionLoteBytes {
				return ports.ErrAutoridadAdministracionPerfilesNoDisponible
			}
			var x preparacionLoteJSON
			if decodificarReciboLote(bruto, &x) != nil {
				return ports.ErrAutoridadAdministracionPerfilesNoDisponible
			}
			p, err := x.dominio()
			if err != nil || p.ValidarPara(s) != nil {
				return ports.ErrAutoridadAdministracionPerfilesNoDisponible
			}
			preparacion = p
			return nil
		})
	if err != nil {
		return vacia, err
	}
	return preparacion, nil
}
