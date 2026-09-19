package interna

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/dietas/adapters/httpinterno"
	dietaspg "vec-diputacion-granada/internal/modules/dietas/adapters/postgres"
	dietasapp "vec-diputacion-granada/internal/modules/dietas/application"
	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	"vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

// DependenciasBorradorDietas recibe autoridades ya compuestas. PerfilActivoRef
// es una selección nominal del servidor; no se deriva de roles ni del cliente.
// El constructor no publica rutas, crea concesiones ni instala esquemas.
type DependenciasBorradorDietas struct {
	Pool                *pgxpool.Pool
	Identidad           *httpseguridad.ServicioIdentidad
	Revalidador         core.RevalidadorAutenticacionActorV1
	Contextos           core.ResolutorContextoActorRegistradoV2
	Emisor              *confianzaatestacion.EmisorMaterialAutorizacionAtestadaV3
	Reloj               core.RelojVinculoAutenticacionActorV2
	PerfilActivoRef     string
	Motivo              core.ReferenciaEntradaCatalogo
	PoliticaKilometraje dietasports.ProveedorPoliticaKilometrajeBorrador
	ZonaHoraria         *time.Location
}

type autoridadBorradorDietas struct{ dependencias DependenciasBorradorDietas }

// NuevoManejadorBorradorDietas conecta HTTP, aplicación y PostgreSQL. La raíz
// debe montarlo únicamente tras autenticar y vincular la cápsula corporativa
// mediante FachadaIdentidadOffline y con roles SQL exclusivos del módulo.
func NuevoManejadorBorradorDietas(d DependenciasBorradorDietas) (*httpinterno.ManejadorBorradorComision, error) {
	if d.Pool == nil || d.Identidad == nil || interfazNulaIdentidadOffline(d.Revalidador) ||
		interfazNulaIdentidadOffline(d.Contextos) || d.Emisor == nil || interfazNulaIdentidadOffline(d.Reloj) ||
		d.Motivo.Validar() != nil {
		return nil, dietasports.ErrBorradorNoDisponible
	}
	// Validar sintaxis de selección sin fabricar un actor ni dar permiso.
	if !regexp.MustCompile(`^prf_[A-Za-z0-9_-]{22,128}$`).MatchString(d.PerfilActivoRef) {
		return nil, dietasports.ErrBorradorNoDisponible
	}
	autoridad := &autoridadBorradorDietas{d}
	repositorio, e := dietaspg.NuevoRepositorioBorradorComision(d.Pool, autoridad)
	if e != nil {
		return nil, e
	}
	servicio, e := dietasapp.NuevoServicioBorradorComision(repositorio, d.PoliticaKilometraje, d.ZonaHoraria)
	if e != nil {
		return nil, e
	}
	return httpinterno.NuevoManejadorBorradorComision(servicio, autoridad)
}

func (a *autoridadBorradorDietas) resolver(ctx context.Context) (core.VinculoAutenticacionActorV2, core.ResultadoContextoActorRegistradoV2, error) {
	vacio := core.VinculoAutenticacionActorV2{}
	resultadoVacio := core.ResultadoContextoActorRegistradoV2{}
	if a == nil || ctx == nil || ctx.Err() != nil || a.dependencias.Identidad == nil {
		return vacio, resultadoVacio, dietasports.ErrAccesoBorradorDenegado
	}
	d := a.dependencias
	cuenta, auditoria, e := d.Identidad.ExtraerCapsulaIdentidadPeticion(ctx)
	if e != nil || auditoria.Superficie() != httpseguridad.SuperficieInternaCorporativa || auditoria.CuentaPrivilegiada() {
		return vacio, resultadoVacio, dietasports.ErrAccesoBorradorDenegado
	}
	vinculo, resultado, e := core.CrearVinculoAutenticacionActorV2ConResultado(ctx, d.Revalidador,
		core.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: auditoria.AutenticacionRef(), SesionRef: auditoria.SesionRef()},
		d.Contextos, core.SolicitudContextoActor{Cuenta: cuenta, PerfilActivoRef: d.PerfilActivoRef}, d.Reloj)
	if e != nil {
		return vacio, resultadoVacio, dietasports.ErrAccesoBorradorDenegado
	}
	datos, e := vinculo.Datos()
	if e != nil || datos.Superficie != core.SuperficieAutenticacionInternaCorporativaV1 || datos.CuentaPrivilegiada ||
		datos.CuentaRef != cuenta.CuentaRef || datos.AutenticacionHuellaSHA256 != auditoria.AutenticacionHuellaSHA256() {
		return vacio, resultadoVacio, dietasports.ErrAccesoBorradorDenegado
	}
	if _, e = dietasapp.EmpleadoBorradorPropio(resultado.Contexto); e != nil {
		return vacio, resultadoVacio, e
	}
	return vinculo, resultado, nil
}

func (a *autoridadBorradorDietas) ResolverContextoActor(ctx context.Context) (core.ContextoActor, error) {
	_, resultado, e := a.resolver(ctx)
	if e != nil {
		return core.ContextoActor{}, e
	}
	return resultado.Contexto.Clonar()
}

func (a *autoridadBorradorDietas) AutorizarBorradorPropio(ctx context.Context, actor core.ContextoActor, accion string, recurso core.RecursoAutorizable) (dietaspg.AutorizacionBorrador, error) {
	var vacio dietaspg.AutorizacionBorrador
	if accion != dietaspg.AccionCrearBorradorPropio && accion != dietaspg.AccionRecuperarBorradorPropio && accion != dietaspg.AccionListarBorradoresPropios {
		return vacio, dietasports.ErrAccesoBorradorDenegado
	}
	vinculo, resultado, e := a.resolver(ctx)
	if e != nil {
		return vacio, e
	}
	// La identidad se vuelve a resolver antes de emitir: un cambio de perfil,
	// persona, empleado o cualquier versión aborta la petición que ya empezó.
	if actor.Validar() != nil || !reflect.DeepEqual(actor.Instantanea, resultado.Contexto.Instantanea) ||
		!reflect.DeepEqual(actor.Principal, resultado.Contexto.Principal) {
		return vacio, dietasports.ErrAccesoBorradorDenegado
	}
	empleado, e := dietasapp.EmpleadoBorradorPropio(resultado.Contexto)
	if e != nil || recurso.ModuloID != "dietas" || recurso.Tipo != "borrador_comision" ||
		len(recurso.Ambitos) != 2 || recurso.Ambitos["persona_ref"] != actor.PersonaRef || recurso.Ambitos["empleado_ref"] != empleado ||
		len(recurso.Atributos) != 1 || recurso.Atributos["material_sha256"] == "" {
		return vacio, dietasports.ErrAccesoBorradorDenegado
	}
	correlacion, e := core.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridad.GeneradorReferenciasCriptograficas{})
	if e != nil {
		return vacio, dietasports.ErrBorradorNoDisponible
	}
	solicitud, e := core.NuevaSolicitudAutorizacionLigadaV3(core.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: vinculo, ReferenciaMotivo: a.dependencias.Motivo, Accion: accion,
		Recurso: recurso, Finalidad: dietaspg.FinalidadBorradorPropio, Correlacion: correlacion})
	if e != nil {
		return vacio, dietasports.ErrAccesoBorradorDenegado
	}
	decision, _, exportador, emisionErr := a.dependencias.Emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, resultado)
	positiva, _, decisionErr := decision.Resultado()
	if decisionErr != nil || !positiva {
		return vacio, dietasports.ErrAccesoBorradorDenegado
	}
	// Una concesión positiva no equivale a un efecto confirmado: la unidad
	// transaccional consume el material junto con recibo y auditoría de negocio.
	base, baseErr := resultadoBaseBorrador(decision, resultado)
	if baseErr != nil {
		return vacio, dietasports.ErrBorradorNoDisponible
	}
	return a.finalizarEmisionBorrador(ctx, base, exportador, emisionErr, func() bool {
		ahora := a.dependencias.Reloj.Ahora().UTC().Truncate(time.Microsecond)
		return vinculo.VigenteEn(ahora, resultado)
	})
}

// resultadoBaseBorrador extrae sólo enlaces desde la decisión sellada y el
// contexto registrado. Nunca utiliza material de HTTP ni una exportación fallida.
func resultadoBaseBorrador(decision core.DecisionAutorizacionLigadaV3, resultado core.ResultadoContextoActorRegistradoV2) (dietaspg.EnlaceAutorizacionBorrador, error) {
	var base dietaspg.EnlaceAutorizacionBorrador
	canon, e := core.RepresentacionCanonicaDecisionAutorizacionV3(decision)
	if e != nil {
		return base, e
	}
	defer clear(canon)
	var datos struct {
		Decision      string `json:"decision_ref"`
		Actor         string `json:"principal_id"`
		Perfil        string `json:"perfil_activo_ref"`
		Correlacion   string `json:"correlacion_ref"`
		Accion        string `json:"accion"`
		Recurso       string `json:"recurso_ref"`
		HuellaRecurso string `json:"contexto_recurso_huella_sha256"`
	}
	if e = json.Unmarshal(canon, &datos); e != nil || resultado.Validar() != nil {
		return base, dietasports.ErrBorradorNoDisponible
	}
	huella := sha256.Sum256(canon)
	base = dietaspg.EnlaceAutorizacionBorrador{
		DecisionRef: datos.Decision, DecisionHuellaSHA256: hex.EncodeToString(huella[:]),
		ContextoRef: resultado.RegistroContextoRef, ContextoHuellaSHA256: resultado.HuellaSHA256,
		ActorRef: datos.Actor, PerfilRef: datos.Perfil, CorrelacionRef: datos.Correlacion,
		Accion: datos.Accion, RecursoRef: datos.Recurso, RecursoHuellaSHA256: datos.HuellaRecurso,
	}
	return base, nil
}

func (a *autoridadBorradorDietas) finalizarEmisionBorrador(ctx context.Context, base dietaspg.EnlaceAutorizacionBorrador, exportador vp.ExportadorMaterialConsumoAutorizacionAtestadaV3, emisionErr error, vigente func() bool) (dietaspg.AutorizacionBorrador, error) {
	if ctx == nil || ctx.Err() != nil {
		return dietaspg.AutorizacionBorrador{}, dietasports.ErrBorradorNoDisponible
	}
	if emisionErr != nil || interfazNulaIdentidadOffline(exportador) {
		return dietaspg.AutorizacionBorrador{}, dietasports.ErrBorradorNoDisponible
	}
	material, e := exportador.ExportarMaterialParaConsumidor()
	if e != nil {
		return dietaspg.AutorizacionBorrador{}, dietasports.ErrBorradorNoDisponible
	}
	if ctx.Err() != nil {
		return dietaspg.AutorizacionBorrador{}, dietasports.ErrBorradorNoDisponible
	}
	if vigente == nil || !vigente() {
		return dietaspg.AutorizacionBorrador{}, dietasports.ErrBorradorNoDisponible
	}
	return dietaspg.AutorizacionBorrador{Material: material, ResultadoBase: base}, nil
}
