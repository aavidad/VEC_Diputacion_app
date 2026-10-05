package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const accionLoteOrdinario = "administracion.perfiles.aplicar_lote_ordinario"
const audienciaLoteOrdinario = "vec_autorizacion.administracion_perfiles.lote_ordinario.v1"
const aplicarLoteOrdinarioSQL = `SELECT vec_autorizacion.aplicar_lote_ordinario_admin_v1($1::text,$2::jsonb,$3::bytea,$4::bytea,$5::bytea,$6::bytea,$7::numeric,$8::numeric,$9::bytea,$10::bytea,$11::bytea,$12::bytea)`

// El emisor recibe el recurso V3 exacto. El ámbito privado se coteja de nuevo
// en AUT44; ni el DTO ni esta interfaz conceden acceso por sí solos.
type EmisorLoteOrdinario interface {
	EmitirLoteOrdinario(context.Context, domain.ContextoActor,
		domain.EvidenciaSesionAdministracionPerfiles, domain.InstantaneaAutorizacion,
		domain.RecursoAutorizable, Efecto) (ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

type AutoridadLoteOrdinario struct {
	pool         conexion
	emisor       EmisorLoteOrdinario
	proveedor    ProveedorAmbitosLote
	registrador  ports.RegistradorIntentosAuditoria
	auditoria    ConfiguracionAuditoriaLote
	reloj        ports.Reloj
	organizacion string
}

var _ ports.AutoridadLotesAdministracionPerfiles = (*AutoridadLoteOrdinario)(nil)

// NuevaAutoridadLoteOrdinario no depende de las fachadas singulares heredadas.
// El LOGIN propio se acredita en SQL antes de recibir la primera orden.
func NuevaAutoridadLoteOrdinario(ctx context.Context, pool *pgxpool.Pool, emisor EmisorLoteOrdinario,
	proveedor ProveedorAmbitosLote, registrador ports.RegistradorIntentosAuditoria,
	auditoria ConfiguracionAuditoriaLote, organizacion string, reloj ports.Reloj) (*AutoridadLoteOrdinario, error) {
	return nuevaAutoridadLoteOrdinario(ctx, pool, emisor, proveedor, registrador, auditoria, organizacion, reloj)
}

func nuevaAutoridadLoteOrdinario(ctx context.Context, pool conexion, emisor EmisorLoteOrdinario,
	proveedor ProveedorAmbitosLote, registrador ports.RegistradorIntentosAuditoria,
	auditoria ConfiguracionAuditoriaLote, organizacion string, reloj ports.Reloj) (*AutoridadLoteOrdinario, error) {
	if ctx == nil || ausente(pool) || ausente(emisor) || ausente(proveedor) || ausente(registrador) ||
		auditoria.validar() != nil || ausente(reloj) ||
		!referenciaAmbitoLote.MatchString(organizacion) || ctx.Err() != nil {
		return nil, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	var acreditado bool
	if err := pool.QueryRow(ctx, `SELECT vec_autorizacion.acreditar_login_lote_ordinario_admin_v1()`).Scan(&acreditado); err != nil || !acreditado {
		return nil, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	return &AutoridadLoteOrdinario{pool: pool, emisor: emisor, proveedor: proveedor,
		registrador: registrador, auditoria: auditoria, reloj: reloj, organizacion: organizacion}, nil
}

type reciboLoteOrdinarioJSON struct {
	OperacionRef          string       `json:"operacion_ref"`
	ActoRef               string       `json:"acto_ref"`
	ReciboRef             string       `json:"recibo_ref"`
	AuditoriaRef          string       `json:"auditoria_ref"`
	HuellaSolicitudSHA256 string       `json:"huella_solicitud_sha256"`
	FuentesSHA256         string       `json:"fuentes_sha256"`
	ConfirmadoEn          time.Time    `json:"confirmado_en"`
	Cambios               []reciboJSON `json:"cambios"`
	Inicios               []struct {
		Modo         domain.InicioVigenciaLoteAdministracion `json:"modo"`
		VigenteDesde time.Time                               `json:"vigente_desde"`
	} `json:"inicios"`
}

func (x reciboLoteOrdinarioJSON) dominio() domain.ReciboLoteAdministracionPerfiles {
	r := domain.ReciboLoteAdministracionPerfiles{
		OperacionRef: x.OperacionRef, ActoRef: x.ActoRef, ReciboRef: x.ReciboRef,
		AuditoriaRef: x.AuditoriaRef, HuellaSolicitudSHA256: x.HuellaSolicitudSHA256,
		FuentesSHA256: x.FuentesSHA256,
		ConfirmadoEn:  x.ConfirmadoEn, Cambios: make([]domain.ReciboAdministracionPerfiles, 0, len(x.Cambios)),
	}
	for _, c := range x.Cambios {
		r.Cambios = append(r.Cambios, c.dominio())
	}
	for _, inicio := range x.Inicios {
		r.Inicios = append(r.Inicios, domain.InicioEfectivoLoteAdministracion{
			Modo: inicio.Modo, VigenteDesde: inicio.VigenteDesde})
	}
	return r
}

func (a *AutoridadLoteOrdinario) aplicarLoteOrdinario(ctx context.Context, s domain.SolicitudLoteAdministracionPerfiles) (domain.ReciboLoteAdministracionPerfiles, error) {
	var vacio domain.ReciboLoteAdministracionPerfiles
	if a == nil || ctx == nil || ausente(a.pool) || ausente(a.emisor) || ausente(a.proveedor) ||
		ausente(a.reloj) || ctx.Err() != nil {
		return vacio, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	if s.Validar() != nil || s.Evidencia.ValidarEn(s.Actor, a.reloj.Ahora()) != nil ||
		s.OrganizacionRef != a.organizacion ||
		s.InstantaneaAutorizacion.VersionRol.RolID != "administracion_perfiles" ||
		!domain.VersionRolAplicacionAdmitida(s.InstantaneaAutorizacion.VersionRol.Referencia()) ||
		s.InstantaneaAutorizacion.VersionRol.Estado != domain.EstadoVersionRolPublicada ||
		s.InstantaneaAutorizacion.ControlVigenciaVersionRol.Estado != domain.EstadoControlVigenciaVersionRolHabilitada ||
		!s.InstantaneaAutorizacion.AsignacionPerfil.VigenteEn(a.reloj.Ahora()) {
		return vacio, domain.ErrActoAdministracionPerfilesInvalido
	}
	for _, cambio := range s.Cambios {
		// AUT44 rechaza estos casos; se cierran aquí para no gastar una decisión.
		if cambio.Objetivo.RevisionContinuidad != 0 || !referenciaAmbitoLote.MatchString(cambio.Objetivo.UnidadRef) {
			return vacio, domain.ErrActoAdministracionPerfilesInvalido
		}
		// Una baja no exige que el perfil siga ofreciéndose: AUT44 comprueba
		// que su versión esté registrada como ordinaria. Sólo las altas se
		// cotejan aquí con el perfil vigente.
		if cambio.Operacion == domain.OperacionRevocarPerfil {
			continue
		}
		rol, err := a.resolverRolLote(ctx, cambio.RolVersionRef)
		if err != nil {
			return vacio, err
		}
		if rol.VersionRef != cambio.RolVersionRef || rol.Clase != domain.ClaseControlPerfilOrdinario ||
			(rol.UnidadRequerida && cambio.Objetivo.UnidadRef == "") ||
			(cambio.Operacion == domain.OperacionOtorgarPerfil &&
				(cambio.InicioVigencia == domain.InicioVigenciaLoteProgramado && cambio.Objetivo.VigenteDesde.Before(rol.VigenteDesde) ||
					cambio.Objetivo.VigenteHasta.After(rol.VigenteHasta))) {
			return vacio, domain.ErrActoAdministracionPerfilesInvalido
		}
	}
	canonico, huella, err := s.CanonicoYHuella()
	if err != nil || huella != s.HuellaSolicitudSHA256 || len(canonico) > 65536 {
		return vacio, domain.ErrActoAdministracionPerfilesInvalido
	}
	actor, err := s.Actor.Clonar()
	if err != nil {
		return vacio, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	resultado, err := s.Evidencia.ResultadoContexto.Clonar()
	if err != nil {
		return vacio, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	evidencia := domain.EvidenciaSesionAdministracionPerfiles{ResultadoContexto: resultado, Vinculo: s.Evidencia.Vinculo}
	if evidencia.ValidarPara(actor) != nil {
		return vacio, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	var instantanea domain.InstantaneaAutorizacion
	b, err := json.Marshal(s.InstantaneaAutorizacion)
	if err != nil || json.Unmarshal(b, &instantanea) != nil || instantanea.Validar() != nil {
		return vacio, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	efecto := Efecto{Accion: accionLoteOrdinario, Audiencia: audienciaLoteOrdinario,
		Referencia: s.Cambios[0].Objetivo.PersonaRef, Material: canonico, CorrelacionAccesoRef: s.CorrelacionRef}
	recurso := domain.RecursoAutorizable{Referencia: efecto.Referencia, ModuloID: "administracion", Tipo: "persona",
		Ambitos:   map[string]string{"organizacion_ref": a.organizacion, "unidad_ref": s.Cambios[0].Objetivo.UnidadRef},
		Atributos: map[string]string{"solicitud_sha256": huella}}
	if recurso.Validar() != nil || s.Cambios[0].Objetivo.CentroRef != "" {
		return vacio, domain.ErrActoAdministracionPerfilesInvalido
	}
	fuentes := materialFuentesPrivadasLote(ctx, a.proveedor, a.organizacion, s)
	defer clear(fuentes)
	// Sin fuentes de ámbito no se emite ninguna decisión: sería un consumo que
	// AUT44 rechazaría, y la falta de fuente no es una denegación.
	if len(fuentes) == 0 {
		return vacio, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	var recibo domain.ReciboLoteAdministracionPerfiles
	err = a.ejecutarLote(ctx, actor, evidencia, instantanea, recurso, efecto, fuentes, func(bruto []byte) error {
		var x reciboLoteOrdinarioJSON
		if decodificarReciboLote(bruto, &x) != nil || len(x.Cambios) != len(s.Cambios) || len(x.Inicios) != len(s.Cambios) {
			return ports.ErrAutoridadAdministracionPerfilesNoDisponible
		}
		for _, c := range x.Cambios {
			if !c.vigenciaHistoricaCompleta() {
				return ports.ErrAutoridadAdministracionPerfilesNoDisponible
			}
		}
		recibo = x.dominio()
		if recibo.ValidarPara(s) != nil {
			return ports.ErrAutoridadAdministracionPerfilesNoDisponible
		}
		return nil
	})
	if err != nil {
		return vacio, err
	}
	return recibo, nil
}

// Un lote válido de 32 cambios produce recibos de más de 64 KiB; el tope del
// recibo del lote es propio y sigue rechazando duplicados y campos ajenos.
const maximoReciboLoteBytes = 256 * 1024

func decodificarReciboLote(b []byte, destino any) error {
	if len(b) == 0 || len(b) > maximoReciboLoteBytes {
		return ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	objeto := bytes.TrimSpace(b)
	if len(objeto) == 0 || objeto[0] != '{' {
		return ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(destino) != nil || d.Decode(new(any)) != io.EOF {
		return ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	return nil
}
