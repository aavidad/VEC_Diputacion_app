package postgres

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"time"
	"unicode/utf8"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

const (
	esquemaExpedienteFlujoFirmaPostgreSQLV1 = "vec.bolsa.firma.expediente-postgresql.v1"
	maximoDocumentoFlujoFirmaPostgreSQL     = 256 * 1024
	maximaProfundidadJSONFlujoFirma         = 16
)

type documentoExpedienteFlujoFirmaPostgreSQLV1 struct {
	Esquema                string                              `json:"esquema"`
	FlujoRef               string                              `json:"flujo_ref"`
	Version                string                              `json:"version"`
	IndiceIdempotenciaHMAC string                              `json:"indice_idempotencia_hmac"`
	HuellaSolicitudHMAC    string                              `json:"huella_solicitud_hmac"`
	VinculoActorHMAC       string                              `json:"vinculo_actor_hmac"`
	PerfilActorClave       string                              `json:"perfil_actor_clave"`
	ProcesoRef             string                              `json:"proceso_ref"`
	SolicitudRef           string                              `json:"solicitud_ref"`
	BaremacionMeritoRef    string                              `json:"baremacion_merito_ref"`
	DecisionRef            string                              `json:"decision_ref"`
	Estado                 string                              `json:"estado"`
	EstadoProtegido        estadoProtegidoFlujoFirmaPostgreSQL `json:"estado_protegido"`
	PuntosControl          []puntoControlFlujoFirmaPostgreSQL  `json:"puntos_control"`
	ProyeccionLanzamiento  *proyeccionFlujoFirmaPostgreSQL     `json:"proyeccion_lanzamiento,omitempty"`
	Resultado              *resultadoFlujoFirmaPostgreSQL      `json:"resultado,omitempty"`
	CreadoEn               string                              `json:"creado_en"`
	ActualizadoEn          string                              `json:"actualizado_en"`
	SelloEstadoHMAC        string                              `json:"sello_estado_hmac"`
}

type estadoProtegidoFlujoFirmaPostgreSQL struct {
	Esquema      string `json:"esquema"`
	Algoritmo    string `json:"algoritmo"`
	ClaveRef     string `json:"clave_ref"`
	NonceHex     string `json:"nonce_hex"`
	HuellaSHA256 string `json:"huella_sha256"`
}

type puntoControlFlujoFirmaPostgreSQL struct {
	Paso                  string `json:"paso"`
	Estado                string `json:"estado"`
	EfectoRef             string `json:"efecto_ref"`
	ClaveIdempotenciaHMAC string `json:"clave_idempotencia_hmac"`
	ResultadoRef          string `json:"resultado_ref,omitempty"`
	HuellaResultadoSHA256 string `json:"huella_resultado_sha256,omitempty"`
	DeclaradoEn           string `json:"declarado_en"`
	CompletadoEn          string `json:"completado_en,omitempty"`
}

type proyeccionFlujoFirmaPostgreSQL struct {
	FlujoRef              string `json:"flujo_ref"`
	SesionFirmaRef        string `json:"sesion_firma_ref"`
	LanzamientoRef        string `json:"lanzamiento_ref"`
	CanalLanzamientoClave string `json:"canal_lanzamiento_clave"`
	PreparadaEn           string `json:"preparada_en"`
	ExpiraEn              string `json:"expira_en"`
}

type resultadoFlujoFirmaPostgreSQL struct {
	FlujoRef                     string `json:"flujo_ref"`
	DecisionRef                  string `json:"decision_ref"`
	DocumentoFirmadoRef          string `json:"documento_firmado_ref"`
	HuellaDocumentoFirmadoSHA256 string `json:"huella_documento_firmado_sha256"`
	VersionBaremacion            string `json:"version_baremacion"`
	EvidenciaConfirmacionRef     string `json:"evidencia_confirmacion_ref"`
	HuellaResultadoSHA256        string `json:"huella_resultado_sha256"`
	CompletadoEn                 string `json:"completado_en"`
}

func serializarExpedienteFlujoFirmaPostgreSQL(
	expediente puertosbolsa.ExpedienteFlujoFirmaBaremacion,
) ([]byte, []byte, error) {
	if expediente.Validar() != nil {
		return nil, nil, puertosbolsa.ErrSolicitudFlujoFirmaBaremacionInvalida
	}
	estado, err := expediente.EstadoProtegido.DatosPersistencia()
	if err != nil {
		return nil, nil, puertosbolsa.ErrEstadoFlujoFirmaAlterado
	}
	documento := documentoExpedienteFlujoFirmaPostgreSQLV1{
		Esquema:  esquemaExpedienteFlujoFirmaPostgreSQLV1,
		FlujoRef: expediente.FlujoRef, Version: strconv.FormatUint(expediente.Version, 10),
		IndiceIdempotenciaHMAC: expediente.IndiceIdempotenciaHMAC,
		HuellaSolicitudHMAC:    expediente.HuellaSolicitudHMAC,
		VinculoActorHMAC:       expediente.VinculoActorHMAC,
		PerfilActorClave:       expediente.PerfilActorClave,
		ProcesoRef:             expediente.ProcesoRef,
		SolicitudRef:           expediente.SolicitudRef,
		BaremacionMeritoRef:    expediente.BaremacionMeritoRef,
		DecisionRef:            expediente.DecisionRef,
		Estado:                 string(expediente.Estado),
		EstadoProtegido: estadoProtegidoFlujoFirmaPostgreSQL{
			Esquema: estado.Esquema, Algoritmo: estado.Algoritmo,
			ClaveRef: estado.ClaveRef, NonceHex: hex.EncodeToString(estado.Nonce),
			HuellaSHA256: estado.HuellaSHA256,
		},
		PuntosControl: make(
			[]puntoControlFlujoFirmaPostgreSQL,
			len(expediente.PuntosControl),
		),
		CreadoEn:        instanteFlujoFirmaPostgreSQL(expediente.CreadoEn),
		ActualizadoEn:   instanteFlujoFirmaPostgreSQL(expediente.ActualizadoEn),
		SelloEstadoHMAC: expediente.SelloEstadoHMAC,
	}
	for indice, punto := range expediente.PuntosControl {
		documento.PuntosControl[indice] = puntoControlFlujoFirmaPostgreSQL{
			Paso: string(punto.Paso), Estado: string(punto.Estado),
			EfectoRef: punto.EfectoRef, ClaveIdempotenciaHMAC: punto.ClaveIdempotenciaHMAC,
			ResultadoRef: punto.ResultadoRef, HuellaResultadoSHA256: punto.HuellaResultadoSHA256,
			DeclaradoEn:  instanteFlujoFirmaPostgreSQL(punto.DeclaradoEn),
			CompletadoEn: instanteOpcionalFlujoFirmaPostgreSQL(punto.CompletadoEn),
		}
	}
	if expediente.ProyeccionLanzamiento != nil {
		proyeccion := expediente.ProyeccionLanzamiento
		documento.ProyeccionLanzamiento = &proyeccionFlujoFirmaPostgreSQL{
			FlujoRef: proyeccion.FlujoRef, SesionFirmaRef: proyeccion.SesionFirmaRef,
			LanzamientoRef:        proyeccion.LanzamientoRef,
			CanalLanzamientoClave: proyeccion.CanalLanzamientoClave,
			PreparadaEn:           instanteFlujoFirmaPostgreSQL(proyeccion.PreparadaEn),
			ExpiraEn:              instanteFlujoFirmaPostgreSQL(proyeccion.ExpiraEn),
		}
	}
	if expediente.Resultado != nil {
		resultado := expediente.Resultado
		documento.Resultado = &resultadoFlujoFirmaPostgreSQL{
			FlujoRef: resultado.FlujoRef, DecisionRef: resultado.DecisionRef,
			DocumentoFirmadoRef:          resultado.DocumentoFirmadoRef,
			HuellaDocumentoFirmadoSHA256: resultado.HuellaDocumentoFirmadoSHA256,
			VersionBaremacion:            strconv.FormatUint(resultado.VersionBaremacion, 10),
			EvidenciaConfirmacionRef:     resultado.EvidenciaConfirmacionRef,
			HuellaResultadoSHA256:        resultado.HuellaResultadoSHA256,
			CompletadoEn:                 instanteFlujoFirmaPostgreSQL(resultado.CompletadoEn),
		}
	}
	contenido, err := json.Marshal(documento)
	if err != nil || len(contenido) == 0 ||
		len(contenido) > maximoDocumentoFlujoFirmaPostgreSQL {
		return nil, nil, puertosbolsa.ErrSolicitudFlujoFirmaBaremacionInvalida
	}
	return contenido, append([]byte(nil), estado.Cifrado...), nil
}

func decodificarExpedienteFlujoFirmaPostgreSQL(
	contenido, cifrado []byte,
) (puertosbolsa.ExpedienteFlujoFirmaBaremacion, error) {
	if len(contenido) == 0 || len(contenido) > maximoDocumentoFlujoFirmaPostgreSQL ||
		!utf8.Valid(contenido) || len(cifrado) < 16 ||
		validarJSONFlujoFirmaPostgreSQLNoAmbiguo(contenido) != nil {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{},
			puertosbolsa.ErrEstadoFlujoFirmaAlterado
	}
	var documento documentoExpedienteFlujoFirmaPostgreSQLV1
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(&documento); err != nil ||
		documento.Esquema != esquemaExpedienteFlujoFirmaPostgreSQLV1 {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{},
			puertosbolsa.ErrEstadoFlujoFirmaAlterado
	}
	version, err := enteroCanonicoFlujoFirmaPostgreSQL(documento.Version)
	if err != nil {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, err
	}
	nonce, err := hex.DecodeString(documento.EstadoProtegido.NonceHex)
	if err != nil {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{},
			puertosbolsa.ErrEstadoFlujoFirmaAlterado
	}
	estado, err := puertosbolsa.ImportarEstadoProtegidoFlujoFirmaBaremacion(
		puertosbolsa.DatosPersistenciaEstadoProtegidoFlujoFirmaBaremacion{
			Esquema:   documento.EstadoProtegido.Esquema,
			Algoritmo: documento.EstadoProtegido.Algoritmo,
			ClaveRef:  documento.EstadoProtegido.ClaveRef,
			Nonce:     nonce, Cifrado: append([]byte(nil), cifrado...),
			HuellaSHA256: documento.EstadoProtegido.HuellaSHA256,
		},
	)
	if err != nil {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, err
	}
	expediente := puertosbolsa.ExpedienteFlujoFirmaBaremacion{
		FlujoRef: documento.FlujoRef, Version: version,
		IndiceIdempotenciaHMAC: documento.IndiceIdempotenciaHMAC,
		HuellaSolicitudHMAC:    documento.HuellaSolicitudHMAC,
		VinculoActorHMAC:       documento.VinculoActorHMAC,
		PerfilActorClave:       documento.PerfilActorClave,
		ProcesoRef:             documento.ProcesoRef, SolicitudRef: documento.SolicitudRef,
		BaremacionMeritoRef: documento.BaremacionMeritoRef, DecisionRef: documento.DecisionRef,
		Estado:          puertosbolsa.EstadoExpedienteFlujoFirmaBaremacion(documento.Estado),
		EstadoProtegido: estado,
		PuntosControl: make(
			[]puertosbolsa.PuntoControlFirmaBaremacion,
			len(documento.PuntosControl),
		),
		SelloEstadoHMAC: documento.SelloEstadoHMAC,
	}
	expediente.CreadoEn, err = leerInstanteFlujoFirmaPostgreSQL(documento.CreadoEn, false)
	if err == nil {
		expediente.ActualizadoEn, err = leerInstanteFlujoFirmaPostgreSQL(documento.ActualizadoEn, false)
	}
	if err != nil {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, err
	}
	for indice, punto := range documento.PuntosControl {
		expediente.PuntosControl[indice], err = decodificarPuntoFlujoFirmaPostgreSQL(punto)
		if err != nil {
			return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, err
		}
	}
	if documento.ProyeccionLanzamiento != nil {
		expediente.ProyeccionLanzamiento, err =
			decodificarProyeccionFlujoFirmaPostgreSQL(*documento.ProyeccionLanzamiento)
		if err != nil {
			return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, err
		}
	}
	if documento.Resultado != nil {
		expediente.Resultado, err = decodificarResultadoFlujoFirmaPostgreSQL(*documento.Resultado)
		if err != nil {
			return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, err
		}
	}
	if expediente.Validar() != nil {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{},
			puertosbolsa.ErrEstadoFlujoFirmaAlterado
	}
	return expediente, nil
}

func decodificarPuntoFlujoFirmaPostgreSQL(
	documento puntoControlFlujoFirmaPostgreSQL,
) (puertosbolsa.PuntoControlFirmaBaremacion, error) {
	declarado, err := leerInstanteFlujoFirmaPostgreSQL(documento.DeclaradoEn, false)
	if err != nil {
		return puertosbolsa.PuntoControlFirmaBaremacion{}, err
	}
	completado, err := leerInstanteFlujoFirmaPostgreSQL(documento.CompletadoEn, true)
	if err != nil {
		return puertosbolsa.PuntoControlFirmaBaremacion{}, err
	}
	return puertosbolsa.PuntoControlFirmaBaremacion{
		Paso:      puertosbolsa.PasoFlujoFirmaBaremacion(documento.Paso),
		Estado:    puertosbolsa.EstadoPuntoControlFirmaBaremacion(documento.Estado),
		EfectoRef: documento.EfectoRef, ClaveIdempotenciaHMAC: documento.ClaveIdempotenciaHMAC,
		ResultadoRef: documento.ResultadoRef, HuellaResultadoSHA256: documento.HuellaResultadoSHA256,
		DeclaradoEn: declarado, CompletadoEn: completado,
	}, nil
}

func decodificarProyeccionFlujoFirmaPostgreSQL(
	documento proyeccionFlujoFirmaPostgreSQL,
) (*puertosbolsa.ProyeccionLanzamientoFirmaBaremacion, error) {
	preparada, err := leerInstanteFlujoFirmaPostgreSQL(documento.PreparadaEn, false)
	if err != nil {
		return nil, err
	}
	expira, err := leerInstanteFlujoFirmaPostgreSQL(documento.ExpiraEn, false)
	if err != nil {
		return nil, err
	}
	return &puertosbolsa.ProyeccionLanzamientoFirmaBaremacion{
		FlujoRef: documento.FlujoRef, SesionFirmaRef: documento.SesionFirmaRef,
		LanzamientoRef:        documento.LanzamientoRef,
		CanalLanzamientoClave: documento.CanalLanzamientoClave,
		PreparadaEn:           preparada, ExpiraEn: expira,
	}, nil
}

func decodificarResultadoFlujoFirmaPostgreSQL(
	documento resultadoFlujoFirmaPostgreSQL,
) (*puertosbolsa.ResultadoFinalFlujoFirmaBaremacion, error) {
	version, err := enteroCanonicoFlujoFirmaPostgreSQL(documento.VersionBaremacion)
	if err != nil {
		return nil, err
	}
	completado, err := leerInstanteFlujoFirmaPostgreSQL(documento.CompletadoEn, false)
	if err != nil {
		return nil, err
	}
	return &puertosbolsa.ResultadoFinalFlujoFirmaBaremacion{
		FlujoRef: documento.FlujoRef, DecisionRef: documento.DecisionRef,
		DocumentoFirmadoRef:          documento.DocumentoFirmadoRef,
		HuellaDocumentoFirmadoSHA256: documento.HuellaDocumentoFirmadoSHA256,
		VersionBaremacion:            version, EvidenciaConfirmacionRef: documento.EvidenciaConfirmacionRef,
		HuellaResultadoSHA256: documento.HuellaResultadoSHA256, CompletadoEn: completado,
	}, nil
}

func instanteFlujoFirmaPostgreSQL(instante time.Time) string {
	return instante.UTC().Format(time.RFC3339Nano)
}

func instanteOpcionalFlujoFirmaPostgreSQL(instante time.Time) string {
	if instante.IsZero() {
		return ""
	}
	return instanteFlujoFirmaPostgreSQL(instante)
}

func leerInstanteFlujoFirmaPostgreSQL(valor string, permiteVacio bool) (time.Time, error) {
	if permiteVacio && valor == "" {
		return time.Time{}, nil
	}
	instante, err := time.Parse(time.RFC3339Nano, valor)
	if err != nil || instante.Location() != time.UTC ||
		instanteFlujoFirmaPostgreSQL(instante) != valor {
		return time.Time{}, puertosbolsa.ErrEstadoFlujoFirmaAlterado
	}
	return instante, nil
}

func enteroCanonicoFlujoFirmaPostgreSQL(valor string) (uint64, error) {
	numero, err := strconv.ParseUint(valor, 10, 64)
	if err != nil || numero < 1 || strconv.FormatUint(numero, 10) != valor {
		return 0, puertosbolsa.ErrEstadoFlujoFirmaAlterado
	}
	return numero, nil
}

func validarJSONFlujoFirmaPostgreSQLNoAmbiguo(contenido []byte) error {
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.UseNumber()
	if err := consumirValorJSONFlujoFirma(decodificador, 0); err != nil {
		return err
	}
	if _, err := decodificador.Token(); !errors.Is(err, io.EOF) {
		return puertosbolsa.ErrEstadoFlujoFirmaAlterado
	}
	return nil
}

func consumirValorJSONFlujoFirma(decodificador *json.Decoder, profundidad int) error {
	if profundidad > maximaProfundidadJSONFlujoFirma {
		return puertosbolsa.ErrEstadoFlujoFirmaAlterado
	}
	token, err := decodificador.Token()
	if err != nil {
		return err
	}
	delimitador, compuesto := token.(json.Delim)
	if !compuesto {
		return nil
	}
	switch delimitador {
	case '{':
		claves := make(map[string]struct{})
		for decodificador.More() {
			tokenClave, err := decodificador.Token()
			clave, valida := tokenClave.(string)
			if err != nil || !valida {
				return puertosbolsa.ErrEstadoFlujoFirmaAlterado
			}
			if _, duplicada := claves[clave]; duplicada {
				return puertosbolsa.ErrEstadoFlujoFirmaAlterado
			}
			claves[clave] = struct{}{}
			if err := consumirValorJSONFlujoFirma(decodificador, profundidad+1); err != nil {
				return err
			}
		}
		cierre, err := decodificador.Token()
		if err != nil || cierre != json.Delim('}') {
			return puertosbolsa.ErrEstadoFlujoFirmaAlterado
		}
	case '[':
		for decodificador.More() {
			if err := consumirValorJSONFlujoFirma(decodificador, profundidad+1); err != nil {
				return err
			}
		}
		cierre, err := decodificador.Token()
		if err != nil || cierre != json.Delim(']') {
			return puertosbolsa.ErrEstadoFlujoFirmaAlterado
		}
	default:
		return puertosbolsa.ErrEstadoFlujoFirmaAlterado
	}
	return nil
}
