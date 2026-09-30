package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const (
	consultaProponerGobiernoRPT  = `SELECT vec_autorizacion_atestada_v3.proponer_gobierno_categoria_rpt_v3_atestada($1::jsonb,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,$8::bytea,$9::bytea,$10::bytea,$11::bytea)`
	consultaAprobarGobiernoRPT   = `SELECT vec_autorizacion_atestada_v3.aprobar_gobierno_categoria_rpt_v3_atestada($1::jsonb,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,$8::bytea,$9::bytea,$10::bytea,$11::bytea)`
	consultaConfirmarGobiernoRPT = `SELECT vec_autorizacion_atestada_v3.confirmar_gobierno_categoria_rpt_v3_atestada($1::jsonb,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,$8::bytea,$9::bytea,$10::bytea,$11::bytea)`
	maximoMaterialGobiernoRPT    = 18 << 20
)

// El pool usa exclusivamente el LOGIN técnico de gobierno; no se elige perfil
// desde material de negocio. La autorización efectiva la decide AD3-134.
type GestorGobiernoCategoriaRPTPostgreSQL struct {
	pool       iniciadorLecturaRPT
	descriptor ports.DescriptorCatalogoRPT
}

var _ ports.PreparadorGobiernoCategoriaRPT = (*GestorGobiernoCategoriaRPTPostgreSQL)(nil)
var _ ports.GestorGobiernoCategoriaRPT = (*GestorGobiernoCategoriaRPTPostgreSQL)(nil)

func NuevoGestorGobiernoCategoriaRPTPostgreSQL(pool *pgxpool.Pool, descriptor ports.DescriptorCatalogoRPT) (*GestorGobiernoCategoriaRPTPostgreSQL, error) {
	return nuevoGestorGobiernoCategoriaRPTPostgreSQL(pool, descriptor)
}

func nuevoGestorGobiernoCategoriaRPTPostgreSQL(pool iniciadorLecturaRPT, descriptor ports.DescriptorCatalogoRPT) (*GestorGobiernoCategoriaRPTPostgreSQL, error) {
	if valorNuloPostgreSQL(pool) {
		return nil, ports.ErrGobiernoCategoriaRPTNoDisponible
	}
	if !claveRPT.MatchString(descriptor.CatalogoID) || !claveRPT.MatchString(descriptor.ModuloID) {
		return nil, ports.ErrGobiernoCategoriaRPTDenegado
	}
	return &GestorGobiernoCategoriaRPTPostgreSQL{pool: pool, descriptor: descriptor}, nil
}

func (g *GestorGobiernoCategoriaRPTPostgreSQL) PrepararPropuestaGobiernoCategoriaRPT(ctx context.Context, b ports.BorradorPropuestaGobiernoCategoriaRPT) (ports.PreparacionPropuestaGobiernoCategoriaRPT, error) {
	var cero ports.PreparacionPropuestaGobiernoCategoriaRPT
	if err := contextoGobiernoRPT(ctx); err != nil {
		return cero, err
	}
	if g == nil || valorNuloPostgreSQL(g.pool) {
		return cero, ports.ErrGobiernoCategoriaRPTNoDisponible
	}
	if !referenciaGobiernoRPTValida(b.PropuestaRef) || !referenciaGobiernoRPTValida(b.ReciboRef) ||
		b.Contenido.CatalogoID != g.descriptor.CatalogoID || b.Contenido.ModuloID != g.descriptor.ModuloID ||
		b.Contenido.PreimagenesControl == nil || b.Contenido.PreimagenesHuellaSHA256 != "" ||
		(b.Contenido.Accion != domain.AccionGobiernoCategoriaRPTPublicar && b.Contenido.Accion != domain.AccionGobiernoCategoriaRPTDeshabilitar) {
		return cero, ports.ErrGobiernoCategoriaRPTInvalido
	}
	tx, err := g.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted, AccessMode: pgx.ReadOnly})
	if err != nil {
		return cero, errorGobiernoRPT(ctx, err)
	}
	if valorNuloPostgreSQL(tx) {
		return cero, ports.ErrGobiernoCategoriaRPTNoDisponible
	}
	defer revertirUsoRPT(tx)
	if _, err := tx.Exec(ctx, configurarLecturaRPT); err != nil {
		return cero, errorGobiernoRPT(ctx, err)
	}
	preimagenes, err := serializarMaterialGobiernoRPT(b.Contenido.PreimagenesControl)
	if err != nil {
		return cero, err
	}
	defer borrarPiezasRPT(preimagenes)
	var huellaPreimagenes string
	if err := tx.QueryRow(ctx, consultaHuellaMaterialRPT, string(preimagenes)).Scan(&huellaPreimagenes); err != nil {
		return cero, errorGobiernoRPT(ctx, err)
	}
	if !huellaRPT.MatchString(huellaPreimagenes) {
		return cero, ports.ErrGobiernoCategoriaRPTNoConfiable
	}
	contenido := b.Contenido
	contenido.PreimagenesHuellaSHA256 = huellaPreimagenes
	contenidoBytes, err := serializarMaterialGobiernoRPT(contenido)
	if err != nil {
		return cero, err
	}
	defer borrarPiezasRPT(contenidoBytes)
	var huellaContenido string
	if err := tx.QueryRow(ctx, consultaHuellaMaterialRPT, string(contenidoBytes)).Scan(&huellaContenido); err != nil {
		return cero, errorGobiernoRPT(ctx, err)
	}
	if !huellaRPT.MatchString(huellaContenido) {
		return cero, ports.ErrGobiernoCategoriaRPTNoConfiable
	}
	m := ports.MaterialPropuestaGobiernoCategoriaRPT{PropuestaRef: b.PropuestaRef, Contenido: contenido,
		HuellaSHA256: huellaContenido, ReciboRef: b.ReciboRef}
	material, err := g.materialPropuesta(m)
	if err != nil {
		return cero, err
	}
	defer borrarPiezasRPT(material)
	var huellaMaterial string
	if err := tx.QueryRow(ctx, consultaHuellaMaterialRPT, string(material)).Scan(&huellaMaterial); err != nil {
		return cero, errorGobiernoRPT(ctx, err)
	}
	if !huellaRPT.MatchString(huellaMaterial) {
		return cero, ports.ErrGobiernoCategoriaRPTNoConfiable
	}
	recurso := g.recurso(m.PropuestaRef, huellaMaterial)
	if recurso.Validar() != nil {
		return cero, ports.ErrGobiernoCategoriaRPTInvalido
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if err := tx.Commit(ctx); err != nil {
		return cero, errorGobiernoRPT(ctx, err)
	}
	return ports.PreparacionPropuestaGobiernoCategoriaRPT{Material: m,
		Autorizable: ports.PreparacionGobiernoCategoriaRPT{Accion: ports.AccionProponerGobiernoCategoriaRPT,
			Finalidad: ports.FinalidadGobiernoCategoriaRPT, Audiencia: ports.AudienciaGobiernoCategoriaRPT,
			Recurso: recurso, HuellaPropuesta: huellaContenido}}, nil
}

func (g *GestorGobiernoCategoriaRPTPostgreSQL) PrepararAprobacionGobiernoCategoriaRPT(ctx context.Context, m ports.MaterialAvanceGobiernoCategoriaRPT) (ports.PreparacionGobiernoCategoriaRPT, error) {
	return g.prepararAvance(ctx, m, ports.AccionAprobarGobiernoCategoriaRPT)
}

func (g *GestorGobiernoCategoriaRPTPostgreSQL) PrepararConfirmacionGobiernoCategoriaRPT(ctx context.Context, m ports.MaterialAvanceGobiernoCategoriaRPT) (ports.PreparacionGobiernoCategoriaRPT, error) {
	return g.prepararAvance(ctx, m, ports.AccionConfirmarGobiernoCategoriaRPT)
}

func (g *GestorGobiernoCategoriaRPTPostgreSQL) prepararAvance(ctx context.Context, m ports.MaterialAvanceGobiernoCategoriaRPT, accion string) (ports.PreparacionGobiernoCategoriaRPT, error) {
	material, err := g.materialAvance(m, accion)
	if err != nil {
		return ports.PreparacionGobiernoCategoriaRPT{}, err
	}
	defer borrarPiezasRPT(material)
	return g.preparar(ctx, accion, m.PropuestaRef, m.HuellaSHA256, material)
}

func (g *GestorGobiernoCategoriaRPTPostgreSQL) materialPropuesta(m ports.MaterialPropuestaGobiernoCategoriaRPT) ([]byte, error) {
	if g == nil || valorNuloPostgreSQL(g.pool) {
		return nil, ports.ErrGobiernoCategoriaRPTNoDisponible
	}
	if !referenciaGobiernoRPTValida(m.PropuestaRef) || !referenciaGobiernoRPTValida(m.ReciboRef) ||
		!huellaRPT.MatchString(m.HuellaSHA256) ||
		m.Contenido.CatalogoID != g.descriptor.CatalogoID || m.Contenido.ModuloID != g.descriptor.ModuloID ||
		(m.Contenido.Accion != domain.AccionGobiernoCategoriaRPTPublicar && m.Contenido.Accion != domain.AccionGobiernoCategoriaRPTDeshabilitar) {
		return nil, ports.ErrGobiernoCategoriaRPTInvalido
	}
	return serializarMaterialGobiernoRPT(m)
}

func (g *GestorGobiernoCategoriaRPTPostgreSQL) materialAvance(m ports.MaterialAvanceGobiernoCategoriaRPT, accion string) ([]byte, error) {
	if g == nil || valorNuloPostgreSQL(g.pool) {
		return nil, ports.ErrGobiernoCategoriaRPTNoDisponible
	}
	if !referenciaGobiernoRPTValida(m.PropuestaRef) || !referenciaGobiernoRPTValida(m.ReciboRef) ||
		!huellaRPT.MatchString(m.HuellaSHA256) || m.CatalogoID != g.descriptor.CatalogoID ||
		m.ModuloID != g.descriptor.ModuloID ||
		(accion == ports.AccionAprobarGobiernoCategoriaRPT && m.RevisionEsperada != 1 && m.RevisionEsperada != 2) ||
		(accion == ports.AccionConfirmarGobiernoCategoriaRPT && m.RevisionEsperada != 3) {
		return nil, ports.ErrGobiernoCategoriaRPTInvalido
	}
	return serializarMaterialGobiernoRPT(m)
}

func serializarMaterialGobiernoRPT(m any) ([]byte, error) {
	b, err := json.Marshal(m)
	if err != nil || len(b) == 0 || len(b) > maximoMaterialGobiernoRPT {
		return nil, ports.ErrGobiernoCategoriaRPTInvalido
	}
	return b, nil
}

func referenciaGobiernoRPTValida(s string) bool {
	if len(s) < 3 || len(s) > 160 {
		return false
	}
	for _, c := range s {
		if c < '!' || c > '~' || c == '*' {
			return false
		}
	}
	return true
}

func (g *GestorGobiernoCategoriaRPTPostgreSQL) preparar(ctx context.Context, accion, ref, huella string, material []byte) (ports.PreparacionGobiernoCategoriaRPT, error) {
	var cero ports.PreparacionGobiernoCategoriaRPT
	if err := contextoGobiernoRPT(ctx); err != nil {
		return cero, err
	}
	tx, err := g.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted, AccessMode: pgx.ReadOnly})
	if err != nil {
		return cero, errorGobiernoRPT(ctx, err)
	}
	if valorNuloPostgreSQL(tx) {
		return cero, ports.ErrGobiernoCategoriaRPTNoDisponible
	}
	defer revertirUsoRPT(tx)
	if _, err := tx.Exec(ctx, configurarLecturaRPT); err != nil {
		return cero, errorGobiernoRPT(ctx, err)
	}
	var huellaMaterial string
	if err := tx.QueryRow(ctx, consultaHuellaMaterialRPT, string(material)).Scan(&huellaMaterial); err != nil {
		return cero, errorGobiernoRPT(ctx, err)
	}
	if !huellaRPT.MatchString(huellaMaterial) {
		return cero, ports.ErrGobiernoCategoriaRPTNoConfiable
	}
	recurso := g.recurso(ref, huellaMaterial)
	if recurso.Validar() != nil {
		return cero, ports.ErrGobiernoCategoriaRPTInvalido
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if err := tx.Commit(ctx); err != nil {
		return cero, errorGobiernoRPT(ctx, err)
	}
	return ports.PreparacionGobiernoCategoriaRPT{Accion: accion, Finalidad: ports.FinalidadGobiernoCategoriaRPT,
		Audiencia: ports.AudienciaGobiernoCategoriaRPT, Recurso: recurso, HuellaPropuesta: huella}, nil
}

func (g *GestorGobiernoCategoriaRPTPostgreSQL) recurso(ref, huella string) domain.RecursoAutorizable {
	return domain.RecursoAutorizable{Referencia: ref, ModuloID: g.descriptor.ModuloID,
		Tipo:      ports.TipoRecursoGobiernoCategoriaRPT,
		Ambitos:   map[string]string{"catalogo_id": g.descriptor.CatalogoID, "modulo_id": g.descriptor.ModuloID},
		Atributos: map[string]string{"material_sha256": huella}}
}

func contextoGobiernoRPT(ctx context.Context) error {
	if ctx == nil {
		return ports.ErrGobiernoCategoriaRPTInvalido
	}
	return ctx.Err()
}

func (g *GestorGobiernoCategoriaRPTPostgreSQL) ProponerGobiernoCategoriaRPT(ctx context.Context, o ports.OrdenPropuestaGobiernoCategoriaRPT) (ports.ResultadoGobiernoCategoriaRPT, error) {
	material, err := g.materialPropuesta(o.Material)
	if err != nil {
		return ports.ResultadoGobiernoCategoriaRPT{}, err
	}
	defer borrarPiezasRPT(material)
	return g.ejecutar(ctx, o.Solicitud, o.Autorizacion, material, o.Material.PropuestaRef, o.Material.HuellaSHA256,
		o.Material.ReciboRef, 1, domain.EstadoGobiernoCategoriaRPTPropuesta,
		ports.AccionProponerGobiernoCategoriaRPT, consultaProponerGobiernoRPT)
}

func (g *GestorGobiernoCategoriaRPTPostgreSQL) AprobarGobiernoCategoriaRPT(ctx context.Context, o ports.OrdenAvanceGobiernoCategoriaRPT) (ports.ResultadoGobiernoCategoriaRPT, error) {
	return g.avanzar(ctx, o, ports.AccionAprobarGobiernoCategoriaRPT, consultaAprobarGobiernoRPT)
}

func (g *GestorGobiernoCategoriaRPTPostgreSQL) ConfirmarGobiernoCategoriaRPT(ctx context.Context, o ports.OrdenAvanceGobiernoCategoriaRPT) (ports.ResultadoGobiernoCategoriaRPT, error) {
	return g.avanzar(ctx, o, ports.AccionConfirmarGobiernoCategoriaRPT, consultaConfirmarGobiernoRPT)
}

func (g *GestorGobiernoCategoriaRPTPostgreSQL) avanzar(ctx context.Context, o ports.OrdenAvanceGobiernoCategoriaRPT, accion, consulta string) (ports.ResultadoGobiernoCategoriaRPT, error) {
	material, err := g.materialAvance(o.Material, accion)
	if err != nil {
		return ports.ResultadoGobiernoCategoriaRPT{}, err
	}
	defer borrarPiezasRPT(material)
	revision, estado := o.Material.RevisionEsperada+1, domain.EstadoGobiernoCategoriaRPTUnaAprobacion
	if revision == 3 {
		estado = domain.EstadoGobiernoCategoriaRPTAprobada
	}
	if accion == ports.AccionConfirmarGobiernoCategoriaRPT {
		estado = domain.EstadoGobiernoCategoriaRPTConfirmada
	}
	return g.ejecutar(ctx, o.Solicitud, o.Autorizacion, material, o.Material.PropuestaRef, o.Material.HuellaSHA256,
		o.Material.ReciboRef, revision, estado, accion, consulta)
}

// El hash de la solicitud compromete vínculo completo, contexto, motivo y
// correlación. Se coteja con la decisión canónica exportada antes de BeginTx.
func (g *GestorGobiernoCategoriaRPTPostgreSQL) validarAutorizacion(s domain.SolicitudAutorizacionLigadaV3,
	a ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, accion, ref string) (string, error) {
	if a.ValidarEstructura() != nil {
		return "", ports.ErrGobiernoCategoriaRPTDenegado
	}
	d, err := s.Datos()
	if err != nil || d.Accion != accion || d.Finalidad != ports.FinalidadGobiernoCategoriaRPT ||
		d.Recurso.Referencia != ref || d.Recurso.ModuloID != g.descriptor.ModuloID ||
		d.Recurso.Tipo != ports.TipoRecursoGobiernoCategoriaRPT || len(d.Recurso.Ambitos) != 2 ||
		d.Recurso.Ambitos["catalogo_id"] != g.descriptor.CatalogoID || d.Recurso.Ambitos["modulo_id"] != g.descriptor.ModuloID ||
		len(d.Recurso.Atributos) != 1 || !huellaRPT.MatchString(d.Recurso.Atributos["material_sha256"]) || d.Recurso.Validar() != nil {
		return "", ports.ErrGobiernoCategoriaRPTDenegado
	}
	hRecurso, errRecurso := d.Recurso.HuellaContextoAutorizacionSHA256()
	motivo, errMotivo := domain.RepresentacionCanonicaMotivoAutorizacionV2(d.ReferenciaMotivo)
	r := a.ResumenCapacidad()
	canon := a.DecisionCanonica()
	v, errVinculo := d.VinculoAutenticacionActor.Datos()
	correlacion, errCorrelacion := d.Correlacion.ValorCanonico()
	hDecision := sha256.Sum256(canon)
	hMotivo := sha256.Sum256(motivo)
	ctxActor, errContexto := domain.RehidratarContextoActorVinculadoV2(a.ContextoActorCanonico())
	hContexto, errHuellaContexto := ctxActor.HuellaSHA256VinculadaV2()
	if err := decisionCoincideSolicitudGobiernoRPT(s, canon, r.DecisionRef()); err != nil {
		return "", err
	}
	proyeccion, errProyeccion := domain.ParsearMensajeAtestacionAutorizacionV3NoAutoritativo(a.PayloadVECAD3())
	refDecision, errRefDecision := proyeccion.DecisionRef()
	refCorrelacion, errRefCorrelacion := proyeccion.CorrelacionRef()
	hProyeccionDecision, errHProyeccionDecision := proyeccion.HuellaDecisionSHA256()
	hProyeccionMotivo, errHProyeccionMotivo := proyeccion.HuellaMotivoSHA256()
	refContexto, errRefContexto := proyeccion.ReferenciaContextoActor()
	hProyeccionContexto, errHProyeccionContexto := proyeccion.HuellaContextoActorSHA256()
	if errRecurso != nil || errMotivo != nil || errVinculo != nil || errCorrelacion != nil ||
		errContexto != nil || errHuellaContexto != nil ||
		errProyeccion != nil || errRefDecision != nil || errRefCorrelacion != nil ||
		errHProyeccionDecision != nil || errHProyeccionMotivo != nil || errRefContexto != nil || errHProyeccionContexto != nil ||
		refDecision != r.DecisionRef() || refCorrelacion != correlacion ||
		hProyeccionDecision != hex.EncodeToString(hDecision[:]) || hProyeccionMotivo != hex.EncodeToString(hMotivo[:]) ||
		refContexto != v.RegistroContextoRef || hProyeccionContexto != v.ContextoActorHuellaSHA256 ||
		!bytes.Equal(motivo, a.MotivoCanonico()) ||
		r.DecisionHuellaSHA256() != hex.EncodeToString(hDecision[:]) || r.MotivoHuellaSHA256() != hex.EncodeToString(hMotivo[:]) ||
		r.ContextoRef() != v.RegistroContextoRef || r.ContextoHuellaSHA256() != v.ContextoActorHuellaSHA256 ||
		hContexto != v.ContextoActorHuellaSHA256 || ctxActor.Principal.ID != v.PrincipalID ||
		ctxActor.PerfilActivoRef != v.PerfilActivoRef ||
		r.Operacion() != accion || r.AudienciaConsumo() != ports.AudienciaGobiernoCategoriaRPT ||
		r.EfectoRef() != ref || r.EfectoHuellaSHA256() != hRecurso ||
		a.PersonaVersion() != ctxActor.Instantanea.PersonaVersion || a.PerfilVersion() != ctxActor.Instantanea.PerfilVersion {
		return "", ports.ErrGobiernoCategoriaRPTDenegado
	}
	return d.Recurso.Atributos["material_sha256"], nil
}

func decisionCoincideSolicitudGobiernoRPT(s domain.SolicitudAutorizacionLigadaV3, canon []byte, ref string) error {
	d, err := s.Datos()
	if err != nil {
		return ports.ErrGobiernoCategoriaRPTDenegado
	}
	hSolicitud, err := domain.HuellaSHA256SolicitudAutorizacionV3(s)
	hRecurso, errRecurso := d.Recurso.HuellaContextoAutorizacionSHA256()
	hMotivo, errMotivo := domain.HuellaSHA256MotivoAutorizacionV2(d.ReferenciaMotivo)
	v, errVinculo := d.VinculoAutenticacionActor.Datos()
	correlacion, errCorrelacion := d.Correlacion.ValorCanonico()
	decision, errDecision := leerDecisionLigaduraUsoRPT(canon)
	if err != nil || errRecurso != nil || errMotivo != nil || errVinculo != nil ||
		errCorrelacion != nil || errDecision != nil ||
		decision.Esquema != domain.EsquemaHuellaDecisionAutorizacionV3 ||
		decision.SolicitudHuellaSHA256 != hSolicitud || decision.DecisionRef != ref ||
		decision.PrincipalID != v.PrincipalID || decision.PerfilActivoRef != v.PerfilActivoRef ||
		decision.CorrelacionRef != correlacion || decision.MotivoHuellaSHA256 != hMotivo ||
		decision.ContextoRecursoHuellaSHA256 != hRecurso {
		return ports.ErrGobiernoCategoriaRPTDenegado
	}
	return nil
}

func (g *GestorGobiernoCategoriaRPTPostgreSQL) ejecutar(ctx context.Context, s domain.SolicitudAutorizacionLigadaV3,
	a ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, material []byte, ref, huella, recibo string,
	revision int64, estado, accion, consulta string) (ports.ResultadoGobiernoCategoriaRPT, error) {
	var cero ports.ResultadoGobiernoCategoriaRPT
	if err := contextoGobiernoRPT(ctx); err != nil {
		return cero, err
	}
	esperada, err := g.validarAutorizacion(s, a, accion, ref)
	if err != nil {
		return cero, err
	}
	tx, err := g.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return cero, errorGobiernoRPT(ctx, err)
	}
	if valorNuloPostgreSQL(tx) {
		return cero, ports.ErrGobiernoCategoriaRPTNoDisponible
	}
	defer revertirUsoRPT(tx)
	if _, err := tx.Exec(ctx, configurarLecturaRPT); err != nil {
		return cero, errorGobiernoRPT(ctx, err)
	}
	var actual string
	if err := tx.QueryRow(ctx, consultaHuellaMaterialRPT, string(material)).Scan(&actual); err != nil {
		return cero, errorGobiernoRPT(ctx, err)
	}
	if !huellaRPT.MatchString(actual) || actual != esperada {
		return cero, ports.ErrGobiernoCategoriaRPTDenegado
	}
	piezas := [][]byte{a.CapacidadCanonica(), a.DecisionCanonica(), a.MotivoCanonico(), a.ContextoActorCanonico(),
		a.PayloadVECAD3(), a.SobreCOSESign1(), a.EvidenciaVerificacion(), a.RaizPublicaSPKI()}
	defer borrarPiezasRPT(piezas...)
	var respuesta []byte
	err = tx.QueryRow(ctx, consulta, string(material), piezas[0], piezas[1], piezas[2], piezas[3],
		strconv.FormatUint(a.PersonaVersion(), 10), strconv.FormatUint(a.PerfilVersion(), 10),
		piezas[4], piezas[5], piezas[6], piezas[7]).Scan(&respuesta)
	if err != nil {
		return cero, errorGobiernoRPT(ctx, err)
	}
	defer borrarPiezasRPT(respuesta)
	r, err := decodificarReciboGobiernoRPT(respuesta, a, ref, huella, recibo, revision, estado, accion)
	if err != nil {
		return cero, err
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if err := tx.Commit(ctx); err != nil {
		return cero, errorGobiernoRPT(ctx, err)
	}
	return r, nil
}

type reciboGobiernoRPTWire struct {
	DecisionRef         string          `json:"decision_ref"`
	EfectoRef           string          `json:"efecto_ref"`
	HuellaEfectoSHA256  string          `json:"huella_efecto_sha256"`
	ConsumoHuellaSHA256 string          `json:"consumo_huella_sha256"`
	AuditoriaRef        string          `json:"auditoria_ref"`
	ConsumidaEn         time.Time       `json:"consumida_en"`
	ConsumoNuevo        bool            `json:"consumo_nuevo"`
	ReciboRef           string          `json:"recibo_ref"`
	Gobierno            json.RawMessage `json:"gobierno"`
}

type estadoGobiernoRPTWire struct {
	PropuestaRef      string `json:"propuesta_ref"`
	HuellaSHA256      string `json:"huella_sha256"`
	Revision          int64  `json:"revision"`
	Estado            string `json:"estado"`
	ReciboRef         string `json:"recibo_ref"`
	Accion            string `json:"accion,omitempty"`
	Version           int    `json:"version,omitempty"`
	RevisionCategoria int64  `json:"revision_categoria,omitempty"`
}

func decodificarReciboGobiernoRPT(bruto []byte, a ports.ExportacionMaterialConsumoAutorizacionAtestadaV3,
	ref, huella, recibo string, revision int64, estado, accion string) (ports.ResultadoGobiernoCategoriaRPT, error) {
	var cero ports.ResultadoGobiernoCategoriaRPT
	var wire reciboGobiernoRPTWire
	if len(bruto) == 0 || len(bruto) > maximoRespuestaLecturaRPT || numeroClavesRPT(bruto) != 9 || decodificarRPT(bruto, &wire) != nil {
		return cero, ports.ErrGobiernoCategoriaRPTNoConfiable
	}
	var gobierno estadoGobiernoRPTWire
	claves := 5
	if accion == ports.AccionConfirmarGobiernoCategoriaRPT {
		claves = 8
	}
	if numeroClavesRPT(wire.Gobierno) != claves || decodificarRPT(wire.Gobierno, &gobierno) != nil {
		return cero, ports.ErrGobiernoCategoriaRPTNoConfiable
	}
	resumen := a.ResumenCapacidad()
	if wire.DecisionRef != resumen.DecisionRef() || wire.EfectoRef != ref ||
		wire.HuellaEfectoSHA256 != resumen.EfectoHuellaSHA256() ||
		!huellaRPT.MatchString(wire.ConsumoHuellaSHA256) || wire.AuditoriaRef == "" ||
		!wire.ConsumoNuevo || !instanteRPTValido(wire.ConsumidaEn) ||
		wire.ConsumidaEn.Before(resumen.EmitidaEn()) || !wire.ConsumidaEn.Before(resumen.ExpiraEn()) ||
		wire.ReciboRef != recibo || gobierno.PropuestaRef != ref || gobierno.HuellaSHA256 != huella ||
		gobierno.ReciboRef != recibo || gobierno.Revision != revision || gobierno.Estado != estado {
		return cero, ports.ErrGobiernoCategoriaRPTNoConfiable
	}
	if accion == ports.AccionConfirmarGobiernoCategoriaRPT {
		if (gobierno.Accion != domain.AccionGobiernoCategoriaRPTPublicar && gobierno.Accion != domain.AccionGobiernoCategoriaRPTDeshabilitar) ||
			gobierno.Version < 1 || gobierno.Version > maximoVersionMaterialRPT || gobierno.RevisionCategoria < 1 {
			return cero, ports.ErrGobiernoCategoriaRPTNoConfiable
		}
	}
	return ports.ResultadoGobiernoCategoriaRPT{PropuestaRef: ref, HuellaSHA256: huella, Revision: revision,
		Estado: estado, ReciboRef: recibo, Accion: gobierno.Accion, Version: gobierno.Version,
		RevisionCategoria: gobierno.RevisionCategoria,
		Evidencia: ports.EvidenciaGobiernoCategoriaRPT{DecisionRef: wire.DecisionRef, EfectoRef: wire.EfectoRef,
			HuellaEfectoSHA256: wire.HuellaEfectoSHA256, ConsumoHuellaSHA256: wire.ConsumoHuellaSHA256,
			AuditoriaRef: wire.AuditoriaRef, ConsumidaEn: wire.ConsumidaEn.UTC(), ConsumoNuevo: true}}, nil
}

func errorGobiernoRPT(ctx context.Context, causa error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(causa, context.Canceled) || errors.Is(causa, context.DeadlineExceeded) {
		return causa
	}
	var pg *pgconn.PgError
	if errors.As(causa, &pg) {
		switch pg.Code {
		case "42501", "PDI03":
			return ports.ErrGobiernoCategoriaRPTDenegado
		case "22023":
			return ports.ErrGobiernoCategoriaRPTInvalido
		case "23505", "40001", "55P03", "55000":
			return ports.ErrGobiernoCategoriaRPTConflicto
		}
	}
	return ports.ErrGobiernoCategoriaRPTNoDisponible
}
