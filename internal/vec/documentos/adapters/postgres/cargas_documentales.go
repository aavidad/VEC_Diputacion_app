package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// RepositorioCargasDirectas usa exclusivamente las fachadas owner D17. Los
// métodos legados de lectura quedan cerrados: la lectura productiva siempre
// consume V3 antes del SELECT.
type RepositorioCargasDirectas struct{ base *Repositorio }

func NuevoRepositorioCargasDirectas(base *Repositorio) (*RepositorioCargasDirectas, error) {
	if base == nil || base.db == nil {
		return nil, ErrRepositorioNoDisponible
	}
	return &RepositorioCargasDirectas{base: base}, nil
}

func materialCargaV3(a vecports.AutorizacionCargaDocumentalV3) (materialV3, error) {
	if a.Material.ValidarEstructura() != nil || a.Material.PersonaVersion() == 0 ||
		a.Material.PerfilVersion() == 0 || a.Material.PersonaVersion() > 9007199254740991 ||
		a.Material.PerfilVersion() > 9007199254740991 {
		return materialV3{}, ErrRepositorioNoDisponible
	}
	return materialV3{capacidad: a.Material.CapacidadCanonica(), decision: a.Material.DecisionCanonica(),
		motivo: a.Material.MotivoCanonico(), contexto: a.Material.ContextoActorCanonico(),
		persona: int64(a.Material.PersonaVersion()), perfil: int64(a.Material.PerfilVersion()),
		payload: a.Material.PayloadVECAD3(), sobre: a.Material.SobreCOSESign1(),
		evidencia: a.Material.EvidenciaVerificacion(), raiz: a.Material.RaizPublicaSPKI()}, nil
}

func autorizacionCargaJSON(a vecports.AutorizacionCargaDocumentalV3) ([]byte, error) {
	return json.Marshal(struct {
		Accion          string `json:"accion"`
		RecursoRef      string `json:"recurso_ref"`
		AmbitoRef       string `json:"ambito_ref"`
		PrincipalID     string `json:"principal_id"`
		PerfilActivoRef string `json:"perfil_activo_ref"`
		Finalidad       string `json:"finalidad"`
		CorrelacionRef  string `json:"correlacion_ref"`
	}{a.Accion, a.RecursoRef, a.AmbitoRef, a.PersonaRef, a.PerfilActivoRef, a.Finalidad, a.CorrelacionRef})
}

func (r *RepositorioCargasDirectas) Reservar(ctx context.Context, s vecports.SolicitudReservarCargaDocumental) (vecports.ReservaCargaDocumental, error) {
	if r == nil || r.base == nil || s.ValidarParaPersistenciaV3(time.Now().UTC()) != nil {
		return vecports.ReservaCargaDocumental{}, vecports.ErrReservaCargaDocumentalInvalida
	}
	pre, err := vecports.PreimagenReservaCargaDocumentalV3(s, s.ReservaRecursoV3)
	if err != nil {
		return vecports.ReservaCargaDocumental{}, err
	}
	carga, err := json.Marshal(s.Carga)
	if err != nil {
		return vecports.ReservaCargaDocumental{}, vecports.ErrReservaCargaDocumentalInvalida
	}
	reserva, err := json.Marshal(struct {
		RecursoRef    string `json:"recurso_ref"`
		PersonaRef    string `json:"persona_ref"`
		ClaveReserva  string `json:"clave_reserva"`
		HuellaReserva string `json:"huella_reserva_sha256"`
		Version       uint64 `json:"version_reserva"`
	}{s.ReservaRecursoV3.RecursoRef, s.ReservaRecursoV3.PersonaRef,
		s.ReservaRecursoV3.ClaveReserva, s.ReservaRecursoV3.HuellaReservaSHA256, s.ReservaRecursoV3.VersionReserva})
	if err != nil {
		return vecports.ReservaCargaDocumental{}, vecports.ErrReservaCargaDocumentalInvalida
	}
	auth, err := autorizacionCargaJSON(s.AutorizacionV3)
	if err != nil {
		return vecports.ReservaCargaDocumental{}, vecports.ErrReservaCargaDocumentalInvalida
	}
	m, err := materialCargaV3(s.AutorizacionV3)
	if err != nil {
		return vecports.ReservaCargaDocumental{}, err
	}
	raw, err := r.base.transaccion(ctx,
		"SELECT vec_documentos.reservar_carga_directa_v1($1,$2,$3::jsonb,$4::jsonb,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)",
		pre, carga, string(reserva), string(auth), m.capacidad, m.decision, m.motivo, m.contexto,
		m.persona, m.perfil, m.payload, m.sobre, m.evidencia, m.raiz)
	if err != nil {
		return vecports.ReservaCargaDocumental{}, err
	}
	var respuesta struct {
		CargaID   string                                  `json:"carga_id"`
		Carga     vecdomain.CargaDocumental               `json:"carga"`
		Estado    vecports.EstadoOperativoCargaDocumental `json:"estado_operativo"`
		Pendiente bool                                    `json:"pendiente"`
	}
	if json.Unmarshal(raw, &respuesta) != nil || respuesta.CargaID != s.Carga.ID ||
		respuesta.CargaID != respuesta.Carga.ID ||
		respuesta.Carga.Validar() != nil || respuesta.Carga.PrincipalID != s.Carga.PrincipalID ||
		respuesta.Carga.RecursoRef != s.Carga.RecursoRef ||
		respuesta.Carga.IndiceIdempotenciaHMAC != s.IndiceIdempotenciaHMAC ||
		respuesta.Carga.HuellaSolicitudHMAC != s.HuellaSolicitudHMAC ||
		respuesta.Carga.MIMEDeclarado != s.Carga.MIMEDeclarado ||
		respuesta.Carga.TamanoDeclarado != s.Carga.TamanoDeclarado ||
		respuesta.Carga.HuellaDeclaradaSHA256 != s.Carga.HuellaDeclaradaSHA256 ||
		respuesta.Carga.Finalidad != s.Carga.Finalidad ||
		respuesta.Carga.Clasificacion != s.Carga.Clasificacion ||
		respuesta.Carga.OperacionRef != s.Carga.OperacionRef ||
		respuesta.Carga.CorrelacionRef != s.Carga.CorrelacionRef || !respuesta.Estado.Valido() {
		return vecports.ReservaCargaDocumental{}, ErrRepositorioNoDisponible
	}
	if respuesta.Pendiente {
		return vecports.ReservaCargaDocumental{Repetida: true, Pendiente: true,
			EstadoOperativo: respuesta.Estado, Carga: respuesta.Carga}, nil
	}
	if respuesta.Estado != vecports.EstadoCargaOperacionPreparacionIniciada {
		return vecports.ReservaCargaDocumental{}, ErrRepositorioNoDisponible
	}
	if respuesta.CargaID != s.Carga.ID {
		return vecports.ReservaCargaDocumental{}, ErrRepositorioNoDisponible
	}
	return vecports.ReservaCargaDocumental{Token: s.Token, Carga: respuesta.Carga,
		EstadoOperativo: respuesta.Estado}, nil
}

func (r *RepositorioCargasDirectas) ConfirmarPreparacion(ctx context.Context, s vecports.SolicitudConfirmarPreparacionCargaDocumental) error {
	if r == nil || r.base == nil || s.ValidarParaPersistenciaV3(time.Now().UTC()) != nil {
		return vecports.ErrConfirmacionCargaDocumentalInvalida
	}
	instantanea, err := vecports.InstantaneaSolicitudConfirmarPreparacionCargaDocumental(s)
	if err != nil {
		return err
	}
	pre, err := vecports.PreimagenConfirmacionPreparacionCargaDocumentalV3(instantanea)
	if err != nil {
		return err
	}
	carga, err := json.Marshal(instantanea.Confirmacion.Carga)
	if err != nil {
		return vecports.ErrConfirmacionCargaDocumentalInvalida
	}
	datos, err := instantanea.Manifiesto.Datos()
	if err != nil {
		return vecports.ErrConfirmacionCargaDocumentalInvalida
	}
	manifiesto, err := json.Marshal(datos)
	if err != nil {
		return vecports.ErrConfirmacionCargaDocumentalInvalida
	}
	auth, err := autorizacionCargaJSON(instantanea.AutorizacionV3)
	if err != nil {
		return vecports.ErrConfirmacionCargaDocumentalInvalida
	}
	m, err := materialCargaV3(instantanea.AutorizacionV3)
	if err != nil {
		return err
	}
	raw, err := r.base.transaccion(ctx,
		"SELECT vec_documentos.confirmar_preparacion_carga_directa_v1($1,$2,$3,$4::jsonb,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)",
		pre, carga, manifiesto, string(auth), m.capacidad, m.decision, m.motivo, m.contexto,
		m.persona, m.perfil, m.payload, m.sobre, m.evidencia, m.raiz)
	if err != nil {
		return err
	}
	var salida struct {
		CargaID string `json:"carga_id"`
		Estado  string `json:"estado_operativo"`
	}
	if json.Unmarshal(raw, &salida) != nil || salida.CargaID != instantanea.Confirmacion.Carga.ID ||
		salida.Estado != string(vecports.EstadoCargaOperacionEmisionReciboIniciada) {
		return ErrRepositorioNoDisponible
	}
	return nil
}

func (r *RepositorioCargasDirectas) ConsultarPropia(ctx context.Context, s vecports.SolicitudConsultarCargaPropiaV3) (vecports.ConsultaCargaPropiaV3, error) {
	if r == nil || r.base == nil || s.Validar(time.Now().UTC()) != nil {
		return vecports.ConsultaCargaPropiaV3{}, vecdomain.ErrAutorizacionDenegada
	}
	pre, err := vecports.PreimagenConsultaCargaPropiaV3(s)
	if err != nil {
		return vecports.ConsultaCargaPropiaV3{}, vecdomain.ErrAutorizacionDenegada
	}
	auth, err := autorizacionCargaJSON(s.AutorizacionV3)
	if err != nil {
		return vecports.ConsultaCargaPropiaV3{}, vecdomain.ErrAutorizacionDenegada
	}
	m, err := materialCargaV3(s.AutorizacionV3)
	if err != nil {
		return vecports.ConsultaCargaPropiaV3{}, vecdomain.ErrAutorizacionDenegada
	}
	raw, err := r.base.transaccion(ctx,
		"SELECT vec_documentos.consultar_carga_directa_propia_v1($1,$2::jsonb,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)",
		pre, string(auth), m.capacidad, m.decision, m.motivo, m.contexto,
		m.persona, m.perfil, m.payload, m.sobre, m.evidencia, m.raiz)
	if err != nil {
		return vecports.ConsultaCargaPropiaV3{}, err
	}
	return decodificarConsultaCargaPropiaV3(raw, s.CargaID, s.RecursoRef,
		s.ActorRegistradoV2.Contexto.PersonaRef, s.IncluyeManifiesto)
}

func decodificarConsultaCargaPropiaV3(raw []byte, cargaID, recursoRef, personaRef string,
	incluirManifiesto bool) (vecports.ConsultaCargaPropiaV3, error) {
	var salida struct {
		Denegado   bool                                    `json:"denegado"`
		Carga      vecdomain.CargaDocumental               `json:"carga"`
		Preparada  vecdomain.CargaDocumental               `json:"carga_preparada"`
		Manifiesto json.RawMessage                         `json:"manifiesto"`
		Estado     vecports.EstadoOperativoCargaDocumental `json:"estado_operativo"`
	}
	if json.Unmarshal(raw, &salida) != nil {
		return vecports.ConsultaCargaPropiaV3{}, ErrRepositorioNoDisponible
	}
	if salida.Denegado {
		return vecports.ConsultaCargaPropiaV3{}, vecdomain.ErrAutorizacionDenegada
	}
	if salida.Carga.Validar() != nil || salida.Carga.ID != cargaID ||
		salida.Carga.PrincipalID != personaRef ||
		salida.Carga.RecursoRef != recursoRef || !salida.Estado.Valido() {
		return vecports.ConsultaCargaPropiaV3{}, ErrRepositorioNoDisponible
	}
	respuesta := vecports.ConsultaCargaPropiaV3{Carga: salida.Carga, EstadoOperativo: salida.Estado}
	if incluirManifiesto {
		if len(salida.Manifiesto) == 0 || string(salida.Manifiesto) == "null" {
			return vecports.ConsultaCargaPropiaV3{}, vecports.ErrManifiestoPreparacionNoEncontrado
		}
		var datos vecdomain.DatosManifiestoPreparacionCargaDirectaV1
		if json.Unmarshal(salida.Manifiesto, &datos) != nil {
			return vecports.ConsultaCargaPropiaV3{}, ErrRepositorioNoDisponible
		}
		manifiesto, err := vecdomain.RestaurarManifiestoPreparacionCargaDirectaV1(datos)
		if err != nil || manifiesto.ValidarContraCarga(salida.Preparada) != nil ||
			!mismaCargaPreparada(salida.Preparada, salida.Carga) {
			return vecports.ConsultaCargaPropiaV3{}, ErrRepositorioNoDisponible
		}
		respuesta.Manifiesto = &manifiesto
	}
	return respuesta, nil
}

// El manifiesto describe la instantanea v2, no la carga actual. El enlace
// conserva todos los atributos de identidad y preparacion que las transiciones
// v3/v5 no pueden sustituir.
func mismaCargaPreparada(preparada, actual vecdomain.CargaDocumental) bool {
	return preparada.Validar() == nil && actual.Validar() == nil &&
		preparada.Estado == vecdomain.EstadoCargaDocumentalPreparada &&
		preparada.ID == actual.ID && preparada.PrincipalID == actual.PrincipalID &&
		preparada.RecursoRef == actual.RecursoRef && preparada.ModuloID == actual.ModuloID &&
		preparada.TipoRecurso == actual.TipoRecurso && preparada.OperacionRef == actual.OperacionRef &&
		preparada.CorrelacionRef == actual.CorrelacionRef && preparada.Finalidad == actual.Finalidad &&
		preparada.Clasificacion == actual.Clasificacion && preparada.MIMEDeclarado == actual.MIMEDeclarado &&
		preparada.TamanoDeclarado == actual.TamanoDeclarado &&
		preparada.HuellaDeclaradaSHA256 == actual.HuellaDeclaradaSHA256 &&
		preparada.IndiceIdempotenciaHMAC == actual.IndiceIdempotenciaHMAC &&
		preparada.HuellaSolicitudHMAC == actual.HuellaSolicitudHMAC &&
		preparada.VinculoSesionHMAC == actual.VinculoSesionHMAC &&
		preparada.AutorizacionPreparacionRef == actual.AutorizacionPreparacionRef &&
		preparada.CreadaEn.Equal(actual.CreadaEn) &&
		preparada.PreparadaEn.Equal(actual.PreparadaEn) && preparada.ExpiraEn.Equal(actual.ExpiraEn)
}

// ConsultarAdmitidaCustodia consume la decisión V3 dentro de D17 antes de
// proyectar la carga. El objeto que se abrirá después procede solo de esta
// fila; la instantánea de entrada se usa exclusivamente para cotejo.
func (r *RepositorioCargasDirectas) ConsultarAdmitidaCustodia(ctx context.Context,
	s vecports.ConsultaCargaAdmitidaCustodiaV3) (vecdomain.CargaDocumental, error) {
	if r == nil || r.base == nil || s.Validar(time.Now().UTC()) != nil {
		return vecdomain.CargaDocumental{}, vecdomain.ErrAutorizacionDenegada
	}
	pre, err := vecports.PreimagenConsultaCargaAdmitidaCustodiaV3(s)
	if err != nil {
		return vecdomain.CargaDocumental{}, vecdomain.ErrAutorizacionDenegada
	}
	auth, err := autorizacionCargaJSON(s.AutorizacionV3)
	if err != nil {
		return vecdomain.CargaDocumental{}, vecdomain.ErrAutorizacionDenegada
	}
	m, err := materialCargaV3(s.AutorizacionV3)
	if err != nil {
		return vecdomain.CargaDocumental{}, vecdomain.ErrAutorizacionDenegada
	}
	raw, err := r.base.transaccion(ctx,
		"SELECT vec_documentos.consultar_carga_admitida_custodia_v1($1,$2::jsonb,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)",
		pre, string(auth), m.capacidad, m.decision, m.motivo, m.contexto,
		m.persona, m.perfil, m.payload, m.sobre, m.evidencia, m.raiz)
	if err != nil {
		if errors.Is(err, docports.ErrAccesoDenegado) || errors.Is(err, docports.ErrNoEncontrado) {
			return vecdomain.CargaDocumental{}, vecdomain.ErrAutorizacionDenegada
		}
		return vecdomain.CargaDocumental{}, err
	}
	var salida struct {
		Denegado bool                      `json:"denegado"`
		Carga    vecdomain.CargaDocumental `json:"carga"`
	}
	if json.Unmarshal(raw, &salida) != nil || salida.Denegado || salida.Carga.Validar() != nil ||
		salida.Carga.Estado != vecdomain.EstadoCargaDocumentalAdmitida || salida.Carga.Version != 5 ||
		salida.Carga.ID != s.CargaID || salida.Carga.RecursoRef != s.RecursoRef ||
		salida.Carga.PrincipalID != s.PersonaRef || salida.Carga.ModuloID != "meritos" ||
		salida.Carga.Finalidad != s.Finalidad || salida.Carga.Analisis == nil ||
		salida.Carga.Analisis.Estado != vecdomain.EstadoAnalisisCargaLimpio ||
		salida.Carga.ContenidoAdmitido == nil {
		return vecdomain.CargaDocumental{}, vecdomain.ErrAutorizacionDenegada
	}
	huella, err := salida.Carga.HuellaSHA256()
	if err != nil {
		return vecdomain.CargaDocumental{}, ErrRepositorioNoDisponible
	}
	if s.Fase == "abrir" && (huella != s.HuellaCargaEsperadaSHA256 ||
		salida.Carga.ContenidoAdmitido.Referencia != s.ObjetoRef ||
		salida.Carga.ContenidoAdmitido.Version != s.ObjetoVersion ||
		salida.Carga.ContenidoAdmitido.HuellaSHA256 != s.HuellaContenidoSHA256 ||
		salida.Carga.ContenidoAdmitido.Tamano != s.Tamano ||
		salida.Carga.ContenidoAdmitido.MIME != s.MIME ||
		salida.Carga.Analisis.EvidenciaRef != s.EvidenciaAnalisisRef) {
		return vecdomain.CargaDocumental{}, vecdomain.ErrAutorizacionDenegada
	}
	return salida.Carga, nil
}

func (r *RepositorioCargasDirectas) CerrarIntentoNoRecuperable(ctx context.Context, s vecports.SolicitudConciliarCargaV3) (vecports.EstadoOperativoCargaDocumental, error) {
	if r == nil || r.base == nil || s.Validar(time.Now().UTC()) != nil {
		return "", vecports.ErrConfirmacionCargaDocumentalInvalida
	}
	pre, err := vecports.PreimagenConciliacionCargaV3(s)
	if err != nil {
		return "", err
	}
	auth, err := autorizacionCargaJSON(s.AutorizacionV3)
	if err != nil {
		return "", vecports.ErrConfirmacionCargaDocumentalInvalida
	}
	m, err := materialCargaV3(s.AutorizacionV3)
	if err != nil {
		return "", err
	}
	raw, err := r.base.transaccion(ctx,
		"SELECT vec_documentos.cerrar_carga_directa_no_recuperable_v1($1,$2::jsonb,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)",
		pre, string(auth), m.capacidad, m.decision, m.motivo, m.contexto,
		m.persona, m.perfil, m.payload, m.sobre, m.evidencia, m.raiz)
	if err != nil {
		return "", err
	}
	var salida struct {
		CargaID string                                  `json:"carga_id"`
		Estado  vecports.EstadoOperativoCargaDocumental `json:"estado_operativo"`
	}
	if json.Unmarshal(raw, &salida) != nil || salida.CargaID != s.CargaID ||
		salida.Estado != vecports.EstadoCargaOperacionNoRecuperable {
		return "", ErrRepositorioNoDisponible
	}
	return salida.Estado, nil
}

var _ vecports.RepositorioConsultaCargasDocumentalesV3 = (*RepositorioCargasDirectas)(nil)
var _ vecports.RepositorioConciliacionCargaV3 = (*RepositorioCargasDirectas)(nil)

func (r *RepositorioCargasDirectas) IniciarPromocion(ctx context.Context, s vecports.SolicitudIniciarPromocionCargaV3) (vecports.EstadoOperativoCargaDocumental, error) {
	if r == nil || r.base == nil || s.Validar(time.Now().UTC()) != nil {
		return "", vecports.ErrConfirmacionCargaDocumentalInvalida
	}
	pre, err := vecports.PreimagenIniciarPromocionCargaV3(s)
	if err != nil {
		return "", err
	}
	auth, err := autorizacionCargaJSON(s.AutorizacionV3)
	if err != nil {
		return "", vecports.ErrConfirmacionCargaDocumentalInvalida
	}
	m, err := materialCargaV3(s.AutorizacionV3)
	if err != nil {
		return "", err
	}
	raw, err := r.base.transaccion(ctx,
		"SELECT vec_documentos.iniciar_promocion_carga_directa_v3_v1($1,$2::jsonb,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)",
		pre, string(auth), m.capacidad, m.decision, m.motivo, m.contexto,
		m.persona, m.perfil, m.payload, m.sobre, m.evidencia, m.raiz)
	if err != nil {
		return "", err
	}
	var salida struct {
		CargaID string                                  `json:"carga_id"`
		Estado  vecports.EstadoOperativoCargaDocumental `json:"estado_operativo"`
		Indice  string                                  `json:"indice_promocion_hmac"`
	}
	if json.Unmarshal(raw, &salida) != nil || salida.CargaID != s.Carga.ID ||
		salida.Estado != vecports.EstadoCargaOperacionPromocionIniciada ||
		salida.Indice != s.IndicePromocionHMAC {
		return "", ErrRepositorioNoDisponible
	}
	return salida.Estado, nil
}

var _ vecports.RepositorioIniciarPromocionCargaV3 = (*RepositorioCargasDirectas)(nil)

// El protocolo V3 usa ConsultarPropia; estos métodos no leen por ID.
func (*RepositorioCargasDirectas) Obtener(context.Context, string) (vecdomain.CargaDocumental, error) {
	return vecdomain.CargaDocumental{}, vecports.ErrRepositorioCargasNoDisponible
}
func (*RepositorioCargasDirectas) ObtenerPreparacion(context.Context, string) (vecports.PreparacionCargaDocumentalPersistida, error) {
	return vecports.PreparacionCargaDocumentalPersistida{}, vecports.ErrRepositorioCargasNoDisponible
}
func (*RepositorioCargasDirectas) AbandonarReserva(context.Context, vecports.TokenReservaCargaDocumental) error {
	// La reserva iniciada no se roba ni se revierte después de una respuesta incierta.
	return vecports.ErrPreparacionCargaDocumentalPendiente
}
func (r *RepositorioCargasDirectas) ConfirmarTransicion(ctx context.Context, c vecports.ConfirmacionTransicionCargaDocumental) error {
	if r == nil || r.base == nil || c.ValidarParaPersistenciaV3(time.Now().UTC()) != nil {
		return vecports.ErrConfirmacionCargaDocumentalInvalida
	}
	var funcion string
	var estadoEsperado string
	switch c.Carga.Estado {
	case vecdomain.EstadoCargaDocumentalCuarentena:
		if c.AutorizacionV3.Accion != vecports.AccionCargaDocumentalConfirmarResultadoV3 {
			return vecports.ErrConfirmacionCargaDocumentalInvalida
		}
		funcion = "vec_documentos.confirmar_resultado_carga_directa_v3_v1"
		estadoEsperado = "cuarentena"
	case vecdomain.EstadoCargaDocumentalAnalizadaLimpia, vecdomain.EstadoCargaDocumentalRetenidaSeguridad:
		if c.AutorizacionV3.Accion != vecports.AccionCargaDocumentalAnalizarV3 {
			return vecports.ErrConfirmacionCargaDocumentalInvalida
		}
		funcion = "vec_documentos.registrar_analisis_carga_directa_v3_v1"
		estadoEsperado = string(c.Carga.Estado)
	case vecdomain.EstadoCargaDocumentalAdmitida:
		if c.AutorizacionV3.Accion != vecports.AccionCargaDocumentalPromoverResultadoV3 {
			return vecports.ErrConfirmacionCargaDocumentalInvalida
		}
		funcion = "vec_documentos.confirmar_promocion_carga_directa_v3_v1"
		estadoEsperado = "admitida"
	default:
		// La promoción necesita una intención V3 anterior al efecto S3.
		return vecports.ErrConfirmacionCargaDocumentalPendiente
	}
	c = vecports.InstantaneaConfirmacionTransicionCargaDocumental(c)
	pre, err := vecports.PreimagenTransicionCargaDocumentalV3(c)
	if err != nil {
		return err
	}
	carga, err := json.Marshal(c.Carga)
	if err != nil {
		return vecports.ErrConfirmacionCargaDocumentalInvalida
	}
	auth, err := autorizacionCargaJSON(c.AutorizacionV3)
	if err != nil {
		return vecports.ErrConfirmacionCargaDocumentalInvalida
	}
	m, err := materialCargaV3(c.AutorizacionV3)
	if err != nil {
		return err
	}
	raw, err := r.base.transaccion(ctx,
		"SELECT "+funcion+"($1,$2,$3::jsonb,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)",
		pre, carga, string(auth), m.capacidad, m.decision, m.motivo, m.contexto,
		m.persona, m.perfil, m.payload, m.sobre, m.evidencia, m.raiz)
	if err != nil {
		return err
	}
	var salida struct {
		CargaID string                    `json:"carga_id"`
		Estado  string                    `json:"estado_operativo"`
		Carga   vecdomain.CargaDocumental `json:"carga"`
	}
	if json.Unmarshal(raw, &salida) != nil || salida.CargaID != c.Carga.ID || salida.Estado != estadoEsperado ||
		salida.Carga.Validar() != nil {
		return ErrRepositorioNoDisponible
	}
	huellaSQL, err := salida.Carga.HuellaSHA256()
	huellaGo, errGo := c.Carga.HuellaSHA256()
	if err != nil || errGo != nil || huellaSQL != huellaGo {
		return ErrRepositorioNoDisponible
	}
	return nil
}

var _ vecports.RepositorioCargasDocumentales = (*RepositorioCargasDirectas)(nil)
