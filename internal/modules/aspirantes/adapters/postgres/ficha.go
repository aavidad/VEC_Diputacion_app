// Package postgres implementa el registro durable de la ficha propia sobre
// las funciones de vec_aspirantes. El pool se conecta con el LOGIN técnico
// del portal externo; cada transacción vuelve a acreditarlo.
package postgres

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/aspirantes/canonico"
	"vec-diputacion-granada/internal/modules/aspirantes/domain"
	"vec-diputacion-granada/internal/modules/aspirantes/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var patronAcceso = regexp.MustCompile(`^aspacc_[0-9a-f]{32}$`)

// RolEjecutor es el único rol que puede ejecutar las fachadas.
const RolEjecutor = "vec_aspirantes_ejecutor_externo"

const (
	firmaV3        = `$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,$8::bytea,$9::bytea,$10::bytea,$11::bytea`
	consultarSQL   = `SELECT vec_aspirantes.consultar_ficha_propia_v1($1::text,` + firmaV3 + `)`
	prepararSQL    = `SELECT vec_aspirantes.preparar_rectificacion_ficha_v1($1::text,` + firmaV3 + `)`
	altaSQL        = `SELECT vec_aspirantes.alta_ficha_propia_v1($1::text,$12::jsonb,` + firmaV3 + `)`
	aplicarSQL     = `SELECT vec_aspirantes.aplicar_rectificacion_ficha_v1($1::jsonb)`
	maxRespuestaDB = 64 << 10
	// Los seis ajustes conservan el valor y el alcance de SET LOCAL. La
	// acreditación del LOGIN sigue en la siguiente llamada de cada transacción.
	ajustesTransaccionSQL = `SELECT pg_catalog.set_config('search_path','pg_catalog',true),
 pg_catalog.set_config('row_security','on',true),
 pg_catalog.set_config('timezone','UTC',true),
 pg_catalog.set_config('lock_timeout','3s',true),
 pg_catalog.set_config('statement_timeout','15s',true),
 pg_catalog.set_config('idle_in_transaction_session_timeout','20s',true)`
)

// El LOGIN solo puede heredar el ejecutor externo, sin SET ROLE ni otras
// membresías, y no tiene acceso directo a ninguna tabla.
const acreditarEjecutorSQL = `SELECT session_user=current_user
 AND l.rolcanlogin AND l.rolinherit AND NOT l.rolsuper AND NOT l.rolcreatedb
 AND NOT l.rolcreaterole AND NOT l.rolreplication AND NOT l.rolbypassrls
 AND g.rolname=$1::text AND NOT g.rolcanlogin AND g.rolinherit AND NOT g.rolbypassrls
 AND pg_catalog.pg_has_role(session_user,g.oid,'MEMBER') AND pg_catalog.pg_has_role(session_user,g.oid,'USAGE')
 AND EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member=l.oid AND m.roleid=g.oid
   AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
 AND (SELECT count(*) FROM pg_catalog.pg_auth_members m WHERE m.member=l.oid)=1
 AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member=g.oid)
 AND pg_catalog.has_function_privilege(session_user,'vec_aspirantes.consultar_ficha_propia_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 AND pg_catalog.has_function_privilege(session_user,'vec_aspirantes.alta_ficha_propia_v1(text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 AND pg_catalog.has_function_privilege(session_user,'vec_aspirantes.preparar_rectificacion_ficha_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 AND pg_catalog.has_function_privilege(session_user,'vec_aspirantes.aplicar_rectificacion_ficha_v1(jsonb)','EXECUTE')
 AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_class c WHERE c.relnamespace='vec_aspirantes'::regnamespace AND c.relkind='r'
   AND pg_catalog.has_table_privilege(session_user,c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER'))
 FROM pg_catalog.pg_roles l JOIN pg_catalog.pg_roles g ON g.rolname=$1::text
 WHERE l.rolname=session_user`

type fila interface{ Scan(...any) error }
type transaccion interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	QueryRow(context.Context, string, ...any) fila
	Commit(context.Context) error
	Rollback(context.Context) error
}
type transaccionPGX struct{ pgx.Tx }

func (t transaccionPGX) QueryRow(ctx context.Context, sql string, args ...any) fila {
	return t.Tx.QueryRow(ctx, sql, args...)
}

type RegistroFichasPostgreSQL struct {
	iniciar func(context.Context) (transaccion, error)
}

var _ ports.RegistroFichas = (*RegistroFichasPostgreSQL)(nil)

// NuevoRegistroFichasPostgreSQL no posee ni cierra el pool. Comprueba el
// LOGIN con el mismo camino SERIALIZABLE que las operaciones.
func NuevoRegistroFichasPostgreSQL(ctx context.Context, pool *pgxpool.Pool) (*RegistroFichasPostgreSQL, error) {
	if ctx == nil || pool == nil || ctx.Err() != nil {
		return nil, ports.ErrNoDisponible
	}
	r := &RegistroFichasPostgreSQL{iniciar: func(ctx context.Context) (transaccion, error) {
		tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
		if err != nil {
			return nil, err
		}
		return transaccionPGX{tx}, nil
	}}
	tx, err := r.abrir(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(context.Background())
	if err := tx.Commit(ctx); err != nil {
		return nil, errorSeguro(ctx, err)
	}
	return r, nil
}

func (r *RegistroFichasPostgreSQL) abrir(ctx context.Context) (transaccion, error) {
	if ctx == nil || r == nil || r.iniciar == nil {
		return nil, ports.ErrNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	tx, err := r.iniciar(ctx)
	if err != nil {
		return nil, errorSeguro(ctx, err)
	}
	if tx == nil {
		return nil, ports.ErrNoDisponible
	}
	fallar := func(err error) (transaccion, error) {
		_ = tx.Rollback(context.Background())
		return nil, errorSeguro(ctx, err)
	}
	if _, err := tx.Exec(ctx, ajustesTransaccionSQL); err != nil {
		return fallar(err)
	}
	var valido bool
	if err := tx.QueryRow(ctx, acreditarEjecutorSQL, RolEjecutor).Scan(&valido); err != nil {
		return fallar(err)
	}
	if !valido {
		_ = tx.Rollback(context.Background())
		return nil, ports.ErrNoDisponible
	}
	return tx, nil
}

// validarOrden repite en el adaptador el cotejo de sesión, recurso y V3
// antes de abrir una transacción.
func validarOrden(orden ports.OrdenFicha, m ports.MaterialFicha, v3 vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, accion string) ([]byte, error) {
	actor, _, err := orden.Sesion.Datos()
	if err != nil {
		return nil, ports.ErrNoAutenticado
	}
	audiencia, err := ports.Audiencia(accion)
	if err != nil {
		return nil, ports.ErrProhibido
	}
	recurso, err := canonico.Recurso(m)
	if err != nil {
		return nil, ports.ErrInvalida
	}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return nil, ports.ErrInvalida
	}
	resumen := v3.ResumenCapacidad()
	if m.PersonaRef != actor.PersonaRef || m.PerfilRef != actor.PerfilActivoRef || m.Accion != accion ||
		m.VersionEsperada >= math.MaxInt64 || v3.ValidarEstructura() != nil ||
		v3.PersonaVersion() != actor.Instantanea.PersonaVersion || v3.PerfilVersion() != actor.Instantanea.PerfilVersion ||
		resumen.Operacion() != accion || resumen.AudienciaConsumo() != audiencia ||
		resumen.EfectoRef() != m.PersonaRef || resumen.EfectoHuellaSHA256() != huella {
		return nil, ports.ErrProhibido
	}
	return canonico.SerializarMaterial(m)
}

func argumentosV3(material []byte, v3 vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) []any {
	return []any{string(material), v3.CapacidadCanonica(), v3.DecisionCanonica(), v3.MotivoCanonico(), v3.ContextoActorCanonico(),
		int64(v3.PersonaVersion()), int64(v3.PerfilVersion()), v3.PayloadVECAD3(), v3.SobreCOSESign1(), v3.EvidenciaVerificacion(), v3.RaizPublicaSPKI()}
}

func errorSeguro(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "42501":
			return ports.ErrProhibido
		case "P1409":
			return ports.ErrConflicto
		case "P1411":
			return ports.ErrFichaExistente
		case "P1404":
			return ports.ErrSinFicha
		case "22023":
			return ports.ErrInvalida
		}
	}
	// Incluye 40001: una transacción abortada nunca se presenta como éxito.
	// La persona repite con la misma clave y obtiene el recibo si llegó a
	// confirmarse.
	return ports.ErrNoDisponible
}

func decodificarEstricto(bruto []byte, destino any) error {
	if len(bruto) == 0 || len(bruto) > maxRespuestaDB {
		return ports.ErrNoDisponible
	}
	d := json.NewDecoder(bytes.NewReader(bruto))
	d.DisallowUnknownFields()
	if d.Decode(destino) != nil || d.Decode(new(any)) != io.EOF {
		return ports.ErrNoDisponible
	}
	return nil
}

// ejecutar abre la transacción, llama a la fachada y confirma. La función
// recibe la transacción para las operaciones de dos llamadas.
func (r *RegistroFichasPostgreSQL) ejecutar(ctx context.Context, cuerpo func(transaccion) error) error {
	tx, err := r.abrir(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	if err := cuerpo(tx); err != nil {
		// Solo salen los errores nominales del módulo; el resto (PostgreSQL,
		// red, lectura del resultado) se reduce sin detalle.
		for _, e := range []error{ports.ErrNoAutenticado, ports.ErrProhibido, ports.ErrInvalida, ports.ErrConflicto, ports.ErrFichaExistente, ports.ErrSinFicha, ports.ErrNoDisponible} {
			if errors.Is(err, e) {
				return e
			}
		}
		return errorSeguro(ctx, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return errorSeguro(ctx, err)
	}
	return nil
}

type sobreSQL struct {
	ClaveRef   string `json:"clave_ref"`
	NonceHex   string `json:"nonce_hex"`
	CifradoHex string `json:"cifrado_hex"`
}

func (s sobreSQL) decodificar(maximo int) (ports.SobreCifrado, error) {
	nonce, e1 := hex.DecodeString(s.NonceHex)
	cifrado, e2 := hex.DecodeString(s.CifradoHex)
	if e1 != nil || e2 != nil || s.ClaveRef == "" || len(s.ClaveRef) > 128 || len(nonce) != 12 || len(cifrado) < 17 || len(cifrado) > maximo {
		return ports.SobreCifrado{}, ports.ErrNoDisponible
	}
	return ports.SobreCifrado{ClaveRef: s.ClaveRef, Nonce: nonce, Cifrado: cifrado}, nil
}

func sobreACampos(s ports.SobreCifrado) sobreSQL {
	return sobreSQL{ClaveRef: s.ClaveRef, NonceHex: hex.EncodeToString(s.Nonce), CifradoHex: hex.EncodeToString(s.Cifrado)}
}

type fichaSQL struct {
	Estado       string `json:"estado"`
	AspiranteRef string `json:"aspirante_ref"`
	Version      uint64 `json:"version"`
	AccesoRef    string `json:"acceso_ref"`
	Documento    *struct {
		DocumentoRef string                `json:"documento_ref"`
		Tipo         string                `json:"tipo"`
		Pais         string                `json:"pais"`
		Sobre        sobreSQL              `json:"sobre"`
		Indice       ports.IndiceDocumento `json:"indice"`
	} `json:"documento"`
	Valores []struct {
		Campo   string    `json:"campo"`
		Version uint64    `json:"version"`
		Origen  string    `json:"origen"`
		Sobre   *sobreSQL `json:"sobre"`
	} `json:"valores"`
}

func (r *RegistroFichasPostgreSQL) ConsultarPropia(ctx context.Context, orden ports.OrdenFicha, m ports.MaterialFicha, v3 vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.FichaCifrada, bool, error) {
	material, err := validarOrden(orden, m, v3, ports.AccionConsultar)
	if err != nil {
		return ports.FichaCifrada{}, false, err
	}
	var bruto []byte
	if err := r.ejecutar(ctx, func(tx transaccion) error {
		return tx.QueryRow(ctx, consultarSQL, argumentosV3(material, v3)...).Scan(&bruto)
	}); err != nil {
		return ports.FichaCifrada{}, false, err
	}
	var f fichaSQL
	if err := decodificarEstricto(bruto, &f); err != nil {
		return ports.FichaCifrada{}, false, err
	}
	if f.Estado == "sin_ficha" && f.AspiranteRef == "" && f.Documento == nil && f.Valores == nil {
		return ports.FichaCifrada{}, false, nil
	}
	if f.Estado != "activa" || f.Documento == nil || f.Version == 0 || f.Version > math.MaxInt64 || len(f.Valores) == 0 || len(f.Valores) > 6 ||
		!patronAcceso.MatchString(f.AccesoRef) {
		return ports.FichaCifrada{}, false, ports.ErrNoDisponible
	}
	sobreDoc, err := f.Documento.Sobre.decodificar(46)
	if err != nil {
		return ports.FichaCifrada{}, false, err
	}
	resultado := ports.FichaCifrada{AspiranteRef: f.AspiranteRef, Version: f.Version, AccesoRef: f.AccesoRef,
		Documento: ports.DocumentoCifrado{DocumentoRef: f.Documento.DocumentoRef, Tipo: domain.TipoDocumento(f.Documento.Tipo),
			Pais: f.Documento.Pais, Sobre: sobreDoc, Indice: f.Documento.Indice}}
	for _, v := range f.Valores {
		valor := ports.ValorCifrado{Campo: domain.CampoFicha(v.Campo), Version: v.Version, Origen: domain.OrigenValor(v.Origen)}
		if v.Sobre != nil {
			s, err := v.Sobre.decodificar(1040)
			if err != nil {
				return ports.FichaCifrada{}, false, err
			}
			valor.Sobre = &s
		}
		resultado.Valores = append(resultado.Valores, valor)
	}
	return resultado, true, nil
}

type reciboSQL struct {
	ReciboRef string    `json:"recibo_ref"`
	Accion    string    `json:"accion"`
	Version   uint64    `json:"version"`
	FechaUTC  time.Time `json:"fecha_utc"`
	Replay    bool      `json:"replay"`
}

func (r reciboSQL) recibo(accion string, version uint64) (ports.ReciboFicha, error) {
	_, offset := r.FechaUTC.Zone()
	if len(r.ReciboRef) != 39 || r.Accion != accion || r.Version != version || r.FechaUTC.IsZero() || offset != 0 {
		return ports.ReciboFicha{}, ports.ErrNoDisponible
	}
	return ports.ReciboFicha{ReciboRef: r.ReciboRef, Accion: r.Accion, Version: r.Version, FechaUTC: r.FechaUTC.UTC(), Replay: r.Replay}, nil
}

type valorAltaSQL struct {
	Campo   string `json:"campo"`
	Version uint64 `json:"version"`
	Origen  string `json:"origen"`
	sobreSQL
}

func (r *RegistroFichasPostgreSQL) Alta(ctx context.Context, orden ports.OrdenFicha, m ports.MaterialFicha, v3 vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, f ports.FichaNueva) (ports.ReciboFicha, error) {
	material, err := validarOrden(orden, m, v3, ports.AccionAlta)
	if err != nil {
		return ports.ReciboFicha{}, err
	}
	if !domain.ReferenciaAspiranteValida(f.AspiranteRef) || f.Documento.Indice != m.IndiceDocumento || len(f.Valores) == 0 {
		return ports.ReciboFicha{}, ports.ErrInvalida
	}
	valores := make([]valorAltaSQL, 0, len(f.Valores))
	for _, v := range f.Valores {
		if v.Sobre == nil || v.Version != 1 {
			return ports.ReciboFicha{}, ports.ErrInvalida
		}
		valores = append(valores, valorAltaSQL{Campo: string(v.Campo), Version: v.Version, Origen: string(v.Origen), sobreSQL: sobreACampos(*v.Sobre)})
	}
	doc := sobreACampos(f.Documento.Sobre)
	ficha, err := json.Marshal(map[string]any{
		"aspirante_ref": f.AspiranteRef, "catalogo_ref": f.CatalogoRef, "valores": valores,
		"documento": map[string]any{"documento_ref": f.Documento.DocumentoRef, "tipo": string(f.Documento.Tipo), "pais": f.Documento.Pais,
			"clave_ref": doc.ClaveRef, "nonce_hex": doc.NonceHex, "cifrado_hex": doc.CifradoHex, "indice": f.Documento.Indice},
	})
	if err != nil {
		return ports.ReciboFicha{}, ports.ErrInvalida
	}
	var bruto []byte
	if err := r.ejecutar(ctx, func(tx transaccion) error {
		return tx.QueryRow(ctx, altaSQL, append(argumentosV3(material, v3), string(ficha))...).Scan(&bruto)
	}); err != nil {
		return ports.ReciboFicha{}, err
	}
	var rec reciboSQL
	if err := decodificarEstricto(bruto, &rec); err != nil {
		return ports.ReciboFicha{}, err
	}
	return rec.recibo(ports.AccionAlta, 1)
}

type estadoSQL struct {
	AspiranteRef string     `json:"aspirante_ref"`
	Version      uint64     `json:"version"`
	Presentes    []string   `json:"presentes"`
	Replay       *reciboSQL `json:"replay"`
}

type valorCambioSQL struct {
	Campo      string  `json:"campo"`
	Version    uint64  `json:"version"`
	Origen     string  `json:"origen"`
	Estado     string  `json:"estado"`
	ClaveRef   *string `json:"clave_ref"`
	NonceHex   *string `json:"nonce_hex"`
	CifradoHex *string `json:"cifrado_hex"`
}

func (r *RegistroFichasPostgreSQL) Rectificar(ctx context.Context, orden ports.OrdenFicha, m ports.MaterialFicha, v3 vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, motivo domain.MotivoCambio, catalogo string, cifrador ports.CifradorCambios) (ports.ReciboFicha, error) {
	material, err := validarOrden(orden, m, v3, ports.AccionRectificar)
	if err != nil {
		return ports.ReciboFicha{}, err
	}
	if cifrador == nil || !motivo.ValidoParaRectificar() {
		return ports.ReciboFicha{}, ports.ErrInvalida
	}
	var rec reciboSQL
	err = r.ejecutar(ctx, func(tx transaccion) error {
		var bruto []byte
		if err := tx.QueryRow(ctx, prepararSQL, argumentosV3(material, v3)...).Scan(&bruto); err != nil {
			return err
		}
		var e estadoSQL
		if err := decodificarEstricto(bruto, &e); err != nil {
			return err
		}
		if e.Replay != nil {
			if e.AspiranteRef != "" || e.Presentes != nil {
				return ports.ErrNoDisponible
			}
			rec = *e.Replay
			return nil
		}
		estado := ports.EstadoParaCambio{AspiranteRef: e.AspiranteRef, Version: e.Version}
		for _, p := range e.Presentes {
			if !domain.CampoFicha(p).EsContacto() {
				return ports.ErrNoDisponible
			}
			estado.Presentes = append(estado.Presentes, domain.CampoFicha(p))
		}
		valores, err := cifrador.CifrarCambios(ctx, estado)
		if err != nil {
			return err
		}
		if len(valores) == 0 {
			return ports.ErrInvalida
		}
		sql := make([]valorCambioSQL, 0, len(valores))
		for _, v := range valores {
			x := valorCambioSQL{Campo: string(v.Campo), Version: v.Version, Origen: string(v.Origen), Estado: "retirado"}
			if v.Sobre != nil {
				s := sobreACampos(*v.Sobre)
				x.Estado, x.ClaveRef, x.NonceHex, x.CifradoHex = "presente", &s.ClaveRef, &s.NonceHex, &s.CifradoHex
			}
			sql = append(sql, x)
		}
		cambios, err := json.Marshal(map[string]any{"catalogo_ref": catalogo, "motivo": string(motivo), "valores": sql})
		if err != nil {
			return ports.ErrInvalida
		}
		if err := tx.QueryRow(ctx, aplicarSQL, string(cambios)).Scan(&bruto); err != nil {
			return err
		}
		return decodificarEstricto(bruto, &rec)
	})
	if err != nil {
		return ports.ReciboFicha{}, err
	}
	return rec.recibo(ports.AccionRectificar, m.VersionEsperada+1)
}

const registrarDenegacionSQL = `SELECT vec_aspirantes.registrar_denegacion_frontera_v1($1::text,$2::text)`

var _ vecports.RegistradorAuditoriaFronteraRutaExacta = (*RegistroFichasPostgreSQL)(nil)

// RegistrarAuditoriaFronteraRutaExacta anota una denegación de la frontera
// del portal externo (000002). Solo admite la superficie de Aspirantes y
// nunca una persona: el motivo y la correlación bastan.
func (r *RegistroFichasPostgreSQL) RegistrarAuditoriaFronteraRutaExacta(ctx context.Context, orden vecports.OrdenAuditoriaFronteraRutaExacta) error {
	if orden.Validar() != nil || orden.Superficie != vecports.SuperficieAuditoriaFronteraRutaExactaAspirantes || orden.ActorRef != "" {
		return ports.ErrInvalida
	}
	return r.ejecutar(ctx, func(tx transaccion) error {
		var ref string
		return tx.QueryRow(ctx, registrarDenegacionSQL, string(orden.Motivo), orden.CorrelacionRef).Scan(&ref)
	})
}
