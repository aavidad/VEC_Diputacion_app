package application

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"mime"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

var (
	ErrDependenciaCargaDocumentalRequerida = errors.New("vec: dependencia de carga documental requerida")
	ErrOrdenCargaDocumentalInvalida        = errors.New("vec: orden de carga documental invalida")
	ErrCargaDocumentalNoCorresponde        = errors.New("vec: carga documental no corresponde al contexto")
	ErrCargaDocumentalNoPreparada          = errors.New("vec: carga documental no preparada")
	ErrResultadoCargaDocumentalInvalido    = errors.New("vec: resultado de carga documental invalido")
	ErrCargaDocumentalYaProcesada          = errors.New("vec: carga documental ya procesada")
)

const (
	AccionPrepararCargaDocumental  = "vec.documentos.carga.preparar"
	AccionConfirmarCargaDocumental = "vec.documentos.carga.confirmar"
	AccionAnalizarCargaDocumental  = "vec.documentos.carga.analizar"
	AccionPromoverCargaDocumental  = "vec.documentos.carga.promover"

	CampoCargaClasificacion       = "clasificacion"
	CampoCargaContenido           = "contenido"
	CampoCargaHuellaSHA256        = "huella_sha256"
	CampoCargaMIME                = "mime"
	CampoCargaTamano              = "tamano"
	CampoCargaContenidoCuarentena = "contenido_cuarentena"
	CampoCargaAnalisisSeguridad   = "analisis_seguridad"
	CampoCargaContenidoAdmitido   = "contenido_admitido"
	CampoCargaEstado              = "estado"

	eventoCargaDocumentalPreparada = "vec.documentos.carga.preparada"
	eventoCargaDocumentalRecibida  = "vec.documentos.carga.recibida"
	eventoCargaDocumentalAnalizada = "vec.documentos.carga.analizada"
	eventoCargaDocumentalPromovida = "vec.documentos.carga.promovida"

	vigenciaCargaDocumentalPredeterminada = 2 * time.Minute
	vigenciaCargaDocumentalMaxima         = 10 * time.Minute
	tamanoCargaDocumentalPredeterminado   = int64(64 * 1024 * 1024)
	tamanoCargaDocumentalMaximo           = int64(512 * 1024 * 1024)
	toleranciaRelojAnalizador             = 2 * time.Minute
)

type OpcionesServicioCargaDocumental struct {
	VigenciaInstrucciones       time.Duration
	TamanoMaximo                int64
	ConectorAlmacenPermitido    string
	ConectorAnalizadorPermitido string
	VersionAnalizadorPermitida  int
}

// ServicioCargaDocumental coordina autorizacion, carga directa, cuarentena,
// analisis y promocion sin conocer S3, MinIO, ICAP, ClamAV ni PostgreSQL.
// Todos los errores de dependencias o respuestas ambiguas fallan cerrados.
type ServicioCargaDocumental struct {
	repositorio           ports.RepositorioCargasDocumentales
	autorizador           ports.Autorizador
	almacen               ports.AlmacenObjetos
	gestorCargaDirecta    ports.GestorCargaDirecta
	analizador            ports.AnalizadorContenido
	selladorIdempotencia  ports.SelladorIdempotenciaCarga
	selladorSolicitud     ports.SelladorSolicitudCargaDocumental
	selladorSesion        ports.SelladorVinculoSesionCarga
	seudonimizadorSujeto  ports.SeudonimizadorSujetoAlmacen
	emisorRecibo          ports.EmisorReciboCargaDirecta
	consumidorRecibo      ports.ConsumidorReciboCargaDirecta
	verificadorRecibo     ports.VerificadorAtestacionConsumoReciboCargaDirecta
	generadorID           ports.GeneradorIDCargaDocumental
	reloj                 ports.Reloj
	vigenciaInstrucciones time.Duration
	tamanoMaximo          int64
	conectorAlmacen       string
	conectorAnalizador    string
	versionAnalizador     int
}

func NuevoServicioCargaDocumental(
	repositorio ports.RepositorioCargasDocumentales,
	autorizador ports.Autorizador,
	almacen ports.AlmacenObjetos,
	gestorCargaDirecta ports.GestorCargaDirecta,
	analizador ports.AnalizadorContenido,
	selladorIdempotencia ports.SelladorIdempotenciaCarga,
	selladorSolicitud ports.SelladorSolicitudCargaDocumental,
	selladorSesion ports.SelladorVinculoSesionCarga,
	seudonimizadorSujeto ports.SeudonimizadorSujetoAlmacen,
	emisorRecibo ports.EmisorReciboCargaDirecta,
	consumidorRecibo ports.ConsumidorReciboCargaDirecta,
	verificadorRecibo ports.VerificadorAtestacionConsumoReciboCargaDirecta,
	generadorID ports.GeneradorIDCargaDocumental,
	reloj ports.Reloj,
	opciones OpcionesServicioCargaDocumental,
) (*ServicioCargaDocumental, error) {
	if dependenciaCargaDocumentalNula(repositorio) || dependenciaCargaDocumentalNula(autorizador) ||
		dependenciaCargaDocumentalNula(almacen) || dependenciaCargaDocumentalNula(gestorCargaDirecta) ||
		dependenciaCargaDocumentalNula(analizador) || dependenciaCargaDocumentalNula(selladorIdempotencia) ||
		dependenciaCargaDocumentalNula(selladorSolicitud) || dependenciaCargaDocumentalNula(selladorSesion) ||
		dependenciaCargaDocumentalNula(seudonimizadorSujeto) || dependenciaCargaDocumentalNula(emisorRecibo) ||
		dependenciaCargaDocumentalNula(consumidorRecibo) || dependenciaCargaDocumentalNula(verificadorRecibo) ||
		dependenciaCargaDocumentalNula(generadorID) || dependenciaCargaDocumentalNula(reloj) ||
		!mismaProteccionRecibosCargaDirecta(emisorRecibo, consumidorRecibo, verificadorRecibo) {
		return nil, ErrDependenciaCargaDocumentalRequerida
	}
	vigencia := opciones.VigenciaInstrucciones
	if vigencia == 0 {
		vigencia = vigenciaCargaDocumentalPredeterminada
	}
	tamano := opciones.TamanoMaximo
	if tamano == 0 {
		tamano = tamanoCargaDocumentalPredeterminado
	}
	if vigencia < time.Second || vigencia > vigenciaCargaDocumentalMaxima || tamano < 1 || tamano > tamanoCargaDocumentalMaximo ||
		!textoCargaSeguro(opciones.ConectorAlmacenPermitido, 128, false) ||
		!textoCargaSeguro(opciones.ConectorAnalizadorPermitido, 128, false) ||
		opciones.VersionAnalizadorPermitida < 1 {
		return nil, ErrDependenciaCargaDocumentalRequerida
	}
	return &ServicioCargaDocumental{
		repositorio: repositorio, autorizador: autorizador, almacen: almacen,
		gestorCargaDirecta: gestorCargaDirecta, analizador: analizador,
		selladorIdempotencia: selladorIdempotencia, selladorSolicitud: selladorSolicitud,
		selladorSesion: selladorSesion, seudonimizadorSujeto: seudonimizadorSujeto,
		emisorRecibo: emisorRecibo, consumidorRecibo: consumidorRecibo, verificadorRecibo: verificadorRecibo,
		generadorID: generadorID, reloj: reloj,
		vigenciaInstrucciones: vigencia, tamanoMaximo: tamano,
		conectorAlmacen:    opciones.ConectorAlmacenPermitido,
		conectorAnalizador: opciones.ConectorAnalizadorPermitido,
		versionAnalizador:  opciones.VersionAnalizadorPermitida,
	}, nil
}

type OrdenPrepararCargaDocumental struct {
	Principal                domain.Principal
	PerfilActivo             string
	Recurso                  domain.RecursoAutorizable
	OperacionRef             string
	Finalidad                string
	Clasificacion            string
	MIME                     string
	Tamano                   int64
	HuellaSHA256             string
	ClaveIdempotenciaCliente string
	Motivo                   string
	CorrelacionRef           string
}

type ResultadoPrepararCargaDocumental struct {
	Carga         domain.CargaDocumental
	Instrucciones ports.InstruccionesCargaDirecta
	Recibo        ports.ReciboCargaDirecta
	Repetida      bool
}

func (s *ServicioCargaDocumental) Preparar(
	ctx context.Context,
	orden OrdenPrepararCargaDocumental,
) (ResultadoPrepararCargaDocumental, error) {
	if s == nil || ctx == nil {
		return ResultadoPrepararCargaDocumental{}, ErrDependenciaCargaDocumentalRequerida
	}
	orden = clonarOrdenPrepararCargaDocumental(orden)
	if err := validarOrdenPrepararCargaDocumental(orden, s.tamanoMaximo); err != nil {
		return ResultadoPrepararCargaDocumental{}, err
	}
	huellaRecursoBase, err := ports.HuellaRecursoBaseCargaDocumental(orden.Recurso)
	if err != nil {
		return ResultadoPrepararCargaDocumental{}, errors.Join(
			domain.ErrAutorizacionDenegada, ports.ErrAutorizacionAlmacenInvalida, err,
		)
	}
	ahora := s.reloj.Ahora().UTC()
	if ahora.IsZero() {
		return ResultadoPrepararCargaDocumental{}, ErrResultadoCargaDocumentalInvalido
	}
	indice, err := s.selladorIdempotencia.SellarIdempotenciaCarga(ctx, ports.SolicitudSellarIdempotenciaCarga{
		OperacionRef: orden.OperacionRef, PrincipalRef: orden.Principal.ID, RecursoRef: orden.Recurso.Referencia,
		MIME: orden.MIME, Tamano: orden.Tamano, HuellaSHA256: orden.HuellaSHA256,
		ClaveSolicitante: orden.ClaveIdempotenciaCliente,
	})
	if err != nil || !hmacCargaDocumentalValido(indice) {
		return ResultadoPrepararCargaDocumental{}, ports.ErrSelladoIdempotenciaCargaNoDisponible
	}
	huellaSolicitud, err := s.selladorSolicitud.SellarSolicitudCargaDocumental(ctx, solicitudCargaDocumentalCanonica(orden))
	if err != nil || !hmacCargaDocumentalValido(huellaSolicitud) {
		return ResultadoPrepararCargaDocumental{}, ports.ErrSelladoIdempotenciaCargaNoDisponible
	}
	id, err := s.generadorID.NuevoIDCargaDocumental()
	if err != nil {
		return ResultadoPrepararCargaDocumental{}, fmt.Errorf("generar identificador de carga: %w", err)
	}
	expiraEn := ahora.Add(s.vigenciaInstrucciones)
	carga, err := domain.NuevaCargaDocumental(
		id, orden.Principal.ID, orden.Recurso.Referencia,
		orden.Recurso.ModuloID, orden.Recurso.Tipo, orden.OperacionRef,
		orden.CorrelacionRef, orden.Finalidad, orden.Clasificacion,
		orden.MIME, orden.Tamano, orden.HuellaSHA256, indice, huellaSolicitud,
		ahora, expiraEn,
	)
	if err != nil {
		return ResultadoPrepararCargaDocumental{}, err
	}
	seudonimoSujeto, err := s.seudonimizarSujetoCarga(ctx, carga)
	if err != nil {
		return ResultadoPrepararCargaDocumental{}, err
	}
	vinculosPreparacion, err := vinculosPrepararCargaDocumental(carga, seudonimoSujeto)
	if err != nil {
		return ResultadoPrepararCargaDocumental{}, err
	}
	recursoAutorizable, err := recursoPrepararCargaDocumental(orden.Recurso, vinculosPreparacion)
	if err != nil {
		return ResultadoPrepararCargaDocumental{}, err
	}
	decision, err := exigirDecisionAutorizacion(
		ctx, s.autorizador, s.reloj, orden.Principal, orden.PerfilActivo, AccionPrepararCargaDocumental,
		recursoAutorizable, orden.Finalidad, orden.CorrelacionRef, orden.Motivo, usoCamposDecisionConsumidos,
	)
	if err != nil {
		return ResultadoPrepararCargaDocumental{}, err
	}
	if err := validarAlcanceDecisionCarga(decision, camposDecisionPrepararCarga()); err != nil {
		return ResultadoPrepararCargaDocumental{}, err
	}
	verificadaEn := s.reloj.Ahora().UTC()
	contextoAlmacen, err := ports.NuevoContextoPrepararCargaDirectaAlmacen(
		decision, recursoAutorizable, vinculosPreparacion, verificadaEn,
	)
	if err != nil || contextoAlmacen.ValidarParaEn(ports.AccionAlmacenPrepararCargaDirecta, verificadaEn) != nil {
		return ResultadoPrepararCargaDocumental{}, errors.Join(domain.ErrAutorizacionDenegada, err)
	}
	decisionPreparacion, err := ports.ConsumoDecisionDesdeContextoPreparacionCargaDocumental(contextoAlmacen)
	if err != nil {
		return ResultadoPrepararCargaDocumental{}, errors.Join(domain.ErrAutorizacionDenegada, err)
	}
	contextoAbandono, err := contextoAlmacen.DerivarPaso(ports.PasoAlmacenAbandonarCargaDirecta)
	if err != nil {
		return ResultadoPrepararCargaDocumental{}, errors.Join(domain.ErrAutorizacionDenegada, err)
	}
	if decision.ValidaHasta.Before(expiraEn) {
		expiraEn = decision.ValidaHasta.UTC()
	}
	if !expiraEn.After(verificadaEn) {
		return ResultadoPrepararCargaDocumental{}, domain.ErrAutorizacionDenegada
	}
	if !carga.ExpiraEn.Equal(expiraEn) {
		carga, err = domain.NuevaCargaDocumental(
			id, orden.Principal.ID, orden.Recurso.Referencia,
			orden.Recurso.ModuloID, orden.Recurso.Tipo, orden.OperacionRef,
			orden.CorrelacionRef, orden.Finalidad, orden.Clasificacion,
			orden.MIME, orden.Tamano, orden.HuellaSHA256, indice, huellaSolicitud,
			ahora, expiraEn,
		)
		if err != nil {
			return ResultadoPrepararCargaDocumental{}, err
		}
	}
	reserva, err := s.repositorio.Reservar(ctx, ports.SolicitudReservarCargaDocumental{
		IndiceIdempotenciaHMAC: indice, HuellaSolicitudHMAC: huellaSolicitud, Carga: carga,
		DecisionPreparacion: decisionPreparacion,
		SolicitadaEn:        ahora, ReservaExpiraEn: expiraEn,
	})
	if err != nil {
		return ResultadoPrepararCargaDocumental{}, err
	}
	if err := reserva.Validar(); err != nil || reserva.Carga.HuellaSolicitudHMAC != huellaSolicitud ||
		reserva.Carga.IndiceIdempotenciaHMAC != indice {
		return ResultadoPrepararCargaDocumental{}, ports.ErrReservaCargaDocumentalInvalida
	}
	if reserva.Repetida {
		// Reemitir exigiria activar coordinadamente otro manifiesto y otro
		// recibo en dos repositorios. Sin una transaccion comun o sin incluir
		// DecisionRef+huella del manifiesto en la atestacion del recibo, cualquier
		// orden de escrituras abre una ventana de cruce. Se mantiene el primer
		// manifiesto inmutable y el reintento falla cerrado.
		return ResultadoPrepararCargaDocumental{}, ErrCargaDocumentalYaProcesada
	}
	huellaReservada, err := reserva.Carga.HuellaSHA256()
	if err != nil {
		s.abandonarReserva(ctx, reserva.Token)
		return ResultadoPrepararCargaDocumental{}, ports.ErrReservaCargaDocumentalInvalida
	}
	huellaPropuesta, err := carga.HuellaSHA256()
	if err != nil || huellaReservada != huellaPropuesta {
		s.abandonarReserva(ctx, reserva.Token)
		return ResultadoPrepararCargaDocumental{}, ports.ErrReservaCargaDocumentalInvalida
	}

	capacidades, err := s.capacidadesAlmacenCargaDirecta(ctx, orden.Tamano)
	if err != nil {
		s.abandonarReserva(ctx, reserva.Token)
		return ResultadoPrepararCargaDocumental{}, err
	}
	solicitudPreparar := solicitudPreparacionCarga(carga, contextoAlmacen)
	instantePreparacion := s.reloj.Ahora().UTC()
	if contextoAlmacen.ValidarParaEn(ports.AccionAlmacenPrepararCargaDirecta, instantePreparacion) != nil {
		s.abandonarReserva(ctx, reserva.Token)
		return ResultadoPrepararCargaDocumental{}, domain.ErrAutorizacionDenegada
	}
	instrucciones, err := s.gestorCargaDirecta.PrepararCargaDirecta(ctx, solicitudPreparar)
	if err != nil {
		s.abandonarReserva(ctx, reserva.Token)
		return ResultadoPrepararCargaDocumental{}, ports.ErrInstruccionesCargaDirectaNoValidas
	}
	confirmada := false
	persistenciaIniciada := false
	defer func() {
		// Tras iniciar el commit, un error puede significar respuesta ambigua.
		// Revocar entonces la sesion podria invalidar un agregado ya confirmado;
		// la reconciliacion idempotente debe resolver ese caso.
		if confirmada || persistenciaIniciada {
			return
		}
		ctxLimpieza, cancelar := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
		defer cancelar()
		instanteAbandono := s.reloj.Ahora().UTC()
		if contextoAbandono.ValidarParaEn(ports.AccionAlmacenAbandonarCargaDirecta, instanteAbandono) == nil {
			_ = instrucciones.Abandonar(ctxLimpieza, s.gestorCargaDirecta, contextoAbandono)
		}
		_ = s.repositorio.AbandonarReserva(ctxLimpieza, reserva.Token)
	}()
	preparadaEn := s.reloj.Ahora().UTC()
	if preparadaEn.Before(ahora) || instrucciones.ValidarPara(solicitudPreparar, capacidades) != nil ||
		!instrucciones.VigenteEn(preparadaEn) {
		return ResultadoPrepararCargaDocumental{}, ports.ErrInstruccionesCargaDirectaNoValidas
	}
	vinculoSesion, err := instrucciones.SellarVinculoSesion(ctx, s.selladorSesion)
	if err != nil {
		return ResultadoPrepararCargaDocumental{}, err
	}
	preparada, err := carga.Preparar(vinculoSesion, decision.DecisionRef, preparadaEn)
	if err != nil {
		return ResultadoPrepararCargaDocumental{}, err
	}
	confirmacion, err := confirmacionCargaDocumental(
		carga, preparada, orden.Principal, orden.PerfilActivo, decision.DecisionRef, orden.Motivo,
		AccionPrepararCargaDocumental, eventoCargaDocumentalPreparada, preparadaEn,
		map[string]string{"almacen_conector": capacidades.ConectorID, "estado": string(preparada.Estado)},
	)
	if err != nil {
		return ResultadoPrepararCargaDocumental{}, err
	}
	manifiesto, err := manifiestoPreparacionCargaDocumental(
		preparada, contextoAlmacen, capacidades, huellaRecursoBase,
	)
	if err != nil {
		return ResultadoPrepararCargaDocumental{}, err
	}
	persistenciaIniciada = true
	ctxPersistencia, cancelarPersistencia := contextoPersistenciaCarga(ctx)
	defer cancelarPersistencia()
	if err := s.repositorio.ConfirmarPreparacion(ctxPersistencia, ports.SolicitudConfirmarPreparacionCargaDocumental{
		Token: reserva.Token, Confirmacion: confirmacion, Manifiesto: manifiesto,
	}); err != nil {
		// Estos conflictos son respuestas inequivocas anteriores al commit por
		// contrato. Se puede compensar la sesion sin riesgo de revocar una
		// preparacion confirmada; cualquier otro error sigue siendo ambiguo y se
		// entrega a reconciliacion sin abandonar.
		if errors.Is(err, ports.ErrDecisionPreparacionCargaNoDisponible) ||
			errors.Is(err, ports.ErrDecisionPreparacionCargaYaConsumida) {
			persistenciaIniciada = false
		}
		return ResultadoPrepararCargaDocumental{}, err
	}
	confirmada = true
	recibo, err := instrucciones.EmitirReciboConfirmacion(ctx, solicitudPreparar, capacidades, s.emisorRecibo)
	if err != nil {
		return ResultadoPrepararCargaDocumental{}, err
	}
	return ResultadoPrepararCargaDocumental{Carga: preparada, Instrucciones: instrucciones, Recibo: recibo}, nil
}

type OrdenConfirmarCargaDocumental struct {
	Principal      domain.Principal
	PerfilActivo   string
	Recurso        domain.RecursoAutorizable
	CargaID        string
	SesionRef      string
	Recibo         ports.ReciboCargaDirecta
	Finalidad      string
	Motivo         string
	CorrelacionRef string
}

func (s *ServicioCargaDocumental) Confirmar(
	ctx context.Context,
	orden OrdenConfirmarCargaDocumental,
) (domain.CargaDocumental, error) {
	// NO EXPONIBLE EN PRODUCCION: entre el consumo durable del recibo y la
	// confirmacion remota existe una frontera no atomica. El caso permanece
	// interno para pruebas de contrato hasta implementar de forma durable la
	// intencion ya definida por el puerto, acreditar la idempotencia exacta del
	// conector y desplegar reconciliacion. Un HMAC atesta el consumo, pero no
	// convierte dos sistemas en una transaccion.
	if s == nil || ctx == nil {
		return domain.CargaDocumental{}, ErrDependenciaCargaDocumentalRequerida
	}
	orden = clonarOrdenConfirmarCargaDocumental(orden)
	if err := validarOrdenConfirmarCargaDocumental(orden); err != nil {
		return domain.CargaDocumental{}, err
	}
	preparacionPersistida, err := s.repositorio.ObtenerPreparacion(ctx, orden.CargaID)
	if err != nil {
		return domain.CargaDocumental{}, err
	}
	if preparacionPersistida.Validar() != nil {
		return domain.CargaDocumental{}, ports.ErrManifiestoPreparacionNoEncontrado
	}
	carga := preparacionPersistida.Carga
	manifiesto := preparacionPersistida.Manifiesto
	ahora := s.reloj.Ahora().UTC()
	if err := validarContextoCarga(carga, orden.Principal, orden.Recurso, orden.Finalidad, orden.CorrelacionRef); err != nil {
		return domain.CargaDocumental{}, err
	}
	if ports.ValidarRecursoBaseManifiestoPreparacionCargaDocumental(
		manifiesto, carga, orden.Recurso,
	) != nil {
		return domain.CargaDocumental{}, errors.Join(
			domain.ErrAutorizacionDenegada, ports.ErrConfirmacionCargaDocumentalInvalida,
		)
	}
	if carga.Estado != domain.EstadoCargaDocumentalPreparada || !carga.VigenteEn(ahora) {
		return domain.CargaDocumental{}, ErrCargaDocumentalNoPreparada
	}
	seudonimoSujeto, err := s.seudonimizarSujetoCarga(ctx, carga)
	if err != nil {
		return domain.CargaDocumental{}, err
	}
	vinculosConfirmacion, err := vinculosConfirmarCargaDocumental(carga, seudonimoSujeto)
	if err != nil {
		return domain.CargaDocumental{}, err
	}
	recursoAutorizable, err := recursoConfirmarCargaDocumental(orden.Recurso, vinculosConfirmacion)
	if err != nil {
		return domain.CargaDocumental{}, err
	}
	decision, err := exigirDecisionAutorizacion(
		ctx, s.autorizador, s.reloj, orden.Principal, orden.PerfilActivo, AccionConfirmarCargaDocumental,
		recursoAutorizable, orden.Finalidad, orden.CorrelacionRef, orden.Motivo, usoCamposDecisionConsumidos,
	)
	if err != nil {
		return domain.CargaDocumental{}, err
	}
	if err := validarAlcanceDecisionCarga(decision, camposDecisionConfirmarCarga()); err != nil {
		return domain.CargaDocumental{}, err
	}
	verificadaEn := s.reloj.Ahora().UTC()
	contextoConfirmacion, err := ports.NuevoContextoConfirmarCargaDirectaAlmacen(
		decision, recursoAutorizable, vinculosConfirmacion, verificadaEn,
	)
	if err != nil || contextoConfirmacion.ValidarParaEn(ports.AccionAlmacenConfirmarCargaDirecta, verificadaEn) != nil {
		return domain.CargaDocumental{}, errors.Join(domain.ErrAutorizacionDenegada, err)
	}
	if ports.ValidarManifiestoPreparacionParaConfirmacion(
		manifiesto, carga, contextoConfirmacion, orden.Recurso,
	) != nil {
		return domain.CargaDocumental{}, errors.Join(
			domain.ErrAutorizacionDenegada, ports.ErrConfirmacionCargaDocumentalInvalida,
		)
	}
	vinculo, err := s.selladorSesion.SellarVinculoSesionCarga(ctx, orden.SesionRef)
	if err != nil || !hmacCargaDocumentalValido(vinculo) ||
		subtle.ConstantTimeCompare([]byte(vinculo), []byte(carga.VinculoSesionHMAC)) != 1 {
		return domain.CargaDocumental{}, ports.ErrSesionCargaDirectaNoValida
	}
	capacidades, err := s.capacidadesAlmacenCargaDirecta(ctx, carga.TamanoDeclarado)
	if err != nil {
		return domain.CargaDocumental{}, err
	}
	validaHasta := carga.ExpiraEn.UTC()
	if decision.ValidaHasta.Before(validaHasta) {
		validaHasta = decision.ValidaHasta.UTC()
	}
	antesDelConsumo := s.reloj.Ahora().UTC()
	if contextoConfirmacion.ValidarParaEn(ports.AccionAlmacenConfirmarCargaDirecta, antesDelConsumo) != nil ||
		!validaHasta.After(antesDelConsumo) {
		return domain.CargaDocumental{}, domain.ErrAutorizacionDenegada
	}
	peticionConsumo := ports.SolicitudConsumirReciboCargaDirecta{
		Contexto: contextoConfirmacion, SesionRef: orden.SesionRef, Recibo: orden.Recibo,
		ValidaHasta: validaHasta,
	}
	if err := peticionConsumo.Validar(); err != nil {
		return domain.CargaDocumental{}, err
	}
	comprobante, err := s.consumidorRecibo.ConsumirReciboCargaDirecta(ctx, peticionConsumo)
	if err != nil {
		if errors.Is(err, ports.ErrReciboCargaDirectaNoValido) {
			return domain.CargaDocumental{}, ports.ErrReciboCargaDirectaNoValido
		}
		return domain.CargaDocumental{}, ports.ErrReciboCargaDirectaNoDisponible
	}
	confirmacionAlmacen, err := ports.NuevaSolicitudConfirmarCargaDirecta(
		ctx, contextoConfirmacion, orden.SesionRef, comprobante, s.verificadorRecibo,
	)
	if err != nil {
		return domain.CargaDocumental{}, errorConfirmacionCargaDocumentalPendiente(err)
	}
	antesDelEfecto := s.reloj.Ahora().UTC()
	if contextoConfirmacion.ValidarParaEn(ports.AccionAlmacenConfirmarCargaDirecta, antesDelEfecto) != nil ||
		!validaHasta.After(antesDelEfecto) {
		return domain.CargaDocumental{}, errorConfirmacionCargaDocumentalPendiente(domain.ErrAutorizacionDenegada)
	}
	resultado, err := s.gestorCargaDirecta.ConfirmarCargaDirecta(ctx, confirmacionAlmacen)
	if err != nil {
		return domain.CargaDocumental{}, errorConfirmacionCargaDocumentalPendiente(
			ports.ErrConfirmacionCargaDirectaNoDisponible,
		)
	}
	if ports.ValidarResultadoCargaDirectaConManifiesto(
		resultado, manifiesto, carga, confirmacionAlmacen, capacidades, orden.Recurso,
	) != nil {
		return domain.CargaDocumental{}, errorConfirmacionCargaDocumentalPendiente(
			ErrResultadoCargaDocumentalInvalido,
		)
	}
	instante := s.reloj.Ahora().UTC()
	contenido, err := contenidoCargaDesdeResultado(resultado, domain.ZonaContenidoCargaCuarentena)
	if err != nil {
		return domain.CargaDocumental{}, errorConfirmacionCargaDocumentalPendiente(err)
	}
	recibida, err := carga.RegistrarCuarentena(contenido, decision.DecisionRef, instante)
	if err != nil {
		return domain.CargaDocumental{}, errorConfirmacionCargaDocumentalPendiente(err)
	}
	huellaManifiesto, err := manifiesto.HuellaSHA256()
	if err != nil {
		return domain.CargaDocumental{}, errorConfirmacionCargaDocumentalPendiente(err)
	}
	confirmacion, err := confirmacionCargaDocumental(
		carga, recibida, orden.Principal, orden.PerfilActivo, decision.DecisionRef, orden.Motivo,
		AccionConfirmarCargaDocumental, eventoCargaDocumentalRecibida, instante,
		map[string]string{
			"almacen_conector": resultado.Objeto.ConectorID, "estado": string(recibida.Estado),
			"evidencia_almacen_ref":         resultado.Evidencia.Referencia,
			"manifiesto_preparacion_sha256": huellaManifiesto,
		},
	)
	if err != nil {
		return domain.CargaDocumental{}, errorConfirmacionCargaDocumentalPendiente(err)
	}
	ctxPersistencia, cancelarPersistencia := contextoPersistenciaCarga(ctx)
	defer cancelarPersistencia()
	if err := s.repositorio.ConfirmarTransicion(ctxPersistencia, confirmacion); err != nil {
		return domain.CargaDocumental{}, errorConfirmacionCargaDocumentalPendiente(err)
	}
	return recibida, nil
}

type OrdenAnalizarCargaDocumental struct {
	Principal      domain.Principal
	PerfilActivo   string
	Recurso        domain.RecursoAutorizable
	CargaID        string
	Finalidad      string
	Motivo         string
	CorrelacionRef string
}

// AnalizarYPromover conserva primero el resultado del analisis. Solo despues
// obtiene una segunda autorizacion independiente para promocionar contenido
// limpio; error, sospecha o falta de conclusion permanecen en cuarentena.
func (s *ServicioCargaDocumental) AnalizarYPromover(
	ctx context.Context,
	orden OrdenAnalizarCargaDocumental,
) (domain.CargaDocumental, error) {
	if s == nil || ctx == nil {
		return domain.CargaDocumental{}, ErrDependenciaCargaDocumentalRequerida
	}
	orden = clonarOrdenAnalizarCargaDocumental(orden)
	if err := validarOrdenAnalizarCargaDocumental(orden); err != nil {
		return domain.CargaDocumental{}, err
	}
	carga, err := s.repositorio.Obtener(ctx, orden.CargaID)
	if err != nil {
		return domain.CargaDocumental{}, err
	}
	if err := validarRecursoCarga(carga, orden.Recurso, orden.Finalidad, orden.CorrelacionRef); err != nil {
		return domain.CargaDocumental{}, err
	}
	if carga.Estado == domain.EstadoCargaDocumentalAnalizadaLimpia && carga.ContenidoCuarentena != nil && carga.Analisis != nil {
		capacidadesAlmacen, err := s.capacidadesAlmacenAnalisis(ctx, carga.TamanoDeclarado)
		if err != nil {
			return domain.CargaDocumental{}, err
		}
		return s.promoverCargaLimpia(ctx, orden, carga, capacidadesAlmacen)
	}
	if carga.Estado != domain.EstadoCargaDocumentalCuarentena || carga.ContenidoCuarentena == nil {
		return domain.CargaDocumental{}, ErrCargaDocumentalYaProcesada
	}
	objeto := objetoAlmacenDesdeContenido(*carga.ContenidoCuarentena)
	seudonimoSujeto, err := s.seudonimizarSujetoCarga(ctx, carga)
	if err != nil {
		return domain.CargaDocumental{}, err
	}
	vinculosAnalisis, err := vinculosAnalizarCargaDocumental(carga, seudonimoSujeto, objeto)
	if err != nil {
		return domain.CargaDocumental{}, err
	}
	recursoAutorizable, err := recursoAnalizarCargaDocumental(orden.Recurso, vinculosAnalisis)
	if err != nil {
		return domain.CargaDocumental{}, err
	}
	decisionAnalisis, err := exigirDecisionAutorizacion(
		ctx, s.autorizador, s.reloj, orden.Principal, orden.PerfilActivo, AccionAnalizarCargaDocumental,
		recursoAutorizable, orden.Finalidad, orden.CorrelacionRef, orden.Motivo, usoCamposDecisionConsumidos,
	)
	if err != nil {
		return domain.CargaDocumental{}, err
	}
	if err := validarAlcanceDecisionCarga(decisionAnalisis, camposDecisionAnalizarCarga()); err != nil {
		return domain.CargaDocumental{}, err
	}
	verificadaEn := s.reloj.Ahora().UTC()
	contextoLectura, err := ports.NuevoContextoAnalizarCargaDocumentalAlmacen(
		decisionAnalisis, recursoAutorizable, vinculosAnalisis, verificadaEn,
	)
	if err != nil || contextoLectura.ValidarParaEn(ports.AccionAlmacenLeer, verificadaEn) != nil {
		return domain.CargaDocumental{}, errors.Join(domain.ErrAutorizacionDenegada, err)
	}
	contextoAnalisis, err := contextoLectura.DerivarPaso(ports.PasoAlmacenAnalizarContenido)
	if err != nil {
		return domain.CargaDocumental{}, errors.Join(domain.ErrAutorizacionDenegada, err)
	}
	capacidadesAlmacen, err := s.capacidadesAlmacenAnalisis(ctx, carga.TamanoDeclarado)
	if err != nil {
		return domain.CargaDocumental{}, err
	}
	capacidadesAnalizador, err := s.analizador.Capacidades(ctx)
	if err != nil {
		return domain.CargaDocumental{}, err
	}
	if err := ports.VerificarCapacidadesAnalizadorContenido(capacidadesAnalizador, ports.RequisitosAnalizadorContenido{
		AnalisisEnFlujo: true, CanalAutenticado: true, CifradoEnTransito: true, IdentidadMutua: true,
		ActualizacionFirmas: true, DetectaMalware: true, DetectaContenidoActivo: true, TamanoMinimo: carga.TamanoDeclarado,
	}); err != nil {
		return domain.CargaDocumental{}, err
	}
	if capacidadesAnalizador.ConectorID != s.conectorAnalizador ||
		capacidadesAnalizador.VersionConector != s.versionAnalizador {
		return domain.CargaDocumental{}, ports.ErrCapacidadAnalisisContenidoNoDisponible
	}
	solicitudAbrir := ports.SolicitudAbrirObjeto{
		Contexto: contextoLectura, Objeto: objeto, Zona: ports.ZonaAlmacenCuarentena, Limite: carga.TamanoDeclarado,
	}
	instanteLectura := s.reloj.Ahora().UTC()
	if contextoLectura.ValidarParaEn(ports.AccionAlmacenLeer, instanteLectura) != nil {
		return domain.CargaDocumental{}, domain.ErrAutorizacionDenegada
	}
	lectura, err := s.almacen.Abrir(ctx, solicitudAbrir)
	if err != nil {
		return domain.CargaDocumental{}, err
	}
	if lectura.ValidarContra(solicitudAbrir) != nil || !objetoAlmacenCorrespondeCarga(lectura.Objeto, *carga.ContenidoCuarentena) {
		_ = lectura.Contenido.Close()
		return domain.CargaDocumental{}, ErrResultadoCargaDocumentalInvalido
	}
	verificador := nuevoLectorVerificadorCarga(lectura.Contenido, carga.TamanoDeclarado)
	solicitudAnalisis := ports.SolicitudAnalizarContenido{
		Contexto: contextoAnalisis, Objeto: objeto, ConectorAlmacenID: lectura.Objeto.ConectorID,
		Zona: ports.ZonaAlmacenCuarentena, MIME: lectura.Objeto.MIME, Tamano: lectura.Objeto.Tamano,
		HuellaSHA256: lectura.Objeto.HuellaSHA256, Contenido: verificador,
	}
	iniciadoServidor := s.reloj.Ahora().UTC()
	if contextoAnalisis.ValidarParaEn(ports.AccionAlmacenAnalizarContenido, iniciadoServidor) != nil {
		_ = lectura.Contenido.Close()
		return domain.CargaDocumental{}, domain.ErrAutorizacionDenegada
	}
	resultadoAnalisis, errAnalisis := s.analizador.Analizar(ctx, solicitudAnalisis)
	resultadoConcluyente := resultadoAnalisis.Estado == ports.EstadoAnalisisContenidoLimpio ||
		resultadoAnalisis.Estado == ports.EstadoAnalisisContenidoMalicioso ||
		resultadoAnalisis.Estado == ports.EstadoAnalisisContenidoSospechoso
	finExacto := false
	if errAnalisis == nil && resultadoConcluyente {
		finExacto = verificador.FinExacto()
	}
	errCierre := lectura.Contenido.Close()
	completadoServidor := s.reloj.Ahora().UTC()
	if errAnalisis != nil {
		return domain.CargaDocumental{}, errAnalisis
	}
	if errCierre != nil {
		return domain.CargaDocumental{}, errCierre
	}
	lecturaCompletaValida := verificador.BytesLeidos() == carga.TamanoDeclarado &&
		verificador.HuellaSHA256() == carga.HuellaDeclaradaSHA256 && finExacto
	if resultadoAnalisis.ValidarContra(solicitudAnalisis) != nil ||
		resultadoAnalisis.ConectorAnalizadorID != capacidadesAnalizador.ConectorID ||
		resultadoAnalisis.VersionConector != capacidadesAnalizador.VersionConector || verificador.Excedido() ||
		verificador.BytesLeidos() != resultadoAnalisis.BytesAnalizados ||
		(resultadoConcluyente && !lecturaCompletaValida) ||
		resultadoAnalisis.AnalisisIniciadoEn.Before(iniciadoServidor.Add(-toleranciaRelojAnalizador)) ||
		resultadoAnalisis.AnalisisCompletadoEn.After(completadoServidor.Add(toleranciaRelojAnalizador)) {
		return domain.CargaDocumental{}, ErrResultadoCargaDocumentalInvalido
	}
	analisisDominio, err := analisisCargaDesdeResultado(resultadoAnalisis, completadoServidor)
	if err != nil {
		return domain.CargaDocumental{}, err
	}
	analizada, err := carga.RegistrarAnalisis(analisisDominio, decisionAnalisis.DecisionRef, completadoServidor)
	if err != nil {
		return domain.CargaDocumental{}, err
	}
	confirmacionAnalisis, err := confirmacionCargaDocumental(
		carga, analizada, orden.Principal, orden.PerfilActivo, decisionAnalisis.DecisionRef, orden.Motivo,
		AccionAnalizarCargaDocumental, eventoCargaDocumentalAnalizada, completadoServidor,
		map[string]string{"analizador_conector": capacidadesAnalizador.ConectorID, "estado": string(analizada.Estado),
			"evidencia_analisis_ref": resultadoAnalisis.EvidenciaRef},
	)
	if err != nil {
		return domain.CargaDocumental{}, err
	}
	ctxPersistencia, cancelarPersistencia := contextoPersistenciaCarga(ctx)
	defer cancelarPersistencia()
	if err := s.repositorio.ConfirmarTransicion(ctxPersistencia, confirmacionAnalisis); err != nil {
		return domain.CargaDocumental{}, err
	}
	if analizada.Estado != domain.EstadoCargaDocumentalAnalizadaLimpia {
		return analizada, nil
	}
	return s.promoverCargaLimpia(ctx, orden, analizada, capacidadesAlmacen)
}

func (s *ServicioCargaDocumental) promoverCargaLimpia(
	ctx context.Context,
	orden OrdenAnalizarCargaDocumental,
	carga domain.CargaDocumental,
	capacidades ports.CapacidadesAlmacenObjetos,
) (domain.CargaDocumental, error) {
	if carga.Validar() != nil || carga.Estado != domain.EstadoCargaDocumentalAnalizadaLimpia ||
		carga.ContenidoCuarentena == nil || carga.Analisis == nil {
		return domain.CargaDocumental{}, ErrCargaDocumentalYaProcesada
	}
	indicePromocion, err := s.selladorIdempotencia.SellarIdempotenciaCarga(ctx, ports.SolicitudSellarIdempotenciaCarga{
		OperacionRef: "promover:" + carga.ID, PrincipalRef: orden.Principal.ID, RecursoRef: carga.RecursoRef,
		MIME: carga.MIMEDeclarado, Tamano: carga.TamanoDeclarado, HuellaSHA256: carga.HuellaDeclaradaSHA256,
		ClaveSolicitante: carga.Analisis.EvidenciaRef,
	})
	if err != nil || !hmacCargaDocumentalValido(indicePromocion) {
		return domain.CargaDocumental{}, ports.ErrSelladoIdempotenciaCargaNoDisponible
	}
	seudonimoSujeto, err := s.seudonimizarSujetoCarga(ctx, carga)
	if err != nil {
		return domain.CargaDocumental{}, err
	}
	objetoOrigen := objetoAlmacenDesdeContenido(*carga.ContenidoCuarentena)
	vinculosPromocion, err := vinculosPromoverCargaDocumental(carga, seudonimoSujeto, objetoOrigen)
	if err != nil {
		return domain.CargaDocumental{}, err
	}
	recursoAutorizable, err := recursoPromoverCargaDocumental(orden.Recurso, vinculosPromocion)
	if err != nil {
		return domain.CargaDocumental{}, err
	}
	decision, err := exigirDecisionAutorizacion(
		ctx, s.autorizador, s.reloj, orden.Principal, orden.PerfilActivo, AccionPromoverCargaDocumental,
		recursoAutorizable, orden.Finalidad, orden.CorrelacionRef, orden.Motivo, usoCamposDecisionConsumidos,
	)
	if err != nil {
		return domain.CargaDocumental{}, err
	}
	if err := validarAlcanceDecisionCarga(decision, camposDecisionPromoverCarga()); err != nil {
		return domain.CargaDocumental{}, err
	}
	verificadaEn := s.reloj.Ahora().UTC()
	contexto, err := ports.NuevoContextoPromoverCargaDocumentalAlmacen(
		decision, recursoAutorizable, vinculosPromocion, verificadaEn,
	)
	if err != nil || contexto.ValidarParaEn(ports.AccionAlmacenPromover, verificadaEn) != nil {
		return domain.CargaDocumental{}, errors.Join(domain.ErrAutorizacionDenegada, err)
	}
	solicitud := ports.SolicitudPromoverObjeto{
		Contexto: contexto, ClaveIdempotencia: indicePromocion,
		Origen: objetoOrigen, EvidenciaAnalisisRef: carga.Analisis.EvidenciaRef,
	}
	instantePromocion := s.reloj.Ahora().UTC()
	if contexto.ValidarParaEn(ports.AccionAlmacenPromover, instantePromocion) != nil {
		return domain.CargaDocumental{}, domain.ErrAutorizacionDenegada
	}
	resultado, err := s.almacen.Promover(ctx, solicitud)
	if err != nil {
		return domain.CargaDocumental{}, err
	}
	origen := objetoAlmacenadoDesdeContenido(*carga.ContenidoCuarentena)
	if resultado.ValidarPromocion(solicitud, origen, capacidades) != nil {
		return domain.CargaDocumental{}, ErrResultadoCargaDocumentalInvalido
	}
	instante := s.reloj.Ahora().UTC()
	contenido, err := contenidoCargaDesdeResultado(resultado, domain.ZonaContenidoCargaAdmitida)
	if err != nil {
		return domain.CargaDocumental{}, err
	}
	admitida, err := carga.Admitir(contenido, decision.DecisionRef, instante)
	if err != nil {
		return domain.CargaDocumental{}, err
	}
	confirmacion, err := confirmacionCargaDocumental(
		carga, admitida, orden.Principal, orden.PerfilActivo, decision.DecisionRef, orden.Motivo,
		AccionPromoverCargaDocumental, eventoCargaDocumentalPromovida, instante,
		map[string]string{"almacen_conector": resultado.Objeto.ConectorID, "estado": string(admitida.Estado),
			"evidencia_promocion_ref": resultado.Evidencia.Referencia},
	)
	if err != nil {
		return domain.CargaDocumental{}, err
	}
	ctxPersistencia, cancelarPersistencia := contextoPersistenciaCarga(ctx)
	defer cancelarPersistencia()
	if err := s.repositorio.ConfirmarTransicion(ctxPersistencia, confirmacion); err != nil {
		return domain.CargaDocumental{}, err
	}
	return admitida, nil
}

func (s *ServicioCargaDocumental) capacidadesAlmacenCargaDirecta(
	ctx context.Context,
	tamano int64,
) (ports.CapacidadesAlmacenObjetos, error) {
	capacidades, err := s.almacen.Capacidades(ctx)
	if err != nil {
		return ports.CapacidadesAlmacenObjetos{}, ports.ErrCapacidadAlmacenNoDisponible
	}
	err = ports.VerificarCapacidadesAlmacen(capacidades, ports.RequisitosAlmacenObjetos{
		LecturaEnFlujo: true, ReferenciasOpacas: true, IntegridadSHA256: true, Versionado: true,
		PromocionAtomica: true, CargaDirectaTemporal: true, CifradoEnTransito: true, CifradoEnReposo: true,
		CifradoPorObjeto: true, TamanoMinimoObjeto: tamano, PreservaObjetoOriginal: true,
	})
	if err != nil {
		return ports.CapacidadesAlmacenObjetos{}, err
	}
	if capacidades.ConectorID != s.conectorAlmacen {
		return ports.CapacidadesAlmacenObjetos{}, ports.ErrCapacidadAlmacenNoDisponible
	}
	return capacidades, nil
}

func (s *ServicioCargaDocumental) capacidadesAlmacenAnalisis(
	ctx context.Context,
	tamano int64,
) (ports.CapacidadesAlmacenObjetos, error) {
	capacidades, err := s.almacen.Capacidades(ctx)
	if err != nil {
		return ports.CapacidadesAlmacenObjetos{}, err
	}
	err = ports.VerificarCapacidadesAlmacen(capacidades, ports.RequisitosAlmacenObjetos{
		LecturaEnFlujo: true, ReferenciasOpacas: true, IntegridadSHA256: true, Versionado: true,
		PromocionAtomica: true, CifradoEnTransito: true, CifradoEnReposo: true, CifradoPorObjeto: true,
		TamanoMinimoObjeto: tamano, PreservaObjetoOriginal: true,
	})
	if err != nil {
		return ports.CapacidadesAlmacenObjetos{}, err
	}
	if capacidades.ConectorID != s.conectorAlmacen {
		return ports.CapacidadesAlmacenObjetos{}, ports.ErrCapacidadAlmacenNoDisponible
	}
	return capacidades, nil
}

func contextoPersistenciaCarga(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
}

func errorConfirmacionCargaDocumentalPendiente(causa error) error {
	if causa == nil {
		causa = ports.ErrConfirmacionCargaDirectaNoDisponible
	}
	return errors.Join(ports.ErrConfirmacionCargaDocumentalPendiente, causa)
}

func camposDecisionPrepararCarga() []string {
	return []string{CampoCargaClasificacion, CampoCargaContenido, CampoCargaHuellaSHA256, CampoCargaMIME, CampoCargaTamano}
}

func camposDecisionConfirmarCarga() []string {
	return []string{CampoCargaContenidoCuarentena, CampoCargaEstado}
}

func camposDecisionAnalizarCarga() []string {
	return []string{CampoCargaAnalisisSeguridad, CampoCargaEstado}
}

func camposDecisionPromoverCarga() []string {
	return []string{CampoCargaContenidoAdmitido, CampoCargaEstado}
}

func validarAlcanceDecisionCarga(decision domain.DecisionAutorizacion, camposNecesarios []string) error {
	if len(decision.Obligaciones) != 0 || len(decision.CamposPermitidos) != len(camposNecesarios) {
		return errors.Join(domain.ErrAutorizacionDenegada, domain.ErrDecisionAutorizacionInvalida)
	}
	permitidos := make(map[string]struct{}, len(decision.CamposPermitidos))
	for _, campo := range decision.CamposPermitidos {
		permitidos[campo] = struct{}{}
	}
	for _, campo := range camposNecesarios {
		if _, permitido := permitidos[campo]; !permitido {
			return errors.Join(domain.ErrAutorizacionDenegada, domain.ErrDecisionAutorizacionInvalida)
		}
	}
	return nil
}

func (s *ServicioCargaDocumental) abandonarReserva(ctx context.Context, token ports.TokenReservaCargaDocumental) {
	ctxLimpieza, cancelar := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancelar()
	_ = s.repositorio.AbandonarReserva(ctxLimpieza, token)
}

func dependenciaCargaDocumentalNula(dependencia any) bool {
	if dependencia == nil {
		return true
	}
	valor := reflect.ValueOf(dependencia)
	switch valor.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return valor.IsNil()
	default:
		return false
	}
}

// mismaProteccionRecibosCargaDirecta exige que emision, consumo y verificacion
// procedan del mismo objeto ensamblado. Asi no pueden mezclarse por error
// claves de raices distintas. Los dobles de contrato deben representar esa
// misma cohesion mediante una unica instancia, igual que el adaptador real.
func mismaProteccionRecibosCargaDirecta(
	emisor ports.EmisorReciboCargaDirecta,
	consumidor ports.ConsumidorReciboCargaDirecta,
	verificador ports.VerificadorAtestacionConsumoReciboCargaDirecta,
) bool {
	dependencias := []any{emisor, consumidor, verificador}
	var tipo reflect.Type
	var puntero uintptr
	for indice, dependencia := range dependencias {
		if dependenciaCargaDocumentalNula(dependencia) {
			return false
		}
		valor := reflect.ValueOf(dependencia)
		if valor.Kind() != reflect.Pointer {
			return false
		}
		if indice == 0 {
			tipo = valor.Type()
			puntero = valor.Pointer()
			continue
		}
		if valor.Type() != tipo || valor.Pointer() != puntero {
			return false
		}
	}
	return puntero != 0
}

func validarOrdenPrepararCargaDocumental(orden OrdenPrepararCargaDocumental, maximo int64) error {
	if orden.Principal.Validate() != nil || orden.Recurso.Validar() != nil ||
		!textoCargaSeguro(orden.PerfilActivo, 512, false) || !textoCargaSeguro(orden.OperacionRef, 512, false) ||
		!textoCargaSeguro(orden.Finalidad, 512, false) || !textoCargaSeguro(orden.Clasificacion, 128, false) ||
		!mimeCargaValido(orden.MIME) || orden.Tamano < 1 || orden.Tamano > maximo ||
		!esSHA256Carga(orden.HuellaSHA256) || !textoCargaSeguro(orden.ClaveIdempotenciaCliente, 512, false) ||
		!textoCargaSeguro(orden.Motivo, 1024, true) || !textoCargaSeguro(orden.CorrelacionRef, 512, false) {
		return ErrOrdenCargaDocumentalInvalida
	}
	return nil
}

func validarOrdenConfirmarCargaDocumental(orden OrdenConfirmarCargaDocumental) error {
	if orden.Principal.Validate() != nil || orden.Recurso.Validar() != nil ||
		!textoCargaSeguro(orden.PerfilActivo, 512, false) || !textoCargaSeguro(orden.CargaID, 512, false) ||
		!textoCargaSeguro(orden.SesionRef, 512, false) || !orden.Recibo.Valido() ||
		!textoCargaSeguro(orden.Finalidad, 512, false) ||
		!textoCargaSeguro(orden.Motivo, 1024, true) || !textoCargaSeguro(orden.CorrelacionRef, 512, false) {
		return ErrOrdenCargaDocumentalInvalida
	}
	return nil
}

func validarOrdenAnalizarCargaDocumental(orden OrdenAnalizarCargaDocumental) error {
	if orden.Principal.Validate() != nil || orden.Recurso.Validar() != nil ||
		!textoCargaSeguro(orden.PerfilActivo, 512, false) || !textoCargaSeguro(orden.CargaID, 512, false) ||
		!textoCargaSeguro(orden.Finalidad, 512, false) || !textoCargaSeguro(orden.Motivo, 1024, true) ||
		!textoCargaSeguro(orden.CorrelacionRef, 512, false) {
		return ErrOrdenCargaDocumentalInvalida
	}
	return nil
}

func validarContextoCarga(
	carga domain.CargaDocumental,
	principal domain.Principal,
	recurso domain.RecursoAutorizable,
	finalidad, correlacionRef string,
) error {
	if carga.Validar() != nil || principal.Validate() != nil || recurso.Validar() != nil ||
		carga.PrincipalID != principal.ID || carga.RecursoRef != recurso.Referencia ||
		carga.ModuloID != recurso.ModuloID || carga.TipoRecurso != recurso.Tipo ||
		carga.Finalidad != finalidad || carga.CorrelacionRef != correlacionRef {
		return ErrCargaDocumentalNoCorresponde
	}
	return nil
}

func validarRecursoCarga(
	carga domain.CargaDocumental,
	recurso domain.RecursoAutorizable,
	finalidad, correlacionRef string,
) error {
	if carga.Validar() != nil || recurso.Validar() != nil ||
		carga.RecursoRef != recurso.Referencia ||
		carga.ModuloID != recurso.ModuloID || carga.TipoRecurso != recurso.Tipo ||
		carga.Finalidad != finalidad || carga.CorrelacionRef != correlacionRef {
		return ErrCargaDocumentalNoCorresponde
	}
	return nil
}

func (s *ServicioCargaDocumental) seudonimizarSujetoCarga(
	ctx context.Context,
	carga domain.CargaDocumental,
) (string, error) {
	if carga.Validar() != nil {
		return "", ErrResultadoCargaDocumentalInvalido
	}
	solicitud, err := ports.NuevaSolicitudSeudonimizarSujetoAlmacen(
		carga.PrincipalID,
		"carga_documental:"+carga.HuellaSolicitudHMAC,
	)
	if err != nil {
		return "", err
	}
	seudonimo, err := s.seudonimizadorSujeto.SeudonimizarSujetoAlmacen(ctx, solicitud)
	if err != nil || !hmacCargaDocumentalValido(seudonimo) {
		return "", ports.ErrSeudonimizacionAlmacenNoDisponible
	}
	return seudonimo, nil
}

func vinculosPrepararCargaDocumental(
	carga domain.CargaDocumental,
	seudonimoSujeto string,
) (ports.VinculosOperacionAlmacen, error) {
	return nuevosVinculosCargaDocumental(
		carga, AccionPrepararCargaDocumental, carga.IndiceIdempotenciaHMAC,
		seudonimoSujeto, ports.ReferenciaObjetoAlmacen{},
	)
}

func vinculosConfirmarCargaDocumental(
	carga domain.CargaDocumental,
	seudonimoSujeto string,
) (ports.VinculosOperacionAlmacen, error) {
	return nuevosVinculosCargaDocumental(
		carga, AccionConfirmarCargaDocumental, carga.IndiceIdempotenciaHMAC,
		seudonimoSujeto, ports.ReferenciaObjetoAlmacen{},
	)
}

func vinculosAnalizarCargaDocumental(
	carga domain.CargaDocumental,
	seudonimoSujeto string,
	objeto ports.ReferenciaObjetoAlmacen,
) (ports.VinculosOperacionAlmacen, error) {
	return nuevosVinculosCargaDocumental(
		carga, AccionAnalizarCargaDocumental, carga.ID, seudonimoSujeto, objeto,
	)
}

func vinculosPromoverCargaDocumental(
	carga domain.CargaDocumental,
	seudonimoSujeto string,
	objeto ports.ReferenciaObjetoAlmacen,
) (ports.VinculosOperacionAlmacen, error) {
	return nuevosVinculosCargaDocumental(
		carga, AccionPromoverCargaDocumental, carga.ID, seudonimoSujeto, objeto,
	)
}

// nuevosVinculosCargaDocumental solo compone datos no autoritativos. La
// capacidad se acuna exclusivamente mediante las fabricas especificas del
// puerto despues de que el PDP haya evaluado el recurso enriquecido exacto.
func nuevosVinculosCargaDocumental(
	carga domain.CargaDocumental,
	accionNegocio, cargaRef, seudonimoSujeto string,
	objeto ports.ReferenciaObjetoAlmacen,
) (ports.VinculosOperacionAlmacen, error) {
	requiereObjeto := false
	switch accionNegocio {
	case AccionPrepararCargaDocumental, AccionConfirmarCargaDocumental:
		if objeto != (ports.ReferenciaObjetoAlmacen{}) {
			return ports.VinculosOperacionAlmacen{}, errorAutorizacionCargaDocumental()
		}
	case AccionAnalizarCargaDocumental, AccionPromoverCargaDocumental:
		requiereObjeto = true
	default:
		return ports.VinculosOperacionAlmacen{}, errorAutorizacionCargaDocumental()
	}
	if carga.Validar() != nil || !textoCargaSeguro(cargaRef, 512, false) ||
		!hmacCargaDocumentalValido(seudonimoSujeto) ||
		strings.ContainsRune(carga.OperacionRef, '*') || strings.ContainsRune(cargaRef, '*') ||
		strings.ContainsRune(carga.Clasificacion, '*') || strings.ContainsRune(carga.HuellaSolicitudHMAC, '*') ||
		(requiereObjeto && (objeto.Validar() != nil || strings.ContainsRune(objeto.Referencia, '*') ||
			strings.ContainsRune(objeto.Version, '*'))) {
		return ports.VinculosOperacionAlmacen{}, errorAutorizacionCargaDocumental()
	}
	efectoRef, err := referenciaEfectoCargaDocumental(accionNegocio, carga, cargaRef, objeto)
	if err != nil {
		return ports.VinculosOperacionAlmacen{}, err
	}
	return ports.VinculosOperacionAlmacen{
		OperacionRef: carga.OperacionRef, CargaRef: cargaRef, Clasificacion: carga.Clasificacion,
		SujetoSeudonimoHMAC: seudonimoSujeto, HuellaSolicitudHMAC: carga.HuellaSolicitudHMAC,
		EfectoRef: efectoRef, ObjetoVinculado: objeto,
	}, nil
}

func referenciaEfectoCargaDocumental(
	accionNegocio string,
	carga domain.CargaDocumental,
	cargaRef string,
	objeto ports.ReferenciaObjetoAlmacen,
) (string, error) {
	switch accionNegocio {
	case AccionPrepararCargaDocumental, AccionConfirmarCargaDocumental,
		AccionAnalizarCargaDocumental, AccionPromoverCargaDocumental:
	default:
		return "", errorAutorizacionCargaDocumental()
	}
	valores := []string{
		"vec.carga-documental.efecto.v1", accionNegocio, carga.OperacionRef,
		cargaRef, carga.HuellaSolicitudHMAC, objeto.Referencia, objeto.Version,
	}
	var canonico strings.Builder
	for _, valor := range valores {
		canonico.WriteString(strconv.Itoa(len(valor)))
		canonico.WriteByte(':')
		canonico.WriteString(valor)
		canonico.WriteByte('\n')
	}
	huella := sha256.Sum256([]byte(canonico.String()))
	return "efecto-carga:sha256:" + hex.EncodeToString(huella[:]), nil
}

func recursoPrepararCargaDocumental(
	recurso domain.RecursoAutorizable,
	vinculos ports.VinculosOperacionAlmacen,
) (domain.RecursoAutorizable, error) {
	return enriquecerRecursoCargaDocumental(recurso, vinculos, false)
}

func recursoConfirmarCargaDocumental(
	recurso domain.RecursoAutorizable,
	vinculos ports.VinculosOperacionAlmacen,
) (domain.RecursoAutorizable, error) {
	return enriquecerRecursoCargaDocumental(recurso, vinculos, false)
}

func recursoAnalizarCargaDocumental(
	recurso domain.RecursoAutorizable,
	vinculos ports.VinculosOperacionAlmacen,
) (domain.RecursoAutorizable, error) {
	return enriquecerRecursoCargaDocumental(recurso, vinculos, true)
}

func recursoPromoverCargaDocumental(
	recurso domain.RecursoAutorizable,
	vinculos ports.VinculosOperacionAlmacen,
) (domain.RecursoAutorizable, error) {
	return enriquecerRecursoCargaDocumental(recurso, vinculos, true)
}

func enriquecerRecursoCargaDocumental(
	recurso domain.RecursoAutorizable,
	vinculos ports.VinculosOperacionAlmacen,
	requiereObjeto bool,
) (domain.RecursoAutorizable, error) {
	if _, err := ports.HuellaRecursoBaseCargaDocumental(recurso); err != nil {
		return domain.RecursoAutorizable{}, errors.Join(errorAutorizacionCargaDocumental(), err)
	}
	if requiereObjeto {
		if vinculos.ObjetoVinculado.Validar() != nil {
			return domain.RecursoAutorizable{}, errorAutorizacionCargaDocumental()
		}
	} else if vinculos.ObjetoVinculado != (ports.ReferenciaObjetoAlmacen{}) {
		return domain.RecursoAutorizable{}, errorAutorizacionCargaDocumental()
	}
	resultado := domain.RecursoAutorizable{
		Referencia: recurso.Referencia, ModuloID: recurso.ModuloID, Tipo: recurso.Tipo,
		Ambitos: copiarMapaCarga(recurso.Ambitos), Atributos: copiarMapaCarga(recurso.Atributos),
	}
	resultado.Atributos[ports.AtributoAlmacenOperacionRef] = vinculos.OperacionRef
	resultado.Atributos[ports.AtributoAlmacenCargaRef] = vinculos.CargaRef
	resultado.Atributos[ports.AtributoAlmacenClasificacion] = vinculos.Clasificacion
	resultado.Atributos[ports.AtributoAlmacenSujetoSeudonimoHMAC] = vinculos.SujetoSeudonimoHMAC
	resultado.Atributos[ports.AtributoAlmacenHuellaSolicitudHMAC] = vinculos.HuellaSolicitudHMAC
	resultado.Atributos[ports.AtributoAlmacenEfectoRef] = vinculos.EfectoRef
	if requiereObjeto {
		resultado.Atributos[ports.AtributoAlmacenObjetoRef] = vinculos.ObjetoVinculado.Referencia
		resultado.Atributos[ports.AtributoAlmacenObjetoVersion] = vinculos.ObjetoVinculado.Version
	}
	if resultado.Validar() != nil {
		return domain.RecursoAutorizable{}, errorAutorizacionCargaDocumental()
	}
	return resultado, nil
}

func errorAutorizacionCargaDocumental() error {
	return errors.Join(domain.ErrAutorizacionDenegada, ports.ErrAutorizacionAlmacenInvalida)
}

func solicitudPreparacionCarga(
	carga domain.CargaDocumental,
	contexto ports.ContextoOperacionAlmacen,
) ports.SolicitudPrepararCargaDirecta {
	return ports.SolicitudPrepararCargaDirecta{
		Contexto: contexto, ClaveIdempotencia: carga.IndiceIdempotenciaHMAC, MIME: carga.MIMEDeclarado,
		Tamano: carga.TamanoDeclarado, HuellaSHA256: carga.HuellaDeclaradaSHA256, ExpiraEn: carga.ExpiraEn,
	}
}

func manifiestoPreparacionCargaDocumental(
	carga domain.CargaDocumental,
	contexto ports.ContextoOperacionAlmacen,
	capacidades ports.CapacidadesAlmacenObjetos,
	huellaRecursoBaseSHA256 string,
) (domain.ManifiestoPreparacionCargaDirectaV1, error) {
	proyeccion, err := contexto.Proyeccion()
	if err != nil || contexto.ValidarParaEn(ports.AccionAlmacenPrepararCargaDirecta, carga.PreparadaEn) != nil {
		return domain.ManifiestoPreparacionCargaDirectaV1{}, domain.ErrManifiestoPreparacionInvalido
	}
	evidencia, err := contexto.EvidenciaAutorizacion()
	if err != nil {
		return domain.ManifiestoPreparacionCargaDirectaV1{}, domain.ErrManifiestoPreparacionInvalido
	}
	datosEvidencia, err := evidencia.Datos()
	if err != nil || datosEvidencia.HuellaDecisionSHA256 != proyeccion.HuellaDecisionSHA256 ||
		datosEvidencia.Decision.DecisionRef != proyeccion.AutorizacionRef ||
		datosEvidencia.Decision.Accion != proyeccion.AccionNegocio ||
		datosEvidencia.Decision.ContextoRecursoHuellaSHA256 != proyeccion.HuellaRecursoSHA256 ||
		capacidades.ConectorID == "" || !esSHA256Carga(huellaRecursoBaseSHA256) {
		return domain.ManifiestoPreparacionCargaDirectaV1{}, domain.ErrManifiestoPreparacionInvalido
	}
	manifiesto, err := domain.NuevoManifiestoPreparacionCargaDirectaV1(
		carga,
		domain.ContextoManifiestoPreparacionCargaDirectaV1{
			CargaRef: proyeccion.CargaRef, SujetoSeudonimoHMAC: proyeccion.SujetoSeudonimoHMAC,
			HuellaRecursoBaseSHA256: huellaRecursoBaseSHA256,
			HuellaRecursoSHA256:     proyeccion.HuellaRecursoSHA256,
			ConectorAlmacenID:       capacidades.ConectorID, EsquemaContexto: proyeccion.Esquema,
			AccionNegocio: proyeccion.AccionNegocio, AccionTecnica: proyeccion.AccionTecnica,
			PasoRef: string(proyeccion.PasoRef), EfectoRef: proyeccion.EfectoRef,
			HuellaPlanEfectoSHA256: proyeccion.HuellaPlanEfectoSHA256,
			EsquemaHuellaDecision:  datosEvidencia.EsquemaHuella,
			DecisionRef:            proyeccion.AutorizacionRef,
			HuellaDecisionSHA256:   proyeccion.HuellaDecisionSHA256,
			ContextoVerificadoEn:   proyeccion.VerificadaEn,
			DecisionValidaHasta:    proyeccion.ValidaHasta,
		},
	)
	if err != nil {
		return domain.ManifiestoPreparacionCargaDirectaV1{}, err
	}
	return manifiesto, nil
}

func contenidoCargaDesdeResultado(
	resultado ports.ResultadoOperacionObjeto,
	zona domain.ZonaContenidoCarga,
) (domain.ContenidoCargaDocumental, error) {
	contenido := domain.ContenidoCargaDocumental{
		ConectorID: resultado.Objeto.ConectorID, Referencia: resultado.Objeto.Objeto.Referencia,
		Version: resultado.Objeto.Objeto.Version, Zona: zona, MIME: resultado.Objeto.MIME,
		Tamano: resultado.Objeto.Tamano, HuellaSHA256: resultado.Objeto.HuellaSHA256,
		EvidenciaRef: resultado.Evidencia.Referencia, RegistradoEn: resultado.Evidencia.RealizadaEn.UTC(),
	}
	if err := contenido.Validar(); err != nil {
		return domain.ContenidoCargaDocumental{}, err
	}
	return contenido, nil
}

func objetoAlmacenDesdeContenido(contenido domain.ContenidoCargaDocumental) ports.ReferenciaObjetoAlmacen {
	return ports.ReferenciaObjetoAlmacen{Referencia: contenido.Referencia, Version: contenido.Version}
}

func objetoAlmacenadoDesdeContenido(contenido domain.ContenidoCargaDocumental) ports.ObjetoAlmacenado {
	zona := ports.ZonaAlmacenCuarentena
	if contenido.Zona == domain.ZonaContenidoCargaAdmitida {
		zona = ports.ZonaAlmacenAdmitida
	}
	return ports.ObjetoAlmacenado{
		Objeto: objetoAlmacenDesdeContenido(contenido), ConectorID: contenido.ConectorID, Zona: zona,
		MIME: contenido.MIME, Tamano: contenido.Tamano, HuellaSHA256: contenido.HuellaSHA256,
		EvidenciaCreacionRef: contenido.EvidenciaRef, AlmacenadoEn: contenido.RegistradoEn,
	}
}

func objetoAlmacenCorrespondeCarga(objeto ports.ObjetoAlmacenado, contenido domain.ContenidoCargaDocumental) bool {
	return objeto.Validar() == nil && objeto.Objeto == objetoAlmacenDesdeContenido(contenido) &&
		objeto.ConectorID == contenido.ConectorID && objeto.Zona == ports.ZonaAlmacenCuarentena &&
		objeto.MIME == contenido.MIME && objeto.Tamano == contenido.Tamano && objeto.HuellaSHA256 == contenido.HuellaSHA256
}

func analisisCargaDesdeResultado(
	resultado ports.ResultadoAnalisisContenido,
	completadoEn time.Time,
) (domain.AnalisisCargaDocumental, error) {
	estado := domain.EstadoAnalisisCarga(resultado.Estado)
	analisis := domain.AnalisisCargaDocumental{
		ObjetoReferencia: resultado.Objeto.Referencia, ObjetoVersion: resultado.Objeto.Version,
		HuellaObjetoSHA256: resultado.HuellaObjetoSHA256, ConectorAnalizadorID: resultado.ConectorAnalizadorID,
		VersionConector: resultado.VersionConector, Estado: estado, CodigoResultado: resultado.CodigoResultado,
		EvidenciaRef: resultado.EvidenciaRef, HuellaEvidenciaSHA256: resultado.HuellaEvidenciaSHA256,
		CompletadoEn: completadoEn.UTC(),
	}
	if err := analisis.Validar(); err != nil {
		return domain.AnalisisCargaDocumental{}, err
	}
	return analisis, nil
}

func confirmacionCargaDocumental(
	anterior, siguiente domain.CargaDocumental,
	principal domain.Principal,
	perfilActivo, autorizacionRef, motivo, accion, tipoEvento string,
	instante time.Time,
	metadatos map[string]string,
) (ports.ConfirmacionTransicionCargaDocumental, error) {
	huellaAnterior, err := anterior.HuellaSHA256()
	if err != nil {
		return ports.ConfirmacionTransicionCargaDocumental{}, err
	}
	huellaSiguiente, err := siguiente.HuellaSHA256()
	if err != nil {
		return ports.ConfirmacionTransicionCargaDocumental{}, err
	}
	auditoria := domain.AuditEntry{
		ActorID: principal.ID, ActorProfile: perfilActivo, ActorRoles: append([]string(nil), principal.Roles...),
		AuthMethod: principal.AuthMethod, AuthAssurance: principal.AuthAssurance, AuthorizationRef: autorizacionRef,
		Purpose: siguiente.Finalidad, Action: accion, ModuleID: siguiente.ModuloID, SubjectRef: siguiente.ID,
		ObjectVersion: siguiente.Version, Reason: motivo, Result: "correcto",
		BeforeHash: huellaAnterior, AfterHash: huellaSiguiente, CorrelationRef: siguiente.CorrelacionRef,
		Metadata: copiarMapaCarga(metadatos), OccurredAt: instante.UTC(),
	}
	evento := domain.Event{
		Type: tipoEvento, ModuleID: siguiente.ModuloID, SubjectRef: siguiente.ID, ActorID: principal.ID,
		OccurredAt: instante.UTC(), Payload: map[string]string{
			"carga_ref": siguiente.ID, "estado": string(siguiente.Estado), "version": strconv.Itoa(siguiente.Version),
		},
	}
	return ports.NuevaConfirmacionTransicionCargaDocumental(anterior, siguiente, auditoria, evento)
}

func solicitudCargaDocumentalCanonica(orden OrdenPrepararCargaDocumental) []byte {
	valores := []string{
		orden.Principal.ID, orden.PerfilActivo, orden.Recurso.Referencia, orden.Recurso.ModuloID, orden.Recurso.Tipo,
		orden.OperacionRef, orden.Finalidad, orden.Clasificacion, orden.MIME, strconv.FormatInt(orden.Tamano, 10),
		orden.HuellaSHA256, orden.ClaveIdempotenciaCliente, orden.Motivo, orden.CorrelacionRef,
	}
	clavesAmbitos := clavesOrdenadasCarga(orden.Recurso.Ambitos)
	for _, clave := range clavesAmbitos {
		valores = append(valores, "ambito:"+clave, orden.Recurso.Ambitos[clave])
	}
	clavesAtributos := clavesOrdenadasCarga(orden.Recurso.Atributos)
	for _, clave := range clavesAtributos {
		valores = append(valores, "atributo:"+clave, orden.Recurso.Atributos[clave])
	}
	var canonico strings.Builder
	for _, valor := range valores {
		canonico.WriteString(strconv.Itoa(len(valor)))
		canonico.WriteByte(':')
		canonico.WriteString(valor)
		canonico.WriteByte('\n')
	}
	return []byte(canonico.String())
}

func clavesOrdenadasCarga(valores map[string]string) []string {
	claves := make([]string, 0, len(valores))
	for clave := range valores {
		claves = append(claves, clave)
	}
	sort.Strings(claves)
	return claves
}

func copiarMapaCarga(origen map[string]string) map[string]string {
	destino := make(map[string]string, len(origen))
	for clave, valor := range origen {
		destino[clave] = valor
	}
	return destino
}

func clonarPrincipalCarga(principal domain.Principal) domain.Principal {
	clon := principal
	clon.Roles = append([]string(nil), principal.Roles...)
	clon.Permissions = append([]string(nil), principal.Permissions...)
	clon.Attributes = copiarMapaCarga(principal.Attributes)
	return clon
}

func clonarRecursoCarga(recurso domain.RecursoAutorizable) domain.RecursoAutorizable {
	clon := recurso
	clon.Ambitos = copiarMapaCarga(recurso.Ambitos)
	clon.Atributos = copiarMapaCarga(recurso.Atributos)
	return clon
}

func clonarOrdenPrepararCargaDocumental(orden OrdenPrepararCargaDocumental) OrdenPrepararCargaDocumental {
	clon := orden
	clon.Principal = clonarPrincipalCarga(orden.Principal)
	clon.Recurso = clonarRecursoCarga(orden.Recurso)
	return clon
}

func clonarOrdenConfirmarCargaDocumental(orden OrdenConfirmarCargaDocumental) OrdenConfirmarCargaDocumental {
	clon := orden
	clon.Principal = clonarPrincipalCarga(orden.Principal)
	clon.Recurso = clonarRecursoCarga(orden.Recurso)
	return clon
}

func clonarOrdenAnalizarCargaDocumental(orden OrdenAnalizarCargaDocumental) OrdenAnalizarCargaDocumental {
	clon := orden
	clon.Principal = clonarPrincipalCarga(orden.Principal)
	clon.Recurso = clonarRecursoCarga(orden.Recurso)
	return clon
}

func textoCargaSeguro(valor string, maximo int, permiteEspacios bool) bool {
	if valor == "" || valor != strings.TrimSpace(valor) || len(valor) > maximo || !utf8.ValidString(valor) {
		return false
	}
	for _, caracter := range valor {
		if unicode.IsControl(caracter) || (!permiteEspacios && unicode.IsSpace(caracter)) {
			return false
		}
	}
	return true
}

func esSHA256Carga(valor string) bool {
	if len(valor) != 64 || valor != strings.ToLower(valor) || valor != strings.TrimSpace(valor) {
		return false
	}
	decodificado, err := hex.DecodeString(valor)
	return err == nil && len(decodificado) == sha256.Size
}

func mimeCargaValido(valor string) bool {
	if !textoCargaSeguro(valor, 255, false) || valor != strings.ToLower(valor) {
		return false
	}
	tipo, parametros, err := mime.ParseMediaType(valor)
	return err == nil && tipo == valor && len(parametros) == 0 && strings.Contains(tipo, "/")
}

func hmacCargaDocumentalValido(valor string) bool {
	partes := strings.Split(valor, ":")
	return len(partes) == 3 && partes[0] == "hmac-sha256" &&
		textoCargaSeguro(partes[1], 64, false) && esSHA256Carga(partes[2])
}

type lectorVerificadorCarga struct {
	origen   io.Reader
	huella   hash.Hash
	leidos   int64
	limite   int64
	excedido bool
}

func nuevoLectorVerificadorCarga(origen io.Reader, limite int64) *lectorVerificadorCarga {
	return &lectorVerificadorCarga{origen: origen, huella: sha256.New(), limite: limite}
}

func (l *lectorVerificadorCarga) Read(destino []byte) (int, error) {
	if l.excedido {
		return 0, ports.ErrLimiteObjetoAlmacenExcedido
	}
	if len(destino) == 0 {
		return 0, nil
	}
	restanteConTestigo := l.limite - l.leidos + 1
	if restanteConTestigo <= 0 {
		l.excedido = true
		return 0, ports.ErrLimiteObjetoAlmacenExcedido
	}
	if int64(len(destino)) > restanteConTestigo {
		destino = destino[:restanteConTestigo]
	}
	n, err := l.origen.Read(destino)
	if n > 0 {
		_, _ = l.huella.Write(destino[:n])
		l.leidos += int64(n)
		if l.leidos > l.limite {
			l.excedido = true
			return n, ports.ErrLimiteObjetoAlmacenExcedido
		}
	}
	return n, err
}

func (l *lectorVerificadorCarga) BytesLeidos() int64 { return l.leidos }
func (l *lectorVerificadorCarga) Excedido() bool     { return l.excedido }
func (l *lectorVerificadorCarga) FinExacto() bool {
	if l.excedido || l.leidos != l.limite {
		return false
	}
	var testigo [1]byte
	n, err := l.origen.Read(testigo[:])
	if n != 0 || err != io.EOF {
		l.excedido = true
		return false
	}
	return true
}
func (l *lectorVerificadorCarga) HuellaSHA256() string {
	return hex.EncodeToString(l.huella.Sum(nil))
}
