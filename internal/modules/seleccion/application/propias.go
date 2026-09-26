package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"

	"vec-diputacion-granada/internal/modules/seleccion/domain"
	"vec-diputacion-granada/internal/modules/seleccion/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// ServiciosExternos declarados en la fase 1: firma, registro en sede, tasas
// y notificación. Sin proveedor, su adaptador responde «no disponible».
type ServiciosExternos struct {
	Firma        ports.FirmaSolicitud
	Registro     ports.RegistroSede
	Tasas        ports.PasarelaTasas
	Notificacion ports.NotificadorSolicitudes
}

func (s ServiciosExternos) completos() bool {
	return !nula(s.Firma) && !nula(s.Registro) && !nula(s.Tasas) && !nula(s.Notificacion)
}

// ServicioSolicitudesPropias atiende a la persona: convocatorias con plazo,
// sus solicitudes, su borrador y la presentación.
type ServicioSolicitudesPropias struct {
	autorizador
	repositorio   ports.RepositorioSolicitudes
	convocatorias ports.RegistroConvocatorias
	protector     ports.ProtectorDatosSolicitud
	referencias   ports.GeneradorReferencias
	externos      ServiciosExternos
}

// NuevoServicioSolicitudesPropias exige todas sus dependencias.
func NuevoServicioSolicitudesPropias(repositorio ports.RepositorioSolicitudes, convocatorias ports.RegistroConvocatorias, emisor ports.EmisorMaterialV3,
	protector ports.ProtectorDatosSolicitud, referencias ports.GeneradorReferencias, externos ServiciosExternos, reloj ports.Reloj) (*ServicioSolicitudesPropias, error) {
	if nula(repositorio) || nula(convocatorias) || nula(emisor) || nula(protector) || nula(referencias) || !externos.completos() || nula(reloj) {
		return nil, ports.ErrNoDisponible
	}
	return &ServicioSolicitudesPropias{autorizador: autorizador{emisor: emisor, reloj: reloj}, repositorio: repositorio,
		convocatorias: convocatorias, protector: protector, referencias: referencias, externos: externos}, nil
}

// Convocatorias devuelve las convocatorias publicadas (información pública
// de las bases, sin datos de ninguna solicitud).
func (s *ServicioSolicitudesPropias) Convocatorias(ctx context.Context) ([]domain.ConvocatoriaPublicada, error) {
	if s == nil || ctx == nil {
		return nil, ports.ErrNoDisponible
	}
	return s.convocatorias.ConvocatoriasVigentes(ctx)
}

// Convocatoria devuelve la versión vigente de una convocatoria.
func (s *ServicioSolicitudesPropias) Convocatoria(ctx context.Context, ref string) (domain.ConvocatoriaPublicada, error) {
	return convocatoriaVigente(ctx, s.convocatorias, ref)
}

func convocatoriaVigente(ctx context.Context, registro ports.RegistroConvocatorias, ref string) (domain.ConvocatoriaPublicada, error) {
	if ctx == nil || nula(registro) {
		return domain.ConvocatoriaPublicada{}, ports.ErrNoDisponible
	}
	vigentes, err := registro.ConvocatoriasVigentes(ctx)
	if err != nil {
		return domain.ConvocatoriaPublicada{}, err
	}
	for _, c := range vigentes {
		if c.Ref == ref {
			return c, nil
		}
	}
	return domain.ConvocatoriaPublicada{}, ports.ErrConvocatoriaNoDisponible
}

func (s *ServicioSolicitudesPropias) autorizarPropia(ctx context.Context, o Orden, accion, audiencia string) (ports.MaterialConsumoV3, string, error) {
	if s == nil || ctx == nil {
		return nil, "", ports.ErrNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	r, persona, err := validarOrden(o, dominiovec.SuperficieAutenticacionExternaPersonalV1, s.reloj.Ahora().UTC())
	if err != nil {
		return nil, "", err
	}
	recurso := dominiovec.RecursoAutorizable{Referencia: ports.PrefijoRecursoPropio + persona, ModuloID: ports.ModuloSeleccion,
		Tipo: ports.TipoRecursoSolicitudesPropias, Ambitos: clonarAmbitos(o.Ambitos), Atributos: map[string]string{"propiedad": "persona"}}
	material, err := s.autorizar(ctx, o, r, accion, audiencia, ports.FinalidadSolicitudesPropias, recurso)
	if err != nil {
		return nil, "", err
	}
	return material, persona, nil
}

// Listar devuelve las solicitudes de la persona.
func (s *ServicioSolicitudesPropias) Listar(ctx context.Context, o Orden) ([]ports.ResumenSolicitudPropia, error) {
	material, persona, err := s.autorizarPropia(ctx, o, ports.AccionConsultarPropias, ports.AudienciaConsultarPropias)
	if err != nil {
		return nil, err
	}
	return s.repositorio.ListarPropias(ctx, persona, material)
}

// BorradorLeido es la última versión de la solicitud propia, descifrada.
type BorradorLeido struct {
	Version  ports.VersionSolicitud
	Borrador domain.Borrador
}

// LeerBorrador devuelve la última versión propia en la convocatoria.
func (s *ServicioSolicitudesPropias) LeerBorrador(ctx context.Context, o Orden, convocatoriaRef string) (BorradorLeido, error) {
	material, persona, err := s.autorizarPropia(ctx, o, ports.AccionConsultarPropias, ports.AudienciaConsultarPropias)
	if err != nil {
		return BorradorLeido{}, err
	}
	version, err := s.repositorio.LeerBorrador(ctx, persona, convocatoriaRef, material)
	if err != nil {
		return BorradorLeido{}, err
	}
	if version.PersonaRef != persona || version.ConvocatoriaRef != convocatoriaRef {
		return BorradorLeido{}, ports.ErrNoDisponible
	}
	datos, err := descifrarDatos(ctx, s.protector, ports.AsociacionDatos{PersonaRef: persona, ConvocatoriaRef: convocatoriaRef, Version: version.Version}, version.Sobre)
	if err != nil {
		return BorradorLeido{}, err
	}
	return BorradorLeido{Version: version, Borrador: domain.Borrador{Turno: version.Turno, Datos: datos, Requisitos: version.Requisitos, Meritos: version.Meritos}}, nil
}

func descifrarDatos(ctx context.Context, protector ports.ProtectorDatosSolicitud, asociacion ports.AsociacionDatos, sobre ports.SobreDatos) (domain.DatosPersonales, error) {
	var datos domain.DatosPersonales
	err := protector.ConDatosSolicitudDescifrados(ctx, asociacion, sobre, func(claro []byte) error {
		var e error
		datos, e = domain.DatosPersonalesDesdeCanonico(claro)
		return e
	})
	if err != nil {
		return domain.DatosPersonales{}, errors.Join(ports.ErrNoDisponible, err)
	}
	return datos, nil
}

// EntradaBorrador es lo que la persona envía; nunca su identidad.
type EntradaBorrador struct {
	ConvocatoriaRef string
	VersionEsperada int
	Clave           string
	Borrador        domain.Borrador
}

// GuardarBorrador valida el borrador contra la convocatoria vigente, calcula
// el autobaremo, cifra los datos personales y guarda una versión nueva.
func (s *ServicioSolicitudesPropias) GuardarBorrador(ctx context.Context, o Orden, e EntradaBorrador) (ports.ResultadoGuardado, error) {
	if s == nil || ctx == nil {
		return ports.ResultadoGuardado{}, ports.ErrNoDisponible
	}
	if !claveValida.MatchString(e.Clave) || e.VersionEsperada < 0 || e.VersionEsperada > 1_000_000 {
		return ports.ResultadoGuardado{}, ports.ErrDatosNoValidos
	}
	convocatoria, err := s.Convocatoria(ctx, e.ConvocatoriaRef)
	if err != nil {
		return ports.ResultadoGuardado{}, err
	}
	preparado, err := e.Borrador.Preparar(convocatoria.Convocatoria, s.reloj.Ahora().UTC())
	if err != nil {
		return ports.ResultadoGuardado{}, errors.Join(ports.ErrDatosNoValidos, err)
	}
	borrador := preparado.Borrador
	material, persona, err := s.autorizarPropia(ctx, o, ports.AccionGuardarBorrador, ports.AudienciaGuardarBorrador)
	if err != nil {
		return ports.ResultadoGuardado{}, err
	}
	claro, err := borrador.Datos.Canonico()
	if err != nil {
		return ports.ResultadoGuardado{}, ports.ErrDatosNoValidos
	}
	huellaMaterial, err := s.huellaMaterialBorrador(ctx, persona, e, borrador, claro)
	if err != nil {
		return ports.ResultadoGuardado{}, err
	}
	var documento string
	if borrador.Datos.DocumentoIdentidad != "" {
		if documento, err = s.protector.HuellaConClave(ctx, "seleccion.documento_identidad.v1", []byte(borrador.Datos.DocumentoIdentidad)); err != nil {
			return ports.ResultadoGuardado{}, errors.Join(ports.ErrNoDisponible, err)
		}
	}
	sobre, err := s.protector.CifrarDatosSolicitud(ctx, ports.AsociacionDatos{PersonaRef: persona, ConvocatoriaRef: e.ConvocatoriaRef, Version: e.VersionEsperada + 1}, claro)
	if err != nil {
		return ports.ResultadoGuardado{}, errors.Join(ports.ErrNoDisponible, err)
	}
	nueva, err := s.referencias.NuevaReferenciaSolicitud(ctx)
	if err != nil || !solicitudRefValida.MatchString(nueva) {
		return ports.ResultadoGuardado{}, errors.Join(ports.ErrNoDisponible, err)
	}
	return s.repositorio.GuardarBorrador(ctx, ports.ComandoGuardarBorrador{
		PersonaRef: persona, ConvocatoriaRef: e.ConvocatoriaRef, ConvocatoriaVersion: convocatoria.Version, VersionEsperada: e.VersionEsperada,
		Clave: e.Clave, HuellaMaterial: huellaMaterial, SolicitudRefNueva: nueva, Turno: borrador.Turno, Sobre: sobre,
		DocumentoHuella: documento, DocumentoParcial: borrador.Datos.DocumentoParcial(), DatosCompletos: preparado.Completo,
		Requisitos: borrador.Requisitos, Meritos: borrador.Meritos, Puntuacion: preparado.Autobaremo.Total, Material: material,
	})
}

// huellaMaterialBorrador fija el material de la clave de idempotencia con
// clave (no expone datos personales): mismo contenido, misma huella.
func (s *ServicioSolicitudesPropias) huellaMaterialBorrador(ctx context.Context, persona string, e EntradaBorrador, b domain.Borrador, datos []byte) (string, error) {
	documento, err := json.Marshal(struct {
		Persona      string                      `json:"persona_ref"`
		Convocatoria string                      `json:"convocatoria_ref"`
		Version      int                         `json:"version_esperada"`
		Turno        string                      `json:"turno"`
		Datos        json.RawMessage             `json:"datos"`
		Requisitos   []domain.RequisitoDeclarado `json:"requisitos"`
		Meritos      []domain.MeritoDeclarado    `json:"meritos"`
	}{persona, e.ConvocatoriaRef, e.VersionEsperada, b.Turno, datos, b.Requisitos, b.Meritos})
	if err != nil {
		return "", ports.ErrDatosNoValidos
	}
	huella, err := s.protector.HuellaConClave(ctx, "seleccion.material_borrador.v1", documento)
	if err != nil {
		return "", errors.Join(ports.ErrNoDisponible, err)
	}
	return huella, nil
}

// EntradaPresentacion: la persona confirma la versión que vio y la
// declaración responsable.
type EntradaPresentacion struct {
	SolicitudRef           string
	VersionEsperada        int
	Clave                  string
	DeclaracionResponsable bool
}

// ReciboPresentacion con el estado expreso de cada servicio externo.
type ReciboPresentacion struct {
	ports.ResultadoPresentacion
	Servicios map[string]ports.EstadoServicioExterno
}

// Presentar presenta la solicitud: plazo, versión, requisitos y numeración
// los decide la base en la misma transacción que consume la decisión.
func (s *ServicioSolicitudesPropias) Presentar(ctx context.Context, o Orden, e EntradaPresentacion) (ReciboPresentacion, error) {
	if s == nil || ctx == nil {
		return ReciboPresentacion{}, ports.ErrNoDisponible
	}
	if !e.DeclaracionResponsable {
		return ReciboPresentacion{}, ports.ErrDeclaracionRequerida
	}
	if !solicitudRefValida.MatchString(e.SolicitudRef) || !claveValida.MatchString(e.Clave) || e.VersionEsperada < 1 || e.VersionEsperada > 1_000_000 {
		return ReciboPresentacion{}, ports.ErrDatosNoValidos
	}
	material, persona, err := s.autorizarPropia(ctx, o, ports.AccionPresentar, ports.AudienciaPresentar)
	if err != nil {
		return ReciboPresentacion{}, err
	}
	documento := []byte(persona + "\x1f" + e.SolicitudRef + "\x1f" + strconv.Itoa(e.VersionEsperada) + "\x1fdeclaracion_responsable")
	huella, err := s.protector.HuellaConClave(ctx, "seleccion.material_presentacion.v1", documento)
	if err != nil {
		return ReciboPresentacion{}, errors.Join(ports.ErrNoDisponible, err)
	}
	suma := sha256.Sum256([]byte(e.SolicitudRef + "\x1f" + e.Clave))
	resultado, err := s.repositorio.Presentar(ctx, ports.ComandoPresentar{
		PersonaRef: persona, SolicitudRef: e.SolicitudRef, VersionEsperada: e.VersionEsperada, Clave: e.Clave,
		HuellaMaterial: huella, ReciboRef: "recibo:seleccion-presentacion:" + hex.EncodeToString(suma[:]), Material: material,
	})
	if err != nil {
		return ReciboPresentacion{}, err
	}
	return ReciboPresentacion{ResultadoPresentacion: resultado, Servicios: s.serviciosExternos(ctx, resultado)}, nil
}

// serviciosExternos informa expresamente de lo que no se ha hecho. Un fallo
// o una indisponibilidad nunca se presentan como éxito.
func (s *ServicioSolicitudesPropias) serviciosExternos(ctx context.Context, r ports.ResultadoPresentacion) map[string]ports.EstadoServicioExterno {
	p := ports.PresentacionExterna{SolicitudRef: r.SolicitudRef, NumeroJustificante: r.NumeroJustificante, ReciboRef: r.ReciboRef, PresentadaEn: r.PresentadaEn}
	estado := func(err error) ports.EstadoServicioExterno {
		switch {
		case err == nil:
			return ports.ServicioRealizado
		case errors.Is(err, ports.ErrServicioExternoNoDisponible):
			return ports.ServicioNoDisponible
		default:
			return ports.ServicioFallido
		}
	}
	return map[string]ports.EstadoServicioExterno{
		"firma":         estado(s.externos.Firma.FirmarPresentacion(ctx, p)),
		"registro_sede": estado(s.externos.Registro.RegistrarPresentacion(ctx, p)),
		"tasas":         estado(s.externos.Tasas.ComprobarTasa(ctx, p)),
		"notificacion":  estado(s.externos.Notificacion.NotificarPresentacion(ctx, p)),
	}
}
