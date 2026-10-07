package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"slices"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type ServicioConsultaResumenRRHHNominal struct {
	lector      ports.LectorResumenBolsasNominal
	autorizador ports.AutorizadorConsultaRRHHV3
	reloj       func() time.Time
}

func NuevoServicioConsultaResumenRRHHNominal(lector ports.LectorResumenBolsasNominal, autorizador ports.AutorizadorConsultaRRHHV3, reloj func() time.Time) (*ServicioConsultaResumenRRHHNominal, error) {
	if dependenciaNulaPoliticaCese(lector) || dependenciaNulaPoliticaCese(autorizador) || reloj == nil {
		return nil, ports.ErrResumenBolsasNoDisponible
	}
	return &ServicioConsultaResumenRRHHNominal{lector: lector, autorizador: autorizador, reloj: reloj}, nil
}

// Consultar emite una sola decisión para la ruta y delega al repositorio el
// consumo V3, la auditoría común y las consultas en una misma transacción.
func (s *ServicioConsultaResumenRRHHNominal) Consultar(ctx context.Context, orden ports.OrdenConsultaResumenRRHH) (ports.ResumenBolsasNominal, error) {
	var vacio ports.ResumenBolsasNominal
	if s == nil || ctx == nil || ctx.Err() != nil || dependenciaNulaPoliticaCese(s.lector) || dependenciaNulaPoliticaCese(s.autorizador) || s.reloj == nil {
		return vacio, ports.ErrResumenBolsasNoDisponible
	}
	audiencia, finalidad, tipo, recurso := ports.AudienciaRRHHBolsasConsultar, ports.FinalidadRRHHBolsasConsultar,
		ports.TipoRecursoRRHHBolsas, ports.RecursoRRHHBolsas
	campos := []string{"bolsas", "conteos"}
	if orden.Accion == ports.AccionRRHHEstadisticasConsultar {
		audiencia, finalidad, tipo, recurso = ports.AudienciaRRHHEstadisticasConsultar, ports.FinalidadRRHHEstadisticasConsultar,
			ports.TipoRecursoRRHHEstadisticas, ports.RecursoRRHHEstadisticas
		campos = []string{"estadisticas"}
	} else if orden.Accion != ports.AccionRRHHBolsasConsultar {
		return vacio, vd.ErrAutorizacionDenegada
	}
	ahora := s.reloj().UTC().Truncate(time.Microsecond)
	actor, err := orden.Resultado.Clonar()
	vinculo, errVinculo := orden.Vinculo.Datos()
	if err != nil || errVinculo != nil || ahora.IsZero() || orden.UnidadRef == "" || orden.AmbitoRef == "" ||
		actor.Contexto.Principal.AuthMethod == vd.AuthMethodDemo || vinculo.MetodoObservado == vd.AuthMethodDemo ||
		vinculo.Superficie != vd.SuperficieAutenticacionInternaCorporativaV1 ||
		orden.Vinculo.ValidarPara(actor) != nil || !orden.Vinculo.VigenteEn(ahora, actor) ||
		!vd.ReferenciaMotivoAutorizacionV2Valida(orden.Motivo) || orden.Correlacion.Validar() != nil {
		return vacio, vd.ErrAutorizacionDenegada
	}
	recursoV3 := vd.RecursoAutorizable{Referencia: recurso, ModuloID: "bolsa", Tipo: tipo,
		Ambitos: map[string]string{"unidad_ref": orden.UnidadRef, "ambito_ref": orden.AmbitoRef}, Atributos: map[string]string{}}
	solicitud, err := vd.NuevaSolicitudAutorizacionLigadaV3(vd.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: orden.Vinculo, ReferenciaMotivo: orden.Motivo,
		Accion: orden.Accion, Recurso: recursoV3, Finalidad: finalidad, Correlacion: orden.Correlacion,
	})
	if err != nil {
		return vacio, vd.ErrAutorizacionDenegada
	}
	decision, confirmacion, exportador, err := s.autorizador.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, actor)
	if err != nil {
		if errors.Is(err, vecports.ErrFuenteAutorizacionNoDisponible) ||
			errors.Is(err, vecports.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible) ||
			errors.Is(err, vecports.ErrRegistroDecisionNoDisponible) {
			return vacio, ports.ErrResumenBolsasNoDisponible
		}
		return vacio, vd.ErrAutorizacionDenegada
	}
	ahora = s.reloj().UTC().Truncate(time.Microsecond)
	if dependenciaNulaPoliticaCese(exportador) || ctx.Err() != nil ||
		!concesionResumenRRHHExacta(solicitud, decision, confirmacion, actor, ahora, campos) ||
		!orden.Vinculo.VigenteEn(ahora, actor) {
		return vacio, vd.ErrAutorizacionDenegada
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !vecports.MaterialAtestadoLigadoV3(solicitud, decision, confirmacion, actor, orden.Motivo, material, audiencia) {
		return vacio, vd.ErrAutorizacionDenegada
	}
	return s.lector.LeerResumenNominal(ctx, orden.Accion, orden.CeseActivo, material)
}

func concesionResumenRRHHExacta(s vd.SolicitudAutorizacionLigadaV3, d vd.DecisionAutorizacionLigadaV3,
	c vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, actor vd.ResultadoContextoActorRegistradoV2,
	ahora time.Time, campos []string) bool {
	if d.ValidarPara(s) != nil {
		return false
	}
	concedida, _, err := d.Resultado()
	if err != nil || !concedida {
		return false
	}
	datos, err := s.Datos()
	if err != nil {
		return false
	}
	registro, err := vecports.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(s, d, datos.ReferenciaMotivo, actor)
	if err != nil || c.ValidarPara(registro) != nil || !c.DentroDeVentanaEn(ahora) {
		return false
	}
	canon, err := vd.RepresentacionCanonicaDecisionAutorizacionV3(d)
	var limites struct {
		Campos       []string `json:"campos_permitidos"`
		Obligaciones []string `json:"obligaciones"`
	}
	return err == nil && json.Unmarshal(canon, &limites) == nil && slices.Equal(limites.Campos, campos) && len(limites.Obligaciones) == 0
}

type ServicioConsultaCandidatosRRHHNominal struct {
	lector      ports.LectorCandidatosRRHHNominal
	autorizador ports.AutorizadorConsultaRRHHV3
	reloj       func() time.Time
}

func NuevoServicioConsultaCandidatosRRHHNominal(lector ports.LectorCandidatosRRHHNominal, autorizador ports.AutorizadorConsultaRRHHV3, reloj func() time.Time) (*ServicioConsultaCandidatosRRHHNominal, error) {
	if dependenciaNulaPoliticaCese(lector) || dependenciaNulaPoliticaCese(autorizador) || reloj == nil {
		return nil, ports.ErrResumenBolsasNoDisponible
	}
	return &ServicioConsultaCandidatosRRHHNominal{lector: lector, autorizador: autorizador, reloj: reloj}, nil
}

// HuellaConsultaCandidatosRRHH liga bolsa, estado, texto, cursor y límite.
// Texto nunca forma parte de la referencia del recurso ni de la auditoría.
func HuellaConsultaCandidatosRRHH(q ports.ConsultaCandidatosRRHHNominal) (string, bool) {
	if q.BolsaRef == "" || !strings.HasPrefix(q.BolsaRef, "bolsa:") || strings.ContainsAny(q.BolsaRef, "/\x00\x1f") ||
		len(q.BolsaRef) > 512 || len(q.Texto) > 300 || strings.ContainsAny(q.Texto, "\x00\x1f") ||
		strings.ToLower(q.Texto) != q.Texto || len(q.CursorToken) > 256 || strings.ContainsAny(q.CursorToken, "\x00\x1f") ||
		len(q.CursorRef) > 512 || q.Limite < 1 || q.Limite > 100 ||
		(q.Estado != "" && q.Estado != "disponible" && q.Estado != "no_disponible" && q.Estado != "trabajando" &&
			q.Estado != "pendiente_incorporacion" && q.Estado != "renuncia" && q.Estado != "excluido" &&
			q.Estado != "disponible_desde" && q.Estado != "en_revision") {
		return "", false
	}
	switch q.Modo {
	case ports.ModoPaginaCandidatosRRHH:
		if q.Texto != "" || len(q.SeleccionRefs) != 0 || (q.CursorToken == "") != (q.CursorRef == "") ||
			(q.CursorToken == "" && q.SnapshotSHA256 != "") || (q.CursorToken != "" && len(q.SnapshotSHA256) != 64) {
			return "", false
		}
	case ports.ModoBarridoTextoCandidatosRRHH:
		if q.Texto == "" || q.CursorToken != "" || q.CursorRef != "" || q.SnapshotSHA256 != "" || len(q.SeleccionRefs) != 0 {
			return "", false
		}
	case ports.ModoPaginaTextoCandidatosRRHH:
		if q.Texto == "" || q.CursorToken == "" || len(q.SnapshotSHA256) != 64 ||
			len(q.SeleccionRefs) < 1 || len(q.SeleccionRefs) > 100 {
			return "", false
		}
	default:
		return "", false
	}
	suma := sha256.Sum256([]byte(q.Estado + "\x1f" + q.Texto + "\x1f" + q.CursorToken + "\x1f" + strconv.Itoa(q.Limite)))
	return hex.EncodeToString(suma[:]), true
}

func (s *ServicioConsultaCandidatosRRHHNominal) Consultar(ctx context.Context, orden ports.OrdenConsultaResumenRRHH, q ports.ConsultaCandidatosRRHHNominal) (ports.PaginaCandidatosRRHHNominal, error) {
	var vacio ports.PaginaCandidatosRRHHNominal
	if s == nil || ctx == nil || ctx.Err() != nil || dependenciaNulaPoliticaCese(s.lector) || dependenciaNulaPoliticaCese(s.autorizador) || s.reloj == nil ||
		orden.Accion != ports.AccionRRHHCandidatosConsultar || orden.UnidadRef == "" || orden.AmbitoRef == "" {
		return vacio, ports.ErrResumenBolsasNoDisponible
	}
	huella, ok := HuellaConsultaCandidatosRRHH(q)
	if !ok {
		return vacio, vd.ErrAutorizacionDenegada
	}
	ahora := s.reloj().UTC().Truncate(time.Microsecond)
	actor, err := orden.Resultado.Clonar()
	vinculo, errVinculo := orden.Vinculo.Datos()
	if err != nil || errVinculo != nil || ahora.IsZero() ||
		actor.Contexto.Principal.AuthMethod == vd.AuthMethodDemo || vinculo.MetodoObservado == vd.AuthMethodDemo ||
		vinculo.Superficie != vd.SuperficieAutenticacionInternaCorporativaV1 ||
		orden.Vinculo.ValidarPara(actor) != nil || !orden.Vinculo.VigenteEn(ahora, actor) ||
		!vd.ReferenciaMotivoAutorizacionV2Valida(orden.Motivo) || orden.Correlacion.Validar() != nil {
		return vacio, vd.ErrAutorizacionDenegada
	}
	recurso := vd.RecursoAutorizable{Referencia: q.BolsaRef + ":filtro:" + huella, ModuloID: "bolsa", Tipo: ports.TipoRecursoRRHHCandidatos,
		Ambitos: map[string]string{"unidad_ref": orden.UnidadRef, "ambito_ref": orden.AmbitoRef}, Atributos: map[string]string{"bolsa_ref": q.BolsaRef}}
	solicitud, err := vd.NuevaSolicitudAutorizacionLigadaV3(vd.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: orden.Vinculo, ReferenciaMotivo: orden.Motivo,
		Accion: orden.Accion, Recurso: recurso, Finalidad: ports.FinalidadRRHHCandidatosConsultar, Correlacion: orden.Correlacion,
	})
	if err != nil {
		return vacio, vd.ErrAutorizacionDenegada
	}
	decision, confirmacion, exportador, err := s.autorizador.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, actor)
	if err != nil {
		if errors.Is(err, vecports.ErrFuenteAutorizacionNoDisponible) ||
			errors.Is(err, vecports.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible) ||
			errors.Is(err, vecports.ErrRegistroDecisionNoDisponible) {
			return vacio, ports.ErrResumenBolsasNoDisponible
		}
		return vacio, vd.ErrAutorizacionDenegada
	}
	ahora = s.reloj().UTC().Truncate(time.Microsecond)
	if dependenciaNulaPoliticaCese(exportador) || ctx.Err() != nil ||
		!concesionResumenRRHHExacta(solicitud, decision, confirmacion, actor, ahora, []string{"candidatos", "contactos", "turno"}) ||
		!orden.Vinculo.VigenteEn(ahora, actor) {
		return vacio, vd.ErrAutorizacionDenegada
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !vecports.MaterialAtestadoLigadoV3(solicitud, decision, confirmacion, actor, orden.Motivo, material, ports.AudienciaRRHHCandidatosConsultar) {
		return vacio, vd.ErrAutorizacionDenegada
	}
	return s.lector.LeerCandidatosNominal(ctx, q, material)
}
