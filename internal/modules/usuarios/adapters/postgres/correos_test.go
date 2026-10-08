package postgres

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/usuarios/canonico"
	"vec-diputacion-granada/internal/modules/usuarios/domain"
	"vec-diputacion-granada/internal/modules/usuarios/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type proveedorCorreoPGPrueba struct {
	v3       vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	llamadas int
}

func (p *proveedorCorreoPGPrueba) ProveerMaterialCorreos(context.Context, vecdomain.VinculoAutenticacionActorV2, ports.MaterialCorreos) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	p.llamadas++
	return p.v3, nil
}

type descifradorCorreoPGPrueba struct {
	llamados     int
	persona, ref string
	version      uint64
}

func (d *descifradorCorreoPGPrueba) ConDireccionCorreoDescifrada(_ context.Context, persona string, sobre ports.SobreDireccionCorreo, usar func([]byte) error) error {
	d.llamados++
	d.persona, d.ref, d.version = persona, sobre.CorreoRef, sobre.Version
	return usar([]byte("ejemplo@example.org"))
}

type llamadaCorreoPG struct {
	sql  string
	args []any
}
type filaCorreoPGPrueba struct {
	valor any
	err   error
}

func (f filaCorreoPGPrueba) Scan(dest ...any) error {
	if f.err != nil {
		return f.err
	}
	switch p := dest[0].(type) {
	case *bool:
		*p = f.valor.(bool)
	case *[]byte:
		if f.valor == nil {
			*p = nil
			return nil
		}
		*p = append([]byte(nil), f.valor.([]byte)...)
	}
	return nil
}

type txCorreoPGPrueba struct {
	respuestas         []filaCorreoPGPrueba
	llamadas           []llamadaCorreoPG
	commits, rollbacks int
	errExec            error
}

func (t *txCorreoPGPrueba) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	t.llamadas = append(t.llamadas, llamadaCorreoPG{sql, args})
	return pgconn.CommandTag{}, t.errExec
}
func (t *txCorreoPGPrueba) QueryRow(_ context.Context, sql string, args ...any) filaCorreos {
	t.llamadas = append(t.llamadas, llamadaCorreoPG{sql, args})
	if len(t.respuestas) == 0 {
		return filaCorreoPGPrueba{err: errors.New("sin respuesta")}
	}
	f := t.respuestas[0]
	t.respuestas = t.respuestas[1:]
	return f
}
func (t *txCorreoPGPrueba) Commit(context.Context) error   { t.commits++; return nil }
func (t *txCorreoPGPrueba) Rollback(context.Context) error { t.rollbacks++; return nil }

func TestAjustesUsuariosEnUnViajeYAntesDeAcreditar(t *testing.T) {
	casos := []struct {
		nombre    string
		ajustes   string
		lock      string
		statement string
		abrir     func(*txCorreoPGPrueba) (transaccionCorreos, error)
	}{
		{"imagen", ajustesTransaccionImagenSQL, "3s", "15s", func(tx *txCorreoPGPrueba) (transaccionCorreos, error) {
			return (&RegistroImagenPostgreSQL{iniciar: func(context.Context) (transaccionCorreos, error) { return tx, nil }, superficie: vecdomain.SuperficieAutenticacionInternaCorporativaV1}).abrir(context.Background())
		}},
		{"correos", ajustesTransaccionCorreosSQL, "3s", "15s", func(tx *txCorreoPGPrueba) (transaccionCorreos, error) {
			return pruebaRegistroCorreos(tx).abrir(context.Background())
		}},
		{"avisos", ajustesTransaccionCorreoAvisosSQL, "2s", "5s", func(tx *txCorreoPGPrueba) (transaccionCorreos, error) {
			return registroAvisosPGPrueba(tx).abrir(context.Background())
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			valores := []string{"'search_path','pg_catalog',true", "'row_security','on',true", "'timezone','UTC',true", "'lock_timeout','" + caso.lock + "',true", "'statement_timeout','" + caso.statement + "',true", "'idle_in_transaction_session_timeout','20s',true"}
			anterior := -1
			for _, valor := range valores {
				indice := strings.Index(caso.ajustes, "pg_catalog.set_config("+valor+")")
				if indice <= anterior {
					t.Fatalf("ajustes ausentes o fuera de orden: %s", valor)
				}
				anterior = indice
			}
			if strings.Count(caso.ajustes, "pg_catalog.set_config(") != 6 {
				t.Fatal("se esperan exactamente seis ajustes")
			}
			tx := &txCorreoPGPrueba{respuestas: []filaCorreoPGPrueba{{valor: false}}}
			if _, err := caso.abrir(tx); err == nil || len(tx.llamadas) != 2 || tx.llamadas[0].sql != caso.ajustes || !strings.Contains(tx.llamadas[1].sql, "session_user=current_user") || tx.rollbacks == 0 {
				t.Fatalf("acreditación falsa: err=%v llamadas=%v rollback=%d", err, tx.llamadas, tx.rollbacks)
			}
			tx = &txCorreoPGPrueba{errExec: errors.New("ajuste rechazado")}
			if _, err := caso.abrir(tx); err == nil || len(tx.llamadas) != 1 || tx.llamadas[0].sql != caso.ajustes || tx.rollbacks == 0 {
				t.Fatalf("fallo de ajustes: err=%v llamadas=%v rollback=%d", err, tx.llamadas, tx.rollbacks)
			}
		})
	}
}

type revalidadorCorreoPGPrueba struct {
	auth vecdomain.AutenticacionRevalidadaV1
}

func (r revalidadorCorreoPGPrueba) RevalidarAutenticacionActorV1(context.Context, vecdomain.SolicitudRevalidacionAutenticacionActorV1) (vecdomain.AutenticacionRevalidadaV1, error) {
	return r.auth, nil
}

type resolutorCorreoPGPrueba struct {
	res vecdomain.ResultadoContextoActorRegistradoV2
}

func (r resolutorCorreoPGPrueba) ResolverContextoActorRegistradoV2(context.Context, vecdomain.SolicitudContextoActor) (vecdomain.ResultadoContextoActorRegistradoV2, error) {
	return r.res, nil
}

type relojCorreoPGPrueba struct{ ahora time.Time }

func (r relojCorreoPGPrueba) Ahora() time.Time { return r.ahora }

func identidadCorreoPGPrueba(t *testing.T) (vecdomain.ContextoActor, vecdomain.VinculoAutenticacionActorV2) {
	t.Helper()
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	z := strings.Repeat("a", 24)
	cuenta := vecdomain.CuentaAutenticadaContextoActor{CuentaRef: "cta_" + z, Metodo: vecdomain.AuthMethodCertificate, Garantia: vecdomain.AuthAssuranceHigh}
	snap := vecdomain.InstantaneaContextoActor{VinculoRef: "vca_" + z, VinculoVersion: 1, CuentaRef: cuenta.CuentaRef, CuentaVersion: 1, PersonaRef: "per_" + strings.Repeat("r", 24), PersonaVersion: 1, PerfilActivoRef: "prf_" + strings.Repeat("p", 24), PerfilVersion: 1, Estado: vecdomain.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour)}
	actor, err := vecdomain.NuevoContextoActor(cuenta, snap, ahora)
	if err != nil {
		t.Fatal(err)
	}
	canon, _ := actor.RepresentacionCanonicaVinculadaV2()
	huella, _ := actor.HuellaSHA256VinculadaV2()
	ac := vecdomain.AcreditacionProcedenciaComponenteContextoActorV1{ProcedenciaRef: "prc_" + z, ProcedenciaVersion: 1, ProcedenciaHuellaSHA256: strings.Repeat("a", 64), ProcedenciaAutoridad: vecdomain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1}
	man := vecdomain.ManifiestoProcedenciaContextoActorV1{Esquema: vecdomain.EsquemaManifiestoProcedenciaContextoActorV1, AutoridadEfectiva: vecdomain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1, Cuenta: vecdomain.ProcedenciaCuentaContextoActorV1{CuentaRef: cuenta.CuentaRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: ac}, Persona: vecdomain.ProcedenciaPersonaContextoActorV1{PersonaRef: actor.PersonaRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: ac}, Perfil: vecdomain.ProcedenciaPerfilContextoActorV1{PerfilRef: actor.PerfilActivoRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: ac}, Contexto: vecdomain.ProcedenciaVinculoContextoActorV1{VinculoRef: snap.VinculoRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: ac}, Vinculos: []vecdomain.ProcedenciaVinculoReferenciaContextoActorV1{}}
	bm, _ := man.RepresentacionCanonicaV1()
	hm, _ := vecdomain.HuellaSHA256ManifiestoProcedenciaContextoActorV1(bm)
	res := vecdomain.ResultadoContextoActorRegistradoV2{RegistroContextoRef: "rca_" + z, Contexto: actor, RepresentacionCanonica: canon, HuellaSHA256: huella, ManifiestoProcedenciaCanonico: bm, ManifiestoProcedenciaHuellaSHA256: hm, AutoridadEfectiva: vecdomain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1, ResueltoEnAutoritativo: ahora}
	if err := res.Validar(); err != nil {
		t.Fatal(err)
	}
	auth := vecdomain.AutenticacionRevalidadaV1{AutenticacionRef: "aut_" + z, AutenticacionHuellaSHA256: strings.Repeat("a", 64), AsercionRef: "ase_" + z, SesionRef: "ses_" + z, ControlSesionRef: "cse_" + z, ControlSesionRevision: 1, ControlSesionHuellaSHA256: strings.Repeat("b", 64), CuentaRef: cuenta.CuentaRef, CuentaOrdinariaRef: cuenta.CuentaRef, Superficie: vecdomain.SuperficieAutenticacionInternaCorporativaV1, MetodoObservado: vecdomain.AuthMethodCertificate, GarantiaObservada: vecdomain.AuthAssuranceHigh, PoliticaGarantiaRef: "pga_" + z, PoliticaGarantiaHuellaSHA256: strings.Repeat("c", 64), AutenticacionVerificadaEn: ahora.Add(-time.Minute), SesionEmitidaEn: ahora.Add(-time.Minute), SesionRevalidadaEn: ahora.Add(-time.Second), SesionValidaHasta: ahora.Add(time.Minute)}
	if err := auth.Validar(); err != nil {
		t.Fatal(err)
	}
	v, err := vecdomain.CrearVinculoAutenticacionActorV2(context.Background(), revalidadorCorreoPGPrueba{auth}, vecdomain.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: auth.AutenticacionRef, SesionRef: auth.SesionRef}, resolutorCorreoPGPrueba{res}, vecdomain.SolicitudContextoActor{Cuenta: cuenta, PerfilActivoRef: actor.PerfilActivoRef}, relojCorreoPGPrueba{ahora})
	if err != nil {
		t.Fatal(err)
	}
	return actor, v
}

const refCorreoPG = "correo:0123456789abcdef0123456789abcdef"

func pruebaOrdenYV3Correos(t *testing.T, accion string) (ports.OrdenCorreos, ports.MaterialCorreos, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) {
	t.Helper()
	actor, vinculo := identidadCorreoPGPrueba(t)
	superficie := vecdomain.SuperficieAutenticacionInternaCorporativaV1
	m := ports.MaterialCorreos{Superficie: superficie, PersonaRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, Accion: accion, FinalidadRef: ports.FinalidadCorreosPropios, VersionEsperada: 1, ClaveOperacion: "clave-operacion-123", CorreoRef: refCorreoPG, HuellasPeticion: ports.HuellasSemanticasCorreo{Activa: ports.HuellaSemanticaCorreo{ClaveRef: "hmac:v2", Valor: strings.Repeat("a", 64)}, Retenidas: []ports.HuellaSemanticaCorreo{{ClaveRef: "hmac:v1", Valor: strings.Repeat("b", 64)}}}}
	if accion == ports.AccionConsultarCorreos {
		m.VersionEsperada, m.ClaveOperacion, m.CorreoRef, m.HuellasPeticion = 0, "", "", ports.HuellasSemanticasCorreo{}
	}
	if accion == ports.AccionAnadirCorreo {
		m.CorreoRef = ""
	}
	recurso, err := canonico.RecursoCorreos(m)
	if err != nil {
		t.Fatal(err)
	}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	identidad, err := domain.NuevaIdentidadCorreos(actor, vinculo, superficie)
	if err != nil {
		t.Fatal(err)
	}
	orden := ports.OrdenCorreos{Identidad: identidad, Proveedor: &proveedorCorreoPGPrueba{}}
	audiencia, err := canonico.AudienciaCorreos(accion, superficie)
	if err != nil {
		t.Fatal(err)
	}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("dec_prueba", strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_prueba", strings.Repeat("c", 64), accion, actor.PersonaRef, huella, audiencia, ahora, ahora.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	raiz, err := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	if err != nil {
		t.Fatal(err)
	}
	canon, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	v3, err := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), resumen, []byte("decision"), []byte("motivo"), canon, 1, 1, []byte("payload"), []byte("sobre"), []byte("evidencia"), raiz)
	if err != nil {
		t.Fatal(err)
	}
	orden.Proveedor.(*proveedorCorreoPGPrueba).v3 = v3
	return orden, m, v3
}

func pruebaRegistroCorreos(tx *txCorreoPGPrueba) *RegistroCorreosPostgreSQL {
	sentencias, _ := sentenciasCorreosSuperficie(vecdomain.SuperficieAutenticacionInternaCorporativaV1)
	return &RegistroCorreosPostgreSQL{iniciar: func(context.Context) (transaccionCorreos, error) { return tx, nil }, descifrador: &descifradorCorreoPGPrueba{}, superficie: vecdomain.SuperficieAutenticacionInternaCorporativaV1, sql: sentencias}
}

// Cada superficie llama solo al esquema de su población y la acreditación
// exige que su LOGIN no alcance el de la otra.
func TestSentenciasCorreosPorPoblacion(t *testing.T) {
	interna, ok := sentenciasCorreosSuperficie(vecdomain.SuperficieAutenticacionInternaCorporativaV1)
	if !ok {
		t.Fatal("superficie interna rechazada")
	}
	externa, ok := sentenciasCorreosSuperficie(vecdomain.SuperficieAutenticacionExternaPersonalV1)
	if !ok {
		t.Fatal("superficie externa rechazada")
	}
	if _, ok := sentenciasCorreosSuperficie(vecdomain.SuperficieAutenticacionActorV1("otra")); ok {
		t.Fatal("superficie desconocida admitida")
	}
	casos := []struct {
		s             sentenciasCorreos
		propio, ajeno string
	}{{interna, "vec_usuarios_correos_interno.", "vec_usuarios_correos_externo."}, {externa, "vec_usuarios_correos_externo.", "vec_usuarios_correos_interno."}}
	for _, c := range casos {
		for _, sql := range []string{c.s.consultar, c.s.recuperar, c.s.aplicar, c.s.preparar, c.s.cerrar, c.s.confirmar} {
			if !strings.HasPrefix(sql, "SELECT "+c.propio) || strings.Contains(sql, c.ajeno) || strings.Contains(sql, "vec_usuarios.") || strings.Contains(sql, "@") {
				t.Fatalf("sentencia fuera de su población: %s", sql)
			}
		}
		ajeno := strings.TrimSuffix(c.ajeno, ".")
		if strings.Contains(c.s.acreditar, "@") || strings.Contains(c.s.acreditar, "vec_usuarios.") ||
			strings.Count(c.s.acreditar, "'"+c.propio) != 7 ||
			!strings.Contains(c.s.acreditar, "NOT pg_catalog.has_schema_privilege(session_user,'"+ajeno+"','USAGE')") {
			t.Fatalf("acreditación incompleta: %s", c.s.acreditar)
		}
	}
}

func fechaPG() string {
	return time.Now().UTC().Truncate(time.Microsecond).Format("2006-01-02T15:04:05.999999-07:00")
}

func sobrePG() map[string]any {
	return map[string]any{"version": 1, "clave_ref": "clave:cifrado", "nonce_hex": strings.Repeat("aa", 12), "cifrado_hex": strings.Repeat("bb", 32)}
}

func reciboPGPrueba(m ports.MaterialCorreos, ref string, replay bool, envios []map[string]any) []byte {
	if envios == nil {
		envios = []map[string]any{}
	}
	b, _ := json.Marshal(map[string]any{"recibo_ref": "correo_recibo:" + strings.Repeat("1", 32), "persona_ref": m.PersonaRef, "accion": m.Accion, "correo_ref": ref, "version": m.VersionEsperada + 1, "fecha_utc": fechaPG(), "replay": replay, "envios": envios})
	return b
}

func envioPG(tipo, ref, desafio string) map[string]any {
	e := map[string]any{"envio_ref": "correo_envio:" + strings.Repeat("2", 32), "reserva_ref": "reserva:" + strings.Repeat("3", 32), "tipo": tipo, "correo_ref": ref, "desafio_ref": nil, "sobre": sobrePG()}
	if desafio != "" {
		e["desafio_ref"] = desafio
	}
	return e
}

func respuestasApertura(extra ...filaCorreoPGPrueba) []filaCorreoPGPrueba {
	return append([]filaCorreoPGPrueba{{valor: true}}, extra...)
}

func TestMaterialCorreosSerializaHuellasSinClaros(t *testing.T) {
	_, m, _ := pruebaOrdenYV3Correos(t, ports.AccionVerificarCorreo)
	b, err := canonico.SerializarMaterialCorreos(m)
	if err != nil || bytes.Contains(b, []byte("@")) || bytes.Contains(b, []byte("codigo")) || !bytes.Contains(b, []byte(`"retenidas":[{"clave_ref":"hmac:v1"`)) {
		t.Fatalf("material: %s %v", b, err)
	}
}

func TestAplicarAltaDevuelveEnvioReservadoYNoFiltraClaros(t *testing.T) {
	orden, m, v3 := pruebaOrdenYV3Correos(t, ports.AccionAnadirCorreo)
	desafio := "desafio:" + strings.Repeat("4", 32)
	tx := &txCorreoPGPrueba{respuestas: respuestasApertura(filaCorreoPGPrueba{valor: reciboPGPrueba(m, refCorreoPG, false, []map[string]any{envioPG(ports.TipoEnvioCodigo, refCorreoPG, desafio)})})}
	r := pruebaRegistroCorreos(tx)
	p := ports.PeticionCorreo{ClaveOperacion: m.ClaveOperacion, VersionEsperada: m.VersionEsperada, CorreoRef: refCorreoPG}
	sobre := ports.SobreDireccionCorreo{CorreoRef: refCorreoPG, Version: 2, ClaveRef: "clave:cifrado", ClaveIgualdadRef: "clave:igualdad", Nonce: bytes.Repeat([]byte{1}, 12), Cifrado: bytes.Repeat([]byte{2}, 30), HuellaIgualdad: bytes.Repeat([]byte{3}, 32)}
	reserva := ports.ReservaDesafio{DesafioRef: desafio, HuellaCodigo: bytes.Repeat([]byte{4}, 32), ClaveRef: "clave:codigo", VenceUTC: time.Now().UTC().Add(time.Hour).Truncate(time.Microsecond)}
	resultado, err := r.Aplicar(context.Background(), orden, p, m, v3, sobre, reserva, nil)
	if err != nil || resultado.Recibo.CorreoRef != refCorreoPG || len(resultado.Envios) != 1 || resultado.Envios[0].DesafioRef != desafio || len(resultado.Envios[0].Sobre.Nonce) != 12 || tx.commits != 1 {
		t.Fatalf("alta: %+v %v", resultado, err)
	}
	llamada := tx.llamadas[len(tx.llamadas)-1]
	if llamada.sql != pruebaRegistroCorreos(nil).sql.aplicar {
		t.Fatal("no se llamó a la fachada de aplicar")
	}
	for _, arg := range llamada.args {
		if b, ok := arg.([]byte); ok && (bytes.Contains(b, []byte("@")) || bytes.Contains(b, []byte(`"codigo"`))) {
			t.Fatal("dirección o código en claro hacia SQL")
		}
	}
	reserva.Codigo = "12345678"
	if _, err := r.Aplicar(context.Background(), orden, p, m, v3, sobre, reserva, nil); !errors.Is(err, ports.ErrCorreosInvalidos) {
		t.Fatalf("código hacia SQL aceptado: %v", err)
	}
}

func TestAplicarRechazaRespuestasSQLIncoherentes(t *testing.T) {
	orden, m, v3 := pruebaOrdenYV3Correos(t, ports.AccionActivarCorreo)
	p := ports.PeticionCorreo{ClaveOperacion: m.ClaveOperacion, VersionEsperada: m.VersionEsperada, CorreoRef: refCorreoPG}
	for nombre, bruto := range map[string][]byte{
		"otra_referencia": reciboPGPrueba(m, "correo:"+strings.Repeat("9", 32), false, nil),
		"aviso_mismo":     reciboPGPrueba(m, refCorreoPG, false, []map[string]any{envioPG(ports.TipoEnvioAviso, refCorreoPG, "")}),
		"replay_envio":    reciboPGPrueba(m, refCorreoPG, true, []map[string]any{envioPG(ports.TipoEnvioAviso, "correo:"+strings.Repeat("9", 32), "")}),
		"campo_extra":     []byte(`{"recibo_ref":"x","extra":1}`),
	} {
		tx := &txCorreoPGPrueba{respuestas: respuestasApertura(filaCorreoPGPrueba{valor: bruto})}
		if _, err := pruebaRegistroCorreos(tx).Aplicar(context.Background(), orden, p, m, v3, ports.SobreDireccionCorreo{}, ports.ReservaDesafio{}, nil); !errors.Is(err, ports.ErrCorreosNoDisponible) || tx.commits != 0 {
			t.Fatalf("%s aceptado: %v", nombre, err)
		}
	}
	tx := &txCorreoPGPrueba{respuestas: respuestasApertura(filaCorreoPGPrueba{valor: reciboPGPrueba(m, refCorreoPG, false, []map[string]any{envioPG(ports.TipoEnvioAviso, "correo:"+strings.Repeat("9", 32), "")})})}
	resultado, err := pruebaRegistroCorreos(tx).Aplicar(context.Background(), orden, p, m, v3, ports.SobreDireccionCorreo{}, ports.ReservaDesafio{}, nil)
	if err != nil || len(resultado.Envios) != 1 || resultado.Envios[0].Tipo != ports.TipoEnvioAviso {
		t.Fatalf("activar con aviso: %+v %v", resultado, err)
	}
}

type comprobadorPGPrueba struct {
	valido bool
	meta   ports.MetadatosDesafioCorreo
}

func (c *comprobadorPGPrueba) Comprobar(_ context.Context, m ports.MetadatosDesafioCorreo) (bool, error) {
	c.meta = m
	return c.valido, nil
}

func metaVerificacionPG(m ports.MaterialCorreos) []byte {
	b, _ := json.Marshal(map[string]any{"persona_ref": m.PersonaRef, "correo_ref": m.CorreoRef, "desafio_ref": "desafio:" + strings.Repeat("5", 32), "huella_codigo_hex": strings.Repeat("ab", 32), "clave_ref": "clave:codigo", "vence_utc": fechaPG(), "intentos": 1})
	return b
}

func TestVerificarComparaFueraDeSQLYConfirmaIntento(t *testing.T) {
	orden, m, v3 := pruebaOrdenYV3Correos(t, ports.AccionVerificarCorreo)
	p := ports.PeticionCorreo{ClaveOperacion: m.ClaveOperacion, VersionEsperada: m.VersionEsperada, CorreoRef: refCorreoPG}
	comprobador := &comprobadorPGPrueba{}
	tx := &txCorreoPGPrueba{respuestas: respuestasApertura(filaCorreoPGPrueba{valor: metaVerificacionPG(m)}, filaCorreoPGPrueba{valor: []byte(`{"valido":false,"intentos_restantes":3}`)})}
	_, err := pruebaRegistroCorreos(tx).Aplicar(context.Background(), orden, p, m, v3, ports.SobreDireccionCorreo{}, ports.ReservaDesafio{}, comprobador)
	var incorrecto ports.CodigoIncorrecto
	if !errors.As(err, &incorrecto) || incorrecto.IntentosRestantes != 3 || tx.commits != 1 || comprobador.meta.DesafioRef == "" || len(comprobador.meta.HuellaCodigo) != 32 {
		t.Fatalf("intento fallido no confirmado: %v commits=%d", err, tx.commits)
	}
	cerrar := tx.llamadas[len(tx.llamadas)-1]
	if cerrar.sql != pruebaRegistroCorreos(nil).sql.cerrar || cerrar.args[2] != false {
		t.Fatal("cierre sin resultado de la comparación")
	}
	comprobador.valido = true
	tx = &txCorreoPGPrueba{respuestas: respuestasApertura(filaCorreoPGPrueba{valor: metaVerificacionPG(m)}, filaCorreoPGPrueba{valor: reciboPGPrueba(m, refCorreoPG, false, nil)})}
	resultado, err := pruebaRegistroCorreos(tx).Aplicar(context.Background(), orden, p, m, v3, ports.SobreDireccionCorreo{}, ports.ReservaDesafio{}, comprobador)
	if err != nil || resultado.Recibo.Version != m.VersionEsperada+1 || tx.commits != 1 {
		t.Fatalf("verificación: %v", err)
	}
	replay, _ := json.Marshal(map[string]any{"replay": map[string]any{"resultado": "codigo_invalido", "persona_ref": m.PersonaRef, "correo_ref": m.CorreoRef, "replay": true}})
	tx = &txCorreoPGPrueba{respuestas: respuestasApertura(filaCorreoPGPrueba{valor: replay})}
	comprobador.meta = ports.MetadatosDesafioCorreo{}
	_, err = pruebaRegistroCorreos(tx).Aplicar(context.Background(), orden, p, m, v3, ports.SobreDireccionCorreo{}, ports.ReservaDesafio{}, comprobador)
	if !errors.Is(err, ports.ErrCorreosCodigoIncorrecto) || comprobador.meta.DesafioRef != "" || tx.commits != 1 {
		t.Fatalf("replay de intento fallido volvió a comparar: %v", err)
	}
}

func TestErroresSQLSeTraducenSinDetalle(t *testing.T) {
	for codigo, esperado := range map[string]error{"42501": ports.ErrCorreosProhibido, "P1409": ports.ErrCorreosConflicto, "P1410": ports.ErrCorreosCodigoCaducado, "P1411": ports.ErrCorreosYaRegistrado, "P1412": ports.ErrCorreosMaximo, "P1413": ports.ErrCorreosEnUso, "P1429": ports.ErrCorreosLimite, "22023": ports.ErrCorreosInvalidos, "40001": ports.ErrCorreosNoDisponible, "XX000": ports.ErrCorreosNoDisponible} {
		err := errorCorreosSeguro(context.Background(), &pgconn.PgError{Code: codigo, Message: "persona@example.org"})
		if !errors.Is(err, esperado) || strings.Contains(err.Error(), "@") {
			t.Fatalf("%s -> %v", codigo, err)
		}
	}
}

func TestConsultarDescifraEnMemoriaYValidaConjunto(t *testing.T) {
	orden, m, v3 := pruebaOrdenYV3Correos(t, ports.AccionConsultarCorreos)
	vista, _ := json.Marshal(map[string]any{"persona_ref": m.PersonaRef, "version": 3, "correos": []map[string]any{{"correo_ref": refCorreoPG, "estado": "pendiente", "activo": false, "creado_utc": fechaPG(), "verificado_utc": nil, "sobre": sobrePG(), "codigo": map[string]any{"vence_utc": fechaPG(), "intentos_restantes": 5}}}})
	tx := &txCorreoPGPrueba{respuestas: respuestasApertura(filaCorreoPGPrueba{valor: vista})}
	r := pruebaRegistroCorreos(tx)
	resultado, err := r.ConsultarPropios(context.Background(), orden, m, v3)
	if err != nil || len(resultado.Correos) != 1 || resultado.Correos[0].Direccion != "ejemplo@example.org" || resultado.Correos[0].Codigo == nil || tx.commits != 1 {
		t.Fatalf("consulta: %+v %v", resultado, err)
	}
	activoPendiente, _ := json.Marshal(map[string]any{"persona_ref": m.PersonaRef, "version": 3, "correos": []map[string]any{{"correo_ref": refCorreoPG, "estado": "pendiente", "activo": true, "creado_utc": fechaPG(), "verificado_utc": nil, "sobre": sobrePG(), "codigo": nil}}})
	tx = &txCorreoPGPrueba{respuestas: respuestasApertura(filaCorreoPGPrueba{valor: activoPendiente})}
	if _, err := pruebaRegistroCorreos(tx).ConsultarPropios(context.Background(), orden, m, v3); !errors.Is(err, ports.ErrCorreosNoDisponible) || tx.commits != 0 {
		t.Fatalf("pendiente activo aceptado: %v", err)
	}
}

func TestConfirmarEnvioExigeReservaYSuperficie(t *testing.T) {
	orden, _, _ := pruebaOrdenYV3Correos(t, ports.AccionAnadirCorreo)
	envio := ports.EnvioPendiente{EnvioRef: "correo_envio:" + strings.Repeat("2", 32), ReservaRef: "reserva:" + strings.Repeat("3", 32)}
	tx := &txCorreoPGPrueba{respuestas: respuestasApertura(filaCorreoPGPrueba{valor: true})}
	if err := pruebaRegistroCorreos(tx).ConfirmarEnvio(context.Background(), orden, envio, true); err != nil || tx.commits != 1 || tx.llamadas[len(tx.llamadas)-1].args[3] != true {
		t.Fatalf("confirmación: %v", err)
	}
	envio.ReservaRef = "reserva:corta"
	tx = &txCorreoPGPrueba{}
	if err := pruebaRegistroCorreos(tx).ConfirmarEnvio(context.Background(), orden, envio, true); !errors.Is(err, ports.ErrCorreosInvalidos) || len(tx.llamadas) != 0 {
		t.Fatalf("reserva mal formada llegó a SQL: %v", err)
	}
	externo := pruebaRegistroCorreos(&txCorreoPGPrueba{})
	externo.superficie = vecdomain.SuperficieAutenticacionExternaPersonalV1
	envio.ReservaRef = "reserva:" + strings.Repeat("3", 32)
	if err := externo.ConfirmarEnvio(context.Background(), orden, envio, true); !errors.Is(err, ports.ErrCorreosInvalidos) {
		t.Fatalf("confirmación desde otra superficie: %v", err)
	}
}

func TestRecuperarSinOperacionYLimitesDeTransaccion(t *testing.T) {
	orden, m, v3 := pruebaOrdenYV3Correos(t, ports.AccionActivarCorreo)
	tx := &txCorreoPGPrueba{respuestas: respuestasApertura(filaCorreoPGPrueba{valor: nil})}
	recibo, existe, err := pruebaRegistroCorreos(tx).RecuperarOperacion(context.Background(), orden, m, v3)
	if err != nil || existe || recibo.ReciboRef != "" || tx.commits != 1 {
		t.Fatalf("NULL de SQL no se trató como operación ausente: %v %v", existe, err)
	}
	// El núcleo AD3 rechaza transacciones con más de 20 s de inactividad.
	var ajustes []string
	for _, l := range tx.llamadas {
		ajustes = append(ajustes, l.sql)
	}
	if !strings.Contains(strings.Join(ajustes, "\n"), "pg_catalog.set_config('idle_in_transaction_session_timeout','20s',true)") {
		t.Fatalf("límite de inactividad incompatible con AD3: %v", ajustes)
	}
}
