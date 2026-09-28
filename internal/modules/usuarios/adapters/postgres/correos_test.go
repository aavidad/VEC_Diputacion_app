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

	"github.com/jackc/pgx/v5/pgconn"

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
		*p = append([]byte(nil), f.valor.([]byte)...)
	}
	return nil
}

type txCorreoPGPrueba struct {
	respuestas         []filaCorreoPGPrueba
	llamadas           []llamadaCorreoPG
	commits, rollbacks int
}

func (t *txCorreoPGPrueba) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	t.llamadas = append(t.llamadas, llamadaCorreoPG{sql, args})
	return pgconn.CommandTag{}, nil
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

func pruebaOrdenYV3Correos(t *testing.T, accion string) (ports.OrdenCorreos, ports.MaterialCorreos, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) {
	t.Helper()
	actor, vinculo := identidadCorreoPGPrueba(t)
	superficie := vecdomain.SuperficieAutenticacionInternaCorporativaV1
	m := ports.MaterialCorreos{Superficie: superficie, PersonaRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, Accion: accion, FinalidadRef: ports.FinalidadCorreosPropios, VersionEsperada: 1, ClaveOperacion: "clave-operacion-123", CorreoRef: "correo:1234567890123456", HuellasPeticion: ports.HuellasSemanticasCorreo{Activa: ports.HuellaSemanticaCorreo{ClaveRef: "hmac:v2", Valor: strings.Repeat("a", 64)}, Retenidas: []ports.HuellaSemanticaCorreo{{ClaveRef: "hmac:v1", Valor: strings.Repeat("b", 64)}}}}
	if accion == ports.AccionConsultarCorreos {
		m.VersionEsperada = 0
		m.ClaveOperacion = ""
		m.CorreoRef = ""
		m.HuellasPeticion = ports.HuellasSemanticasCorreo{}
	}
	if accion == ports.AccionAnadirCorreo {
		m.CorreoRef = ""
	}
	material, err := ports.SerializarMaterialCorreos(m)
	if err != nil {
		t.Fatal(err)
	}
	huella := sha256.Sum256(material)
	orden, err := ports.NuevaOrdenCorreos(actor, vinculo, superficie, &proveedorCorreoPGPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	audiencia, err := ports.AudienciaCorreos(accion, superficie)
	if err != nil {
		t.Fatal(err)
	}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("dec_prueba", strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_prueba", strings.Repeat("c", 64), accion, actor.PersonaRef, hex.EncodeToString(huella[:]), audiencia, ahora, ahora.Add(time.Second))
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
	return orden, m, v3
}

func pruebaRegistroCorreos(tx *txCorreoPGPrueba) *RegistroCorreosPostgreSQL {
	return &RegistroCorreosPostgreSQL{iniciar: func(context.Context) (transaccionCorreos, error) { return tx, nil }, descifrador: &descifradorCorreoPGPrueba{}, superficie: vecdomain.SuperficieAutenticacionInternaCorporativaV1}
}
func reciboPGPrueba(m ports.MaterialCorreos, replay bool) []byte {
	b, _ := json.Marshal(ports.ReciboCorreos{ReciboRef: "recibo:original", PersonaRef: m.PersonaRef, Accion: m.Accion, CorreoRef: m.CorreoRef, Version: m.VersionEsperada + 1, FechaUTC: time.Now().UTC().Truncate(time.Microsecond), Replay: replay})
	return b
}

func TestMaterialCorreosSerializaHuellasSinClaros(t *testing.T) {
	_, m, _ := pruebaOrdenYV3Correos(t, ports.AccionVerificarCorreo)
	b, err := ports.SerializarMaterialCorreos(m)
	if err != nil {
		t.Fatal(err)
	}
	var j map[string]any
	if err := json.Unmarshal(b, &j); err != nil {
		t.Fatal(err)
	}
	if _, ok := j["huellas_peticion"].(map[string]any)["retenidas"]; !ok {
		t.Fatal("faltan huellas retenidas")
	}
	if strings.Contains(string(b), "direccion") || strings.Contains(string(b), "codigo") || strings.Contains(string(b), "ret_enidas") {
		t.Fatal("material incluye secretos o nombre incorrecto")
	}
}

func TestVerificacionMismaTransaccionNoEnviaCodigoSQL(t *testing.T) {
	o, m, v3 := pruebaOrdenYV3Correos(t, ports.AccionVerificarCorreo)
	meta := []byte(`{"persona_ref":"` + m.PersonaRef + `","correo_ref":"` + m.CorreoRef + `","desafio_ref":"desafio:uno","huella_codigo_hex":"` + strings.Repeat("ab", 32) + `","clave_ref":"clave:v1","vence_utc":"2026-09-30T10:00:00Z"}`)
	tx := &txCorreoPGPrueba{respuestas: []filaCorreoPGPrueba{{valor: true}, {valor: meta}, {valor: reciboPGPrueba(m, false)}}}
	r := pruebaRegistroCorreos(tx)
	c := comprobadorCorreoPGPrueba{valido: true}
	p := ports.PeticionCorreo{VersionEsperada: 1, ClaveOperacion: m.ClaveOperacion, CorreoRef: m.CorreoRef}
	recibo, err := r.Aplicar(context.Background(), o, p, m, v3, ports.SobreDireccionCorreo{}, ports.ReservaDesafio{}, &c)
	if err != nil || recibo.ReciboRef == "" || tx.commits != 1 || !c.llamado {
		t.Fatalf("verificación fallida: %v", err)
	}
	if tx.llamadas[len(tx.llamadas)-2].sql != prepararVerificacionSQL || tx.llamadas[len(tx.llamadas)-1].sql != cerrarVerificacionSQL {
		t.Fatal("preparación y cierre no comparten transacción")
	}
	for _, llamada := range tx.llamadas {
		for _, arg := range llamada.args {
			if s, ok := arg.(string); ok && strings.Contains(s, "CODIGO_SECRETO") {
				t.Fatal("código enviado a SQL")
			}
		}
	}
}

type comprobadorCorreoPGPrueba struct{ valido, llamado bool }

func (c *comprobadorCorreoPGPrueba) Comprobar(_ context.Context, m ports.MetadatosDesafioCorreo) (bool, error) {
	c.llamado = true
	return c.valido, nil
}

func TestVerificacionIncorrectaConfirmaIntentoSinRecibo(t *testing.T) {
	o, m, v3 := pruebaOrdenYV3Correos(t, ports.AccionVerificarCorreo)
	meta := []byte(`{"persona_ref":"` + m.PersonaRef + `","correo_ref":"` + m.CorreoRef + `","desafio_ref":"desafio:uno","huella_codigo_hex":"` + strings.Repeat("ab", 32) + `","clave_ref":"clave:v1","vence_utc":"2026-09-30T10:00:00Z"}`)
	tx := &txCorreoPGPrueba{respuestas: []filaCorreoPGPrueba{{valor: true}, {valor: meta}, {valor: []byte(`{"valido":false}`)}}}
	r := pruebaRegistroCorreos(tx)
	_, err := r.Aplicar(context.Background(), o, ports.PeticionCorreo{VersionEsperada: 1, ClaveOperacion: m.ClaveOperacion, CorreoRef: m.CorreoRef}, m, v3, ports.SobreDireccionCorreo{}, ports.ReservaDesafio{}, &comprobadorCorreoPGPrueba{})
	if !errors.Is(err, ports.ErrCorreosInvalidos) || tx.commits != 1 {
		t.Fatalf("intento no duradero: %v", err)
	}
}

func TestReplayConservaReciboOriginalYConflictoSerializableFalla(t *testing.T) {
	o, m, v3 := pruebaOrdenYV3Correos(t, ports.AccionReenviarCorreo)
	tx := &txCorreoPGPrueba{respuestas: []filaCorreoPGPrueba{{valor: true}, {valor: reciboPGPrueba(m, true)}}}
	r := pruebaRegistroCorreos(tx)
	recibo, si, err := r.RecuperarOperacion(context.Background(), o, m, v3)
	if err != nil || !si || !recibo.Replay || recibo.ReciboRef != "recibo:original" || tx.commits != 1 {
		t.Fatalf("replay incorrecto: %v", err)
	}
	tx = &txCorreoPGPrueba{respuestas: []filaCorreoPGPrueba{{valor: true}, {err: &pgconn.PgError{Code: "40001", Message: "dato sensible"}}}}
	r = pruebaRegistroCorreos(tx)
	_, _, err = r.RecuperarOperacion(context.Background(), o, m, v3)
	if !errors.Is(err, ports.ErrCorreosNoDisponible) || strings.Contains(err.Error(), "dato sensible") || tx.commits != 0 {
		t.Fatalf("40001 tratado como éxito: %v", err)
	}
}

func TestAltaEnviaSoloSobreYReservaConJSONCerrado(t *testing.T) {
	o, m, v3 := pruebaOrdenYV3Correos(t, ports.AccionAnadirCorreo)
	m.CorreoRef = ""
	ref := "correo:1234567890123456"
	p := ports.PeticionCorreo{VersionEsperada: m.VersionEsperada, ClaveOperacion: m.ClaveOperacion, CorreoRef: ref}
	sobre := ports.SobreDireccionCorreo{CorreoRef: ref, Version: 2, ClaveRef: "clave:sobre", Nonce: bytes.Repeat([]byte{1}, 12), Cifrado: bytes.Repeat([]byte{2}, 32), HuellaIgualdad: bytes.Repeat([]byte{3}, 32)}
	reserva := ports.ReservaDesafio{DesafioRef: "desafio:1234567890123456", Desafio: bytes.Repeat([]byte{4}, 16), HuellaCodigo: bytes.Repeat([]byte{5}, 32), ClaveRef: "clave:desafio", VenceUTC: time.Now().UTC().Truncate(time.Microsecond).Add(time.Hour)}
	recibo := reciboPGPrueba(m, false)
	var j map[string]any
	_ = json.Unmarshal(recibo, &j)
	j["correo_ref"] = ref
	recibo, _ = json.Marshal(j)
	tx := &txCorreoPGPrueba{respuestas: []filaCorreoPGPrueba{{valor: true}, {valor: recibo}}}
	r := pruebaRegistroCorreos(tx)
	got, err := r.Aplicar(context.Background(), o, p, m, v3, sobre, reserva, nil)
	if err != nil || got.CorreoRef != ref || tx.commits != 1 {
		t.Fatalf("alta fallida: %v", err)
	}
	llamada := tx.llamadas[len(tx.llamadas)-1]
	if llamada.sql != aplicarCorreosSQL || len(llamada.args) != 13 {
		t.Fatal("firma SQL de alta incorrecta")
	}
	material := llamada.args[0].(string)
	if strings.Contains(material, "example.org") || strings.Contains(material, "CODIGO") || strings.Contains(material, ref) {
		t.Fatal("material contiene claro o ref aleatorio no semántico")
	}
	var sj, rj map[string]any
	if json.Unmarshal(llamada.args[1].([]byte), &sj) != nil || json.Unmarshal(llamada.args[2].([]byte), &rj) != nil {
		t.Fatal("sobre/reserva no son JSON")
	}
	if len(sj) != 6 || len(rj) != 5 || sj["correo_ref"] != ref || rj["huella_codigo_hex"] != hex.EncodeToString(reserva.HuellaCodigo) {
		t.Fatal("campos de sobre/reserva incorrectos")
	}
	for _, arg := range llamada.args {
		if s, ok := arg.(string); ok && strings.Contains(s, "example.org") {
			t.Fatal("dirección clara enviada a SQL")
		}
	}
}

func TestActorAjenoSeRechazaAntesDeAbrirConexion(t *testing.T) {
	o, m, v3 := pruebaOrdenYV3Correos(t, ports.AccionReenviarCorreo)
	m.PersonaRef = "per_otra_persona_0123456789abcd"
	abierto := false
	r := &RegistroCorreosPostgreSQL{iniciar: func(context.Context) (transaccionCorreos, error) { abierto = true; return nil, nil }, descifrador: &descifradorCorreoPGPrueba{}, superficie: vecdomain.SuperficieAutenticacionInternaCorporativaV1}
	_, _, err := r.RecuperarOperacion(context.Background(), o, m, v3)
	if !errors.Is(err, ports.ErrCorreosProhibido) || abierto {
		t.Fatalf("actor ajeno alcanzó DB: %v", err)
	}
}

func Test40001SoloAceptaReplayTrasRecuperacionConV3Nuevo(t *testing.T) {
	o, m, v3 := pruebaOrdenYV3Correos(t, ports.AccionReenviarCorreo)
	actor, err := o.ContextoActor()
	if err != nil {
		t.Fatal(err)
	}
	proveedor := &proveedorCorreoPGPrueba{v3: v3}
	vinculo, err := o.Vinculo()
	if err != nil {
		t.Fatal(err)
	}
	o, err = ports.NuevaOrdenCorreos(actor, vinculo, vecdomain.SuperficieAutenticacionInternaCorporativaV1, proveedor)
	if err != nil {
		t.Fatal(err)
	}
	primera := &txCorreoPGPrueba{respuestas: []filaCorreoPGPrueba{{valor: true}, {err: &pgconn.PgError{Code: "40001"}}}}
	segunda := &txCorreoPGPrueba{respuestas: []filaCorreoPGPrueba{{valor: true}, {valor: reciboPGPrueba(m, true)}}}
	veces := 0
	r := &RegistroCorreosPostgreSQL{descifrador: &descifradorCorreoPGPrueba{}, superficie: vecdomain.SuperficieAutenticacionInternaCorporativaV1, iniciar: func(context.Context) (transaccionCorreos, error) {
		veces++
		if veces == 1 {
			return primera, nil
		}
		return segunda, nil
	}}
	reserva := ports.ReservaDesafio{DesafioRef: "desafio:1234567890123456", Desafio: bytes.Repeat([]byte{4}, 16), HuellaCodigo: bytes.Repeat([]byte{5}, 32), ClaveRef: "clave:desafio", VenceUTC: time.Now().UTC().Truncate(time.Microsecond).Add(time.Hour)}
	p := ports.PeticionCorreo{VersionEsperada: 1, ClaveOperacion: m.ClaveOperacion, CorreoRef: m.CorreoRef}
	recibo, err := r.Aplicar(context.Background(), o, p, m, v3, ports.SobreDireccionCorreo{}, reserva, nil)
	if err != nil || !recibo.Replay || recibo.ReciboRef != "recibo:original" || proveedor.llamadas != 1 || primera.commits != 0 || segunda.commits != 1 || veces != 2 {
		t.Fatalf("40001 sin recibo acreditado: %v", err)
	}
}

func TestConsultaPropiaDescifraBajoTransaccionAutorizada(t *testing.T) {
	o, m, v3 := pruebaOrdenYV3Correos(t, ports.AccionConsultarCorreos)
	m.VersionEsperada = 0
	m.ClaveOperacion = ""
	m.CorreoRef = ""
	m.HuellasPeticion = ports.HuellasSemanticasCorreo{}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	bruto, _ := json.Marshal(map[string]any{"persona_ref": m.PersonaRef, "version": uint64(2), "correos": []any{map[string]any{"correo_ref": "correo:1234567890123456", "estado": "verificado", "activo": true, "creado_utc": ahora, "verificado_utc": ahora, "sobre": map[string]any{"version": uint64(1), "clave_ref": "clave:sobre", "nonce_hex": strings.Repeat("ab", 12), "cifrado_hex": strings.Repeat("cd", 32)}}}})
	tx := &txCorreoPGPrueba{respuestas: []filaCorreoPGPrueba{{valor: true}, {valor: bruto}}}
	d := &descifradorCorreoPGPrueba{}
	r := pruebaRegistroCorreos(tx)
	r.descifrador = d
	vista, err := r.ConsultarPropios(context.Background(), o, m, v3)
	if err != nil || tx.commits != 1 || d.llamados != 1 || d.persona != m.PersonaRef || d.version != 1 || vista.Version != 2 || len(vista.Correos) != 1 || vista.Correos[0].Direccion != "ejemplo@example.org" {
		t.Fatalf("consulta/descifrado incorrecto: %v", err)
	}
	if tx.llamadas[len(tx.llamadas)-1].sql != consultarCorreosSQL {
		t.Fatal("consulta fuera de función nominal")
	}
}

func TestLectorActivoRespetaPersonaObjetivoYVersionAAD(t *testing.T) {
	o, _, _ := pruebaOrdenYV3Correos(t, ports.AccionConsultarCorreos)
	actor, err := o.ContextoActor()
	if err != nil {
		t.Fatal(err)
	}
	objetivo := "per_otra_persona_0123456789abcd"
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("dec_ct", strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_ct", strings.Repeat("c", 64), accionActivoCT, objetivo, strings.Repeat("d", 64), audienciaActivoCT, ahora, ahora.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	raiz, _ := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	canon, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	v3, err := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), resumen, []byte("decision"), []byte("motivo"), canon, 1, 1, []byte("payload"), []byte("sobre"), []byte("evidencia"), raiz)
	if err != nil {
		t.Fatal(err)
	}
	bruto, _ := json.Marshal(map[string]any{"persona_ref": objetivo, "correo_ref": "correo:1234567890123456", "version": uint64(1), "conjunto_version": uint64(4), "clave_ref": "clave:sobre", "nonce_hex": strings.Repeat("ab", 12), "cifrado_hex": strings.Repeat("cd", 32)})
	tx := &txCorreoPGPrueba{respuestas: []filaCorreoPGPrueba{{valor: true}, {valor: bruto}}}
	d := &descifradorCorreoPGPrueba{}
	r := pruebaRegistroCorreos(tx)
	r.descifrador = d
	l := &LectorCorreoActivoPostgreSQL{registro: r, habilitado: true} // solo fake; constructor real permanece cerrado.
	activo, si, err := l.ConsultarActivoVerificado(context.Background(), ports.SolicitudCorreoActivo{PersonaRef: objetivo, FinalidadRef: finalidadActivoCT, Material: v3})
	if err != nil || !si || tx.commits != 1 || d.persona != objetivo || d.version != 1 || activo.Version != 4 {
		t.Fatalf("lector activo incorrecto: %v", err)
	}
	llamada := tx.llamadas[len(tx.llamadas)-1]
	var m map[string]any
	if json.Unmarshal([]byte(llamada.args[0].(string)), &m) != nil || m["persona_ref"] != objetivo || m["perfil_ref"] != actor.PerfilActivoRef || m["accion"] != accionActivoCT {
		t.Fatal("lector no ligó persona objetivo y perfil actor")
	}
}

func TestLectorTecnicoNoSeConstruyeSinAD3Nominal(t *testing.T) {
	l, err := NuevoLectorCorreoActivoPostgreSQL(context.Background(), nil, &descifradorCorreoPGPrueba{})
	if l != nil || !errors.Is(err, ports.ErrCorreosNoDisponible) {
		t.Fatalf("lector publicado sin contrato técnico: %v", err)
	}
	_, _, err = (&LectorCorreoActivoPostgreSQL{}).ConsultarActivoVerificado(context.Background(), ports.SolicitudCorreoActivo{})
	if !errors.Is(err, ports.ErrCorreosNoDisponible) {
		t.Fatal("lector vacío no falla cerrado")
	}
}

func TestEjecutorPropioSeparadoPorSuperficie(t *testing.T) {
	if rolEjecutorCorreos(vecdomain.SuperficieAutenticacionInternaCorporativaV1) != "vec_usuarios_ejecutor_interno" ||
		rolEjecutorCorreos(vecdomain.SuperficieAutenticacionExternaPersonalV1) != "vec_usuarios_ejecutor_externo" || rolEjecutorCorreos("superficie-libre") != "" {
		t.Fatal("selección de rol insegura")
	}
	tx := &txCorreoPGPrueba{respuestas: []filaCorreoPGPrueba{{valor: true}}}
	r := pruebaRegistroCorreos(tx)
	opened, err := r.abrir(context.Background())
	if err != nil || opened == nil {
		t.Fatalf("preflight interno: %v", err)
	}
	llamada := tx.llamadas[len(tx.llamadas)-1]
	if llamada.sql != acreditarEjecutorCorreosSQL || len(llamada.args) != 1 || llamada.args[0] != "vec_usuarios_ejecutor_interno" {
		t.Fatal("preflight no ligó rol interno")
	}
}
