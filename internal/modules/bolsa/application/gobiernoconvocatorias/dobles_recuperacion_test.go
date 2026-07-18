package gobiernoconvocatorias

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	pruebasvec "vec-diputacion-granada/internal/vec/pruebas"
)

func (d *diarioPrueba) Reconciliar(
	_ context.Context,
	s SolicitudReconciliacionBorrador,
) (ResultadoReconciliacionBorrador, error) {
	if s.Validar() != nil {
		return ResultadoReconciliacionBorrador{}, ErrReconciliacionBorradorInvalida
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	clave := claveL(s.Identidad)
	fila, existe := d.filas[clave]
	if !existe || !mismaF(fila.identidad, s.Identidad) {
		return ResultadoReconciliacionBorrador{}, errors.New("fila no encontrada")
	}
	pruebaRef, huellaPrueba := "", ""
	switch fila.resultado.Estado {
	case ResultadoDiarioConfirmado:
	case ResultadoDiarioNoAplicado:
		pruebaRef, huellaPrueba = "prueba:rollback:borrador:001", huellaHexPrueba('c')
	case ResultadoDiarioReservado, ResultadoDiarioEnCurso:
		if s.SolicitadaEn.Before(fila.resultado.ArrendamientoVenceEn) {
			break
		}
		fila.resultado.Revision++
		fila.resultado.Cercado++
		fila.resultado.Estado = ResultadoDiarioNoAplicado
		pruebaRef, huellaPrueba = "prueba:rollback:borrador:001", huellaHexPrueba('c')
		d.filas[clave] = fila
	case ResultadoDiarioIndeterminado:
		fila.resultado.Revision++
		if fila.confirmacionOculta != nil {
			recibo := construirReciboPrueba(
				*fila.confirmacionOculta, fila.resultado.Revision, fila.resultado.Cercado,
			)
			fila.resultado.Estado = ResultadoDiarioConfirmado
			fila.resultado.Recibo = &recibo
			fila.confirmacionOculta = nil
		} else {
			fila.resultado.Cercado++
			fila.resultado.Estado = ResultadoDiarioNoAplicado
			pruebaRef, huellaPrueba = "prueba:rollback:borrador:001", huellaHexPrueba('c')
		}
		d.filas[clave] = fila
	default:
		return ResultadoReconciliacionBorrador{}, ErrReconciliacionBorradorInvalida
	}
	return ResultadoReconciliacionBorrador{
		Resultado: copiarResultado(fila.resultado), ComprobadaEn: s.SolicitadaEn,
		PruebaDesenlaceRef: pruebaRef, HuellaPruebaSHA256: huellaPrueba,
	}, nil
}

func (d *diarioPrueba) ReclamarDecision(
	_ context.Context,
	s SolicitudReclamacionDecisionBorrador,
) (ResultadoOperacionDiario, error) {
	if s.Validar() != nil {
		return ResultadoOperacionDiario{}, ErrReclamacionBorradorInvalida
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	clave := claveL(s.IdentidadAnterior)
	fila, existe := d.filas[clave]
	anterior := s.Reconciliacion.Resultado
	if !existe || !mismaF(fila.identidad, s.IdentidadAnterior) ||
		fila.resultado.Estado != ResultadoDiarioNoAplicado ||
		fila.resultado.Revision != anterior.Revision || fila.resultado.Cercado != anterior.Cercado {
		return ResultadoOperacionDiario{}, ErrCercadoDiarioObsoleto
	}
	fila.resultado = ResultadoOperacionDiario{
		Estado: ResultadoDiarioReservado, Revision: anterior.Revision + 1, Cercado: anterior.Cercado + 1,
		ArrendamientoIniciaEn: s.Nueva.Proyeccion.ArrendamientoIniciaEn,
		ArrendamientoVenceEn:  s.Nueva.Proyeccion.ArrendamientoVenceEn,
	}
	fila.identidad = s.IdentidadAnterior
	d.filas[clave] = fila
	d.reclamos++
	return copiarResultado(fila.resultado), nil
}

type selladorPrueba struct{}

func (selladorPrueba) VerificarYSellarMotivo(
	_ context.Context,
	s SolicitudSelladoMotivoBorrador,
) (ProyeccionSelladoMotivoBorrador, error) {
	if err := s.Validar(); err != nil {
		return ProyeccionSelladoMotivoBorrador{}, err
	}
	datos, err := s.Compromiso.DatosParaMaterial()
	if err != nil {
		return ProyeccionSelladoMotivoBorrador{}, err
	}
	hmacDurable, err := datos.HMAC.ProyeccionDurable()
	if err != nil {
		return ProyeccionSelladoMotivoBorrador{}, err
	}
	return ProyeccionSelladoMotivoBorrador{
		Accion: s.Material.Accion, ConvocatoriaRef: s.Material.EstadoPrincipalNuevo.Referencia,
		HMAC: hmacDurable, AtestacionRef: "atestacion:motivo:001", VersionAtestacion: 1,
		EstadoAtestacion: "verificada", HuellaAtestacionSHA256: huellaHexPrueba('5'),
		TokenConsumoRef: "consumo:motivo:001", MaterializadorRef: "materializador:motivo:v1",
		AtestacionEmitidaEn: s.SolicitadaEn, AtestacionValidaHasta: s.SolicitadaEn.Add(4 * time.Minute),
	}, nil
}

type modoConfirmacion string

const (
	confirmarBien                  modoConfirmacion = "confirmar"
	confirmarIndeterminadoCommit   modoConfirmacion = "indeterminado_commit"
	confirmarIndeterminadoRollback modoConfirmacion = "indeterminado_rollback"
	confirmarRollback              modoConfirmacion = "rollback"
)

type confirmadorPrueba struct {
	diario            *diarioPrueba
	mu                sync.Mutex
	modo              modoConfirmacion
	llamadas, efectos int
	ultima            *SolicitudConfirmacionBorrador
}

func (c *confirmadorPrueba) cambiarModo(modo modoConfirmacion) {
	c.mu.Lock()
	c.modo = modo
	c.mu.Unlock()
}

func (c *confirmadorPrueba) ConfirmarBorrador(
	_ context.Context,
	s SolicitudConfirmacionBorrador,
) (ResultadoConfirmacionAtomica, error) {
	if s.Validar() != nil {
		return ResultadoConfirmacionAtomica{}, ErrResultadoBorradorInseguro
	}
	c.mu.Lock()
	c.llamadas++
	modo := c.modo
	copia := s
	c.ultima = &copia
	c.mu.Unlock()
	c.diario.mu.Lock()
	defer c.diario.mu.Unlock()
	clave := claveL(s.Reserva.Identidad)
	fila, existe := c.diario.filas[clave]
	if !existe || !mismaF(fila.identidad, s.Reserva.Identidad) ||
		fila.resultado.Estado != ResultadoDiarioReservado ||
		fila.resultado.Revision != s.Control.Revision || fila.resultado.Cercado != s.Control.Cercado {
		return ResultadoConfirmacionAtomica{Estado: ResultadoDiarioIndeterminado}, errors.New("CAS o cercado rechazado")
	}
	fila.resultado.Revision++
	switch modo {
	case confirmarRollback:
		fila.resultado.Cercado++
		fila.resultado.Estado = ResultadoDiarioNoAplicado
		c.diario.filas[clave] = fila
		return ResultadoConfirmacionAtomica{Estado: ResultadoDiarioNoAplicado}, errors.New("rollback confirmado")
	case confirmarIndeterminadoRollback:
		fila.resultado.Estado = ResultadoDiarioIndeterminado
		c.diario.filas[clave] = fila
		return ResultadoConfirmacionAtomica{Estado: ResultadoDiarioIndeterminado}, errors.New("respuesta perdida")
	case confirmarIndeterminadoCommit:
		fila.resultado.Estado = ResultadoDiarioIndeterminado
		fila.confirmacionOculta = &copia
		c.diario.filas[clave] = fila
		c.mu.Lock()
		c.efectos++
		c.mu.Unlock()
		return ResultadoConfirmacionAtomica{Estado: ResultadoDiarioIndeterminado}, errors.New("commit sin respuesta")
	default:
		recibo := construirReciboPrueba(s, fila.resultado.Revision, fila.resultado.Cercado)
		fila.resultado.Estado = ResultadoDiarioConfirmado
		fila.resultado.Recibo = &recibo
		c.diario.filas[clave] = fila
		c.mu.Lock()
		c.efectos++
		c.mu.Unlock()
		return ResultadoConfirmacionAtomica{Estado: ResultadoDiarioConfirmado, Recibo: recibo}, nil
	}
}

func construirReciboPrueba(
	s SolicitudConfirmacionBorrador,
	revision, cercado uint64,
) ProyeccionReciboBorrador {
	return ProyeccionReciboBorrador{
		Esquema: esquemaReciboBorradorV2, ReciboRef: "recibo:convocatoria:001",
		TransaccionRef: "transaccion:convocatoria:001", Accion: s.Material.Accion,
		EstadoPrincipal: s.Material.EstadoPrincipalNuevo, Identidad: s.Reserva.Identidad,
		Decision: s.Reserva.Decision, SelladoMotivo: s.SelladoMotivo,
		RevisionConfirmada: revision, CercadoConfirmado: cercado,
		ArrendamientoIniciaEn: s.Reserva.ArrendamientoIniciaEn,
		ArrendamientoVenceEn:  s.Reserva.ArrendamientoVenceEn,
		AuditoriaRef:          "auditoria:convocatoria:001", HuellaAuditoriaSHA256: huellaHexPrueba('6'),
		EventoOutboxRef: "outbox:convocatoria:001", HuellaEventoOutboxSHA256: huellaHexPrueba('7'),
		ConfirmadaEn: s.SolicitadaEn,
	}
}

type escenarioPrueba struct {
	servicio    *ServicioBorradores
	reloj       *relojPrueba
	catalogo    *catalogoPrueba
	diario      *diarioPrueba
	autorizador *autorizadorPrueba
	confirmador *confirmadorPrueba
	derivador   derivadorPrueba
	orden       OrdenCrearBorrador
	inicial     dominiobolsa.VersionConvocatoriaGobernada
}

func nuevoEscenario(t *testing.T, modo modoConfirmacion, generaciones ...uint32) escenarioPrueba {
	t.Helper()
	actor, vinculo, err := pruebasvec.NuevoContextoYVinculo(
		instanteBorradorPrueba, "per_0123456789abcdefghijkl", "prf_0123456789abcdefghijkl",
		dominiovec.AuthMethodCertificate, dominiovec.AuthAssuranceHigh,
	)
	if err != nil {
		t.Fatal(err)
	}
	contenido, configuracion, ambito := datosPublicablesPrueba(t)
	catalogo := &catalogoPrueba{
		plantilla: PlantillaBorradorResuelta{
			Referencia: dominiobolsa.ReferenciaConfiguracionConvocatoria{
				ID: "plantilla:bolsa:general", Version: 2, HuellaContenidoSHA256: huellaHexPrueba('8'),
			},
			Configuracion: configuracion,
		},
		ambito: ambito,
	}
	clave, err := NuevaClaveClienteIdempotenciaConvocatoria("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")
	if err != nil {
		t.Fatal(err)
	}
	orden := OrdenCrearBorrador{
		ClaveCliente: clave, Actor: actor, VinculoAutenticacionActor: vinculo,
		Plantilla: SelectorPlantillaBorrador{
			ID: catalogo.plantilla.Referencia.ID, Version: 2,
			HuellaContenidoSHA256: catalogo.plantilla.Referencia.HuellaContenidoSHA256,
		},
		CodigoVersionPublica: "v1", Contenido: contenido,
		ExpedienteRef: "expediente:seleccion:2026-001",
		MotivoCatalogo: dominiovec.ReferenciaEntradaCatalogo{
			CatalogoID: "motivos_rrhh", CatalogoVersion: 1,
			CatalogoHuellaSHA256: huellaHexPrueba('9'), EntradaClave: "crear_borrador",
		},
		CorrelacionRef: "correlacion:convocatoria:001",
	}
	inicial, err := dominiobolsa.NuevaVersionConvocatoriaGobernada(
		dominiobolsa.DatosNuevaVersionConvocatoriaGobernada{
			ID: "proceso:bolsa:auxiliar-inicial", CodigoVersionPublica: "v1",
			InstanciaFlujoRef: "instancia:flujo:convocatoria:inicial", AmbitoOrganizativo: ambito,
			Contenido: contenido, Configuracion: configuracion, ExpedienteRef: orden.ExpedienteRef,
			Motivo: orden.MotivoCatalogo.Referencia(), ActorID: actor.PersonaRef,
			Instante: instanteBorradorPrueba.Add(-time.Hour),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	reloj := &relojPrueba{instante: instanteBorradorPrueba, paso: time.Millisecond}
	diario := nuevoDiarioPrueba()
	autorizador := &autorizadorPrueba{modo: pdpConceder}
	confirmador := &confirmadorPrueba{diario: diario, modo: modo}
	derivador := derivadorPrueba{generaciones: generaciones}
	servicio, err := NuevoServicioBorradores(
		reloj, catalogo, catalogo, lectorPrueba{inicial}, comprometedorPrueba{}, derivador,
		autorizador, diario, selladorPrueba{}, confirmador,
	)
	if err != nil {
		t.Fatal(err)
	}
	return escenarioPrueba{servicio, reloj, catalogo, diario, autorizador, confirmador, derivador, orden, inicial}
}

func (e escenarioPrueba) reiniciar(t *testing.T, generaciones ...uint32) *ServicioBorradores {
	t.Helper()
	derivador := derivadorPrueba{generaciones: generaciones}
	servicio, err := NuevoServicioBorradores(
		e.reloj, e.catalogo, e.catalogo, lectorPrueba{e.inicial}, comprometedorPrueba{}, derivador,
		e.autorizador, e.diario, selladorPrueba{}, e.confirmador,
	)
	if err != nil {
		t.Fatal(err)
	}
	return servicio
}
