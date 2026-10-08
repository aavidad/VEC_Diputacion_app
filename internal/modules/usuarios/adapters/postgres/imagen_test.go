package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"regexp"
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

type filaImagenPGPrueba struct {
	valores []any
	err     error
}

func (f filaImagenPGPrueba) Scan(dest ...any) error {
	if f.err != nil {
		return f.err
	}
	for i, d := range dest {
		switch p := d.(type) {
		case *bool:
			*p = f.valores[i].(bool)
		case *[]byte:
			if f.valores[i] == nil {
				*p = nil
				continue
			}
			*p = append([]byte(nil), f.valores[i].([]byte)...)
		}
	}
	return nil
}

type txImagenPGPrueba struct {
	respuestas         []filaImagenPGPrueba
	llamadas           []llamadaCorreoPG
	commits, rollbacks int
	errExec            error
}

func (t *txImagenPGPrueba) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	t.llamadas = append(t.llamadas, llamadaCorreoPG{sql, args})
	return pgconn.CommandTag{}, t.errExec
}
func (t *txImagenPGPrueba) QueryRow(_ context.Context, sql string, args ...any) filaCorreos {
	t.llamadas = append(t.llamadas, llamadaCorreoPG{sql, args})
	if len(t.respuestas) == 0 {
		return filaImagenPGPrueba{err: errors.New("sin respuesta")}
	}
	f := t.respuestas[0]
	t.respuestas = t.respuestas[1:]
	return f
}
func (t *txImagenPGPrueba) Commit(context.Context) error   { t.commits++; return nil }
func (t *txImagenPGPrueba) Rollback(context.Context) error { t.rollbacks++; return nil }

type proveedorImagenPGPrueba struct {
	v3       vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	llamadas int
}

func (p *proveedorImagenPGPrueba) ProveerMaterialImagen(context.Context, vecdomain.VinculoAutenticacionActorV2, ports.MaterialImagen) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	p.llamadas++
	return p.v3, nil
}

var jpegPrueba = []byte{0xff, 0xd8, 0xff, 0xe0, 1, 2, 3, 0xff, 0xd9}

func pruebaOrdenYV3Imagen(t *testing.T, accion string, conFoto bool) (ports.OrdenImagen, ports.MaterialImagen, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) {
	t.Helper()
	actor, vinculo := identidadCorreoPGPrueba(t)
	superficie := vecdomain.SuperficieAutenticacionInternaCorporativaV1
	m := ports.MaterialImagen{Superficie: superficie, PersonaRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, Accion: accion,
		FinalidadRef: ports.FinalidadImagenPropia, CatalogoVersionRef: "usuarios-imagen-v1"}
	if accion == ports.AccionActualizarImagen {
		m.VersionEsperada, m.ClaveOperacion = 2, "web-imagen-1234567890"
		m.Eleccion = domain.EleccionImagen{Modo: domain.ModoImagenIcono, Paleta: "verde", Icono: "sol"}
		if conFoto {
			m.Eleccion = domain.EleccionImagen{Modo: domain.ModoImagenFoto, Paleta: "verde"}
			h := sha256Hex(jpegPrueba)
			m.FotoSHA256 = h
		}
		m.HuellaPeticion = canonico.HuellaPeticionImagen(m.PersonaRef, m.VersionEsperada, m.CatalogoVersionRef, m.Eleccion, m.FotoSHA256)
	}
	recurso, err := canonico.RecursoImagen(m)
	if err != nil {
		t.Fatal(err)
	}
	huella, _ := recurso.HuellaContextoAutorizacionSHA256()
	identidad, err := domain.NuevaIdentidadImagen(actor, vinculo, superficie)
	if err != nil {
		t.Fatal(err)
	}
	audiencia, _ := canonico.AudienciaImagen(accion, superficie)
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("dec_prueba", strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_prueba", strings.Repeat("c", 64), accion, actor.PersonaRef, huella, audiencia, ahora, ahora.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	raiz, _ := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	canon, _ := actor.RepresentacionCanonicaVinculadaV2()
	v3, err := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), resumen, []byte("decision"), []byte("motivo"), canon, 1, 1, []byte("payload"), []byte("sobre"), []byte("evidencia"), raiz)
	if err != nil {
		t.Fatal(err)
	}
	return ports.OrdenImagen{Identidad: identidad, Proveedor: &proveedorImagenPGPrueba{v3: v3}}, m, v3
}

func sha256Hex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func pruebaRegistroImagen(tx *txImagenPGPrueba) *RegistroImagenPostgreSQL {
	return &RegistroImagenPostgreSQL{iniciar: func(context.Context) (transaccionCorreos, error) { return tx, nil }, superficie: vecdomain.SuperficieAutenticacionInternaCorporativaV1}
}

func apertura(extra ...filaImagenPGPrueba) []filaImagenPGPrueba {
	return append([]filaImagenPGPrueba{{valores: []any{true}}}, extra...)
}

func reciboImagenPG(m ports.MaterialImagen, replay, fotoNueva bool) []byte {
	b, _ := json.Marshal(map[string]any{"recibo_ref": "img_" + strings.Repeat("1", 32), "persona_ref": m.PersonaRef, "version": m.VersionEsperada + 1,
		"catalogo_version_ref": m.CatalogoVersionRef, "eleccion": m.Eleccion, "foto_nueva": fotoNueva, "foto_retirada": false,
		"fecha_utc": fechaPG(), "replay": replay})
	return b
}

func TestConsultarImagenEntregaFotoSoloEnModoFoto(t *testing.T) {
	orden, m, v3 := pruebaOrdenYV3Imagen(t, ports.AccionConsultarImagen, false)
	estado := func(modo string) []byte {
		b, _ := json.Marshal(map[string]any{"existe": true, "persona_ref": m.PersonaRef, "version": 4, "catalogo_version_ref": "usuarios-imagen-v1",
			"eleccion": map[string]string{"modo": modo, "paleta": "verde", "icono": ""}})
		return b
	}
	tx := &txImagenPGPrueba{respuestas: apertura(filaImagenPGPrueba{valores: []any{estado("foto"), jpegPrueba}})}
	e, existe, foto, err := pruebaRegistroImagen(tx).Consultar(context.Background(), orden, m, v3)
	if err != nil || !existe || e.Version != 4 || foto == nil || !bytes.Equal(foto.Datos, jpegPrueba) || foto.Tipo != ports.TipoFotoImagen || tx.commits != 1 {
		t.Fatalf("consulta con foto: %v %+v", err, e)
	}
	if tx.llamadas[len(tx.llamadas)-1].sql != consultarImagenSQL {
		t.Fatal("no se usó la fachada de consulta")
	}
	tx = &txImagenPGPrueba{respuestas: apertura(filaImagenPGPrueba{valores: []any{estado("iniciales"), jpegPrueba}})}
	if _, _, _, err := pruebaRegistroImagen(tx).Consultar(context.Background(), orden, m, v3); !errors.Is(err, ports.ErrImagenNoDisponible) || tx.commits != 0 {
		t.Fatalf("foto fuera del modo foto aceptada: %v", err)
	}
}

func TestGuardarImagenPasaLosBytesSoloComoParametro(t *testing.T) {
	orden, m, v3 := pruebaOrdenYV3Imagen(t, ports.AccionActualizarImagen, true)
	tx := &txImagenPGPrueba{respuestas: apertura(filaImagenPGPrueba{valores: []any{reciboImagenPG(m, false, true)}})}
	recibo, err := pruebaRegistroImagen(tx).Guardar(context.Background(), orden, m, jpegPrueba, v3)
	if err != nil || recibo.Version != 3 || !recibo.FotoNueva || tx.commits != 1 {
		t.Fatalf("guardar foto: %v %+v", err, recibo)
	}
	llamada := tx.llamadas[len(tx.llamadas)-1]
	if llamada.sql != guardarImagenSQL || !bytes.Equal(llamada.args[1].([]byte), jpegPrueba) {
		t.Fatal("la foto no viaja como parámetro bytea propio")
	}
	if material, ok := llamada.args[0].(string); !ok || strings.Contains(material, string(jpegPrueba)) || !strings.Contains(material, m.FotoSHA256) {
		t.Fatal("el material debe llevar la huella de la foto, nunca sus bytes")
	}
	if _, err := pruebaRegistroImagen(&txImagenPGPrueba{}).Guardar(context.Background(), orden, m, nil, v3); !errors.Is(err, ports.ErrImagenPeticionInvalida) {
		t.Fatalf("material con huella de foto y sin bytes aceptado: %v", err)
	}
	orden, m, v3 = pruebaOrdenYV3Imagen(t, ports.AccionActualizarImagen, false)
	tx = &txImagenPGPrueba{respuestas: apertura(filaImagenPGPrueba{valores: []any{reciboImagenPG(m, false, false)}})}
	if _, err := pruebaRegistroImagen(tx).Guardar(context.Background(), orden, m, nil, v3); err != nil || tx.llamadas[len(tx.llamadas)-1].args[1] != nil {
		t.Fatalf("elección sin foto debe pasar NULL: %v", err)
	}
}

func TestErroresSQLDeImagen(t *testing.T) {
	orden, m, v3 := pruebaOrdenYV3Imagen(t, ports.AccionActualizarImagen, false)
	for codigo, esperado := range map[string]error{"P1409": ports.ErrImagenConflicto, "22023": ports.ErrImagenPeticionInvalida, "42501": ports.ErrImagenProhibido, "55000": ports.ErrImagenNoDisponible} {
		tx := &txImagenPGPrueba{respuestas: apertura(filaImagenPGPrueba{err: &pgconn.PgError{Code: codigo, Message: "detalle interno per_x docimg_y"}})}
		_, err := pruebaRegistroImagen(tx).Guardar(context.Background(), orden, m, nil, v3)
		if !errors.Is(err, esperado) || strings.Contains(err.Error(), "per_") || strings.Contains(err.Error(), "docimg_") {
			t.Fatalf("%s: esperado %v, obtenido %v", codigo, esperado, err)
		}
	}
	// Recuperar sin operación previa: NULL.
	tx := &txImagenPGPrueba{respuestas: apertura(filaImagenPGPrueba{valores: []any{[]byte("null")}})}
	if _, existe, err := pruebaRegistroImagen(tx).RecuperarOperacion(context.Background(), orden, m, v3); err != nil || existe || tx.commits != 1 {
		t.Fatalf("recuperación ausente: %v", err)
	}
}

func TestCarreraSerializableRecuperaConV3Fresca(t *testing.T) {
	orden, m, v3 := pruebaOrdenYV3Imagen(t, ports.AccionActualizarImagen, false)
	tx := &txImagenPGPrueba{respuestas: []filaImagenPGPrueba{{valores: []any{true}}, {err: &pgconn.PgError{Code: "40001"}},
		{valores: []any{true}}, {valores: []any{reciboImagenPG(m, true, false)}}}}
	recibo, err := pruebaRegistroImagen(tx).Guardar(context.Background(), orden, m, nil, v3)
	if err != nil || !recibo.Replay || orden.Proveedor.(*proveedorImagenPGPrueba).llamadas != 1 {
		t.Fatalf("40001 no se resolvió recuperando el recibo con V3 nueva: %v %+v", err, recibo)
	}
	tx = &txImagenPGPrueba{respuestas: []filaImagenPGPrueba{{valores: []any{true}}, {err: &pgconn.PgError{Code: "40001"}},
		{valores: []any{true}}, {valores: []any{[]byte("null")}}}}
	if _, err := pruebaRegistroImagen(tx).Guardar(context.Background(), orden, m, nil, v3); !errors.Is(err, ports.ErrImagenConflicto) {
		t.Fatalf("40001 sin recibo propio debe ser conflicto: %v", err)
	}
}

func TestMaterialDeOtraPersonaNoAbreTransaccion(t *testing.T) {
	orden, m, v3 := pruebaOrdenYV3Imagen(t, ports.AccionConsultarImagen, false)
	m.PersonaRef = "per_" + strings.Repeat("z", 24)
	tx := &txImagenPGPrueba{}
	if _, _, _, err := pruebaRegistroImagen(tx).Consultar(context.Background(), orden, m, v3); err == nil || len(tx.llamadas) != 0 {
		t.Fatalf("material ajeno llegó a SQL: %v", err)
	}
}

// El catálogo que publica Usuarios 000006 es exactamente el del dominio.
func TestCatalogoSQLCoincideConDominio(t *testing.T) {
	sql, err := os.ReadFile("../../../../../deploy/postgresql/usuarios_vec/migraciones/000006_imagen_propia.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	coincidencia := regexp.MustCompile(`(?s)FROM \(SELECT '(\{.*?\})'::jsonb AS c\) q;`).FindSubmatch(sql)
	if coincidencia == nil {
		t.Fatal("no se encuentra el catálogo en la migración")
	}
	var c domain.CatalogoImagen
	if err := json.Unmarshal(coincidencia[1], &c); err != nil || c.Validar() != nil {
		t.Fatalf("catálogo SQL inválido: %v", err)
	}
	if !reflect.DeepEqual(c, domain.CatalogoBaseImagen()) {
		t.Fatalf("catálogo SQL distinto del dominio:\n%+v\n%+v", c, domain.CatalogoBaseImagen())
	}
}
