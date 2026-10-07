package auditoria

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"reflect"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type EmisorMaterialV3 interface {
	EmitirMaterialAutorizacionAtestadaV3(context.Context, vecdomain.SolicitudAutorizacionLigadaV3, vecdomain.ResultadoContextoActorRegistradoV2) (vecdomain.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error)
}

// ContextoConsulta procede solo de la frontera de identidad y catalogos.
// La solicitud HTTP no construye ni modifica estas capacidades.
type ContextoConsulta struct {
	Vinculo     vecdomain.VinculoAutenticacionActorV2
	Resultado   vecdomain.ResultadoContextoActorRegistradoV2
	Motivo      vecdomain.ReferenciaEntradaCatalogo
	Correlacion vecdomain.ReferenciaCorrelacionAutorizacionV2
}

type Peticion struct {
	Filtro   Filtro
	Cursor   string
	Contexto ContextoConsulta
}

type Pagina struct {
	Registros       []Registro `json:"registros"`
	SiguienteCursor string     `json:"siguiente_cursor"`
}

type Servicio struct {
	emisor      EmisorMaterialV3
	ct          FuenteAuditoria
	bolsa       FuenteAuditoria
	intentos    *ConfiguracionIntentos
	claveCursor [32]byte
	ahora       func() time.Time
}

// ConfiguracionIntentos pertenece al servidor. El motivo procede del catálogo
// publicado y el proceso y canal, de la configuración privada de AD169.
type ConfiguracionIntentos struct {
	Registrador vecports.RegistradorIntentosAuditoria
	Proceso     string
	Canal       string
	Finalidad   string
	Motivo      vecdomain.ReferenciaEntradaCatalogo
}

const (
	recursoIntentoConsultaInvalida = "auditoria:consulta_invalida"
	recursoIntentoConsultaCT       = "auditoria:consulta_ct"
	recursoIntentoConsultaBolsa    = "auditoria:consulta_bolsa"
	recursoIntentoOpcionesRRHH     = "auditoria:opciones_rrhh"
)

func (s *Servicio) ConfigurarIntentos(c ConfiguracionIntentos) error {
	if s == nil || dependenciaNula(c.Registrador) || c.Proceso == "" ||
		c.Canal != string(vecdomain.SuperficieAutenticacionInternaCorporativaV1) ||
		c.Finalidad == "" || !vecdomain.ReferenciaMotivoAutorizacionV2Valida(c.Motivo) ||
		(vecdomain.DatosIntentoAuditoria{Accion: AccionConsultar, ModuloID: ModuloAutorizacion,
			RecursoRef: recursoIntentoConsultaInvalida, FinalidadRef: c.Finalidad,
			Resultado: vecdomain.ResultadoIntentoAuditoriaError, Motivo: c.Motivo,
			Proceso: c.Proceso, Canal: c.Canal,
			CorrelacionRef: "correlacion_00000000000000000000000000000000"}).Validar() != nil {
		return ErrNoDisponible
	}
	s.intentos = &c
	return nil
}

func (s *Servicio) registrarIntento(ctx context.Context, identidad IdentidadResuelta, recurso string,
	resultado vecdomain.ResultadoIntentoAuditoria, actuales *Opciones) (vecports.AcuseIntentoAuditoria, error) {
	var vacio vecports.AcuseIntentoAuditoria
	switch recurso {
	case recursoIntentoConsultaInvalida, recursoIntentoConsultaCT, recursoIntentoConsultaBolsa, recursoIntentoOpcionesRRHH:
	default:
		return vacio, ErrNoDisponible
	}
	if s == nil || s.intentos == nil || ctx == nil || identidad.Resultado.Validar() != nil ||
		identidad.Vinculo.ValidarPara(identidad.Resultado) != nil {
		return vacio, ErrNoDisponible
	}
	correlacion, err := identidad.Correlacion.ValorCanonico()
	if err != nil {
		return vacio, ErrNoDisponible
	}
	ref, err := vecports.NuevaReferenciaIntentoAuditoria()
	if err != nil {
		return vacio, ErrNoDisponible
	}
	c := s.intentos
	finalidad, motivo := c.Finalidad, c.Motivo
	if actuales != nil {
		if actuales.PermisoRequerido != AccionConsultar || actuales.MotivoRef != actuales.Motivo.Referencia() ||
			!vecdomain.ReferenciaMotivoAutorizacionV2Valida(actuales.Motivo) {
			return vacio, ErrNoDisponible
		}
		finalidad, motivo = actuales.FinalidadRef, actuales.Motivo
	}
	orden, err := vecports.NuevaOrdenIntentoAuditoria(ref, identidad.Resultado, identidad.Vinculo,
		vecdomain.DatosIntentoAuditoria{Accion: AccionConsultar, ModuloID: ModuloAutorizacion,
			RecursoRef: recurso, FinalidadRef: finalidad, Resultado: resultado,
			Motivo: motivo, Proceso: c.Proceso, Canal: c.Canal, CorrelacionRef: correlacion})
	if err != nil {
		return vacio, ErrNoDisponible
	}
	acuse, err := c.Registrador.AppendIntentoAuditoria(ctx, orden)
	if errors.Is(err, vecports.ErrIntentoAuditoriaNoDisponible) {
		acuse, err = c.Registrador.AppendIntentoAuditoria(ctx, orden)
	}
	if err != nil || acuse.ValidarPara(orden) != nil {
		return vacio, ErrNoDisponible
	}
	return acuse, nil
}

func dependenciaNula(x any) bool {
	if x == nil {
		return true
	}
	v := reflect.ValueOf(x)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return v.IsNil()
	}
	return false
}

func NuevoServicio(emisor EmisorMaterialV3, ct, bolsa FuenteAuditoria) (*Servicio, error) {
	if dependenciaNula(emisor) || dependenciaNula(ct) || dependenciaNula(bolsa) {
		return nil, ErrNoDisponible
	}
	s := &Servicio{emisor: emisor, ct: ct, bolsa: bolsa, ahora: time.Now}
	if _, err := rand.Read(s.claveCursor[:]); err != nil {
		return nil, ErrNoDisponible
	}
	return s, nil
}

func errorValidacionConsulta(ctx context.Context) error {
	if ctx == nil || ctx.Err() != nil {
		return ErrNoDisponible
	}
	return ErrDenegada
}

func (s *Servicio) Consultar(ctx context.Context, p Peticion) (Pagina, error) {
	if s == nil || ctx == nil || ctx.Err() != nil ||
		dependenciaNula(s.emisor) || dependenciaNula(s.ct) || dependenciaNula(s.bolsa) {
		return Pagina{}, ErrNoDisponible
	}
	if p.Filtro.Validar() != nil {
		return Pagina{}, errorValidacionConsulta(ctx)
	}
	f := p.Filtro
	if !f.Antes.vacia() {
		return Pagina{}, errorValidacionConsulta(ctx)
	} // solo el cursor firmado fija la pagina
	if p.Cursor != "" {
		pos, err := s.decodificarCursor(p.Cursor, f)
		if err != nil || pos.Fuente != f.Fuente {
			return Pagina{}, errorValidacionConsulta(ctx)
		}
		f.Antes = pos
	}
	if ctx.Err() != nil {
		return Pagina{}, ErrNoDisponible
	}
	if p.Contexto.Motivo.Validar() != nil ||
		p.Contexto.Motivo.Referencia() != f.MotivoRef ||
		p.Contexto.Resultado.Validar() != nil ||
		p.Contexto.Vinculo.ValidarPara(p.Contexto.Resultado) != nil ||
		!p.Contexto.Vinculo.VigenteEn(s.ahora().UTC().Truncate(time.Microsecond), p.Contexto.Resultado) ||
		p.Contexto.Correlacion.Validar() != nil {
		return Pagina{}, errorValidacionConsulta(ctx)
	}
	recurso, err := RecursoFiltro(f)
	if err != nil {
		return Pagina{}, errorValidacionConsulta(ctx)
	}
	solicitud, err := vecdomain.NuevaSolicitudAutorizacionLigadaV3(vecdomain.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: p.Contexto.Vinculo,
		ReferenciaMotivo:          p.Contexto.Motivo, Accion: AccionConsultar,
		Recurso: recurso, Finalidad: f.FinalidadRef, Correlacion: p.Contexto.Correlacion,
	})
	if err != nil {
		return Pagina{}, errorValidacionConsulta(ctx)
	}
	if ctx.Err() != nil {
		return Pagina{}, ErrNoDisponible
	}
	lector := s.ct
	if f.Fuente == "bolsa" {
		lector = s.bolsa
	}
	decision, confirmacion, exportador, e := s.emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, p.Contexto.Resultado)
	if e != nil {
		if ctx.Err() == nil && (errors.Is(e, ErrDenegada) ||
			errors.Is(e, vecports.ErrDenegacionExplicitaAutorizacionLigadaV3)) {
			return Pagina{}, ErrDenegada
		}
		return Pagina{}, ErrNoDisponible
	}
	if dependenciaNula(exportador) || ctx.Err() != nil {
		return Pagina{}, ErrNoDisponible
	}
	material, e := exportador.ExportarMaterialParaConsumidor()
	if e != nil {
		return Pagina{}, ErrNoDisponible
	}
	q := ConsultaAutorizada{Filtro: f, Material: material, Solicitud: solicitud, Decision: decision, Confirmacion: confirmacion, ResultadoContexto: p.Contexto.Resultado}
	if ValidarConsultaAutorizadaEn(q, s.ahora().UTC().Truncate(time.Microsecond)) != nil {
		return Pagina{}, ErrNoDisponible
	}
	pagina, e := lector.ConsultarAuditoria(ctx, q)
	if e != nil {
		if ctx.Err() == nil && errors.Is(e, ErrDenegada) {
			return Pagina{}, ErrDenegada
		}
		return Pagina{}, ErrNoDisponible
	}
	if ctx.Err() != nil {
		return Pagina{}, ErrNoDisponible
	}
	if len(pagina.Registros) > int(f.Limite)+1 {
		return Pagina{}, ErrFuenteInvalida
	}
	filas := make([]Registro, 0, len(pagina.Registros))
	for i, r := range pagina.Registros {
		if !registroValido(r, f, f.Fuente) || (i > 0 && !posicionMenor(posicion(r), posicion(pagina.Registros[i-1]))) {
			return Pagina{}, ErrFuenteInvalida
		}
		filas = append(filas, clonarRegistro(r))
	}
	mas := len(filas) > int(f.Limite)
	if mas {
		filas = filas[:f.Limite]
	}
	resultado := Pagina{Registros: filas, SiguienteCursor: ""}
	if mas {
		resultado.SiguienteCursor, err = s.codificarCursor(f, posicion(filas[len(filas)-1]))
		if err != nil {
			return Pagina{}, ErrNoDisponible
		}
	}
	return resultado, nil
}

func posicion(r Registro) Posicion {
	return Posicion{OcurridoEn: r.OcurridoEn, Fuente: r.Fuente, ID: r.ID}
}

func posicionMenor(a, b Posicion) bool {
	if !a.OcurridoEn.Equal(b.OcurridoEn) {
		return a.OcurridoEn.Before(b.OcurridoEn)
	}
	if a.Fuente != b.Fuente {
		return a.Fuente < b.Fuente
	}
	return a.ID < b.ID
}

func clonarRegistro(r Registro) Registro {
	r.Antes = clonarValores(r.Antes)
	r.Despues = clonarValores(r.Despues)
	return r
}
func clonarValores(m map[string]string) map[string]string {
	c := make(map[string]string, len(m))
	for k, v := range m {
		c[k] = v
	}
	return c
}

var patronVersionContacto = regexp.MustCompile(`^version:[1-9][0-9]{0,18}$`)
var patronHuella = regexp.MustCompile(`^[0-9a-f]{64}$`)

func valoresValidos(m map[string]string, fuente string) bool {
	if len(m) > 6 {
		return false
	}
	for k, v := range m {
		if len(v) > 160 || !strings.EqualFold(v, strings.TrimSpace(v)) {
			return false
		}
		switch fuente + ":" + k {
		case "ct:fase", "ct:estado", "bolsa:situacion":
			if v != "" && !referenciaExacta(v, 160) {
				return false
			}
		case "bolsa:fecha_disponible":
			if v != "" {
				t, e := time.Parse(time.RFC3339Nano, v)
				if e != nil || t.Location() != time.UTC {
					return false
				}
			}
		case "bolsa:datos_contacto", "bolsa:correo", "bolsa:telefono_1", "bolsa:telefono_2":
			if v != "" && !patronVersionContacto.MatchString(v) {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func registroValido(r Registro, f Filtro, fuente string) bool {
	if !referenciaExacta(r.ID, 512) || r.Fuente != fuente ||
		!referenciaExacta(r.ModuloID, 128) || !referenciaExacta(r.Accion, 160) ||
		(r.ActorRef != "" && !referenciaExacta(r.ActorRef, 512)) ||
		!instanteValido(r.OcurridoEn) || r.OcurridoEn.Before(f.Desde) || !r.OcurridoEn.Before(f.Hasta) ||
		r.ExpedienteRef != f.ExpedienteRef || (f.ActorRef != "" && r.ActorRef != f.ActorRef) ||
		(r.ReciboRef != "" && !referenciaExacta(r.ReciboRef, 512)) ||
		(r.Motivo != "" && !textoMotivoMinimizado(r.Motivo)) ||
		!referenciaExacta(r.Resultado, 128) ||
		(r.AntesSHA256 != "" && !patronHuella.MatchString(r.AntesSHA256)) ||
		(r.DespuesSHA256 != "" && !patronHuella.MatchString(r.DespuesSHA256)) ||
		!valoresValidos(r.Antes, fuente) || !valoresValidos(r.Despues, fuente) ||
		(!r.DatosDisponibles && (len(r.Antes) > 0 || len(r.Despues) > 0)) ||
		(!f.Antes.vacia() && !posicionMenor(posicion(r), f.Antes)) {
		return false
	}
	return true
}

func textoMotivoMinimizado(valor string) bool {
	if len(valor) > 1000 || valor != strings.TrimSpace(valor) || !utf8.ValidString(valor) {
		return false
	}
	for _, r := range valor {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

type cursorDocumento struct {
	Version      int      `json:"v"`
	FiltroSHA256 string   `json:"f"`
	Posicion     Posicion `json:"p"`
}

func (s *Servicio) codificarCursor(f Filtro, p Posicion) (string, error) {
	f.Antes = Posicion{}
	h, e := HuellaFiltro(f)
	if e != nil {
		return "", e
	}
	b, e := json.Marshal(cursorDocumento{Version: 1, FiltroSHA256: h, Posicion: p})
	if e != nil {
		return "", ErrNoDisponible
	}
	mac := hmac.New(sha256.New, s.claveCursor[:])
	_, _ = mac.Write(b)
	return base64.RawURLEncoding.EncodeToString(append(b, mac.Sum(nil)...)), nil
}

func (s *Servicio) decodificarCursor(token string, f Filtro) (Posicion, error) {
	if len(token) > 2048 {
		return Posicion{}, ErrDenegada
	}
	b, e := base64.RawURLEncoding.DecodeString(token)
	if e != nil || len(b) <= sha256.Size {
		return Posicion{}, ErrDenegada
	}
	cuerpo, marca := b[:len(b)-sha256.Size], b[len(b)-sha256.Size:]
	mac := hmac.New(sha256.New, s.claveCursor[:])
	_, _ = mac.Write(cuerpo)
	if !hmac.Equal(marca, mac.Sum(nil)) {
		return Posicion{}, ErrDenegada
	}
	var c cursorDocumento
	if json.Unmarshal(cuerpo, &c) != nil || c.Version != 1 || !c.Posicion.validar() || c.Posicion.vacia() {
		return Posicion{}, ErrDenegada
	}
	f.Antes = Posicion{}
	h, e := HuellaFiltro(f)
	if e != nil || h != c.FiltroSHA256 {
		return Posicion{}, ErrDenegada
	}
	return c.Posicion, nil
}
