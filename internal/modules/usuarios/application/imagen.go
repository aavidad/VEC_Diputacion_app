package application

import (
	"bytes"
	"context"
	"errors"
	"math"
	"time"

	"vec-diputacion-granada/internal/modules/usuarios/canonico"
	"vec-diputacion-granada/internal/modules/usuarios/domain"
	"vec-diputacion-granada/internal/modules/usuarios/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// ServicioImagen gestiona «Mi imagen»: iniciales, icono o foto propia. Solo
// la persona titular consulta y cambia su imagen; la foto se custodia en
// Documentos y aquí solo se conoce su referencia opaca.
type ServicioImagen struct {
	registro      ports.RegistroImagen
	transformador ports.TransformadorFoto
	ahoraUTC      func() time.Time
}

func NuevoServicioImagen(registro ports.RegistroImagen, transformador ports.TransformadorFoto, ahoraUTC func() time.Time) (*ServicioImagen, error) {
	if registro == nil || transformador == nil || ahoraUTC == nil {
		return nil, ports.ErrImagenNoDisponible
	}
	return &ServicioImagen{registro: registro, transformador: transformador, ahoraUTC: ahoraUTC}, nil
}

// NuevaOrdenImagen coteja la superficie de la ruta con el vínculo V2 antes de
// construir la orden; la ruta nunca elige persona.
func NuevaOrdenImagen(actor vecdomain.ContextoActor, vinculo vecdomain.VinculoAutenticacionActorV2, superficieRuta vecdomain.SuperficieAutenticacionActorV1, proveedor ports.ProveedorMaterialImagen) (ports.OrdenImagen, error) {
	if proveedor == nil {
		return ports.OrdenImagen{}, ports.ErrImagenNoAutenticado
	}
	identidad, err := domain.NuevaIdentidadImagen(actor, vinculo, superficieRuta)
	if errors.Is(err, domain.ErrSuperficieCorreosProhibida) {
		return ports.OrdenImagen{}, ports.ErrImagenProhibido
	}
	if err != nil {
		return ports.OrdenImagen{}, ports.ErrImagenNoAutenticado
	}
	return ports.OrdenImagen{Identidad: identidad, Proveedor: proveedor}, nil
}

func (s *ServicioImagen) actor(ctx context.Context, orden ports.OrdenImagen) (vecdomain.ContextoActor, vecdomain.SuperficieAutenticacionActorV1, error) {
	if s == nil || s.registro == nil || s.transformador == nil || s.ahoraUTC == nil || ctx == nil || ctx.Err() != nil {
		return vecdomain.ContextoActor{}, "", ports.ErrImagenNoDisponible
	}
	if orden.Proveedor == nil {
		return vecdomain.ContextoActor{}, "", ports.ErrImagenNoAutenticado
	}
	actor, vinculo, superficie, err := orden.Identidad.Datos()
	if errors.Is(err, domain.ErrSuperficieCorreosProhibida) {
		return vecdomain.ContextoActor{}, "", ports.ErrImagenProhibido
	}
	if err != nil {
		return vecdomain.ContextoActor{}, "", ports.ErrImagenNoAutenticado
	}
	ahora := s.ahoraUTC().UTC().Truncate(time.Microsecond)
	datos, err := vinculo.Datos()
	if err != nil || ahora.IsZero() || !actor.Instantanea.VigenteEn(ahora) || ahora.Before(datos.SesionRevalidadaEn) || !ahora.Before(datos.SesionValidaHasta) {
		return vecdomain.ContextoActor{}, "", ports.ErrImagenNoAutenticado
	}
	return actor, superficie, nil
}

// autorizarImagen obtiene una exportación V3 fresca ligada al material exacto.
func autorizarImagen(ctx context.Context, orden ports.OrdenImagen, m ports.MaterialImagen) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacia := vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	_, vinculo, superficie, err := orden.Identidad.Datos()
	if err != nil || orden.Proveedor == nil {
		return vacia, ports.ErrImagenNoAutenticado
	}
	if superficie != m.Superficie {
		return vacia, ports.ErrImagenProhibido
	}
	audiencia, err := canonico.AudienciaImagen(m.Accion, superficie)
	if err != nil {
		return vacia, err
	}
	recurso, err := canonico.RecursoImagen(m)
	if err != nil {
		return vacia, ports.ErrImagenPeticionInvalida
	}
	huellaContexto, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return vacia, ports.ErrImagenPeticionInvalida
	}
	v3, err := orden.Proveedor.ProveerMaterialImagen(ctx, vinculo, m)
	if err != nil {
		if errors.Is(err, ports.ErrImagenNoAutenticado) || errors.Is(err, ports.ErrImagenProhibido) {
			return vacia, err
		}
		return vacia, ports.ErrImagenNoDisponible
	}
	if v3.ValidarEstructura() != nil {
		return vacia, ports.ErrImagenNoDisponible
	}
	resumen := v3.ResumenCapacidad()
	if resumen.Operacion() != m.Accion || resumen.EfectoRef() != m.PersonaRef || resumen.AudienciaConsumo() != audiencia || resumen.EfectoHuellaSHA256() != huellaContexto {
		return vacia, ports.ErrImagenNoDisponible
	}
	return v3, nil
}

func (s *ServicioImagen) catalogo(ctx context.Context, orden ports.OrdenImagen) (domain.CatalogoImagen, error) {
	c, err := s.registro.CatalogoVigente(ctx, orden)
	if err != nil {
		if errors.Is(err, ports.ErrImagenNoAutenticado) || errors.Is(err, ports.ErrImagenProhibido) {
			return domain.CatalogoImagen{}, err
		}
		return domain.CatalogoImagen{}, ports.ErrImagenNoDisponible
	}
	if c.Validar() != nil {
		return domain.CatalogoImagen{}, ports.ErrImagenNoDisponible
	}
	return c, nil
}

// FotoValida es la comprobación mínima de lo que Documentos devuelve: un
// JPEG completo dentro del límite de custodia.
func FotoValida(f *ports.FotoImagen) bool {
	return f != nil && f.Tipo == ports.TipoFotoImagen && len(f.Datos) >= 4 && len(f.Datos) <= ports.TamanoMaximoFotoCustodia &&
		bytes.HasPrefix(f.Datos, []byte{0xff, 0xd8, 0xff}) && bytes.HasSuffix(f.Datos, []byte{0xff, 0xd9})
}

func (s *ServicioImagen) Consultar(ctx context.Context, orden ports.OrdenImagen) (ports.VistaImagen, error) {
	actor, superficie, err := s.actor(ctx, orden)
	if err != nil {
		return ports.VistaImagen{}, err
	}
	catalogo, err := s.catalogo(ctx, orden)
	if err != nil {
		return ports.VistaImagen{}, err
	}
	m := ports.MaterialImagen{Superficie: superficie, PersonaRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef,
		Accion: ports.AccionConsultarImagen, FinalidadRef: ports.FinalidadImagenPropia, CatalogoVersionRef: catalogo.VersionRef}
	v3, err := autorizarImagen(ctx, orden, m)
	if err != nil {
		return ports.VistaImagen{}, err
	}
	estado, existe, foto, err := s.registro.Consultar(ctx, orden, m, v3)
	if err != nil {
		return ports.VistaImagen{}, err
	}
	if !existe {
		estado = ports.EstadoImagen{PersonaRef: actor.PersonaRef, CatalogoVersionRef: catalogo.VersionRef, Eleccion: catalogo.Predeterminada}
		foto = nil
	}
	if estado.PersonaRef != actor.PersonaRef || estado.Eleccion.ValidarCodigos() != nil || estado.Version > math.MaxInt64 ||
		(existe && (estado.Version == 0 || estado.CatalogoVersionRef == "")) || (!existe && estado.Version != 0) ||
		(foto != nil && (estado.Eleccion.Modo != domain.ModoImagenFoto || !FotoValida(foto))) {
		return ports.VistaImagen{}, ports.ErrImagenNoDisponible
	}
	// Sin foto legible (p. ej. retirada por Documentos) la persona ve su modo
	// real y la interfaz muestra las iniciales con su paleta.
	return ports.VistaImagen{Catalogo: catalogo.Clonar(), Estado: estado, Foto: foto}, nil
}

func peticionImagenValida(p ports.PeticionImagen) bool {
	if !claveValida(p.ClaveOperacion) || p.VersionEsperada >= math.MaxInt64 || p.CatalogoVersionRef == "" || len(p.CatalogoVersionRef) > 96 ||
		p.Eleccion.ValidarCodigos() != nil {
		return false
	}
	switch p.Operacion {
	case ports.OperacionElegirImagen:
		return len(p.Foto) == 0
	case ports.OperacionSubirFotoImagen:
		return p.Eleccion.Modo == domain.ModoImagenFoto && len(p.Foto) > 0
	}
	return false
}

// Guardar aplica «elegir» o «subir_foto». La foto se recodifica antes de
// pedir la autorización: la V3 queda ligada a la huella del JPEG resultante,
// nunca a los bytes originales, que no salen de esta memoria.
func (s *ServicioImagen) Guardar(ctx context.Context, orden ports.OrdenImagen, p ports.PeticionImagen) (ports.ReciboImagen, error) {
	actor, superficie, err := s.actor(ctx, orden)
	if err != nil {
		return ports.ReciboImagen{}, err
	}
	if len(p.Foto) > ports.TamanoMaximoFotoImagen {
		return ports.ReciboImagen{}, ports.ErrImagenFotoGrande
	}
	if !peticionImagenValida(p) {
		return ports.ReciboImagen{}, ports.ErrImagenPeticionInvalida
	}
	var foto ports.FotoProcesada
	if p.Operacion == ports.OperacionSubirFotoImagen {
		foto, err = s.transformador.Procesar(ctx, p.Foto)
		if err != nil {
			if errors.Is(err, ports.ErrImagenFotoGrande) || errors.Is(err, ports.ErrImagenFotoNoAdmitida) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return ports.ReciboImagen{}, err
			}
			return ports.ReciboImagen{}, ports.ErrImagenNoDisponible
		}
		if len(foto.Bytes) == 0 || len(foto.Bytes) > ports.TamanoMaximoFotoCustodia || len(foto.SHA256) != 64 {
			return ports.ReciboImagen{}, ports.ErrImagenNoDisponible
		}
		defer clear(foto.Bytes)
	}
	huella := canonico.HuellaPeticionImagen(actor.PersonaRef, p.VersionEsperada, p.CatalogoVersionRef, p.Eleccion, foto.SHA256)
	m := ports.MaterialImagen{Superficie: superficie, PersonaRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef,
		Accion: ports.AccionActualizarImagen, FinalidadRef: ports.FinalidadImagenPropia, CatalogoVersionRef: p.CatalogoVersionRef,
		VersionEsperada: p.VersionEsperada, ClaveOperacion: p.ClaveOperacion, HuellaPeticion: huella, Eleccion: p.Eleccion, FotoSHA256: foto.SHA256}
	v3, err := autorizarImagen(ctx, orden, m)
	if err != nil {
		return ports.ReciboImagen{}, err
	}
	// La recuperación consume V3 aunque la clave no exista: la existencia de
	// una operación no se observa antes de autorizarla.
	recibo, existe, err := s.registro.RecuperarOperacion(ctx, orden, m, v3)
	if err != nil {
		return ports.ReciboImagen{}, err
	}
	if existe {
		if !reciboImagenValido(recibo, actor.PersonaRef, p) {
			return ports.ReciboImagen{}, ports.ErrImagenNoDisponible
		}
		recibo.Replay = true
		return recibo, nil
	}
	catalogo, err := s.catalogo(ctx, orden)
	if err != nil {
		return ports.ReciboImagen{}, err
	}
	if catalogo.VersionRef != p.CatalogoVersionRef {
		return ports.ReciboImagen{}, ports.ErrImagenConflicto
	}
	if catalogo.ValidarEleccion(p.Eleccion) != nil {
		return ports.ReciboImagen{}, ports.ErrImagenPeticionInvalida
	}
	v3Guardado, err := autorizarImagen(ctx, orden, m)
	if err != nil {
		return ports.ReciboImagen{}, err
	}
	huellaRecuperacion, errRecuperacion := v3.HuellaConjuntoSHA256()
	huellaGuardado, errGuardado := v3Guardado.HuellaConjuntoSHA256()
	if errRecuperacion != nil || errGuardado != nil || huellaRecuperacion == huellaGuardado {
		return ports.ReciboImagen{}, ports.ErrImagenNoDisponible
	}
	recibo, err = s.registro.Guardar(ctx, orden, m, foto.Bytes, v3Guardado)
	if err != nil {
		return ports.ReciboImagen{}, err
	}
	if recibo.Replay || !reciboImagenValido(recibo, actor.PersonaRef, p) || recibo.FotoNueva != (foto.SHA256 != "") {
		return ports.ReciboImagen{}, ports.ErrImagenNoDisponible
	}
	return recibo, nil
}

func reciboImagenValido(r ports.ReciboImagen, persona string, p ports.PeticionImagen) bool {
	return r.ReciboRef != "" && r.PersonaRef == persona && r.Version == p.VersionEsperada+1 &&
		r.CatalogoVersionRef == p.CatalogoVersionRef && r.Eleccion == p.Eleccion && !r.FechaUTC.IsZero()
}
