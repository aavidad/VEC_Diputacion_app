package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"image/png"
	"io"
	"time"

	"vec-diputacion-granada/internal/vec/documentos/domain"
	"vec-diputacion-granada/internal/vec/documentos/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const maxBytesImagen = 2 * 1024 * 1024

type ServicioCustodiaImagen struct {
	Registro  ports.RegistroImagen
	Autoridad ports.AutoridadImagen
	Contextos ports.ContextosAlmacenImagen
	Almacen   vecports.AlmacenObjetos
	Admisor   ports.AdmisorImagen
	Usuarios  ports.ReferenciaActivaUsuarios
	AhoraUTC  func() time.Time
}

func (s *ServicioCustodiaImagen) listo() bool {
	return s != nil && s.Registro != nil && s.Autoridad != nil && s.Contextos != nil && s.Almacen != nil && s.Admisor != nil && s.Usuarios != nil && s.AhoraUTC != nil
}
func (s *ServicioCustodiaImagen) autorizar(ctx context.Context, op ports.OperacionImagen) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	if !s.listo() || ctx == nil || ctx.Err() != nil || op.Actor.Validar() != nil ||
		!op.Actor.Instantanea.VigenteEn(s.AhoraUTC().UTC()) || op.Actor.PersonaRef == "" ||
		op.Actor.PerfilActivoRef == "" || op.TitularPersonaRef == "" || op.Audiencia == "" || op.Finalidad == "" || op.Accion == "" {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrImagenNoDisponible
	}
	if op.Audiencia != ports.AudienciaImagenPersonal && op.Audiencia != ports.AudienciaImagenInterna {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrImagenProhibida
	}
	switch op.Accion {
	case ports.AccionImagenReservar, ports.AccionImagenRecuperar, ports.AccionImagenConfirmar, ports.AccionImagenDisponible, ports.AccionImagenAbrirPropia, ports.AccionImagenAbrirAjena:
	default:
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrImagenProhibida
	}
	if op.Actor.PersonaRef == op.TitularPersonaRef {
		if op.Finalidad != ports.FinalidadImagenPropia {
			return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrImagenProhibida
		}
	} else if op.Audiencia != ports.AudienciaImagenInterna || op.Finalidad != ports.FinalidadImagenInterna || (op.Accion != ports.AccionImagenAbrirAjena && op.Accion != ports.AccionImagenDisponible) {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrImagenProhibida
	}
	if op.Accion == ports.AccionImagenAbrirAjena && op.Actor.PersonaRef == op.TitularPersonaRef {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrImagenProhibida
	}
	if op.Accion == ports.AccionImagenAbrirPropia && op.Actor.PersonaRef != op.TitularPersonaRef {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrImagenProhibida
	}
	material, err := s.Autoridad.AutorizarImagen(ctx, op)
	if err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrImagenProhibida
	}
	if material.ValidarEstructura() != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrImagenNoDisponible
	}
	return material, nil
}
func (s *ServicioCustodiaImagen) permisoEfecto(ctx context.Context, op ports.OperacionImagen) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	// Cada transición durable obtiene material nuevo. Una autorización consumida
	// al recuperar la reserva no puede repetirse para reservar o admitir.
	return s.autorizar(ctx, op)
}
func bytesPNG256(contenido []byte) bool {
	if len(contenido) == 0 || len(contenido) > maxBytesImagen {
		return false
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(contenido))
	if err != nil || cfg.Width != 256 || cfg.Height != 256 {
		return false
	}
	_, err = png.Decode(bytes.NewReader(contenido))
	return err == nil
}
func huella(contenido []byte) string { h := sha256.Sum256(contenido); return hex.EncodeToString(h[:]) }
func objetoExacto(o vecports.ObjetoAlmacenado, id domain.IdentidadCustodiaImagen, zona vecports.ZonaAlmacen, tamano int64) bool {
	return o.Validar() == nil && o.Zona == zona && o.MIME == "image/png" && o.Tamano == tamano && o.HuellaSHA256 == id.ContenidoSHA256 && !o.Eliminado && !o.Inmovilizado
}
func reservaExacta(r ports.ReservaImagen, id domain.IdentidadCustodiaImagen) bool {
	return r.Identidad.Validar() == nil && r.Identidad.DocumentoRef != "" && r.Identidad.MismaPeticion(id) && r.Estado.Valido()
}

// Reservar reclama primero una fila durable. Reintentos recuperan esa fila y
// usan claves técnicas estables: una caída entre S3 y SQL no crea otra intención.
// Nunca publica la referencia en Usuarios; eso pertenece a su transacción.
func (s *ServicioCustodiaImagen) Reservar(ctx context.Context, op ports.OperacionImagen, id domain.IdentidadCustodiaImagen, contenido []byte) (ports.ReservaImagen, error) {
	if id.Validar() != nil || id.DocumentoRef != "" || op.Actor.PersonaRef != id.PersonaRef || op.TitularPersonaRef != id.PersonaRef || op.ClaveOperacion != id.ClaveOperacion || op.HuellaPeticion != id.HuellaPeticion || op.Accion != ports.AccionImagenReservar || !bytesPNG256(contenido) || huella(contenido) != id.ContenidoSHA256 {
		return ports.ReservaImagen{}, ports.ErrImagenInvalida
	}
	permiso, err := s.autorizar(ctx, op)
	if err != nil {
		return ports.ReservaImagen{}, err
	}
	r, existe, err := s.Registro.RecuperarImagen(ctx, op, permiso)
	if err != nil {
		return ports.ReservaImagen{}, err
	}
	if existe {
		if !reservaExacta(r, id) {
			return ports.ReservaImagen{}, ports.ErrImagenConflicto
		}
	} else {
		permiso, err = s.permisoEfecto(ctx, op)
		if err != nil {
			return ports.ReservaImagen{}, err
		}
		r, err = s.Registro.ReservarImagen(ctx, op, permiso, id)
		if err != nil {
			return ports.ReservaImagen{}, err
		}
		if !reservaExacta(r, id) || r.Estado != domain.EstadoImagenReservada {
			return ports.ReservaImagen{}, ports.ErrImagenNoDisponible
		}
	}
	return s.reconciliarObjetos(ctx, op, permiso, r, contenido)
}
func (s *ServicioCustodiaImagen) reconciliarObjetos(ctx context.Context, op ports.OperacionImagen, permiso vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, r ports.ReservaImagen, contenido []byte) (ports.ReservaImagen, error) {
	caps, err := s.Almacen.Capacidades(ctx)
	if err != nil || !caps.EscrituraEnFlujo || !caps.IntegridadSHA256 || !caps.ReferenciasOpacas || !caps.PromocionAtomica || !caps.PreservaObjetoOriginal || caps.TamanoMaximoObjeto < int64(len(contenido)) {
		return ports.ReservaImagen{}, ports.ErrImagenNoDisponible
	}
	if r.Estado == domain.EstadoImagenReservada {
		contexto, err := s.Contextos.EscribirCuarentena(ctx, op, r.Identidad, int64(len(contenido)))
		if err != nil {
			return ports.ReservaImagen{}, ports.ErrImagenNoDisponible
		}
		req := vecports.SolicitudEscribirObjeto{Contexto: contexto, ClaveIdempotencia: r.Identidad.DocumentoRef + ":cuarentena", Zona: vecports.ZonaAlmacenCuarentena, MIME: "image/png", Tamano: int64(len(contenido)), HuellaSHA256: r.Identidad.ContenidoSHA256, Contenido: bytes.NewReader(contenido)}
		if req.Validar() != nil {
			return ports.ReservaImagen{}, ports.ErrImagenNoDisponible
		}
		escrito, err := s.Almacen.Escribir(ctx, req)
		if err != nil {
			return ports.ReservaImagen{}, err
		}
		if escrito.ValidarEscritura(req, caps) != nil || !objetoExacto(escrito.Objeto, r.Identidad, vecports.ZonaAlmacenCuarentena, int64(len(contenido))) {
			return ports.ReservaImagen{}, ports.ErrImagenNoDisponible
		}
		permiso, err = s.permisoEfecto(ctx, op)
		if err != nil {
			return ports.ReservaImagen{}, err
		}
		r, err = s.Registro.RegistrarObjetoImagen(ctx, op, permiso, r, escrito.Objeto)
		if err != nil {
			return ports.ReservaImagen{}, err
		}
	}
	if r.Estado == domain.EstadoImagenCuarentena {
		if !objetoExacto(r.ObjetoCuarentena, r.Identidad, vecports.ZonaAlmacenCuarentena, int64(len(contenido))) {
			return ports.ReservaImagen{}, ports.ErrImagenNoDisponible
		}
		evidencia, err := s.Admisor.EvidenciaLimpia(ctx, r.ObjetoCuarentena)
		if err != nil || evidencia == "" {
			return ports.ReservaImagen{}, ports.ErrImagenNoDisponible
		}
		contexto, err := s.Contextos.PromoverAdmitida(ctx, op, r)
		if err != nil {
			return ports.ReservaImagen{}, ports.ErrImagenNoDisponible
		}
		req := vecports.SolicitudPromoverObjeto{Contexto: contexto, ClaveIdempotencia: r.Identidad.DocumentoRef + ":admitida", Origen: r.ObjetoCuarentena.Objeto, EvidenciaAnalisisRef: evidencia}
		if req.Validar() != nil {
			return ports.ReservaImagen{}, ports.ErrImagenNoDisponible
		}
		admitido, err := s.Almacen.Promover(ctx, req)
		if err != nil {
			return ports.ReservaImagen{}, err
		}
		if admitido.ValidarPromocion(req, r.ObjetoCuarentena, caps) != nil || !objetoExacto(admitido.Objeto, r.Identidad, vecports.ZonaAlmacenAdmitida, int64(len(contenido))) {
			return ports.ReservaImagen{}, ports.ErrImagenNoDisponible
		}
		permiso, err = s.permisoEfecto(ctx, op)
		if err != nil {
			return ports.ReservaImagen{}, err
		}
		r, err = s.Registro.AdmitirImagen(ctx, op, permiso, r, admitido.Objeto)
		if err != nil {
			return ports.ReservaImagen{}, err
		}
	}
	if r.Estado != domain.EstadoImagenAdmitida && r.Estado != domain.EstadoImagenConfirmada {
		return ports.ReservaImagen{}, ports.ErrImagenNoDisponible
	}
	if !objetoExacto(r.ObjetoAdmitido, r.Identidad, vecports.ZonaAlmacenAdmitida, int64(len(contenido))) {
		return ports.ReservaImagen{}, ports.ErrImagenNoDisponible
	}
	return r, nil
}

func (s *ServicioCustodiaImagen) Recuperar(ctx context.Context, op ports.OperacionImagen) (ports.ReservaImagen, bool, error) {
	if op.Accion != ports.AccionImagenRecuperar || op.Actor.PersonaRef != op.TitularPersonaRef || op.ClaveOperacion == "" {
		return ports.ReservaImagen{}, false, ports.ErrImagenInvalida
	}
	permiso, err := s.autorizar(ctx, op)
	if err != nil {
		return ports.ReservaImagen{}, false, err
	}
	r, ok, err := s.Registro.RecuperarImagen(ctx, op, permiso)
	if err != nil {
		return ports.ReservaImagen{}, false, err
	}
	if ok && (r.Identidad.Validar() != nil || r.Identidad.PersonaRef != op.TitularPersonaRef || r.Identidad.ClaveOperacion != op.ClaveOperacion || r.Identidad.DocumentoRef == "" || !r.Estado.Valido()) {
		return ports.ReservaImagen{}, false, ports.ErrImagenNoDisponible
	}
	// La recuperación sin bytes solo puede continuar cuando el objeto ya está
	// admitido. Antes de eso se reintenta la carga con el original del usuario.
	if ok && r.Estado != domain.EstadoImagenAdmitida && r.Estado != domain.EstadoImagenConfirmada {
		return ports.ReservaImagen{}, false, ports.ErrImagenNoDisponible
	}
	return r, ok, nil
}
func (s *ServicioCustodiaImagen) referenciaActiva(ctx context.Context, op ports.OperacionImagen) error {
	if s == nil || s.Usuarios == nil {
		return ports.ErrImagenNoDisponible
	}
	activa, err := s.Usuarios.ReferenciaActiva(ctx, op)
	if err != nil || !activa {
		return ports.ErrImagenProhibida
	}
	return nil
}
func (s *ServicioCustodiaImagen) Confirmar(ctx context.Context, op ports.OperacionImagen, r ports.ReservaImagen) error {
	if op.Accion != ports.AccionImagenConfirmar || op.Actor.PersonaRef != op.TitularPersonaRef || r.Estado != domain.EstadoImagenAdmitida && r.Estado != domain.EstadoImagenConfirmada || op.DocumentoRef != r.Identidad.DocumentoRef || !objetoExacto(r.ObjetoAdmitido, r.Identidad, vecports.ZonaAlmacenAdmitida, r.ObjetoAdmitido.Tamano) {
		return ports.ErrImagenInvalida
	}
	permiso, err := s.autorizar(ctx, op)
	if err != nil {
		return err
	}
	if err := s.referenciaActiva(ctx, op); err != nil {
		return err
	}
	confirmada, err := s.Registro.ConfirmarImagen(ctx, op, permiso, r)
	if err != nil {
		return err
	}
	if confirmada.Estado != domain.EstadoImagenConfirmada || confirmada.Identidad != r.Identidad || confirmada.ObjetoAdmitido.Objeto != r.ObjetoAdmitido.Objeto {
		return ports.ErrImagenNoDisponible
	}
	return nil
}
func (s *ServicioCustodiaImagen) Disponible(ctx context.Context, op ports.OperacionImagen) (bool, error) {
	if op.Accion != ports.AccionImagenDisponible || op.DocumentoRef == "" {
		return false, ports.ErrImagenInvalida
	}
	permiso, err := s.autorizar(ctx, op)
	if err != nil {
		return false, err
	}
	r, ok, err := s.Registro.LeerImagen(ctx, op, permiso)
	if err != nil {
		return false, err
	}
	if !ok || r.Estado != domain.EstadoImagenConfirmada || r.Identidad.PersonaRef != op.TitularPersonaRef || r.Identidad.DocumentoRef != op.DocumentoRef || !objetoExacto(r.ObjetoAdmitido, r.Identidad, vecports.ZonaAlmacenAdmitida, r.ObjetoAdmitido.Tamano) {
		return false, nil
	}
	return true, nil
}
func (s *ServicioCustodiaImagen) Abrir(ctx context.Context, op ports.OperacionImagen) ([]byte, error) {
	if op.Accion != ports.AccionImagenAbrirPropia && op.Accion != ports.AccionImagenAbrirAjena || op.DocumentoRef == "" {
		return nil, ports.ErrImagenInvalida
	}
	permiso, err := s.autorizar(ctx, op)
	if err != nil {
		return nil, err
	}
	r, ok, err := s.Registro.LeerImagen(ctx, op, permiso)
	if err != nil {
		return nil, err
	}
	if !ok || r.Estado != domain.EstadoImagenConfirmada || r.Identidad.PersonaRef != op.TitularPersonaRef || r.Identidad.DocumentoRef != op.DocumentoRef || !objetoExacto(r.ObjetoAdmitido, r.Identidad, vecports.ZonaAlmacenAdmitida, r.ObjetoAdmitido.Tamano) {
		return nil, ports.ErrImagenNoDisponible
	}
	if err := s.referenciaActiva(ctx, op); err != nil {
		return nil, err
	}
	contexto, err := s.Contextos.AbrirAdmitida(ctx, op, r)
	if err != nil {
		return nil, ports.ErrImagenNoDisponible
	}
	solicitud := vecports.SolicitudAbrirObjeto{Contexto: contexto, Objeto: r.ObjetoAdmitido.Objeto, Zona: vecports.ZonaAlmacenAdmitida, Limite: maxBytesImagen}
	if solicitud.Validar() != nil {
		return nil, ports.ErrImagenNoDisponible
	}
	lectura, err := s.Almacen.Abrir(ctx, solicitud)
	if err != nil {
		return nil, err
	}
	if lectura.Contenido == nil {
		return nil, ports.ErrImagenNoDisponible
	}
	defer lectura.Contenido.Close()
	if lectura.ValidarContra(solicitud) != nil || lectura.Objeto.Objeto != r.ObjetoAdmitido.Objeto || lectura.Objeto.HuellaSHA256 != r.Identidad.ContenidoSHA256 || lectura.Objeto.Tamano != r.ObjetoAdmitido.Tamano {
		return nil, ports.ErrImagenNoDisponible
	}
	contenido, err := io.ReadAll(io.LimitReader(lectura.Contenido, maxBytesImagen+1))
	if err != nil {
		return nil, ports.ErrImagenNoDisponible
	}
	if !bytesPNG256(contenido) || huella(contenido) != r.Identidad.ContenidoSHA256 || int64(len(contenido)) != r.ObjetoAdmitido.Tamano {
		return nil, ports.ErrImagenNoDisponible
	}
	// Revalidación final: una retirada mientras S3 respondía no entrega bytes.
	if err := s.referenciaActiva(ctx, op); err != nil {
		return nil, err
	}
	return contenido, nil
}
