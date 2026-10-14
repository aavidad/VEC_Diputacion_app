package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"

	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const (
	sqlReservarSubidaLocal           = "SELECT vec_documentos.reservar_subida_local_v1($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)"
	sqlConfirmarRecepcionSubidaLocal = "SELECT vec_documentos.confirmar_recepcion_subida_local_v1($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)"
	sqlIniciarAnalisisSubidaLocal    = "SELECT vec_documentos.iniciar_analisis_subida_local_v1($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)"
	sqlRegistrarAnalisisSubidaLocal  = "SELECT vec_documentos.registrar_analisis_subida_local_v1($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)"
	sqlCerrarAnalisisSubidaLocal     = "SELECT vec_documentos.cerrar_analisis_subida_local_v1($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)"
	sqlConsultarSubidaLocal          = "SELECT vec_documentos.consultar_subida_local_propia_v1($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)"
)

type RepositorioSubidaLocal struct{ base *Repositorio }

// El pool debe usar exclusivamente LOGIN vec_externo_documentos_datos. D20
// revalida session_user y su único grupo en cada fachada; el constructor no
// toma ni deriva credenciales del repositorio D17 de Méritos/Mayan.
func NuevoRepositorioSubidaLocalConPool(db *pgxpool.Pool) (*RepositorioSubidaLocal,error) {
	if db==nil { return nil,ErrRepositorioNoDisponible }
	return &RepositorioSubidaLocal{base:&Repositorio{db:db}},nil
}

// El hecho aún no existe al subir el PDF. El acto reservado corresponde a
// persona y clave de declaración certificadas; D20 revalida titular y versión
// del contexto en la misma transacción de reserva.
type ResolutorActoSubidaLocalMeritos struct{ base *Repositorio }

func NuevoResolutorActoSubidaLocalMeritosConPool(db *pgxpool.Pool) (*ResolutorActoSubidaLocalMeritos,error) {
	if db==nil { return nil,ErrRepositorioNoDisponible }
	return &ResolutorActoSubidaLocalMeritos{base:&Repositorio{db:db}},nil
}

func NuevoResolutorActoSubidaLocalMeritos(base *Repositorio) (*ResolutorActoSubidaLocalMeritos, error) {
	if base == nil || base.db == nil {
		return nil, ErrRepositorioNoDisponible
	}
	return &ResolutorActoSubidaLocalMeritos{base: base}, nil
}

func (*ResolutorActoSubidaLocalMeritos) ResolverActoSubidaLocal(ctx context.Context,
	q docports.SolicitudSubidaLocal) (docports.ActoSubidaLocal, error) {
	if ctx == nil || ctx.Err() != nil || q.ClaveIdempotencia == "" || len(q.ClaveIdempotencia) > 256 ||
		q.ActorRegistradoV2.Validar() != nil || q.VinculoV2.ValidarPara(q.ActorRegistradoV2) != nil {
		return docports.ActoSubidaLocal{}, docports.ErrSubidaLocalDenegada
	}
	persona := q.ActorRegistradoV2.Contexto.PersonaRef
	suma := sha256.Sum256([]byte("vec.meritos.cartera.declaracion.v1:" + persona + ":" + q.ClaveIdempotencia))
	ref := "ref:" + hex.EncodeToString(suma[:])
	recurso := vecdomain.RecursoAutorizable{Referencia: ref, ModuloID: "meritos", Tipo: "hecho",
		Ambitos: map[string]string{"persona_ref": persona}}
	if recurso.Validar() != nil {
		return docports.ActoSubidaLocal{}, docports.ErrSubidaLocalDenegada
	}
	return docports.ActoSubidaLocal{Recurso: recurso, PersonaRef: persona,
		Finalidad: "custodiar_evidencia_meritos", Clasificacion: "confidencial",
		VersionDueno: q.ActorRegistradoV2.Contexto.Instantanea.PersonaVersion}, nil
}

var _ docports.ResolutorActoSubidaLocal = (*ResolutorActoSubidaLocalMeritos)(nil)

func NuevoRepositorioSubidaLocal(base *Repositorio) (*RepositorioSubidaLocal, error) {
	if base == nil || base.db == nil {
		return nil, ErrRepositorioNoDisponible
	}
	return &RepositorioSubidaLocal{base: base}, nil
}

func materialSubidaLocal(a docports.AutorizacionActoSubidaLocalV3) (materialV3, error) {
	m := a.Material
	if m.ValidarEstructura() != nil || m.PersonaVersion() == 0 || m.PerfilVersion() == 0 ||
		m.PersonaVersion() > 9007199254740991 || m.PerfilVersion() > 9007199254740991 {
		return materialV3{}, docports.ErrSubidaLocalDenegada
	}
	return materialV3{capacidad: m.CapacidadCanonica(), decision: m.DecisionCanonica(),
		motivo: m.MotivoCanonico(), contexto: m.ContextoActorCanonico(),
		persona: int64(m.PersonaVersion()), perfil: int64(m.PerfilVersion()),
		payload: m.PayloadVECAD3(), sobre: m.SobreCOSESign1(), evidencia: m.EvidenciaVerificacion(),
		raiz: m.RaizPublicaSPKI()}, nil
}

func authSubidaLocalJSON(a docports.AutorizacionActoSubidaLocalV3,
	actorPersona, perfil, ambito string) ([]byte, error) {
	if a.Accion == "" || a.CargaID == "" || actorPersona == "" || perfil == "" || ambito == "" ||
		a.Finalidad == "" || a.CorrelacionRef == "" {
		return nil, docports.ErrSubidaLocalDenegada
	}
	return json.Marshal(struct {
		Accion          string `json:"accion"`
		RecursoRef      string `json:"recurso_ref"`
		AmbitoRef       string `json:"ambito_ref"`
		PrincipalID     string `json:"principal_id"`
		PerfilActivoRef string `json:"perfil_activo_ref"`
		Finalidad       string `json:"finalidad"`
		CorrelacionRef  string `json:"correlacion_ref"`
	}{a.Accion, a.CargaID, ambito, actorPersona, perfil, a.Finalidad, a.CorrelacionRef})
}

func (r *RepositorioSubidaLocal) ejecutar(ctx context.Context, sql string, pre []byte,
	a docports.AutorizacionActoSubidaLocalV3, persona, perfil, ambito string) (docports.InstantaneaSubidaLocal, error) {
	var vacia docports.InstantaneaSubidaLocal
	if r == nil || r.base == nil || ctx == nil || ctx.Err() != nil || len(pre) == 0 || len(pre) > 65536 {
		return vacia, docports.ErrSubidaLocalNoDisponible
	}
	auth, err := authSubidaLocalJSON(a, persona, perfil, ambito)
	if err != nil {
		return vacia, err
	}
	m, err := materialSubidaLocal(a)
	if err != nil {
		return vacia, err
	}
	raw, err := r.base.transaccion(ctx, sql, pre, string(auth), m.capacidad, m.decision, m.motivo,
		m.contexto, m.persona, m.perfil, m.payload, m.sobre, m.evidencia, m.raiz)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return vacia, err
		}
		return vacia, err
	}
	var i docports.InstantaneaSubidaLocal
	if json.Unmarshal(raw, &i) != nil || !instantaneaSubidaLocalValida(i) || i.PersonaRef != persona ||
		i.Resultado.CargaID == "" || (a.CargaID != i.Resultado.CargaID && a.CargaID != i.Resultado.ComprobanteRef) {
		return vacia, docports.ErrSubidaLocalNoDisponible
	}
	i.Resultado.RecursoRef = i.RecursoRef
	return i, nil
}

func instantaneaSubidaLocalValida(i docports.InstantaneaSubidaLocal) bool {
	if !i.Resultado.Estado.Valido() || i.Resultado.CargaID == "" || i.RecursoRef == "" ||
		i.PersonaRef == "" || i.CorrelacionRef == "" || i.Clasificacion == "" ||
		i.HuellaSolicitudHMAC == "" || i.HuellaPDFSHA256 == "" || i.MIME != "application/pdf" ||
		i.Tamano < 1 || i.Tamano > 16<<20 || i.Version == 0 {
		return false
	}
	switch i.Resultado.Estado {
	case docports.EstadoSubidaLocalReservada:
		return i.Version == 1 && i.Resultado.ReciboRef == "" && i.Cuarentena == nil &&
			i.Analisis == nil && i.Admitido == nil && i.Resultado.ComprobanteRef == ""
	case docports.EstadoSubidaLocalRecibida, docports.EstadoSubidaLocalAnalizando:
		return (i.Version == 2 && i.Resultado.Estado == docports.EstadoSubidaLocalRecibida ||
			(i.Version == 3 || i.Version == 4) && i.Resultado.Estado == docports.EstadoSubidaLocalAnalizando) &&
			i.Resultado.ReciboRef != "" && i.Cuarentena != nil && i.Cuarentena.Validar() == nil &&
			i.Cuarentena.Objeto.Zona == vecports.ZonaAlmacenCuarentena &&
			(i.Version != 4 && i.Analisis == nil || i.Version == 4 && i.Analisis != nil && i.Analisis.Validar() == nil) &&
			i.Admitido == nil &&
			i.Resultado.ComprobanteRef == ""
	case docports.EstadoSubidaLocalAdmitida:
		return i.Version == 5 && i.Resultado.ReciboRef != "" && i.Resultado.ComprobanteRef != "" &&
			i.Resultado.DocumentoVersion > 0 && !i.Resultado.RegistradoEn.IsZero() &&
			i.Resultado.CodigoRechazo == "" && i.Cuarentena != nil && i.Analisis != nil &&
			i.Analisis.Estado == vecports.EstadoAnalisisContenidoLimpio && i.Admitido != nil &&
			i.Admitido.Validar() == nil && i.Admitido.Objeto.Zona == vecports.ZonaAlmacenAdmitida &&
			i.Admitido.Objeto.HuellaSHA256 == i.HuellaPDFSHA256
	case docports.EstadoSubidaLocalRechazada:
		return i.Version == 5 && i.Resultado.ReciboRef != "" && i.Resultado.ComprobanteRef == "" &&
			i.Resultado.DocumentoVersion == 0 && i.Resultado.RegistradoEn.IsZero() &&
			(i.Resultado.CodigoRechazo == "pdf_no_valido" || i.Resultado.CodigoRechazo == "contenido_no_permitido" ||
				i.Resultado.CodigoRechazo == "analisis_no_concluyente") && i.Cuarentena != nil &&
			i.Analisis != nil && i.Admitido == nil
	default:
		return false
	}
}

func (r *RepositorioSubidaLocal) Reservar(ctx context.Context, q docports.ReservaSubidaLocal) (docports.InstantaneaSubidaLocal, error) {
	pre, err := docports.PreimagenReservaSubidaLocal(q)
	if err != nil || q.AutorizacionV3.Accion != q.AccionSubir || q.AutorizacionV3.CargaID != q.CargaID ||
		q.AutorizacionV3.RecursoRef != q.Acto.Recurso.Referencia ||
		q.AutorizacionV3.CorrelacionRef != q.CorrelacionRef {
		return docports.InstantaneaSubidaLocal{}, docports.ErrSubidaLocalDenegada
	}
	i, err := r.ejecutar(ctx, sqlReservarSubidaLocal, pre, q.AutorizacionV3, q.ActorRegistradoV2.Contexto.PersonaRef,
		q.ActorRegistradoV2.Contexto.PerfilActivoRef, q.Acto.Recurso.Referencia)
	if err != nil {
		return i, err
	}
	if i.Resultado.CargaID != q.CargaID || i.RecursoRef != q.Acto.Recurso.Referencia ||
		i.HuellaPDFSHA256 != q.HuellaPDFSHA256 || i.Tamano != q.Tamano ||
		i.HuellaSolicitudHMAC != q.HuellaSolicitudHMAC {
		return docports.InstantaneaSubidaLocal{}, docports.ErrSubidaLocalDenegada
	}
	return i, nil
}

func (r *RepositorioSubidaLocal) movimiento(ctx context.Context, q docports.MovimientoSubidaLocal,
	fase docports.FasePersistenciaSubidaLocal, sql string) (docports.InstantaneaSubidaLocal, error) {
	if q.Fase != fase || q.AutorizacionV3.Accion != q.Accion || q.AutorizacionV3.CargaID != q.CargaID ||
		q.AutorizacionV3.RecursoRef != q.RecursoRef || q.AutorizacionV3.CorrelacionRef != q.CorrelacionRef {
		return docports.InstantaneaSubidaLocal{}, docports.ErrSubidaLocalDenegada
	}
	pre, err := docports.PreimagenMovimientoSubidaLocal(q)
	if err != nil {
		return docports.InstantaneaSubidaLocal{}, err
	}
	i, err := r.ejecutar(ctx, sql, pre, q.AutorizacionV3, q.ActorRegistradoV2.Contexto.PersonaRef,
		q.ActorRegistradoV2.Contexto.PerfilActivoRef, q.RecursoRef)
	if err != nil {
		return i, err
	}
	if i.Resultado.CargaID != q.CargaID || i.RecursoRef != q.RecursoRef ||
		i.Version < q.VersionEsperada {
		return docports.InstantaneaSubidaLocal{}, docports.ErrSubidaLocalDenegada
	}
	return i, nil
}

func (r *RepositorioSubidaLocal) ConfirmarRecepcion(ctx context.Context, q docports.MovimientoSubidaLocal) (docports.InstantaneaSubidaLocal, error) {
	return r.movimiento(ctx, q, docports.FaseSubidaLocalConfirmarRecepcion, sqlConfirmarRecepcionSubidaLocal)
}
func (r *RepositorioSubidaLocal) IniciarAnalisis(ctx context.Context, q docports.MovimientoSubidaLocal) (docports.InstantaneaSubidaLocal, error) {
	return r.movimiento(ctx, q, docports.FaseSubidaLocalIniciarAnalisis, sqlIniciarAnalisisSubidaLocal)
}
func (r *RepositorioSubidaLocal) RegistrarAnalisis(ctx context.Context, q docports.MovimientoSubidaLocal) (docports.InstantaneaSubidaLocal, error) {
	return r.movimiento(ctx, q, docports.FaseSubidaLocalRegistrarAnalisis, sqlRegistrarAnalisisSubidaLocal)
}
func (r *RepositorioSubidaLocal) CerrarAnalisis(ctx context.Context, q docports.MovimientoSubidaLocal) (docports.InstantaneaSubidaLocal, error) {
	return r.movimiento(ctx, q, docports.FaseSubidaLocalCerrarAnalisis, sqlCerrarAnalisisSubidaLocal)
}
func (r *RepositorioSubidaLocal) ConsultarPropia(ctx context.Context, q docports.ConsultaPersistenciaSubidaLocal) (docports.InstantaneaSubidaLocal, error) {
	pre, err := docports.PreimagenConsultaSubidaLocal(q)
	if err != nil {
		return docports.InstantaneaSubidaLocal{}, err
	}
	id := q.CargaID
	if id == "" {
		id = q.DocumentoID
	}
	ambito := q.RecursoRef
	if q.Fase == docports.FaseSubidaLocalConsultarCarga || q.Fase == docports.FaseSubidaLocalConsultarDocumento {
		ambito = id
	}
	if ambito == "" || q.AutorizacionV3.CargaID != id || q.AutorizacionV3.Accion != q.Accion ||
		q.AutorizacionV3.CorrelacionRef != q.CorrelacionRef {
		return docports.InstantaneaSubidaLocal{}, docports.ErrSubidaLocalDenegada
	}
	i, err := r.ejecutar(ctx, sqlConsultarSubidaLocal, pre, q.AutorizacionV3,
		q.ActorRegistradoV2.Contexto.PersonaRef, q.ActorRegistradoV2.Contexto.PerfilActivoRef, ambito)
	if err != nil {
		return i, err
	}
	if q.Fase == docports.FaseSubidaLocalAbrir && i.RecursoRef != q.RecursoRef {
		return docports.InstantaneaSubidaLocal{}, docports.ErrSubidaLocalDenegada
	}
	return i, nil
}

var _ docports.RegistroSubidaLocal = (*RepositorioSubidaLocal)(nil)
