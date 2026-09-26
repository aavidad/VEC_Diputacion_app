package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
)

const tiempoMaximoOperacionSeguimiento = 20 * time.Second

var (
	ErrServicioSeguimientoInvalido  = errors.New("contratacion temporal: servicio de cese, cierre y modificacion invalido")
	ErrSolicitudSeguimientoInvalida = errors.New("contratacion temporal: solicitud de cese, cierre o modificacion invalida")
	ErrSeguimientoDenegado          = ports.ErrAutorizacionDenegada
)

// CalculadorCosteModificacion recalcula el coste con la misma fuente que el
// análisis: grupo, periodo y jornada nuevos. No lo decide el navegador.
type CalculadorCosteModificacion interface {
	CalcularCosteModificacion(ctx context.Context, expediente domain.Expediente, periodo domain.PeriodoPrevisto, jornada domain.JornadaDiezmilesimas) (domain.Importe, string, error)
}

// ContextoCanalSeguimiento procede de la frontera autenticada, nunca del cuerpo.
type ContextoCanalSeguimiento struct {
	AutenticacionRef, SesionRef, PerfilRef, OrganizacionRef string
}

func (c ContextoCanalSeguimiento) solicitud() ports.SolicitudResolverContextoAutorizacionAltaV3 {
	return ports.SolicitudResolverContextoAutorizacionAltaV3{AutenticacionRef: c.AutenticacionRef, SesionRef: c.SesionRef, PerfilRef: c.PerfilRef}
}

func (c ContextoCanalSeguimiento) Valido() bool {
	return c.solicitud().Validar() == nil && domain.ReferenciaOpacaValida(c.OrganizacionRef)
}

type SolicitudRegistrarCese struct {
	Canal              ContextoCanalSeguimiento
	ExpedienteRef      string
	VersionEsperada    uint64
	ClaveIdempotencia  string
	CausaClave         domain.ClaveCatalogo
	FechaEfecto        time.Time
	JustificanteRef    string
	JustificanteSHA256 string
	Observaciones      string
}

type SolicitudCerrarExpediente struct {
	Canal              ContextoCanalSeguimiento
	ExpedienteRef      string
	VersionEsperada    uint64
	ClaveIdempotencia  string
	GINPIXNumero       string
	GINPIXConfirmadaEn *time.Time
	Observaciones      string
}

type SolicitudModificarTrasNombramiento struct {
	Canal             ContextoCanalSeguimiento
	ExpedienteRef     string
	VersionEsperada   uint64
	ClaveIdempotencia string
	MotivoClave       domain.ClaveCatalogo
	Periodo           domain.PeriodoPrevisto
	Jornada           domain.JornadaDiezmilesimas
	Observaciones     string
}

// ServicioOperacionesSeguimiento coordina cese, cierre y modificación. Las
// reglas inventadas (causas, justificantes, condiciones del cierre, fase de
// vuelta y motivos) proceden siempre de FuenteReglasSeguimiento.
type ServicioOperacionesSeguimiento struct {
	contextos   ports.ResolutorContextoAutorizacionAltaV3
	sellos      ports.SelladorOperacionSeguimiento
	repositorio ports.RepositorioOperacionSeguimiento
	reglas      ports.FuenteReglasSeguimiento
	autorizador ports.AutorizadorOperacionSeguimiento
	referencias ports.GeneradorReferenciasSeguimiento
	coste       CalculadorCosteModificacion
	lector      ports.LectorEstadoSeguimiento
	reloj       ports.Reloj
}

type DependenciasOperacionesSeguimiento struct {
	Contextos   ports.ResolutorContextoAutorizacionAltaV3
	Sellos      ports.SelladorOperacionSeguimiento
	Repositorio ports.RepositorioOperacionSeguimiento
	Reglas      ports.FuenteReglasSeguimiento
	Autorizador ports.AutorizadorOperacionSeguimiento
	Referencias ports.GeneradorReferenciasSeguimiento
	Coste       CalculadorCosteModificacion
	Lector      ports.LectorEstadoSeguimiento
	Reloj       ports.Reloj
}

func NuevoServicioOperacionesSeguimiento(d DependenciasOperacionesSeguimiento) (*ServicioOperacionesSeguimiento, error) {
	if dependenciaNula(d.Contextos) || dependenciaNula(d.Sellos) || dependenciaNula(d.Repositorio) || dependenciaNula(d.Reglas) ||
		dependenciaNula(d.Autorizador) || dependenciaNula(d.Referencias) || dependenciaNula(d.Coste) || dependenciaNula(d.Lector) ||
		dependenciaNula(d.Reloj) {
		return nil, ErrServicioSeguimientoInvalido
	}
	return &ServicioOperacionesSeguimiento{contextos: d.Contextos, sellos: d.Sellos, repositorio: d.Repositorio, reglas: d.Reglas,
		autorizador: d.Autorizador, referencias: d.Referencias, coste: d.Coste, lector: d.Lector, reloj: d.Reloj}, nil
}

// identidad resuelve actor y perfil del vínculo autenticado.
func (s *ServicioOperacionesSeguimiento) identidad(ctx context.Context, canal ContextoCanalSeguimiento) (string, string, error) {
	if !canal.Valido() {
		return "", "", ErrSolicitudSeguimientoInvalida
	}
	contexto, err := s.contextos.ResolverContextoAutorizacionAltaV3(ctx, canal.solicitud())
	if err != nil || contexto.ValidarPara(canal.solicitud(), instanteCanonico(s.reloj.Ahora())) != nil {
		return "", "", ErrSeguimientoDenegado
	}
	v, err := contexto.Vinculo.Datos()
	if err != nil || !domain.ReferenciaOpacaValida(v.PrincipalID) || !domain.ReferenciaOpacaValida(v.PerfilActivoRef) {
		return "", "", ErrSeguimientoDenegado
	}
	return v.PrincipalID, v.PerfilActivoRef, nil
}

// preparar sella la intención y lee el estado durable. Una confirmación
// previa de la misma intención devuelve su recibo sin otra escritura.
func (s *ServicioOperacionesSeguimiento) preparar(ctx context.Context, operacion, org, actor, perfil, clave string, material any, huella []byte) (ports.PreparacionOperacionSeguimiento, error) {
	ambitos, err := s.sellos.SellarAmbitoOperacionSeguimiento(ctx, operacion, ports.SolicitudSellarAmbitoIdempotencia{ClaveIdempotencia: clave, OrganizacionRef: org, ActorRef: actor, PerfilRef: perfil})
	if err != nil {
		return ports.PreparacionOperacionSeguimiento{}, ErrSeguimientoDenegado
	}
	huellas, err := s.sellos.DerivarHuellaOperacionSeguimiento(ctx, operacion, huella)
	if err != nil {
		return ports.PreparacionOperacionSeguimiento{}, ErrSeguimientoDenegado
	}
	refs, err := s.referencias.GenerarReferenciasSeguimiento(ctx)
	if err != nil || !refs.Validas() {
		return ports.PreparacionOperacionSeguimiento{}, ports.ErrOperacionSeguimientoNoDisponible
	}
	p, err := s.repositorio.PrepararOperacionSeguimiento(ctx, operacion, material, ports.SellosOperacionSeguimiento{Ambitos: ambitos, Huellas: huellas}, refs)
	if err != nil {
		return ports.PreparacionOperacionSeguimiento{}, err
	}
	dominioAmbito, dominioHuella, _ := ports.DominiosHMACOperacionSeguimiento(operacion)
	if !p.Referencias.Validas() || !ports.ColeccionesHMACContienenPar(ambitos, dominioAmbito, huellas, dominioHuella, p.AmbitoIdempotenciaHMAC, p.HuellaPeticionHMAC) {
		return ports.PreparacionOperacionSeguimiento{}, ports.ErrResultadoSeguimientoNoConfiable
	}
	return p, nil
}

// confirmar autoriza el recurso exacto y confirma. Una intención ya
// confirmada también exige autorización nueva antes de devolver su recibo.
func (s *ServicioOperacionesSeguimiento) confirmar(ctx context.Context, orden ports.OrdenConfirmarOperacionSeguimiento, org, exp string, version uint64) (ports.ReciboOperacionSeguimiento, error) {
	recurso := vd.RecursoAutorizable{Referencia: exp, ModuloID: ports.ModuloContratacion, Tipo: tipoRecursoSeguimiento(orden.Operacion),
		Ambitos: orden.Contexto.Ambitos, Atributos: orden.Contexto.Atributos}
	material, err := s.autorizador.AutorizarOperacionSeguimiento(ctx, ports.SolicitudAutorizarOperacionSeguimiento{
		Accion: orden.Accion, Finalidad: orden.Finalidad, Audiencia: orden.Audiencia, Motivo: orden.Politica.MotivoAutorizacion, Recurso: recurso})
	if err != nil {
		return ports.ReciboOperacionSeguimiento{}, ErrSeguimientoDenegado
	}
	if orden.Preparacion.Confirmada {
		if orden.Preparacion.Recibo == nil || !orden.Preparacion.Recibo.ValidoPara(orden.Operacion, org, exp, version) {
			return ports.ReciboOperacionSeguimiento{}, ports.ErrResultadoSeguimientoNoConfiable
		}
		return *orden.Preparacion.Recibo, nil
	}
	orden.Autorizacion = material
	recibo, err := s.repositorio.ConfirmarOperacionSeguimiento(ctx, orden)
	if err != nil {
		return ports.ReciboOperacionSeguimiento{}, err
	}
	if !recibo.ValidoPara(orden.Operacion, org, exp, version) || recibo.VersionResultante != orden.Siguiente.Version ||
		recibo.FaseResultante != orden.Siguiente.FaseActual || recibo.EstadoResultante != orden.Siguiente.EstadoActual ||
		!recibo.RegistradaEn.Equal(orden.InstanteEfecto) {
		return ports.ReciboOperacionSeguimiento{}, ports.ErrResultadoSeguimientoNoConfiable
	}
	return recibo, nil
}

func tipoRecursoSeguimiento(operacion string) string {
	switch operacion {
	case ports.OperacionRegistrarCese:
		return ports.TipoRecursoCese
	case ports.OperacionCerrarExpediente:
		return ports.TipoRecursoCierreExpediente
	case ports.OperacionCancelarExpediente:
		return ports.TipoRecursoCancelacion
	default:
		return ports.TipoRecursoModificacionNombramiento
	}
}

func huellaTexto(texto string) string {
	h := sha256.Sum256([]byte(texto))
	return hex.EncodeToString(h[:])
}

func contextoOperacionSeguimiento(ctx context.Context) (context.Context, context.CancelFunc, error) {
	if ctx == nil {
		return nil, nil, ErrSolicitudSeguimientoInvalida
	}
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	c, cancelar := context.WithTimeout(ctx, tiempoMaximoOperacionSeguimiento)
	return c, cancelar, nil
}

func ambitosSeguimiento(org, exp string) map[string]string {
	return map[string]string{"organizacion_ref": org, "expediente_ref": exp, "fase_previa": string(domain.FaseNombramiento), "estado_previo": string(domain.EstadoEnCurso)}
}

func atributosPoliticaSeguimiento(a map[string]string, p ports.PoliticaOperacionSeguimiento, prep ports.PreparacionOperacionSeguimiento, version uint64) map[string]string {
	a["version_expediente"] = strconv.FormatUint(version, 10)
	a["politica_ref"] = p.DefinicionRef
	a["politica_version"] = strconv.FormatUint(p.DefinicionVersion, 10)
	a["politica_huella_sha256"] = p.DefinicionHuellaSHA256
	a["ambito_idempotencia_hmac"] = prep.AmbitoIdempotenciaHMAC
	a["huella_peticion_hmac"] = prep.HuellaPeticionHMAC
	return a
}

// RegistrarCese valida la causa contra el catálogo vigente y deriva de él el
// tipo de justificante; la fecha y la incorporación las comprueba SQL.
func (s *ServicioOperacionesSeguimiento) RegistrarCese(ctx context.Context, sol SolicitudRegistrarCese) (ports.ReciboOperacionSeguimiento, error) {
	if s == nil {
		return ports.ReciboOperacionSeguimiento{}, ErrServicioSeguimientoInvalido
	}
	ctx, cancelar, err := contextoOperacionSeguimiento(ctx)
	if err != nil {
		return ports.ReciboOperacionSeguimiento{}, err
	}
	defer cancelar()
	actor, perfil, err := s.identidad(ctx, sol.Canal)
	if err != nil {
		return ports.ReciboOperacionSeguimiento{}, err
	}
	ahora := instanteCanonico(s.reloj.Ahora())
	causas, politica, err := s.reglas.CausasCese(ctx, ahora)
	if err != nil || !politica.ValidaEn(ahora) {
		return ports.ReciboOperacionSeguimiento{}, ports.ErrOperacionSeguimientoNoDisponible
	}
	var causa *ports.CausaCese
	for i := range causas {
		if causas[i].Clave == sol.CausaClave {
			causa = &causas[i]
		}
	}
	if causa == nil {
		return ports.ReciboOperacionSeguimiento{}, ErrSolicitudSeguimientoInvalida
	}
	datos := domain.DatosCese{CausaClave: causa.Clave, FechaEfecto: sol.FechaEfecto, JustificanteTipo: causa.JustificanteTipo,
		JustificanteRef: sol.JustificanteRef, JustificanteSHA256: sol.JustificanteSHA256, Observaciones: sol.Observaciones}
	material := ports.MaterialCese{OrganizacionRef: sol.Canal.OrganizacionRef, ExpedienteRef: sol.ExpedienteRef, ActorRef: actor, PerfilRef: perfil,
		VersionEsperada: sol.VersionEsperada, ClaveIdempotencia: sol.ClaveIdempotencia, Datos: datos}
	if !material.Valido() {
		return ports.ReciboOperacionSeguimiento{}, ErrSolicitudSeguimientoInvalida
	}
	huella, _ := json.Marshal(struct {
		Operacion, Organizacion, Expediente, Actor, Perfil, Causa, Fecha, Tipo, Justificante, Huella, Observaciones string
		Version                                                                                                     uint64
	}{ports.OperacionRegistrarCese, material.OrganizacionRef, material.ExpedienteRef, actor, perfil, string(datos.CausaClave),
		datos.FechaEfecto.Format(time.DateOnly), string(datos.JustificanteTipo), datos.JustificanteRef, datos.JustificanteSHA256, datos.Observaciones, sol.VersionEsperada})
	prep, err := s.preparar(ctx, ports.OperacionRegistrarCese, material.OrganizacionRef, actor, perfil, sol.ClaveIdempotencia, material, huella)
	if err != nil {
		return ports.ReciboOperacionSeguimiento{}, err
	}
	instante := instanteCanonico(s.reloj.Ahora())
	orden := ports.OrdenConfirmarOperacionSeguimiento{Operacion: ports.OperacionRegistrarCese, Material: material, Preparacion: prep,
		Politica: politica, InstanteEfecto: instante, Accion: domain.AccionCesarNombramiento, Finalidad: ports.FinalidadRegistrarCese,
		Audiencia: ports.AudienciaConsumoCeseV1}
	orden.Contexto = ports.ContextoAutorizadoSeguimiento{Ambitos: ambitosSeguimiento(material.OrganizacionRef, material.ExpedienteRef),
		Atributos: atributosPoliticaSeguimiento(map[string]string{"causa_clave": string(datos.CausaClave), "fecha_efecto": datos.FechaEfecto.Format(time.DateOnly),
			"justificante_tipo": string(datos.JustificanteTipo), "justificante_ref": datos.JustificanteRef, "justificante_sha256": datos.JustificanteSHA256,
			"observaciones_huella_sha256": huellaTexto(datos.Observaciones), "incorporacion_ref": prep.IncorporacionRef}, politica, prep, sol.VersionEsperada)}
	if !prep.Confirmada {
		if !domain.ReferenciaOpacaValida(prep.IncorporacionRef) || prep.Expediente.Version != sol.VersionEsperada ||
			prep.Expediente.Referencia != sol.ExpedienteRef || prep.Expediente.OrganizacionRef != material.OrganizacionRef ||
			prep.Expediente.Asignacion == nil {
			return ports.ReciboOperacionSeguimiento{}, ports.ErrResultadoSeguimientoNoConfiable
		}
		orden.Siguiente, err = prep.Expediente.RegistrarCese(sol.VersionEsperada, datos, domain.DatosActuacion{AccionClave: domain.AccionCesarNombramiento,
			ActorRef: actor, UnidadRef: prep.Expediente.Asignacion.UnidadRef, ReciboRef: prep.Referencias.ReciboRef, RealizadaEn: instante,
			FaseDestino: domain.FaseNombramiento, EstadoDestino: domain.EstadoEnCurso, Observaciones: datos.Observaciones, DocumentosRef: []string{datos.JustificanteRef}})
		if err != nil {
			return ports.ReciboOperacionSeguimiento{}, err
		}
	}
	return s.confirmar(ctx, orden, material.OrganizacionRef, material.ExpedienteRef, sol.VersionEsperada)
}

// CerrarExpediente aplica las condiciones de la regla vigente: si exige
// GINPIX, pide número y fecha; el cese lo comprueban dominio y SQL.
func (s *ServicioOperacionesSeguimiento) CerrarExpediente(ctx context.Context, sol SolicitudCerrarExpediente) (ports.ReciboOperacionSeguimiento, error) {
	if s == nil {
		return ports.ReciboOperacionSeguimiento{}, ErrServicioSeguimientoInvalido
	}
	ctx, cancelar, err := contextoOperacionSeguimiento(ctx)
	if err != nil {
		return ports.ReciboOperacionSeguimiento{}, err
	}
	defer cancelar()
	actor, perfil, err := s.identidad(ctx, sol.Canal)
	if err != nil {
		return ports.ReciboOperacionSeguimiento{}, err
	}
	ahora := instanteCanonico(s.reloj.Ahora())
	regla, politica, err := s.reglas.ReglaCierre(ctx, ahora)
	if err != nil || !politica.ValidaEn(ahora) || !domain.CondicionesCierreValidas(regla.Condiciones) {
		return ports.ReciboOperacionSeguimiento{}, ports.ErrOperacionSeguimientoNoDisponible
	}
	datos := domain.DatosCierreExpediente{Condiciones: append([]string(nil), regla.Condiciones...), GINPIXNumero: sol.GINPIXNumero,
		GINPIXConfirmadaEn: sol.GINPIXConfirmadaEn, Observaciones: sol.Observaciones}
	material := ports.MaterialCierreExpediente{OrganizacionRef: sol.Canal.OrganizacionRef, ExpedienteRef: sol.ExpedienteRef, ActorRef: actor, PerfilRef: perfil,
		VersionEsperada: sol.VersionEsperada, ClaveIdempotencia: sol.ClaveIdempotencia, Datos: datos}
	if !material.Valido() {
		return ports.ReciboOperacionSeguimiento{}, ErrSolicitudSeguimientoInvalida
	}
	fecha := ""
	if datos.GINPIXConfirmadaEn != nil {
		fecha = datos.GINPIXConfirmadaEn.Format(time.DateOnly)
	}
	huella, _ := json.Marshal(struct {
		Operacion, Organizacion, Expediente, Actor, Perfil, Condiciones, GINPIX, Fecha, Observaciones string
		Version                                                                                       uint64
	}{ports.OperacionCerrarExpediente, material.OrganizacionRef, material.ExpedienteRef, actor, perfil, strings.Join(datos.Condiciones, ","),
		datos.GINPIXNumero, fecha, datos.Observaciones, sol.VersionEsperada})
	prep, err := s.preparar(ctx, ports.OperacionCerrarExpediente, material.OrganizacionRef, actor, perfil, sol.ClaveIdempotencia, material, huella)
	if err != nil {
		return ports.ReciboOperacionSeguimiento{}, err
	}
	instante := instanteCanonico(s.reloj.Ahora())
	orden := ports.OrdenConfirmarOperacionSeguimiento{Operacion: ports.OperacionCerrarExpediente, Material: material, Preparacion: prep,
		Politica: politica, InstanteEfecto: instante, Accion: domain.AccionCerrarExpediente, Finalidad: ports.FinalidadCerrarExpediente,
		Audiencia: ports.AudienciaConsumoCierreExpedienteV1}
	orden.Contexto = ports.ContextoAutorizadoSeguimiento{Ambitos: ambitosSeguimiento(material.OrganizacionRef, material.ExpedienteRef),
		Atributos: atributosPoliticaSeguimiento(map[string]string{"cese_recibo_ref": prep.CeseReciboRef, "condiciones": strings.Join(datos.Condiciones, ","),
			"ginpix_numero": datos.GINPIXNumero, "ginpix_confirmada_en": fecha, "observaciones_huella_sha256": huellaTexto(datos.Observaciones)},
			politica, prep, sol.VersionEsperada)}
	if !prep.Confirmada {
		if !domain.ReferenciaOpacaValida(prep.CeseReciboRef) || prep.Expediente.Version != sol.VersionEsperada ||
			prep.Expediente.Referencia != sol.ExpedienteRef || prep.Expediente.OrganizacionRef != material.OrganizacionRef ||
			prep.Expediente.Asignacion == nil {
			return ports.ReciboOperacionSeguimiento{}, ports.ErrResultadoSeguimientoNoConfiable
		}
		var documentos []string
		if ref := datos.DocumentoGINPIX(); ref != "" {
			documentos = []string{ref}
		}
		orden.Siguiente, err = prep.Expediente.CerrarTrasCese(sol.VersionEsperada, datos, domain.DatosActuacion{AccionClave: domain.AccionCerrarExpediente,
			ActorRef: actor, UnidadRef: prep.Expediente.Asignacion.UnidadRef, ReciboRef: prep.Referencias.ReciboRef, RealizadaEn: instante,
			FaseDestino: domain.FaseNombramiento, EstadoDestino: domain.EstadoCompletado, Observaciones: datos.Observaciones, DocumentosRef: documentos})
		if err != nil {
			return ports.ReciboOperacionSeguimiento{}, err
		}
	}
	return s.confirmar(ctx, orden, material.OrganizacionRef, material.ExpedienteRef, sol.VersionEsperada)
}

// ModificarTrasNombramiento recalcula el coste con la fuente del análisis y
// devuelve el expediente a la fase de la regla c09. La intención es motivo,
// periodo, jornada y observaciones; el coste se deriva del expediente leído y
// no forma parte de la huella de idempotencia. Un coste por encima de la
// retención de crédito se rechaza: hace falta otra retención.
func (s *ServicioOperacionesSeguimiento) ModificarTrasNombramiento(ctx context.Context, sol SolicitudModificarTrasNombramiento) (ports.ReciboOperacionSeguimiento, error) {
	if s == nil {
		return ports.ReciboOperacionSeguimiento{}, ErrServicioSeguimientoInvalido
	}
	ctx, cancelar, err := contextoOperacionSeguimiento(ctx)
	if err != nil {
		return ports.ReciboOperacionSeguimiento{}, err
	}
	defer cancelar()
	actor, perfil, err := s.identidad(ctx, sol.Canal)
	if err != nil {
		return ports.ReciboOperacionSeguimiento{}, err
	}
	ahora := instanteCanonico(s.reloj.Ahora())
	regla, politica, err := s.reglas.ReglaModificacion(ctx, ahora)
	if err != nil || !politica.ValidaEn(ahora) {
		return ports.ReciboOperacionSeguimiento{}, ports.ErrOperacionSeguimientoNoDisponible
	}
	admitido := false
	for _, m := range regla.Motivos {
		admitido = admitido || m.Clave == sol.MotivoClave
	}
	if !admitido {
		return ports.ReciboOperacionSeguimiento{}, ErrSolicitudSeguimientoInvalida
	}
	// Coste provisional solo para leer: SQL no lo compara en la preparación.
	datos := domain.DatosModificacionTrasNombramiento{MotivoClave: sol.MotivoClave, Periodo: sol.Periodo, Jornada: sol.Jornada,
		Coste: domain.Importe{Moneda: "EUR", Centimos: 1}, FuenteCoste: "fuente:coste:pendiente", FaseRetorno: regla.FaseRetorno,
		Observaciones: sol.Observaciones}
	material := ports.MaterialModificacionNombramiento{OrganizacionRef: sol.Canal.OrganizacionRef, ExpedienteRef: sol.ExpedienteRef, ActorRef: actor,
		PerfilRef: perfil, VersionEsperada: sol.VersionEsperada, ClaveIdempotencia: sol.ClaveIdempotencia, Datos: datos}
	if !material.Valido() {
		return ports.ReciboOperacionSeguimiento{}, ErrSolicitudSeguimientoInvalida
	}
	huella, _ := json.Marshal(struct {
		Operacion, Organizacion, Expediente, Actor, Perfil, Motivo, Inicio, Fin, Fase, Observaciones string
		Version                                                                                      uint64
		Jornada                                                                                      uint16
	}{ports.OperacionModificarTrasNombramiento, material.OrganizacionRef, material.ExpedienteRef, actor, perfil, string(datos.MotivoClave),
		datos.Periodo.Inicio.Format(time.DateOnly), datos.Periodo.Fin.Format(time.DateOnly), string(datos.FaseRetorno),
		datos.Observaciones, sol.VersionEsperada, uint16(datos.Jornada)})
	prep, err := s.preparar(ctx, ports.OperacionModificarTrasNombramiento, material.OrganizacionRef, actor, perfil, sol.ClaveIdempotencia, material, huella)
	if err != nil {
		return ports.ReciboOperacionSeguimiento{}, err
	}
	if prep.Confirmada {
		if prep.Recibo == nil {
			return ports.ReciboOperacionSeguimiento{}, ports.ErrResultadoSeguimientoNoConfiable
		}
		// La autorización de la repetición liga el coste original del recibo.
		material.Datos.Coste.Centimos = prep.Recibo.CosteCentimos
		material.Datos.FuenteCoste = prep.FuenteCosteRef
	} else {
		if prep.Expediente.Version != sol.VersionEsperada || prep.Expediente.Referencia != sol.ExpedienteRef ||
			prep.Expediente.OrganizacionRef != material.OrganizacionRef || prep.Expediente.Asignacion == nil {
			return ports.ReciboOperacionSeguimiento{}, ports.ErrResultadoSeguimientoNoConfiable
		}
		coste, fuente, errCoste := s.coste.CalcularCosteModificacion(ctx, prep.Expediente, sol.Periodo, sol.Jornada)
		if errCoste != nil {
			return ports.ReciboOperacionSeguimiento{}, ports.ErrOperacionSeguimientoNoDisponible
		}
		material.Datos.Coste, material.Datos.FuenteCoste = coste, fuente
		if !material.Valido() {
			return ports.ReciboOperacionSeguimiento{}, ports.ErrOperacionSeguimientoNoDisponible
		}
		if a := prep.Expediente.Analisis; a != nil && a.ValidacionRC.Importe != nil && a.ValidacionRC.Resultado == domain.RCValidada &&
			coste.Centimos > a.ValidacionRC.Importe.Centimos {
			return ports.ReciboOperacionSeguimiento{}, ports.ErrModificacionCreditoInsuficiente
		}
	}
	datos = material.Datos
	instante := instanteCanonico(s.reloj.Ahora())
	orden := ports.OrdenConfirmarOperacionSeguimiento{Operacion: ports.OperacionModificarTrasNombramiento, Material: material, Preparacion: prep,
		Politica: politica, InstanteEfecto: instante, Accion: domain.AccionModificarTrasNombramiento, Finalidad: ports.FinalidadModificarTrasNombramiento,
		Audiencia: ports.AudienciaConsumoModificacionNombramientoV1}
	orden.Contexto = ports.ContextoAutorizadoSeguimiento{Ambitos: ambitosSeguimiento(material.OrganizacionRef, material.ExpedienteRef),
		Atributos: atributosPoliticaSeguimiento(map[string]string{"motivo_clave": string(datos.MotivoClave),
			"periodo_inicio": datos.Periodo.Inicio.Format(time.DateOnly), "periodo_fin": datos.Periodo.Fin.Format(time.DateOnly),
			"porcentaje_jornada": strconv.FormatUint(uint64(datos.Jornada), 10), "coste_centimos": strconv.FormatInt(datos.Coste.Centimos, 10),
			"fuente_coste_ref": datos.FuenteCoste, "fase_retorno": string(datos.FaseRetorno), "estado_retorno": string(domain.EstadoEnCurso),
			"observaciones_huella_sha256": huellaTexto(datos.Observaciones)}, politica, prep, sol.VersionEsperada)}
	if !prep.Confirmada {
		orden.Siguiente, err = prep.Expediente.ModificarTrasNombramiento(sol.VersionEsperada, datos, domain.DatosActuacion{
			AccionClave: domain.AccionModificarTrasNombramiento, ActorRef: actor, UnidadRef: prep.Expediente.Asignacion.UnidadRef,
			ReciboRef: prep.Referencias.ReciboRef, RealizadaEn: instante, FaseDestino: datos.FaseRetorno, EstadoDestino: domain.EstadoEnCurso,
			Observaciones: datos.Observaciones})
		if err != nil {
			return ports.ReciboOperacionSeguimiento{}, err
		}
	}
	return s.confirmar(ctx, orden, material.OrganizacionRef, material.ExpedienteRef, sol.VersionEsperada)
}

// Opciones publica las opciones vigentes de los catálogos para la pantalla.
func (s *ServicioOperacionesSeguimiento) Opciones(ctx context.Context) (ports.OpcionesSeguimiento, error) {
	if s == nil || ctx == nil {
		return ports.OpcionesSeguimiento{}, ErrServicioSeguimientoInvalido
	}
	ahora := instanteCanonico(s.reloj.Ahora())
	causas, _, err := s.reglas.CausasCese(ctx, ahora)
	if err != nil {
		return ports.OpcionesSeguimiento{}, ports.ErrOperacionSeguimientoNoDisponible
	}
	cierre, _, err := s.reglas.ReglaCierre(ctx, ahora)
	if err != nil {
		return ports.OpcionesSeguimiento{}, ports.ErrOperacionSeguimientoNoDisponible
	}
	modificacion, _, err := s.reglas.ReglaModificacion(ctx, ahora)
	if err != nil {
		return ports.OpcionesSeguimiento{}, ports.ErrOperacionSeguimientoNoDisponible
	}
	o := ports.OpcionesSeguimiento{Causas: causas, Condiciones: cierre.Condiciones, FaseRetorno: modificacion.FaseRetorno, Motivos: modificacion.Motivos}
	if !o.Validas() {
		return ports.OpcionesSeguimiento{}, ports.ErrOperacionSeguimientoNoDisponible
	}
	return o, nil
}

// Estado devuelve cese y cierre registrados. La composición solo lo invoca
// tras acreditar la lectura del detalle del mismo expediente.
func (s *ServicioOperacionesSeguimiento) Estado(ctx context.Context, organizacionRef, expedienteRef string) (ports.EstadoSeguimientoExpediente, error) {
	if s == nil || ctx == nil || !domain.ReferenciaOpacaValida(organizacionRef) || !domain.ReferenciaOpacaValida(expedienteRef) {
		return ports.EstadoSeguimientoExpediente{}, ErrSolicitudSeguimientoInvalida
	}
	return s.lector.ConsultarEstadoSeguimiento(ctx, organizacionRef, expedienteRef)
}
