package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/usuarios/domain"
	"vec-diputacion-granada/internal/modules/usuarios/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type ServicioImagen struct {
	registro      ports.RegistroImagen
	transformador ports.TransformadorImagen
	custodia      ports.CustodiaImagen
	nombres       ports.NombreImagenAutorizado
	ahoraUTC      func() time.Time
}

func NuevoServicioImagen(r ports.RegistroImagen, t ports.TransformadorImagen, c ports.CustodiaImagen, n ports.NombreImagenAutorizado, ahora func() time.Time) (*ServicioImagen, error) {
	if r == nil || t == nil || c == nil || ahora == nil {
		return nil, ports.ErrImagenNoDisponible
	}
	return &ServicioImagen{r, t, c, n, ahora}, nil
}

func (s *ServicioImagen) actor(ctx context.Context, o ports.OrdenImagen) (vecdomain.ContextoActor, error) {
	if s == nil || s.registro == nil || s.ahoraUTC == nil || ctx == nil {
		return vecdomain.ContextoActor{}, ports.ErrImagenNoDisponible
	}
	a, err := o.ContextoActor()
	if err != nil {
		return vecdomain.ContextoActor{}, ports.ErrImagenNoAutenticado
	}
	ahora := s.ahoraUTC().UTC().Truncate(time.Microsecond)
	if ahora.IsZero() || !a.Instantanea.VigenteEn(ahora) {
		return vecdomain.ContextoActor{}, ports.ErrImagenNoAutenticado
	}
	return a, nil
}

func materialImagen(a vecdomain.ContextoActor, o ports.OrdenImagen, titular, accion, finalidad string, version uint64, catalogo, clave, huella string, e domain.EleccionImagen) ports.MaterialImagen {
	return ports.MaterialImagen{ActorPersonaRef: a.PersonaRef, TitularPersonaRef: titular, PerfilRef: a.PerfilActivoRef, Accion: accion, FinalidadRef: finalidad, Audiencia: o.Audiencia(), VersionEsperada: version, CatalogoVersionRef: catalogo, ClaveOperacion: clave, HuellaPeticion: huella, Eleccion: e}
}

func autorizacionImagen(ctx context.Context, o ports.OrdenImagen, m ports.MaterialImagen) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	p := o.Proveedor()
	if p == nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrImagenNoAutenticado
	}
	v3, err := p.ProveerMaterialImagen(ctx, m)
	if err != nil {
		if errors.Is(err, ports.ErrImagenProhibido) || errors.Is(err, ports.ErrImagenNoAutenticado) {
			return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, err
		}
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrImagenNoDisponible
	}
	if v3.ValidarEstructura() != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrImagenNoDisponible
	}
	return v3, nil
}

func (s *ServicioImagen) catalogo(ctx context.Context) (domain.CatalogoImagen, error) {
	c, err := s.registro.CatalogoVigente(ctx)
	if err != nil || c.Validar() != nil {
		return domain.CatalogoImagen{}, ports.ErrImagenNoDisponible
	}
	return c, nil
}

func (s *ServicioImagen) ConsultarPropia(ctx context.Context, o ports.OrdenImagen) (ports.VistaImagen, error) {
	a, err := s.actor(ctx, o)
	if err != nil {
		return ports.VistaImagen{}, err
	}
	return s.consultar(ctx, o, a, a.PersonaRef, ports.AccionImagenConsultarPropia, ports.FinalidadImagenPropia)
}

func referenciaPersonaImagenValida(ref string) bool {
	if !strings.HasPrefix(ref, "per_") || len(ref) < 20 || len(ref) > 96 {
		return false
	}
	for _, r := range ref[4:] {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '-' && r != '_' {
			return false
		}
	}
	return true
}

func (s *ServicioImagen) ConsultarAjena(ctx context.Context, o ports.OrdenImagen, titularRef string) (ports.VistaImagen, error) {
	a, err := s.actor(ctx, o)
	if err != nil {
		return ports.VistaImagen{}, err
	}
	if o.Audiencia() != ports.AudienciaImagenInterna {
		return ports.VistaImagen{}, ports.ErrImagenProhibido
	}
	if !referenciaPersonaImagenValida(titularRef) || titularRef == a.PersonaRef {
		return ports.VistaImagen{}, ports.ErrImagenPeticionInvalida
	}
	return s.consultar(ctx, o, a, titularRef, ports.AccionImagenConsultarAjena, ports.FinalidadImagenInterna)
}

func alcanceLecturaImagen(a vecdomain.ContextoActor, o ports.OrdenImagen, titularRef string) (string, string, error) {
	if titularRef == a.PersonaRef {
		return ports.AccionImagenConsultarPropia, ports.FinalidadImagenPropia, nil
	}
	if o.Audiencia() != ports.AudienciaImagenInterna {
		return "", "", ports.ErrImagenProhibido
	}
	if !referenciaPersonaImagenValida(titularRef) {
		return "", "", ports.ErrImagenPeticionInvalida
	}
	return ports.AccionImagenConsultarAjena, ports.FinalidadImagenInterna, nil
}

// ReferenciaActiva es la comprobación fresca que Documentos debe consumir
// inmediatamente antes de abrir los bytes. Revalida V3 y lee sólo el estado
// actual de Usuarios; un recibo previo no conserva acceso tras la retirada.
func (s *ServicioImagen) ReferenciaActiva(ctx context.Context, o ports.OrdenImagen, titularRef, documentoRef string) (bool, error) {
	a, err := s.actor(ctx, o)
	if err != nil {
		return false, err
	}
	_, _, activa, err := s.comprobarReferenciaActiva(ctx, o, a, titularRef, documentoRef)
	return activa, err
}

func (s *ServicioImagen) comprobarReferenciaActiva(ctx context.Context, o ports.OrdenImagen, a vecdomain.ContextoActor, titularRef, documentoRef string) (ports.EstadoImagen, ports.MaterialImagen, bool, error) {
	if !domain.ReferenciaDocumentoValida(documentoRef) {
		return ports.EstadoImagen{}, ports.MaterialImagen{}, false, ports.ErrImagenPeticionInvalida
	}
	accion, finalidad, err := alcanceLecturaImagen(a, o, titularRef)
	if err != nil {
		return ports.EstadoImagen{}, ports.MaterialImagen{}, false, err
	}
	c, err := s.catalogo(ctx)
	if err != nil {
		return ports.EstadoImagen{}, ports.MaterialImagen{}, false, err
	}
	m := materialImagen(a, o, titularRef, accion, finalidad, 0, c.VersionRef, "", "", domain.EleccionImagen{})
	v3, err := autorizacionImagen(ctx, o, m)
	if err != nil {
		return ports.EstadoImagen{}, ports.MaterialImagen{}, false, err
	}
	estado, existe, err := s.registro.Leer(ctx, o, m, v3)
	if err != nil {
		return ports.EstadoImagen{}, ports.MaterialImagen{}, false, err
	}
	if !existe {
		return ports.EstadoImagen{}, m, false, nil
	}
	if estado.PersonaRef != titularRef || estado.Version == 0 || estado.CatalogoVersionRef == "" || domain.CatalogoBaseImagen().ValidarEleccion(estado.Eleccion) != nil {
		return ports.EstadoImagen{}, ports.MaterialImagen{}, false, ports.ErrImagenNoDisponible
	}
	activa := estado.Eleccion.Modo == domain.ModoFoto && estado.Eleccion.DocumentoRef == documentoRef
	return estado, m, activa, nil
}

// AbrirFoto entrega sólo la foto propia actualmente elegida. HTTP debe usar
// Cache-Control privado con revalidación o no-store y nunca cachearla en Mi Bolsa.
func (s *ServicioImagen) AbrirFoto(ctx context.Context, o ports.OrdenImagen, documentoRef string) (ports.ContenidoImagen, error) {
	a, err := s.actor(ctx, o)
	if err != nil {
		return ports.ContenidoImagen{}, err
	}
	return s.abrirFoto(ctx, o, a, a.PersonaRef, documentoRef)
}

func (s *ServicioImagen) AbrirFotoAjena(ctx context.Context, o ports.OrdenImagen, titularRef, documentoRef string) (ports.ContenidoImagen, error) {
	a, err := s.actor(ctx, o)
	if err != nil {
		return ports.ContenidoImagen{}, err
	}
	if titularRef == a.PersonaRef {
		return ports.ContenidoImagen{}, ports.ErrImagenPeticionInvalida
	}
	return s.abrirFoto(ctx, o, a, titularRef, documentoRef)
}

func (s *ServicioImagen) abrirFoto(ctx context.Context, o ports.OrdenImagen, a vecdomain.ContextoActor, titularRef, documentoRef string) (ports.ContenidoImagen, error) {
	estado, m, activa, err := s.comprobarReferenciaActiva(ctx, o, a, titularRef, documentoRef)
	if err != nil {
		return ports.ContenidoImagen{}, err
	}
	if !activa {
		return ports.ContenidoImagen{}, ports.ErrImagenNoEncontrada
	}
	// El material documental se deriva del estado durable, no del cliente.
	m.VersionEsperada, m.CatalogoVersionRef, m.Eleccion = estado.Version, estado.CatalogoVersionRef, estado.Eleccion
	contenido, err := s.custodia.Abrir(ctx, o, m, documentoRef)
	if err != nil {
		clear(contenido.Bytes)
		if errors.Is(err, ports.ErrImagenNoEncontrada) || errors.Is(err, ports.ErrImagenProhibido) || errors.Is(err, ports.ErrImagenNoAutenticado) {
			return ports.ContenidoImagen{}, err
		}
		return ports.ContenidoImagen{}, ports.ErrImagenNoDisponible
	}
	if contenido.DocumentoRef != documentoRef || contenido.Tipo != "image/png" || len(contenido.Bytes) == 0 || len(contenido.Bytes) > ports.TamanoMaximoOriginalImagen {
		clear(contenido.Bytes)
		return ports.ContenidoImagen{}, ports.ErrImagenNoDisponible
	}
	return contenido, nil
}

func (s *ServicioImagen) consultar(ctx context.Context, o ports.OrdenImagen, a vecdomain.ContextoActor, titular, accion, finalidad string) (ports.VistaImagen, error) {
	c, err := s.catalogo(ctx)
	if err != nil {
		return ports.VistaImagen{}, err
	}
	m := materialImagen(a, o, titular, accion, finalidad, 0, c.VersionRef, "", "", domain.EleccionImagen{})
	v3, err := autorizacionImagen(ctx, o, m)
	if err != nil {
		return ports.VistaImagen{}, err
	}
	estado, existe, err := s.registro.Leer(ctx, o, m, v3)
	if err != nil {
		return ports.VistaImagen{}, err
	}
	if !existe {
		estado = ports.EstadoImagen{PersonaRef: titular, Version: 0, CatalogoVersionRef: c.VersionRef, Eleccion: domain.EleccionImagen{Modo: domain.ModoIniciales, Paleta: c.Paletas[0].Codigo}}
	}
	// Una opción válida guardada con una versión anterior sigue siendo legible
	// aunque el catálogo vigente deje de ofrecerla para nuevas elecciones.
	if estado.PersonaRef != titular || (existe && (estado.Version == 0 || estado.CatalogoVersionRef == "")) || domain.CatalogoBaseImagen().ValidarEleccion(estado.Eleccion) != nil {
		return ports.VistaImagen{}, ports.ErrImagenNoDisponible
	}
	vista := ports.VistaImagen{Catalogo: c.Clonar(), Estado: estado}
	if estado.Eleccion.Modo == domain.ModoFoto {
		disponible, err := s.custodia.Disponible(ctx, o, m, estado.Eleccion.DocumentoRef)
		if err != nil {
			return ports.VistaImagen{}, ports.ErrImagenNoDisponible
		}
		vista.FotoDisponible = disponible
	}
	if s.nombres != nil {
		nombre, err := s.nombres.NombreVisible(ctx, o, m, titular)
		if err == nil {
			vista.NombreAutorizado = nombre
		}
	}
	return vista, nil
}

func claveImagenValida(clave string) bool {
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

func huellaImagen(persona string, version uint64, catalogo string, e domain.EleccionImagen, originalSHA string) string {
	b, _ := json.Marshal(struct {
		Persona     string
		Version     uint64
		Catalogo    string
		Eleccion    domain.EleccionImagen
		OriginalSHA string
	}{persona, version, catalogo, e, originalSHA})
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func validarReciboImagen(r ports.ReciboImagen, persona string, version uint64, catalogo string, e domain.EleccionImagen) error {
	if r.ReciboRef == "" || r.PersonaRef != persona || r.Version != version+1 || r.CatalogoVersionRef != catalogo || r.Eleccion != e || r.FechaUTC.IsZero() {
		return ports.ErrImagenNoDisponible
	}
	return nil
}

func (s *ServicioImagen) Elegir(ctx context.Context, o ports.OrdenImagen, p ports.PeticionElegirImagen) (ports.ReciboImagen, error) {
	a, err := s.actor(ctx, o)
	if err != nil {
		return ports.ReciboImagen{}, err
	}
	if !claveImagenValida(p.ClaveOperacion) || p.VersionEsperada >= math.MaxInt64 || p.CatalogoVersionRef == "" || len(p.CatalogoVersionRef) > 96 || p.Modo == domain.ModoFoto {
		return ports.ReciboImagen{}, ports.ErrImagenPeticionInvalida
	}
	e := domain.EleccionImagen{Modo: p.Modo, Paleta: p.Paleta, Icono: p.Icono}
	h := huellaImagen(a.PersonaRef, p.VersionEsperada, p.CatalogoVersionRef, e, "")
	m := materialImagen(a, o, a.PersonaRef, ports.AccionImagenActualizar, ports.FinalidadImagenPropia, p.VersionEsperada, p.CatalogoVersionRef, p.ClaveOperacion, h, e)
	v3, err := autorizacionImagen(ctx, o, m)
	if err != nil {
		return ports.ReciboImagen{}, err
	}
	r, existe, err := s.registro.Recuperar(ctx, o, m, v3)
	if err != nil {
		return ports.ReciboImagen{}, err
	}
	if existe {
		if err := validarReciboImagen(r, a.PersonaRef, p.VersionEsperada, p.CatalogoVersionRef, e); err != nil {
			return ports.ReciboImagen{}, err
		}
		r.Replay = true
		return r, nil
	}
	c, err := s.catalogo(ctx)
	if err != nil {
		return ports.ReciboImagen{}, err
	}
	if c.VersionRef != p.CatalogoVersionRef {
		return ports.ReciboImagen{}, ports.ErrImagenConflicto
	}
	if c.ValidarEleccion(e) != nil {
		return ports.ReciboImagen{}, ports.ErrImagenPeticionInvalida
	}
	r, err = s.registro.Guardar(ctx, o, m, v3)
	if err != nil {
		return ports.ReciboImagen{}, err
	}
	if err := validarReciboImagen(r, a.PersonaRef, p.VersionEsperada, p.CatalogoVersionRef, e); err != nil {
		return ports.ReciboImagen{}, err
	}
	return r, nil
}

func (s *ServicioImagen) Retirar(ctx context.Context, o ports.OrdenImagen, version uint64, catalogo, clave, paleta string) (ports.ReciboImagen, error) {
	return s.Elegir(ctx, o, ports.PeticionElegirImagen{VersionEsperada: version, CatalogoVersionRef: catalogo, ClaveOperacion: clave, Modo: domain.ModoIniciales, Paleta: paleta})
}

func validarProcesadaImagen(p ports.ImagenProcesada, tipoDeclarado string) error {
	if p.TipoReal != "image/jpeg" && p.TipoReal != "image/png" && p.TipoReal != "image/webp" {
		return ports.ErrImagenPeticionInvalida
	}
	if tipoDeclarado != "" && tipoDeclarado != p.TipoReal {
		return ports.ErrImagenPeticionInvalida
	}
	if p.TipoSalida != "image/jpeg" && p.TipoSalida != "image/png" && p.TipoSalida != "image/webp" {
		return ports.ErrImagenPeticionInvalida
	}
	if p.AnchoOriginal < 1 || p.AltoOriginal < 1 || p.AnchoOriginal > ports.DimensionMaximaOriginalImagen || p.AltoOriginal > ports.DimensionMaximaOriginalImagen || int64(p.AnchoOriginal)*int64(p.AltoOriginal) > ports.PixelesMaximosOriginalImagen || p.Ancho != ports.LadoImagenProcesada || p.Alto != ports.LadoImagenProcesada || len(p.Bytes) == 0 || len(p.Bytes) > ports.TamanoMaximoOriginalImagen || !p.OrientacionAplicada || !p.MetadatosEliminados {
		return ports.ErrImagenPeticionInvalida
	}
	return nil
}

func (s *ServicioImagen) Subir(ctx context.Context, o ports.OrdenImagen, p ports.PeticionSubirImagen) (ports.ReciboImagen, error) {
	a, err := s.actor(ctx, o)
	if err != nil {
		return ports.ReciboImagen{}, err
	}
	if !claveImagenValida(p.ClaveOperacion) || p.VersionEsperada >= math.MaxInt64 || p.CatalogoVersionRef == "" || len(p.CatalogoVersionRef) > 96 || len(p.Original) == 0 || len(p.Original) > ports.TamanoMaximoOriginalImagen {
		return ports.ReciboImagen{}, ports.ErrImagenPeticionInvalida
	}
	originalSHA := sha256.Sum256(p.Original)
	h := huellaImagen(a.PersonaRef, p.VersionEsperada, p.CatalogoVersionRef, domain.EleccionImagen{Modo: domain.ModoFoto, Paleta: p.Paleta}, hex.EncodeToString(originalSHA[:]))
	pre := materialImagen(a, o, a.PersonaRef, ports.AccionImagenActualizar, ports.FinalidadImagenPropia, p.VersionEsperada, p.CatalogoVersionRef, p.ClaveOperacion, h, domain.EleccionImagen{Modo: domain.ModoFoto, Paleta: p.Paleta})
	if _, err := autorizacionImagen(ctx, o, pre); err != nil {
		return ports.ReciboImagen{}, err
	}
	reserva, existe, err := s.custodia.RecuperarReserva(ctx, o, pre, a.PersonaRef, p.ClaveOperacion)
	if err != nil {
		return ports.ReciboImagen{}, ports.ErrImagenNoDisponible
	}
	if existe {
		if reserva.ClaveOperacion != p.ClaveOperacion || reserva.HuellaPeticion != h {
			return ports.ReciboImagen{}, ports.ErrImagenConflicto
		}
	} else {
		c, err := s.catalogo(ctx)
		if err != nil {
			return ports.ReciboImagen{}, err
		}
		if c.VersionRef != p.CatalogoVersionRef {
			return ports.ReciboImagen{}, ports.ErrImagenConflicto
		}
		if c.ValidarEleccion(domain.EleccionImagen{Modo: domain.ModoIniciales, Paleta: p.Paleta}) != nil {
			return ports.ReciboImagen{}, ports.ErrImagenPeticionInvalida
		}
		original := bytes.Clone(p.Original)
		defer clear(original)
		procesada, err := s.transformador.Procesar(ctx, original, ports.LimitesTransformacionImagen{MaxBytes: ports.TamanoMaximoOriginalImagen, MaxDimension: ports.DimensionMaximaOriginalImagen, MaxPixeles: ports.PixelesMaximosOriginalImagen, LadoSalida: ports.LadoImagenProcesada})
		if err != nil {
			if errors.Is(err, ports.ErrImagenPeticionInvalida) {
				return ports.ReciboImagen{}, err
			}
			return ports.ReciboImagen{}, ports.ErrImagenNoDisponible
		}
		defer clear(procesada.Bytes)
		if err := validarProcesadaImagen(procesada, p.TipoDeclarado); err != nil {
			return ports.ReciboImagen{}, err
		}
		contenidoSHA := sha256.Sum256(procesada.Bytes)
		reserva = ports.ReservaImagen{ActorPersonaRef: a.PersonaRef, PerfilRef: a.PerfilActivoRef, Audiencia: o.Audiencia(), FinalidadRef: ports.FinalidadImagenPropia, PersonaRef: a.PersonaRef, ClaveOperacion: p.ClaveOperacion, HuellaPeticion: h, OriginalSHA: hex.EncodeToString(originalSHA[:]), VersionEsperada: p.VersionEsperada, CatalogoVersionRef: p.CatalogoVersionRef, Paleta: p.Paleta, HuellaContenido: hex.EncodeToString(contenidoSHA[:])}
		reserva, err = s.custodia.Reservar(ctx, o, pre, reserva, procesada)
		if err != nil {
			return ports.ReciboImagen{}, ports.ErrImagenNoDisponible
		}
	}
	return s.consumarReserva(ctx, o, a, reserva, h)
}

func reservaImagenValida(r ports.ReservaImagen, a vecdomain.ContextoActor, o ports.OrdenImagen, h string) bool {
	contenido, err := hex.DecodeString(r.HuellaContenido)
	original, errOriginal := hex.DecodeString(r.OriginalSHA)
	esperada := huellaImagen(a.PersonaRef, r.VersionEsperada, r.CatalogoVersionRef, domain.EleccionImagen{Modo: domain.ModoFoto, Paleta: r.Paleta}, r.OriginalSHA)
	return err == nil && len(contenido) == sha256.Size && errOriginal == nil && len(original) == sha256.Size && r.ActorPersonaRef == a.PersonaRef && r.PersonaRef == a.PersonaRef && r.PerfilRef == a.PerfilActivoRef && r.Audiencia == o.Audiencia() && r.FinalidadRef == ports.FinalidadImagenPropia && r.HuellaPeticion == h && h == esperada && claveImagenValida(r.ClaveOperacion) && r.VersionEsperada < math.MaxInt64 && r.CatalogoVersionRef != "" && domain.ReferenciaDocumentoValida(r.DocumentoRef)
}

func (s *ServicioImagen) consumarReserva(ctx context.Context, o ports.OrdenImagen, a vecdomain.ContextoActor, r ports.ReservaImagen, h string) (ports.ReciboImagen, error) {
	if !reservaImagenValida(r, a, o, h) {
		return ports.ReciboImagen{}, ports.ErrImagenNoDisponible
	}
	e := domain.EleccionImagen{Modo: domain.ModoFoto, Paleta: r.Paleta, DocumentoRef: r.DocumentoRef}
	m := materialImagen(a, o, a.PersonaRef, ports.AccionImagenActualizar, ports.FinalidadImagenPropia, r.VersionEsperada, r.CatalogoVersionRef, r.ClaveOperacion, r.HuellaPeticion, e)
	v3, err := autorizacionImagen(ctx, o, m)
	if err != nil {
		return ports.ReciboImagen{}, err
	}
	recibo, existe, err := s.registro.Recuperar(ctx, o, m, v3)
	if err != nil {
		return ports.ReciboImagen{}, err
	}
	if !existe {
		c, err := s.catalogo(ctx)
		if err != nil {
			return ports.ReciboImagen{}, err
		}
		if c.VersionRef != r.CatalogoVersionRef {
			return ports.ReciboImagen{}, ports.ErrImagenConflicto
		}
		if c.ValidarEleccion(e) != nil {
			return ports.ReciboImagen{}, ports.ErrImagenPeticionInvalida
		}
		recibo, err = s.registro.Guardar(ctx, o, m, v3)
		if err != nil {
			return ports.ReciboImagen{}, err
		}
	}
	if err := validarReciboImagen(recibo, a.PersonaRef, r.VersionEsperada, r.CatalogoVersionRef, e); err != nil {
		return ports.ReciboImagen{}, err
	}
	if err := s.custodia.ConfirmarReserva(ctx, o, m, r); err != nil {
		return ports.ReciboImagen{}, ports.ErrImagenNoDisponible
	}
	recibo.Replay = existe || recibo.Replay
	return recibo, nil
}

// ReconciliarFoto se invoca con la clave original por una tarea interna
// autenticada. Recupera la intención durable en Documentos y reautoriza V3;
// no acepta referencia de documento ni persona indicadas por el cliente.
func (s *ServicioImagen) ReconciliarFoto(ctx context.Context, o ports.OrdenImagen, clave string) (ports.ReciboImagen, error) {
	a, err := s.actor(ctx, o)
	if err != nil {
		return ports.ReciboImagen{}, err
	}
	if !claveImagenValida(clave) {
		return ports.ReciboImagen{}, ports.ErrImagenPeticionInvalida
	}
	pre := materialImagen(a, o, a.PersonaRef, ports.AccionImagenRecuperarReserva, ports.FinalidadImagenPropia, 0, "", clave, "", domain.EleccionImagen{})
	if _, err := autorizacionImagen(ctx, o, pre); err != nil {
		return ports.ReciboImagen{}, err
	}
	r, existe, err := s.custodia.RecuperarReserva(ctx, o, pre, a.PersonaRef, clave)
	if err != nil {
		return ports.ReciboImagen{}, ports.ErrImagenNoDisponible
	}
	if !existe {
		return ports.ReciboImagen{}, ports.ErrImagenConflicto
	}
	if r.ClaveOperacion != clave {
		return ports.ReciboImagen{}, ports.ErrImagenNoDisponible
	}
	return s.consumarReserva(ctx, o, a, r, r.HuellaPeticion)
}
