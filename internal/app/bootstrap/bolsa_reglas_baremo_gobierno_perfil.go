package bootstrap

import (
	"context"
	"errors"
	"reflect"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	bolsagobierno "vec-diputacion-granada/internal/modules/bolsa/application/gobiernoreglasbaremo"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

var errPerfilGobiernoReglasBaremoV3 = errors.New("bootstrap: perfil de gobierno de reglas de baremo no disponible")

const (
	actoAsignacionGobiernoReglasBaremoV3 = "acto:bolsa:reglas-baremo:gobierno-v3:asignacion:v1"
	actoControlGobiernoReglasBaremoV3    = "acto:bolsa:reglas-baremo:gobierno-v3:control-rol:v1"
	actoSesionGobiernoReglasBaremoV3     = "acto:bolsa:reglas-baremo:gobierno-v3:sesion:v1"
)

// PerfilGobiernoReglasBaremoV3 fija al componer la identidad RRHH y un único
// par convocatoria/expediente. No deriva autoridad de un rol de llamamientos,
// de la petición ni de un identificador que llegue por HTTP.
type PerfilGobiernoReglasBaremoV3 struct {
	soporte         *soporteAltaContratacionTemporalDesarrollo
	plantilla       vecdomain.InstantaneaAutorizacion
	convocatoriaRef string
	expedienteRef   string
}

func discriminadorGobiernoReglasBaremoV3() discriminadorContextoSinteticoDesarrollo {
	const etiqueta = "bolsa-gobierno-reglas-baremo-v3"
	return discriminadorContextoSinteticoDesarrollo{
		perfil: "perfil-" + etiqueta, vinculo: "vinculo-" + etiqueta,
		procedencia: "procedencia", registro: "registro-contexto-" + etiqueta,
		autenticacion: "autenticacion-" + etiqueta, asercion: "asercion-" + etiqueta,
		sesion: "sesion-" + etiqueta, controlSesion: "control-sesion-" + etiqueta,
		politicaGarantia: "politica-garantia-" + etiqueta,
	}
}

// NuevoPerfilGobiernoReglasBaremoV3 sólo recibe un soporte RRHH mTLS ya
// compuesto y ámbitos de configuración privada, nunca datos del formulario.
func NuevoPerfilGobiernoReglasBaremoV3(base *soporteAltaContratacionTemporalDesarrollo, convocatoriaRef, expedienteRef string, ahora time.Time) (*PerfilGobiernoReglasBaremoV3, error) {
	alcance := vecdomain.RecursoAutorizable{Referencia: "reglas-baremo:alcance", ModuloID: "bolsa",
		Tipo: "version_reglas_baremo_gobernada", Ambitos: map[string]string{
			"convocatoria_ref": convocatoriaRef, "expediente_ref": expedienteRef}, Atributos: map[string]string{}}
	if alcance.Validar() != nil || len(alcance.Ambitos) != 2 {
		return nil, errPerfilGobiernoReglasBaremoV3
	}
	soporte, perfil, err := nuevoSoportePlantillasCTDesdeBaseDesarrollo(base, ahora, discriminadorGobiernoReglasBaremoV3())
	if err != nil || soporte == nil || perfil == "" {
		return nil, errPerfilGobiernoReglasBaremoV3
	}
	actor := soporte.contexto.Resultado.Contexto
	plantilla, err := nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(actor.PersonaRef, perfil, ahora,
		"bolsa_reglas_baremo_gobierno_v3", "Gobierno de borradores de reglas de baremo", "bolsa-reglas-baremo-gobierno-v3",
		[]vecdomain.ConcesionRol{
			{Accion: "bolsa.reglas_baremo.borrador.crear", ModuloID: "bolsa", TipoRecurso: "intencion_gobierno_reglas_baremo",
				Finalidades: []string{"gobierno_reglas_baremo"}, CamposPermitidos: []string{"auditoria", "estado_reglas_baremo", "salida_eventos"},
				GarantiaMinima: vecdomain.AuthAssuranceHigh},
			{Accion: "bolsa.reglas_baremo.version.consultar", ModuloID: "bolsa", TipoRecurso: "version_reglas_baremo_gobernada",
				Finalidades: []string{"consulta_gobierno_reglas_baremo"}, CamposPermitidos: []string{"estado_reglas_baremo"},
				GarantiaMinima: vecdomain.AuthAssuranceHigh},
			{Accion: "bolsa.reglas_baremo.recibo.consultar", ModuloID: "bolsa", TipoRecurso: "version_reglas_baremo_gobernada",
				Finalidades: []string{"consulta_gobierno_reglas_baremo"}, CamposPermitidos: []string{"estado_reglas_baremo", "recibo"},
				GarantiaMinima: vecdomain.AuthAssuranceHigh},
		}, []vecdomain.AmbitoPerfil{
			{Clave: "convocatoria_ref", Valores: []string{convocatoriaRef}},
			{Clave: "expediente_ref", Valores: []string{expedienteRef}},
		})
	if err != nil || plantilla.Validar() != nil || actor.Principal.AuthMethod != vecdomain.AuthMethodCertificate ||
		actor.Principal.AuthAssurance != vecdomain.AuthAssuranceHigh ||
		soporte.contexto.Vinculo.ValidarPara(soporte.contexto.Resultado) != nil {
		return nil, errPerfilGobiernoReglasBaremoV3
	}
	return &PerfilGobiernoReglasBaremoV3{soporte, plantilla, convocatoriaRef, expedienteRef}, nil
}

// DescriptorMaterialGobiernoReglasBaremoV3 registra la audiencia del
// consumidor AD144. El registro por sí mismo no publica un rol ni una ruta.
func DescriptorMaterialGobiernoReglasBaremoV3() descriptorMaterialConsumidorV3Desarrollo {
	return descriptorMaterialConsumidorV3Desarrollo{
		Audiencia:        bolsagobierno.AudienciaGobiernoBorradorReglasV3,
		Dominio:          "vec.bolsa.reglas-baremo.gobierno-borrador.capacidad-v3",
		Prefijo:          "clave:capacidad:bolsa-reglas-baremo-gobierno:",
		ProveedorNominal: "proveedor-material-bolsa-reglas-baremo-gobierno",
	}
}

func (p *PerfilGobiernoReglasBaremoV3) PerfilRef() string {
	if p == nil {
		return ""
	}
	return p.plantilla.AsignacionPerfil.PerfilActivoRef
}

// AsegurarPerfilGobiernoReglasBaremoV3 sólo se invoca en arranque o CLI por
// composición. Una asignación existente jamás se sustituye por la semilla.
// Para provisionar una discrepancia se exige aprobación ligada a la huella
// exacta de preimagen, propia y todavía operativa; el publicador hace CAS bajo
// bloqueo y revalida rol, control y catálogo antes del commit.
func AsegurarPerfilGobiernoReglasBaremoV3(ctx context.Context, pool *pgxpool.Pool, p *PerfilGobiernoReglasBaremoV3, aprobacionRef, preimagenSHA256 string, ahora time.Time) error {
	if ctx == nil || ctx.Err() != nil || pool == nil || p == nil || p.soporte == nil ||
		p.plantilla.Validar() != nil || ahora.IsZero() {
		return errPerfilGobiernoReglasBaremoV3
	}
	a := autoridadPostgreSQLDesarrollo{pool: pool, vinculo: p.soporte.contexto.Vinculo,
		prefijoBloqueo: "vec:bolsa:reglas-baremo:gobierno-v3:autorizacion:",
		actoControlRol: actoControlGobiernoReglasBaremoV3,
		actoAsignacion: actoAsignacionGobiernoReglasBaremoV3, actoSesion: actoSesionGobiernoReglasBaremoV3,
		exigirOrigenOperativo: true}
	actual, existe, err := leerInstantaneaPublicadaPostgreSQLDesarrollo(ctx, pool, p.PerfilRef())
	if err != nil {
		return errPerfilGobiernoReglasBaremoV3
	}
	if !existe {
		// Solo la ausencia demostrada permite crear. La segunda lectura ocurre
		// bajo el mismo lock que la escritura, con soloInicial=true.
		a.soloInicial = true
		preparada, err := a.prepararInstantanea(ctx, p.plantilla, true)
		if err != nil || preparada.AsignacionPerfil.Version != 1 || a.publicarInstantanea(ctx, preparada) != nil {
			return errPerfilGobiernoReglasBaremoV3
		}
		return nil
	}
	if _, ok := instantaneaConsumible(actual, p.plantilla, ahora); ok &&
		actual.actoAsignacion == actoAsignacionGobiernoReglasBaremoV3 && actual.actoControl == actoControlGobiernoReglasBaremoV3 {
		return nil
	}
	if !preimagenProvisionableGobiernoReglasBaremoV3(actual, p, aprobacionRef, preimagenSHA256, ahora) {
		return errPerfilGobiernoReglasBaremoV3
	}
	objetivo, err := a.prepararInstantanea(ctx, p.plantilla, false)
	if err != nil || objetivo.AsignacionPerfil.Version != actual.instantanea.AsignacionPerfil.Version+1 ||
		objetivo.AsignacionPerfil.AsignacionID != actual.instantanea.AsignacionPerfil.AsignacionID ||
		a.publicarInstantaneaDesdePreimagen(ctx, objetivo, actual.instantanea) != nil {
		return errPerfilGobiernoReglasBaremoV3
	}
	return nil
}

func preimagenProvisionableGobiernoReglasBaremoV3(actual instantaneaPublicadaDesarrollo,
	p *PerfilGobiernoReglasBaremoV3, aprobacionRef, preimagenSHA256 string, ahora time.Time) bool {
	if p == nil || aprobacionRef == "" || !shaHexGobiernoV3(preimagenSHA256) {
		return false
	}
	huella, err := actual.instantanea.AsignacionPerfil.HuellaSHA256()
	return err == nil && huella == preimagenSHA256 &&
		origenOperativoPublicadoCTDesarrollo(actual, actoAsignacionGobiernoReglasBaremoV3, ahora) &&
		actual.actoControl == actoControlGobiernoReglasBaremoV3 &&
		actual.instantanea.VersionRol.RolID == p.plantilla.VersionRol.RolID &&
		actual.instantanea.AsignacionPerfil.PrincipalID == p.plantilla.AsignacionPerfil.PrincipalID &&
		reflect.DeepEqual(actual.instantanea.VersionRol.Concesiones, p.plantilla.VersionRol.Concesiones) &&
		reflect.DeepEqual(actual.instantanea.AsignacionPerfil.Ambitos, p.plantilla.AsignacionPerfil.Ambitos)
}
