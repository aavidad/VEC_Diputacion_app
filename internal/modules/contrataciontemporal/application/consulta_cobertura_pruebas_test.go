package application

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"fmt"
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

type autenticadorCoberturaPrueba struct {
	organizacion string
	audiencia    string
	identidades  map[ports.RolAutoridadFuenteAnalisis]ports.IdentidadAutoridadFuenteAnalisis
	antes        func(ports.RolAutoridadFuenteAnalisis, time.Time) error
	resolver     func(
		ports.EvidenciaPublicaAutoridadFuenteAnalisis,
	) (ports.IdentidadAutoridadFuenteAnalisis, error)
}

func (a *autenticadorCoberturaPrueba) OrganizacionAutoridadFuenteAnalisis() string {
	return a.organizacion
}

func (a *autenticadorCoberturaPrueba) AudienciaAutoridadFuenteAnalisis() string {
	return a.audiencia
}

func (a *autenticadorCoberturaPrueba) VerificarEvidenciaPublicaAutoridadFuenteAnalisis(
	evidencia ports.EvidenciaPublicaAutoridadFuenteAnalisis,
) (ports.IdentidadAutoridadFuenteAnalisis, error) {
	if a == nil {
		return ports.IdentidadAutoridadFuenteAnalisis{},
			ports.ErrResultadoFuenteAnalisisNoConfiable
	}
	_, _, rol, comprobadaEn, err := evidencia.Datos()
	if err != nil {
		return ports.IdentidadAutoridadFuenteAnalisis{},
			ports.ErrResultadoFuenteAnalisisNoConfiable
	}
	if a.antes != nil {
		if err := a.antes(rol, comprobadaEn); err != nil {
			return ports.IdentidadAutoridadFuenteAnalisis{}, err
		}
	}
	if a.resolver != nil {
		return a.resolver(evidencia)
	}
	identidad, ok := a.identidades[rol]
	if !ok || identidad.Rol() != rol {
		return ports.IdentidadAutoridadFuenteAnalisis{},
			ports.ErrResultadoFuenteAnalisisNoConfiable
	}
	return identidad, nil
}

type fuenteCoberturaAplicacionPrueba struct {
	identidad ports.IdentidadAutoridadFuenteAnalisis
	presentar func(
		context.Context,
		ports.DesafioAutoridadFuenteAnalisis,
	) (ports.PresentacionAutoridadFuenteAnalisis, error)
	consultar func(
		context.Context,
		ports.SolicitudConsultarCobertura,
	) (ports.ResultadoConsultaCobertura, error)
}

func (f *fuenteCoberturaAplicacionPrueba) identidadCoberturaPrueba() ports.IdentidadAutoridadFuenteAnalisis {
	return f.identidad
}
func (f *fuenteCoberturaAplicacionPrueba) PresentarAutoridadFuenteAnalisis(
	ctx context.Context,
	desafio ports.DesafioAutoridadFuenteAnalisis,
) (ports.PresentacionAutoridadFuenteAnalisis, error) {
	if f.presentar != nil {
		return f.presentar(ctx, desafio)
	}
	return presentacionEstructuralCoberturaPrueba(f.identidad)
}
func (f *fuenteCoberturaAplicacionPrueba) ConsultarCobertura(
	ctx context.Context,
	solicitud ports.SolicitudConsultarCobertura,
) (ports.ResultadoConsultaCobertura, error) {
	return f.consultar(ctx, solicitud)
}

type verificadorCoberturaAplicacionPrueba struct {
	identidad ports.IdentidadAutoridadFuenteAnalisis
	presentar func(
		context.Context,
		ports.DesafioAutoridadFuenteAnalisis,
	) (ports.PresentacionAutoridadFuenteAnalisis, error)
	verificar func(
		context.Context,
		ports.SolicitudVerificarRespuestaCobertura,
	) (ports.ConfirmacionRespuestaCobertura, error)
}

func (v *verificadorCoberturaAplicacionPrueba) identidadCoberturaPrueba() ports.IdentidadAutoridadFuenteAnalisis {
	return v.identidad
}
func (v *verificadorCoberturaAplicacionPrueba) PresentarAutoridadFuenteAnalisis(
	ctx context.Context,
	desafio ports.DesafioAutoridadFuenteAnalisis,
) (ports.PresentacionAutoridadFuenteAnalisis, error) {
	if v.presentar != nil {
		return v.presentar(ctx, desafio)
	}
	return presentacionEstructuralCoberturaPrueba(v.identidad)
}
func (v *verificadorCoberturaAplicacionPrueba) VerificarRespuestaCobertura(
	ctx context.Context,
	solicitud ports.SolicitudVerificarRespuestaCobertura,
) (ports.ConfirmacionRespuestaCobertura, error) {
	return v.verificar(ctx, solicitud)
}

type publicadorCoberturaAplicacionPrueba struct {
	identidad ports.IdentidadAutoridadFuenteAnalisis
	presentar func(
		context.Context,
		ports.DesafioAutoridadFuenteAnalisis,
	) (ports.PresentacionAutoridadFuenteAnalisis, error)
	publicar func(
		context.Context,
		ports.SolicitudConsultarCobertura,
	) (ports.ConfirmacionPublicacionCobertura, error)
}

func (p *publicadorCoberturaAplicacionPrueba) identidadCoberturaPrueba() ports.IdentidadAutoridadFuenteAnalisis {
	return p.identidad
}
func (p *publicadorCoberturaAplicacionPrueba) PresentarAutoridadFuenteAnalisis(
	ctx context.Context,
	desafio ports.DesafioAutoridadFuenteAnalisis,
) (ports.PresentacionAutoridadFuenteAnalisis, error) {
	if p.presentar != nil {
		return p.presentar(ctx, desafio)
	}
	return presentacionEstructuralCoberturaPrueba(p.identidad)
}
func (p *publicadorCoberturaAplicacionPrueba) ConsultarPublicacionCobertura(
	ctx context.Context,
	solicitud ports.SolicitudConsultarCobertura,
) (ports.ConfirmacionPublicacionCobertura, error) {
	return p.publicar(ctx, solicitud)
}

func presentacionEstructuralCoberturaPrueba(
	identidad ports.IdentidadAutoridadFuenteAnalisis,
) (ports.PresentacionAutoridadFuenteAnalisis, error) {
	credencial, err := ports.NuevaCredencialAutoridadFuenteAnalisis(
		ports.DatosCredencialAutoridadFuenteAnalisis{
			RaizClaveID:        "raiz_estructural_cobertura_prueba_01",
			AutoridadRef:       identidad.AutoridadRef(),
			BackendRef:         identidad.BackendRef(),
			OrganizacionRef:    organizacionCoberturaPrueba,
			Audiencia:          "servicio_contratacion_temporal",
			Rol:                identidad.Rol(),
			Serie:              1,
			Generacion:         1,
			ClavePruebaEd25519: identidad.ClavePruebaEd25519(),
			EmitidaEn: time.Date(
				2026, 1, 1, 0, 0, 0, 0, time.UTC,
			),
			ValidaHasta: time.Date(
				2027, 1, 1, 0, 0, 0, 0, time.UTC,
			),
		},
		make([]byte, ed25519.SignatureSize),
	)
	if err != nil {
		return ports.PresentacionAutoridadFuenteAnalisis{}, err
	}
	return ports.NuevaPresentacionAutoridadFuenteAnalisis(
		credencial,
		make([]byte, ed25519.SignatureSize),
	)
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
	huellaPeticion  string
	huellaResultado string
	recibo          ports.ReciboConsumoCobertura
}

type evidenciaConsumoCoberturaAplicacionPrueba struct {
	claveEfecto     string
	huellaRespuesta string
}

type consumidorCoberturaAplicacionPrueba struct {
	mu         sync.Mutex
	reloj      *relojCoberturaAplicacionPrueba
	registros  map[string]registroConsumoCoberturaAplicacionPrueba
	evidencias map[string]evidenciaConsumoCoberturaAplicacionPrueba
	ordenes    []ports.OrdenConsumoCobertura
	despues    func(context.Context)
	responder  func(
		ports.ReciboConsumoCobertura,
	) (ports.ReciboConsumoCobertura, error)
	consumir func(
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
	claveEfecto := datos.OrganizacionRef + ":" + datos.PeticionRef
	claveEvidencia := datos.AutoridadRef + ":" +
		fmt.Sprint(datos.Generacion) + ":" +
		datos.ReciboRespuestaRef
	c.mu.Lock()
	c.ordenes = append(c.ordenes, orden)
	if c.registros == nil {
		c.registros = make(
			map[string]registroConsumoCoberturaAplicacionPrueba,
		)
	}
	if c.evidencias == nil {
		c.evidencias = make(
			map[string]evidenciaConsumoCoberturaAplicacionPrueba,
		)
	}
	evidencia, evidenciaExiste := c.evidencias[claveEvidencia]
	if evidenciaExiste &&
		(evidencia.claveEfecto != claveEfecto ||
			evidencia.huellaRespuesta != datos.HuellaRespuestaSHA256) {
		c.mu.Unlock()
		return ports.ReciboConsumoCobertura{},
			ports.ErrRespuestaCoberturaYaConsumida
	}
	if anterior, existe := c.registros[claveEfecto]; existe {
		if anterior.huellaPeticion != datos.HuellaPeticionSHA256 ||
			anterior.huellaResultado != datos.HuellaResultadoSHA256 {
			c.mu.Unlock()
			return ports.ReciboConsumoCobertura{},
				ports.ErrRespuestaCoberturaYaConsumida
		}
		if !evidenciaExiste {
			c.evidencias[claveEvidencia] =
				evidenciaConsumoCoberturaAplicacionPrueba{
					claveEfecto:     claveEfecto,
					huellaRespuesta: datos.HuellaRespuestaSHA256,
				}
		}
		recibo := anterior.recibo
		c.mu.Unlock()
		if c.despues != nil {
			c.despues(ctx)
		}
		if c.responder != nil {
			return c.responder(recibo)
		}
		return recibo, nil
	}
	recibo, err := ports.NuevoReciboConsumoCobertura(
		orden,
		"consumo_cobertura_0123456789",
		c.reloj.Ahora(),
	)
	if err != nil {
		c.mu.Unlock()
		return ports.ReciboConsumoCobertura{}, err
	}
	c.registros[claveEfecto] = registroConsumoCoberturaAplicacionPrueba{
		huellaPeticion:  datos.HuellaPeticionSHA256,
		huellaResultado: datos.HuellaResultadoSHA256,
		recibo:          recibo,
	}
	c.evidencias[claveEvidencia] =
		evidenciaConsumoCoberturaAplicacionPrueba{
			claveEfecto:     claveEfecto,
			huellaRespuesta: datos.HuellaRespuestaSHA256,
		}
	c.mu.Unlock()
	if c.despues != nil {
		c.despues(ctx)
	}
	if c.responder != nil {
		return c.responder(recibo)
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
			audiencia:    "servicio_contratacion_temporal",
			identidades: map[ports.RolAutoridadFuenteAnalisis]ports.IdentidadAutoridadFuenteAnalisis{
				ports.RolFuenteCobertura:             fuente.identidad,
				ports.RolVerificadorCobertura:        verificador.identidad,
				ports.RolPublicadorCatalogoCobertura: publicador.identidad,
			},
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
