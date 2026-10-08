package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"reflect"
	"sync"

	usuarioshttp "vec-diputacion-granada/internal/modules/usuarios/adapters/httpapi"
	usuariosports "vec-diputacion-granada/internal/modules/usuarios/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type registradorIntentosConsultaCorreos interface {
	vecports.RegistradorIntentosAuditoria
	PreflightIntentoAuditoria(context.Context) error
}

type registroIntentosConsultaCorreos struct {
	destino registradorIntentosConsultaCorreos
	proceso string
	motivo  core.ReferenciaEntradaCatalogo
}

func nuevoRegistroIntentosConsultaCorreos(ctx context.Context, d registradorIntentosConsultaCorreos, proceso string, motivo core.ReferenciaEntradaCatalogo) (*registroIntentosConsultaCorreos, error) {
	if ctx == nil || ctx.Err() != nil || d == nil || reflect.ValueOf(d).Kind() == reflect.Pointer && reflect.ValueOf(d).IsNil() ||
		!procesoAuditoriaIntentosConfigurado(proceso) || !core.ReferenciaMotivoAutorizacionV2Valida(motivo) || d.PreflightIntentoAuditoria(ctx) != nil {
		return nil, errComposicionUsuariosCorreos
	}
	return &registroIntentosConsultaCorreos{destino: d, proceso: proceso, motivo: motivo}, nil
}

// El abridor de producción es el loader común del proceso. Su closer solo
// posee el pool nominal que acaba de abrir; los pools de Usuarios son ajenos.
func abrirRegistroIntentosConsultaCorreos(ctx context.Context, abrir func() (vecports.RegistradorIntentosAuditoria, string, func(), error), motivo core.ReferenciaEntradaCatalogo) (*registroIntentosConsultaCorreos, func(), error) {
	if ctx == nil || ctx.Err() != nil || abrir == nil || !core.ReferenciaMotivoAutorizacionV2Valida(motivo) {
		return nil, nil, errComposicionUsuariosCorreos
	}
	d, proceso, cerrar, err := abrir()
	if cerrar != nil {
		cerrar = sync.OnceFunc(cerrar)
	}
	completa := false
	defer func() {
		if !completa && cerrar != nil {
			cerrar()
		}
	}()
	destino, ok := d.(registradorIntentosConsultaCorreos)
	if err != nil || !ok || cerrar == nil {
		return nil, nil, errComposicionUsuariosCorreos
	}
	r, err := nuevoRegistroIntentosConsultaCorreos(ctx, destino, proceso, motivo)
	if err != nil {
		return nil, nil, err
	}
	completa = true
	return r, cerrar, nil
}

type claveIntentoConsultaCorreos struct{}
type intentoConsultaCorreos struct {
	autoridad  *autoridadPreferenciasUsuariosDesarrollo
	registro   *registroIntentosConsultaCorreos
	resultado  core.ResultadoContextoActorRegistradoV2
	vinculo    core.VinculoAutenticacionActorV2
	datos      core.DatosIntentoAuditoria
	referencia string
	mu         sync.Mutex
	orden      *vecports.OrdenIntentoAuditoria
	acuse      *vecports.AcuseIntentoAuditoria
}

// La captura se hace justo después de la sesión y antes de resolver la orden
// o abrir SQL. Conserva el sello histórico incluso si caduca la lectura.
func (a *autoridadPreferenciasUsuariosDesarrollo) capturarIntentoConsultaCorreos(ctx context.Context) (context.Context, error) {
	if a == nil || ctx == nil || a.intentosConsultaCorreos == nil || a.proveedorCorreos == nil ||
		a.superficie != core.SuperficieAutenticacionInternaCorporativaV1 || a.ruta != usuarioshttp.RutaMisCorreos ||
		a.intentosConsultaCorreos.motivo != a.proveedorCorreos.motivoConsulta {
		return nil, errComposicionUsuariosCorreos
	}
	c, ok := ctx.Value(claveContextoPreferenciasUsuarios{}).(contextoPreferenciasUsuarios)
	if !ok || c.autoridad != a || c.resultado.Validar() != nil || c.vinculo.ValidarPara(c.resultado) != nil {
		return nil, errComposicionUsuariosCorreos
	}
	correlacion, err := vecports.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
	if err != nil {
		return nil, errComposicionUsuariosCorreos
	}
	refCorrelacion, err := correlacion.ValorCanonico()
	if err != nil {
		return nil, errComposicionUsuariosCorreos
	}
	resultado, err := c.resultado.Clonar()
	if err != nil {
		return nil, errComposicionUsuariosCorreos
	}
	// La referencia propia procede del mismo contexto que usa el PDP. Si su
	// alfabeto no cabe en auditoría, la huella conserva sus bytes exactos.
	datos := core.DatosIntentoAuditoria{Accion: usuariosports.AccionConsultarCorreos, ModuloID: "usuarios",
		RecursoRef: resultado.Contexto.PersonaRef, FinalidadRef: usuariosports.FinalidadCorreosPropios,
		Resultado: core.ResultadoIntentoAuditoriaError, Motivo: a.intentosConsultaCorreos.motivo,
		Proceso: a.intentosConsultaCorreos.proceso, Canal: string(a.superficie), CorrelacionRef: refCorrelacion}
	if datos.Validar() != nil {
		h := sha256.Sum256([]byte(resultado.Contexto.PersonaRef))
		datos.RecursoRef = "usuarios:correos_propios:sha256:" + hex.EncodeToString(h[:])
		if datos.Validar() != nil {
			return nil, errComposicionUsuariosCorreos
		}
	}
	referencia, err := vecports.NuevaReferenciaIntentoAuditoria()
	if err != nil {
		return nil, errComposicionUsuariosCorreos
	}
	return context.WithValue(ctx, claveIntentoConsultaCorreos{}, &intentoConsultaCorreos{autoridad: a,
		registro: a.intentosConsultaCorreos, resultado: resultado, vinculo: c.vinculo, datos: datos, referencia: referencia}), nil
}

func (a *autoridadPreferenciasUsuariosDesarrollo) intentoConsultaCorreos(ctx context.Context) (*intentoConsultaCorreos, error) {
	if a == nil || ctx == nil || a.superficie != core.SuperficieAutenticacionInternaCorporativaV1 || a.ruta != usuarioshttp.RutaMisCorreos {
		return nil, errComposicionUsuariosCorreos
	}
	c, ok := ctx.Value(claveContextoPreferenciasUsuarios{}).(contextoPreferenciasUsuarios)
	i, presente := ctx.Value(claveIntentoConsultaCorreos{}).(*intentoConsultaCorreos)
	if !ok || !presente || i == nil || i.autoridad != a || i.registro == nil || i.registro != a.intentosConsultaCorreos ||
		c.autoridad != a || c.resultado.Validar() != nil || c.vinculo.ValidarPara(c.resultado) != nil ||
		c.resultado.HuellaSHA256 != i.resultado.HuellaSHA256 || c.resultado.RegistroContextoRef != i.resultado.RegistroContextoRef {
		return nil, errComposicionUsuariosCorreos
	}
	correlacion, err := vecports.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
	if err != nil {
		return nil, errComposicionUsuariosCorreos
	}
	ref, err := correlacion.ValorCanonico()
	if err != nil || ref != i.datos.CorrelacionRef {
		return nil, errComposicionUsuariosCorreos
	}
	return i, nil
}

func (a *autoridadPreferenciasUsuariosDesarrollo) PrepararIntentoConsultaCorreos(ctx context.Context) error {
	_, err := a.intentoConsultaCorreos(ctx)
	return err
}

// El handler llama después del retorno del servicio/repositorio y antes de
// escribir la respuesta. El registrador común posee el plazo de auditoría.
func (a *autoridadPreferenciasUsuariosDesarrollo) AuditarIntentoConsultaCorreos(ctx context.Context, estado int) error {
	i, err := a.intentoConsultaCorreos(ctx)
	if err != nil {
		return err
	}
	datos := i.datos
	switch estado {
	case http.StatusUnauthorized, http.StatusForbidden:
		datos.Resultado = core.ResultadoIntentoAuditoriaDenegado
	case http.StatusConflict, http.StatusUnprocessableEntity, http.StatusTooManyRequests, http.StatusServiceUnavailable:
		datos.Resultado = core.ResultadoIntentoAuditoriaError
	default:
		return errComposicionUsuariosCorreos
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.orden == nil {
		orden, err := vecports.NuevaOrdenIntentoAuditoria(i.referencia, i.resultado, i.vinculo, datos)
		if err != nil {
			return errComposicionUsuariosCorreos
		}
		i.orden = &orden
	} else {
		original, err := i.orden.Datos()
		if err != nil || original.Datos != datos {
			return errComposicionUsuariosCorreos
		}
	}
	if i.acuse != nil {
		return nil
	}
	for range 2 {
		acuse, err := i.registro.destino.AppendIntentoAuditoria(context.WithoutCancel(ctx), *i.orden)
		if err == nil && acuse.ValidarPara(*i.orden) == nil {
			i.acuse = &acuse
			return nil
		}
	}
	return errComposicionUsuariosCorreos
}

var _ usuarioshttp.AuditorIntentosConsultaCorreos = (*autoridadPreferenciasUsuariosDesarrollo)(nil)
