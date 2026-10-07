package registrocopias

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"regexp"
	"sort"
	"time"

	app "vec-diputacion-granada/internal/modules/administracion/application/registrocopias"
	"vec-diputacion-granada/internal/modules/administracion/domain/operacionescopias"
	port "vec-diputacion-granada/internal/modules/administracion/ports/registrocopias"
)

type peticion struct {
	AbandonoDenegado bool                         `json:"abandono_denegado,omitempty"`
	Orden            *port.RecepcionOrden         `json:"orden,omitempty"`
	Accion           string                       `json:"accion"`
	Declaracion      port.Declaracion             `json:"declaracion"`
	Operacion        string                       `json:"operacion"`
	Solicitud        *operacionescopias.Solicitud `json:"solicitud,omitempty"`
	Comando          *operacionescopias.Comando   `json:"comando,omitempty"`
	Consulta         *port.Consulta               `json:"consulta,omitempty"`
}
type registro struct {
	Secuencia uint64                    `json:"secuencia"`
	Anterior  string                    `json:"anterior"`
	Instante  string                    `json:"instante"`
	Peticion  peticion                  `json:"peticion"`
	Recibo    port.Recibo               `json:"recibo"`
	Auditoria port.Auditoria            `json:"auditoria"`
	Evento    *operacionescopias.Evento `json:"evento,omitempty"`
}
type trama struct {
	Registro registro `json:"registro"`
	SHA256   string   `json:"sha256"`
}
type estado struct {
	Solicitud operacionescopias.Solicitud
	Operacion operacionescopias.Operacion
	Reserva   port.Recibo
	Recibos   map[string]port.Recibo
	Ultimo    port.Recibo
}
type motor struct {
	ordenes           map[string]port.Aceptacion
	ordenPorOperacion map[string]string
	fences            map[string]port.RecepcionOrden
	operaciones       map[string]*estado
	claves            map[string]string
	secuencia         uint64
	anterior          string
}

func nuevoMotor() *motor {
	return &motor{operaciones: make(map[string]*estado), claves: make(map[string]string), ordenes: make(map[string]port.Aceptacion), ordenPorOperacion: make(map[string]string), fences: make(map[string]port.RecepcionOrden)}
}

var referencia = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9:_-]{0,95}$`)
var huella = regexp.MustCompile(`^[a-f0-9]{64}$`)

// Shape checks keep paths, free prose and credentials out of journal records.
func validar(p peticion) error {
	if app.ValidarDeclaracion(p.Declaracion) != nil || (p.Accion != "listar" && !referencia.MatchString(p.Operacion)) || (p.Accion != "listar" && p.Consulta != nil) || (p.Accion != "aceptar_orden" && p.Orden != nil) || (p.Accion != "abandonar_captura" && p.AbandonoDenegado) {
		return port.ErrEntrada
	}
	switch p.Accion {
	case "aceptar_orden":
		if p.Solicitud != nil || p.Comando != nil || p.Orden == nil || p.Orden.Operacion != p.Operacion {
			return port.ErrEntrada
		}
		return validarOrden(*p.Orden)
	case "listar":
		if p.Operacion != "" || p.Solicitud != nil || p.Comando != nil || p.Consulta == nil || p.Consulta.Limite < 1 || p.Consulta.Limite > 100 || (p.Consulta.Despues != "" && !referencia.MatchString(p.Consulta.Despues)) {
			return port.ErrEntrada
		}
	case "reservar":
		if p.Solicitud == nil || p.Comando != nil || p.Solicitud.Operacion != p.Operacion {
			return port.ErrEntrada
		}
		_, err := operacionescopias.Nueva(*p.Solicitud)
		return err
	case "consultar":
		if p.Solicitud != nil || p.Comando != nil {
			return port.ErrEntrada
		}
	case "aplicar", "abandonar_captura":
		if p.Solicitud != nil || p.Comando == nil {
			return port.ErrEntrada
		}
		c := p.Comando
		if p.Accion == "aplicar" && (c.Abandono != nil || c.Accion == "abandonar_captura") {
			return port.ErrEntrada
		}
		if p.Accion == "abandonar_captura" && (c.Accion != "abandonar_captura" || c.Abandono == nil || c.Abandono.Operacion != p.Operacion || !observacionSegura(*c.Abandono)) {
			return port.ErrEntrada
		}
		if !referencia.MatchString(c.Clave) || !huella.MatchString(c.SolicitudSHA256) || !referencia.MatchString(c.Accion) || (c.ManifiestoSHA256 != "" && !huella.MatchString(c.ManifiestoSHA256)) || (c.Ejecucion != "" && !referencia.MatchString(c.Ejecucion)) {
			return port.ErrEntrada
		}
		if e := c.Evidencia; e != nil {
			if !referencia.MatchString(e.Conjunto) || !huella.MatchString(e.ManifiestoSHA256) || !referencia.MatchString(e.Ejecucion) || !referencia.MatchString(e.Referencia) || !huella.MatchString(e.SHA256) || !referencia.MatchString(e.Modo) || !referencia.MatchString(e.Resultado) {
				return port.ErrEntrada
			}
		}
	default:
		return port.ErrEntrada
	}
	return nil
}

func (m *motor) procesar(p peticion, instante string) (port.Resultado, *operacionescopias.Evento, error) {
	a := port.Auditoria{Secuencia: m.secuencia + 1, Declaracion: p.Declaracion, Accion: p.Accion, Operacion: p.Operacion, Instante: instante, Autoridad: "actor_declarado_sin_autorizacion"}
	a.Referencia = identidad("auditoria", a)
	res := port.Resultado{Auditoria: a}
	s := m.operaciones[p.Operacion]
	var evento *operacionescopias.Evento
	var err error
	switch p.Accion {
	case "aceptar_orden":
		res.Recibo, res.Replay, err = m.aceptarOrden(*p.Orden, a)
	case "listar":
		keys := make([]string, 0, len(m.operaciones))
		for key := range m.operaciones {
			if key > p.Consulta.Despues {
				keys = append(keys, key)
			}
		}
		sort.Strings(keys)
		if len(keys) > p.Consulta.Limite {
			keys = keys[:p.Consulta.Limite]
			res.Siguiente = keys[len(keys)-1]
		}
		res.Operaciones = make([]port.Vista, 0, len(keys))
		for _, key := range keys {
			e := m.operaciones[key]
			res.Operaciones = append(res.Operaciones, port.Vista{Solicitud: e.Solicitud, Recibo: e.Ultimo, Reconciliacion: e.Operacion.Reconciliar()})
		}
	case "reservar":
		if op, ok := m.claves[p.Solicitud.Clave]; ok && op != p.Operacion {
			err = operacionescopias.ErrConflicto
			break
		}
		if s == nil {
			for _, existente := range m.operaciones {
				if existente.Solicitud.Destino == p.Solicitud.Destino && existente.Operacion.Estado() != operacionescopias.VerificadaDeclarada && existente.Operacion.Estado() != operacionescopias.NoValidaDeclarada && existente.Operacion.Estado() != operacionescopias.AbandonadaDeclarada {
					err = port.ErrDestinoOcupado
					break
				}
			}
			if err != nil {
				res.Auditoria.Resultado = err.Error()
				return res, nil, err
			}
		}
		var anterior *operacionescopias.Operacion
		if s != nil {
			anterior = &s.Operacion
		}
		o, replay, e := operacionescopias.Reservar(anterior, *p.Solicitud)
		err, res.Replay = e, replay
		if e == nil && !replay {
			recibo := port.Recibo{Instante: instante, Version: 0, Estado: o.Estado()}
			recibo.Referencia = identidad("recibo", a)
			s = &estado{Solicitud: *p.Solicitud, Operacion: o, Reserva: recibo, Ultimo: recibo, Recibos: make(map[string]port.Recibo)}
			m.operaciones[p.Operacion], m.claves[p.Solicitud.Clave] = s, p.Operacion
		}
		if e == nil {
			res.Recibo = s.Reserva
		}
	case "aplicar", "abandonar_captura":
		if p.AbandonoDenegado {
			err = port.ErrAbandonoNoAutorizado
			break
		}
		if s == nil {
			err = port.ErrNoExiste
			break
		}
		next, e, replay, fallo := s.Operacion.Aplicar(*p.Comando)
		err, res.Replay = fallo, replay
		if fallo == nil {
			if !replay {
				recibo := port.Recibo{Instante: instante, Version: e.Version, Estado: e.Estado}
				recibo.Referencia = identidad("recibo", a)
				s.Operacion, s.Ultimo, s.Recibos[p.Comando.Clave] = next, recibo, recibo
				evento = &e
			}
			res.Recibo = s.Recibos[p.Comando.Clave]
		}
	case "consultar":
		if s == nil {
			err = port.ErrNoExiste
			break
		}
		res.Recibo = s.Ultimo
	}
	if err != nil {
		res.Auditoria.Resultado = err.Error()
	} else if res.Replay {
		res.Auditoria.Resultado = "replay"
	} else {
		res.Auditoria.Resultado = "registrado"
	}
	if s != nil {
		res.Solicitud, res.Historia, res.Reconciliacion = s.Solicitud, s.Operacion.Historia(), s.Operacion.Reconciliar()
	}
	return res, evento, err
}

func identidad(prefijo string, v any) string {
	b, _ := json.Marshal(v)
	h := sha256.Sum256(b)
	return prefijo + ":" + hex.EncodeToString(h[:])
}
func sello(r registro) string {
	b, _ := json.Marshal(r)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func (m *motor) recuperar(t trama) error {
	r := t.Registro
	instante, err := time.Parse(time.RFC3339Nano, r.Instante)
	if err != nil || instante.UTC().Format(time.RFC3339Nano) != r.Instante || validar(r.Peticion) != nil || r.Secuencia != m.secuencia+1 || r.Anterior != m.anterior || sello(r) != t.SHA256 {
		return port.ErrCorrupto
	}
	res, evento, _ := m.procesar(r.Peticion, r.Instante)
	if !reflect.DeepEqual(res.Auditoria, r.Auditoria) || !reflect.DeepEqual(res.Recibo, r.Recibo) || !reflect.DeepEqual(evento, r.Evento) {
		return port.ErrCorrupto
	}
	m.secuencia, m.anterior = r.Secuencia, t.SHA256
	return nil
}

// Codigo maps only nominal errors; callers never expose filesystem error text.
func Codigo(err error) string {
	for _, e := range []error{port.ErrConfiguracion, port.ErrEntrada, port.ErrCorrupto, port.ErrNoExiste, port.ErrIO, port.ErrDestinoOcupado, port.ErrAbandonoNoAutorizado, operacionescopias.ErrAbandono, port.ErrOrdenConflicto, port.ErrFence, operacionescopias.ErrEntrada, operacionescopias.ErrConflicto, operacionescopias.ErrVersion, operacionescopias.ErrVinculo, operacionescopias.ErrTransicion, operacionescopias.ErrHistoria} {
		if errors.Is(err, e) {
			return e.Error()
		}
	}
	return port.ErrIO.Error()
}
