package main

import (
	"encoding/json"

	"vec-diputacion-granada/internal/vec/auditoria"
)

// La lectura local valida la proyección; la autoridad del encadenado sigue en
// auditoria. La firma del recibo no autentica el origen de este fichero.
func verificarCadenaCheckpoint(b []byte, cobertura auditoria.CoberturaCadena, max uint64) (auditoria.InformeVerificacion, error) {
	var objeto map[string]json.RawMessage
	if decodificar(b, &objeto) != nil || !clavesExactas(objeto, "esquema", "manifiesto", "registros") {
		return auditoria.InformeVerificacion{}, errEntrada
	}
	var esquema string
	var manifiesto map[string]json.RawMessage
	var registros []map[string]json.RawMessage
	if json.Unmarshal(objeto["esquema"], &esquema) != nil ||
		json.Unmarshal(objeto["manifiesto"], &manifiesto) != nil ||
		!clavesExactas(manifiesto, "cadena_id", "primera_secuencia", "ultima_secuencia", "anterior_sha256", "cabeza_sha256", "registros") ||
		json.Unmarshal(objeto["registros"], &registros) != nil {
		return auditoria.InformeVerificacion{}, errEntrada
	}
	historicos := false
	for _, registro := range registros {
		if esquema == auditoria.EsquemaVerificacion {
			if !clavesConsumo(registro, "consumo_confirmado") {
				return auditoria.InformeVerificacion{}, errEntrada
			}
			historicos = true
			continue
		}
		var tipo string
		if json.Unmarshal(registro["tipo_registro"], &tipo) != nil {
			return auditoria.InformeVerificacion{}, errEntrada
		}
		var campos map[string]json.RawMessage
		switch tipo {
		case "consumo_confirmado", auditoria.TipoConsumoOrigenV2, auditoria.TipoConsumoFechaV3:
			if !clavesExactas(registro, "tipo_registro", "consumo") || json.Unmarshal(registro["consumo"], &campos) != nil || !clavesConsumo(campos, tipo) {
				return auditoria.InformeVerificacion{}, errEntrada
			}
			historicos = historicos || tipo != auditoria.TipoConsumoFechaV3
		case "intento_nominal":
			if !clavesExactas(registro, "tipo_registro", "intento") || json.Unmarshal(registro["intento"], &campos) != nil ||
				!clavesExactas(campos, "auditoria_ref", "secuencia", "anterior_sha256", "huella_sha256", "registrada_en", "intento_ref", "intento_material_sha256", "actor_ref", "perfil_activo_ref", "registro_contexto_ref", "contexto_sha256", "procedencia_sha256", "autenticacion_ref", "sesion_ref", "autenticacion_sha256", "accion", "modulo_id", "recurso_ref", "finalidad_ref", "resultado", "motivo_ref", "proceso", "canal", "correlacion_ref", "vinculo_sha256", "contexto_canonico_base64") {
				return auditoria.InformeVerificacion{}, errEntrada
			}
		case "preperfil_autenticado", "bootstrap_operador":
			if esquema != auditoria.EsquemaVerificacionPreperfil {
				return auditoria.InformeVerificacion{}, errEntrada
			}
			clave, extra := "preperfil", []string{"actor_ref"}
			if tipo == "bootstrap_operador" {
				clave, extra = "bootstrap", []string{"operador_login", "plan_sha256", "aprobacion_ref"}
			}
			if !clavesExactas(registro, "tipo_registro", clave) || json.Unmarshal(registro[clave], &campos) != nil || !clavesEvento(campos, extra) {
				return auditoria.InformeVerificacion{}, errEntrada
			}
		default:
			return auditoria.InformeVerificacion{}, errEntrada
		}
	}
	var informe auditoria.InformeVerificacion
	switch esquema {
	case auditoria.EsquemaVerificacion:
		var documento auditoria.DocumentoVerificacion
		if decodificar(b, &documento) != nil {
			return informe, errEntrada
		}
		informe = auditoria.VerificarCadenaV3(documento, cobertura, max)
	case auditoria.EsquemaVerificacionMixta, auditoria.EsquemaVerificacionPreperfil:
		var documento auditoria.DocumentoVerificacionMixta
		if decodificar(b, &documento) != nil {
			return informe, errEntrada
		}
		if esquema == auditoria.EsquemaVerificacionPreperfil {
			informe = auditoria.VerificarCadenaMixtaV3(documento, cobertura, max).InformeVerificacion
		} else {
			informe = auditoria.VerificarCadenaMixtaV2(documento, cobertura, max)
		}
	default:
		return informe, errEntrada
	}
	// El aviso describe la familia histórica de la proyección recibida. Persiste
	// incluso si el cotejo posterior del rango o de otro eslabón se rechaza.
	informe.ConsumosHistoricosSinFechaLigada = historicos
	return informe, nil
}

func clavesConsumo(campos map[string]json.RawMessage, tipo string) bool {
	claves := []string{"auditoria_ref", "secuencia", "decision_ref", "efecto_ref", "huella_efecto_sha256", "anterior_sha256", "huella_sha256", "consumo_huella_sha256"}
	if tipo == auditoria.TipoConsumoOrigenV2 || tipo == auditoria.TipoConsumoFechaV3 {
		claves = append(claves, "tipo_registro", "version_consumo", "proceso", "canal")
	}
	if tipo == auditoria.TipoConsumoFechaV3 {
		claves = append(claves, "registrada_en", "consumida_en", "actor_ref", "perfil_activo_ref", "finalidad_ref")
	}
	return clavesExactas(campos, claves...)
}

func clavesEvento(campos map[string]json.RawMessage, extra []string) bool {
	claves := []string{"auditoria_ref", "secuencia", "anterior_sha256", "huella_sha256", "registrada_en", "evento_ref", "evento_material_sha256", "modulo_id", "accion", "recurso_ref", "resultado", "motivo_ref", "proceso", "canal", "finalidad_ref", "correlacion_ref", "fuente_ref", "fuente_sha256"}
	return clavesExactas(campos, append(claves, extra...)...)
}

func clavesExactas(objeto map[string]json.RawMessage, claves ...string) bool {
	if len(objeto) != len(claves) {
		return false
	}
	for _, clave := range claves {
		if _, existe := objeto[clave]; !existe {
			return false
		}
	}
	return true
}
