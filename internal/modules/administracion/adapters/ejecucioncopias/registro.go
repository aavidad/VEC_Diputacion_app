// Package ejecucioncopias contains adapters for the typed copy execution ports.
package ejecucioncopias

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"

	"vec-diputacion-granada/internal/modules/administracion/domain/operacionescopias"
	ej "vec-diputacion-granada/internal/modules/administracion/ports/ejecucioncopias"
	cs07 "vec-diputacion-granada/internal/modules/administracion/ports/registrocopias"
)

// RegistroCS07 translates the CS11 execution contract to the independent CS07
// declared-progress journal. It deliberately has no authorization dependency:
// authority is consumed by the application before this adapter is called.
type RegistroCS07 struct {
	registro cs07.Registro
	destino  ej.Destino
	diario   *diarioExterior
}

type ConfigRegistroCS07 struct {
	Registro           cs07.Registro
	Destino            ej.Destino
	DirectorioExterior string
	RaicesRestauradas  []string
}

func AbrirRegistroCS07(c ConfigRegistroCS07) (*RegistroCS07, error) {
	if c.Registro == nil || c.Destino == nil {
		return nil, errRegistroConfiguracion
	}
	d, err := abrirDiarioExterior(c.DirectorioExterior, c.RaicesRestauradas)
	if err != nil {
		return nil, err
	}
	return &RegistroCS07{registro: c.Registro, destino: c.Destino, diario: d}, nil
}

func (r *RegistroCS07) Close() error {
	if r == nil || r.diario == nil {
		return nil
	}
	return r.diario.Close()
}

func (r *RegistroCS07) Reservar(ctx context.Context, p ej.Peticion) (ej.Operacion, error) {
	if err := validarPeticion(p); err != nil {
		return ej.Operacion{}, err
	}
	if previo, err := r.diario.leer(ctx, p.OperacionRef); err == nil {
		if previo.Actor != p.ActorRef || previo.ConjuntoRef != p.ConjuntoRef || previo.DestinoRef != p.DestinoRef || previo.PoliticaRef != p.PoliticaRef {
			return ej.Operacion{}, errRegistroVinculo
		}
		res, err := r.registro.Consultar(ctx, previo.declaracion(), p.OperacionRef)
		if err != nil || res.Solicitud != solicitudCS07(p) {
			return ej.Operacion{}, errRegistroVinculo
		}
		return operacionDesde(previo), nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return ej.Operacion{}, err
	}
	s := solicitudCS07(p)
	d := declaracion(p)
	res, err := r.registro.Reservar(ctx, d, s)
	if err != nil {
		return ej.Operacion{}, err
	}
	estado := estadoExterior{Ref: p.OperacionRef, Actor: p.ActorRef, Correlacion: p.OperacionRef, SolicitudSHA256: s.SHA256, ConjuntoRef: p.ConjuntoRef, DestinoRef: p.DestinoRef, PoliticaRef: p.PoliticaRef, Estado: string(res.Recibo.Estado), Version: res.Recibo.Version}
	if err := r.diario.guardar(ctx, "reserva_copia", estado); err != nil {
		return ej.Operacion{}, err
	}
	return operacionDesde(estado), nil
}

func (r *RegistroCS07) AplicarCopia(ctx context.Context, e ej.EventoCopia) error {
	if r == nil || r.registro == nil || r.diario == nil || e.OperacionRef == "" {
		return errRegistroConfiguracion
	}
	estado, err := r.diario.leer(ctx, e.OperacionRef)
	if err != nil {
		return err
	}
	res, err := r.registro.Consultar(ctx, estado.declaracion(), e.OperacionRef)
	if err != nil {
		return err
	}
	c, err := comandoCS07(e, res)
	if err != nil {
		return err
	}
	aplicado := res
	encontrado := false
	for _, evento := range res.Historia {
		if evento.Comando.Clave == c.Clave {
			c.VersionEsperada = evento.VersionPrevia
			if !reflect.DeepEqual(evento.Comando, c) {
				return errRegistroVinculo
			}
			aplicado.Recibo.Version, aplicado.Recibo.Estado, encontrado = evento.Version, evento.Estado, true
			break
		}
	}
	if !encontrado {
		aplicado, err = r.registro.Aplicar(ctx, estado.declaracion(), e.OperacionRef, c)
		if err != nil {
			return err
		}
	}
	estado.Estado, estado.Version = string(aplicado.Recibo.Estado), aplicado.Recibo.Version
	return r.diario.guardar(ctx, "aplicar_copia", estado)
}

func (r *RegistroCS07) ReservarRestauracion(ctx context.Context, p ej.Propuesta) (ej.Operacion, error) {
	if err := validarPeticion(p.Peticion); err != nil || p.HuellaPropuesta == "" || p.PreimagenSHA256 == "" || p.ConjuntoPreviaRef == "" || p.ConjuntoPreviaRef == p.ConjuntoRef {
		return ej.Operacion{}, errRegistroEntrada
	}
	estado := estadoExterior{Ref: p.OperacionRef, Actor: p.ActorRef, Correlacion: p.OperacionRef, SolicitudSHA256: huellaSemantica(p.Peticion), ConjuntoRef: p.ConjuntoRef, ConjuntoPreviaPlaneadaRef: p.ConjuntoPreviaRef, DestinoRef: p.DestinoRef, PoliticaRef: p.PoliticaRef, Estado: "solicitada", HuellaPropuesta: p.HuellaPropuesta, PreimagenSHA256: p.PreimagenSHA256}
	if err := r.diario.reservarRestauracion(ctx, estado); err != nil {
		return ej.Operacion{}, err
	}
	actual, err := r.diario.leer(ctx, p.OperacionRef)
	if err != nil {
		return ej.Operacion{}, err
	}
	return operacionDesde(actual), nil
}

func (r *RegistroCS07) Anotar(ctx context.Context, ref, etapa, valor string) error {
	if r == nil || r.diario == nil {
		return errRegistroConfiguracion
	}
	return r.diario.actualizar(ctx, ref, "anotar", func(estado estadoExterior) (estadoExterior, error) {
		if etapa == "valida" {
			if estado.Estado != string(operacionescopias.VerificadaDeclarada) {
				return estado, errRegistroTransicion
			}
			actual, err := r.registro.Consultar(ctx, estado.declaracion(), ref)
			if err != nil || actual.Solicitud.SHA256 != estado.SolicitudSHA256 || actual.Recibo.Estado != operacionescopias.VerificadaDeclarada || actual.Recibo.Version != estado.Version {
				return estado, errRegistroVinculo
			}
			c, err := r.destino.Recuperar(ctx, estado.ConjuntoRef)
			if err != nil || c.Ref != estado.ConjuntoRef || c.IndiceAutenticadoRef != valor || c.Manifiesto.ConjuntoRef != estado.ConjuntoRef || c.Manifiesto.OperacionRef != ref || c.Manifiesto.PoliticaRef != estado.PoliticaRef || c.Manifiesto.Verificacion.Estado != "valida" {
				return estado, errRegistroVinculo
			}
		}
		if etapa == "copia_previa_verificada" {
			if valor != estado.ConjuntoPreviaPlaneadaRef {
				return estado, errRegistroVinculo
			}
			c, err := r.destino.Recuperar(ctx, valor)
			if err != nil || c.Ref != valor || c.Manifiesto.ConjuntoRef != valor || c.Manifiesto.PoliticaRef != estado.PoliticaRef || c.Manifiesto.Verificacion.Estado != "valida" {
				return estado, errRegistroVinculo
			}
		}
		if !etapaPermitida(estado.Estado, etapa, valor, estado) {
			return estado, errRegistroTransicion
		}
		estado.Estado = etapa
		switch etapa {
		case "copia_previa_verificada", "reversion_iniciada", "revertida":
			estado.CopiaPreviaRef = valor
		case "plan_preparado":
			estado.PlanRef = valor
		case "valida":
			estado.IndiceFinalRef = valor
		}
		return estado, nil
	})
}

func (r *RegistroCS07) CAS(ctx context.Context, ref, esperada, preimagen, etapa string) (ej.Operacion, error) {
	if etapa != "sustitucion_iniciada" {
		return ej.Operacion{}, errRegistroTransicion
	}
	estado, err := r.diario.cas(ctx, ref, esperada, preimagen, etapa)
	if err != nil {
		return ej.Operacion{}, err
	}
	return operacionDesde(estado), nil
}

func (r *RegistroCS07) Leer(ctx context.Context, ref string) (ej.Operacion, error) {
	if r == nil || r.diario == nil || r.registro == nil {
		return ej.Operacion{}, errRegistroConfiguracion
	}
	estado, err := r.diario.leer(ctx, ref)
	if err != nil {
		return ej.Operacion{}, err
	}
	if estado.Estado == "valida" {
		actual, err := r.registro.Consultar(ctx, estado.declaracion(), ref)
		if err != nil || actual.Solicitud.SHA256 != estado.SolicitudSHA256 || actual.Recibo.Estado != operacionescopias.VerificadaDeclarada || actual.Recibo.Version+1 != estado.Version {
			return ej.Operacion{}, errRegistroVinculo
		}
	}
	return operacionDesde(estado), nil
}

func solicitudCS07(p ej.Peticion) operacionescopias.Solicitud {
	return operacionescopias.Solicitud{Operacion: p.OperacionRef, Clave: "cs11:" + huellaSemantica(p)[:56], SHA256: huellaSemantica(p), Conjunto: p.ConjuntoRef, Destino: p.DestinoRef, Politica: p.PoliticaRef}
}
func declaracion(p ej.Peticion) cs07.Declaracion {
	return cs07.Declaracion{Actor: p.ActorRef, Correlacion: p.OperacionRef}
}
func (e estadoExterior) declaracion() cs07.Declaracion {
	return cs07.Declaracion{Actor: e.Actor, Correlacion: e.Correlacion}
}

func huellaSemantica(p ej.Peticion) string {
	b, _ := json.Marshal(struct{ Operacion, Actor, Origen, Destino, Motivo, Conjunto, Politica string }{p.OperacionRef, p.ActorRef, p.OrigenRef, p.DestinoRef, p.MotivoRef, p.ConjuntoRef, p.PoliticaRef})
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func claveEvento(e ej.EventoCopia) string {
	b, _ := json.Marshal(e)
	h := sha256.Sum256(b)
	return "cs11:" + hex.EncodeToString(h[:])[:56]
}
func huellaEvidencia(v any) string {
	b, _ := json.Marshal(v)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func comandoCS07(e ej.EventoCopia, r cs07.Resultado) (operacionescopias.Comando, error) {
	c := operacionescopias.Comando{Clave: claveEvento(e), VersionEsperada: r.Recibo.Version, SolicitudSHA256: r.Solicitud.SHA256, Accion: e.Transicion}
	switch e.Transicion {
	case "iniciar_captura":
		if e.ConjuntoRef != r.Solicitud.Conjunto || e.ManifiestoSHA256 != "" || e.EjecucionRef != "" || e.Ensayo != nil {
			return c, errRegistroVinculo
		}
	case "confirmar_captura":
		if e.ConjuntoRef != r.Solicitud.Conjunto || e.ManifiestoSHA256 == "" || e.Ensayo != nil {
			return c, errRegistroEntrada
		}
		c.ManifiestoSHA256 = e.ManifiestoSHA256
	case "iniciar_verificacion":
		if e.ConjuntoRef != r.Solicitud.Conjunto || e.ManifiestoSHA256 == "" || e.EjecucionRef == "" || e.Ensayo != nil {
			return c, errRegistroEntrada
		}
		c.ManifiestoSHA256, c.Ejecucion = e.ManifiestoSHA256, e.EjecucionRef
	case "declarar_ensayo":
		if e.Ensayo == nil || e.ConjuntoRef != r.Solicitud.Conjunto || e.ManifiestoSHA256 == "" || e.EjecucionRef == "" {
			return c, errRegistroEntrada
		}
		x := e.Ensayo
		if x.Modo != ej.Fisico && x.Modo != ej.Logico || x.Evidencia.Ref == "" || (x.Resultado != "satisfactorio" && x.Resultado != "fallido") {
			return c, errRegistroVinculo
		}
		c.Evidencia = &operacionescopias.Evidencia{Modo: string(x.Modo), Conjunto: e.ConjuntoRef, ManifiestoSHA256: e.ManifiestoSHA256, Ejecucion: e.EjecucionRef, Referencia: x.Evidencia.Ref, SHA256: huellaEvidencia(x.Evidencia), Resultado: x.Resultado}
	default:
		return c, errRegistroTransicion
	}
	return c, nil
}

func validarPeticion(p ej.Peticion) error {
	if p.OperacionRef == "" || p.ActorRef == "" || p.DestinoRef == "" || p.ConjuntoRef == "" || p.PoliticaRef == "" {
		return errRegistroEntrada
	}
	return nil
}
func operacionDesde(e estadoExterior) ej.Operacion {
	return ej.Operacion{Ref: e.Ref, VersionRef: fmt.Sprintf("%d", e.Version), Estado: e.Estado, ConjuntoRef: e.ConjuntoRef, ConjuntoPreviaPlaneadaRef: e.ConjuntoPreviaPlaneadaRef, PoliticaRef: e.PoliticaRef, CopiaPreviaRef: e.CopiaPreviaRef, PlanRef: e.PlanRef, PreimagenSHA256: e.PreimagenSHA256, HuellaPropuesta: e.HuellaPropuesta, IndiceAutenticadoRef: e.IndiceFinalRef}
}
func etapaPermitida(estado, etapa, valor string, e estadoExterior) bool {
	if valor == "" {
		return false
	}
	switch etapa {
	case "exclusion_solicitada":
		return estado == "solicitada" && valor == e.PreimagenSHA256
	case "copia_previa_verificada":
		return estado == "exclusion_solicitada"
	case "plan_preparado":
		return estado == "copia_previa_verificada"
	case "pendiente_conciliacion":
		return estado == "sustitucion_iniciada" || estado == "reversion_iniciada"
	case "reversion_iniciada":
		return estado == "sustitucion_iniciada" || estado == "instalado_pendiente_conciliacion"
	case "revertida":
		return estado == "reversion_iniciada"
	case "instalado_pendiente_conciliacion":
		return estado == "sustitucion_iniciada"
	case "valida":
		return estado == string(operacionescopias.VerificadaDeclarada)
	case "no_valida":
		// Only a copy may be declared invalid by the capture path. A restoration
		// has a preimage and must remain reconcilable rather than being relabelled.
		return e.PreimagenSHA256 == "" && estado != "valida" && estado != "no_valida"
	}
	return false
}

var (
	errRegistroConfiguracion = errors.New("registro_ejecucion_configuracion_invalida")
	errRegistroEntrada       = errors.New("registro_ejecucion_entrada_invalida")
	errRegistroVinculo       = errors.New("registro_ejecucion_vinculo_distinto")
	errRegistroTransicion    = errors.New("registro_ejecucion_transicion_invalida")
)

var _ ej.Registro = (*RegistroCS07)(nil)
