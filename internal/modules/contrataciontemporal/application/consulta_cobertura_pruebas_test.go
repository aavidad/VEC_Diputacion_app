package application

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"strconv"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	organizacionCoberturaPrueba = "organizacion_diputacion_granada"
	claveHMACCoberturaPrueba    = "clave-prueba-respuesta-cobertura"
)

func claveEd25519CoberturaPrueba(etiqueta string) ed25519.PrivateKey {
	semilla := sha256.Sum256([]byte("VEC-CT-COBERTURA:" + etiqueta))
	return ed25519.NewKeyFromSeed(semilla[:])
}

type presentadorIdentificadoCoberturaPrueba interface {
	identidadCoberturaPrueba() ports.IdentidadAutoridadFuenteAnalisis
}

type autenticadorCoberturaPrueba struct {
	organizacion string
	antes        func(ports.RolAutoridadFuenteAnalisis) error
}

func (a *autenticadorCoberturaPrueba) OrganizacionAutoridadFuenteAnalisis() string {
	return a.organizacion
}

func (a *autenticadorCoberturaPrueba) AutenticarAutoridadFuenteAnalisis(
	ctx context.Context,
	presentador ports.PresentadorAutoridadFuenteAnalisis,
	material []byte,
	rol ports.RolAutoridadFuenteAnalisis,
	_ time.Time,
) (ports.IdentidadAutoridadFuenteAnalisis, error) {
	if err := ctx.Err(); err != nil {
		return ports.IdentidadAutoridadFuenteAnalisis{}, err
	}
	if len(material) == 0 {
		return ports.IdentidadAutoridadFuenteAnalisis{},
			ports.ErrResultadoFuenteAnalisisNoConfiable
	}
	if a.antes != nil {
		if err := a.antes(rol); err != nil {
			return ports.IdentidadAutoridadFuenteAnalisis{}, err
		}
	}
	identificado, ok := presentador.(presentadorIdentificadoCoberturaPrueba)
	if !ok {
		return ports.IdentidadAutoridadFuenteAnalisis{},
			ports.ErrResultadoFuenteAnalisisNoConfiable
	}
	identidad := identificado.identidadCoberturaPrueba()
	if identidad.Rol() != rol {
		return ports.IdentidadAutoridadFuenteAnalisis{},
			ports.ErrResultadoFuenteAnalisisNoConfiable
	}
	return identidad, nil
}

type fuenteCoberturaAplicacionPrueba struct {
	identidad ports.IdentidadAutoridadFuenteAnalisis
	consultar func(
		context.Context,
		ports.SolicitudConsultarCobertura,
	) (ports.ResultadoConsultaCobertura, error)
}

func (f *fuenteCoberturaAplicacionPrueba) identidadCoberturaPrueba() ports.IdentidadAutoridadFuenteAnalisis {
	return f.identidad
}
func (*fuenteCoberturaAplicacionPrueba) PresentarAutoridadFuenteAnalisis(
	context.Context,
	ports.DesafioAutoridadFuenteAnalisis,
) (ports.PresentacionAutoridadFuenteAnalisis, error) {
	return ports.PresentacionAutoridadFuenteAnalisis{}, nil
}
func (f *fuenteCoberturaAplicacionPrueba) ConsultarCobertura(
	ctx context.Context,
	solicitud ports.SolicitudConsultarCobertura,
) (ports.ResultadoConsultaCobertura, error) {
	return f.consultar(ctx, solicitud)
}

type verificadorCoberturaAplicacionPrueba struct {
	identidad ports.IdentidadAutoridadFuenteAnalisis
	verificar func(
		context.Context,
		ports.SolicitudVerificarRespuestaCobertura,
	) (ports.ConfirmacionRespuestaCobertura, error)
}

func (v *verificadorCoberturaAplicacionPrueba) identidadCoberturaPrueba() ports.IdentidadAutoridadFuenteAnalisis {
	return v.identidad
}
func (*verificadorCoberturaAplicacionPrueba) PresentarAutoridadFuenteAnalisis(
	context.Context,
	ports.DesafioAutoridadFuenteAnalisis,
) (ports.PresentacionAutoridadFuenteAnalisis, error) {
	return ports.PresentacionAutoridadFuenteAnalisis{}, nil
}
func (v *verificadorCoberturaAplicacionPrueba) VerificarRespuestaCobertura(
	ctx context.Context,
	solicitud ports.SolicitudVerificarRespuestaCobertura,
) (ports.ConfirmacionRespuestaCobertura, error) {
	return v.verificar(ctx, solicitud)
}

type publicadorCoberturaAplicacionPrueba struct {
	identidad ports.IdentidadAutoridadFuenteAnalisis
	publicar  func(
		context.Context,
		ports.SolicitudConsultarCobertura,
	) (ports.ConfirmacionPublicacionCobertura, error)
}

func (p *publicadorCoberturaAplicacionPrueba) identidadCoberturaPrueba() ports.IdentidadAutoridadFuenteAnalisis {
	return p.identidad
}
func (*publicadorCoberturaAplicacionPrueba) PresentarAutoridadFuenteAnalisis(
	context.Context,
	ports.DesafioAutoridadFuenteAnalisis,
) (ports.PresentacionAutoridadFuenteAnalisis, error) {
	return ports.PresentacionAutoridadFuenteAnalisis{}, nil
}
func (p *publicadorCoberturaAplicacionPrueba) ConsultarPublicacionCobertura(
	ctx context.Context,
	solicitud ports.SolicitudConsultarCobertura,
) (ports.ConfirmacionPublicacionCobertura, error) {
	return p.publicar(ctx, solicitud)
}

type relojCoberturaAplicacionPrueba struct {
	mu    sync.RWMutex
	ahora time.Time
}

func (r *relojCoberturaAplicacionPrueba) Ahora() time.Time {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.ahora
}
func (r *relojCoberturaAplicacionPrueba) fijar(ahora time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ahora = ahora
}

type registroConsumoCoberturaAplicacionPrueba struct {
	huella string
	recibo ports.ReciboConsumoCobertura
}

type consumidorCoberturaAplicacionPrueba struct {
	mu        sync.Mutex
	reloj     *relojCoberturaAplicacionPrueba
	registros map[string]registroConsumoCoberturaAplicacionPrueba
	ordenes   []ports.OrdenConsumoCobertura
	consumir  func(
		context.Context,
		ports.OrdenConsumoCobertura,
	) (ports.ReciboConsumoCobertura, error)
}

func (c *consumidorCoberturaAplicacionPrueba) ConsumirCobertura(
	ctx context.Context,
	orden ports.OrdenConsumoCobertura,
) (ports.ReciboConsumoCobertura, error) {
	if c.consumir != nil {
		return c.consumir(ctx, orden)
	}
	datos, err := orden.Datos()
	if err != nil {
		return ports.ReciboConsumoCobertura{}, err
	}
	clave := datos.AutoridadRef + ":" +
		strconv.FormatUint(uint64(datos.Generacion), 10) + ":" +
		datos.ReciboRespuestaRef
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ordenes = append(c.ordenes, orden)
	if c.registros == nil {
		c.registros = make(
			map[string]registroConsumoCoberturaAplicacionPrueba,
		)
	}
	if anterior, existe := c.registros[clave]; existe {
		if anterior.huella != datos.HuellaRespuestaSHA256 {
			return ports.ReciboConsumoCobertura{},
				ports.ErrRespuestaCoberturaYaConsumida
		}
		return anterior.recibo, nil
	}
	recibo, err := ports.NuevoReciboConsumoCobertura(
		orden,
		"consumo_cobertura_0123456789",
		c.reloj.Ahora(),
	)
	if err != nil {
		return ports.ReciboConsumoCobertura{}, err
	}
	c.registros[clave] = registroConsumoCoberturaAplicacionPrueba{
		huella: datos.HuellaRespuestaSHA256,
		recibo: recibo,
	}
	return recibo, nil
}

type entornoCoberturaAplicacionPrueba struct {
	inicio        time.Time
	solicitud     ports.SolicitudConsultarCobertura
	catalogo      domain.CatalogoViasCobertura
	fuente        *fuenteCoberturaAplicacionPrueba
	verificador   *verificadorCoberturaAplicacionPrueba
	publicador    *publicadorCoberturaAplicacionPrueba
	consumidor    *consumidorCoberturaAplicacionPrueba
	autenticador  *autenticadorCoberturaPrueba
	reloj         *relojCoberturaAplicacionPrueba
	servicio      *ServicioConsultaCobertura
	claveVerifica ed25519.PrivateKey
}

func nuevoEntornoCoberturaAplicacionPrueba(
	t *testing.T,
) *entornoCoberturaAplicacionPrueba {
	t.Helper()
	inicio := time.Date(2026, 7, 23, 10, 0, 0, 0, time.UTC)
	solicitud, catalogo := solicitudCatalogoCoberturaAplicacionPrueba(
		t,
		inicio,
	)
	reloj := &relojCoberturaAplicacionPrueba{
		ahora: inicio.Add(2 * time.Second),
	}
	claveFuente := claveEd25519CoberturaPrueba("fuente")
	claveVerifica := claveEd25519CoberturaPrueba("verificador")
	clavePublica := claveEd25519CoberturaPrueba("publicador")
	fuente := &fuenteCoberturaAplicacionPrueba{
		identidad: identidadCoberturaAplicacionPrueba(
			t,
			"fuente_cobertura_bolsa_012345",
			solicitud.Comprobacion.Procedencia.DefinicionFuenteRef,
			claveFuente,
			ports.RolFuenteCobertura,
		),
	}
	verificador := &verificadorCoberturaAplicacionPrueba{
		identidad: identidadCoberturaAplicacionPrueba(
			t,
			"verificador_cobertura_tcb_012345",
			"backend_verificador_cobertura_01",
			claveVerifica,
			ports.RolVerificadorCobertura,
		),
	}
	publicador := &publicadorCoberturaAplicacionPrueba{
		identidad: identidadCoberturaAplicacionPrueba(
			t,
			"publicador_catalogo_cobertura_01",
			"backend_publicador_cobertura_01",
			clavePublica,
			ports.RolPublicadorCatalogoCobertura,
		),
	}
	entorno := &entornoCoberturaAplicacionPrueba{
		inicio: inicio, solicitud: solicitud, catalogo: catalogo,
		fuente: fuente, verificador: verificador, publicador: publicador,
		autenticador: &autenticadorCoberturaPrueba{
			organizacion: organizacionCoberturaPrueba,
		},
		reloj: reloj, claveVerifica: claveVerifica,
	}
	entorno.consumidor = &consumidorCoberturaAplicacionPrueba{reloj: reloj}
	fuente.consultar = func(
		_ context.Context,
		solicitud ports.SolicitudConsultarCobertura,
	) (ports.ResultadoConsultaCobertura, error) {
		return resultadoCoberturaAplicacionPrueba(t, solicitud, nil), nil
	}
	verificador.verificar = func(
		_ context.Context,
		solicitud ports.SolicitudVerificarRespuestaCobertura,
	) (ports.ConfirmacionRespuestaCobertura, error) {
		return verificarRespuestaCoberturaAplicacionPrueba(
			solicitud,
			verificador.identidad.AutoridadRef(),
			claveVerifica,
			reloj.Ahora(),
		)
	}
	publicador.publicar = func(
		context.Context,
		ports.SolicitudConsultarCobertura,
	) (ports.ConfirmacionPublicacionCobertura, error) {
		return ports.NuevaConfirmacionPublicacionCobertura(
			publicador.identidad.AutoridadRef(),
			catalogo.Publicacion(),
			reloj.Ahora(),
		)
	}
	entorno.reconstruirServicio(t, time.Second)
	return entorno
}

func (e *entornoCoberturaAplicacionPrueba) reconstruirServicio(
	t *testing.T,
	tiempoMaximo time.Duration,
) {
	t.Helper()
	servicio, err := NuevoServicioConsultaCobertura(
		e.fuente,
		e.verificador,
		e.publicador,
		e.consumidor,
		e.autenticador,
		e.reloj,
		tiempoMaximo,
	)
	if err != nil {
		t.Fatal(err)
	}
	e.servicio = servicio
}
