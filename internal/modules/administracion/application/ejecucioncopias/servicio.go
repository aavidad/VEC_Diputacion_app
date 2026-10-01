// Package ejecucioncopias coordina las autoridades externas de copia y
// restauración sin ejecutar SQL, shell ni sustituir servicios por sí mismo.
package ejecucioncopias

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"time"

	"vec-diputacion-granada/internal/modules/administracion/domain/copias"
	puertos "vec-diputacion-granada/internal/modules/administracion/ports/ejecucioncopias"
)

var (
	ErrBloqueada    = errors.New("operacion_bloqueada")
	ErrConciliacion = errors.New("conciliacion_pendiente")
)

// Dependencias proceden de la composición ADMIN; el servicio nunca construye
// dobles ni defaults operativos. Plataforma solo se omite en NuevoCopia.
type Dependencias struct {
	Inventario  puertos.Inventario
	Autorizador puertos.Autorizador
	Registro    puertos.Registro
	Ventana     puertos.Ventana
	Destino     puertos.Destino
	Ensayador   puertos.Ensayador
	Plataforma  puertos.Plataforma
}

type Servicio struct{ d Dependencias }

func Nuevo(d Dependencias) (*Servicio, error) {
	s, err := NuevoCopia(d)
	if err != nil {
		return nil, err
	}
	if d.Plataforma == nil {
		return nil, fmt.Errorf("dependencia_ausente: %w", ErrBloqueada)
	}
	return s, nil
}

// NuevoCopia compone el recorrido de captura y verificación sin capacidad de
// sustitución. Restaurar y Conciliar rechazan esta composición antes de actuar.
func NuevoCopia(d Dependencias) (*Servicio, error) {
	if d.Inventario == nil || d.Autorizador == nil || d.Registro == nil || d.Ventana == nil || d.Destino == nil || d.Ensayador == nil {
		return nil, fmt.Errorf("dependencia_ausente: %w", ErrBloqueada)
	}
	return &Servicio{d: d}, nil
}

type Recibo struct {
	OperacionRef, ConjuntoRef, Estado, IndiceAutenticadoRef string
}

func denegar(causa string) error { return fmt.Errorf("%s: %w", causa, ErrBloqueada) }

func (s *Servicio) lectura(ctx context.Context, destino string) (puertos.Lectura, error) {
	l, err := s.d.Inventario.LeerActual(ctx, destino)
	if err != nil {
		return l, err
	}
	if l.VersionRef == "" || copias.CompararInventarios(l.Esperado, l.Observado).Estado != copias.Compatible {
		return l, denegar("inventario_actual_incompatible")
	}
	if len(copias.ValidarPolitica(l.Politica)) != 0 {
		return l, denegar("politica_no_comprobable")
	}
	return l, nil
}

func concesionVigente(c puertos.Concesion, p puertos.Peticion, accion string) bool {
	recurso := p.DestinoRef
	if accion == "copiar" {
		recurso = p.OrigenRef
	}
	return c.ActorRef == p.ActorRef && c.Accion == accion && c.RecursoRef == recurso && c.DecisionRef != "" && time.Now().UTC().Before(c.Vence)
}

func (s *Servicio) autorizar(ctx context.Context, p puertos.Peticion, accion string) error {
	c, err := s.d.Autorizador.Autorizar(ctx, p, accion)
	if err != nil {
		return err
	}
	if !concesionVigente(c, p, accion) {
		return denegar("concesion_no_vigente")
	}
	return nil
}

func (s *Servicio) Copiar(ctx context.Context, p puertos.Peticion) (resultado Recibo, retErr error) {
	if p.OperacionRef == "" || p.ActorRef == "" || p.OrigenRef == "" || p.DestinoRef == "" || p.MotivoRef == "" || p.ConjuntoRef == "" || p.PoliticaRef == "" {
		return Recibo{}, denegar("peticion_incompleta")
	}
	if err := s.autorizar(ctx, p, "copiar"); err != nil {
		return Recibo{}, err
	}
	l, err := s.lectura(ctx, p.OrigenRef)
	if err != nil {
		return Recibo{}, err
	}
	if l.Politica.Ref != p.PoliticaRef {
		return Recibo{}, denegar("politica_distinta")
	}
	op, err := s.d.Registro.Reservar(ctx, p)
	if err != nil {
		return Recibo{}, err
	}
	if op.Ref != p.OperacionRef || op.VersionRef == "" || op.ConjuntoRef != p.ConjuntoRef || op.PoliticaRef != p.PoliticaRef {
		return Recibo{}, denegar("reserva_no_vinculada")
	}
	if op.Estado == "valida" {
		c, err := s.d.Destino.Recuperar(ctx, op.ConjuntoRef)
		if err != nil {
			return Recibo{}, ErrConciliacion
		}
		if err := validarConjunto(c, "valida"); err != nil {
			return Recibo{}, err
		}
		if err := vincularConjunto(c, p); err != nil {
			return Recibo{}, err
		}
		if op.IndiceAutenticadoRef != c.IndiceAutenticadoRef {
			return Recibo{}, denegar("indice_distinto_del_diario")
		}
		return recibo(op.Ref, c), nil
	}
	var c puertos.Conjunto
	recuperada := op.Estado != "solicitada"
	switch op.Estado {
	case "solicitada":
		if err := s.autorizar(ctx, p, "copiar"); err != nil {
			return Recibo{}, err
		}
		if err := s.d.Registro.AplicarCopia(ctx, puertos.EventoCopia{OperacionRef: op.Ref, Transicion: "iniciar_captura", ConjuntoRef: p.ConjuntoRef}); err != nil {
			return Recibo{}, err
		}
		captura, err := s.d.Ventana.Capturar(ctx, p, l)
		if err != nil {
			return Recibo{}, s.capturaFallida(ctx, op.Ref, "captura_fallida", err)
		}
		if err := validarCaptura(captura, l, p); err != nil {
			return Recibo{}, s.capturaFallida(ctx, op.Ref, "captura_no_comprobable", err)
		}
		c, err = s.d.Destino.Publicar(ctx, captura)
		if err != nil {
			return Recibo{}, ErrConciliacion
		}
		if err := validarConjunto(c, "pendiente_verificacion"); err != nil {
			return Recibo{}, s.publicadoFallido(ctx, op.Ref, c, err)
		}
		if c.Manifiesto.InventarioSHA256 != captura.Manifiesto.InventarioSHA256 || !reflect.DeepEqual(c.Manifiesto.Componentes, captura.Manifiesto.Componentes) || !reflect.DeepEqual(c.Origen, captura.Origen) {
			return Recibo{}, s.publicadoFallido(ctx, op.Ref, c, denegar("sello_otro_conjunto"))
		}
		op.Estado = "capturando"
	case "capturando", "capturada", "verificando", "verificada_declarada":
		c, err = s.d.Destino.Recuperar(ctx, p.ConjuntoRef)
		if err != nil {
			return Recibo{}, ErrConciliacion
		}
	case "captura_pendiente_conciliacion":
		return Recibo{}, s.capturaFallida(ctx, op.Ref, op.FalloCapturaRef, denegar("captura_fallida"))
	default:
		return Recibo{}, denegar("operacion_no_reanudable")
	}
	if err := vincularConjunto(c, p); err != nil {
		if recuperada {
			return Recibo{}, ErrConciliacion
		}
		return Recibo{}, err
	}
	c, err = s.reanudarPublicado(ctx, c, op.Estado)
	if err != nil {
		return Recibo{}, s.publicadoFallido(ctx, op.Ref, c, err)
	}
	if err := s.d.Registro.Anotar(ctx, op.Ref, "valida", c.IndiceAutenticadoRef); err != nil {
		return Recibo{}, ErrConciliacion
	}
	return recibo(op.Ref, c), nil
}

func (s *Servicio) capturaFallida(ctx context.Context, ref, fallo string, causa error) error {
	if fallo != "captura_fallida" && fallo != "captura_no_comprobable" && fallo != "verificacion_fallida" {
		return errors.Join(causa, ErrConciliacion)
	}
	ctx = context.WithoutCancel(ctx)
	if r, ok := s.d.Registro.(puertos.RegistroAbandono); ok {
		if err := r.AbandonarCaptura(ctx, ref, fallo); err == nil {
			return causa
		}
	}
	if err := s.d.Registro.Anotar(ctx, ref, "captura_pendiente_conciliacion", fallo); err != nil {
		return errors.Join(causa, ErrConciliacion, err)
	}
	return errors.Join(causa, ErrConciliacion)
}

func (s *Servicio) publicadoFallido(ctx context.Context, ref string, c puertos.Conjunto, causa error) error {
	ctx = context.WithoutCancel(ctx)
	r, ok := s.d.Registro.(puertos.RegistroFalloPublicado)
	if !ok {
		anotacion := s.d.Registro.Anotar(ctx, ref, "captura_pendiente_conciliacion", "verificacion_fallida")
		return errors.Join(causa, ErrConciliacion, anotacion)
	}
	if err := r.RegistrarFalloPublicado(ctx, ref, c); err != nil {
		anotacion := s.d.Registro.Anotar(ctx, ref, "captura_pendiente_conciliacion", "verificacion_fallida")
		return errors.Join(causa, ErrConciliacion, err, anotacion)
	}
	return s.capturaFallida(ctx, ref, "verificacion_fallida", causa)
}

func (s *Servicio) copiaRestaurable(ctx context.Context, c puertos.Conjunto) error {
	op, err := s.d.Registro.Leer(ctx, c.Manifiesto.OperacionRef)
	if err != nil {
		return err
	}
	if op.Estado != "valida" || op.ConjuntoRef != c.Ref || op.PoliticaRef != c.Manifiesto.PoliticaRef || op.IndiceAutenticadoRef != c.IndiceAutenticadoRef {
		return denegar("copia_no_validada_en_diario")
	}
	return nil
}

func validarCaptura(c puertos.Captura, l puertos.Lectura, p puertos.Peticion) error {
	m := c.Manifiesto
	if m.OperacionRef != p.OperacionRef || m.ConjuntoRef != p.ConjuntoRef || m.PoliticaRef != p.PoliticaRef || m.PoliticaRef != l.Politica.Ref || m.InventarioSHA256 != copias.HuellaInventario(m.Inventario) || m.Verificacion.Estado != "pendiente_verificacion" {
		return denegar("captura_sin_vinculo")
	}
	if copias.CompararInventarios(l.Observado, m.Inventario).Estado != copias.Compatible {
		return denegar("captura_otro_inventario")
	}
	if !m.Consistencia.EscritoresExcluidos || !m.Consistencia.ParadaLimpia || m.Consistencia.Modo != "fisica_fria_y_logica" {
		return denegar("ventana_no_acreditada")
	}
	if !huellasCompletas(c.Origen) {
		return denegar("origen_sin_contraste")
	}
	return nil
}

func recibo(op string, c puertos.Conjunto) Recibo {
	return Recibo{op, c.Ref, c.Manifiesto.Verificacion.Estado, c.IndiceAutenticadoRef}
}

func vincularConjunto(c puertos.Conjunto, p puertos.Peticion) error {
	m := c.Manifiesto
	if c.Ref != p.ConjuntoRef || m.ConjuntoRef != p.ConjuntoRef || m.OperacionRef != p.OperacionRef || m.SolicitanteRef != p.ActorRef || m.MotivoRef != p.MotivoRef || m.PoliticaRef != p.PoliticaRef {
		return denegar("conjunto_otro_vinculo")
	}
	return nil
}

func (s *Servicio) reanudarPublicado(ctx context.Context, c puertos.Conjunto, estado string) (puertos.Conjunto, error) {
	// Recuperar verifica de nuevo índice, manifiesto y todos los componentes.
	r, err := s.d.Destino.Recuperar(ctx, c.Ref)
	if err != nil {
		return c, ErrConciliacion
	}
	if r.Ref != c.Ref || r.ManifiestoBaseSHA256 != c.ManifiestoBaseSHA256 || r.EjecucionVerificacionRef != c.EjecucionVerificacionRef || !reflect.DeepEqual(r.Origen, c.Origen) || r.Manifiesto.InventarioSHA256 != c.Manifiesto.InventarioSHA256 {
		return c, ErrConciliacion
	}
	if err := validarConjunto(r, r.Manifiesto.Verificacion.Estado); err != nil {
		return c, ErrConciliacion
	}
	if estado == "verificada_declarada" && r.Manifiesto.Verificacion.Estado != "valida" {
		return c, ErrConciliacion
	}
	base := puertos.EventoCopia{OperacionRef: r.Manifiesto.OperacionRef, ConjuntoRef: r.Ref, ManifiestoSHA256: r.ManifiestoBaseSHA256, EjecucionRef: r.EjecucionVerificacionRef}
	if estado == "capturando" {
		e := base
		e.Transicion = "confirmar_captura"
		if err := s.d.Registro.AplicarCopia(ctx, e); err != nil {
			return c, ErrConciliacion
		}
	}
	if estado == "capturando" || estado == "capturada" {
		e := base
		e.Transicion = "iniciar_verificacion"
		if err := s.d.Registro.AplicarCopia(ctx, e); err != nil {
			return c, ErrConciliacion
		}
	}
	if r.Manifiesto.Verificacion.Estado == "pendiente_verificacion" {
		r, err = s.ensayarPublicado(ctx, r)
		if err != nil {
			return c, err
		}
	}
	if err := validarConjunto(r, "valida"); err != nil {
		return c, ErrConciliacion
	}
	if estado != "verificada_declarada" {
		for _, x := range []struct {
			modo      puertos.ModoEnsayo
			evidencia copias.Evidencia
		}{
			{puertos.Fisico, r.Manifiesto.Verificacion.Fisica},
			{puertos.Logico, r.Manifiesto.Verificacion.Logica},
		} {
			e := base
			e.Transicion = "declarar_ensayo"
			e.Ensayo = &puertos.EnsayoRegistrado{Modo: x.modo, Evidencia: x.evidencia, Resultado: "satisfactorio"}
			if err := s.d.Registro.AplicarCopia(ctx, e); err != nil {
				return c, ErrConciliacion
			}
		}
	}
	return r, nil
}

func (s *Servicio) sellarYEnsayar(ctx context.Context, captura puertos.Captura) (puertos.Conjunto, error) {
	c, err := s.d.Destino.Publicar(ctx, captura)
	if err != nil {
		return c, err
	}
	if err := validarConjunto(c, "pendiente_verificacion"); err != nil {
		return c, err
	}
	if c.Manifiesto.ConjuntoRef != captura.Manifiesto.ConjuntoRef || c.Manifiesto.InventarioSHA256 != captura.Manifiesto.InventarioSHA256 || !reflect.DeepEqual(c.Manifiesto.Componentes, captura.Manifiesto.Componentes) || !reflect.DeepEqual(c.Origen, captura.Origen) {
		return c, denegar("sello_otro_conjunto")
	}
	r, err := s.d.Destino.Recuperar(ctx, c.Ref)
	if err != nil {
		return c, err
	}
	if r.Ref != c.Ref || !reflect.DeepEqual(r.Manifiesto, c.Manifiesto) || !reflect.DeepEqual(r.Origen, c.Origen) || r.IndiceAutenticadoRef != c.IndiceAutenticadoRef || r.ManifiestoSHA256 != c.ManifiestoSHA256 || r.ManifiestoBaseSHA256 != c.ManifiestoBaseSHA256 || r.EjecucionVerificacionRef != c.EjecucionVerificacionRef {
		return c, denegar("indice_alterado")
	}
	return s.ensayarPublicado(ctx, r)
}

func (s *Servicio) ensayarPublicado(ctx context.Context, r puertos.Conjunto) (puertos.Conjunto, error) {
	f, err := s.d.Ensayador.Ensayar(ctx, r, puertos.Fisico)
	if err != nil {
		return r, denegar("ensayo_fisico_fallido")
	}
	g, err := s.d.Ensayador.Ensayar(ctx, r, puertos.Logico)
	if err != nil {
		return r, denegar("ensayo_logico_fallido")
	}
	if !ensayosValidos(r.Origen, f, g) {
		return r, denegar("contraste_fallido")
	}
	v := copias.Verificacion{Estado: "valida", VerificadorVersion: f.VerificadorVersion, Fecha: time.Now().UTC(), Fisica: f.Evidencia, Logica: g.Evidencia}
	c, err := s.d.Destino.CerrarVerificacion(ctx, r, v)
	if err != nil {
		return r, ErrConciliacion
	}
	final, err := s.d.Destino.Recuperar(ctx, c.Ref)
	if err != nil {
		return c, ErrConciliacion
	}
	if err := validarConjunto(final, "valida"); err != nil {
		return c, err
	}
	if final.IndiceAutenticadoRef != c.IndiceAutenticadoRef || final.ManifiestoSHA256 != c.ManifiestoSHA256 || final.ManifiestoBaseSHA256 != r.ManifiestoBaseSHA256 || final.EjecucionVerificacionRef != r.EjecucionVerificacionRef || !reflect.DeepEqual(final.Manifiesto, c.Manifiesto) || !reflect.DeepEqual(final.Origen, r.Origen) || !reflect.DeepEqual(final.Manifiesto.Verificacion, v) {
		return c, denegar("evidencia_alterada")
	}
	return final, nil
}

func validarConjunto(c puertos.Conjunto, estado string) error {
	if c.Ref == "" || c.IndiceAutenticadoRef == "" || c.EjecucionVerificacionRef == "" || !huellaValida(c.ManifiestoSHA256) || !huellaValida(c.ManifiestoBaseSHA256) || c.Ref != c.Manifiesto.ConjuntoRef || c.Manifiesto.Verificacion.Estado != estado || len(copias.ValidarManifiesto(c.Manifiesto)) != 0 {
		return denegar("conjunto_no_comprobable")
	}
	if estado == "pendiente_verificacion" && (c.ManifiestoSHA256 != c.ManifiestoBaseSHA256 || c.IndiceAutenticadoRef != c.EjecucionVerificacionRef) {
		return denegar("indice_base_incoherente")
	}
	if c.Manifiesto.InventarioSHA256 != copias.HuellaInventario(c.Manifiesto.Inventario) {
		return denegar("inventario_alterado")
	}
	if !huellasCompletas(c.Origen) {
		return denegar("origen_no_contrastable")
	}
	if estado == "valida" && (!evidenciaCompleta(c.Manifiesto.Verificacion.Fisica) || !evidenciaCompleta(c.Manifiesto.Verificacion.Logica) || !mismoContenido(c.Origen, c.Manifiesto.Verificacion.Fisica) || !mismoContenido(c.Origen, c.Manifiesto.Verificacion.Logica)) {
		return denegar("verificacion_no_coincide_origen")
	}
	return nil
}

func evidenciaCompleta(e copias.Evidencia) bool {
	return e.ArranqueRef != "" && huellasCompletas(e)
}

func huellaValida(s string) bool {
	if len(s) != 64 {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}

func huellasCompletas(e copias.Evidencia) bool {
	if e.Ref == "" {
		return false
	}
	for _, s := range []string{e.RecuentosSHA256, e.ContenidoSHA256, e.EsquemaSHA256, e.RolesSHA256, e.ACLSHA256, e.SecuenciasSHA256, e.ObjetosGrandesSHA256, e.FicherosSHA256} {
		if len(s) != 64 {
			return false
		}
		if _, err := hex.DecodeString(s); err != nil {
			return false
		}
	}
	return true
}

func mismoContenido(a, b copias.Evidencia) bool {
	return a.RecuentosSHA256 == b.RecuentosSHA256 && a.ContenidoSHA256 == b.ContenidoSHA256 && a.EsquemaSHA256 == b.EsquemaSHA256 && a.RolesSHA256 == b.RolesSHA256 && a.ACLSHA256 == b.ACLSHA256 && a.SecuenciasSHA256 == b.SecuenciasSHA256 && a.ObjetosGrandesSHA256 == b.ObjetosGrandesSHA256 && a.FicherosSHA256 == b.FicherosSHA256
}

func ensayosValidos(origen copias.Evidencia, f, g puertos.Ensayo) bool {
	return f.Modo == puertos.Fisico && g.Modo == puertos.Logico && f.VerificadorVersion != "" && f.VerificadorVersion == g.VerificadorVersion && f.ArranqueRef != "" && g.ArranqueRef != "" && f.ArranqueRef == f.Evidencia.ArranqueRef && g.ArranqueRef == g.Evidencia.ArranqueRef && evidenciaCompleta(f.Evidencia) && evidenciaCompleta(g.Evidencia) && mismoContenido(origen, f.Evidencia) && mismoContenido(origen, g.Evidencia)
}
