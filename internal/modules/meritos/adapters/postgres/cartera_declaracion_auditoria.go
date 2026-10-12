package postgres

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/meritos/application"
	"vec-diputacion-granada/internal/modules/meritos/ports"
	vec "vec-diputacion-granada/internal/vec/domain"
)

const registrarIntentoDeclaracionMeritos = `SELECT auditoria_ref,secuencia::text,huella_sha256,correlacion_ref,registrada_en
FROM vec_autorizacion_atestada_v3.registrar_intento_declaracion_meritos_v1($1::bytea,$2::bytea,$3::jsonb)`

// El servicio crea esta capacidad por petición. Conserva únicamente el
// contexto registrado y el vínculo de identidad que ya validó la frontera.
type auditoriaDeclaracionCartera struct {
	auditor     *AuditoriaIntentosCartera
	personaRef  string
	perfilRef   string
	correlacion string
	contexto    []byte
	vinculo     []byte
	datos       vec.DatosVinculoAutenticacionActorV2
}

var _ ports.AuditoriaIntentos = (*auditoriaDeclaracionCartera)(nil)

func NuevaAuditoriaDeclaracionCartera(pool *pgxpool.Pool,
	solicitud application.Solicitud) (ports.AuditoriaIntentos, error) {
	if nulo(pool) || solicitud.Contexto.Validar() != nil ||
		solicitud.Vinculo.ValidarPara(solicitud.Contexto) != nil ||
		solicitud.Correlacion.Validar() != nil {
		return nil, ports.ErrCarteraNoDisponible
	}
	datos, err := solicitud.Vinculo.Datos()
	if err != nil || datos.Superficie != vec.SuperficieAutenticacionExternaPersonalV1 ||
		datos.GarantiaObservada != vec.AuthAssuranceHigh || datos.CuentaPrivilegiada ||
		datos.PrincipalID != solicitud.Contexto.Contexto.PersonaRef ||
		datos.PerfilActivoRef != solicitud.Contexto.Contexto.PerfilActivoRef ||
		!datos.SesionValidaHasta.After(time.Now().UTC()) {
		return nil, ports.ErrCarteraNoDisponible
	}
	correlacion, err := solicitud.Correlacion.ValorCanonico()
	if err != nil {
		return nil, ports.ErrCarteraNoDisponible
	}
	vinculo, err := json.Marshal(solicitud.Vinculo)
	if err != nil || len(vinculo) == 0 || len(vinculo) > 16384 ||
		len(solicitud.Contexto.RepresentacionCanonica) == 0 ||
		len(solicitud.Contexto.RepresentacionCanonica) > 65536 {
		return nil, ports.ErrCarteraNoDisponible
	}
	return &auditoriaDeclaracionCartera{
		auditor:     &AuditoriaIntentosCartera{pool: pool},
		personaRef:  datos.PrincipalID,
		perfilRef:   datos.PerfilActivoRef,
		correlacion: correlacion,
		contexto:    append([]byte(nil), solicitud.Contexto.RepresentacionCanonica...),
		vinculo:     vinculo,
		datos:       datos,
	}, nil
}

func (a *auditoriaDeclaracionCartera) AppendAudit(ctx context.Context,
	entrada vec.AuditEntry) (vec.AuditEntry, error) {
	if ctx == nil || a == nil || a.auditor == nil ||
		entrada.ActorID != a.personaRef || entrada.ActorProfile != a.perfilRef ||
		entrada.CorrelationRef != a.correlacion ||
		entrada.Action != "meritos.hecho.declarar" ||
		entrada.ModuleID != "meritos" ||
		entrada.Purpose != "declaracion_hecho_propio" ||
		entrada.Result != "no_confirmado" ||
		entrada.OccurredAt.IsZero() || entrada.OccurredAt.After(time.Now().UTC().Add(time.Second)) {
		return vec.AuditEntry{}, ports.ErrCarteraNoDisponible
	}
	recurso := entrada.SubjectRef
	if recurso == "" {
		var aleatorio [32]byte
		if _, err := rand.Read(aleatorio[:]); err != nil {
			return vec.AuditEntry{}, ports.ErrCarteraNoDisponible
		}
		recurso = "ref:" + hex.EncodeToString(aleatorio[:])
	}
	if len(recurso) != 68 || recurso[:4] != "ref:" || !huellaSHA256CarteraSQL(recurso[4:]) {
		return vec.AuditEntry{}, ports.ErrCarteraNoDisponible
	}
	var aleatorio [16]byte
	if _, err := rand.Read(aleatorio[:]); err != nil {
		return vec.AuditEntry{}, ports.ErrCarteraNoDisponible
	}
	huella := sha256.Sum256(a.contexto)
	orden := map[string]string{
		"esquema":               "vec.meritos.hecho.declaracion.v1",
		"persona_ref":           a.personaRef,
		"intento_ref":           "intento_" + hex.EncodeToString(aleatorio[:]),
		"registro_contexto_ref": a.datos.RegistroContextoRef,
		"contexto_sha256":       hex.EncodeToString(huella[:]),
		"procedencia_sha256":    a.datos.ManifiestoProcedenciaHuellaSHA256,
		"autenticacion_ref":     a.datos.AutenticacionRef,
		"sesion_ref":            a.datos.SesionRef,
		"autenticacion_sha256":  a.datos.AutenticacionHuellaSHA256,
		"accion":                "meritos.hecho.declarar",
		"modulo_id":             "meritos",
		"tipo_recurso":          "hecho",
		"recurso_ref":           recurso,
		"finalidad_ref":         "declaracion_hecho_propio",
		"resultado":             "error",
		"motivo_ref":            "intento_declaracion_meritos_error",
		"proceso":               "vec-server",
		"canal":                 string(vec.SuperficieAutenticacionExternaPersonalV1),
		"correlacion_ref":       a.correlacion,
	}
	b, err := json.Marshal(orden)
	if err != nil {
		return vec.AuditEntry{}, ports.ErrCarteraNoDisponible
	}
	captura, err := ports.NuevaCapturaLecturaCartera(a.personaRef, a.correlacion,
		a.contexto, a.vinculo, b, a.datos.SesionValidaHasta)
	if err != nil {
		return vec.AuditEntry{}, err
	}
	if err := a.auditor.registrar(ctx, captura, registrarIntentoDeclaracionMeritos); err != nil {
		return vec.AuditEntry{}, err
	}
	return entrada, nil
}
