// Package rptusosfixture coordina exclusivamente el ensayo sintético AD3/Cat3.
package rptusosfixture

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"maps"
	"reflect"
	"strings"
	"sync"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const (
	audienciaUsos      = "vec_catalogos_configurables.usos_categorias.v1"
	finalidadUsos      = "vincular_categoria_a_operacion"
	accionReserva      = "vec.catalogos.categorias.reservar_uso"
	accionConfirmacion = "vec.catalogos.categorias.confirmar_uso"
	accionCancelacion  = "vec.catalogos.categorias.cancelar_uso"
	// RotuloEntradaEnsayo es un marcador del contrato del fixture, sin efecto legal.
	RotuloEntradaEnsayo  = "entrada de ensayo"
	maximoBytesEvidencia = 64 * 1024
	maximoActos          = 64
)

// Emisor pertenece a la composición confiable. Construye una solicitud V3
// nueva con actor, motivo y correlación propios y exporta material del emisor
// nominal. El recorrido nunca fabrica identidad, permisos ni material firmado.
type Emisor interface {
	Emitir(context.Context, ports.PreparacionAutorizacionUsoCategoriaRPT) (domain.SolicitudAutorizacionLigadaV3, ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

// EvidenciaSintetica conserva los bytes exactos que el bootstrap obtiene del
// documento privado de ensayo. No acredita un efecto de CT, Personal o Bolsa.
type EvidenciaSintetica struct {
	Referencia string
	Rotulo     string
	Bytes      []byte
}

type Entrada struct {
	Confirmacion          ports.MaterialTerminalUsoCategoriaRPT
	Cancelacion           ports.MaterialTerminalUsoCategoriaRPT
	EvidenciaConfirmacion EvidenciaSintetica
	EvidenciaCancelacion  EvidenciaSintetica
}

// Resultado contiene recibos de RPT; no expone documentos ni credenciales.
// La historia completa queda fuera de estos puertos y exige cotejo autorizado.
type Resultado struct {
	ReservaConfirmacion ports.ResultadoUsoCategoriaRPT
	Confirmacion        ports.ResultadoUsoCategoriaRPT
	ReservaCancelacion  ports.ResultadoUsoCategoriaRPT
	Cancelacion         ports.ResultadoUsoCategoriaRPT
}

type Recorrido struct {
	preparador ports.PreparadorUsosCategoriaRPT
	emisor     Emisor
	gestor     ports.GestorUsosCategoriaRPT
	mu         sync.Mutex
	decisiones map[string]struct{}
}

func NuevoRecorrido(preparador ports.PreparadorUsosCategoriaRPT, emisor Emisor, gestor ports.GestorUsosCategoriaRPT) (*Recorrido, error) {
	if nulo(preparador) || nulo(emisor) || nulo(gestor) {
		return nil, ports.ErrUsoCategoriaRPTNoDisponible
	}
	return &Recorrido{preparador: preparador, emisor: emisor, gestor: gestor, decisiones: make(map[string]struct{})}, nil
}

func nulo(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return r.IsNil()
	}
	return false
}

// Ejecutar admite su repetición con la entrada original. Cada uno de los cuatro
// actos vuelve a preparar y emitir V3: no conserva capacidades para el replay.
// En un fallo devuelve los resultados ya recibidos para conservar los recibos;
// no intenta compensaciones, SQL de reparación ni borrado de historia.
func (r *Recorrido) Ejecutar(ctx context.Context, e Entrada) (Resultado, error) {
	var out Resultado
	if err := validarEntrada(e); err != nil {
		return out, err
	}
	var err error
	out.ReservaConfirmacion, err = r.Reserva(ctx, e.Confirmacion.Reserva)
	if err != nil {
		return out, err
	}
	out.Confirmacion, err = r.Confirmacion(ctx, e.Confirmacion, e.EvidenciaConfirmacion)
	if err != nil {
		return out, err
	}
	out.ReservaCancelacion, err = r.Reserva(ctx, e.Cancelacion.Reserva)
	if err != nil {
		return out, err
	}
	out.Cancelacion, err = r.Cancelacion(ctx, e.Cancelacion, e.EvidenciaCancelacion)
	return out, err
}

func validarEntrada(e Entrada) error {
	if e.Confirmacion.Reserva.UsoRef == e.Cancelacion.Reserva.UsoRef ||
		e.Confirmacion.Reserva.ReservaReciboRef == e.Cancelacion.Reserva.ReservaReciboRef ||
		e.Confirmacion.TerminalReciboRef == e.Cancelacion.TerminalReciboRef ||
		e.Confirmacion.TerminalReciboRef == e.Cancelacion.Reserva.ReservaReciboRef ||
		e.Cancelacion.TerminalReciboRef == e.Confirmacion.Reserva.ReservaReciboRef {
		return ports.ErrUsoCategoriaRPTInvalido
	}
	if err := validarEvidencia(e.Confirmacion, e.EvidenciaConfirmacion); err != nil {
		return err
	}
	return validarEvidencia(e.Cancelacion, e.EvidenciaCancelacion)
}

func validarEvidencia(m ports.MaterialTerminalUsoCategoriaRPT, e EvidenciaSintetica) error {
	if e.Rotulo != RotuloEntradaEnsayo || len(e.Bytes) == 0 || len(e.Bytes) > maximoBytesEvidencia ||
		!bytes.HasPrefix(e.Bytes, []byte(RotuloEntradaEnsayo+"\n")) ||
		!strings.HasPrefix(e.Referencia, "fixture:rpt:terminal:") || len(e.Referencia) <= len("fixture:rpt:terminal:") ||
		len(e.Referencia) > 160 || strings.ContainsAny(e.Referencia, "\x00\r\n") || e.Referencia != m.EvidenciaRef ||
		m.TerminalReciboRef == "" || m.TerminalReciboRef == m.Reserva.ReservaReciboRef {
		return ports.ErrUsoCategoriaRPTInvalido
	}
	h := sha256.Sum256(e.Bytes)
	if hex.EncodeToString(h[:]) != m.EvidenciaSHA256 {
		return ports.ErrUsoCategoriaRPTInvalido
	}
	return nil
}

func (r *Recorrido) Reserva(ctx context.Context, m ports.MaterialReservaUsoCategoriaRPT) (ports.ResultadoUsoCategoriaRPT, error) {
	return r.actuar(ctx, accionReserva, m, ports.MaterialTerminalUsoCategoriaRPT{})
}
func (r *Recorrido) Confirmacion(ctx context.Context, m ports.MaterialTerminalUsoCategoriaRPT, e EvidenciaSintetica) (ports.ResultadoUsoCategoriaRPT, error) {
	if err := validarEvidencia(m, e); err != nil {
		return ports.ResultadoUsoCategoriaRPT{}, err
	}
	return r.actuar(ctx, accionConfirmacion, m.Reserva, m)
}
func (r *Recorrido) Cancelacion(ctx context.Context, m ports.MaterialTerminalUsoCategoriaRPT, e EvidenciaSintetica) (ports.ResultadoUsoCategoriaRPT, error) {
	if err := validarEvidencia(m, e); err != nil {
		return ports.ResultadoUsoCategoriaRPT{}, err
	}
	return r.actuar(ctx, accionCancelacion, m.Reserva, m)
}

func (r *Recorrido) actuar(ctx context.Context, accion string, m ports.MaterialReservaUsoCategoriaRPT, terminal ports.MaterialTerminalUsoCategoriaRPT) (ports.ResultadoUsoCategoriaRPT, error) {
	var cero ports.ResultadoUsoCategoriaRPT
	if ctx == nil || r == nil || nulo(r.preparador) || nulo(r.emisor) || nulo(r.gestor) {
		return cero, ports.ErrUsoCategoriaRPTNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	var p ports.PreparacionAutorizacionUsoCategoriaRPT
	var err error
	switch accion {
	case accionReserva:
		p, err = r.preparador.PrepararReservaUsoCategoriaRPT(ctx, m)
	case accionConfirmacion:
		p, err = r.preparador.PrepararConfirmacionUsoCategoriaRPT(ctx, terminal)
	case accionCancelacion:
		p, err = r.preparador.PrepararCancelacionUsoCategoriaRPT(ctx, terminal)
	}
	if err != nil {
		return cero, err
	}
	if !preparacionValida(p, accion, m) {
		return cero, ports.ErrUsoCategoriaRPTDenegado
	}
	// La copia evita que un emisor altere los mapas del recurso ya preparado.
	copia := p
	copia.Recurso.Ambitos = maps.Clone(p.Recurso.Ambitos)
	copia.Recurso.Atributos = maps.Clone(p.Recurso.Atributos)
	s, a, err := r.emisor.Emitir(ctx, copia)
	if err != nil {
		return cero, err
	}
	if err := cotejarMaterial(p, s, a); err != nil {
		return cero, err
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	// Es una guarda local de ensayo. El consumo único durable sigue en AD3.
	dref := a.ResumenCapacidad().DecisionRef()
	r.mu.Lock()
	if r.decisiones == nil || len(r.decisiones) >= maximoActos {
		r.mu.Unlock()
		return cero, ports.ErrUsoCategoriaRPTNoDisponible
	}
	if _, usada := r.decisiones[dref]; usada {
		r.mu.Unlock()
		return cero, ports.ErrUsoCategoriaRPTDenegado
	}
	r.decisiones[dref] = struct{}{}
	r.mu.Unlock()
	var out ports.ResultadoUsoCategoriaRPT
	switch accion {
	case accionReserva:
		out, err = r.gestor.ReservarUsoCategoriaRPT(ctx, ports.OrdenReservaUsoCategoriaRPT{Material: m, Solicitud: s, Autorizacion: a})
	case accionConfirmacion:
		out, err = r.gestor.ConfirmarUsoCategoriaRPT(ctx, ports.OrdenConfirmacionUsoCategoriaRPT{Material: terminal, Solicitud: s, Autorizacion: a})
	case accionCancelacion:
		out, err = r.gestor.CancelarUsoCategoriaRPT(ctx, ports.OrdenCancelacionUsoCategoriaRPT{Material: terminal, Solicitud: s, Autorizacion: a})
	}
	if err != nil {
		return out, err
	}
	if !resultadoValido(out, m, terminal, accion, a.ResumenCapacidad()) {
		return out, ports.ErrUsoCategoriaRPTNoConfiable
	}
	return out, nil
}

func preparacionValida(p ports.PreparacionAutorizacionUsoCategoriaRPT, accion string, m ports.MaterialReservaUsoCategoriaRPT) bool {
	x := p.Recurso
	h, err := hex.DecodeString(x.Atributos["material_sha256"])
	return p.Accion == accion && p.Finalidad == finalidadUsos && p.AudienciaConsumo == audienciaUsos && x.Validar() == nil &&
		x.Tipo == "uso_categoria" && x.Referencia == m.UsoRef && x.ModuloID == x.Ambitos["modulo_id"] &&
		len(x.Ambitos) == 3 && x.Ambitos["catalogo_id"] == m.Publicacion.CatalogoID && x.Ambitos["consumidor"] == m.Consumidor &&
		len(x.Atributos) == 1 && err == nil && len(h) == sha256.Size && !bytes.Equal(h, make([]byte, sha256.Size)) && x.Atributos["material_sha256"] == hex.EncodeToString(h)
}

// Este cotejo no verifica COSE, MAC, gobierno, vigencia ni consumo. El emisor
// nominal y el gestor AD3 conservan esas autoridades; el parser es informativo.
func cotejarMaterial(p ports.PreparacionAutorizacionUsoCategoriaRPT, s domain.SolicitudAutorizacionLigadaV3, a ports.ExportacionMaterialConsumoAutorizacionAtestadaV3) error {
	d, err := s.Datos()
	if err != nil || a.ValidarEstructura() != nil || d.Accion != p.Accion || d.Finalidad != p.Finalidad ||
		d.Recurso.Referencia != p.Recurso.Referencia || d.Recurso.ModuloID != p.Recurso.ModuloID || d.Recurso.Tipo != p.Recurso.Tipo ||
		!maps.Equal(d.Recurso.Ambitos, p.Recurso.Ambitos) || !maps.Equal(d.Recurso.Atributos, p.Recurso.Atributos) {
		return ports.ErrUsoCategoriaRPTDenegado
	}
	h, err := d.Recurso.HuellaContextoAutorizacionSHA256()
	resumen := a.ResumenCapacidad()
	v, errV := d.VinculoAutenticacionActor.Datos()
	motivo, errM := domain.RepresentacionCanonicaMotivoAutorizacionV2(d.ReferenciaMotivo)
	ctx, errCtx := domain.RehidratarContextoActorVinculadoV2(a.ContextoActorCanonico())
	proyeccion, errP := domain.ParsearMensajeAtestacionAutorizacionV3NoAutoritativo(a.PayloadVECAD3())
	corr, errC := d.Correlacion.ValorCanonico()
	corrPayload, errCP := proyeccion.CorrelacionRef()
	decisionRef, errD := proyeccion.DecisionRef()
	huellaDecision, errHD := proyeccion.HuellaDecisionSHA256()
	huellaMotivo, errHM := proyeccion.HuellaMotivoSHA256()
	contextoRef, errCR := proyeccion.ReferenciaContextoActor()
	huellaContexto, errHC := proyeccion.HuellaContextoActorSHA256()
	if err != nil || errV != nil || errM != nil || errCtx != nil || errP != nil || errC != nil || errCP != nil || errD != nil || errHD != nil || errHM != nil || errCR != nil || errHC != nil ||
		resumen.Operacion() != p.Accion || resumen.EfectoRef() != p.Recurso.Referencia || resumen.EfectoHuellaSHA256() != h ||
		resumen.AudienciaConsumo() != p.AudienciaConsumo || !bytes.Equal(motivo, a.MotivoCanonico()) ||
		hash(motivo) != resumen.MotivoHuellaSHA256() || hash(a.DecisionCanonica()) != resumen.DecisionHuellaSHA256() ||
		hash(a.ContextoActorCanonico()) != resumen.ContextoHuellaSHA256() ||
		v.RegistroContextoRef != resumen.ContextoRef() || v.ContextoActorHuellaSHA256 != resumen.ContextoHuellaSHA256() ||
		v.PrincipalID != ctx.Principal.ID || v.PerfilActivoRef != ctx.PerfilActivoRef ||
		v.ContextoActorRef != ctx.Instantanea.VinculoRef || v.ContextoActorVersion != ctx.Instantanea.VinculoVersion ||
		v.ContextoActorCuentaVersion != ctx.Instantanea.CuentaVersion ||
		a.PersonaVersion() != ctx.Instantanea.PersonaVersion || a.PerfilVersion() != ctx.Instantanea.PerfilVersion ||
		corr != corrPayload || decisionRef != resumen.DecisionRef() ||
		huellaDecision != resumen.DecisionHuellaSHA256() || huellaMotivo != resumen.MotivoHuellaSHA256() ||
		contextoRef != resumen.ContextoRef() || huellaContexto != resumen.ContextoHuellaSHA256() {
		return ports.ErrUsoCategoriaRPTDenegado
	}
	return nil
}

func hash(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

func resultadoValido(o ports.ResultadoUsoCategoriaRPT, m ports.MaterialReservaUsoCategoriaRPT, t ports.MaterialTerminalUsoCategoriaRPT, accion string, a ports.ResumenCapacidadAtestacionAutorizacionV3) bool {
	if !o.Encontrado || o.Uso == nil {
		return false
	}
	u, e := o.Uso, o.Evidencia
	if u.Consumidor != m.Consumidor || u.UsoRef != m.UsoRef || u.CategoriaID != m.CategoriaID || u.Publicacion != m.Publicacion ||
		u.ReservaReciboRef != m.ReservaReciboRef || u.ReservadoEn.IsZero() ||
		e.DecisionRef != a.DecisionRef() || e.EfectoRef != a.EfectoRef() || e.HuellaEfectoSHA256 != a.EfectoHuellaSHA256() ||
		e.AuditoriaRef == "" || e.ConsumoHuellaSHA256 == "" || e.ConsumidaEn.IsZero() || !e.ConsumoNuevo {
		return false
	}
	switch u.Estado {
	case "reservado":
		return accion == accionReserva && u.Revision == 1 && u.TerminalReciboRef == nil && u.TerminalEn == nil
	case "confirmado", "cancelado":
		if u.Revision != 2 || u.TerminalReciboRef == nil || *u.TerminalReciboRef == "" || u.TerminalEn == nil || u.TerminalEn.Before(u.ReservadoEn) {
			return false
		}
		if accion == accionReserva {
			return true
		}
		return *u.TerminalReciboRef == t.TerminalReciboRef && ((accion == accionConfirmacion && u.Estado == "confirmado") || (accion == accionCancelacion && u.Estado == "cancelado"))
	}
	return false
}

// CotejarReplay compara los usos terminales completos y exige nueva evidencia
// AD3 en los cuatro actos. La reserva repetida ya puede estar en estado terminal.
// No afirma haber consultado la historia de Cat3 ni tablas de otros módulos.
func CotejarReplay(anterior, actual Resultado) error {
	if !mismoUso(anterior.Confirmacion.Uso, actual.Confirmacion.Uso) ||
		!mismoUso(anterior.Cancelacion.Uso, actual.Cancelacion.Uso) ||
		!mismoUso(actual.ReservaConfirmacion.Uso, actual.Confirmacion.Uso) ||
		!mismoUso(actual.ReservaCancelacion.Uso, actual.Cancelacion.Uso) {
		return ports.ErrUsoCategoriaRPTNoConfiable
	}
	prev := []ports.ResultadoUsoCategoriaRPT{anterior.ReservaConfirmacion, anterior.Confirmacion, anterior.ReservaCancelacion, anterior.Cancelacion}
	curr := []ports.ResultadoUsoCategoriaRPT{actual.ReservaConfirmacion, actual.Confirmacion, actual.ReservaCancelacion, actual.Cancelacion}
	refs := make(map[string]bool, 8)
	for i := range prev {
		for _, e := range []ports.EvidenciaLecturaRPT{prev[i].Evidencia, curr[i].Evidencia} {
			if e.DecisionRef == "" || refs[e.DecisionRef] || !e.ConsumoNuevo {
				return ports.ErrUsoCategoriaRPTNoConfiable
			}
			refs[e.DecisionRef] = true
		}
	}
	return nil
}

func mismoUso(a, b *ports.UsoCategoriaRPT) bool {
	if a == nil || b == nil {
		return false
	}
	return a.Consumidor == b.Consumidor && a.UsoRef == b.UsoRef && a.CategoriaID == b.CategoriaID && a.Publicacion == b.Publicacion &&
		a.Estado == b.Estado && a.Revision == b.Revision && a.ReservaReciboRef == b.ReservaReciboRef &&
		a.ReservadoEn.Equal(b.ReservadoEn) && mismoTexto(a.TerminalReciboRef, b.TerminalReciboRef) && mismoInstante(a.TerminalEn, b.TerminalEn)
}
func mismoTexto(a, b *string) bool {
	return (a == nil && b == nil) || (a != nil && b != nil && *a == *b)
}
func mismoInstante(a, b *time.Time) bool {
	return (a == nil && b == nil) || (a != nil && b != nil && a.Equal(*b))
}
