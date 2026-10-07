package postgres

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

func validarExportacion(m ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, actor domain.ContextoActor, evidencia domain.EvidenciaSesionAdministracionPerfiles, p *peticion, ahora time.Time) (string, string, error) {
	fallo := ports.ErrLecturaUsuariosAdministrablesNoDisponible
	huellaActor, err := actor.HuellaSHA256VinculadaV2()
	if err != nil {
		return "", "", falloValidacionRedactado(err)
	}
	if errEvidencia := evidencia.ValidarPara(actor); errEvidencia != nil {
		return "", "", falloValidacionRedactado(errEvidencia)
	}
	if huellaActor != evidencia.ResultadoContexto.HuellaSHA256 ||
		m.ValidarEstructura() != nil || m.PersonaVersion() != actor.Instantanea.PersonaVersion || m.PerfilVersion() != actor.Instantanea.PerfilVersion ||
		!bytes.Equal(m.ContextoActorCanonico(), evidencia.ResultadoContexto.RepresentacionCanonica) {
		return "", "", fallo
	}
	huella, err := p.recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return "", "", falloValidacionRedactado(err)
	}
	r := m.ResumenCapacidad()
	motivoSHA := sha256.Sum256(m.MotivoCanonico())
	contextoSHA := sha256.Sum256(m.ContextoActorCanonico())
	if r.Operacion() != p.accion || r.AudienciaConsumo() != p.audiencia || r.EfectoRef() != p.recurso.Referencia || r.EfectoHuellaSHA256() != huella ||
		r.MotivoHuellaSHA256() != hex.EncodeToString(motivoSHA[:]) || r.ContextoRef() != evidencia.ResultadoContexto.RegistroContextoRef ||
		r.ContextoHuellaSHA256() != evidencia.ResultadoContexto.HuellaSHA256 || r.ContextoHuellaSHA256() != hex.EncodeToString(contextoSHA[:]) ||
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
		textoCampo(c, "contexto_ref") != evidencia.ResultadoContexto.RegistroContextoRef || textoCampo(c, "huella_contexto_sha256") != evidencia.ResultadoContexto.HuellaSHA256 ||
		textoCampo(c, "huella_efecto_sha256") != huella || textoCampo(c, "decision_ref") != r.DecisionRef() ||
		textoCampo(c, "huella_decision_sha256") != hex.EncodeToString(h[:]) || r.DecisionHuellaSHA256() != hex.EncodeToString(h[:]) ||
		!boolCampo(d, "concedida") || textoCampo(d, "decision_ref") != r.DecisionRef() || textoCampo(d, "principal_id") != actor.PersonaRef ||
		textoCampo(d, "perfil_activo_ref") != actor.PerfilActivoRef || textoCampo(d, "correlacion_ref") != p.correlacion ||
		textoCampo(d, "accion") != p.accion || textoCampo(d, "modulo_id") != "administracion" || textoCampo(d, "tipo_recurso") != p.recurso.Tipo ||
		textoCampo(d, "recurso_ref") != p.recurso.Referencia || textoCampo(d, "contexto_recurso_huella_sha256") != huella ||
		textoCampo(d, "finalidad") != "gestion_usuarios" || textoCampo(d, "garantia_minima") != "alto" ||
		!versionRolUsuariosAdmitida(textoCampo(d, "version_rol_ref")) {
		return "", "", fallo
	}
	var vinculo map[string]json.RawMessage
	if errVinculo := json.Unmarshal(d["vinculo_autenticacion_actor"], &vinculo); errVinculo != nil {
		return "", "", falloValidacionRedactado(errVinculo)
	}
	if textoCampo(vinculo, "superficie") != "administracion_privilegiada" || !boolCampo(vinculo, "cuenta_privilegiada") {
		return "", "", fallo
	}
	if errVinculo := vinculoDecisionOriginal(d["vinculo_autenticacion_actor"], evidencia.Vinculo); errVinculo != nil {
		return "", "", falloValidacionRedactado(errVinculo)
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

func vinculoDecisionOriginal(bruto json.RawMessage, original domain.VinculoAutenticacionActorV2) error {
	datos, err := original.Datos()
	if err != nil {
		return err
	}
	var recibido domain.DatosVinculoAutenticacionActorV2
	if err := json.Unmarshal(bruto, &recibido); err != nil {
		return err
	}
	if err := recibido.Validar(); err != nil {
		return err
	}
	var clavesOriginal, clavesRecibidas map[string]json.RawMessage
	originalJSON, err := json.Marshal(datos)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(originalJSON, &clavesOriginal); err != nil {
		return err
	}
	if err := json.Unmarshal(bruto, &clavesRecibidas); err != nil {
		return err
	}
	if len(clavesOriginal) != len(clavesRecibidas) {
		return ports.ErrLecturaUsuariosAdministrablesNoDisponible
	}
	for clave := range clavesOriginal {
		if _, ok := clavesRecibidas[clave]; !ok {
			return ports.ErrLecturaUsuariosAdministrablesNoDisponible
		}
	}
	if !recibido.AutenticacionVerificadaEn.Equal(datos.AutenticacionVerificadaEn) ||
		!recibido.SesionEmitidaEn.Equal(datos.SesionEmitidaEn) ||
		!recibido.SesionValidaHasta.Equal(datos.SesionValidaHasta) ||
		!recibido.SesionRevalidadaEn.Equal(datos.SesionRevalidadaEn) {
		return ports.ErrLecturaUsuariosAdministrablesNoDisponible
	}
	recibido.AutenticacionVerificadaEn = time.Time{}
	datos.AutenticacionVerificadaEn = time.Time{}
	recibido.SesionEmitidaEn = time.Time{}
	datos.SesionEmitidaEn = time.Time{}
	recibido.SesionValidaHasta = time.Time{}
	datos.SesionValidaHasta = time.Time{}
	recibido.SesionRevalidadaEn = time.Time{}
	datos.SesionRevalidadaEn = time.Time{}
	if recibido != datos {
		return ports.ErrLecturaUsuariosAdministrablesNoDisponible
	}
	return nil
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
