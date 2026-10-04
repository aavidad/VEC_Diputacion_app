package postgres

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

func validarExportacion(m ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, actor domain.ContextoActor, p *peticion, ahora time.Time) (string, string, error) {
	fallo := ports.ErrLecturaUsuariosAdministrablesNoDisponible
	if m.ValidarEstructura() != nil || m.PersonaVersion() != actor.Instantanea.PersonaVersion || m.PerfilVersion() != actor.Instantanea.PerfilVersion {
		return "", "", fallo
	}
	huella, err := p.recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return "", "", fallo
	}
	r := m.ResumenCapacidad()
	motivoSHA := sha256.Sum256(m.MotivoCanonico())
	contextoSHA := sha256.Sum256(m.ContextoActorCanonico())
	if r.Operacion() != p.accion || r.AudienciaConsumo() != p.audiencia || r.EfectoRef() != p.recurso.Referencia || r.EfectoHuellaSHA256() != huella ||
		r.MotivoHuellaSHA256() != hex.EncodeToString(motivoSHA[:]) || r.ContextoHuellaSHA256() != hex.EncodeToString(contextoSHA[:]) ||
		ahora.Before(r.EmitidaEn()) || !ahora.Before(r.ExpiraEn()) {
		return "", "", fallo
	}
	var c, d map[string]json.RawMessage
	capacidad, decision := m.CapacidadCanonica(), m.DecisionCanonica()
	if json.Unmarshal(capacidad, &c) != nil || json.Unmarshal(decision, &d) != nil {
		return "", "", fallo
	}
	textoCampo := func(obj map[string]json.RawMessage, clave string) string {
		var v string
		_ = json.Unmarshal(obj[clave], &v)
		return v
	}
	boolCampo := func(obj map[string]json.RawMessage, clave string) bool {
		var v bool
		_ = json.Unmarshal(obj[clave], &v)
		return v
	}
	h := sha256.Sum256(decision)
	if textoCampo(c, "operacion") != p.accion || textoCampo(c, "audiencia_consumo") != p.audiencia || textoCampo(c, "efecto_ref") != p.recurso.Referencia ||
		textoCampo(c, "huella_efecto_sha256") != huella || textoCampo(c, "decision_ref") != r.DecisionRef() ||
		textoCampo(c, "huella_decision_sha256") != hex.EncodeToString(h[:]) || r.DecisionHuellaSHA256() != hex.EncodeToString(h[:]) ||
		!boolCampo(d, "concedida") || textoCampo(d, "decision_ref") != r.DecisionRef() || textoCampo(d, "principal_id") != actor.PersonaRef ||
		textoCampo(d, "perfil_activo_ref") != actor.PerfilActivoRef || textoCampo(d, "correlacion_ref") != p.correlacion ||
		textoCampo(d, "accion") != p.accion || textoCampo(d, "modulo_id") != "administracion" || textoCampo(d, "tipo_recurso") != p.recurso.Tipo ||
		textoCampo(d, "recurso_ref") != p.recurso.Referencia || textoCampo(d, "contexto_recurso_huella_sha256") != huella ||
		textoCampo(d, "finalidad") != "gestion_usuarios" || textoCampo(d, "garantia_minima") != "alto" ||
		textoCampo(d, "version_rol_ref") != "rol:administracion_perfiles:v5" {
		return "", "", fallo
	}
	var vinculo map[string]json.RawMessage
	if json.Unmarshal(d["vinculo_autenticacion_actor"], &vinculo) != nil ||
		textoCampo(vinculo, "superficie") != "administracion_privilegiada" || !boolCampo(vinculo, "cuenta_privilegiada") {
		return "", "", fallo
	}
	var campos, obligaciones []string
	if json.Unmarshal(d["campos_permitidos"], &campos) != nil || json.Unmarshal(d["obligaciones"], &obligaciones) != nil {
		return "", "", fallo
	}
	esperados := []string{"denominacion_version", "perfiles", "persona_ref", "unidad_ref"}
	if p.accion == accionListar {
		esperados = []string{"denominacion_version", "perfiles", "persona_ref", "siguiente_cursor", "unidad_ref"}
	}
	if !reflect.DeepEqual(campos, esperados) || !reflect.DeepEqual(obligaciones, []string{"auditar"}) {
		return "", "", fallo
	}
	return r.DecisionRef(), huella, nil
}

func validarRespuesta(bruto []byte, p *peticion, consulta string) error {
	if consulta == listarSQL {
		_, err := listaRespuesta(bruto, p.ambito, p.filtros, p.decisionRef, p.recurso.Referencia, p.contextoSHA)
		return err
	}
	if consulta == consultarSQL {
		_, err := fichaRespuesta(bruto, p.ambito, p.recurso.Referencia, p.decisionRef, p.recurso.Referencia, p.contextoSHA)
		return err
	}
	return ports.ErrLecturaUsuariosAdministrablesNoDisponible
}
