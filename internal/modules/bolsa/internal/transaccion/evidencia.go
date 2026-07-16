// Package transaccion concentra la derivacion canonica de la evidencia
// probatoria del modulo Bolsa. Los adaptadores duraderos y efimeros comparten
// asi exactamente las mismas huellas, referencias y reglas de encadenado.
//
// La ruta internal impide que un cliente HTTP pueda presentar auditorias o
// eventos ya construidos: solo los componentes del modulo Bolsa pueden
// derivarlos a partir de una solicitud de confirmacion validada.
package transaccion

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"hash"
	"strconv"
	"strings"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const moduloBaremacion = "bolsa"

// UsoAutorizacion liga de manera durable una decision exacta con un unico
// efecto. Un adaptador puede persistir estos tres valores, pero solo esta
// fabrica interna puede extraerlos de la capacidad opaca del nucleo.
type UsoAutorizacion struct {
	DecisionRef          string
	HuellaDecisionSHA256 string
	HuellaEfectoSHA256   string
}

// PruebaDecisionAutorizacion es la proyeccion privada que un adaptador
// duradero necesita para comparar la fila de autorizacion y su representacion
// reforzada. Los bytes son una copia de la codificacion que produjo la huella;
// el adaptador no conoce ni replica ese serializador.
type PruebaDecisionAutorizacion struct {
	Uso                    UsoAutorizacion
	EsquemaHuella          string
	Decision               dominiovec.DecisionAutorizacion
	RepresentacionCanonica []byte
	VerificadaEn           time.Time
}

func (p PruebaDecisionAutorizacion) Validar() error {
	if p.Uso.Validar() != nil ||
		p.EsquemaHuella != puertosvec.EsquemaHuellaDecisionAutorizacionReforzadaV1 ||
		len(p.RepresentacionCanonica) == 0 ||
		p.Decision.ValidarEvidenciaInstantanea() != nil ||
		p.Decision.DecisionRef != p.Uso.DecisionRef || p.VerificadaEn.IsZero() {
		return puertosbolsa.ErrAutorizacionBaremacionInvalida
	}
	suma := sha256.Sum256(p.RepresentacionCanonica)
	if hex.EncodeToString(suma[:]) != p.Uso.HuellaDecisionSHA256 {
		return puertosbolsa.ErrAutorizacionBaremacionInvalida
	}
	return nil
}

// ExtraerPruebaDecisionAutorizacion solo acepta una capacidad viva del
// nucleo. Devuelve copias independientes para que la serializacion SQL no
// pueda mutar la evidencia mantenida en memoria.
func ExtraerPruebaDecisionAutorizacion(
	contexto puertosbolsa.ContextoOperacionBaremacion,
	instante time.Time,
	huellaEfecto string,
) (PruebaDecisionAutorizacion, error) {
	uso, err := NuevoUsoAutorizacion(contexto, instante, huellaEfecto)
	if err != nil {
		return PruebaDecisionAutorizacion{}, err
	}
	evidencia, err := contexto.EvidenciaUsoAutorizacion()
	if err != nil || evidencia.ValidarEn(instante) != nil {
		return PruebaDecisionAutorizacion{}, puertosbolsa.ErrAutorizacionBaremacionInvalida
	}
	datos, err := evidencia.Datos()
	if err != nil || datos.HuellaDecisionSHA256 != uso.HuellaDecisionSHA256 {
		return PruebaDecisionAutorizacion{}, puertosbolsa.ErrAutorizacionBaremacionInvalida
	}
	representacion, err := datos.RepresentacionCanonica()
	if err != nil {
		return PruebaDecisionAutorizacion{}, puertosbolsa.ErrAutorizacionBaremacionInvalida
	}
	prueba := PruebaDecisionAutorizacion{
		Uso: uso, EsquemaHuella: datos.EsquemaHuella, Decision: datos.Decision,
		RepresentacionCanonica: append([]byte(nil), representacion...), VerificadaEn: datos.VerificadaEn.UTC(),
	}
	return prueba, prueba.Validar()
}

func (u UsoAutorizacion) Validar() error {
	if u.DecisionRef == "" || !huellaSHA256Valida(u.HuellaDecisionSHA256) ||
		!huellaSHA256Valida(u.HuellaEfectoSHA256) {
		return puertosbolsa.ErrAutorizacionBaremacionInvalida
	}
	return nil
}

// NuevoUsoAutorizacion obtiene la decision y su huella desde la evidencia
// opaca creada por el nucleo. Nunca acepta esos datos como argumentos sueltos.
func NuevoUsoAutorizacion(
	contexto puertosbolsa.ContextoOperacionBaremacion,
	instante time.Time,
	huellaEfecto string,
) (UsoAutorizacion, error) {
	evidencia, err := contexto.EvidenciaUsoAutorizacion()
	if err != nil || evidencia.ValidarEn(instante) != nil || !huellaSHA256Valida(huellaEfecto) {
		return UsoAutorizacion{}, puertosbolsa.ErrAutorizacionBaremacionInvalida
	}
	datos, err := evidencia.Datos()
	proyeccion := contexto.Proyeccion()
	if err != nil || datos.Decision.DecisionRef != proyeccion.AutorizacionRef ||
		!huellaSHA256Valida(datos.HuellaDecisionSHA256) {
		return UsoAutorizacion{}, puertosbolsa.ErrAutorizacionBaremacionInvalida
	}
	uso := UsoAutorizacion{
		DecisionRef: datos.Decision.DecisionRef, HuellaDecisionSHA256: datos.HuellaDecisionSHA256,
		HuellaEfectoSHA256: huellaEfecto,
	}
	return uso, uso.Validar()
}

// HuellaEfectoReserva cubre la representacion canonica completa, incluida la
// decision de autorizacion y el vinculo de autenticacion.
func HuellaEfectoReserva(solicitud puertosbolsa.SolicitudReservarCambioBaremacion) (string, error) {
	carga, err := puertosbolsa.RepresentacionCanonicaReservaBaremacion(solicitud)
	if err != nil {
		return "", err
	}
	return HuellaCanonica("efecto-reserva-baremacion-v1", hex.EncodeToString(carga.Revelar())), nil
}

// HuellaEfectoConfirmacionV2 cubre la version, el agregado, la trazabilidad y
// las dos autorizaciones usadas por la mutacion definitiva.
func HuellaEfectoConfirmacionV2(solicitud puertosbolsa.SolicitudConfirmarCambioBaremacion) (string, error) {
	carga, err := puertosbolsa.RepresentacionCanonicaConfirmacionBaremacion(solicitud)
	if err != nil {
		return "", err
	}
	return HuellaCanonica("efecto-confirmacion-baremacion-v2", hex.EncodeToString(carga.Revelar())), nil
}

// HuellaEfectoPrevalidacionArchivoProbatorio liga el permiso consumible de
// prevalidacion al efecto completo que se confirmara. Una autorizacion no puede
// trasladarse a otra version, manifiesto, actor ni autorizacion de confirmacion.
func HuellaEfectoPrevalidacionArchivoProbatorio(
	solicitud puertosbolsa.SolicitudConfirmarCambioBaremacion,
) (string, error) {
	huellaConfirmacion, err := HuellaEfectoConfirmacionV2(solicitud)
	if err != nil {
		return "", err
	}
	return HuellaCanonica(
		"efecto-prevalidacion-archivo-probatorio-baremacion-v3", huellaConfirmacion,
	), nil
}

// HuellaEfectoAbandono liga el abandono al token, clase y baremacion exactos.
func HuellaEfectoAbandono(solicitud puertosbolsa.SolicitudAbandonarReservaBaremacion) (string, error) {
	if solicitud.Validar() != nil {
		return "", puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	return HuellaCanonica(
		"abandono-reserva-baremacion-v1", HuellaTokenReserva(solicitud.Token),
		string(solicitud.Clase), solicitud.BaremacionMeritoRef,
	), nil
}

// HuellaTokenReserva permite localizar una reserva sin guardar la capacidad
// en claro. El token original solo se devuelve una vez al llamador.
func HuellaTokenReserva(token puertosbolsa.TokenReservaBaremacion) string {
	if token.Validar() != nil {
		return ""
	}
	suma := sha256.Sum256([]byte(token.Revelar()))
	return hex.EncodeToString(suma[:])
}

// MismoUso comprueba en tiempo constante las huellas antes de tratar un
// reintento como la misma operacion.
func MismoUso(a, b UsoAutorizacion) bool {
	return a.Validar() == nil && b.Validar() == nil && a.DecisionRef == b.DecisionRef &&
		constanteIgual(a.HuellaDecisionSHA256, b.HuellaDecisionSHA256) &&
		constanteIgual(a.HuellaEfectoSHA256, b.HuellaEfectoSHA256)
}

func huellaSHA256Valida(valor string) bool {
	if len(valor) != sha256.Size*2 || strings.ToLower(valor) != valor {
		return false
	}
	contenido, err := hex.DecodeString(valor)
	return err == nil && len(contenido) == sha256.Size
}

func constanteIgual(a, b string) bool {
	return len(a) == len(b) && subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// DerivarEvidencia crea el registro de auditoria, el evento outbox y el recibo
// que deben persistirse junto a la nueva version bajo una unica transaccion.
// Las referencias se generan con 256 bits aleatorios y no contienen datos
// personales ni referencias de negocio.
func DerivarEvidencia(
	solicitud puertosbolsa.SolicitudConfirmarCambioBaremacion,
	versionAnterior, versionNueva uint64,
	huellaAnterior, huellaNueva, huellaAuditoriaAnterior, huellaEventoAnterior string,
	secuenciaAuditoria, secuenciaEvento uint64,
	registradaEn time.Time,
) (
	puertosbolsa.RegistroAuditoriaBaremacion,
	puertosbolsa.EventoOutboxBaremacion,
	puertosbolsa.EvidenciaTransaccionBaremacion,
	error,
) {
	if solicitud.Validar() != nil || secuenciaAuditoria == 0 || secuenciaEvento == 0 ||
		registradaEn.IsZero() || registradaEn.Before(solicitud.ConfirmadaEn) {
		return puertosbolsa.RegistroAuditoriaBaremacion{}, puertosbolsa.EventoOutboxBaremacion{},
			puertosbolsa.EvidenciaTransaccionBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	auditoriaRef, err := NuevaReferenciaOpaca()
	if err != nil {
		return puertosbolsa.RegistroAuditoriaBaremacion{}, puertosbolsa.EventoOutboxBaremacion{},
			puertosbolsa.EvidenciaTransaccionBaremacion{}, err
	}
	eventoRef, err := NuevaReferenciaOpaca()
	if err != nil {
		return puertosbolsa.RegistroAuditoriaBaremacion{}, puertosbolsa.EventoOutboxBaremacion{},
			puertosbolsa.EvidenciaTransaccionBaremacion{}, err
	}
	if auditoriaRef == eventoRef {
		return puertosbolsa.RegistroAuditoriaBaremacion{}, puertosbolsa.EventoOutboxBaremacion{},
			puertosbolsa.EvidenciaTransaccionBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}

	accion := puertosbolsa.AccionAuditoriaCrearBaremacion
	tipoEvento := puertosbolsa.TipoEventoBaremacionCreada
	decisionRef := ""
	manifiestoRef, huellaManifiesto := "", ""
	documentoFirmadoRef, evidenciaCustodiaFirmadoRef, evidenciaRetencionFirmadoRef := "", "", ""
	if solicitud.Clase == puertosbolsa.ClaseCambioIncorporarDecision {
		accion = puertosbolsa.AccionAuditoriaIncorporarDecision
		tipoEvento = puertosbolsa.TipoEventoDecisionIncorporada
		ultima, existe := solicitud.Agregado.UltimaDecision()
		if !existe {
			return puertosbolsa.RegistroAuditoriaBaremacion{}, puertosbolsa.EventoOutboxBaremacion{},
				puertosbolsa.EvidenciaTransaccionBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
		}
		decisionRef = ultima.Contenido.ID
		manifiestoRef = ultima.Firma.ManifiestoProbatorioRef
		huellaManifiesto = ultima.Firma.HuellaManifiestoProbatorioSHA256
		documentoFirmadoRef = ultima.Firma.DocumentoFirmadoCustodiadoRef
		evidenciaCustodiaFirmadoRef = ultima.Firma.EvidenciaCustodiaDocumentoFirmadoRef
		evidenciaRetencionFirmadoRef = ultima.Firma.EvidenciaRetencionDocumentoFirmadoRef
	}

	proyeccion := solicitud.Contexto.Proyeccion()
	auditoria := puertosbolsa.RegistroAuditoriaBaremacion{
		Referencia: auditoriaRef, Secuencia: secuenciaAuditoria,
		PrincipalRef: proyeccion.PrincipalRef, SujetoRef: proyeccion.SujetoRef,
		PerfilActorClave: proyeccion.PerfilActorClave, MetodoAutenticacion: proyeccion.MetodoAutenticacion,
		NivelAutenticacion: proyeccion.NivelAutenticacion, GarantiaMinima: proyeccion.GarantiaMinima,
		AutenticacionRef: proyeccion.AutenticacionRef, AutorizacionRef: proyeccion.AutorizacionRef,
		AccionAutorizada: proyeccion.Accion, ClaseRecursoAutorizada: proyeccion.ClaseRecurso,
		RecursoAutorizadoRef: proyeccion.RecursoRef,
		CamposPermitidos:     append([]string(nil), proyeccion.CamposPermitidos...),
		FinalidadClave:       proyeccion.FinalidadClave, CorrelacionRef: proyeccion.CorrelacionRef,
		Modulo: moduloBaremacion, Accion: accion, ClaseCambio: solicitud.Clase,
		ProcesoRef: solicitud.Agregado.ProcesoRef, SolicitudRef: solicitud.Agregado.SolicitudRef,
		BaremacionMeritoRef: solicitud.Agregado.ID, DecisionRef: decisionRef,
		ManifiestoProbatorioRef: manifiestoRef, HuellaManifiestoSHA256: huellaManifiesto,
		DocumentoFirmadoCustodiadoRef: documentoFirmadoRef,
		EvidenciaCustodiaFirmadoRef:   evidenciaCustodiaFirmadoRef,
		EvidenciaRetencionFirmadoRef:  evidenciaRetencionFirmadoRef,
		VersionAnterior:               versionAnterior, VersionNueva: versionNueva,
		HuellaAnteriorSHA256: huellaAnterior, HuellaNuevaSHA256: huellaNueva,
		MotivoClave: solicitud.Trazabilidad.MotivoClave, Motivo: solicitud.Trazabilidad.Motivo,
		HuellaSolicitudHMAC: solicitud.HuellaSolicitudHMAC, Resultado: "correcto",
		SolicitadaConfirmacionEn: solicitud.ConfirmadaEn.UTC(), RegistradaEn: registradaEn.UTC(),
		HuellaAnteriorAuditoriaSHA256: huellaAuditoriaAnterior,
	}
	auditoria.HuellaRegistroSHA256 = HuellaAuditoria(auditoria)

	evento := puertosbolsa.EventoOutboxBaremacion{
		Referencia: eventoRef, Secuencia: secuenciaEvento, Tipo: tipoEvento,
		Estado: puertosbolsa.EstadoEventoOutboxBaremacionPendiente, Modulo: moduloBaremacion,
		ProcesoRef: solicitud.Agregado.ProcesoRef, SolicitudRef: solicitud.Agregado.SolicitudRef,
		BaremacionMeritoRef: solicitud.Agregado.ID, DecisionRef: decisionRef,
		ManifiestoProbatorioRef: manifiestoRef, HuellaManifiestoSHA256: huellaManifiesto,
		DocumentoFirmadoRef:          documentoFirmadoRef,
		EvidenciaCustodiaFirmadoRef:  evidenciaCustodiaFirmadoRef,
		EvidenciaRetencionFirmadoRef: evidenciaRetencionFirmadoRef,
		SujetoRef:                    proyeccion.SujetoRef, PrincipalRef: proyeccion.PrincipalRef,
		VersionNueva: versionNueva, HuellaNuevaSHA256: huellaNueva,
		AuditoriaRef: auditoria.Referencia, HuellaAuditoriaSHA256: auditoria.HuellaRegistroSHA256,
		CorrelacionRef: proyeccion.CorrelacionRef, RegistradoEn: registradaEn.UTC(),
		HuellaEventoAnteriorSHA256: huellaEventoAnterior,
	}
	evento.HuellaRegistroSHA256 = HuellaEvento(evento)
	if auditoria.Validar() != nil || evento.Validar() != nil {
		return puertosbolsa.RegistroAuditoriaBaremacion{}, puertosbolsa.EventoOutboxBaremacion{},
			puertosbolsa.EvidenciaTransaccionBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}

	evidencia := puertosbolsa.EvidenciaTransaccionBaremacion{
		AuditoriaRef: auditoria.Referencia, HuellaAuditoriaSHA256: auditoria.HuellaRegistroSHA256,
		EventoOutboxRef: evento.Referencia, HuellaEventoOutboxSHA256: evento.HuellaRegistroSHA256,
		ConfirmadaEn: registradaEn.UTC(),
	}
	if err := evidencia.Validar(); err != nil {
		return puertosbolsa.RegistroAuditoriaBaremacion{}, puertosbolsa.EventoOutboxBaremacion{},
			puertosbolsa.EvidenciaTransaccionBaremacion{}, err
	}
	return auditoria, evento, evidencia, nil
}

// HuellaAuditoria calcula la cadena canonica sin incluir la propia huella.
func HuellaAuditoria(a puertosbolsa.RegistroAuditoriaBaremacion) string {
	return HuellaCanonica(
		a.Referencia, strconv.FormatUint(a.Secuencia, 10), a.PrincipalRef, a.SujetoRef,
		a.PerfilActorClave, string(a.MetodoAutenticacion), string(a.NivelAutenticacion),
		string(a.GarantiaMinima), a.AutenticacionRef, a.AutorizacionRef,
		string(a.AccionAutorizada), string(a.ClaseRecursoAutorizada), a.RecursoAutorizadoRef,
		strings.Join(a.CamposPermitidos, "\x00"), a.FinalidadClave, a.CorrelacionRef,
		a.Modulo, string(a.Accion), string(a.ClaseCambio), a.ProcesoRef, a.SolicitudRef,
		a.BaremacionMeritoRef, a.DecisionRef, a.ManifiestoProbatorioRef,
		a.HuellaManifiestoSHA256, a.DocumentoFirmadoCustodiadoRef,
		a.EvidenciaCustodiaFirmadoRef, a.EvidenciaRetencionFirmadoRef,
		strconv.FormatUint(a.VersionAnterior, 10), strconv.FormatUint(a.VersionNueva, 10),
		a.HuellaAnteriorSHA256, a.HuellaNuevaSHA256, a.MotivoClave, a.Motivo,
		a.HuellaSolicitudHMAC, a.Resultado,
		a.SolicitadaConfirmacionEn.UTC().Format(time.RFC3339Nano),
		a.RegistradaEn.UTC().Format(time.RFC3339Nano), a.HuellaAnteriorAuditoriaSHA256,
	)
}

// HuellaEvento calcula la cadena canonica sin incluir la propia huella.
func HuellaEvento(e puertosbolsa.EventoOutboxBaremacion) string {
	return HuellaCanonica(
		e.Referencia, strconv.FormatUint(e.Secuencia, 10), string(e.Tipo), string(e.Estado),
		e.Modulo, e.ProcesoRef, e.SolicitudRef, e.BaremacionMeritoRef, e.DecisionRef,
		e.ManifiestoProbatorioRef, e.HuellaManifiestoSHA256, e.DocumentoFirmadoRef,
		e.EvidenciaCustodiaFirmadoRef, e.EvidenciaRetencionFirmadoRef,
		e.SujetoRef, e.PrincipalRef, strconv.FormatUint(e.VersionNueva, 10),
		e.HuellaNuevaSHA256, e.AuditoriaRef, e.HuellaAuditoriaSHA256, e.CorrelacionRef,
		e.RegistradoEn.UTC().Format(time.RFC3339Nano), e.HuellaEventoAnteriorSHA256,
	)
}

// HuellaCanonica usa longitudes binarias para evitar concatenaciones ambiguas.
func HuellaCanonica(partes ...string) string {
	digestor := sha256.New()
	for _, parte := range partes {
		escribirParte(digestor, parte)
	}
	return hex.EncodeToString(digestor.Sum(nil))
}

func escribirParte(destino hash.Hash, parte string) {
	var longitud [8]byte
	binary.BigEndian.PutUint64(longitud[:], uint64(len(parte)))
	_, _ = destino.Write(longitud[:])
	_, _ = destino.Write([]byte(parte))
}

// NuevaReferenciaOpaca devuelve una referencia Base64URL de 256 bits.
func NuevaReferenciaOpaca() (string, error) {
	contenido := make([]byte, 32)
	if _, err := rand.Read(contenido); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(contenido), nil
}

// GenerarTokenReserva crea una capacidad temporal apta para el contrato del
// puerto, sin exponer el material aleatorio en logs o serializadores.
func GenerarTokenReserva() (puertosbolsa.TokenReservaBaremacion, error) {
	referencia, err := NuevaReferenciaOpaca()
	if err != nil {
		return puertosbolsa.TokenReservaBaremacion{}, err
	}
	return puertosbolsa.NuevoTokenReservaBaremacion(referencia)
}
