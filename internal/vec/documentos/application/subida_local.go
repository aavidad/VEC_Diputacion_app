package application

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strings"

	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const limitePDFSubidaLocal = 16 << 20

// La clave estable del servidor permite recuperar una carga por persona y
// clave de declaración tras renovar sesión, sin aceptar correlación del HTTP.
type DerivadorCorrelacionSubidaLocalHMAC struct{ clave [32]byte }

func NuevoDerivadorCorrelacionSubidaLocalHMAC(clave []byte) (*DerivadorCorrelacionSubidaLocalHMAC, error) {
	if len(clave) != 32 {
		return nil, docports.ErrSubidaLocalNoDisponible
	}
	d := &DerivadorCorrelacionSubidaLocalHMAC{}
	copy(d.clave[:], clave)
	return d, nil
}

func (d *DerivadorCorrelacionSubidaLocalHMAC) CorrelacionSubida(ctx context.Context, persona, clave string) (string, error) {
	if d == nil || ctx == nil || ctx.Err() != nil || persona == "" || clave == "" || len(clave) > 256 {
		return "", docports.ErrSubidaLocalNoDisponible
	}
	mac := hmac.New(sha256.New, d.clave[:])
	_, _ = mac.Write([]byte("vec.documentos.subida.local.correlacion.v1\x00"))
	var longitud [8]byte
	binary.BigEndian.PutUint64(longitud[:], uint64(len(persona)))
	_, _ = mac.Write(longitud[:])
	_, _ = mac.Write([]byte(persona))
	binary.BigEndian.PutUint64(longitud[:], uint64(len(clave)))
	_, _ = mac.Write(longitud[:])
	_, _ = mac.Write([]byte(clave))
	suma := mac.Sum(nil)
	return "correlacion_" + hex.EncodeToString(suma[:16]), nil
}

var _ docports.DerivadorCorrelacionSubidaLocal = (*DerivadorCorrelacionSubidaLocalHMAC)(nil)

type DependenciasServicioSubidaLocal struct {
	Resolutor         docports.ResolutorActoSubidaLocal
	Autoridad         docports.AutoridadActoSubidaLocalV3
	Registro          docports.RegistroSubidaLocal
	Almacen           vecports.AlmacenObjetos
	Analizador        vecports.AnalizadorContenido
	Seudonimizador    vecports.SeudonimizadorSujetoAlmacen
	SelladorSolicitud vecports.SelladorSolicitudCargaDocumental
	Correlaciones     docports.DerivadorCorrelacionSubidaLocal
	Reloj             vecports.Reloj
}

type ConfiguracionServicioSubidaLocal struct {
	ModuloID, TipoRecurso, Finalidad, Clasificacion string
	AccionSubir, AccionDescargar                    string
}

type ServicioSubidaLocal struct {
	d DependenciasServicioSubidaLocal
	c ConfiguracionServicioSubidaLocal
}

func NuevoServicioSubidaLocal(c ConfiguracionServicioSubidaLocal,
	d DependenciasServicioSubidaLocal) (*ServicioSubidaLocal, error) {
	if !docports.ParejaActoSubidaLocalValida(c.ModuloID, c.TipoRecurso, c.Finalidad, c.AccionSubir, c.AccionDescargar) ||
		c.Clasificacion == "" || dependenciaNula(d.Resolutor) || dependenciaNula(d.Autoridad) ||
		dependenciaNula(d.Registro) || dependenciaNula(d.Almacen) || dependenciaNula(d.Analizador) ||
		dependenciaNula(d.Seudonimizador) || dependenciaNula(d.SelladorSolicitud) ||
		dependenciaNula(d.Correlaciones) || dependenciaNula(d.Reloj) {
		return nil, docports.ErrSubidaLocalNoDisponible
	}
	if _, ok := d.Almacen.(vecports.RecuperadorOperacionSubidaLocal); !ok {
		return nil, docports.ErrSubidaLocalNoDisponible
	}
	return &ServicioSubidaLocal{d: d, c: c}, nil
}

func (s *ServicioSubidaLocal) verificarActor(q docports.SolicitudSubidaLocal) (vecdomain.ResultadoContextoActorRegistradoV2, error) {
	if s == nil || q.ClaveIdempotencia == "" || len(q.ClaveIdempotencia) > 256 {
		return vecdomain.ResultadoContextoActorRegistradoV2{}, docports.ErrSubidaLocalDenegada
	}
	a, err := q.ActorRegistradoV2.Clonar()
	if err != nil || q.VinculoV2.ValidarPara(a) != nil {
		return vecdomain.ResultadoContextoActorRegistradoV2{}, docports.ErrSubidaLocalDenegada
	}
	return a, nil
}

func (s *ServicioSubidaLocal) acto(ctx context.Context, q docports.SolicitudSubidaLocal, a vecdomain.ResultadoContextoActorRegistradoV2) (docports.ActoSubidaLocal, error) {
	acto, err := s.d.Resolutor.ResolverActoSubidaLocal(ctx, q)
	if err != nil || acto.Recurso.Validar() != nil || acto.Recurso.ModuloID != s.c.ModuloID ||
		acto.Recurso.Tipo != s.c.TipoRecurso || acto.Finalidad != s.c.Finalidad ||
		acto.Clasificacion != s.c.Clasificacion || acto.PersonaRef != a.Contexto.PersonaRef ||
		acto.VersionDueno == 0 {
		return docports.ActoSubidaLocal{}, docports.ErrSubidaLocalDenegada
	}
	return acto, nil
}

func (s *ServicioSubidaLocal) referencias(ctx context.Context, a vecdomain.ResultadoContextoActorRegistradoV2, clave string) (string, string, string, error) {
	corr, err := s.d.Correlaciones.CorrelacionSubida(ctx, a.Contexto.PersonaRef, clave)
	if err != nil || !vecdomain.ReferenciaCorrelacionAutorizacionV2Valida(corr) {
		return "", "", "", docports.ErrSubidaLocalNoDisponible
	}
	idSum := sha256.Sum256([]byte("vec.documentos.subida.local.id.v1\x00" + corr))
	claveSum := sha256.Sum256([]byte("vec.documentos.subida.local.clave.v1\x00" + corr))
	return corr, "ref:" + hex.EncodeToString(idSum[:]), "hmac-sha256:local_v1:" + hex.EncodeToString(claveSum[:]), nil
}

func (s *ServicioSubidaLocal) Subir(ctx context.Context, q docports.SolicitudSubidaLocal,
	contenido []byte) (docports.ResultadoSubidaLocal, error) {
	if s == nil || ctx == nil || ctx.Err() != nil || len(contenido) == 0 || len(contenido) > limitePDFSubidaLocal ||
		!bytes.HasPrefix(contenido, []byte("%PDF-")) {
		return docports.ResultadoSubidaLocal{}, docports.ErrSubidaLocalDenegada
	}
	actor, err := s.verificarActor(q)
	if err != nil {
		return docports.ResultadoSubidaLocal{}, err
	}
	q.ActorRegistradoV2 = actor
	acto, err := s.acto(ctx, q, actor)
	if err != nil {
		return docports.ResultadoSubidaLocal{}, err
	}
	corr, id, claveHMAC, err := s.referencias(ctx, actor, q.ClaveIdempotencia)
	if err != nil {
		return docports.ResultadoSubidaLocal{}, err
	}
	suma := sha256.Sum256(contenido)
	huella := hex.EncodeToString(suma[:])
	canon, err := json.Marshal(struct {
		Modulo, Recurso, Persona, Finalidad, MIME, Huella string
		Tamano                                            int64
	}{s.c.ModuloID, acto.Recurso.Referencia, actor.Contexto.PersonaRef, s.c.Finalidad, "application/pdf", huella, int64(len(contenido))})
	if err != nil {
		return docports.ResultadoSubidaLocal{}, docports.ErrSubidaLocalNoDisponible
	}
	huellaSolicitud, err := s.d.SelladorSolicitud.SellarSolicitudCargaDocumental(ctx, canon)
	if err != nil || !strings.HasPrefix(huellaSolicitud, "hmac-sha256:") {
		return docports.ResultadoSubidaLocal{}, docports.ErrSubidaLocalNoDisponible
	}
	reserva := docports.ReservaSubidaLocal{ActorRegistradoV2: actor, VinculoV2: q.VinculoV2,
		Acto: acto, AccionSubir: s.c.AccionSubir, AccionDescargar: s.c.AccionDescargar,
		CargaID: id, CorrelacionRef: corr, ClaveHMAC: claveHMAC, HuellaSolicitudHMAC: huellaSolicitud,
		HuellaPDFSHA256: huella, Tamano: int64(len(contenido)), MIME: "application/pdf"}
	pre, err := docports.PreimagenReservaSubidaLocal(reserva)
	if err != nil {
		return docports.ResultadoSubidaLocal{}, err
	}
	reserva.AutorizacionV3, err = s.d.Autoridad.AutorizarSubidaLocal(ctx, pre, actor, id, acto.Recurso.Referencia, acto.Finalidad, corr)
	if err != nil {
		return docports.ResultadoSubidaLocal{}, err
	}
	i, err := s.d.Registro.Reservar(ctx, reserva)
	if err != nil {
		return docports.ResultadoSubidaLocal{}, err
	}
	if !s.coincide(i, id, acto.Recurso.Referencia, actor.Contexto.PersonaRef, huella, int64(len(contenido))) {
		return docports.ResultadoSubidaLocal{}, docports.ErrSubidaLocalDenegada
	}
	return s.continuar(ctx, reserva, i, contenido)
}

func (s *ServicioSubidaLocal) coincide(i docports.InstantaneaSubidaLocal, id, recurso, persona, huella string, tamano int64) bool {
	return i.Resultado.CargaID == id && i.RecursoRef == recurso && i.PersonaRef == persona &&
		i.HuellaPDFSHA256 == huella && i.MIME == "application/pdf" && i.Tamano == tamano &&
		i.Resultado.Estado.Valido() && i.Version > 0
}

func (s *ServicioSubidaLocal) seudonimo(ctx context.Context, persona, id string) (string, error) {
	sol, err := vecports.NuevaSolicitudSeudonimizarSujetoAlmacen(persona, "subida_local:"+id)
	if err != nil {
		return "", docports.ErrSubidaLocalNoDisponible
	}
	valor, err := s.d.Seudonimizador.SeudonimizarSujetoAlmacen(ctx, sol)
	if err != nil || !strings.HasPrefix(valor, "hmac-sha256:") {
		return "", docports.ErrSubidaLocalNoDisponible
	}
	return valor, nil
}

func efectoSubidaLocal(id string, fase vecports.FaseAlmacenSubidaLocal, obj vecports.ReferenciaObjetoAlmacen) string {
	suma := sha256.Sum256([]byte("vec.documentos.subida.local.efecto.v1\x00" + id + "\x00" + string(fase) + "\x00" + obj.Referencia + "\x00" + obj.Version))
	return "efecto-carga:sha256:" + hex.EncodeToString(suma[:])
}

func (s *ServicioSubidaLocal) vinculos(ctx context.Context, i docports.InstantaneaSubidaLocal,
	fase vecports.FaseAlmacenSubidaLocal, obj vecports.ReferenciaObjetoAlmacen) (vecports.VinculosOperacionAlmacen, error) {
	seudonimo, err := s.seudonimo(ctx, i.PersonaRef, i.Resultado.CargaID)
	if err != nil {
		return vecports.VinculosOperacionAlmacen{}, err
	}
	return vecports.VinculosOperacionAlmacen{OperacionRef: i.Resultado.CargaID, CargaRef: i.Resultado.CargaID,
		Clasificacion: i.Clasificacion, SujetoSeudonimoHMAC: seudonimo,
		HuellaSolicitudHMAC: i.HuellaSolicitudHMAC, EfectoRef: efectoSubidaLocal(i.Resultado.CargaID, fase, obj),
		ObjetoVinculado: obj}, nil
}

func (s *ServicioSubidaLocal) recuperarOperacion(ctx context.Context, r docports.ReservaSubidaLocal,
	i docports.InstantaneaSubidaLocal, fase vecports.FaseAlmacenSubidaLocal,
	origen vecports.ReferenciaObjetoAlmacen, evidenciaAnalisis string) (vecports.ResultadoOperacionObjeto, bool, error) {
	var vacio vecports.ResultadoOperacionObjeto
	lector, ok := s.d.Almacen.(vecports.RecuperadorOperacionSubidaLocal)
	if !ok {
		return vacio, false, docports.ErrSubidaLocalNoDisponible
	}
	clave := r.CargaID + ":escribir"
	if fase == vecports.FaseAlmacenSubidaLocalRecuperarPromocion {
		clave = r.CargaID + ":promover"
	}
	sol := vecports.SolicitudRecuperarOperacionSubidaLocal{Fase: fase, ClaveIdempotencia: clave,
		MIME: "application/pdf", Tamano: i.Tamano, HuellaSHA256: i.HuellaPDFSHA256,
		Origen: origen, EvidenciaAnalisisRef: evidenciaAnalisis}
	efecto, err := vecports.HuellaEfectoRecuperacionSubidaLocal(sol, r.CargaID)
	if err != nil {
		return vacio, false, err
	}
	v, err := s.vinculos(ctx, i, fase, vecports.ReferenciaObjetoAlmacen{})
	if err != nil {
		return vacio, false, err
	}
	v.EfectoRef = efecto
	contexto, err := s.d.Autoridad.AutorizarAlmacenSubidaLocal(ctx, r.ActorRegistradoV2, r.Acto,
		s.c.AccionSubir, r.CorrelacionRef, fase, v)
	if err != nil {
		return vacio, false, err
	}
	sol.Contexto = contexto
	if sol.Validar() != nil {
		return vacio, false, docports.ErrSubidaLocalDenegada
	}
	return lector.RecuperarOperacionSubidaLocal(ctx, sol)
}

func (s *ServicioSubidaLocal) movimiento(ctx context.Context, r docports.ReservaSubidaLocal,
	i docports.InstantaneaSubidaLocal, fase docports.FasePersistenciaSubidaLocal,
	cuarentena *vecports.ResultadoOperacionObjeto, analisis *vecports.ResultadoAnalisisContenido,
	promocion *vecports.ResultadoOperacionObjeto, final docports.EstadoSubidaLocal, codigo string,
) (docports.MovimientoSubidaLocal, error) {
	m := docports.MovimientoSubidaLocal{ActorRegistradoV2: r.ActorRegistradoV2, VinculoV2: r.VinculoV2,
		Accion: s.c.AccionSubir, Fase: fase, CargaID: r.CargaID, RecursoRef: r.Acto.Recurso.Referencia,
		Finalidad: r.Acto.Finalidad, CorrelacionRef: r.CorrelacionRef, VersionEsperada: i.Version,
		HuellaPDFSHA256: r.HuellaPDFSHA256, ReciboRef: i.Resultado.ReciboRef,
		EstadoFinal: final, CodigoRechazo: codigo, Cuarentena: cuarentena, Analisis: analisis, Promocion: promocion}
	pre, err := docports.PreimagenMovimientoSubidaLocal(m)
	if err != nil {
		return m, err
	}
	m.AutorizacionV3, err = s.d.Autoridad.AutorizarSubidaLocal(ctx, pre, r.ActorRegistradoV2,
		r.CargaID, r.Acto.Recurso.Referencia, r.Acto.Finalidad, r.CorrelacionRef)
	return m, err
}

func (s *ServicioSubidaLocal) continuar(ctx context.Context, r docports.ReservaSubidaLocal,
	i docports.InstantaneaSubidaLocal, contenido []byte) (docports.ResultadoSubidaLocal, error) {
	switch i.Resultado.Estado {
	case docports.EstadoSubidaLocalAdmitida:
		if i.Resultado.ComprobanteRef == "" || i.Resultado.DocumentoVersion < 1 || i.Resultado.RegistradoEn.IsZero() {
			return docports.ResultadoSubidaLocal{}, docports.ErrSubidaLocalPendiente
		}
		i.Resultado.RecursoRef = i.RecursoRef
		return i.Resultado, nil
	case docports.EstadoSubidaLocalRechazada:
		return i.Resultado, docports.ErrSubidaLocalRechazada
	case docports.EstadoSubidaLocalReservada:
		guardado, encontrado, err := s.recuperarOperacion(ctx, r, i,
			vecports.FaseAlmacenSubidaLocalRecuperarEscritura, vecports.ReferenciaObjetoAlmacen{}, "")
		if err != nil {
			return i.Resultado, errors.Join(docports.ErrSubidaLocalPendiente, err)
		}
		if !encontrado {
			v, err := s.vinculos(ctx, i, vecports.FaseAlmacenSubidaLocalEscribir, vecports.ReferenciaObjetoAlmacen{})
			if err != nil {
				return i.Resultado, err
			}
			contexto, err := s.d.Autoridad.AutorizarAlmacenSubidaLocal(ctx, r.ActorRegistradoV2, r.Acto,
				s.c.AccionSubir, r.CorrelacionRef, vecports.FaseAlmacenSubidaLocalEscribir, v)
			if err != nil {
				return i.Resultado, err
			}
			caps, err := s.d.Almacen.Capacidades(ctx)
			if err != nil || !caps.EscrituraEnFlujo || !caps.IntegridadSHA256 || caps.TamanoMaximoObjeto < int64(len(contenido)) {
				return i.Resultado, docports.ErrSubidaLocalNoDisponible
			}
			sol := vecports.SolicitudEscribirObjeto{Contexto: contexto, ClaveIdempotencia: r.CargaID + ":escribir",
				Zona: vecports.ZonaAlmacenCuarentena, MIME: "application/pdf", Tamano: int64(len(contenido)),
				HuellaSHA256: r.HuellaPDFSHA256, Contenido: bytes.NewReader(contenido)}
			if sol.Validar() != nil || contexto.ValidarParaEn(vecports.AccionAlmacenEscribir, s.d.Reloj.Ahora().UTC()) != nil {
				return i.Resultado, docports.ErrSubidaLocalDenegada
			}
			guardado, err = s.d.Almacen.Escribir(ctx, sol)
			if err != nil {
				return i.Resultado, errors.Join(docports.ErrSubidaLocalPendiente, err)
			}
			if guardado.ValidarEscritura(sol, caps) != nil {
				return i.Resultado, docports.ErrSubidaLocalPendiente
			}
		}
		if guardado.Validar() != nil || guardado.Objeto.HuellaSHA256 != r.HuellaPDFSHA256 ||
			guardado.Objeto.Tamano != r.Tamano || guardado.Objeto.Zona != vecports.ZonaAlmacenCuarentena {
			return i.Resultado, docports.ErrSubidaLocalPendiente
		}
		m, err := s.movimiento(ctx, r, i, docports.FaseSubidaLocalConfirmarRecepcion, &guardado, nil, nil, "", "")
		if err != nil {
			return i.Resultado, err
		}
		i, err = s.d.Registro.ConfirmarRecepcion(ctx, m)
		if err != nil {
			return docports.ResultadoSubidaLocal{}, errors.Join(docports.ErrSubidaLocalPendiente, err)
		}
		fallthrough
	case docports.EstadoSubidaLocalRecibida, docports.EstadoSubidaLocalAnalizando:
		return s.analizar(ctx, r, i)
	default:
		return docports.ResultadoSubidaLocal{}, docports.ErrSubidaLocalPendiente
	}
}

func (s *ServicioSubidaLocal) analizar(ctx context.Context, r docports.ReservaSubidaLocal,
	i docports.InstantaneaSubidaLocal) (docports.ResultadoSubidaLocal, error) {
	if i.Cuarentena == nil || i.Cuarentena.Validar() != nil || i.Cuarentena.Objeto.Zona != vecports.ZonaAlmacenCuarentena ||
		i.Cuarentena.Objeto.HuellaSHA256 != r.HuellaPDFSHA256 {
		return i.Resultado, docports.ErrSubidaLocalPendiente
	}
	if i.Resultado.Estado == docports.EstadoSubidaLocalRecibida {
		m, err := s.movimiento(ctx, r, i, docports.FaseSubidaLocalIniciarAnalisis, nil, nil, nil, "", "")
		if err != nil {
			return i.Resultado, err
		}
		i, err = s.d.Registro.IniciarAnalisis(ctx, m)
		if err != nil {
			return i.Resultado, errors.Join(docports.ErrSubidaLocalPendiente, err)
		}
	}
	if i.Resultado.Estado != docports.EstadoSubidaLocalAnalizando {
		return i.Resultado, docports.ErrSubidaLocalPendiente
	}
	obj := i.Cuarentena.Objeto.Objeto
	if i.Analisis == nil {
		v, err := s.vinculos(ctx, i, vecports.FaseAlmacenSubidaLocalAnalizar, obj)
		if err != nil {
			return i.Resultado, err
		}
		contexto, err := s.d.Autoridad.AutorizarAlmacenSubidaLocal(ctx, r.ActorRegistradoV2, r.Acto,
			s.c.AccionSubir, r.CorrelacionRef, vecports.FaseAlmacenSubidaLocalAnalizar, v)
		if err != nil {
			return i.Resultado, err
		}
		lecturaSol := vecports.SolicitudAbrirObjeto{Contexto: contexto, Objeto: obj,
			Zona: vecports.ZonaAlmacenCuarentena, Limite: limitePDFSubidaLocal}
		lectura, err := s.d.Almacen.Abrir(ctx, lecturaSol)
		if err != nil {
			return i.Resultado, errors.Join(docports.ErrSubidaLocalPendiente, err)
		}
		defer lectura.Contenido.Close()
		if lectura.ValidarContra(lecturaSol) != nil || lectura.Objeto.HuellaSHA256 != r.HuellaPDFSHA256 {
			return i.Resultado, docports.ErrSubidaLocalPendiente
		}
		contextoAnalisis, err := contexto.DerivarPaso(vecports.PasoAlmacenAnalizarContenido)
		if err != nil {
			return i.Resultado, err
		}
		sol := vecports.SolicitudAnalizarContenido{Contexto: contextoAnalisis, Objeto: obj,
			ConectorAlmacenID: lectura.Objeto.ConectorID, Zona: vecports.ZonaAlmacenCuarentena,
			MIME: "application/pdf", Tamano: i.Tamano, HuellaSHA256: r.HuellaPDFSHA256, Contenido: lectura.Contenido}
		cap, err := s.d.Analizador.Capacidades(ctx)
		if err != nil || !cap.DetectaMalware || !cap.DetectaContenidoActivo || cap.TamanoMaximo < i.Tamano ||
			sol.Validar() != nil || contextoAnalisis.ValidarParaEn(vecports.AccionAlmacenAnalizarContenido, s.d.Reloj.Ahora().UTC()) != nil {
			return i.Resultado, docports.ErrSubidaLocalNoDisponible
		}
		resultado, err := s.d.Analizador.Analizar(ctx, sol)
		if err != nil {
			return i.Resultado, errors.Join(docports.ErrSubidaLocalPendiente, err)
		}
		if resultado.ValidarContra(sol) != nil {
			return i.Resultado, docports.ErrSubidaLocalPendiente
		}
		m, err := s.movimiento(ctx, r, i, docports.FaseSubidaLocalRegistrarAnalisis, nil, &resultado, nil, "", "")
		if err != nil {
			return i.Resultado, err
		}
		i, err = s.d.Registro.RegistrarAnalisis(ctx, m)
		if err != nil {
			return docports.ResultadoSubidaLocal{}, errors.Join(docports.ErrSubidaLocalPendiente, err)
		}
	}
	if i.Version != 4 || i.Analisis == nil || i.Analisis.Validar() != nil || i.Analisis.Objeto != obj ||
		i.Analisis.HuellaObjetoSHA256 != r.HuellaPDFSHA256 {
		return i.Resultado, docports.ErrSubidaLocalPendiente
	}
	resultado := *i.Analisis
	var promovido *vecports.ResultadoOperacionObjeto
	final := docports.EstadoSubidaLocalRechazada
	codigo := "contenido_no_permitido"
	if resultado.Estado == vecports.EstadoAnalisisContenidoLimpio {
		p, encontrado, err := s.recuperarOperacion(ctx, r, i,
			vecports.FaseAlmacenSubidaLocalRecuperarPromocion, obj, resultado.EvidenciaRef)
		if err != nil {
			return i.Resultado, errors.Join(docports.ErrSubidaLocalPendiente, err)
		}
		if !encontrado {
			caps, err := s.d.Almacen.Capacidades(ctx)
			if err != nil || !caps.PromocionAtomica || !caps.PreservaObjetoOriginal {
				return i.Resultado, docports.ErrSubidaLocalNoDisponible
			}
			v, err := s.vinculos(ctx, i, vecports.FaseAlmacenSubidaLocalPromover, obj)
			if err != nil {
				return i.Resultado, err
			}
			ctxPromover, err := s.d.Autoridad.AutorizarAlmacenSubidaLocal(ctx, r.ActorRegistradoV2, r.Acto,
				s.c.AccionSubir, r.CorrelacionRef, vecports.FaseAlmacenSubidaLocalPromover, v)
			if err != nil {
				return i.Resultado, err
			}
			promSol := vecports.SolicitudPromoverObjeto{Contexto: ctxPromover, ClaveIdempotencia: r.CargaID + ":promover",
				Origen: obj, EvidenciaAnalisisRef: resultado.EvidenciaRef}
			if promSol.Validar() != nil || ctxPromover.ValidarParaEn(vecports.AccionAlmacenPromover, s.d.Reloj.Ahora().UTC()) != nil {
				return i.Resultado, docports.ErrSubidaLocalDenegada
			}
			p, err = s.d.Almacen.Promover(ctx, promSol)
			if err != nil {
				return i.Resultado, errors.Join(docports.ErrSubidaLocalPendiente, err)
			}
			if p.ValidarPromocion(promSol, i.Cuarentena.Objeto, caps) != nil {
				return i.Resultado, docports.ErrSubidaLocalPendiente
			}
		}
		if p.Validar() != nil || p.Objeto.Zona != vecports.ZonaAlmacenAdmitida ||
			p.Objeto.HuellaSHA256 != i.HuellaPDFSHA256 || p.Evidencia.FundamentoRef != resultado.EvidenciaRef {
			return i.Resultado, docports.ErrSubidaLocalPendiente
		}
		promovido = &p
		final = docports.EstadoSubidaLocalAdmitida
		codigo = ""
	} else if resultado.Estado == vecports.EstadoAnalisisContenidoNoConcluyente ||
		resultado.Estado == vecports.EstadoAnalisisContenidoError {
		codigo = "analisis_no_concluyente"
	}
	m, err := s.movimiento(ctx, r, i, docports.FaseSubidaLocalCerrarAnalisis, nil, &resultado, promovido, final, codigo)
	if err != nil {
		return i.Resultado, err
	}
	i, err = s.d.Registro.CerrarAnalisis(ctx, m)
	if err != nil {
		return docports.ResultadoSubidaLocal{}, errors.Join(docports.ErrSubidaLocalPendiente, err)
	}
	if final == docports.EstadoSubidaLocalRechazada {
		return i.Resultado, docports.ErrSubidaLocalRechazada
	}
	if i.Resultado.Estado != docports.EstadoSubidaLocalAdmitida || i.Resultado.ComprobanteRef == "" ||
		i.Resultado.DocumentoVersion < 1 || i.Resultado.RegistradoEn.IsZero() {
		return i.Resultado, docports.ErrSubidaLocalPendiente
	}
	i.Resultado.RecursoRef = i.RecursoRef
	return i.Resultado, nil
}

func (s *ServicioSubidaLocal) consultarSQL(ctx context.Context, q docports.ConsultaPersistenciaSubidaLocal,
	recurso, finalidad string) (docports.InstantaneaSubidaLocal, error) {
	pre, err := docports.PreimagenConsultaSubidaLocal(q)
	if err != nil {
		return docports.InstantaneaSubidaLocal{}, err
	}
	id := q.CargaID
	if id == "" {
		id = q.DocumentoID
	}
	q.AutorizacionV3, err = s.d.Autoridad.AutorizarSubidaLocal(ctx, pre, q.ActorRegistradoV2, id, recurso, finalidad, q.CorrelacionRef)
	if err != nil {
		return docports.InstantaneaSubidaLocal{}, err
	}
	i, err := s.d.Registro.ConsultarPropia(ctx, q)
	if err != nil {
		return docports.InstantaneaSubidaLocal{}, err
	}
	if i.PersonaRef != q.ActorRegistradoV2.Contexto.PersonaRef || i.Resultado.CargaID == "" ||
		(q.CargaID != "" && i.Resultado.CargaID != q.CargaID) ||
		(q.DocumentoID != "" && i.Resultado.ComprobanteRef != q.DocumentoID) {
		return docports.InstantaneaSubidaLocal{}, docports.ErrSubidaLocalDenegada
	}
	i.Resultado.RecursoRef = i.RecursoRef
	return i, nil
}

func (s *ServicioSubidaLocal) Consultar(ctx context.Context, q docports.SolicitudConsultaLocal) (docports.ResultadoSubidaLocal, error) {
	if s == nil || ctx == nil || ctx.Err() != nil {
		return docports.ResultadoSubidaLocal{}, docports.ErrSubidaLocalNoDisponible
	}
	a, err := q.ActorRegistradoV2.Clonar()
	if err != nil || q.VinculoV2.ValidarPara(a) != nil {
		return docports.ResultadoSubidaLocal{}, docports.ErrSubidaLocalDenegada
	}
	if q.DocumentoID != "" {
		i, err := s.consultarSQL(ctx, docports.ConsultaPersistenciaSubidaLocal{
			ActorRegistradoV2: a, VinculoV2: q.VinculoV2, Accion: s.c.AccionDescargar,
			Fase: docports.FaseSubidaLocalConsultarDocumento, DocumentoID: q.DocumentoID,
			CorrelacionRef: q.CorrelacionRef, RecursoRef: q.DocumentoID}, q.DocumentoID, s.c.Finalidad)
		return i.Resultado, err
	}
	if q.ClaveIdempotencia == "" {
		return docports.ResultadoSubidaLocal{}, docports.ErrSubidaLocalDenegada
	}
	r := docports.SolicitudSubidaLocal{ActorRegistradoV2: a, VinculoV2: q.VinculoV2,
		ClaveIdempotencia: q.ClaveIdempotencia}
	acto, err := s.acto(ctx, r, a)
	if err != nil {
		return docports.ResultadoSubidaLocal{}, err
	}
	corr, id, _, err := s.referencias(ctx, a, q.ClaveIdempotencia)
	if err != nil {
		return docports.ResultadoSubidaLocal{}, err
	}
	i, err := s.consultarSQL(ctx, docports.ConsultaPersistenciaSubidaLocal{
		ActorRegistradoV2: a, VinculoV2: q.VinculoV2, Accion: s.c.AccionSubir,
		Fase: docports.FaseSubidaLocalConsultarCarga, CargaID: id, CorrelacionRef: corr,
		RecursoRef: id}, id, acto.Finalidad)
	if err != nil {
		return docports.ResultadoSubidaLocal{}, err
	}
	if i.RecursoRef != acto.Recurso.Referencia {
		return docports.ResultadoSubidaLocal{}, docports.ErrSubidaLocalDenegada
	}
	return i.Resultado, nil
}

func (s *ServicioSubidaLocal) AbrirOriginal(ctx context.Context, q docports.SolicitudConsultaLocal) (io.ReadCloser, error) {
	if s == nil || ctx == nil || ctx.Err() != nil || q.DocumentoID == "" || q.CorrelacionRef == "" {
		return nil, docports.ErrSubidaLocalDenegada
	}
	a, err := q.ActorRegistradoV2.Clonar()
	if err != nil || q.VinculoV2.ValidarPara(a) != nil {
		return nil, docports.ErrSubidaLocalDenegada
	}
	primera, err := s.consultarSQL(ctx, docports.ConsultaPersistenciaSubidaLocal{
		ActorRegistradoV2: a, VinculoV2: q.VinculoV2, Accion: s.c.AccionDescargar,
		Fase: docports.FaseSubidaLocalConsultarDocumento, DocumentoID: q.DocumentoID,
		CorrelacionRef: q.CorrelacionRef, RecursoRef: q.DocumentoID}, q.DocumentoID, s.c.Finalidad)
	if err != nil {
		return nil, err
	}
	if primera.Admitido == nil || primera.Admitido.Validar() != nil ||
		primera.Resultado.Estado != docports.EstadoSubidaLocalAdmitida {
		return nil, docports.ErrSubidaLocalDenegada
	}
	segunda, err := s.consultarSQL(ctx, docports.ConsultaPersistenciaSubidaLocal{
		ActorRegistradoV2: a, VinculoV2: q.VinculoV2, Accion: s.c.AccionDescargar,
		Fase: docports.FaseSubidaLocalAbrir, CargaID: primera.Resultado.CargaID,
		DocumentoID: q.DocumentoID, CorrelacionRef: q.CorrelacionRef,
		Objeto: primera.Admitido.Objeto.Objeto, HuellaPDFSHA256: primera.HuellaPDFSHA256,
		Tamano: primera.Tamano, MIME: primera.MIME,
		RecursoRef: primera.RecursoRef}, primera.RecursoRef, s.c.Finalidad)
	if err != nil || segunda.Resultado.Estado != docports.EstadoSubidaLocalAdmitida || segunda.Admitido == nil ||
		segunda.Admitido.Validar() != nil || segunda.Resultado.DocumentoVersion < 1 ||
		segunda.RecursoRef != primera.RecursoRef || segunda.CorrelacionRef != primera.CorrelacionRef {
		return nil, docports.ErrSubidaLocalDenegada
	}
	if segunda.Admitido.Objeto.Objeto != primera.Admitido.Objeto.Objeto ||
		segunda.HuellaPDFSHA256 != primera.HuellaPDFSHA256 || segunda.Tamano != primera.Tamano ||
		segunda.MIME != primera.MIME {
		return nil, docports.ErrSubidaLocalDenegada
	}
	acto := docports.ActoSubidaLocal{Recurso: vecdomain.RecursoAutorizable{
		Referencia: segunda.RecursoRef, ModuloID: s.c.ModuloID, Tipo: s.c.TipoRecurso},
		PersonaRef: a.Contexto.PersonaRef, Finalidad: s.c.Finalidad, Clasificacion: segunda.Clasificacion,
		VersionDueno: a.Contexto.Instantanea.PersonaVersion}
	obj := segunda.Admitido.Objeto.Objeto
	v, err := s.vinculos(ctx, segunda, vecports.FaseAlmacenSubidaLocalDescargar, obj)
	if err != nil {
		return nil, err
	}
	contexto, err := s.d.Autoridad.AutorizarAlmacenSubidaLocal(ctx, a, acto, s.c.AccionDescargar,
		q.CorrelacionRef, vecports.FaseAlmacenSubidaLocalDescargar, v)
	if err != nil {
		return nil, err
	}
	sol := vecports.SolicitudAbrirObjeto{Contexto: contexto, Objeto: obj, Zona: vecports.ZonaAlmacenAdmitida,
		Limite: limitePDFSubidaLocal}
	lectura, err := s.d.Almacen.Abrir(ctx, sol)
	if err != nil {
		return nil, err
	}
	defer lectura.Contenido.Close()
	if lectura.ValidarContra(sol) != nil || lectura.Objeto.HuellaSHA256 != segunda.HuellaPDFSHA256 ||
		lectura.Objeto.Tamano != segunda.Tamano || lectura.Objeto.MIME != "application/pdf" {
		return nil, docports.ErrSubidaLocalDenegada
	}
	bytesOriginal, err := io.ReadAll(io.LimitReader(lectura.Contenido, limitePDFSubidaLocal+1))
	if err != nil || len(bytesOriginal) != int(segunda.Tamano) {
		return nil, docports.ErrSubidaLocalNoDisponible
	}
	suma := sha256.Sum256(bytesOriginal)
	if hex.EncodeToString(suma[:]) != segunda.HuellaPDFSHA256 {
		return nil, docports.ErrSubidaLocalDenegada
	}
	return io.NopCloser(bytes.NewReader(bytesOriginal)), nil
}

var _ docports.SubidorLocal = (*ServicioSubidaLocal)(nil)

// ConsultarOriginalAdmitido relee la postimagen propia y abre los bytes con
// una autorización nueva. D20 solo admite el acto M4; las parejas C/D del
// puerto Go permanecen cerradas hasta sus autoridades y SQL nominales.
func (s *ServicioSubidaLocal) ConsultarOriginalAdmitido(ctx context.Context,
	q docports.SolicitudOriginalSubidaLocalPropio) (docports.OriginalSubidaLocalAcreditado, error) {
	var vacio docports.OriginalSubidaLocalAcreditado
	if s == nil || ctx == nil || ctx.Err() != nil || q.DocumentoID == "" ||
		q.DocumentoVersion == 0 || q.CorrelacionRef == "" {
		return vacio, docports.ErrSubidaLocalDenegada
	}
	if s.c.ModuloID != "meritos" || s.c.TipoRecurso != "hecho" ||
		s.c.AccionDescargar != docports.AccionMeritosComprobanteDescargarLocal {
		return vacio, docports.ErrSubidaLocalNoDisponible
	}
	actor, err := q.ActorRegistradoV2.Clonar()
	if err != nil || q.VinculoV2.ValidarPara(actor) != nil {
		return vacio, docports.ErrSubidaLocalDenegada
	}
	consulta := docports.SolicitudConsultaLocal{ActorRegistradoV2: actor, VinculoV2: q.VinculoV2,
		DocumentoID: q.DocumentoID, CorrelacionRef: q.CorrelacionRef}
	primera, err := s.consultarSQL(ctx, docports.ConsultaPersistenciaSubidaLocal{
		ActorRegistradoV2: actor, VinculoV2: q.VinculoV2, Accion: s.c.AccionDescargar,
		Fase: docports.FaseSubidaLocalConsultarDocumento, DocumentoID: q.DocumentoID,
		CorrelacionRef: q.CorrelacionRef, RecursoRef: q.DocumentoID}, q.DocumentoID, s.c.Finalidad)
	if err != nil {
		return vacio, err
	}
	if primera.Resultado.DocumentoVersion < 1 ||
		uint64(primera.Resultado.DocumentoVersion) != q.DocumentoVersion ||
		primera.Resultado.Estado != docports.EstadoSubidaLocalAdmitida {
		return vacio, docports.ErrSubidaLocalDenegada
	}
	original, err := s.AbrirOriginal(ctx, consulta)
	if err != nil {
		return vacio, err
	}
	contenido, errorLectura := io.ReadAll(io.LimitReader(original, limitePDFSubidaLocal+1))
	errorCierre := original.Close()
	if errorLectura != nil || errorCierre != nil {
		return vacio, docports.ErrSubidaLocalNoDisponible
	}
	return acreditarOriginalSubidaLocal(primera, actor.Contexto.PersonaRef,
		q.DocumentoID, q.DocumentoVersion, contenido, s.c)
}

func acreditarOriginalSubidaLocal(i docports.InstantaneaSubidaLocal, persona, documentoID string,
	version uint64, contenido []byte, c ConfiguracionServicioSubidaLocal,
) (docports.OriginalSubidaLocalAcreditado, error) {
	var vacio docports.OriginalSubidaLocalAcreditado
	if c.ModuloID != "meritos" || c.TipoRecurso != "hecho" ||
		c.Finalidad != "custodiar_evidencia_meritos" ||
		i.PersonaRef == "" || i.PersonaRef != persona || i.RecursoRef == "" ||
		i.Resultado.ComprobanteRef != documentoID || i.Resultado.DocumentoVersion < 1 ||
		uint64(i.Resultado.DocumentoVersion) != version ||
		i.Resultado.Estado != docports.EstadoSubidaLocalAdmitida ||
		i.Resultado.ReciboRef == "" || i.Resultado.RegistradoEn.IsZero() ||
		i.Analisis == nil || i.Analisis.Estado != vecports.EstadoAnalisisContenidoLimpio ||
		i.Admitido == nil || i.Admitido.Objeto.Zona != vecports.ZonaAlmacenAdmitida ||
		i.Admitido.Objeto.ConectorID == "" ||
		i.Admitido.Objeto.HuellaSHA256 != i.HuellaPDFSHA256 ||
		i.Admitido.Objeto.Tamano != i.Tamano || i.Admitido.Objeto.MIME != "application/pdf" ||
		i.MIME != "application/pdf" || i.Tamano < 1 || i.Tamano > limitePDFSubidaLocal ||
		int64(len(contenido)) != i.Tamano {
		return vacio, docports.ErrSubidaLocalDenegada
	}
	suma := sha256.Sum256(contenido)
	if hex.EncodeToString(suma[:]) != i.HuellaPDFSHA256 {
		return vacio, docports.ErrSubidaLocalDenegada
	}
	return docports.OriginalSubidaLocalAcreditado{
		DocumentoID: documentoID, DocumentoVersion: version, HuellaSHA256Bytes: i.HuellaPDFSHA256,
		ReciboRef: i.Resultado.ReciboRef, RecursoRef: i.RecursoRef,
		PersonaTitularRef: i.PersonaRef, PersonaAportanteRef: i.PersonaRef,
		CustodioID: i.Admitido.Objeto.ConectorID, ModuloID: c.ModuloID,
		TipoRecurso: c.TipoRecurso, Finalidad: c.Finalidad,
		EstadoAdmision: docports.EstadoSubidaLocalAdmitida,
	}, nil
}

var _ docports.ConsultaOriginalSubidaLocalPropio = (*ServicioSubidaLocal)(nil)
