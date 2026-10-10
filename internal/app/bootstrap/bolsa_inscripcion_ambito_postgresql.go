package bootstrap

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/bolsa/application/inscripcion"
)

const consultaConjuntoRRHHInscripcionPrevia = `SELECT vec_bolsa_llamamientos.resolver_conjunto_rrhh_inscripcion_previa_v1()`
const consultaAmbitoRRHHInscripcionPrevia = `SELECT vec_bolsa_llamamientos.resolver_ambito_rrhh_inscripcion_previa_v1($1::text)`
const consultaAmbitoRRHHInscripcionAuditada = `SELECT vec_bolsa_llamamientos.resolver_ambito_rrhh_inscripcion_v1($1::text,$2::bytea,$3::bytea,$4::jsonb)`

type iniciadorTransaccionesAmbitoInscripcion interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

// Usa exclusivamente el LOGIN lector RRHH: las funciones SECDEF comprueban
// session_user y las ACL deniegan los otros tres canales de inscripción.
type fuenteAmbitoRRHHInscripcionPostgreSQL struct {
	pool        iniciadorTransaccionesAmbitoInscripcion
	reloj       interface{ Ahora() time.Time }
	rrhhNominal []identidadRRHHInscripcionBolsa
}

var _ FuenteAmbitoLecturaRRHHInscripcionBolsa = (*fuenteAmbitoRRHHInscripcionPostgreSQL)(nil)
var _ FuenteAmbitoRecursoInscripcionBolsa = (*fuenteAmbitoRRHHInscripcionPostgreSQL)(nil)

func NuevaFuenteAmbitoRRHHInscripcionPostgreSQL(pool *pgxpool.Pool, reloj interface{ Ahora() time.Time }, rrhh []identidadConsultaRRHHDesarrollo) (*fuenteAmbitoRRHHInscripcionPostgreSQL, error) {
	if pool == nil {
		return nil, inscripcion.ErrNoDisponible
	}
	return nuevaFuenteAmbitoRRHHInscripcionPostgreSQL(pool, reloj, rrhh)
}

func nuevaFuenteAmbitoRRHHInscripcionPostgreSQL(pool iniciadorTransaccionesAmbitoInscripcion, reloj interface{ Ahora() time.Time }, rrhh []identidadConsultaRRHHDesarrollo) (*fuenteAmbitoRRHHInscripcionPostgreSQL, error) {
	if nuloInscripcionBolsa(pool) || nuloInscripcionBolsa(reloj) || !identidadesRRHHInscripcionValidas(rrhh) {
		return nil, inscripcion.ErrNoDisponible
	}
	return &fuenteAmbitoRRHHInscripcionPostgreSQL{pool: pool, reloj: reloj,
		rrhhNominal: copiarIdentidadesRRHHInscripcion(rrhh)}, nil
}

func (f *fuenteAmbitoRRHHInscripcionPostgreSQL) sesionRRHHValida(ctx context.Context, s contextoSeguridadComunDesarrollo, a AcreditacionSesionInscripcionBolsa) bool {
	if f == nil || ctx == nil || ctx.Err() != nil || nuloInscripcionBolsa(f.pool) || nuloInscripcionBolsa(f.reloj) {
		return false
	}
	ahora := f.reloj.Ahora().UTC().Truncate(time.Microsecond)
	return acreditacionInscripcionBolsaActual(s, a, ahora) && rrhhNominalInscripcionEnLista(s, a, f.rrhhNominal)
}

func (f *fuenteAmbitoRRHHInscripcionPostgreSQL) transaccion(ctx context.Context, trabajo func(pgx.Tx) error) error {
	tx, err := f.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return inscripcion.ErrNoDisponible
	}
	defer func() {
		rollbackCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = tx.Rollback(rollbackCtx)
	}()
	if err := trabajo(tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return inscripcion.ErrNoDisponible
	}
	return nil
}

func decodificarAmbitoInscripcion(raw []byte, destino any) bool {
	if len(raw) == 0 || len(raw) > 4096 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return false
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(destino) != nil {
		return false
	}
	var sobra any
	return d.Decode(&sobra) == io.EOF
}

func (f *fuenteAmbitoRRHHInscripcionPostgreSQL) ResolverAmbitoLecturaRRHH(ctx context.Context, s contextoSeguridadComunDesarrollo, a AcreditacionSesionInscripcionBolsa, accion, recurso string, filtro inscripcion.Filtro) (AmbitoLecturaRRHHInscripcionBolsa, error) {
	vacio := AmbitoLecturaRRHHInscripcionBolsa{}
	if !f.sesionRRHHValida(ctx, s, a) || !accionRRHHInscripcion(accion) || !accionLecturaInscripcion(accion) || recurso == "" {
		return vacio, inscripcion.ErrAccesoDenegado
	}
	if accion == inscripcion.AccionListarRRHH && (filtro.Validar() != nil || filtro.ConvocatoriaRef == "") {
		return vacio, inscripcion.ErrSolicitudInvalida
	}
	if accion == inscripcion.AccionConvocatoriasRRHH &&
		(filtro.Validar() != nil || filtro.Estado != "" || filtro.ConvocatoriaRef != "" ||
			(filtro.Cursor != "" && !inscripcion.ConvocatoriaRefValida(filtro.Cursor))) {
		return vacio, inscripcion.ErrSolicitudInvalida
	}
	var conjunto ConjuntoGestionRRHHInscripcionBolsa
	var historico *AmbitoSolicitudRRHHInscripcionBolsa
	err := f.transaccion(ctx, func(tx pgx.Tx) error {
		var raw []byte
		if tx.QueryRow(ctx, consultaConjuntoRRHHInscripcionPrevia).Scan(&raw) != nil || !decodificarAmbitoInscripcion(raw, &conjunto) {
			return inscripcion.ErrAccesoDenegado
		}
		base := inscripcion.AmbitoGestionInscripcion{ConjuntoRef: conjunto.ConjuntoRef, UnidadRef: conjunto.UnidadRef,
			AmbitoRef: conjunto.AmbitoRef, FuenteRef: conjunto.FuenteRef, FuenteVersion: conjunto.FuenteVersion,
			FuenteSHA256: conjunto.FuenteHuellaSHA256}
		if !base.Valido(true) {
			return inscripcion.ErrAccesoDenegado
		}
		if accion == inscripcion.AccionDetalleRRHH {
			raw = nil
			if tx.QueryRow(ctx, consultaAmbitoRRHHInscripcionPrevia, recurso).Scan(&raw) != nil {
				return inscripcion.ErrAccesoDenegado
			}
			var h AmbitoSolicitudRRHHInscripcionBolsa
			if !decodificarAmbitoInscripcion(raw, &h) || h.SolicitudRef != recurso ||
				h.UnidadRef != conjunto.UnidadRef || h.AmbitoRef != conjunto.AmbitoRef ||
				!(inscripcion.AmbitoGestionInscripcion{UnidadRef: h.UnidadRef, AmbitoRef: h.AmbitoRef, FuenteRef: h.FuenteRef,
					FuenteVersion: h.FuenteVersion, FuenteSHA256: h.FuenteHuellaSHA256}).Valido(false) {
				return inscripcion.ErrAccesoDenegado
			}
			historico = &h
		}
		return nil
	})
	if err != nil || !f.sesionRRHHValida(ctx, s, a) {
		return vacio, inscripcion.ErrAccesoDenegado
	}
	return AmbitoLecturaRRHHInscripcionBolsa{RecursoRef: recurso, ConvocatoriaRef: filtro.ConvocatoriaRef,
		ConjuntoGestion: conjunto, AmbitoSolicitud: historico}, nil
}

// El POST usa el permiso RRHH.consultar ya capturado; la función B96 asienta
// una lectura en la misma TX y devuelve sólo el ámbito histórico opaco.
func (f *fuenteAmbitoRRHHInscripcionPostgreSQL) ResolverAmbitoRRHH(ctx context.Context, s contextoSeguridadComunDesarrollo, a AcreditacionSesionInscripcionBolsa, recurso string, captura inscripcion.CapturaLectura) (AmbitoRecursoRRHHInscripcionBolsa, error) {
	vacio := AmbitoRecursoRRHHInscripcionBolsa{}
	if !f.sesionRRHHValida(ctx, s, a) || captura.Accion != inscripcion.AccionDetalleRRHH ||
		captura.RecursoRef != recurso || captura.ConjuntoGestion == nil || captura.AmbitoSolicitud == nil {
		return vacio, inscripcion.ErrAccesoDenegado
	}
	actor := inscripcion.Actor{PersonaRef: a.PersonaRef, PerfilRef: a.PerfilRef, SesionRef: a.SesionRef, Idioma: "es",
		Canal: a.Canal, ResultadoContexto: s.Resultado, Vinculo: s.Vinculo, Lectura: &captura}
	if !actor.LecturaValida(inscripcion.AccionDetalleRRHH, recurso, inscripcion.Filtro{}) {
		return vacio, inscripcion.ErrAccesoDenegado
	}
	contexto := bytes.Clone(s.Resultado.RepresentacionCanonica)
	vinculo, err := json.Marshal(s.Vinculo)
	if err != nil || len(contexto) == 0 {
		return vacio, inscripcion.ErrNoDisponible
	}
	capturaJSON, err := capturaSQLAmbitoRRHHInscripcion(captura)
	if err != nil {
		return vacio, inscripcion.ErrNoDisponible
	}
	var salida struct {
		Resultado    string          `json:"resultado"`
		Ambito       json.RawMessage `json:"ambito"`
		AuditoriaRef string          `json:"auditoria_ref"`
	}
	err = f.transaccion(ctx, func(tx pgx.Tx) error {
		var raw []byte
		if tx.QueryRow(ctx, consultaAmbitoRRHHInscripcionAuditada, recurso, contexto, vinculo, capturaJSON).Scan(&raw) != nil ||
			!decodificarAmbitoInscripcion(raw, &salida) || salida.AuditoriaRef == "" {
			return inscripcion.ErrNoDisponible
		}
		if salida.Resultado == "no_encontrada" {
			if !bytes.Equal(bytes.TrimSpace(salida.Ambito), []byte("null")) {
				return inscripcion.ErrNoDisponible
			}
			return nil // El asiento de acceso denegado también se confirma.
		}
		if salida.Resultado != "obtenida" || !decodificarAmbitoInscripcion(salida.Ambito, &vacio) ||
			vacio.SolicitudRef != recurso || vacio.UnidadRef != captura.AmbitoSolicitud.UnidadRef ||
			vacio.AmbitoRef != captura.AmbitoSolicitud.AmbitoRef || vacio.FuenteRef != captura.AmbitoSolicitud.FuenteRef ||
			vacio.FuenteVersion != captura.AmbitoSolicitud.FuenteVersion || vacio.FuenteHuellaSHA256 != captura.AmbitoSolicitud.FuenteSHA256 {
			return inscripcion.ErrAccesoDenegado
		}
		vacio.AuditoriaRef = salida.AuditoriaRef
		return nil
	})
	if err != nil || salida.Resultado != "obtenida" || !f.sesionRRHHValida(ctx, s, a) {
		return AmbitoRecursoRRHHInscripcionBolsa{}, inscripcion.ErrAccesoDenegado
	}
	return vacio, nil
}

func capturaSQLAmbitoRRHHInscripcion(c inscripcion.CapturaLectura) ([]byte, error) {
	var aleatorio [16]byte
	if _, err := rand.Read(aleatorio[:]); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		IntentoRef              string                                `json:"intento_ref"`
		PersonaRef              string                                `json:"persona_ref"`
		PerfilRef               string                                `json:"perfil_ref"`
		CuentaRef               string                                `json:"cuenta_ref"`
		SesionRef               string                                `json:"sesion_ref"`
		AutenticacionRef        string                                `json:"autenticacion_ref"`
		CertificadoHuellaSHA256 string                                `json:"certificado_huella_sha256"`
		Canal                   string                                `json:"canal"`
		Accion                  string                                `json:"accion"`
		RecursoRef              string                                `json:"recurso_ref"`
		Finalidad               string                                `json:"finalidad"`
		CorrelacionRef          string                                `json:"correlacion_ref"`
		RevisionPermisos        uint64                                `json:"revision_permisos"`
		HuellaInstantaneaSHA256 string                                `json:"huella_instantanea_sha256"`
		ConjuntoGestion         *inscripcion.AmbitoGestionInscripcion `json:"conjunto_gestion,omitempty"`
		AmbitoSolicitud         *inscripcion.AmbitoGestionInscripcion `json:"ambito_solicitud,omitempty"`
		Filtro                  struct {
			Estado          string `json:"estado"`
			ConvocatoriaRef string `json:"convocatoria_ref"`
			Limite          int    `json:"limite"`
			Cursor          string `json:"cursor"`
		} `json:"filtro"`
		Idioma      string    `json:"idioma"`
		EmitidaEn   time.Time `json:"emitida_en"`
		ValidaHasta time.Time `json:"valida_hasta"`
	}{IntentoRef: "lectura_" + hex.EncodeToString(aleatorio[:]), PersonaRef: c.PersonaRef, PerfilRef: c.PerfilRef,
		CuentaRef: c.CuentaRef, SesionRef: c.SesionRef, AutenticacionRef: c.AutenticacionRef,
		CertificadoHuellaSHA256: c.CertificadoHuellaSHA256, Canal: c.Canal, Accion: c.Accion,
		RecursoRef: c.RecursoRef, Finalidad: c.Finalidad, CorrelacionRef: c.CorrelacionRef,
		RevisionPermisos: c.RevisionPermisos, HuellaInstantaneaSHA256: c.HuellaInstantaneaSHA256,
		ConjuntoGestion: c.ConjuntoGestion, AmbitoSolicitud: c.AmbitoSolicitud,
		Filtro: struct {
			Estado          string `json:"estado"`
			ConvocatoriaRef string `json:"convocatoria_ref"`
			Limite          int    `json:"limite"`
			Cursor          string `json:"cursor"`
		}{
			c.Filtro.Estado, c.Filtro.ConvocatoriaRef, c.Filtro.Limite, c.Filtro.Cursor},
		Idioma: "es", EmitidaEn: c.EmitidaEn, ValidaHasta: c.ValidaHasta})
}
