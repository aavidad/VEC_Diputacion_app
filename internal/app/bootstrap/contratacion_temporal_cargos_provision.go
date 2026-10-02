package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// ResumenProvisionCargoCT no revela la identidad. La huella del plan liga la
// aprobación técnica a la preimagen, el ámbito, la vigencia y el acto opcional.
type ResumenProvisionCargoCT struct {
	Cargo            string `json:"cargo"`
	Estado           string `json:"estado"`
	PreimagenSHA256  string `json:"preimagen_sha256"`
	PlanSHA256       string `json:"plan_sha256"`
	VersionSiguiente int    `json:"version_siguiente"`
}

type PlanProvisionCargoCT struct {
	config    ConfiguracionProvisionCargoCT
	preimagen core.InstantaneaAutorizacion
	objetivo  core.InstantaneaAutorizacion
	resumen   ResumenProvisionCargoCT
}

func (p PlanProvisionCargoCT) Resumen() ResumenProvisionCargoCT { return p.resumen }

// fuenteCargoCTPostgreSQL adapta la lectura central existente al puerto
// FuenteAutorizacion. No guarda perfiles ni inventa otra autoridad.
type fuenteCargoCTPostgreSQL struct{ pool *pgxpool.Pool }

func (f fuenteCargoCTPostgreSQL) ObtenerInstantaneaAutorizacion(ctx context.Context, principalID, perfilRef string) (core.InstantaneaAutorizacion, error) {
	publicada, existe, err := leerInstantaneaPublicadaPostgreSQLDesarrollo(ctx, f.pool, perfilRef)
	if err != nil || !existe || publicada.instantanea.AsignacionPerfil.PrincipalID != principalID {
		return core.InstantaneaAutorizacion{}, vecports.ErrFuenteAutorizacionNoDisponible
	}
	return publicada.instantanea, nil
}

var _ vecports.FuenteAutorizacion = fuenteCargoCTPostgreSQL{}

// PrepararProvisionCargoCT solo lee. Una ausencia, revocación, expiración,
// retirada del rol, origen ajeno o huella incorrecta deja el plan cerrado.
func PrepararProvisionCargoCT(ctx context.Context, pool *pgxpool.Pool, c ConfiguracionProvisionCargoCT, ahora time.Time) (PlanProvisionCargoCT, error) {
	if pool == nil {
		return PlanProvisionCargoCT{}, ErrProvisionCargoCT
	}
	return prepararProvisionCargoCT(ctx, fuenteCargoCTPostgreSQL{pool}, func(ctx context.Context, ref string) (instantaneaPublicadaDesarrollo, bool, error) {
		return leerInstantaneaPublicadaPostgreSQLDesarrollo(ctx, pool, ref)
	}, c, ahora)
}

type lectorCargoCT func(context.Context, string) (instantaneaPublicadaDesarrollo, bool, error)

func prepararProvisionCargoCT(ctx context.Context, fuente vecports.FuenteAutorizacion, leer lectorCargoCT, c ConfiguracionProvisionCargoCT, ahora time.Time) (PlanProvisionCargoCT, error) {
	fallo := PlanProvisionCargoCT{}
	if ctx == nil || ctx.Err() != nil || fuente == nil || leer == nil || !c.validar(ahora) {
		return fallo, ErrProvisionCargoCT
	}
	publicada, existe, err := leer(ctx, c.PerfilRef)
	if err != nil || !existe || publicada.instantanea.Validar() != nil ||
		!origenOperativoPublicadoCTDesarrollo(publicada, actoAsignacionPerfilFijoCTDesarrollo, ahora) {
		return fallo, ErrProvisionCargoCT
	}
	previa := publicada.instantanea
	deFuente, err := fuente.ObtenerInstantaneaAutorizacion(ctx, c.PrincipalID, c.PerfilRef)
	if err != nil || deFuente.Validar() != nil || !mismasHuellasInstantaneaDesarrollo(previa, deFuente) ||
		previa.AsignacionPerfil.PrincipalID != c.PrincipalID || previa.AsignacionPerfil.PerfilActivoRef != c.PerfilRef ||
		previa.VersionRol.RolID != rolesCargoCT[c.Cargo] ||
		previa.AsignacionPerfil.Version >= int(^uint(0)>>1) {
		return fallo, ErrProvisionCargoCT
	}
	huella, err := previa.AsignacionPerfil.HuellaSHA256()
	if err != nil || huella != c.PreimagenSHA256 {
		return fallo, ErrProvisionCargoCT
	}
	huellaRol, err := previa.VersionRol.HuellaSHA256()
	if err != nil || huellaRol != c.VersionRolSHA256 {
		return fallo, ErrProvisionCargoCT
	}
	objetivo := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(previa)
	objetivo.AsignacionPerfil.Version++
	objetivo.AsignacionPerfil.Ambitos = []core.AmbitoPerfil{
		{Clave: "organizacion_ref", Valores: []string{c.OrganizacionRef}},
		{Clave: "unidad_ref", Valores: []string{c.UnidadRef}},
	}
	objetivo.AsignacionPerfil.VigenteDesde = c.VigenteDesde
	objetivo.AsignacionPerfil.VigenteHasta = c.VigenteHasta
	objetivo.AsignacionPerfil.EmitidaEn = ahora
	if objetivo.Validar() != nil {
		return fallo, ErrProvisionCargoCT
	}
	// La huella del plan usa JSON canónico de estructuras Go cerradas. No
	// incluye secretos, nombre personal ni concesiones configurables.
	huellaObjetivo, err := objetivo.AsignacionPerfil.HuellaSHA256()
	if err != nil {
		return fallo, ErrProvisionCargoCT
	}
	material, err := json.Marshal(struct {
		Config    ConfiguracionProvisionCargoCT `json:"config"`
		Preimagen string                        `json:"preimagen"`
		Objetivo  string                        `json:"objetivo"`
	}{c, huella, huellaObjetivo})
	if err != nil {
		return fallo, ErrProvisionCargoCT
	}
	suma := sha256.Sum256(material)
	return PlanProvisionCargoCT{config: c, preimagen: previa, objetivo: objetivo,
		resumen: ResumenProvisionCargoCT{Cargo: c.Cargo, Estado: "pendiente_canal_autorizado",
			PreimagenSHA256: huella, PlanSHA256: hex.EncodeToString(suma[:]), VersionSiguiente: objetivo.AsignacionPerfil.Version}}, nil
}
