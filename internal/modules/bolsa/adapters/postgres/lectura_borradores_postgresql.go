package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	gobiernoconvocatorias "vec-diputacion-granada/internal/modules/bolsa/application/gobiernoconvocatorias"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const (
	funcionListarBorradoresPostgreSQL = "vec_bolsa_convocatorias.listar_borradores_v1"
	funcionObtenerBorradorPostgreSQL  = "vec_bolsa_convocatorias.obtener_borrador_v1"
	duracionMaximaDescifradoBorrador  = 5 * time.Second
)

var _ gobiernoconvocatorias.RepositorioLecturaBorradoresGobernada = (*RepositorioLecturaBorradoresPostgreSQL)(nil)

type RepositorioLecturaBorradoresPostgreSQL struct {
	pool        iniciadorTransacciones
	descifrador gobiernoconvocatorias.DescifradorBorradorDurable
}

type selectorListaBorradoresPostgreSQL struct {
	Categoria string `json:"categoria"`
	Cursor    string `json:"cursor"`
	Limite    int    `json:"limite"`
	Texto     string `json:"texto"`
}

type lecturaBorradorPostgreSQL struct {
	Accion                 string `json:"accion"`
	DecisionRef            string `json:"decision_ref"`
	HuellaDecisionSHA256   string `json:"huella_decision_sha256"`
	AtestacionRef          string `json:"atestacion_ref"`
	AtestacionVersion      uint32 `json:"atestacion_version"`
	EstadoAtestacion       string `json:"estado_atestacion"`
	HuellaAtestacionSHA256 string `json:"huella_atestacion_sha256"`
	RecursoRef             string `json:"recurso_ref"`
	OrganizacionRef        string `json:"organizacion_ref"`
	UnidadGestionRef       string `json:"unidad_gestion_ref"`
}

func NuevoRepositorioLecturaBorradoresPostgreSQL(pool *pgxpool.Pool, descifrador gobiernoconvocatorias.DescifradorBorradorDurable) (*RepositorioLecturaBorradoresPostgreSQL, error) {
	return nuevoRepositorioLecturaBorradoresPostgreSQL(pool, descifrador)
}

func nuevoRepositorioLecturaBorradoresPostgreSQL(pool iniciadorTransacciones, descifrador gobiernoconvocatorias.DescifradorBorradorDurable) (*RepositorioLecturaBorradoresPostgreSQL, error) {
	if valorNulo(pool) || valorNulo(descifrador) {
		return nil, gobiernoconvocatorias.ErrLecturaBorradoresGobernadaInvalida
	}
	return &RepositorioLecturaBorradoresPostgreSQL{pool: pool, descifrador: descifrador}, nil
}

func (r *RepositorioLecturaBorradoresPostgreSQL) ListarBorradoresGobernados(ctx context.Context, s gobiernoconvocatorias.SolicitudListadoBorradoresGobernada) (gobiernoconvocatorias.ListaBorradores, error) {
	if ctx == nil || r == nil || valorNulo(r.pool) || ctx.Err() != nil || s.Selector.Validar() != nil {
		return gobiernoconvocatorias.ListaBorradores{}, errorLecturaBorrador(ctx, nil)
	}
	carga, err := cargaLecturaBorrador(s.Contexto, s.Capacidad, gobiernoconvocatorias.AccionListarBorradoresGobernados, "")
	if err != nil {
		return gobiernoconvocatorias.ListaBorradores{}, gobiernoconvocatorias.ErrPreautorizacionLecturaBorrador
	}
	defer carga.borrar()
	selector, err := json.Marshal(selectorListaBorradoresPostgreSQL{
		Categoria: s.Selector.Categoria, Cursor: s.Selector.Cursor,
		Limite: s.Selector.Limite, Texto: s.Selector.Texto,
	})
	if err != nil {
		return gobiernoconvocatorias.ListaBorradores{}, gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	defer borrarBytesPostgreSQL(selector)
	tx, err := iniciarTransaccionBorradorPostgreSQL(ctx, r.pool, pgx.ReadWrite)
	if err != nil {
		return gobiernoconvocatorias.ListaBorradores{}, errorLecturaBorrador(ctx, err)
	}
	defer revertir(tx)
	var salida []byte
	err = tx.QueryRow(ctx, `SELECT `+funcionListarBorradoresPostgreSQL+`($1::jsonb,$2::jsonb,$3::jsonb,$4::bytea,$5::bytea)`, selector, carga.lectura, carga.prueba, carga.decision, carga.contexto).Scan(&salida)
	defer borrarBytesPostgreSQL(salida)
	if err != nil {
		return gobiernoconvocatorias.ListaBorradores{}, errorLecturaBorrador(ctx, err)
	}
	resultado, err := restaurarListaBorradores(salida, s.Selector)
	if err != nil {
		return gobiernoconvocatorias.ListaBorradores{}, err
	}
	if err := ctx.Err(); err != nil {
		return gobiernoconvocatorias.ListaBorradores{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return gobiernoconvocatorias.ListaBorradores{}, errorLecturaBorrador(ctx, err)
	}
	return resultado, nil
}

func (r *RepositorioLecturaBorradoresPostgreSQL) ObtenerBorradorGobernado(ctx context.Context, s gobiernoconvocatorias.SolicitudDetalleBorradorGobernada) (gobiernoconvocatorias.DetalleBorrador, error) {
	if ctx == nil || r == nil || valorNulo(r.pool) || valorNulo(r.descifrador) || ctx.Err() != nil || s.Selector.Validar() != nil {
		return gobiernoconvocatorias.DetalleBorrador{}, errorLecturaBorrador(ctx, nil)
	}
	carga, err := cargaLecturaBorrador(s.Contexto, s.Capacidad, gobiernoconvocatorias.AccionConsultarBorradorGobernado, s.Selector.Referencia())
	if err != nil {
		return gobiernoconvocatorias.DetalleBorrador{}, gobiernoconvocatorias.ErrPreautorizacionLecturaBorrador
	}
	defer carga.borrar()
	solicitud, metadatos, err := leerSolicitudBorradorCifrado(
		ctx, r.pool, s.Selector.Referencia(), carga,
	)
	if err != nil {
		return gobiernoconvocatorias.DetalleBorrador{}, err
	}
	descifrado, err := descifrarBorradorConPlazo(
		ctx, r.descifrador, solicitud, duracionMaximaDescifradoBorrador,
	)
	if err != nil {
		return gobiernoconvocatorias.DetalleBorrador{}, err
	}
	version, err := descifrado.VersionConvocatoria()
	if err != nil || !metadatos.coincide(version, s.Selector) {
		return gobiernoconvocatorias.DetalleBorrador{}, gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	detalle, err := detalleBorrador(version, metadatos.Estado)
	if err != nil {
		return gobiernoconvocatorias.DetalleBorrador{}, gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	return detalle, nil
}

func descifrarBorradorConPlazo(
	ctx context.Context,
	descifrador gobiernoconvocatorias.DescifradorBorradorDurable,
	solicitud gobiernoconvocatorias.SolicitudDescifradoBorradorDurable,
	plazo time.Duration,
) (gobiernoconvocatorias.ResultadoDescifradoBorradorDurable, error) {
	if ctx == nil || valorNulo(descifrador) || plazo <= 0 || ctx.Err() != nil {
		return gobiernoconvocatorias.ResultadoDescifradoBorradorDurable{}, errorLecturaBorrador(ctx, nil)
	}
	ctxKMS, cancelar := context.WithTimeout(ctx, plazo)
	defer cancelar()
	resultado, err := descifrador.DescifrarBorrador(ctxKMS, solicitud)
	if err != nil {
		if ctx.Err() != nil {
			return gobiernoconvocatorias.ResultadoDescifradoBorradorDurable{}, ctx.Err()
		}
		if ctxKMS.Err() != nil {
			return gobiernoconvocatorias.ResultadoDescifradoBorradorDurable{}, gobiernoconvocatorias.ErrOperacionBorradorEnCurso
		}
		return gobiernoconvocatorias.ResultadoDescifradoBorradorDurable{}, gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	if ctx.Err() != nil {
		return gobiernoconvocatorias.ResultadoDescifradoBorradorDurable{}, ctx.Err()
	}
	if ctxKMS.Err() != nil {
		return gobiernoconvocatorias.ResultadoDescifradoBorradorDurable{}, gobiernoconvocatorias.ErrOperacionBorradorEnCurso
	}
	return resultado, nil
}

func leerSolicitudBorradorCifrado(
	ctx context.Context, pool iniciadorTransacciones, referencia string, carga cargaLecturaPostgreSQL,
) (gobiernoconvocatorias.SolicitudDescifradoBorradorDurable, metadatosBorrador, error) {
	tx, err := iniciarTransaccionBorradorPostgreSQL(ctx, pool, pgx.ReadWrite)
	if err != nil {
		return gobiernoconvocatorias.SolicitudDescifradoBorradorDurable{}, metadatosBorrador{}, errorLecturaBorrador(ctx, err)
	}
	defer revertir(tx)
	fila, err := consultarBorradorCifrado(ctx, tx, referencia, carga)
	if err != nil {
		fila.borrar()
		return gobiernoconvocatorias.SolicitudDescifradoBorradorDurable{}, metadatosBorrador{}, errorLecturaBorrador(ctx, err)
	}
	solicitud, metadatos, err := fila.restaurarSolicitud()
	fila.borrar()
	if err != nil {
		return gobiernoconvocatorias.SolicitudDescifradoBorradorDurable{}, metadatosBorrador{}, gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	if err := ctx.Err(); err != nil {
		return gobiernoconvocatorias.SolicitudDescifradoBorradorDurable{}, metadatosBorrador{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return gobiernoconvocatorias.SolicitudDescifradoBorradorDurable{}, metadatosBorrador{}, errorLecturaBorrador(ctx, err)
	}
	return solicitud, metadatos, nil
}

type cargaLecturaPostgreSQL struct{ lectura, prueba, decision, contexto []byte }

func (c cargaLecturaPostgreSQL) borrar() {
	borrarBytesPostgreSQL(c.lectura, c.prueba, c.decision, c.contexto)
}
func cargaLecturaBorrador(contexto gobiernoconvocatorias.ContextoOperacionBorrador, c gobiernoconvocatorias.CapacidadLecturaBorrador, accion, referencia string) (cargaLecturaPostgreSQL, error) {
	if err := validarCapacidadLecturaLocal(contexto, c, accion, referencia); err != nil {
		return cargaLecturaPostgreSQL{}, err
	}
	datos, err := c.Evidencia.Datos()
	if err != nil {
		return cargaLecturaPostgreSQL{}, err
	}
	decision, err := datos.RepresentacionCanonica()
	if err != nil {
		return cargaLecturaPostgreSQL{}, err
	}
	contextoCanonico, err := serializarContextoRecursoBorradorPostgreSQL(c.Recurso)
	if err != nil {
		borrarBytesPostgreSQL(decision)
		return cargaLecturaPostgreSQL{}, err
	}
	lectura, err := json.Marshal(lecturaBorradorPostgreSQL{
		Accion: accion, DecisionRef: datos.Decision.DecisionRef,
		HuellaDecisionSHA256: datos.HuellaDecisionSHA256,
		AtestacionRef:        c.AtestacionRef, AtestacionVersion: c.VersionAtestacion,
		EstadoAtestacion: c.EstadoAtestacion, HuellaAtestacionSHA256: c.HuellaAtestacionSHA256,
		RecursoRef: c.Recurso.Referencia, OrganizacionRef: c.OrganizacionRef,
		UnidadGestionRef: c.UnidadGestionRef,
	})
	prueba, errP := json.Marshal(pruebaDecisionBorradorPostgreSQL{EsquemaHuella: datos.EsquemaHuella, DecisionRef: datos.Decision.DecisionRef, HuellaDecisionSHA256: datos.HuellaDecisionSHA256, VerificadaEn: datos.VerificadaEn.UTC().Format(formatoInstanteMicrosegundo), PrincipalRef: datos.Decision.PrincipalID})
	if err != nil || errP != nil {
		borrarBytesPostgreSQL(decision, contextoCanonico, lectura, prueba)
		return cargaLecturaPostgreSQL{}, gobiernoconvocatorias.ErrPreautorizacionLecturaBorrador
	}
	return cargaLecturaPostgreSQL{lectura, prueba, decision, contextoCanonico}, nil
}

func validarCapacidadLecturaLocal(contexto gobiernoconvocatorias.ContextoOperacionBorrador, c gobiernoconvocatorias.CapacidadLecturaBorrador, accion, referencia string) error {
	actor, ea := contexto.Actor.Clonar()
	datos, ed := c.Evidencia.Datos()
	solicitud, es := c.Solicitud.Datos()
	correlacion, ec := solicitud.Correlacion.ValorCanonico()
	huellaSolicitud, ehSolicitud := dominiovec.HuellaSHA256SolicitudAutorizacionV2(c.Solicitud)
	huella, eh := c.Recurso.HuellaContextoAutorizacionSHA256()
	d := datos.Decision
	esperado, tipo := referencia, puertosbolsa.TipoRecursoVersionConvocatoriaGobernada
	if accion == gobiernoconvocatorias.AccionListarBorradoresGobernados {
		esperado = "borradores:" + c.OrganizacionRef
		tipo = gobiernoconvocatorias.TipoColeccionBorradoresGobernados
	}
	if ea != nil || ed != nil || es != nil || ec != nil || ehSolicitud != nil || eh != nil ||
		contexto.Vinculo.ValidarPara(actor) != nil || c.Evidencia.ValidarEn(datos.VerificadaEn) != nil ||
		c.Evidencia.ValidarMotivo(c.Motivo) != nil || c.Recurso.Validar() != nil ||
		c.EstadoAtestacion != "activa" || c.VersionAtestacion == 0 || len(c.HuellaAtestacionSHA256) != 64 ||
		!dominiovec.ReferenciaCorrelacionAutorizacionV2Valida(contexto.CorrelacionRef) ||
		d.Accion != accion || d.RecursoRef != esperado || c.Recurso.Referencia != esperado ||
		d.ModuloID != puertosbolsa.ModuloGobiernoConvocatorias || d.TipoRecurso != tipo ||
		d.Finalidad != gobiernoconvocatorias.FinalidadLecturaBorradoresGobernada ||
		d.PrincipalID != actor.PersonaRef || d.PerfilActivoRef != actor.PerfilActivoRef ||
		d.CorrelacionRef != contexto.CorrelacionRef || d.ContextoRecursoHuellaSHA256 != huella ||
		d.EsquemaHuellaSolicitud != dominiovec.EsquemaHuellaSolicitudAutorizacionV2 ||
		d.SolicitudHuellaSHA256 != huellaSolicitud ||
		d.EsquemaHuellaMotivo != dominiovec.EsquemaHuellaMotivoAutorizacionV2 ||
		solicitud.Accion != accion || solicitud.Finalidad != gobiernoconvocatorias.FinalidadLecturaBorradoresGobernada ||
		correlacion != contexto.CorrelacionRef || solicitud.ReferenciaMotivo != c.Motivo ||
		!reflect.DeepEqual(solicitud.ContextoActor, actor) ||
		!reflect.DeepEqual(solicitud.VinculoAutenticacionActor, contexto.Vinculo) ||
		!reflect.DeepEqual(solicitud.Recurso, c.Recurso) ||
		!reflect.DeepEqual(c.Recurso.Atributos, map[string]string{}) ||
		c.Recurso.Ambitos["organizacion_ref"] != c.OrganizacionRef ||
		c.Recurso.Ambitos["unidad_gestion_ref"] != c.UnidadGestionRef ||
		len(c.Recurso.Ambitos) != numeroAmbitosLecturaPostgreSQL(c.UnidadGestionRef) {
		return gobiernoconvocatorias.ErrPreautorizacionLecturaBorrador
	}
	return nil
}

func numeroAmbitosLecturaPostgreSQL(unidad string) int {
	if unidad == "" {
		return 1
	}
	return 2
}

func errorLecturaBorrador(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var p *pgconn.PgError
	if errors.As(err, &p) {
		switch p.Code {
		case "42501":
			return dominiovec.ErrAutorizacionDenegada
		case "40001", "40P01", "55P03", "57014":
			return gobiernoconvocatorias.ErrOperacionBorradorEnCurso
		case "22000", "22003", "22023", "23503", "23514", "55000", "P0002":
			return gobiernoconvocatorias.ErrResultadoBorradorInseguro
		}
	}
	return gobiernoconvocatorias.ErrLecturaBorradoresGobernadaInvalida
}

type filaBorradorCifrado struct {
	metadatos, aad, perfil, atestacion, procedencia            []byte
	huellaAAD, esqEnv, clave, huellaEnv, esqSobre, huellaSobre string
	versionClave                                               int64
	envuelto, nonce, cifrado                                   []byte
}

func (f *filaBorradorCifrado) borrar() {
	borrarBytesPostgreSQL(f.metadatos, f.aad, f.perfil, f.atestacion, f.procedencia, f.envuelto, f.nonce, f.cifrado)
}
func consultarBorradorCifrado(ctx context.Context, tx pgx.Tx, referencia string, c cargaLecturaPostgreSQL) (filaBorradorCifrado, error) {
	var f filaBorradorCifrado
	err := tx.QueryRow(ctx, `SELECT * FROM `+funcionObtenerBorradorPostgreSQL+`($1::text,$2::jsonb,$3::jsonb,$4::bytea,$5::bytea)`, referencia, c.lectura, c.prueba, c.decision, c.contexto).Scan(&f.metadatos, &f.aad, &f.huellaAAD, &f.perfil, &f.esqEnv, &f.clave, &f.versionClave, &f.envuelto, &f.huellaEnv, &f.esqSobre, &f.nonce, &f.cifrado, &f.huellaSobre, &f.atestacion, &f.procedencia)
	return f, err
}

type metadatosBorrador struct {
	Estado               puertosbolsa.ReferenciaEstadoVersionConvocatoria `json:"referencia_estado"`
	ETag                 string                                           `json:"etag"`
	CodigoVersionPublica string                                           `json:"codigo_version_publica"`
	IdentificadorPublico string                                           `json:"identificador_publico"`
	Ambito               struct {
		OrganizacionRef  string `json:"organizacion_ref"`
		UnidadGestionRef string `json:"unidad_gestion_ref"`
	} `json:"ambito_lectura"`
	ExpedienteRef string `json:"expediente_ref"`
}

func (m metadatosBorrador) coincide(v dominiobolsa.VersionConvocatoriaGobernada, selector puertosbolsa.SelectorVersionConvocatoriaExacta) bool {
	return m.Estado.Validar() == nil && v.Validar() == nil &&
		m.Estado.Referencia == selector.Referencia() && v.Referencia() == m.Estado.Referencia &&
		v.Revision == m.Estado.Revision &&
		m.ETag == `"`+strconv.Itoa(m.Estado.Revision)+`-`+m.Estado.HuellaEstadoSHA256+`"` &&
		m.CodigoVersionPublica == v.CodigoVersionPublica &&
		m.IdentificadorPublico == v.Contenido.IdentificadorPublico &&
		m.Ambito.OrganizacionRef == v.AmbitoOrganizativo.OrganizacionRef() &&
		m.Ambito.UnidadGestionRef == v.AmbitoOrganizativo.UnidadGestionRef() &&
		m.ExpedienteRef == v.ExpedienteRef
}

func (f filaBorradorCifrado) restaurarSolicitud() (gobiernoconvocatorias.SolicitudDescifradoBorradorDurable, metadatosBorrador, error) {
	var m metadatosBorrador
	var p perfilCifradoReciboPostgreSQL
	var a atestacionKMSBorradorPostgreSQL
	var pr procedenciaReciboPostgreSQL
	if decodificarJSONCerradoDiarioPostgreSQL(f.metadatos, &m) != nil || decodificarJSONCerradoDiarioPostgreSQL(f.perfil, &p) != nil || decodificarJSONCerradoDiarioPostgreSQL(f.atestacion, &a) != nil || decodificarJSONCerradoDiarioPostgreSQL(f.procedencia, &pr) != nil || m.Estado.Validar() != nil || f.versionClave < 1 || f.versionClave > 1<<32-1 {
		return gobiernoconvocatorias.SolicitudDescifradoBorradorDurable{}, metadatosBorrador{}, gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	perfil, e := gobiernoconvocatorias.NuevoPerfilCifradoBorrador(p.Referencia, p.Version, p.HuellaContenidoSHA256, p.AlgoritmoAEAD, p.AlgoritmoEnvolturaClave)
	proc, ep := restaurarProcedenciaReciboPostgreSQL(pr)
	aad, ea := gobiernoconvocatorias.RestaurarAADCanonicaCifradoBorrador(f.aad, f.huellaAAD)
	env, ee := gobiernoconvocatorias.RestaurarEnvolturaClaveKMSBorrador(f.esqEnv, perfil, f.clave, uint32(f.versionClave), f.envuelto, f.huellaAAD, f.huellaEnv)
	sobre, es := gobiernoconvocatorias.RestaurarSobreCifradoAEADBorrador(f.esqSobre, perfil, f.nonce, f.cifrado, f.huellaAAD, f.huellaSobre)
	firma, ef := restaurarFirmaEvidenciaReciboPostgreSQL(a.Firma)
	emitida, ei := instanteJSONDiarioPostgreSQL(a.EmitidaEn)
	vence, ev := instanteJSONDiarioPostgreSQL(a.ValidaHasta)
	atest, et := gobiernoconvocatorias.RestaurarAtestacionKMSBorrador(a.Esquema, a.AtestacionRef, a.VersionAtestacion, a.Estado, perfil, a.ClaveMaestraRef, a.VersionClave, a.HuellaAAD, a.HuellaEnvolturaSHA256, a.HuellaSobreSHA256, a.VerificadorRef, proc, firma, emitida, vence)
	s, er := gobiernoconvocatorias.NuevaSolicitudDescifradoBorradorDurable(m.Estado, aad, perfil, env, sobre, atest, proc)
	if errors.Join(e, ep, ea, ee, es, ef, ei, ev, et, er) != nil {
		return gobiernoconvocatorias.SolicitudDescifradoBorradorDurable{}, metadatosBorrador{}, gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	return s, m, nil
}

func detalleBorrador(v dominiobolsa.VersionConvocatoriaGobernada, e puertosbolsa.ReferenciaEstadoVersionConvocatoria) (gobiernoconvocatorias.DetalleBorrador, error) {
	if v.Validar() != nil || e.Validar() != nil {
		return gobiernoconvocatorias.DetalleBorrador{}, gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	docs := make([]gobiernoconvocatorias.DocumentoLecturaBorrador, 0, len(v.Configuracion.Documentos))
	for _, d := range v.Configuracion.Documentos {
		docs = append(docs, gobiernoconvocatorias.DocumentoLecturaBorrador{Rol: d.Rol, PublicacionRef: d.PublicacionRef, DocumentoRef: d.DocumentoRef, VersionDocumento: d.VersionDocumento, RepresentacionRef: d.RepresentacionRef, HuellaContenidoSHA256: d.HuellaContenidoSHA256, FirmaValidadaRef: d.FirmaValidadaRef, ReciboCustodiaRef: d.ReciboCustodiaRef})
	}
	r := func(x dominiobolsa.ReferenciaConfiguracionConvocatoria) gobiernoconvocatorias.ReferenciaConfiguracionLecturaBorrador {
		return gobiernoconvocatorias.ReferenciaConfiguracionLecturaBorrador{Referencia: x.ID, Version: x.Version, HuellaSHA256: x.HuellaContenidoSHA256}
	}
	return gobiernoconvocatorias.DetalleBorrador{Estado: e, CodigoVersionPublica: v.CodigoVersionPublica, IdentificadorPublico: v.Contenido.IdentificadorPublico, Ambito: gobiernoconvocatorias.AmbitoLecturaBorrador{OrganizacionRef: v.AmbitoOrganizativo.OrganizacionRef(), UnidadGestionRef: v.AmbitoOrganizativo.UnidadGestionRef()}, ExpedienteRef: v.ExpedienteRef, Contenido: gobiernoconvocatorias.ContenidoEditableBorrador{Tipo: v.Contenido.Tipo, Categorias: append([]string(nil), v.Contenido.Categorias...), Titulo: v.Contenido.Titulo, Resumen: v.Contenido.Resumen, Descripcion: v.Contenido.Descripcion, Plazos: append([]dominiobolsa.PlazoConvocatoria(nil), v.Contenido.Plazos...), Requisitos: append([]dominiobolsa.RequisitoConvocatoria(nil), v.Contenido.Requisitos...), Ayuda: append([]dominiobolsa.AyudaConvocatoria(nil), v.Contenido.Ayuda...)}, Configuracion: gobiernoconvocatorias.ConfiguracionLecturaBorrador{Catalogos: r(v.Configuracion.Catalogos), Calendario: r(v.Configuracion.Calendario), ReglasBaremacion: r(v.Configuracion.ReglasBaremacion), FlujoProceso: r(v.Configuracion.FlujoProceso), FlujoSolicitud: r(v.Configuracion.FlujoSolicitud), Plantilla: r(v.Configuracion.Plantilla), Documentos: docs}, Capacidades: gobiernoconvocatorias.CapacidadesFilaBorrador{Consultar: true}}, nil
}

type listaBorradoresPersistida struct {
	Esquema    string                            `json:"esquema"`
	Selector   selectorListaBorradoresPostgreSQL `json:"selector"`
	Paginacion struct {
		Limite    int    `json:"limite"`
		Total     int    `json:"total"`
		Siguiente string `json:"siguiente_cursor"`
	} `json:"paginacion"`
	Capacidades gobiernoconvocatorias.CapacidadesGlobalesBorrador `json:"capacidades"`
	Elementos   []filaBorradorPersistida                          `json:"elementos"`
}

type filaBorradorPersistida struct {
	Estado               puertosbolsa.ReferenciaEstadoVersionConvocatoria `json:"referencia_estado"`
	ETag                 string                                           `json:"etag"`
	CodigoVersionPublica string                                           `json:"codigo_version_publica"`
	IdentificadorPublico string                                           `json:"identificador_publico"`
	Titulo               string                                           `json:"titulo"`
	Tipo                 string                                           `json:"tipo"`
	Categorias           []string                                         `json:"categorias"`
	ExpedienteRef        string                                           `json:"expediente_ref"`
	CreadaEn             time.Time                                        `json:"creada_en"`
	ActualizadaEn        time.Time                                        `json:"actualizada_en"`
	NumeroPlazos         int                                              `json:"numero_plazos"`
	NumeroRequisitos     int                                              `json:"numero_requisitos"`
	NumeroDocumentos     int                                              `json:"numero_documentos"`
	NumeroAyudas         int                                              `json:"numero_ayudas"`
	Capacidades          gobiernoconvocatorias.CapacidadesFilaBorrador    `json:"capacidades"`
}

func restaurarListaBorradores(b []byte, s gobiernoconvocatorias.SelectorListaBorradores) (gobiernoconvocatorias.ListaBorradores, error) {
	var p listaBorradoresPersistida
	esperado := selectorListaBorradoresPostgreSQL{Categoria: s.Categoria, Cursor: s.Cursor, Limite: s.Limite, Texto: s.Texto}
	if decodificarJSONCerradoDiarioPostgreSQL(b, &p) != nil || p.Esquema != "vec.bolsa.borradores.lista.v1" || p.Selector != esperado || p.Paginacion.Limite != s.Limite || p.Paginacion.Total < 0 || len(p.Elementos) > s.Limite || p.Capacidades.Consultar != true {
		return gobiernoconvocatorias.ListaBorradores{}, gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	elementos := make([]gobiernoconvocatorias.FilaBorrador, 0, len(p.Elementos))
	for i := range p.Elementos {
		f := p.Elementos[i]
		etag := `"` + strconv.Itoa(f.Estado.Revision) + `-` + f.Estado.HuellaEstadoSHA256 + `"`
		if f.Estado.Validar() != nil || f.ETag != etag || f.CodigoVersionPublica == "" || f.IdentificadorPublico == "" || f.Titulo == "" || f.Tipo == "" || len(f.Categorias) == 0 || f.ExpedienteRef == "" || !instanteLecturaPostgreSQLCanonico(f.CreadaEn) || !instanteLecturaPostgreSQLCanonico(f.ActualizadaEn) || f.ActualizadaEn.Before(f.CreadaEn) || f.NumeroPlazos < 0 || f.NumeroRequisitos < 0 || f.NumeroDocumentos < 0 || f.NumeroAyudas < 0 || !f.Capacidades.Consultar {
			return gobiernoconvocatorias.ListaBorradores{}, gobiernoconvocatorias.ErrResultadoBorradorInseguro
		}
		elementos = append(elementos, gobiernoconvocatorias.FilaBorrador{Estado: f.Estado, CodigoVersionPublica: f.CodigoVersionPublica, IdentificadorPublico: f.IdentificadorPublico, Titulo: f.Titulo, Tipo: f.Tipo, Categorias: append([]string(nil), f.Categorias...), ExpedienteRef: f.ExpedienteRef, CreadaEn: f.CreadaEn, ActualizadaEn: f.ActualizadaEn, NumeroPlazos: f.NumeroPlazos, NumeroRequisitos: f.NumeroRequisitos, NumeroDocumentos: f.NumeroDocumentos, NumeroAyudas: f.NumeroAyudas, Capacidades: f.Capacidades})
	}
	return gobiernoconvocatorias.ListaBorradores{Selector: s, Total: p.Paginacion.Total, SiguienteCursor: p.Paginacion.Siguiente, Capacidades: p.Capacidades, Elementos: elementos}, nil
}

func instanteLecturaPostgreSQLCanonico(instante time.Time) bool {
	return !instante.IsZero() && instante.Location() == time.UTC && instante.Nanosecond()%1_000 == 0
}
