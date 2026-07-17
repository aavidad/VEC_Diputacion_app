package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"reflect"
	"strings"
)

const (
	// VersionFormatoAtestacionAutorizacionV2 identifica exclusivamente la
	// representacion binaria VEC-AD-2. No identifica ni aprueba una suite de
	// firma, un proveedor criptografico o un formato de sobre.
	VersionFormatoAtestacionAutorizacionV2 uint16 = 2

	// EsquemaMensajeAtestacionAutorizacionV2 separa VEC-AD-2 de VEC-AD-1 y de
	// cualquier otro uso criptografico. El byte cero posterior forma parte del
	// dominio binario firmado.
	EsquemaMensajeAtestacionAutorizacionV2 = "VEC-AUTORIZACION-ATESTACION-V2-SOLICITUD-LIGADA-MOTIVO-CATALOGADO"

	// TamanoMaximoMensajeAtestacionAutorizacionV2 conserva el mismo presupuesto
	// acotado de 512 KiB que VEC-AD-1. Se declara por separado para que el perfil
	// V2 sea autocontenido aunque reutilice sus primitivas binarias seguras.
	TamanoMaximoMensajeAtestacionAutorizacionV2 = 512 * 1024
)

// CabeceraAtestacionAutorizacionV2 es una cabecera nominal distinta de V1.
// Toda su configuracion debe seleccionarse antes de construir el mensaje. Este
// corte solo fija los bytes canonicos: no implementa firma, COSE ni runtime.
type CabeceraAtestacionAutorizacionV2 struct {
	FormatoVersion uint16
	Suite          string
	ClaveID        string
	Audiencia      string
}

type campoDecisionAtestacionAutorizacionV2 struct {
	nombreGo string
	etiqueta string
}

// camposDecisionAtestacionAutorizacionV2 congela exhaustivamente el modelo
// vivo que VEC-AD-2 sabe representar. Si DecisionAutorizacion crece, cambia de
// orden o altera una etiqueta, el serializador falla cerrado hasta que se
// publique conscientemente un nuevo contrato que cubra el cambio.
var camposDecisionAtestacionAutorizacionV2 = [...]campoDecisionAtestacionAutorizacionV2{
	{"DecisionRef", "decision_ref"},
	{"Concedida", "concedida"},
	{"Codigo", "codigo"},
	{"PrincipalID", "principal_id"},
	{"PerfilActivoRef", "perfil_activo_ref"},
	{"Accion", "accion"},
	{"RecursoRef", "recurso_ref"},
	{"ModuloID", "modulo_id"},
	{"TipoRecurso", "tipo_recurso"},
	{"ContextoRecursoHuellaSHA256", "contexto_recurso_huella_sha256"},
	{"Finalidad", "finalidad"},
	{"CorrelacionRef", "correlacion_ref"},
	{"EsquemaHuellaSolicitud", "esquema_huella_solicitud"},
	{"SolicitudHuellaSHA256", "solicitud_huella_sha256"},
	{"EsquemaHuellaMotivo", "esquema_huella_motivo"},
	{"MotivoHuellaSHA256", "motivo_huella_sha256"},
	{"VinculoAutenticacionActor", "vinculo_autenticacion_actor"},
	{"AsignacionRef", "asignacion_ref"},
	{"AsignacionHuellaSHA256", "asignacion_huella_sha256"},
	{"VersionRolRef", "version_rol_ref"},
	{"VersionRolHuellaSHA256", "version_rol_huella_sha256"},
	{"ControlVigenciaVersionRolRef", "control_vigencia_version_rol_ref"},
	{"ControlVigenciaVersionRolRevision", "control_vigencia_version_rol_revision"},
	{"ControlVigenciaVersionRolHuellaSHA256", "control_vigencia_version_rol_huella_sha256"},
	{"RevisionCatalogoPoliticas", "revision_catalogo_politicas"},
	{"CatalogoPoliticasHuellaSHA256", "catalogo_politicas_huella_sha256"},
	{"PoliticasEvaluadasRefs", "politicas_evaluadas_refs"},
	{"PoliticasEvaluadasHuellasSHA256", "politicas_evaluadas_huellas_sha256"},
	{"PoliticasRefs", "politicas_refs"},
	{"PoliticasHuellasSHA256", "politicas_huellas_sha256"},
	{"GarantiaMinima", "garantia_minima"},
	{"CamposPermitidos", "campos_permitidos"},
	{"Obligaciones", "obligaciones"},
	{"EmitidaEn", "emitida_en"},
	{"ValidaHasta", "valida_hasta"},
}

func (c CabeceraAtestacionAutorizacionV2) Validar() error {
	if c.FormatoVersion != VersionFormatoAtestacionAutorizacionV2 ||
		!identificadorCabeceraAtestacionValido(c.Suite, 128) ||
		!identificadorCabeceraAtestacionValido(c.ClaveID, 512) ||
		!identificadorCabeceraAtestacionValido(c.Audiencia, 512) {
		return errors.Join(ErrConfiguracionAccesoInvalida, ErrMensajeAtestacionAutorizacionInvalido)
	}
	return nil
}

// SerializarMensajeAtestacionAutorizacionV2 produce la unica representacion
// binaria VEC-AD-2 de una concesion ligada a su solicitud y a una entrada
// catalogada. La referencia del motivo se recibe completa para recomputar el
// compromiso que contiene la decision; no se confia en una huella declarada.
//
// VEC-AD-2 conserva el orden contractual VEC-AD-1 de DecisionAutorizacion e
// inserta sus cuatro campos V2 en el lugar que ocupan tras CorrelacionRef. Las
// cuatro coordenadas completas del motivo se escriben despues de la decision.
// Las listas deben llegar ya ordenadas por bytes UTF-8; nunca se corrigen.
func SerializarMensajeAtestacionAutorizacionV2(
	cabecera CabeceraAtestacionAutorizacionV2,
	decision DecisionAutorizacion,
	referenciaMotivo ReferenciaEntradaCatalogo,
) ([]byte, error) {
	if !comprobarEsquemaDecisionAtestacionAutorizacionV2() ||
		!limitesEscritorAtestacionAutorizacionV2Compatibles(
			TamanoMaximoMensajeAtestacionAutorizacionV2,
			TamanoMaximoMensajeAtestacionAutorizacionV1,
		) {
		// El escritor binario reutilizado contiene el limite V1 internamente. La
		// igualdad explicita evita aceptar un contrato V2 con un techo efectivo
		// distinto si cualquiera de las dos versiones evoluciona.
		return nil, errors.Join(ErrConfiguracionAccesoInvalida, ErrMensajeAtestacionAutorizacionInvalido)
	}
	if err := cabecera.Validar(); err != nil {
		return nil, err
	}
	if err := validarDecisionParaAtestacionAutorizacionV2(decision, referenciaMotivo); err != nil {
		return nil, err
	}

	escritor := nuevoEscritorAtestacionAutorizacionV1()
	escritor.escribirBytes([]byte(EsquemaMensajeAtestacionAutorizacionV2))
	escritor.escribirByte(0)
	escritor.escribirUint16(cabecera.FormatoVersion)
	escritor.escribirTexto(cabecera.Suite)
	escritor.escribirTexto(cabecera.ClaveID)
	escritor.escribirTexto(cabecera.Audiencia)

	escribirDecisionAtestacionAutorizacionSolicitudLigadaV2(escritor, decision)
	escribirReferenciaMotivoAtestacionAutorizacionSolicitudLigadaV2(
		escritor,
		referenciaMotivo,
	)
	if escritor.err != nil {
		return nil, escritor.err
	}

	// La longitud final incluye todo el mensaje, incluidos sus propios 8 bytes.
	if escritor.buffer.Len() > TamanoMaximoMensajeAtestacionAutorizacionV2-8 {
		return nil, errorMensajeAtestacionAutorizacionInvalido()
	}
	longitudTotal := uint64(escritor.buffer.Len() + 8)
	escritor.escribirUint64(longitudTotal)
	if escritor.err != nil || escritor.buffer.Len() != int(longitudTotal) ||
		escritor.buffer.Len() > TamanoMaximoMensajeAtestacionAutorizacionV2 {
		return nil, errorMensajeAtestacionAutorizacionInvalido()
	}
	return append([]byte(nil), escritor.buffer.Bytes()...), nil
}

func comprobarEsquemaDecisionAtestacionAutorizacionV2() bool {
	tipo := reflect.TypeOf(DecisionAutorizacion{})
	if tipo.NumField() != len(camposDecisionAtestacionAutorizacionV2) {
		return false
	}
	for indice, esperado := range camposDecisionAtestacionAutorizacionV2 {
		campo := tipo.Field(indice)
		etiqueta, _, _ := strings.Cut(campo.Tag.Get("json"), ",")
		if campo.PkgPath != "" || campo.Name != esperado.nombreGo || etiqueta != esperado.etiqueta {
			return false
		}
	}
	return true
}

func limitesEscritorAtestacionAutorizacionV2Compatibles(limiteV2, limiteEscritorV1 int) bool {
	return limiteV2 == 512*1024 && limiteV2 == limiteEscritorV1
}

// HuellaSHA256MensajeAtestacionAutorizacionV2 publica un vector de integridad
// del mensaje canonico. La huella no constituye firma ni autorizacion.
func HuellaSHA256MensajeAtestacionAutorizacionV2(
	cabecera CabeceraAtestacionAutorizacionV2,
	decision DecisionAutorizacion,
	referenciaMotivo ReferenciaEntradaCatalogo,
) (string, error) {
	mensaje, err := SerializarMensajeAtestacionAutorizacionV2(cabecera, decision, referenciaMotivo)
	if err != nil {
		return "", err
	}
	suma := sha256.Sum256(mensaje)
	return hex.EncodeToString(suma[:]), nil
}

func validarDecisionParaAtestacionAutorizacionV2(
	decision DecisionAutorizacion,
	referenciaMotivo ReferenciaEntradaCatalogo,
) error {
	if err := validarDecisionParaAtestacionAutorizacionSolicitudLigadaV2(
		decision,
		referenciaMotivo,
	); err != nil || !decision.Concedida || decision.Codigo != "concedida" {
		return errors.Join(errorMensajeAtestacionAutorizacionInvalido(), err)
	}
	return nil
}

func validarDecisionParaAtestacionAutorizacionSolicitudLigadaV2(
	decision DecisionAutorizacion,
	referenciaMotivo ReferenciaEntradaCatalogo,
) error {
	// Acota las colecciones antes de cualquier recorrido, copia u ordenacion.
	if len(decision.PoliticasEvaluadasRefs) > maximoElementosAutorizacion ||
		len(decision.PoliticasEvaluadasHuellasSHA256) > maximoElementosAutorizacion ||
		len(decision.PoliticasRefs) > maximoElementosAutorizacion ||
		len(decision.PoliticasHuellasSHA256) > maximoElementosAutorizacion ||
		len(decision.CamposPermitidos) > maximoElementosAutorizacion ||
		len(decision.Obligaciones) > maximoElementosAutorizacion {
		return errorMensajeAtestacionAutorizacionInvalido()
	}

	if decision.ValidarEvidenciaInstantaneaSolicitudLigadaV2() != nil ||
		contieneComodinAtestacionAutorizacion(decision) ||
		!listaAtestacionAutorizacionCanonica(decision.PoliticasEvaluadasRefs) ||
		!listaAtestacionAutorizacionCanonica(decision.PoliticasRefs) ||
		!listaAtestacionAutorizacionCanonica(decision.CamposPermitidos) ||
		!listaAtestacionAutorizacionCanonica(decision.Obligaciones) ||
		!mapaAtestacionAutorizacionV2Canonico(
			decision.PoliticasEvaluadasRefs,
			decision.PoliticasEvaluadasHuellasSHA256,
		) ||
		!mapaAtestacionAutorizacionV2Canonico(
			decision.PoliticasRefs,
			decision.PoliticasHuellasSHA256,
		) ||
		!ReferenciaMotivoAutorizacionV2Valida(referenciaMotivo) {
		return errorMensajeAtestacionAutorizacionInvalido()
	}

	huellaMotivo, err := HuellaSHA256MotivoAutorizacionV2(referenciaMotivo)
	if err != nil || huellaMotivo != decision.MotivoHuellaSHA256 {
		return errors.Join(errorMensajeAtestacionAutorizacionInvalido(), err)
	}
	return nil
}

func escribirDecisionAtestacionAutorizacionSolicitudLigadaV2(
	escritor *escritorAtestacionAutorizacionV1,
	decision DecisionAutorizacion,
) {
	// Orden contractual cerrado de los 35 campos de DecisionAutorizacion.
	escritor.escribirTexto(decision.DecisionRef)
	escritor.escribirBooleano(decision.Concedida)
	escritor.escribirTexto(decision.Codigo)
	escritor.escribirTexto(decision.PrincipalID)
	escritor.escribirTexto(decision.PerfilActivoRef)
	escritor.escribirTexto(decision.Accion)
	escritor.escribirTexto(decision.RecursoRef)
	escritor.escribirTexto(decision.ModuloID)
	escritor.escribirTexto(decision.TipoRecurso)
	escritor.escribirTexto(decision.ContextoRecursoHuellaSHA256)
	escritor.escribirTexto(decision.Finalidad)
	escritor.escribirTexto(decision.CorrelacionRef)
	escritor.escribirTexto(decision.EsquemaHuellaSolicitud)
	escritor.escribirTexto(decision.SolicitudHuellaSHA256)
	escritor.escribirTexto(decision.EsquemaHuellaMotivo)
	escritor.escribirTexto(decision.MotivoHuellaSHA256)
	escribirVinculoAutenticacionActorV1(escritor, decision.VinculoAutenticacionActor)
	escritor.escribirTexto(decision.AsignacionRef)
	escritor.escribirTexto(decision.AsignacionHuellaSHA256)
	escritor.escribirTexto(decision.VersionRolRef)
	escritor.escribirTexto(decision.VersionRolHuellaSHA256)
	escritor.escribirTexto(decision.ControlVigenciaVersionRolRef)
	escritor.escribirUint64(decision.ControlVigenciaVersionRolRevision)
	escritor.escribirTexto(decision.ControlVigenciaVersionRolHuellaSHA256)
	escritor.escribirUint64(decision.RevisionCatalogoPoliticas)
	escritor.escribirTexto(decision.CatalogoPoliticasHuellaSHA256)
	escritor.escribirLista(decision.PoliticasEvaluadasRefs)
	escritor.escribirMapa(decision.PoliticasEvaluadasHuellasSHA256)
	escritor.escribirLista(decision.PoliticasRefs)
	escritor.escribirMapa(decision.PoliticasHuellasSHA256)
	escritor.escribirTexto(string(decision.GarantiaMinima))
	escritor.escribirLista(decision.CamposPermitidos)
	escritor.escribirLista(decision.Obligaciones)
	escritor.escribirInstante(decision.EmitidaEn)
	escritor.escribirInstante(decision.ValidaHasta)
}

func escribirReferenciaMotivoAtestacionAutorizacionSolicitudLigadaV2(
	escritor *escritorAtestacionAutorizacionV1,
	referenciaMotivo ReferenciaEntradaCatalogo,
) {
	// Coordenadas completas e inmutables de la entrada de motivo publicada.
	escritor.escribirTexto(referenciaMotivo.CatalogoID)
	// uint64 evita que el formato binario dependa del ancho de int del proceso.
	// El perfil de motivos acota actualmente la version al intervalo portable
	// de PostgreSQL, pero el formato no debe truncarla si ese perfil evoluciona.
	escritor.escribirUint64(uint64(referenciaMotivo.CatalogoVersion))
	escritor.escribirTexto(referenciaMotivo.CatalogoHuellaSHA256)
	escritor.escribirTexto(referenciaMotivo.EntradaClave)
}

func mapaAtestacionAutorizacionV2Canonico(
	referencias []string,
	huellas map[string]string,
) bool {
	if len(referencias) != len(huellas) || len(referencias) > maximoElementosAutorizacion {
		return false
	}
	for _, referencia := range referencias {
		huella, existe := huellas[referencia]
		if !existe || !huellaSHA256AutorizacionValida(huella) {
			return false
		}
	}
	return true
}
