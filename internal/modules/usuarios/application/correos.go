package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/usuarios/canonico"
	"vec-diputacion-granada/internal/modules/usuarios/domain"
	"vec-diputacion-granada/internal/modules/usuarios/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// VigenciaCodigoCorreoPredeterminada es el tiempo que vale un código. SQL
// admite entre 10 minutos y 72 horas; la composición puede fijar otro.
const VigenciaCodigoCorreoPredeterminada = time.Hour

type PoliticaDesafioCorreo struct{ Vigencia time.Duration }

func (p PoliticaDesafioCorreo) vigenciaValida() bool {
	return p.Vigencia >= 15*time.Minute && p.Vigencia <= 72*time.Hour && p.Vigencia%time.Microsecond == 0
}

// DependenciasCorreos agrupa los puertos del caso de uso. Ninguno es opcional.
type DependenciasCorreos struct {
	Registro    ports.RegistroCorreos
	Protector   ports.ProtectorDireccionCorreo
	Sellador    ports.SelladorHuellaCorreos
	Desafios    ports.PreparadorDesafioCorreo
	Validador   ports.ValidadorCodigoCorreo
	Transporte  ports.TransportadorCorreosPropios
	AhoraUTC    func() time.Time
	Politica    PoliticaDesafioCorreo
	LimiteEnvio time.Duration
}

type ServicioCorreos struct{ d DependenciasCorreos }

// La ruta aporta la superficie intentada; el dominio la coteja con el vínculo
// V2 certificado antes de construir esta orden.
func NuevaOrdenCorreos(actor vecdomain.ContextoActor, vinculo vecdomain.VinculoAutenticacionActorV2, superficieRuta vecdomain.SuperficieAutenticacionActorV1, proveedor ports.ProveedorMaterialCorreos) (ports.OrdenCorreos, error) {
	if proveedor == nil {
		return ports.OrdenCorreos{}, ports.ErrCorreosNoAutenticado
	}
	identidad, err := domain.NuevaIdentidadCorreos(actor, vinculo, superficieRuta)
	if errors.Is(err, domain.ErrSuperficieCorreosProhibida) {
		return ports.OrdenCorreos{}, ports.ErrCorreosProhibido
	}
	if err != nil {
		return ports.OrdenCorreos{}, ports.ErrCorreosNoAutenticado
	}
	return ports.OrdenCorreos{Identidad: identidad, Proveedor: proveedor}, nil
}

func NuevoServicioCorreos(d DependenciasCorreos) (*ServicioCorreos, error) {
	if d.Politica.Vigencia == 0 {
		d.Politica.Vigencia = VigenciaCodigoCorreoPredeterminada
	}
	if d.LimiteEnvio == 0 {
		d.LimiteEnvio = 15 * time.Second
	}
	if d.Registro == nil || d.Protector == nil || d.Sellador == nil || d.Desafios == nil || d.Validador == nil || d.Transporte == nil || d.AhoraUTC == nil || !d.Politica.vigenciaValida() || d.LimiteEnvio < time.Second || d.LimiteEnvio > time.Minute {
		return nil, ports.ErrCorreosNoDisponible
	}
	return &ServicioCorreos{d: d}, nil
}

func (s *ServicioCorreos) actor(ctx context.Context, orden ports.OrdenCorreos) (vecdomain.ContextoActor, vecdomain.SuperficieAutenticacionActorV1, error) {
	if s == nil || s.d.Registro == nil || s.d.AhoraUTC == nil || ctx == nil || ctx.Err() != nil {
		return vecdomain.ContextoActor{}, "", ports.ErrCorreosNoDisponible
	}
	if orden.Proveedor == nil {
		return vecdomain.ContextoActor{}, "", ports.ErrCorreosNoAutenticado
	}
	actor, vinculo, superficie, err := orden.Identidad.Datos()
	if errors.Is(err, domain.ErrSuperficieCorreosProhibida) {
		return vecdomain.ContextoActor{}, "", ports.ErrCorreosProhibido
	}
	if err != nil {
		return vecdomain.ContextoActor{}, "", ports.ErrCorreosNoAutenticado
	}
	ahora := s.d.AhoraUTC().UTC().Truncate(time.Microsecond)
	datos, err := vinculo.Datos()
	if err != nil || ahora.IsZero() || !actor.Instantanea.VigenteEn(ahora) || ahora.Before(datos.SesionRevalidadaEn) || !ahora.Before(datos.SesionValidaHasta) {
		return vecdomain.ContextoActor{}, "", ports.ErrCorreosNoAutenticado
	}
	return actor, superficie, nil
}

func materialCorreos(actor vecdomain.ContextoActor, superficie vecdomain.SuperficieAutenticacionActorV1, accion string, p ports.PeticionCorreo, huellas ports.HuellasSemanticasCorreo) ports.MaterialCorreos {
	return ports.MaterialCorreos{Superficie: superficie, PersonaRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, Accion: accion, FinalidadRef: ports.FinalidadCorreosPropios, VersionEsperada: p.VersionEsperada, ClaveOperacion: p.ClaveOperacion, HuellasPeticion: huellas, CorreoRef: p.CorreoRef}
}

// autorizarCorreos obtiene una exportación V3 fresca ligada al material exacto.
func autorizarCorreos(ctx context.Context, orden ports.OrdenCorreos, m ports.MaterialCorreos) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacia := vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	proveedor := orden.Proveedor
	if proveedor == nil {
		return vacia, ports.ErrCorreosNoAutenticado
	}
	_, vinculo, superficie, err := orden.Identidad.Datos()
	if err != nil {
		return vacia, ports.ErrCorreosNoAutenticado
	}
	if superficie != m.Superficie {
		return vacia, ports.ErrCorreosProhibido
	}
	audiencia, err := canonico.AudienciaCorreos(m.Accion, superficie)
	if err != nil {
		return vacia, err
	}
	recurso, err := canonico.RecursoCorreos(m)
	if err != nil {
		return vacia, ports.ErrCorreosInvalidos
	}
	huellaContexto, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return vacia, ports.ErrCorreosInvalidos
	}
	v3, err := proveedor.ProveerMaterialCorreos(ctx, vinculo, m)
	if err != nil {
		if errors.Is(err, ports.ErrCorreosNoAutenticado) || errors.Is(err, ports.ErrCorreosProhibido) {
			return vacia, err
		}
		return vacia, ports.ErrCorreosNoDisponible
	}
	if v3.ValidarEstructura() != nil {
		return vacia, ports.ErrCorreosNoDisponible
	}
	resumen := v3.ResumenCapacidad()
	if resumen.Operacion() != m.Accion || resumen.EfectoRef() != m.PersonaRef || resumen.AudienciaConsumo() != audiencia || resumen.EfectoHuellaSHA256() != huellaContexto {
		return vacia, ports.ErrCorreosNoDisponible
	}
	return v3, nil
}

func (s *ServicioCorreos) Consultar(ctx context.Context, orden ports.OrdenCorreos) (ports.VistaCorreos, error) {
	actor, superficie, err := s.actor(ctx, orden)
	if err != nil {
		return ports.VistaCorreos{}, err
	}
	m := materialCorreos(actor, superficie, ports.AccionConsultarCorreos, ports.PeticionCorreo{}, ports.HuellasSemanticasCorreo{})
	v3, err := autorizarCorreos(ctx, orden, m)
	if err != nil {
		return ports.VistaCorreos{}, err
	}
	vista, err := s.d.Registro.ConsultarPropios(ctx, orden, m, v3)
	if err != nil {
		return ports.VistaCorreos{}, err
	}
	if vista.PersonaRef != actor.PersonaRef || vista.Version > math.MaxInt64 || vista.Correos == nil || domain.ValidarConjuntoCorreos(vista.Correos) != nil {
		return ports.VistaCorreos{}, ports.ErrCorreosNoDisponible
	}
	return vista, nil
}

func claveCorreosValida(clave string) bool {
	if len(clave) < 16 || len(clave) > 128 {
		return false
	}
	for _, r := range clave {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '-' && r != '_' && r != ':' && r != '.' {
			return false
		}
	}
	return true
}

// ReferenciaCorreoValida acepta sólo el formato que emite este módulo.
func ReferenciaCorreoValida(ref string) bool {
	hexa, ok := strings.CutPrefix(ref, "correo:")
	if !ok || len(hexa) != 32 {
		return false
	}
	_, err := hex.DecodeString(hexa)
	return err == nil && strings.ToLower(hexa) == hexa
}

func peticionCorreoValida(accion string, p ports.PeticionCorreo) bool {
	if !claveCorreosValida(p.ClaveOperacion) || p.VersionEsperada >= math.MaxInt64 {
		return false
	}
	switch accion {
	case ports.AccionAnadirCorreo:
		return p.CorreoRef == "" && domain.DireccionCorreoValida(p.Direccion) && p.Codigo == ""
	case ports.AccionReenviarCorreo, ports.AccionActivarCorreo, ports.AccionRetirarCorreo:
		return ReferenciaCorreoValida(p.CorreoRef) && p.Direccion == "" && p.Codigo == ""
	case ports.AccionVerificarCorreo:
		_, ok := domain.NormalizarCodigoCorreo(p.Codigo)
		return ReferenciaCorreoValida(p.CorreoRef) && p.Direccion == "" && ok
	default:
		return false
	}
}

func preimagenCorreo(persona, accion string, p ports.PeticionCorreo) ([]byte, error) {
	// Sólo vive en memoria durante SellarHuellaCorreo. Ni la dirección ni el
	// código pueden tener un SHA-256 público que permita diccionario offline.
	b, err := json.Marshal(struct {
		Esquema, Persona, Accion, CorreoRef, Direccion, Codigo string
		Version                                                uint64
	}{"usuarios.correos.peticion.v3", persona, accion, p.CorreoRef, strings.ToLower(p.Direccion), p.Codigo, p.VersionEsperada})
	if err != nil {
		return nil, ports.ErrCorreosInvalidos
	}
	return b, nil
}

func nuevaReferenciaCorreo() (string, error) {
	var aleatorio [16]byte
	if _, err := rand.Read(aleatorio[:]); err != nil {
		return "", ports.ErrCorreosNoDisponible
	}
	return "correo:" + hex.EncodeToString(aleatorio[:]), nil
}

type comprobadorCorreo struct {
	validador ports.ValidadorCodigoCorreo
	codigo    string
}

func (c comprobadorCorreo) Comprobar(ctx context.Context, meta ports.MetadatosDesafioCorreo) (bool, error) {
	if c.validador == nil || c.codigo == "" {
		return false, ports.ErrCorreosNoDisponible
	}
	return c.validador.ComprobarCodigoCorreo(ctx, meta, c.codigo)
}

func borrarTexto(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

func (s *ServicioCorreos) mutar(ctx context.Context, orden ports.OrdenCorreos, accion string, p ports.PeticionCorreo) (ports.ReciboCorreos, error) {
	actor, superficie, err := s.actor(ctx, orden)
	if err != nil {
		return ports.ReciboCorreos{}, err
	}
	p.Direccion = strings.TrimSpace(p.Direccion)
	if !peticionCorreoValida(accion, p) {
		return ports.ReciboCorreos{}, ports.ErrCorreosInvalidos
	}
	if accion == ports.AccionVerificarCorreo {
		p.Codigo, _ = domain.NormalizarCodigoCorreo(p.Codigo)
	}
	preimagen, err := preimagenCorreo(actor.PersonaRef, accion, p)
	if err != nil {
		return ports.ReciboCorreos{}, err
	}
	huellas, err := s.d.Sellador.SellarHuellaCorreo(ctx, preimagen)
	borrarTexto(preimagen)
	if err != nil || !canonico.HuellasSemanticasValidas(huellas) {
		return ports.ReciboCorreos{}, ports.ErrCorreosNoDisponible
	}
	m := materialCorreos(actor, superficie, accion, p, huellas)
	v3, err := autorizarCorreos(ctx, orden, m)
	if err != nil {
		return ports.ReciboCorreos{}, err
	}
	// La recuperación consume V3 aunque la clave no exista: la existencia de
	// una operación no se observa antes de autorizarla.
	recibo, existe, err := s.d.Registro.RecuperarOperacion(ctx, orden, m, v3)
	if err != nil {
		return ports.ReciboCorreos{}, err
	}
	if existe {
		if !reciboCorreoValido(recibo, actor.PersonaRef, accion) || recibo.Version != p.VersionEsperada+1 || (accion != ports.AccionAnadirCorreo && recibo.CorreoRef != p.CorreoRef) {
			return ports.ReciboCorreos{}, ports.ErrCorreosNoDisponible
		}
		recibo.Replay, recibo.Envio = true, ports.EnvioSinCorreo
		return recibo, nil
	}
	v3, err = autorizarCorreos(ctx, orden, m)
	if err != nil {
		return ports.ReciboCorreos{}, err
	}
	var reserva ports.ReservaDesafio
	var sobre ports.SobreDireccionCorreo
	if accion == ports.AccionAnadirCorreo {
		p.CorreoRef, err = nuevaReferenciaCorreo()
		if err != nil {
			return ports.ReciboCorreos{}, err
		}
		claro := []byte(p.Direccion)
		sobre, err = s.d.Protector.CifrarDireccionCorreo(ctx, actor.PersonaRef, p.CorreoRef, p.VersionEsperada+1, claro)
		borrarTexto(claro)
		if err != nil || sobre.CorreoRef != p.CorreoRef || sobre.Version != p.VersionEsperada+1 || sobre.ClaveRef == "" || sobre.ClaveIgualdadRef == "" || sobre.ClaveIgualdadRef == sobre.ClaveRef || len(sobre.Nonce) != 12 || len(sobre.Cifrado) == 0 || len(sobre.HuellaIgualdad) != 32 {
			return ports.ReciboCorreos{}, ports.ErrCorreosNoDisponible
		}
		p.Direccion = "" // El registro recibe sólo el sobre, nunca el claro.
	}
	if accion == ports.AccionAnadirCorreo || accion == ports.AccionReenviarCorreo {
		vence := s.d.AhoraUTC().UTC().Truncate(time.Microsecond).Add(s.d.Politica.Vigencia)
		reserva, err = s.d.Desafios.PrepararDesafioCorreo(ctx, actor.PersonaRef, p.CorreoRef, vence)
		if err != nil || reserva.DesafioRef == "" || len(reserva.HuellaCodigo) != 32 || reserva.ClaveRef == "" || !reserva.VenceUTC.Equal(vence) {
			return ports.ReciboCorreos{}, ports.ErrCorreosNoDisponible
		}
		if _, ok := domain.NormalizarCodigoCorreo(reserva.Codigo); !ok {
			return ports.ReciboCorreos{}, ports.ErrCorreosNoDisponible
		}
		reserva.VenceUTC = reserva.VenceUTC.UTC()
	}
	comprobador := comprobadorCorreo{}
	if accion == ports.AccionVerificarCorreo {
		comprobador = comprobadorCorreo{validador: s.d.Validador, codigo: p.Codigo}
		p.Codigo = "" // El registro sólo recibe el callback, nunca el código.
	}
	codigo := reserva.Codigo
	reserva.Codigo = ""
	resultado, err := s.d.Registro.Aplicar(ctx, orden, p, m, v3, sobre, reserva, comprobador)
	if err != nil {
		return ports.ReciboCorreos{}, err
	}
	recibo = resultado.Recibo
	if !reciboCorreoValido(recibo, actor.PersonaRef, accion) || recibo.Version != m.VersionEsperada+1 || (accion != ports.AccionAnadirCorreo && recibo.CorreoRef != m.CorreoRef) || (accion == ports.AccionAnadirCorreo && !recibo.Replay && recibo.CorreoRef != p.CorreoRef) {
		return ports.ReciboCorreos{}, ports.ErrCorreosNoDisponible
	}
	if recibo.Replay {
		recibo.Envio = ports.EnvioSinCorreo
		return recibo, nil
	}
	recibo.Envio = s.enviar(ctx, orden, actor.PersonaRef, accion, resultado.Envios, reserva, codigo)
	return recibo, nil
}

// enviar entrega después del COMMIT las salidas reservadas por la operación.
// Un fallo del relay nunca deshace el efecto: queda anotado y la persona
// puede pedir otro código. El código sólo se conoce en esta memoria.
func (s *ServicioCorreos) enviar(ctx context.Context, orden ports.OrdenCorreos, persona, accion string, envios []ports.EnvioPendiente, reserva ports.ReservaDesafio, codigo string) ports.EstadoEnvioCorreo {
	esperados := 0
	switch accion {
	case ports.AccionAnadirCorreo, ports.AccionReenviarCorreo:
		esperados = 1
	case ports.AccionActivarCorreo:
		if len(envios) == 1 {
			esperados = 1
		}
	}
	if len(envios) != esperados {
		return ports.EnvioNoAceptado
	}
	if esperados == 0 {
		return ports.EnvioSinCorreo
	}
	envio := envios[0]
	tipoEsperado := ports.TipoEnvioAviso
	if accion != ports.AccionActivarCorreo {
		tipoEsperado = ports.TipoEnvioCodigo
	}
	if envio.Tipo != tipoEsperado || (tipoEsperado == ports.TipoEnvioCodigo && (envio.DesafioRef != reserva.DesafioRef || codigo == "")) {
		return ports.EnvioNoAceptado
	}
	ctxEnvio, cancelar := context.WithTimeout(context.WithoutCancel(ctx), s.d.LimiteEnvio)
	defer cancelar()
	aceptado := false
	err := s.d.Protector.ConDireccionCorreoDescifrada(ctxEnvio, persona, envio.Sobre, func(claro []byte) error {
		mensaje := ports.MensajeCorreoPropio{EnvioRef: envio.EnvioRef, Tipo: envio.Tipo, Destino: string(claro)}
		if envio.Tipo == ports.TipoEnvioCodigo {
			mensaje.Codigo, mensaje.VenceUTC = codigo, reserva.VenceUTC
		}
		aceptado = s.d.Transporte.EnviarCorreoPropio(ctxEnvio, mensaje)
		return nil
	})
	if err != nil {
		aceptado = false
	}
	// La anotación usa su propio plazo: el relay puede haber agotado el suyo.
	ctxNota, cancelarNota := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancelarNota()
	if s.d.Registro.ConfirmarEnvio(ctxNota, orden, envio, aceptado) != nil {
		// El resultado externo queda sin anotar; no se inventa éxito.
		return ports.EnvioNoAceptado
	}
	if tipoEsperado == ports.TipoEnvioAviso {
		// El aviso al correo anterior no condiciona el cambio: se informa
		// como correo sin enviar sólo si el relay lo rechazó.
		if aceptado {
			return ports.EnvioSinCorreo
		}
		return ports.EnvioNoAceptado
	}
	if aceptado {
		return ports.EnvioAceptado
	}
	return ports.EnvioNoAceptado
}

func reciboCorreoValido(r ports.ReciboCorreos, persona, accion string) bool {
	return r.ReciboRef != "" && r.PersonaRef == persona && r.Accion == accion && ReferenciaCorreoValida(r.CorreoRef) && r.Version > 0 && !r.FechaUTC.IsZero()
}

func (s *ServicioCorreos) Anadir(ctx context.Context, o ports.OrdenCorreos, p ports.PeticionCorreo) (ports.ReciboCorreos, error) {
	return s.mutar(ctx, o, ports.AccionAnadirCorreo, p)
}
func (s *ServicioCorreos) Reenviar(ctx context.Context, o ports.OrdenCorreos, p ports.PeticionCorreo) (ports.ReciboCorreos, error) {
	return s.mutar(ctx, o, ports.AccionReenviarCorreo, p)
}
func (s *ServicioCorreos) Verificar(ctx context.Context, o ports.OrdenCorreos, p ports.PeticionCorreo) (ports.ReciboCorreos, error) {
	return s.mutar(ctx, o, ports.AccionVerificarCorreo, p)
}
func (s *ServicioCorreos) Activar(ctx context.Context, o ports.OrdenCorreos, p ports.PeticionCorreo) (ports.ReciboCorreos, error) {
	return s.mutar(ctx, o, ports.AccionActivarCorreo, p)
}
func (s *ServicioCorreos) Retirar(ctx context.Context, o ports.OrdenCorreos, p ports.PeticionCorreo) (ports.ReciboCorreos, error) {
	return s.mutar(ctx, o, ports.AccionRetirarCorreo, p)
}

// Operar despacha por el nombre de operación que fija la API.
func (s *ServicioCorreos) Operar(ctx context.Context, o ports.OrdenCorreos, operacion string, p ports.PeticionCorreo) (ports.ReciboCorreos, error) {
	switch operacion {
	case "anadir":
		return s.Anadir(ctx, o, p)
	case "reenviar":
		return s.Reenviar(ctx, o, p)
	case "verificar":
		return s.Verificar(ctx, o, p)
	case "activar":
		return s.Activar(ctx, o, p)
	case "retirar":
		return s.Retirar(ctx, o, p)
	}
	return ports.ReciboCorreos{}, ports.ErrCorreosInvalidos
}
