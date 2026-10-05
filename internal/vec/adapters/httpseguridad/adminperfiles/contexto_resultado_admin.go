package adminperfiles

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type eventoContextoADMIN struct {
	TipoRegistro    string  `json:"tipo_registro"`
	EventoRef       string  `json:"evento_ref"`
	OperadorLogin   string  `json:"operador_login"`
	ActorRef        *string `json:"actor_ref"`
	PerfilActivoRef *string `json:"perfil_activo_ref"`
	Accion          string  `json:"accion"`
	RecursoRef      string  `json:"recurso_ref"`
	Resultado       string  `json:"resultado"`
	MotivoRef       string  `json:"motivo_ref"`
	Proceso         string  `json:"proceso"`
	Canal           string  `json:"canal"`
	FinalidadRef    string  `json:"finalidad_ref"`
	CorrelacionRef  string  `json:"correlacion_ref"`
	FuenteRef       *string `json:"fuente_ref"`
	FuenteSHA256    *string `json:"fuente_sha256"`
}

type acuseContextoADMIN struct {
	AuditoriaRef   string    `json:"auditoria_ref"`
	Secuencia      int64     `json:"secuencia"`
	HuellaSHA256   string    `json:"huella_sha256"`
	CorrelacionRef string    `json:"correlacion_ref"`
	RegistradaEn   time.Time `json:"registrada_en"`
}

type contextoConfirmadoADMIN struct {
	OperacionRef                        string    `json:"operacion_ref"`
	RegistroContextoRef                 string    `json:"registro_contexto_ref"`
	RepresentacionCanonicaBase64        string    `json:"representacion_canonica_base64"`
	HuellaSHA256                        string    `json:"huella_sha256"`
	ManifiestoProcedenciaCanonicoBase64 string    `json:"manifiesto_procedencia_canonico_base64"`
	ManifiestoProcedenciaHuellaSHA256   string    `json:"manifiesto_procedencia_huella_sha256"`
	AutoridadEfectiva                   string    `json:"autoridad_efectiva"`
	ResueltoEn                          time.Time `json:"resuelto_en"`
}

type respuestaContextoADMIN struct {
	Estado    string                   `json:"estado"`
	MotivoRef string                   `json:"motivo_ref"`
	Evento    json.RawMessage          `json:"evento"`
	Acuse     json.RawMessage          `json:"acuse"`
	Contexto  *contextoConfirmadoADMIN `json:"contexto"`
	EventoDTO eventoContextoADMIN      `json:"-"`
	AcuseDTO  acuseContextoADMIN       `json:"-"`
}

func decodificarCerradoContextoADMIN(bruto []byte, destino any, claves int) error {
	if len(bruto) == 0 || len(bruto) > 262144 {
		return ports.ErrResolutorRegistroContextoActorNoDisponible
	}
	// encoding/json acepta por defecto claves repetidas. Recorremos los tokens
	// antes del DTO, también en evento/acuse/contexto anidados.
	lexico := json.NewDecoder(bytes.NewReader(bruto))
	lexico.UseNumber()
	if clavesUnicasContextoADMIN(lexico, 0) != nil {
		return ports.ErrResolutorRegistroContextoActorNoDisponible
	}
	if _, err := lexico.Token(); err != io.EOF {
		return ports.ErrResolutorRegistroContextoActorNoDisponible
	}
	var forma map[string]json.RawMessage
	if json.Unmarshal(bruto, &forma) != nil || len(forma) != claves {
		return ports.ErrResolutorRegistroContextoActorNoDisponible
	}
	d := json.NewDecoder(bytes.NewReader(bruto))
	d.DisallowUnknownFields()
	if d.Decode(destino) != nil || d.Decode(new(any)) != io.EOF {
		return ports.ErrResolutorRegistroContextoActorNoDisponible
	}
	return nil
}

func clavesUnicasContextoADMIN(d *json.Decoder, profundidad int) error {
	if profundidad > 4 {
		return ports.ErrResolutorRegistroContextoActorNoDisponible
	}
	token, err := d.Token()
	if err != nil {
		return ports.ErrResolutorRegistroContextoActorNoDisponible
	}
	delim, compuesto := token.(json.Delim)
	if !compuesto {
		return nil
	}
	switch delim {
	case '{':
		vistas := make(map[string]struct{}, 20)
		for d.More() {
			if len(vistas) >= 20 {
				return ports.ErrResolutorRegistroContextoActorNoDisponible
			}
			claveToken, err := d.Token()
			clave, ok := claveToken.(string)
			if err != nil || !ok {
				return ports.ErrResolutorRegistroContextoActorNoDisponible
			}
			if _, duplicada := vistas[clave]; duplicada {
				return ports.ErrResolutorRegistroContextoActorNoDisponible
			}
			vistas[clave] = struct{}{}
			if clavesUnicasContextoADMIN(d, profundidad+1) != nil {
				return ports.ErrResolutorRegistroContextoActorNoDisponible
			}
		}
	case '[':
		for elementos := 0; d.More(); elementos++ {
			if elementos >= 32 || clavesUnicasContextoADMIN(d, profundidad+1) != nil {
				return ports.ErrResolutorRegistroContextoActorNoDisponible
			}
		}
	default:
		return ports.ErrResolutorRegistroContextoActorNoDisponible
	}
	esperado := json.Delim('}')
	if delim == '[' {
		esperado = ']'
	}
	cierre, err := d.Token()
	if err != nil || cierre != esperado {
		return ports.ErrResolutorRegistroContextoActorNoDisponible
	}
	return nil
}

func (r *respuestaContextoADMIN) validar(bruto []byte, solicitud ports.SolicitudResolucionRegistroContextoActorV2,
	v VinculoSesionADMIN, operacion, recibo, eventoRef, correlacion, proceso, login string) (ports.ConfirmacionRegistroContextoActorV2, error) {
	vacio := ports.ConfirmacionRegistroContextoActorV2{}
	if r == nil || decodificarCerradoContextoADMIN(bruto, r, 5) != nil ||
		decodificarCerradoContextoADMIN(r.Evento, &r.EventoDTO, 15) != nil ||
		decodificarCerradoContextoADMIN(r.Acuse, &r.AcuseDTO, 5) != nil {
		return vacio, ports.ErrResolutorRegistroContextoActorNoDisponible
	}
	e := r.EventoDTO
	a := r.AcuseDTO
	_, zonaAcuse := a.RegistradaEn.Zone()
	motivo := "contexto_admin_pre_v2_" + r.Estado
	if (r.Estado != "permitido" && r.Estado != "denegado" && r.Estado != "error") ||
		r.MotivoRef != motivo || e.TipoRegistro != "contexto_admin_pre_v2" ||
		(e.EventoRef != eventoRef && (r.Estado == "permitido" || !referenciaEventoContextoADMIN(e.EventoRef))) ||
		e.OperadorLogin != login || e.Accion != "registrar_contexto_admin" ||
		e.RecursoRef != operacion || e.Resultado != r.Estado || e.MotivoRef != motivo ||
		e.Proceso != proceso || e.Canal != "administracion_privilegiada" ||
		e.FinalidadRef != "establecer_contexto_admin" || e.CorrelacionRef != correlacion ||
		a.AuditoriaRef != "aud_v3_ap2_"+strings.TrimPrefix(e.EventoRef, "evento_") ||
		a.Secuencia <= 0 || a.Secuencia > 9007199254740991 || !huella(a.HuellaSHA256) || a.CorrelacionRef != correlacion ||
		a.RegistradaEn.IsZero() || zonaAcuse != 0 || a.RegistradaEn.Nanosecond()%1000 != 0 {
		return vacio, ports.ErrResolutorRegistroContextoActorNoDisponible
	}
	// AD192 admite identidad desconocida antes de V2. Toda coordenada que sí
	// aparece debe pertenecer al vínculo original de esta petición.
	if (e.FuenteRef == nil) != (e.FuenteSHA256 == nil) {
		return vacio, ports.ErrResolutorRegistroContextoActorNoDisponible
	}
	if (e.FuenteRef != nil && (*e.FuenteRef != v.FuenteRef || *e.FuenteSHA256 != v.FuenteSHA256)) ||
		(e.ActorRef != nil && (e.FuenteRef == nil || *e.ActorRef != v.PersonaRef)) ||
		(e.PerfilActivoRef != nil && (e.ActorRef == nil || *e.PerfilActivoRef != v.PerfilActivoRef)) {
		return vacio, ports.ErrResolutorRegistroContextoActorNoDisponible
	}
	if r.Estado != "permitido" {
		if r.Contexto != nil {
			return vacio, ports.ErrResolutorRegistroContextoActorNoDisponible
		}
		return vacio, nil
	}
	if r.Contexto == nil || a.RegistradaEn.Before(r.Contexto.ResueltoEn) || e.ActorRef == nil || e.PerfilActivoRef == nil ||
		e.FuenteRef == nil || e.FuenteSHA256 == nil || *e.ActorRef != v.PersonaRef ||
		*e.PerfilActivoRef != v.PerfilActivoRef || *e.FuenteRef != v.FuenteRef || *e.FuenteSHA256 != v.FuenteSHA256 {
		return vacio, ports.ErrResolutorRegistroContextoActorNoDisponible
	}
	c := r.Contexto
	canon, err := base64.StdEncoding.DecodeString(c.RepresentacionCanonicaBase64)
	if err != nil || len(canon) == 0 || len(canon) > 65536 {
		return vacio, ports.ErrResolutorRegistroContextoActorNoDisponible
	}
	manifiesto, err := base64.StdEncoding.DecodeString(c.ManifiestoProcedenciaCanonicoBase64)
	if err != nil || len(manifiesto) == 0 || len(manifiesto) > 65536 {
		return vacio, ports.ErrResolutorRegistroContextoActorNoDisponible
	}
	actor, err := domain.RehidratarContextoActorVinculadoV2(canon)
	if err != nil {
		return vacio, ports.ErrResolutorRegistroContextoActorNoDisponible
	}
	confirmada := ports.ConfirmacionRegistroContextoActorV2{
		OperacionRef: c.OperacionRef, RegistroContextoRef: c.RegistroContextoRef,
		Contexto: actor, RepresentacionCanonica: canon, HuellaSHA256: c.HuellaSHA256,
		ManifiestoProcedenciaCanonico:     manifiesto,
		ManifiestoProcedenciaHuellaSHA256: c.ManifiestoProcedenciaHuellaSHA256,
		AutoridadEfectiva:                 domain.AutoridadProcedenciaContextoActorV1(c.AutoridadEfectiva),
		ResueltoEnAutoritativo:            c.ResueltoEn.UTC().Truncate(time.Microsecond),
	}
	if confirmada.OperacionRef != operacion || confirmada.RegistroContextoRef != recibo ||
		confirmada.Contexto.PersonaRef != v.PersonaRef || confirmada.ValidarParaProductiva(solicitud) != nil {
		return vacio, ports.ErrResolutorRegistroContextoActorNoDisponible
	}
	return confirmada, nil
}

func referenciaEventoContextoADMIN(ref string) bool {
	if len(ref) != 7+32 || !strings.HasPrefix(ref, "evento_") {
		return false
	}
	for _, c := range ref[7:] {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}
