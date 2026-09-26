package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"

	"vec-diputacion-granada/internal/modules/seleccion/domain"
	"vec-diputacion-granada/internal/modules/seleccion/ports"
	"vec-diputacion-granada/internal/shared/baremacion"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// ServicioConsultaRRHH atiende a RRHH: listado minimizado de las solicitudes
// presentadas en una convocatoria y ficha completa (auditada en la base).
type ServicioConsultaRRHH struct {
	autorizador
	repositorio   ports.RepositorioSolicitudes
	convocatorias ports.RegistroConvocatorias
	protector     ports.ProtectorDatosSolicitud
}

// NuevoServicioConsultaRRHH exige todas sus dependencias.
func NuevoServicioConsultaRRHH(repositorio ports.RepositorioSolicitudes, convocatorias ports.RegistroConvocatorias, emisor ports.EmisorMaterialV3,
	protector ports.ProtectorDatosSolicitud, reloj ports.Reloj) (*ServicioConsultaRRHH, error) {
	if nula(repositorio) || nula(convocatorias) || nula(emisor) || nula(protector) || nula(reloj) {
		return nil, ports.ErrNoDisponible
	}
	return &ServicioConsultaRRHH{autorizador: autorizador{emisor: emisor, reloj: reloj}, repositorio: repositorio, convocatorias: convocatorias, protector: protector}, nil
}

// Convocatorias devuelve las convocatorias publicadas.
func (s *ServicioConsultaRRHH) Convocatorias(ctx context.Context) ([]domain.ConvocatoriaPublicada, error) {
	if s == nil || ctx == nil {
		return nil, ports.ErrNoDisponible
	}
	return s.convocatorias.ConvocatoriasVigentes(ctx)
}

func (s *ServicioConsultaRRHH) autorizarRRHH(ctx context.Context, o Orden, accion, audiencia, recursoRef string) (ports.MaterialConsumoV3, error) {
	if s == nil || ctx == nil {
		return nil, ports.ErrNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r, _, err := validarOrden(o, dominiovec.SuperficieAutenticacionInternaCorporativaV1, s.reloj.Ahora().UTC())
	if err != nil {
		return nil, err
	}
	recurso := dominiovec.RecursoAutorizable{Referencia: recursoRef, ModuloID: ports.ModuloSeleccion, Tipo: ports.TipoRecursoSolicitudes, Ambitos: clonarAmbitos(o.Ambitos)}
	return s.autorizar(ctx, o, r, accion, audiencia, ports.FinalidadConsultaSolicitudes, recurso)
}

// FilaVisible es una solicitud del listado: nombre visible y documento
// parcial, nunca el documento completo ni el contacto.
type FilaVisible struct {
	SolicitudRef, NumeroJustificante, NombreVisible, DocumentoParcial string
	Estado                                                            domain.EstadoSolicitud
	PresentadaEn                                                      string
	Puntuacion                                                        baremacion.Puntos
	Turno                                                             string
}

// PaginaRRHH del listado, con cursor opaco.
type PaginaRRHH struct {
	Filas           []FilaVisible
	CursorSiguiente string
}

// ConsultaListado de RRHH.
type ConsultaListado struct {
	ConvocatoriaRef string
	Cursor          string
	Limite          int
}

// Listar devuelve las solicitudes presentadas en la convocatoria.
func (s *ServicioConsultaRRHH) Listar(ctx context.Context, o Orden, q ConsultaListado) (PaginaRRHH, error) {
	if s == nil || ctx == nil {
		return PaginaRRHH{}, ports.ErrNoDisponible
	}
	var cursor int64
	if q.Cursor != "" {
		var err error
		if cursor, err = strconv.ParseInt(q.Cursor, 10, 64); err != nil || cursor < 1 || strconv.FormatInt(cursor, 10) != q.Cursor {
			return PaginaRRHH{}, ports.ErrDatosNoValidos
		}
	}
	if q.Limite < 1 || q.Limite > 100 {
		return PaginaRRHH{}, ports.ErrDatosNoValidos
	}
	if _, err := convocatoriaVigente(ctx, s.convocatorias, q.ConvocatoriaRef); err != nil {
		return PaginaRRHH{}, err
	}
	suma := sha256.Sum256([]byte(q.ConvocatoriaRef))
	material, err := s.autorizarRRHH(ctx, o, ports.AccionConsultarSolicitudes, ports.AudienciaConsultarSolicitudes, ports.PrefijoRecursoConvocatoria+hex.EncodeToString(suma[:]))
	if err != nil {
		return PaginaRRHH{}, err
	}
	filas, err := s.repositorio.ListarPresentadas(ctx, q.ConvocatoriaRef, cursor, q.Limite, material)
	if err != nil {
		return PaginaRRHH{}, err
	}
	pagina := PaginaRRHH{Filas: make([]FilaVisible, 0, len(filas))}
	for _, f := range filas {
		if f.ConvocatoriaRef != q.ConvocatoriaRef {
			return PaginaRRHH{}, ports.ErrNoDisponible
		}
		datos, err := descifrarDatos(ctx, s.protector, ports.AsociacionDatos{PersonaRef: f.PersonaRef, ConvocatoriaRef: f.ConvocatoriaRef, Version: f.Version}, f.Sobre)
		if err != nil {
			return PaginaRRHH{}, err
		}
		pagina.Filas = append(pagina.Filas, FilaVisible{SolicitudRef: f.SolicitudRef, NumeroJustificante: f.NumeroJustificante,
			NombreVisible: datos.NombreVisible(), DocumentoParcial: f.DocumentoParcial, Estado: domain.EstadoPresentada,
			PresentadaEn: f.PresentadaEn.UTC().Format("2006-01-02T15:04:05.000000Z"), Puntuacion: f.Puntuacion, Turno: f.Turno})
	}
	if len(filas) == q.Limite {
		pagina.CursorSiguiente = strconv.FormatInt(filas[len(filas)-1].PresentacionID, 10)
	}
	return pagina, nil
}

// RequisitoFicha es un requisito con su título, estado, procedencia y fecha
// de referencia (hito) de la convocatoria.
type RequisitoFicha struct {
	Clave, Titulo   string
	Estado          domain.EstadoRequisito
	Procedencia     string
	FechaReferencia string
}

// FichaRRHH es la solicitud presentada completa, descifrada para RRHH.
type FichaRRHH struct {
	Fila         ports.FilaSolicitudRRHH
	Convocatoria domain.ConvocatoriaPublicada
	Datos        domain.DatosPersonales
	Requisitos   []RequisitoFicha
	Meritos      []domain.PuntosMerito
	Historia     []ports.EventoHistoria
}

// Detalle devuelve la ficha completa de una solicitud presentada. La base
// audita el acceso con la decisión consumida.
func (s *ServicioConsultaRRHH) Detalle(ctx context.Context, o Orden, solicitudRef string) (FichaRRHH, error) {
	if !solicitudRefValida.MatchString(solicitudRef) {
		return FichaRRHH{}, ports.ErrDatosNoValidos
	}
	material, err := s.autorizarRRHH(ctx, o, ports.AccionConsultarDetalle, ports.AudienciaConsultarDetalle, ports.PrefijoRecursoSolicitud+solicitudRef)
	if err != nil {
		return FichaRRHH{}, err
	}
	ficha, err := s.repositorio.LeerFicha(ctx, solicitudRef, material)
	if err != nil {
		return FichaRRHH{}, err
	}
	if ficha.Fila.SolicitudRef != solicitudRef {
		return FichaRRHH{}, ports.ErrNoDisponible
	}
	datos, err := descifrarDatos(ctx, s.protector, ports.AsociacionDatos{PersonaRef: ficha.Fila.PersonaRef, ConvocatoriaRef: ficha.Fila.ConvocatoriaRef, Version: ficha.Fila.Version}, ficha.Fila.Sobre)
	if err != nil {
		return FichaRRHH{}, err
	}
	convocatoria := ficha.Convocatoria.Convocatoria
	requisitos := make([]RequisitoFicha, 0, len(ficha.Requisitos))
	for _, d := range ficha.Requisitos {
		r, ok := convocatoria.Requisito(d.Clave)
		if !ok {
			return FichaRRHH{}, ports.ErrNoDisponible
		}
		requisitos = append(requisitos, RequisitoFicha{Clave: d.Clave, Titulo: r.Titulo, Estado: d.Estado,
			Procedencia: domain.ProcedenciaDeclaradaPersona, FechaReferencia: convocatoria.FechaReferencia})
	}
	autobaremo, err := convocatoria.Baremo.Calcular(ficha.Meritos)
	if err != nil {
		return FichaRRHH{}, errors.Join(ports.ErrNoDisponible, err)
	}
	return FichaRRHH{Fila: ficha.Fila, Convocatoria: ficha.Convocatoria, Datos: datos, Requisitos: requisitos, Meritos: autobaremo.Lineas, Historia: ficha.Historia}, nil
}
