package application

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

var instanteServicioCarga = time.Date(2026, 7, 15, 10, 0, 0, 0, time.UTC)

type relojCargaPrueba struct {
	mu    sync.Mutex
	ahora time.Time
}

func (r *relojCargaPrueba) Ahora() time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()
	instante := r.ahora
	r.ahora = r.ahora.Add(time.Millisecond)
	return instante
}

type autorizadorCargaPrueba struct {
	mu               sync.Mutex
	reloj            *relojCargaPrueba
	permitidas       map[string]bool
	campos           map[string][]string
	obligaciones     map[string][]string
	vigencias        map[string]time.Duration
	solicitudes      []domain.SolicitudAutorizacion
	decisiones       []domain.DecisionAutorizacion
	detenidoEnExigir chan struct{}
	continuarExigir  chan struct{}
}

func (a *autorizadorCargaPrueba) Exigir(
	_ context.Context,
	solicitud domain.SolicitudAutorizacion,
) (domain.DecisionAutorizacion, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.solicitudes = append(a.solicitudes, solicitud)
	if a.detenidoEnExigir != nil {
		close(a.detenidoEnExigir)
		<-a.continuarExigir
	}
	if !a.permitidas[solicitud.Accion] {
		return domain.DecisionAutorizacion{}, domain.ErrAutorizacionDenegada
	}
	indice := len(a.solicitudes)
	emitidaEn := a.reloj.Ahora()
	campos, definidos := a.campos[solicitud.Accion]
	if !definidos {
		campos = camposDecisionCargaPrueba(solicitud.Accion)
	}
	vigencia := 4 * time.Minute
	if configurada, existe := a.vigencias[solicitud.Accion]; existe {
		vigencia = configurada
	}
	decision := domain.DecisionAutorizacion{
		DecisionRef: fmt.Sprintf("decision:carga:%03d", indice), Concedida: true, Codigo: "concedida",
		PrincipalID: solicitud.Principal.ID, PerfilActivoRef: solicitud.PerfilActivoRef,
		Accion: solicitud.Accion, RecursoRef: solicitud.Recurso.Referencia, Finalidad: solicitud.Finalidad,
		CorrelacionRef: solicitud.CorrelacionRef, AsignacionRef: "asignacion:carga:uno",
		AsignacionHuellaSHA256: strings.Repeat("a", 64), VersionRolRef: "rol:carga:v1",
		VersionRolHuellaSHA256: strings.Repeat("b", 64), GarantiaMinima: domain.AuthAssuranceHigh,
		CamposPermitidos: append([]string(nil), campos...),
		Obligaciones:     append([]string(nil), a.obligaciones[solicitud.Accion]...),
		EmitidaEn:        emitidaEn, ValidaHasta: emitidaEn.Add(vigencia),
	}
	decision = completarDecisionAutorizacionPrueba(solicitud, decision)
	a.decisiones = append(a.decisiones, decision)
	return decision, nil
}

func camposDecisionCargaPrueba(accion string) []string {
	switch accion {
	case AccionPrepararCargaDocumental:
		return camposDecisionPrepararCarga()
	case AccionConfirmarCargaDocumental:
		return camposDecisionConfirmarCarga()
	case AccionAnalizarCargaDocumental:
		return camposDecisionAnalizarCarga()
	case AccionPromoverCargaDocumental:
		return camposDecisionPromoverCarga()
	default:
		return nil
	}
}

type repositorioCargaPrueba struct {
	mu                        sync.Mutex
	carga                     domain.CargaDocumental
	manifiesto                domain.ManifiestoPreparacionCargaDirectaV1
	token                     ports.TokenReservaCargaDocumental
	decisionesPreparacion     map[string]ports.ConsumoDecisionPreparacionCargaDocumentalV1
	decisionPreparacionActiva string
	confirmaciones            []ports.ConfirmacionTransicionCargaDocumental
	abandonada                bool
}

func (r *repositorioCargaPrueba) Reservar(
	_ context.Context,
	solicitud ports.SolicitudReservarCargaDocumental,
) (ports.ReservaCargaDocumental, error) {
	if err := solicitud.Validar(); err != nil {
		return ports.ReservaCargaDocumental{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.carga.ID != "" {
		if r.carga.IndiceIdempotenciaHMAC != solicitud.IndiceIdempotenciaHMAC ||
			r.carga.HuellaSolicitudHMAC != solicitud.HuellaSolicitudHMAC {
			return ports.ReservaCargaDocumental{}, ports.ErrReservaCargaDocumentalOcupada
		}
		return ports.ReservaCargaDocumental{Repetida: true, Carga: r.carga}, nil
	}
	if _, reclamada := r.decisionesPreparacion[solicitud.DecisionPreparacion.DecisionRef]; reclamada {
		return ports.ReservaCargaDocumental{}, ports.ErrDecisionPreparacionCargaNoDisponible
	}
	if r.decisionesPreparacion == nil {
		r.decisionesPreparacion = make(map[string]ports.ConsumoDecisionPreparacionCargaDocumentalV1)
	}
	token, err := ports.NuevoTokenReservaCargaDocumental("token:reserva:carga:0123456789abcdef")
	if err != nil {
		return ports.ReservaCargaDocumental{}, err
	}
	r.carga = solicitud.Carga
	r.token = token
	r.decisionesPreparacion[solicitud.DecisionPreparacion.DecisionRef] = solicitud.DecisionPreparacion
	r.decisionPreparacionActiva = solicitud.DecisionPreparacion.DecisionRef
	return ports.ReservaCargaDocumental{Token: token, Carga: solicitud.Carga}, nil
}

func (r *repositorioCargaPrueba) ConfirmarPreparacion(
	_ context.Context,
	solicitud ports.SolicitudConfirmarPreparacionCargaDocumental,
) error {
	instantanea, err := ports.InstantaneaSolicitudConfirmarPreparacionCargaDocumental(solicitud)
	if err != nil {
		return ports.ErrConfirmacionCargaDocumentalInvalida
	}
	consumo, err := ports.ConsumoDecisionDesdeManifiestoPreparacionCargaDocumental(instantanea.Manifiesto)
	if err != nil {
		return ports.ErrConfirmacionCargaDocumentalInvalida
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	esperado, _ := r.token.RevelarParaPersistencia()
	recibido, _ := instantanea.Token.RevelarParaPersistencia()
	if esperado == "" || recibido != esperado ||
		instantanea.Confirmacion.ValidarContra(r.carga) != nil ||
		instantanea.Manifiesto.ValidarContraCarga(instantanea.Confirmacion.Carga) != nil {
		return ports.ErrConfirmacionCargaDocumentalInvalida
	}
	reclamada, existe := r.decisionesPreparacion[consumo.DecisionRef]
	if !existe || reclamada != consumo || r.decisionPreparacionActiva != consumo.DecisionRef {
		return ports.ErrDecisionPreparacionCargaNoDisponible
	}
	r.carga = instantanea.Confirmacion.Carga
	r.manifiesto = instantanea.Manifiesto
	r.decisionPreparacionActiva = ""
	r.confirmaciones = append(r.confirmaciones, instantanea.Confirmacion)
	r.token = ports.TokenReservaCargaDocumental{}
	return nil
}

func (r *repositorioCargaPrueba) ConfirmarTransicion(
	_ context.Context,
	confirmacion ports.ConfirmacionTransicionCargaDocumental,
) error {
	instantanea := ports.InstantaneaConfirmacionTransicionCargaDocumental(confirmacion)
	r.mu.Lock()
	defer r.mu.Unlock()
	if instantanea.ValidarContra(r.carga) != nil {
		return ports.ErrConfirmacionCargaDocumentalInvalida
	}
	r.carga = instantanea.Carga
	r.confirmaciones = append(r.confirmaciones, instantanea)
	return nil
}

func (r *repositorioCargaPrueba) AbandonarReserva(_ context.Context, token ports.TokenReservaCargaDocumental) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !token.Valido() {
		return ports.ErrReservaCargaDocumentalInvalida
	}
	r.abandonada = true
	r.carga = domain.CargaDocumental{}
	r.manifiesto = domain.ManifiestoPreparacionCargaDirectaV1{}
	r.decisionPreparacionActiva = ""
	r.token = ports.TokenReservaCargaDocumental{}
	return nil
}

func (r *repositorioCargaPrueba) Obtener(_ context.Context, id string) (domain.CargaDocumental, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.carga.ID != id {
		return domain.CargaDocumental{}, ports.ErrCargaDocumentalNoEncontrada
	}
	return r.carga, nil
}

func (r *repositorioCargaPrueba) ObtenerPreparacion(
	_ context.Context,
	id string,
) (ports.PreparacionCargaDocumentalPersistida, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	resultado := ports.PreparacionCargaDocumentalPersistida{Carga: r.carga, Manifiesto: r.manifiesto}
	if r.carga.ID != id || resultado.Validar() != nil {
		return ports.PreparacionCargaDocumentalPersistida{}, ports.ErrManifiestoPreparacionNoEncontrado
	}
	return resultado, nil
}

type selladorCargaPrueba struct{ dominio string }

func (s selladorCargaPrueba) sellar(contenido []byte) string {
	suma := sha256.Sum256(append([]byte(s.dominio+"\x00"), contenido...))
	return "hmac-sha256:" + s.dominio + "_v1:" + hex.EncodeToString(suma[:])
}

func (s selladorCargaPrueba) SellarIdempotenciaCarga(
	_ context.Context,
	solicitud ports.SolicitudSellarIdempotenciaCarga,
) (string, error) {
	if err := solicitud.Validar(); err != nil {
		return "", err
	}
	return s.sellar([]byte(solicitud.OperacionRef + "\x00" + solicitud.PrincipalRef + "\x00" +
		solicitud.RecursoRef + "\x00" + solicitud.MIME + "\x00" + fmt.Sprint(solicitud.Tamano) + "\x00" +
		solicitud.HuellaSHA256 + "\x00" + solicitud.ClaveSolicitante)), nil
}

func (s selladorCargaPrueba) SellarSolicitudCargaDocumental(_ context.Context, contenido []byte) (string, error) {
	if len(contenido) == 0 {
		return "", errors.New("contenido vacio")
	}
	return s.sellar(contenido), nil
}

func (s selladorCargaPrueba) SellarVinculoSesionCarga(_ context.Context, sesion string) (string, error) {
	if strings.TrimSpace(sesion) == "" {
		return "", errors.New("sesion vacia")
	}
	return s.sellar([]byte(sesion)), nil
}

type generadorIDCargaPrueba struct {
	siguiente int
	valor     string
}

func (g *generadorIDCargaPrueba) NuevoIDCargaDocumental() (string, error) {
	if g.valor != "" {
		return g.valor, nil
	}
	g.siguiente++
	return fmt.Sprintf("carga:documental:%016d", g.siguiente), nil
}

type registroReciboCargaPrueba struct {
	contexto     ports.ContextoOperacionAlmacen
	sesionRef    string
	registradoEn time.Time
	expiraEn     time.Time
	consumido    bool
}

func debeProyectarContextoCargaPrueba(
	contexto ports.ContextoOperacionAlmacen,
) ports.ProyeccionContextoOperacionAlmacen {
	proyeccion, err := contexto.Proyeccion()
	if err != nil {
		panic(fmt.Sprintf("contexto de almacen de prueba invalido: %v", err))
	}
	return proyeccion
}

// seguridadCargaPrueba representa cuatro finalidades criptograficas. En
// produccion deben usar claves separadas y un consumo atomico duradero.
type seguridadCargaPrueba struct {
	mu                sync.Mutex
	reloj             *relojCargaPrueba
	siguienteEmision  int
	siguienteConsumo  int
	recibos           map[string]registroReciboCargaPrueba
	ultimaValidaHasta time.Time
}

func (s *seguridadCargaPrueba) SeudonimizarSujetoAlmacen(
	_ context.Context,
	solicitud ports.SolicitudSeudonimizarSujetoAlmacen,
) (string, error) {
	sujetoRef, ambitoRef, err := solicitud.RevelarParaSellado()
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, []byte("clave-exclusiva-seudonimizacion-prueba"))
	_, _ = mac.Write([]byte(sujetoRef + "\x00" + ambitoRef))
	return "hmac-sha256:seudonimo_v1:" + hex.EncodeToString(mac.Sum(nil)), nil
}

func (s *seguridadCargaPrueba) EmitirReciboCargaDirecta(
	_ context.Context,
	solicitud ports.SolicitudEmitirReciboCargaDirecta,
) (ports.ReciboCargaDirecta, error) {
	contexto, sesionRef, expiraEn, _, err := solicitud.RevelarParaEmision()
	if err != nil {
		return ports.ReciboCargaDirecta{}, err
	}
	proyeccion := debeProyectarContextoCargaPrueba(contexto)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.recibos == nil {
		s.recibos = make(map[string]registroReciboCargaPrueba)
	}
	s.siguienteEmision++
	mac := hmac.New(sha256.New, []byte("clave-exclusiva-recibos-prueba"))
	_, _ = mac.Write([]byte(fmt.Sprintf(
		"%s\x00%s\x00%s\x00%s\x00%d",
		proyeccion.OperacionRef, proyeccion.CargaRef, proyeccion.HuellaSolicitudHMAC, sesionRef, s.siguienteEmision,
	)))
	valor := "recibo:carga:directa:" + hex.EncodeToString(mac.Sum(nil))
	recibo, err := ports.NuevoReciboCargaDirecta(valor)
	if err != nil {
		return ports.ReciboCargaDirecta{}, err
	}
	s.recibos[valor] = registroReciboCargaPrueba{
		contexto: contexto, sesionRef: sesionRef, registradoEn: s.reloj.Ahora().UTC(), expiraEn: expiraEn,
	}
	return recibo, nil
}

func (s *seguridadCargaPrueba) ConsumirReciboCargaDirecta(
	_ context.Context,
	solicitud ports.SolicitudConsumirReciboCargaDirecta,
) (ports.ComprobanteConsumoReciboCargaDirecta, error) {
	if err := solicitud.Validar(); err != nil {
		return ports.ComprobanteConsumoReciboCargaDirecta{}, err
	}
	valor, err := solicitud.Recibo.RevelarParaEntregaOConsumo()
	if err != nil {
		return ports.ComprobanteConsumoReciboCargaDirecta{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ultimaValidaHasta = solicitud.ValidaHasta
	registro, existe := s.recibos[valor]
	consumidoEn := s.reloj.Ahora().UTC()
	if !existe || registro.consumido || consumidoEn.Before(registro.registradoEn) ||
		!consumidoEn.Before(registro.expiraEn) || !consumidoEn.Before(solicitud.ValidaHasta) ||
		registro.sesionRef != solicitud.SesionRef ||
		!contextosReciboMismaOperacion(registro.contexto, solicitud.Contexto) {
		return ports.ComprobanteConsumoReciboCargaDirecta{}, ports.ErrReciboCargaDirectaNoValido
	}
	proyeccionSolicitud := debeProyectarContextoCargaPrueba(solicitud.Contexto)
	proyeccionRegistro := debeProyectarContextoCargaPrueba(registro.contexto)
	s.siguienteConsumo++
	evidenciaRef := fmt.Sprintf("evidencia:consumo:recibo:%016d", s.siguienteConsumo)
	intencionRef := fmt.Sprintf("intencion:confirmacion:carga:%016d", s.siguienteConsumo)
	huellaIntencion := hmacCargaPrueba(
		"intencion_v1", proyeccionSolicitud.CargaRef, proyeccionSolicitud.AutorizacionRef,
		solicitud.SesionRef, intencionRef, evidenciaRef, registro.registradoEn.Format(time.RFC3339Nano),
		solicitud.ValidaHasta.Format(time.RFC3339Nano),
	)
	resultado := ports.ResultadoConsumoReciboCargaDirecta{
		IndiceHMAC:               hmacCargaPrueba("indice_v1", valor),
		GrupoHMAC:                hmacCargaPrueba("grupo_v1", proyeccionSolicitud.CargaRef, solicitud.SesionRef),
		VinculoHMAC:              hmacCargaPrueba("vinculo_v1", proyeccionRegistro.CargaRef, registro.sesionRef),
		EvidenciaConsumoRef:      evidenciaRef,
		IntencionConfirmacionRef: intencionRef,
		HuellaIntencionHMAC:      huellaIntencion,
		RegistradoEn:             registro.registradoEn,
		ConsumidoEn:              consumidoEn,
		ExpiraEn:                 registro.expiraEn,
	}
	atestacion := atestacionCargaPrueba(solicitud.Contexto, solicitud.SesionRef, resultado, solicitud.ValidaHasta)
	comprobante, err := ports.NuevoComprobanteConsumoReciboCargaDirecta(
		solicitud, resultado, atestacion,
	)
	if err != nil {
		return ports.ComprobanteConsumoReciboCargaDirecta{}, err
	}
	registro.consumido = true
	s.recibos[valor] = registro
	return comprobante, nil
}

func (s *seguridadCargaPrueba) VerificarAtestacionConsumoReciboCargaDirecta(
	_ context.Context,
	contexto ports.ContextoOperacionAlmacen,
	sesionRef string,
	comprobante ports.ComprobanteConsumoReciboCargaDirecta,
) error {
	indice, grupo, vinculo, evidencia, intencion, huellaIntencion,
		registradoEn, consumidoEn, expiraEn, validaHasta, atestacion, err := comprobante.RevelarParaVerificacion()
	if err != nil {
		return ports.ErrAtestacionReciboCargaDirectaNoValida
	}
	resultado := ports.ResultadoConsumoReciboCargaDirecta{
		IndiceHMAC: indice, GrupoHMAC: grupo, VinculoHMAC: vinculo,
		EvidenciaConsumoRef: evidencia, IntencionConfirmacionRef: intencion,
		HuellaIntencionHMAC: huellaIntencion, RegistradoEn: registradoEn,
		ConsumidoEn: consumidoEn, ExpiraEn: expiraEn,
	}
	esperada := atestacionCargaPrueba(contexto, sesionRef, resultado, validaHasta)
	if !hmac.Equal([]byte(esperada), []byte(atestacion)) {
		return ports.ErrAtestacionReciboCargaDirectaNoValida
	}
	return nil
}

func hmacCargaPrueba(dominio string, valores ...string) string {
	mac := hmac.New(sha256.New, []byte("clave-"+dominio+"-separada-prueba-0123456789abcdef"))
	for _, valor := range valores {
		_, _ = fmt.Fprintf(mac, "%d:%s\n", len(valor), valor)
	}
	return "hmac-sha256:" + dominio + ":" + hex.EncodeToString(mac.Sum(nil))
}

func atestacionCargaPrueba(
	contexto ports.ContextoOperacionAlmacen,
	sesionRef string,
	resultado ports.ResultadoConsumoReciboCargaDirecta,
	validaHasta time.Time,
) string {
	proyeccion := debeProyectarContextoCargaPrueba(contexto)
	return hmacCargaPrueba("atestacion_v1",
		proyeccion.Esquema, proyeccion.OperacionRef, proyeccion.CorrelacionRef, proyeccion.AutorizacionRef,
		proyeccion.Finalidad, proyeccion.Clasificacion, proyeccion.AccionNegocio, proyeccion.AccionTecnica,
		proyeccion.CargaRef, proyeccion.SujetoSeudonimoHMAC, proyeccion.RecursoRef, proyeccion.ModuloID,
		proyeccion.TipoRecurso, proyeccion.HuellaRecursoSHA256, proyeccion.HuellaSolicitudHMAC,
		proyeccion.EfectoRef, proyeccion.HuellaPlanEfectoSHA256, string(proyeccion.PasoRef),
		proyeccion.ObjetoVinculado.Referencia, proyeccion.ObjetoVinculado.Version,
		proyeccion.HuellaDecisionSHA256, sesionRef, resultado.IndiceHMAC, resultado.GrupoHMAC,
		resultado.VinculoHMAC, resultado.EvidenciaConsumoRef,
		resultado.IntencionConfirmacionRef, resultado.HuellaIntencionHMAC,
		resultado.RegistradoEn.Format(time.RFC3339Nano),
		resultado.ConsumidoEn.Format(time.RFC3339Nano),
		resultado.ExpiraEn.Format(time.RFC3339Nano), validaHasta.Format(time.RFC3339Nano),
	)
}

func contextosReciboMismaOperacion(
	preparacion ports.ContextoOperacionAlmacen,
	confirmacion ports.ContextoOperacionAlmacen,
) bool {
	proyeccionPreparacion := debeProyectarContextoCargaPrueba(preparacion)
	proyeccionConfirmacion := debeProyectarContextoCargaPrueba(confirmacion)
	return proyeccionPreparacion.AccionTecnica == ports.AccionAlmacenPrepararCargaDirecta &&
		proyeccionConfirmacion.AccionTecnica == ports.AccionAlmacenConfirmarCargaDirecta &&
		proyeccionPreparacion.OperacionRef == proyeccionConfirmacion.OperacionRef &&
		proyeccionPreparacion.CorrelacionRef == proyeccionConfirmacion.CorrelacionRef &&
		proyeccionPreparacion.Finalidad == proyeccionConfirmacion.Finalidad &&
		proyeccionPreparacion.Clasificacion == proyeccionConfirmacion.Clasificacion &&
		proyeccionPreparacion.CargaRef == proyeccionConfirmacion.CargaRef &&
		proyeccionPreparacion.SujetoSeudonimoHMAC == proyeccionConfirmacion.SujetoSeudonimoHMAC &&
		proyeccionPreparacion.RecursoRef == proyeccionConfirmacion.RecursoRef &&
		proyeccionPreparacion.ModuloID == proyeccionConfirmacion.ModuloID &&
		proyeccionPreparacion.HuellaSolicitudHMAC == proyeccionConfirmacion.HuellaSolicitudHMAC
}

type almacenCargaPrueba struct {
	reloj             *relojCargaPrueba
	contenido         []byte
	contenidoLectura  []byte
	sesion            string
	sesionReemision   string
	preparaciones     int
	confirmaciones    int
	intentosConfirmar int
	promociones       int
	abandonos         int
	errorPreparacion  error
	falloPreparacion  error
	errorConfirmacion error
	objetoCuarentena  ports.ObjetoAlmacenado
	contextos         []ports.ContextoOperacionAlmacen
}

func (a *almacenCargaPrueba) Capacidades(context.Context) (ports.CapacidadesAlmacenObjetos, error) {
	return ports.CapacidadesAlmacenObjetos{
		ConectorID: "almacen_s3_corporativo", EscrituraEnFlujo: true, LecturaEnFlujo: true,
		ReferenciasOpacas: true, IntegridadSHA256: true, Versionado: true, Retencion: true,
		BloqueoLegal: true, PromocionAtomica: true, CargaDirectaTemporal: true, CifradoEnTransito: true,
		CifradoEnReposo: true, CifradoPorObjeto: true, TamanoMaximoObjeto: 1024 * 1024,
		PreservaObjetoOriginal: true, OrigenesCargaDirecta: []string{"https://objetos.example.test"},
	}, nil
}

func (a *almacenCargaPrueba) PrepararCargaDirecta(
	_ context.Context,
	solicitud ports.SolicitudPrepararCargaDirecta,
) (ports.InstruccionesCargaDirecta, error) {
	if err := solicitud.Validar(); err != nil {
		return ports.InstruccionesCargaDirecta{}, err
	}
	a.preparaciones++
	a.contextos = append(a.contextos, solicitud.Contexto)
	if a.falloPreparacion != nil {
		return ports.InstruccionesCargaDirecta{}, a.falloPreparacion
	}
	a.sesion = "sesion:carga:0123456789abcdef"
	if a.preparaciones > 1 && a.sesionReemision != "" {
		a.sesion = a.sesionReemision
	}
	instrucciones, err := ports.NuevasInstruccionesCargaDirectaParaSolicitud(
		solicitud, "almacen_s3_corporativo", a.sesion, ports.MetodoCargaDirectaPUT,
		"https://objetos.example.test/cuarentena?firma=opaca", []ports.CabeceraCargaDirecta{
			{Nombre: "content-type", Valor: solicitud.MIME},
		}, a.reloj.Ahora(),
	)
	a.errorPreparacion = err
	return instrucciones, err
}

func (a *almacenCargaPrueba) ConfirmarCargaDirecta(
	_ context.Context,
	solicitud ports.SolicitudConfirmarCargaDirecta,
) (ports.ResultadoOperacionObjeto, error) {
	contexto, sesionRef, intencionRef, huellaIntencion, _, emitidoEn, consumidoEn, expiraEn, validaHasta, err := solicitud.RevelarParaConector()
	if err != nil || sesionRef != a.sesion || intencionRef == "" || !hmacCargaDocumentalValido(huellaIntencion) ||
		consumidoEn.Before(emitidoEn) || !consumidoEn.Before(expiraEn) || !consumidoEn.Before(validaHasta) {
		return ports.ResultadoOperacionObjeto{}, ports.ErrSesionCargaDirectaNoValida
	}
	a.intentosConfirmar++
	if a.errorConfirmacion != nil {
		return ports.ResultadoOperacionObjeto{}, a.errorConfirmacion
	}
	a.confirmaciones++
	a.contextos = append(a.contextos, contexto)
	suma := sha256.Sum256(a.contenido)
	referencia := ports.ReferenciaObjetoAlmacen{Referencia: "objeto:cuarentena:0123456789abcdef", Version: "v1"}
	almacenadoEn := a.reloj.Ahora().UTC()
	if almacenadoEn.Before(consumidoEn) {
		return ports.ResultadoOperacionObjeto{}, ports.ErrSolicitudAlmacenInvalida
	}
	evidencia := evidenciaAlmacenCargaPrueba(
		ports.AccionAlmacenConfirmarCargaDirecta, referencia, contexto, intencionRef, almacenadoEn,
	)
	a.objetoCuarentena = ports.ObjetoAlmacenado{
		Objeto: referencia, ConectorID: "almacen_s3_corporativo", Zona: ports.ZonaAlmacenCuarentena,
		MIME: "application/pdf", Tamano: int64(len(a.contenido)), HuellaSHA256: hex.EncodeToString(suma[:]),
		EvidenciaCreacionRef: evidencia.Referencia, AlmacenadoEn: almacenadoEn,
	}
	return ports.ResultadoOperacionObjeto{Objeto: a.objetoCuarentena, Evidencia: evidencia}, nil
}

func (a *almacenCargaPrueba) AbandonarCargaDirecta(
	_ context.Context,
	contexto ports.ContextoOperacionAlmacen,
	sesion string,
) error {
	if contexto.ValidarParaEn(ports.AccionAlmacenAbandonarCargaDirecta, a.reloj.Ahora().UTC()) != nil || sesion != a.sesion {
		return ports.ErrSesionCargaDirectaNoValida
	}
	a.abandonos++
	a.contextos = append(a.contextos, contexto)
	return nil
}

func (a *almacenCargaPrueba) Escribir(context.Context, ports.SolicitudEscribirObjeto) (ports.ResultadoOperacionObjeto, error) {
	return ports.ResultadoOperacionObjeto{}, ports.ErrCapacidadAlmacenNoDisponible
}

func (a *almacenCargaPrueba) Abrir(
	_ context.Context,
	solicitud ports.SolicitudAbrirObjeto,
) (ports.LecturaObjetoAlmacen, error) {
	if solicitud.Validar() != nil || solicitud.Objeto != a.objetoCuarentena.Objeto {
		return ports.LecturaObjetoAlmacen{}, ports.ErrObjetoAlmacenNoEncontrado
	}
	a.contextos = append(a.contextos, solicitud.Contexto)
	evidencia := evidenciaAlmacenCargaPrueba(
		ports.AccionAlmacenLeer, solicitud.Objeto, solicitud.Contexto, "", a.reloj.Ahora().UTC(),
	)
	contenido := a.contenido
	if a.contenidoLectura != nil {
		contenido = a.contenidoLectura
	}
	return ports.LecturaObjetoAlmacen{
		Objeto: a.objetoCuarentena, Evidencia: evidencia, Contenido: io.NopCloser(bytes.NewReader(contenido)),
	}, nil
}

func (a *almacenCargaPrueba) Promover(
	_ context.Context,
	solicitud ports.SolicitudPromoverObjeto,
) (ports.ResultadoOperacionObjeto, error) {
	if solicitud.Validar() != nil || solicitud.Origen != a.objetoCuarentena.Objeto {
		return ports.ResultadoOperacionObjeto{}, ports.ErrSolicitudAlmacenInvalida
	}
	a.promociones++
	a.contextos = append(a.contextos, solicitud.Contexto)
	objeto := a.objetoCuarentena
	objeto.Objeto = ports.ReferenciaObjetoAlmacen{Referencia: "objeto:admitido:0123456789abcdef", Version: "v1"}
	objeto.Zona = ports.ZonaAlmacenAdmitida
	almacenadoEn := a.reloj.Ahora().UTC()
	evidencia := evidenciaAlmacenCargaPrueba(
		ports.AccionAlmacenPromover, objeto.Objeto, solicitud.Contexto,
		solicitud.EvidenciaAnalisisRef, almacenadoEn,
	)
	objeto.EvidenciaCreacionRef = evidencia.Referencia
	objeto.AlmacenadoEn = almacenadoEn
	return ports.ResultadoOperacionObjeto{Objeto: objeto, Evidencia: evidencia}, nil
}

func (a *almacenCargaPrueba) AplicarRetencion(context.Context, ports.SolicitudRetenerObjeto) (ports.ResultadoOperacionObjeto, error) {
	return ports.ResultadoOperacionObjeto{}, ports.ErrCapacidadAlmacenNoDisponible
}

func (a *almacenCargaPrueba) Inmovilizar(context.Context, ports.SolicitudInmovilizarObjeto) (ports.ResultadoOperacionObjeto, error) {
	return ports.ResultadoOperacionObjeto{}, ports.ErrCapacidadAlmacenNoDisponible
}

func (a *almacenCargaPrueba) LevantarInmovilizacion(context.Context, ports.SolicitudLevantarInmovilizacionObjeto) (ports.ResultadoOperacionObjeto, error) {
	return ports.ResultadoOperacionObjeto{}, ports.ErrCapacidadAlmacenNoDisponible
}

func (a *almacenCargaPrueba) Eliminar(context.Context, ports.SolicitudEliminarObjeto) (ports.EvidenciaOperacionAlmacen, error) {
	return ports.EvidenciaOperacionAlmacen{}, ports.ErrCapacidadAlmacenNoDisponible
}

func evidenciaAlmacenCargaPrueba(
	accion string,
	objeto ports.ReferenciaObjetoAlmacen,
	contexto ports.ContextoOperacionAlmacen,
	fundamentoRef string,
	instante time.Time,
) ports.EvidenciaOperacionAlmacen {
	proyeccion := debeProyectarContextoCargaPrueba(contexto)
	return ports.EvidenciaOperacionAlmacen{
		Referencia: "evidencia:" + accion + ":0123456789abcdef", ConectorID: "almacen_s3_corporativo",
		EsquemaContexto: proyeccion.Esquema, AccionNegocio: proyeccion.AccionNegocio,
		Accion: accion, EfectoRef: proyeccion.EfectoRef,
		HuellaPlanEfectoSHA256: proyeccion.HuellaPlanEfectoSHA256,
		PasoRef:                proyeccion.PasoRef, HuellaDecisionSHA256: proyeccion.HuellaDecisionSHA256,
		Objeto: objeto, OperacionRef: proyeccion.OperacionRef,
		CorrelacionRef: proyeccion.CorrelacionRef, AutorizacionRef: proyeccion.AutorizacionRef,
		Finalidad: proyeccion.Finalidad, Clasificacion: proyeccion.Clasificacion, RealizadaEn: instante,
		CargaRef: proyeccion.CargaRef, SujetoSeudonimoHMAC: proyeccion.SujetoSeudonimoHMAC,
		RecursoRef: proyeccion.RecursoRef, ModuloID: proyeccion.ModuloID,
		HuellaSolicitudHMAC: proyeccion.HuellaSolicitudHMAC, FundamentoRef: fundamentoRef,
	}
}

type analizadorCargaPrueba struct {
	reloj          *relojCargaPrueba
	estado         ports.EstadoAnalisisContenido
	leerSolo       int64
	objetoAlterado bool
	conectorID     string
	version        int
	mimeDetectado  string
	llamadas       int
	contextos      []ports.ContextoOperacionAlmacen
}

func (a *analizadorCargaPrueba) Capacidades(context.Context) (ports.CapacidadesAnalizadorContenido, error) {
	conectorID, version := a.identidad()
	return ports.CapacidadesAnalizadorContenido{
		ConectorID: conectorID, VersionConector: version, AnalisisEnFlujo: true,
		CanalAutenticado: true, CifradoEnTransito: true, IdentidadMutua: true, ActualizacionFirmas: true,
		DetectaMalware: true, DetectaContenidoActivo: true, TamanoMaximo: 1024 * 1024,
	}, nil
}

func (a *analizadorCargaPrueba) Analizar(
	_ context.Context,
	solicitud ports.SolicitudAnalizarContenido,
) (ports.ResultadoAnalisisContenido, error) {
	if err := solicitud.Validar(); err != nil {
		return ports.ResultadoAnalisisContenido{}, err
	}
	a.llamadas++
	a.contextos = append(a.contextos, solicitud.Contexto)
	limite := solicitud.Tamano
	if a.leerSolo > 0 {
		limite = a.leerSolo
	}
	leidos, err := io.CopyN(io.Discard, solicitud.Contenido, limite)
	if err != nil && !errors.Is(err, io.EOF) {
		return ports.ResultadoAnalisisContenido{}, err
	}
	estado := a.estado
	if estado == "" {
		estado = ports.EstadoAnalisisContenidoLimpio
	}
	objeto := solicitud.Objeto
	if a.objetoAlterado {
		objeto.Referencia = "objeto:cuarentena:otro"
	}
	conectorID, version := a.identidad()
	mimeDetectado := a.mimeDetectado
	if mimeDetectado == "" {
		mimeDetectado = "application/pdf"
	}
	proyeccion := debeProyectarContextoCargaPrueba(solicitud.Contexto)
	resultado := ports.ResultadoAnalisisContenido{
		Objeto: objeto, ConectorAlmacenID: solicitud.ConectorAlmacenID, HuellaObjetoSHA256: solicitud.HuellaSHA256,
		TamanoObjeto: solicitud.Tamano, MIMEDeclarado: solicitud.MIME, MIMEDetectado: mimeDetectado,
		ConectorAnalizadorID: conectorID, VersionConector: version,
		Estado: estado, CodigoResultado: string(estado), BytesAnalizados: leidos,
		EvidenciaRef: "evidencia:analisis:0123456789abcdef", HuellaEvidenciaSHA256: strings.Repeat("e", 64),
		AnalisisIniciadoEn: a.reloj.Ahora(), AnalisisCompletadoEn: a.reloj.Ahora(),
		CorrelacionRef: proyeccion.CorrelacionRef, AutorizacionRef: proyeccion.AutorizacionRef,
		Finalidad: proyeccion.Finalidad, Clasificacion: proyeccion.Clasificacion,
	}
	if estado == ports.EstadoAnalisisContenidoLimpio || estado == ports.EstadoAnalisisContenidoMalicioso ||
		estado == ports.EstadoAnalisisContenidoSospechoso {
		resultado.MotorRef = "motor:antivirus:uno"
		resultado.VersionMotor = "1.0.0"
		resultado.FirmasRef = "firmas:20260715"
	}
	if estado == ports.EstadoAnalisisContenidoMalicioso || estado == ports.EstadoAnalisisContenidoSospechoso {
		resultado.Detecciones = []ports.DeteccionContenido{{Clase: ports.ClaseDeteccionMalware, Codigo: "prueba"}}
	}
	return resultado, nil
}

func (a *analizadorCargaPrueba) identidad() (string, int) {
	conectorID := a.conectorID
	if conectorID == "" {
		conectorID = "analizador_icap_corporativo"
	}
	version := a.version
	if version == 0 {
		version = 1
	}
	return conectorID, version
}

type entornoServicioCarga struct {
	servicio    *ServicioCargaDocumental
	repositorio *repositorioCargaPrueba
	autorizador *autorizadorCargaPrueba
	almacen     *almacenCargaPrueba
	analizador  *analizadorCargaPrueba
	seguridad   *seguridadCargaPrueba
	reloj       *relojCargaPrueba
	recurso     domain.RecursoAutorizable
	externo     domain.Principal
	servicioAV  domain.Principal
	contenido   []byte
}

func nuevoEntornoServicioCarga(t *testing.T) *entornoServicioCarga {
	t.Helper()
	contenido := []byte("%PDF-1.7\ncontenido probatorio de prueba\n%%EOF")
	reloj := &relojCargaPrueba{ahora: instanteServicioCarga}
	autorizador := &autorizadorCargaPrueba{reloj: reloj, permitidas: map[string]bool{
		AccionPrepararCargaDocumental: true, AccionConfirmarCargaDocumental: true,
		AccionAnalizarCargaDocumental: true, AccionPromoverCargaDocumental: true,
	}}
	repositorio := &repositorioCargaPrueba{}
	almacen := &almacenCargaPrueba{reloj: reloj, contenido: contenido}
	analizador := &analizadorCargaPrueba{reloj: reloj}
	seguridad := &seguridadCargaPrueba{reloj: reloj}
	sellador := selladorCargaPrueba{dominio: "carga"}
	servicio, err := NuevoServicioCargaDocumental(
		repositorio, autorizador, almacen, almacen, analizador, sellador, sellador, sellador,
		seguridad, seguridad, seguridad, seguridad, &generadorIDCargaPrueba{}, reloj, OpcionesServicioCargaDocumental{
			ConectorAlmacenPermitido:    "almacen_s3_corporativo",
			ConectorAnalizadorPermitido: "analizador_icap_corporativo",
			VersionAnalizadorPermitida:  1,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return &entornoServicioCarga{
		servicio: servicio, repositorio: repositorio, autorizador: autorizador, almacen: almacen,
		analizador: analizador, seguridad: seguridad, reloj: reloj, contenido: contenido,
		recurso: domain.RecursoAutorizable{
			Referencia: "solicitud:bolsa:0123456789abcdef", ModuloID: "bolsa", Tipo: "solicitud",
			Ambitos: map[string]string{"proceso": "proceso:0123456789abcdef"},
		},
		externo: domain.Principal{
			ID: personaAutorizacionPrueba("persona:0123456789abcdef"), AuthMethod: domain.AuthMethodCertificate,
			AuthAssurance: domain.AuthAssuranceHigh,
		},
		servicioAV: domain.Principal{
			ID: personaAutorizacionPrueba("servicio:antivirus:0123456789abcdef"), AuthMethod: domain.AuthMethodCertificate,
			AuthAssurance: domain.AuthAssuranceHigh,
		},
	}
}

func TestServicioCargaDocumentalExigeDependenciasNoNulasYUnaSolaRaizDeRecibos(t *testing.T) {
	e := nuevoEntornoServicioCarga(t)
	sellador := selladorCargaPrueba{dominio: "carga"}
	opciones := OpcionesServicioCargaDocumental{
		ConectorAlmacenPermitido:    "almacen_s3_corporativo",
		ConectorAnalizadorPermitido: "analizador_icap_corporativo",
		VersionAnalizadorPermitida:  1,
	}
	otraRaiz := &seguridadCargaPrueba{reloj: e.reloj}
	if _, err := NuevoServicioCargaDocumental(
		e.repositorio, e.autorizador, e.almacen, e.almacen, e.analizador,
		sellador, sellador, sellador, e.seguridad,
		e.seguridad, e.seguridad, otraRaiz, &generadorIDCargaPrueba{}, e.reloj, opciones,
	); !errors.Is(err, ErrDependenciaCargaDocumentalRequerida) {
		t.Fatalf("se mezclaron raices criptograficas: %v", err)
	}
	var repositorioNulo *repositorioCargaPrueba
	if _, err := NuevoServicioCargaDocumental(
		repositorioNulo, e.autorizador, e.almacen, e.almacen, e.analizador,
		sellador, sellador, sellador, e.seguridad,
		e.seguridad, e.seguridad, e.seguridad, &generadorIDCargaPrueba{}, e.reloj, opciones,
	); !errors.Is(err, ErrDependenciaCargaDocumentalRequerida) {
		t.Fatalf("se acepto un repositorio nulo tipado: %v", err)
	}
	var seguridadNula *seguridadCargaPrueba
	if _, err := NuevoServicioCargaDocumental(
		e.repositorio, e.autorizador, e.almacen, e.almacen, e.analizador,
		sellador, sellador, sellador, seguridadNula,
		seguridadNula, seguridadNula, seguridadNula, &generadorIDCargaPrueba{}, e.reloj, opciones,
	); !errors.Is(err, ErrDependenciaCargaDocumentalRequerida) {
		t.Fatalf("se acepto una proteccion de recibos nula tipada: %v", err)
	}
}

func (e *entornoServicioCarga) ordenPreparar() OrdenPrepararCargaDocumental {
	suma := sha256.Sum256(e.contenido)
	return OrdenPrepararCargaDocumental{
		Principal: e.externo, PerfilActivo: perfilAutorizacionPrueba("externo:0123456789abcdef"), Recurso: e.recurso,
		OperacionRef: "operacion:carga:0123456789abcdef", Finalidad: "aportar_documentacion_bolsa",
		Clasificacion: "datos_personales", MIME: "application/pdf", Tamano: int64(len(e.contenido)),
		HuellaSHA256: hex.EncodeToString(suma[:]), ClaveIdempotenciaCliente: "reintento:0123456789abcdef",
		Motivo: "Aportacion de merito", CorrelacionRef: "correlacion:0123456789abcdef",
	}
}

func exigirRecursoAlmacenCargaPrueba(
	t *testing.T,
	solicitud domain.SolicitudAutorizacion,
	carga domain.CargaDocumental,
	objeto ports.ReferenciaObjetoAlmacen,
) {
	t.Helper()
	atributos := solicitud.Recurso.Atributos
	cargaRef := carga.ID
	if solicitud.Accion == AccionPrepararCargaDocumental || solicitud.Accion == AccionConfirmarCargaDocumental {
		cargaRef = carga.IndiceIdempotenciaHMAC
	}
	esperados := map[string]string{
		ports.AtributoAlmacenOperacionRef:        carga.OperacionRef,
		ports.AtributoAlmacenCargaRef:            cargaRef,
		ports.AtributoAlmacenClasificacion:       carga.Clasificacion,
		ports.AtributoAlmacenHuellaSolicitudHMAC: carga.HuellaSolicitudHMAC,
	}
	for clave, esperado := range esperados {
		if atributos[clave] != esperado {
			t.Fatalf("accion %s: atributo %s=%q; esperado %q", solicitud.Accion, clave, atributos[clave], esperado)
		}
	}
	seudonimo := atributos[ports.AtributoAlmacenSujetoSeudonimoHMAC]
	efecto := atributos[ports.AtributoAlmacenEfectoRef]
	if !hmacCargaDocumentalValido(seudonimo) || seudonimo == carga.PrincipalID ||
		!strings.HasPrefix(efecto, "efecto-carga:sha256:") {
		t.Fatalf("accion %s: seudonimo o efecto no acotado", solicitud.Accion)
	}
	requiereObjeto := solicitud.Accion == AccionAnalizarCargaDocumental ||
		solicitud.Accion == AccionPromoverCargaDocumental
	objetoRef, existeRef := atributos[ports.AtributoAlmacenObjetoRef]
	objetoVersion, existeVersion := atributos[ports.AtributoAlmacenObjetoVersion]
	if requiereObjeto {
		if !existeRef || !existeVersion || objetoRef != objeto.Referencia || objetoVersion != objeto.Version {
			t.Fatalf("accion %s: objeto del recurso no es exacto", solicitud.Accion)
		}
	} else if existeRef || existeVersion {
		t.Fatalf("accion %s: aparecio un objeto no autorizado", solicitud.Accion)
	}
	if solicitud.Recurso.Ambitos["proceso"] != referenciaProcesoCargaPrueba || solicitud.Recurso.Validar() != nil {
		t.Fatalf("accion %s: se perdio o invalido el contexto servidor", solicitud.Accion)
	}
}

const referenciaProcesoCargaPrueba = "proceso:0123456789abcdef"

func (e *entornoServicioCarga) prepararYConfirmarPrueba(t *testing.T) domain.CargaDocumental {
	t.Helper()
	preparada, err := e.servicio.Preparar(context.Background(), e.ordenPreparar())
	if err != nil {
		t.Fatalf("preparar: %v", err)
	}
	cuarentena, err := e.servicio.Confirmar(context.Background(), OrdenConfirmarCargaDocumental{
		Principal: e.externo, PerfilActivo: perfilAutorizacionPrueba("externo:0123456789abcdef"), Recurso: e.recurso,
		CargaID: preparada.Carga.ID, SesionRef: e.almacen.sesion, Finalidad: preparada.Carga.Finalidad,
		Recibo: preparada.Recibo, Motivo: "Confirmar aportacion", CorrelacionRef: preparada.Carga.CorrelacionRef,
	})
	if err != nil {
		t.Fatalf("confirmar: %v", err)
	}
	return cuarentena
}

func TestServicioCargaDocumentalCompletaLaCadenaConCuatroAutorizaciones(t *testing.T) {
	e := nuevoEntornoServicioCarga(t)
	cuarentena := e.prepararYConfirmarPrueba(t)
	if cuarentena.Estado != domain.EstadoCargaDocumentalCuarentena || e.almacen.confirmaciones != 1 {
		t.Fatalf("estado tras confirmacion: %#v", cuarentena)
	}
	admitida, err := e.servicio.AnalizarYPromover(context.Background(), OrdenAnalizarCargaDocumental{
		Principal: e.servicioAV, PerfilActivo: perfilAutorizacionPrueba("servicio:antivirus"), Recurso: e.recurso,
		CargaID: cuarentena.ID, Finalidad: cuarentena.Finalidad, Motivo: "Analisis automatico obligatorio",
		CorrelacionRef: cuarentena.CorrelacionRef,
	})
	if err != nil {
		t.Fatalf("analizar y promover: %v", err)
	}
	if admitida.Estado != domain.EstadoCargaDocumentalAdmitida || admitida.Version != 5 ||
		e.analizador.llamadas != 1 || e.almacen.promociones != 1 || len(e.repositorio.confirmaciones) != 4 {
		t.Fatalf("cadena incompleta: carga=%#v analisis=%d promociones=%d transiciones=%d",
			admitida, e.analizador.llamadas, e.almacen.promociones, len(e.repositorio.confirmaciones))
	}
	acciones := make([]string, len(e.autorizador.solicitudes))
	for indice, solicitud := range e.autorizador.solicitudes {
		acciones[indice] = solicitud.Accion
	}
	esperadas := []string{AccionPrepararCargaDocumental, AccionConfirmarCargaDocumental,
		AccionAnalizarCargaDocumental, AccionPromoverCargaDocumental}
	if fmt.Sprint(acciones) != fmt.Sprint(esperadas) {
		t.Fatalf("autorizaciones: %v, esperadas %v", acciones, esperadas)
	}
	efectos := make(map[string]struct{}, len(e.autorizador.solicitudes))
	objetoCuarentena := objetoAlmacenDesdeContenido(*cuarentena.ContenidoCuarentena)
	for _, solicitud := range e.autorizador.solicitudes {
		objeto := ports.ReferenciaObjetoAlmacen{}
		if solicitud.Accion == AccionAnalizarCargaDocumental || solicitud.Accion == AccionPromoverCargaDocumental {
			objeto = objetoCuarentena
		}
		exigirRecursoAlmacenCargaPrueba(t, solicitud, admitida, objeto)
		efectos[solicitud.Recurso.Atributos[ports.AtributoAlmacenEfectoRef]] = struct{}{}
	}
	if len(efectos) != len(e.autorizador.solicitudes) {
		t.Fatal("dos acciones de negocio compartieron el mismo efecto autorizado")
	}
	if e.repositorio.confirmaciones[2].Auditoria.ActorID != e.servicioAV.ID ||
		e.repositorio.confirmaciones[3].Auditoria.ActorID != e.servicioAV.ID {
		t.Fatal("el analisis o la promocion se atribuyeron al usuario externo")
	}
}

func TestServicioCargaDocumentalConfirmarFallaCerradaSinManifiestoNiEfectos(t *testing.T) {
	e := nuevoEntornoServicioCarga(t)
	preparada, err := e.servicio.Preparar(context.Background(), e.ordenPreparar())
	if err != nil {
		t.Fatal(err)
	}
	// Simula una persistencia historica incompleta: el agregado preparado no
	// concede por si solo autoridad para reconstruir ni confirmar la carga.
	e.repositorio.mu.Lock()
	e.repositorio.manifiesto = domain.ManifiestoPreparacionCargaDirectaV1{}
	e.repositorio.mu.Unlock()
	_, err = e.servicio.Confirmar(context.Background(), OrdenConfirmarCargaDocumental{
		Principal: e.externo, PerfilActivo: perfilAutorizacionPrueba("externo:0123456789abcdef"), Recurso: e.recurso,
		CargaID: preparada.Carga.ID, SesionRef: e.almacen.sesion, Recibo: preparada.Recibo,
		Finalidad: preparada.Carga.Finalidad, Motivo: "Confirmar aportacion",
		CorrelacionRef: preparada.Carga.CorrelacionRef,
	})
	if !errors.Is(err, ports.ErrManifiestoPreparacionNoEncontrado) {
		t.Fatalf("confirmacion sin manifiesto durable no denegada: %v", err)
	}
	if len(e.autorizador.solicitudes) != 1 {
		t.Fatalf("el agregado sin manifiesto alcanzo el PDP de confirmacion: %d solicitudes", len(e.autorizador.solicitudes))
	}
	if e.seguridad.siguienteConsumo != 0 || e.almacen.intentosConfirmar != 0 ||
		e.almacen.confirmaciones != 0 || e.repositorio.carga.Estado != domain.EstadoCargaDocumentalPreparada {
		t.Fatalf("confirmacion cerrada produjo efectos: consumos=%d intentos=%d confirmaciones=%d estado=%s",
			e.seguridad.siguienteConsumo, e.almacen.intentosConfirmar,
			e.almacen.confirmaciones, e.repositorio.carga.Estado)
	}
}

func TestServicioCargaDocumentalConfirmarRechazaTamperDelContextoABACBaseAntesDelPDP(t *testing.T) {
	pruebas := []struct {
		nombre string
		muta   func(*domain.RecursoAutorizable)
	}{
		{"ambito", func(r *domain.RecursoAutorizable) {
			r.Ambitos["proceso"] = "proceso:alterado:0123456789abcdef"
		}},
		{"atributo_existente", func(r *domain.RecursoAutorizable) {
			r.Atributos["categoria"] = "administrativo"
		}},
		{"atributo_nuevo", func(r *domain.RecursoAutorizable) {
			r.Atributos["unidad"] = "recursos_humanos"
		}},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			e := nuevoEntornoServicioCarga(t)
			e.recurso.Atributos = map[string]string{"categoria": "auxiliar_administrativo"}
			preparada, err := e.servicio.Preparar(context.Background(), e.ordenPreparar())
			if err != nil {
				t.Fatal(err)
			}
			prueba.muta(&e.recurso)
			_, err = e.servicio.Confirmar(context.Background(), OrdenConfirmarCargaDocumental{
				Principal: e.externo, PerfilActivo: perfilAutorizacionPrueba("externo:0123456789abcdef"),
				Recurso: e.recurso, CargaID: preparada.Carga.ID, SesionRef: e.almacen.sesion,
				Recibo: preparada.Recibo, Finalidad: preparada.Carga.Finalidad, Motivo: "Confirmar aportacion",
				CorrelacionRef: preparada.Carga.CorrelacionRef,
			})
			if !errors.Is(err, domain.ErrAutorizacionDenegada) ||
				!errors.Is(err, ports.ErrConfirmacionCargaDocumentalInvalida) {
				t.Fatalf("tamper ABAC no denegado: %v", err)
			}
			if len(e.autorizador.solicitudes) != 1 || e.seguridad.siguienteConsumo != 0 ||
				e.almacen.intentosConfirmar != 0 || e.repositorio.carga.Estado != domain.EstadoCargaDocumentalPreparada {
				t.Fatalf("el tamper alcanzo PDP o produjo efectos: pdp=%d consumos=%d intentos=%d estado=%s",
					len(e.autorizador.solicitudes), e.seguridad.siguienteConsumo,
					e.almacen.intentosConfirmar, e.repositorio.carga.Estado)
			}
		})
	}
}

func TestServicioCargaDocumentalCopiaLaOrdenAntesDeAutorizarYPersistir(t *testing.T) {
	e := nuevoEntornoServicioCarga(t)
	e.recurso.Atributos = map[string]string{"categoria": "auxiliar_administrativo"}
	orden := e.ordenPreparar()
	huellaBaseInicial, err := ports.HuellaRecursoBaseCargaDocumental(orden.Recurso)
	if err != nil {
		t.Fatal(err)
	}
	e.autorizador.detenidoEnExigir = make(chan struct{})
	e.autorizador.continuarExigir = make(chan struct{})
	type resultadoPreparacion struct {
		resultado ResultadoPrepararCargaDocumental
		err       error
	}
	resultados := make(chan resultadoPreparacion, 1)
	go func() {
		resultado, errPreparacion := e.servicio.Preparar(context.Background(), orden)
		resultados <- resultadoPreparacion{resultado: resultado, err: errPreparacion}
	}()
	<-e.autorizador.detenidoEnExigir
	orden.Recurso.Ambitos["proceso"] = "proceso:mutado:0123456789abcdef"
	orden.Recurso.Atributos["categoria"] = "administrativo"
	orden.Principal.Roles = append(orden.Principal.Roles, "rol_mutado")
	close(e.autorizador.continuarExigir)
	resultado := <-resultados
	if resultado.err != nil {
		t.Fatalf("la mutacion externa alcanzo la instantanea: %v", resultado.err)
	}
	datos, err := e.repositorio.manifiesto.Datos()
	if err != nil {
		t.Fatal(err)
	}
	if datos.HuellaRecursoBaseSHA256 != huellaBaseInicial ||
		resultado.resultado.Carga.Estado != domain.EstadoCargaDocumentalPreparada {
		t.Fatal("se persistio una orden distinta de la copia validada")
	}
}

func TestServicioCargaDocumentalDeniegaAccionNoConcedidaSinEfectos(t *testing.T) {
	e := nuevoEntornoServicioCarga(t)
	delete(e.autorizador.permitidas, AccionPrepararCargaDocumental)
	_, err := e.servicio.Preparar(context.Background(), e.ordenPreparar())
	if !errors.Is(err, domain.ErrAutorizacionDenegada) {
		t.Fatalf("se esperaba denegacion: %v", err)
	}
	if e.almacen.preparaciones != 0 || e.repositorio.carga.ID != "" {
		t.Fatal("una accion no concedida produjo efectos")
	}
}

func TestServicioCargaDocumentalDeniegaCamposNoConcedidosSinEfectos(t *testing.T) {
	e := nuevoEntornoServicioCarga(t)
	e.autorizador.campos = map[string][]string{AccionPrepararCargaDocumental: {CampoCargaMIME}}
	_, err := e.servicio.Preparar(context.Background(), e.ordenPreparar())
	if !errors.Is(err, domain.ErrAutorizacionDenegada) {
		t.Fatalf("se esperaba denegacion por campos incompletos: %v", err)
	}
	if e.almacen.preparaciones != 0 || e.repositorio.carga.ID != "" {
		t.Fatal("una decision sin todos los campos produjo efectos")
	}
}

func TestServicioCargaDocumentalDeniegaObligacionNoImplementadaSinEfectos(t *testing.T) {
	e := nuevoEntornoServicioCarga(t)
	e.autorizador.obligaciones = map[string][]string{
		AccionPrepararCargaDocumental: {"doble_control_no_implementado"},
	}
	_, err := e.servicio.Preparar(context.Background(), e.ordenPreparar())
	if !errors.Is(err, domain.ErrAutorizacionDenegada) {
		t.Fatalf("se esperaba denegacion por obligacion desconocida: %v", err)
	}
	if e.almacen.preparaciones != 0 || e.repositorio.carga.ID != "" {
		t.Fatal("una obligacion ignorada produjo efectos")
	}
}

func TestServicioCargaDocumentalRechazaAtributoAlmacenInyectadoAntesDelPDP(t *testing.T) {
	e := nuevoEntornoServicioCarga(t)
	e.recurso.Atributos = map[string]string{
		ports.AtributoAlmacenEfectoRef: "efecto:carga:declarado-por-cliente",
	}
	_, err := e.servicio.Preparar(context.Background(), e.ordenPreparar())
	if !errors.Is(err, domain.ErrAutorizacionDenegada) ||
		!errors.Is(err, ports.ErrAutorizacionAlmacenInvalida) {
		t.Fatalf("atributo reservado no fue rechazado en cerrado: %v", err)
	}
	if len(e.autorizador.solicitudes) != 0 || e.almacen.preparaciones != 0 || e.repositorio.carga.ID != "" {
		t.Fatalf("la inyeccion alcanzo PDP o produjo efectos: pdp=%d almacen=%d",
			len(e.autorizador.solicitudes), e.almacen.preparaciones)
	}
}

func TestServicioCargaDocumentalNoReemiteSinTransaccionComunEntreManifiestoYRecibo(t *testing.T) {
	e := nuevoEntornoServicioCarga(t)
	if _, err := e.servicio.Preparar(context.Background(), e.ordenPreparar()); err != nil {
		t.Fatalf("primera preparacion: %v", err)
	}
	huellaAntes, err := e.repositorio.manifiesto.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	_, err = e.servicio.Preparar(context.Background(), e.ordenPreparar())
	if !errors.Is(err, ErrCargaDocumentalYaProcesada) {
		t.Fatalf("se acepto una reemision sin commit comun: %v", err)
	}
	huellaDespues, err := e.repositorio.manifiesto.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	if huellaAntes != huellaDespues || e.almacen.preparaciones != 1 || e.almacen.abandonos != 0 ||
		e.seguridad.siguienteEmision != 1 || len(e.repositorio.confirmaciones) != 1 ||
		e.repositorio.carga.Estado != domain.EstadoCargaDocumentalPreparada {
		t.Fatalf("la reemision altero autoridad durable: manifiesto=%v preparaciones=%d abandonos=%d recibos=%d transiciones=%d estado=%s",
			huellaAntes == huellaDespues, e.almacen.preparaciones, e.almacen.abandonos,
			e.seguridad.siguienteEmision, len(e.repositorio.confirmaciones), e.repositorio.carga.Estado)
	}
	if len(e.almacen.contextos) != 1 {
		t.Fatalf("la reemision alcanzo el almacen: %d contextos", len(e.almacen.contextos))
	}
}

func TestServicioCargaDocumentalFalloRemotoRetiraClaimYReintentaConDecisionNueva(t *testing.T) {
	e := nuevoEntornoServicioCarga(t)
	e.almacen.falloPreparacion = errors.New("fallo remoto de preparacion")
	if _, err := e.servicio.Preparar(context.Background(), e.ordenPreparar()); !errors.Is(err, ports.ErrInstruccionesCargaDirectaNoValidas) {
		t.Fatalf("fallo remoto no normalizado: %v", err)
	}
	if !e.repositorio.abandonada || e.repositorio.decisionPreparacionActiva != "" ||
		e.almacen.preparaciones != 1 || len(e.repositorio.decisionesPreparacion) != 1 {
		t.Fatalf("claim remoto fallido ambiguo: abandono=%v activa=%q intentos=%d decisiones=%d",
			e.repositorio.abandonada, e.repositorio.decisionPreparacionActiva,
			e.almacen.preparaciones, len(e.repositorio.decisionesPreparacion))
	}
	e.almacen.falloPreparacion = nil
	preparada, err := e.servicio.Preparar(context.Background(), e.ordenPreparar())
	if err != nil {
		t.Fatalf("reintento con decision nueva: %v", err)
	}
	if preparada.Carga.Estado != domain.EstadoCargaDocumentalPreparada ||
		e.almacen.preparaciones != 2 || len(e.repositorio.decisionesPreparacion) != 2 {
		t.Fatalf("reintento incompleto: estado=%s intentos=%d decisiones=%d",
			preparada.Carga.Estado, e.almacen.preparaciones, len(e.repositorio.decisionesPreparacion))
	}
}

func TestServicioCargaDocumentalManifiestoLigaLaDecisionExactaDePreparacion(t *testing.T) {
	e := nuevoEntornoServicioCarga(t)
	preparada, err := e.servicio.Preparar(context.Background(), e.ordenPreparar())
	if err != nil {
		t.Fatalf("preparar: %v", err)
	}
	datos, err := e.repositorio.manifiesto.Datos()
	if err != nil {
		t.Fatal(err)
	}
	if len(e.almacen.contextos) != 1 {
		t.Fatalf("contextos de preparacion=%d", len(e.almacen.contextos))
	}
	contexto := debeProyectarContextoCargaPrueba(e.almacen.contextos[0])
	huellaBase, err := ports.HuellaRecursoBaseCargaDocumental(e.recurso)
	if err != nil {
		t.Fatal(err)
	}
	if datos.DecisionRef != preparada.Carga.AutorizacionPreparacionRef ||
		datos.DecisionRef != contexto.AutorizacionRef ||
		datos.HuellaRecursoBaseSHA256 != huellaBase ||
		datos.HuellaDecisionSHA256 != contexto.HuellaDecisionSHA256 ||
		datos.EfectoRef != contexto.EfectoRef ||
		datos.HuellaPlanEfectoSHA256 != contexto.HuellaPlanEfectoSHA256 {
		t.Fatal("el manifiesto no fijo la decision y el plan exactos de preparacion")
	}
}

func TestServicioCargaDocumentalRechazaSesionDeOtraCargaAntesDeConfirmar(t *testing.T) {
	e := nuevoEntornoServicioCarga(t)
	preparada, err := e.servicio.Preparar(context.Background(), e.ordenPreparar())
	if err != nil {
		t.Fatal(err)
	}
	_, err = e.servicio.Confirmar(context.Background(), OrdenConfirmarCargaDocumental{
		Principal: e.externo, PerfilActivo: perfilAutorizacionPrueba("externo:0123456789abcdef"), Recurso: e.recurso,
		CargaID: preparada.Carga.ID, SesionRef: "sesion:carga:otra:0123456789abcdef",
		Recibo: preparada.Recibo, Finalidad: preparada.Carga.Finalidad, Motivo: "Confirmar aportacion",
		CorrelacionRef: preparada.Carga.CorrelacionRef,
	})
	if !errors.Is(err, ports.ErrSesionCargaDirectaNoValida) || e.almacen.confirmaciones != 0 {
		t.Fatalf("sesion cruzada no denegada antes del almacen: %v", err)
	}
}

func TestServicioCargaDocumentalNoNormalizaIdentificadoresParaConceder(t *testing.T) {
	e := nuevoEntornoServicioCarga(t)
	preparada, err := e.servicio.Preparar(context.Background(), e.ordenPreparar())
	if err != nil {
		t.Fatal(err)
	}
	base := OrdenConfirmarCargaDocumental{
		Principal: e.externo, PerfilActivo: perfilAutorizacionPrueba("externo:0123456789abcdef"), Recurso: e.recurso,
		CargaID: preparada.Carga.ID, SesionRef: e.almacen.sesion, Recibo: preparada.Recibo,
		Finalidad: preparada.Carga.Finalidad, Motivo: "Confirmar aportacion",
		CorrelacionRef: preparada.Carga.CorrelacionRef,
	}
	pruebas := []struct {
		nombre string
		muta   func(*OrdenConfirmarCargaDocumental)
	}{
		{"carga", func(o *OrdenConfirmarCargaDocumental) { o.CargaID = " " + o.CargaID }},
		{"sesion", func(o *OrdenConfirmarCargaDocumental) { o.SesionRef += " " }},
		{"perfil", func(o *OrdenConfirmarCargaDocumental) { o.PerfilActivo += " " }},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			orden := base
			prueba.muta(&orden)
			if _, err := e.servicio.Confirmar(context.Background(), orden); !errors.Is(err, ErrOrdenCargaDocumentalInvalida) {
				t.Fatalf("entrada no canonica aceptada o normalizada: %v", err)
			}
		})
	}
	if e.seguridad.siguienteConsumo != 0 || e.almacen.intentosConfirmar != 0 {
		t.Fatalf("una entrada no canonica produjo efectos: consumos=%d intentos=%d",
			e.seguridad.siguienteConsumo, e.almacen.intentosConfirmar)
	}
}

func TestServicioCargaDocumentalNoCorrigeIdentificadorGenerado(t *testing.T) {
	e := nuevoEntornoServicioCarga(t)
	e.servicio.generadorID = &generadorIDCargaPrueba{valor: " carga:documental:no-canonica "}
	if _, err := e.servicio.Preparar(context.Background(), e.ordenPreparar()); !errors.Is(err, domain.ErrCargaDocumentalInvalida) {
		t.Fatalf("identificador no canonico corregido: %v", err)
	}
	if e.repositorio.carga.ID != "" || e.almacen.preparaciones != 0 {
		t.Fatal("el identificador generado no canonico produjo efectos")
	}
}

func TestServicioCargaDocumentalRespuestaRemotaAmbiguaQuedaPendienteSinReusarRecibo(t *testing.T) {
	e := nuevoEntornoServicioCarga(t)
	preparada, err := e.servicio.Preparar(context.Background(), e.ordenPreparar())
	if err != nil {
		t.Fatal(err)
	}
	errorRemoto := errors.New("detalle interno sensible del almacen")
	e.almacen.errorConfirmacion = errorRemoto
	orden := OrdenConfirmarCargaDocumental{
		Principal: e.externo, PerfilActivo: perfilAutorizacionPrueba("externo:0123456789abcdef"), Recurso: e.recurso,
		CargaID: preparada.Carga.ID, SesionRef: e.almacen.sesion, Recibo: preparada.Recibo,
		Finalidad: preparada.Carga.Finalidad, Motivo: "Confirmar aportacion",
		CorrelacionRef: preparada.Carga.CorrelacionRef,
	}
	if _, err = e.servicio.Confirmar(context.Background(), orden); !errors.Is(err, ports.ErrConfirmacionCargaDirectaNoDisponible) ||
		!errors.Is(err, ports.ErrConfirmacionCargaDocumentalPendiente) {
		t.Fatalf("primer intento: %v", err)
	}
	if errors.Is(err, errorRemoto) || strings.Contains(err.Error(), errorRemoto.Error()) {
		t.Fatalf("se filtro el error remoto: %v", err)
	}
	if e.seguridad.siguienteConsumo != 1 || e.almacen.intentosConfirmar != 1 ||
		e.almacen.confirmaciones != 0 || e.repositorio.carga.Estado != domain.EstadoCargaDocumentalPreparada {
		t.Fatalf("la respuesta ambigua no quedo pendiente: consumos=%d intentos=%d confirmaciones=%d estado=%s",
			e.seguridad.siguienteConsumo, e.almacen.intentosConfirmar, e.almacen.confirmaciones,
			e.repositorio.carga.Estado)
	}
	e.almacen.errorConfirmacion = nil
	if _, err = e.servicio.Confirmar(context.Background(), orden); !errors.Is(err, ports.ErrReciboCargaDirectaNoValido) {
		t.Fatalf("el recibo consumido se reutilizo: %v", err)
	}
	if e.seguridad.siguienteConsumo != 1 || e.almacen.intentosConfirmar != 1 || e.almacen.confirmaciones != 0 {
		t.Fatalf("el reintento repitio el efecto: consumos=%d intentos=%d confirmaciones=%d",
			e.seguridad.siguienteConsumo, e.almacen.intentosConfirmar, e.almacen.confirmaciones)
	}
}

func TestServicioCargaDocumentalConfirmacionConcurrenteConsumeUnaSolaVez(t *testing.T) {
	e := nuevoEntornoServicioCarga(t)
	preparada, err := e.servicio.Preparar(context.Background(), e.ordenPreparar())
	if err != nil {
		t.Fatal(err)
	}
	orden := OrdenConfirmarCargaDocumental{
		Principal: e.externo, PerfilActivo: perfilAutorizacionPrueba("externo:0123456789abcdef"), Recurso: e.recurso,
		CargaID: preparada.Carga.ID, SesionRef: e.almacen.sesion, Recibo: preparada.Recibo,
		Finalidad: preparada.Carga.Finalidad, Motivo: "Confirmar aportacion",
		CorrelacionRef: preparada.Carga.CorrelacionRef,
	}
	type resultadoConfirmacion struct {
		carga domain.CargaDocumental
		err   error
	}
	inicio := make(chan struct{})
	resultados := make(chan resultadoConfirmacion, 2)
	for i := 0; i < 2; i++ {
		go func() {
			<-inicio
			carga, errConfirmacion := e.servicio.Confirmar(context.Background(), orden)
			resultados <- resultadoConfirmacion{carga: carga, err: errConfirmacion}
		}()
	}
	close(inicio)
	exitos := 0
	for i := 0; i < 2; i++ {
		resultado := <-resultados
		if resultado.err == nil {
			exitos++
			if resultado.carga.Estado != domain.EstadoCargaDocumentalCuarentena {
				t.Fatalf("estado confirmado=%s", resultado.carga.Estado)
			}
			continue
		}
		if !errors.Is(resultado.err, ports.ErrReciboCargaDirectaNoValido) &&
			!errors.Is(resultado.err, ports.ErrManifiestoPreparacionNoEncontrado) {
			t.Fatalf("error concurrente inesperado: %v", resultado.err)
		}
	}
	if exitos != 1 || e.seguridad.siguienteConsumo != 1 || e.almacen.intentosConfirmar != 1 ||
		e.almacen.confirmaciones != 1 || e.repositorio.carga.Estado != domain.EstadoCargaDocumentalCuarentena {
		t.Fatalf("confirmacion concurrente no fue unica: exitos=%d consumos=%d intentos=%d confirmaciones=%d estado=%s",
			exitos, e.seguridad.siguienteConsumo, e.almacen.intentosConfirmar,
			e.almacen.confirmaciones, e.repositorio.carga.Estado)
	}
}

func TestServicioCargaDocumentalRechazaReciboDeOtraPreparacionAntesDelAlmacen(t *testing.T) {
	destino := nuevoEntornoServicioCarga(t)
	preparadaDestino, err := destino.servicio.Preparar(context.Background(), destino.ordenPreparar())
	if err != nil {
		t.Fatal(err)
	}
	origen := nuevoEntornoServicioCarga(t)
	ordenOrigen := origen.ordenPreparar()
	ordenOrigen.OperacionRef = "operacion:carga:otra:0123456789abcdef"
	ordenOrigen.CorrelacionRef = "correlacion:carga:otra:0123456789abcdef"
	preparadaOrigen, err := origen.servicio.Preparar(context.Background(), ordenOrigen)
	if err != nil {
		t.Fatal(err)
	}
	_, err = destino.servicio.Confirmar(context.Background(), OrdenConfirmarCargaDocumental{
		Principal: destino.externo, PerfilActivo: perfilAutorizacionPrueba("externo:0123456789abcdef"), Recurso: destino.recurso,
		CargaID: preparadaDestino.Carga.ID, SesionRef: destino.almacen.sesion, Recibo: preparadaOrigen.Recibo,
		Finalidad: preparadaDestino.Carga.Finalidad, Motivo: "Confirmar aportacion",
		CorrelacionRef: preparadaDestino.Carga.CorrelacionRef,
	})
	if !errors.Is(err, ports.ErrReciboCargaDirectaNoValido) ||
		destino.seguridad.siguienteConsumo != 0 || destino.almacen.intentosConfirmar != 0 {
		t.Fatalf("recibo cruzado no quedo cerrado: error=%v consumos=%d intentos=%d",
			err, destino.seguridad.siguienteConsumo, destino.almacen.intentosConfirmar)
	}
}

func TestServicioCargaDocumentalSeudonimizaElSujetoYSeparaCadaAccionTecnica(t *testing.T) {
	e := nuevoEntornoServicioCarga(t)
	cuarentena := e.prepararYConfirmarPrueba(t)
	if _, err := e.servicio.AnalizarYPromover(context.Background(), OrdenAnalizarCargaDocumental{
		Principal: e.servicioAV, PerfilActivo: perfilAutorizacionPrueba("servicio:antivirus"), Recurso: e.recurso,
		CargaID: cuarentena.ID, Finalidad: cuarentena.Finalidad, Motivo: "Analisis automatico obligatorio",
		CorrelacionRef: cuarentena.CorrelacionRef,
	}); err != nil {
		t.Fatal(err)
	}
	esperadas := []string{
		ports.AccionAlmacenPrepararCargaDirecta,
		ports.AccionAlmacenConfirmarCargaDirecta,
		ports.AccionAlmacenLeer,
		ports.AccionAlmacenPromover,
	}
	if len(e.almacen.contextos) != len(esperadas) {
		t.Fatalf("contextos de almacen=%d, esperados=%d", len(e.almacen.contextos), len(esperadas))
	}
	for indice, contexto := range e.almacen.contextos {
		proyeccion := debeProyectarContextoCargaPrueba(contexto)
		if proyeccion.AccionTecnica != esperadas[indice] || proyeccion.SujetoSeudonimoHMAC == e.externo.ID ||
			!strings.HasPrefix(proyeccion.SujetoSeudonimoHMAC, "hmac-sha256:seudonimo_v1:") ||
			strings.Contains(fmt.Sprintf("%#v", contexto), e.externo.ID) {
			t.Fatalf("contexto tecnico %d no acotado o contiene el sujeto real: %#v", indice, contexto)
		}
	}
	if len(e.analizador.contextos) != 1 {
		t.Fatalf("contextos del analizador=%d", len(e.analizador.contextos))
	}
	lectura := debeProyectarContextoCargaPrueba(e.almacen.contextos[2])
	analisis := debeProyectarContextoCargaPrueba(e.analizador.contextos[0])
	if analisis.AccionTecnica != ports.AccionAlmacenAnalizarContenido ||
		analisis.AccionTecnica == lectura.AccionTecnica || analisis.PasoRef == lectura.PasoRef ||
		analisis.AccionNegocio != AccionAnalizarCargaDocumental ||
		analisis.EfectoRef != lectura.EfectoRef ||
		analisis.HuellaPlanEfectoSHA256 != lectura.HuellaPlanEfectoSHA256 ||
		analisis.HuellaDecisionSHA256 != lectura.HuellaDecisionSHA256 {
		t.Fatalf("lectura y analisis no quedaron separados: almacen=%#v analizador=%#v",
			e.almacen.contextos[2], e.analizador.contextos)
	}
}

func TestAnalisisNoConcluyenteParcialQuedaRetenidoSinPromocion(t *testing.T) {
	e := nuevoEntornoServicioCarga(t)
	cuarentena := e.prepararYConfirmarPrueba(t)
	e.analizador.estado = ports.EstadoAnalisisContenidoNoConcluyente
	e.analizador.leerSolo = int64(len(e.contenido) / 2)
	retenida, err := e.servicio.AnalizarYPromover(context.Background(), OrdenAnalizarCargaDocumental{
		Principal: e.servicioAV, PerfilActivo: perfilAutorizacionPrueba("servicio:antivirus"), Recurso: e.recurso,
		CargaID: cuarentena.ID, Finalidad: cuarentena.Finalidad, Motivo: "Analisis automatico obligatorio",
		CorrelacionRef: cuarentena.CorrelacionRef,
	})
	if err != nil {
		t.Fatalf("registrar resultado no concluyente: %v", err)
	}
	if retenida.Estado != domain.EstadoCargaDocumentalRetenidaSeguridad || e.almacen.promociones != 0 ||
		len(e.autorizador.solicitudes) != 3 {
		t.Fatalf("no quedo retenida: %#v promociones=%d autorizaciones=%d",
			retenida, e.almacen.promociones, len(e.autorizador.solicitudes))
	}
}

func TestAnalisisDeOtroObjetoNoCambiaElAgregado(t *testing.T) {
	e := nuevoEntornoServicioCarga(t)
	cuarentena := e.prepararYConfirmarPrueba(t)
	e.analizador.objetoAlterado = true
	_, err := e.servicio.AnalizarYPromover(context.Background(), OrdenAnalizarCargaDocumental{
		Principal: e.servicioAV, PerfilActivo: perfilAutorizacionPrueba("servicio:antivirus"), Recurso: e.recurso,
		CargaID: cuarentena.ID, Finalidad: cuarentena.Finalidad, Motivo: "Analisis automatico obligatorio",
		CorrelacionRef: cuarentena.CorrelacionRef,
	})
	if !errors.Is(err, ErrResultadoCargaDocumentalInvalido) {
		t.Fatalf("se esperaba rechazo del objeto cruzado: %v", err)
	}
	guardada, _ := e.repositorio.Obtener(context.Background(), cuarentena.ID)
	if guardada.Estado != domain.EstadoCargaDocumentalCuarentena || e.almacen.promociones != 0 {
		t.Fatalf("el resultado invalido cambio el estado: %#v", guardada)
	}
}

func TestAnalisisRechazaConectorNoIncluidoEnLaListaPositiva(t *testing.T) {
	e := nuevoEntornoServicioCarga(t)
	cuarentena := e.prepararYConfirmarPrueba(t)
	e.analizador.conectorID = "analizador_no_autorizado"
	_, err := e.servicio.AnalizarYPromover(context.Background(), OrdenAnalizarCargaDocumental{
		Principal: e.servicioAV, PerfilActivo: perfilAutorizacionPrueba("servicio:antivirus"), Recurso: e.recurso,
		CargaID: cuarentena.ID, Finalidad: cuarentena.Finalidad, Motivo: "Analisis automatico obligatorio",
		CorrelacionRef: cuarentena.CorrelacionRef,
	})
	if !errors.Is(err, ports.ErrCapacidadAnalisisContenidoNoDisponible) || e.analizador.llamadas != 0 {
		t.Fatalf("conector no autorizado aceptado: error=%v llamadas=%d", err, e.analizador.llamadas)
	}
}

func TestAnalisisLimpioRechazaMIMEDetectadoDistinto(t *testing.T) {
	e := nuevoEntornoServicioCarga(t)
	cuarentena := e.prepararYConfirmarPrueba(t)
	e.analizador.mimeDetectado = "application/zip"
	_, err := e.servicio.AnalizarYPromover(context.Background(), OrdenAnalizarCargaDocumental{
		Principal: e.servicioAV, PerfilActivo: perfilAutorizacionPrueba("servicio:antivirus"), Recurso: e.recurso,
		CargaID: cuarentena.ID, Finalidad: cuarentena.Finalidad, Motivo: "Analisis automatico obligatorio",
		CorrelacionRef: cuarentena.CorrelacionRef,
	})
	if !errors.Is(err, ErrResultadoCargaDocumentalInvalido) || e.almacen.promociones != 0 {
		t.Fatalf("mime discordante aceptado: error=%v promociones=%d", err, e.almacen.promociones)
	}
}

func TestAnalisisDetectaBytesAdicionalesAunqueElConectorDeclareElTamanoEsperado(t *testing.T) {
	e := nuevoEntornoServicioCarga(t)
	cuarentena := e.prepararYConfirmarPrueba(t)
	e.almacen.contenidoLectura = append(append([]byte(nil), e.contenido...), byte('X'))
	_, err := e.servicio.AnalizarYPromover(context.Background(), OrdenAnalizarCargaDocumental{
		Principal: e.servicioAV, PerfilActivo: perfilAutorizacionPrueba("servicio:antivirus"), Recurso: e.recurso,
		CargaID: cuarentena.ID, Finalidad: cuarentena.Finalidad, Motivo: "Analisis automatico obligatorio",
		CorrelacionRef: cuarentena.CorrelacionRef,
	})
	if !errors.Is(err, ErrResultadoCargaDocumentalInvalido) || e.almacen.promociones != 0 {
		t.Fatalf("flujo sobredimensionado aceptado: error=%v promociones=%d", err, e.almacen.promociones)
	}
}

var _ ports.RepositorioCargasDocumentales = (*repositorioCargaPrueba)(nil)
var _ ports.AlmacenObjetos = (*almacenCargaPrueba)(nil)
var _ ports.GestorCargaDirecta = (*almacenCargaPrueba)(nil)
var _ ports.AnalizadorContenido = (*analizadorCargaPrueba)(nil)
var _ ports.SeudonimizadorSujetoAlmacen = (*seguridadCargaPrueba)(nil)
var _ ports.EmisorReciboCargaDirecta = (*seguridadCargaPrueba)(nil)
var _ ports.ConsumidorReciboCargaDirecta = (*seguridadCargaPrueba)(nil)
var _ ports.VerificadorAtestacionConsumoReciboCargaDirecta = (*seguridadCargaPrueba)(nil)
