package gobiernoconvocatorias

import (
	"context"
	"crypto/sha256"
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
	primariaReconciliada := s.IdentidadPrimaria
	d.ultimaReconciliada = &primariaReconciliada
	clave := claveL(s.IdentidadPrimaria)
	fila, existe := d.filas[clave]
	if !existe || !mismaF(fila.identidadPrimaria, s.IdentidadPrimaria) {
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
	clave := claveL(s.ResolucionAnterior.IdentidadPrimaria)
	fila, existe := d.filas[clave]
	anterior := s.Reconciliacion.Resultado
	if !existe || !mismaF(fila.identidadPrimaria, s.ResolucionAnterior.IdentidadPrimaria) ||
		fila.resultado.Estado != ResultadoDiarioNoAplicado ||
		fila.resultado.Revision != anterior.Revision || fila.resultado.Cercado != anterior.Cercado {
		return ResultadoOperacionDiario{}, ErrCercadoDiarioObsoleto
	}
	fila.resultado = ResultadoOperacionDiario{
		Estado: ResultadoDiarioReservado, Revision: anterior.Revision + 1, Cercado: anterior.Cercado + 1,
		ArrendamientoIniciaEn: s.Nueva.Proyeccion.ArrendamientoIniciaEn,
		ArrendamientoVenceEn:  s.Nueva.Proyeccion.ArrendamientoVenceEn,
	}
	fila.identidadPrimaria = s.ResolucionAnterior.IdentidadPrimaria
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

// cifradorPrueba es deliberadamente un doble: sus bytes no representan
// criptografia productiva y nunca se compilan fuera de _test.go.
type resolvedorPerfilCifradoPrueba struct {
	mu       sync.Mutex
	llamadas int
}

func (r *resolvedorPerfilCifradoPrueba) ResolverPerfilCifradoBorrador(
	_ context.Context,
	s SolicitudResolucionPerfilCifradoBorrador,
) (PerfilCifradoBorrador, error) {
	if s.Validar() != nil {
		return PerfilCifradoBorrador{}, ErrCifradoBorradorInvalido
	}
	perfil, err := NuevoPerfilCifradoBorrador(
		"perfil:cifrado:borradores:v1", 1, huellaHexPrueba('a'),
		"A256GCM", "A256KW",
	)
	if err != nil {
		return PerfilCifradoBorrador{}, err
	}
	r.mu.Lock()
	r.llamadas++
	r.mu.Unlock()
	return perfil, nil
}

type cifradorPrueba struct {
	mu       sync.Mutex
	llamadas int
	ultima   *SolicitudCifradoBorrador
}

func (c *cifradorPrueba) CifrarBorrador(
	_ context.Context,
	s SolicitudCifradoBorrador,
) (ResultadoCifradoBorrador, error) {
	if s.Validar() != nil {
		return ResultadoCifradoBorrador{}, ErrCifradoBorradorInvalido
	}
	versionCanonica, err := s.VersionCanonicaParaCifrado()
	if err != nil {
		return ResultadoCifradoBorrador{}, err
	}
	aad, err := s.AADCanonica()
	if err != nil {
		clear(versionCanonica)
		return ResultadoCifradoBorrador{}, err
	}
	material := append(append([]byte(nil), versionCanonica...), aad...)
	suma := sha256.Sum256(material)
	clear(material)
	clear(versionCanonica)
	huellaAAD, _ := s.aad.HuellaSHA256()
	perfil := s.PerfilEsperado
	envoltura, err := NuevaEnvolturaClaveKMSBorrador(
		perfil, "clave:kms:borradores:v1", 1, suma[:], huellaAAD,
	)
	if err != nil {
		return ResultadoCifradoBorrador{}, err
	}
	nonce := make([]byte, 12)
	copy(nonce, suma[:12])
	sobre, err := NuevoSobreCifradoAEADBorrador(perfil, nonce, suma[:], huellaAAD)
	if err != nil {
		return ResultadoCifradoBorrador{}, err
	}
	atestacion, err := NuevaAtestacionKMSBorrador(
		"atestacion:kms:borrador:001", 1, perfil, "clave:kms:borradores:v1", 1,
		huellaAAD, envoltura.huellaSHA256, sobre.huellaSHA256, "verificador:kms:v1",
		s.SolicitadaEn, s.SolicitadaEn.Add(4*time.Minute),
	)
	if err != nil {
		return ResultadoCifradoBorrador{}, err
	}
	c.mu.Lock()
	c.llamadas++
	copia := s
	c.ultima = &copia
	c.mu.Unlock()
	return ResultadoCifradoBorrador{
		AAD: s.aad, EnvolturaClave: envoltura, SobreCifrado: sobre,
		AtestacionKMS: atestacion, SolicitadaEn: s.SolicitadaEn, CifradoEn: s.SolicitadaEn,
	}, nil
}

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
	clave := claveL(s.Reserva.IdentidadPrimaria)
	fila, existe := c.diario.filas[clave]
	if !existe || !mismaF(fila.identidadPrimaria, s.Reserva.IdentidadPrimaria) ||
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
		EstadoPrincipal:   s.Material.EstadoPrincipalNuevo,
		IdentidadPrimaria: s.Reserva.IdentidadPrimaria,
		Decision:          s.Reserva.Decision, SelladoMotivo: s.SelladoMotivo,
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
	cifrador    *cifradorPrueba
	perfiles    *resolvedorPerfilCifradoPrueba
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
	cifrador := &cifradorPrueba{}
	perfiles := &resolvedorPerfilCifradoPrueba{}
	derivador := derivadorPrueba{generaciones: generaciones}
	servicio, err := NuevoServicioBorradores(
		reloj, catalogo, catalogo, lectorPrueba{inicial}, comprometedorPrueba{}, derivador,
		autorizador, diario, selladorPrueba{}, perfiles, cifrador, confirmador,
	)
	if err != nil {
		t.Fatal(err)
	}
	return escenarioPrueba{
		servicio: servicio, reloj: reloj, catalogo: catalogo, diario: diario,
		autorizador: autorizador, confirmador: confirmador, cifrador: cifrador,
		perfiles:  perfiles,
		derivador: derivador, orden: orden, inicial: inicial,
	}
}

func (e escenarioPrueba) reiniciar(t *testing.T, generaciones ...uint32) *ServicioBorradores {
	t.Helper()
	derivador := derivadorPrueba{generaciones: generaciones}
	servicio, err := NuevoServicioBorradores(
		e.reloj, e.catalogo, e.catalogo, lectorPrueba{e.inicial}, comprometedorPrueba{}, derivador,
		e.autorizador, e.diario, selladorPrueba{}, e.perfiles, e.cifrador, e.confirmador,
	)
	if err != nil {
		t.Fatal(err)
	}
	return servicio
}
