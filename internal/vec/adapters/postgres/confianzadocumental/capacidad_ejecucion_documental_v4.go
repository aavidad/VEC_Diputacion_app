package confianzadocumental

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/veraison/go-cose"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const (
	esquemaCapacidadEjecucionDocumentalV4         = "vec.documentos.capacidad-ejecucion.v4"
	audienciaCapacidadEjecucionDocumentalV4       = "vec_ejecucion_documental_v4.ejecutar_plan_atestado"
	esquemaPaqueteEjecucionDocumentalV4           = "vec.documentos.paquete-ejecucion-atestada.v4"
	esquemaSolicitudRemotaCapacidadV4             = "vec.documentos.solicitud-remota-capacidad.v4"
	rutaHTTPEmisorCapacidadDocumentalV4           = "/v1/capacidades-ejecucion-documental"
	emisionCapacidadMaximaV4                      = 15 * time.Second
	emisionCapacidadPreferidaV4                   = 10 * time.Second
	maximoSolicitudRemotaCapacidadV4        int64 = 4 * 1024 * 1024
	maximoPaqueteRemotoCapacidadV4          int64 = 12 * 1024 * 1024
)

var errCapacidadEjecucionDocumentalV4Invalida = errors.New(
	"vec: capacidad de ejecucion documental v4 invalida",
)

// materialEmisorCapacidadDocumentalV4 solo existe en el proceso verificador.
// El proceso web no posee este tipo, el pool emisor ni el secreto HMAC.
type materialEmisorCapacidadDocumentalV4 struct {
	claveID     string
	version     uint64
	secreto     []byte
	emisorID    string
	validaDesde time.Time
	validaHasta time.Time
	estado      string
	revocadaEn  time.Time
}

func (m materialEmisorCapacidadDocumentalV4) validarEn(instante time.Time) error {
	if !referenciaDurableAtestacionPDPValida(m.claveID) || m.version == 0 ||
		len(m.secreto) < 32 || len(m.secreto) > 4096 ||
		!referenciaDurableAtestacionPDPValida(m.emisorID) ||
		!instanteCanonicoDocumental(m.validaDesde) ||
		!instanteCanonicoDocumental(m.validaHasta) ||
		m.estado != "activa" || !m.revocadaEn.IsZero() ||
		instante.Before(m.validaDesde) || !instante.Before(m.validaHasta) {
		return errCapacidadEjecucionDocumentalV4Invalida
	}
	return nil
}

type capacidadEjecucionDocumentalV4JSON struct {
	Esquema                   string `json:"esquema"`
	ClaveID                   string `json:"clave_id"`
	ClaveVersion              uint64 `json:"clave_version"`
	EmisorID                  string `json:"emisor_id"`
	Audiencia                 string `json:"audiencia"`
	Nonce                     string `json:"nonce"`
	EmitidaEn                 string `json:"emitida_en"`
	ExpiraEn                  string `json:"expira_en"`
	HuellaMetadatosSHA256     string `json:"huella_metadatos_sha256"`
	HuellaPayloadSHA256       string `json:"huella_payload_sha256"`
	HuellaSobreSHA256         string `json:"huella_sobre_sha256"`
	HuellaEvidenciaSHA256     string `json:"huella_evidencia_sha256"`
	HuellaPreimagenSHA256     string `json:"huella_preimagen_sha256"`
	HuellaDecisionSHA256      string `json:"huella_decision_sha256"`
	HuellaEfectoSHA256        string `json:"huella_efecto_sha256"`
	RevisionConfianza         string `json:"revision_confianza"`
	HuellaConfiguracionSHA256 string `json:"huella_configuracion_sha256"`
	RaizClaveID               string `json:"raiz_clave_id"`
	HuellaRaizSHA256          string `json:"huella_raiz_sha256"`
	MACSHA256                 string `json:"mac_sha256"`
}

func (c capacidadEjecucionDocumentalV4JSON) valoresAutenticados() []string {
	return []string{
		c.Esquema, c.ClaveID, strconv.FormatUint(c.ClaveVersion, 10), c.EmisorID,
		c.Audiencia, c.Nonce, c.EmitidaEn, c.ExpiraEn,
		c.HuellaMetadatosSHA256, c.HuellaPayloadSHA256, c.HuellaSobreSHA256,
		c.HuellaEvidenciaSHA256, c.HuellaPreimagenSHA256,
		c.HuellaDecisionSHA256, c.HuellaEfectoSHA256, c.RevisionConfianza,
		c.HuellaConfiguracionSHA256, c.RaizClaveID, c.HuellaRaizSHA256,
	}
}

func preimagenMACCapacidadDocumentalV4(valores []string) []byte {
	var salida bytes.Buffer
	for _, valor := range valores {
		salida.WriteString(strconv.Itoa(len([]byte(valor))))
		salida.WriteByte(':')
		salida.WriteString(valor)
		salida.WriteByte('\n')
	}
	return salida.Bytes()
}

func (c capacidadEjecucionDocumentalV4JSON) validarSinSecretoEn(
	instante time.Time,
	artefactos artefactosEjecucionDocumentalV4,
) error {
	emitidaEn, errEmitida := time.Parse(time.RFC3339Nano, c.EmitidaEn)
	expiraEn, errExpira := time.Parse(time.RFC3339Nano, c.ExpiraEn)
	if errEmitida != nil || errExpira != nil {
		return errCapacidadEjecucionDocumentalV4Invalida
	}
	emitidaEn = emitidaEn.UTC()
	expiraEn = expiraEn.UTC()
	if c.Esquema != esquemaCapacidadEjecucionDocumentalV4 ||
		!referenciaDurableAtestacionPDPValida(c.ClaveID) || c.ClaveVersion == 0 ||
		!referenciaDurableAtestacionPDPValida(c.EmisorID) ||
		c.Audiencia != audienciaCapacidadEjecucionDocumentalV4 ||
		len(c.Nonce) != 64 || !huellaSHA256DocumentalValida(c.Nonce) ||
		!instanteCanonicoDocumental(emitidaEn) || !instanteCanonicoDocumental(expiraEn) ||
		!expiraEn.After(emitidaEn) || expiraEn.Sub(emitidaEn) > emisionCapacidadMaximaV4 ||
		instante.Before(emitidaEn) || !instante.Before(expiraEn) ||
		!huellaSHA256DocumentalValida(c.MACSHA256) ||
		c.HuellaMetadatosSHA256 != huellaBytesDocumentales(artefactos.metadatos) ||
		c.HuellaPayloadSHA256 != huellaBytesDocumentales(artefactos.payload) ||
		c.HuellaSobreSHA256 != huellaBytesDocumentales(artefactos.sobre) ||
		c.HuellaEvidenciaSHA256 != huellaBytesDocumentales(artefactos.evidencia) ||
		c.HuellaPreimagenSHA256 != huellaBytesDocumentales(artefactos.preimagen) ||
		c.HuellaDecisionSHA256 != huellaBytesDocumentales(artefactos.decisionCanonica) ||
		c.HuellaEfectoSHA256 != huellaBytesDocumentales(artefactos.efecto) {
		return errCapacidadEjecucionDocumentalV4Invalida
	}
	var metadatos metadatosAtestacionEjecucionDocumentalV4PostgreSQL
	if decodificarJSONExactoDocumentalV4(artefactos.metadatos, &metadatos) != nil ||
		c.RevisionConfianza != metadatos.RevisionConfianza ||
		c.HuellaConfiguracionSHA256 != metadatos.HuellaConfiguracionSHA256 ||
		c.RaizClaveID != metadatos.ClaveID ||
		c.HuellaRaizSHA256 != metadatos.HuellaClaveSHA256 {
		return errCapacidadEjecucionDocumentalV4Invalida
	}
	return nil
}

func (c capacidadEjecucionDocumentalV4JSON) validarConMaterialEn(
	instante time.Time,
	artefactos artefactosEjecucionDocumentalV4,
	material materialEmisorCapacidadDocumentalV4,
) error {
	if c.validarSinSecretoEn(instante, artefactos) != nil ||
		material.validarEn(instante) != nil || c.ClaveID != material.claveID ||
		c.ClaveVersion != material.version || c.EmisorID != material.emisorID {
		return errCapacidadEjecucionDocumentalV4Invalida
	}
	mac := hmac.New(sha256.New, material.secreto)
	_, _ = mac.Write(preimagenMACCapacidadDocumentalV4(c.valoresAutenticados()))
	esperada, err := hex.DecodeString(c.MACSHA256)
	if err != nil || subtle.ConstantTimeCompare(mac.Sum(nil), esperada) != 1 {
		return errCapacidadEjecucionDocumentalV4Invalida
	}
	return nil
}

type artefactosEjecucionDocumentalV4 struct {
	metadatos        []byte
	payload          []byte
	sobre            []byte
	evidencia        []byte
	preimagen        []byte
	decisionCanonica []byte
	efecto           []byte
	capacidad        []byte
}

func (a artefactosEjecucionDocumentalV4) validarEn(instante time.Time) error {
	limites := []struct {
		valor  []byte
		minimo int
		maximo int
	}{
		{a.metadatos, 2, 524288}, {a.payload, 1, 524288},
		{a.sobre, 16, 528384}, {a.evidencia, 1, 2097152},
		{a.preimagen, 1, 2097152}, {a.decisionCanonica, 128, 524288},
		{a.efecto, 2, 524288}, {a.capacidad, 2, 32768},
	}
	for _, limite := range limites {
		if len(limite.valor) < limite.minimo || len(limite.valor) > limite.maximo {
			return errCapacidadEjecucionDocumentalV4Invalida
		}
	}
	var capacidad capacidadEjecucionDocumentalV4JSON
	if decodificarJSONExactoDocumentalV4(a.capacidad, &capacidad) != nil ||
		capacidad.validarSinSecretoEn(instante, a) != nil {
		return errCapacidadEjecucionDocumentalV4Invalida
	}
	return nil
}

type paqueteEjecucionDocumentalV4JSON struct {
	Esquema          string          `json:"esquema"`
	Version          uint16          `json:"version"`
	Metadatos        []byte          `json:"metadatos"`
	Payload          []byte          `json:"payload"`
	Sobre            []byte          `json:"sobre"`
	Evidencia        []byte          `json:"evidencia"`
	Preimagen        []byte          `json:"preimagen"`
	DecisionCanonica []byte          `json:"decision_canonica"`
	Efecto           []byte          `json:"efecto"`
	Capacidad        json.RawMessage `json:"capacidad"`
}

func serializarPaqueteEjecucionDocumentalV4(a artefactosEjecucionDocumentalV4) ([]byte, error) {
	var capacidad capacidadEjecucionDocumentalV4JSON
	if decodificarJSONExactoDocumentalV4(a.capacidad, &capacidad) != nil {
		return nil, errCapacidadEjecucionDocumentalV4Invalida
	}
	emitidaEn, err := time.Parse(time.RFC3339Nano, capacidad.EmitidaEn)
	if err != nil || a.validarEn(emitidaEn.UTC()) != nil {
		return nil, errCapacidadEjecucionDocumentalV4Invalida
	}
	return json.Marshal(paqueteEjecucionDocumentalV4JSON{
		Esquema: esquemaPaqueteEjecucionDocumentalV4, Version: 1,
		Metadatos: append([]byte(nil), a.metadatos...), Payload: append([]byte(nil), a.payload...),
		Sobre: append([]byte(nil), a.sobre...), Evidencia: append([]byte(nil), a.evidencia...),
		Preimagen:        append([]byte(nil), a.preimagen...),
		DecisionCanonica: append([]byte(nil), a.decisionCanonica...),
		Efecto:           append([]byte(nil), a.efecto...), Capacidad: append(json.RawMessage(nil), a.capacidad...),
	})
}

func interpretarPaqueteEjecucionDocumentalV4(
	contenido []byte,
	instante time.Time,
) (artefactosEjecucionDocumentalV4, error) {
	if len(contenido) == 0 || int64(len(contenido)) > maximoPaqueteRemotoCapacidadV4 {
		return artefactosEjecucionDocumentalV4{}, errCapacidadEjecucionDocumentalV4Invalida
	}
	var paquete paqueteEjecucionDocumentalV4JSON
	if decodificarJSONExactoDocumentalV4(contenido, &paquete) != nil ||
		paquete.Esquema != esquemaPaqueteEjecucionDocumentalV4 || paquete.Version != 1 {
		return artefactosEjecucionDocumentalV4{}, errCapacidadEjecucionDocumentalV4Invalida
	}
	a := artefactosEjecucionDocumentalV4{
		metadatos: append([]byte(nil), paquete.Metadatos...),
		payload:   append([]byte(nil), paquete.Payload...), sobre: append([]byte(nil), paquete.Sobre...),
		evidencia: append([]byte(nil), paquete.Evidencia...), preimagen: append([]byte(nil), paquete.Preimagen...),
		decisionCanonica: append([]byte(nil), paquete.DecisionCanonica...),
		efecto:           append([]byte(nil), paquete.Efecto...), capacidad: append([]byte(nil), paquete.Capacidad...),
	}
	if a.validarEn(instante) != nil {
		return artefactosEjecucionDocumentalV4{}, errCapacidadEjecucionDocumentalV4Invalida
	}
	return a, nil
}

type cabeceraRemotaCapacidadV4 struct {
	FormatoVersion uint16 `json:"formato_version"`
	Suite          string `json:"suite"`
	ClaveID        string `json:"clave_id"`
	Audiencia      string `json:"audiencia"`
}

type solicitudRemotaCapacidadV4 struct {
	Esquema   string                    `json:"esquema"`
	Cabecera  cabeceraRemotaCapacidadV4 `json:"cabecera"`
	Sobre     []byte                    `json:"sobre"`
	Preimagen []byte                    `json:"preimagen"`
}

func (s solicitudRemotaCapacidadV4) cabeceraDominio() domain.CabeceraAtestacionAutorizacionV1 {
	return domain.CabeceraAtestacionAutorizacionV1{
		FormatoVersion: s.Cabecera.FormatoVersion, Suite: s.Cabecera.Suite,
		ClaveID: s.Cabecera.ClaveID, Audiencia: s.Cabecera.Audiencia,
	}
}

// clienteEmisorCapacidadUnixV4 es concreto y solo conoce un socket Unix. No
// existe un puerto publico inyectable capaz de sustituir al verificador.
type clienteEmisorCapacidadUnixV4 struct {
	cliente *http.Client
	url     string
}

func nuevoClienteEmisorCapacidadUnixV4(rutaSocket string) (*clienteEmisorCapacidadUnixV4, error) {
	if !rutaSocketUnixCapacidadV4Valida(rutaSocket) {
		return nil, errCapacidadEjecucionDocumentalV4Invalida
	}
	marcador := rutaSocket
	transporte := &http.Transport{
		Proxy: nil, DisableCompression: true, ForceAttemptHTTP2: false,
		MaxIdleConns: 2, MaxIdleConnsPerHost: 2, IdleConnTimeout: 30 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			marcadorDial := net.Dialer{Timeout: 3 * time.Second, KeepAlive: 30 * time.Second}
			return marcadorDial.DialContext(ctx, "unix", marcador)
		},
	}
	return &clienteEmisorCapacidadUnixV4{
		cliente: &http.Client{Transport: transporte, Timeout: 8 * time.Second},
		url:     "http://unix" + rutaHTTPEmisorCapacidadDocumentalV4,
	}, nil
}

func (c *clienteEmisorCapacidadUnixV4) solicitar(
	ctx context.Context,
	cabecera domain.CabeceraAtestacionAutorizacionV1,
	sobre, preimagen []byte,
) (artefactosEjecucionDocumentalV4, error) {
	if c == nil || c.cliente == nil || ctx == nil || cabecera.Validar() != nil {
		return artefactosEjecucionDocumentalV4{}, errCapacidadEjecucionDocumentalV4Invalida
	}
	cuerpo, err := json.Marshal(solicitudRemotaCapacidadV4{
		Esquema: esquemaSolicitudRemotaCapacidadV4,
		Cabecera: cabeceraRemotaCapacidadV4{
			FormatoVersion: cabecera.FormatoVersion, Suite: cabecera.Suite,
			ClaveID: cabecera.ClaveID, Audiencia: cabecera.Audiencia,
		},
		Sobre: append([]byte(nil), sobre...), Preimagen: append([]byte(nil), preimagen...),
	})
	if err != nil || int64(len(cuerpo)) > maximoSolicitudRemotaCapacidadV4 {
		return artefactosEjecucionDocumentalV4{}, errCapacidadEjecucionDocumentalV4Invalida
	}
	peticion, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(cuerpo))
	if err != nil {
		return artefactosEjecucionDocumentalV4{}, errCapacidadEjecucionDocumentalV4Invalida
	}
	peticion.Header.Set("Content-Type", "application/json")
	peticion.Header.Set("Accept", "application/json")
	respuesta, err := c.cliente.Do(peticion)
	if err != nil {
		return artefactosEjecucionDocumentalV4{}, errCapacidadEjecucionDocumentalV4Invalida
	}
	defer respuesta.Body.Close()
	if respuesta.StatusCode != http.StatusOK ||
		!strings.HasPrefix(respuesta.Header.Get("Content-Type"), "application/json") {
		_, _ = io.Copy(io.Discard, io.LimitReader(respuesta.Body, 4096))
		return artefactosEjecucionDocumentalV4{}, errCapacidadEjecucionDocumentalV4Invalida
	}
	contenido, err := io.ReadAll(io.LimitReader(respuesta.Body, maximoPaqueteRemotoCapacidadV4+1))
	if err != nil || int64(len(contenido)) > maximoPaqueteRemotoCapacidadV4 {
		return artefactosEjecucionDocumentalV4{}, errCapacidadEjecucionDocumentalV4Invalida
	}
	return interpretarPaqueteEjecucionDocumentalV4(
		contenido, time.Now().UTC().Truncate(time.Microsecond),
	)
}

func rutaSocketUnixCapacidadV4Valida(ruta string) bool {
	return strings.HasPrefix(ruta, "/") && len(ruta) >= 2 && len(ruta) <= 4096 &&
		ruta == strings.TrimSpace(ruta) && !strings.ContainsAny(ruta, "\x00\r\n\t")
}

func decodificarJSONExactoDocumentalV4(contenido []byte, destino any) error {
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(destino); err != nil {
		return err
	}
	if err := decodificador.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errCapacidadEjecucionDocumentalV4Invalida
	}
	return nil
}

type politicaDecisionCanonicaRemotaV4 struct {
	Referencia   string `json:"referencia"`
	HuellaSHA256 string `json:"huella_sha256"`
}

type vinculoDecisionCanonicaRemotaV4 struct {
	BloqueVersion                uint16 `json:"bloque_version"`
	AutenticacionRef             string `json:"autenticacion_ref"`
	AutenticacionHuellaSHA256    string `json:"autenticacion_huella_sha256"`
	AsercionRef                  string `json:"asercion_ref"`
	SesionRef                    string `json:"sesion_ref"`
	ControlSesionRef             string `json:"control_sesion_ref"`
	ControlSesionRevision        uint64 `json:"control_sesion_revision"`
	ControlSesionHuellaSHA256    string `json:"control_sesion_huella_sha256"`
	CuentaRef                    string `json:"cuenta_ref"`
	CuentaOrdinariaRef           string `json:"cuenta_ordinaria_ref"`
	PrincipalID                  string `json:"principal_id"`
	PerfilActivoRef              string `json:"perfil_activo_ref"`
	CuentaPrivilegiada           bool   `json:"cuenta_privilegiada"`
	Superficie                   string `json:"superficie"`
	MetodoObservado              string `json:"metodo_observado"`
	GarantiaObservada            string `json:"garantia_observada"`
	PoliticaGarantiaRef          string `json:"politica_garantia_ref"`
	PoliticaGarantiaHuellaSHA256 string `json:"politica_garantia_huella_sha256"`
	AutenticacionVerificadaEn    string `json:"autenticacion_verificada_en"`
	SesionEmitidaEn              string `json:"sesion_emitida_en"`
	SesionValidaHasta            string `json:"sesion_valida_hasta"`
	SesionRevalidadaEn           string `json:"sesion_revalidada_en"`
	ContextoActorRef             string `json:"contexto_actor_ref"`
	ContextoActorVersion         uint64 `json:"contexto_actor_version"`
	ContextoActorHuellaSHA256    string `json:"contexto_actor_huella_sha256"`
}

type decisionCanonicaRemotaV4 struct {
	Esquema                               string                             `json:"esquema"`
	DecisionRef                           string                             `json:"decision_ref"`
	Concedida                             bool                               `json:"concedida"`
	Codigo                                string                             `json:"codigo"`
	PrincipalID                           string                             `json:"principal_id"`
	PerfilActivoRef                       string                             `json:"perfil_activo_ref"`
	Accion                                string                             `json:"accion"`
	RecursoRef                            string                             `json:"recurso_ref"`
	ModuloID                              string                             `json:"modulo_id"`
	TipoRecurso                           string                             `json:"tipo_recurso"`
	ContextoRecursoHuellaSHA256           string                             `json:"contexto_recurso_huella_sha256"`
	Finalidad                             string                             `json:"finalidad"`
	CorrelacionRef                        string                             `json:"correlacion_ref"`
	VinculoAutenticacionActor             vinculoDecisionCanonicaRemotaV4    `json:"vinculo_autenticacion_actor"`
	AsignacionRef                         string                             `json:"asignacion_ref"`
	AsignacionHuellaSHA256                string                             `json:"asignacion_huella_sha256"`
	VersionRolRef                         string                             `json:"version_rol_ref"`
	VersionRolHuellaSHA256                string                             `json:"version_rol_huella_sha256"`
	ControlVigenciaVersionRolRef          string                             `json:"control_vigencia_version_rol_ref"`
	ControlVigenciaVersionRolRevision     uint64                             `json:"control_vigencia_version_rol_revision"`
	ControlVigenciaVersionRolHuellaSHA256 string                             `json:"control_vigencia_version_rol_huella_sha256"`
	RevisionCatalogoPoliticas             uint64                             `json:"revision_catalogo_politicas"`
	CatalogoPoliticasHuellaSHA256         string                             `json:"catalogo_politicas_huella_sha256"`
	PoliticasEvaluadas                    []politicaDecisionCanonicaRemotaV4 `json:"politicas_evaluadas"`
	PoliticasAplicables                   []politicaDecisionCanonicaRemotaV4 `json:"politicas_aplicables"`
	GarantiaMinima                        string                             `json:"garantia_minima"`
	CamposPermitidos                      []string                           `json:"campos_permitidos"`
	Obligaciones                          []string                           `json:"obligaciones"`
	EmitidaEn                             string                             `json:"emitida_en"`
	ValidaHasta                           string                             `json:"valida_hasta"`
}

func decisionCanonicaDesdeHistoricoV4(
	d domain.DatosDecisionHistoricaAtestacionAutorizacionV1,
) ([]byte, error) {
	politicasEvaluadas, err := politicasCanonicasRemotasV4(
		d.PoliticasEvaluadasRefs, d.PoliticasEvaluadasHuellasSHA256,
	)
	if err != nil {
		return nil, err
	}
	politicasAplicables, err := politicasCanonicasRemotasV4(d.PoliticasRefs, d.PoliticasHuellasSHA256)
	if err != nil {
		return nil, err
	}
	v := d.VinculoAutenticacionActor
	const formato = "2006-01-02T15:04:05.000000Z"
	return json.Marshal(decisionCanonicaRemotaV4{
		Esquema:     ports.EsquemaHuellaDecisionAutorizacionReforzadaV1,
		DecisionRef: d.DecisionRef, Concedida: d.Concedida, Codigo: d.Codigo,
		PrincipalID: d.PrincipalID, PerfilActivoRef: d.PerfilActivoRef, Accion: d.Accion,
		RecursoRef: d.RecursoRef, ModuloID: d.ModuloID, TipoRecurso: d.TipoRecurso,
		ContextoRecursoHuellaSHA256: d.ContextoRecursoHuellaSHA256,
		Finalidad:                   d.Finalidad, CorrelacionRef: d.CorrelacionRef,
		VinculoAutenticacionActor: vinculoDecisionCanonicaRemotaV4{
			BloqueVersion: v.BloqueVersion, AutenticacionRef: v.AutenticacionRef,
			AutenticacionHuellaSHA256: v.AutenticacionHuellaSHA256, AsercionRef: v.AsercionRef,
			SesionRef: v.SesionRef, ControlSesionRef: v.ControlSesionRef,
			ControlSesionRevision:     v.ControlSesionRevision,
			ControlSesionHuellaSHA256: v.ControlSesionHuellaSHA256,
			CuentaRef:                 v.CuentaRef, CuentaOrdinariaRef: v.CuentaOrdinariaRef,
			PrincipalID: v.PrincipalID, PerfilActivoRef: v.PerfilActivoRef,
			CuentaPrivilegiada: v.CuentaPrivilegiada, Superficie: string(v.Superficie),
			MetodoObservado: string(v.MetodoObservado), GarantiaObservada: string(v.GarantiaObservada),
			PoliticaGarantiaRef:          v.PoliticaGarantiaRef,
			PoliticaGarantiaHuellaSHA256: v.PoliticaGarantiaHuellaSHA256,
			AutenticacionVerificadaEn:    v.AutenticacionVerificadaEn.UTC().Format(formato),
			SesionEmitidaEn:              v.SesionEmitidaEn.UTC().Format(formato),
			SesionValidaHasta:            v.SesionValidaHasta.UTC().Format(formato),
			SesionRevalidadaEn:           v.SesionRevalidadaEn.UTC().Format(formato),
			ContextoActorRef:             v.ContextoActorRef, ContextoActorVersion: v.ContextoActorVersion,
			ContextoActorHuellaSHA256: v.ContextoActorHuellaSHA256,
		},
		AsignacionRef: d.AsignacionRef, AsignacionHuellaSHA256: d.AsignacionHuellaSHA256,
		VersionRolRef: d.VersionRolRef, VersionRolHuellaSHA256: d.VersionRolHuellaSHA256,
		ControlVigenciaVersionRolRef:          d.ControlVigenciaVersionRolRef,
		ControlVigenciaVersionRolRevision:     d.ControlVigenciaVersionRolRevision,
		ControlVigenciaVersionRolHuellaSHA256: d.ControlVigenciaVersionRolHuellaSHA256,
		RevisionCatalogoPoliticas:             d.RevisionCatalogoPoliticas,
		CatalogoPoliticasHuellaSHA256:         d.CatalogoPoliticasHuellaSHA256,
		PoliticasEvaluadas:                    politicasEvaluadas, PoliticasAplicables: politicasAplicables,
		GarantiaMinima:   string(d.GarantiaMinima),
		CamposPermitidos: append([]string{}, d.CamposPermitidos...),
		Obligaciones:     append([]string{}, d.Obligaciones...),
		EmitidaEn:        d.EmitidaEn.UTC().Format(formato), ValidaHasta: d.ValidaHasta.UTC().Format(formato),
	})
}

func politicasCanonicasRemotasV4(refs []string, huellas map[string]string) ([]politicaDecisionCanonicaRemotaV4, error) {
	if len(refs) != len(huellas) {
		return nil, errCapacidadEjecucionDocumentalV4Invalida
	}
	ordenadas := append([]string(nil), refs...)
	sort.Strings(ordenadas)
	resultado := make([]politicaDecisionCanonicaRemotaV4, 0, len(ordenadas))
	for _, ref := range ordenadas {
		huella, existe := huellas[ref]
		if !existe || !huellaSHA256DocumentalValida(huella) {
			return nil, errCapacidadEjecucionDocumentalV4Invalida
		}
		resultado = append(resultado, politicaDecisionCanonicaRemotaV4{Referencia: ref, HuellaSHA256: huella})
	}
	return resultado, nil
}

func huellaCanonicaRemotaV4(valores ...string) string {
	calculador := sha256.New()
	_, _ = calculador.Write(preimagenMACCapacidadDocumentalV4(valores))
	return hex.EncodeToString(calculador.Sum(nil))
}

func huellaListaRemotaV4(esquema string, valores []string) string {
	ordenados := append([]string(nil), valores...)
	sort.Strings(ordenados)
	return huellaCanonicaRemotaV4(append([]string{esquema}, ordenados...)...)
}

func huellaMapaRemotoV4(esquema string, valores map[string]string) string {
	claves := make([]string, 0, len(valores))
	for clave := range valores {
		claves = append(claves, clave)
	}
	sort.Strings(claves)
	campos := []string{esquema}
	for _, clave := range claves {
		campos = append(campos, clave, valores[clave])
	}
	return huellaCanonicaRemotaV4(campos...)
}

func extraerPayloadCOSEDocumentalV4(sobre []byte) ([]byte, error) {
	var mensaje cose.Sign1Message
	if err := mensaje.UnmarshalCBOR(sobre); err != nil || len(mensaje.Payload) == 0 ||
		len(mensaje.Payload) > domain.TamanoMaximoMensajeAtestacionAutorizacionV1 {
		return nil, errCapacidadEjecucionDocumentalV4Invalida
	}
	return append([]byte(nil), mensaje.Payload...), nil
}

func nonceCapacidadDocumentalV4() (string, error) {
	var nonce [32]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(nonce[:]), nil
}

type manejadorHTTPEmisorCapacidadDocumentalV4 struct {
	servicio *Servicio
	material materialEmisorCapacidadDocumentalV4
}

// NuevoManejadorHTTPEmisorCapacidadDocumentalV4 es el unico ensamblado del
// proceso emisor. Debe ejecutarse en un binario aislado que solo disponga de
// la credencial emisor_capacidad. El handler no contiene repositorio de
// efectos ni puede invocar ejecutar_plan_atestado.
func NuevoManejadorHTTPEmisorCapacidadDocumentalV4(
	ctx context.Context,
	pool *pgxpool.Pool,
) (http.Handler, error) {
	configuracion, material, err := cargarMaterialEmisorCapacidadPostgreSQLV4(ctx, pool)
	if err != nil {
		return nil, errorCapacidadDocumentalV4(err)
	}
	servicio, err := nuevoServicioConRelojSistema(configuracion)
	if err != nil || material.validarEn(time.Now().UTC().Truncate(time.Microsecond)) != nil {
		return nil, errorCapacidadDocumentalV4(err)
	}
	return &manejadorHTTPEmisorCapacidadDocumentalV4{
		servicio: servicio, material: material,
	}, nil
}

func (m *manejadorHTTPEmisorCapacidadDocumentalV4) ServeHTTP(
	respuesta http.ResponseWriter,
	peticion *http.Request,
) {
	respuesta.Header().Set("Cache-Control", "no-store")
	respuesta.Header().Set("X-Content-Type-Options", "nosniff")
	if m == nil || m.servicio == nil || peticion == nil ||
		peticion.URL.Path != rutaHTTPEmisorCapacidadDocumentalV4 ||
		peticion.Method != http.MethodPost ||
		!strings.HasPrefix(peticion.Header.Get("Content-Type"), "application/json") ||
		(peticion.ContentLength > maximoSolicitudRemotaCapacidadV4) {
		http.Error(respuesta, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	contenido, err := io.ReadAll(io.LimitReader(
		peticion.Body, maximoSolicitudRemotaCapacidadV4+1,
	))
	if err != nil || int64(len(contenido)) > maximoSolicitudRemotaCapacidadV4 {
		http.Error(respuesta, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	var solicitud solicitudRemotaCapacidadV4
	if decodificarJSONExactoDocumentalV4(contenido, &solicitud) != nil ||
		solicitud.Esquema != esquemaSolicitudRemotaCapacidadV4 {
		http.Error(respuesta, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	artefactos, err := m.emitir(peticion.Context(), solicitud)
	if err != nil {
		http.Error(respuesta, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}
	paquete, err := serializarPaqueteEjecucionDocumentalV4(artefactos)
	if err != nil {
		http.Error(respuesta, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		return
	}
	respuesta.Header().Set("Content-Type", "application/json")
	respuesta.Header().Set("Content-Length", strconv.Itoa(len(paquete)))
	respuesta.WriteHeader(http.StatusOK)
	_, _ = respuesta.Write(paquete)
}

func (m *manejadorHTTPEmisorCapacidadDocumentalV4) emitir(
	ctx context.Context,
	solicitud solicitudRemotaCapacidadV4,
) (artefactosEjecucionDocumentalV4, error) {
	if ctx == nil || m == nil || m.servicio == nil ||
		solicitud.Esquema != esquemaSolicitudRemotaCapacidadV4 {
		return artefactosEjecucionDocumentalV4{}, errorCapacidadDocumentalV4(nil)
	}
	instante, err := m.servicio.capturarInstanteAtestacionPDP(ctx)
	if err != nil || m.material.validarEn(instante) != nil {
		return artefactosEjecucionDocumentalV4{}, errorCapacidadDocumentalV4(err)
	}
	cabecera := solicitud.cabeceraDominio()
	sobre, err := ports.NuevoSobreCriptograficoDocumentalCrudoV4(solicitud.Sobre)
	payload, errPayload := extraerPayloadCOSEDocumentalV4(solicitud.Sobre)
	verificacion, errVerificacion := NuevaSolicitudVerificacionCOSESign1(
		payload, AudienciaCOSEAtestacionAutorizacionPDP,
	)
	proyeccion, errParseo := domain.ParsearMensajeAtestacionAutorizacionV1NoAutoritativo(payload)
	cabeceraFirmada, errCabecera := proyeccion.Cabecera()
	datos, errDatos := proyeccion.Datos()
	if err != nil || errPayload != nil || errVerificacion != nil || errParseo != nil ||
		errCabecera != nil || errDatos != nil || cabecera.Validar() != nil ||
		cabeceraFirmada != cabecera || len(datos.Obligaciones) != 0 ||
		instante.Before(datos.EmitidaEn) || !instante.Before(datos.ValidaHasta) {
		return artefactosEjecucionDocumentalV4{}, errorCapacidadDocumentalV4(nil)
	}
	prueba, err := m.servicio.verificarCOSESign1En(
		ctx, verificacion, sobre, instante,
	)
	if err != nil || validarCabeceraAtestacionPDPContraPrueba(cabecera, prueba) != nil {
		return artefactosEjecucionDocumentalV4{}, errorCapacidadDocumentalV4(err)
	}
	huellaPreimagen := huellaBytesDocumentales(solicitud.Preimagen)
	preimagen, err := ports.InterpretarPreimagenRecursoAutorizacionEjecucionDocumentalV4(
		solicitud.Preimagen, huellaPreimagen,
	)
	recurso, errRecurso := preimagen.RecursoCanonico()
	huellaRecurso, errHuellaRecurso := preimagen.HuellaContextoRecursoSHA256()
	huellaAmbitos, errHuellaAmbitos := preimagen.HuellaAmbitosSHA256()
	plan := recurso.Atributos[ports.AtributoAutorizacionDocumentalHuellaPlanSHA256]
	efectoRef := recurso.Atributos[ports.AtributoAutorizacionDocumentalEfectoRef]
	if err != nil || errRecurso != nil || errHuellaRecurso != nil || errHuellaAmbitos != nil ||
		recurso.Referencia != datos.RecursoRef || recurso.ModuloID != datos.ModuloID ||
		recurso.Tipo != datos.TipoRecurso || huellaRecurso != datos.ContextoRecursoHuellaSHA256 ||
		!huellaSHA256DocumentalValida(plan) || !referenciaDurableAtestacionPDPValida(efectoRef) {
		return artefactosEjecucionDocumentalV4{}, errorCapacidadDocumentalV4(nil)
	}
	decisionCanonica, err := decisionCanonicaDesdeHistoricoV4(datos)
	if err != nil {
		return artefactosEjecucionDocumentalV4{}, errorCapacidadDocumentalV4(err)
	}
	huellaDecision := huellaBytesDocumentales(decisionCanonica)
	huellaCampos := huellaListaRemotaV4(
		"vec.documentos.autorizacion-ejecucion.campos.v4", datos.CamposPermitidos,
	)
	huellaObligaciones := huellaListaRemotaV4(
		"vec.documentos.autorizacion-ejecucion.obligaciones.v4", datos.Obligaciones,
	)
	huellaCumplimientos := huellaMapaRemotoV4(
		"vec.documentos.autorizacion-ejecucion.cumplimientos.v4", map[string]string{},
	)
	v := datos.VinculoAutenticacionActor
	huellaVinculo := huellaCanonicaRemotaV4(
		ports.EsquemaSolicitudVinculadaAutorizacionEjecucionDocumentalV4,
		datos.PrincipalID, datos.PerfilActivoRef, v.AutenticacionRef, v.SesionRef,
		v.ControlSesionRef, strconv.FormatUint(v.ControlSesionRevision, 10),
		v.ControlSesionHuellaSHA256, v.ContextoActorRef,
		strconv.FormatUint(v.ContextoActorVersion, 10), v.ContextoActorHuellaSHA256,
		datos.Accion, recurso.Referencia, recurso.ModuloID, recurso.Tipo,
		huellaRecurso, huellaAmbitos, datos.Finalidad, datos.CorrelacionRef,
		efectoRef, plan, huellaCampos, huellaObligaciones, huellaCumplimientos,
		ports.EsquemaHuellaDecisionAutorizacionReforzadaV1, huellaDecision,
		datos.DecisionRef, instante.Format(time.RFC3339Nano),
		instante.Format(time.RFC3339Nano), datos.ValidaHasta.Format(time.RFC3339Nano),
	)
	huellaAplicacion := huellaCanonicaRemotaV4(
		ports.EsquemaSolicitudAplicacionAutorizacionEjecucionDocumentalV4,
		huellaVinculo, datos.DecisionRef, plan, efectoRef,
		instante.Format(time.RFC3339Nano),
	)
	aplicacion := aplicacionEjecucionDocumentalV4PostgreSQL{
		Esquema:     ports.EsquemaSolicitudAplicacionAutorizacionEjecucionDocumentalV4,
		DecisionRef: datos.DecisionRef, HuellaPlanSHA256: plan, EfectoRef: efectoRef,
		EsquemaHuellaDecision: ports.EsquemaHuellaDecisionAutorizacionReforzadaV1,
		HuellaDecisionSHA256:  huellaDecision, PerfilActivoRef: datos.PerfilActivoRef,
		ContextoActorHuellaSHA256: v.ContextoActorHuellaSHA256, Accion: datos.Accion,
		RecursoRef: recurso.Referencia, ModuloID: recurso.ModuloID, TipoRecurso: recurso.Tipo,
		HuellaRecursoSHA256: huellaRecurso, HuellaAmbitosSHA256: huellaAmbitos,
		Finalidad: datos.Finalidad, CorrelacionRef: datos.CorrelacionRef,
		HuellaCamposPermitidosSHA256: huellaCampos,
		HuellaObligacionesSHA256:     huellaObligaciones,
		HuellaCumplimientosSHA256:    huellaCumplimientos,
		VerificadaEn:                 instante, VinculadaEn: instante, SolicitadaEn: instante,
		ValidaHasta:                     datos.ValidaHasta,
		HuellaSolicitudVinculadaSHA256:  huellaVinculo,
		HuellaSolicitudAplicacionSHA256: huellaAplicacion,
	}
	evidencia := EvidenciaDurableAtestacionAutorizacionPDPV4{
		marca: marcaEvidenciaDurableAtestacionPDPV4,
		metadatos: MetadatosEvidenciaDurableAtestacionPDPV4{
			Esquema:     esquemaEvidenciaDurableAtestacionPDPV4,
			Version:     versionEvidenciaDurableAtestacionPDPV4,
			DecisionRef: datos.DecisionRef, HuellaPlanSHA256: plan, EfectoRef: efectoRef,
			HuellaSolicitudVinculadaSHA256: huellaVinculo,
			FormatoVECADVersion:            cabecera.FormatoVersion, Suite: cabecera.Suite,
			ClaveID: cabecera.ClaveID, AudienciaDespliegue: cabecera.Audiencia,
			AlgoritmoCOSE: prueba.algoritmo, AudienciaCOSE: prueba.audiencia,
			EstadoConfianza: prueba.estadoConfianza, HuellaClaveSHA256: prueba.huellaClaveSHA256,
			HuellaPayloadSHA256: prueba.huellaPayloadSHA256,
			HuellaSobreSHA256:   prueba.huellaSobreSHA256, VerificadaEn: instante,
			RaizValidaDesde: prueba.raizValidaDesde, RaizValidaHasta: prueba.raizValidaHasta,
			RevisionConfianza:            prueba.revisionConfianza,
			HuellaConfiguracionSHA256:    prueba.huellaConfiguracionSHA256,
			ConfiguracionPublicadaEn:     prueba.configuracionPublicadaEn,
			ConfiguracionExpiraEn:        prueba.configuracionExpiraEn,
			HuellaPreimagenRecursoSHA256: huellaPreimagen,
			HuellaContextoRecursoSHA256:  huellaRecurso,
			HuellaAmbitosRecursoSHA256:   huellaAmbitos,
		},
		preimagenRecurso: append([]byte(nil), solicitud.Preimagen...),
		payloadVECAD1:    append([]byte(nil), payload...),
		sobreCOSESign1:   append([]byte(nil), solicitud.Sobre...),
	}
	evidencia.serializacionCanonica = evidencia.calcularSerializacionCanonica()
	evidencia.metadatos.HuellaEvidenciaDurableSHA256 = huellaBytesDocumentales(
		evidencia.serializacionCanonica,
	)
	if evidencia.Validar() != nil {
		return artefactosEjecucionDocumentalV4{}, errorCapacidadDocumentalV4(nil)
	}
	metadatos, err := json.Marshal(metadatosAtestacionEjecucionDocumentalV4PostgreSQL{
		Aplicacion: aplicacion, HuellaPreimagenSHA256: huellaPreimagen,
		FormatoVECADVersion: cabecera.FormatoVersion, Suite: cabecera.Suite,
		ClaveID: cabecera.ClaveID, AudienciaDespliegue: cabecera.Audiencia,
		AlgoritmoCOSE: string(prueba.algoritmo), AudienciaCOSE: string(prueba.audiencia),
		EstadoConfianza: string(prueba.estadoConfianza), HuellaClaveSHA256: prueba.huellaClaveSHA256,
		HuellaPayloadSHA256: prueba.huellaPayloadSHA256,
		HuellaSobreSHA256:   prueba.huellaSobreSHA256, VerificadaEn: instante,
		RaizValidaDesde: prueba.raizValidaDesde, RaizValidaHasta: prueba.raizValidaHasta,
		RevisionConfianza:         prueba.revisionConfianza,
		HuellaConfiguracionSHA256: prueba.huellaConfiguracionSHA256,
		ConfiguracionPublicadaEn:  prueba.configuracionPublicadaEn,
		ConfiguracionExpiraEn:     prueba.configuracionExpiraEn,
		HuellaEvidenciaSHA256:     evidencia.metadatos.HuellaEvidenciaDurableSHA256,
	})
	orden := nuevaOrdenGeneracionDesdeAplicacionRemotaV4(aplicacion)
	efecto, errEfecto := json.Marshal(efectoEjecucionDocumentalV4PostgreSQL{
		Esquema: orden.Esquema, OrdenRef: orden.OrdenRef, Estado: orden.Estado,
		DecisionRef: orden.DecisionRef, EfectoRef: orden.EfectoRef,
		HuellaPlanSHA256: orden.HuellaPlanSHA256, HuellaDecision: orden.HuellaDecision,
		HuellaAplicacion: orden.HuellaAplicacion, HuellaOrdenSHA256: orden.HuellaOrdenSHA256,
		AuditoriaRef: orden.AuditoriaRef, EventoOutboxRef: orden.EventoOutboxRef,
		CorrelacionRef: orden.CorrelacionRef, SolicitadaEn: orden.SolicitadaEn,
	})
	if err != nil || errEfecto != nil {
		return artefactosEjecucionDocumentalV4{}, errorCapacidadDocumentalV4(errors.Join(err, errEfecto))
	}
	artefactos := artefactosEjecucionDocumentalV4{
		metadatos: metadatos, payload: payload, sobre: append([]byte(nil), solicitud.Sobre...),
		evidencia:        append([]byte(nil), evidencia.serializacionCanonica...),
		preimagen:        append([]byte(nil), solicitud.Preimagen...),
		decisionCanonica: decisionCanonica, efecto: efecto,
	}
	capacidad, err := emitirCapacidadEjecucionDocumentalV4(
		instante, artefactos, m.material, prueba, datos.ValidaHasta,
	)
	if err != nil {
		return artefactosEjecucionDocumentalV4{}, errorCapacidadDocumentalV4(err)
	}
	artefactos.capacidad = capacidad
	if artefactos.validarEn(instante) != nil {
		return artefactosEjecucionDocumentalV4{}, errorCapacidadDocumentalV4(nil)
	}
	return artefactos, nil
}

func nuevaOrdenGeneracionDesdeAplicacionRemotaV4(
	p aplicacionEjecucionDocumentalV4PostgreSQL,
) ordenGeneracionDocumentalV4 {
	base := []byte(p.DecisionRef + "\x00" + p.EfectoRef + "\x00" +
		p.HuellaPlanSHA256 + "\x00" + p.HuellaDecisionSHA256 + "\x00" +
		p.HuellaSolicitudAplicacionSHA256)
	return ordenGeneracionDocumentalV4{
		Esquema: esquemaOrdenGeneracionDocumentalV4, OrdenRef: p.EfectoRef,
		Estado: estadoOrdenGeneracionPendienteV4, DecisionRef: p.DecisionRef,
		EfectoRef: p.EfectoRef, HuellaPlanSHA256: p.HuellaPlanSHA256,
		HuellaDecision:   p.HuellaDecisionSHA256,
		HuellaAplicacion: p.HuellaSolicitudAplicacionSHA256,
		HuellaOrdenSHA256: huellaBytesDocumentales(append(
			[]byte(esquemaOrdenGeneracionDocumentalV4+"\x00"), base...,
		)),
		AuditoriaRef: "auditoria:documental:v4:" + huellaBytesDocumentales(
			append([]byte("auditoria\x00"), base...),
		),
		EventoOutboxRef: "evento:documental:v4:" + huellaBytesDocumentales(
			append([]byte("outbox\x00"), base...),
		),
		CorrelacionRef: p.CorrelacionRef, SolicitadaEn: p.SolicitadaEn,
	}
}

func emitirCapacidadEjecucionDocumentalV4(
	instante time.Time,
	artefactos artefactosEjecucionDocumentalV4,
	material materialEmisorCapacidadDocumentalV4,
	prueba PruebaCOSESign1DocumentalVerificada,
	decisionValidaHasta time.Time,
) ([]byte, error) {
	if material.validarEn(instante) != nil || prueba.Validar() != nil {
		return nil, errCapacidadEjecucionDocumentalV4Invalida
	}
	nonce, err := nonceCapacidadDocumentalV4()
	if err != nil {
		return nil, err
	}
	expiraEn := minInstanteCapacidadV4(
		instante.Add(emisionCapacidadPreferidaV4), material.validaHasta,
		prueba.raizValidaHasta, prueba.configuracionExpiraEn, decisionValidaHasta,
	).UTC().Truncate(time.Microsecond)
	if !expiraEn.After(instante) || expiraEn.Sub(instante) > emisionCapacidadMaximaV4 {
		return nil, errCapacidadEjecucionDocumentalV4Invalida
	}
	capacidad := capacidadEjecucionDocumentalV4JSON{
		Esquema: esquemaCapacidadEjecucionDocumentalV4,
		ClaveID: material.claveID, ClaveVersion: material.version, EmisorID: material.emisorID,
		Audiencia: audienciaCapacidadEjecucionDocumentalV4, Nonce: nonce,
		EmitidaEn: instante.Format(time.RFC3339Nano), ExpiraEn: expiraEn.Format(time.RFC3339Nano),
		HuellaMetadatosSHA256:     huellaBytesDocumentales(artefactos.metadatos),
		HuellaPayloadSHA256:       huellaBytesDocumentales(artefactos.payload),
		HuellaSobreSHA256:         huellaBytesDocumentales(artefactos.sobre),
		HuellaEvidenciaSHA256:     huellaBytesDocumentales(artefactos.evidencia),
		HuellaPreimagenSHA256:     huellaBytesDocumentales(artefactos.preimagen),
		HuellaDecisionSHA256:      huellaBytesDocumentales(artefactos.decisionCanonica),
		HuellaEfectoSHA256:        huellaBytesDocumentales(artefactos.efecto),
		RevisionConfianza:         prueba.revisionConfianza,
		HuellaConfiguracionSHA256: prueba.huellaConfiguracionSHA256,
		RaizClaveID:               string(prueba.claveID), HuellaRaizSHA256: prueba.huellaClaveSHA256,
	}
	mac := hmac.New(sha256.New, material.secreto)
	_, _ = mac.Write(preimagenMACCapacidadDocumentalV4(capacidad.valoresAutenticados()))
	capacidad.MACSHA256 = hex.EncodeToString(mac.Sum(nil))
	if capacidad.validarConMaterialEn(instante, artefactos, material) != nil {
		return nil, errCapacidadEjecucionDocumentalV4Invalida
	}
	return json.Marshal(capacidad)
}

func minInstanteCapacidadV4(valores ...time.Time) time.Time {
	var minimo time.Time
	for _, valor := range valores {
		if valor.IsZero() {
			continue
		}
		if minimo.IsZero() || valor.Before(minimo) {
			minimo = valor
		}
	}
	return minimo
}

func errorCapacidadDocumentalV4(err error) error {
	return errors.Join(domain.ErrAutorizacionDenegada, errCapacidadEjecucionDocumentalV4Invalida, err)
}

func (c capacidadEjecucionDocumentalV4JSON) String() string {
	return "[CAPACIDAD-EJECUCION-DOCUMENTAL-V4-REDACTADA]"
}
func (c capacidadEjecucionDocumentalV4JSON) GoString() string { return c.String() }
func (c capacidadEjecucionDocumentalV4JSON) Format(s fmt.State, _ rune) {
	_, _ = io.WriteString(s, c.String())
}
