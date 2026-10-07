package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"reflect"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// Firma de prueba de los borradores según el circuito del catálogo. El
// servicio nunca da por buena una firma sin verificación positiva: con el
// validador apagado o sin dictamen «válida/verificada» la firma se rechaza y
// no se registra nada. Una firma registrada sigue sin eficacia administrativa
// hasta el portafirmas corporativo.

var (
	// ErrVerificacionFirmaApagada: la verificación no está compuesta.
	ErrVerificacionFirmaApagada = errors.New("contratacion temporal: verificacion de firma apagada")
	// ErrFirmaNoVerificada: el validador no acredita la firma.
	ErrFirmaNoVerificada = errors.New("contratacion temporal: firma no verificada")
	// ErrPasoFirmaNoPendiente: el paso indicado no es el que espera firma.
	ErrPasoFirmaNoPendiente = errors.New("contratacion temporal: el paso no espera firma")
	// ErrCircuitoFirmaNoDisponible: no hay catálogo de circuito o no es coherente.
	ErrCircuitoFirmaNoDisponible = errors.New("contratacion temporal: circuito de firma no disponible")
)

// SolicitudFirmaDocumento llega del canal. Original es el borrador tal como
// se descargó y Firmado el resultado de AutoFirma; en una devolución ambos
// van vacíos y solo cuenta el motivo.
type SolicitudFirmaDocumento struct {
	OrganizacionRef   string
	ExpedienteRef     string
	VersionExpediente uint64
	Documento         string
	PasoOrden         int
	Resultado         domain.ResultadoFirmaDocumento
	MotivoDevolucion  string
	Original          []byte
	// OriginalRef/OriginalVersion identifican el original que devuelve una
	// fuente autorizada al componer la vía segura. Original queda solo como
	// compatibilidad del ejercicio anterior.
	OriginalRef       string
	OriginalVersion   uint64
	Firmado           []byte
	ClaveIdempotencia string
}

// ResultadoFirmaDocumento acompaña el recibo con el dictamen que se registró.
type ResultadoFirmaDocumento struct {
	Recibo             ports.ReciboFirmaDocumento
	Material           ports.MaterialFirmaDocumento
	MotivoVerificacion docports.MotivoVerificacionFirma
	// Custodiado es el PDF firmado que guardó Documentos (vacío si el
	// documento no se custodia).
	Custodiado ports.DocumentoCustodiado
}

// DictamenRechazado describe por qué el validador no acreditó la firma.
type DictamenRechazado struct {
	Estado docports.EstadoVerificacionFirma
	Motivo docports.MotivoVerificacionFirma
}

func (d DictamenRechazado) Error() string {
	return ErrFirmaNoVerificada.Error() + ": " + string(d.Motivo)
}
func (d DictamenRechazado) Unwrap() error { return ErrFirmaNoVerificada }

// EstadoFirmasExpediente es el circuito de cada documento con su estado real.
type EstadoFirmasExpediente struct {
	Circuito   domain.CircuitoFirma
	Documentos []domain.EstadoCircuitoDocumento
	Firmas     []ports.FirmaRegistrada
}

// ServicioFirmaDocumento coordina circuito, verificación, autorización y
// registro. Verificador nulo significa verificación apagada.
type ServicioFirmaDocumento struct {
	circuito    ports.FuenteCircuitoFirma
	registro    ports.RegistroFirmasDocumento
	autorizador ports.AutorizadorFirmaDocumento
	verificador docports.VerificadorFirmaMotivado
	original    ports.FuenteOriginalFirmaAutorizado
	// rondaPolitica y rondaFuente son nil salvo con catálogo que exige
	// informe nuevo tras subsanar: entonces el documento que declara vuelve a
	// firmarse desde el paso 1 con el informe nuevo.
	rondaPolitica ports.FuenteInformeTrasSubsanacion
	rondaFuente   ports.FuenteRondaFirmaInforme
	// custodio y tiposCustodia son nil salvo con Documentos compuesto: el
	// PDF firmado de cada documento listado se custodia antes de registrar la
	// firma, con el tipo documental del catálogo de conservación indicado.
	custodio      ports.CustodioDocumentoFirmado
	tiposCustodia map[string]string
}

// ComponerOriginalAutorizado habilita la obtención del original por referencia
// y versión. Sin esta fuente, Firmar conserva únicamente el ejercicio legado;
// FirmarConOriginalAutorizado y la vía externa permanecen cerrados.
func (s *ServicioFirmaDocumento) ComponerOriginalAutorizado(f ports.FuenteOriginalFirmaAutorizado) error {
	if s == nil || nula(f) || s.original != nil {
		return ports.ErrFuenteOriginalFirmaNoDisponible
	}
	s.original = f
	return nil
}

// FirmarConOriginalAutorizado conserva CT118 para pruebas con original de
// fuente confiable. CT118 no incluye ref/version del original en su canon y
// esta entrada no acredita un hito R5; ese avance usa ServicioFirmaVec.
func (s *ServicioFirmaDocumento) FirmarConOriginalAutorizado(ctx context.Context, sol SolicitudFirmaDocumento) (ResultadoFirmaDocumento, error) {
	if s == nil || s.original == nil {
		return ResultadoFirmaDocumento{}, ports.ErrFuenteOriginalFirmaNoDisponible
	}
	return s.Firmar(ctx, sol)
}

// ComponerCustodia hace que el PDF firmado de los documentos indicados
// (clave del circuito → tipo documental de conservación) se custodie en
// Documentos antes de registrar la firma. Se fija una sola vez.
func (s *ServicioFirmaDocumento) ComponerCustodia(c ports.CustodioDocumentoFirmado, tipos map[string]string) error {
	if s == nil || nula(c) || len(tipos) == 0 || len(tipos) > 64 || s.custodio != nil {
		return ports.ErrCustodiaFirmadoNoDisponible
	}
	copia := make(map[string]string, len(tipos))
	for documento, tipo := range tipos {
		if !domain.ClaveDocumentoFirmaValida(documento) || tipo == "" || len(tipo) > 128 {
			return ports.ErrCustodiaFirmadoNoDisponible
		}
		copia[documento] = tipo
	}
	s.custodio, s.tiposCustodia = c, copia
	return nil
}

// AbrirRondaInformeNuevo compone la segunda ronda de firma del documento que
// el catálogo liga al informe nuevo tras subsanar. Se fija una sola vez.
func (s *ServicioFirmaDocumento) AbrirRondaInformeNuevo(p ports.FuenteInformeTrasSubsanacion, f ports.FuenteRondaFirmaInforme) error {
	if s == nil || nula(p) || nula(f) || s.rondaPolitica != nil {
		return ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	s.rondaPolitica, s.rondaFuente = p, f
	return nil
}

// inicioRonda devuelve, para cada documento, la versión del expediente en la
// que empieza su ronda vigente (sin entrada: ronda única).
func (s *ServicioFirmaDocumento) inicioRonda(ctx context.Context, organizacionRef, expedienteRef string) (map[string]uint64, error) {
	if s.rondaPolitica == nil {
		return nil, nil
	}
	politica, err := s.rondaPolitica.InformeTrasSubsanacion(ctx)
	if err != nil || politica.Validar() != nil {
		return nil, ErrCircuitoFirmaNoDisponible
	}
	if !politica.ExigeInformeNuevo || politica.DocumentoFirma == "" {
		return nil, nil
	}
	inicio, err := s.rondaFuente.InicioRondaInformeNuevo(ctx, organizacionRef, expedienteRef)
	if err != nil {
		return nil, err
	}
	return map[string]uint64{politica.DocumentoFirma: inicio}, nil
}

func nula(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Func, reflect.Interface, reflect.Chan:
		return r.IsNil()
	}
	return false
}

// NuevoServicioFirmaDocumento exige circuito, registro y autorizador. El
// verificador puede faltar: entonces toda firma se rechaza con
// ErrVerificacionFirmaApagada y las devoluciones siguen disponibles.
func NuevoServicioFirmaDocumento(c ports.FuenteCircuitoFirma, r ports.RegistroFirmasDocumento, a ports.AutorizadorFirmaDocumento, v docports.VerificadorFirmaMotivado) (*ServicioFirmaDocumento, error) {
	if nula(c) || nula(r) || nula(a) {
		return nil, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	if nula(v) {
		v = nil
	}
	return &ServicioFirmaDocumento{circuito: c, registro: r, autorizador: a, verificador: v}, nil
}

// VerificacionDisponible indica si el validador está compuesto.
func (s *ServicioFirmaDocumento) VerificacionDisponible() bool {
	return s != nil && s.verificador != nil
}

func huella(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// Consultar calcula el estado real de cada documento del circuito desde la
// historia registrada del expediente.
func (s *ServicioFirmaDocumento) Consultar(ctx context.Context, organizacionRef, expedienteRef string) (EstadoFirmasExpediente, error) {
	if s == nil || ctx == nil || !domain.ReferenciaOpacaValida(organizacionRef) || !domain.ReferenciaOpacaValida(expedienteRef) {
		return EstadoFirmasExpediente{}, ports.ErrSolicitudFirmaDocumentoInvalida
	}
	circuito, err := s.circuitoValido(ctx)
	if err != nil {
		return EstadoFirmasExpediente{}, err
	}
	firmas, err := s.registro.ConsultarFirmas(ctx, organizacionRef, expedienteRef)
	if err != nil {
		return EstadoFirmasExpediente{}, err
	}
	rondas, err := s.inicioRonda(ctx, organizacionRef, expedienteRef)
	if err != nil {
		return EstadoFirmasExpediente{}, err
	}
	estado := EstadoFirmasExpediente{Circuito: circuito, Firmas: firmas}
	for _, d := range circuito.Documentos {
		calculado, err := domain.CalcularEstadoCircuitoFirmaEnRonda(d, circuito.HuellaCatalogo, eventosDocumento(firmas, d.Documento), rondas[d.Documento])
		if err != nil {
			return EstadoFirmasExpediente{}, err
		}
		estado.Documentos = append(estado.Documentos, calculado)
	}
	return estado, nil
}

func (s *ServicioFirmaDocumento) circuitoValido(ctx context.Context) (domain.CircuitoFirma, error) {
	circuito, err := s.circuito.CircuitoFirma(ctx)
	if err != nil || !domain.ReferenciaOpacaValida(circuito.CatalogoRef) || !domain.HuellaSHA256FirmaValida(circuito.HuellaCatalogo) || len(circuito.Documentos) == 0 {
		return domain.CircuitoFirma{}, ErrCircuitoFirmaNoDisponible
	}
	for _, d := range circuito.Documentos {
		if d.Validar() != nil {
			return domain.CircuitoFirma{}, ErrCircuitoFirmaNoDisponible
		}
	}
	return circuito, nil
}

func eventosDocumento(firmas []ports.FirmaRegistrada, documento string) []domain.EventoFirmaDocumento {
	var eventos []domain.EventoFirmaDocumento
	for _, f := range firmas {
		if f.Documento != documento {
			continue
		}
		eventos = append(eventos, domain.EventoFirmaDocumento{
			Secuencia: f.Secuencia, CatalogoHuella: f.CatalogoHuella, PasoOrden: f.PasoOrden, Resultado: f.Resultado,
			ConMotivoDevolucion: f.ConMotivoDevolucion, OriginalHuella: f.OriginalHuella, FirmadoHuella: f.FirmadoHuella,
			ReciboRef: f.ReciboRef, RegistradaEn: f.RegistradaEn, ExpedienteVersion: f.ExpedienteVersion,
		})
	}
	return eventos
}

// identificadorDocumentoVerificacion es la referencia opaca con la que el
// puerto de verificación identifica el borrador (no se custodia).
func identificadorDocumentoVerificacion(expediente, documento string) string {
	return "ref:" + huella([]byte("vec.contratacion-temporal.borrador-firma.v1\n"+expediente+"\n"+documento+"\n"))
}

// Firmar comprueba que el paso es el pendiente, verifica la firma (si es una
// firma), obtiene la autorización ligada al material exacto y registra.
func (s *ServicioFirmaDocumento) Firmar(ctx context.Context, sol SolicitudFirmaDocumento) (ResultadoFirmaDocumento, error) {
	var cero ResultadoFirmaDocumento
	if s == nil || ctx == nil || !domain.ReferenciaOpacaValida(sol.OrganizacionRef) || !domain.ReferenciaOpacaValida(sol.ExpedienteRef) ||
		sol.VersionExpediente == 0 || !domain.ClaveDocumentoFirmaValida(sol.Documento) || sol.PasoOrden < 1 ||
		!ports.ClaveIdempotenciaFirmaValida(sol.ClaveIdempotencia) {
		return cero, ports.ErrSolicitudFirmaDocumentoInvalida
	}
	if sol.Resultado == domain.ResultadoFirmaFirmado && s.original != nil {
		original, err := obtenerOriginalFirmaAutorizado(ctx, s.original, ports.SolicitudOriginalFirma{
			OrganizacionRef: sol.OrganizacionRef, ExpedienteRef: sol.ExpedienteRef,
			Documento: sol.Documento, OriginalRef: sol.OriginalRef, OriginalVersion: sol.OriginalVersion,
		})
		if err != nil {
			return cero, err
		}
		if len(sol.Original) != 0 && !bytes.Equal(sol.Original, original.Contenido) {
			return cero, ports.ErrOriginalFirmaNoAutorizado
		}
		sol.Original = original.Contenido
	}
	switch sol.Resultado {
	case domain.ResultadoFirmaFirmado:
		if len(sol.Original) == 0 || len(sol.Firmado) == 0 || len(sol.Original) > ports.MaximoDocumentoFirmaBytes ||
			len(sol.Firmado) > ports.MaximoDocumentoFirmaBytes || sol.MotivoDevolucion != "" || bytes.Equal(sol.Original, sol.Firmado) {
			return cero, ports.ErrSolicitudFirmaDocumentoInvalida
		}
		if s.verificador == nil {
			return cero, ErrVerificacionFirmaApagada
		}
	case domain.ResultadoFirmaDevuelto:
		if len(sol.Original) != 0 || len(sol.Firmado) != 0 || !ports.MotivoDevolucionFirmaValido(sol.MotivoDevolucion) {
			return cero, ports.ErrSolicitudFirmaDocumentoInvalida
		}
	default:
		return cero, ports.ErrSolicitudFirmaDocumentoInvalida
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	estado, err := s.Consultar(ctx, sol.OrganizacionRef, sol.ExpedienteRef)
	if err != nil {
		return cero, err
	}
	circuitoDoc, existe := estado.Circuito.Documento(sol.Documento)
	if !existe {
		return cero, ports.ErrSolicitudFirmaDocumentoInvalida
	}
	var actual domain.EstadoCircuitoDocumento
	for _, d := range estado.Documentos {
		if d.Documento == sol.Documento {
			actual = d
		}
	}
	secuencia, originalEsperado := actual.UltimaSecuencia+1, actual.OriginalEsperadoHuella
	previa, repetida := firmaConClave(estado.Firmas, sol.Documento, sol.ClaveIdempotencia)
	switch {
	case repetida:
		// Reintento de una firma ya registrada (respuesta perdida): el paso ya
		// no está pendiente, pero se rehace el mismo material para que el
		// registro devuelva el recibo original. Si no puede ser el mismo, la
		// clave se está usando para otra cosa.
		if previa.PasoOrden != sol.PasoOrden || previa.Resultado != sol.Resultado ||
			previa.CatalogoHuella != estado.Circuito.HuellaCatalogo || previa.ExpedienteVersion != sol.VersionExpediente ||
			sol.PasoOrden > len(circuitoDoc.Pasos) {
			return cero, ports.ErrClaveFirmaDocumentoUsada
		}
		secuencia, originalEsperado = previa.Secuencia, previa.OriginalHuella
	case actual.Completo || actual.PasoPendiente != sol.PasoOrden:
		return cero, ErrPasoFirmaNoPendiente
	}
	paso := circuitoDoc.Pasos[sol.PasoOrden-1]
	material := ports.MaterialFirmaDocumento{
		OrganizacionRef: sol.OrganizacionRef, ExpedienteRef: sol.ExpedienteRef, VersionExpediente: sol.VersionExpediente,
		Documento: sol.Documento, CatalogoRef: estado.Circuito.CatalogoRef, CatalogoHuella: estado.Circuito.HuellaCatalogo,
		PasoRef: paso.Referencia, PasoOrden: paso.Orden, Secuencia: secuencia, Resultado: sol.Resultado,
		MotivoDevolucion: sol.MotivoDevolucion, ClaveIdempotencia: sol.ClaveIdempotencia,
	}
	var motivo docports.MotivoVerificacionFirma
	if sol.Resultado == domain.ResultadoFirmaFirmado {
		original := huella(sol.Original)
		// A partir del paso 2 se firma exactamente el borrador que firmó el
		// paso anterior: todos los pasos firman el mismo documento.
		if originalEsperado != "" && original != originalEsperado {
			return cero, ports.ErrCadenaFirmaDocumentoRota
		}
		documentoID, versionOriginal := identificadorDocumentoVerificacion(sol.ExpedienteRef, sol.Documento), sol.VersionExpediente
		if s.original != nil {
			documentoID, versionOriginal = sol.OriginalRef, sol.OriginalVersion
		}
		peticion := docports.SolicitudVerificacionFirma{
			DocumentoID: documentoID, Version: versionOriginal,
			FormatoEsperado:      "PAdES",
			HuellaOriginalSHA256: original, ContenidoOriginal: sol.Original, ContenidoFirmado: sol.Firmado,
		}
		dictamen, err := s.verificador.VerificarMotivado(ctx, peticion)
		if err != nil {
			if ctx.Err() != nil {
				return cero, ctx.Err()
			}
			return cero, DictamenRechazado{Estado: docports.EstadoVerificacionIndeterminada, Motivo: docports.MotivoRespuestaNoInterpretable}
		}
		// Solo «válida/verificada» y ligada a ambos contenidos acredita.
		if dictamen.ValidarContra(peticion) != nil || dictamen.Motivo != docports.MotivoFirmaVerificada {
			m := dictamen.Motivo
			if m.EstadoAsociado() == "" || m == docports.MotivoFirmaVerificada {
				m = docports.MotivoRespuestaNoInterpretable
			}
			return cero, DictamenRechazado{Estado: dictamen.Resultado.Estado, Motivo: m}
		}
		r := dictamen.Resultado
		material.OriginalHuella, material.FirmadoHuella = original, r.HuellaFirmadoSHA256
		material.CertificadoHuella, material.FirmanteRef = r.CertificadoHuellaSHA256, r.FirmanteRef
		material.PoliticaVerificacion = ports.PoliticaVerificacionFirma
		material.RevocacionEstado, material.SelloTiempoEstado = r.RevocacionEstado, r.SelloTiempoEstado
		motivo = dictamen.Motivo
	}
	var custodiado ports.DocumentoCustodiado
	tipo, custodiar := s.tiposCustodia[sol.Documento]
	if custodiar && sol.Resultado == domain.ResultadoFirmaFirmado {
		material.DocumentoCustodiaRef = ports.DocumentoCustodiaRef(sol.OrganizacionRef, sol.ExpedienteRef, sol.ClaveIdempotencia)
		material.DocumentoCustodiaVersion = ports.VersionDocumentoCustodiado
	}
	if material.Validar() != nil {
		return cero, ports.ErrSolicitudFirmaDocumentoInvalida
	}
	// Autorizar el material completo antes de custodiar evita guardar un PDF
	// firmado por un certificado ajeno al canal. El registro consume la misma
	// capacidad; si caduca durante la custodia, falla cerrado y un reintento
	// recupera el documento con la misma clave y los mismos bytes.
	capacidad, err := s.autorizador.AutorizarFirmaDocumento(ctx, material)
	if err != nil {
		if ctx.Err() != nil {
			return cero, ctx.Err()
		}
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	if err := ValidarCapacidadFirmaDocumento(capacidad, material); err != nil {
		return cero, err
	}
	if material.DocumentoCustodiaRef != "" {
		custodiado, err = s.custodiarFirmado(ctx, sol, material, tipo)
		if err != nil {
			return cero, err
		}
	}
	recibo, err := s.registro.RegistrarFirma(ctx, material, capacidad)
	if err != nil {
		return cero, err
	}
	h, _ := material.HuellaSHA256()
	if recibo.SolicitudHuella != h || recibo.Resultado != material.Resultado ||
		recibo.DocumentoCustodiaRef != material.DocumentoCustodiaRef ||
		recibo.DocumentoCustodiaVersion != material.DocumentoCustodiaVersion ||
		(!recibo.YaRegistrada && (recibo.Secuencia != material.Secuencia || recibo.ExpedienteVersion != material.VersionExpediente)) {
		return cero, ports.ErrResultadoFirmaDocumentoInvalido
	}
	return ResultadoFirmaDocumento{Recibo: recibo, Material: material, MotivoVerificacion: motivo, Custodiado: custodiado}, nil
}

// firmaConClave busca en la historia la firma del documento registrada con
// esta clave de idempotencia.
func firmaConClave(firmas []ports.FirmaRegistrada, documento, clave string) (ports.FirmaRegistrada, bool) {
	for _, f := range firmas {
		if f.Documento == documento && f.ClaveIdempotencia == clave {
			return f, true
		}
	}
	return ports.FirmaRegistrada{}, false
}

// custodiarFirmado entrega a Documentos el PDF firmado ya verificado y
// comprueba que lo custodiado es exactamente ese PDF.
func (s *ServicioFirmaDocumento) custodiarFirmado(ctx context.Context, sol SolicitudFirmaDocumento, m ports.MaterialFirmaDocumento, tipo string) (ports.DocumentoCustodiado, error) {
	var cero ports.DocumentoCustodiado
	if s.custodio == nil || huella(sol.Firmado) != m.FirmadoHuella {
		return cero, ports.ErrCustodiaFirmadoNoDisponible
	}
	orden := ports.OrdenCustodiaFirmado{
		DocumentoRef:      m.DocumentoCustodiaRef,
		ClaveIdempotencia: ports.ClaveCustodiaRef(m.OrganizacionRef, m.ExpedienteRef, m.ClaveIdempotencia),
		ExpedienteRef:     ports.ExpedienteDocumentalRef(m.OrganizacionRef, m.ExpedienteRef),
		TipoDocumental:    tipo, Version: m.DocumentoCustodiaVersion,
		Contenido: sol.Firmado, HuellaOriginalSHA256: m.OriginalHuella,
		FirmaOperacionRef: ports.OperacionFirmaRef(m.OrganizacionRef, m.ExpedienteRef, m.ClaveIdempotencia),
	}
	d, err := s.custodio.CustodiarFirmado(ctx, orden)
	if err != nil {
		if ctx.Err() != nil {
			return cero, ctx.Err()
		}
		for _, centinela := range []error{ports.ErrCustodiaFirmadoNoDisponible, ports.ErrCustodiaFirmadoInvalida, ports.ErrCustodiaFirmadoEnConflicto} {
			if errors.Is(err, centinela) {
				return cero, centinela
			}
		}
		return cero, ports.ErrCustodiaFirmadoDenegada
	}
	if d.Ref != orden.DocumentoRef || d.Version != orden.Version || d.HuellaSHA256 != m.FirmadoHuella {
		return cero, ports.ErrResultadoFirmaDocumentoInvalido
	}
	return d, nil
}

// RecursoFirmaDocumento es el recurso V3 exacto de la operación: el efecto es
// la clave de idempotencia y la huella liga el material completo.
func RecursoFirmaDocumento(m ports.MaterialFirmaDocumento) (vecdomain.RecursoAutorizable, error) {
	h, err := m.HuellaSHA256()
	if err != nil {
		return vecdomain.RecursoAutorizable{}, ports.ErrSolicitudFirmaDocumentoInvalida
	}
	return vecdomain.RecursoAutorizable{
		Referencia: m.RecursoRef(), ModuloID: ports.ModuloContratacion, Tipo: ports.TipoRecursoFirmaDocumento,
		Ambitos:   map[string]string{"organizacion_ref": m.OrganizacionRef},
		Atributos: map[string]string{"material_sha256": h},
	}, nil
}

// ValidarCapacidadFirmaDocumento comprueba que el material V3 es de la
// audiencia de firma y está ligado al recurso exacto de este material.
func ValidarCapacidadFirmaDocumento(c ports.CapacidadFirmaDocumento, m ports.MaterialFirmaDocumento) error {
	material := c.ExportarMaterialParaConsumidor()
	recurso, err := RecursoFirmaDocumento(m)
	if err != nil || material.ValidarEstructura() != nil {
		return ports.ErrFirmaDocumentoDenegada
	}
	contexto, err := recurso.HuellaContextoAutorizacionSHA256()
	r := material.ResumenCapacidad()
	if err != nil || r.Operacion() != ports.AccionFirmarDocumento || r.EfectoRef() != recurso.Referencia ||
		r.EfectoHuellaSHA256() != contexto || r.AudienciaConsumo() != ports.AudienciaFirmaDocumentoV3 {
		return ports.ErrFirmaDocumentoDenegada
	}
	return nil
}
