package composicion

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	personalpostgres "vec-diputacion-granada/internal/modules/personal/adapters/postgres"
	personalapp "vec-diputacion-granada/internal/modules/personal/application"
	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// IdentidadRegistradaLectorServiciosCertificados procede sólo de la frontera
// de sesión de la petición: ContextoActor registrado y vínculo revalidado.
type IdentidadRegistradaLectorServiciosCertificados struct {
	Vinculo   vecdomain.VinculoAutenticacionActorV2
	Resultado vecdomain.ResultadoContextoActorRegistradoV2
}

type ResolutorIdentidadLectorServiciosCertificados interface {
	ResolverIdentidadLectorServiciosCertificados(context.Context) (IdentidadRegistradaLectorServiciosCertificados, error)
}

// EmisorMaterialLectorServiciosCertificadosV3 es la autoridad común V3 ya
// compuesta (PDP, firmante y verificador). No admite permisos del consumidor.
type EmisorMaterialLectorServiciosCertificadosV3 interface {
	EmitirMaterialAutorizacionAtestadaV3(context.Context, vecdomain.SolicitudAutorizacionLigadaV3, vecdomain.ResultadoContextoActorRegistradoV2) (vecdomain.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error)
}

// RegistradorIntentosLectorServiciosCertificados es el destino común con su
// preflight real. No concede permiso ni mantiene un registro propio.
type RegistradorIntentosLectorServiciosCertificados interface {
	vecports.RegistradorIntentosAuditoria
	PreflightIntentoAuditoria(context.Context) error
}

type ConfiguracionIntentosLectorServiciosCertificados struct {
	Proceso                string                              `json:"proceso"`
	Canal                  string                              `json:"canal"`
	RecursoEntradaInvalida string                              `json:"recurso_entrada_invalida"`
	MotivoDenegado         vecdomain.ReferenciaEntradaCatalogo `json:"motivo_denegado"`
	MotivoEntradaInvalida  vecdomain.ReferenciaEntradaCatalogo `json:"motivo_entrada_invalida"`
	MotivoNoDisponible     vecdomain.ReferenciaEntradaCatalogo `json:"motivo_no_disponible"`
}

// LectorServiciosCertificados es lo que recibe el consumidor de Certificados.
type LectorServiciosCertificados interface {
	personalports.LectorServiciosParaCertificadosV1
	personalports.LectorServiciosParaCertificadosV2
}

type DependenciasLectorServiciosCertificados struct {
	Identidad             ResolutorIdentidadLectorServiciosCertificados
	Emisor                EmisorMaterialLectorServiciosCertificadosV3
	Motivo                vecdomain.ReferenciaEntradaCatalogo
	Lectura               *pgxpool.Pool
	Intentos              RegistradorIntentosLectorServiciosCertificados
	ConfiguracionIntentos ConfiguracionIntentosLectorServiciosCertificados
	Ahora                 func() time.Time
	LimiteIdentidad       time.Duration
}

// ComponerLectorServiciosCertificados prepara el lector con autoridades reales
// y el registrador común. No monta HTTP ni provisiona permisos: sin SQL,
// concesión o registrador cada lectura queda cerrada.
func ComponerLectorServiciosCertificados(d DependenciasLectorServiciosCertificados) (LectorServiciosCertificados, error) {
	if d.Lectura == nil || dependenciaNula(d.Identidad) || dependenciaNula(d.Intentos) || d.Ahora == nil || d.LimiteIdentidad <= 0 || d.LimiteIdentidad > 30*time.Second {
		return nil, personaldomain.ErrLectorServiciosCertificadosNoDisponible
	}
	a, err := NuevoProveedorAutorizacionLectorServiciosCertificados(d.Emisor, d.Motivo)
	if err != nil {
		return nil, personaldomain.ErrLectorServiciosCertificadosNoDisponible
	}
	r, err := personalpostgres.NuevoRepositorioLectorServiciosCertificadosPostgreSQL(d.Lectura)
	if err != nil {
		return nil, personaldomain.ErrLectorServiciosCertificadosNoDisponible
	}
	i, err := NuevoRegistroIntentosLectorServiciosCertificados(d.Intentos, d.ConfiguracionIntentos)
	if err != nil {
		return nil, personaldomain.ErrLectorServiciosCertificadosNoDisponible
	}
	s, err := personalapp.NuevoServicioLectorServiciosCertificados(a, r, i, d.Ahora)
	if err != nil {
		return nil, personaldomain.ErrLectorServiciosCertificadosNoDisponible
	}
	return NuevoLectorServiciosCertificadosConIdentidad(s, d.Identidad, d.LimiteIdentidad)
}

// --- Frontera: identidad capturada antes de validar o leer ---

type claveIntentoLectorServiciosCertificados struct{}

type intentoLectorServiciosCertificados struct {
	identidad  IdentidadRegistradaLectorServiciosCertificados
	referencia string
	mu         sync.Mutex
	orden      *vecports.OrdenIntentoAuditoria
	acuse      *vecports.AcuseIntentoAuditoria
}

type lectorServiciosCertificadosConIdentidad struct {
	siguiente LectorServiciosCertificados
	identidad ResolutorIdentidadLectorServiciosCertificados
	limite    time.Duration
}

func NuevoLectorServiciosCertificadosConIdentidad(l LectorServiciosCertificados, i ResolutorIdentidadLectorServiciosCertificados, limite time.Duration) (LectorServiciosCertificados, error) {
	if dependenciaNula(l) || dependenciaNula(i) || limite <= 0 || limite > 30*time.Second {
		return nil, personaldomain.ErrLectorServiciosCertificadosNoDisponible
	}
	return lectorServiciosCertificadosConIdentidad{siguiente: l, identidad: i, limite: limite}, nil
}

func (l lectorServiciosCertificadosConIdentidad) ConsultarServiciosParaCertificados(ctx context.Context, in personalports.ConsultaServiciosParaCertificadosV1) (personalports.ResultadoServiciosParaCertificadosV1, error) {
	c, err := l.capturar(ctx)
	if err != nil {
		return personalports.ResultadoServiciosParaCertificadosV1{}, err
	}
	return l.siguiente.ConsultarServiciosParaCertificados(c, in)
}

func (l lectorServiciosCertificadosConIdentidad) ConsultarServiciosParaCertificadosV2(ctx context.Context, in personalports.ConsultaServiciosParaCertificadosV1) (personalports.ResultadoServiciosParaCertificadosV2, error) {
	c, err := l.capturar(ctx)
	if err != nil {
		return personalports.ResultadoServiciosParaCertificadosV2{}, err
	}
	return l.siguiente.ConsultarServiciosParaCertificadosV2(c, in)
}

// capturar fija persona, perfil y vínculo originales de la petición. Una
// cancelación posterior no los sustituye ni permite auditar otra identidad.
func (l lectorServiciosCertificadosConIdentidad) capturar(ctx context.Context) (context.Context, error) {
	if ctx == nil || dependenciaNula(l.siguiente) || dependenciaNula(l.identidad) || l.limite <= 0 {
		return nil, personaldomain.ErrLectorServiciosCertificadosNoDisponible
	}
	if _, err := vecports.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx); err != nil {
		return nil, personaldomain.ErrLectorServiciosCertificadosNoDisponible
	}
	auditCtx, cancelar := context.WithTimeout(context.WithoutCancel(ctx), l.limite)
	defer cancelar()
	identidad, err := l.identidad.ResolverIdentidadLectorServiciosCertificados(auditCtx)
	if err != nil || identidad.Resultado.Validar() != nil || identidad.Vinculo.ValidarPara(identidad.Resultado) != nil {
		return nil, personaldomain.ErrLectorServiciosCertificadosNoDisponible
	}
	copia, err := identidad.Resultado.Clonar()
	if err != nil {
		return nil, personaldomain.ErrLectorServiciosCertificadosNoDisponible
	}
	identidad.Resultado = copia
	ref, err := vecports.NuevaReferenciaIntentoAuditoria()
	if err != nil {
		return nil, personaldomain.ErrLectorServiciosCertificadosNoDisponible
	}
	return context.WithValue(ctx, claveIntentoLectorServiciosCertificados{}, &intentoLectorServiciosCertificados{identidad: identidad, referencia: ref}), nil
}

func estadoIntentoLectorServiciosCertificados(ctx context.Context) (*intentoLectorServiciosCertificados, error) {
	if ctx != nil {
		if intento, ok := ctx.Value(claveIntentoLectorServiciosCertificados{}).(*intentoLectorServiciosCertificados); ok && intento != nil {
			return intento, nil
		}
	}
	return nil, personaldomain.ErrLectorServiciosCertificadosNoDisponible
}

// --- Autorización: concesión V3 propia para el empleado de la persona ---

type ProveedorAutorizacionLectorServiciosCertificados struct {
	emisor EmisorMaterialLectorServiciosCertificadosV3
	motivo vecdomain.ReferenciaEntradaCatalogo
}

func NuevoProveedorAutorizacionLectorServiciosCertificados(emisor EmisorMaterialLectorServiciosCertificadosV3, motivo vecdomain.ReferenciaEntradaCatalogo) (*ProveedorAutorizacionLectorServiciosCertificados, error) {
	if dependenciaNula(emisor) || !vecdomain.ReferenciaMotivoAutorizacionV2Valida(motivo) {
		return nil, personaldomain.ErrLectorServiciosCertificadosNoDisponible
	}
	return &ProveedorAutorizacionLectorServiciosCertificados{emisor: emisor, motivo: motivo}, nil
}

func (p *ProveedorAutorizacionLectorServiciosCertificados) AutorizarServiciosParaCertificados(ctx context.Context, material personaldomain.MaterialLectorServiciosCertificados) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacio := vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	if p == nil || dependenciaNula(p.emisor) || ctx == nil || ctx.Err() != nil || len(material.Canonico()) == 0 {
		return vacio, personaldomain.ErrLectorServiciosCertificadosNoDisponible
	}
	intento, err := estadoIntentoLectorServiciosCertificados(ctx)
	if err != nil {
		return vacio, err
	}
	identidad := intento.identidad
	resultado, err := identidad.Resultado.Clonar()
	if err != nil || resultado.Validar() != nil || identidad.Vinculo.ValidarPara(resultado) != nil {
		return vacio, personaldomain.ErrLectorServiciosCertificadosNoDisponible
	}
	// El actor de la consulta es el de esta petición: otro actor no se presta.
	canonActor, err := material.Actor().RepresentacionCanonicaVinculadaV2()
	if err != nil {
		return vacio, personaldomain.ErrLectorServiciosCertificadosNoDisponible
	}
	if !bytes.Equal(canonActor, resultado.RepresentacionCanonica) {
		return vacio, personaldomain.ErrLectorServiciosCertificadosDenegado
	}
	correlacion, err := vecports.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
	if err != nil {
		return vacio, personaldomain.ErrLectorServiciosCertificadosNoDisponible
	}
	solicitud, err := vecdomain.NuevaSolicitudAutorizacionLigadaV3(vecdomain.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: identidad.Vinculo,
		ReferenciaMotivo:          p.motivo,
		Accion:                    personalports.AccionServiciosParaCertificadosV1,
		Recurso:                   material.Recurso(),
		Finalidad:                 personaldomain.FinalidadLectorServiciosCertificados,
		Correlacion:               correlacion,
	})
	if err != nil {
		return vacio, personaldomain.ErrLectorServiciosCertificadosNoDisponible
	}
	decision, confirmacion, exportador, err := p.emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, resultado)
	if err != nil {
		return vacio, clasificarErrorAutorizacionLectorServiciosCertificados(ctx, err)
	}
	if decision.ValidarPara(solicitud) != nil || dependenciaNula(exportador) {
		return vacio, personaldomain.ErrLectorServiciosCertificadosNoDisponible
	}
	autorizacion, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !vecports.MaterialAtestadoLigadoV3(solicitud, decision, confirmacion, resultado, p.motivo, autorizacion, personalports.AudienciaServiciosParaCertificadosV1) {
		return vacio, personaldomain.ErrLectorServiciosCertificadosNoDisponible
	}
	return autorizacion, nil
}

// Sólo una denegación explícita y registrada por el PDP es «denegado»; el
// resto, también una cancelación, es «no disponible».
func clasificarErrorAutorizacionLectorServiciosCertificados(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() == nil &&
		errors.Is(err, vecports.ErrDenegacionExplicitaAutorizacionLigadaV3) &&
		!errors.Is(err, vecports.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible) &&
		!errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		return personaldomain.ErrLectorServiciosCertificadosDenegado
	}
	return personaldomain.ErrLectorServiciosCertificadosNoDisponible
}

// --- Intentos fallidos en la auditoría común, tras cerrar la lectura ---

type RegistroIntentosLectorServiciosCertificados struct {
	destino       RegistradorIntentosLectorServiciosCertificados
	configuracion ConfiguracionIntentosLectorServiciosCertificados
}

func NuevoRegistroIntentosLectorServiciosCertificados(d RegistradorIntentosLectorServiciosCertificados, c ConfiguracionIntentosLectorServiciosCertificados) (*RegistroIntentosLectorServiciosCertificados, error) {
	if dependenciaNula(d) {
		return nil, personaldomain.ErrLectorServiciosCertificadosNoDisponible
	}
	for _, motivo := range []vecdomain.ReferenciaEntradaCatalogo{c.MotivoDenegado, c.MotivoEntradaInvalida, c.MotivoNoDisponible} {
		prueba := vecdomain.DatosIntentoAuditoria{Accion: personalports.AccionServiciosParaCertificadosV1, ModuloID: "personal", RecursoRef: c.RecursoEntradaInvalida, FinalidadRef: personaldomain.FinalidadLectorServiciosCertificados, Resultado: vecdomain.ResultadoIntentoAuditoriaError, Motivo: motivo, Proceso: c.Proceso, Canal: c.Canal, CorrelacionRef: "correlacion_00000000000000000000000000000000"}
		if prueba.Validar() != nil {
			return nil, personaldomain.ErrLectorServiciosCertificadosNoDisponible
		}
	}
	return &RegistroIntentosLectorServiciosCertificados{destino: d, configuracion: c}, nil
}

func (r *RegistroIntentosLectorServiciosCertificados) VerificarRegistroServiciosCertificados(ctx context.Context) error {
	if r == nil || ctx == nil || ctx.Err() != nil || dependenciaNula(r.destino) || r.destino.PreflightIntentoAuditoria(ctx) != nil {
		return personaldomain.ErrLectorServiciosCertificadosNoDisponible
	}
	return nil
}

func (r *RegistroIntentosLectorServiciosCertificados) RegistrarIntentoServiciosCertificados(ctx context.Context, in personalports.IntentoLectorServiciosCertificados) error {
	if r == nil || ctx == nil || dependenciaNula(r.destino) || (in.Motivo != "entrada_invalida" && in.Motivo != "denegado" && in.Motivo != "no_disponible") {
		return personaldomain.ErrLectorServiciosCertificadosNoDisponible
	}
	intento, err := estadoIntentoLectorServiciosCertificados(ctx)
	if err != nil {
		return personaldomain.ErrLectorServiciosCertificadosNoDisponible
	}
	correlacion, err := vecports.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
	if err != nil {
		return personaldomain.ErrLectorServiciosCertificadosNoDisponible
	}
	ref, err := correlacion.ValorCanonico()
	if err != nil {
		return personaldomain.ErrLectorServiciosCertificadosNoDisponible
	}
	motivo, resultado := r.configuracion.MotivoNoDisponible, vecdomain.ResultadoIntentoAuditoriaError
	if in.Motivo == "entrada_invalida" {
		motivo = r.configuracion.MotivoEntradaInvalida
	}
	if in.Motivo == "denegado" {
		motivo, resultado = r.configuracion.MotivoDenegado, vecdomain.ResultadoIntentoAuditoriaDenegado
	}
	// El recurso es la huella del empleado propio de la identidad capturada,
	// nunca el empleado pedido: una petición ajena no escribe su referencia.
	recurso := r.configuracion.RecursoEntradaInvalida
	if in.Motivo != "entrada_invalida" {
		empleados, err := intento.identidad.Resultado.Contexto.Referencias(vecdomain.TipoReferenciaContextoActorEmpleado)
		if err == nil && len(empleados) == 1 && personaldomain.ReferenciaEmpleadoValida(empleados[0]) && intento.identidad.Resultado.Contexto.AlcanceProyecciones().IncluyeEmpleado() {
			h := sha256.Sum256([]byte(empleados[0]))
			recurso = "personal:servicios_certificados:sha256:" + hex.EncodeToString(h[:])
		}
	}
	datos := vecdomain.DatosIntentoAuditoria{Accion: personalports.AccionServiciosParaCertificadosV1, ModuloID: "personal", RecursoRef: recurso, FinalidadRef: personaldomain.FinalidadLectorServiciosCertificados, Resultado: resultado, Motivo: motivo, Proceso: r.configuracion.Proceso, Canal: r.configuracion.Canal, CorrelacionRef: ref}
	intento.mu.Lock()
	defer intento.mu.Unlock()
	if intento.orden == nil {
		orden, err := vecports.NuevaOrdenIntentoAuditoria(intento.referencia, intento.identidad.Resultado, intento.identidad.Vinculo, datos)
		if err != nil {
			return personaldomain.ErrLectorServiciosCertificadosNoDisponible
		}
		intento.orden = &orden
	} else {
		original, err := intento.orden.Datos()
		if err != nil || original.Datos != datos {
			return personaldomain.ErrLectorServiciosCertificadosNoDisponible
		}
	}
	if intento.acuse != nil {
		return nil
	}
	// COMMIT ambiguo: se repite sólo el append común con la misma orden.
	for range 2 {
		acuse, err := r.destino.AppendIntentoAuditoria(ctx, *intento.orden)
		if err == nil && acuse.ValidarPara(*intento.orden) == nil {
			intento.acuse = &acuse
			return nil
		}
	}
	return personaldomain.ErrLectorServiciosCertificadosNoDisponible
}

var (
	_ personalports.ProveedorAutorizacionLectorServiciosCertificados = (*ProveedorAutorizacionLectorServiciosCertificados)(nil)
	_ personalports.RegistroIntentosLectorServiciosCertificados      = (*RegistroIntentosLectorServiciosCertificados)(nil)
	_ LectorServiciosCertificados                                    = lectorServiciosCertificadosConIdentidad{}
)
