package bootstrap

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"sync"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application/diagnostico"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// El límite es deliberadamente pequeño: este puente sólo conserva la
// continuidad efímera entre las dos peticiones de una página RRHH.
const limiteContinuidadesCursorRRHHDesarrollo = 64

type claveContinuidadCursorRRHHDesarrollo struct{}

type continuidadCursorRRHHDesarrollo struct {
	dueno           *continuadorSesionCursorRRHHDesarrollo
	contexto        ports.ContextoAutorizacionAltaV3
	certificado     string
	vinculoCanalTLS [sha256.Size]byte
	validaHasta     time.Time
}

type continuadorSesionCursorRRHHDesarrollo struct {
	autoridad *autoridadConsultasRRHHDesarrollo
	reloj     ports.Reloj
	mu        sync.Mutex
	entradas  map[[sha256.Size]byte]continuidadCursorRRHHDesarrollo
}

type consultorCuadroConSesionCursorRRHHDesarrollo struct {
	delegado    httpinterno.ConsultorCuadroRRHH
	continuador *continuadorSesionCursorRRHHDesarrollo
}

func falloContinuidadCursorRRHHDesarrollo(etapa diagnostico.EtapaConsultaRRHH, causa error) error {
	return &diagnostico.FalloConsultaRRHH{Etapa: etapa, Sentinela: ports.ErrAutorizacionDenegada, Causa: causa}
}

func nuevoContinuadorSesionCursorRRHHDesarrollo(
	autoridad *autoridadConsultasRRHHDesarrollo, reloj ports.Reloj,
) (*continuadorSesionCursorRRHHDesarrollo, error) {
	if autoridad == nil || dependenciaEsNulaContratacionTemporalDesarrollo(reloj) {
		return nil, ports.ErrConsultaRRHHNoDisponible
	}
	return &continuadorSesionCursorRRHHDesarrollo{
		autoridad: autoridad, reloj: reloj,
		entradas: make(map[[sha256.Size]byte]continuidadCursorRRHHDesarrollo),
	}, nil
}

func nuevoConsultorCuadroConSesionCursorRRHHDesarrollo(
	delegado httpinterno.ConsultorCuadroRRHH, continuador *continuadorSesionCursorRRHHDesarrollo,
) (httpinterno.ConsultorCuadroRRHH, error) {
	if dependenciaEsNulaContratacionTemporalDesarrollo(delegado) || continuador == nil {
		return nil, ports.ErrConsultaRRHHNoDisponible
	}
	return &consultorCuadroConSesionCursorRRHHDesarrollo{delegado: delegado, continuador: continuador}, nil
}

func (c *consultorCuadroConSesionCursorRRHHDesarrollo) Consultar(
	ctx context.Context, solicitud ports.SolicitudCuadroRRHH,
) (ports.PaginaCuadroRRHH, error) {
	if c == nil || dependenciaEsNulaContratacionTemporalDesarrollo(c.delegado) || c.continuador == nil {
		return ports.PaginaCuadroRRHH{}, ports.ErrConsultaRRHHNoDisponible
	}
	preparado, err := c.continuador.preparar(ctx, solicitud.Cursor())
	if err != nil {
		return ports.PaginaCuadroRRHH{}, err
	}
	pagina, err := c.delegado.Consultar(preparado, solicitud)
	if err != nil {
		return ports.PaginaCuadroRRHH{}, err
	}
	if err := c.continuador.recordar(preparado, pagina); err != nil {
		return ports.PaginaCuadroRRHH{}, err
	}
	return pagina, nil
}

func (c *continuadorSesionCursorRRHHDesarrollo) preparar(ctx context.Context, cursor string) (context.Context, error) {
	if c == nil || ctx == nil || ctx.Err() != nil {
		return nil, falloContinuidadCursorRRHHDesarrollo(diagnostico.EtapaContinuidadCursor, nil)
	}
	if cursor == "" {
		return ctx, nil
	}
	clave, ok := huellaCursorSesionRRHHDesarrollo(cursor)
	if !ok {
		return nil, falloContinuidadCursorRRHHDesarrollo(diagnostico.EtapaContinuidadCursor, nil)
	}
	ahora := c.reloj.Ahora()
	if !domain.InstanteUTCCanonico(ahora) {
		return nil, falloContinuidadCursorRRHHDesarrollo(diagnostico.EtapaContinuidadCursor, nil)
	}
	c.mu.Lock()
	c.limpiarCaducadasBloqueado(ahora)
	entrada, existe := c.entradas[clave]
	// La reserva ocurre antes de delegar: dos peticiones concurrentes nunca
	// usan la misma continuidad. Un fallo posterior queda cerrado de forma
	// conservadora; el consumo durable sigue perteneciendo a PostgreSQL.
	if existe {
		delete(c.entradas, clave)
	}
	c.mu.Unlock()
	if !existe || !ahora.Before(entrada.validaHasta) {
		return nil, falloContinuidadCursorRRHHDesarrollo(diagnostico.EtapaContinuidadCursor, nil)
	}
	return context.WithValue(ctx, claveContinuidadCursorRRHHDesarrollo{}, entrada), nil
}

func (c *continuadorSesionCursorRRHHDesarrollo) recordar(ctx context.Context, pagina ports.PaginaCuadroRRHH) error {
	if c == nil || c.autoridad == nil || c.autoridad.soporte == nil || ctx == nil || !pagina.HayMas || pagina.CursorSiguiente == "" {
		return nil
	}
	clave, ok := huellaCursorSesionRRHHDesarrollo(pagina.CursorSiguiente)
	if !ok {
		return ports.ErrConsultaRRHHNoDisponible
	}
	capacidad, valida := c.autoridad.soporte.capacidadValida(ctx)
	if !valida || capacidad.consultaRRHH == nil || capacidad.ruta != httpinterno.RutaConsultaCuadroRRHH ||
		capacidad.principal.Attributes["certificate_sha256"] == "" {
		return ports.ErrAutorizacionDenegada
	}
	peticion := capacidad.consultaRRHH
	peticion.mu.Lock()
	contexto, err := clonarContextoSesionCursorRRHHDesarrollo(peticion.contexto)
	peticion.mu.Unlock()
	if err != nil || peticion.autoridad != c.autoridad || !c.autoridad.contextoConsultaRRHHConservaActor(contexto) {
		return ports.ErrAutorizacionDenegada
	}
	datos, err := contexto.Vinculo.Datos()
	ahora := c.reloj.Ahora()
	if err != nil || !domain.InstanteUTCCanonico(ahora) || !ahora.Before(datos.SesionValidaHasta) ||
		!ahora.Before(capacidad.certificadoValidoHasta) {
		return ports.ErrAutorizacionDenegada
	}
	validaHasta := datos.SesionValidaHasta
	if capacidad.certificadoValidoHasta.Before(validaHasta) {
		validaHasta = capacidad.certificadoValidoHasta
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.limpiarCaducadasBloqueado(ahora)
	if len(c.entradas) >= limiteContinuidadesCursorRRHHDesarrollo {
		c.expulsarMasProximaBloqueado()
	}
	if capacidad.vinculoCanalTLS == ([sha256.Size]byte{}) {
		return ports.ErrAutorizacionDenegada
	}
	c.entradas[clave] = continuidadCursorRRHHDesarrollo{dueno: c, contexto: contexto,
		certificado: capacidad.principal.Attributes["certificate_sha256"], vinculoCanalTLS: capacidad.vinculoCanalTLS, validaHasta: validaHasta}
	return nil
}

func (c *continuadorSesionCursorRRHHDesarrollo) contextoContinuado(ctx context.Context) (ports.ContextoAutorizacionAltaV3, error) {
	if c == nil || ctx == nil {
		return ports.ContextoAutorizacionAltaV3{}, falloContinuidadCursorRRHHDesarrollo(diagnostico.EtapaContinuidadCursor, nil)
	}
	entrada, ok := ctx.Value(claveContinuidadCursorRRHHDesarrollo{}).(continuidadCursorRRHHDesarrollo)
	if !ok || entrada.dueno != c {
		return ports.ContextoAutorizacionAltaV3{}, falloContinuidadCursorRRHHDesarrollo(diagnostico.EtapaContinuidadCursor, nil)
	}
	if !c.continuidadCanalValida(ctx, entrada) {
		return ports.ContextoAutorizacionAltaV3{}, falloContinuidadCursorRRHHDesarrollo(diagnostico.EtapaCanalContinuidad, nil)
	}
	c.autoridad.mu.Lock()
	proveedor, ok := c.autoridad.proveedor.(*proveedorSesionConsultaRRHHDesarrollo)
	c.autoridad.mu.Unlock()
	if !ok || proveedor == nil {
		return ports.ContextoAutorizacionAltaV3{}, falloContinuidadCursorRRHHDesarrollo(diagnostico.EtapaSesionRevalidada, nil)
	}
	contexto, err := proveedor.revalidarSesionCursorRRHHDesarrollo(ctx, entrada.contexto)
	if err != nil {
		var fallo *diagnostico.FalloConsultaRRHH
		if errors.As(err, &fallo) {
			return ports.ContextoAutorizacionAltaV3{}, err
		}
		return ports.ContextoAutorizacionAltaV3{}, falloContinuidadCursorRRHHDesarrollo(diagnostico.EtapaSesionRevalidada, err)
	}
	if !c.autoridad.contextoConsultaRRHHConservaActor(contexto) {
		return ports.ContextoAutorizacionAltaV3{}, falloContinuidadCursorRRHHDesarrollo(diagnostico.EtapaActorContexto, nil)
	}
	return contexto, nil
}

func (c *continuadorSesionCursorRRHHDesarrollo) continuidadCanalValida(ctx context.Context, entrada continuidadCursorRRHHDesarrollo) bool {
	if c == nil || c.autoridad == nil || c.autoridad.soporte == nil ||
		!domain.InstanteUTCCanonico(c.reloj.Ahora()) || !c.reloj.Ahora().Before(entrada.validaHasta) {
		return false
	}
	capacidad, ok := c.autoridad.soporte.capacidadValida(ctx)
	return ok && capacidad.ruta == httpinterno.RutaConsultaCuadroRRHH && capacidad.consultaRRHH != nil &&
		capacidad.principal.Attributes["certificate_sha256"] == entrada.certificado &&
		capacidad.vinculoCanalTLS == entrada.vinculoCanalTLS &&
		capacidad.vinculoCanalTLS != ([sha256.Size]byte{}) &&
		c.reloj.Ahora().Before(capacidad.certificadoValidoHasta)
}

func vinculoCanalTLSCursorRRHHDesarrollo(estado *tls.ConnectionState) (vinculo [sha256.Size]byte, valido bool) {
	if estado == nil || !estado.HandshakeComplete {
		return vinculo, false
	}
	defer func() {
		if recover() != nil {
			vinculo, valido = [sha256.Size]byte{}, false
		}
	}()
	material, err := estado.ExportKeyingMaterial(
		"VEC-Diputacion-Consulta-RRHH-Cursor-Sesion-v1", []byte("contratacion-temporal"), sha256.Size,
	)
	if err != nil || len(material) != sha256.Size {
		return vinculo, false
	}
	defer clear(material)
	return sha256.Sum256(material), true
}

func (c *continuadorSesionCursorRRHHDesarrollo) limpiarCaducadasBloqueado(ahora time.Time) {
	for clave, entrada := range c.entradas {
		if !ahora.Before(entrada.validaHasta) {
			delete(c.entradas, clave)
		}
	}
}

func (c *continuadorSesionCursorRRHHDesarrollo) expulsarMasProximaBloqueado() {
	var clave [sha256.Size]byte
	var hasta time.Time
	for candidata, entrada := range c.entradas {
		if hasta.IsZero() || entrada.validaHasta.Before(hasta) {
			clave, hasta = candidata, entrada.validaHasta
		}
	}
	if !hasta.IsZero() {
		delete(c.entradas, clave)
	}
}

func huellaCursorSesionRRHHDesarrollo(cursor string) ([sha256.Size]byte, bool) {
	material, err := base64.RawURLEncoding.Strict().DecodeString(cursor)
	if err != nil || len(material) != sha256.Size || base64.RawURLEncoding.EncodeToString(material) != cursor {
		return [sha256.Size]byte{}, false
	}
	defer clear(material)
	return sha256.Sum256(material), true
}

func clonarContextoSesionCursorRRHHDesarrollo(origen ports.ContextoAutorizacionAltaV3) (ports.ContextoAutorizacionAltaV3, error) {
	resultado, err := origen.Resultado.Clonar()
	if err != nil {
		return ports.ContextoAutorizacionAltaV3{}, err
	}
	return ports.ContextoAutorizacionAltaV3{Vinculo: origen.Vinculo, Resultado: resultado}, nil
}

// revalidarSesionCursorRRHHDesarrollo conserva la sesión acreditada y genera
// un vínculo fresco. No registra una sesión ni reutiliza una decisión previa.
func (p *proveedorSesionConsultaRRHHDesarrollo) revalidarSesionCursorRRHHDesarrollo(
	ctx context.Context, anterior ports.ContextoAutorizacionAltaV3,
) (ports.ContextoAutorizacionAltaV3, error) {
	vacio := ports.ContextoAutorizacionAltaV3{}
	if p == nil || p.soporte == nil || dependenciaEsNulaContratacionTemporalDesarrollo(p.revalidador) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(p.resolutor) || dependenciaEsNulaContratacionTemporalDesarrollo(p.reloj) {
		return vacio, falloContinuidadCursorRRHHDesarrollo(diagnostico.EtapaSesionPrecondicion, nil)
	}
	canal, ok := p.soporte.capacidadValida(ctx)
	ahora := p.reloj.Ahora()
	if !ok || !rutaConsultaRRHHContratacionTemporalDesarrollo(canal.ruta) || !domain.InstanteUTCCanonico(ahora) ||
		!ahora.Before(canal.certificadoValidoHasta) {
		return vacio, falloContinuidadCursorRRHHDesarrollo(diagnostico.EtapaSesionPrecondicion, nil)
	}
	datos, err := anterior.Vinculo.Datos()
	if err != nil || anterior.ValidarPara(ports.SolicitudResolverContextoAutorizacionAltaV3{
		AutenticacionRef: datos.AutenticacionRef, SesionRef: datos.SesionRef, PerfilRef: p.base.Contexto.PerfilActivoRef,
	}, ahora) != nil {
		return vacio, falloContinuidadCursorRRHHDesarrollo(diagnostico.EtapaSesionVinculoPrevio, err)
	}
	vinculo, resultado, err := dominiovec.CrearVinculoAutenticacionActorV2ConResultado(ctx, p.revalidador,
		dominiovec.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: datos.AutenticacionRef, SesionRef: datos.SesionRef},
		p.resolutor, dominiovec.SolicitudContextoActor{Cuenta: dominiovec.CuentaAutenticadaContextoActor{
			CuentaRef: p.base.Contexto.Instantanea.CuentaRef, Metodo: dominiovec.AuthMethodCertificate, Garantia: dominiovec.AuthAssuranceHigh,
		}, PerfilActivoRef: p.base.Contexto.PerfilActivoRef}, p.reloj)
	if err != nil {
		var fallo *diagnostico.FalloConsultaRRHH
		if errors.As(err, &fallo) {
			return vacio, err
		}
		etapa := diagnostico.EtapaSesionVinculoCreacion
		if errors.Is(err, dominiovec.ErrAutenticacionRevalidadaInvalida) {
			etapa = diagnostico.EtapaSesionRevalidador
		}
		return vacio, falloContinuidadCursorRRHHDesarrollo(etapa, err)
	}
	if !mismaIdentidadVersionadaSesionDesarrollo(p.base, resultado) {
		return vacio, falloContinuidadCursorRRHHDesarrollo(diagnostico.EtapaSesionIdentidadDistinta, nil)
	}
	nuevosDatos, err := vinculo.Datos()
	if err != nil || nuevosDatos.AutenticacionRef != datos.AutenticacionRef || nuevosDatos.SesionRef != datos.SesionRef {
		return vacio, falloContinuidadCursorRRHHDesarrollo(diagnostico.EtapaSesionVinculoIncoherente, err)
	}
	ahoraValidacion := p.reloj.Ahora()
	contexto := ports.ContextoAutorizacionAltaV3{Vinculo: vinculo, Resultado: resultado}
	if contexto.ValidarPara(ports.SolicitudResolverContextoAutorizacionAltaV3{
		AutenticacionRef: datos.AutenticacionRef, SesionRef: datos.SesionRef, PerfilRef: p.base.Contexto.PerfilActivoRef,
	}, ahoraValidacion) != nil {
		return vacio, falloContinuidadCursorRRHHDesarrollo(diagnostico.EtapaSesionContextoInvalido, nil)
	}
	return contexto, nil
}
