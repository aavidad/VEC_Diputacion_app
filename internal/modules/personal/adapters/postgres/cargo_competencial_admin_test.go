package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	efecto "vec-diputacion-granada/internal/vec/adapters/postgres/efectonominaladmin"
	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/pruebas"
)

const (
	organizacionCargoPrueba = "org_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	unidadCargoPrueba       = "unidad_admin_sintetica"
	objetoCargoAdminPrueba  = "car_AAAAAAAAAAAAAAAAAAAAAAAA"
)

func materialCargoAdminPrueba(t *testing.T, cambiar func(map[string]any)) []byte {
	t.Helper()
	m := map[string]any{"esquema": "vec.personal.cargo-competencial.publicacion.v1", "operacion": "cargo",
		"clave_idempotencia": strings.Repeat("a", 32), "objeto_ref": objetoCargoAdminPrueba,
		"organizacion_ref": organizacionCargoPrueba, "unidad_ref": unidadCargoPrueba,
		"version_esperada": 0, "huella_esperada": nil, "datos": map[string]any{"version": 1, "cargo_ref": objetoCargoAdminPrueba}}
	if cambiar != nil {
		cambiar(m)
	}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// La huella del recurso es la que Personal28 calcula en SQL: ámbitos
// organización y unidad del material y atributos SHA-256 y operación.
func TestRecursoCargoCoincideConPersonal28(t *testing.T) {
	material := materialCargoAdminPrueba(t, nil)
	accion, r, err := RecursoPublicacionCargoCompetencial(material, vd.AsignacionPerfil{})
	if err != nil || accion != "personal.cargo_competencial.publicar" || r.Referencia != objetoCargoAdminPrueba ||
		r.ModuloID != "personal" || r.Tipo != "cargo_competencial" {
		t.Fatalf("recurso distinto: %v %+v", err, r)
	}
	suma := sha256.Sum256(material)
	sha := hex.EncodeToString(suma[:])
	canon := `{"ambitos":{"organizacion_ref":"` + organizacionCargoPrueba + `","unidad_ref":"` + unidadCargoPrueba +
		`"},"atributos":{"material_sha256":"` + sha + `","operacion":"cargo"}}`
	esperada := sha256.Sum256([]byte(canon))
	if h, err := r.HuellaContextoAutorizacionSHA256(); err != nil || h != hex.EncodeToString(esperada[:]) {
		t.Fatalf("huella distinta de la de Personal28: %s", h)
	}
	for nombre, cambiar := range map[string]func(map[string]any){
		"otro_esquema":     func(m map[string]any) { m["esquema"] = "vec.personal.otro.v1" },
		"operacion":        func(m map[string]any) { m["operacion"] = "borrar" },
		"objeto_cruzado":   func(m map[string]any) { m["operacion"] = "enlace" },
		"unidad_mayuscula": func(m map[string]any) { m["unidad_ref"] = "Unidad_x" },
		"clave_de_mas":     func(m map[string]any) { m["extra"] = 1 },
		"clave_de_menos":   func(m map[string]any) { delete(m, "datos") },
		"organizacion_num": func(m map[string]any) { m["organizacion_ref"] = 7 },
	} {
		if _, _, err := RecursoPublicacionCargoCompetencial(materialCargoAdminPrueba(t, cambiar), vd.AsignacionPerfil{}); err == nil {
			t.Fatalf("%s aceptado", nombre)
		}
	}
}

type relojCargoPrueba struct{ t time.Time }

func (r relojCargoPrueba) Ahora() time.Time { return r.t }

type filaCargoPrueba struct {
	bruto string
	err   error
}

func (f filaCargoPrueba) Scan(d ...any) error {
	if f.err != nil {
		return f.err
	}
	*(d[0].(*[]byte)) = []byte(f.bruto)
	return nil
}

type txCargoPrueba struct {
	pgx.Tx
	fila               filaCargoPrueba
	args               []any
	sql                string
	commits, rollbacks int
}

func (t *txCargoPrueba) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	t.sql = sql
	t.args = make([]any, len(args))
	for i, a := range args {
		if b, ok := a.([]byte); ok {
			a = bytes.Clone(b)
		}
		t.args[i] = a
	}
	return t.fila
}
func (t *txCargoPrueba) Commit(context.Context) error   { t.commits++; return nil }
func (t *txCargoPrueba) Rollback(context.Context) error { t.rollbacks++; return nil }

type poolCargoPrueba struct {
	tx      *txCargoPrueba
	inicios int
}

func (p *poolCargoPrueba) BeginTx(_ context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	p.inicios++
	if o.IsoLevel != pgx.Serializable || o.AccessMode != pgx.ReadWrite {
		return nil, errors.New("aislamiento alterado")
	}
	return p.tx, nil
}

type emisorCargoPrueba struct {
	t        *testing.T
	alterar  string
	err      error
	llamadas int
	ahora    time.Time
}

func (e *emisorCargoPrueba) Emitir(_ context.Context, actor vd.ContextoActor, evidencia vd.EvidenciaSesionAdministracionPerfiles,
	_ vd.InstantaneaAutorizacion, material []byte, _ string) (efecto.Emision, error) {
	e.llamadas++
	if e.err != nil {
		return efecto.Emision{}, e.err
	}
	accion, recurso, err := RecursoPublicacionCargoCompetencial(material, vd.AsignacionPerfil{})
	if err != nil {
		e.t.Fatal(err)
	}
	audiencia := AudienciaPublicarCargoCompetencial
	switch e.alterar {
	case "audiencia":
		audiencia = "vec.admin.usuarios.consultar.v1"
	case "recurso":
		recurso.Atributos["operacion"] = "enlace"
	case "contexto":
		evidencia.ResultadoContexto.HuellaSHA256 = strings.Repeat("c", 64)
	case "caducada":
		e.ahora = e.ahora.Add(-time.Minute)
	}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		e.t.Fatal(err)
	}
	resumen, err := vp.NuevoResumenCapacidadAtestacionAutorizacionV3("dec_prueba", strings.Repeat("a", 64), strings.Repeat("b", 64),
		evidencia.ResultadoContexto.RegistroContextoRef, evidencia.ResultadoContexto.HuellaSHA256, accion, recurso.Referencia, huella, audiencia, e.ahora, e.ahora.Add(3*time.Second))
	if err != nil {
		e.t.Fatal(err)
	}
	raiz, _ := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	m, err := vp.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), resumen, []byte("decision"),
		[]byte("motivo"), bytes.Clone(evidencia.ResultadoContexto.RepresentacionCanonica),
		actor.Instantanea.PersonaVersion, actor.Instantanea.PerfilVersion, []byte("payload"), []byte("sobre"), []byte("evidencia"), raiz)
	if err != nil {
		e.t.Fatal(err)
	}
	return efecto.Emision{Accion: accion, Recurso: recurso, Material: m}, nil
}

type registradorCargoPrueba struct {
	llamadas int
	acepta   bool
	datos    []vd.DatosIntentoAuditoria
}

func (r *registradorCargoPrueba) AppendIntentoAuditoria(_ context.Context, o vp.OrdenIntentoAuditoria) (vp.AcuseIntentoAuditoria, error) {
	r.llamadas++
	if !r.acepta {
		return vp.AcuseIntentoAuditoria{}, errors.New("sin registro en la prueba")
	}
	d, err := o.Datos()
	if err != nil {
		return vp.AcuseIntentoAuditoria{}, err
	}
	r.datos = append(r.datos, d.Datos)
	return vp.AcuseIntentoAuditoria{AuditoriaRef: "auditoria_intento_prueba", Secuencia: 1, HuellaSHA256: strings.Repeat("a", 64),
		CorrelacionRef: d.Datos.CorrelacionRef, RegistradaEn: time.Now().UTC()}, nil
}

type revalidadorCargoPrueba struct{ valor vd.AutenticacionRevalidadaV1 }

func (r revalidadorCargoPrueba) RevalidarAutenticacionActorV1(context.Context, vd.SolicitudRevalidacionAutenticacionActorV1) (vd.AutenticacionRevalidadaV1, error) {
	return r.valor, nil
}

type contextoCargoPrueba struct {
	valor vd.ResultadoContextoActorRegistradoV2
}

func (r contextoCargoPrueba) ResolverContextoActorRegistradoV2(context.Context, vd.SolicitudContextoActor) (vd.ResultadoContextoActorRegistradoV2, error) {
	return r.valor, nil
}

// sesionADMINCargoPrueba es una sesión de la superficie de administración
// privilegiada, como la que resuelve la frontera de vec-admin.
func sesionADMINCargoPrueba(t *testing.T, ahora time.Time) (vd.ResultadoContextoActorRegistradoV2, vd.VinculoAutenticacionActorV2) {
	t.Helper()
	resultado, v, err := pruebas.NuevoContextoRegistradoYVinculoV2(ahora, "per_"+strings.Repeat("a", 22), "prf_"+strings.Repeat("b", 22),
		vd.AuthMethodCertificate, vd.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	datos, err := v.Datos()
	if err != nil {
		t.Fatal(err)
	}
	a := datos.Autenticacion()
	a.CuentaOrdinariaRef, a.CuentaPrivilegiada, a.Superficie = "cta_"+strings.Repeat("f", 24), true, vd.SuperficieAutenticacionAdministracionPrivilegiadaV1
	v, resultado, err = vd.CrearVinculoAutenticacionActorV2ConResultado(context.Background(), revalidadorCargoPrueba{a},
		vd.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: a.AutenticacionRef, SesionRef: a.SesionRef}, contextoCargoPrueba{resultado},
		vd.SolicitudContextoActor{Cuenta: vd.CuentaAutenticadaContextoActor{CuentaRef: resultado.Contexto.Instantanea.CuentaRef,
			Metodo: vd.AuthMethodCertificate, Garantia: vd.AuthAssuranceHigh}, PerfilActivoRef: resultado.Contexto.PerfilActivoRef}, relojCargoPrueba{ahora})
	if err != nil {
		t.Fatal(err)
	}
	return resultado, v
}

func escenarioCargoAdmin(t *testing.T) (*efecto.Ejecutor, *poolCargoPrueba, *emisorCargoPrueba, *registradorCargoPrueba, efecto.Solicitud) {
	t.Helper()
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	resultado, vinculo := sesionADMINCargoPrueba(t, ahora)
	pool := &poolCargoPrueba{tx: &txCargoPrueba{}}
	emisor := &emisorCargoPrueba{t: t, ahora: ahora}
	registrador := &registradorCargoPrueba{}
	motivo := vd.ReferenciaEntradaCatalogo{CatalogoID: "motivos_admin", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("d", 64), EntradaClave: "motivo_cargos"}
	e, err := efecto.NuevoEjecutor(pool, emisor, registrador, efecto.ConfiguracionAuditoria{
		Proceso: "vec-admin", MotivoDenegado: motivo, MotivoError: motivo, Plazo: time.Second}, relojCargoPrueba{ahora}, ContratoPublicacionCargoCompetencial())
	if err != nil {
		t.Fatal(err)
	}
	s := efecto.Solicitud{Actor: resultado.Contexto,
		Evidencia:      vd.EvidenciaSesionAdministracionPerfiles{ResultadoContexto: resultado, Vinculo: vinculo},
		Material:       materialCargoAdminPrueba(t, nil),
		CorrelacionRef: "correlacion_" + strings.Repeat("e", 32)}
	return e, pool, emisor, registrador, s
}

func respuestaCargo(t *testing.T, cambiar func(map[string]any)) filaCargoPrueba {
	t.Helper()
	m := map[string]any{"recibo_ref": "percar_" + strings.Repeat("1", 32), "objeto_ref": objetoCargoAdminPrueba, "version": 1,
		"huella_sha256": strings.Repeat("f", 64), "registrada_en": "2026-10-05T18:00:00.123456+00:00",
		"auditoria_ref": "aud_v3_" + strings.Repeat("2", 32), "estado_replay": "nueva"}
	if cambiar != nil {
		cambiar(m)
	}
	b, _ := json.Marshal(m)
	return filaCargoPrueba{bruto: string(b)}
}

func replayCargo(m map[string]any) {
	delete(m, "huella_sha256")
	m["estado_replay"] = "recuperada"
	m["acceso_actual_auditoria_ref"] = "aud_v3_" + strings.Repeat("3", 32)
}

// Una publicación coherente llama a Personal28 con el material exacto y los
// diez argumentos V3 en una transacción serializable y confirma. El replay
// devuelve el recibo original y la auditoría del acceso actual.
func TestEjecutorCargoConfirmaConPersonal28(t *testing.T) {
	for _, caso := range []string{"nueva", "recuperada"} {
		t.Run(caso, func(t *testing.T) {
			e, pool, emisor, registrador, s := escenarioCargoAdmin(t)
			pool.tx.fila = respuestaCargo(t, nil)
			consumo := "aud_v3_" + strings.Repeat("2", 32)
			if caso == "recuperada" {
				pool.tx.fila, consumo = respuestaCargo(t, replayCargo), "aud_v3_"+strings.Repeat("3", 32)
			}
			r, err := e.Aplicar(t.Context(), s)
			if err != nil {
				t.Fatalf("publicación coherente rechazada: %v", err)
			}
			tx := pool.tx
			if emisor.llamadas != 1 || pool.inicios != 1 || tx.commits != 1 || registrador.llamadas != 0 || tx.sql != publicarCargoCompetencialSQL ||
				len(tx.args) != 11 || !bytes.Equal(tx.args[0].([]byte), s.Material) || r.ConsumoAuditoriaRef != consumo {
				t.Fatalf("efecto o recibo distintos: %+v", r)
			}
			var cuerpo map[string]any
			if json.Unmarshal(r.Cuerpo, &cuerpo) != nil || cuerpo["estado_replay"] != caso || cuerpo["objeto_ref"] != objetoCargoAdminPrueba ||
				cuerpo["acceso_actual_auditoria_ref"] != nil || cuerpo["auditoria_ref"] != "aud_v3_"+strings.Repeat("2", 32) {
				t.Fatalf("cuerpo distinto: %s", r.Cuerpo)
			}
		})
	}
}

// Ninguna incoherencia confirma: emisión ajena o caducada, respuesta que no es
// la de Personal28, o SQL rechazado. Sin sesión ADMIN válida para auditar, la
// respuesta es indisponibilidad y nunca un permiso o un 403.
func TestEjecutorCargoNoConfirmaIncoherencias(t *testing.T) {
	for _, caso := range []string{"audiencia", "recurso", "contexto", "caducada", "emisor_denegado", "sql_denegado", "sql_conflicto",
		"otro_objeto", "recibo_mal", "auditoria_mal", "campo_de_mas", "nueva_sin_huella", "replay_con_huella", "replay_mismo_acceso",
		"estado_desconocido", "material_ilegible"} {
		t.Run(caso, func(t *testing.T) {
			e, pool, emisor, _, s := escenarioCargoAdmin(t)
			pool.tx.fila = respuestaCargo(t, nil)
			switch caso {
			case "audiencia", "recurso", "contexto", "caducada":
				emisor.alterar = caso
			case "emisor_denegado":
				emisor.err = vd.ErrAutorizacionDenegada
			case "sql_denegado":
				pool.tx.fila = filaCargoPrueba{err: &pgconn.PgError{Code: "42501"}}
			case "sql_conflicto":
				pool.tx.fila = filaCargoPrueba{err: &pgconn.PgError{Code: "40001"}}
			case "otro_objeto":
				pool.tx.fila = respuestaCargo(t, func(m map[string]any) { m["objeto_ref"] = "car_BBBBBBBBBBBBBBBBBBBBBBBB" })
			case "recibo_mal":
				pool.tx.fila = respuestaCargo(t, func(m map[string]any) { m["recibo_ref"] = "recibo:x" })
			case "auditoria_mal":
				pool.tx.fila = respuestaCargo(t, func(m map[string]any) { m["auditoria_ref"] = "aud-x" })
			case "campo_de_mas":
				pool.tx.fila = respuestaCargo(t, func(m map[string]any) { m["datos"] = "x" })
			case "nueva_sin_huella":
				pool.tx.fila = respuestaCargo(t, func(m map[string]any) { delete(m, "huella_sha256") })
			case "replay_con_huella":
				pool.tx.fila = respuestaCargo(t, func(m map[string]any) {
					replayCargo(m)
					m["huella_sha256"] = strings.Repeat("f", 64)
				})
			case "replay_mismo_acceso":
				pool.tx.fila = respuestaCargo(t, func(m map[string]any) {
					replayCargo(m)
					m["acceso_actual_auditoria_ref"] = m["auditoria_ref"]
				})
			case "estado_desconocido":
				pool.tx.fila = respuestaCargo(t, func(m map[string]any) { m["estado_replay"] = "otra" })
			case "material_ilegible":
				s.Material = []byte(`{"esquema":`)
			}
			_, err := e.Aplicar(t.Context(), s)
			if !errors.Is(err, efecto.ErrNoDisponible) || pool.tx.commits != 0 {
				t.Fatalf("%s confirmado o sin cerrar: %v", caso, err)
			}
			if (caso == "audiencia" || caso == "recurso" || caso == "contexto" || caso == "caducada" || caso == "emisor_denegado") && pool.inicios != 0 {
				t.Fatal("emisión ajena alcanzó la base")
			}
			if caso == "material_ilegible" && (emisor.llamadas != 0 || pool.inicios != 0) {
				t.Fatal("material ilegible alcanzó el PDP o la base")
			}
		})
	}
}

// Con el registro común disponible, cada fallo deja su intento con la clase
// correcta y una referencia de recurso que la auditoría admite (las de
// Personal llevan mayúsculas), y el error conserva su clase para el 403/400.
func TestEjecutorCargoDejaIntentoConClase(t *testing.T) {
	for caso, x := range map[string]struct {
		fila      filaCargoPrueba
		emisorErr error
		material  []byte
		clase     vd.ResultadoIntentoAuditoria
		err       error
	}{
		"sql_denegado":    {filaCargoPrueba{err: &pgconn.PgError{Code: "42501"}}, nil, nil, vd.ResultadoIntentoAuditoriaDenegado, vd.ErrAutorizacionDenegada},
		"sql_conflicto":   {filaCargoPrueba{err: &pgconn.PgError{Code: "40001"}}, nil, nil, vd.ResultadoIntentoAuditoriaError, efecto.ErrConflicto},
		"sql_invalido":    {filaCargoPrueba{err: &pgconn.PgError{Code: "22023"}}, nil, nil, vd.ResultadoIntentoAuditoriaDenegado, vd.ErrActoAdministracionPerfilesInvalido},
		"sql_caido":       {filaCargoPrueba{err: &pgconn.PgError{Code: "2201B"}}, nil, nil, vd.ResultadoIntentoAuditoriaError, efecto.ErrNoDisponible},
		"emisor_denegado": {filaCargoPrueba{}, vd.ErrAutorizacionDenegada, nil, vd.ResultadoIntentoAuditoriaDenegado, vd.ErrAutorizacionDenegada},
		"ilegible":        {filaCargoPrueba{}, nil, []byte(`{"x":1}`), vd.ResultadoIntentoAuditoriaDenegado, vd.ErrActoAdministracionPerfilesInvalido},
	} {
		t.Run(caso, func(t *testing.T) {
			e, pool, emisor, registrador, s := escenarioCargoAdmin(t)
			registrador.acepta, pool.tx.fila, emisor.err = true, x.fila, x.emisorErr
			if x.material != nil {
				s.Material = x.material
			}
			_, err := e.Aplicar(t.Context(), s)
			if !errors.Is(err, x.err) || pool.tx.commits != 0 || len(registrador.datos) != 1 {
				t.Fatalf("error o intento distintos: %v (%d intentos)", err, len(registrador.datos))
			}
			d := registrador.datos[0]
			if d.Resultado != x.clase || d.ModuloID != "personal" || d.FinalidadRef != "administrar_cargos_competenciales" ||
				d.Accion != "personal.cargo_competencial.publicar" || strings.ToLower(d.RecursoRef) != d.RecursoRef ||
				strings.Contains(d.RecursoRef, objetoCargoAdminPrueba) {
				t.Fatalf("intento distinto: %+v", d)
			}
		})
	}
}
