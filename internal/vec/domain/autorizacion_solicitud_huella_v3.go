package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"time"
)

// EsquemaHuellaSolicitudAutorizacionV3 identifica el contrato cerrado de la
// solicitud minimizada ligada al actor V2. Su huella acredita integridad, no
// procedencia ni autorizacion.
const EsquemaHuellaSolicitudAutorizacionV3 = "vec.autorizacion.solicitud.v3.efectiva-minimizada.actor-v2"

const formatoInstanteSolicitudAutorizacionV3 = "2006-01-02T15:04:05.000000Z"

type entradaSolicitudAutorizacionCanonicaV3 struct {
	Clave string `json:"clave"`
	Valor string `json:"valor"`
}

type recursoSolicitudAutorizacionCanonicoV3 struct {
	Referencia string                                   `json:"referencia"`
	ModuloID   string                                   `json:"modulo_id"`
	Tipo       string                                   `json:"tipo"`
	Ambitos    []entradaSolicitudAutorizacionCanonicaV3 `json:"ambitos"`
	Atributos  []entradaSolicitudAutorizacionCanonicaV3 `json:"atributos"`
}

type referenciaMotivoAutorizacionCanonicaV3 struct {
	CatalogoID           string `json:"catalogo_id"`
	CatalogoVersion      int    `json:"catalogo_version"`
	CatalogoHuellaSHA256 string `json:"catalogo_huella_sha256"`
	EntradaClave         string `json:"entrada_clave"`
}

// vinculoSolicitudAutorizacionCanonicoV3 congela la preimagen de V3. No debe
// sustituirse por json.Marshal(DatosVinculoAutenticacionActorV2): ese DTO vivo
// puede crecer sin que cambie deliberadamente este esquema historico.
type vinculoSolicitudAutorizacionCanonicoV3 struct {
	Esquema                           string                              `json:"esquema"`
	BloqueVersion                     uint16                              `json:"bloque_version"`
	AutenticacionRef                  string                              `json:"autenticacion_ref"`
	AutenticacionHuellaSHA256         string                              `json:"autenticacion_huella_sha256"`
	AsercionRef                       string                              `json:"asercion_ref"`
	SesionRef                         string                              `json:"sesion_ref"`
	ControlSesionRef                  string                              `json:"control_sesion_ref"`
	ControlSesionRevision             uint64                              `json:"control_sesion_revision"`
	ControlSesionHuellaSHA256         string                              `json:"control_sesion_huella_sha256"`
	CuentaRef                         string                              `json:"cuenta_ref"`
	CuentaOrdinariaRef                string                              `json:"cuenta_ordinaria_ref"`
	PrincipalID                       string                              `json:"principal_id"`
	PerfilActivoRef                   string                              `json:"perfil_activo_ref"`
	CuentaPrivilegiada                bool                                `json:"cuenta_privilegiada"`
	Superficie                        SuperficieAutenticacionActorV1      `json:"superficie"`
	MetodoObservado                   AuthMethod                          `json:"metodo_observado"`
	GarantiaObservada                 AuthAssurance                       `json:"garantia_observada"`
	PoliticaGarantiaRef               string                              `json:"politica_garantia_ref"`
	PoliticaGarantiaHuellaSHA256      string                              `json:"politica_garantia_huella_sha256"`
	AutenticacionVerificadaEn         string                              `json:"autenticacion_verificada_en"`
	SesionEmitidaEn                   string                              `json:"sesion_emitida_en"`
	SesionValidaHasta                 string                              `json:"sesion_valida_hasta"`
	SesionRevalidadaEn                string                              `json:"sesion_revalidada_en"`
	RegistroContextoRef               string                              `json:"registro_contexto_ref"`
	ContextoActorEsquema              string                              `json:"contexto_actor_esquema"`
	ContextoActorRef                  string                              `json:"contexto_actor_ref"`
	ContextoActorVersion              uint64                              `json:"contexto_actor_version"`
	ContextoActorCuentaVersion        uint64                              `json:"contexto_actor_cuenta_version"`
	ContextoActorHuellaSHA256         string                              `json:"contexto_actor_huella_sha256"`
	ManifiestoProcedenciaHuellaSHA256 string                              `json:"manifiesto_procedencia_huella_sha256"`
	AutoridadEfectiva                 AutoridadProcedenciaContextoActorV1 `json:"autoridad_efectiva"`
}

// solicitudAutorizacionCanonicaV3 fija expresamente los campos de solicitud.
// Vinculo contiene el DTO cerrado de V2: su constructor ya excluye PII, roles
// y claims, y se serializa entero para comprometer todos sus datos, incluidos
// los que V2 añade a su bloque de procedencia.
type solicitudAutorizacionCanonicaV3 struct {
	Esquema          string                                 `json:"esquema"`
	Vinculo          vinculoSolicitudAutorizacionCanonicoV3 `json:"vinculo_autenticacion_actor"`
	Accion           string                                 `json:"accion"`
	Recurso          recursoSolicitudAutorizacionCanonicoV3 `json:"recurso"`
	Finalidad        string                                 `json:"finalidad"`
	CorrelacionRef   string                                 `json:"correlacion_ref"`
	ReferenciaMotivo referenciaMotivoAutorizacionCanonicaV3 `json:"referencia_motivo"`
}

func representacionCanonicaSolicitudAutorizacionV3(s SolicitudAutorizacionLigadaV3) ([]byte, error) {
	datos, err := s.Datos()
	if err != nil {
		return nil, ErrSolicitudAutorizacionLigadaV3Invalida
	}
	vinculo, err := datos.VinculoAutenticacionActor.Datos()
	if err != nil {
		return nil, ErrSolicitudAutorizacionLigadaV3Invalida
	}
	correlacion, err := datos.Correlacion.ValorCanonico()
	if err != nil {
		return nil, ErrSolicitudAutorizacionLigadaV3Invalida
	}
	contenido, err := json.Marshal(solicitudAutorizacionCanonicaV3{
		Esquema: EsquemaHuellaSolicitudAutorizacionV3,
		Vinculo: vinculoSolicitudAutorizacionCanonicoV3Desde(vinculo), Accion: datos.Accion,
		Recurso: recursoSolicitudAutorizacionCanonicoV3{
			Referencia: datos.Recurso.Referencia, ModuloID: datos.Recurso.ModuloID, Tipo: datos.Recurso.Tipo,
			Ambitos:   entradasSolicitudAutorizacionCanonicasV3(datos.Recurso.Ambitos),
			Atributos: entradasSolicitudAutorizacionCanonicasV3(datos.Recurso.Atributos),
		},
		Finalidad: datos.Finalidad, CorrelacionRef: correlacion,
		ReferenciaMotivo: referenciaMotivoAutorizacionCanonicaV3{
			CatalogoID: datos.ReferenciaMotivo.CatalogoID, CatalogoVersion: datos.ReferenciaMotivo.CatalogoVersion,
			CatalogoHuellaSHA256: datos.ReferenciaMotivo.CatalogoHuellaSHA256,
			EntradaClave:         datos.ReferenciaMotivo.EntradaClave,
		},
	})
	if err != nil {
		return nil, ErrSolicitudAutorizacionLigadaV3Invalida
	}
	return contenido, nil
}

func vinculoSolicitudAutorizacionCanonicoV3Desde(
	v DatosVinculoAutenticacionActorV2,
) vinculoSolicitudAutorizacionCanonicoV3 {
	instante := func(valor time.Time) string {
		return valor.UTC().Format(formatoInstanteSolicitudAutorizacionV3)
	}
	return vinculoSolicitudAutorizacionCanonicoV3{
		Esquema: v.Esquema, BloqueVersion: v.BloqueVersion,
		AutenticacionRef: v.AutenticacionRef, AutenticacionHuellaSHA256: v.AutenticacionHuellaSHA256,
		AsercionRef: v.AsercionRef, SesionRef: v.SesionRef, ControlSesionRef: v.ControlSesionRef,
		ControlSesionRevision: v.ControlSesionRevision, ControlSesionHuellaSHA256: v.ControlSesionHuellaSHA256,
		CuentaRef: v.CuentaRef, CuentaOrdinariaRef: v.CuentaOrdinariaRef,
		PrincipalID: v.PrincipalID, PerfilActivoRef: v.PerfilActivoRef,
		CuentaPrivilegiada: v.CuentaPrivilegiada, Superficie: v.Superficie,
		MetodoObservado: v.MetodoObservado, GarantiaObservada: v.GarantiaObservada,
		PoliticaGarantiaRef: v.PoliticaGarantiaRef, PoliticaGarantiaHuellaSHA256: v.PoliticaGarantiaHuellaSHA256,
		AutenticacionVerificadaEn: instante(v.AutenticacionVerificadaEn), SesionEmitidaEn: instante(v.SesionEmitidaEn),
		SesionValidaHasta: instante(v.SesionValidaHasta), SesionRevalidadaEn: instante(v.SesionRevalidadaEn),
		RegistroContextoRef: v.RegistroContextoRef, ContextoActorEsquema: v.ContextoActorEsquema,
		ContextoActorRef: v.ContextoActorRef, ContextoActorVersion: v.ContextoActorVersion,
		ContextoActorCuentaVersion:        v.ContextoActorCuentaVersion,
		ContextoActorHuellaSHA256:         v.ContextoActorHuellaSHA256,
		ManifiestoProcedenciaHuellaSHA256: v.ManifiestoProcedenciaHuellaSHA256,
		AutoridadEfectiva:                 v.AutoridadEfectiva,
	}
}

func HuellaSHA256SolicitudAutorizacionV3(s SolicitudAutorizacionLigadaV3) (string, error) {
	contenido, err := representacionCanonicaSolicitudAutorizacionV3(s)
	if err != nil {
		return "", err
	}
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:]), nil
}

func entradasSolicitudAutorizacionCanonicasV3(mapa map[string]string) []entradaSolicitudAutorizacionCanonicaV3 {
	claves := make([]string, 0, len(mapa))
	for clave := range mapa {
		claves = append(claves, clave)
	}
	sort.Strings(claves)
	resultado := make([]entradaSolicitudAutorizacionCanonicaV3, 0, len(claves))
	for _, clave := range claves {
		resultado = append(resultado, entradaSolicitudAutorizacionCanonicaV3{Clave: clave, Valor: mapa[clave]})
	}
	return resultado
}
